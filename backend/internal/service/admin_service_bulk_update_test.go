//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type accountRepoStubForBulkUpdate struct {
	accountRepoStub
	bulkUpdateErr       error
	bulkUpdateIDs       []int64
	bulkUpdatePayload   AccountBulkUpdate
	bindGroupErrByID    map[int64]error
	bindGroupsCalls     []int64
	bindGroupsByAccount map[int64][]int64
	createAccount       *Account
	createdAccount      *Account
	createID            int64
	createErr           error
	updatedAccount      *Account
	updatedAccounts     []*Account
	updateErr           error
	updateExtraPayload  map[string]any
	getByIDsAccounts    []*Account
	getByIDsErr         error
	getByIDsCalled      bool
	getByIDsIDs         []int64
	getByIDAccounts     map[int64]*Account
	getByIDErrByID      map[int64]error
	getByIDCalled       []int64
	listByGroupData     map[int64][]Account
	listByGroupErr      map[int64]error
	listData            []Account
	listResult          *pagination.PaginationResult
	listErr             error
	listCalled          bool
	lastListParams      pagination.PaginationParams
	lastListFilters     struct {
		platform    string
		accountType string
		status      string
		search      string
		groupID     int64
		privacyMode string
	}
}

type sessionLimitCacheStubForBulkUpdate struct {
	SessionLimitCache
	clearedAccountIDs []int64
}

func (s *sessionLimitCacheStubForBulkUpdate) ClearOpenAISessions(_ context.Context, accountID int64) error {
	s.clearedAccountIDs = append(s.clearedAccountIDs, accountID)
	return nil
}

func (s *accountRepoStubForBulkUpdate) Create(_ context.Context, account *Account) error {
	s.createAccount = account
	s.createdAccount = account
	if s.createID > 0 {
		account.ID = s.createID
	} else {
		account.ID = 101
	}
	return s.createErr
}

func (s *accountRepoStubForBulkUpdate) Update(_ context.Context, account *Account) error {
	s.updatedAccount = account
	s.updatedAccounts = append(s.updatedAccounts, account)
	return s.updateErr
}

func (s *accountRepoStubForBulkUpdate) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	s.updateExtraPayload = updates
	return nil
}

func (s *accountRepoStubForBulkUpdate) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	s.bulkUpdateIDs = append([]int64{}, ids...)
	s.bulkUpdatePayload = updates
	if s.bulkUpdateErr != nil {
		return 0, s.bulkUpdateErr
	}
	return int64(len(ids)), nil
}

func (s *accountRepoStubForBulkUpdate) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	s.bindGroupsCalls = append(s.bindGroupsCalls, accountID)
	if s.bindGroupsByAccount == nil {
		s.bindGroupsByAccount = make(map[int64][]int64)
	}
	s.bindGroupsByAccount[accountID] = append([]int64{}, groupIDs...)
	if err, ok := s.bindGroupErrByID[accountID]; ok {
		return err
	}
	return nil
}

func (s *accountRepoStubForBulkUpdate) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	s.getByIDsCalled = true
	s.getByIDsIDs = append([]int64{}, ids...)
	if s.getByIDsErr != nil {
		return nil, s.getByIDsErr
	}
	return s.getByIDsAccounts, nil
}

func (s *accountRepoStubForBulkUpdate) GetByID(_ context.Context, id int64) (*Account, error) {
	s.getByIDCalled = append(s.getByIDCalled, id)
	if err, ok := s.getByIDErrByID[id]; ok {
		return nil, err
	}
	if account, ok := s.getByIDAccounts[id]; ok {
		return account, nil
	}
	return nil, errors.New("account not found")
}

func (s *accountRepoStubForBulkUpdate) ListByGroup(_ context.Context, groupID int64) ([]Account, error) {
	if err, ok := s.listByGroupErr[groupID]; ok {
		return nil, err
	}
	if rows, ok := s.listByGroupData[groupID]; ok {
		return rows, nil
	}
	return nil, nil
}

