-- 100_subscription_plan_purchase_limits.sql
--
-- Adds per-plan purchase limits and first-purchase discount fields.
-- purchase_limit_count = 0 means unlimited.
--
-- Idempotent: only adds missing columns.

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS purchase_limit_count integer NOT NULL DEFAULT 0;

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS first_purchase_discount_enabled boolean NOT NULL DEFAULT false;

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS first_purchase_discount_price numeric(20,2);
