package handler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type concurrencyCacheMock struct {
	acquireUserSlotFn      func(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error)
	acquireAccountSlotFn   func(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error)
	acquireGroupUserSlotFn func(ctx context.Context, groupID, userID int64, maxConcurrency int, requestID string) (bool, error)
	releaseUserCalled      int32
	releaseAccountCalled   int32
	releaseGroupUserCalled int32
}

func (m *concurrencyCacheMock) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	if m.acquireAccountSlotFn != nil {
		return m.acquireAccountSlotFn(ctx, accountID, maxConcurrency, requestID)
	}
	return false, nil
}

func (m *concurrencyCacheMock) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	atomic.AddInt32(&m.releaseAccountCalled, 1)
	return nil
}

func (m *concurrencyCacheMock) GetAccountConcurrency(ctx context.Context, accountID int64) (int, error) {
	return 0, nil
}

func (m *concurrencyCacheMock) GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = 0
	}
	return result, nil
}

func (m *concurrencyCacheMock) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	return true, nil
}

func (m *concurrencyCacheMock) DecrementAccountWaitCount(ctx context.Context, accountID int64) error {
	return nil
}

func (m *concurrencyCacheMock) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	return 0, nil
}

func (m *concurrencyCacheMock) AcquireGroupUserSlot(ctx context.Context, groupID, userID int64, maxConcurrency int, requestID string) (bool, error) {
	if m.acquireGroupUserSlotFn != nil {
		return m.acquireGroupUserSlotFn(ctx, groupID, userID, maxConcurrency, requestID)
	}
	return true, nil
}

func (m *concurrencyCacheMock) ReleaseGroupUserSlot(ctx context.Context, groupID, userID int64, requestID string) error {
	atomic.AddInt32(&m.releaseGroupUserCalled, 1)
	return nil
}

func (m *concurrencyCacheMock) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
	if m.acquireUserSlotFn != nil {
		return m.acquireUserSlotFn(ctx, userID, maxConcurrency, requestID)
	}
	return false, nil
}

func (m *concurrencyCacheMock) ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error {
	atomic.AddInt32(&m.releaseUserCalled, 1)
	return nil
}

func (m *concurrencyCacheMock) GetUserConcurrency(ctx context.Context, userID int64) (int, error) {
	return 0, nil
}

func (m *concurrencyCacheMock) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	return true, nil
}

func (m *concurrencyCacheMock) DecrementWaitCount(ctx context.Context, userID int64) error {
	return nil
}

func (m *concurrencyCacheMock) GetAccountsLoadBatch(ctx context.Context, accounts []service.AccountWithConcurrency) (map[int64]*service.AccountLoadInfo, error) {
	return map[int64]*service.AccountLoadInfo{}, nil
}

func (m *concurrencyCacheMock) GetUsersLoadBatch(ctx context.Context, users []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	return map[int64]*service.UserLoadInfo{}, nil
}

func (m *concurrencyCacheMock) CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error {
	return nil
}

func (m *concurrencyCacheMock) CleanupExpiredAccountSlotKeys(ctx context.Context) error {
	return nil
}

func (m *concurrencyCacheMock) CleanupStaleProcessSlots(ctx context.Context, activeRequestPrefix string) error {
	return nil
}

func TestConcurrencyHelper_TryAcquireUserSlot(t *testing.T) {
	cache := &concurrencyCacheMock{
		acquireUserSlotFn: func(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
			return true, nil
		},
	}
	helper := NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second)

	release, acquired, err := helper.TryAcquireUserSlot(context.Background(), 101, 2)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, release)

	release()
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseUserCalled))
}

func TestConcurrencyHelper_TryAcquireAccountSlot_NotAcquired(t *testing.T) {
	cache := &concurrencyCacheMock{
		acquireAccountSlotFn: func(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
			return false, nil
		},
	}
	helper := NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second)

	release, acquired, err := helper.TryAcquireAccountSlot(context.Background(), 201, 1)
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, release)
	require.Equal(t, int32(0), atomic.LoadInt32(&cache.releaseAccountCalled))
}

func TestConcurrencyHelper_TryAcquireGroupUserSlot(t *testing.T) {
	cache := &concurrencyCacheMock{
		acquireGroupUserSlotFn: func(ctx context.Context, groupID, userID int64, maxConcurrency int, requestID string) (bool, error) {
			require.Equal(t, int64(2), groupID)
			require.Equal(t, int64(101), userID)
			require.Equal(t, 1, maxConcurrency)
			return true, nil
		},
	}
	helper := NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second)

	release, acquired, err := helper.TryAcquireGroupUserSlot(context.Background(), 2, 101, 1)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, release)

	release()
	require.Equal(t, int32(1), atomic.LoadInt32(&cache.releaseGroupUserCalled))
}

func TestConcurrencyHelper_GroupUserGateCanBeAcquiredBeforeUserGate(t *testing.T) {
	var callOrder []string
	cache := &concurrencyCacheMock{
		acquireGroupUserSlotFn: func(ctx context.Context, groupID, userID int64, maxConcurrency int, requestID string) (bool, error) {
			callOrder = append(callOrder, "group_user")
			require.Equal(t, int64(7), groupID)
			require.Equal(t, int64(101), userID)
			require.Equal(t, 3, maxConcurrency)
			return true, nil
		},
		acquireUserSlotFn: func(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
			callOrder = append(callOrder, "user")
			require.Equal(t, int64(101), userID)
			require.Equal(t, 5, maxConcurrency)
			return true, nil
		},
	}
	helper := NewConcurrencyHelper(service.NewConcurrencyService(cache), SSEPingFormatNone, time.Second)

	groupRelease, groupAcquired, err := helper.TryAcquireGroupUserSlot(context.Background(), 7, 101, 3)
	require.NoError(t, err)
	require.True(t, groupAcquired)
	require.NotNil(t, groupRelease)
	defer groupRelease()

	userRelease, userAcquired, err := helper.TryAcquireUserSlot(context.Background(), 101, 5)
	require.NoError(t, err)
	require.True(t, userAcquired)
	require.NotNil(t, userRelease)
	defer userRelease()

	require.Equal(t, []string{"group_user", "user"}, callOrder)
}
