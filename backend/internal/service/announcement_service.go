package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type AnnouncementService struct {
	announcementRepo AnnouncementRepository
	readRepo         AnnouncementReadRepository
	userRepo         UserRepository
	userSubRepo      UserSubscriptionRepository
	groupRepo        GroupRepository
	userTagRepo      UserTagRepository
}

func NewAnnouncementService(
	announcementRepo AnnouncementRepository,
	readRepo AnnouncementReadRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	groupRepo GroupRepository,
	userTagRepo UserTagRepository,
) *AnnouncementService {
	return &AnnouncementService{
		announcementRepo: announcementRepo,
		readRepo:         readRepo,
		userRepo:         userRepo,
		userSubRepo:      userSubRepo,
		groupRepo:        groupRepo,
		userTagRepo:      userTagRepo,
	}
}

type CreateAnnouncementInput struct {
	Title      string
	Content    string
	Status     string
	NotifyMode string
	Targeting  AnnouncementTargeting
	StartsAt   *time.Time
	EndsAt     *time.Time
	ActorID    *int64 // 管理员用户ID
}

type UpdateAnnouncementInput struct {
	Title      *string
	Content    *string
	Status     *string
	NotifyMode *string
	Targeting  *AnnouncementTargeting
	StartsAt   **time.Time
	EndsAt     **time.Time
	ActorID    *int64 // 管理员用户ID
}

type UserAnnouncement struct {
	Announcement Announcement
	ReadAt       *time.Time
}

type AnnouncementUserReadStatus struct {
	UserID   int64      `json:"user_id"`
	Email    string     `json:"email"`
	Username string     `json:"username"`
	Balance  float64    `json:"balance"`
	Eligible bool       `json:"eligible"`
	ReadAt   *time.Time `json:"read_at,omitempty"`
}

func (s *AnnouncementService) Create(ctx context.Context, input *CreateAnnouncementInput) (*Announcement, error) {
	if input == nil {
		return nil, ErrAnnouncementNilInput
	}

	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if title == "" || len(title) > 200 {
		return nil, ErrAnnouncementInvalidTitle
	}
	if content == "" {
		return nil, ErrAnnouncementContentRequired
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = AnnouncementStatusDraft
	}
	if !isValidAnnouncementStatus(status) {
		return nil, ErrAnnouncementInvalidStatus
	}

	targeting, err := domain.AnnouncementTargeting(input.Targeting).NormalizeAndValidate()
	if err != nil {
		return nil, err
	}

	notifyMode := strings.TrimSpace(input.NotifyMode)
	if notifyMode == "" {
		notifyMode = AnnouncementNotifyModeSilent
	}
	if !isValidAnnouncementNotifyMode(notifyMode) {
		return nil, ErrAnnouncementInvalidNotifyMode
	}

	if input.StartsAt != nil && input.EndsAt != nil {
		if !input.StartsAt.Before(*input.EndsAt) {
			return nil, ErrAnnouncementInvalidSchedule
		}
	}

	a := &Announcement{
		Kind:       AnnouncementKindManual,
		Title:      title,
		Content:    content,
		Status:     status,
		NotifyMode: notifyMode,
		Targeting:  targeting,
		StartsAt:   input.StartsAt,
		EndsAt:     input.EndsAt,
	}
	if input.ActorID != nil && *input.ActorID > 0 {
		a.CreatedBy = input.ActorID
		a.UpdatedBy = input.ActorID
	}

	if err := s.announcementRepo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return a, nil
}

