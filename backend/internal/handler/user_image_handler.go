package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

var userImageFallbackModels = []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1"}

const (
	userImagePlaygroundSummaryContextKey = "user_image_playground_summary"
	userImagePlaygroundResponseField     = "_sub2api_image_playground"
	userImageGenerateFieldAPIKeyID       = "api_key_id"
	userImageMultipartMemoryLimit        = 32 << 20
)

type userImageAPIKeyLister interface {
	List(ctx context.Context, userID int64, params pagination.PaginationParams, filters service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error)
}

type userImageAPIKeyGetter interface {
	GetByID(ctx context.Context, id int64) (*service.APIKey, error)
}

type userImageAPIKeyService interface {
	userImageAPIKeyLister
	userImageAPIKeyGetter
}

type userImageGateway interface {
	Images(*gin.Context)
}

type userImageGeminiGateway interface {
	GeminiV1BetaModels(*gin.Context)
}

type userImageSubscriptionService interface {
	GetActiveSubscription(ctx context.Context, userID, groupID int64) (*service.UserSubscription, error)
	ValidateAndCheckLimits(subscription *service.UserSubscription, group *service.Group) (bool, error)
	DoWindowMaintenance(subscription *service.UserSubscription)
}

// UserImageHandler handles authenticated image playground requests.
type UserImageHandler struct {
	apiKeyService       userImageAPIKeyService
	subscriptionService userImageSubscriptionService
	gateway             userImageGateway
	geminiGateway       userImageGeminiGateway
}

// NewUserImageHandler creates a new UserImageHandler.
func NewUserImageHandler(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService) *UserImageHandler {
	return NewUserImageHandlerWithDeps(apiKeyService, nil, subscriptionService)
}

// NewUserImageHandlerWithDeps creates a new UserImageHandler with testable dependencies.
func NewUserImageHandlerWithDeps(apiKeyService userImageAPIKeyService, gateway userImageGateway, subscriptionService ...userImageSubscriptionService) *UserImageHandler {
	h := &UserImageHandler{apiKeyService: apiKeyService, gateway: gateway}
	if len(subscriptionService) > 0 {
		h.subscriptionService = subscriptionService[0]
	}
	return h
}

// SetGateway injects the delegated OpenAI images gateway handler.
func (h *UserImageHandler) SetGateway(gateway userImageGateway) {
	if h == nil {
		return
	}
	h.gateway = gateway
}

// SetGeminiGateway injects the delegated Gemini native gateway handler.
func (h *UserImageHandler) SetGeminiGateway(gateway userImageGeminiGateway) {
	if h == nil {
		return
	}
	h.geminiGateway = gateway
}

type userImageOptionsResponse struct {
	Keys           []userImageKeyOption `json:"keys"`
	FallbackModels []string             `json:"fallback_models"`
}

type userImageKeyOption struct {
	ID                   int64    `json:"id"`
	Name                 string   `json:"name"`
	MaskedKey            string   `json:"masked_key"`
	GroupID              *int64   `json:"group_id"`
	GroupName            string   `json:"group_name"`
	AllowImageGeneration bool     `json:"allow_image_generation"`
	Models               []string `json:"models"`
	DefaultModel         string   `json:"default_model"`
}

// Options handles GET /api/v1/user/images/options.
func (h *UserImageHandler) Options(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	keys, _, err := h.apiKeyService.List(c.Request.Context(), subject.UserID, pagination.PaginationParams{
		Page:      1,
		PageSize:  1000,
		SortBy:    "created_at",
		SortOrder: "desc",
	}, service.APIKeyListFilters{})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	options := make([]userImageKeyOption, 0, len(keys))
	for i := range keys {
		key := keys[i]
		if key.UserID != subject.UserID {
			continue
		}
		if !key.IsActive() || key.IsExpired() {
			continue
		}
		if !service.GroupAllowsImageGeneration(key.Group) {
			continue
		}

		models := userImageModelsForGroup(key.Group)
		defaultModel := ""
		if len(models) > 0 {
			defaultModel = models[0]
		}

		option := userImageKeyOption{
			ID:                   key.ID,
			Name:                 key.Name,
			MaskedKey:            maskUserImageAPIKey(key.Key),
			GroupID:              key.GroupID,
			AllowImageGeneration: service.GroupAllowsImageGeneration(key.Group),
			Models:               models,
			DefaultModel:         defaultModel,
		}
		if key.Group != nil {
			option.GroupName = key.Group.Name
		}

		options = append(options, option)
	}

	response.Success(c, userImageOptionsResponse{
		Keys:           options,
		FallbackModels: append([]string(nil), userImageFallbackModels...),
	})
}

