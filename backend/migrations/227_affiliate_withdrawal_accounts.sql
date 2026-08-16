CREATE TABLE IF NOT EXISTS user_affiliate_withdrawal_accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_type VARCHAR(16) NOT NULL DEFAULT 'alipay'
        CHECK (account_type = 'alipay'),
    account_encrypted TEXT NOT NULL,
    account_masked VARCHAR(128) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_withdrawal_accounts_user
    ON user_affiliate_withdrawal_accounts(user_id, is_default DESC, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_withdrawal_accounts_default
    ON user_affiliate_withdrawal_accounts(user_id)
    WHERE is_default;

-- Existing users can immediately reuse their most recent withdrawal account.
INSERT INTO user_affiliate_withdrawal_accounts (
    user_id,
    account_type,
    account_encrypted,
    account_masked,
    is_default,
    created_at,
    updated_at
)
SELECT DISTINCT ON (w.user_id)
       w.user_id,
       'alipay',
       w.alipay_account_encrypted,
       w.alipay_account_masked,
       TRUE,
       NOW(),
       NOW()
FROM user_affiliate_withdrawals w
WHERE NOT EXISTS (
    SELECT 1
    FROM user_affiliate_withdrawal_accounts account
    WHERE account.user_id = w.user_id
)
ORDER BY w.user_id, w.created_at DESC, w.id DESC;

COMMENT ON TABLE user_affiliate_withdrawal_accounts IS '用户保存的返利提现账号簿';
COMMENT ON COLUMN user_affiliate_withdrawal_accounts.account_encrypted IS 'AES-256-GCM 加密的支付宝账号';
COMMENT ON COLUMN user_affiliate_withdrawal_accounts.account_masked IS '用户端可见的支付宝账号脱敏快照';
COMMENT ON COLUMN user_affiliate_withdrawal_accounts.is_default IS '用户发起提现时默认选择的账号';
