package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const defaultHighestSchedulingRotationCount = 1

type HighestSchedulingRotationConfig struct {
	Enabled       bool     `json:"enabled"`
	GroupIDs      []int64  `json:"group_ids"`
	AccountTypes  []string `json:"account_types"`
	RotationCount int      `json:"rotation_count"`
}

type HighestSchedulingRotationState struct {
	Config           HighestSchedulingRotationConfig `json:"config"`
	ActiveAccountIDs []int64                         `json:"active_account_ids"`
	CandidateCount   int                             `json:"candidate_count"`
}

type HighestSchedulingRotationReconciler interface {
	ReconcileHighestSchedulingRotation(ctx context.Context, reason string) (*HighestSchedulingRotationState, error)
	ShouldReconcileHighestSchedulingRotation(ctx context.Context) (bool, error)
}

type highestSchedulingRotationReconcilerSink interface {
	SetHighestSchedulingRotationReconciler(HighestSchedulingRotationReconciler)
}

type highestSchedulingRotationAccountRepository interface {
	ListAllWithFilters(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, error)
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type highestSchedulingRotationCore struct {
	accountRepo highestSchedulingRotationAccountRepository
	settingRepo SettingRepository
}

func DefaultHighestSchedulingRotationConfig() HighestSchedulingRotationConfig {
	return HighestSchedulingRotationConfig{
		Enabled:       false,
		GroupIDs:      []int64{},
		AccountTypes:  []string{AccountTypeOAuth, AccountTypeAPIKey},
		RotationCount: defaultHighestSchedulingRotationCount,
	}
}

func NewHighestSchedulingRotationReconciler(accountRepo AccountRepository, settingService *SettingService) HighestSchedulingRotationReconciler {
	core := newHighestSchedulingRotationCore(accountRepo, settingService)
	InjectHighestSchedulingRotationReconciler(accountRepo, core)
	return core
}

func InjectHighestSchedulingRotationReconciler(accountRepo AccountRepository, reconciler HighestSchedulingRotationReconciler) {
	if sink, ok := accountRepo.(highestSchedulingRotationReconcilerSink); ok {
		sink.SetHighestSchedulingRotationReconciler(reconciler)
	}
}

func newHighestSchedulingRotationCore(accountRepo AccountRepository, settingService *SettingService) *highestSchedulingRotationCore {
	var settingRepo SettingRepository
	if settingService != nil {
		settingRepo = settingService.settingRepo
	}
	return &highestSchedulingRotationCore{accountRepo: accountRepo, settingRepo: settingRepo}
}

func (s *adminServiceImpl) highestSchedulingRotationCore() *highestSchedulingRotationCore {
	if s == nil {
		return &highestSchedulingRotationCore{}
	}
	return newHighestSchedulingRotationCore(s.accountRepo, s.settingService)
}

func (s *adminServiceImpl) GetHighestSchedulingRotationConfig(ctx context.Context) (*HighestSchedulingRotationState, error) {
	return s.highestSchedulingRotationCore().GetHighestSchedulingRotationConfig(ctx)
}

func (s *adminServiceImpl) UpdateHighestSchedulingRotationConfig(ctx context.Context, config HighestSchedulingRotationConfig) (*HighestSchedulingRotationState, error) {
	return s.highestSchedulingRotationCore().UpdateHighestSchedulingRotationConfig(ctx, config)
}

func (s *adminServiceImpl) ReconcileHighestSchedulingRotation(ctx context.Context, reason string) (*HighestSchedulingRotationState, error) {
	return s.highestSchedulingRotationCore().ReconcileHighestSchedulingRotation(ctx, reason)
}

func (s *adminServiceImpl) IsHighestSchedulingRotationManagedAccount(ctx context.Context, account *Account) (bool, error) {
	if s == nil || s.settingService == nil {
		return false, nil
	}
	return s.highestSchedulingRotationCore().IsHighestSchedulingRotationManagedAccount(ctx, account)
}

func (c *highestSchedulingRotationCore) GetHighestSchedulingRotationConfig(ctx context.Context) (*HighestSchedulingRotationState, error) {
	config, err := c.loadHighestSchedulingRotationConfig(ctx)
	if err != nil {
		return nil, err
	}
	state, err := c.highestSchedulingRotationState(ctx, config)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (c *highestSchedulingRotationCore) UpdateHighestSchedulingRotationConfig(ctx context.Context, config HighestSchedulingRotationConfig) (*HighestSchedulingRotationState, error) {
	previous, previousConfigured, err := c.loadHighestSchedulingRotationConfigWithState(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeHighestSchedulingRotationConfig(config)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	if c == nil || c.settingRepo == nil {
		return nil, errors.New("setting repository is not configured")
	}
	if err := c.settingRepo.Set(ctx, SettingKeyHighestSchedulingRotationConfig, string(payload)); err != nil {
		return nil, err
	}
	if normalized.Enabled {
		if _, err := c.ReconcileHighestSchedulingRotation(ctx, "config_update"); err != nil {
			return nil, err
		}
	} else if previousConfigured && previous.Enabled {
		if err := c.clearHighestSchedulingRotation(ctx, previous); err != nil {
			return nil, err
		}
	}
	return c.GetHighestSchedulingRotationConfig(ctx)
}

func (c *highestSchedulingRotationCore) ReconcileHighestSchedulingRotation(ctx context.Context, reason string) (*HighestSchedulingRotationState, error) {
	config, configured, err := c.loadHighestSchedulingRotationConfigWithState(ctx)
	if err != nil {
		return nil, err
	}
	if !configured {
		return &HighestSchedulingRotationState{
			Config:           config,
			ActiveAccountIDs: []int64{},
			CandidateCount:   0,
		}, nil
	}
	if !config.Enabled {
		return c.highestSchedulingRotationState(ctx, config)
	}
	if err := c.applyHighestSchedulingRotation(ctx, config); err != nil {
		return nil, err
	}
	return c.highestSchedulingRotationState(ctx, config)
}

func (c *highestSchedulingRotationCore) ShouldReconcileHighestSchedulingRotation(ctx context.Context) (bool, error) {
	config, configured, err := c.loadHighestSchedulingRotationConfigWithState(ctx)
	if err != nil {
		return false, err
	}
	if !configured || !config.Enabled {
		return false, nil
	}
	accounts, err := c.listHighestSchedulingRotationScopeAccounts(ctx, config)
	if err != nil {
		return false, err
	}
	activeCandidateCount := 0
	for i := range accounts {
		account := &accounts[i]
		if account.IsHighestSchedulingModeConfigured() && highestSchedulingRotationCandidate(account) {
			activeCandidateCount++
		}
	}
	return activeCandidateCount != config.RotationCount, nil
}

func (c *highestSchedulingRotationCore) IsHighestSchedulingRotationManagedAccount(ctx context.Context, account *Account) (bool, error) {
	config, configured, err := c.loadHighestSchedulingRotationConfigWithState(ctx)
	if err != nil {
		return false, err
	}
	if !configured || !config.Enabled {
		return false, nil
	}
	return accountInHighestSchedulingRotationScope(account, config), nil
}

func (c *highestSchedulingRotationCore) loadHighestSchedulingRotationConfig(ctx context.Context) (HighestSchedulingRotationConfig, error) {
	config, _, err := c.loadHighestSchedulingRotationConfigWithState(ctx)
	return config, err
}

func (c *highestSchedulingRotationCore) loadHighestSchedulingRotationConfigWithState(ctx context.Context) (HighestSchedulingRotationConfig, bool, error) {
	defaults := DefaultHighestSchedulingRotationConfig()
	if c == nil || c.settingRepo == nil {
		return defaults, false, nil
	}
	raw, err := c.settingRepo.GetValue(ctx, SettingKeyHighestSchedulingRotationConfig)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaults, false, nil
		}
		return defaults, false, err
	}
	if strings.TrimSpace(raw) == "" {
		return defaults, false, nil
	}
	var config HighestSchedulingRotationConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return defaults, false, nil
	}
	normalized, err := normalizeHighestSchedulingRotationConfig(config)
	if err != nil {
		return defaults, false, nil
	}
	return normalized, true, nil
}

