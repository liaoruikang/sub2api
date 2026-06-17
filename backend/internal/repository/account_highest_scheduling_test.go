package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestBuildHighestSchedulingSuppressionExtraUpdates_TemporaryRecovery(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	account := &service.Account{Extra: map[string]any{
		service.AccountExtraHighestSchedulingMode:            true,
		service.AccountExtraHighestSchedulingRecoveryMinutes: 15,
	}}

	updates, deleteKeys, ok := buildHighestSchedulingSuppressionExtraUpdates(account, "temporary upstream error", now)
	if !ok {
		t.Fatalf("expected highest scheduling account to produce suppression updates")
	}
	if got := updates[service.AccountExtraHighestSchedulingSuppressed]; got != false {
		t.Fatalf("suppressed = %v, want false for timed recovery", got)
	}
	wantUntil := now.Add(15 * time.Minute).Format(time.RFC3339)
	if got := updates[service.AccountExtraHighestSchedulingSuppressedUntil]; got != wantUntil {
		t.Fatalf("suppressed_until = %v, want %s", got, wantUntil)
	}
	if got := updates[service.AccountExtraHighestSchedulingSuppressedAt]; got != now.Format(time.RFC3339) {
		t.Fatalf("suppressed_at = %v, want %s", got, now.Format(time.RFC3339))
	}
	if got := updates[service.AccountExtraHighestSchedulingSuppressedReason]; got != "temporary upstream error" {
		t.Fatalf("suppressed_reason = %v", got)
	}
	if len(deleteKeys) != 0 {
		t.Fatalf("deleteKeys = %v, want none for timed recovery", deleteKeys)
	}
}

func TestBuildHighestSchedulingSuppressionExtraUpdates_ManualRecoveryDeletesTimedSuppression(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	account := &service.Account{Extra: map[string]any{
		service.AccountExtraHighestSchedulingMode: true,
	}}

	updates, deleteKeys, ok := buildHighestSchedulingSuppressionExtraUpdates(account, "manual restore required", now)
	if !ok {
		t.Fatalf("expected highest scheduling account to produce suppression updates")
	}
	if got := updates[service.AccountExtraHighestSchedulingSuppressed]; got != true {
		t.Fatalf("suppressed = %v, want true for manual recovery", got)
	}
	if _, exists := updates[service.AccountExtraHighestSchedulingSuppressedUntil]; exists {
		t.Fatalf("manual recovery should not write suppressed_until update: %#v", updates)
	}
	if len(deleteKeys) != 1 || deleteKeys[0] != service.AccountExtraHighestSchedulingSuppressedUntil {
		t.Fatalf("deleteKeys = %v, want suppressed_until", deleteKeys)
	}
}

func TestBuildHighestSchedulingSuppressionExtraUpdates_NonHighestNoop(t *testing.T) {
	updates, deleteKeys, ok := buildHighestSchedulingSuppressionExtraUpdates(&service.Account{Extra: map[string]any{}}, "error", time.Now())
	if ok || updates != nil || deleteKeys != nil {
		t.Fatalf("expected non-highest account to produce no updates, got updates=%v deleteKeys=%v ok=%v", updates, deleteKeys, ok)
	}
}

func TestBuildHighestSchedulingSuppressionExtraUpdates_TruncatesReason(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	account := &service.Account{Extra: map[string]any{service.AccountExtraHighestSchedulingMode: true}}

	updates, _, ok := buildHighestSchedulingSuppressionExtraUpdates(account, strings.Repeat("x", highestSchedulingSuppressionReasonMaxLen+20), now)
	if !ok {
		t.Fatalf("expected highest scheduling account to produce suppression updates")
	}
	reason, _ := updates[service.AccountExtraHighestSchedulingSuppressedReason].(string)
	if len([]rune(reason)) != highestSchedulingSuppressionReasonMaxLen {
		t.Fatalf("reason rune length = %d, want %d", len([]rune(reason)), highestSchedulingSuppressionReasonMaxLen)
	}
}
