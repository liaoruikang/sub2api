package service

import (
	"testing"
	"time"
)

func highestSchedulingTestAccount(id int64, priority int, lastUsed *time.Time, mode any) *Account {
	extra := map[string]any{}
	if mode != nil {
		extra[AccountExtraHighestSchedulingMode] = mode
	}
	return &Account{
		ID:          id,
		Priority:    priority,
		LastUsedAt:  lastUsed,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Extra:       extra,
	}
}

func TestIsBetterAccountByHighestSchedulingPriorityAndLastUsed_HighestBeatsPriority(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	highest := highestSchedulingTestAccount(2, 99, nil, true)

	if !isBetterAccountByHighestSchedulingPriorityAndLastUsed(highest, normal, false) {
		t.Fatalf("expected highest scheduling account to beat normal account even with lower normal priority")
	}
	if isBetterAccountByHighestSchedulingPriorityAndLastUsed(normal, highest, false) {
		t.Fatalf("expected normal account not to beat highest scheduling account")
	}
}

func TestIsBetterAccountByHighestSchedulingPriorityAndLastUsed_DeprecatedSuppressionDoesNotDisableMode(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	highest.Extra["highest_scheduling_suppressed"] = true
	highest.Extra["highest_scheduling_suppressed_until"] = time.Now().Add(time.Hour).Format(time.RFC3339)

	if !isBetterAccountByHighestSchedulingPriorityAndLastUsed(highest, normal, false) {
		t.Fatalf("deprecated suppression keys must not disable a boolean true mode")
	}
}

func TestIsBetterAccountByHighestSchedulingPriorityAndLastUsed_NonBooleanModeIsNormal(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	invalidMode := highestSchedulingTestAccount(2, 99, nil, "true")

	if isBetterAccountByHighestSchedulingPriorityAndLastUsed(invalidMode, normal, false) {
		t.Fatalf("non-boolean mode must fall back to normal priority ordering")
	}
}

func TestSortAccountsByHighestSchedulingPriorityAndLastUsed_PreservesOrderingInsideHighestSubset(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	later := now.Add(-5 * time.Minute)
	earlier := now.Add(-30 * time.Minute)
	accounts := []*Account{
		highestSchedulingTestAccount(1, 1, &later, nil),
		highestSchedulingTestAccount(2, 5, &later, true),
		highestSchedulingTestAccount(3, 1, &later, true),
		highestSchedulingTestAccount(4, 1, &earlier, true),
	}

	sortAccountsByHighestSchedulingPriorityAndLastUsed(accounts, false)

	want := []int64{4, 3, 2, 1}
	for i, id := range want {
		if accounts[i].ID != id {
			t.Fatalf("accounts[%d].ID = %d, want %d; order=%v", i, accounts[i].ID, id, []int64{accounts[0].ID, accounts[1].ID, accounts[2].ID, accounts[3].ID})
		}
	}
}

func TestSortAccountsByPriorityOnly_HighestSchedulingBeatsRandomFallbackPriority(t *testing.T) {
	accounts := []*Account{
		highestSchedulingTestAccount(1, 1, nil, nil),
		highestSchedulingTestAccount(2, 99, nil, true),
		highestSchedulingTestAccount(3, 1, nil, nil),
	}

	sortAccountsByPriorityOnly(accounts, false)

	if accounts[0].ID != 2 {
		t.Fatalf("highest scheduling account must stay first in random fallback order, got order=%v", []int64{accounts[0].ID, accounts[1].ID, accounts[2].ID})
	}
}

func TestHighestSchedulingLoadTier(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	invalidMode := highestSchedulingTestAccount(3, 1, nil, "true")
	candidates := []accountWithLoad{
		{account: normal, loadInfo: &AccountLoadInfo{AccountID: normal.ID, LoadRate: 0}},
		{account: highest, loadInfo: &AccountLoadInfo{AccountID: highest.ID, LoadRate: 0}},
		{account: invalidMode, loadInfo: &AccountLoadInfo{AccountID: invalidMode.ID, LoadRate: 0}},
	}

	filtered := highestSchedulingLoadTier(candidates)
	if len(filtered) != 1 || filtered[0].account.ID != highest.ID {
		t.Fatalf("expected only strict-boolean highest tier, got %#v", filtered)
	}
}

func TestSortByHighestSchedulingLoadCandidates(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	invalidMode := highestSchedulingTestAccount(3, 1, nil, "true")
	candidates := []accountWithLoad{
		{account: normal, loadInfo: &AccountLoadInfo{AccountID: normal.ID, LoadRate: 0}},
		{account: highest, loadInfo: &AccountLoadInfo{AccountID: highest.ID, LoadRate: 0}},
		{account: invalidMode, loadInfo: &AccountLoadInfo{AccountID: invalidMode.ID, LoadRate: 0}},
	}

	sortByHighestSchedulingLoadCandidates(candidates, false)
	want := []int64{highest.ID, normal.ID, invalidMode.ID}
	for i, id := range want {
		if candidates[i].account.ID != id {
			t.Fatalf("candidates[%d].ID = %d, want %d", i, candidates[i].account.ID, id)
		}
	}
}

func TestBuildOpenAISelectionOrder_DeprecatedSuppressionDoesNotDisableMode(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	highest.Extra["highest_scheduling_suppressed"] = true
	scheduler := &defaultOpenAIAccountScheduler{}
	plan := openAIAccountLoadPlan{
		topK: 2,
		candidates: []openAIAccountCandidateScore{
			{account: normal, score: 100},
			{account: highest, score: 1},
		},
	}

	ordered := scheduler.buildOpenAISelectionOrder(OpenAIAccountScheduleRequest{}, plan)

	if len(ordered) != 2 || ordered[0].account.ID != highest.ID {
		t.Fatalf("deprecated suppression must not demote true mode; got order=%v", []int64{ordered[0].account.ID, ordered[1].account.ID})
	}
}

func TestSortOpenAICompactRetryCandidates_HighestBeatsPriorityWithinStaleTier(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, nil)
	normal.Platform = PlatformOpenAI
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	highest.Platform = PlatformOpenAI
	pool := []openAIAccountCandidateScore{
		{account: normal, loadInfo: &AccountLoadInfo{AccountID: normal.ID, LoadRate: 0}},
		{account: highest, loadInfo: &AccountLoadInfo{AccountID: highest.ID, LoadRate: 0}},
	}

	ordered := sortOpenAICompactRetryCandidates(pool)
	if len(ordered) != 2 || ordered[0].account.ID != highest.ID {
		t.Fatalf("expected highest stale compact retry candidate first, got order=%v", []int64{ordered[0].account.ID, ordered[1].account.ID})
	}
}
