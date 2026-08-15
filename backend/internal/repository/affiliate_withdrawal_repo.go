package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const affiliateWithdrawalSelect = `
SELECT w.id,
       w.request_no,
       w.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       w.amount::double precision,
       w.fee_rate::double precision,
       w.fee_amount::double precision,
       w.payout_amount::double precision,
       w.alipay_account_encrypted,
       w.alipay_account_masked,
       w.status,
       COALESCE(w.reject_reason, ''),
       w.operator_id,
       COALESCE(operator.email, ''),
       w.processed_at,
       w.created_at,
       w.updated_at
FROM user_affiliate_withdrawals w
JOIN users u ON u.id = w.user_id
LEFT JOIN users operator ON operator.id = w.operator_id`

type affiliateWithdrawalScanner interface {
	Scan(dest ...any) error
}

func scanAffiliateWithdrawal(scanner affiliateWithdrawalScanner) (*service.AffiliateWithdrawal, error) {
	var item service.AffiliateWithdrawal
	var operatorID sql.NullInt64
	var processedAt sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.RequestNo,
		&item.UserID,
		&item.UserEmail,
		&item.Username,
		&item.Amount,
		&item.FeeRate,
		&item.FeeAmount,
		&item.PayoutAmount,
		&item.AlipayAccountEncrypted,
		&item.AlipayAccountMasked,
		&item.Status,
		&item.RejectReason,
		&operatorID,
		&item.OperatorEmail,
		&processedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if operatorID.Valid {
		item.OperatorID = &operatorID.Int64
	}
	if processedAt.Valid {
		item.ProcessedAt = &processedAt.Time
	}
	return &item, nil
}

func (r *affiliateRepository) CreateAffiliateWithdrawal(ctx context.Context, input service.AffiliateWithdrawalCreateInput) (*service.AffiliateWithdrawal, error) {
	var withdrawalID int64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, input.UserID); err != nil {
			return err
		}
		if _, err := thawFrozenQuotaTx(txCtx, txClient, input.UserID); err != nil {
			return fmt.Errorf("thaw before affiliate withdrawal: %w", err)
		}

		rows, err := txClient.QueryContext(txCtx, `
UPDATE user_affiliates
SET aff_quota = aff_quota - $2,
    aff_frozen_quota = aff_frozen_quota + $2,
    updated_at = NOW()
WHERE user_id = $1
  AND aff_quota >= $2
RETURNING user_id`, input.UserID, input.Amount)
		if err != nil {
			return fmt.Errorf("freeze affiliate withdrawal quota: %w", err)
		}
		claimed := rows.Next()
		rowsErr := rows.Err()
		if closeErr := rows.Close(); closeErr != nil {
			return closeErr
		}
		if rowsErr != nil {
			return rowsErr
		}
		if !claimed {
			return service.ErrAffiliateWithdrawalQuota
		}

		rows, err = txClient.QueryContext(txCtx, `
INSERT INTO user_affiliate_withdrawals (
    request_no,
    user_id,
    amount,
    fee_rate,
    fee_amount,
    payout_amount,
    alipay_account_encrypted,
    alipay_account_masked,
    status,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', NOW(), NOW())
RETURNING id`,
			input.RequestNo,
			input.UserID,
			input.Amount,
			input.FeeRate,
			input.FeeAmount,
			input.PayoutAmount,
			input.AlipayAccountEncrypted,
			input.AlipayAccountMasked,
		)
		if err != nil {
			return fmt.Errorf("insert affiliate withdrawal: %w", err)
		}
		if !rows.Next() {
			rowsErr := rows.Err()
			_ = rows.Close()
			if rowsErr != nil {
				return rowsErr
			}
			return fmt.Errorf("insert affiliate withdrawal: missing returned id")
		}
		if err := rows.Scan(&withdrawalID); err != nil {
			_ = rows.Close()
			return err
		}
		return rows.Close()
	})
	if err != nil {
		return nil, err
	}
	return r.getAffiliateWithdrawal(ctx, withdrawalID)
}

func (r *affiliateRepository) ListUserAffiliateWithdrawals(ctx context.Context, userID int64, filter service.AffiliateWithdrawalFilter) ([]service.AffiliateWithdrawal, int64, error) {
	return r.listAffiliateWithdrawals(ctx, &userID, filter)
}

func (r *affiliateRepository) ListAdminAffiliateWithdrawals(ctx context.Context, filter service.AffiliateWithdrawalFilter) ([]service.AffiliateWithdrawal, int64, error) {
	return r.listAffiliateWithdrawals(ctx, nil, filter)
}

