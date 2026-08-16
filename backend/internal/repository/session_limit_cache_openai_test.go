package repository

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestOpenAISessionLimitConcurrentAdmissionNeverExceedsMax(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()

	var allowedCount atomic.Int32
	var wg sync.WaitGroup
	errCh := make(chan error, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			allowed, err := cache.RegisterOpenAISessionID(ctx, 99, fmt.Sprintf("session-%d", index), 3, time.Minute)
			if err != nil {
				errCh <- err
				return
			}
			if allowed {
				allowedCount.Add(1)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	require.EqualValues(t, 3, allowedCount.Load())
	count, err := client.ZCard(ctx, openAISessionLimitKey(99)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
}

func TestOpenAISessionLimitSlotsAreAtomicAndIsolated(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()

	for _, sessionID := range []string{"session-a", "session-b", "session-c"} {
		allowed, err := cache.RegisterOpenAISessionID(ctx, 42, sessionID, 3, time.Minute)
		require.NoError(t, err)
		require.True(t, allowed)
	}

	allowed, err := cache.RegisterOpenAISessionID(ctx, 42, "session-d", 3, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)

	allowed, err = cache.RegisterOpenAISessionID(ctx, 42, "session-a", 3, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed, "an admitted SessionID must refresh even when all slots are full")

	allowed, err = cache.RegisterSession(ctx, 42, "anthropic-session", 1, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)
	openAICount, err := client.ZCard(ctx, openAISessionLimitKey(42)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 3, openAICount)
	anthropicCount, err := client.ZCard(ctx, sessionLimitKey(42)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, anthropicCount)
}

func TestOpenAISessionLimitRotationStagesLeastRecentlyActiveSlot(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 43

	for _, sessionID := range []string{"oldest", "middle", "newest"} {
		allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, time.Minute)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	for index, sessionID := range []string{"oldest", "middle", "newest"} {
		require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
			Score:  float64(redisTime.Unix() - int64(30-index*10)),
			Member: sessionID,
		}).Err())
	}

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, "incoming", 3, time.Minute, true)
	require.NoError(t, err)
	require.True(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), "oldest", false)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), "middle", true)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), "newest", true)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), "incoming", true)
	count, err := client.ZCard(ctx, openAISessionLimitKey(accountID)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), "oldest", true)
	owner, err := client.Get(ctx, openAISessionOwnerKey("oldest")).Result()
	require.NoError(t, err)
	require.Equal(t, strconv.FormatInt(accountID, 10), owner)
	stagedAccountID, err := cache.GetOpenAIStagedSessionAccountID(ctx, "oldest")
	require.NoError(t, err)
	require.Equal(t, accountID, stagedAccountID)
}

func TestOpenAISessionLimitRotationRestoresLoweredHardCap(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 44

	for _, sessionID := range []string{"session-a", "session-b", "session-c", "session-d", "session-e"} {
		allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 5, time.Minute)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	for index, sessionID := range []string{"session-a", "session-b", "session-c", "session-d", "session-e"} {
		require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
			Score:  float64(redisTime.Unix() - int64(50-index*10)),
			Member: sessionID,
		}).Err())
	}

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, "incoming", 3, time.Minute, true)
	require.NoError(t, err)
	require.True(t, allowed)
	count, err := client.ZCard(ctx, openAISessionLimitKey(accountID)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
	for _, sessionID := range []string{"session-a", "session-b", "session-c"} {
		requireZMember(t, ctx, client, openAISessionLimitKey(accountID), sessionID, false)
		requireZMember(t, ctx, client, openAISessionStagingKey(accountID), sessionID, true)
		stagedAccountID, err := cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
		require.NoError(t, err)
		require.Equal(t, accountID, stagedAccountID)
	}
	for _, sessionID := range []string{"session-d", "session-e", "incoming"} {
		requireZMember(t, ctx, client, openAISessionLimitKey(accountID), sessionID, true)
	}
}

