package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/dgraph-io/ristretto"
)

const apiKeyAuthSnapshotVersion = 26 // v26: include tag-derived standard exclusive groups

type apiKeyAuthCacheConfig struct {
	l1Size        int
	l1TTL         time.Duration
	l2TTL         time.Duration
	negativeTTL   time.Duration
	jitterPercent int
	singleflight  bool
}

func newAPIKeyAuthCacheConfig(cfg *config.Config) apiKeyAuthCacheConfig {
	if cfg == nil {
		return apiKeyAuthCacheConfig{}
	}
	auth := cfg.APIKeyAuth
	return apiKeyAuthCacheConfig{
		l1Size:        auth.L1Size,
		l1TTL:         time.Duration(auth.L1TTLSeconds) * time.Second,
		l2TTL:         time.Duration(auth.L2TTLSeconds) * time.Second,
		negativeTTL:   time.Duration(auth.NegativeTTLSeconds) * time.Second,
		jitterPercent: auth.JitterPercent,
		singleflight:  auth.Singleflight,
	}
}

func (c apiKeyAuthCacheConfig) l1Enabled() bool {
	return c.l1Size > 0 && c.l1TTL > 0
}

func (c apiKeyAuthCacheConfig) l2Enabled() bool {
	return c.l2TTL > 0
}

func (c apiKeyAuthCacheConfig) negativeEnabled() bool {
	return c.negativeTTL > 0
}

// jitterTTL 为缓存 TTL 添加抖动，避免多个请求在同一时刻同时过期触发集中回源。
// 这里直接使用 rand/v2 的顶层函数：并发安全，无需全局互斥锁。
func (c apiKeyAuthCacheConfig) jitterTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	if c.jitterPercent <= 0 {
		return ttl
	}
	percent := c.jitterPercent
	if percent > 100 {
		percent = 100
	}
	delta := float64(percent) / 100
	randVal := rand.Float64()
	factor := 1 - delta + randVal*(2*delta)
	if factor <= 0 {
		return ttl
	}
	return time.Duration(float64(ttl) * factor)
}

func (s *APIKeyService) initAuthCache(cfg *config.Config) {
	s.authCfg = newAPIKeyAuthCacheConfig(cfg)
	if s.authCfg.negativeEnabled() {
		negativeSize := defaultNegativeAuthCacheSize
		if s.authCfg.l1Size > 0 && s.authCfg.l1Size < negativeSize {
			negativeSize = s.authCfg.l1Size
		}
		cache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: int64(negativeSize) * 10,
			MaxCost:     int64(negativeSize),
			BufferItems: 64,
		})
		if err == nil {
			s.authNegativeCacheL1 = cache
		}
	}
	if s.authCfg.l1Enabled() {
		cache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: int64(s.authCfg.l1Size) * 10,
			MaxCost:     int64(s.authCfg.l1Size),
			BufferItems: 64,
		})
		if err == nil {
			s.authCacheL1 = cache
		}
	}
}

