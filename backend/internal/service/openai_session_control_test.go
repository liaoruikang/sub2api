package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAISessionControlCacheStub struct {
	SessionLimitCache
	allowedByAccount      map[int64]bool
	errByAccount          map[int64]error
	ownerAccountID        int64
	ownerLookupErr        error
	registeredIDs         []int64
	registeredRotations   []bool
	ownerLookupSessionIDs []string
}

func (c *openAISessionControlCacheStub) RegisterOpenAISessionID(_ context.Context, accountID int64, _ string, _ int, _ time.Duration, rotateWhenFull ...bool) (bool, error) {
	c.registeredIDs = append(c.registeredIDs, accountID)
	c.registeredRotations = append(c.registeredRotations, len(rotateWhenFull) > 0 && rotateWhenFull[0])
	if err := c.errByAccount[accountID]; err != nil {
		return false, err
	}
	allowed, exists := c.allowedByAccount[accountID]
	return !exists || allowed, nil
}

func (c *openAISessionControlCacheStub) GetOpenAIStagedSessionAccountID(_ context.Context, sessionIDHash string) (int64, error) {
	c.ownerLookupSessionIDs = append(c.ownerLookupSessionIDs, sessionIDHash)
	if c.ownerLookupErr != nil {
		return 0, c.ownerLookupErr
	}
	return c.ownerAccountID, nil
}

type openAISessionControlSchedulerStub struct {
	accounts    []*Account
	selectedIDs []int64
}

func (s *openAISessionControlSchedulerStub) Select(_ context.Context, req OpenAIAccountScheduleRequest) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	if req.SessionControlPreferredAccountID > 0 {
		for _, account := range s.accounts {
			if account.ID != req.SessionControlPreferredAccountID {
				continue
			}
			if _, excluded := req.ExcludedIDs[account.ID]; excluded {
				break
			}
			s.selectedIDs = append(s.selectedIDs, account.ID)
			return &AccountSelectionResult{Account: account}, OpenAIAccountScheduleDecision{SelectedAccountID: account.ID}, nil
		}
	}
	for _, account := range s.accounts {
		if _, excluded := req.ExcludedIDs[account.ID]; excluded {
			continue
		}
		s.selectedIDs = append(s.selectedIDs, account.ID)
		return &AccountSelectionResult{Account: account}, OpenAIAccountScheduleDecision{SelectedAccountID: account.ID}, nil
	}
	return nil, OpenAIAccountScheduleDecision{}, ErrNoAvailableAccounts
}

func TestOpenAISessionControlPrefersReservedAccount(t *testing.T) {
	first := controlledOpenAIAccount(91)
	second := controlledOpenAIAccount(92)
	scheduler := &openAISessionControlSchedulerStub{accounts: []*Account{first, second}}
	cache := &openAISessionControlCacheStub{ownerAccountID: second.ID}
	svc := &OpenAIGatewayService{
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		openaiScheduler:   scheduler,
		sessionLimitCache: cache,
	}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "reserved-session", Resolved: true})

	selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "sticky", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, second.ID, selection.Account.ID)
	require.Equal(t, []int64{second.ID}, scheduler.selectedIDs)
	require.Equal(t, []int64{second.ID}, cache.registeredIDs)
	require.Equal(t, []string{"reserved-session"}, cache.ownerLookupSessionIDs)
}

func TestOpenAISessionControlReservationWorksWithAdvancedAndLegacySchedulers(t *testing.T) {
	for _, tt := range []struct {
		name            string
		advancedEnabled string
		expectedLayer   string
	}{
		{name: "advanced", advancedEnabled: "true", expectedLayer: openAIAccountScheduleLayerSessionControl},
		{name: "legacy", advancedEnabled: "false", expectedLayer: openAIAccountScheduleLayerLoadBalance},
	} {
		t.Run(tt.name, func(t *testing.T) {
			normal := Account{
				ID: 111, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
				Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 100,
			}
			preferred := *controlledOpenAIAccount(112)
			preferred.Status = StatusActive
			preferred.Schedulable = true
			preferred.Concurrency = 1
			preferred.Priority = 0
			gatewayCache := &schedulerTestGatewayCache{}
			sessionCache := &openAISessionControlCacheStub{ownerAccountID: preferred.ID}
			cfg := &config.Config{}
			cfg.Gateway.Scheduling.LoadBatchEnabled = false
			svc := &OpenAIGatewayService{
				accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{normal, preferred}},
				cache:              gatewayCache,
				cfg:                cfg,
				rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService(tt.advancedEnabled),
				concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
				sessionLimitCache:  sessionCache,
			}
			ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "reserved-session", Resolved: true})

			selection, decision, err := svc.SelectAccountWithScheduler(ctx, nil, "", "reserved-sticky", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
			require.NoError(t, err)
			require.NotNil(t, selection)
			require.Equal(t, preferred.ID, selection.Account.ID)
			require.Equal(t, tt.expectedLayer, decision.Layer)
			require.Equal(t, preferred.ID, gatewayCache.sessionBindings["openai:reserved-sticky"])
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
		})
	}
}

