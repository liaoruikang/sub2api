package repository

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 会话限制缓存常量定义
//
// 设计说明：
// 使用 Redis 有序集合（Sorted Set）跟踪每个账号的活跃会话：
// - Key: session_limit:account:{accountID}
// - Member: sessionUUID（从 metadata.user_id 中提取）
// - Score: Unix 时间戳（会话最后活跃时间）
//
// 通过 ZREMRANGEBYSCORE 自动清理过期会话，无需手动管理 TTL
const (
	// 会话限制键前缀
	// 格式: session_limit:account:{accountID}
	sessionLimitKeyPrefix = "session_limit:account:"

	// OpenAI SessionID 控制使用独立命名空间，避免与 Anthropic 会话限制互相污染。
	openAISessionLimitKeyPrefix     = "openai_session_limit:account:"
	openAISessionStagingKeyPrefix   = "openai_session_limit:staged:"
	openAISessionOwnerKeyPrefix     = "openai_session_limit:owner:"
	openAISessionPreferredKeyPrefix = "openai_session_limit:preferred:"

	// 窗口费用缓存键前缀
	// 格式: window_cost:account:{accountID}
	windowCostKeyPrefix = "window_cost:account:"

	// 窗口费用缓存 TTL（30秒）
	windowCostCacheTTL = 30 * time.Second
)

