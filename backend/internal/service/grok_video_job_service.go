package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

type GrokVideoJobService struct {
	repo    GrokVideoJobRepository
	gateway *OpenAIGatewayService
}

func NewGrokVideoJobService(repo GrokVideoJobRepository, gateway *OpenAIGatewayService) *GrokVideoJobService {
	return &GrokVideoJobService{repo: repo, gateway: gateway}
}

func (s *GrokVideoJobService) CreateJobIfAbsent(ctx context.Context, params CreateGrokVideoJobParams) (*GrokVideoJob, error) {
	params.RequestID = strings.TrimSpace(params.RequestID)
	params.Model = strings.TrimSpace(params.Model)
	params.PromptPreview = NormalizeGrokVideoJobPromptPreview(params.PromptPreview)
	params.Status = NormalizeGrokVideoJobStatus(params.Status)
	if params.Status == "" {
		params.Status = GrokVideoJobStatusPending
	}
	if params.SubmittedAt.IsZero() {
		params.SubmittedAt = time.Now()
	}
	job, err := s.repo.CreateGrokVideoJob(ctx, params)
	if err == nil {
		return job, nil
	}
	if !errors.Is(err, ErrGrokVideoJobExists) {
		return nil, err
	}
	return s.repo.GetGrokVideoJobByRequestID(ctx, params.RequestID)
}

func (s *GrokVideoJobService) Get(ctx context.Context, userID int64, requestID string) (*GrokVideoJob, error) {
	return s.repo.GetGrokVideoJobByRequestIDForUser(ctx, userID, strings.TrimSpace(requestID))
}

func (s *GrokVideoJobService) List(ctx context.Context, userID int64, query GrokVideoJobsQuery) ([]*GrokVideoJob, int64, int, int, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	status := strings.TrimSpace(query.Status)
	if status != "" {
		status = NormalizeGrokVideoJobStatus(status)
	}
	jobs, total, err := s.repo.ListGrokVideoJobsForUser(ctx, userID, GrokVideoJobFilter{
		Status:     status,
		APIKeyID:   query.APIKeyID,
		Model:      strings.TrimSpace(query.Model),
		ActiveOnly: query.ActiveOnly,
		Limit:      pageSize,
		Offset:     (page - 1) * pageSize,
	})
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return jobs, total, page, pageSize, nil
}

func (s *GrokVideoJobService) SyncStatus(ctx context.Context, requestID string, snapshot *GrokMediaVideoStatusSnapshot) (*GrokVideoJob, error) {
	if snapshot == nil {
		return s.repo.GetGrokVideoJobByRequestID(ctx, strings.TrimSpace(requestID))
	}
	status := NormalizeGrokVideoJobStatus(snapshot.Status)
	if status == "" {
		status = GrokVideoJobStatusPending
	}
	progress := snapshot.ProgressPercent
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	if status == GrokVideoJobStatusCompleted && progress < 100 {
		progress = 100
	}
	resultURL := strings.TrimSpace(snapshot.ResultURL)
	resultURLs := sanitizeGrokVideoURLs(snapshot.ResultURLs)
	if resultURL == "" && len(resultURLs) > 0 {
		resultURL = resultURLs[0]
	}
	coverImageURL := strings.TrimSpace(snapshot.CoverImageURL)
	return s.repo.UpdateGrokVideoJobStatus(ctx, strings.TrimSpace(requestID), UpdateGrokVideoJobStatusParams{
		Status:           status,
		ProgressPercent:  progress,
		ProgressText:     strings.TrimSpace(snapshot.ProgressText),
		ResultURL:        resultURL,
		ResultURLs:       resultURLs,
		CoverImageURL:    coverImageURL,
		LastErrorCode:    strings.TrimSpace(snapshot.LastErrorCode),
		LastErrorMessage: strings.TrimSpace(snapshot.LastErrorMessage),
		LastPolledAt:     time.Now(),
		Finished:         IsTerminalGrokVideoJobStatus(status),
	})
}

func (s *GrokVideoJobService) Refresh(ctx context.Context, userID int64, query GrokVideoJobsRefreshQuery) ([]*GrokVideoJob, error) {
	requestIDs := dedupeNonEmptyStrings(query.RequestIDs)
	if len(requestIDs) == 0 && !query.ActiveOnly {
		return nil, ErrGrokVideoJobRefreshTargetMissing
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	jobs, _, err := s.repo.ListGrokVideoJobsForUser(ctx, userID, GrokVideoJobFilter{
		RequestIDs:  requestIDs,
		ActiveOnly:  query.ActiveOnly,
		Limit:       limit,
		Offset:      0,
	})
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if job == nil || IsTerminalGrokVideoJobStatus(job.Status) || job.AccountID == nil || *job.AccountID <= 0 {
			continue
		}
		snapshot, err := s.gateway.RefreshGrokVideoStatusByAccountID(ctx, *job.AccountID, job.RequestID)
		if err != nil {
			continue
		}
		updated, err := s.SyncStatus(ctx, job.RequestID, snapshot)
		if err == nil && updated != nil {
			*job = *updated
		}
	}
	if len(requestIDs) == 0 {
		return jobs, nil
	}
	ordered := make([]*GrokVideoJob, 0, len(jobs))
	jobByRequestID := make(map[string]*GrokVideoJob, len(jobs))
	for _, job := range jobs {
		if job != nil {
			jobByRequestID[job.RequestID] = job
		}
	}
	for _, requestID := range requestIDs {
		if job := jobByRequestID[requestID]; job != nil {
			ordered = append(ordered, job)
		}
	}
	return ordered, nil
}

func dedupeNonEmptyStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func sanitizeGrokVideoURLs(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
