//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type billingRPMCacheStub struct {
	UserRPMCache
	groupCount int
	userCount  int
}

func (s *billingRPMCacheStub) IncrementUserGroupRPM(context.Context, int64, int64) (int, error) {
	s.groupCount++
	return s.groupCount, nil
}

func (s *billingRPMCacheStub) IncrementUserRPM(context.Context, int64) (int, error) {
	s.userCount++
	return s.userCount, nil
}

type billingRPMRateRepoStub struct {
	UserGroupRateRepository
	regularOverride     *int
	limitedTimeOverride *int
}

func (s *billingRPMRateRepoStub) GetRPMOverrideByUserAndGroup(context.Context, int64, int64) (*int, error) {
	return s.regularOverride, nil
}

func (s *billingRPMRateRepoStub) GetLimitedTimeRPMOverrideByUserAndGroup(context.Context, int64, int64) (*int, error) {
	return s.limitedTimeOverride, nil
}

func activeLimitedTimeRPMGroup(limit, regularLimit int) *Group {
	return &Group{
		ID:                                   7,
		SubscriptionType:                     SubscriptionTypeStandard,
		LimitedTimeMultiplierEnabled:         true,
		LimitedTimeMultiplierCron:            "* * * * *",
		LimitedTimeMultiplierDurationMinutes: 1,
		LimitedTimeMultiplierValue:           0.5,
		LimitedTimeRPMLimit:                  limit,
		RPMLimit:                             regularLimit,
	}
}

func TestBillingCacheCheckRPM_UsesLimitedTimeGroupLimitBeforeRegularGroupLimit(t *testing.T) {
	rpmCache := &billingRPMCacheStub{}
	svc := &BillingCacheService{userRPMCache: rpmCache}

	err := svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(2, 100))
	require.NoError(t, err)
	require.NoError(t, svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(2, 100)))
	err = svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(2, 100))
	require.ErrorIs(t, err, ErrGroupRPMExceeded)
	require.Equal(t, 3, rpmCache.groupCount)
}

func TestBillingCacheCheckRPM_LimitedTimeOverrideBeatsGroupLimit(t *testing.T) {
	limitedOverride := 1
	rpmCache := &billingRPMCacheStub{}
	svc := &BillingCacheService{
		userRPMCache:      rpmCache,
		userGroupRateRepo: &billingRPMRateRepoStub{limitedTimeOverride: &limitedOverride},
	}

	require.NoError(t, svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(10, 100)))
	err := svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(10, 100))
	require.ErrorIs(t, err, ErrGroupRPMExceeded)
	require.Equal(t, 2, rpmCache.groupCount)
}

func TestBillingCacheCheckRPM_ZeroLimitedTimeGroupLimitFallsBackToRegularGroupLimit(t *testing.T) {
	rpmCache := &billingRPMCacheStub{}
	svc := &BillingCacheService{userRPMCache: rpmCache}

	require.NoError(t, svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(0, 2)))
	require.NoError(t, svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(0, 2)))
	err := svc.checkRPM(context.Background(), &User{ID: 42}, activeLimitedTimeRPMGroup(0, 2))
	require.ErrorIs(t, err, ErrGroupRPMExceeded)
	require.Equal(t, 3, rpmCache.groupCount)
}

func TestBillingCacheCheckRPM_LimitedTimeGroupLimitDoesNotBypassUserLimit(t *testing.T) {
	rpmCache := &billingRPMCacheStub{}
	svc := &BillingCacheService{userRPMCache: rpmCache}

	require.NoError(t, svc.checkRPM(context.Background(), &User{ID: 42, RPMLimit: 1}, activeLimitedTimeRPMGroup(10, 100)))
	err := svc.checkRPM(context.Background(), &User{ID: 42, RPMLimit: 1}, activeLimitedTimeRPMGroup(10, 100))
	require.ErrorIs(t, err, ErrUserRPMExceeded)
	require.Equal(t, 2, rpmCache.groupCount)
	require.Equal(t, 2, rpmCache.userCount)
}