func (s *AnnouncementService) Update(ctx context.Context, id int64, input *UpdateAnnouncementInput) (*Announcement, error) {
	if input == nil {
		return nil, ErrAnnouncementNilInput
	}

	a, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" || len(title) > 200 {
			return nil, ErrAnnouncementInvalidTitle
		}
		a.Title = title
	}
	if input.Content != nil {
		content := strings.TrimSpace(*input.Content)
		if content == "" {
			return nil, ErrAnnouncementContentRequired
		}
		a.Content = content
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if !isValidAnnouncementStatus(status) {
			return nil, ErrAnnouncementInvalidStatus
		}
		a.Status = status
	}

	if input.NotifyMode != nil {
		notifyMode := strings.TrimSpace(*input.NotifyMode)
		if !isValidAnnouncementNotifyMode(notifyMode) {
			return nil, ErrAnnouncementInvalidNotifyMode
		}
		a.NotifyMode = notifyMode
	}

	if input.Targeting != nil {
		targeting, err := domain.AnnouncementTargeting(*input.Targeting).NormalizeAndValidate()
		if err != nil {
			return nil, err
		}
		a.Targeting = targeting
	}

	if input.StartsAt != nil {
		a.StartsAt = *input.StartsAt
	}
	if input.EndsAt != nil {
		a.EndsAt = *input.EndsAt
	}

	if a.StartsAt != nil && a.EndsAt != nil {
		if !a.StartsAt.Before(*a.EndsAt) {
			return nil, ErrAnnouncementInvalidSchedule
		}
	}

	if input.ActorID != nil && *input.ActorID > 0 {
		a.UpdatedBy = input.ActorID
	}

	if err := s.announcementRepo.Update(ctx, a); err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	return a, nil
}

func (s *AnnouncementService) Delete(ctx context.Context, id int64) error {
	if err := s.announcementRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete announcement: %w", err)
	}
	return nil
}

func (s *AnnouncementService) GetByID(ctx context.Context, id int64) (*Announcement, error) {
	return s.announcementRepo.GetByID(ctx, id)
}

func (s *AnnouncementService) List(ctx context.Context, params pagination.PaginationParams, filters AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return s.announcementRepo.List(ctx, params, filters)
}

func (s *AnnouncementService) ListForUser(ctx context.Context, userID int64, unreadOnly bool) ([]UserAnnouncement, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}
	activeGroupIDs := make(map[int64]struct{}, len(activeSubs))
	for i := range activeSubs {
		activeGroupIDs[activeSubs[i].GroupID] = struct{}{}
	}
	now := time.Now()
	visible := make([]Announcement, 0, 200)
	visiblePriceGroups := make(map[int64][]int64)
	var groupAccess map[int64]struct{}
	var userTagIDs map[int64]struct{}
	userTagsLoaded := false
	beforeID := int64(0)
	for len(visible) < 200 {
		anns, err := s.announcementRepo.ListActivePage(ctx, now, beforeID, 200)
		if err != nil {
			return nil, fmt.Errorf("list active announcements: %w", err)
		}
		if len(anns) == 0 {
			break
		}
		beforeID = anns[len(anns)-1].ID

		priceIDs := make([]int64, 0)
		for i := range anns {
			if anns[i].Kind == AnnouncementKindGroupPriceChange {
				priceIDs = append(priceIDs, anns[i].ID)
			}
		}
		priceChanges, err := s.announcementRepo.ListGroupPriceChanges(ctx, priceIDs)
		if err != nil {
			return nil, fmt.Errorf("list announcement group price changes: %w", err)
		}
		if len(priceIDs) > 0 && groupAccess == nil {
			groupAccess, err = s.resolveCurrentGroupAccess(ctx, user, activeGroupIDs)
			if err != nil {
				return nil, err
			}
		}

		for i := range anns {
			a := anns[i]
			if !a.IsActiveAt(now) || !a.Targeting.Matches(user.Balance, activeGroupIDs) {
				continue
			}
			if a.Kind == AnnouncementKindGroupPriceChange {
				changes := priceChanges[a.ID]
				if containsGroupStatusMonitorChanges(changes) && !userTagsLoaded {
					userTagIDs, err = s.resolveUserTagIDSet(ctx, user.ID)
					if err != nil {
						return nil, err
					}
					userTagsLoaded = true
				}
				allowedChanges := filterVisibleGroupMonitorChanges(changes, groupAccess, user, activeGroupIDs, userTagIDs)
				if len(allowedChanges) == 0 {
					continue
				}
				a.Title = groupMonitorAnnouncementTitle(allowedChanges)
				a.Content = renderAnnouncementGroupPriceChanges(allowedChanges)
				visiblePriceGroups[a.ID] = uniqueGroupIDs(allowedChanges)
			}
			visible = append(visible, a)
			if len(visible) == 200 {
				break
			}
		}
		if len(anns) < 200 {
			break
		}
	}

	if len(visible) == 0 {
		return []UserAnnouncement{}, nil
	}

	ids := make([]int64, 0, len(visible))
	priceIDs := make([]int64, 0, len(visiblePriceGroups))
	for i := range visible {
		ids = append(ids, visible[i].ID)
		if visible[i].Kind == AnnouncementKindGroupPriceChange {
			priceIDs = append(priceIDs, visible[i].ID)
		}
	}
	readMap, err := s.readRepo.GetReadMapByUser(ctx, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("get read map: %w", err)
	}
	priceReadMap := make(map[int64]map[int64]time.Time)
	if len(priceIDs) > 0 {
		priceReadMap, err = s.readRepo.GetGroupPriceReadMapByUser(ctx, userID, priceIDs)
		if err != nil {
			return nil, fmt.Errorf("get group price read map: %w", err)
		}
	}

	out := make([]UserAnnouncement, 0, len(visible))
	for i := range visible {
		a := visible[i]
		readAt, ok := readMap[a.ID]
		if a.Kind == AnnouncementKindGroupPriceChange {
			readAt, ok = allVisibleGroupsReadAt(visiblePriceGroups[a.ID], priceReadMap[a.ID])
		}
		if unreadOnly && ok {
			continue
		}
		var ptr *time.Time
		if ok {
			t := readAt
			ptr = &t
		}
		out = append(out, UserAnnouncement{
			Announcement: a,
			ReadAt:       ptr,
		})
	}

	// 未读优先、同状态按创建时间倒序
	sort.Slice(out, func(i, j int) bool {
		ai, aj := out[i], out[j]
		if (ai.ReadAt == nil) != (aj.ReadAt == nil) {
			return ai.ReadAt == nil
		}
		return ai.Announcement.ID > aj.Announcement.ID
	})

	return out, nil
}

