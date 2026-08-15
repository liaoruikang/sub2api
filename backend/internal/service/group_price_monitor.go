package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	SettingKeyGroupPriceMonitor  = "group_price_monitor_config"
	GroupPriceMonitorTitle       = "分组价格调整通知"
	GroupStatusMonitorTitle      = "分组状态调整通知"
	GroupPriceStatusMonitorTitle = "分组价格与状态调整通知"
	GroupMonitorChangeTypePrice  = "price"
	GroupMonitorChangeTypeStatus = "status"
	GroupMonitorEventTypeCreated = "created"
	GroupMonitorEventTypeStatus  = "status"
	GroupMonitorEventTypeDeleted = "deleted"
	groupMonitorDeletedStatus    = "deleted"
)

type GroupPriceMonitorConfig struct {
	Enabled         bool                  `json:"enabled"`
	GroupIDs        []int64               `json:"group_ids,omitempty"`
	ChangeTypes     []string              `json:"change_types"`
	IntervalSeconds int                   `json:"interval_seconds"`
	Status          string                `json:"status"`
	NotifyMode      string                `json:"notify_mode"`
	DurationDays    int                   `json:"duration_days"`
	StartsAt        *time.Time            `json:"starts_at,omitempty"`
	EndsAt          *time.Time            `json:"ends_at,omitempty"`
	Targeting       AnnouncementTargeting `json:"targeting,omitempty"`
	NextCheckAt     *time.Time            `json:"next_check_at,omitempty"`
	ServerTime      *time.Time            `json:"server_time,omitempty"`
}

type GroupPriceMonitorService struct {
	settingRepo      SettingRepository
	groupRepo        GroupRepository
	announcementRepo AnnouncementRepository
	mu               sync.Mutex
	cancel           context.CancelFunc
	running          bool
	nextCheckAt      time.Time
	reschedule       chan struct{}
	lastPrices       map[int64]float64
	pendingChanges   []GroupMonitorChange
}

// GroupPriceChangeObserver receives administrator-driven group changes and flushes
// the selected event types together at the end of the current monitoring cycle.
type GroupPriceChangeObserver interface {
	RecordGroupPriceChange(groupID int64, groupName string, oldPrice, newPrice float64)
	RecordGroupStatusChange(change GroupMonitorChange)
}

// GroupMonitorAudienceSnapshotter preserves recipients for a group that may be
// disabled or deleted before its status announcement is published.
type GroupMonitorAudienceSnapshotter interface {
	ListGroupAnnouncementAudienceUserIDs(ctx context.Context, groupID int64) ([]int64, error)
}

func NewGroupPriceMonitorService(settingRepo SettingRepository, groupRepo GroupRepository, announcementRepo AnnouncementRepository) *GroupPriceMonitorService {
	return &GroupPriceMonitorService{
		settingRepo: settingRepo, groupRepo: groupRepo, announcementRepo: announcementRepo,
		reschedule: make(chan struct{}, 1),
	}
}

func (s *GroupPriceMonitorService) GetConfig(ctx context.Context) (*GroupPriceMonitorConfig, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyGroupPriceMonitor)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return nil, err
	}
	cfg := defaultGroupPriceMonitorConfig()
	if strings.TrimSpace(value) != "" {
		if err := json.Unmarshal([]byte(value), &cfg); err != nil {
			return nil, fmt.Errorf("decode group price monitor config: %w", err)
		}
	}
	normalizeGroupPriceMonitorConfig(&cfg)
	s.attachRuntimeState(&cfg)
	return &cfg, nil
}