var (
	// registerSessionScript 注册会话活动
	// 使用 Redis TIME 命令获取服务器时间，避免多实例时钟不同步
	// KEYS[1] = session_limit:account:{accountID}
	// ARGV[1] = maxSessions
	// ARGV[2] = idleTimeout（秒）
	// ARGV[3] = sessionUUID
	// 返回: 1 = 允许, 0 = 拒绝
	registerSessionScript = redis.NewScript(`
		-- Redis 3.2-4.x compat: opt into effects replication so redis.call('TIME')
		-- replicates correctly. No-op on Redis 5.0+ (effects replication is default).
		redis.replicate_commands()
		local key = KEYS[1]
		local maxSessions = tonumber(ARGV[1])
		local idleTimeout = tonumber(ARGV[2])
		local sessionUUID = ARGV[3]

		-- 使用 Redis 服务器时间，确保多实例时钟一致
		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - idleTimeout

		-- 清理过期会话
		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)

		-- 检查会话是否已存在（支持刷新时间戳）
		local exists = redis.call('ZSCORE', key, sessionUUID)
		if exists ~= false then
			-- 会话已存在，刷新时间戳
			redis.call('ZADD', key, now, sessionUUID)
			redis.call('EXPIRE', key, idleTimeout + 60)
			return 1
		end

		-- 检查是否达到会话数量上限
		local count = redis.call('ZCARD', key)
		if count < maxSessions then
			-- 未达上限，添加新会话
			redis.call('ZADD', key, now, sessionUUID)
			redis.call('EXPIRE', key, idleTimeout + 60)
			return 1
		end

		-- 达到上限，拒绝新会话
		return 0
		`)

	// registerOpenAISessionScript keeps one SessionID in exactly one account.
	// Expired or rotated active slots are retained in a per-account staging set
	// for one additional idle window so returning sessions can prefer their last owner.
	registerOpenAISessionScript = redis.NewScript(`
		redis.replicate_commands()
		local targetKey = KEYS[1]
		local ownerKey = KEYS[2]
		local targetStagingKey = KEYS[3]
		local preferredKey = KEYS[4]
		local maxSessions = tonumber(ARGV[1])
		local idleTimeout = tonumber(ARGV[2])
		local sessionID = ARGV[3]
		local targetAccountID = ARGV[4]
		local accountKeyPrefix = ARGV[5]
		local ownerKeyPrefix = ARGV[6]
		local stagingKeyPrefix = ARGV[7]
		local preferredKeyPrefix = ARGV[8]
		local rotateWhenFull = ARGV[9] == '1'

		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - idleTimeout
		local activeKeyTTL = (idleTimeout * 2) + 60
		local stagingKeyTTL = idleTimeout + 60
		local ownerTTL = idleTimeout * 2
		local ownershipCleanupBuffer = 120

		local expiredSessions = redis.call('ZRANGEBYSCORE', targetKey, '-inf', expireBefore, 'WITHSCORES')
		for index = 1, #expiredSessions, 2 do
			local expiredSessionID = expiredSessions[index]
			local lastActiveAt = tonumber(expiredSessions[index + 1])
			local stagedAt = lastActiveAt + idleTimeout
			local expiredOwnerKey = ownerKeyPrefix .. expiredSessionID
			local expiredPreferredKey = preferredKeyPrefix .. expiredSessionID
			if redis.call('GET', expiredOwnerKey) == targetAccountID then
				local remainingOwnerTTL = math.ceil((stagedAt + idleTimeout) - now)
				if remainingOwnerTTL > 0 then
					redis.call('ZADD', targetStagingKey, stagedAt, expiredSessionID)
					redis.call('SET', expiredOwnerKey, targetAccountID, 'EX', remainingOwnerTTL + idleTimeout + ownershipCleanupBuffer)
					local reservationValue = targetAccountID .. ':' .. tostring(stagedAt) .. ':' .. tostring(stagedAt + idleTimeout)
					redis.call('SET', expiredPreferredKey, reservationValue, 'EX', remainingOwnerTTL)
				else
					redis.call('ZREM', targetStagingKey, expiredSessionID)
					redis.call('DEL', expiredOwnerKey)
					redis.call('DEL', expiredPreferredKey)
				end
			elseif string.match(redis.call('GET', expiredPreferredKey) or '', '^(%d+)') == targetAccountID then
				redis.call('DEL', expiredPreferredKey)
			end
		end
		if #expiredSessions > 0 then
			redis.call('ZREMRANGEBYSCORE', targetKey, '-inf', expireBefore)
		end

		local expiredStagedSessions = redis.call('ZRANGEBYSCORE', targetStagingKey, '-inf', expireBefore)
		for _, expiredSessionID in ipairs(expiredStagedSessions) do
			local expiredOwnerKey = ownerKeyPrefix .. expiredSessionID
			if redis.call('GET', expiredOwnerKey) == targetAccountID then
				redis.call('DEL', expiredOwnerKey)
				redis.call('DEL', preferredKeyPrefix .. expiredSessionID)
			elseif string.match(redis.call('GET', preferredKeyPrefix .. expiredSessionID) or '', '^(%d+)') == targetAccountID then
				redis.call('DEL', preferredKeyPrefix .. expiredSessionID)
			end
		end
		if #expiredStagedSessions > 0 then
			redis.call('ZREMRANGEBYSCORE', targetStagingKey, '-inf', expireBefore)
		end
		if redis.call('ZCARD', targetStagingKey) > 0 then
			redis.call('EXPIRE', targetStagingKey, stagingKeyTTL)
		end

		local currentOwner = redis.call('GET', ownerKey)
		local existsInTarget = redis.call('ZSCORE', targetKey, sessionID)
		if existsInTarget ~= false then
			if currentOwner ~= false and currentOwner ~= targetAccountID then
				local previousAccountID = tonumber(currentOwner)
				if previousAccountID ~= nil then
					redis.call('ZREM', accountKeyPrefix .. currentOwner, sessionID)
					redis.call('ZREM', stagingKeyPrefix .. currentOwner, sessionID)
				end
			end
			redis.call('ZREM', targetStagingKey, sessionID)
			redis.call('ZADD', targetKey, now, sessionID)
			redis.call('EXPIRE', targetKey, activeKeyTTL)
			redis.call('SET', ownerKey, targetAccountID, 'EX', ownerTTL + idleTimeout + ownershipCleanupBuffer)
			local reservationValue = targetAccountID .. ':' .. tostring(now + idleTimeout) .. ':' .. tostring(now + ownerTTL)
			redis.call('SET', preferredKey, reservationValue, 'EX', ownerTTL)
			return 1
		end

		local stagedInTarget = redis.call('ZSCORE', targetStagingKey, sessionID)
		local activeCount = redis.call('ZCARD', targetKey)
		if activeCount >= maxSessions and rotateWhenFull then
			-- Keep the configured hard cap even when an administrator lowered it
			-- below the current membership count. Oldest activity is evicted first.
			local slotsToEvict = activeCount - maxSessions + 1
			local evictedSessions = redis.call('ZRANGE', targetKey, 0, slotsToEvict - 1)
			for _, evictedSessionID in ipairs(evictedSessions) do
				local evictedOwnerKey = ownerKeyPrefix .. evictedSessionID
				local evictedPreferredKey = preferredKeyPrefix .. evictedSessionID
				local evictedPreviousOwner = redis.call('GET', evictedOwnerKey)
				if evictedPreviousOwner ~= false and evictedPreviousOwner ~= targetAccountID then
					local previousAccountID = tonumber(evictedPreviousOwner)
					if previousAccountID ~= nil then
						redis.call('ZREM', accountKeyPrefix .. evictedPreviousOwner, evictedSessionID)
						redis.call('ZREM', stagingKeyPrefix .. evictedPreviousOwner, evictedSessionID)
					end
				end
				redis.call('ZREM', targetKey, evictedSessionID)
				redis.call('ZADD', targetStagingKey, now, evictedSessionID)
				redis.call('SET', evictedOwnerKey, targetAccountID, 'EX', ownerTTL + ownershipCleanupBuffer)
				local evictedReservationValue = targetAccountID .. ':' .. tostring(now) .. ':' .. tostring(now + idleTimeout)
				redis.call('SET', evictedPreferredKey, evictedReservationValue, 'EX', idleTimeout)
			end
			redis.call('EXPIRE', targetStagingKey, stagingKeyTTL)
		elseif activeCount >= maxSessions then
			-- A returning staged session loses its reservation when its previous
			-- account has no free slot, allowing the caller to select another one.
			if stagedInTarget ~= false or currentOwner == targetAccountID then
				redis.call('ZREM', targetStagingKey, sessionID)
				if redis.call('GET', ownerKey) == targetAccountID then
					redis.call('DEL', ownerKey)
				end
				if string.match(redis.call('GET', preferredKey) or '', '^(%d+)') == targetAccountID then
					redis.call('DEL', preferredKey)
				end
			end
			return 0
		end

		if currentOwner ~= false and currentOwner ~= targetAccountID then
			local previousAccountID = tonumber(currentOwner)
			if previousAccountID ~= nil then
				redis.call('ZREM', accountKeyPrefix .. currentOwner, sessionID)
				redis.call('ZREM', stagingKeyPrefix .. currentOwner, sessionID)
			end
		end
		redis.call('ZREM', targetStagingKey, sessionID)
		redis.call('ZADD', targetKey, now, sessionID)
		redis.call('EXPIRE', targetKey, activeKeyTTL)
		redis.call('SET', ownerKey, targetAccountID, 'EX', ownerTTL + idleTimeout + ownershipCleanupBuffer)
		local reservationValue = targetAccountID .. ':' .. tostring(now + idleTimeout) .. ':' .. tostring(now + ownerTTL)
		redis.call('SET', preferredKey, reservationValue, 'EX', ownerTTL)
		return 1
	`)

	getOpenAIActiveSessionCountScript = redis.NewScript(`
		redis.replicate_commands()
		local accountKey = KEYS[1]
		local stagingKey = KEYS[2]
		local idleTimeout = tonumber(ARGV[1])
		local accountID = ARGV[2]
		local ownerKeyPrefix = ARGV[3]
		local preferredKeyPrefix = ARGV[4]
		local ownershipCleanupBuffer = 120

		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - idleTimeout
		local expiredSessions = redis.call('ZRANGEBYSCORE', accountKey, '-inf', expireBefore, 'WITHSCORES')
		for index = 1, #expiredSessions, 2 do
			local sessionID = expiredSessions[index]
			local lastActiveAt = tonumber(expiredSessions[index + 1])
			local stagedAt = lastActiveAt + idleTimeout
			local ownerKey = ownerKeyPrefix .. sessionID
			local preferredKey = preferredKeyPrefix .. sessionID
			if redis.call('GET', ownerKey) == accountID then
				local remainingOwnerTTL = math.ceil((stagedAt + idleTimeout) - now)
				if remainingOwnerTTL > 0 then
					redis.call('ZADD', stagingKey, stagedAt, sessionID)
					redis.call('SET', ownerKey, accountID, 'EX', remainingOwnerTTL + idleTimeout + ownershipCleanupBuffer)
					local reservationValue = accountID .. ':' .. tostring(stagedAt) .. ':' .. tostring(stagedAt + idleTimeout)
					redis.call('SET', preferredKey, reservationValue, 'EX', remainingOwnerTTL)
				else
					redis.call('ZREM', stagingKey, sessionID)
					redis.call('DEL', ownerKey)
					redis.call('DEL', preferredKey)
				end
			elseif string.match(redis.call('GET', preferredKey) or '', '^(%d+)') == accountID then
				redis.call('DEL', preferredKey)
			end
		end
		if #expiredSessions > 0 then
			redis.call('ZREMRANGEBYSCORE', accountKey, '-inf', expireBefore)
		end
		local expiredStagedSessions = redis.call('ZRANGEBYSCORE', stagingKey, '-inf', expireBefore)
		for _, sessionID in ipairs(expiredStagedSessions) do
			local ownerKey = ownerKeyPrefix .. sessionID
			if redis.call('GET', ownerKey) == accountID then
				redis.call('DEL', ownerKey)
				redis.call('DEL', preferredKeyPrefix .. sessionID)
			elseif string.match(redis.call('GET', preferredKeyPrefix .. sessionID) or '', '^(%d+)') == accountID then
				redis.call('DEL', preferredKeyPrefix .. sessionID)
			end
		end
		if #expiredStagedSessions > 0 then
			redis.call('ZREMRANGEBYSCORE', stagingKey, '-inf', expireBefore)
		end
		if redis.call('ZCARD', stagingKey) > 0 then
			redis.call('EXPIRE', stagingKey, idleTimeout + 60)
		end
		return redis.call('ZCARD', accountKey)
	`)

	getOpenAIStagedSessionAccountScript = redis.NewScript(`
		redis.replicate_commands()
		local reservationKey = KEYS[1]
		local reservation = redis.call('GET', reservationKey)
		if reservation == false then
			return 0
		end
		local accountID, activeUntil, stagedUntil = string.match(reservation, '^(%d+):(%d+):(%d+)$')
		if accountID == nil then
			return 0
		end
		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		activeUntil = tonumber(activeUntil)
		stagedUntil = tonumber(stagedUntil)
		if now >= stagedUntil then
			redis.call('DEL', reservationKey)
			return 0
		end
		if now >= activeUntil then
			return tonumber(accountID)
		end
		return 0
	`)

	clearOpenAISessionsScript = redis.NewScript(`
		redis.replicate_commands()
		local accountKey = KEYS[1]
		local stagingKey = KEYS[2]
		local accountID = ARGV[1]
		local ownerKeyPrefix = ARGV[2]
		local preferredKeyPrefix = ARGV[3]
		local sessions = redis.call('ZRANGE', accountKey, 0, -1)
		local stagedSessions = redis.call('ZRANGE', stagingKey, 0, -1)
		for _, sessionID in ipairs(stagedSessions) do
			table.insert(sessions, sessionID)
		end
		for _, sessionID in ipairs(sessions) do
			local ownerKey = ownerKeyPrefix .. sessionID
			if redis.call('GET', ownerKey) == accountID then
				redis.call('DEL', ownerKey)
				redis.call('DEL', preferredKeyPrefix .. sessionID)
			elseif string.match(redis.call('GET', preferredKeyPrefix .. sessionID) or '', '^(%d+)') == accountID then
				redis.call('DEL', preferredKeyPrefix .. sessionID)
			end
		end
		return redis.call('DEL', accountKey, stagingKey)
	`)

	// refreshSessionScript 刷新会话时间戳
	// KEYS[1] = session_limit:account:{accountID}
	// ARGV[1] = idleTimeout（秒）
	// ARGV[2] = sessionUUID
	refreshSessionScript = redis.NewScript(`
		-- Redis 3.2-4.x compat: opt into effects replication so redis.call('TIME')
		-- replicates correctly. No-op on Redis 5.0+ (effects replication is default).
		redis.replicate_commands()
		local key = KEYS[1]
		local idleTimeout = tonumber(ARGV[1])
		local sessionUUID = ARGV[2]

		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])

		-- 检查会话是否存在
		local exists = redis.call('ZSCORE', key, sessionUUID)
		if exists ~= false then
			redis.call('ZADD', key, now, sessionUUID)
			redis.call('EXPIRE', key, idleTimeout + 60)
		end
		return 1
	`)

	// getActiveSessionCountScript 获取活跃会话数
	// KEYS[1] = session_limit:account:{accountID}
	// ARGV[1] = idleTimeout（秒）
	getActiveSessionCountScript = redis.NewScript(`
		-- Redis 3.2-4.x compat: opt into effects replication so redis.call('TIME')
		-- replicates correctly. No-op on Redis 5.0+ (effects replication is default).
		redis.replicate_commands()
		local key = KEYS[1]
		local idleTimeout = tonumber(ARGV[1])

		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - idleTimeout

		-- 清理过期会话
		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)

		return redis.call('ZCARD', key)
	`)

	// isSessionActiveScript 检查会话是否活跃
	// KEYS[1] = session_limit:account:{accountID}
	// ARGV[1] = idleTimeout（秒）
	// ARGV[2] = sessionUUID
	isSessionActiveScript = redis.NewScript(`
		-- Redis 3.2-4.x compat: opt into effects replication so redis.call('TIME')
		-- replicates correctly. No-op on Redis 5.0+ (effects replication is default).
		redis.replicate_commands()
		local key = KEYS[1]
		local idleTimeout = tonumber(ARGV[1])
		local sessionUUID = ARGV[2]

		local timeResult = redis.call('TIME')
		local now = tonumber(timeResult[1])
		local expireBefore = now - idleTimeout

		-- 获取会话的时间戳
		local score = redis.call('ZSCORE', key, sessionUUID)
		if score == false then
			return 0
		end

		-- 检查是否过期
		if tonumber(score) <= expireBefore then
			return 0
		end

		return 1
	`)
)

