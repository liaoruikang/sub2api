package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AffiliateWithdrawalStatusPending  = "pending"
	AffiliateWithdrawalStatusPaid     = "paid"
	AffiliateWithdrawalStatusRejected = "rejected"

	affiliateAlipayAccountMinLength = 5
	affiliateAlipayAccountMaxLength = 128
	affiliateRejectReasonMaxLength  = 500
)

var (
	ErrAffiliateWithdrawalDisabled = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_DISABLED", "affiliate withdrawal is disabled")
	ErrAffiliateWithdrawalAmount   = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_AMOUNT_INVALID", "invalid withdrawal amount")
	ErrAffiliateWithdrawalMinimum  = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_BELOW_MINIMUM", "withdrawal amount is below the minimum")
	ErrAffiliateWithdrawalAccount  = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_ACCOUNT_INVALID", "invalid Alipay account")
	ErrAffiliateWithdrawalQuota    = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_INSUFFICIENT_QUOTA", "insufficient affiliate quota")
	ErrAffiliateWithdrawalNotFound = infraerrors.NotFound("AFFILIATE_WITHDRAWAL_NOT_FOUND", "affiliate withdrawal not found")
	ErrAffiliateWithdrawalState    = infraerrors.Conflict("AFFILIATE_WITHDRAWAL_STATE_CONFLICT", "affiliate withdrawal has already been processed")
	ErrAffiliateWithdrawalReason   = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_REJECT_REASON_REQUIRED", "reject reason is required")
	ErrAffiliateWithdrawalStatus   = infraerrors.BadRequest("AFFILIATE_WITHDRAWAL_STATUS_INVALID", "invalid withdrawal status")
)

type AffiliateWithdrawalConfig struct {
	Enabled   bool    `json:"enabled"`
	MinAmount float64 `json:"min_amount"`
	FeeRate   float64 `json:"fee_rate"`
}

type AffiliateWithdrawalCreateInput struct {
	RequestNo              string
	UserID                 int64
	Amount                 float64
	FeeRate                float64
	FeeAmount              float64
	PayoutAmount           float64
	AlipayAccountEncrypted string
	AlipayAccountMasked    string
}

type AffiliateWithdrawalFilter struct {
	Search   string
	Status   string
	Page     int
	PageSize int
	StartAt  *time.Time
	EndAt    *time.Time
	SortBy   string
	SortDesc bool
}

