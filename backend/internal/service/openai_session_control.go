package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
)

type openAISessionControlIdentity struct {
	Hash     string
	Resolved bool
}

type openAISessionControlContextKey struct{}

type openAISessionControlPreferredAccountContextKey struct{}

func attachOpenAISessionControlIdentityToGin(c *gin.Context, rawSessionID string) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := c.Request.Context()
	if existing, ok := ctx.Value(openAISessionControlContextKey{}).(openAISessionControlIdentity); ok &&
		existing.Resolved && existing.Hash != "" && strings.TrimSpace(rawSessionID) == "" {
		return
	}
	c.Request = c.Request.WithContext(withOpenAISessionControlIdentity(ctx, getAPIKeyIDFromContext(c), rawSessionID))
}

func withOpenAISessionControlIdentity(ctx context.Context, apiKeyID int64, rawSessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	identity := openAISessionControlIdentity{Resolved: true}
	if rawSessionID = strings.TrimSpace(rawSessionID); rawSessionID != "" {
		identity.Hash = isolateOpenAISessionID(apiKeyID, rawSessionID)
	}
	return context.WithValue(ctx, openAISessionControlContextKey{}, identity)
}

func openAISessionControlIdentityFromContext(ctx context.Context) openAISessionControlIdentity {
	if ctx == nil {
		return openAISessionControlIdentity{}
	}
	identity, _ := ctx.Value(openAISessionControlContextKey{}).(openAISessionControlIdentity)
	return identity
}

func withOpenAISessionControlPreferredAccountID(ctx context.Context, accountID int64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openAISessionControlPreferredAccountContextKey{}, accountID)
}

func openAISessionControlPreferredAccountIDFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	accountID, _ := ctx.Value(openAISessionControlPreferredAccountContextKey{}).(int64)
	return accountID
}

func (s *OpenAIGatewayService) getOpenAISessionControlPreferredAccountID(ctx context.Context) int64 {
	if s == nil || s.sessionLimitCache == nil {
		return 0
	}
	identity := openAISessionControlIdentityFromContext(ctx)
	if !identity.Resolved || identity.Hash == "" {
		return 0
	}
	accountID, err := s.sessionLimitCache.GetOpenAIStagedSessionAccountID(ctx, identity.Hash)
	if err != nil {
		slog.Warn("openai_session_control_owner_lookup_failed", "err", err)
		return 0
	}
	return accountID
}

func (s *OpenAIGatewayService) resolveOpenAISessionControlOwner(ctx context.Context, account *Account) (*Account, error) {
	if account == nil || account.ParentAccountID == nil {
		return account, nil
	}
	parentID := *account.ParentAccountID
	if s.schedulerSnapshot != nil {
		if parent, err := s.schedulerSnapshot.GetAccount(ctx, parentID); err == nil && parent != nil {
			return parent, nil
		}
	}
	if s.accountRepo == nil {
		return nil, fmt.Errorf("OpenAI SessionID control parent account repository is unavailable")
	}
	parent, err := s.accountRepo.GetByID(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("resolve OpenAI SessionID control parent account %d: %w", parentID, err)
	}
	return parent, nil
}

func (s *OpenAIGatewayService) registerOpenAISessionControl(ctx context.Context, account *Account) (bool, string) {
	if s == nil || account == nil || normalizeOpenAICompatiblePlatform(account.Platform) != PlatformOpenAI {
		return true, ""
	}
	identity := openAISessionControlIdentityFromContext(ctx)
	if !identity.Resolved {
		// Internal probes/model discovery do not carry a gateway client identity.
		return true, ""
	}
	owner, err := s.resolveOpenAISessionControlOwner(ctx, account)
	if err != nil {
		slog.Error("openai_session_control_owner_resolve_failed", "account_id", account.ID, "err", err)
		return false, "owner_resolve_failed"
	}
	if owner == nil || !owner.IsOpenAISessionControlEnabled() {
		return true, ""
	}
	if identity.Hash == "" {
		return false, "session_id_missing"
	}
	if s.sessionLimitCache == nil {
		slog.Error("openai_session_control_cache_unavailable", "account_id", owner.ID)
		return false, "cache_unavailable"
	}
	allowed, err := s.sessionLimitCache.RegisterOpenAISessionID(
		ctx,
		owner.ID,
		identity.Hash,
		owner.GetOpenAISessionMaxCount(),
		owner.GetOpenAISessionIdleTimeout(),
		owner.IsOpenAISessionSlotRotationEnabled(),
	)
	if err != nil {
		slog.Error("openai_session_control_register_failed", "account_id", owner.ID, "err", err)
		return false, "cache_error"
	}
	if !allowed {
		return false, "session_limit_reached"
	}
	return true, ""
}
