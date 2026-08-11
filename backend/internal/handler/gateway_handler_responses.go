package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// Responses handles OpenAI Responses API endpoint for Anthropic platform groups.
// POST /v1/responses
// This converts Responses API requests to Anthropic format, forwards to Anthropic
// upstream, and converts responses back to Responses format.
func (h *GatewayHandler) Responses(c *gin.Context) {
	streamStarted := false

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.responsesErrorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.responsesErrorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.gateway.responses",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	// Read request body
	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.responsesErrorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.responsesErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}

	if len(body) == 0 {
		h.responsesErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	setOpsRequestContext(c, "", false)

	// Validate JSON
	if !gjson.ValidBytes(body) {
		logRequestBodyParseFailure(reqLog, body, nil)
		h.responsesErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	// Extract model and stream using gjson (like OpenAI handler)
	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.responsesErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	baseRequestCtx := c.Request.Context()
	reqStream, ok := parseOpenAICompatibleStream(body)
	if !ok {
		h.responsesErrorResponse(c, http.StatusBadRequest, "invalid_request_error", invalidStreamFieldTypeMessage)
		return
	}
	orderedGroups, _ := resolveOrderedAPIKeyGroups(baseRequestCtx, h.gatewayService, apiKey)
	if len(orderedGroups) == 0 && apiKey.Group != nil {
		orderedGroups = []*service.Group{apiKey.Group}
	}
	groupIDs := make([]int64, 0, len(orderedGroups))
	groupsByID := make(map[int64]*service.Group, len(orderedGroups))
	for _, group := range orderedGroups {
		groupIDs = append(groupIDs, group.ID)
		groupsByID[group.ID] = group
	}
	requestBudget := NewRequestFailoverBudget(h.maxAccountSwitches)
	groupFailover := NewOrderedGroupFailoverStateWithBudget(groupIDs, len(groupIDs)-1, requestBudget)
	if len(orderedGroups) <= 1 && apiKey.Group != nil && apiKey.Group.ClaudeCodeOnly {
		h.responsesErrorResponse(c, http.StatusForbidden, "permission_error", "This group is restricted to Claude Code clients (/v1/messages only)")
		return
	}
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	setOpsRequestContext(c, reqModel, reqStream)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(reqStream, false)))

	if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIResponses, reqModel, body); decision != nil && !decision.AllowNextStage {
		h.responsesSecurityAuditError(c, decision)
		return
	}

	// Error passthrough binding
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())

	// 1. Acquire user concurrency slot
	maxWait := service.CalculateMaxWait(subject.Concurrency)
	canWait, err := h.concurrencyHelper.IncrementWaitCount(c.Request.Context(), subject.UserID, maxWait)
	waitCounted := false
	if err != nil {
		reqLog.Warn("gateway.responses.user_wait_counter_increment_failed", zap.Error(err))
	} else if !canWait {
		h.responsesErrorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return
	}
	if err == nil && canWait {
		waitCounted = true
	}
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), subject.UserID)
		}
	}()

	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted)
	if err != nil {
		reqLog.Warn("gateway.responses.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", streamStarted)
		return
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// Parse request for session hash
	bodyRef := service.NewRequestBodyRef(body)
	parsedReq, _ := service.ParseGatewayRequest(bodyRef, "responses")
	if parsedReq == nil {
		parsedReq = &service.ParsedRequest{Model: reqModel, Stream: reqStream, Body: bodyRef}
	}
	parsedReq.SessionContext = &service.SessionContext{
		ClientIP:  ip.GetClientIP(c),
		UserAgent: c.GetHeader("User-Agent"),
		APIKeyID:  apiKey.ID,
	}
	sessionHash := h.gatewayService.GenerateSessionHash(parsedReq)

	// 3. Ordered Group attempts; account selection and failover remain inside each attempt.
	var currentAPIKey *service.APIKey
	var currentSubscription *service.UserSubscription
	var channelMapping service.ChannelMappingResult
	var pricingAt time.Time
	var groupUserReleaseFunc func()
	defer func() {
		if groupUserReleaseFunc != nil {
			groupUserReleaseFunc()
		}
	}()
	releaseGroupUserSlot := func() {
		if groupUserReleaseFunc != nil {
			groupUserReleaseFunc()
			groupUserReleaseFunc = nil
		}
	}

groupAttempt:
	for {
		releaseGroupUserSlot()
		groupID, ok := groupFailover.CurrentGroupID()
		if !ok {
			h.responsesErrorResponse(c, http.StatusBadGateway, "server_error", "All available groups exhausted")
			return
		}
		currentGroup := groupsByID[groupID]
		attemptCtx, attemptPricingAt := service.WithGatewayTokenRequestPricing(baseRequestCtx)
		var attemptErr error
		currentSubscription = nil
		if currentGroup.IsSubscriptionType() {
			currentSubscription, attemptErr = h.gatewayService.ResolveActiveSubscription(attemptCtx, subject.UserID, currentGroup.ID)
			if attemptErr != nil {
				if isRetryableGroupBillingError(attemptErr) && groupFailover.Advance(attemptCtx, nil, false) == GroupFailoverContinue {
					continue groupAttempt
				}
				status, code, message, retryAfter := billingErrorDetails(attemptErr)
				if retryAfter > 0 {
					c.Header("Retry-After", strconv.Itoa(retryAfter))
				}
				h.responsesErrorResponse(c, status, code, message)
				return
			}
		}
		currentAPIKey, _, attemptErr = applyAPIKeyGroupAttemptWithBase(c, attemptCtx, apiKey, currentGroup, currentSubscription)
		if attemptErr != nil {
			if groupFailover.Advance(attemptCtx, nil, false) == GroupFailoverContinue {
				continue groupAttempt
			}
			h.responsesErrorResponse(c, http.StatusForbidden, "permission_error", attemptErr.Error())
			return
		}
		requestCtx := c.Request.Context()
		if service.IsImageGenerationIntentForPlatform("/v1/responses", reqModel, body, openAICompatibleRequestPlatform(requestCtx, currentAPIKey)) {
			requestCtx = service.WithOpenAIImageGenerationIntent(requestCtx)
			c.Request = c.Request.WithContext(requestCtx)
		}
		ensureCompositeTargetPlatform(c, currentAPIKey, reqModel)
		if !compositeTargetPlatformResolved(c, currentAPIKey, reqModel) {
			if groupFailover.Advance(requestCtx, nil, false) == GroupFailoverContinue {
				continue groupAttempt
			}
			h.responsesErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Model is not supported by composite groups")
			return
		}
		requestCtx = c.Request.Context()
		requestCtx = service.WithSingleAccountRetry(
			requestCtx,
			h.gatewayService.IsSingleAntigravityAccountGroup(requestCtx, currentAPIKey.GroupID),
			h.metadataBridgeEnabled(),
		)
		c.Request = c.Request.WithContext(requestCtx)
		channelMapping, _ = h.gatewayService.ResolveChannelMappingAndRestrict(requestCtx, currentAPIKey.GroupID, reqModel)
		if currentAPIKey.Group != nil && currentAPIKey.Group.ClaudeCodeOnly {
			if groupFailover.Advance(requestCtx, nil, false) == GroupFailoverContinue {
				continue groupAttempt
			}
			h.responsesErrorResponse(c, http.StatusForbidden, "permission_error", "This group is restricted to Claude Code clients (/v1/messages only)")
			return
		}
		if maxGroupConcurrency := currentGroup.EffectiveUserConcurrencyLimitAt(time.Now()); maxGroupConcurrency > 0 {
			groupUserReleaseFunc, attemptErr = h.concurrencyHelper.AcquireGroupUserSlotWithWait(c, currentGroup.ID, subject.UserID, maxGroupConcurrency, reqStream, &streamStarted)
			if attemptErr != nil {
				reqLog.Warn("gateway.responses.group_user_slot_acquire_failed", zap.Int64("group_id", currentGroup.ID), zap.Error(attemptErr))
				h.handleConcurrencyError(c, attemptErr, "group_user", streamStarted)
				return
			}
			groupUserReleaseFunc = wrapReleaseOnDone(requestCtx, groupUserReleaseFunc)
		}
		if attemptErr = h.billingCacheService.CheckBillingEligibility(requestCtx, currentAPIKey.User, currentAPIKey, currentAPIKey.Group, currentSubscription, service.QuotaPlatform(requestCtx, currentAPIKey)); attemptErr != nil {
			if isRetryableGroupBillingError(attemptErr) && groupFailover.Advance(requestCtx, nil, false) == GroupFailoverContinue {
				continue groupAttempt
			}
			status, code, message, retryAfter := billingErrorDetails(attemptErr)
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
			}
			h.responsesErrorResponse(c, status, code, message)
			return
		}
		pricingAt = attemptPricingAt
		fs := NewFailoverStateWithBudget(h.maxAccountSwitches, false, requestBudget)

		for {
			if requestCtx.Err() != nil {
				return
			}
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(requestCtx, currentAPIKey.GroupID, sessionHash, reqModel, fs.FailedAccountIDs, "", int64(0))
			if err != nil {
				if len(fs.FailedAccountIDs) == 0 {
					if groupAction := groupFailover.Advance(requestCtx, nil, false); groupAction == GroupFailoverContinue {
						continue groupAttempt
					} else if groupAction == GroupFailoverCanceled {
						failoverClientGone(c)
						return
					}
					cls := classifyNoAccountErrorFromGin(c, h.gatewayService, currentAPIKey, reqModel, reqModel, effectiveAPIKeyPlatform(c, currentAPIKey))
					if !cls.ModelNotFound {
						markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
					}
					message := cls.Message
					if !cls.ModelNotFound {
						message = "No available accounts: " + err.Error()
					}
					h.responsesErrorResponse(c, cls.Status, cls.ErrType, message)
					return
				}
				action := fs.HandleSelectionExhausted(requestCtx)
				switch action {
				case FailoverContinue:
					continue
				case FailoverCanceled:
					failoverClientGone(c)
					return
				default:
					if fs.LastFailoverErr != nil {
						if groupAction := groupFailover.Advance(requestCtx, fs.LastFailoverErr, false); groupAction == GroupFailoverContinue {
							continue groupAttempt
						} else if groupAction == GroupFailoverCanceled {
							failoverClientGone(c)
							return
						}
						h.handleResponsesFailoverExhausted(c, fs.LastFailoverErr, streamStarted)
					} else {
						h.responsesErrorResponse(c, http.StatusBadGateway, "server_error", "All available accounts exhausted")
					}
					return
				}
			}
			account := selection.Account
			setOpsSelectedAccount(c, account.ID, account.Platform)

			// 4. Acquire account concurrency slot
			accountReleaseFunc := selection.ReleaseFunc
			if !selection.Acquired {
				if selection.WaitPlan == nil {
					markOpsRoutingCapacityLimited(c)
					h.responsesErrorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts")
					return
				}
				accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
					c,
					account.ID,
					selection.WaitPlan.MaxConcurrency,
					selection.WaitPlan.Timeout,
					reqStream,
					&streamStarted,
				)
				if err != nil {
					reqLog.Warn("gateway.responses.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
					h.handleConcurrencyError(c, err, "account", streamStarted)
					return
				}
			}
			// 终检与准入后绑定必须使用选号结果携带的门：门安装在调度栈的局部
			// ctx 上（composite/fallback 还可能解析出与入口分组不同的门），直接用
			// requestCtx 会退化为空操作。
			admissionCtx := service.ContextWithSelectionProfitGate(requestCtx, selection)
			latest, vetoed, reason := h.gatewayService.GatewayProfitControlVetoLatest(admissionCtx, account)
			if vetoed {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
				reqLog.Debug("gateway.responses.account_slot_profit_vetoed", zap.Int64("account_id", account.ID), zap.String("reason", reason))
				if fs.RecordProfitVeto(account.ID) == FailoverExhausted {
					reqLog.Warn("gateway.responses.profit_veto_attempts_exhausted", zap.Int("profit_veto_count", fs.ProfitVetoCount()))
					h.responsesErrorResponse(c, http.StatusServiceUnavailable, "api_error", profitVetoExhaustedMessage)
					return
				}
				continue
			}
			account = latest
			selection.Account = latest
			if selection.ProfitGateActive() {
				if err := h.gatewayService.BindStickySessionAfterProfitAdmission(admissionCtx, currentAPIKey.GroupID, sessionHash, account.ID); err != nil {
					reqLog.Warn("gateway.responses.bind_sticky_session_after_profit_admission_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

			// 5. Forward request
			writerSizeBeforeForward := service.OpenAICompactKeepaliveAdjustedWrittenSize(c)
			forwardBody := body
			if channelMapping.Mapped {
				forwardBody = h.gatewayService.ReplaceModelInBody(body, channelMapping.MappedModel)
			}
			var result *service.ForwardResult
			setActualUpstreamEndpoint(c, "")
			if shouldUseAntigravityCompat(account) {
				if h.antigravityGatewayService == nil {
					h.responsesErrorResponse(c, http.StatusBadGateway, "upstream_error", "Antigravity compatibility service is not configured")
					if accountReleaseFunc != nil {
						accountReleaseFunc()
					}
					return
				}
				setActualUpstreamEndpoint(c, EndpointAntigravityGenerateContent)
				result, err = h.antigravityGatewayService.ForwardAsResponses(requestCtx, c, account, forwardBody, parsedReq)
			} else {
				result, err = h.gatewayService.ForwardAsResponses(requestCtx, c, account, forwardBody, parsedReq)
			}

			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}

			if err != nil {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					// Can't failover if streaming content already sent
					if service.OpenAICompactKeepaliveAdjustedWrittenSize(c) != writerSizeBeforeForward {
						h.handleResponsesFailoverExhausted(c, failoverErr, true)
						return
					}
					action := fs.HandleFailoverError(requestCtx, h.gatewayService, account.ID, account.Platform, account.GetPoolModeRetryCount(), failoverErr)
					switch action {
					case FailoverContinue:
						continue
					case FailoverExhausted:
						if groupAction := groupFailover.Advance(requestCtx, fs.LastFailoverErr, false); groupAction == GroupFailoverContinue {
							continue groupAttempt
						} else if groupAction == GroupFailoverCanceled {
							failoverClientGone(c)
							return
						}
						h.handleResponsesFailoverExhausted(c, fs.LastFailoverErr, streamStarted)
						return
					case FailoverCanceled:
						failoverClientGone(c)
						return
					}
				}
				upstreamErrorAlreadyCommunicated := gatewayForwardErrorAlreadyCommunicated(c, writerSizeBeforeForward, err)
				wroteFallback := false
				if !upstreamErrorAlreadyCommunicated {
					wroteFallback = h.ensureForwardErrorResponse(c, streamStarted)
				}
				reqLog.Error("gateway.responses.forward_failed",
					zap.Int64("account_id", account.ID),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Bool("upstream_error_response_already_written", upstreamErrorAlreadyCommunicated),
					zap.Error(err),
				)
				return
			}

			// 6. Record usage
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			requestPayloadHash := service.HashUsageRequestPayload(body)
			inboundEndpoint := GetInboundEndpoint(c)
			upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)

			usageAPIKey := currentAPIKey
			usageUser := currentAPIKey.User
			usageSubscription := currentSubscription
			usageQuotaPlatform := service.QuotaPlatform(c.Request.Context(), currentAPIKey)
			usageChannelMapping := channelMapping
			sessionID := service.ExtractClientSessionID(c)
			h.submitUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{
					Result:             result,
					QuotaPlatform:      usageQuotaPlatform,
					APIKey:             usageAPIKey,
					User:               usageUser,
					Account:            account,
					Subscription:       usageSubscription,
					PricingAt:          pricingAt,
					InboundEndpoint:    inboundEndpoint,
					UpstreamEndpoint:   upstreamEndpoint,
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: requestPayloadHash,
					APIKeyService:      h.apiKeyService,
					SessionID:          sessionID,
					ChannelUsageFields: clientRequestedUsageFields(c, usageChannelMapping, reqModel, result.UpstreamModel),
				}); err != nil {
					reqLog.Error("gateway.responses.record_usage_failed",
						zap.Int64("account_id", account.ID),
						zap.Error(err),
					)
				}
			})
			return
		}
	}
}