func (r *affiliateRepository) listAffiliateWithdrawals(ctx context.Context, userID *int64, filter service.AffiliateWithdrawalFilter) ([]service.AffiliateWithdrawal, int64, error) {
	client := clientFromContext(ctx, r.client)
	clauses := make([]string, 0, 5)
	args := make([]any, 0, 7)
	if userID != nil {
		args = append(args, *userID)
		clauses = append(clauses, fmt.Sprintf("w.user_id = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, fmt.Sprintf("w.status = $%d", len(args)))
	}
	if filter.StartAt != nil {
		args = append(args, *filter.StartAt)
		clauses = append(clauses, fmt.Sprintf("w.created_at >= $%d", len(args)))
	}
	if filter.EndAt != nil {
		args = append(args, *filter.EndAt)
		clauses = append(clauses, fmt.Sprintf("w.created_at <= $%d", len(args)))
	}
	if search := strings.TrimSpace(filter.Search); search != "" && userID == nil {
		args = append(args, "%"+strings.ToLower(search)+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(LOWER(w.request_no) LIKE "+placeholder+" OR LOWER(u.email) LIKE "+placeholder+" OR LOWER(u.username) LIKE "+placeholder+" OR w.user_id::text LIKE "+placeholder+")")
	}
	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countRows, err := client.QueryContext(ctx, `
SELECT COUNT(*)
FROM user_affiliate_withdrawals w
JOIN users u ON u.id = w.user_id`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if countRows.Next() {
		err = countRows.Scan(&total)
	} else if err == nil {
		err = countRows.Err()
	}
	if closeErr := countRows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, 0, err
	}

	sortColumns := map[string]string{
		"request_no":    "w.request_no",
		"user":          "u.email",
		"amount":        "w.amount",
		"fee_amount":    "w.fee_amount",
		"payout_amount": "w.payout_amount",
		"status":        "w.status",
		"processed_at":  "w.processed_at",
		"created_at":    "w.created_at",
	}
	sortColumn := sortColumns[filter.SortBy]
	if sortColumn == "" {
		sortColumn = "w.created_at"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := affiliateWithdrawalSelect + where + " ORDER BY " + sortColumn + " " + direction + " NULLS LAST LIMIT $" + fmt.Sprint(len(args)-1) + " OFFSET $" + fmt.Sprint(len(args))
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.AffiliateWithdrawal, 0)
	for rows.Next() {
		item, scanErr := scanAffiliateWithdrawal(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *affiliateRepository) ProcessAffiliateWithdrawal(ctx context.Context, withdrawalID, operatorID int64, status, rejectReason string) (*service.AffiliateWithdrawal, error) {
	if status != service.AffiliateWithdrawalStatusPaid && status != service.AffiliateWithdrawalStatusRejected {
		return nil, service.ErrAffiliateWithdrawalStatus
	}
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		rows, err := txClient.QueryContext(txCtx, `
SELECT user_id, status, amount::double precision
FROM user_affiliate_withdrawals
WHERE id = $1
FOR UPDATE`, withdrawalID)
		if err != nil {
			return err
		}
		if !rows.Next() {
			rowsErr := rows.Err()
			_ = rows.Close()
			if rowsErr != nil {
				return rowsErr
			}
			return service.ErrAffiliateWithdrawalNotFound
		}
		var userID int64
		var currentStatus string
		var amount float64
		if err := rows.Scan(&userID, &currentStatus, &amount); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if currentStatus != service.AffiliateWithdrawalStatusPending {
			return service.ErrAffiliateWithdrawalState
		}

		var result sql.Result
		if status == service.AffiliateWithdrawalStatusPaid {
			result, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_frozen_quota = aff_frozen_quota - $2,
    updated_at = NOW()
WHERE user_id = $1
  AND aff_frozen_quota >= $2`, userID, amount)
		} else {
			result, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliates
SET aff_quota = aff_quota + $2,
    aff_frozen_quota = aff_frozen_quota - $2,
    updated_at = NOW()
WHERE user_id = $1
  AND aff_frozen_quota >= $2`, userID, amount)
		}
		if err != nil {
			return fmt.Errorf("settle affiliate withdrawal quota: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return fmt.Errorf("settle affiliate withdrawal quota: frozen quota invariant violated")
		}

		_, err = txClient.ExecContext(txCtx, `
UPDATE user_affiliate_withdrawals
SET status = $2,
    reject_reason = NULLIF($3, ''),
    operator_id = $4,
    processed_at = NOW(),
    updated_at = NOW()
WHERE id = $1`, withdrawalID, status, rejectReason, operatorID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r.getAffiliateWithdrawal(ctx, withdrawalID)
}

func (r *affiliateRepository) getAffiliateWithdrawal(ctx context.Context, withdrawalID int64) (*service.AffiliateWithdrawal, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, affiliateWithdrawalSelect+" WHERE w.id = $1 LIMIT 1", withdrawalID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrAffiliateWithdrawalNotFound
	}
	return scanAffiliateWithdrawal(rows)
}
