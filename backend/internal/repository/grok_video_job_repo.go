package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type grokVideoJobRepository struct {
	sql *sql.DB
}

func NewGrokVideoJobRepository(db *sql.DB) service.GrokVideoJobRepository {
	return &grokVideoJobRepository{sql: db}
}

func (r *grokVideoJobRepository) CreateGrokVideoJob(ctx context.Context, params service.CreateGrokVideoJobParams) (*service.GrokVideoJob, error) {
	job, err := scanGrokVideoJob(r.sql.QueryRowContext(ctx, `
INSERT INTO grok_video_jobs (
    request_id, user_id, api_key_id, group_id, account_id, model, prompt_preview, status, submitted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING `+grokVideoJobColumns,
		params.RequestID,
		params.UserID,
		params.APIKeyID,
		params.GroupID,
		params.AccountID,
		params.Model,
		params.PromptPreview,
		params.Status,
		params.SubmittedAt,
	))
	if err != nil {
		return nil, translatePersistenceError(err, nil, service.ErrGrokVideoJobExists)
	}
	return job, nil
}

func (r *grokVideoJobRepository) GetGrokVideoJobByRequestID(ctx context.Context, requestID string) (*service.GrokVideoJob, error) {
	job, err := scanGrokVideoJob(r.sql.QueryRowContext(ctx, grokVideoJobSelectSQL+` WHERE request_id = $1`, requestID))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGrokVideoJobNotFound, nil)
	}
	return job, nil
}

func (r *grokVideoJobRepository) GetGrokVideoJobByRequestIDForUser(ctx context.Context, userID int64, requestID string) (*service.GrokVideoJob, error) {
	job, err := scanGrokVideoJob(r.sql.QueryRowContext(ctx, grokVideoJobSelectSQL+` WHERE user_id = $1 AND request_id = $2`, userID, requestID))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGrokVideoJobNotFound, nil)
	}
	return job, nil
}

func (r *grokVideoJobRepository) ListGrokVideoJobsForUser(ctx context.Context, userID int64, filter service.GrokVideoJobFilter) ([]*service.GrokVideoJob, int64, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	where, args := buildGrokVideoJobWhere(userID, filter)
	countQuery := `SELECT COUNT(*) FROM grok_video_jobs` + where
	var total int64
	if err := r.sql.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), limit, filter.Offset)
	query := grokVideoJobSelectSQL + where + ` ORDER BY created_at DESC, id DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	rows, err := r.sql.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	jobs, err := scanGrokVideoJobs(rows)
	if err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (r *grokVideoJobRepository) UpdateGrokVideoJobStatus(ctx context.Context, requestID string, params service.UpdateGrokVideoJobStatusParams) (*service.GrokVideoJob, error) {
	if params.LastPolledAt.IsZero() {
		params.LastPolledAt = time.Now()
	}
	var resultURLsArg any
	if len(params.ResultURLs) > 0 {
		payload, err := json.Marshal(params.ResultURLs)
		if err != nil {
			return nil, err
		}
		resultURLsArg = string(payload)
	}
	job, err := scanGrokVideoJob(r.sql.QueryRowContext(ctx, `
UPDATE grok_video_jobs
SET status = $2,
    progress_percent = $3,
    progress_text = $4,
    result_url = CASE WHEN NULLIF($5, '') IS NOT NULL THEN NULLIF($5, '') ELSE result_url END,
    result_urls = CASE WHEN NULLIF($6, '') IS NOT NULL THEN NULLIF($6, '') ELSE result_urls END,
    cover_image_url = CASE WHEN NULLIF($7, '') IS NOT NULL THEN NULLIF($7, '') ELSE cover_image_url END,
    last_error_code = CASE WHEN NULLIF($8, '') IS NOT NULL THEN NULLIF($8, '') ELSE last_error_code END,
    last_error_message = CASE WHEN NULLIF($9, '') IS NOT NULL THEN NULLIF($9, '') ELSE last_error_message END,
    last_polled_at = $10,
    finished_at = CASE WHEN $11 AND finished_at IS NULL THEN $10 ELSE finished_at END,
    updated_at = $10
