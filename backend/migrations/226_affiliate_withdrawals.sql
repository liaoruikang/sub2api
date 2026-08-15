CREATE TABLE IF NOT EXISTS user_affiliate_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    request_no VARCHAR(32) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,8) NOT NULL CHECK (amount > 0),
    fee_rate DECIMAL(7,4) NOT NULL CHECK (fee_rate >= 0 AND fee_rate <= 100),
    fee_amount DECIMAL(20,8) NOT NULL CHECK (fee_amount >= 0 AND fee_amount <= amount),
    payout_amount DECIMAL(20,8) NOT NULL CHECK (payout_amount > 0 AND payout_amount = amount - fee_amount),
    alipay_account_encrypted TEXT NOT NULL,
    alipay_account_masked VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'paid', 'rejected')),
    reject_reason VARCHAR(500),
    operator_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_user_created
    ON user_affiliate_withdrawals(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawals_status_created
    ON user_affiliate_withdrawals(status, created_at DESC);

COMMENT ON TABLE user_affiliate_withdrawals IS '邀请返利支付宝提现申请';
COMMENT ON COLUMN user_affiliate_withdrawals.amount IS '从可用返利冻结并最终扣除的提现总额';
COMMENT ON COLUMN user_affiliate_withdrawals.fee_rate IS '申请时的百分比手续费率快照';
COMMENT ON COLUMN user_affiliate_withdrawals.fee_amount IS '申请时计算的手续费金额快照';
COMMENT ON COLUMN user_affiliate_withdrawals.payout_amount IS '管理员应向支付宝实际转账的净额';
COMMENT ON COLUMN user_affiliate_withdrawals.alipay_account_encrypted IS 'AES-256-GCM 加密的支付宝账号';
COMMENT ON COLUMN user_affiliate_withdrawals.status IS 'pending|paid|rejected';