type AffiliateWithdrawal struct {
	ID                     int64      `json:"id"`
	RequestNo              string     `json:"request_no"`
	UserID                 int64      `json:"user_id"`
	UserEmail              string     `json:"user_email,omitempty"`
	Username               string     `json:"username,omitempty"`
	Amount                 float64    `json:"amount"`
	FeeRate                float64    `json:"fee_rate"`
	FeeAmount              float64    `json:"fee_amount"`
	PayoutAmount           float64    `json:"payout_amount"`
	AlipayAccount          string     `json:"alipay_account,omitempty"`
	AlipayAccountEncrypted string     `json:"-"`
	AlipayAccountMasked    string     `json:"alipay_account_masked"`
	Status                 string     `json:"status"`
	RejectReason           string     `json:"reject_reason,omitempty"`
	OperatorID             *int64     `json:"operator_id,omitempty"`
	OperatorEmail          string     `json:"operator_email,omitempty"`
	ProcessedAt            *time.Time `json:"processed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (s *AffiliateService) affiliateWithdrawalConfig(ctx context.Context) AffiliateWithdrawalConfig {
	config := AffiliateWithdrawalConfig{
		Enabled:   false,
		MinAmount: AffiliateWithdrawalMinAmountDefault,
		FeeRate:   AffiliateWithdrawalFeeRateDefault,
	}
	if s == nil || s.settingService == nil {
		return config
	}
	config.Enabled = s.IsEnabled(ctx) && s.settingService.IsAffiliateWithdrawalEnabled(ctx)
	config.MinAmount = s.settingService.GetAffiliateWithdrawalMinAmount(ctx)
	config.FeeRate = s.settingService.GetAffiliateWithdrawalFeeRate(ctx)
	return config
}

func (s *AffiliateService) CreateAffiliateWithdrawal(ctx context.Context, userID int64, amount float64, withdrawalAccountID int64, rawAlipayAccount string) (*AffiliateWithdrawal, error) {
	if s == nil || s.withdrawalRepo == nil || s.encryptor == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal service unavailable")
	}
	config := s.affiliateWithdrawalConfig(ctx)
	if !config.Enabled {
		return nil, ErrAffiliateWithdrawalDisabled
	}
	amount = roundTo(amount, 8)
	if userID <= 0 || amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, ErrAffiliateWithdrawalAmount
	}
	if amount < config.MinAmount {
		return nil, ErrAffiliateWithdrawalMinimum
	}

	var accountEncrypted string
	var accountMasked string
	if withdrawalAccountID > 0 {
		if s.withdrawalAccountRepo == nil {
			return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal account service unavailable")
		}
		account, err := s.withdrawalAccountRepo.GetAffiliateWithdrawalAccount(ctx, userID, withdrawalAccountID)
		if err != nil {
			return nil, err
		}
		accountEncrypted = account.AccountEncrypted
		accountMasked = account.AccountMasked
	} else {
		_, encrypted, masked, err := s.protectAffiliateWithdrawalAccount(rawAlipayAccount)
		if err != nil {
			return nil, err
		}
		accountEncrypted = encrypted
		accountMasked = masked
	}
	feeAmount := roundTo(amount*(config.FeeRate/100), 8)
	payoutAmount := roundTo(amount-feeAmount, 8)
	if payoutAmount <= 0 {
		return nil, ErrAffiliateWithdrawalAmount
	}
	requestNo, err := generateAffiliateWithdrawalNo()
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("AFFILIATE_WITHDRAWAL_ID_FAILED", "failed to create withdrawal request")
	}

	item, err := s.withdrawalRepo.CreateAffiliateWithdrawal(ctx, AffiliateWithdrawalCreateInput{
		RequestNo:              requestNo,
		UserID:                 userID,
		Amount:                 amount,
		FeeRate:                config.FeeRate,
		FeeAmount:              feeAmount,
		PayoutAmount:           payoutAmount,
		AlipayAccountEncrypted: accountEncrypted,
		AlipayAccountMasked:    accountMasked,
	})
	if err != nil {
		return nil, err
	}
	item.AlipayAccountEncrypted = ""
	return item, nil
}

func (s *AffiliateService) ListUserAffiliateWithdrawals(ctx context.Context, userID int64, filter AffiliateWithdrawalFilter) ([]AffiliateWithdrawal, int64, error) {
	if s == nil || s.withdrawalRepo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate service unavailable")
	}
	if userID <= 0 {
		return nil, 0, infraerrors.BadRequest("INVALID_USER", "invalid user")
	}
	if err := normalizeAffiliateWithdrawalFilter(&filter); err != nil {
		return nil, 0, err
	}
	items, total, err := s.withdrawalRepo.ListUserAffiliateWithdrawals(ctx, userID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].AlipayAccountEncrypted = ""
		items[i].AlipayAccount = ""
	}
	return items, total, nil
}

func (s *AffiliateService) AdminListAffiliateWithdrawals(ctx context.Context, filter AffiliateWithdrawalFilter) ([]AffiliateWithdrawal, int64, error) {
	if s == nil || s.withdrawalRepo == nil || s.encryptor == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal service unavailable")
	}
	if err := normalizeAffiliateWithdrawalFilter(&filter); err != nil {
		return nil, 0, err
	}
	items, total, err := s.withdrawalRepo.ListAdminAffiliateWithdrawals(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		account, decryptErr := s.encryptor.Decrypt(items[i].AlipayAccountEncrypted)
		if decryptErr != nil {
			return nil, 0, infraerrors.ServiceUnavailable("AFFILIATE_WITHDRAWAL_DECRYPT_FAILED", "failed to read Alipay account")
		}
		items[i].AlipayAccount = account
		items[i].AlipayAccountEncrypted = ""
	}
	return items, total, nil
}

func (s *AffiliateService) AdminProcessAffiliateWithdrawal(ctx context.Context, withdrawalID, operatorID int64, status, rawRejectReason string) (*AffiliateWithdrawal, error) {
	if s == nil || s.withdrawalRepo == nil || s.encryptor == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "affiliate withdrawal service unavailable")
	}
	if withdrawalID <= 0 || operatorID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_REQUEST", "invalid withdrawal or operator")
	}
	rejectReason := strings.TrimSpace(rawRejectReason)
	if status == AffiliateWithdrawalStatusRejected {
		if rejectReason == "" || utf8.RuneCountInString(rejectReason) > affiliateRejectReasonMaxLength {
			return nil, ErrAffiliateWithdrawalReason
		}
	} else if status != AffiliateWithdrawalStatusPaid {
		return nil, ErrAffiliateWithdrawalStatus
	} else {
		rejectReason = ""
	}

	item, err := s.withdrawalRepo.ProcessAffiliateWithdrawal(ctx, withdrawalID, operatorID, status, rejectReason)
	if err != nil {
		return nil, err
	}
	// Settlement has already committed at this point. A legacy or damaged
	// ciphertext must not make a successful financial operation look failed.
	if account, decryptErr := s.encryptor.Decrypt(item.AlipayAccountEncrypted); decryptErr == nil {
		item.AlipayAccount = account
	}
	item.AlipayAccountEncrypted = ""
	return item, nil
}

func normalizeAffiliateWithdrawalFilter(filter *AffiliateWithdrawalFilter) error {
	if filter == nil {
		return nil
	}
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	if filter.Status == "all" {
		filter.Status = ""
	}
	if filter.Status != "" && filter.Status != AffiliateWithdrawalStatusPending && filter.Status != AffiliateWithdrawalStatusPaid && filter.Status != AffiliateWithdrawalStatusRejected {
		return ErrAffiliateWithdrawalStatus
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	return nil
}

func validAffiliateAlipayAccount(account string) bool {
	if !utf8.ValidString(account) {
		return false
	}
	length := utf8.RuneCountInString(account)
	if length < affiliateAlipayAccountMinLength || length > affiliateAlipayAccountMaxLength {
		return false
	}
	for _, r := range account {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func normalizeAffiliateAlipayAccount(account string) string {
	return strings.TrimSpace(account)
}

func maskAlipayAccount(account string) string {
	account = strings.TrimSpace(account)
	if at := strings.Index(account, "@"); at > 0 {
		return maskEmail(account)
	}
	runes := []rune(account)
	if len(runes) <= 4 {
		return "***"
	}
	prefix := 2
	if len(runes) >= 7 {
		prefix = 3
	}
	return string(runes[:prefix]) + "****" + string(runes[len(runes)-2:])
}

func generateAffiliateWithdrawalNo() (string, error) {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "AW" + strings.ToUpper(hex.EncodeToString(raw)), nil
}
