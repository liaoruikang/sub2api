package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SeedanceHandler struct {
	seedance          *service.SeedanceService
	billing           *service.BillingCacheService
	apiKeys           *service.APIKeyService
	concurrencyHelper *ConcurrencyHelper
	contentModeration *service.ContentModerationService
	coordinator       *securityaudit.Coordinator
}

func NewSeedanceHandler(
	seedance *service.SeedanceService,
	billing *service.BillingCacheService,
	apiKeys *service.APIKeyService,
	concurrency *service.ConcurrencyService,
	contentModeration *service.ContentModerationService,
	coordinator *securityaudit.Coordinator,
) *SeedanceHandler {
	return &SeedanceHandler{
		seedance:          seedance,
		billing:           billing,
		apiKeys:           apiKeys,
		concurrencyHelper: NewConcurrencyHelper(concurrency, SSEPingFormatNone, 0),
		contentModeration: contentModeration,
		coordinator:       coordinator,
	}
}

func (h *SeedanceHandler) checkSecurityAudit(c *gin.Context, apiKey *service.APIKey, subject middleware.AuthSubject, model string, body []byte) *securityaudit.Decision {
	if h == nil {
		return nil
	}
	return runSecurityAudit(c, zap.NewNop(), h.coordinator, h.contentModeration, apiKey, subject, service.ContentModerationProtocolOpenAIImages, model, body, "http")
}

type seedanceRequestContext struct {
	apiKey       *service.APIKey
	subject      middleware.AuthSubject
	subscription *service.UserSubscription
	release      func()
}

func (h *SeedanceHandler) beginMutation(c *gin.Context) (*seedanceRequestContext, bool) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		h.writeError(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil, false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || apiKey.User == nil {
		h.writeError(c, http.StatusInternalServerError, "api_error", "User context not found")
		return nil, false
	}
	if apiKey.Group == nil || apiKey.GroupID == nil || apiKey.Group.Platform != service.PlatformSeedance {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance API is not supported for this group")
		return nil, false
	}

	streamStarted := false
	releases := make([]func(), 0, 2)
	if limit := apiKey.Group.EffectiveUserConcurrencyLimitAt(time.Now()); limit > 0 {
		release, err := h.concurrencyHelper.AcquireGroupUserSlotWithWait(c, apiKey.Group.ID, subject.UserID, limit, false, &streamStarted)
		if err != nil {
			h.writeConcurrencyError(c, err)
			return nil, false
		}
		if release != nil {
			releases = append(releases, release)
		}
	}
	userRelease, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, false, &streamStarted)
	if err != nil {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
		h.writeConcurrencyError(c, err)
		return nil, false
	}
	if userRelease != nil {
		releases = append(releases, userRelease)
	}

	subscription, _ := middleware.GetSubscriptionFromContext(c)
	if err := h.billing.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.PlatformSeedance); err != nil {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.writeError(c, status, code, message)
		return nil, false
	}
	return &seedanceRequestContext{
		apiKey: apiKey, subject: subject, subscription: subscription,
		release: func() {
			for i := len(releases) - 1; i >= 0; i-- {
				releases[i]()
			}
		},
	}, true
}

func (h *SeedanceHandler) readContext(c *gin.Context) (*seedanceRequestContext, bool) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		h.writeError(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil, false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		h.writeError(c, http.StatusInternalServerError, "api_error", "User context not found")
		return nil, false
	}
	if apiKey.Group == nil || apiKey.GroupID == nil || apiKey.Group.Platform != service.PlatformSeedance {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance API is not supported for this group")
		return nil, false
	}
	subscription, _ := middleware.GetSubscriptionFromContext(c)
	return &seedanceRequestContext{apiKey: apiKey, subject: subject, subscription: subscription}, true
}

func (h *SeedanceHandler) readJSONBody(c *gin.Context) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.writeError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return nil, nil, false
	}
	var payload map[string]any
	if len(body) == 0 || json.Unmarshal(body, &payload) != nil {
		h.writeError(c, http.StatusBadRequest, "invalid_request_error", "Request body must be a JSON object")
		return nil, nil, false
	}
	return body, payload, true
}

func apiKeyIDPointer(apiKey *service.APIKey) *int64 {
	if apiKey == nil || apiKey.ID <= 0 {
		return nil
	}
	id := apiKey.ID
	return &id
}