func (s *accountRepoStubForBulkUpdate) ListAllWithFilters(context.Context, string, string, string, string, int64, string) ([]Account, error) {
	return nil, nil
}

func (s *accountRepoStubForBulkUpdate) ListWithFilters(_ context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	s.listCalled = true
	s.lastListParams = params
	s.lastListFilters.platform = platform
	s.lastListFilters.accountType = accountType
	s.lastListFilters.status = status
	s.lastListFilters.search = search
	s.lastListFilters.groupID = groupID
	s.lastListFilters.privacyMode = privacyMode
	if s.listErr != nil {
		return nil, nil, s.listErr
	}
	if s.listResult != nil {
		return s.listData, s.listResult, nil
	}
	return s.listData, &pagination.PaginationResult{Total: int64(len(s.listData))}, nil
}

func modeOnlyServiceTestExtra() map[string]any {
	return map[string]any{
		AccountExtraHighestSchedulingMode:      false,
		"highest_scheduling_recovery_minutes":  15,
		"highest_scheduling_suppressed":        true,
		"highest_scheduling_suppressed_until":  "2026-06-09T12:15:00Z",
		"highest_scheduling_suppressed_at":     "2026-06-09T12:00:00Z",
		"highest_scheduling_suppressed_reason": "boom",
		"unrelated":                            1,
	}
}

func requireModeOnlyServiceTestExtra(t *testing.T, extra map[string]any) {
	t.Helper()
	require.Equal(t, false, extra[AccountExtraHighestSchedulingMode])
	require.Equal(t, 1, extra["unrelated"])
	for _, key := range deprecatedHighestSchedulingExtraKeys {
		require.NotContains(t, extra, key)
	}
}

func TestAdminService_CreateAccount_SanitizesHighestSchedulingExtra(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "mode-only-create",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeAPIKey,
		Extra:                modeOnlyServiceTestExtra(),
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, repo.createdAccount)
	requireModeOnlyServiceTestExtra(t, repo.createdAccount.Extra)
}

func TestAdminService_UpdateAccount_SanitizesHighestSchedulingExtra(t *testing.T) {
	account := &Account{ID: 7, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive}
	repo := &accountRepoStubForBulkUpdate{getByIDAccounts: map[int64]*Account{7: account}}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.UpdateAccount(context.Background(), 7, &UpdateAccountInput{Extra: modeOnlyServiceTestExtra()})

	require.NoError(t, err)
	require.NotNil(t, repo.updatedAccount)
	requireModeOnlyServiceTestExtra(t, repo.updatedAccount.Extra)
}

func TestAdminService_UpdateAccountExtra_SanitizesHighestSchedulingExtra(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}

	err := svc.UpdateAccountExtra(context.Background(), 7, modeOnlyServiceTestExtra())

	require.NoError(t, err)
	requireModeOnlyServiceTestExtra(t, repo.updateExtraPayload)
}

func TestAdminService_BulkUpdateAccounts_SanitizesHighestSchedulingExtra(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}
	input := &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		Extra:      modeOnlyServiceTestExtra(),
	}

	_, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.NoError(t, err)
	requireModeOnlyServiceTestExtra(t, repo.bulkUpdatePayload.Extra)
}

func TestAdminService_BulkUpdateAccounts_OpenAISessionControl(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{getByIDsAccounts: []*Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
	}}
	svc := &adminServiceImpl{accountRepo: repo}

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1, 2},
		Extra: map[string]any{
			OpenAISessionControlEnabledExtraKey:      true,
			OpenAISessionMaxCountExtraKey:            8,
			OpenAISessionIdleTimeoutSecondsExtraKey:  120,
			OpenAISessionSlotRotationEnabledExtraKey: true,
		},
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.Success)
	require.Equal(t, true, repo.bulkUpdatePayload.Extra[OpenAISessionControlEnabledExtraKey])
	require.Equal(t, 8, repo.bulkUpdatePayload.Extra[OpenAISessionMaxCountExtraKey])
	require.Equal(t, 120, repo.bulkUpdatePayload.Extra[OpenAISessionIdleTimeoutSecondsExtraKey])
	require.Equal(t, true, repo.bulkUpdatePayload.Extra[OpenAISessionSlotRotationEnabledExtraKey])
}

