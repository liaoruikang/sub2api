package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const highestSchedulingSuppressionReasonMaxLen = 512

func buildHighestSchedulingSuppressionExtraUpdates(account *service.Account, errorMsg string, now time.Time) (map[string]any, []string, bool) {
	if account == nil || !account.IsHighestSchedulingModeConfigured() {
		return nil, nil, false
	}

	reason := truncateHighestSchedulingSuppressionReason(strings.TrimSpace(errorMsg))
	if reason == "" {
		reason = "account error"
	}

	nowText := now.UTC().Format(time.RFC3339)
	updates := map[string]any{
		service.AccountExtraHighestSchedulingSuppressedAt:     nowText,
		service.AccountExtraHighestSchedulingSuppressedReason: reason,
	}

	recoveryMinutes := account.GetHighestSchedulingRecoveryMinutes()
	if recoveryMinutes > 0 {
		updates[service.AccountExtraHighestSchedulingSuppressed] = false
		updates[service.AccountExtraHighestSchedulingSuppressedUntil] = now.UTC().Add(time.Duration(recoveryMinutes) * time.Minute).Format(time.RFC3339)
		return updates, nil, true
	}

	updates[service.AccountExtraHighestSchedulingSuppressed] = true
	return updates, []string{service.AccountExtraHighestSchedulingSuppressedUntil}, true
}

func truncateHighestSchedulingSuppressionReason(value string) string {
	runes := []rune(value)
	if len(runes) <= highestSchedulingSuppressionReasonMaxLen {
		return value
	}
	return string(runes[:highestSchedulingSuppressionReasonMaxLen])
}

func mergeHighestSchedulingSuppressionExtra(extra map[string]any, updates map[string]any, deleteKeys []string) map[string]any {
	out := copyJSONMap(extra)
	if out == nil {
		out = map[string]any{}
	}
	for _, key := range deleteKeys {
		delete(out, key)
	}
	for key, value := range updates {
		out[key] = value
	}
	return out
}

func (r *accountRepository) applyHighestSchedulingSuppressionForErrorStatus(ctx context.Context, ids []int64, reason string) error {
	if len(ids) == 0 {
		return nil
	}
	accounts, err := r.GetByIDs(ctx, ids)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, account := range accounts {
		updates, deleteKeys, ok := buildHighestSchedulingSuppressionExtraUpdates(account, reason, now)
		if !ok {
			continue
		}
		if err := r.applyHighestSchedulingSuppressionExtraUpdates(ctx, account.ID, updates, deleteKeys); err != nil {
			return err
		}
	}
	return nil
}

func (r *accountRepository) applyHighestSchedulingSuppressionExtraUpdates(ctx context.Context, id int64, updates map[string]any, deleteKeys []string) error {
	if len(updates) == 0 && len(deleteKeys) == 0 {
		return nil
	}

	payload, err := json.Marshal(updates)
	if err != nil {
		return err
	}

	args := make([]any, 0, len(deleteKeys)+2)
	args = append(args, string(payload))
	extraExpr := "COALESCE(extra, '{}'::jsonb) || $1::jsonb"
	for _, key := range deleteKeys {
		placeholder := itoa(len(args) + 1)
		extraExpr += " - $" + placeholder + "::text"
		args = append(args, key)
	}
	idPlaceholder := itoa(len(args) + 1)
	args = append(args, id)

	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx,
		"UPDATE accounts SET extra = "+extraExpr+", updated_at = NOW() WHERE id = $"+idPlaceholder+" AND deleted_at IS NULL",
		args...,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}
	return nil
}
