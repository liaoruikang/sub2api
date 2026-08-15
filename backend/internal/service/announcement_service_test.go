package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item                             *Announcement
	changes                          map[int64][]AnnouncementGroupPriceChange
	createWithGroupPriceChangesCalls int
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) CreateWithGroupPriceChanges(_ context.Context, a *Announcement, changes []AnnouncementGroupPriceChange) error {
	s.item = a
	s.createWithGroupPriceChangesCalls++
	if s.changes == nil {
		s.changes = make(map[int64][]AnnouncementGroupPriceChange)
	}
	s.changes[a.ID] = append([]AnnouncementGroupPriceChange(nil), changes...)
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (*announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) {
	return nil, nil
}

func (*announcementRepoStub) ListActivePage(context.Context, time.Time, int64, int) ([]Announcement, error) {
	return nil, nil
}

func (*announcementRepoStub) ListGroupPriceChanges(context.Context, []int64) (map[int64][]AnnouncementGroupPriceChange, error) {
	return map[int64][]AnnouncementGroupPriceChange{}, nil
}

type announcementVisibilityRepoStub struct {
	announcementRepoStub
}

func (s *announcementVisibilityRepoStub) ListActivePage(_ context.Context, _ time.Time, beforeID int64, _ int) ([]Announcement, error) {
	if beforeID > 0 || s.item == nil {
		return nil, nil
	}
	return []Announcement{*s.item}, nil
}

func (s *announcementVisibilityRepoStub) ListGroupPriceChanges(_ context.Context, announcementIDs []int64) (map[int64][]AnnouncementGroupPriceChange, error) {
	out := make(map[int64][]AnnouncementGroupPriceChange, len(announcementIDs))
	for _, id := range announcementIDs {
		out[id] = append([]AnnouncementGroupPriceChange(nil), s.changes[id]...)
	}
	return out, nil
}

type announcementVisibilityUserRepoStub struct {
	UserRepository
	user *User
}

func (s *announcementVisibilityUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	copy := *s.user
	copy.AllowedGroups = append([]int64(nil), s.user.AllowedGroups...)
	return &copy, nil
}

type announcementVisibilityGroupRepoStub struct {
	GroupRepository
	groups []Group
}

func (s *announcementVisibilityGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return append([]Group(nil), s.groups...), nil
}

type announcementVisibilityTagRepoStub struct {
	UserTagRepository
	groupIDs []int64
	tagIDs   []int64
}

func (s *announcementVisibilityTagRepoStub) GetGroupIDsByUserID(context.Context, int64) ([]int64, error) {
	return append([]int64(nil), s.groupIDs...), nil
}

func (s *announcementVisibilityTagRepoStub) GetByUserID(context.Context, int64) ([]UserTag, error) {
	tags := make([]UserTag, 0, len(s.tagIDs))
	for _, tagID := range s.tagIDs {
		tags = append(tags, UserTag{ID: tagID})
	}
	return tags, nil
}

type announcementVisibilitySubRepoStub struct {
	UserSubscriptionRepository
	subs []UserSubscription
}

func (s *announcementVisibilitySubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return append([]UserSubscription(nil), s.subs...), nil
}

type announcementVisibilityReadRepoStub struct {
	AnnouncementReadRepository
	groupReads map[int64]map[int64]time.Time
}

func (s *announcementVisibilityReadRepoStub) GetReadMapByUser(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}

func (s *announcementVisibilityReadRepoStub) GetGroupPriceReadMapByUser(_ context.Context, _ int64, announcementIDs []int64) (map[int64]map[int64]time.Time, error) {
	out := make(map[int64]map[int64]time.Time, len(announcementIDs))
	for _, announcementID := range announcementIDs {
		out[announcementID] = make(map[int64]time.Time)
		for groupID, readAt := range s.groupReads[announcementID] {
			out[announcementID][groupID] = readAt
		}
	}
	return out, nil
}

func (s *announcementVisibilityReadRepoStub) MarkGroupPriceChangesRead(_ context.Context, announcementID, _ int64, groupIDs []int64, readAt time.Time) error {
	if s.groupReads == nil {
		s.groupReads = make(map[int64]map[int64]time.Time)
	}
	if s.groupReads[announcementID] == nil {
		s.groupReads[announcementID] = make(map[int64]time.Time)
	}
	for _, groupID := range groupIDs {
		if _, exists := s.groupReads[announcementID][groupID]; !exists {
			s.groupReads[announcementID][groupID] = readAt
		}
	}
	return nil
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestGroupPriceAnnouncementUsesCurrentAccessAndNewlyGrantedGroupsBecomeUnread(t *testing.T) {
	now := time.Now()
	repo := &announcementVisibilityRepoStub{announcementRepoStub: announcementRepoStub{
		item: &Announcement{
			ID: 101, Kind: AnnouncementKindGroupPriceChange, Title: GroupPriceMonitorTitle,
			Content: "must not leak", Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup,
			StartsAt: &now,
		},
		changes: map[int64][]AnnouncementGroupPriceChange{101: {
			{AnnouncementID: 101, GroupID: 1, GroupName: "Public", OldRate: 0.10, NewRate: 0.09, Sequence: 0},
			{AnnouncementID: 101, GroupID: 2, GroupName: "Manual", OldRate: 0.20, NewRate: 0.18, Sequence: 1},
			{AnnouncementID: 101, GroupID: 3, GroupName: "Tagged", OldRate: 0.30, NewRate: 0.28, Sequence: 2},
			{AnnouncementID: 101, GroupID: 4, GroupName: "Subscription", OldRate: 0.40, NewRate: 0.38, Sequence: 3},
		}},
	}}
	userRepo := &announcementVisibilityUserRepoStub{user: &User{ID: 7}}
	groupRepo := &announcementVisibilityGroupRepoStub{groups: []Group{
		{ID: 1, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
		{ID: 2, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, IsExclusive: true},
		{ID: 3, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, IsExclusive: true},
		{ID: 4, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription, IsExclusive: true},
	}}
	tagRepo := &announcementVisibilityTagRepoStub{}
	subRepo := &announcementVisibilitySubRepoStub{}
	readRepo := &announcementVisibilityReadRepoStub{}
	svc := NewAnnouncementService(repo, readRepo, userRepo, subRepo, groupRepo, tagRepo)

	items, err := svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Contains(t, items[0].Announcement.Content, "Public")
	require.NotContains(t, items[0].Announcement.Content, "Manual")
	require.NotContains(t, items[0].Announcement.Content, "Tagged")
	require.NotContains(t, items[0].Announcement.Content, "Subscription")
	require.NotContains(t, items[0].Announcement.Content, "must not leak")
	require.Nil(t, items[0].ReadAt)

	require.NoError(t, svc.MarkRead(context.Background(), 7, 101))
	items, err = svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.NotNil(t, items[0].ReadAt)

	userRepo.user.AllowedGroups = []int64{2}
	tagRepo.groupIDs = []int64{3}
	subRepo.subs = []UserSubscription{{UserID: 7, GroupID: 4, Status: SubscriptionStatusActive, ExpiresAt: now.Add(24 * time.Hour)}}
	items, err = svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.Contains(t, items[0].Announcement.Content, "Manual")
	require.Contains(t, items[0].Announcement.Content, "Tagged")
	require.Contains(t, items[0].Announcement.Content, "Subscription")
	require.Nil(t, items[0].ReadAt, "newly visible group changes must make the logical announcement unread")

	require.NoError(t, svc.MarkRead(context.Background(), 7, 101))
	items, err = svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.NotNil(t, items[0].ReadAt)

	userRepo.user.AllowedGroups = nil
	tagRepo.groupIDs = nil
	subRepo.subs = nil
	items, err = svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.Contains(t, items[0].Announcement.Content, "Public")
	require.NotContains(t, items[0].Announcement.Content, "Manual")
	require.NotContains(t, items[0].Announcement.Content, "Tagged")
	require.NotContains(t, items[0].Announcement.Content, "Subscription")
	require.NotNil(t, items[0].ReadAt)
}

func TestGroupPriceAnnouncementHiddenWhenUserHasNoCurrentGroupAccess(t *testing.T) {
	now := time.Now()
	repo := &announcementVisibilityRepoStub{announcementRepoStub: announcementRepoStub{
		item:    &Announcement{ID: 102, Kind: AnnouncementKindGroupPriceChange, Title: GroupPriceMonitorTitle, Status: AnnouncementStatusActive, StartsAt: &now},
		changes: map[int64][]AnnouncementGroupPriceChange{102: {{AnnouncementID: 102, GroupID: 2, GroupName: "Private", OldRate: 1, NewRate: 2}}},
	}}
	svc := NewAnnouncementService(
		repo,
		&announcementVisibilityReadRepoStub{},
		&announcementVisibilityUserRepoStub{user: &User{ID: 7}},
		&announcementVisibilitySubRepoStub{},
		&announcementVisibilityGroupRepoStub{groups: []Group{{ID: 2, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, IsExclusive: true}}},
		&announcementVisibilityTagRepoStub{},
	)

	items, err := svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.Empty(t, items)
	require.ErrorIs(t, svc.MarkRead(context.Background(), 7, 102), ErrAnnouncementNotFound)
}

func TestGroupStatusAnnouncementUsesDeletedAudienceAndTagSnapshots(t *testing.T) {
	now := time.Now()
	repo := &announcementVisibilityRepoStub{announcementRepoStub: announcementRepoStub{
		item: &Announcement{
			ID: 103, Kind: AnnouncementKindGroupPriceChange, Title: GroupStatusMonitorTitle,
			Status: AnnouncementStatusActive, StartsAt: &now,
		},
		changes: map[int64][]AnnouncementGroupPriceChange{103: {
			{AnnouncementID: 103, GroupID: 1, GroupName: "Public deleted", ChangeType: GroupMonitorEventTypeDeleted, OldStatus: StatusActive, NewStatus: groupMonitorDeletedStatus, SubscriptionType: SubscriptionTypeStandard},
			{AnnouncementID: 103, GroupID: 2, GroupName: "Manual deleted", ChangeType: GroupMonitorEventTypeDeleted, OldStatus: StatusActive, NewStatus: groupMonitorDeletedStatus, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard, AccessUserIDs: []int64{7}},
			{AnnouncementID: 103, GroupID: 3, GroupName: "Tagged deleted", ChangeType: GroupMonitorEventTypeDeleted, OldStatus: StatusActive, NewStatus: groupMonitorDeletedStatus, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard, TagIDs: []int64{9}},
			{AnnouncementID: 103, GroupID: 4, GroupName: "Hidden deleted", ChangeType: GroupMonitorEventTypeDeleted, OldStatus: StatusActive, NewStatus: groupMonitorDeletedStatus, IsExclusive: true, SubscriptionType: SubscriptionTypeStandard, AccessUserIDs: []int64{8}},
		}},
	}}
	tagRepo := &announcementVisibilityTagRepoStub{tagIDs: []int64{9}}
	svc := NewAnnouncementService(
		repo,
		&announcementVisibilityReadRepoStub{},
		&announcementVisibilityUserRepoStub{user: &User{ID: 7}},
		&announcementVisibilitySubRepoStub{},
		&announcementVisibilityGroupRepoStub{},
		tagRepo,
	)

	items, err := svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Contains(t, items[0].Announcement.Content, "Public deleted")
	require.Contains(t, items[0].Announcement.Content, "Manual deleted")
	require.Contains(t, items[0].Announcement.Content, "Tagged deleted")
	require.NotContains(t, items[0].Announcement.Content, "Hidden deleted")

	tagRepo.tagIDs = nil
	items, err = svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.NotContains(t, items[0].Announcement.Content, "Tagged deleted")
}

func TestGroupMonitorAnnouncementTitleMatchesVisibleChanges(t *testing.T) {
	now := time.Now()
	repo := &announcementVisibilityRepoStub{announcementRepoStub: announcementRepoStub{
		item: &Announcement{
			ID: 104, Kind: AnnouncementKindGroupPriceChange, Title: GroupPriceStatusMonitorTitle,
			Status: AnnouncementStatusActive, StartsAt: &now,
		},
		changes: map[int64][]AnnouncementGroupPriceChange{104: {
			{AnnouncementID: 104, GroupID: 1, GroupName: "Public price", ChangeType: GroupMonitorChangeTypePrice, OldRate: 0.1, NewRate: 0.09},
			{AnnouncementID: 104, GroupID: 2, GroupName: "Hidden status", ChangeType: GroupMonitorEventTypeStatus, OldStatus: StatusActive, NewStatus: "inactive", IsExclusive: true, SubscriptionType: SubscriptionTypeStandard},
		}},
	}}
	svc := NewAnnouncementService(
		repo,
		&announcementVisibilityReadRepoStub{},
		&announcementVisibilityUserRepoStub{user: &User{ID: 7}},
		&announcementVisibilitySubRepoStub{},
		&announcementVisibilityGroupRepoStub{groups: []Group{
			{ID: 1, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
			{ID: 2, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, IsExclusive: true},
		}},
		&announcementVisibilityTagRepoStub{},
	)

	items, err := svc.ListForUser(context.Background(), 7, false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, GroupPriceMonitorTitle, items[0].Announcement.Title)
	require.Contains(t, items[0].Announcement.Content, "Public price")
	require.NotContains(t, items[0].Announcement.Content, "Hidden status")
}