func TestOpenAISessionLimitExpiresAndCanBeCleared(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()

	allowed, err := cache.RegisterOpenAISessionID(ctx, 7, "old-session", 3, time.Second)
	require.NoError(t, err)
	require.True(t, allowed)

	// miniredis FastForward advances key TTLs but not the Redis TIME value used
	// by the Lua script. Move the sorted-set score into the past explicitly.
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(7), redis.Z{
		Score:  1,
		Member: "old-session",
	}).Err())
	allowed, err = cache.RegisterOpenAISessionID(ctx, 7, "new-session", 3, time.Second)
	require.NoError(t, err)
	require.True(t, allowed)
	_, err = client.Get(ctx, openAISessionOwnerKey("old-session")).Result()
	require.ErrorIs(t, err, redis.Nil)

	counts, err := cache.GetOpenAIActiveSessionCountBatch(ctx, []int64{7}, map[int64]time.Duration{7: time.Second})
	require.NoError(t, err)
	require.Equal(t, 1, counts[7])

	require.NoError(t, cache.ClearOpenAISessions(ctx, 7))
	require.False(t, redisServer.Exists(openAISessionLimitKey(7)))
	require.False(t, redisServer.Exists(openAISessionOwnerKey("new-session")))
}

func TestOpenAISessionLimitExpiredSlotMovesToStagingAndReturns(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 8
	const sessionID = "returning-session"
	const idleTimeout = 10 * time.Second

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 11),
		Member: sessionID,
	}).Err())

	counts, err := cache.GetOpenAIActiveSessionCountBatch(ctx, []int64{accountID}, map[int64]time.Duration{accountID: idleTimeout})
	require.NoError(t, err)
	require.Equal(t, 0, counts[accountID])
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), sessionID, false)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), sessionID, true)
	ownerID, err := cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, accountID, ownerID)

	allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), sessionID, true)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), sessionID, false)
}

func TestOpenAISessionLimitRequestAdmissionMovesExpiredSlotsToStaging(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 18
	const expiredSessionID = "expired-on-next-request"
	const idleTimeout = 10 * time.Second

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, expiredSessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 11),
		Member: expiredSessionID,
	}).Err())

	allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, "incoming-session", 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), expiredSessionID, false)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), expiredSessionID, true)
	stagedAccountID, err := cache.GetOpenAIStagedSessionAccountID(ctx, expiredSessionID)
	require.NoError(t, err)
	require.Equal(t, accountID, stagedAccountID)
}

func TestOpenAISessionLimitStagedLookupOnlyMatchesStagingWindow(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 14
	const sessionID = "windowed-session"

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, 10*time.Second)
	require.NoError(t, err)
	require.True(t, allowed)
	stagedAccountID, err := cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
	require.NoError(t, err)
	require.Zero(t, stagedAccountID, "an active slot must not override normal scheduling")

	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	now := redisTime.Unix()
	require.NoError(t, client.Set(ctx, openAISessionPreferredKey(sessionID), fmt.Sprintf("%d:%d:%d", accountID, now-1, now+9), time.Minute).Err())
	stagedAccountID, err = cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, accountID, stagedAccountID)

	require.NoError(t, client.Set(ctx, openAISessionPreferredKey(sessionID), fmt.Sprintf("%d:%d:%d", accountID, now-11, now), time.Minute).Err())
	stagedAccountID, err = cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
	require.NoError(t, err)
	require.Zero(t, stagedAccountID)
	require.False(t, redisServer.Exists(openAISessionPreferredKey(sessionID)))
}

func TestOpenAISessionLimitFullAccountDropsStagedReservation(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 9
	const sessionID = "staged-session"
	const idleTimeout = 10 * time.Second

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 11),
		Member: sessionID,
	}).Err())
	_, err = cache.GetOpenAIActiveSessionCountBatch(ctx, []int64{accountID}, map[int64]time.Duration{accountID: idleTimeout})
	require.NoError(t, err)

	for _, activeSessionID := range []string{"active-a", "active-b", "active-c"} {
		allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, activeSessionID, 3, idleTimeout)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.False(t, allowed)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), sessionID, false)
	ownerID, err := cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
	require.NoError(t, err)
	require.Zero(t, ownerID)

	allowed, err = cache.RegisterOpenAISessionID(ctx, 10, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(10), sessionID, true)
}

