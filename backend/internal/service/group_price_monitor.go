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
	SettingKeyGroupPriceMonitor = "group_price_monitor_config"
	GroupPriceMonitorTitle      = "分组价格调整通知"
)

type GroupPriceMonitorConfig struct {
	Enabled         bool                  `json:"enabled"`
	GroupIDs        []int64               `json:"group_ids,omitempty"`
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
	pendingChanges   []GroupPriceChange
}

// GroupPriceChangeObserver receives every administrator-driven group price change.
// The monitor flushes these changes together at the end of the current polling cycle.
type GroupPriceChangeObserver interface {
	RecordGroupPriceChange(groupID int64, groupName string, oldPrice, newPrice float64)
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
	pending := append([]GroupPriceChange(nil), s.pendingChanges...)
	s.pendingChanges = nil
	s.mu.Unlock()
	if previous == nil {
		return s.publishChanges(ctx, cfg, pending)
	}
	changes := make([]GroupPriceChange, 0)
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
		changes = append(changes, GroupPriceChange{GroupID: id, GroupName: name, Old: oldPrice, New: newPrice})
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
	s.pendingChanges = append(s.pendingChanges, GroupPriceChange{GroupID: groupID, GroupName: groupName, Old: oldPrice, New: newPrice})
}

func (s *GroupPriceMonitorService) publishChanges(ctx context.Context, cfg *GroupPriceMonitorConfig, changes []GroupPriceChange) error {
	changes = filterGroupPriceChanges(changes, cfg.GroupIDs)
	if len(changes) == 0 {
		return nil
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].GroupName < changes[j].GroupName })
	startsAt := time.Now()
	endsAt := startsAt.AddDate(0, 0, cfg.DurationDays)
	announcement := &Announcement{Kind: AnnouncementKindGroupPriceChange, Title: GroupPriceMonitorTitle, Content: renderGroupPriceChanges(changes), Status: cfg.Status, NotifyMode: cfg.NotifyMode, Targeting: cfg.Targeting, StartsAt: &startsAt, EndsAt: &endsAt}
	structured := make([]AnnouncementGroupPriceChange, 0, len(changes))
	for i, change := range changes {
		structured = append(structured, AnnouncementGroupPriceChange{
			GroupID: change.GroupID, GroupName: change.GroupName,
			OldRate: change.Old, NewRate: change.New, Sequence: i,
		})
	}
	return s.announcementRepo.CreateWithGroupPriceChanges(ctx, announcement, structured)
}

func filterGroupPriceChanges(changes []GroupPriceChange, groupIDs []int64) []GroupPriceChange {
	if len(changes) == 0 || len(groupIDs) == 0 {
		return changes
	}
	wanted := make(map[int64]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		wanted[id] = struct{}{}
	}
	filtered := make([]GroupPriceChange, 0, len(changes))
	for _, change := range changes {
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

func renderGroupPriceChanges(changes []GroupPriceChange) string {
	structured := make([]AnnouncementGroupPriceChange, 0, len(changes))
	for i, change := range changes {
		structured = append(structured, AnnouncementGroupPriceChange{
			GroupID: change.GroupID, GroupName: change.GroupName,
			OldRate: change.Old, NewRate: change.New, Sequence: i,
		})
	}
	return renderAnnouncementGroupPriceChanges(structured)
}

func renderAnnouncementGroupPriceChanges(changes []AnnouncementGroupPriceChange) string {
	var b strings.Builder
	b.WriteString("## 分组价格调整\n\n")
	fmt.Fprintf(&b, "> 本次共检测到 **%d** 项倍率调整，最新价格如下。\n\n", len(changes))
	b.WriteString("| 分组 | 调整类型 | 调整前 | 调整后 |\n")
	b.WriteString("| :--- | :---: | ---: | ---: |\n")
	for _, change := range changes {
		if change.NewRate > change.OldRate {
			fmt.Fprintf(&b, "| %s | **▲ 涨价** | `%.2f` | **`%.2f`** |\n", change.GroupName, change.OldRate, change.NewRate)
		} else {
			fmt.Fprintf(&b, "| %s | **▼ 降价** | `%.2f` | **`%.2f`** |\n", change.GroupName, change.OldRate, change.NewRate)
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString("价格调整已生效，实际计费以请求时匹配到的分组倍率为准。")
	return b.String()
}

func defaultGroupPriceMonitorConfig() GroupPriceMonitorConfig {
	return GroupPriceMonitorConfig{IntervalSeconds: 60, Status: AnnouncementStatusActive, NotifyMode: AnnouncementNotifyModePopup}
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
}