// StartAuthCacheInvalidationSubscriber starts the Pub/Sub subscriber for L1 cache invalidation.
// This should be called after the service is fully initialized.
func (s *APIKeyService) StartAuthCacheInvalidationSubscriber(ctx context.Context) {
	if s.cache == nil || (s.authCacheL1 == nil && s.authNegativeCacheL1 == nil) {
		return
	}
	s.authInvalidationStart.Do(func() {
		subscriberCtx, cancel := context.WithCancel(ctx)
		subscriberCtx = withAuthCacheSubscriptionReady(subscriberCtx, func() {
			s.authInvalidationConnected.Store(true)
		})
		s.authInvalidationCancel = cancel
		s.authInvalidationWG.Add(1)
		go func() {
			defer s.authInvalidationWG.Done()
			backoff := time.Second
			for {
				err := s.cache.SubscribeAuthCacheInvalidation(subscriberCtx, func(cacheKey string) {
					s.invalidateLocalAuthCache(cacheKey)
				})
				wasConnected := s.authInvalidationConnected.Swap(false)
				if subscriberCtx.Err() != nil {
					return
				}
				if wasConnected {
					backoff = time.Second
				}
				s.authInvalidationFailures.Add(1)
				if err == nil {
					err = errors.New("auth cache invalidation subscription closed")
				}
				slog.Warn("failed to start auth cache invalidation subscriber; retrying", "error", err, "retry_in", backoff)
				timer := time.NewTimer(backoff)
				select {
				case <-subscriberCtx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				if backoff < 30*time.Second {
					backoff *= 2
					if backoff > 30*time.Second {
						backoff = 30 * time.Second
					}
				}
			}
		}()
	})
}

func (s *APIKeyService) invalidateLocalAuthCache(cacheKey string) {
	if s == nil {
		return
	}
	if s.authCacheL1 != nil {
		s.authCacheL1.Del(cacheKey)
	}
	if s.authNegativeCacheL1 != nil {
		s.authNegativeCacheL1.Del(cacheKey)
	}
}

type AuthCacheInvalidationSubscriberHealth struct {
	Connected bool   `json:"connected"`
	Failures  uint64 `json:"failures"`
}

func (s *APIKeyService) AuthCacheInvalidationSubscriberHealth() AuthCacheInvalidationSubscriberHealth {
	if s == nil {
		return AuthCacheInvalidationSubscriberHealth{}
	}
	return AuthCacheInvalidationSubscriberHealth{
		Connected: s.authInvalidationConnected.Load(),
		Failures:  s.authInvalidationFailures.Load(),
	}
}

func (s *APIKeyService) StopAuthCacheInvalidationSubscriber() {
	if s == nil {
		return
	}
	s.authInvalidationStop.Do(func() {
		if s.authInvalidationCancel != nil {
			s.authInvalidationCancel()
		}
		s.authInvalidationWG.Wait()
	})
}

func (s *APIKeyService) authCacheKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (s *APIKeyService) getAuthCacheEntry(ctx context.Context, cacheKey string) (*APIKeyAuthCacheEntry, bool) {
	if s.authCacheL1 != nil {
		if val, ok := s.authCacheL1.Get(cacheKey); ok {
			if entry, ok := val.(*APIKeyAuthCacheEntry); ok {
				return entry, true
			}
		}
	}
	if s.authNegativeCacheL1 != nil {
		if val, ok := s.authNegativeCacheL1.Get(cacheKey); ok {
			if entry, ok := val.(*APIKeyAuthCacheEntry); ok && entry.NotFound {
				return entry, true
			}
		}
	}
	if s.cache == nil || !s.authCfg.l2Enabled() {
		return nil, false
	}
	entry, err := s.cache.GetAuthCache(ctx, cacheKey)
	if err != nil {
		return nil, false
	}
	s.setAuthCacheL1(cacheKey, entry)
	return entry, true
}

func (s *APIKeyService) setAuthCacheL1(cacheKey string, entry *APIKeyAuthCacheEntry) {
	if entry == nil {
		return
	}
	if entry.NotFound {
		if s.authNegativeCacheL1 != nil && s.authCfg.negativeTTL > 0 {
			_ = s.authNegativeCacheL1.SetWithTTL(cacheKey, entry, 1, s.authCfg.jitterTTL(s.authCfg.negativeTTL))
		}
		return
	}
	if s.authCacheL1 == nil {
		return
	}
	ttl := s.authCfg.l1TTL
	ttl = s.authCfg.jitterTTL(ttl)
	_ = s.authCacheL1.SetWithTTL(cacheKey, entry, 1, ttl)
}

func (s *APIKeyService) setAuthCacheEntry(ctx context.Context, cacheKey string, entry *APIKeyAuthCacheEntry, ttl time.Duration) {
	if entry == nil {
		return
	}
	s.setAuthCacheL1(cacheKey, entry)
	if s.cache == nil || !s.authCfg.l2Enabled() {
		return
	}
	_ = s.cache.SetAuthCache(ctx, cacheKey, entry, s.authCfg.jitterTTL(ttl))
}

func (s *APIKeyService) deleteAuthCache(ctx context.Context, cacheKey string) {
	if s.authCacheL1 != nil {
		s.authCacheL1.Del(cacheKey)
	}
	if s.authNegativeCacheL1 != nil {
		s.authNegativeCacheL1.Del(cacheKey)
	}
	if s.cache == nil {
		return
	}
	_ = s.cache.DeleteAuthCache(ctx, cacheKey)
	// Publish invalidation message to other instances
	_ = s.cache.PublishAuthCacheInvalidation(ctx, cacheKey)
}

func (s *APIKeyService) loadAuthCacheEntry(ctx context.Context, key, cacheKey string) (*APIKeyAuthCacheEntry, error) {
	apiKey, err := s.lookupAPIKeyForAuth(ctx, key)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			entry := &APIKeyAuthCacheEntry{NotFound: true}
			if s.authCfg.negativeEnabled() {
				// Invalid keys are attacker-controlled and high-cardinality. Keep their
				// negative entries in the bounded process-local cache; do not amplify
				// random-key scans into Redis writes on every instance.
				s.setAuthCacheL1(cacheKey, entry)
			}
			return entry, nil
		}
		return nil, fmt.Errorf("get api key: %w", err)
	}
	apiKey.Key = key
	if err := s.loadTagDerivedGroups(ctx, apiKey); err != nil {
		return nil, err
	}
	snapshot := s.snapshotFromAPIKey(ctx, apiKey)
	if snapshot == nil {
		return nil, fmt.Errorf("get api key: %w", ErrAPIKeyNotFound)
	}
	entry := &APIKeyAuthCacheEntry{Snapshot: snapshot}
	s.setAuthCacheEntry(ctx, cacheKey, entry, s.authCfg.l2TTL)
	return entry, nil
}

