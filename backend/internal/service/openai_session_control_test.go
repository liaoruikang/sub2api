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

func TestOpenAISessionControlRejectsMissingIdentityAndCacheErrorsPerAccount(t *testing.T) {
	tests := []struct {
		name     string
		identity openAISessionControlIdentity
		cache    *openAISessionControlCacheStub
	}{
		{
			name:     "missing SessionID",
			identity: openAISessionControlIdentity{Resolved: true},
			cache:    &openAISessionControlCacheStub{},
		},
		{
			name:     "cache failure",
			identity: openAISessionControlIdentity{Hash: "client-session", Resolved: true},
			cache:    &openAISessionControlCacheStub{errByAccount: map[int64]error{201: errors.New("redis unavailable")}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlled := controlledOpenAIAccount(201)
			fallback := &Account{ID: 202, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
			scheduler := &openAISessionControlSchedulerStub{accounts: []*Account{controlled, fallback}}
			svc := &OpenAIGatewayService{
				cfg:               &config.Config{},
				rateLimitService:  newOpenAIAdvancedSchedulerRateLimitService("true"),
				openaiScheduler:   scheduler,
				sessionLimitCache: tt.cache,
			}
			ctx := context.WithValue(context.Background(), openAISessionControlContextKey{}, tt.identity)

			selection, _, err := svc.SelectAccountWithScheduler(ctx, nil, "", "sticky", "gpt-5", nil, OpenAIUpstreamTransportAny, false)
			require.NoError(t, err)
			require.Equal(t, int64(202), selection.Account.ID)
		})
	}
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