WHERE request_id = $1
RETURNING `+grokVideoJobColumns,
		requestID,
		params.Status,
		params.ProgressPercent,
		params.ProgressText,
		params.ResultURL,
		resultURLsArg,
		params.CoverImageURL,
		params.LastErrorCode,
		params.LastErrorMessage,
		params.LastPolledAt,
		params.Finished,
	))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrGrokVideoJobNotFound, nil)
	}
	return job, nil
}

func buildGrokVideoJobWhere(userID int64, filter service.GrokVideoJobFilter) (string, []any) {
	parts := []string{" WHERE user_id = $1"}
	args := []any{userID}
	if status := strings.TrimSpace(filter.Status); status != "" {
		normalizedStatus := service.NormalizeGrokVideoJobStatus(status)
		if normalizedStatus == service.GrokVideoJobStatusCompleted {
			parts = append(parts, ` AND LOWER(TRIM(status)) IN ('done', 'succeeded', 'success', 'completed', 'complete', 'finished')`)
		} else if normalizedStatus == service.GrokVideoJobStatusFailed {
			parts = append(parts, ` AND LOWER(TRIM(status)) IN ('failed', 'error')`)
		} else if normalizedStatus == service.GrokVideoJobStatusCancelled {
			parts = append(parts, ` AND LOWER(TRIM(status)) IN ('cancelled', 'canceled')`)
		} else if normalizedStatus == service.GrokVideoJobStatusRunning {
			parts = append(parts, ` AND LOWER(TRIM(status)) IN ('in_progress', 'processing', 'running')`)
		} else if normalizedStatus == service.GrokVideoJobStatusPending {
			parts = append(parts, ` AND LOWER(TRIM(status)) IN ('', 'queued', 'created', 'submitted', 'pending')`)
		} else {
			parts = append(parts, ` AND LOWER(TRIM(status)) = $`+strconv.Itoa(len(args)+1))
			args = append(args, strings.ToLower(status))
		}
	}
	if filter.APIKeyID != nil && *filter.APIKeyID > 0 {
		parts = append(parts, ` AND api_key_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *filter.APIKeyID)
	}
	if model := strings.TrimSpace(filter.Model); model != "" {
		parts = append(parts, ` AND model = $`+strconv.Itoa(len(args)+1))
		args = append(args, model)
	}
	if filter.ActiveOnly {
		parts = append(parts, ` AND LOWER(TRIM(status)) NOT IN ('done', 'succeeded', 'success', 'completed', 'complete', 'finished', 'failed', 'error', 'cancelled', 'canceled')`)
	}
	if len(filter.RequestIDs) > 0 {
		placeholders := make([]string, 0, len(filter.RequestIDs))
		for _, requestID := range filter.RequestIDs {
			requestID = strings.TrimSpace(requestID)
			if requestID == "" {
				continue
			}
			placeholders = append(placeholders, `$`+strconv.Itoa(len(args)+1))
			args = append(args, requestID)
		}
		if len(placeholders) > 0 {
			parts = append(parts, ` AND request_id IN (`+strings.Join(placeholders, ", ")+`)`)
		}
	}
	return strings.Join(parts, ""), args
}

type grokVideoRowScanner interface {
	Scan(dest ...any) error
}

const grokVideoJobColumns = `
id, request_id, user_id, api_key_id, group_id, account_id, model, prompt_preview,
status, progress_percent, progress_text, result_url, result_urls, cover_image_url,
last_error_code, last_error_message, created_at, updated_at, submitted_at, last_polled_at, finished_at`

const grokVideoJobSelectSQL = `SELECT ` + grokVideoJobColumns + ` FROM grok_video_jobs`

func scanGrokVideoJob(row grokVideoRowScanner) (*service.GrokVideoJob, error) {
	var job service.GrokVideoJob
	var apiKeyID, groupID, accountID sql.NullInt64
	var progressText, resultURL, resultURLsRaw, coverImageURL sql.NullString
	var lastErrorCode, lastErrorMessage sql.NullString
	var lastPolledAt, finishedAt sql.NullTime
	if err := row.Scan(
		&job.ID,
		&job.RequestID,
		&job.UserID,
		&apiKeyID,
		&groupID,
		&accountID,
		&job.Model,
		&job.PromptPreview,
		&job.Status,
		&job.ProgressPercent,
		&progressText,
		&resultURL,
		&resultURLsRaw,
		&coverImageURL,
		&lastErrorCode,
		&lastErrorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.SubmittedAt,
		&lastPolledAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}
	job.APIKeyID = grokVideoNullInt64Ptr(apiKeyID)
	job.GroupID = grokVideoNullInt64Ptr(groupID)
	job.AccountID = grokVideoNullInt64Ptr(accountID)
	if progressText.Valid {
		job.ProgressText = progressText.String
	}
	job.ResultURL = grokVideoNullStringPtr(resultURL)
	job.CoverImageURL = grokVideoNullStringPtr(coverImageURL)
	job.LastErrorCode = grokVideoNullStringPtr(lastErrorCode)
	job.LastErrorMessage = grokVideoNullStringPtr(lastErrorMessage)
	job.LastPolledAt = grokVideoNullTimePtr(lastPolledAt)
	job.FinishedAt = grokVideoNullTimePtr(finishedAt)
	if resultURLsRaw.Valid && strings.TrimSpace(resultURLsRaw.String) != "" {
		_ = json.Unmarshal([]byte(resultURLsRaw.String), &job.ResultURLs)
	}
	return &job, nil
}

func scanGrokVideoJobs(rows *sql.Rows) ([]*service.GrokVideoJob, error) {
	var jobs []*service.GrokVideoJob
	for rows.Next() {
		job, err := scanGrokVideoJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func grokVideoNullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func grokVideoNullInt64Ptr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func grokVideoNullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}

var _ service.GrokVideoJobRepository = (*grokVideoJobRepository)(nil)