type sessionLimitCache struct {
	rdb                *redis.Client
	defaultIdleTimeout time.Duration // 默认空闲超时（用于 GetActiveSessionCount）
}

// NewSessionLimitCache 创建会话限制缓存
// defaultIdleTimeoutMinutes: 默认空闲超时时间（分钟），用于无参数查询
func NewSessionLimitCache(rdb *redis.Client, defaultIdleTimeoutMinutes int) service.SessionLimitCache {
	if defaultIdleTimeoutMinutes <= 0 {
		defaultIdleTimeoutMinutes = 5 // 默认 5 分钟
	}

	// 预加载 Lua 脚本到 Redis，避免 Pipeline 中出现 NOSCRIPT 错误
	ctx := context.Background()
	scripts := []*redis.Script{
		registerSessionScript,
		registerOpenAISessionScript,
		refreshSessionScript,
		getActiveSessionCountScript,
		getOpenAIActiveSessionCountScript,
		getOpenAIStagedSessionAccountScript,
		isSessionActiveScript,
		clearOpenAISessionsScript,
	}
	for _, script := range scripts {
		if err := script.Load(ctx, rdb).Err(); err != nil {
			log.Printf("[SessionLimitCache] Failed to preload Lua script: %v", err)
		}
	}

	cache := &sessionLimitCache{
		rdb:                rdb,
		defaultIdleTimeout: time.Duration(defaultIdleTimeoutMinutes) * time.Minute,
	}
	if err := cache.reconcileOpenAISessionOwnership(ctx); err != nil {
		log.Printf("[SessionLimitCache] Failed to reconcile OpenAI SessionID ownership: %v", err)
	}
	return cache
}

