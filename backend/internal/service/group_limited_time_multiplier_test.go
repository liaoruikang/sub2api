//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGroupBillingRateMultiplierAt_LimitedTimeWindow(t *testing.T) {
	group := &Group{
		RateMultiplier:                       1.2,
		SubscriptionType:                     SubscriptionTypeStandard,
		LimitedTimeMultiplierEnabled:         true,
		LimitedTimeMultiplierCron:            "0 9 * * *",
		LimitedTimeMultiplierDurationMinutes: 120,
		LimitedTimeMultiplierValue:           0.5,
	}

	activeAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)
	inactiveAt := time.Date(2026, 5, 30, 12, 1, 0, 0, time.Local)

	require.Equal(t, 0.5, group.BillingRateMultiplierAt(activeAt))
	require.Equal(t, 1.2, group.BillingRateMultiplierAt(inactiveAt))
}

func TestGroupBillingRateMultiplierForBaseAt_LimitedTimeWindowCapsBase(t *testing.T) {
	group := &Group{
		RateMultiplier:                       1.2,
		SubscriptionType:                     SubscriptionTypeStandard,
		LimitedTimeMultiplierEnabled:         true,
		LimitedTimeMultiplierCron:            "0 9 * * *",
		LimitedTimeMultiplierDurationMinutes: 120,
		LimitedTimeMultiplierValue:           0.5,
	}

	activeAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)
	inactiveAt := time.Date(2026, 5, 30, 12, 1, 0, 0, time.Local)

	require.Equal(t, 0.5, group.BillingRateMultiplierForBaseAt(1.8, activeAt))
	require.Equal(t, 0.4, group.BillingRateMultiplierForBaseAt(0.4, activeAt))
	require.Equal(t, 1.8, group.BillingRateMultiplierForBaseAt(1.8, inactiveAt))
}

func TestGroupBillingRateMultiplierAt_SubscriptionIgnoresLimitedTimeWindow(t *testing.T) {
	group := &Group{
		RateMultiplier:                       1.2,
		SubscriptionType:                     SubscriptionTypeSubscription,
		LimitedTimeMultiplierEnabled:         true,
		LimitedTimeMultiplierCron:            "0 9 * * *",
		LimitedTimeMultiplierDurationMinutes: 120,
		LimitedTimeMultiplierValue:           0.5,
	}

	activeAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)

	require.Equal(t, 1.2, group.BillingRateMultiplierAt(activeAt))
	require.Equal(t, 1.8, group.BillingRateMultiplierForBaseAt(1.8, activeAt))
}