func (s *AnnouncementService) MarkRead(ctx context.Context, userID, announcementID int64) error {
	// 安全：仅允许标记当前用户“可见”的公告
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	a, err := s.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return err
	}

	now := time.Now()
	if !a.IsActiveAt(now) {
		return ErrAnnouncementNotFound
	}

	activeSubs, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("list active subscriptions: %w", err)
	}
	activeGroupIDs := make(map[int64]struct{}, len(activeSubs))
	for i := range activeSubs {
		activeGroupIDs[activeSubs[i].GroupID] = struct{}{}
	}

	if !a.Targeting.Matches(user.Balance, activeGroupIDs) {
		return ErrAnnouncementNotFound
	}
	if a.Kind == AnnouncementKindGroupPriceChange {
		groupAccess, err := s.resolveCurrentGroupAccess(ctx, user, activeGroupIDs)
		if err != nil {
			return err
		}
		changesByAnnouncement, err := s.announcementRepo.ListGroupPriceChanges(ctx, []int64{a.ID})
		if err != nil {
			return fmt.Errorf("list announcement group price changes: %w", err)
		}
		changes := changesByAnnouncement[a.ID]
		var userTagIDs map[int64]struct{}
		if containsGroupStatusMonitorChanges(changes) {
			userTagIDs, err = s.resolveUserTagIDSet(ctx, user.ID)
			if err != nil {
				return err
			}
		}
		visibleChanges := filterVisibleGroupMonitorChanges(changes, groupAccess, user, activeGroupIDs, userTagIDs)
		if len(visibleChanges) == 0 {
			return ErrAnnouncementNotFound
		}
		if err := s.readRepo.MarkGroupPriceChangesRead(ctx, announcementID, userID, uniqueGroupIDs(visibleChanges), now); err != nil {
			return fmt.Errorf("mark group price changes read: %w", err)
		}
		return nil
	}

	if err := s.readRepo.MarkRead(ctx, announcementID, userID, now); err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	return nil
}