// Generate handles POST /api/v1/user/images/generations.
func (h *UserImageHandler) Generate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			response.Error(c, http.StatusRequestEntityTooLarge, buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		response.BadRequest(c, "Failed to read request body")
		return
	}
	if len(body) == 0 {
		response.BadRequest(c, "Request body is empty")
		return
	}

	apiKeyID, sanitizedBody, sanitizedContentType, upstreamPath, err := prepareUserImageGenerateRequest(c.GetHeader("Content-Type"), body)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	apiKey, err := h.apiKeyService.GetByID(c.Request.Context(), apiKeyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if apiKey == nil {
		response.NotFound(c, "API key not found")
		return
	}
	if apiKey.UserID != subject.UserID {
		response.Forbidden(c, "API key does not belong to current user")
		return
	}
	if !apiKey.IsActive() || apiKey.IsExpired() {
		response.Forbidden(c, "API key is unavailable")
		return
	}
	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		response.Forbidden(c, service.ImageGenerationPermissionMessage())
		return
	}

	if apiKey.User == nil {
		role, _ := middleware2.GetUserRoleFromContext(c)
		apiKey.User = &service.User{ID: subject.UserID, Role: role}
	}
	if !h.setSubscriptionContextForUserImage(c, subject, apiKey) {
		return
	}
	setUserImageGroupContext(c, apiKey.Group)

	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), subject)
	if role, ok := middleware2.GetUserRoleFromContext(c); ok {
		c.Set(string(middleware2.ContextKeyUserRole), role)
	}

	if userImageShouldUseGemini(apiKey.Group) {
		h.generateWithGemini(c, sanitizedContentType, sanitizedBody)
		return
	}

	if h.gateway == nil {
		response.InternalError(c, "Image gateway unavailable")
		return
	}

	c.Request.URL.Path = upstreamPath
	c.Request.Body = io.NopCloser(bytes.NewReader(sanitizedBody))
	c.Request.ContentLength = int64(len(sanitizedBody))
	c.Request.Header.Set("Content-Type", sanitizedContentType)
	c.ContentType()

	if userImageRequestWantsStream(sanitizedContentType, sanitizedBody) {
		h.gateway.Images(c)
		return
	}

	delegateRecorder := httptest.NewRecorder()
	delegateCtx, _ := gin.CreateTestContext(delegateRecorder)
	delegateCtx.Request = c.Request
	delegateCtx.Params = c.Params
	delegateCtx.Keys = cloneGinKeys(c.Keys)

	if len(c.Errors) > 0 {
		delegateCtx.Errors = append(delegateCtx.Errors, c.Errors...)
	}

	h.gateway.Images(delegateCtx)

	statusCode := delegateRecorder.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	responseBody := delegateRecorder.Body.Bytes()
	responseHeaders := delegateRecorder.Header()
	if summary := getUserImagePlaygroundSummary(delegateCtx); summary != nil {
		if bodyWithSummary, ok := appendUserImagePlaygroundSummary(responseBody, summary); ok {
			responseBody = bodyWithSummary
			if responseHeaders.Get("Content-Type") == "" {
				responseHeaders.Set("Content-Type", "application/json")
			}
			responseHeaders.Del("Content-Length")
		}
	}

	copyUserImageProxyHeaders(c.Writer.Header(), responseHeaders)
	c.Status(statusCode)
	_, _ = c.Writer.Write(responseBody)
}

