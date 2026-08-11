package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// applyAPIKeyGroupAttempt materializes a target group without mutating the
// authenticated API key. The returned key is safe to capture in usage work.
func applyAPIKeyGroupAttempt(c *gin.Context, apiKey *service.APIKey, group *service.Group, subscription *service.UserSubscription) (*service.APIKey, *service.UserSubscription, error) {
	if c == nil || c.Request == nil {
		return nil, nil, fmt.Errorf("invalid group attempt")
	}
	return applyAPIKeyGroupAttemptWithBase(c, c.Request.Context(), apiKey, group, subscription)
}

func applyAPIKeyGroupAttemptWithBase(c *gin.Context, baseCtx context.Context, apiKey *service.APIKey, group *service.Group, subscription *service.UserSubscription) (*service.APIKey, *service.UserSubscription, error) {
	if c == nil || c.Request == nil || baseCtx == nil || apiKey == nil || group == nil || group.ID <= 0 {
		return nil, nil, fmt.Errorf("invalid group attempt")
	}
	if group.Status == "deleted" || !group.IsActive() {
		return nil, nil, fmt.Errorf("group %d is unavailable", group.ID)
	}
	if apiKey.User != nil && !group.IsSubscriptionType() && !apiKey.User.CanBindGroup(group.ID, group.IsExclusive) {
		return nil, nil, fmt.Errorf("group %d is not allowed", group.ID)
	}

	attemptAPIKey := cloneAPIKeyWithGroup(apiKey, group)
	attemptAPIKey.GroupIDs = append([]int64(nil), apiKey.GroupIDs...)
	if len(attemptAPIKey.GroupIDs) == 0 {
		attemptAPIKey.GroupIDs = []int64{group.ID}
	}
	attemptAPIKey.Groups = append([]*service.Group(nil), apiKey.Groups...)
	if len(attemptAPIKey.Groups) == 0 {
		attemptAPIKey.Groups = []*service.Group{group}
	}

	ctx := context.WithValue(baseCtx, ctxkey.Group, group)
	c.Request = c.Request.WithContext(ctx)
	c.Set(string(middleware.ContextKeyAPIKey), attemptAPIKey)
	if subscription != nil {
		c.Set(string(middleware.ContextKeySubscription), subscription)
	} else {
		c.Set(string(middleware.ContextKeySubscription), (*service.UserSubscription)(nil))
	}
	return attemptAPIKey, subscription, nil
}

func resolveOrderedAPIKeyGroups(ctx context.Context, gatewayService *service.GatewayService, apiKey *service.APIKey) ([]*service.Group, error) {
	if apiKey == nil {
		return nil, errors.New("api key is nil")
	}
	if gatewayService == nil {
		return nil, service.ErrGroupNotFound
	}
	if len(apiKey.GroupIDs) == 0 && apiKey.Group != nil {
		return []*service.Group{apiKey.Group}, nil
	}
	groupIDs := apiKey.GroupIDs
	if len(groupIDs) == 0 && apiKey.GroupID != nil {
		groupIDs = []int64{*apiKey.GroupID}
	}
	if len(groupIDs) == 0 {
		return nil, service.ErrGroupNotFound
	}
	groups := make([]*service.Group, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, err := gatewayService.ResolveGroupByID(ctx, groupID)
		if err != nil {
			continue
		}
		groups = append(groups, group)
	}
	if len(groups) == 0 {
		return nil, service.ErrGroupNotFound
	}
	return groups, nil
}

func resolveOrderedOpenAIGatewayGroups(apiKey *service.APIKey) ([]*service.Group, error) {
	if apiKey == nil {
		return nil, errors.New("api key is nil")
	}
	groupIDs := append([]int64(nil), apiKey.GroupIDs...)
	if len(groupIDs) == 0 && apiKey.GroupID != nil {
		groupIDs = []int64{*apiKey.GroupID}
	}
	if len(groupIDs) == 0 && apiKey.Group != nil {
		return []*service.Group{apiKey.Group}, nil
	}
	if len(groupIDs) == 0 {
		return nil, service.ErrGroupNotFound
	}

	groupsByID := make(map[int64]*service.Group, len(apiKey.Groups)+1)
	if apiKey.Group != nil {
		groupsByID[apiKey.Group.ID] = apiKey.Group
	}
	for _, group := range apiKey.Groups {
		if group != nil {
			groupsByID[group.ID] = group
		}
	}

	groups := make([]*service.Group, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, ok := groupsByID[groupID]
		if !ok || group == nil {
			return nil, fmt.Errorf("group %d is not materialized in API key: %w", groupID, service.ErrGroupNotFound)
		}
		groups = append(groups, group)
	}
	return groups, nil
}

func isRetryableGroupBillingError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, service.ErrSubscriptionNotFound) ||
		errors.Is(err, service.ErrSubscriptionInvalid) ||
		errors.Is(err, service.ErrSubscriptionExpired) ||
		errors.Is(err, service.ErrSubscriptionSuspended) ||
		errors.Is(err, service.ErrDailyLimitExceeded) ||
		errors.Is(err, service.ErrWeeklyLimitExceeded) ||
		errors.Is(err, service.ErrMonthlyLimitExceeded) ||
		errors.Is(err, service.ErrGroupRPMExceeded)
}
