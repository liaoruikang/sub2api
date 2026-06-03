ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS new_user_only boolean NOT NULL DEFAULT false;

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS listed_at timestamptz;

UPDATE subscription_plans
SET listed_at = created_at
WHERE listed_at IS NULL;