// sessionLimitKey 生成会话限制的 Redis 键
func sessionLimitKey(accountID int64) string {
	return fmt.Sprintf("%s%d", sessionLimitKeyPrefix, accountID)
}

func openAISessionLimitKey(accountID int64) string {
	return fmt.Sprintf("%s%d", openAISessionLimitKeyPrefix, accountID)
}

func openAISessionStagingKey(accountID int64) string {
	return fmt.Sprintf("%s%d", openAISessionStagingKeyPrefix, accountID)
}

func openAISessionOwnerKey(sessionIDHash string) string {
	return openAISessionOwnerKeyPrefix + sessionIDHash
}

func openAISessionPreferredKey(sessionIDHash string) string {
	return openAISessionPreferredKeyPrefix + sessionIDHash
}

// windowCostKey 生成窗口费用缓存的 Redis 键
func windowCostKey(accountID int64) string {
	return fmt.Sprintf("%s%d", windowCostKeyPrefix, accountID)
}

// RegisterSession 注册会话活动
func (c *sessionLimitCache) RegisterSession(ctx context.Context, accountID int64, sessionUUID string, maxSessions int, idleTimeout time.Duration) (bool, error) {
	if sessionUUID == "" || maxSessions <= 0 {
		return true, nil // 无效参数，默认允许
	}

	key := sessionLimitKey(accountID)
	idleTimeoutSeconds := int(idleTimeout.Seconds())
	if idleTimeoutSeconds <= 0 {
		idleTimeoutSeconds = int(c.defaultIdleTimeout.Seconds())
	}

	result, err := registerSessionScript.Run(ctx, c.rdb, []string{key}, maxSessions, idleTimeoutSeconds, sessionUUID).Int()
	if err != nil {
		return true, err // 失败开放：缓存错误时允许请求通过
	}
	return result == 1, nil
}

