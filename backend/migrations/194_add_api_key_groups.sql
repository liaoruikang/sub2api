CREATE TABLE IF NOT EXISTS api_key_groups (
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    group_id   BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (api_key_id, group_id),
    CONSTRAINT api_key_groups_position_non_negative CHECK (position >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_key_groups_api_key_position
    ON api_key_groups(api_key_id, position);

CREATE INDEX IF NOT EXISTS idx_api_key_groups_group_id
    ON api_key_groups(group_id);

INSERT INTO api_key_groups (api_key_id, group_id, position)
SELECT k.id, k.group_id, 0
FROM api_keys AS k
JOIN groups AS g ON g.id = k.group_id
WHERE k.group_id IS NOT NULL
  AND k.deleted_at IS NULL
ON CONFLICT DO NOTHING;

CREATE OR REPLACE FUNCTION enqueue_group_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_group_id BIGINT;
BEGIN
    target_group_id := OLD.id;
    IF TG_OP = 'UPDATE'
       AND OLD.status IS NOT DISTINCT FROM NEW.status
       AND OLD.is_exclusive IS NOT DISTINCT FROM NEW.is_exclusive
       AND OLD.allow_image_generation IS NOT DISTINCT FROM NEW.allow_image_generation
       AND OLD.platform IS NOT DISTINCT FROM NEW.platform
       AND OLD.subscription_type IS NOT DISTINCT FROM NEW.subscription_type
       AND OLD.rate_multiplier IS NOT DISTINCT FROM NEW.rate_multiplier
       AND OLD.peak_rate_enabled IS NOT DISTINCT FROM NEW.peak_rate_enabled
       AND OLD.peak_start IS NOT DISTINCT FROM NEW.peak_start
       AND OLD.peak_end IS NOT DISTINCT FROM NEW.peak_end
       AND OLD.peak_rate_multiplier IS NOT DISTINCT FROM NEW.peak_rate_multiplier
       AND OLD.profit_control_enabled IS NOT DISTINCT FROM NEW.profit_control_enabled
       AND OLD.profit_min_margin IS NOT DISTINCT FROM NEW.profit_min_margin
       AND OLD.profit_safety_buffer IS NOT DISTINCT FROM NEW.profit_safety_buffer
       AND OLD.deleted_at IS NOT DISTINCT FROM NEW.deleted_at THEN
        RETURN NEW;
    END IF;

    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT DISTINCT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE k.deleted_at IS NULL
      AND k.key <> ''
      AND (
          k.group_id = target_group_id
          OR EXISTS (
              SELECT 1
              FROM api_key_groups AS akg
              WHERE akg.api_key_id = k.id
                AND akg.group_id = target_group_id
          )
      );
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION enqueue_api_key_group_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.api_key_id IS DISTINCT FROM NEW.api_key_id THEN
        INSERT INTO auth_cache_invalidation_outbox (cache_key)
        SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
        FROM api_keys AS k
        WHERE k.id = OLD.api_key_id
          AND k.deleted_at IS NULL
          AND k.key <> '';
    END IF;

    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE k.id = CASE WHEN TG_OP = 'DELETE' THEN OLD.api_key_id ELSE NEW.api_key_id END
      AND k.deleted_at IS NULL
      AND k.key <> '';

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_api_key_groups_auth_cache_invalidation ON api_key_groups;
CREATE TRIGGER trg_api_key_groups_auth_cache_invalidation
AFTER INSERT OR UPDATE OR DELETE ON api_key_groups
FOR EACH ROW
EXECUTE FUNCTION enqueue_api_key_group_auth_cache_invalidation();