func TestOpenAISessionLimitRotationPromotesStagedSessionWhenAccountIsFull(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 15
	const sessionID = "staged-rotation-session"
	const idleTimeout = 10 * time.Second

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 11),
		Member: sessionID,
	}).Err())
	_, err = cache.GetOpenAIActiveSessionCountBatch(ctx, []int64{accountID}, map[int64]time.Duration{accountID: idleTimeout})
	require.NoError(t, err)

	for _, activeSessionID := range []string{"active-a", "active-b", "active-c"} {
		allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, activeSessionID, 3, idleTimeout)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	redisTime, err = client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 3),
		Member: "active-a",
	}).Err())

	allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout, true)
	require.NoError(t, err)
	require.True(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), "active-a", false)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), "active-a", true)
	requireZMember(t, ctx, client, openAISessionLimitKey(accountID), sessionID, true)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), sessionID, false)
	stagedAccountID, err := cache.GetOpenAIStagedSessionAccountID(ctx, "active-a")
	require.NoError(t, err)
	require.Equal(t, accountID, stagedAccountID)
	owner, err := client.Get(ctx, openAISessionOwnerKey(sessionID)).Result()
	require.NoError(t, err)
	require.Equal(t, strconv.FormatInt(accountID, 10), owner)
	count, err := client.ZCard(ctx, openAISessionLimitKey(accountID)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
}

func TestOpenAISessionLimitStagingExpiresAndClearRemovesIt(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()
	const accountID int64 = 13
	const sessionID = "staging-expiry-session"
	const idleTimeout = 10 * time.Second

	allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 11),
		Member: sessionID,
	}).Err())
	_, err = cache.GetOpenAIActiveSessionCountBatch(ctx, []int64{accountID}, map[int64]time.Duration{accountID: idleTimeout})
	require.NoError(t, err)
	require.NoError(t, cache.ClearOpenAISessions(ctx, accountID))
	require.False(t, redisServer.Exists(openAISessionStagingKey(accountID)))
	require.False(t, redisServer.Exists(openAISessionOwnerKey(sessionID)))

	allowed, err = cache.RegisterOpenAISessionID(ctx, accountID, sessionID, 3, idleTimeout)
	require.NoError(t, err)
	require.True(t, allowed)
	redisTime, err = client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZRem(ctx, openAISessionLimitKey(accountID), sessionID).Err())
	require.NoError(t, client.ZAdd(ctx, openAISessionStagingKey(accountID), redis.Z{
		Score:  float64(redisTime.Unix() - 11),
		Member: sessionID,
	}).Err())
	_, err = cache.GetOpenAIActiveSessionCountBatch(ctx, []int64{accountID}, map[int64]time.Duration{accountID: idleTimeout})
	require.NoError(t, err)
	requireZMember(t, ctx, client, openAISessionStagingKey(accountID), sessionID, false)
	ownerID, err := cache.GetOpenAIStagedSessionAccountID(ctx, sessionID)
	require.NoError(t, err)
	require.Zero(t, ownerID)
}

