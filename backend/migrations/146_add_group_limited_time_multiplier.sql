-- 146_add_group_limited_time_multiplier.sql
-- Add optional scheduled limited-time multiplier settings for standard balance groups.

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS limited_time_multiplier_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS limited_time_multiplier_cron VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS limited_time_multiplier_duration_minutes INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS limited_time_multiplier_value DECIMAL(10,4) NOT NULL DEFAULT 1.0;