func (h *UserImageHandler) generateWithGemini(c *gin.Context, contentType string, body []byte) {
	if h.geminiGateway == nil {
		response.InternalError(c, "Gemini image gateway unavailable")
		return
	}

	model, nativeBody, err := buildUserImageGeminiNativeRequest(contentType, body)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	modelAction := "/" + model + ":generateContent"
	upstreamPath := "/v1beta/models/" + model + ":generateContent"

	delegateRecorder := httptest.NewRecorder()
	delegateCtx, _ := gin.CreateTestContext(delegateRecorder)
	delegateReq := c.Request.Clone(c.Request.Context())
	if delegateReq.URL != nil {
		delegateReq.URL.Path = upstreamPath
		delegateReq.URL.RawPath = ""
	}
	delegateReq.Body = io.NopCloser(bytes.NewReader(nativeBody))
	delegateReq.ContentLength = int64(len(nativeBody))
	delegateReq.Header.Set("Content-Type", "application/json")
	delegateCtx.Request = delegateReq
	delegateCtx.Params = append(append([]gin.Param(nil), c.Params...), gin.Param{Key: "modelAction", Value: modelAction})
	delegateCtx.Keys = cloneGinKeys(c.Keys)
	if len(c.Errors) > 0 {
		delegateCtx.Errors = append(delegateCtx.Errors, c.Errors...)
	}

	h.geminiGateway.GeminiV1BetaModels(delegateCtx)

	statusCode := delegateRecorder.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	responseBody := delegateRecorder.Body.Bytes()
	responseHeaders := delegateRecorder.Header()
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		if convertedBody, ok := convertGeminiNativeImageResponse(responseBody); ok {
			responseBody = convertedBody
			responseHeaders.Set("Content-Type", "application/json")
			responseHeaders.Del("Content-Length")
		}
	}

	copyUserImageProxyHeaders(c.Writer.Header(), responseHeaders)
	c.Status(statusCode)
	_, _ = c.Writer.Write(responseBody)
}

func buildUserImageGeminiNativeRequest(contentType string, body []byte) (string, []byte, error) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		return buildUserImageGeminiNativeMultipartRequest(contentType, body)
	}
	return buildUserImageGeminiNativeJSONRequest(body)
}

func buildUserImageGeminiNativeJSONRequest(body []byte) (string, []byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil, fmt.Errorf("invalid JSON request body")
	}

	model := normalizeUserImageGeminiModelName(stringFromMap(payload, "model"))
	prompt := strings.TrimSpace(stringFromMap(payload, "prompt"))
	size := strings.TrimSpace(stringFromMap(payload, "size"))
	return buildUserImageGeminiNativeBody(model, prompt, nil, size)
}

func buildUserImageGeminiNativeMultipartRequest(contentType string, body []byte) (string, []byte, error) {
	_, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return "", nil, fmt.Errorf("invalid multipart content type")
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return "", nil, fmt.Errorf("multipart boundary is required")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	fields := map[string]string{}
	imageParts := []map[string]any{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse multipart request")
		}

		content, readErr := io.ReadAll(part)
		if readErr != nil {
			return "", nil, fmt.Errorf("failed to read multipart request")
		}
		if part.FormName() == "image" {
			mimeType := userImageMultipartImageMimeType(part, content)
			imageParts = append(imageParts, map[string]any{
				"inlineData": map[string]any{
					"mimeType": mimeType,
					"data":     base64.StdEncoding.EncodeToString(content),
				},
			})
			continue
		}
		fields[part.FormName()] = string(content)
	}

	model := normalizeUserImageGeminiModelName(fields["model"])
	prompt := strings.TrimSpace(fields["prompt"])
	size := strings.TrimSpace(fields["size"])
	return buildUserImageGeminiNativeBody(model, prompt, imageParts, size)
}

func buildUserImageGeminiNativeBody(model string, prompt string, imageParts []map[string]any, size string) (string, []byte, error) {
	model = normalizeUserImageGeminiModelName(model)
	if model == "" {
		return "", nil, fmt.Errorf("model is required")
	}
	if !userImageIsGeminiImageModelName(model) {
		return "", nil, fmt.Errorf("model is not a Gemini image generation model")
	}

	parts := make([]map[string]any, 0, 1+len(imageParts))
	if strings.TrimSpace(prompt) != "" {
		parts = append(parts, map[string]any{"text": strings.TrimSpace(prompt)})
	}
	parts = append(parts, imageParts...)
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("prompt or reference image is required")
	}

	generationConfig := map[string]any{
		"responseModalities": []string{"TEXT", "IMAGE"},
		"imageConfig":        userImageGeminiImageConfig(size),
	}
	nativePayload := map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": parts,
			},
		},
		"generationConfig": generationConfig,
	}

	nativeBody, err := json.Marshal(nativePayload)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build Gemini image request")
	}
	return model, nativeBody, nil
}