func (s *APIKeyService) lookupAPIKeyForAuth(ctx context.Context, key string) (*APIKey, error) {
	if s == nil || s.apiKeyRepo == nil {
		return nil, ErrAPIKeyNotFound
	}
	if s.authLookupSlots == nil {
		return s.apiKeyRepo.GetByKeyForAuth(ctx, key)
	}
	s.authLookupTotal.Add(1)
	select {
	case s.authLookupSlots <- struct{}{}:
		s.authLookupInFlight.Add(1)
		defer func() {
			s.authLookupInFlight.Add(-1)
			<-s.authLookupSlots
		}()
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.authLookupRejected.Add(1)
		return nil, ErrAPIKeyAuthOverloaded
	}
	return s.apiKeyRepo.GetByKeyForAuth(ctx, key)
}

func (s *APIKeyService) applyAuthCacheEntry(key string, entry *APIKeyAuthCacheEntry) (*APIKey, bool, error) {
	if entry == nil {
		return nil, false, nil
	}
	if entry.NotFound {
		return nil, true, ErrAPIKeyNotFound
	}
	if entry.Snapshot == nil {
		return nil, false, nil
	}
	if entry.Snapshot.Version != apiKeyAuthSnapshotVersion {
		return nil, false, nil
	}
	return s.snapshotToAPIKey(key, entry.Snapshot), true, nil
}