func (s *GroupPriceMonitorService) SetConfig(ctx context.Context, cfg *GroupPriceMonitorConfig) (*GroupPriceMonitorConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("group price monitor config is required")
	}
	normalizeGroupPriceMonitorConfig(cfg)
	if cfg.IntervalSeconds < 5 {
		return nil, fmt.Errorf("interval_seconds must be at least 5")
	}
	if cfg.Enabled && cfg.DurationDays < 1 {
		return nil, fmt.Errorf("duration_days must be at least 1 when enabled")
	}
	if len(cfg.ChangeTypes) == 0 {
		return nil, fmt.Errorf("change_types must contain at least one item")
	}
	for _, changeType := range cfg.ChangeTypes {
		if changeType != GroupMonitorChangeTypePrice && changeType != GroupMonitorChangeTypeStatus {
			return nil, fmt.Errorf("unsupported group monitor change type: %s", changeType)
		}
	}
	if !isValidAnnouncementStatus(cfg.Status) || !isValidAnnouncementNotifyMode(cfg.NotifyMode) {
		return nil, fmt.Errorf("invalid announcement status or notify mode")
	}
	targeting, err := cfg.Targeting.NormalizeAndValidate()
	if err != nil {
		return nil, err
	}
	cfg.Targeting = targeting
	// The schedule belongs to each generated announcement, not to the monitor
	// configuration. Reusing configuration timestamps makes later announcements
	// appear to have started before they were created.
	cfg.StartsAt = nil
	cfg.EndsAt = nil
	persisted := *cfg
	persisted.NextCheckAt = nil
	persisted.ServerTime = nil
	encoded, err := json.Marshal(&persisted)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyGroupPriceMonitor, string(encoded)); err != nil {
		return nil, err
	}
	if cfg.Enabled {
		if err := s.captureBaseline(ctx, cfg); err != nil {
			return nil, fmt.Errorf("capture group price monitor baseline: %w", err)
		}
	} else {
		s.mu.Lock()
		s.lastPrices = nil
		s.pendingChanges = nil
		s.mu.Unlock()
	}
	s.resetNextCheck(cfg.Enabled, time.Duration(cfg.IntervalSeconds)*time.Second)
	s.requestReschedule()
	s.attachRuntimeState(cfg)
	return cfg, nil
}

func (s *GroupPriceMonitorService) Start() {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()
	go s.run(ctx)
}

func (s *GroupPriceMonitorService) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.running = false
	s.nextCheckAt = time.Time{}
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *GroupPriceMonitorService) run(ctx context.Context) {
	for {
		cfg, err := s.GetConfig(ctx)
		if err != nil {
			slog.Warn("group price monitor config load failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				continue
			}
			continue
		}
		interval := time.Duration(cfg.IntervalSeconds) * time.Second
		if interval < 5*time.Second {
			interval = 5 * time.Second
		}
		if cfg.Enabled {
			s.mu.Lock()
			baselineMissing := s.lastPrices == nil
			s.mu.Unlock()
			if baselineMissing {
				if err := s.captureBaseline(ctx, cfg); err != nil {
					slog.Warn("group price monitor baseline capture failed", "error", err)
				}
			}
		}
		s.resetNextCheck(cfg.Enabled, interval)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-s.reschedule:
			timer.Stop()
			continue
		case <-timer.C:
			if cfg.Enabled {
				if err := s.check(ctx, cfg); err != nil {
					slog.Warn("group price monitor check failed", "error", err)
				}
			}
		}
	}
}

func (s *GroupPriceMonitorService) attachRuntimeState(cfg *GroupPriceMonitorConfig) {
	now := time.Now()
	cfg.ServerTime = &now
	cfg.NextCheckAt = nil
	s.mu.Lock()
	if cfg.Enabled && s.running && !s.nextCheckAt.IsZero() {
		next := s.nextCheckAt
		cfg.NextCheckAt = &next
	}
	s.mu.Unlock()
}

func (s *GroupPriceMonitorService) resetNextCheck(enabled bool, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !enabled || !s.running {
		s.nextCheckAt = time.Time{}
		return
	}
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	s.nextCheckAt = time.Now().Add(interval)
}

func (s *GroupPriceMonitorService) requestReschedule() {
	select {
	case s.reschedule <- struct{}{}:
	default:
	}
}

func (s *GroupPriceMonitorService) check(ctx context.Context, cfg *GroupPriceMonitorConfig) error {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return err
	}
	current := groupPriceSnapshot(groups, cfg.GroupIDs)

	s.mu.Lock()
	previous := s.lastPrices
	s.lastPrices = current
	pending := append([]GroupMonitorChange(nil), s.pendingChanges...)
	s.pendingChanges = nil
	s.mu.Unlock()
	if previous == nil {
		return s.publishChanges(ctx, cfg, pending)
	}
	changes := make([]GroupMonitorChange, 0)
	for id, oldPrice := range previous {
		newPrice, ok := current[id]
		if !ok || oldPrice == newPrice {
			continue
		}
		name := fmt.Sprintf("分组 #%d", id)
		for _, group := range groups {
			if group.ID == id {
				name = group.Name
				break
			}
		}
		changes = append(changes, GroupMonitorChange{
			ChangeType: GroupMonitorChangeTypePrice,
			GroupID:    id, GroupName: name, OldRate: oldPrice, NewRate: newPrice,
		})
	}
	changes = append(changes, pending...)
	return s.publishChanges(ctx, cfg, changes)
}

