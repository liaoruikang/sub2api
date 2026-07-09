package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type listCaptureGrokVideoJobRepo struct {
	lastUserID int64
	lastFilter GrokVideoJobFilter
}

func (r *listCaptureGrokVideoJobRepo) CreateGrokVideoJob(ctx context.Context, params CreateGrokVideoJobParams) (*GrokVideoJob, error) {
	panic("unexpected call")
}

func (r *listCaptureGrokVideoJobRepo) GetGrokVideoJobByRequestID(ctx context.Context, requestID string) (*GrokVideoJob, error) {
	panic("unexpected call")
}

func (r *listCaptureGrokVideoJobRepo) GetGrokVideoJobByRequestIDForUser(ctx context.Context, userID int64, requestID string) (*GrokVideoJob, error) {
	panic("unexpected call")
}

func (r *listCaptureGrokVideoJobRepo) ListGrokVideoJobsForUser(ctx context.Context, userID int64, filter GrokVideoJobFilter) ([]*GrokVideoJob, int64, error) {
	r.lastUserID = userID
	r.lastFilter = filter
	return []*GrokVideoJob{}, 0, nil
}

func (r *listCaptureGrokVideoJobRepo) UpdateGrokVideoJobStatus(ctx context.Context, requestID string, params UpdateGrokVideoJobStatusParams) (*GrokVideoJob, error) {
	panic("unexpected call")
}

func TestGrokVideoJobServiceListDoesNotDefaultEmptyStatusToPending(t *testing.T) {
	repo := &listCaptureGrokVideoJobRepo{}
	svc := NewGrokVideoJobService(repo, nil)

	_, _, page, pageSize, err := svc.List(context.Background(), 123, GrokVideoJobsQuery{})
	require.NoError(t, err)
	require.Equal(t, 1, page)
	require.Equal(t, 20, pageSize)
	require.Equal(t, int64(123), repo.lastUserID)
	require.Empty(t, repo.lastFilter.Status)
	require.False(t, repo.lastFilter.ActiveOnly)
}

func TestGrokVideoJobServiceListNormalizesExplicitStatus(t *testing.T) {
	repo := &listCaptureGrokVideoJobRepo{}
	svc := NewGrokVideoJobService(repo, nil)

	_, _, _, _, err := svc.List(context.Background(), 456, GrokVideoJobsQuery{Status: "finished"})
	require.NoError(t, err)
	require.Equal(t, GrokVideoJobStatusCompleted, repo.lastFilter.Status)
}
