ALTER TABLE announcements
    ADD COLUMN IF NOT EXISTS kind VARCHAR(30) NOT NULL DEFAULT 'manual';

CREATE INDEX IF NOT EXISTS idx_announcements_kind ON announcements(kind);

CREATE TABLE IF NOT EXISTS announcement_group_price_changes (
    id BIGSERIAL PRIMARY KEY,
    announcement_id BIGINT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL,
    group_name VARCHAR(100) NOT NULL,
    old_rate DECIMAL(20,8) NOT NULL,
    new_rate DECIMAL(20,8) NOT NULL,
    sequence INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_announcement_group_price_changes_announcement_sequence
    ON announcement_group_price_changes(announcement_id, sequence);
CREATE INDEX IF NOT EXISTS idx_announcement_group_price_changes_group_id
    ON announcement_group_price_changes(group_id);

CREATE TABLE IF NOT EXISTS announcement_group_price_reads (
    id BIGSERIAL PRIMARY KEY,
    announcement_id BIGINT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(announcement_id, user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_announcement_group_price_reads_user_announcement
    ON announcement_group_price_reads(user_id, announcement_id);
CREATE INDEX IF NOT EXISTS idx_announcement_group_price_reads_group_id
    ON announcement_group_price_reads(group_id);

COMMENT ON COLUMN announcements.kind IS '公告类型: manual, group_price_change';
COMMENT ON TABLE announcement_group_price_changes IS '价格监测公告的结构化分组倍率变更明细';
COMMENT ON TABLE announcement_group_price_reads IS '用户对价格公告中各分组明细的已读记录';
