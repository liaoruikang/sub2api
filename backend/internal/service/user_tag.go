package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrUserTagNotFound      = infraerrors.NotFound("USER_TAG_NOT_FOUND", "user tag not found")
	ErrUserTagNameExists    = infraerrors.Conflict("USER_TAG_NAME_EXISTS", "user tag name already exists")
	ErrUserTagNameRequired  = infraerrors.BadRequest("USER_TAG_NAME_REQUIRED", "user tag name is required")
	ErrUserTagNameTooLong   = infraerrors.BadRequest("USER_TAG_NAME_TOO_LONG", "user tag name is too long")
	ErrUserNotFoundForTags  = infraerrors.NotFound("USER_NOT_FOUND", "user not found")
	ErrGroupNotFoundForTags = infraerrors.NotFound("GROUP_NOT_FOUND", "group not found")
	ErrGroupTagNotPermitted = infraerrors.BadRequest("GROUP_TAG_NOT_PERMITTED", "only active standard exclusive groups can use user tags")
	ErrUserTagIDInvalid     = infraerrors.BadRequest("USER_TAG_ID_INVALID", "user tag id must be positive")
	ErrUserIDsRequired      = infraerrors.BadRequest("USER_IDS_REQUIRED", "user ids are required")
	ErrTooManyUserIDs       = infraerrors.BadRequest("TOO_MANY_USER_IDS", "no more than 500 user ids can be added at once")
)

const maxUserTagNameLength = 100

type UserTag struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateUserTagInput struct {
	Name string
}

type UpdateUserTagInput struct {
	Name string
}

type UserTagRepository interface {
	Create(ctx context.Context, tag *UserTag) error
	GetByID(ctx context.Context, id int64) (*UserTag, error)
	List(ctx context.Context) ([]UserTag, error)
	Update(ctx context.Context, tag *UserTag) error
	Delete(ctx context.Context, id int64) error
	GetByIDs(ctx context.Context, ids []int64) ([]UserTag, error)
	GetByUserID(ctx context.Context, userID int64) ([]UserTag, error)
	ReplaceUserTags(ctx context.Context, userID int64, tagIDs []int64) error
	GetByGroupID(ctx context.Context, groupID int64) ([]UserTag, error)
	ReplaceGroupTags(ctx context.Context, groupID int64, tagIDs []int64) error
	GetUserIDsByTagID(ctx context.Context, tagID int64) ([]int64, error)
	AddUsersToTag(ctx context.Context, tagID int64, userIDs []int64) ([]int64, error)
	GetGroupIDsByUserID(ctx context.Context, userID int64) ([]int64, error)
}