// RefreshSession 刷新会话时间戳
func (c *sessionLimitCache) RefreshSession(ctx context.Context, accountID int64, sessionUUID string, idleTimeout time.Duration) error {
	if sessionUUID == "" {
		return nil
	}

	key := sessionLimitKey(accountID)
	idleTimeoutSeconds := int(idleTimeout.Seconds())
	if idleTimeoutSeconds <= 0 {
		idleTimeoutSeconds = int(c.defaultIdleTimeout.Seconds())
	}

	_, err := refreshSessionScript.Run(ctx, c.rdb, []string{key}, idleTimeoutSeconds, sessionUUID).Result()
	return err
}

// GetActiveSessionCount 获取活跃会话数
func (c *sessionLimitCache) GetActiveSessionCount(ctx context.Context, accountID int64) (int, error) {
	key := sessionLimitKey(accountID)
	idleTimeoutSeconds := int(c.defaultIdleTimeout.Seconds())

	result, err := getActiveSessionCountScript.Run(ctx, c.rdb, []string{key}, idleTimeoutSeconds).Int()
	if err != nil {
		return 0, err
	}
	return result, nil
}

// GetActiveSessionCountBatch 批量获取多个账号的活跃会话数
func (c *sessionLimitCache) GetActiveSessionCountBatch(ctx context.Context, accountIDs []int64, idleTimeouts map[int64]time.Duration) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return make(map[int64]int), nil
	}

	results := make(map[int64]int, len(accountIDs))

	// 使用 pipeline 批量执行
	pipe := c.rdb.Pipeline()

	cmds := make(map[int64]*redis.Cmd, len(accountIDs))
	for _, accountID := range accountIDs {
		key := sessionLimitKey(accountID)
		// 使用各账号自己的 idleTimeout，如果没有则用默认值
		idleTimeout := c.defaultIdleTimeout
		if idleTimeouts != nil {
			if t, ok := idleTimeouts[accountID]; ok && t > 0 {
				idleTimeout = t
			}
		}
		idleTimeoutSeconds := int(idleTimeout.Seconds())
		cmds[accountID] = getActiveSessionCountScript.Run(ctx, pipe, []string{key}, idleTimeoutSeconds)
	}

	// 执行 pipeline，即使部分失败也尝试获取成功的结果
	_, _ = pipe.Exec(ctx)

	for accountID, cmd := range cmds {
		if result, err := cmd.Int(); err == nil {
			results[accountID] = result
		}
	}

	return results, nil
}

