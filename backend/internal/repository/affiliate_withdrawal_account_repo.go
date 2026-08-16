package repository

import (
	"context"
	"database/sql"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const affiliateWithdrawalAccountSelect = `
SELECT id,
       user_id,
       account_type,
       account_encrypted,
       account_masked,
       is_default,
       created_at,
       updated_at
FROM user_affiliate_withdrawal_accounts`

func scanAffiliateWithdrawalAccount(scanner affiliateWithdrawalScanner) (*service.AffiliateWithdrawalAccount, error) {
	var item service.AffiliateWithdrawalAccount
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.AccountType,
		&item.AccountEncrypted,
		&item.AccountMasked,
		&item.IsDefault,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *affiliateRepository) CreateAffiliateWithdrawalAccount(ctx context.Context, input service.AffiliateWithdrawalAccountCreateInput) (*service.AffiliateWithdrawalAccount, error) {
	var accountID int64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateWithdrawalAccountUser(txCtx, txClient, input.UserID); err != nil {
			return err
		}

		count, err := countAffiliateWithdrawalAccounts(txCtx, txClient, input.UserID)
		if err != nil {
			return err
		}
		if count >= service.AffiliateWithdrawalAccountMaxCount {
			return service.ErrAffiliateWithdrawalAccountLimit
		}

		rows, err := txClient.QueryContext(txCtx, `
INSERT INTO user_affiliate_withdrawal_accounts (
    user_id,
    account_type,
    account_encrypted,
    account_masked,
    is_default,
    created_at,
    updated_at
)
VALUES ($1, 'alipay', $2, $3, $4, NOW(), NOW())
RETURNING id`, input.UserID, input.AccountEncrypted, input.AccountMasked, count == 0)
		if err != nil {
			return fmt.Errorf("insert affiliate withdrawal account: %w", err)
		}
		if !rows.Next() {
			rowsErr := rows.Err()
			_ = rows.Close()
			if rowsErr != nil {
				return rowsErr
			}
			return fmt.Errorf("insert affiliate withdrawal account: missing returned id")
		}
		if err := rows.Scan(&accountID); err != nil {
			_ = rows.Close()
			return err
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return r.GetAffiliateWithdrawalAccount(ctx, input.UserID, accountID)
}

func (r *affiliateRepository) ListAffiliateWithdrawalAccounts(ctx context.Context, userID int64) ([]service.AffiliateWithdrawalAccount, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, affiliateWithdrawalAccountSelect+`
WHERE user_id = $1
ORDER BY is_default DESC, created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.AffiliateWithdrawalAccount, 0)
	for rows.Next() {
		item, scanErr := scanAffiliateWithdrawalAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *affiliateRepository) GetAffiliateWithdrawalAccount(ctx context.Context, userID, accountID int64) (*service.AffiliateWithdrawalAccount, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, affiliateWithdrawalAccountSelect+`
WHERE user_id = $1 AND id = $2
LIMIT 1`, userID, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateWithdrawalAccountNotFound
	}
	return scanAffiliateWithdrawalAccount(rows)
}

func (r *affiliateRepository) UpdateAffiliateWithdrawalAccount(ctx context.Context, input service.AffiliateWithdrawalAccountUpdateInput) (*service.AffiliateWithdrawalAccount, error) {
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, `
UPDATE user_affiliate_withdrawal_accounts
SET account_encrypted = $3,
    account_masked = $4,
    updated_at = NOW()
WHERE user_id = $1 AND id = $2`, input.UserID, input.ID, input.AccountEncrypted, input.AccountMasked)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrAffiliateWithdrawalAccountNotFound
	}
	return r.GetAffiliateWithdrawalAccount(ctx, input.UserID, input.ID)
}

func (r *affiliateRepository) SetDefaultAffiliateWithdrawalAccount(ctx context.Context, userID, accountID int64) (*service.AffiliateWithdrawalAccount, error) {
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateWithdrawalAccountUser(txCtx, txClient, userID); err != nil {
			return err
		}
		if err := requireAffiliateWithdrawalAccount(txCtx, txClient, userID, accountID); err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliate_withdrawal_accounts
SET is_default = FALSE, updated_at = NOW()
WHERE user_id = $1 AND is_default`, userID); err != nil {
			return err
		}
		_, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliate_withdrawal_accounts
SET is_default = TRUE, updated_at = NOW()
WHERE user_id = $1 AND id = $2`, userID, accountID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r.GetAffiliateWithdrawalAccount(ctx, userID, accountID)
}

func (r *affiliateRepository) DeleteAffiliateWithdrawalAccount(ctx context.Context, userID, accountID int64) error {
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if err := lockAffiliateWithdrawalAccountUser(txCtx, txClient, userID); err != nil {
			return err
		}
		rows, err := txClient.QueryContext(txCtx, `
SELECT is_default
FROM user_affiliate_withdrawal_accounts
WHERE user_id = $1 AND id = $2
FOR UPDATE`, userID, accountID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			rowsErr := rows.Err()
			_ = rows.Close()
			if rowsErr != nil {
				return rowsErr
			}
			return service.ErrAffiliateWithdrawalAccountNotFound
		}
		var wasDefault bool
		if err := rows.Scan(&wasDefault); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}

		if _, err := txClient.ExecContext(txCtx, `
DELETE FROM user_affiliate_withdrawal_accounts
WHERE user_id = $1 AND id = $2`, userID, accountID); err != nil {
			return err
		}
		if !wasDefault {
			return nil
		}
		_, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliate_withdrawal_accounts
SET is_default = TRUE, updated_at = NOW()
WHERE id = (
    SELECT id
    FROM user_affiliate_withdrawal_accounts
    WHERE user_id = $1
    ORDER BY created_at DESC, id DESC
    LIMIT 1
)`, userID)
		return err
	})
}

func lockAffiliateWithdrawalAccountUser(ctx context.Context, client *dbent.Client, userID int64) error {
	rows, err := client.QueryContext(ctx, `SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrUserNotFound
	}
	var lockedID int64
	return rows.Scan(&lockedID)
}

func countAffiliateWithdrawalAccounts(ctx context.Context, client *dbent.Client, userID int64) (int, error) {
	rows, err := client.QueryContext(ctx, `
SELECT COUNT(*)
FROM user_affiliate_withdrawal_accounts
WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, sql.ErrNoRows
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func requireAffiliateWithdrawalAccount(ctx context.Context, client *dbent.Client, userID, accountID int64) error {
	rows, err := client.QueryContext(ctx, `
SELECT id
FROM user_affiliate_withdrawal_accounts
WHERE user_id = $1 AND id = $2
FOR UPDATE`, userID, accountID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrAffiliateWithdrawalAccountNotFound
	}
	var id int64
	return rows.Scan(&id)
}
