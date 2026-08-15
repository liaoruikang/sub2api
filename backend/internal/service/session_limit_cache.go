package service

import (
	"context"
	"time"
)

// SessionLimitCache 管理账号级别的活跃会话跟踪。
// Anthropic 会话限制与 OpenAI OAuth SessionID 控制使用独立键空间。
//
// Key 格式: session_limit:account:{accountID}
// 数据结构: Sorted Set (member=sessionUUID, score=timestamp)
//
// 会话在空闲超时后自动过期，无需手动清理
type SessionLimitCache interface {
	// RegisterSession 注册会话活动
	// - 如果会话已存在，刷新其时间戳并返回 true
	// - 如果会话不存在且活跃会话数 < maxSessions，添加新会话并返回 true
	// - 如果会话不存在且活跃会话数 >= maxSessions，返回 false（拒绝）
	//
	// 参数:
	//   accountID: 账号 ID
	//   sessionUUID: 从 metadata.user_id 中提取的会话 UUID
	//   maxSessions: 最大并发会话数限制
	//   idleTimeout: 会话空闲超时时间
	//
	// 返回:
	//   allowed: true 表示允许（在限制内或会话已存在），false 表示拒绝（超出限制且是新会话）
	//   error: 操作错误
	RegisterSession(ctx context.Context, accountID int64, sessionUUID string, maxSessions int, idleTimeout time.Duration) (allowed bool, err error)

	// RefreshSession 刷新现有会话的时间戳
	// 用于活跃会话保持活动状态
	RefreshSession(ctx context.Context, accountID int64, sessionUUID string, idleTimeout time.Duration) error

	// GetActiveSessionCount 获取当前活跃会话数
	// 返回未过期的会话数量
	GetActiveSessionCount(ctx context.Context, accountID int64) (int, error)

	// GetActiveSessionCountBatch 批量获取多个账号的活跃会话数
	// idleTimeouts: 每个账号的空闲超时时间配置，key 为 accountID；若为 nil 或某账号不在其中，则使用默认超时
	// 返回 map[accountID]count，查询失败的账号不在 map 中
	GetActiveSessionCountBatch(ctx context.Context, accountIDs []int64, idleTimeouts map[int64]time.Duration) (map[int64]int, error)

	// IsSessionActive 检查特定会话是否活跃（未过期）
	IsSessionActive(ctx context.Context, accountID int64, sessionUUID string) (bool, error)

	// RegisterOpenAISessionID atomically registers or refreshes one OpenAI
	// client SessionID in an isolated namespace. It never shares state with the
	// Anthropic session limiter above.
	RegisterOpenAISessionID(ctx context.Context, accountID int64, sessionIDHash string, maxSessions int, idleTimeout time.Duration, rotateWhenFull ...bool) (allowed bool, err error)
	// GetOpenAIStagedSessionAccountID returns the account whose staging window
	// currently retains the client SessionID. Active reservations return zero.
	GetOpenAIStagedSessionAccountID(ctx context.Context, sessionIDHash string) (int64, error)
	GetOpenAIActiveSessionCountBatch(ctx context.Context, accountIDs []int64, idleTimeouts map[int64]time.Duration) (map[int64]int, error)
	ClearOpenAISessions(ctx context.Context, accountID int64) error

	// ========== 5h窗口费用缓存 ==========
	// Key 格式: window_cost:account:{accountID}
	// 用于缓存账号在当前5h窗口内的标准费用，减少数据库聚合查询压力

	// GetWindowCost 获取缓存的窗口费用
	// 返回 (cost, true, nil) 如果缓存命中
	// 返回 (0, false, nil) 如果缓存未命中
	// 返回 (0, false, err) 如果发生错误
	GetWindowCost(ctx context.Context, accountID int64) (cost float64, hit bool, err error)

	// SetWindowCost 设置窗口费用缓存
	SetWindowCost(ctx context.Context, accountID int64, cost float64) error

	// GetWindowCostBatch 批量获取窗口费用缓存
	// 返回 map[accountID]cost，缓存未命中的账号不在 map 中
	GetWindowCostBatch(ctx context.Context, accountIDs []int64) (map[int64]float64, error)
}