func (s *GroupPriceMonitorService) RecordGroupPriceChange(groupID int64, groupName string, oldPrice, newPrice float64) {
	if oldPrice == newPrice {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastPrices != nil {
		if lastPrice, ok := s.lastPrices[groupID]; ok {
			oldPrice = lastPrice
		}
		s.lastPrices[groupID] = newPrice
	}
	s.pendingChanges = append(s.pendingChanges, GroupMonitorChange{
		ChangeType: GroupMonitorChangeTypePrice,
		GroupID:    groupID, GroupName: groupName, OldRate: oldPrice, NewRate: newPrice,
	})
}

func (s *GroupPriceMonitorService) RecordGroupStatusChange(change GroupMonitorChange) {
	if change.ChangeType != GroupMonitorEventTypeCreated &&
		change.ChangeType != GroupMonitorEventTypeStatus &&
		change.ChangeType != GroupMonitorEventTypeDeleted {
		return
	}
	change.TagIDs = uniqueInt64s(change.TagIDs)
	change.AccessUserIDs = uniqueInt64s(change.AccessUserIDs)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastPrices != nil {
		if change.NewStatus == StatusActive {
			s.lastPrices[change.GroupID] = change.NewRate
		} else {
			delete(s.lastPrices, change.GroupID)
		}
	}
	s.pendingChanges = append(s.pendingChanges, change)
}

func (s *GroupPriceMonitorService) publishChanges(ctx context.Context, cfg *GroupPriceMonitorConfig, changes []GroupMonitorChange) error {
	changes = filterGroupMonitorChanges(changes, cfg.GroupIDs, cfg.ChangeTypes)
	if len(changes) == 0 {
		return nil
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].GroupName < changes[j].GroupName })
	startsAt := time.Now()
	endsAt := startsAt.AddDate(0, 0, cfg.DurationDays)
	structured := groupMonitorChangesToAnnouncementChanges(changes)
	announcement := &Announcement{
		Kind: AnnouncementKindGroupPriceChange, Title: groupMonitorAnnouncementTitle(structured),
		Content: renderAnnouncementGroupPriceChanges(structured), Status: cfg.Status,
		NotifyMode: cfg.NotifyMode, Targeting: cfg.Targeting, StartsAt: &startsAt, EndsAt: &endsAt,
	}
	return s.announcementRepo.CreateWithGroupPriceChanges(ctx, announcement, structured)
}

func filterGroupMonitorChanges(changes []GroupMonitorChange, groupIDs []int64, changeTypes []string) []GroupMonitorChange {
	if len(changes) == 0 || len(groupIDs) == 0 {
		return filterGroupMonitorChangesByType(changes, changeTypes)
	}
	wanted := make(map[int64]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		wanted[id] = struct{}{}
	}
	filtered := make([]GroupMonitorChange, 0, len(changes))
	for _, change := range changes {
		if !groupMonitorConfigIncludesChange(changeTypes, change.ChangeType) {
			continue
		}
		if change.GroupID == 0 {
			filtered = append(filtered, change)
			continue
		}
		if _, ok := wanted[change.GroupID]; ok {
			filtered = append(filtered, change)
		}
	}
	return filtered
}

func filterGroupMonitorChangesByType(changes []GroupMonitorChange, changeTypes []string) []GroupMonitorChange {
	filtered := make([]GroupMonitorChange, 0, len(changes))
	for _, change := range changes {
		if groupMonitorConfigIncludesChange(changeTypes, change.ChangeType) {
			filtered = append(filtered, change)
		}
	}
	return filtered
}

func groupMonitorConfigIncludesChange(changeTypes []string, eventType string) bool {
	wanted := GroupMonitorChangeTypeStatus
	if eventType == "" || eventType == GroupMonitorChangeTypePrice {
		wanted = GroupMonitorChangeTypePrice
	}
	if len(changeTypes) == 0 {
		return wanted == GroupMonitorChangeTypePrice
	}
	for _, changeType := range changeTypes {
		if changeType == wanted {
			return true
		}
	}
	return false
}

