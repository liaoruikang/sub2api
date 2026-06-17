package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestSanitizeAccountHighestSchedulingExtra_CreateNormalizesAndDropsSuppression(t *testing.T) {
	extra := map[string]any{
		service.AccountExtraHighestSchedulingMode:             true,
		service.AccountExtraHighestSchedulingRecoveryMinutes:  "10081",
		service.AccountExtraHighestSchedulingSuppressed:       true,
		service.AccountExtraHighestSchedulingSuppressedUntil:  "2026-06-09T12:15:00Z",
		service.AccountExtraHighestSchedulingSuppressedAt:     "2026-06-09T12:00:00Z",
		service.AccountExtraHighestSchedulingSuppressedReason: "boom",
		"unrelated": 1,
	}

	sanitizeAccountHighestSchedulingExtra(extra, true)

	if got := extra[service.AccountExtraHighestSchedulingMode]; got != true {
		t.Fatalf("mode = %v, want true", got)
	}
	if got := extra[service.AccountExtraHighestSchedulingRecoveryMinutes]; got != 10080 {
		t.Fatalf("recovery minutes = %v, want 10080", got)
	}
	for _, key := range []string{
		service.AccountExtraHighestSchedulingSuppressed,
		service.AccountExtraHighestSchedulingSuppressedUntil,
		service.AccountExtraHighestSchedulingSuppressedAt,
		service.AccountExtraHighestSchedulingSuppressedReason,
	} {
		if _, exists := extra[key]; exists {
			t.Fatalf("create should drop suppression key %q from extra: %#v", key, extra)
		}
	}
	if got := extra["unrelated"]; got != 1 {
		t.Fatalf("unrelated key = %v, want preserved", got)
	}
}

func TestSanitizeAccountHighestSchedulingExtra_InvalidModeAndRecovery(t *testing.T) {
	extra := map[string]any{
		service.AccountExtraHighestSchedulingMode:            "true",
		service.AccountExtraHighestSchedulingRecoveryMinutes: -5,
	}

	sanitizeAccountHighestSchedulingExtra(extra, false)

	if _, exists := extra[service.AccountExtraHighestSchedulingMode]; exists {
		t.Fatalf("invalid non-boolean highest scheduling mode should be removed: %#v", extra)
	}
	if got := extra[service.AccountExtraHighestSchedulingRecoveryMinutes]; got != 0 {
		t.Fatalf("negative recovery minutes = %v, want 0", got)
	}
}

func TestSanitizeAccountHighestSchedulingExtra_UpdatePreservesSuppressionKeys(t *testing.T) {
	extra := map[string]any{
		service.AccountExtraHighestSchedulingMode:             true,
		service.AccountExtraHighestSchedulingSuppressed:       true,
		service.AccountExtraHighestSchedulingSuppressedReason: "still paused",
	}

	sanitizeAccountHighestSchedulingExtra(extra, false)

	if got := extra[service.AccountExtraHighestSchedulingSuppressed]; got != true {
		t.Fatalf("update should preserve suppression flag, got %v", got)
	}
	if got := extra[service.AccountExtraHighestSchedulingSuppressedReason]; got != "still paused" {
		t.Fatalf("update should preserve suppression reason, got %v", got)
	}
}