func (*openAISessionControlSchedulerStub) ReportResult(int64, bool, *int) {}
func (*openAISessionControlSchedulerStub) ReportSwitch()                  {}
func (*openAISessionControlSchedulerStub) SnapshotMetrics() OpenAIAccountSchedulerMetricsSnapshot {
	return OpenAIAccountSchedulerMetricsSnapshot{}
}

func controlledOpenAIAccount(id int64) *Account {
	return &Account{
		ID:       id,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			OpenAISessionControlEnabledExtraKey:     true,
			OpenAISessionMaxCountExtraKey:           3,
			OpenAISessionIdleTimeoutSecondsExtraKey: 60,
		},
	}
}

func TestOpenAISessionControlPassesSlotRotationSetting(t *testing.T) {
	account := controlledOpenAIAccount(401)
	account.Extra[OpenAISessionSlotRotationEnabledExtraKey] = true
	cache := &openAISessionControlCacheStub{}
	svc := &OpenAIGatewayService{sessionLimitCache: cache}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "client-session", Resolved: true})

	allowed, reason := svc.registerOpenAISessionControl(ctx, account)
	require.True(t, allowed)
	require.Empty(t, reason)
	require.Equal(t, []bool{true}, cache.registeredRotations)
}

func TestOpenAISessionControlSkipsFullAccountAndContinuesScheduling(t *testing.T) {
	first := controlledOpenAIAccount(101)
	second := &Account{ID: 102, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	scheduler := &openAISessionControlSchedulerStub{accounts: []*Account{first, second}}
	cache := &openAISessionControlCacheStub{allowedByAccount: map[int64]bool{101: false}}
	svc := &OpenAIGatewayService{
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		openaiScheduler:   scheduler,
		sessionLimitCache: cache,
	}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "client-session", Resolved: true})

	selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "sticky", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, int64(102), selection.Account.ID)
	require.Equal(t, []int64{101, 102}, scheduler.selectedIDs)
	require.Equal(t, []int64{101}, cache.registeredIDs)
}

func TestOpenAISessionControlMissingIdentityUsesOrdinaryScheduling(t *testing.T) {
	controlled := controlledOpenAIAccount(201)
	missingParentID := int64(999)
	controlled.ParentAccountID = &missingParentID
	fallback := &Account{ID: 202, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	scheduler := &openAISessionControlSchedulerStub{accounts: []*Account{controlled, fallback}}
	cache := &openAISessionControlCacheStub{}
	svc := &OpenAIGatewayService{
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		openaiScheduler:   scheduler,
		sessionLimitCache: cache,
	}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Resolved: true})

	selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "sticky", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, controlled.ID, selection.Account.ID)
	require.Empty(t, cache.registeredIDs)
}

func TestOpenAISessionControlCacheErrorsSkipControlledAccount(t *testing.T) {
	controlled := controlledOpenAIAccount(201)
	fallback := &Account{ID: 202, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	scheduler := &openAISessionControlSchedulerStub{accounts: []*Account{controlled, fallback}}
	cache := &openAISessionControlCacheStub{errByAccount: map[int64]error{controlled.ID: errors.New("redis unavailable")}}
	svc := &OpenAIGatewayService{
		cfg:               &config.Config{},
		rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
		openaiScheduler:   scheduler,
		sessionLimitCache: cache,
	}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "client-session", Resolved: true})

	selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "sticky", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, fallback.ID, selection.Account.ID)
}