func userImageMultipartImageMimeType(part *multipart.Part, content []byte) string {
	if part != nil {
		if contentType := strings.TrimSpace(part.Header.Get("Content-Type")); contentType != "" {
			if mediaType, _, err := mime.ParseMediaType(contentType); err == nil && strings.HasPrefix(strings.ToLower(mediaType), "image/") {
				return mediaType
			}
		}
		if ext := strings.ToLower(filepath.Ext(part.FileName())); ext != "" {
			if contentType := mime.TypeByExtension(ext); strings.HasPrefix(strings.ToLower(contentType), "image/") {
				return contentType
			}
		}
	}
	if detected := http.DetectContentType(content); strings.HasPrefix(strings.ToLower(detected), "image/") {
		return detected
	}
	return "image/png"
}

func userImageGeminiImageConfig(size string) map[string]any {
	imageConfig := map[string]any{
		"aspectRatio": userImageGeminiAspectRatio(size),
	}
	if tier, ok := service.ClassifyImageBillingTier(size); ok {
		imageConfig["imageSize"] = tier
	}
	return imageConfig
}

func userImageGeminiAspectRatio(size string) string {
	width, height, ok := parseUserImageDimensions(size)
	if !ok || width <= 0 || height <= 0 {
		return "1:1"
	}
	divisor := gcd(width, height)
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor)
}

func parseUserImageDimensions(size string) (int, int, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || width <= 0 {
		return 0, 0, false
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	if a == 0 {
		return 1
	}
	return a
}

func convertGeminiNativeImageResponse(body []byte) ([]byte, bool) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body, false
	}

	candidates, ok := payload["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		return body, false
	}

	images := []map[string]any{}
	for _, candidateValue := range candidates {
		candidate, ok := candidateValue.(map[string]any)
		if !ok {
			continue
		}
		content, ok := candidate["content"].(map[string]any)
		if !ok {
			continue
		}
		parts, ok := content["parts"].([]any)
		if !ok {
			continue
		}

		textParts := []string{}
		candidateImages := []map[string]any{}
		for _, partValue := range parts {
			part, ok := partValue.(map[string]any)
			if !ok {
				continue
			}
			if text := strings.TrimSpace(stringFromMap(part, "text")); text != "" {
				textParts = append(textParts, text)
			}
			if image := geminiInlineImageToOpenAIResult(part); image != nil {
				candidateImages = append(candidateImages, image)
			}
		}
		revisedPrompt := strings.Join(textParts, "\n")
		for _, image := range candidateImages {
			if revisedPrompt != "" {
				image["revised_prompt"] = revisedPrompt
			}
			images = append(images, image)
		}
	}

	if len(images) == 0 {
		return body, false
	}
	converted := map[string]any{"data": images}
	convertedBody, err := json.Marshal(converted)
	if err != nil {
		return body, false
	}
	return convertedBody, true
}

func geminiInlineImageToOpenAIResult(part map[string]any) map[string]any {
	inlineData, ok := part["inlineData"].(map[string]any)
	if !ok {
		inlineData, ok = part["inline_data"].(map[string]any)
	}
	if !ok {
		return nil
	}

	mimeType := strings.TrimSpace(stringFromMap(inlineData, "mimeType"))
	if mimeType == "" {
		mimeType = strings.TrimSpace(stringFromMap(inlineData, "mime_type"))
	}
	if mimeType == "" {
		mimeType = "image/png"
	}
	data := strings.TrimSpace(stringFromMap(inlineData, "data"))
	if data == "" {
		return nil
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return nil
	}
	return map[string]any{"url": fmt.Sprintf("data:%s;base64,%s", mimeType, data)}
}

func stringFromMap(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func setUserImageGroupContext(c *gin.Context, group *service.Group) {
	if c == nil || c.Request == nil || !service.IsGroupContextValid(group) {
		return
	}
	if existing, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group); ok && existing != nil && existing.ID == group.ID && service.IsGroupContextValid(existing) {
		return
	}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Group, group))
}

func (h *UserImageHandler) setSubscriptionContextForUserImage(c *gin.Context, subject middleware2.AuthSubject, apiKey *service.APIKey) bool {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.IsSubscriptionType() {
		return true
	}
	if h.subscriptionService == nil {
		return true
	}

	userID := subject.UserID
	if apiKey.User != nil && apiKey.User.ID > 0 {
		userID = apiKey.User.ID
	}
	subscription, err := h.subscriptionService.GetActiveSubscription(c.Request.Context(), userID, apiKey.Group.ID)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionNotFound) {
			response.Forbidden(c, "No active subscription found for this group")
			return false
		}
		response.ErrorFrom(c, err)
		return false
	}
	needsMaintenance, err := h.subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
	if err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	if needsMaintenance {
		subscriptionCopy := *subscription
		h.subscriptionService.DoWindowMaintenance(&subscriptionCopy)
	}
	c.Set(string(middleware2.ContextKeySubscription), subscription)
	return true
}