func (s *AnnouncementService) resolveCurrentGroupAccess(ctx context.Context, user *User, activeSubscriptionGroupIDs map[int64]struct{}) (map[int64]struct{}, error) {
	if s.groupRepo == nil || s.userTagRepo == nil {
		return nil, fmt.Errorf("announcement group access dependencies are not configured")
	}
	tagDerived, err := s.userTagRepo.GetGroupIDsByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("get tag-derived announcement groups: %w", err)
	}
	manual := make(map[int64]struct{}, len(user.AllowedGroups))
	for _, groupID := range user.AllowedGroups {
		manual[groupID] = struct{}{}
	}
	tagged := make(map[int64]struct{}, len(tagDerived))
	for _, groupID := range tagDerived {
		tagged[groupID] = struct{}{}
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups for announcement access: %w", err)
	}
	allowed := make(map[int64]struct{}, len(groups))
	for i := range groups {
		group := groups[i]
		if group.IsSubscriptionType() {
			if _, ok := activeSubscriptionGroupIDs[group.ID]; ok {
				allowed[group.ID] = struct{}{}
			}
			continue
		}
		if !group.IsExclusive {
			allowed[group.ID] = struct{}{}
			continue
		}
		if _, ok := manual[group.ID]; ok {
			allowed[group.ID] = struct{}{}
			continue
		}
		if _, ok := tagged[group.ID]; ok {
			allowed[group.ID] = struct{}{}
		}
	}
	return allowed, nil
}

func (s *AnnouncementService) resolveUserTagIDSet(ctx context.Context, userID int64) (map[int64]struct{}, error) {
	if s.userTagRepo == nil {
		return nil, fmt.Errorf("announcement user tag repository is not configured")
	}
	tags, err := s.userTagRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get announcement user tags: %w", err)
	}
	result := make(map[int64]struct{}, len(tags))
	for _, tag := range tags {
		result[tag.ID] = struct{}{}
	}
	return result, nil
}

func filterVisibleGroupMonitorChanges(
	changes []AnnouncementGroupPriceChange,
	allowed map[int64]struct{},
	user *User,
	activeSubscriptionGroupIDs map[int64]struct{},
	userTagIDs map[int64]struct{},
) []AnnouncementGroupPriceChange {
	visible := make([]AnnouncementGroupPriceChange, 0, len(changes))
	for _, change := range changes {
		if _, ok := allowed[change.GroupID]; ok {
			visible = append(visible, change)
			continue
		}
		if change.ChangeType == "" || change.ChangeType == GroupMonitorChangeTypePrice {
			continue
		}
		if !change.IsExclusive && change.SubscriptionType != SubscriptionTypeSubscription {
			visible = append(visible, change)
			continue
		}
		if change.ChangeType == GroupMonitorEventTypeDeleted && user != nil && containsGroupMonitorInt64(change.AccessUserIDs, user.ID) {
			visible = append(visible, change)
			continue
		}
		if _, ok := activeSubscriptionGroupIDs[change.GroupID]; ok {
			visible = append(visible, change)
			continue
		}
		if user != nil && containsGroupMonitorInt64(user.AllowedGroups, change.GroupID) {
			visible = append(visible, change)
			continue
		}
		if intersectsInt64Set(change.TagIDs, userTagIDs) {
			visible = append(visible, change)
		}
	}
	return visible
}

func containsGroupStatusMonitorChanges(changes []AnnouncementGroupPriceChange) bool {
	for _, change := range changes {
		if change.ChangeType != "" && change.ChangeType != GroupMonitorChangeTypePrice {
			return true
		}
	}
	return false
}

