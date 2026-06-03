ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS ip_purchase_limit_count integer NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS payment_orders_plan_client_ip_paid_at_idx
    ON payment_orders (plan_id, client_ip, paid_at);
