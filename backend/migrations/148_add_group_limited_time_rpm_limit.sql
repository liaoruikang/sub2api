-- Add group-level RPM limit for limited-time multiplier windows.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS limited_time_rpm_limit INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN groups.limited_time_rpm_limit IS '限时倍率窗口内分组 RPM 上限，0 表示不设置专属限时 RPM';