func (s *GroupPriceMonitorService) captureBaseline(ctx context.Context, cfg *GroupPriceMonitorConfig) error {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.lastPrices = groupPriceSnapshot(groups, cfg.GroupIDs)
	s.pendingChanges = nil
	s.mu.Unlock()
	return nil
}

func groupPriceSnapshot(groups []Group, groupIDs []int64) map[int64]float64 {
	wanted := make(map[int64]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		wanted[id] = struct{}{}
	}
	snapshot := make(map[int64]float64)
	for _, group := range groups {
		if len(wanted) > 0 {
			if _, ok := wanted[group.ID]; !ok {
				continue
			}
		}
		snapshot[group.ID] = group.RateMultiplier
	}
	return snapshot
}

type GroupPriceChange struct {
	GroupID   int64
	GroupName string
	Old, New  float64
}

type GroupMonitorChange struct {
	ChangeType       string
	GroupID          int64
	GroupName        string
	OldRate          float64
	NewRate          float64
	OldStatus        string
	NewStatus        string
	IsExclusive      bool
	SubscriptionType string
	TagIDs           []int64
	AccessUserIDs    []int64
}

func renderGroupPriceChanges(changes []GroupPriceChange) string {
	structured := make([]AnnouncementGroupPriceChange, 0, len(changes))
	for i, change := range changes {
		structured = append(structured, AnnouncementGroupPriceChange{
			ChangeType: GroupMonitorChangeTypePrice,
			GroupID:    change.GroupID, GroupName: change.GroupName,
			OldRate: change.Old, NewRate: change.New, Sequence: i,
		})
	}
	return renderAnnouncementGroupPriceChanges(structured)
}

func renderAnnouncementGroupPriceChanges(changes []AnnouncementGroupPriceChange) string {
	priceChanges := make([]AnnouncementGroupPriceChange, 0, len(changes))
	statusChanges := make([]AnnouncementGroupPriceChange, 0, len(changes))
	for _, change := range changes {
		if change.ChangeType == "" || change.ChangeType == GroupMonitorChangeTypePrice {
			priceChanges = append(priceChanges, change)
		} else {
			statusChanges = append(statusChanges, change)
		}
	}

	var b strings.Builder
	b.WriteString("## 分组变更汇总\n\n")
	fmt.Fprintf(&b, "> 本周期共检测到 **%d** 项变更", len(changes))
	if len(priceChanges) > 0 && len(statusChanges) > 0 {
		fmt.Fprintf(&b, "，其中价格变化 **%d** 项、状态变化 **%d** 项", len(priceChanges), len(statusChanges))
	}
	b.WriteString("。\n\n")
	if len(priceChanges) > 0 {
		b.WriteString("### 价格变化\n\n")
		b.WriteString("| 分组 | 调整类型 | 调整前 | 调整后 |\n")
		b.WriteString("| :--- | :---: | ---: | ---: |\n")
		for _, change := range priceChanges {
			name := escapeMarkdownTableCell(change.GroupName)
			if change.NewRate > change.OldRate {
				fmt.Fprintf(&b, "| %s | **▲ 涨价** | `%.2f` | **`%.2f`** |\n", name, change.OldRate, change.NewRate)
			} else {
				fmt.Fprintf(&b, "| %s | **▼ 降价** | `%.2f` | **`%.2f`** |\n", name, change.OldRate, change.NewRate)
			}
		}
		b.WriteString("\n")
	}
	if len(statusChanges) > 0 {
		b.WriteString("### 状态变化\n\n")
		b.WriteString("| 分组 | 变更类型 | 变更前 | 变更后 |\n")
		b.WriteString("| :--- | :---: | :---: | :---: |\n")
		for _, change := range statusChanges {
			label := groupStatusChangeLabel(change)
			fmt.Fprintf(&b, "| %s | **%s** | %s | **%s** |\n",
				escapeMarkdownTableCell(change.GroupName), label,
				groupStatusDisplay(change.OldStatus), groupStatusDisplay(change.NewStatus))
		}
		b.WriteString("\n")
	}
	b.WriteString("---\n\n")
	b.WriteString("分组调整已生效；价格以请求时匹配到的分组倍率为准，可用状态以当前分组配置为准。")
	return b.String()
}

