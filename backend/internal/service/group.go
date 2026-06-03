package service

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/robfig/cron/v3"
)

type OpenAIMessagesDispatchModelConfig = domain.OpenAIMessagesDispatchModelConfig
type GroupModelsListConfig = domain.GroupModelsListConfig

var groupLimitedTimeMultiplierCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type Group struct {
	ID                                   int64
	Name                                 string
	Description                          string
	Platform                             string
	RateMultiplier                       float64
	LimitedTimeMultiplierEnabled         bool
	LimitedTimeMultiplierCron            string
	LimitedTimeMultiplierDurationMinutes int
	LimitedTimeMultiplierValue           float64
	IsExclusive                          bool
	Status                               string
	Hydrated                             bool // indicates the group was loaded from a trusted repository source

	SubscriptionType    string
	DailyLimitUSD       *float64
	WeeklyLimitUSD      *float64
	MonthlyLimitUSD     *float64
	DefaultValidityDays int

	// 图片生成计费配置（antigravity 和 gemini 平台使用）
	AllowImageGeneration bool
	ImageRateIndependent bool
	ImageRateMultiplier  float64
	ImagePrice1K         *float64
	ImagePrice2K         *float64
	ImagePrice4K         *float64

	// Claude Code 客户端限制
	ClaudeCodeOnly  bool
	FallbackGroupID *int64
	// 无效请求兜底分组（仅 anthropic 平台使用）
	FallbackGroupIDOnInvalidRequest *int64

	// 模型路由配置
	// key: 模型匹配模式（支持 * 通配符，如 "claude-opus-*"）
	// value: 优先账号 ID 列表
	ModelRouting        map[string][]int64
	ModelRoutingEnabled bool

	// MCP XML 协议注入开关（仅 antigravity 平台使用）
	MCPXMLInject bool

	// 支持的模型系列（仅 antigravity 平台使用）
	// 可选值: claude, gemini_text, gemini_image
	SupportedModelScopes []string

	// 分组排序
	SortOrder int

	// OpenAI Messages 调度配置（仅 openai 平台使用）
	AllowMessagesDispatch       bool
	RequireOAuthOnly            bool // 仅允许非 apikey 类型账号关联（OpenAI/Antigravity/Anthropic/Gemini）
	RequirePrivacySet           bool // 调度时仅允许 privacy 已成功设置的账号（OpenAI/Antigravity/Anthropic/Gemini）
	DefaultMappedModel          string
	MessagesDispatchModelConfig OpenAIMessagesDispatchModelConfig
	ModelsListConfig            GroupModelsListConfig

	// RPMLimit 分组级每分钟请求数上限（0 = 不限制）。
	// 一旦设置即接管该分组用户的限流（覆盖用户级 rpm_limit），可被 user-group rpm_override 进一步覆盖。
	RPMLimit int
	// LimitedTimeRPMLimit 限时倍率窗口内分组级 RPM 上限（0 = 不设置专属限时 RPM，回落普通 RPM 规则）。
	LimitedTimeRPMLimit int
	// LimitedTimeUserConcurrencyLimit 限时倍率窗口内分组内每用户最大并发数（0 = 不设置专属限时并发，回落普通分组并发规则）。
	LimitedTimeUserConcurrencyLimit int
	// UserConcurrencyLimit 分组内每用户最大并发数（0 = 不限制）。
	UserConcurrencyLimit int

	CreatedAt time.Time
	UpdatedAt time.Time

	AccountGroups           []AccountGroup
	AccountCount            int64
	ActiveAccountCount      int64
	RateLimitedAccountCount int64
}

func (g *Group) IsActive() bool {
	return g.Status == StatusActive
}

func (g *Group) IsSubscriptionType() bool {
	return g.SubscriptionType == SubscriptionTypeSubscription
}

func (g *Group) BillingRateMultiplierAt(now time.Time) float64 {
	if g == nil {
		return 1
	}
	return g.BillingRateMultiplierForBaseAt(g.RateMultiplier, now)
}