func normalizeHighestSchedulingRotationConfig(config HighestSchedulingRotationConfig) (HighestSchedulingRotationConfig, error) {
	if config.RotationCount <= 0 {
		config.RotationCount = defaultHighestSchedulingRotationCount
	}
	config.GroupIDs = uniquePositiveInt64Values(config.GroupIDs)
	accountTypes, err := normalizeHighestSchedulingRotationAccountTypes(config.AccountTypes)
	if err != nil {
		return config, err
	}
	config.AccountTypes = accountTypes
	return config, nil
}

func normalizeHighestSchedulingRotationAccountTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{AccountTypeOAuth, AccountTypeAPIKey}, nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		accountType := strings.ToLower(strings.TrimSpace(value))
		switch accountType {
		case AccountTypeOAuth, AccountTypeAPIKey:
		default:
			return nil, infraerrors.BadRequest("INVALID_HIGHEST_SCHEDULING_ROTATION_ACCOUNT_TYPE", "highest scheduling rotation account_types only supports oauth and apikey")
		}
		if _, ok := seen[accountType]; ok {
			continue
		}
		seen[accountType] = struct{}{}
		out = append(out, accountType)
	}
	if len(out) == 0 {
		return []string{AccountTypeOAuth, AccountTypeAPIKey}, nil
	}
	return out, nil
}