func TestOpenAISessionControlBusyStickyMovesToAvailableAccount(t *testing.T) {
	for _, tt := range []struct {
		name             string
		advancedEnabled  string
		loadBatchEnabled bool
		stagedPreference bool
	}{
		{name: "advanced", advancedEnabled: "true", loadBatchEnabled: true},
		{name: "legacy load batch", advancedEnabled: "false", loadBatchEnabled: true},
		{name: "legacy direct selection", advancedEnabled: "false", loadBatchEnabled: false},
		{name: "advanced staged preference", advancedEnabled: "true", loadBatchEnabled: true, stagedPreference: true},
		{name: "legacy load batch staged preference", advancedEnabled: "false", loadBatchEnabled: true, stagedPreference: true},
		{name: "legacy direct staged preference", advancedEnabled: "false", loadBatchEnabled: false, stagedPreference: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			busy := *controlledOpenAIAccount(211)
			busy.Status = StatusActive
			busy.Schedulable = true
			busy.Concurrency = 1
			busy.Priority = 0
			available := *controlledOpenAIAccount(212)
			available.Status = StatusActive
			available.Schedulable = true
			available.Concurrency = 1
			available.Priority = 10
			gatewayCache := &schedulerTestGatewayCache{}
			sessionCache := &openAISessionControlCacheStub{}
			if tt.stagedPreference {
				sessionCache.ownerAccountID = busy.ID
			} else {
				gatewayCache.sessionBindings = map[string]int64{"openai:busy-session": busy.ID}
			}
			cfg := &config.Config{}
			cfg.Gateway.Scheduling.LoadBatchEnabled = tt.loadBatchEnabled
			cfg.Gateway.Scheduling.StickySessionMaxWaiting = 3
			cfg.Gateway.Scheduling.StickySessionWaitTimeout = time.Second
			cfg.Gateway.Scheduling.FallbackWaitTimeout = time.Second
			cfg.Gateway.Scheduling.FallbackMaxWaiting = 10
			svc := &OpenAIGatewayService{
				accountRepo:      schedulerTestOpenAIAccountRepo{accounts: []Account{busy, available}},
				cache:            gatewayCache,
				cfg:              cfg,
				rateLimitService: newOpenAIAdvancedSchedulerRateLimitService(tt.advancedEnabled),
				concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
					loadMap: map[int64]*AccountLoadInfo{
						busy.ID:      {AccountID: busy.ID, LoadRate: 100},
						available.ID: {AccountID: available.ID, LoadRate: 0},
					},
					acquireResults: map[int64]bool{busy.ID: false, available.ID: true},
				}),
				sessionLimitCache: sessionCache,
			}
			ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "client-session", Resolved: true})

			selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "busy-session", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
			require.NoError(t, err)
			require.NotNil(t, selection)
			require.Equal(t, available.ID, selection.Account.ID)
			require.True(t, selection.Acquired)
			require.Nil(t, selection.WaitPlan)
			require.Equal(t, []int64{available.ID}, sessionCache.registeredIDs)
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
		})
	}
}

func TestOpenAISessionControlLegacyDirectSelectionAllBusyFallsBackToWait(t *testing.T) {
	first := *controlledOpenAIAccount(221)
	first.Status = StatusActive
	first.Schedulable = true
	first.Concurrency = 1
	first.Priority = 0
	second := *controlledOpenAIAccount(222)
	second.Status = StatusActive
	second.Schedulable = true
	second.Concurrency = 1
	second.Priority = 10
	gatewayCache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{
		"openai:all-busy-session": first.ID,
	}}
	sessionCache := &openAISessionControlCacheStub{}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	cfg.Gateway.Scheduling.FallbackWaitTimeout = time.Second
	cfg.Gateway.Scheduling.FallbackMaxWaiting = 10
	svc := &OpenAIGatewayService{
		accountRepo:      schedulerTestOpenAIAccountRepo{accounts: []Account{first, second}},
		cache:            gatewayCache,
		cfg:              cfg,
		rateLimitService: newOpenAIAdvancedSchedulerRateLimitService("false"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
			acquireResults: map[int64]bool{first.ID: false, second.ID: false},
		}),
		sessionLimitCache: sessionCache,
	}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "client-session", Resolved: true})

	selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "all-busy-session", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.Equal(t, first.ID, selection.Account.ID)
	require.False(t, selection.Acquired)
	require.NotNil(t, selection.WaitPlan)
	require.Equal(t, first.ID, selection.WaitPlan.AccountID)
	require.Equal(t, []int64{first.ID}, sessionCache.registeredIDs)
}

func TestOpenAISessionControlShadowUsesParentSlots(t *testing.T) {
	parent := controlledOpenAIAccount(301)
	parentID := parent.ID
	shadow := &Account{ID: 302, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ParentAccountID: &parentID}
	cache := &openAISessionControlCacheStub{}
	svc := &OpenAIGatewayService{
		accountRepo:       schedulerTestOpenAIAccountRepo{accounts: []Account{*parent}},
		sessionLimitCache: cache,
	}
	ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, openAISessionControlIdentity{Hash: "client-session", Resolved: true})

	allowed, reason := svc.registerOpenAISessionControl(ctx, shadow)
	require.True(t, allowed)
	require.Empty(t, reason)
	require.Equal(t, []int64{parent.ID}, cache.registeredIDs)
}

func TestAttachOpenAISessionControlIdentityIsTenantIsolated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	newContext := func(apiKeyID int64) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
		c.Set("api_key", &APIKey{ID: apiKeyID})
		return c
	}

	first := newContext(1)
	second := newContext(2)
	attachOpenAISessionControlIdentityToGin(first, "same-session")
	attachOpenAISessionControlIdentityToGin(second, "same-session")

	firstIdentity := openAISessionControlIdentityFromContext(first.Request.Context())
	secondIdentity := openAISessionControlIdentityFromContext(second.Request.Context())
	require.True(t, firstIdentity.Resolved)
	require.NotEmpty(t, firstIdentity.Hash)
	require.NotEqual(t, firstIdentity.Hash, secondIdentity.Hash)
}