func containsGroupMonitorInt64(values []int64, wanted int64) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func intersectsInt64Set(values []int64, set map[int64]struct{}) bool {
	for _, value := range values {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}

func uniqueGroupIDs(changes []AnnouncementGroupPriceChange) []int64 {
	seen := make(map[int64]struct{}, len(changes))
	groupIDs := make([]int64, 0, len(changes))
	for _, change := range changes {
		if _, ok := seen[change.GroupID]; ok {
			continue
		}
		seen[change.GroupID] = struct{}{}
		groupIDs = append(groupIDs, change.GroupID)
	}
	return groupIDs
}

func allVisibleGroupsReadAt(groupIDs []int64, readMap map[int64]time.Time) (time.Time, bool) {
	var latest time.Time
	if len(groupIDs) == 0 {
		return latest, false
	}
	for _, groupID := range groupIDs {
		readAt, ok := readMap[groupID]
		if !ok {
			return time.Time{}, false
		}
		if readAt.After(latest) {
			latest = readAt
		}
	}
	return latest, true
}

func (s *AnnouncementService) ListUserReadStatus(
	ctx context.Context,
	announcementID int64,
	params pagination.PaginationParams,
	search string,
) ([]AnnouncementUserReadStatus, *pagination.PaginationResult, error) {
	ann, err := s.announcementRepo.GetByID(ctx, announcementID)
	if err != nil {
		return nil, nil, err
	}
	var priceChanges []AnnouncementGroupPriceChange
	if ann.Kind == AnnouncementKindGroupPriceChange {
		changesByAnnouncement, err := s.announcementRepo.ListGroupPriceChanges(ctx, []int64{announcementID})
		if err != nil {
			return nil, nil, fmt.Errorf("list announcement group price changes: %w", err)
		}
		priceChanges = changesByAnnouncement[announcementID]
	}

	filters := UserListFilters{
		Search: strings.TrimSpace(search),
	}

	users, page, err := s.userRepo.ListWithFilters(ctx, params, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("list users: %w", err)
	}

	userIDs := make([]int64, 0, len(users))
	for i := range users {
		userIDs = append(userIDs, users[i].ID)
	}

	readMap, err := s.readRepo.GetReadMapByUsers(ctx, announcementID, userIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("get read map: %w", err)
	}

	out := make([]AnnouncementUserReadStatus, 0, len(users))
	for i := range users {
		u := users[i]
		subs, err := s.userSubRepo.ListActiveByUserID(ctx, u.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("list active subscriptions: %w", err)
		}
		activeGroupIDs := make(map[int64]struct{}, len(subs))
		for j := range subs {
			activeGroupIDs[subs[j].GroupID] = struct{}{}
		}
		eligible := domain.AnnouncementTargeting(ann.Targeting).Matches(u.Balance, activeGroupIDs)

		readAt, ok := readMap[u.ID]
		if eligible && ann.Kind == AnnouncementKindGroupPriceChange {
			groupAccess, accessErr := s.resolveCurrentGroupAccess(ctx, &u, activeGroupIDs)
			if accessErr != nil {
				return nil, nil, accessErr
			}
			var userTagIDs map[int64]struct{}
			if containsGroupStatusMonitorChanges(priceChanges) {
				userTagIDs, accessErr = s.resolveUserTagIDSet(ctx, u.ID)
				if accessErr != nil {
					return nil, nil, accessErr
				}
			}
			visibleChanges := filterVisibleGroupMonitorChanges(priceChanges, groupAccess, &u, activeGroupIDs, userTagIDs)
			eligible = len(visibleChanges) > 0
			if eligible {
				groupReads, readErr := s.readRepo.GetGroupPriceReadMapByUser(ctx, u.ID, []int64{announcementID})
				if readErr != nil {
					return nil, nil, fmt.Errorf("get group price read map: %w", readErr)
				}
				readAt, ok = allVisibleGroupsReadAt(uniqueGroupIDs(visibleChanges), groupReads[announcementID])
			}
		}
		var ptr *time.Time
		if ok {
			t := readAt
			ptr = &t
		}

		out = append(out, AnnouncementUserReadStatus{
			UserID:   u.ID,
			Email:    u.Email,
			Username: u.Username,
			Balance:  u.Balance,
			Eligible: eligible,
			ReadAt:   ptr,
		})
	}

	return out, page, nil
}

func isValidAnnouncementStatus(status string) bool {
	switch status {
	case AnnouncementStatusDraft, AnnouncementStatusActive, AnnouncementStatusArchived:
		return true
	default:
		return false
	}
}

func isValidAnnouncementNotifyMode(mode string) bool {
	switch mode {
	case AnnouncementNotifyModeSilent, AnnouncementNotifyModePopup:
		return true
	default:
		return false
	}
}