func (g *Group) BillingRateMultiplierForBaseAt(base float64, now time.Time) float64 {
	if g == nil {
		return base
	}
	if g.IsLimitedTimeMultiplierActiveAt(now) && g.LimitedTimeMultiplierValue < base {
		return g.LimitedTimeMultiplierValue
	}
	return base
}

func (g *Group) EffectiveUserConcurrencyLimitAt(now time.Time) int {
	if g == nil {
		return 0
	}
	if g.IsLimitedTimeMultiplierActiveAt(now) && g.LimitedTimeUserConcurrencyLimit > 0 {
		return g.LimitedTimeUserConcurrencyLimit
	}
	return g.UserConcurrencyLimit
}

func (g *Group) IsLimitedTimeMultiplierActiveAt(now time.Time) bool {
	if g == nil || !g.LimitedTimeMultiplierEnabled || g.SubscriptionType == SubscriptionTypeSubscription {
		return false
	}
	if g.LimitedTimeMultiplierValue <= 0 || g.LimitedTimeMultiplierDurationMinutes <= 0 {
		return false
	}
	cronExpr := strings.TrimSpace(g.LimitedTimeMultiplierCron)
	if cronExpr == "" {
		return false
	}
	schedule, err := groupLimitedTimeMultiplierCronParser.Parse(cronExpr)
	if err != nil {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	duration := time.Duration(g.LimitedTimeMultiplierDurationMinutes) * time.Minute
	start := schedule.Next(now.Add(-duration))
	return !start.After(now) && now.Before(start.Add(duration))
}

func (g *Group) HasDailyLimit() bool {
	return g.DailyLimitUSD != nil && *g.DailyLimitUSD > 0
}

func (g *Group) HasWeeklyLimit() bool {
	return g.WeeklyLimitUSD != nil && *g.WeeklyLimitUSD > 0
}

func (g *Group) HasMonthlyLimit() bool {
	return g.MonthlyLimitUSD != nil && *g.MonthlyLimitUSD > 0
}

// GetImagePrice 根据 image_size 返回对应的图片生成价格
// 如果分组未配置价格，返回 nil（调用方应使用默认值）
func (g *Group) GetImagePrice(imageSize string) *float64 {
	switch imageSize {
	case "1K":
		return g.ImagePrice1K
	case "2K":
		return g.ImagePrice2K
	case "4K":
		return g.ImagePrice4K
	default:
		// 未知尺寸默认按 2K 计费
		return g.ImagePrice2K
	}
}

// IsGroupContextValid reports whether a group from context has the fields required for routing decisions.
func IsGroupContextValid(group *Group) bool {
	if group == nil {
		return false
	}
	if group.ID <= 0 {
		return false
	}
	if !group.Hydrated {
		return false
	}
	if group.Platform == "" || group.Status == "" {
		return false
	}
	return true
}

// GetRoutingAccountIDs 根据请求模型获取路由账号 ID 列表
// 返回匹配的优先账号 ID 列表，如果没有匹配规则则返回 nil
func (g *Group) GetRoutingAccountIDs(requestedModel string) []int64 {
	if !g.ModelRoutingEnabled || len(g.ModelRouting) == 0 || requestedModel == "" {
		return nil
	}

	// 1. 精确匹配优先
	if accountIDs, ok := g.ModelRouting[requestedModel]; ok && len(accountIDs) > 0 {
		return accountIDs
	}

	// 2. 通配符匹配（前缀匹配）
	for pattern, accountIDs := range g.ModelRouting {
		if matchModelPattern(pattern, requestedModel) && len(accountIDs) > 0 {
			return accountIDs
		}
	}

	return nil
}

// matchModelPattern 检查模型是否匹配模式
// 支持 * 通配符，如 "claude-opus-*" 匹配 "claude-opus-4-20250514"
func matchModelPattern(pattern, model string) bool {
	if pattern == model {
		return true
	}

	// 处理 * 通配符（仅支持末尾通配符）
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(model, prefix)
	}

	return false
}