func (s *APIKeyService) snapshotFromAPIKey(ctx context.Context, apiKey *APIKey) *APIKeyAuthSnapshot {
	if apiKey == nil || apiKey.User == nil {
		return nil
	}
	snapshot := &APIKeyAuthSnapshot{
		Version:     apiKeyAuthSnapshotVersion,
		APIKeyID:    apiKey.ID,
		UserID:      apiKey.UserID,
		GroupID:     apiKey.GroupID,
		GroupIDs:    append([]int64(nil), apiKey.GroupIDs...),
		Name:        apiKey.Name,
		Status:      apiKey.Status,
		IPWhitelist: apiKey.IPWhitelist,
		IPBlacklist: apiKey.IPBlacklist,
		Quota:       apiKey.Quota,
		QuotaUsed:   apiKey.QuotaUsed,
		ExpiresAt:   apiKey.ExpiresAt,
		RateLimit5h: apiKey.RateLimit5h,
		RateLimit1d: apiKey.RateLimit1d,
		RateLimit7d: apiKey.RateLimit7d,
		User: APIKeyAuthUserSnapshot{
			ID:                         apiKey.User.ID,
			Status:                     apiKey.User.Status,
			Role:                       apiKey.User.Role,
			Balance:                    apiKey.User.Balance,
			Concurrency:                apiKey.User.Concurrency,
			AllowedGroups:              append([]int64(nil), apiKey.User.AllowedGroups...),
			TagDerivedGroups:           append([]int64(nil), apiKey.User.TagDerivedGroups...),
			Email:                      apiKey.User.Email,
			Username:                   apiKey.User.Username,
			BalanceNotifyEnabled:       apiKey.User.BalanceNotifyEnabled,
			BalanceNotifyThresholdType: apiKey.User.BalanceNotifyThresholdType,
			BalanceNotifyThreshold:     apiKey.User.BalanceNotifyThreshold,
			BalanceNotifyExtraEmails:   apiKey.User.BalanceNotifyExtraEmails,
			TotalRecharged:             apiKey.User.TotalRecharged,
			RPMLimit:                   apiKey.User.RPMLimit,
		},
	}

	if apiKey.GroupID != nil && *apiKey.GroupID > 0 && s.userGroupRateRepo != nil {
		override, err := s.userGroupRateRepo.GetRPMOverrideByUserAndGroup(ctx, apiKey.UserID, *apiKey.GroupID)
		if err == nil && override != nil {
			snapshot.User.UserGroupRPMOverride = override
		}
		limitedTimeOverride, err := s.userGroupRateRepo.GetLimitedTimeRPMOverrideByUserAndGroup(ctx, apiKey.UserID, *apiKey.GroupID)
		if err == nil && limitedTimeOverride != nil {
			snapshot.User.UserGroupLimitedTimeRPMOverride = limitedTimeOverride
		}
	}
	if apiKey.Group != nil {
		snapshot.Group = &APIKeyAuthGroupSnapshot{
			ID:                                   apiKey.Group.ID,
			Name:                                 apiKey.Group.Name,
			Platform:                             apiKey.Group.Platform,
			IsExclusive:                          apiKey.Group.IsExclusive,
			Status:                               apiKey.Group.Status,
			SubscriptionType:                     apiKey.Group.SubscriptionType,
			RateMultiplier:                       apiKey.Group.RateMultiplier,
			LimitedTimeMultiplierEnabled:         apiKey.Group.LimitedTimeMultiplierEnabled,
			LimitedTimeMultiplierCron:            apiKey.Group.LimitedTimeMultiplierCron,
			LimitedTimeMultiplierDurationMinutes: apiKey.Group.LimitedTimeMultiplierDurationMinutes,
			LimitedTimeMultiplierValue:           apiKey.Group.LimitedTimeMultiplierValue,
			DailyLimitUSD:                        apiKey.Group.DailyLimitUSD,
			WeeklyLimitUSD:                       apiKey.Group.WeeklyLimitUSD,
			MonthlyLimitUSD:                      apiKey.Group.MonthlyLimitUSD,
			AllowImageGeneration:                 apiKey.Group.AllowImageGeneration,
			AllowBatchImageGeneration:            apiKey.Group.AllowBatchImageGeneration,
			ImageRateIndependent:                 apiKey.Group.ImageRateIndependent,
			ImageRateMultiplier:                  apiKey.Group.ImageRateMultiplier,
			ImagePrice1K:                         apiKey.Group.ImagePrice1K,
			ImagePrice2K:                         apiKey.Group.ImagePrice2K,
			ImagePrice4K:                         apiKey.Group.ImagePrice4K,
			BatchImageDiscountMultiplier:         apiKey.Group.BatchImageDiscountMultiplier,
			BatchImageHoldMultiplier:             apiKey.Group.BatchImageHoldMultiplier,
			VideoRateIndependent:                 apiKey.Group.VideoRateIndependent,
			VideoRateMultiplier:                  apiKey.Group.VideoRateMultiplier,
			VideoPrice480P:                       apiKey.Group.VideoPrice480P,
			VideoPrice720P:                       apiKey.Group.VideoPrice720P,
			VideoPrice1080P:                      apiKey.Group.VideoPrice1080P,
			WebSearchPricePerCall:                apiKey.Group.WebSearchPricePerCall,
			ClaudeCodeOnly:                       apiKey.Group.ClaudeCodeOnly,
			FallbackGroupID:                      apiKey.Group.FallbackGroupID,
			FallbackGroupIDOnInvalidRequest:      apiKey.Group.FallbackGroupIDOnInvalidRequest,
			ModelRouting:                         apiKey.Group.ModelRouting,
			ModelRoutingEnabled:                  apiKey.Group.ModelRoutingEnabled,
			MCPXMLInject:                         apiKey.Group.MCPXMLInject,
			SupportedModelScopes:                 apiKey.Group.SupportedModelScopes,
			AllowMessagesDispatch:                apiKey.Group.AllowMessagesDispatch,
			AllowLive:                            apiKey.Group.AllowLive,
			RequireOAuthOnly:                     apiKey.Group.RequireOAuthOnly,
			RequirePrivacySet:                    apiKey.Group.RequirePrivacySet,
			DefaultMappedModel:                   apiKey.Group.DefaultMappedModel,
			MessagesDispatchModelConfig:          apiKey.Group.MessagesDispatchModelConfig,
			ModelsListConfig:                     apiKey.Group.ModelsListConfig,
			RPMLimit:                             apiKey.Group.RPMLimit,
			LimitedTimeRPMLimit:                  apiKey.Group.LimitedTimeRPMLimit,
			LimitedTimeUserConcurrencyLimit:      apiKey.Group.LimitedTimeUserConcurrencyLimit,
			UserConcurrencyLimit:                 apiKey.Group.UserConcurrencyLimit,
			MaxReasoningEffort:                   apiKey.Group.MaxReasoningEffort,
			ReasoningEffortMappings:              apiKey.Group.ReasoningEffortMappings,
			PeakRateEnabled:                      apiKey.Group.PeakRateEnabled,
			PeakStart:                            apiKey.Group.PeakStart,
			PeakEnd:                              apiKey.Group.PeakEnd,
			PeakRateMultiplier:                   apiKey.Group.PeakRateMultiplier,
			ProfitControlEnabled:                 apiKey.Group.ProfitControlEnabled,
			ProfitMinMargin:                      apiKey.Group.ProfitMinMargin,
			ProfitSafetyBuffer:                   apiKey.Group.ProfitSafetyBuffer,
			VideoModelPrices:                     NormalizeVideoModelPrices(apiKey.Group.VideoModelPrices),
			SearchPricePer1k:                     apiKey.Group.SearchPricePer1k,
			AudioRealtimePricePerMin:             apiKey.Group.AudioRealtimePricePerMin,
			AudioTTSPricePerMillionChars:         apiKey.Group.AudioTTSPricePerMillionChars,
			AudioSTTPricePerHour:                 apiKey.Group.AudioSTTPricePerHour,
		}
	}

	for _, group := range apiKey.Groups {
		if group == nil {
			continue
		}
		snapshot.Groups = append(snapshot.Groups, apiKeyAuthGroupSnapshotFromGroup(group))
	}
	if len(snapshot.Groups) == 0 && snapshot.Group != nil {
		snapshot.Groups = []*APIKeyAuthGroupSnapshot{snapshot.Group}
	}

	if len(snapshot.GroupIDs) == 0 && snapshot.GroupID != nil && *snapshot.GroupID > 0 {
		snapshot.GroupIDs = []int64{*snapshot.GroupID}
	}
	if snapshot.Group == nil && len(snapshot.Groups) > 0 {
		snapshot.Group = snapshot.Groups[0]
	}
	if snapshot.GroupID == nil && len(snapshot.GroupIDs) > 0 {
		groupID := snapshot.GroupIDs[0]
		snapshot.GroupID = &groupID
	}
	return snapshot
}