func TestAdminService_BulkUpdateAccounts_OpenAISessionControlRejectsIneligibleOrMissingTarget(t *testing.T) {
	tests := []struct {
		name     string
		ids      []int64
		accounts []*Account
	}{
		{
			name: "API key target",
			ids:  []int64{1},
			accounts: []*Account{
				{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
			},
		},
		{
			name: "missing target",
			ids:  []int64{1, 2},
			accounts: []*Account{
				{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &accountRepoStubForBulkUpdate{getByIDsAccounts: tt.accounts}
			svc := &adminServiceImpl{accountRepo: repo}

			result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
				AccountIDs: tt.ids,
				Extra: map[string]any{
					OpenAISessionControlEnabledExtraKey: true,
				},
			})

			require.Nil(t, result)
			require.Error(t, err)
			require.Empty(t, repo.bulkUpdateIDs)
		})
	}
}

func TestAdminService_BulkUpdateAccounts_DisablingOpenAISessionControlClearsSlots(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{getByIDsAccounts: []*Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
	}}
	cache := &sessionLimitCacheStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo, sessionLimitCache: cache}

	_, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{1, 2},
		Extra: map[string]any{
			OpenAISessionControlEnabledExtraKey: false,
		},
	})

	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, cache.clearedAccountIDs)
}

// TestAdminService_BulkUpdateAccounts_AllSuccessIDs 验证批量更新成功时返回 success_ids/failed_ids。
func TestAdminService_BulkUpdateAccounts_AllSuccessIDs(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}

	schedulable := true
	input := &BulkUpdateAccountsInput{
		AccountIDs:  []int64{1, 2, 3},
		Schedulable: &schedulable,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, 3, result.Success)
	require.Equal(t, 0, result.Failed)
	require.ElementsMatch(t, []int64{1, 2, 3}, result.SuccessIDs)
	require.Empty(t, result.FailedIDs)
	require.Len(t, result.Results, 3)
}

func TestAdminService_BulkUpdateAccounts_RejectsRateChangeForSyncedAccounts(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		getByIDsAccounts: []*Account{
			{
				ID: 1,
				Extra: map[string]any{
					UpstreamBillingProbeEnabledExtraKey:    true,
					UpstreamBillingRateSyncEnabledExtraKey: true,
				},
			},
			{ID: 2, Extra: map[string]any{}},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}
	rateMultiplier := 0.5

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs:     []int64{1, 2},
		RateMultiplier: &rateMultiplier,
	})

	require.Nil(t, result)
	require.Error(t, err)
	var appErr *infraerrors.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, int32(http.StatusConflict), appErr.Code)
	require.Equal(t, "UPSTREAM_BILLING_RATE_SYNC_BULK_CONFLICT", appErr.Reason)
	require.Equal(t, "1", appErr.Metadata["count"])
	require.True(t, repo.getByIDsCalled)
	require.Empty(t, repo.bulkUpdateIDs, "rate conflict must be rejected before any write")
}

// TestAdminService_BulkUpdateAccounts_PartialFailureIDs 验证部分失败时 success_ids/failed_ids 正确。
func TestAdminService_BulkUpdateAccounts_PartialFailureIDs(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		bindGroupErrByID: map[int64]error{
			2: errors.New("bind failed"),
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &groupRepoStubForAdmin{getByID: &Group{ID: 10, Name: "g10"}},
	}

	groupIDs := []int64{10}
	schedulable := false
	input := &BulkUpdateAccountsInput{
		AccountIDs:            []int64{1, 2, 3},
		GroupIDs:              &groupIDs,
		Schedulable:           &schedulable,
		SkipMixedChannelCheck: true,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, 2, result.Success)
	require.Equal(t, 1, result.Failed)
	require.ElementsMatch(t, []int64{1, 3}, result.SuccessIDs)
	require.ElementsMatch(t, []int64{2}, result.FailedIDs)
	require.Len(t, result.Results, 3)
}

