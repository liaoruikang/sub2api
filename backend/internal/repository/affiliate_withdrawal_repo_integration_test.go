//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAffiliateWithdrawalRepository_FreezesAndSettlesGrossAmount(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	baseRepo := NewAffiliateRepository(client, integrationDB)
	repo, ok := baseRepo.(service.AffiliateWithdrawalRepository)
	require.True(t, ok, "affiliate repository must expose withdrawal transactions")

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-withdrawal-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  1,
	})
	operator := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-withdrawal-admin-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleAdmin,
		Status:       service.StatusActive,
		Concurrency:  1,
	})

	_, err := client.ExecContext(txCtx, `
INSERT INTO user_affiliates (user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at)
VALUES ($1, $2, 100, 100, NOW(), NOW())`, user.ID, fmt.Sprintf("WD%010d", time.Now().UnixNano()%10_000_000_000))
	require.NoError(t, err)

	_, err = repo.CreateAffiliateWithdrawal(txCtx, service.AffiliateWithdrawalCreateInput{
		RequestNo:              fmt.Sprintf("AWX%d", time.Now().UnixNano()),
		UserID:                 user.ID,
		Amount:                 101,
		FeeRate:                1,
		FeeAmount:              1.01,
		PayoutAmount:           99.99,
		AlipayAccountEncrypted: "ciphertext",
		AlipayAccountMasked:    "buy****om",
	})
	require.ErrorIs(t, err, service.ErrAffiliateWithdrawalQuota)
	require.InDelta(t, 100, querySingleFloat(t, txCtx, client, "SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)
	require.InDelta(t, 0, querySingleFloat(t, txCtx, client, "SELECT aff_frozen_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)

	rejected, err := repo.CreateAffiliateWithdrawal(txCtx, service.AffiliateWithdrawalCreateInput{
		RequestNo:              fmt.Sprintf("AWR%d", time.Now().UnixNano()),
		UserID:                 user.ID,
		Amount:                 100,
		FeeRate:                1,
		FeeAmount:              1,
		PayoutAmount:           99,
		AlipayAccountEncrypted: "ciphertext",
		AlipayAccountMasked:    "buy****om",
	})
	require.NoError(t, err)
	require.InDelta(t, 0, querySingleFloat(t, txCtx, client, "SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)
	require.InDelta(t, 100, querySingleFloat(t, txCtx, client, "SELECT aff_frozen_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)

	_, err = repo.ProcessAffiliateWithdrawal(txCtx, rejected.ID, operator.ID, service.AffiliateWithdrawalStatusRejected, "account mismatch")
	require.NoError(t, err)
	require.InDelta(t, 100, querySingleFloat(t, txCtx, client, "SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)
	require.InDelta(t, 0, querySingleFloat(t, txCtx, client, "SELECT aff_frozen_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)
	_, err = repo.ProcessAffiliateWithdrawal(txCtx, rejected.ID, operator.ID, service.AffiliateWithdrawalStatusRejected, "again")
	require.ErrorIs(t, err, service.ErrAffiliateWithdrawalState)

	paid, err := repo.CreateAffiliateWithdrawal(txCtx, service.AffiliateWithdrawalCreateInput{
		RequestNo:              fmt.Sprintf("AWP%d", time.Now().UnixNano()),
		UserID:                 user.ID,
		Amount:                 60,
		FeeRate:                1,
		FeeAmount:              0.6,
		PayoutAmount:           59.4,
		AlipayAccountEncrypted: "ciphertext",
		AlipayAccountMasked:    "buy****om",
	})
	require.NoError(t, err)
	_, err = repo.ProcessAffiliateWithdrawal(txCtx, paid.ID, operator.ID, service.AffiliateWithdrawalStatusPaid, "")
	require.NoError(t, err)
	require.InDelta(t, 40, querySingleFloat(t, txCtx, client, "SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)
	require.InDelta(t, 0, querySingleFloat(t, txCtx, client, "SELECT aff_frozen_quota::double precision FROM user_affiliates WHERE user_id = $1", user.ID), 1e-9)
}

func TestAffiliateWithdrawalAccountRepository_EnforcesOwnershipAndDefaultPromotion(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()
	baseRepo := NewAffiliateRepository(client, integrationDB)
	repo, ok := baseRepo.(service.AffiliateWithdrawalAccountRepository)
	require.True(t, ok, "affiliate repository must expose withdrawal account management")

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-account-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  1,
	})
	otherUser := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-account-other-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  1,
	})

	first, err := repo.CreateAffiliateWithdrawalAccount(txCtx, service.AffiliateWithdrawalAccountCreateInput{
		UserID:           user.ID,
		AccountEncrypted: "ciphertext-1",
		AccountMasked:    "138****0000",
	})
	require.NoError(t, err)
	require.True(t, first.IsDefault)

	second, err := repo.CreateAffiliateWithdrawalAccount(txCtx, service.AffiliateWithdrawalAccountCreateInput{
		UserID:           user.ID,
		AccountEncrypted: "ciphertext-2",
		AccountMasked:    "buy****om",
	})
	require.NoError(t, err)
	require.False(t, second.IsDefault)

	_, err = repo.GetAffiliateWithdrawalAccount(txCtx, otherUser.ID, first.ID)
	require.ErrorIs(t, err, service.ErrAffiliateWithdrawalAccountNotFound)

	second, err = repo.SetDefaultAffiliateWithdrawalAccount(txCtx, user.ID, second.ID)
	require.NoError(t, err)
	require.True(t, second.IsDefault)
	require.NoError(t, repo.DeleteAffiliateWithdrawalAccount(txCtx, user.ID, second.ID))

	accounts, err := repo.ListAffiliateWithdrawalAccounts(txCtx, user.ID)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, first.ID, accounts[0].ID)
	require.True(t, accounts[0].IsDefault)

	for i := 1; i < service.AffiliateWithdrawalAccountMaxCount; i++ {
		_, err := repo.CreateAffiliateWithdrawalAccount(txCtx, service.AffiliateWithdrawalAccountCreateInput{
			UserID:           user.ID,
			AccountEncrypted: fmt.Sprintf("ciphertext-limit-%d", i),
			AccountMasked:    fmt.Sprintf("account-%d", i),
		})
		require.NoError(t, err)
	}
	_, err = repo.CreateAffiliateWithdrawalAccount(txCtx, service.AffiliateWithdrawalAccountCreateInput{
		UserID:           user.ID,
		AccountEncrypted: "ciphertext-over-limit",
		AccountMasked:    "over-limit",
	})
	require.ErrorIs(t, err, service.ErrAffiliateWithdrawalAccountLimit)
}