// UserTagService manages reusable user tags and their user/group assignments.
type UserTagService struct {
	tagRepo              UserTagRepository
	userRepo             UserRepository
	groupRepo            GroupRepository
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewUserTagService(
	tagRepo UserTagRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
) *UserTagService {
	return &UserTagService{
		tagRepo:              tagRepo,
		userRepo:             userRepo,
		groupRepo:            groupRepo,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *UserTagService) Create(ctx context.Context, input CreateUserTagInput) (*UserTag, error) {
	name, err := normalizeUserTagName(input.Name)
	if err != nil {
		return nil, err
	}
	tag := &UserTag{Name: name}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *UserTagService) GetByID(ctx context.Context, id int64) (*UserTag, error) {
	if id <= 0 {
		return nil, ErrUserTagIDInvalid
	}
	return s.tagRepo.GetByID(ctx, id)
}

func (s *UserTagService) List(ctx context.Context) ([]UserTag, error) {
	return s.tagRepo.List(ctx)
}

func (s *UserTagService) Update(ctx context.Context, id int64, input UpdateUserTagInput) (*UserTag, error) {
	if id <= 0 {
		return nil, ErrUserTagIDInvalid
	}
	name, err := normalizeUserTagName(input.Name)
	if err != nil {
		return nil, err
	}
	tag, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	tag.Name = name
	if err := s.tagRepo.Update(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

func (s *UserTagService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrUserTagIDInvalid
	}
	if _, err := s.tagRepo.GetByID(ctx, id); err != nil {
		return err
	}
	userIDs, err := s.tagRepo.GetUserIDsByTagID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.tagRepo.Delete(ctx, id); err != nil {
		return err
	}
	if s.authCacheInvalidator != nil {
		for _, userID := range userIDs {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
	}
	return nil
}

func (s *UserTagService) GetUserTags(ctx context.Context, userID int64) ([]UserTag, error) {
	if err := s.ensureActiveUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.tagRepo.GetByUserID(ctx, userID)
}

func (s *UserTagService) ListTagUsers(ctx context.Context, tagID int64, page, pageSize int, search, status string) ([]User, int64, error) {
	if err := s.ensureTag(ctx, tagID); err != nil {
		return nil, 0, err
	}
	users, result, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "email",
		SortOrder: pagination.SortOrderAsc,
	}, UserListFilters{
		UserTagID: tagID,
		Search:    search,
		Status:    status,
	})
	if err != nil {
		return nil, 0, err
	}
	if result == nil {
		return users, 0, nil
	}
	return users, result.Total, nil
}

func (s *UserTagService) AddUsersToTag(ctx context.Context, tagID int64, userIDs []int64) (int, error) {
	if err := s.ensureTag(ctx, tagID); err != nil {
		return 0, err
	}
	normalized, err := normalizeUserIDs(userIDs)
	if err != nil {
		return 0, err
	}
	for _, userID := range normalized {
		if err := s.ensureActiveUser(ctx, userID); err != nil {
			return 0, err
		}
	}
	addedUserIDs, err := s.tagRepo.AddUsersToTag(ctx, tagID, normalized)
	if err != nil {
		return 0, err
	}
	if s.authCacheInvalidator != nil {
		for _, userID := range addedUserIDs {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
	}
	return len(addedUserIDs), nil
}

func (s *UserTagService) ReplaceUserTags(ctx context.Context, userID int64, tagIDs []int64) ([]UserTag, error) {
	if err := s.ensureActiveUser(ctx, userID); err != nil {
		return nil, err
	}
	normalized, err := normalizeUserTagIDs(tagIDs)
	if err != nil {
		return nil, err
	}
	if _, err := s.tagRepo.GetByIDs(ctx, normalized); err != nil {
		return nil, err
	}
	if err := s.tagRepo.ReplaceUserTags(ctx, userID, normalized); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	return s.tagRepo.GetByUserID(ctx, userID)
}

func (s *UserTagService) GetGroupTags(ctx context.Context, groupID int64) ([]UserTag, error) {
	if err := s.ensureTaggableGroup(ctx, groupID); err != nil {
		return nil, err
	}
	return s.tagRepo.GetByGroupID(ctx, groupID)
}

func (s *UserTagService) ReplaceGroupTags(ctx context.Context, groupID int64, tagIDs []int64) ([]UserTag, error) {
	if err := s.ensureTaggableGroup(ctx, groupID); err != nil {
		return nil, err
	}
	normalized, err := normalizeUserTagIDs(tagIDs)
	if err != nil {
		return nil, err
	}
	if _, err := s.tagRepo.GetByIDs(ctx, normalized); err != nil {
		return nil, err
	}
	previousTags, err := s.tagRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if err := s.tagRepo.ReplaceGroupTags(ctx, groupID, normalized); err != nil {
		return nil, err
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
		userIDs := make(map[int64]struct{})
		for _, tag := range previousTags {
			s.collectTagUserIDs(ctx, tag.ID, userIDs)
		}
		for _, tagID := range normalized {
			s.collectTagUserIDs(ctx, tagID, userIDs)
		}
		for userID := range userIDs {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
	}
	return s.tagRepo.GetByGroupID(ctx, groupID)
}

func (s *UserTagService) collectTagUserIDs(ctx context.Context, tagID int64, userIDs map[int64]struct{}) {
	ids, err := s.tagRepo.GetUserIDsByTagID(ctx, tagID)
	if err != nil {
		return
	}
	for _, userID := range ids {
		userIDs[userID] = struct{}{}
	}
}

func (s *UserTagService) ensureTag(ctx context.Context, tagID int64) error {
	if tagID <= 0 {
		return ErrUserTagIDInvalid
	}
	_, err := s.tagRepo.GetByID(ctx, tagID)
	return err
}

func (s *UserTagService) ensureActiveUser(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrUserNotFoundForTags
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFoundForTags.WithCause(err)
	}
	if !user.IsActive() {
		return ErrUserNotFoundForTags
	}
	return nil
}

func (s *UserTagService) ensureTaggableGroup(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return ErrGroupNotFoundForTags
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return ErrGroupNotFoundForTags.WithCause(err)
	}
	if !group.IsActive() || !group.IsExclusive || group.IsSubscriptionType() {
		return ErrGroupTagNotPermitted
	}
	return nil
}

func normalizeUserTagName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrUserTagNameRequired
	}
	if len([]rune(name)) > maxUserTagNameLength {
		return "", ErrUserTagNameTooLong
	}
	return name, nil
}

func normalizeUserTagIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return []int64{}, nil
	}
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrUserTagIDInvalid
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

func normalizeUserIDs(ids []int64) ([]int64, error) {
	if len(ids) == 0 {
		return nil, ErrUserIDsRequired
	}
	if len(ids) > 500 {
		return nil, ErrTooManyUserIDs
	}
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrUserNotFoundForTags
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}
