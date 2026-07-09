package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GrokVideoJobHandler struct {
	service *service.GrokVideoJobService
}

func NewGrokVideoJobHandler(service *service.GrokVideoJobService) *GrokVideoJobHandler {
	return &GrokVideoJobHandler{service: service}
}

type grokVideoJobRefreshRequest struct {
	RequestIDs []string `json:"request_ids"`
	ActiveOnly bool     `json:"active_only"`
	Limit      int      `json:"limit"`
}

func (h *GrokVideoJobHandler) List(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, pageSize := response.ParsePagination(c)
	apiKeyID, err := parseOptionalInt64(c.Query("api_key_id"))
	if err != nil {
		response.BadRequest(c, "invalid api_key_id")
		return
	}
	jobs, total, page, pageSize, err := h.service.List(c.Request.Context(), subject.UserID, service.GrokVideoJobsQuery{
		Page:       page,
		PageSize:   pageSize,
		Status:     c.Query("status"),
		APIKeyID:   apiKeyID,
		Model:      c.Query("model"),
		ActiveOnly: parseBoolQuery(c.Query("active_only")),
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Paginated(c, jobs, total, page, pageSize)
}

func (h *GrokVideoJobHandler) Get(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	job, err := h.service.Get(c.Request.Context(), subject.UserID, c.Param("request_id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, job)
}

func (h *GrokVideoJobHandler) Refresh(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req grokVideoJobRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}
	jobs, err := h.service.Refresh(c.Request.Context(), subject.UserID, service.GrokVideoJobsRefreshQuery{
		RequestIDs: req.RequestIDs,
		ActiveOnly: req.ActiveOnly,
		Limit:      req.Limit,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"items": jobs})
}

func parseOptionalInt64(value string) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

