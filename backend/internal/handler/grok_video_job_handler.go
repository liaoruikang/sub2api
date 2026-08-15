package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GrokVideoJobHandler struct {
	service  *service.GrokVideoJobService
	seedance *SeedanceHandler
}

func NewGrokVideoJobHandler(service *service.GrokVideoJobService) *GrokVideoJobHandler {
	return &GrokVideoJobHandler{service: service}
}

func (h *GrokVideoJobHandler) SetSeedanceHandler(seedance *SeedanceHandler) {
	h.seedance = seedance
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
	if h.seedance != nil {
		seedanceList, seedanceErr := h.seedance.ListUserVideoTasks(c.Request.Context(), subject.UserID, apiKeyID, page, pageSize)
		if seedanceErr == nil && seedanceList != nil {
			for i := range seedanceList.Items {
				job := seedanceTaskToGrokVideoJob(&seedanceList.Items[i])
				if !matchesUnifiedVideoJob(job, c.Query("status"), apiKeyID, c.Query("model"), parseBoolQuery(c.Query("active_only"))) {
					continue
				}
				jobs = append(jobs, job)
			}
			total += seedanceList.Total
			sort.SliceStable(jobs, func(i, j int) bool { return jobs[i].SubmittedAt.After(jobs[j].SubmittedAt) })
			if len(jobs) > pageSize {
				jobs = jobs[:pageSize]
			}
		}
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
	if err == nil {
		response.Success(c, job)
		return
	}
	if h.seedance != nil {
		keys, _, keyErr := h.seedance.apiKeys.List(c.Request.Context(), subject.UserID, pagination.PaginationParams{Page: 1, PageSize: 100}, service.APIKeyListFilters{})
		if keyErr == nil {
			for i := range keys {
				if keys[i].Group == nil || keys[i].Group.Platform != service.PlatformSeedance {
					continue
				}
				keyID := keys[i].ID
				if task, taskErr := h.seedance.GetUserVideoTask(c.Request.Context(), subject.UserID, &keyID, c.Param("request_id")); taskErr == nil {
					response.Success(c, seedanceTaskToGrokVideoJob(task))
					return
				}
			}
		}
	}
	response.ErrorFrom(c, err)
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
	if h.seedance != nil {
		seedanceJobs, _ := h.seedance.RefreshUserVideoTasks(c, subject.UserID, req.RequestIDs, req.ActiveOnly, req.Limit)
		for _, task := range seedanceJobs {
			jobs = append(jobs, seedanceTaskToGrokVideoJob(task))
		}
	}
	response.Success(c, gin.H{"items": jobs})
}

func matchesUnifiedVideoJob(job *service.GrokVideoJob, status string, apiKeyID *int64, model string, activeOnly bool) bool {
	if job == nil {
		return false
	}
	if status != "" && service.NormalizeGrokVideoJobStatus(job.Status) != service.NormalizeGrokVideoJobStatus(status) {
		return false
	}
	if apiKeyID != nil && (job.APIKeyID == nil || *job.APIKeyID != *apiKeyID) {
		return false
	}
	if model != "" && !strings.EqualFold(strings.TrimSpace(job.Model), strings.TrimSpace(model)) {
		return false
	}
	if activeOnly && service.IsTerminalGrokVideoJobStatus(job.Status) {
		return false
	}
	return true
}

func seedanceTaskToGrokVideoJob(task *service.SeedanceVideoTask) *service.GrokVideoJob {
	if task == nil {
		return nil
	}
	status := service.NormalizeGrokVideoJobStatus(task.Status)
	progress := 0
	if status == service.GrokVideoJobStatusCompleted {
		progress = 100
	}
	finishedAt := task.CompletedAt
	job := &service.GrokVideoJob{
		ID: task.ID, RequestID: task.TaskID, UserID: task.UserID, APIKeyID: task.APIKeyID,
		GroupID: task.GroupID, AccountID: &task.AccountID, Model: task.Model,
		Status: status, ProgressPercent: progress, CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt, SubmittedAt: task.CreatedAt, FinishedAt: finishedAt,
	}
	if len(task.RequestBody) > 0 {
		var payload map[string]any
		if json.Unmarshal(task.RequestBody, &payload) == nil {
			if prompt, ok := payload["prompt"].(string); ok {
				job.PromptPreview = service.NormalizeGrokVideoJobPromptPreview(prompt)
			}
		}
	}
	if len(task.ResponseBody) > 0 {
		var payload map[string]any
		if json.Unmarshal(task.ResponseBody, &payload) == nil {
			if taskMap, ok := payload["task"].(map[string]any); ok {
				if outputs, ok := taskMap["outputs"].([]any); ok {
					for _, item := range outputs {
						if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
							clean := strings.TrimSpace(value)
							job.ResultURLs = append(job.ResultURLs, clean)
							continue
						}
						if output, ok := item.(map[string]any); ok {
							for _, key := range []string{"url", "video_url", "download_url"} {
								if value, ok := output[key].(string); ok && strings.TrimSpace(value) != "" {
									job.ResultURLs = append(job.ResultURLs, value)
									break
								}
							}
						}
					}
				}
			}
		}
	}
	if len(job.ResultURLs) > 0 {
		first := job.ResultURLs[0]
		job.ResultURL = &first
	}
	return job
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
