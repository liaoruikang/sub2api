package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeUserImageAPIKeyService struct {
	keys           []service.APIKey
	receivedUserID int64
	receivedParams pagination.PaginationParams
	receivedFilter service.APIKeyListFilters
	receivedKeyID  int64
	getByIDFunc    func(ctx context.Context, id int64) (*service.APIKey, error)
}

func (f *fakeUserImageAPIKeyService) List(_ context.Context, userID int64, params pagination.PaginationParams, filters service.APIKeyListFilters) ([]service.APIKey, *pagination.PaginationResult, error) {
	f.receivedUserID = userID
	f.receivedParams = params
	f.receivedFilter = filters
	return append([]service.APIKey(nil), f.keys...), &pagination.PaginationResult{Total: int64(len(f.keys))}, nil
}

func (f *fakeUserImageAPIKeyService) GetByID(ctx context.Context, id int64) (*service.APIKey, error) {
	f.receivedKeyID = id
	if f.getByIDFunc != nil {
		return f.getByIDFunc(ctx, id)
	}
	for i := range f.keys {
		if f.keys[i].ID == id {
			key := f.keys[i]
			return &key, nil
		}
	}
	return nil, nil
}

type fakeUserImageGateway struct {
	called        bool
	assertContext func(c *gin.Context)
	statusCode    int
	contentType   string
	body          string
	summary       *service.OpenAIImageCostSummary
}

func (f *fakeUserImageGateway) Images(c *gin.Context) {
	f.called = true
	if f.assertContext != nil {
		f.assertContext(c)
	}
	if f.summary != nil {
		setUserImagePlaygroundSummary(c, f.summary)
	}
	statusCode := f.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	contentType := f.contentType
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(statusCode, contentType, []byte(f.body))
}

type fakeUserImageSubscriptionService struct {
	activeSubscription *service.UserSubscription
	receivedUserID     int64
	receivedGroupID    int64
}

func (f *fakeUserImageSubscriptionService) GetActiveSubscription(_ context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	f.receivedUserID = userID
	f.receivedGroupID = groupID
	if f.activeSubscription != nil {
		return f.activeSubscription, nil
	}
	return nil, service.ErrSubscriptionNotFound
}

func (f *fakeUserImageSubscriptionService) ValidateAndCheckLimits(_ *service.UserSubscription, _ *service.Group) (bool, error) {
	return false, nil
}

func (f *fakeUserImageSubscriptionService) DoWindowMaintenance(_ *service.UserSubscription) {}

