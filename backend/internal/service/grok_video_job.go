package service

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	GrokVideoJobStatusPending   = "pending"
	GrokVideoJobStatusRunning   = "running"
	GrokVideoJobStatusCompleted = "completed"
	GrokVideoJobStatusFailed    = "failed"
	GrokVideoJobStatusCancelled = "cancelled"
)

var (
	ErrGrokVideoJobNotFound             = infraerrors.New(http.StatusNotFound, "GROK_VIDEO_JOB_NOT_FOUND", "grok video job not found")
	ErrGrokVideoJobExists               = infraerrors.New(http.StatusConflict, "GROK_VIDEO_JOB_EXISTS", "grok video job already exists")
	ErrGrokVideoJobRefreshTargetMissing = infraerrors.New(http.StatusBadRequest, "GROK_VIDEO_REFRESH_TARGET_REQUIRED", "refresh target is required")
)

type GrokVideoJob struct {
	ID               int64      `json:"id"`
	RequestID        string     `json:"request_id"`
	UserID           int64      `json:"user_id"`
	APIKeyID         *int64     `json:"api_key_id,omitempty"`
	GroupID          *int64     `json:"group_id,omitempty"`
	AccountID        *int64     `json:"account_id,omitempty"`
	Model            string     `json:"model"`
	PromptPreview    string     `json:"prompt_preview"`
	Status           string     `json:"status"`
	ProgressPercent  int        `json:"progress"`
	ProgressText     string     `json:"progress_text,omitempty"`
	ResultURL        *string    `json:"result_url,omitempty"`
	ResultURLs       []string   `json:"result_urls,omitempty"`
	CoverImageURL    *string    `json:"cover_image_url,omitempty"`
	LastErrorCode    *string    `json:"last_error_code,omitempty"`
	LastErrorMessage *string    `json:"last_error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	SubmittedAt      time.Time  `json:"submitted_at"`
	LastPolledAt     *time.Time `json:"last_polled_at,omitempty"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

type CreateGrokVideoJobParams struct {
	RequestID     string
	UserID        int64
	APIKeyID      *int64
	GroupID       *int64
	AccountID     *int64
	Model         string
	PromptPreview string
	Status        string
	SubmittedAt   time.Time
}

type UpdateGrokVideoJobStatusParams struct {
	Status           string
	ProgressPercent  int
	ProgressText     string
	ResultURL        string
	ResultURLs       []string
	CoverImageURL    string
	LastErrorCode    string
	LastErrorMessage string
	LastPolledAt     time.Time
	Finished         bool
}

type GrokVideoJobFilter struct {
	Status     string
	APIKeyID   *int64
	Model      string
	ActiveOnly bool
	RequestIDs []string
	Limit      int
	Offset     int
}

type GrokVideoJobsQuery struct {
	Page       int
	PageSize   int
	Status     string
	APIKeyID   *int64
	Model      string
	ActiveOnly bool
}

type GrokVideoJobsRefreshQuery struct {
	RequestIDs []string
	ActiveOnly bool
	Limit      int
}

type GrokVideoJobRepository interface {
	CreateGrokVideoJob(ctx context.Context, params CreateGrokVideoJobParams) (*GrokVideoJob, error)
	GetGrokVideoJobByRequestID(ctx context.Context, requestID string) (*GrokVideoJob, error)
	GetGrokVideoJobByRequestIDForUser(ctx context.Context, userID int64, requestID string) (*GrokVideoJob, error)
	ListGrokVideoJobsForUser(ctx context.Context, userID int64, filter GrokVideoJobFilter) ([]*GrokVideoJob, int64, error)
	UpdateGrokVideoJobStatus(ctx context.Context, requestID string, params UpdateGrokVideoJobStatusParams) (*GrokVideoJob, error)
}

type GrokMediaVideoStatusSnapshot struct {
	RequestID        string   `json:"request_id,omitempty"`
	Status           string   `json:"status,omitempty"`
	ProgressPercent  int      `json:"progress,omitempty"`
	ProgressText     string   `json:"progress_text,omitempty"`
	ResultURL        string   `json:"result_url,omitempty"`
	ResultURLs       []string `json:"result_urls,omitempty"`
	CoverImageURL    string   `json:"cover_image_url,omitempty"`
	LastErrorCode    string   `json:"last_error_code,omitempty"`
	LastErrorMessage string   `json:"last_error_message,omitempty"`
}

func NormalizeGrokVideoJobStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "", "queued", "created", "submitted":
		return GrokVideoJobStatusPending
	case "pending":
		return GrokVideoJobStatusPending
	case "in_progress", "processing", "running":
		return GrokVideoJobStatusRunning
	case "done", "succeeded", "success", "completed", "complete", "finished":
		return GrokVideoJobStatusCompleted
	case "failed", "error":
		return GrokVideoJobStatusFailed
	case "cancelled", "canceled":
		return GrokVideoJobStatusCancelled
	default:
		return status
	}
}

func IsTerminalGrokVideoJobStatus(status string) bool {
	switch NormalizeGrokVideoJobStatus(status) {
	case GrokVideoJobStatusCompleted, GrokVideoJobStatusFailed, GrokVideoJobStatusCancelled:
		return true
	default:
		return false
	}
}

func NormalizeGrokVideoJobPromptPreview(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	const maxRunes = 240
	if utf8.RuneCountInString(prompt) <= maxRunes {
		return prompt
	}
	runes := []rune(prompt)
	return strings.TrimSpace(string(runes[:maxRunes]))
}
