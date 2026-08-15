ALTER TABLE announcement_group_price_changes
    ADD COLUMN IF NOT EXISTS change_type VARCHAR(20) NOT NULL DEFAULT 'price',
    ADD COLUMN IF NOT EXISTS old_status VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS new_status VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_exclusive BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS subscription_type VARCHAR(20) NOT NULL DEFAULT 'standard',
    ADD COLUMN IF NOT EXISTS tag_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS access_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN announcement_group_price_changes.change_type IS '监测事件类型: price, created, status, deleted';
COMMENT ON COLUMN announcement_group_price_changes.tag_ids IS '事件发生时的标签授权快照';
COMMENT ON COLUMN announcement_group_price_changes.access_user_ids IS '事件发生时的手工授权和订阅用户快照';
COMMENT ON TABLE announcement_group_price_changes IS '分组价格与状态监测公告的结构化变更明细';
