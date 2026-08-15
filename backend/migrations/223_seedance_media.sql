CREATE TABLE IF NOT EXISTS seedance_resources (
    id BIGSERIAL PRIMARY KEY,
    resource_id VARCHAR(255) NOT NULL,
    resource_type VARCHAR(32) NOT NULL,
    channel VARCHAR(32) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT,
    group_id BIGINT,
    account_id BIGINT NOT NULL,
    parent_id VARCHAR(255) NOT NULL DEFAULT '',
    task_id VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT seedance_resources_type_check CHECK (resource_type IN ('asset_group', 'asset')),
    CONSTRAINT seedance_resources_channel_check CHECK (channel IN ('group', 'sd', 'doubao'))
);

CREATE UNIQUE INDEX IF NOT EXISTS seedance_resources_type_id_key
    ON seedance_resources (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS seedance_resources_owner_idx
    ON seedance_resources (user_id, api_key_id, resource_type, resource_id);
CREATE INDEX IF NOT EXISTS seedance_resources_account_idx
    ON seedance_resources (account_id, created_at DESC);

CREATE TABLE IF NOT EXISTS seedance_video_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT,
    group_id BIGINT,
    account_id BIGINT NOT NULL,
    model VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    duration_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
    resolution VARCHAR(32) NOT NULL DEFAULT '',
    request_body JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_body JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_error_code VARCHAR(128) NOT NULL DEFAULT '',
    last_error_message TEXT NOT NULL DEFAULT '',
    billed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS seedance_video_tasks_owner_created_idx
    ON seedance_video_tasks (user_id, api_key_id, created_at DESC);
CREATE INDEX IF NOT EXISTS seedance_video_tasks_status_updated_idx
    ON seedance_video_tasks (status, updated_at DESC);
CREATE INDEX IF NOT EXISTS seedance_video_tasks_account_idx
    ON seedance_video_tasks (account_id, created_at DESC);