func TestUserImageHandlerOptionsFiltersEligibleKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	enabledGroupID := int64(101)
	disabledGroupID := int64(102)
	otherUserGroupID := int64(103)
	expiredAt := time.Now().Add(-time.Hour)

	lister := &fakeUserImageAPIKeyService{
		keys: []service.APIKey{
			{
				ID:      1,
				UserID:  7,
				Key:     "sk-test-eligible-1234",
				Name:    "Eligible custom models",
				Status:  service.StatusActive,
				GroupID: &enabledGroupID,
				Group: &service.Group{
					ID:                   enabledGroupID,
					Name:                 "Image Enabled",
					AllowImageGeneration: true,
					ModelsListConfig: service.GroupModelsListConfig{
						Enabled: true,
						Models:  []string{" custom-image-a ", "", "gpt-5.4", "custom-image-b"},
					},
				},
			},
			{
				ID:     2,
				UserID: 7,
				Key:    "sk-test-inactive-5678",
				Name:   "Inactive key",
				Status: service.StatusDisabled,
				Group:  &service.Group{AllowImageGeneration: true},
			},
			{
				ID:      3,
				UserID:  7,
				Key:     "sk-test-disabled-9012",
				Name:    "Disabled image group",
				Status:  service.StatusActive,
				GroupID: &disabledGroupID,
				Group: &service.Group{
					ID:                   disabledGroupID,
					Name:                 "Image Disabled",
					AllowImageGeneration: false,
				},
			},
			{
				ID:        4,
				UserID:    7,
				Key:       "sk-test-expired-3456",
				Name:      "Expired key",
				Status:    service.StatusActive,
				ExpiresAt: &expiredAt,
				Group:     &service.Group{AllowImageGeneration: true},
			},
			{
				ID:      5,
				UserID:  8,
				Key:     "sk-test-other-user-7890",
				Name:    "Other user key",
				Status:  service.StatusActive,
				GroupID: &otherUserGroupID,
				Group: &service.Group{
					ID:                   otherUserGroupID,
					Name:                 "Other User Group",
					AllowImageGeneration: true,
				},
			},
			{
				ID:     6,
				UserID: 7,
				Key:    "sk-test-ungrouped-2468",
				Name:   "Ungrouped key",
				Status: service.StatusActive,
			},
		},
	}

	h := NewUserImageHandlerWithDeps(lister, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/images/options", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})

	h.Options(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(7), lister.receivedUserID)
	require.Equal(t, 1, lister.receivedParams.Page)
	require.Equal(t, 1000, lister.receivedParams.PageSize)
	require.Empty(t, lister.receivedFilter.Search)
	require.Empty(t, lister.receivedFilter.Status)
	require.Nil(t, lister.receivedFilter.GroupID)

	body := recorder.Body.String()
	require.NotContains(t, body, "sk-test-eligible-1234")
	require.NotContains(t, body, "sk-test-ungrouped-2468")
	require.NotContains(t, body, "sk-test-inactive-5678")
	require.Contains(t, body, "sk-...1234")
	require.Contains(t, body, "sk-...2468")

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Keys []struct {
				ID                   int64    `json:"id"`
				Name                 string   `json:"name"`
				MaskedKey            string   `json:"masked_key"`
				GroupID              *int64   `json:"group_id"`
				GroupName            string   `json:"group_name"`
				AllowImageGeneration bool     `json:"allow_image_generation"`
				Models               []string `json:"models"`
				DefaultModel         string   `json:"default_model"`
			} `json:"keys"`
			FallbackModels []string `json:"fallback_models"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Keys, 2)
	require.Equal(t, []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1"}, resp.Data.FallbackModels)
	for _, model := range resp.Data.FallbackModels {
		require.Truef(t, strings.HasPrefix(model, "gpt-image-"), "fallback model %q must pass the images endpoint model validator", model)
	}

	byName := make(map[string]struct {
		MaskedKey            string
		GroupID              *int64
		GroupName            string
		AllowImageGeneration bool
		Models               []string
		DefaultModel         string
	})
	for _, key := range resp.Data.Keys {
		byName[key.Name] = struct {
			MaskedKey            string
			GroupID              *int64
			GroupName            string
			AllowImageGeneration bool
			Models               []string
			DefaultModel         string
		}{
			MaskedKey:            key.MaskedKey,
			GroupID:              key.GroupID,
			GroupName:            key.GroupName,
			AllowImageGeneration: key.AllowImageGeneration,
			Models:               key.Models,
			DefaultModel:         key.DefaultModel,
		}
	}

	eligible, ok := byName["Eligible custom models"]
	require.True(t, ok)
	require.Equal(t, "sk-...1234", eligible.MaskedKey)
	require.NotNil(t, eligible.GroupID)
	require.Equal(t, enabledGroupID, *eligible.GroupID)
	require.Equal(t, "Image Enabled", eligible.GroupName)
	require.True(t, eligible.AllowImageGeneration)
	require.Equal(t, []string{"custom-image-a", "custom-image-b"}, eligible.Models)
	require.Equal(t, "custom-image-a", eligible.DefaultModel)

	ungrouped, ok := byName["Ungrouped key"]
	require.True(t, ok)
	require.Equal(t, "sk-...2468", ungrouped.MaskedKey)
	require.Nil(t, ungrouped.GroupID)
	require.Empty(t, ungrouped.GroupName)
	require.True(t, ungrouped.AllowImageGeneration)
	require.Equal(t, []string{"gpt-image-2", "gpt-image-1.5", "gpt-image-1"}, ungrouped.Models)
	require.Equal(t, "gpt-image-2", ungrouped.DefaultModel)

	_, hasInactive := byName["Inactive key"]
	require.False(t, hasInactive)
	_, hasDisabled := byName["Disabled image group"]
	require.False(t, hasDisabled)
	_, hasExpired := byName["Expired key"]
	require.False(t, hasExpired)
	_, hasOtherUser := byName["Other user key"]
	require.False(t, hasOtherUser)
}

func TestUserImageHandlerGenerateJSONDelegatesToGenerations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(301)
	apiKey := &service.APIKey{
		ID:      42,
		UserID:  7,
		Key:     "sk-test-4242",
		Name:    "Playable key",
		Status:  service.StatusActive,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			Name:                 "Images",
			Platform:             service.PlatformOpenAI,
			Status:               service.StatusActive,
			Hydrated:             true,
			AllowImageGeneration: true,
		},
		User: &service.User{ID: 7, Role: service.RoleUser},
	}
	keyService := &fakeUserImageAPIKeyService{
		getByIDFunc: func(_ context.Context, id int64) (*service.APIKey, error) {
			require.Equal(t, int64(42), id)
			return apiKey, nil
		},
	}
	actualCost := 0.12
	totalCost := 0.12
	estimatedPrice := 0.12
	gateway := &fakeUserImageGateway{
		body: `{"data":[{"b64_json":"abc"}]}`,
		summary: &service.OpenAIImageCostSummary{
			EstimatedPrice: &estimatedPrice,
			ActualCost:     &actualCost,
			TotalCost:      &totalCost,
			ImageCount:     1,
			ImageSize:      "1K",
			BillingMode:    "image",
		},
		assertContext: func(c *gin.Context) {
			selectedKey, ok := middleware2.GetAPIKeyFromContext(c)
			require.True(t, ok)
			require.Equal(t, int64(42), selectedKey.ID)

			contextGroup, ok := c.Request.Context().Value(ctxkey.Group).(*service.Group)
			require.True(t, ok)
			require.Equal(t, groupID, contextGroup.ID)

			subject, ok := middleware2.GetAuthSubjectFromContext(c)
			require.True(t, ok)
			require.Equal(t, int64(7), subject.UserID)

			require.Equal(t, "/v1/images/generations", c.Request.URL.Path)
			require.Equal(t, http.MethodPost, c.Request.Method)
			require.Contains(t, c.GetHeader("Content-Type"), "application/json")

			body, err := io.ReadAll(c.Request.Body)
			require.NoError(t, err)

			var payload map[string]any
			require.NoError(t, json.Unmarshal(body, &payload))
			_, hasAPIKeyID := payload["api_key_id"]
			require.False(t, hasAPIKeyID)
			require.Equal(t, "gpt-image-1", payload["model"])
			require.Equal(t, "draw a cat", payload["prompt"])
			require.Equal(t, "1024x1024", payload["size"])
			require.EqualValues(t, 2, payload["n"])
		},
	}
	h := NewUserImageHandlerWithDeps(keyService, gateway)

	body := strings.NewReader(`{"api_key_id":42,"model":"gpt-image-1","prompt":"draw a cat","size":"1024x1024","n":2}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/generations", body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleUser)

	h.Generate(c)

	require.True(t, gateway.called)
	require.Equal(t, int64(42), keyService.receivedKeyID)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"data":[{"b64_json":"abc"}],"_sub2api_image_playground":{"estimated_price":0.12,"actual_cost":0.12,"total_cost":0.12,"image_count":1,"image_size":"1K","billing_mode":"image"}}`, recorder.Body.String())
}

func TestUserImageHandlerGenerateMultipartDelegatesToEdits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(302)
	apiKey := &service.APIKey{
		ID:      51,
		UserID:  7,
		Key:     "sk-test-5151",
		Status:  service.StatusActive,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			Name:                 "Images",
			AllowImageGeneration: true,
		},
		User: &service.User{ID: 7, Role: service.RoleUser},
	}
	keyService := &fakeUserImageAPIKeyService{
		getByIDFunc: func(_ context.Context, id int64) (*service.APIKey, error) {
			require.Equal(t, int64(51), id)
			return apiKey, nil
		},
	}
	gateway := &fakeUserImageGateway{
		body: `{"data":[{"url":"https://example.com/image.png"}]}`,
		assertContext: func(c *gin.Context) {
			require.Equal(t, "/v1/images/edits", c.Request.URL.Path)
			require.Contains(t, c.GetHeader("Content-Type"), "multipart/form-data")

			body, err := io.ReadAll(c.Request.Body)
			require.NoError(t, err)

			mediaType, params, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
			require.NoError(t, err)
			require.Equal(t, "multipart/form-data", mediaType)

			reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
			fields := map[string]string{}
			hasImage := false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				content, err := io.ReadAll(part)
				require.NoError(t, err)
				if part.FormName() == "image" {
					hasImage = true
					require.Equal(t, "reference.png", part.FileName())
					require.Equal(t, "fake image bytes", string(content))
					continue
				}
				fields[part.FormName()] = string(content)
			}

			require.True(t, hasImage)
			require.Equal(t, "gpt-image-1", fields["model"])
			require.Equal(t, "edit this image", fields["prompt"])
			_, hasAPIKeyID := fields["api_key_id"]
			require.False(t, hasAPIKeyID)
		},
	}
	h := NewUserImageHandlerWithDeps(keyService, gateway)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("api_key_id", "51"))
	require.NoError(t, writer.WriteField("model", "gpt-image-1"))
	require.NoError(t, writer.WriteField("prompt", "edit this image"))
	fileWriter, err := writer.CreateFormFile("image", "reference.png")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("fake image bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/generations", bytes.NewReader(body.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleUser)

	h.Generate(c)

	require.True(t, gateway.called)
	require.Equal(t, int64(51), keyService.receivedKeyID)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"data":[{"url":"https://example.com/image.png"}]}`, recorder.Body.String())
}

