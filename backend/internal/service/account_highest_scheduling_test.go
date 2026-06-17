package service

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAccountHighestSchedulingModeDefaultDisabled(t *testing.T) {
	account := &Account{}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	if account.IsHighestSchedulingModeConfigured() {
		t.Fatalf("expected highest scheduling mode to be disabled by default")
	}
	if account.IsHighestSchedulingModeEffective(now) {
		t.Fatalf("expected highest scheduling mode to be ineffective by default")
	}
}

func TestAccountHighestSchedulingModeEffectiveWhenEnabled(t *testing.T) {
	account := &Account{Extra: map[string]any{"highest_scheduling_mode": true}}
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)

	if !account.IsHighestSchedulingModeConfigured() {
		t.Fatalf("expected highest scheduling mode to be configured")
	}
	if !account.IsHighestSchedulingModeEffective(now) {
		t.Fatalf("expected highest scheduling mode to be effective")
	}
}

func TestAccountHighestSchedulingModeSuppressedUntilFuture(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	account := &Account{Extra: map[string]any{
		"highest_scheduling_mode":             true,
		"highest_scheduling_suppressed_until": now.Add(10 * time.Minute).Format(time.RFC3339),
	}}

	if !account.IsHighestSchedulingModeSuppressed(now) {
		t.Fatalf("expected future suppression timestamp to suppress highest scheduling mode")
	}
	if account.IsHighestSchedulingModeEffective(now) {
		t.Fatalf("expected future suppression timestamp to make highest scheduling ineffective")
	}
}

func TestAccountHighestSchedulingModeSuppressedUntilPastResumes(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	account := &Account{Extra: map[string]any{
		"highest_scheduling_mode":             true,
		"highest_scheduling_suppressed_until": now.Add(-10 * time.Minute).Format(time.RFC3339),
	}}

	if account.IsHighestSchedulingModeSuppressed(now) {
		t.Fatalf("expected past suppression timestamp not to suppress highest scheduling mode")
	}
	if !account.IsHighestSchedulingModeEffective(now) {
		t.Fatalf("expected past suppression timestamp to let highest scheduling resume")
	}
}

func TestAccountHighestSchedulingModeManualSuppression(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	account := &Account{Extra: map[string]any{
		"highest_scheduling_mode":       true,
		"highest_scheduling_suppressed": true,
	}}

	if !account.IsHighestSchedulingModeSuppressed(now) {
		t.Fatalf("expected manual suppression flag to suppress highest scheduling mode")
	}
	if account.IsHighestSchedulingModeEffective(now) {
		t.Fatalf("expected manual suppression flag to make highest scheduling ineffective")
	}
}

func TestAccountHighestSchedulingRecoveryMinutesParsing(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int
	}{
		{name: "int", value: 15, want: 15},
		{name: "int64", value: int64(16), want: 16},
		{name: "float64", value: float64(17), want: 17},
		{name: "json number", value: json.Number("18"), want: 18},
		{name: "string", value: "19", want: 19},
		{name: "negative", value: -1, want: 0},
		{name: "invalid", value: "soon", want: 0},
		{name: "missing", value: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extra := map[string]any{}
			if tt.value != nil {
				extra["highest_scheduling_recovery_minutes"] = tt.value
			}
			account := &Account{Extra: extra}
			if got := account.GetHighestSchedulingRecoveryMinutes(); got != tt.want {
				t.Fatalf("GetHighestSchedulingRecoveryMinutes() = %d, want %d", got, tt.want)
			}
		})
	}
}