func uniquePositiveInt64Values(values []int64) []int64 {
	if len(values) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (c *highestSchedulingRotationCore) clearHighestSchedulingRotation(ctx context.Context, config HighestSchedulingRotationConfig) error {
	accounts, err := c.listHighestSchedulingRotationScopeAccounts(ctx, config)
	if err != nil {
		return err
	}
	for i := range accounts {
		account := &accounts[i]
		if !account.IsHighestSchedulingModeConfigured() {
			continue
		}
		if err := c.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{AccountExtraHighestSchedulingMode: false}); err != nil {
			return err
		}
	}
	return nil
}

func (c *highestSchedulingRotationCore) applyHighestSchedulingRotation(ctx context.Context, config HighestSchedulingRotationConfig) error {
	accounts, err := c.listHighestSchedulingRotationScopeAccounts(ctx, config)
	if err != nil {
		return err
	}

	candidates := highestSchedulingRotationCandidates(accounts, config)
	sortHighestSchedulingRotationCandidates(candidates)
	candidateByID := make(map[int64]Account, len(candidates))
	for _, account := range candidates {
		candidateByID[account.ID] = account
	}

	activeCandidates := make([]Account, 0, len(candidates))
	for i := range accounts {
		account := &accounts[i]
		if !account.IsHighestSchedulingModeConfigured() {
			continue
		}
		if candidate, ok := candidateByID[account.ID]; ok {
			activeCandidates = append(activeCandidates, candidate)
			continue
		}
		if err := c.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{AccountExtraHighestSchedulingMode: false}); err != nil {
			return err
		}
	}

	sortHighestSchedulingRotationCandidates(activeCandidates)
	keepActiveIDs := make(map[int64]struct{}, config.RotationCount)
	limit := config.RotationCount
	if limit > len(activeCandidates) {
		limit = len(activeCandidates)
	}
	for i := 0; i < limit; i++ {
		keepActiveIDs[activeCandidates[i].ID] = struct{}{}
	}
	for i := limit; i < len(activeCandidates); i++ {
		if err := c.accountRepo.UpdateExtra(ctx, activeCandidates[i].ID, map[string]any{AccountExtraHighestSchedulingMode: false}); err != nil {
			return err
		}
	}

	activeCount := len(keepActiveIDs)
	if activeCount >= config.RotationCount {
		return nil
	}
	for i := range candidates {
		account := &candidates[i]
		if _, ok := keepActiveIDs[account.ID]; ok {
			continue
		}
		if err := c.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{AccountExtraHighestSchedulingMode: true}); err != nil {
			return err
		}
		keepActiveIDs[account.ID] = struct{}{}
		activeCount++
		if activeCount >= config.RotationCount {
			break
		}
	}
	return nil
}

func (c *highestSchedulingRotationCore) highestSchedulingRotationState(ctx context.Context, config HighestSchedulingRotationConfig) (*HighestSchedulingRotationState, error) {
	accounts, err := c.listHighestSchedulingRotationScopeAccounts(ctx, config)
	if err != nil {
		return nil, err
	}
	activeIDs := make([]int64, 0)
	candidateCount := 0
	for i := range accounts {
		account := &accounts[i]
		if account.IsHighestSchedulingModeConfigured() {
			activeIDs = append(activeIDs, account.ID)
		}
		if config.Enabled && highestSchedulingRotationCandidate(account) {
			candidateCount++
		}
	}
	sort.Slice(activeIDs, func(i, j int) bool { return activeIDs[i] < activeIDs[j] })
	return &HighestSchedulingRotationState{
		Config:           config,
		ActiveAccountIDs: activeIDs,
		CandidateCount:   candidateCount,
	}, nil
}

