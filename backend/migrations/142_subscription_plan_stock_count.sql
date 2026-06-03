-- 142_subscription_plan_stock_count.sql
--
-- Adds total stock for subscription plans. stock_count = 0 means unlimited.

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS stock_count integer NOT NULL DEFAULT 0;