// IsSessionActive 检查会话是否活跃
func (c *sessionLimitCache) IsSessionActive(ctx context.Context, accountID int64, sessionUUID string) (bool, error) {
	if sessionUUID == "" {
		return false, nil
	}

	key := sessionLimitKey(accountID)
	idleTimeoutSeconds := int(c.defaultIdleTimeout.Seconds())

	result, err := isSessionActiveScript.Run(ctx, c.rdb, []string{key}, idleTimeoutSeconds, sessionUUID).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *sessionLimitCache) RegisterOpenAISessionID(ctx context.Context, accountID int64, sessionIDHash string, maxSessions int, idleTimeout time.Duration, rotateWhenFull ...bool) (bool, error) {
	if sessionIDHash == "" || maxSessions <= 0 {
		return true, nil
	}
	idleTimeoutSeconds := int(idleTimeout.Seconds())
	if idleTimeoutSeconds <= 0 {
		idleTimeoutSeconds = int(c.defaultIdleTimeout.Seconds())
	}
	result, err := registerOpenAISessionScript.Run(
		ctx,
		c.rdb,
		[]string{openAISessionLimitKey(accountID), openAISessionOwnerKey(sessionIDHash), openAISessionStagingKey(accountID), openAISessionPreferredKey(sessionIDHash)},
		maxSessions,
		idleTimeoutSeconds,
		sessionIDHash,
		strconv.FormatInt(accountID, 10),
		openAISessionLimitKeyPrefix,
		openAISessionOwnerKeyPrefix,
		openAISessionStagingKeyPrefix,
		openAISessionPreferredKeyPrefix,
		boolToRedisFlag(len(rotateWhenFull) > 0 && rotateWhenFull[0]),
	).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func boolToRedisFlag(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func (c *sessionLimitCache) GetOpenAIStagedSessionAccountID(ctx context.Context, sessionIDHash string) (int64, error) {
	if sessionIDHash == "" {
		return 0, nil
	}
	accountID, err := getOpenAIStagedSessionAccountScript.Run(ctx, c.rdb, []string{openAISessionPreferredKey(sessionIDHash)}).Int64()
	if err != nil {
		return 0, err
	}
	return accountID, nil
}

func (c *sessionLimitCache) GetOpenAIActiveSessionCountBatch(ctx context.Context, accountIDs []int64, idleTimeouts map[int64]time.Duration) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return make(map[int64]int), nil
	}
	results := make(map[int64]int, len(accountIDs))
	pipe := c.rdb.Pipeline()
	cmds := make(map[int64]*redis.Cmd, len(accountIDs))
	for _, accountID := range accountIDs {
		idleTimeout := c.defaultIdleTimeout
		if configured, ok := idleTimeouts[accountID]; ok && configured > 0 {
			idleTimeout = configured
		}
		cmds[accountID] = getOpenAIActiveSessionCountScript.Run(
			ctx,
			pipe,
			[]string{openAISessionLimitKey(accountID), openAISessionStagingKey(accountID)},
			int(idleTimeout.Seconds()),
			strconv.FormatInt(accountID, 10),
			openAISessionOwnerKeyPrefix,
			openAISessionPreferredKeyPrefix,
		)
	}
	_, _ = pipe.Exec(ctx)
	for accountID, cmd := range cmds {
		if result, err := cmd.Int(); err == nil {
			results[accountID] = result
		}
	}
	return results, nil
}

