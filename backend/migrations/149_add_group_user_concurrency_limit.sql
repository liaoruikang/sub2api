-- Add per-user concurrency limit within a group.
-- 0 means unlimited.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS user_concurrency_limit INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN groups.user_concurrency_limit IS 'Per-user concurrency limit within this group; 0 means unlimited';