// responsesErrorResponse writes an error in OpenAI Responses API format.
func (h *GatewayHandler) responsesErrorResponse(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// handleResponsesFailoverExhausted writes a failover-exhausted error in Responses format.
func (h *GatewayHandler) handleResponsesFailoverExhausted(c *gin.Context, lastErr *service.UpstreamFailoverError, streamStarted bool) {
	if streamStarted {
		return // Can't write error after stream started
	}
	if lastErr != nil {
		copyFailoverRetryAfter(c, lastErr.ResponseHeaders)
	}
	if lastErr != nil && lastErr.IsCredentialFailure() {
		status, message := credentialFailoverClientResponse(lastErr)
		h.responsesErrorResponse(c, status, "server_error", message)
		return
	}
	statusCode := http.StatusBadGateway
	if lastErr != nil && lastErr.StatusCode > 0 {
		statusCode = lastErr.StatusCode
	}
	if lastErr != nil && service.IsOpenAISilentRefusalErrorBody(lastErr.ResponseBody) {
		service.SetOpsUpstreamError(c, statusCode, service.OpenAISilentRefusalClientMessage(), "")
		h.responsesErrorResponse(c, http.StatusBadGateway, "upstream_error", service.OpenAISilentRefusalClientMessage())
		return
	}
	h.responsesErrorResponse(c, statusCode, "server_error", "All available accounts exhausted")
}
