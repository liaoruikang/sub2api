package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type emailVerificationCacheStub struct {
	verificationCodes map[string]*service.VerificationCodeData
}

func newEmailVerificationCacheStub() *emailVerificationCacheStub {
	return &emailVerificationCacheStub{verificationCodes: map[string]*service.VerificationCodeData{}}
}

func (s *emailVerificationCacheStub) GetVerificationCode(ctx context.Context, email string) (*service.VerificationCodeData, error) {
	return s.verificationCodes[email], nil
}

func (s *emailVerificationCacheStub) SetVerificationCode(ctx context.Context, email string, data *service.VerificationCodeData, ttl time.Duration) error {
	s.verificationCodes[email] = data
	return nil
}

func (s *emailVerificationCacheStub) DeleteVerificationCode(ctx context.Context, email string) error {
	delete(s.verificationCodes, email)
	return nil
}

func (s *emailVerificationCacheStub) SetNotifyVerifyCode(ctx context.Context, email string, data *service.VerificationCodeData, ttl time.Duration) error {
	panic("unexpected SetNotifyVerifyCode call")
}

func (s *emailVerificationCacheStub) GetNotifyVerifyCode(ctx context.Context, email string) (*service.VerificationCodeData, error) {
	panic("unexpected GetNotifyVerifyCode call")
}

func (s *emailVerificationCacheStub) DeleteNotifyVerifyCode(ctx context.Context, email string) error {
	panic("unexpected DeleteNotifyVerifyCode call")
}

func (s *emailVerificationCacheStub) IncrNotifyCodeEmailRate(ctx context.Context, email string, window time.Duration) (int64, error) {
	panic("unexpected IncrNotifyCodeEmailRate call")
}

func (s *emailVerificationCacheStub) GetNotifyCodeEmailRate(ctx context.Context, email string) (int64, error) {
	panic("unexpected GetNotifyCodeEmailRate call")
}

func (s *emailVerificationCacheStub) IncrNotifyCodeUserRate(ctx context.Context, userID int64, window time.Duration) (int64, error) {
	panic("unexpected IncrNotifyCodeUserRate call")
}

func (s *emailVerificationCacheStub) GetNotifyCodeUserRate(ctx context.Context, userID int64) (int64, error) {
	panic("unexpected GetNotifyCodeUserRate call")
}

func (s *emailVerificationCacheStub) GetPasswordResetToken(ctx context.Context, email string) (*service.PasswordResetTokenData, error) {
	panic("unexpected GetPasswordResetToken call")
}

func (s *emailVerificationCacheStub) SetPasswordResetToken(ctx context.Context, email string, data *service.PasswordResetTokenData, ttl time.Duration) error {
	panic("unexpected SetPasswordResetToken call")
}

func (s *emailVerificationCacheStub) DeletePasswordResetToken(ctx context.Context, email string) error {
	panic("unexpected DeletePasswordResetToken call")
}

func (s *emailVerificationCacheStub) IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool {
	panic("unexpected IsPasswordResetEmailInCooldown call")
}

func (s *emailVerificationCacheStub) SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error {
	panic("unexpected SetPasswordResetEmailCooldown call")
}

func newEmailVerificationTestRouter(cache *emailVerificationCacheStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	emailService := service.NewEmailService(nil, cache)
	authService := service.NewAuthService(nil, nil, nil, nil, nil, nil, emailService, nil, nil, nil, nil, nil, nil)
	handler := NewEmailVerificationHandler(authService)
	router.POST("/api/v1/admin/email-verifications/verify", handler.Verify)
	return router
}

func postEmailVerification(router *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/email-verifications/verify", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func TestEmailVerificationHandlerVerifySuccess(t *testing.T) {
	cache := newEmailVerificationCacheStub()
	cache.verificationCodes["user@example.com"] = &service.VerificationCodeData{
		Code:      "246810",
		Attempts:  0,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
	}
	router := newEmailVerificationTestRouter(cache)

	rec := postEmailVerification(router, map[string]any{
		"email":       "User@Example.COM",
		"verify_code": "246810",
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":0`)
	require.Contains(t, rec.Body.String(), `"valid":true`)
	require.Contains(t, rec.Body.String(), `"email":"user@example.com"`)
	require.Nil(t, cache.verificationCodes["user@example.com"])
}

func TestEmailVerificationHandlerRejectsWrongCode(t *testing.T) {
	cache := newEmailVerificationCacheStub()
	cache.verificationCodes["user@example.com"] = &service.VerificationCodeData{
		Code:      "246810",
		Attempts:  0,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
	}
	router := newEmailVerificationTestRouter(cache)

	rec := postEmailVerification(router, map[string]any{
		"email":       "user@example.com",
		"verify_code": "000000",
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid or expired verification code")
	entry := cache.verificationCodes["user@example.com"]
	require.NotNil(t, entry)
	require.Equal(t, 1, entry.Attempts)
}

func TestEmailVerificationHandlerRejectsInvalidBody(t *testing.T) {
	cache := newEmailVerificationCacheStub()
	router := newEmailVerificationTestRouter(cache)

	rec := postEmailVerification(router, map[string]any{
		"email": "not-an-email",
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid request")
}

func TestEmailVerificationHandlerServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewEmailVerificationHandler(nil)
	router.POST("/api/v1/admin/email-verifications/verify", handler.Verify)

	rec := postEmailVerification(router, map[string]any{
		"email":       "user@example.com",
		"verify_code": "246810",
	})

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "service temporarily unavailable")
}