func (h *SeedanceHandler) acquireBoundAccount(c *gin.Context, accountID int64) (*service.Account, func(), bool) {
	account, err := h.seedance.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		h.writeError(c, http.StatusBadGateway, "upstream_account_error", err.Error())
		return nil, nil, false
	}
	streamStarted := false
	release, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, account.Concurrency, 30*time.Second, false, &streamStarted)
	if err != nil {
		h.writeConcurrencyError(c, err)
		return nil, nil, false
	}
	return account, release, true
}

func (h *SeedanceHandler) forwardBound(c *gin.Context, accountID int64, method, path string, body []byte) (*service.SeedanceUpstreamResponse, bool) {
	account, release, ok := h.acquireBoundAccount(c, accountID)
	if !ok {
		return nil, false
	}
	if release != nil {
		defer release()
	}
	resp, err := h.seedance.Forward(c.Request.Context(), account, method, path, c.Request.URL.RawQuery, body, c.Request.Header)
	if err != nil {
		h.writeError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return nil, false
	}
	return resp, true
}

func (h *SeedanceHandler) forwardNew(c *gin.Context, rc *seedanceRequestContext, model, method, path string, body []byte) (*service.SeedanceUpstreamResponse, *service.Account, bool) {
	excluded := make(map[int64]struct{})
	for attempt := 0; attempt < 3; attempt++ {
		selection, err := h.seedance.SelectAccount(c.Request.Context(), rc.apiKey.GroupID, model, excluded, rc.subject.UserID)
		if err != nil || selection == nil || selection.Account == nil {
			if err == nil {
				err = service.ErrNoAvailableAccounts
			}
			h.writeError(c, http.StatusServiceUnavailable, "no_available_accounts", err.Error())
			return nil, nil, false
		}
		account := selection.Account
		release := selection.ReleaseFunc
		if !selection.Acquired {
			if selection.WaitPlan == nil {
				h.writeError(c, http.StatusTooManyRequests, "concurrency_limit", "No Seedance account concurrency slot is available")
				return nil, nil, false
			}
			streamStarted := false
			release, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, account.Concurrency, selection.WaitPlan.Timeout, false, &streamStarted)
			if err != nil {
				h.writeConcurrencyError(c, err)
				return nil, nil, false
			}
		}
		resp, forwardErr := h.seedance.Forward(c.Request.Context(), account, method, path, c.Request.URL.RawQuery, body, c.Request.Header)
		if release != nil {
			release()
		}
		if forwardErr != nil {
			// A transport error has an ambiguous outcome for POST; do not risk duplicate creation.
			h.writeError(c, http.StatusBadGateway, "upstream_error", forwardErr.Error())
			return nil, nil, false
		}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < http.StatusInternalServerError {
			return resp, account, true
		}
		excluded[account.ID] = struct{}{}
		if attempt == 2 {
			return resp, account, true
		}
	}
	return nil, nil, false
}

func (h *SeedanceHandler) CreateAssetGroup(c *gin.Context) {
	h.createUnboundResource(c, service.SeedanceResourceAssetGroup, service.SeedanceChannelGroup, "/v1/asset-groups", "dreamina-seedance-2-0-260128")
}

func (h *SeedanceHandler) CreateSDAsset(c *gin.Context) {
	h.createUnboundResource(c, service.SeedanceResourceAsset, service.SeedanceChannelSD, "/v1/sd/assets", "dreamina-seedance-2-0-hc")
}

func (h *SeedanceHandler) CreateDoubaoAsset(c *gin.Context) {
	h.createUnboundResource(c, service.SeedanceResourceAsset, service.SeedanceChannelDoubao, "/v1/doubao-sd-1/assets", "doubao-seedance-2-0-260128-a")
}

func (h *SeedanceHandler) createUnboundResource(c *gin.Context, resourceType, channel, path, model string) {
	rc, ok := h.beginMutation(c)
	if !ok {
		return
	}
	defer rc.release()
	body, _, ok := h.readJSONBody(c)
	if !ok {
		return
	}
	resp, account, ok := h.forwardNew(c, rc, model, http.MethodPost, path, body)
	if !ok {
		return
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		resourceID := seedanceResponseResourceID(resp.Body, channel)
		if resourceID == "" {
			h.writeError(c, http.StatusBadGateway, "upstream_response_error", "Seedance response did not include a resource ID")
			return
		}
		resource := &service.SeedanceResource{
			ResourceID: resourceID, ResourceType: resourceType, Channel: channel,
			UserID: rc.subject.UserID, APIKeyID: apiKeyIDPointer(rc.apiKey), GroupID: rc.apiKey.GroupID,
			AccountID: account.ID,
		}
		if err := h.seedance.Repository().CreateResource(c.Request.Context(), resource); err != nil {
			h.writeError(c, http.StatusInternalServerError, "resource_persistence_error", "Failed to persist Seedance resource ownership")
			return
		}
	}
	h.writeUpstream(c, resp)
}

