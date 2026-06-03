-- 143_subscription_plan_off_sale_at.sql
-- Adds optional automatic off-sale time for subscription plans.

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS off_sale_at timestamptz;