func apiKeyAuthGroupSnapshotFromGroup(group *Group) *APIKeyAuthGroupSnapshot {
	if group == nil {
		return nil
	}
	return &APIKeyAuthGroupSnapshot{
		ID:                                   group.ID,
		Name:                                 group.Name,
		Platform:                             group.Platform,
		IsExclusive:                          group.IsExclusive,
		Status:                               group.Status,
		SubscriptionType:                     group.SubscriptionType,
		RateMultiplier:                       group.RateMultiplier,
		LimitedTimeMultiplierEnabled:         group.LimitedTimeMultiplierEnabled,
		LimitedTimeMultiplierCron:            group.LimitedTimeMultiplierCron,
		LimitedTimeMultiplierDurationMinutes: group.LimitedTimeMultiplierDurationMinutes,
		LimitedTimeMultiplierValue:           group.LimitedTimeMultiplierValue,
		DailyLimitUSD:                        group.DailyLimitUSD,
		WeeklyLimitUSD:                       group.WeeklyLimitUSD,
		MonthlyLimitUSD:                      group.MonthlyLimitUSD,
		AllowImageGeneration:                 group.AllowImageGeneration,
		AllowBatchImageGeneration:            group.AllowBatchImageGeneration,
		ImageRateIndependent:                 group.ImageRateIndependent,
		ImageRateMultiplier:                  group.ImageRateMultiplier,
		ImagePrice1K:                         group.ImagePrice1K,
		ImagePrice2K:                         group.ImagePrice2K,
		ImagePrice4K:                         group.ImagePrice4K,
		BatchImageDiscountMultiplier:         group.BatchImageDiscountMultiplier,
		BatchImageHoldMultiplier:             group.BatchImageHoldMultiplier,
		VideoRateIndependent:                 group.VideoRateIndependent,
		VideoRateMultiplier:                  group.VideoRateMultiplier,
		VideoPrice480P:                       group.VideoPrice480P,
		VideoPrice720P:                       group.VideoPrice720P,
		VideoPrice1080P:                      group.VideoPrice1080P,
		WebSearchPricePerCall:                group.WebSearchPricePerCall,
		ClaudeCodeOnly:                       group.ClaudeCodeOnly,
		FallbackGroupID:                      group.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest:      group.FallbackGroupIDOnInvalidRequest,
		ModelRouting:                         group.ModelRouting,
		ModelRoutingEnabled:                  group.ModelRoutingEnabled,
		MCPXMLInject:                         group.MCPXMLInject,
		SupportedModelScopes:                 group.SupportedModelScopes,
		AllowMessagesDispatch:                group.AllowMessagesDispatch,
		AllowLive:                            group.AllowLive,
		RequireOAuthOnly:                     group.RequireOAuthOnly,
		RequirePrivacySet:                    group.RequirePrivacySet,
		DefaultMappedModel:                   group.DefaultMappedModel,
		MessagesDispatchModelConfig:          group.MessagesDispatchModelConfig,
		ModelsListConfig:                     group.ModelsListConfig,
		RPMLimit:                             group.RPMLimit,
		LimitedTimeRPMLimit:                  group.LimitedTimeRPMLimit,
		LimitedTimeUserConcurrencyLimit:      group.LimitedTimeUserConcurrencyLimit,
		UserConcurrencyLimit:                 group.UserConcurrencyLimit,
		MaxReasoningEffort:                   group.MaxReasoningEffort,
		ReasoningEffortMappings:              group.ReasoningEffortMappings,
		PeakRateEnabled:                      group.PeakRateEnabled,
		PeakStart:                            group.PeakStart,
		PeakEnd:                              group.PeakEnd,
		PeakRateMultiplier:                   group.PeakRateMultiplier,
		ProfitControlEnabled:                 group.ProfitControlEnabled,
		ProfitMinMargin:                      group.ProfitMinMargin,
		ProfitSafetyBuffer:                   group.ProfitSafetyBuffer,
	}
}

