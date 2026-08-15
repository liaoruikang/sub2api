package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type affiliateWithdrawalSettingRepo struct {
	values map[string]string
}

func (r *affiliateWithdrawalSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *affiliateWithdrawalSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}
func (r *affiliateWithdrawalSettingRepo) Set(context.Context, string, string) error { return nil }
func (r *affiliateWithdrawalSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}
func (r *affiliateWithdrawalSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return nil
}
func (r *affiliateWithdrawalSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *affiliateWithdrawalSettingRepo) Delete(context.Context, string) error { return nil }

type affiliateWithdrawalRepoStub struct {
	created *AffiliateWithdrawalCreateInput
}

func (r *affiliateWithdrawalRepoStub) CreateAffiliateWithdrawal(_ context.Context, input AffiliateWithdrawalCreateInput) (*AffiliateWithdrawal, error) {
	r.created = &input
	return &AffiliateWithdrawal{
		ID:                     1,
		RequestNo:              input.RequestNo,
		UserID:                 input.UserID,
		Amount:                 input.Amount,
		FeeRate:                input.FeeRate,
		FeeAmount:              input.FeeAmount,
		PayoutAmount:           input.PayoutAmount,
		AlipayAccountEncrypted: input.AlipayAccountEncrypted,
		AlipayAccountMasked:    input.AlipayAccountMasked,
		Status:                 AffiliateWithdrawalStatusPending,
	}, nil
}
func (r *affiliateWithdrawalRepoStub) ListUserAffiliateWithdrawals(context.Context, int64, AffiliateWithdrawalFilter) ([]AffiliateWithdrawal, int64, error) {
	return nil, 0, nil
}
func (r *affiliateWithdrawalRepoStub) ListAdminAffiliateWithdrawals(context.Context, AffiliateWithdrawalFilter) ([]AffiliateWithdrawal, int64, error) {
	return nil, 0, nil
}
func (r *affiliateWithdrawalRepoStub) ProcessAffiliateWithdrawal(context.Context, int64, int64, string, string) (*AffiliateWithdrawal, error) {
	return nil, errors.New("not implemented")
}

type affiliateWithdrawalEncryptor struct{}

func (affiliateWithdrawalEncryptor) Encrypt(value string) (string, error) { return "enc:" + value, nil }
func (affiliateWithdrawalEncryptor) Decrypt(value string) (string, error) {
	if len(value) < 4 || value[:4] != "enc:" {
		return "", errors.New("invalid ciphertext")
	}
	return value[4:], nil
}

func newAffiliateWithdrawalServiceForTest(values map[string]string, repo AffiliateWithdrawalRepository) *AffiliateService {
	return &AffiliateService{
		withdrawalRepo: repo,
		settingService: NewSettingService(&affiliateWithdrawalSettingRepo{values: values}, nil),
		encryptor:      affiliateWithdrawalEncryptor{},
	}
}

func TestCreateAffiliateWithdrawalSnapshotsPercentageFee(t *testing.T) {
	repo := &affiliateWithdrawalRepoStub{}
	svc := newAffiliateWithdrawalServiceForTest(map[string]string{
		SettingKeyAffiliateEnabled:             "true",
		SettingKeyAffiliateWithdrawalEnabled:   "true",
		SettingKeyAffiliateWithdrawalMinAmount: "10",
		SettingKeyAffiliateWithdrawalFeeRate:   "1",
	}, repo)

	item, err := svc.CreateAffiliateWithdrawal(context.Background(), 42, 100, "buyer@example.com")
	require.NoError(t, err)
	require.NotNil(t, repo.created)
	require.InDelta(t, 100, repo.created.Amount, 1e-9)
	require.InDelta(t, 1, repo.created.FeeRate, 1e-9)
	require.InDelta(t, 1, repo.created.FeeAmount, 1e-9)
	require.InDelta(t, 99, repo.created.PayoutAmount, 1e-9)
	require.Equal(t, "enc:buyer@example.com", repo.created.AlipayAccountEncrypted)
	require.NotEqual(t, "buyer@example.com", repo.created.AlipayAccountMasked)
	require.Empty(t, item.AlipayAccountEncrypted)
}

func TestCreateAffiliateWithdrawalRejectsBelowMinimum(t *testing.T) {
	repo := &affiliateWithdrawalRepoStub{}
	svc := newAffiliateWithdrawalServiceForTest(map[string]string{
		SettingKeyAffiliateEnabled:             "true",
		SettingKeyAffiliateWithdrawalEnabled:   "true",
		SettingKeyAffiliateWithdrawalMinAmount: "50",
	}, repo)

	_, err := svc.CreateAffiliateWithdrawal(context.Background(), 42, 49.99, "buyer@example.com")
	require.ErrorIs(t, err, ErrAffiliateWithdrawalMinimum)
	require.Nil(t, repo.created)
}

func TestAffiliateWithdrawalRequiresBothFeatureSwitches(t *testing.T) {
	repo := &affiliateWithdrawalRepoStub{}
	svc := newAffiliateWithdrawalServiceForTest(map[string]string{
		SettingKeyAffiliateEnabled:           "false",
		SettingKeyAffiliateWithdrawalEnabled: "true",
	}, repo)

	_, err := svc.CreateAffiliateWithdrawal(context.Background(), 42, 100, "buyer@example.com")
	require.ErrorIs(t, err, ErrAffiliateWithdrawalDisabled)
}