func (h *SeedanceHandler) GetAssetGroup(c *gin.Context) {
	h.getBoundResource(c, service.SeedanceResourceAssetGroup, c.Param("group_id"), service.SeedanceChannelGroup, "/v1/asset-groups/"+c.Param("group_id"))
}

func (h *SeedanceHandler) GetSDAsset(c *gin.Context) {
	id := c.Param("asset_id")
	h.getBoundResource(c, service.SeedanceResourceAsset, id, service.SeedanceChannelSD, "/v1/sd/assets/"+id)
}

func (h *SeedanceHandler) GetDoubaoAsset(c *gin.Context) {
	id := c.Param("asset_id")
	h.getBoundResource(c, service.SeedanceResourceAsset, id, service.SeedanceChannelDoubao, "/v1/doubao-sd-1/assets/"+id)
}

func (h *SeedanceHandler) getBoundResource(c *gin.Context, resourceType, resourceID, expectedChannel, path string) {
	rc, ok := h.readContext(c)
	if !ok {
		return
	}
	resource, err := h.seedance.Repository().GetResource(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), resourceType, resourceID)
	if err != nil || resource.Channel != expectedChannel {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance resource not found")
		return
	}
	resp, ok := h.forwardBound(c, resource.AccountID, http.MethodGet, path, nil)
	if ok {
		h.writeUpstream(c, resp)
	}
}

func (h *SeedanceHandler) CreateAsset(c *gin.Context) {
	rc, ok := h.beginMutation(c)
	if !ok {
		return
	}
	defer rc.release()
	body, payload, ok := h.readJSONBody(c)
	if !ok {
		return
	}
	groupID, _ := payload["group_id"].(string)
	group, err := h.seedance.Repository().GetResource(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), service.SeedanceResourceAssetGroup, groupID)
	if err != nil || group.Channel != service.SeedanceChannelGroup {
		h.writeError(c, http.StatusBadRequest, "invalid_request_error", "group_id is not owned by this API key")
		return
	}
	resp, ok := h.forwardBound(c, group.AccountID, http.MethodPost, "/v1/assets", body)
	if !ok {
		return
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		assetID := seedanceResponseResourceID(resp.Body, service.SeedanceChannelGroup)
		if assetID == "" {
			h.writeError(c, http.StatusBadGateway, "upstream_response_error", "Seedance response did not include an asset ID")
			return
		}
		resource := &service.SeedanceResource{
			ResourceID: assetID, ResourceType: service.SeedanceResourceAsset, Channel: service.SeedanceChannelGroup,
			UserID: rc.subject.UserID, APIKeyID: apiKeyIDPointer(rc.apiKey), GroupID: rc.apiKey.GroupID,
			AccountID: group.AccountID, ParentID: groupID, TaskID: jsonString(payloadFromJSON(resp.Body), "task_id"),
		}
		if err := h.seedance.Repository().CreateResource(c.Request.Context(), resource); err != nil {
			h.writeError(c, http.StatusInternalServerError, "resource_persistence_error", "Failed to persist Seedance asset ownership")
			return
		}
	}
	h.writeUpstream(c, resp)
}

func (h *SeedanceHandler) GetAsset(c *gin.Context) { h.mutateBoundResource(c, "/v1/assets/get", false) }
func (h *SeedanceHandler) UpdateAsset(c *gin.Context) {
	h.mutateBoundResource(c, "/v1/assets/update", true)
}
func (h *SeedanceHandler) UpdateAssetGroup(c *gin.Context) {
	h.mutateBoundGroup(c, "/v1/asset-groups/update")
}

func (h *SeedanceHandler) mutateBoundResource(c *gin.Context, path string, mutation bool) {
	var rc *seedanceRequestContext
	var ok bool
	if mutation {
		rc, ok = h.beginMutation(c)
	} else {
		rc, ok = h.readContext(c)
	}
	if !ok {
		return
	}
	if rc.release != nil {
		defer rc.release()
	}
	body, payload, ok := h.readJSONBody(c)
	if !ok {
		return
	}
	assetID, _ := payload["asset_id"].(string)
	resource, err := h.seedance.Repository().GetResource(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), service.SeedanceResourceAsset, assetID)
	if err != nil || resource.Channel != service.SeedanceChannelGroup {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance asset not found")
		return
	}
	resp, ok := h.forwardBound(c, resource.AccountID, http.MethodPost, path, body)
	if ok {
		h.writeUpstream(c, resp)
	}
}