func groupMonitorChangesToAnnouncementChanges(changes []GroupMonitorChange) []AnnouncementGroupPriceChange {
	structured := make([]AnnouncementGroupPriceChange, 0, len(changes))
	for i, change := range changes {
		structured = append(structured, AnnouncementGroupPriceChange{
			GroupID: change.GroupID, GroupName: change.GroupName, ChangeType: change.ChangeType,
			OldRate: change.OldRate, NewRate: change.NewRate, OldStatus: change.OldStatus, NewStatus: change.NewStatus,
			IsExclusive: change.IsExclusive, SubscriptionType: change.SubscriptionType,
			TagIDs: uniqueInt64s(change.TagIDs), AccessUserIDs: uniqueInt64s(change.AccessUserIDs), Sequence: i,
		})
	}
	return structured
}

func groupMonitorAnnouncementTitle(changes []AnnouncementGroupPriceChange) string {
	hasPrice := false
	hasStatus := false
	for _, change := range changes {
		if change.ChangeType == "" || change.ChangeType == GroupMonitorChangeTypePrice {
			hasPrice = true
		} else {
			hasStatus = true
		}
	}
	if hasPrice && hasStatus {
		return GroupPriceStatusMonitorTitle
	}
	if hasStatus {
		return GroupStatusMonitorTitle
	}
	return GroupPriceMonitorTitle
}

func groupStatusChangeLabel(change AnnouncementGroupPriceChange) string {
	switch change.ChangeType {
	case GroupMonitorEventTypeCreated:
		return "新增"
	case GroupMonitorEventTypeDeleted:
		return "删除"
	case GroupMonitorEventTypeStatus:
		if change.NewStatus == StatusActive {
			return "启用"
		}
		return "停用"
	default:
		return "状态调整"
	}
}

func groupStatusDisplay(status string) string {
	switch status {
	case StatusActive:
		return "`已启用`"
	case "inactive", StatusDisabled:
		return "`已停用`"
	case groupMonitorDeletedStatus:
		return "`已删除`"
	case "":
		return "—"
	default:
		return "`" + strings.ReplaceAll(status, "`", "") + "`"
	}
}

func escapeMarkdownTableCell(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "|", "\\|"), "\n", " ")
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func defaultGroupPriceMonitorConfig() GroupPriceMonitorConfig {
	return GroupPriceMonitorConfig{ChangeTypes: []string{GroupMonitorChangeTypePrice}, IntervalSeconds: 60, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
}

func normalizeGroupPriceMonitorConfig(cfg *GroupPriceMonitorConfig) {
	if cfg.IntervalSeconds <= 0 {
		cfg.IntervalSeconds = 60
	}
	if cfg.Status == "" {
		cfg.Status = AnnouncementStatusActive
	}
	if cfg.NotifyMode == "" {
		cfg.NotifyMode = AnnouncementNotifyModePopup
	}
	if cfg.DurationDays < 0 {
		cfg.DurationDays = 0
	}
	if cfg.DurationDays == 0 && cfg.StartsAt != nil && cfg.EndsAt != nil && cfg.EndsAt.After(*cfg.StartsAt) {
		cfg.DurationDays = int(cfg.EndsAt.Sub(*cfg.StartsAt) / (24 * time.Hour))
		if cfg.DurationDays < 1 {
			cfg.DurationDays = 1
		}
	}
	if cfg.GroupIDs == nil {
		cfg.GroupIDs = []int64{}
	}
	if cfg.ChangeTypes == nil {
		cfg.ChangeTypes = []string{GroupMonitorChangeTypePrice}
	} else {
		normalized := make([]string, 0, len(cfg.ChangeTypes))
		seen := make(map[string]struct{}, len(cfg.ChangeTypes))
		for _, raw := range cfg.ChangeTypes {
			changeType := strings.ToLower(strings.TrimSpace(raw))
			if changeType == "" {
				continue
			}
			if _, exists := seen[changeType]; exists {
				continue
			}
			seen[changeType] = struct{}{}
			normalized = append(normalized, changeType)
		}
		cfg.ChangeTypes = normalized
	}
}
