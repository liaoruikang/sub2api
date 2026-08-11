//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type userTagServiceTagRepoStub struct {
	UserTagRepository
	getByIDFn  func(context.Context, int64) (*UserTag, error)
	addUsersFn func(context.Context, int64, []int64) ([]int64, error)
	addCalls   [][]int64
}

func (s *userTagServiceTagRepoStub) GetByID(ctx context.Context, id int64) (*UserTag, error) {
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return &UserTag{ID: id}, nil
}

func (s *userTagServiceTagRepoStub) AddUsersToTag(ctx context.Context, tagID int64, userIDs []int64) ([]int64, error) {
	s.addCalls = append(s.addCalls, append([]int64(nil), userIDs...))
	if s.addUsersFn != nil {
		return s.addUsersFn(ctx, tagID, userIDs)
	}
	return append([]int64(nil), userIDs...), nil
}

type userTagServiceUserRepoStub struct {
	UserRepository
	usersByID        map[int64]*User
	getByIDCalls     []int64
	listUsers        []User
	listResult       *pagination.PaginationResult
	listErr          error
	listParams       pagination.PaginationParams
	listFilters      UserListFilters
	listFiltersCalls int
}

func (s *userTagServiceUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	s.getByIDCalls = append(s.getByIDCalls, id)
	user, ok := s.usersByID[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *userTagServiceUserRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	s.listFiltersCalls++
	s.listParams = params
	s.listFilters = filters
	return s.listUsers, s.listResult, s.listErr
}

type userTagAuthCacheStub struct {
	userIDs []int64
}

func (*userTagAuthCacheStub) InvalidateAuthCacheByKey(context.Context, string) {}
func (s *userTagAuthCacheStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}
func (*userTagAuthCacheStub) InvalidateAuthCacheByGroupID(context.Context, int64) {}

func TestNormalizeUserIDs(t *testing.T) {
	t.Run("requires at least one id", func(t *testing.T) {
		ids, err := normalizeUserIDs(nil)
		require.Nil(t, ids)
		require.Equal(t, ErrUserIDsRequired, err)
	})

	t.Run("rejects more than 500 ids before deduplication", func(t *testing.T) {
		ids, err := normalizeUserIDs(make([]int64, 501))
		require.Nil(t, ids)
		require.Equal(t, ErrTooManyUserIDs, err)
	})

	t.Run("rejects non-positive ids", func(t *testing.T) {
		ids, err := normalizeUserIDs([]int64{1, 0, 2})
		require.Nil(t, ids)
		require.Equal(t, ErrUserNotFoundForTags, err)
	})

	t.Run("deduplicates while preserving first occurrence order", func(t *testing.T) {
		ids, err := normalizeUserIDs([]int64{3, 1, 3, 2, 1})
		require.NoError(t, err)
		require.Equal(t, []int64{3, 1, 2}, ids)
	})
}

func TestUserTagServiceListTagUsersPassesPaginationAndFilters(t *testing.T) {
	tagRepo := &userTagServiceTagRepoStub{}
	userRepo := &userTagServiceUserRepoStub{
		listUsers:  []User{{ID: 7, Email: "user@example.com", Status: StatusActive}},
		listResult: &pagination.PaginationResult{Total: 31},
	}
	svc := NewUserTagService(tagRepo, userRepo, nil, nil)

	users, total, err := svc.ListTagUsers(context.Background(), 9, 3, 15, "user", StatusDisabled)

	require.NoError(t, err)
	require.Equal(t, int64(31), total)
	require.Equal(t, int64(7), users[0].ID)
	require.Equal(t, 1, userRepo.listFiltersCalls)
	require.Equal(t, pagination.PaginationParams{Page: 3, PageSize: 15, SortBy: "email", SortOrder: pagination.SortOrderAsc}, userRepo.listParams)
	require.Equal(t, UserListFilters{UserTagID: 9, Search: "user", Status: StatusDisabled}, userRepo.listFilters)
}

func TestUserTagServiceListTagUsersStopsWhenTagDoesNotExist(t *testing.T) {
	tagRepo := &userTagServiceTagRepoStub{
		getByIDFn: func(context.Context, int64) (*UserTag, error) {
			return nil, ErrUserTagNotFound
		},
	}
	userRepo := &userTagServiceUserRepoStub{}
	svc := NewUserTagService(tagRepo, userRepo, nil, nil)

	users, total, err := svc.ListTagUsers(context.Background(), 9, 1, 20, "", "")

	require.Nil(t, users)
	require.Zero(t, total)
	require.Equal(t, ErrUserTagNotFound, err)
	require.Zero(t, userRepo.listFiltersCalls)
}

func TestUserTagServiceAddUsersToTagUsesActualInsertions(t *testing.T) {
	tagRepo := &userTagServiceTagRepoStub{
		addUsersFn: func(_ context.Context, tagID int64, userIDs []int64) ([]int64, error) {
			require.Equal(t, int64(9), tagID)
			require.Equal(t, []int64{3, 1, 2}, userIDs)
			return []int64{1, 2}, nil
		},
	}
	userRepo := &userTagServiceUserRepoStub{usersByID: map[int64]*User{
		1: {ID: 1, Status: StatusActive},
		2: {ID: 2, Status: StatusActive},
		3: {ID: 3, Status: StatusActive},
	}}
	cache := &userTagAuthCacheStub{}
	svc := NewUserTagService(tagRepo, userRepo, nil, cache)

	affected, err := svc.AddUsersToTag(context.Background(), 9, []int64{3, 1, 3, 2})

	require.NoError(t, err)
	require.Equal(t, 2, affected)
	require.Equal(t, []int64{3, 1, 2}, userRepo.getByIDCalls)
	require.Equal(t, [][]int64{{3, 1, 2}}, tagRepo.addCalls)
	require.Equal(t, []int64{1, 2}, cache.userIDs)
}

func TestUserTagServiceAddUsersToTagValidatesAllUsersBeforeWriting(t *testing.T) {
	tagRepo := &userTagServiceTagRepoStub{}
	userRepo := &userTagServiceUserRepoStub{usersByID: map[int64]*User{
		1: {ID: 1, Status: StatusActive},
		2: {ID: 2, Status: StatusDisabled},
		3: {ID: 3, Status: StatusActive},
	}}
	cache := &userTagAuthCacheStub{}
	svc := NewUserTagService(tagRepo, userRepo, nil, cache)

	affected, err := svc.AddUsersToTag(context.Background(), 9, []int64{1, 2, 3})

	require.Zero(t, affected)
	require.Equal(t, ErrUserNotFoundForTags, err)
	require.Empty(t, tagRepo.addCalls)
	require.Empty(t, cache.userIDs)
}