func (h *SeedanceHandler) mutateBoundGroup(c *gin.Context, path string) {
	rc, ok := h.beginMutation(c)
	if !ok {
		return
	}
	defer rc.release()
	body, payload, ok := h.readJSONBody(c)
	if !ok {
		return
	}
	groupID, _ := payload["group_id"].(string)
	resource, err := h.seedance.Repository().GetResource(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), service.SeedanceResourceAssetGroup, groupID)
	if err != nil || resource.Channel != service.SeedanceChannelGroup {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance asset group not found")
		return
	}
	resp, ok := h.forwardBound(c, resource.AccountID, http.MethodPost, path, body)
	if ok {
		h.writeUpstream(c, resp)
	}
}

func (h *SeedanceHandler) GenerateVideo(c *gin.Context) {
	rc, ok := h.beginMutation(c)
	if !ok {
		return
	}
	defer rc.release()
	body, payload, ok := h.readJSONBody(c)
	if !ok {
		return
	}
	model, _ := payload["model"].(string)
	if !service.IsSeedanceModel(model) {
		h.writeError(c, http.StatusBadRequest, "invalid_request_error", "Unsupported Seedance model")
		return
	}
	if !service.GroupAllowsImageGeneration(rc.apiKey.Group) {
		h.writeError(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
		return
	}
	resolution := jsonString(payload, "resolution")
	if rc.apiKey.Group.GetVideoPriceForModel(model, resolution) == nil {
		h.writeError(c, http.StatusServiceUnavailable, "billing_configuration_error", "Seedance video price is not configured for this model and resolution")
		return
	}
	if moderationBody := seedanceModerationBody(payload); len(moderationBody) > 0 {
		decision := h.checkSecurityAudit(c, rc.apiKey, rc.subject, model, moderationBody)
		if decision != nil && !decision.AllowNextStage {
			h.writeError(c, securityAuditStatus(decision), securityAuditErrorCode(decision), securityAuditMessage(decision))
			return
		}
	}
	assetIDs := collectSeedanceAssetIDs(payload)
	var resp *service.SeedanceUpstreamResponse
	var account *service.Account
	if len(assetIDs) > 0 {
		var accountID int64
		var channel string
		for _, assetID := range assetIDs {
			resource, err := h.seedance.Repository().GetResource(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), service.SeedanceResourceAsset, assetID)
			if err != nil {
				h.writeError(c, http.StatusBadRequest, "invalid_request_error", "Referenced Seedance asset is not owned by this API key")
				return
			}
			if accountID == 0 {
				accountID, channel = resource.AccountID, resource.Channel
			} else if accountID != resource.AccountID {
				h.writeError(c, http.StatusBadRequest, "invalid_request_error", "All asset:// references must belong to the same Seedance account")
				return
			}
		}
		if !seedanceChannelSupportsModel(channel, model) {
			h.writeError(c, http.StatusBadRequest, "invalid_request_error", "Referenced asset channel is incompatible with the requested model")
			return
		}
		var accountErr error
		account, accountErr = h.seedance.GetAccount(c.Request.Context(), accountID)
		if accountErr != nil {
			h.writeError(c, http.StatusBadGateway, "upstream_account_error", accountErr.Error())
			return
		}
		resp, ok = h.forwardBound(c, accountID, http.MethodPost, "/v1/video/generate", body)
	} else {
		resp, account, ok = h.forwardNew(c, rc, model, http.MethodPost, "/v1/video/generate", body)
	}
	if !ok {
		return
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		taskMap := jsonObject(payloadFromJSON(resp.Body), "task")
		taskID := jsonString(taskMap, "id")
		if taskID == "" {
			h.writeError(c, http.StatusBadGateway, "upstream_response_error", "Seedance response did not include task.id")
			return
		}
		task := &service.SeedanceVideoTask{
			TaskID: taskID, UserID: rc.subject.UserID, APIKeyID: apiKeyIDPointer(rc.apiKey), GroupID: rc.apiKey.GroupID,
			AccountID: account.ID, Model: model, Status: jsonString(taskMap, "status"),
			DurationSeconds: jsonFloat(payload, "duration"), Resolution: resolution,
			RequestBody: append([]byte(nil), body...), ResponseBody: append([]byte(nil), resp.Body...),
		}
		if err := h.seedance.Repository().CreateTask(c.Request.Context(), task); err != nil {
			h.writeError(c, http.StatusInternalServerError, "task_persistence_error", "Failed to persist Seedance task ownership")
			return
		}
	}
	h.writeUpstream(c, resp)
}

