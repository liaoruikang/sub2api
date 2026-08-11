CREATE TABLE IF NOT EXISTS user_tags (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_tags_name_active
    ON user_tags (LOWER(BTRIM(name)))
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_tags_deleted_at
    ON user_tags (deleted_at);

CREATE TABLE IF NOT EXISTS user_tag_assignments (
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES user_tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_user_tag_assignments_tag_id
    ON user_tag_assignments (tag_id);

CREATE TABLE IF NOT EXISTS group_user_tags (
    group_id   BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES user_tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_group_user_tags_tag_id
    ON group_user_tags (tag_id);

CREATE OR REPLACE FUNCTION enqueue_user_all_auth_cache_invalidation(target_user_id BIGINT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    IF target_user_id IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT DISTINCT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM api_keys AS k
    WHERE k.user_id = target_user_id
      AND k.deleted_at IS NULL
      AND k.key <> '';
END;
$$;

CREATE OR REPLACE FUNCTION enqueue_tag_users_auth_cache_invalidation(target_tag_id BIGINT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    IF target_tag_id IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO auth_cache_invalidation_outbox (cache_key)
    SELECT DISTINCT encode(sha256(convert_to(k.key, 'UTF8')), 'hex')
    FROM user_tag_assignments AS uta
    JOIN api_keys AS k ON k.user_id = uta.user_id
    WHERE uta.tag_id = target_tag_id
      AND k.deleted_at IS NULL
      AND k.key <> '';
END;
$$;

CREATE OR REPLACE FUNCTION enqueue_user_tag_assignment_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.user_id IS NOT DISTINCT FROM NEW.user_id
           AND OLD.tag_id IS NOT DISTINCT FROM NEW.tag_id THEN
            RETURN NEW;
        END IF;
        PERFORM enqueue_user_all_auth_cache_invalidation(OLD.user_id);
    END IF;

    PERFORM enqueue_user_all_auth_cache_invalidation(
        CASE WHEN TG_OP = 'DELETE' THEN OLD.user_id ELSE NEW.user_id END
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_user_tag_assignments_auth_cache_invalidation ON user_tag_assignments;
CREATE TRIGGER trg_user_tag_assignments_auth_cache_invalidation
AFTER INSERT OR UPDATE OR DELETE ON user_tag_assignments
FOR EACH ROW EXECUTE FUNCTION enqueue_user_tag_assignment_auth_cache_invalidation();

CREATE OR REPLACE FUNCTION enqueue_group_user_tag_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.group_id IS NOT DISTINCT FROM NEW.group_id
           AND OLD.tag_id IS NOT DISTINCT FROM NEW.tag_id THEN
            RETURN NEW;
        END IF;
        PERFORM enqueue_tag_users_auth_cache_invalidation(OLD.tag_id);
    END IF;

    PERFORM enqueue_tag_users_auth_cache_invalidation(
        CASE WHEN TG_OP = 'DELETE' THEN OLD.tag_id ELSE NEW.tag_id END
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_group_user_tags_auth_cache_invalidation ON group_user_tags;
CREATE TRIGGER trg_group_user_tags_auth_cache_invalidation
AFTER INSERT OR UPDATE OR DELETE ON group_user_tags
FOR EACH ROW EXECUTE FUNCTION enqueue_group_user_tag_auth_cache_invalidation();

CREATE OR REPLACE FUNCTION enqueue_user_tag_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE'
       AND OLD.deleted_at IS NOT DISTINCT FROM NEW.deleted_at THEN
        RETURN NEW;
    END IF;

    PERFORM enqueue_tag_users_auth_cache_invalidation(OLD.id);

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_user_tags_auth_cache_invalidation ON user_tags;
CREATE TRIGGER trg_user_tags_auth_cache_invalidation
AFTER UPDATE OR DELETE ON user_tags
FOR EACH ROW EXECUTE FUNCTION enqueue_user_tag_auth_cache_invalidation();

CREATE OR REPLACE FUNCTION enqueue_allowed_group_auth_cache_invalidation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.user_id IS NOT DISTINCT FROM NEW.user_id
           AND OLD.group_id IS NOT DISTINCT FROM NEW.group_id THEN
            RETURN NEW;
        END IF;
        PERFORM enqueue_user_all_auth_cache_invalidation(OLD.user_id);
    END IF;

    PERFORM enqueue_user_all_auth_cache_invalidation(
        CASE WHEN TG_OP = 'DELETE' THEN OLD.user_id ELSE NEW.user_id END
    );

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

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

COMMENT ON TABLE user_tags IS
    'Reusable user authorization tags; active names are unique case-insensitively';
COMMENT ON TABLE user_tag_assignments IS
    'User tag membership; tag-derived group access is resolved without writing user_allowed_groups';
COMMENT ON TABLE group_user_tags IS
    'Tags granting automatic access to standard exclusive groups with OR semantics';
