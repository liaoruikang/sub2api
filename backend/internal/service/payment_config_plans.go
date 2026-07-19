package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// normalizePlanCurrency validates and normalizes the display-only currency label.
// Empty means "no label" and is kept as-is so existing plans stay unchanged.
func normalizePlanCurrency(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	currency, err := payment.NormalizePaymentCurrency(raw)
	if err != nil {
		return "", infraerrors.BadRequest("PLAN_CURRENCY_INVALID", "currency must be a 3-letter ISO currency code")
	}
	return currency, nil
}

// validatePlanRequired checks that all required fields for a plan are provided.
func validatePlanRequired(name string, groupID int64, price float64, validityDays int, validityUnit string, originalPrice *float64, purchaseLimitCount int, ipPurchaseLimitCount int, stockCount int, firstPurchaseDiscountEnabled bool, firstPurchaseDiscountPrice *float64) error {
	if strings.TrimSpace(name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if groupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if validityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if strings.TrimSpace(validityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if originalPrice != nil && *originalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if purchaseLimitCount < 0 {
		return infraerrors.BadRequest("PLAN_PURCHASE_LIMIT_INVALID", "purchase limit count must be >= 0")
	}
	if ipPurchaseLimitCount < 0 {
		return infraerrors.BadRequest("PLAN_IP_PURCHASE_LIMIT_INVALID", "IP purchase limit count must be >= 0")
	}
	if stockCount < 0 {
		return infraerrors.BadRequest("PLAN_STOCK_INVALID", "stock count must be >= 0")
	}
	if firstPurchaseDiscountEnabled {
		if firstPurchaseDiscountPrice == nil || *firstPurchaseDiscountPrice <= 0 {
			return infraerrors.BadRequest("PLAN_FIRST_PURCHASE_DISCOUNT_INVALID", "first purchase discount price must be > 0 when enabled")
		}
		if *firstPurchaseDiscountPrice >= price {
			return infraerrors.BadRequest("PLAN_FIRST_PURCHASE_DISCOUNT_INVALID", "first purchase discount price must be less than price")
		}
	}
	return nil
}

// validatePlanPatch validates only the non-nil fields in a patch update.
func validatePlanPatch(req UpdatePlanRequest) error {
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return infraerrors.BadRequest("PLAN_NAME_REQUIRED", "plan name is required")
	}
	if req.GroupID != nil && *req.GroupID <= 0 {
		return infraerrors.BadRequest("PLAN_GROUP_REQUIRED", "group is required")
	}
	if req.Price != nil && *req.Price <= 0 {
		return infraerrors.BadRequest("PLAN_PRICE_INVALID", "price must be > 0")
	}
	if req.ValidityDays != nil && *req.ValidityDays <= 0 {
		return infraerrors.BadRequest("PLAN_VALIDITY_REQUIRED", "validity days must be > 0")
	}
	if req.ValidityUnit != nil && strings.TrimSpace(*req.ValidityUnit) == "" {
		return infraerrors.BadRequest("PLAN_VALIDITY_UNIT_REQUIRED", "validity unit is required")
	}
	if req.OriginalPrice != nil && *req.OriginalPrice < 0 {
		return infraerrors.BadRequest("PLAN_ORIGINAL_PRICE_INVALID", "original price must be >= 0")
	}
	if req.PurchaseLimitCount != nil && *req.PurchaseLimitCount < 0 {
		return infraerrors.BadRequest("PLAN_PURCHASE_LIMIT_INVALID", "purchase limit count must be >= 0")
	}
	if req.IPPurchaseLimitCount != nil && *req.IPPurchaseLimitCount < 0 {
		return infraerrors.BadRequest("PLAN_IP_PURCHASE_LIMIT_INVALID", "IP purchase limit count must be >= 0")
	}
	if req.StockCount != nil && *req.StockCount < 0 {
		return infraerrors.BadRequest("PLAN_STOCK_INVALID", "stock count must be >= 0")
	}
	if req.FirstPurchaseDiscountPrice != nil && *req.FirstPurchaseDiscountPrice <= 0 {
		return infraerrors.BadRequest("PLAN_FIRST_PURCHASE_DISCOUNT_INVALID", "first purchase discount price must be > 0")
	}
	return nil
}

// --- Plan CRUD ---

// PlanGroupInfo holds the group details needed for subscription plan display.
type PlanGroupInfo struct {
	Platform           string   `json:"platform"`
	Name               string   `json:"name"`
	RateMultiplier     float64  `json:"rate_multiplier"`
	PeakRateEnabled    bool     `json:"peak_rate_enabled"`
	PeakStart          string   `json:"peak_start"`
	PeakEnd            string   `json:"peak_end"`
	PeakRateMultiplier float64  `json:"peak_rate_multiplier"`
	DailyLimitUSD      *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD     *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD    *float64 `json:"monthly_limit_usd"`
	ModelScopes        []string `json:"supported_model_scopes"`
}

// GetGroupInfoMap returns a map of group_id → PlanGroupInfo for the given plans.
func (s *PaymentConfigService) GetGroupInfoMap(ctx context.Context, plans []*dbent.SubscriptionPlan) map[int64]PlanGroupInfo {
	ids := make([]int64, 0, len(plans))
	seen := make(map[int64]bool)
	for _, p := range plans {
		if !seen[p.GroupID] {
			seen[p.GroupID] = true
			ids = append(ids, p.GroupID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	groups, err := s.entClient.Group.Query().Where(group.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil
	}
	m := make(map[int64]PlanGroupInfo, len(groups))
	for _, g := range groups {
		m[int64(g.ID)] = PlanGroupInfo{
			Platform:           g.Platform,
			Name:               g.Name,
			RateMultiplier:     g.RateMultiplier,
			PeakRateEnabled:    g.PeakRateEnabled,
			PeakStart:          g.PeakStart,
			PeakEnd:            g.PeakEnd,
			PeakRateMultiplier: g.PeakRateMultiplier,
			DailyLimitUSD:      g.DailyLimitUsd,
			WeeklyLimitUSD:     g.WeeklyLimitUsd,
			MonthlyLimitUSD:    g.MonthlyLimitUsd,
			ModelScopes:        g.SupportedModelScopes,
		}
	}
	return m
}

func (s *PaymentConfigService) ListPlans(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	plans, err := s.entClient.SubscriptionPlan.Query().Order(subscriptionplan.BySortOrder()).All(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.hydratePlanStockCounts(ctx, plans); err != nil {
		return nil, err
	}
	applyCurrentPlanSaleState(plans)
	return plans, nil
}

func (s *PaymentConfigService) ListPlansForSale(ctx context.Context) ([]*dbent.SubscriptionPlan, error) {
	now := time.Now()
	plans, err := s.entClient.SubscriptionPlan.Query().Where(
		subscriptionplan.ForSaleEQ(true),
		subscriptionplan.Or(
			subscriptionplan.OffSaleAtIsNil(),
			subscriptionplan.OffSaleAtGT(now),
		),
	).Order(subscriptionplan.BySortOrder()).All(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.hydratePlanStockCounts(ctx, plans); err != nil {
		return nil, err
	}
	return plans, nil
}

func IsSubscriptionPlanCurrentlyForSale(plan *dbent.SubscriptionPlan) bool {
	if plan == nil || !plan.ForSale {
		return false
	}
	return plan.OffSaleAt == nil || time.Now().Before(*plan.OffSaleAt)
}

func applyCurrentPlanSaleState(plans []*dbent.SubscriptionPlan) {
	now := time.Now()
	for _, plan := range plans {
		if plan.OffSaleAt != nil && !now.Before(*plan.OffSaleAt) {
			plan.ForSale = false
		}
	}
}

func shouldRefreshPlanListedAt(existing *dbent.SubscriptionPlan, req UpdatePlanRequest) bool {
	if existing == nil {
		return false
	}
	forSale := existing.ForSale
	if req.ForSale != nil {
		forSale = *req.ForSale
	}
	if !forSale {
		return false
	}
	newUserOnly := existing.NewUserOnly
	if req.NewUserOnly != nil {
		newUserOnly = *req.NewUserOnly
	}
	if newUserOnly && (existing.ListedAt == nil || (req.ListedAt.Set && !req.ListedAt.Valid)) {
		return true
	}
	return req.ForSale != nil && *req.ForSale && (!existing.ForSale || (existing.OffSaleAt != nil && !time.Now().Before(*existing.OffSaleAt)))
}

func (s *PaymentConfigService) CreatePlan(ctx context.Context, req CreatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanRequired(req.Name, req.GroupID, req.Price, req.ValidityDays, req.ValidityUnit, req.OriginalPrice, req.PurchaseLimitCount, req.IPPurchaseLimitCount, req.StockCount, req.FirstPurchaseDiscountEnabled, req.FirstPurchaseDiscountPrice); err != nil {
		return nil, err
	}
	currency, err := normalizePlanCurrency(req.Currency)
	if err != nil {
		return nil, err
	}
	b := s.entClient.SubscriptionPlan.Create().
		SetGroupID(req.GroupID).SetName(req.Name).SetDescription(req.Description).
		SetPrice(req.Price).SetCurrency(currency).SetValidityDays(req.ValidityDays).SetValidityUnit(req.ValidityUnit).
		SetFeatures(req.Features).SetProductName(req.ProductName).
		SetForSale(req.ForSale).SetNewUserOnly(req.NewUserOnly).SetPurchaseLimitCount(req.PurchaseLimitCount).
		SetIPPurchaseLimitCount(req.IPPurchaseLimitCount).
		SetFirstPurchaseDiscountEnabled(req.FirstPurchaseDiscountEnabled).
		SetSortOrder(req.SortOrder)
	if req.OriginalPrice != nil {
		b.SetOriginalPrice(*req.OriginalPrice)
	}
	if req.ListedAt != nil {
		b.SetListedAt(*req.ListedAt)
	} else if req.ForSale {
		b.SetListedAt(time.Now())
	}
	if req.OffSaleAt != nil {
		b.SetOffSaleAt(*req.OffSaleAt)
	}
	if req.FirstPurchaseDiscountEnabled && req.FirstPurchaseDiscountPrice != nil {
		b.SetFirstPurchaseDiscountPrice(*req.FirstPurchaseDiscountPrice)
	}
	plan, err := b.Save(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.setPlanStockCount(ctx, plan.ID, req.StockCount); err != nil {
		return nil, err
	}
	plan.StockCount = req.StockCount
	return plan, nil
}

// UpdatePlan updates a subscription plan by ID (patch semantics).
// NOTE: This function exceeds 30 lines due to per-field nil-check patch update boilerplate
// plus a validation guard for non-nil fields.
func (s *PaymentConfigService) UpdatePlan(ctx context.Context, id int64, req UpdatePlanRequest) (*dbent.SubscriptionPlan, error) {
	if err := validatePlanPatch(req); err != nil {
		return nil, err
	}
	var existing *dbent.SubscriptionPlan
	needsExisting := req.ForSale != nil || req.NewUserOnly != nil || req.ListedAt.Set
	if needsExisting {
		plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
		if err != nil {
			return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
		}
		existing = plan
	}
	u := s.entClient.SubscriptionPlan.UpdateOneID(id)
	if req.GroupID != nil {
		u.SetGroupID(*req.GroupID)
	}
	if req.Name != nil {
		u.SetName(*req.Name)
	}
	if req.Description != nil {
		u.SetDescription(*req.Description)
	}
	if req.Price != nil {
		u.SetPrice(*req.Price)
	}
	if req.OriginalPrice != nil {
		u.SetOriginalPrice(*req.OriginalPrice)
	}
	if req.Currency != nil {
		currency, err := normalizePlanCurrency(*req.Currency)
		if err != nil {
			return nil, err
		}
		u.SetCurrency(currency)
	}
	if req.ValidityDays != nil {
		u.SetValidityDays(*req.ValidityDays)
	}
	if req.ValidityUnit != nil {
		u.SetValidityUnit(*req.ValidityUnit)
	}
	if req.Features != nil {
		u.SetFeatures(*req.Features)
	}
	if req.ProductName != nil {
		u.SetProductName(*req.ProductName)
	}
	if req.ForSale != nil {
		u.SetForSale(*req.ForSale)
		if existing != nil && existing.OffSaleAt != nil && !time.Now().Before(*existing.OffSaleAt) {
			u.ClearOffSaleAt()
		}
	}
	if req.NewUserOnly != nil {
		u.SetNewUserOnly(*req.NewUserOnly)
	}
	refreshListedAt := shouldRefreshPlanListedAt(existing, req)
	if req.ListedAt.Set {
		if req.ListedAt.Valid {
			u.SetListedAt(req.ListedAt.Time)
		} else if refreshListedAt {
			u.SetListedAt(time.Now())
		} else {
			u.ClearListedAt()
		}
	} else if refreshListedAt {
		u.SetListedAt(time.Now())
	}
	if req.OffSaleAt.Set {
		if req.OffSaleAt.Valid {
			u.SetOffSaleAt(req.OffSaleAt.Time)
		} else {
			u.ClearOffSaleAt()
		}
	}
	if req.PurchaseLimitCount != nil {
		u.SetPurchaseLimitCount(*req.PurchaseLimitCount)
	}
	if req.IPPurchaseLimitCount != nil {
		u.SetIPPurchaseLimitCount(*req.IPPurchaseLimitCount)
	}
	if req.FirstPurchaseDiscountEnabled != nil {
		u.SetFirstPurchaseDiscountEnabled(*req.FirstPurchaseDiscountEnabled)
		if !*req.FirstPurchaseDiscountEnabled {
			u.ClearFirstPurchaseDiscountPrice()
		}
	}
	if req.FirstPurchaseDiscountPrice != nil {
		u.SetFirstPurchaseDiscountPrice(*req.FirstPurchaseDiscountPrice)
	}
	if req.SortOrder != nil {
		u.SetSortOrder(*req.SortOrder)
	}
	plan, err := u.Save(ctx)
	if err != nil {
		return nil, err
	}
	if req.StockCount != nil {
		if err := s.setPlanStockCount(ctx, id, *req.StockCount); err != nil {
			return nil, err
		}
		plan.StockCount = *req.StockCount
	}
	return plan, nil
}

func (s *PaymentConfigService) DeletePlan(ctx context.Context, id int64) error {
	count, err := s.countPendingOrdersByPlan(ctx, id)
	if err != nil {
		return fmt.Errorf("check pending orders: %w", err)
	}
	if count > 0 {
		return infraerrors.Conflict("PENDING_ORDERS",
			fmt.Sprintf("this plan has %d in-progress orders and cannot be deleted — wait for orders to complete first", count))
	}
	return s.entClient.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
}

// GetPlan returns a subscription plan by ID.
func (s *PaymentConfigService) GetPlan(ctx context.Context, id int64) (*dbent.SubscriptionPlan, error) {
	plan, err := s.entClient.SubscriptionPlan.Get(ctx, id)
	if err != nil {
		return nil, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	stockCount, err := s.getPlanStockCount(ctx, id)
	if err == nil {
		plan.StockCount = stockCount
	}
	return plan, nil
}

func (s *PaymentConfigService) setPlanStockCount(ctx context.Context, planID int64, stockCount int) error {
	_, err := s.entClient.ExecContext(ctx, `UPDATE subscription_plans SET stock_count = $1 WHERE id = $2`, stockCount, planID)
	if err != nil {
		return fmt.Errorf("set plan stock count: %w", err)
	}
	return nil
}

func (s *PaymentConfigService) getPlanStockCount(ctx context.Context, planID int64) (int, error) {
	rows, err := s.entClient.QueryContext(ctx, `SELECT stock_count FROM subscription_plans WHERE id = $1`, planID)
	if err != nil {
		return 0, fmt.Errorf("query plan stock count: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, infraerrors.NotFound("PLAN_NOT_FOUND", "subscription plan not found")
	}
	var stockCount int
	if err := rows.Scan(&stockCount); err != nil {
		return 0, fmt.Errorf("scan plan stock count: %w", err)
	}
	return stockCount, rows.Err()
}

func (s *PaymentConfigService) hydratePlanStockCounts(ctx context.Context, plans []*dbent.SubscriptionPlan) error {
	for _, plan := range plans {
		stockCount, err := s.getPlanStockCount(ctx, plan.ID)
		if err != nil {
			return err
		}
		plan.StockCount = stockCount
	}
	return nil
}
