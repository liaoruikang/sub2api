package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const AffiliateWithdrawalAccountMaxCount = 10

var (
	ErrAffiliateWithdrawalAccountNotFound = infraerrors.NotFound("AFFILIATE_WITHDRAWAL_ACCOUNT_NOT_FOUND", "withdrawal account not found")
	ErrAffiliateWithdrawalAccountLimit    = infraerrors.Conflict("AFFILIATE_WITHDRAWAL_ACCOUNT_LIMIT", "withdrawal account limit reached")
)

type AffiliateWithdrawalAccount struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	AccountType      string    `json:"account_type"`
	AccountEncrypted string    `json:"-"`
	AccountMasked    string    `json:"account_masked"`
	IsDefault        bool      `json:"is_default"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AffiliateWithdrawalAccountCreateInput struct {
	UserID           int64
	AccountEncrypted string
	AccountMasked    string
}

type AffiliateWithdrawalAccountUpdateInput struct {
	ID               int64
	UserID           int64
	AccountEncrypted string
	AccountMasked    string
}

func (s *AffiliateService) ListAffiliateWithdrawalAccounts(ctx context.Context, userID int64) ([]AffiliateWithdrawalAccount, error) {
	if s == nil || s.withdrawalAccountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal account service unavailable")
	}
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	items, err := s.withdrawalAccountRepo.ListAffiliateWithdrawalAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].AccountEncrypted = ""
	}
	return items, nil
}

func (s *AffiliateService) CreateAffiliateWithdrawalAccount(ctx context.Context, userID int64, rawAccount string) (*AffiliateWithdrawalAccount, error) {
	if s == nil || s.withdrawalAccountRepo == nil || s.encryptor == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal account service unavailable")
	}
	if userID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	_, encrypted, masked, err := s.protectAffiliateWithdrawalAccount(rawAccount)
	if err != nil {
		return nil, err
	}
	item, err := s.withdrawalAccountRepo.CreateAffiliateWithdrawalAccount(ctx, AffiliateWithdrawalAccountCreateInput{
		UserID:           userID,
		AccountEncrypted: encrypted,
		AccountMasked:    masked,
	})
	if err != nil {
		return nil, err
	}
	item.AccountEncrypted = ""
	return item, nil
}

func (s *AffiliateService) UpdateAffiliateWithdrawalAccount(ctx context.Context, userID, accountID int64, rawAccount string) (*AffiliateWithdrawalAccount, error) {
	if s == nil || s.withdrawalAccountRepo == nil || s.encryptor == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal account service unavailable")
	}
	if userID <= 0 || accountID <= 0 {
		return nil, ErrAffiliateWithdrawalAccountNotFound
	}
	_, encrypted, masked, err := s.protectAffiliateWithdrawalAccount(rawAccount)
	if err != nil {
		return nil, err
	}
	item, err := s.withdrawalAccountRepo.UpdateAffiliateWithdrawalAccount(ctx, AffiliateWithdrawalAccountUpdateInput{
		ID:               accountID,
		UserID:           userID,
		AccountEncrypted: encrypted,
		AccountMasked:    masked,
	})
	if err != nil {
		return nil, err
	}
	item.AccountEncrypted = ""
	return item, nil
}

func (s *AffiliateService) SetDefaultAffiliateWithdrawalAccount(ctx context.Context, userID, accountID int64) (*AffiliateWithdrawalAccount, error) {
	if s == nil || s.withdrawalAccountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal account service unavailable")
	}
	if userID <= 0 || accountID <= 0 {
		return nil, ErrAffiliateWithdrawalAccountNotFound
	}
	item, err := s.withdrawalAccountRepo.SetDefaultAffiliateWithdrawalAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	item.AccountEncrypted = ""
	return item, nil
}

func (s *AffiliateService) DeleteAffiliateWithdrawalAccount(ctx context.Context, userID, accountID int64) error {
	if s == nil || s.withdrawalAccountRepo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal account service unavailable")
	}
	if userID <= 0 || accountID <= 0 {
		return ErrAffiliateWithdrawalAccountNotFound
	}
	return s.withdrawalAccountRepo.DeleteAffiliateWithdrawalAccount(ctx, userID, accountID)
}

func (s *AffiliateService) protectAffiliateWithdrawalAccount(rawAccount string) (string, string, string, error) {
	account := normalizeAffiliateAlipayAccount(rawAccount)
	if !validAffiliateAlipayAccount(account) {
		return "", "", "", ErrAffiliateWithdrawalAccount
	}
	encrypted, err := s.encryptor.Encrypt(account)
	if err != nil {
		return "", "", "", infraerrors.ServiceUnavailable("AFFILIATE_WITHDRAWAL_ENCRYPT_FAILED", "failed to protect Alipay account")
	}
	return account, encrypted, maskAlipayAccount(account), nil
}
