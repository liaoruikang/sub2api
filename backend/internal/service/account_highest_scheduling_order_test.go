package service

import (
	"context"
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

type highestSchedulingRotationSettingRepoStub struct {
	values map[string]string
}

func (s *highestSchedulingRotationSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (s *highestSchedulingRotationSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s == nil || s.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *highestSchedulingRotationSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *highestSchedulingRotationSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *highestSchedulingRotationSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *highestSchedulingRotationSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *highestSchedulingRotationSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type highestSchedulingRotationAccountRepoStub struct {
	accounts         []Account
	updateExtraCalls int
}

func (r *highestSchedulingRotationAccountRepoStub) ListAllWithFilters(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, error) {
	out := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if platform != "" && account.Platform != platform {
			continue
		}
		if accountType != "" && account.Type != accountType {
			continue
		}
		if status != "" && account.Status != status {
			continue
		}
		if groupID > 0 && !highestSchedulingRotationTestAccountInGroup(account, groupID) {
			continue
		}
		out = append(out, account)
	}
	return out, nil
}

func (r *highestSchedulingRotationAccountRepoStub) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	r.updateExtraCalls++
	for i := range r.accounts {
		if r.accounts[i].ID != id {
			continue
		}
		if r.accounts[i].Extra == nil {
			r.accounts[i].Extra = map[string]any{}
		}
		for key, value := range updates {
			r.accounts[i].Extra[key] = value
		}
		return nil
	}
	return ErrAccountNotFound
}

func highestSchedulingRotationTestAccountInGroup(account Account, groupID int64) bool {
	for _, id := range account.GroupIDs {
		if id == groupID {
			return true
		}
	}
	for _, group := range account.AccountGroups {
		if group.GroupID == groupID {
			return true
		}
	}
	return false
}

func TestReconcileHighestSchedulingRotation_DisabledConfigDoesNotClearManualHighestScheduling(t *testing.T) {
	repo := &highestSchedulingRotationAccountRepoStub{
		accounts: []Account{
			*highestSchedulingTestAccount(1, 1, nil, true),
			*highestSchedulingTestAccount(2, 1, nil, nil),
		},
	}
	settings := &highestSchedulingRotationSettingRepoStub{values: map[string]string{
		SettingKeyHighestSchedulingRotationConfig: `{"enabled":false,"group_ids":[],"account_types":["apikey"],"rotation_count":1}`,
	}}
	core := &highestSchedulingRotationCore{accountRepo: repo, settingRepo: settings}

	state, err := core.ReconcileHighestSchedulingRotation(context.Background(), "account_update")
	if err != nil {
		t.Fatalf("reconcile disabled rotation: %v", err)
	}
	if repo.updateExtraCalls != 0 {
		t.Fatalf("disabled rotation must not update account extras, got calls=%d", repo.updateExtraCalls)
	}
	if len(state.ActiveAccountIDs) != 1 || state.ActiveAccountIDs[0] != 1 {
		t.Fatalf("disabled rotation must keep manually configured highest scheduling account, got active IDs=%v", state.ActiveAccountIDs)
	}
}