func apiKeyAuthGroupFromSnapshot(snapshot *APIKeyAuthGroupSnapshot) *Group {
	if snapshot == nil {
		return nil
	}
	return &Group{
		ID:                                   snapshot.ID,
		Hydrated:                             true,
		Name:                                 snapshot.Name,
		Platform:                             snapshot.Platform,
		IsExclusive:                          snapshot.IsExclusive,
		Status:                               snapshot.Status,
		SubscriptionType:                     snapshot.SubscriptionType,
		RateMultiplier:                       snapshot.RateMultiplier,
		LimitedTimeMultiplierEnabled:         snapshot.LimitedTimeMultiplierEnabled,
		LimitedTimeMultiplierCron:            snapshot.LimitedTimeMultiplierCron,
		LimitedTimeMultiplierDurationMinutes: snapshot.LimitedTimeMultiplierDurationMinutes,
		LimitedTimeMultiplierValue:           snapshot.LimitedTimeMultiplierValue,
		DailyLimitUSD:                        snapshot.DailyLimitUSD,
		WeeklyLimitUSD:                       snapshot.WeeklyLimitUSD,
		MonthlyLimitUSD:                      snapshot.MonthlyLimitUSD,
		AllowImageGeneration:                 snapshot.AllowImageGeneration,
		AllowBatchImageGeneration:            snapshot.AllowBatchImageGeneration,
		ImageRateIndependent:                 snapshot.ImageRateIndependent,
		ImageRateMultiplier:                  snapshot.ImageRateMultiplier,
		ImagePrice1K:                         snapshot.ImagePrice1K,
		ImagePrice2K:                         snapshot.ImagePrice2K,
		ImagePrice4K:                         snapshot.ImagePrice4K,
		BatchImageDiscountMultiplier:         snapshot.BatchImageDiscountMultiplier,
		BatchImageHoldMultiplier:             snapshot.BatchImageHoldMultiplier,
		VideoRateIndependent:                 snapshot.VideoRateIndependent,
		VideoRateMultiplier:                  snapshot.VideoRateMultiplier,
		VideoPrice480P:                       snapshot.VideoPrice480P,
		VideoPrice720P:                       snapshot.VideoPrice720P,
		VideoPrice1080P:                      snapshot.VideoPrice1080P,
		WebSearchPricePerCall:                snapshot.WebSearchPricePerCall,
		ClaudeCodeOnly:                       snapshot.ClaudeCodeOnly,
		FallbackGroupID:                      snapshot.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest:      snapshot.FallbackGroupIDOnInvalidRequest,
		ModelRouting:                         snapshot.ModelRouting,
		ModelRoutingEnabled:                  snapshot.ModelRoutingEnabled,
		MCPXMLInject:                         snapshot.MCPXMLInject,
		SupportedModelScopes:                 snapshot.SupportedModelScopes,
		AllowMessagesDispatch:                snapshot.AllowMessagesDispatch,
		AllowLive:                            snapshot.AllowLive,
		RequireOAuthOnly:                     snapshot.RequireOAuthOnly,
		RequirePrivacySet:                    snapshot.RequirePrivacySet,
		DefaultMappedModel:                   snapshot.DefaultMappedModel,
		MessagesDispatchModelConfig:          snapshot.MessagesDispatchModelConfig,
		ModelsListConfig:                     snapshot.ModelsListConfig,
		RPMLimit:                             snapshot.RPMLimit,
		LimitedTimeRPMLimit:                  snapshot.LimitedTimeRPMLimit,
		LimitedTimeUserConcurrencyLimit:      snapshot.LimitedTimeUserConcurrencyLimit,
		UserConcurrencyLimit:                 snapshot.UserConcurrencyLimit,
		MaxReasoningEffort:                   snapshot.MaxReasoningEffort,
		ReasoningEffortMappings:              snapshot.ReasoningEffortMappings,
		PeakRateEnabled:                      snapshot.PeakRateEnabled,
		PeakStart:                            snapshot.PeakStart,
		PeakEnd:                              snapshot.PeakEnd,
		PeakRateMultiplier:                   snapshot.PeakRateMultiplier,
		ProfitControlEnabled:                 snapshot.ProfitControlEnabled,
		ProfitMinMargin:                      snapshot.ProfitMinMargin,
		ProfitSafetyBuffer:                   snapshot.ProfitSafetyBuffer,
	}
}