func TestAdminService_BulkUpdateAccounts_NilGroupRepoReturnsError(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{}
	svc := &adminServiceImpl{accountRepo: repo}

	groupIDs := []int64{10}
	input := &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		GroupIDs:   &groupIDs,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "group repository not configured")
}

// TestAdminService_BulkUpdateAccounts_MixedChannelPreCheckBlocksOnExistingConflict verifies
// that the global pre-check detects a conflict with existing group members and returns an
// error before any DB write is performed.
func TestAdminService_BulkUpdateAccounts_MixedChannelPreCheckBlocksOnExistingConflict(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		getByIDsAccounts: []*Account{
			{ID: 1, Platform: PlatformAntigravity},
		},
		// Group 10 already contains an Anthropic account.
		listByGroupData: map[int64][]Account{
			10: {{ID: 99, Platform: PlatformAnthropic}},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &groupRepoStubForAdmin{getByID: &Group{ID: 10, Name: "target-group"}},
	}

	groupIDs := []int64{10}
	input := &BulkUpdateAccountsInput{
		AccountIDs: []int64{1},
		GroupIDs:   &groupIDs,
	}

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.Nil(t, result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mixed channel")
	// No BindGroups should have been called since the check runs before any write.
	require.Empty(t, repo.bindGroupsCalls)
}

func TestAdminServiceBulkUpdateAccounts_ResolvesIDsFromFilters(t *testing.T) {
	repo := &accountRepoStubForBulkUpdate{
		listData: []Account{
			{ID: 7},
			{ID: 11},
		},
		listResult: &pagination.PaginationResult{Total: 2},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	schedulable := true
	input := &BulkUpdateAccountsInput{
		Schedulable: &schedulable,
	}

	filtersField := reflect.ValueOf(input).Elem().FieldByName("Filters")
	require.True(t, filtersField.IsValid(), "BulkUpdateAccountsInput should expose Filters for filter-target bulk update")
	require.Equal(t, reflect.Ptr, filtersField.Kind(), "BulkUpdateAccountsInput.Filters should be a pointer field")

	filtersValue := reflect.New(filtersField.Type().Elem())
	filtersValue.Elem().FieldByName("Platform").SetString(PlatformOpenAI)
	filtersValue.Elem().FieldByName("Type").SetString(AccountTypeOAuth)
	filtersValue.Elem().FieldByName("Status").SetString(StatusActive)
	filtersValue.Elem().FieldByName("Group").SetString("12")
	filtersValue.Elem().FieldByName("PrivacyMode").SetString(PrivacyModeCFBlocked)
	filtersValue.Elem().FieldByName("Search").SetString("bulk-target")
	filtersField.Set(filtersValue)

	result, err := svc.BulkUpdateAccounts(context.Background(), input)
	require.NoError(t, err)
	require.True(t, repo.listCalled, "expected filter-target bulk update to resolve matching IDs via account list filters")
	require.Equal(t, PlatformOpenAI, repo.lastListFilters.platform)
	require.Equal(t, AccountTypeOAuth, repo.lastListFilters.accountType)
	require.Equal(t, StatusActive, repo.lastListFilters.status)
	require.Equal(t, "bulk-target", repo.lastListFilters.search)
	require.Equal(t, int64(12), repo.lastListFilters.groupID)
	require.Equal(t, PrivacyModeCFBlocked, repo.lastListFilters.privacyMode)
	require.Equal(t, []int64{7, 11}, repo.bulkUpdateIDs)
	require.Equal(t, 2, result.Success)
	require.Equal(t, 0, result.Failed)
	require.Equal(t, []int64{7, 11}, result.SuccessIDs)
}