func (c *sessionLimitCache) ClearOpenAISessions(ctx context.Context, accountID int64) error {
	return clearOpenAISessionsScript.Run(
		ctx,
		c.rdb,
		[]string{openAISessionLimitKey(accountID), openAISessionStagingKey(accountID)},
		strconv.FormatInt(accountID, 10),
		openAISessionOwnerKeyPrefix,
		openAISessionPreferredKeyPrefix,
	).Err()
}

type openAISessionOwner struct {
	accountID  int64
	accountKey string
	score      float64
	ttl        time.Duration
	staged     bool
}

// reconcileOpenAISessionOwnership migrates legacy per-account slots and keeps
// the most recently active account when older builds left duplicate entries.
func (c *sessionLimitCache) reconcileOpenAISessionOwnership(ctx context.Context) error {
	owners := make(map[string]openAISessionOwner)
	removals := make(map[string][]any)
	duplicateMemberships := 0
	for _, namespace := range []struct {
		prefix string
		staged bool
	}{
		{prefix: openAISessionLimitKeyPrefix},
		{prefix: openAISessionStagingKeyPrefix, staged: true},
	} {
		iterator := c.rdb.Scan(ctx, 0, namespace.prefix+"*", 256).Iterator()
		for iterator.Next(ctx) {
			accountKey := iterator.Val()
			accountID, err := strconv.ParseInt(strings.TrimPrefix(accountKey, namespace.prefix), 10, 64)
			if err != nil || accountID <= 0 {
				continue
			}
			ttl, err := c.rdb.TTL(ctx, accountKey).Result()
			if err != nil {
				return err
			}
			entries, err := c.rdb.ZRangeWithScores(ctx, accountKey, 0, -1).Result()
			if err != nil {
				return err
			}
			for _, entry := range entries {
				sessionID, ok := entry.Member.(string)
				if !ok || sessionID == "" {
					continue
				}
				candidate := openAISessionOwner{accountID: accountID, accountKey: accountKey, score: entry.Score, ttl: ttl, staged: namespace.staged}
				current, exists := owners[sessionID]
				if !exists {
					owners[sessionID] = candidate
					continue
				}
				duplicateMemberships++
				candidateWins := (current.staged && !candidate.staged) ||
					(current.staged == candidate.staged && (candidate.score > current.score ||
						(candidate.score == current.score && candidate.accountID < current.accountID)))
				if candidateWins {
					removals[current.accountKey] = append(removals[current.accountKey], sessionID)
					owners[sessionID] = candidate
				} else {
					removals[accountKey] = append(removals[accountKey], sessionID)
				}
			}
		}
		if err := iterator.Err(); err != nil {
			return err
		}
	}

	pipe := c.rdb.Pipeline()
	for accountKey, sessions := range removals {
		pipe.ZRem(ctx, accountKey, sessions...)
	}
	for sessionID, owner := range owners {
		ttl := owner.ttl
		if ttl > time.Minute {
			ttl -= time.Minute
		} else if ttl <= 0 {
			ttl = c.defaultIdleTimeout
		}
		expectedOwner := strconv.FormatInt(owner.accountID, 10)
		existingOwner, ownerErr := c.rdb.Get(ctx, openAISessionOwnerKey(sessionID)).Result()
		if ownerErr != nil && ownerErr != redis.Nil {
			return ownerErr
		}
		if existingOwner != expectedOwner {
			pipe.Set(ctx, openAISessionOwnerKey(sessionID), expectedOwner, ttl+c.defaultIdleTimeout+2*time.Minute)
		}
		// A missing preference may mean its active+staging retention already
		// elapsed. Do not recreate it during startup reconciliation. Legacy
		// active entries gain a preference on their next atomic registration or
		// active-to-staging transition.
		existingPreferred, preferredErr := c.rdb.Get(ctx, openAISessionPreferredKey(sessionID)).Result()
		if preferredErr != nil && preferredErr != redis.Nil {
			return preferredErr
		}
		preferredOwner := existingPreferred
		if separator := strings.IndexByte(preferredOwner, ':'); separator >= 0 {
			preferredOwner = preferredOwner[:separator]
		}
		if preferredErr == nil && preferredOwner != expectedOwner {
			pipe.Del(ctx, openAISessionPreferredKey(sessionID))
		}
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return err
	}
	if duplicateMemberships > 0 {
		log.Printf("[SessionLimitCache] Removed %d duplicate OpenAI SessionID slot memberships", duplicateMemberships)
	}
	return nil
}