func (s *APIKeyService) snapshotToAPIKey(key string, snapshot *APIKeyAuthSnapshot) *APIKey {
	if snapshot == nil {
		return nil
	}
	groupIDs := append([]int64(nil), snapshot.GroupIDs...)
	if len(groupIDs) == 0 && snapshot.GroupID != nil && *snapshot.GroupID > 0 {
		groupIDs = []int64{*snapshot.GroupID}
	}
	if snapshot.GroupID == nil && len(groupIDs) > 0 {
		groupID := groupIDs[0]
		snapshot.GroupID = &groupID
	}

	apiKey := &APIKey{
		ID:          snapshot.APIKeyID,
		UserID:      snapshot.UserID,
		GroupID:     snapshot.GroupID,
		GroupIDs:    groupIDs,
		Key:         key,
		Name:        snapshot.Name,
		Status:      snapshot.Status,
		IPWhitelist: snapshot.IPWhitelist,
		IPBlacklist: snapshot.IPBlacklist,
		Quota:       snapshot.Quota,
		QuotaUsed:   snapshot.QuotaUsed,
		ExpiresAt:   snapshot.ExpiresAt,
		RateLimit5h: snapshot.RateLimit5h,
		RateLimit1d: snapshot.RateLimit1d,
		RateLimit7d: snapshot.RateLimit7d,
		User: &User{
			ID:                              snapshot.User.ID,
			Status:                          snapshot.User.Status,
			Role:                            snapshot.User.Role,
			Balance:                         snapshot.User.Balance,
			Concurrency:                     snapshot.User.Concurrency,
			AllowedGroups:                   append([]int64(nil), snapshot.User.AllowedGroups...),
			TagDerivedGroups:                append([]int64(nil), snapshot.User.TagDerivedGroups...),
			Email:                           snapshot.User.Email,
			Username:                        snapshot.User.Username,
			BalanceNotifyEnabled:            snapshot.User.BalanceNotifyEnabled,
			BalanceNotifyThresholdType:      snapshot.User.BalanceNotifyThresholdType,
			BalanceNotifyThreshold:          snapshot.User.BalanceNotifyThreshold,
			BalanceNotifyExtraEmails:        snapshot.User.BalanceNotifyExtraEmails,
			TotalRecharged:                  snapshot.User.TotalRecharged,
			RPMLimit:                        snapshot.User.RPMLimit,
			UserGroupRPMOverride:            snapshot.User.UserGroupRPMOverride,
			UserGroupLimitedTimeRPMOverride: snapshot.User.UserGroupLimitedTimeRPMOverride,
		},
	}
	if snapshot.Group != nil {
		apiKey.Group = &Group{
			ID:                                   snapshot.Group.ID,
			Hydrated:                             true,
			Name:                                 snapshot.Group.Name,
			Platform:                             snapshot.Group.Platform,
			IsExclusive:                          snapshot.Group.IsExclusive,
			Status:                               snapshot.Group.Status,
			SubscriptionType:                     snapshot.Group.SubscriptionType,
			RateMultiplier:                       snapshot.Group.RateMultiplier,
			LimitedTimeMultiplierEnabled:         snapshot.Group.LimitedTimeMultiplierEnabled,
			LimitedTimeMultiplierCron:            snapshot.Group.LimitedTimeMultiplierCron,
			LimitedTimeMultiplierDurationMinutes: snapshot.Group.LimitedTimeMultiplierDurationMinutes,
			LimitedTimeMultiplierValue:           snapshot.Group.LimitedTimeMultiplierValue,
			DailyLimitUSD:                        snapshot.Group.DailyLimitUSD,
			WeeklyLimitUSD:                       snapshot.Group.WeeklyLimitUSD,
			MonthlyLimitUSD:                      snapshot.Group.MonthlyLimitUSD,
			AllowImageGeneration:                 snapshot.Group.AllowImageGeneration,
			AllowBatchImageGeneration:            snapshot.Group.AllowBatchImageGeneration,
			ImageRateIndependent:                 snapshot.Group.ImageRateIndependent,
			ImageRateMultiplier:                  snapshot.Group.ImageRateMultiplier,
			ImagePrice1K:                         snapshot.Group.ImagePrice1K,
			ImagePrice2K:                         snapshot.Group.ImagePrice2K,
			ImagePrice4K:                         snapshot.Group.ImagePrice4K,
			BatchImageDiscountMultiplier:         snapshot.Group.BatchImageDiscountMultiplier,
			BatchImageHoldMultiplier:             snapshot.Group.BatchImageHoldMultiplier,
			VideoRateIndependent:                 snapshot.Group.VideoRateIndependent,
			VideoRateMultiplier:                  snapshot.Group.VideoRateMultiplier,
			VideoPrice480P:                       snapshot.Group.VideoPrice480P,
			VideoPrice720P:                       snapshot.Group.VideoPrice720P,
			VideoPrice1080P:                      snapshot.Group.VideoPrice1080P,
			WebSearchPricePerCall:                snapshot.Group.WebSearchPricePerCall,
			ClaudeCodeOnly:                       snapshot.Group.ClaudeCodeOnly,
			FallbackGroupID:                      snapshot.Group.FallbackGroupID,
			FallbackGroupIDOnInvalidRequest:      snapshot.Group.FallbackGroupIDOnInvalidRequest,
			ModelRouting:                         snapshot.Group.ModelRouting,
			ModelRoutingEnabled:                  snapshot.Group.ModelRoutingEnabled,
			MCPXMLInject:                         snapshot.Group.MCPXMLInject,
			SupportedModelScopes:                 snapshot.Group.SupportedModelScopes,
			AllowMessagesDispatch:                snapshot.Group.AllowMessagesDispatch,
			AllowLive:                            snapshot.Group.AllowLive,
			RequireOAuthOnly:                     snapshot.Group.RequireOAuthOnly,
			RequirePrivacySet:                    snapshot.Group.RequirePrivacySet,
			DefaultMappedModel:                   snapshot.Group.DefaultMappedModel,
			MessagesDispatchModelConfig:          snapshot.Group.MessagesDispatchModelConfig,
			ModelsListConfig:                     snapshot.Group.ModelsListConfig,
			RPMLimit:                             snapshot.Group.RPMLimit,
			LimitedTimeRPMLimit:                  snapshot.Group.LimitedTimeRPMLimit,
			LimitedTimeUserConcurrencyLimit:      snapshot.Group.LimitedTimeUserConcurrencyLimit,
			UserConcurrencyLimit:                 snapshot.Group.UserConcurrencyLimit,
			MaxReasoningEffort:                   snapshot.Group.MaxReasoningEffort,
			ReasoningEffortMappings:              snapshot.Group.ReasoningEffortMappings,
			PeakRateEnabled:                      snapshot.Group.PeakRateEnabled,
			PeakStart:                            snapshot.Group.PeakStart,
			PeakEnd:                              snapshot.Group.PeakEnd,
			PeakRateMultiplier:                   snapshot.Group.PeakRateMultiplier,
			ProfitControlEnabled:                 snapshot.Group.ProfitControlEnabled,
			ProfitMinMargin:                      snapshot.Group.ProfitMinMargin,
			ProfitSafetyBuffer:                   snapshot.Group.ProfitSafetyBuffer,
			VideoModelPrices:                     NormalizeVideoModelPrices(snapshot.Group.VideoModelPrices),
			SearchPricePer1k:                     snapshot.Group.SearchPricePer1k,
			AudioRealtimePricePerMin:             snapshot.Group.AudioRealtimePricePerMin,
			AudioTTSPricePerMillionChars:         snapshot.Group.AudioTTSPricePerMillionChars,
			AudioSTTPricePerHour:                 snapshot.Group.AudioSTTPricePerHour,
		}
	}
	groupSnapshots := snapshot.Groups
	if len(groupSnapshots) == 0 && snapshot.Group != nil {
		groupSnapshots = []*APIKeyAuthGroupSnapshot{snapshot.Group}
	}
	for _, groupSnapshot := range groupSnapshots {
		if groupSnapshot == nil {
			continue
		}
		apiKey.Groups = append(apiKey.Groups, apiKeyAuthGroupFromSnapshot(groupSnapshot))
	}
	if apiKey.Group == nil && len(apiKey.Groups) > 0 {
		apiKey.Group = apiKey.Groups[0]
	}
	s.compileAPIKeyIPRules(apiKey)
	return apiKey
}
