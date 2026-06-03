-- 在已有的用户专属分组配置表上扩展限时倍率窗口专属 RPM。
-- 语义：
--   - limited_time_rpm_override NULL     → 未配置，限时窗口内继续回退普通 rpm_override / group.rpm_limit
--   - limited_time_rpm_override 非 NULL  → 仅在分组限时倍率窗口生效时覆盖普通分组 RPM（0 = 不限制）
-- 用户级 users.rpm_limit 仍独立生效（跨分组总配额）。
ALTER TABLE user_group_rate_multipliers
    ADD COLUMN IF NOT EXISTS limited_time_rpm_override integer NULL;

COMMENT ON COLUMN user_group_rate_multipliers.limited_time_rpm_override IS '限时倍率窗口专属 RPM 上限；NULL 表示未配置；0 表示该用户在此分组限时窗口内不受分组 RPM 限制。';