func TestUserImageHandlerGeneratePreservesExistingResponseHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKey := &service.APIKey{
		ID:     88,
		UserID: 7,
		Status: service.StatusActive,
		Group:  &service.Group{AllowImageGeneration: true},
		User:   &service.User{ID: 7, Role: service.RoleUser},
	}
	keyService := &fakeUserImageAPIKeyService{
		getByIDFunc: func(_ context.Context, id int64) (*service.APIKey, error) {
			require.Equal(t, int64(88), id)
			return apiKey, nil
		},
	}
	gateway := &fakeUserImageGateway{body: `{"data":[{"b64_json":"abc"}]}`}
	h := NewUserImageHandlerWithDeps(keyService, gateway)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/generations", strings.NewReader(`{"api_key_id":88,"model":"gpt-image-1","prompt":"draw"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleUser)
	c.Writer.Header().Set("Access-Control-Allow-Origin", "https://app.example")
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

	h.Generate(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "https://app.example", recorder.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
}

func TestUserImageHandlerGenerateSetsSubscriptionContextForSubscriptionGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(89)
	subscription := &service.UserSubscription{
		ID:        123,
		UserID:    7,
		GroupID:   groupID,
		Status:    service.SubscriptionStatusActive,
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	apiKey := &service.APIKey{
		ID:      89,
		UserID:  7,
		Status:  service.StatusActive,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                   groupID,
			AllowImageGeneration: true,
			SubscriptionType:     service.SubscriptionTypeSubscription,
		},
		User: &service.User{ID: 7, Role: service.RoleUser, Status: service.StatusActive},
	}
	keyService := &fakeUserImageAPIKeyService{
		getByIDFunc: func(_ context.Context, id int64) (*service.APIKey, error) {
			require.Equal(t, int64(89), id)
			return apiKey, nil
		},
	}
	subscriptionService := &fakeUserImageSubscriptionService{activeSubscription: subscription}
	gateway := &fakeUserImageGateway{
		body: `{"data":[{"b64_json":"abc"}]}`,
		assertContext: func(c *gin.Context) {
			got, ok := middleware2.GetSubscriptionFromContext(c)
			require.True(t, ok)
			require.Equal(t, subscription.ID, got.ID)
		},
	}
	h := NewUserImageHandlerWithDeps(keyService, gateway, subscriptionService)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/generations", strings.NewReader(`{"api_key_id":89,"model":"gpt-image-1","prompt":"draw"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleUser)

	h.Generate(c)

	require.True(t, gateway.called)
	require.Equal(t, int64(7), subscriptionService.receivedUserID)
	require.Equal(t, groupID, subscriptionService.receivedGroupID)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestUserImageHandlerGenerateRejectsOtherUsersKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(303)
	keyService := &fakeUserImageAPIKeyService{
		getByIDFunc: func(_ context.Context, id int64) (*service.APIKey, error) {
			require.Equal(t, int64(77), id)
			return &service.APIKey{
				ID:      77,
				UserID:  99,
				Status:  service.StatusActive,
				GroupID: &groupID,
				Group: &service.Group{
					ID:                   groupID,
					AllowImageGeneration: true,
				},
				User: &service.User{ID: 99, Role: service.RoleUser},
			}, nil
		},
	}
	gateway := &fakeUserImageGateway{}
	h := NewUserImageHandlerWithDeps(keyService, gateway)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/user/images/generations", strings.NewReader(`{"api_key_id":77,"model":"gpt-image-1","prompt":"draw a cat"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	c.Set(string(middleware2.ContextKeyUserRole), service.RoleUser)

	h.Generate(c)

	require.False(t, gateway.called)
	require.Equal(t, int64(77), keyService.receivedKeyID)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}
