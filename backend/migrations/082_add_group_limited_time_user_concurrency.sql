-- Add group-level per-user concurrency override during limited-time multiplier windows.
-- 0 means no dedicated limited-time override; fall back to groups.user_concurrency_limit.
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS limited_time_user_concurrency_limit INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN groups.limited_time_user_concurrency_limit IS '限时倍率窗口内分组内每用户最大并发数，0 表示不设置专属限时并发';