func (h *SeedanceHandler) GetVideoTask(c *gin.Context) {
	rc, ok := h.readContext(c)
	if !ok {
		return
	}
	taskID := c.Param("task_id")
	task, err := h.seedance.Repository().GetTask(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), taskID)
	if err != nil {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance task not found")
		return
	}
	resp, ok := h.forwardBound(c, task.AccountID, http.MethodGet, "/v1/video/tasks/"+taskID, nil)
	if !ok {
		return
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		h.refreshTaskFromResponse(c, rc, task, resp.Body)
	}
	h.writeUpstream(c, resp)
}

// GetVideoUsage forwards the documented per-task usage endpoint while keeping
// task ownership scoped to the calling user and API key.
func (h *SeedanceHandler) GetVideoUsage(c *gin.Context) {
	rc, ok := h.readContext(c)
	if !ok {
		return
	}
	taskID := c.Param("task_id")
	task, err := h.seedance.Repository().GetTask(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), taskID)
	if err != nil {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance task not found")
		return
	}
	resp, ok := h.forwardBound(c, task.AccountID, http.MethodGet, "/v1/video/usages/"+taskID, nil)
	if ok {
		h.writeUpstream(c, resp)
	}
}

// ListUserVideoTasks exposes Seedance tasks to the authenticated web console.
// The native endpoint is API-key scoped; this companion path is user scoped and
// is only used after ownership is checked by the repository.
func (h *SeedanceHandler) ListUserVideoTasks(ctx context.Context, userID int64, apiKeyID *int64, page, pageSize int) (*service.SeedanceVideoTaskList, error) {
	if apiKeyID != nil {
		return h.seedance.Repository().ListTasks(ctx, userID, apiKeyID, page, pageSize)
	}
	keys, _, err := h.apiKeys.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, service.APIKeyListFilters{})
	if err != nil {
		return nil, err
	}
	result := &service.SeedanceVideoTaskList{}
	for i := range keys {
		if keys[i].Group == nil || keys[i].Group.Platform != service.PlatformSeedance {
			continue
		}
		keyID := keys[i].ID
		list, listErr := h.seedance.Repository().ListTasks(ctx, userID, &keyID, 1, 100)
		if listErr != nil || list == nil {
			continue
		}
		result.Total += list.Total
		result.Items = append(result.Items, list.Items...)
	}
	sort.SliceStable(result.Items, func(i, j int) bool { return result.Items[i].CreatedAt.After(result.Items[j].CreatedAt) })
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(result.Items) {
		result.Items = nil
		return result, nil
	}
	end := start + pageSize
	if end > len(result.Items) {
		end = len(result.Items)
	}
	result.Items = result.Items[start:end]
	return result, nil
}

func (h *SeedanceHandler) GetUserVideoTask(ctx context.Context, userID int64, apiKeyID *int64, taskID string) (*service.SeedanceVideoTask, error) {
	return h.seedance.Repository().GetTask(ctx, userID, apiKeyID, strings.TrimSpace(taskID))
}