// ========== 5h窗口费用缓存实现 ==========

// GetWindowCost 获取缓存的窗口费用
func (c *sessionLimitCache) GetWindowCost(ctx context.Context, accountID int64) (float64, bool, error) {
	key := windowCostKey(accountID)
	val, err := c.rdb.Get(ctx, key).Float64()
	if err == redis.Nil {
		return 0, false, nil // 缓存未命中
	}
	if err != nil {
		return 0, false, err
	}
	return val, true, nil
}

// SetWindowCost 设置窗口费用缓存
func (c *sessionLimitCache) SetWindowCost(ctx context.Context, accountID int64, cost float64) error {
	key := windowCostKey(accountID)
	return c.rdb.Set(ctx, key, cost, windowCostCacheTTL).Err()
}

// GetWindowCostBatch 批量获取窗口费用缓存
func (c *sessionLimitCache) GetWindowCostBatch(ctx context.Context, accountIDs []int64) (map[int64]float64, error) {
	if len(accountIDs) == 0 {
		return make(map[int64]float64), nil
	}

	// 构建批量查询的 keys
	keys := make([]string, len(accountIDs))
	for i, accountID := range accountIDs {
		keys[i] = windowCostKey(accountID)
	}

	// 使用 MGET 批量获取
	vals, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	results := make(map[int64]float64, len(accountIDs))
	for i, val := range vals {
		if val == nil {
			continue // 缓存未命中
		}
		// 尝试解析为 float64
		switch v := val.(type) {
		case string:
			if cost, err := strconv.ParseFloat(v, 64); err == nil {
				results[accountIDs[i]] = cost
			}
		case float64:
			results[accountIDs[i]] = v
		}
	}

	return results, nil
}
