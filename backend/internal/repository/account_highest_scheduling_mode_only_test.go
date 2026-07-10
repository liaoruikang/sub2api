package repository

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func repositoryModeOnlyTestExtra() map[string]any {
	return map[string]any{
		service.AccountExtraHighestSchedulingMode: false,
		"highest_scheduling_recovery_minutes":     15,
		"highest_scheduling_suppressed":           true,
		"highest_scheduling_suppressed_until":     "2026-06-09T12:15:00Z",
		"highest_scheduling_suppressed_at":        "2026-06-09T12:00:00Z",
		"highest_scheduling_suppressed_reason":    "boom",
		"unrelated":                               1,
	}
}

func requireRepositoryModeOnlyExtra(t *testing.T, extra map[string]any) {
	t.Helper()
	require.Equal(t, false, extra[service.AccountExtraHighestSchedulingMode])
	require.Equal(t, 1, extra["unrelated"])
	for _, key := range []string{
		"highest_scheduling_recovery_minutes",
		"highest_scheduling_suppressed",
		"highest_scheduling_suppressed_until",
		"highest_scheduling_suppressed_at",
		"highest_scheduling_suppressed_reason",
	} {
		require.NotContains(t, extra, key)
	}
}

func TestSanitizeAccountExtraForPersistenceDefendsCreateAndUpdate(t *testing.T) {
	extra := repositoryModeOnlyTestExtra()

	sanitized := sanitizeAccountExtraForPersistence(extra)

	requireRepositoryModeOnlyExtra(t, sanitized)
	require.Contains(t, extra, "highest_scheduling_recovery_minutes")
	require.Contains(t, extra, "highest_scheduling_suppressed")
	require.Equal(t, false, extra[service.AccountExtraHighestSchedulingMode])
	require.Equal(t, 1, extra["unrelated"])
}

func TestSanitizeAccountExtraForPersistenceRemovesInvalidMode(t *testing.T) {
	extra := map[string]any{
		service.AccountExtraHighestSchedulingMode: "true",
		"unrelated": "keep",
	}

	sanitized := sanitizeAccountExtraForPersistence(extra)

	require.NotContains(t, sanitized, service.AccountExtraHighestSchedulingMode)
	require.Equal(t, "keep", sanitized["unrelated"])
}

func TestSanitizeAccountBulkUpdateDefendsExtra(t *testing.T) {
	updates := service.AccountBulkUpdate{Extra: repositoryModeOnlyTestExtra()}

	sanitizeAccountBulkUpdateForPersistence(&updates)

	requireRepositoryModeOnlyExtra(t, updates.Extra)
}
