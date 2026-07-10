package service

import "testing"

func TestAccountHighestSchedulingModeUsesStrictBooleanSemantics(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "missing", value: nil, want: false},
		{name: "true", value: true, want: true},
		{name: "false", value: false, want: false},
		{name: "string true", value: "true", want: false},
		{name: "number one", value: 1, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extra := map[string]any{}
			if tt.value != nil {
				extra[AccountExtraHighestSchedulingMode] = tt.value
			}
			account := &Account{Extra: extra}
			if got := account.IsHighestSchedulingModeConfigured(); got != tt.want {
				t.Fatalf("IsHighestSchedulingModeConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeAccountHighestSchedulingExtraRemovesDeprecatedKeys(t *testing.T) {
	extra := map[string]any{
		AccountExtraHighestSchedulingMode:      false,
		"highest_scheduling_recovery_minutes":  15,
		"highest_scheduling_suppressed":        true,
		"highest_scheduling_suppressed_until":  "2026-06-09T12:15:00Z",
		"highest_scheduling_suppressed_at":     "2026-06-09T12:00:00Z",
		"highest_scheduling_suppressed_reason": "boom",
		"unrelated":                            1,
	}

	sanitized := SanitizeAccountHighestSchedulingExtra(extra)

	if got, exists := sanitized[AccountExtraHighestSchedulingMode]; !exists || got != false {
		t.Fatalf("valid false mode = %v (exists=%v), want preserved", got, exists)
	}
	for _, key := range []string{
		"highest_scheduling_recovery_minutes",
		"highest_scheduling_suppressed",
		"highest_scheduling_suppressed_until",
		"highest_scheduling_suppressed_at",
		"highest_scheduling_suppressed_reason",
	} {
		if _, exists := sanitized[key]; exists {
			t.Fatalf("deprecated key %q was not removed: %#v", key, sanitized)
		}
		if _, exists := extra[key]; !exists {
			t.Fatalf("input key %q was modified: %#v", key, extra)
		}
	}
	if got := sanitized["unrelated"]; got != 1 {
		t.Fatalf("unrelated key = %v, want 1", got)
	}
}

func TestSanitizeAccountHighestSchedulingExtraRemovesNonBooleanMode(t *testing.T) {
	extra := map[string]any{
		AccountExtraHighestSchedulingMode: "true",
		"unrelated":                       "keep",
	}

	sanitized := SanitizeAccountHighestSchedulingExtra(extra)

	if _, exists := sanitized[AccountExtraHighestSchedulingMode]; exists {
		t.Fatalf("non-boolean mode should be removed: %#v", sanitized)
	}
	if got := sanitized["unrelated"]; got != "keep" {
		t.Fatalf("unrelated key = %v, want keep", got)
	}
	if got := extra[AccountExtraHighestSchedulingMode]; got != "true" {
		t.Fatalf("input mode = %v, want unchanged", got)
	}
}