func TestOpenAISessionLimitMovesSessionBetweenAccountsWithoutDuplicate(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()

	allowed, err := cache.RegisterOpenAISessionID(ctx, 11, "shared-session", 3, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)
	// The scheduling preference expires before the longer-lived ownership
	// locator. A later move must still remove the old physical membership.
	require.NoError(t, client.Del(ctx, openAISessionPreferredKey("shared-session")).Err())
	allowed, err = cache.RegisterOpenAISessionID(ctx, 12, "shared-session", 3, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)

	count, err := client.ZCard(ctx, openAISessionLimitKey(11)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
	count, err = client.ZCard(ctx, openAISessionLimitKey(12)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	owner, err := client.Get(ctx, openAISessionOwnerKey("shared-session")).Result()
	require.NoError(t, err)
	require.Equal(t, "12", owner)
}

func TestOpenAISessionLimitFullTargetDoesNotDropExistingOwner(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()

	allowed, err := cache.RegisterOpenAISessionID(ctx, 21, "owned-session", 3, time.Minute)
	require.NoError(t, err)
	require.True(t, allowed)
	for _, sessionID := range []string{"target-a", "target-b", "target-c"} {
		allowed, err = cache.RegisterOpenAISessionID(ctx, 22, sessionID, 3, time.Minute)
		require.NoError(t, err)
		require.True(t, allowed)
	}
	redisTime, err := client.Time(ctx).Result()
	require.NoError(t, err)
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(22), redis.Z{
		Score:  float64(redisTime.Unix() - 1),
		Member: "target-a",
	}).Err())

	allowed, err = cache.RegisterOpenAISessionID(ctx, 22, "owned-session", 3, time.Minute)
	require.NoError(t, err)
	require.False(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(21), "owned-session", true)
	requireZMember(t, ctx, client, openAISessionLimitKey(22), "owned-session", false)
	owner, err := client.Get(ctx, openAISessionOwnerKey("owned-session")).Result()
	require.NoError(t, err)
	require.Equal(t, "21", owner)

	allowed, err = cache.RegisterOpenAISessionID(ctx, 22, "owned-session", 3, time.Minute, true)
	require.NoError(t, err)
	require.True(t, allowed)
	requireZMember(t, ctx, client, openAISessionLimitKey(21), "owned-session", false)
	requireZMember(t, ctx, client, openAISessionLimitKey(22), "owned-session", true)
	requireZMember(t, ctx, client, openAISessionLimitKey(22), "target-a", false)
	requireZMember(t, ctx, client, openAISessionStagingKey(22), "target-a", true)
	stagedAccountID, err := cache.GetOpenAIStagedSessionAccountID(ctx, "target-a")
	require.NoError(t, err)
	require.Equal(t, int64(22), stagedAccountID)
	owner, err = client.Get(ctx, openAISessionOwnerKey("owned-session")).Result()
	require.NoError(t, err)
	require.Equal(t, "22", owner)
	targetCount, err := client.ZCard(ctx, openAISessionLimitKey(22)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 3, targetCount)
}

func TestOpenAISessionLimitConcurrentMovesKeepOneMembership(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := NewSessionLimitCache(client, 5)
	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for index := 0; index < 32; index++ {
		wg.Add(1)
		go func(accountID int64) {
			defer wg.Done()
			allowed, err := cache.RegisterOpenAISessionID(ctx, accountID, "moving-session", 3, time.Minute)
			if err != nil {
				errCh <- err
				return
			}
			if !allowed {
				errCh <- fmt.Errorf("account %d unexpectedly rejected the moving session", accountID)
			}
		}(int64(31 + index%2))
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	firstCount, err := client.ZCard(ctx, openAISessionLimitKey(31)).Result()
	require.NoError(t, err)
	secondCount, err := client.ZCard(ctx, openAISessionLimitKey(32)).Result()
	require.NoError(t, err)
	totalMemberships := firstCount + secondCount
	require.EqualValues(t, 1, totalMemberships)
	owner, err := client.Get(ctx, openAISessionOwnerKey("moving-session")).Result()
	require.NoError(t, err)
	require.Contains(t, []string{"31", "32"}, owner)
	requireZMember(t, ctx, client, openAISessionLimitKey(mustParseAccountID(t, owner)), "moving-session", true)
}

func TestOpenAISessionLimitStartupReconcilesLegacyDuplicates(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	ctx := context.Background()
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(41), redis.Z{Score: 100, Member: "legacy-session"}).Err())
	require.NoError(t, client.ZAdd(ctx, openAISessionLimitKey(42), redis.Z{Score: 200, Member: "legacy-session"}).Err())
	require.NoError(t, client.Expire(ctx, openAISessionLimitKey(41), time.Hour).Err())
	require.NoError(t, client.Expire(ctx, openAISessionLimitKey(42), time.Hour).Err())

	_ = NewSessionLimitCache(client, 5)

	requireZMember(t, ctx, client, openAISessionLimitKey(41), "legacy-session", false)
	requireZMember(t, ctx, client, openAISessionLimitKey(42), "legacy-session", true)
	owner, err := client.Get(ctx, openAISessionOwnerKey("legacy-session")).Result()
	require.NoError(t, err)
	require.Equal(t, "42", owner)
}

func mustParseAccountID(t *testing.T, raw string) int64 {
	t.Helper()
	accountID, err := strconv.ParseInt(raw, 10, 64)
	require.NoError(t, err)
	return accountID
}

func requireZMember(t *testing.T, ctx context.Context, client *redis.Client, key, member string, expected bool) {
	t.Helper()
	_, err := client.ZScore(ctx, key, member).Result()
	if expected {
		require.NoError(t, err)
		return
	}
	require.ErrorIs(t, err, redis.Nil)
}