func (c *highestSchedulingRotationCore) listHighestSchedulingRotationScopeAccounts(ctx context.Context, config HighestSchedulingRotationConfig) ([]Account, error) {
	if c == nil || c.accountRepo == nil {
		return []Account{}, nil
	}
	accountsByID := make(map[int64]Account)
	appendAccounts := func(accounts []Account) {
		for _, account := range accounts {
			if !accountInHighestSchedulingRotationScope(&account, config) {
				continue
			}
			accountsByID[account.ID] = account
		}
	}
	for _, accountType := range config.AccountTypes {
		if len(config.GroupIDs) == 0 {
			accounts, err := c.accountRepo.ListAllWithFilters(ctx, "", accountType, "", "", 0, "")
			if err != nil {
				return nil, err
			}
			appendAccounts(accounts)
			continue
		}
		for _, groupID := range config.GroupIDs {
			accounts, err := c.accountRepo.ListAllWithFilters(ctx, "", accountType, "", "", groupID, "")
			if err != nil {
				return nil, err
			}
			appendAccounts(accounts)
		}
	}
	accounts := make([]Account, 0, len(accountsByID))
	for _, account := range accountsByID {
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func accountInHighestSchedulingRotationScope(account *Account, config HighestSchedulingRotationConfig) bool {
	if account == nil {
		return false
	}
	if !highestSchedulingRotationAccountTypeAllowed(account.Type, config.AccountTypes) {
		return false
	}
	if len(config.GroupIDs) == 0 {
		return true
	}
	allowedGroupIDs := make(map[int64]struct{}, len(config.GroupIDs))
	for _, groupID := range config.GroupIDs {
		allowedGroupIDs[groupID] = struct{}{}
	}
	for _, groupID := range account.GroupIDs {
		if _, ok := allowedGroupIDs[groupID]; ok {
			return true
		}
	}
	for _, accountGroup := range account.AccountGroups {
		if _, ok := allowedGroupIDs[accountGroup.GroupID]; ok {
			return true
		}
	}
	return false
}

func highestSchedulingRotationAccountTypeAllowed(accountType string, allowed []string) bool {
	for _, item := range allowed {
		if accountType == item {
			return true
		}
	}
	return false
}

func highestSchedulingRotationCandidates(accounts []Account, config HighestSchedulingRotationConfig) []Account {
	candidates := make([]Account, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if !accountInHighestSchedulingRotationScope(account, config) {
			continue
		}
		if highestSchedulingRotationCandidate(account) {
			candidates = append(candidates, *account)
		}
	}
	return candidates
}

func highestSchedulingRotationCandidate(account *Account) bool {
	if account == nil || account.Status != StatusActive || !account.Schedulable {
		return false
	}
	now := time.Now()
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now) {
		return false
	}
	if account.OverloadUntil != nil && account.OverloadUntil.After(now) {
		return false
	}
	if account.RateLimitResetAt != nil && account.RateLimitResetAt.After(now) {
		return false
	}
	if account.ExpiresAt != nil && account.AutoPauseOnExpired && !account.ExpiresAt.After(now) {
		return false
	}
	return true
}

func sortHighestSchedulingRotationCandidates(accounts []Account) {
	sort.SliceStable(accounts, func(i, j int) bool {
		left := accounts[i]
		right := accounts[j]
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		leftLoad := effectiveHighestSchedulingRotationLoadFactor(&left)
		rightLoad := effectiveHighestSchedulingRotationLoadFactor(&right)
		if leftLoad != rightLoad {
			return leftLoad > rightLoad
		}
		switch {
		case left.LastUsedAt == nil && right.LastUsedAt != nil:
			return true
		case left.LastUsedAt != nil && right.LastUsedAt == nil:
			return false
		case left.LastUsedAt != nil && right.LastUsedAt != nil && !left.LastUsedAt.Equal(*right.LastUsedAt):
			return left.LastUsedAt.Before(*right.LastUsedAt)
		default:
			return left.ID < right.ID
		}
	})
}

func effectiveHighestSchedulingRotationLoadFactor(account *Account) int {
	if account == nil {
		return 0
	}
	if account.LoadFactor != nil && *account.LoadFactor > 0 {
		return *account.LoadFactor
	}
	if account.Concurrency > 0 {
		return account.Concurrency
	}
	return 1
}

func ErrHighestSchedulingRotationManaged() error {
	return infraerrors.BadRequest("HIGHEST_SCHEDULING_ROTATION_MANAGED", "highest scheduling mode is managed by rotation for this account range")
}