// RefreshUserVideoTasks polls Seedance tasks from the web console while keeping
// the same ownership and billing path as the native API.
func (h *SeedanceHandler) RefreshUserVideoTasks(c *gin.Context, userID int64, requestIDs []string, activeOnly bool, limit int) ([]*service.SeedanceVideoTask, error) {
	ctx := c.Request.Context()
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	apiKeyIDSet := make(map[int64]struct{})
	keys, _, keyErr := h.apiKeys.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, service.APIKeyListFilters{})
	if keyErr == nil {
		for i := range keys {
			if keys[i].Group != nil && keys[i].Group.Platform == service.PlatformSeedance {
				apiKeyIDSet[keys[i].ID] = struct{}{}
			}
		}
	}
	if len(apiKeyIDSet) == 0 && len(requestIDs) == 0 {
		return nil, nil
	}

	result := make([]*service.SeedanceVideoTask, 0, limit)
	seen := make(map[string]struct{})
	appendTask := func(task *service.SeedanceVideoTask) {
		if task == nil || task.UserID != userID || task.TaskID == "" {
			return
		}
		if _, ok := seen[task.TaskID]; ok {
			return
		}
		seen[task.TaskID] = struct{}{}
		result = append(result, task)
	}
	if len(requestIDs) > 0 {
		for _, requestID := range requestIDs {
			requestID = strings.TrimSpace(requestID)
			if requestID == "" {
				continue
			}
			for keyID := range apiKeyIDSet {
				keyPtr := keyID
				if task, err := h.seedance.Repository().GetTask(ctx, userID, &keyPtr, requestID); err == nil {
					appendTask(task)
					break
				}
			}
		}
	} else {
		for keyID := range apiKeyIDSet {
			id := keyID
			list, err := h.seedance.Repository().ListTasks(ctx, userID, &id, 1, limit)
			if err != nil {
				continue
			}
			for i := range list.Items {
				task := &list.Items[i]
				if !isSeedanceTaskTerminal(task.Status) {
					appendTask(task)
				}
			}
		}
	}
	for _, task := range result {
		if task == nil || isSeedanceTaskTerminal(task.Status) {
			continue
		}
		account, err := h.seedance.GetAccount(ctx, task.AccountID)
		if err != nil {
			continue
		}
		resp, err := h.seedance.Forward(ctx, account, http.MethodGet, "/v1/video/tasks/"+url.PathEscape(task.TaskID), "", nil, c.Request.Header)
		if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}
		if task.APIKeyID == nil {
			continue
		}
		apiKey, keyErr := h.apiKeys.GetByID(ctx, *task.APIKeyID)
		if keyErr != nil || apiKey == nil || apiKey.UserID != userID {
			continue
		}
		rc := &seedanceRequestContext{apiKey: apiKey, subject: middleware.AuthSubject{UserID: userID}}
		h.refreshTaskFromResponse(c, rc, task, resp.Body)
	}
	return result, nil
}

func isSeedanceTaskTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "succeeded", "success", "done", "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func (h *SeedanceHandler) ListVideoTasks(c *gin.Context) {
	rc, ok := h.readContext(c)
	if !ok {
		return
	}
	page := queryPositiveInt(c, "page", 1)
	pageSize := queryPositiveInt(c, "pageSize", 20)
	list, err := h.seedance.Repository().ListTasks(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), page, pageSize)
	if err != nil {
		h.writeError(c, http.StatusInternalServerError, "api_error", "Failed to list Seedance tasks")
		return
	}
	tasks := make([]json.RawMessage, 0, len(list.Items))
	for i := range list.Items {
		task := &list.Items[i]
		if task.Status == "pending" || task.Status == "processing" {
			if account, err := h.seedance.GetAccount(c.Request.Context(), task.AccountID); err == nil {
				if release, acquired, _ := h.concurrencyHelper.TryAcquireAccountSlot(c.Request.Context(), account.ID, account.Concurrency); acquired {
					if resp, forwardErr := h.seedance.Forward(c.Request.Context(), account, http.MethodGet, "/v1/video/tasks/"+task.TaskID, "", nil, c.Request.Header); forwardErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
						h.refreshTaskFromResponse(c, rc, task, resp.Body)
					}
					release()
				}
			}
		}
		tasks = append(tasks, seedanceTaskJSON(task))
	}
	totalPages := int(math.Ceil(float64(list.Total) / float64(max(pageSize, 1))))
	c.JSON(http.StatusOK, gin.H{"tasks": tasks, "total": list.Total, "totalPages": totalPages})
}

func (h *SeedanceHandler) GetVideoFile(c *gin.Context) {
	rc, ok := h.readContext(c)
	if !ok {
		return
	}
	taskID := c.Param("task_id")
	task, err := h.seedance.Repository().GetTask(c.Request.Context(), rc.subject.UserID, apiKeyIDPointer(rc.apiKey), taskID)
	if err != nil {
		h.writeError(c, http.StatusNotFound, "not_found_error", "Seedance task not found")
		return
	}
	filePath := strings.TrimPrefix(c.Param("file_path"), "/")
	if strings.Contains(filePath, "..") {
		h.writeError(c, http.StatusBadRequest, "invalid_request_error", "Invalid Seedance file path")
		return
	}
	path := "/v1/video/files/" + taskID
	if filePath != "" {
		path += "/" + filePath
	}
	resp, ok := h.forwardBound(c, task.AccountID, http.MethodGet, path, nil)
	if ok {
		h.writeUpstream(c, resp)
	}
}