func userImageModelsForGroup(group *service.Group) []string {
	if userImageShouldUseGemini(group) {
		if group != nil && group.CustomModelsListEnabled() {
			models := make([]string, 0, len(group.ModelsListConfig.Models))
			for _, model := range group.ModelsListConfig.Models {
				model = normalizeUserImageGeminiModelName(model)
				if model == "" || !userImageIsGeminiImageModelName(model) {
					continue
				}
				models = append(models, model)
			}
			if len(models) > 0 {
				return models
			}
		}
		return userImageGeminiFallbackModels()
	}

	if group != nil && group.CustomModelsListEnabled() {
		models := make([]string, 0, len(group.ModelsListConfig.Models))
		for _, model := range group.ModelsListConfig.Models {
			model = strings.TrimSpace(model)
			if model == "" || !service.IsOpenAIImageModelName(model) {
				continue
			}
			models = append(models, model)
		}
		if len(models) > 0 {
			return models
		}
	}
	return append([]string(nil), userImageFallbackModels...)
}

func userImageShouldUseGemini(group *service.Group) bool {
	return group != nil && group.Platform == service.PlatformGemini
}

func userImageGeminiFallbackModels() []string {
	models := []string{}
	seen := map[string]struct{}{}
	for _, model := range gemini.DefaultModels() {
		name := normalizeUserImageGeminiModelName(model.Name)
		if name == "" || !userImageIsGeminiImageModelName(name) {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}
	return models
}

func normalizeUserImageGeminiModelName(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "models/")
	return model
}

func userImageIsGeminiImageModelName(model string) bool {
	model = strings.ToLower(normalizeUserImageGeminiModelName(model))
	return model == "gemini-3.1-flash-image" ||
		model == "gemini-3.1-flash-image-preview" ||
		strings.HasPrefix(model, "gemini-3.1-flash-image-") ||
		model == "gemini-3-pro-image" ||
		model == "gemini-3-pro-image-preview" ||
		strings.HasPrefix(model, "gemini-3-pro-image-") ||
		model == "gemini-2.5-flash-image" ||
		model == "gemini-2.5-flash-image-preview" ||
		strings.HasPrefix(model, "gemini-2.5-flash-image-")
}

func maskUserImageAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) < 4 {
		return "sk-..."
	}
	return "sk-..." + key[len(key)-4:]
}

func prepareUserImageGenerateRequest(contentType string, body []byte) (int64, []byte, string, string, error) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		apiKeyID, sanitizedBody, sanitizedType, sanitizeErr := sanitizeUserImageMultipartBody(body, contentType)
		if sanitizeErr != nil {
			return 0, nil, "", "", sanitizeErr
		}
		return apiKeyID, sanitizedBody, sanitizedType, "/v1/images/edits", nil
	}

	apiKeyID, sanitizedBody, sanitizeErr := sanitizeUserImageJSONBody(body)
	if sanitizeErr != nil {
		return 0, nil, "", "", sanitizeErr
	}
	return apiKeyID, sanitizedBody, "application/json", "/v1/images/generations", nil
}

func sanitizeUserImageJSONBody(body []byte) (int64, []byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, nil, fmt.Errorf("invalid JSON request body")
	}
	apiKeyValue, exists := payload[userImageGenerateFieldAPIKeyID]
	if !exists {
		return 0, nil, fmt.Errorf("api_key_id is required")
	}
	apiKeyID, err := parseUserImageAPIKeyID(apiKeyValue)
	if err != nil {
		return 0, nil, err
	}
	delete(payload, userImageGenerateFieldAPIKeyID)
	if strings.TrimSpace(stringFromMap(payload, "prompt")) == "" {
		return 0, nil, fmt.Errorf("prompt is required")
	}
	sanitizedBody, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to sanitize request body")
	}
	return apiKeyID, sanitizedBody, nil
}

