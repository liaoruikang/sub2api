package service

import (
	"testing"
	"time"
)

func highestSchedulingTestAccount(id int64, priority int, lastUsed *time.Time, highest bool) *Account {
	extra := map[string]any{}
	if highest {
		extra[AccountExtraHighestSchedulingMode] = true
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
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	normal := highestSchedulingTestAccount(1, 1, nil, false)
	highest := highestSchedulingTestAccount(2, 99, nil, true)

	if !isBetterAccountByHighestSchedulingPriorityAndLastUsed(highest, normal, false, now) {
		t.Fatalf("expected highest scheduling account to beat normal account even with lower normal priority")
	}
	if isBetterAccountByHighestSchedulingPriorityAndLastUsed(normal, highest, false, now) {
		t.Fatalf("expected normal account not to beat effective highest scheduling account")
	}
}

func TestIsBetterAccountByHighestSchedulingPriorityAndLastUsed_SuppressedHighestIsNormal(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	normal := highestSchedulingTestAccount(1, 1, nil, false)
	suppressedHighest := highestSchedulingTestAccount(2, 99, nil, true)
	suppressedHighest.Extra[AccountExtraHighestSchedulingSuppressedUntil] = now.Add(10 * time.Minute).Format(time.RFC3339)

	if isBetterAccountByHighestSchedulingPriorityAndLastUsed(suppressedHighest, normal, false, now) {
		t.Fatalf("expected suppressed highest scheduling account to fall back to normal priority ordering")
	}
}

func TestSortAccountsByHighestSchedulingPriorityAndLastUsed_PreservesOrderingInsideHighestSubset(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	later := now.Add(-5 * time.Minute)
	earlier := now.Add(-30 * time.Minute)
	accounts := []*Account{
		highestSchedulingTestAccount(1, 1, &later, false),
		highestSchedulingTestAccount(2, 5, &later, true),
		highestSchedulingTestAccount(3, 1, &later, true),
		highestSchedulingTestAccount(4, 1, &earlier, true),
	}

	sortAccountsByHighestSchedulingPriorityAndLastUsed(accounts, false, now)

	want := []int64{4, 3, 2, 1}
	for i, id := range want {
		if accounts[i].ID != id {
			t.Fatalf("accounts[%d].ID = %d, want %d; order=%v", i, accounts[i].ID, id, []int64{accounts[0].ID, accounts[1].ID, accounts[2].ID, accounts[3].ID})
		}
	}
}

func TestFilterByHighestSchedulingLoadCandidates(t *testing.T) {
	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	normal := highestSchedulingTestAccount(1, 1, nil, false)
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	suppressedHighest := highestSchedulingTestAccount(3, 1, nil, true)
	suppressedHighest.Extra[AccountExtraHighestSchedulingSuppressedUntil] = now.Add(time.Hour).Format(time.RFC3339)
	candidates := []accountWithLoad{
		{account: normal, loadInfo: &AccountLoadInfo{AccountID: normal.ID, LoadRate: 0}},
		{account: highest, loadInfo: &AccountLoadInfo{AccountID: highest.ID, LoadRate: 0}},
		{account: suppressedHighest, loadInfo: &AccountLoadInfo{AccountID: suppressedHighest.ID, LoadRate: 0}},
	}

	filtered := filterByHighestSchedulingLoadCandidates(candidates, now)
	if len(filtered) != 1 || filtered[0].account.ID != highest.ID {
		t.Fatalf("expected only effective highest candidate, got %#v", filtered)
	}
}

func TestSortOpenAICompactRetryCandidates_HighestBeatsPriorityWithinStaleTier(t *testing.T) {
	normal := highestSchedulingTestAccount(1, 1, nil, false)
	normal.Platform = PlatformOpenAI
	highest := highestSchedulingTestAccount(2, 99, nil, true)
	highest.Platform = PlatformOpenAI
	pool := []openAIAccountCandidateScore{
		{account: normal, loadInfo: &AccountLoadInfo{AccountID: normal.ID, LoadRate: 0}},
		{account: highest, loadInfo: &AccountLoadInfo{AccountID: highest.ID, LoadRate: 0}},
	}

	ordered := sortOpenAICompactRetryCandidates(pool)
	if len(ordered) != 2 || ordered[0].account.ID != highest.ID {
		t.Fatalf("expected effective highest stale compact retry candidate first, got order=%v", []int64{ordered[0].account.ID, ordered[1].account.ID})
	}
}

func TestSortCandidatesForFallbackRandom_PreservesHighestSubsetAheadOfNormal(t *testing.T) {
	svc := &GatewayService{}

	for i := 0; i < 200; i++ {
		highest := highestSchedulingTestAccount(1, 1, nil, true)
		normal := highestSchedulingTestAccount(2, 1, nil, false)
		accounts := []*Account{highest, normal}

		svc.sortCandidatesForFallback(accounts, false, "random")

		if accounts[0].ID != highest.ID {
			t.Fatalf("expected random fallback to keep effective highest account ahead of normal accounts, got order=%v", []int64{accounts[0].ID, accounts[1].ID})
		}
	}
}