func (h *SeedanceHandler) refreshTaskFromResponse(c *gin.Context, rc *seedanceRequestContext, task *service.SeedanceVideoTask, body []byte) {
	root := payloadFromJSON(body)
	taskMap := jsonObject(root, "task")
	if taskMap == nil {
		return
	}
	task.Status = jsonString(taskMap, "status")
	if model := jsonString(taskMap, "model"); model != "" {
		task.Model = model
	}
	if duration := jsonFloat(taskMap, "duration_seconds"); duration > 0 {
		task.DurationSeconds = duration
	} else if duration := jsonFloat(taskMap, "duration"); duration > 0 {
		task.DurationSeconds = duration
	}
	if resolution := jsonString(taskMap, "resolution"); resolution != "" {
		task.Resolution = resolution
	}
	if metadata := jsonObject(taskMap, "metadata"); metadata != nil {
		if resolution := jsonString(metadata, "resolution"); resolution != "" {
			task.Resolution = resolution
		}
	}
	if taskError := jsonObject(taskMap, "error"); taskError != nil {
		task.LastErrorCode = jsonString(taskError, "code")
		task.LastErrorMessage = jsonString(taskError, "message")
	}
	task.ResponseBody = append(task.ResponseBody[:0], body...)
	if task.Status == "completed" || task.Status == "failed" {
		now := time.Now()
		task.CompletedAt = &now
	}
	if err := h.seedance.Repository().UpdateTask(c.Request.Context(), task); err != nil {
		return
	}
	if task.Status != "completed" {
		return
	}
	claimed, err := h.seedance.Repository().ClaimTaskBilling(c.Request.Context(), task.TaskID)
	if err != nil || !claimed {
		return
	}
	account, err := h.seedance.GetAccount(c.Request.Context(), task.AccountID)
	if err != nil {
		_ = h.seedance.Repository().ReleaseTaskBilling(c.Request.Context(), task.TaskID)
		return
	}
	result := &service.OpenAIForwardResult{
		RequestID: "seedance-video:" + task.TaskID, ResponseID: task.TaskID,
		Model: task.Model, UpstreamModel: task.Model, VideoCount: 1,
		VideoResolution: task.Resolution, VideoDurationSeconds: int(math.Round(task.DurationSeconds)),
	}
	// Seedance includes usage in the completed task response. Prefer that
	// payload so providers without the optional per-task usage endpoint do not
	// lose the upstream token count. Keep the endpoint as a compatibility
	// fallback for gateways that omit usage from the task response.
	result.Usage = parseSeedanceUsage(body)
	if result.Usage.InputTokens == 0 && result.Usage.OutputTokens == 0 {
		if usageResp, usageErr := h.seedance.Forward(c.Request.Context(), account, http.MethodGet, "/v1/video/usages/"+url.PathEscape(task.TaskID), "", nil, c.Request.Header); usageErr == nil && usageResp.StatusCode >= 200 && usageResp.StatusCode < 300 {
			result.Usage = parseSeedanceUsage(usageResp.Body)
		}
	}
	if task.CompletedAt != nil && !task.CreatedAt.IsZero() {
		if duration := task.CompletedAt.Sub(task.CreatedAt); duration >= 0 {
			result.Duration = duration
		}
	}
	err = h.seedance.UsageService().RecordUsage(c.Request.Context(), &service.OpenAIRecordUsageInput{
		Result: result, APIKey: rc.apiKey, User: rc.apiKey.User, Account: account,
		Subscription: rc.subscription, InboundEndpoint: "/v1/video/tasks/:task_id",
		UpstreamEndpoint: "/v1/video/tasks/" + task.TaskID, UserAgent: c.GetHeader("User-Agent"),
		IPAddress: ip.GetClientIP(c), APIKeyService: h.apiKeys, QuotaPlatform: service.PlatformSeedance,
		ChannelUsageFields: service.ChannelUsageFields{OriginalModel: task.Model},
	})
	if err != nil {
		_ = h.seedance.Repository().ReleaseTaskBilling(c.Request.Context(), task.TaskID)
	}
}