func sanitizeUserImageMultipartBody(body []byte, contentType string) (int64, []byte, string, error) {
	_, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return 0, nil, "", fmt.Errorf("invalid multipart content type")
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return 0, nil, "", fmt.Errorf("multipart boundary is required")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	var apiKeyID int64
	var apiKeyFound bool

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, nil, "", fmt.Errorf("failed to parse multipart request")
		}

		if part.FormName() == userImageGenerateFieldAPIKeyID {
			value, readErr := io.ReadAll(io.LimitReader(part, userImageMultipartMemoryLimit))
			if readErr != nil {
				return 0, nil, "", fmt.Errorf("failed to read api_key_id")
			}
			apiKeyID, err = parseUserImageAPIKeyID(strings.TrimSpace(string(value)))
			if err != nil {
				return 0, nil, "", err
			}
			apiKeyFound = true
			continue
		}

		newPart, err := writer.CreatePart(cloneMultipartHeader(part.Header))
		if err != nil {
			return 0, nil, "", fmt.Errorf("failed to rebuild multipart request")
		}
		if _, err := io.Copy(newPart, part); err != nil {
			return 0, nil, "", fmt.Errorf("failed to copy multipart request")
		}
	}

	if !apiKeyFound {
		return 0, nil, "", fmt.Errorf("api_key_id is required")
	}
	if err := writer.Close(); err != nil {
		return 0, nil, "", fmt.Errorf("failed to finalize multipart request")
	}
	return apiKeyID, buffer.Bytes(), writer.FormDataContentType(), nil
}

func parseUserImageAPIKeyID(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		if v <= 0 {
			return 0, fmt.Errorf("api_key_id must be greater than 0")
		}
		return v, nil
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("api_key_id must be greater than 0")
		}
		return int64(v), nil
	case float64:
		if v <= 0 || v != float64(int64(v)) {
			return 0, fmt.Errorf("api_key_id must be a positive integer")
		}
		return int64(v), nil
	case json.Number:
		parsed, err := v.Int64()
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("api_key_id must be a positive integer")
		}
		return parsed, nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("api_key_id must be a positive integer")
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("api_key_id must be a positive integer")
	}
}

func cloneMultipartHeader(src textproto.MIMEHeader) textproto.MIMEHeader {
	dst := make(textproto.MIMEHeader, len(src))
	for key, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}

func cloneGinKeys(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func copyUserImageProxyHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
}

func appendUserImagePlaygroundSummary(body []byte, summary *service.OpenAIImageCostSummary) ([]byte, bool) {
	if summary == nil || len(body) == 0 {
		return body, false
	}

	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		return body, false
	}
	if envelope == nil {
		return body, false
	}

	metadata := map[string]any{}
	if existing, ok := envelope[userImagePlaygroundResponseField].(map[string]any); ok && existing != nil {
		for key, value := range existing {
			metadata[key] = value
		}
	}
	if summary.EstimatedPrice != nil {
		metadata["estimated_price"] = *summary.EstimatedPrice
	}
	if summary.ActualCost != nil {
		metadata["actual_cost"] = *summary.ActualCost
	}
	if summary.TotalCost != nil {
		metadata["total_cost"] = *summary.TotalCost
	}
	if summary.ImageCount > 0 {
		metadata["image_count"] = summary.ImageCount
	}
	if summary.ImageSize != "" {
		metadata["image_size"] = summary.ImageSize
	}
	if summary.BillingMode != "" {
		metadata["billing_mode"] = summary.BillingMode
	}
	if len(metadata) == 0 {
		return body, false
	}

	envelope[userImagePlaygroundResponseField] = metadata
	augmentedBody, err := json.Marshal(envelope)
	if err != nil {
		return body, false
	}
	return augmentedBody, true
}

func setUserImagePlaygroundSummary(c *gin.Context, summary *service.OpenAIImageCostSummary) {
	if c == nil || summary == nil {
		return
	}
	c.Set(userImagePlaygroundSummaryContextKey, summary)
}

func getUserImagePlaygroundSummary(c *gin.Context) *service.OpenAIImageCostSummary {
	if c == nil {
		return nil
	}
	value, ok := c.Get(userImagePlaygroundSummaryContextKey)
	if !ok {
		return nil
	}
	summary, _ := value.(*service.OpenAIImageCostSummary)
	return summary
}

func userImageRequestWantsStream(contentType string, body []byte) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		return false
	}
	var payload struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return payload.Stream
}
