package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type EmailVerificationHandler struct {
	authService *service.AuthService
}

func NewEmailVerificationHandler(authService *service.AuthService) *EmailVerificationHandler {
	return &EmailVerificationHandler{authService: authService}
}

type VerifyEmailCodeRequest struct {
	Email      string `json:"email" binding:"required,email"`
	VerifyCode string `json:"verify_code" binding:"required"`
}

type VerifyEmailCodeResponse struct {
	Valid bool   `json:"valid"`
	Email string `json:"email"`
}

func (h *EmailVerificationHandler) Verify(c *gin.Context) {
	if h == nil || h.authService == nil {
		response.ErrorFrom(c, service.ErrServiceUnavailable)
		return
	}

	var req VerifyEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	verifyCode := strings.TrimSpace(req.VerifyCode)

	if err := h.authService.VerifyOAuthEmailCode(c.Request.Context(), email, verifyCode); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, VerifyEmailCodeResponse{Valid: true, Email: email})
}