// parseSeedanceUsage accepts both the documented input/output/total token
// spelling and the prompt/completion spelling used by compatible gateways.
// A total-only response is retained as input tokens so the usage table does
// not discard the upstream's reported total (for example 87300).
func parseSeedanceUsage(body []byte) service.OpenAIUsage {
	var root any
	if json.Unmarshal(body, &root) != nil {
		return service.OpenAIUsage{}
	}
	var usage map[string]any
	var find func(any) bool
	find = func(value any) bool {
		m, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for key, candidate := range m {
			if strings.EqualFold(key, "usage") {
				if found, ok := candidate.(map[string]any); ok {
					usage = found
					return true
				}
			}
		}
		for _, key := range []string{"input_tokens", "output_tokens", "prompt_tokens", "completion_tokens", "total_tokens", "tokens"} {
			if _, ok := m[key]; ok {
				usage = m
				return true
			}
		}
		for _, candidate := range m {
			if find(candidate) {
				return true
			}
		}
		return false
	}
	if !find(root) || usage == nil {
		return service.OpenAIUsage{}
	}
	read := func(keys ...string) int {
		for _, key := range keys {
			if value, ok := usage[key]; ok {
				switch number := value.(type) {
				case float64:
					return int(math.Max(0, math.Round(number)))
				case json.Number:
					if parsed, err := number.Int64(); err == nil {
						return int(parsed)
					}
				}
			}
		}
		return 0
	}
	input := read("input_tokens", "prompt_tokens")
	output := read("output_tokens", "completion_tokens")
	if input == 0 && output == 0 {
		input = read("total_tokens", "tokens")
	}
	return service.OpenAIUsage{InputTokens: input, OutputTokens: output}
}

func seedanceTaskJSON(task *service.SeedanceVideoTask) json.RawMessage {
	if task != nil && len(task.ResponseBody) > 0 {
		if taskMap := jsonObject(payloadFromJSON(task.ResponseBody), "task"); taskMap != nil {
			if data, err := json.Marshal(taskMap); err == nil {
				return data
			}
		}
	}
	fallback := map[string]any{
		"id": task.TaskID, "status": task.Status, "model": task.Model,
		"duration_seconds": task.DurationSeconds, "outputs": []string{}, "error": nil,
		"created_at": task.CreatedAt, "completed_at": task.CompletedAt,
	}
	data, _ := json.Marshal(fallback)
	return data
}

func seedanceResponseResourceID(body []byte, channel string) string {
	root := payloadFromJSON(body)
	if channel == service.SeedanceChannelGroup {
		return jsonString(root, "id")
	}
	return jsonString(jsonObject(root, "data"), "Id")
}

func payloadFromJSON(body []byte) map[string]any {
	var result map[string]any
	_ = json.Unmarshal(body, &result)
	return result
}

func jsonObject(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return nil
	}
	value, _ := parent[key].(map[string]any)
	return value
}

func jsonString(parent map[string]any, key string) string {
	if parent == nil {
		return ""
	}
	value, _ := parent[key].(string)
	return strings.TrimSpace(value)
}

func jsonFloat(parent map[string]any, key string) float64 {
	if parent == nil {
		return 0
	}
	value, _ := parent[key].(float64)
	return value
}

func collectSeedanceAssetIDs(value any) []string {
	seen := make(map[string]struct{})
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		case string:
			if id, ok := service.SeedanceAssetIDFromURI(typed); ok {
				seen[id] = struct{}{}
			}
		}
	}
	walk(value)
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result
}

func seedanceModerationBody(payload map[string]any) []byte {
	content, _ := payload["content"].([]any)
	parts := make([]string, 0, len(content))
	for _, raw := range content {
		item, _ := raw.(map[string]any)
		if jsonString(item, "type") != "text" {
			continue
		}
		if text := jsonString(item, "text"); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return nil
	}
	body, _ := json.Marshal(map[string]string{"prompt": strings.Join(parts, "\n")})
	return body
}

func seedanceChannelSupportsModel(channel, model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	switch channel {
	case service.SeedanceChannelSD:
		return strings.HasSuffix(model, "-hc")
	case service.SeedanceChannelDoubao:
		return strings.HasPrefix(model, "doubao-seedance-")
	case service.SeedanceChannelGroup:
		return strings.HasPrefix(model, "dreamina-seedance-") && !strings.HasSuffix(model, "-hc")
	default:
		return false
	}
}

func queryPositiveInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (h *SeedanceHandler) writeUpstream(c *gin.Context, resp *service.SeedanceUpstreamResponse) {
	if resp == nil {
		h.writeError(c, http.StatusBadGateway, "upstream_error", "Empty Seedance upstream response")
		return
	}
	for _, name := range []string{"Content-Type", "Retry-After", "Request-Id", "X-Request-Id", "Location", "Content-Disposition", "Cache-Control"} {
		if value := resp.Header.Get(name); value != "" {
			c.Header(name, value)
		}
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, resp.Body)
}

func (h *SeedanceHandler) writeConcurrencyError(c *gin.Context, err error) {
	status := http.StatusTooManyRequests
	code := "concurrency_limit"
	var queueErr *WaitQueueFullError
	if errors.As(err, &queueErr) {
		code = "wait_queue_full"
	}
	h.writeError(c, status, code, err.Error())
}

func (h *SeedanceHandler) writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"type": code, "message": message}})
}
