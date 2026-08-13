package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type groupPriceMonitorSettingStub struct{ value string }

func (s *groupPriceMonitorSettingStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (s *groupPriceMonitorSettingStub) GetValue(context.Context, string) (string, error) {
	return s.value, nil
}
func (s *groupPriceMonitorSettingStub) Set(_ context.Context, _ string, value string) error {
	s.value = value
	return nil
}
func (s *groupPriceMonitorSettingStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (s *groupPriceMonitorSettingStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (s *groupPriceMonitorSettingStub) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}
func (s *groupPriceMonitorSettingStub) Delete(context.Context, string) error { return nil }

type groupPriceMonitorGroupStub struct{ groups []Group }

func (s *groupPriceMonitorGroupStub) Create(context.Context, *Group) error { return nil }
func (s *groupPriceMonitorGroupStub) GetByID(context.Context, int64) (*Group, error) {
	return nil, ErrGroupNotFound
}
func (s *groupPriceMonitorGroupStub) GetByIDLite(context.Context, int64) (*Group, error) {
	return nil, ErrGroupNotFound
}
func (s *groupPriceMonitorGroupStub) Update(context.Context, *Group) error { return nil }
func (s *groupPriceMonitorGroupStub) Delete(context.Context, int64) error  { return nil }
func (s *groupPriceMonitorGroupStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (s *groupPriceMonitorGroupStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *groupPriceMonitorGroupStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *groupPriceMonitorGroupStub) ListActive(context.Context) ([]Group, error) {
	return s.groups, nil
}
func (s *groupPriceMonitorGroupStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return s.groups, nil
}
func (s *groupPriceMonitorGroupStub) ExistsByName(context.Context, string) (bool, error) {
	return false, nil
}
func (s *groupPriceMonitorGroupStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	return 0, 0, nil
}
func (s *groupPriceMonitorGroupStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	return 0, nil
}
func (s *groupPriceMonitorGroupStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}
func (s *groupPriceMonitorGroupStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	return nil
}
func (s *groupPriceMonitorGroupStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	return nil
}

func TestRenderGroupPriceChangesHighlightsDirectionAndValues(t *testing.T) {
	content := renderGroupPriceChanges([]GroupPriceChange{{GroupName: "GPT Pro", Old: 0.08, New: 0.07}, {GroupName: "Grok Heavy", Old: 0.07, New: 0.08}})
	require.Contains(t, content, "降价")
	require.Contains(t, content, "涨价")
	require.Contains(t, content, "> 本次共检测到 **2** 项倍率调整")
	require.Contains(t, content, "| 分组 | 调整类型 | 调整前 | 调整后 |")
	require.Contains(t, content, "**▼ 降价**")
	require.Contains(t, content, "**▲ 涨价**")
	require.Contains(t, content, "| GPT Pro | **▼ 降价** | `0.08` | **`0.07`** |")
	require.Contains(t, content, "| Grok Heavy | **▲ 涨价** | `0.07` | **`0.08`** |")
	require.NotContains(t, content, "**GPT Pro**")
	require.NotContains(t, content, "**Grok Heavy**")
}

func TestGroupPriceMonitorSetConfigCapturesBaselineBeforeFirstChange(t *testing.T) {
	settings := &groupPriceMonitorSettingStub{}
	groups := &groupPriceMonitorGroupStub{groups: []Group{{ID: 1, Name: "GPT Pro", RateMultiplier: 0.08}}}
	announcements := &announcementRepoStub{}
	svc := NewGroupPriceMonitorService(settings, groups, announcements)
	cfg := &GroupPriceMonitorConfig{Enabled: true, DurationDays: 3, IntervalSeconds: 5, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
	_, err := svc.SetConfig(context.Background(), cfg)
	require.NoError(t, err)
	groups.groups[0].RateMultiplier = 0.07
	require.NoError(t, svc.check(context.Background(), cfg))
	require.NotNil(t, announcements.item)
	require.Contains(t, announcements.item.Content, "▼ 降价")
}

func TestGroupPriceMonitorSetConfigDoesNotPersistAnnouncementSchedule(t *testing.T) {
	settings := &groupPriceMonitorSettingStub{}
	groups := &groupPriceMonitorGroupStub{groups: []Group{{ID: 1, Name: "GPT Pro", RateMultiplier: 0.08}}}
	svc := NewGroupPriceMonitorService(settings, groups, &announcementRepoStub{})
	cfg := &GroupPriceMonitorConfig{Enabled: true, DurationDays: 3, IntervalSeconds: 5, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
	saved, err := svc.SetConfig(context.Background(), cfg)
	require.NoError(t, err)
	require.Nil(t, saved.StartsAt)
	require.Nil(t, saved.EndsAt)
	require.Equal(t, 3, saved.DurationDays)
	require.NotNil(t, saved.ServerTime)
	require.NotContains(t, settings.value, "next_check_at")
	require.NotContains(t, settings.value, "server_time")
}

func TestGroupPriceMonitorConfigReportsRuntimeNextCheck(t *testing.T) {
	settings := &groupPriceMonitorSettingStub{}
	groups := &groupPriceMonitorGroupStub{groups: []Group{{ID: 1, Name: "GPT Pro", RateMultiplier: 0.08}}}
	svc := NewGroupPriceMonitorService(settings, groups, &announcementRepoStub{})
	svc.mu.Lock()
	svc.running = true
	svc.nextCheckAt = time.Now().Add(5 * time.Second)
	svc.mu.Unlock()
	t.Cleanup(svc.Stop)

	cfg := &GroupPriceMonitorConfig{Enabled: true, DurationDays: 3, IntervalSeconds: 5, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
	_, err := svc.SetConfig(context.Background(), cfg)
	require.NoError(t, err)

	loaded, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, loaded.ServerTime)
	require.NotNil(t, loaded.NextCheckAt)
	require.True(t, loaded.NextCheckAt.After(*loaded.ServerTime))
	require.LessOrEqual(t, loaded.NextCheckAt.Sub(*loaded.ServerTime), 5*time.Second)
}

func TestGroupPriceMonitorAnnouncementScheduleStartsWhenPublished(t *testing.T) {
	settings := &groupPriceMonitorSettingStub{}
	groups := &groupPriceMonitorGroupStub{groups: []Group{{ID: 1, Name: "GPT Pro", RateMultiplier: 0.08}}}
	announcements := &announcementRepoStub{}
	svc := NewGroupPriceMonitorService(settings, groups, announcements)
	cfg := &GroupPriceMonitorConfig{Enabled: true, DurationDays: 3, IntervalSeconds: 5, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
	_, err := svc.SetConfig(context.Background(), cfg)
	require.NoError(t, err)
	groups.groups[0].RateMultiplier = 0.07
	before := time.Now()
	require.NoError(t, svc.check(context.Background(), cfg))
	after := time.Now()
	require.NotNil(t, announcements.item)
	require.NotNil(t, announcements.item.StartsAt)
	require.NotNil(t, announcements.item.EndsAt)
	require.False(t, announcements.item.StartsAt.Before(before))
	require.False(t, announcements.item.StartsAt.After(after))
	require.Equal(t, announcements.item.StartsAt.AddDate(0, 0, 3), *announcements.item.EndsAt)
}

func TestGroupPriceMonitorRecordsRapidChangesForNextCheck(t *testing.T) {
	settings := &groupPriceMonitorSettingStub{}
	groups := &groupPriceMonitorGroupStub{groups: []Group{{ID: 1, Name: "Grok Heavy", RateMultiplier: 0.13}}}
	announcements := &announcementRepoStub{}
	svc := NewGroupPriceMonitorService(settings, groups, announcements)
	cfg := &GroupPriceMonitorConfig{Enabled: true, DurationDays: 3, IntervalSeconds: 5, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
	_, err := svc.SetConfig(context.Background(), cfg)
	require.NoError(t, err)
	svc.RecordGroupPriceChange(1, "Grok Heavy", 0.13, 0.12)
	svc.RecordGroupPriceChange(1, "Grok Heavy", 0.12, 0.13)
	require.NoError(t, svc.check(context.Background(), cfg))
	require.NotNil(t, announcements.item)
	require.Contains(t, announcements.item.Content, "| Grok Heavy | **▼ 降价** | `0.13` | **`0.12`** |")
	require.Contains(t, announcements.item.Content, "| Grok Heavy | **▲ 涨价** | `0.12` | **`0.13`** |")
}

func TestGroupPriceMonitorFirstCheckOnlyEstablishesBaseline(t *testing.T) {
	settings := &groupPriceMonitorSettingStub{}
	groups := &groupPriceMonitorGroupStub{groups: []Group{{ID: 1, Name: "GPT Pro", RateMultiplier: 0.08}}}
	announcements := &announcementRepoStub{}
	svc := NewGroupPriceMonitorService(settings, groups, announcements)
	cfg := &GroupPriceMonitorConfig{Enabled: true, DurationDays: 3, IntervalSeconds: 5, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
	require.NoError(t, svc.check(context.Background(), cfg))
	require.Nil(t, announcements.item)
	groups.groups[0].RateMultiplier = 0.07
	require.NoError(t, svc.check(context.Background(), cfg))
	require.NotNil(t, announcements.item)
	require.Equal(t, GroupPriceMonitorTitle, announcements.item.Title)
	require.Contains(t, announcements.item.Content, "降价")
}
