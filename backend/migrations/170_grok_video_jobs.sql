CREATE TABLE IF NOT EXISTS grok_video_jobs (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT,
    group_id BIGINT,
    account_id BIGINT,
    model VARCHAR(128) NOT NULL DEFAULT '',
    prompt_preview VARCHAR(500) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    progress_percent INTEGER NOT NULL DEFAULT 0,
    progress_text VARCHAR(255) NOT NULL DEFAULT '',
    result_url TEXT,
    result_urls TEXT,
    cover_image_url TEXT,
    last_error_code VARCHAR(128),
    last_error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_polled_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS grok_video_jobs_request_id_key
    ON grok_video_jobs (request_id);

CREATE INDEX IF NOT EXISTS grok_video_jobs_user_created_at_idx
    ON grok_video_jobs (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS grok_video_jobs_status_updated_at_idx
    ON grok_video_jobs (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS grok_video_jobs_api_key_created_at_idx
    ON grok_video_jobs (api_key_id, created_at DESC);
