package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// CouponRepository defines the interface for coupon operations.
type CouponRepository interface {
	// Coupon operations
	CreateCoupon(ctx context.Context, coupon *core.Coupon) error
	GetCoupon(ctx context.Context, id xid.ID) (*core.Coupon, error)
	GetCouponByCode(ctx context.Context, appID xid.ID, code string) (*core.Coupon, error)
	ListCoupons(ctx context.Context, appID xid.ID, status *core.CouponStatus, page, pageSize int) ([]*core.Coupon, int, error)
	UpdateCoupon(ctx context.Context, coupon *core.Coupon) error
	DeleteCoupon(ctx context.Context, id xid.ID) error
	IncrementRedemptions(ctx context.Context, id xid.ID) error

	// Redemption operations
	CreateRedemption(ctx context.Context, redemption *core.CouponRedemption) error
	GetRedemption(ctx context.Context, id xid.ID) (*core.CouponRedemption, error)
	GetRedemptionByCouponAndOrg(ctx context.Context, couponID, orgID xid.ID) (*core.CouponRedemption, error)
	ListRedemptions(ctx context.Context, couponID xid.ID, page, pageSize int) ([]*core.CouponRedemption, int, error)
	ListRedemptionsByOrg(ctx context.Context, orgID xid.ID) ([]*core.CouponRedemption, error)
	CountRedemptionsByOrg(ctx context.Context, couponID, orgID xid.ID) (int, error)

	// Promotion code operations
	CreatePromotionCode(ctx context.Context, code *core.PromotionCode) error
	GetPromotionCode(ctx context.Context, id xid.ID) (*core.PromotionCode, error)
	GetPromotionCodeByCode(ctx context.Context, appID xid.ID, code string) (*core.PromotionCode, error)
	ListPromotionCodes(ctx context.Context, couponID xid.ID, page, pageSize int) ([]*core.PromotionCode, int, error)
	UpdatePromotionCode(ctx context.Context, code *core.PromotionCode) error
	DeletePromotionCode(ctx context.Context, id xid.ID) error
	IncrementPromoRedemptions(ctx context.Context, id xid.ID) error
}

// couponRepository implements CouponRepository using Bun.
type couponRepository struct {
	db *bun.DB
}

// NewCouponRepository creates a new coupon repository.
func NewCouponRepository(db *bun.DB) CouponRepository {
	return &couponRepository{db: db}
}

// CreateCoupon creates a new coupon.
func (r *couponRepository) CreateCoupon(ctx context.Context, coupon *core.Coupon) error {
	model := couponToSchema(coupon)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)

	return err
}

// GetCoupon returns a coupon by ID.
func (r *couponRepository) GetCoupon(ctx context.Context, id xid.ID) (*core.Coupon, error) {
	var coupon schema.SubscriptionCoupon

	err := r.db.NewSelect().
		Model(&coupon).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return schemaToCoupon(&coupon), nil
}

// GetCouponByCode returns a coupon by code.
func (r *couponRepository) GetCouponByCode(ctx context.Context, appID xid.ID, code string) (*core.Coupon, error) {
	var coupon schema.SubscriptionCoupon

	err := r.db.NewSelect().
		Model(&coupon).
		Where("app_id = ?", appID).
		Where("code = ?", code).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return schemaToCoupon(&coupon), nil
}

// ListCoupons returns all coupons for an app.
func (r *couponRepository) ListCoupons(ctx context.Context, appID xid.ID, status *core.CouponStatus, page, pageSize int) ([]*core.Coupon, int, error) {
	var coupons []schema.SubscriptionCoupon

	query := r.db.NewSelect().
		Model(&coupons).
		Where("app_id = ?", appID)

	if status != nil {
		query = query.Where("status = ?", string(*status))
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	err = query.
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*core.Coupon, len(coupons))
	for i, c := range coupons {
		result[i] = schemaToCoupon(&c)
	}

	return result, count, nil
}

// UpdateCoupon updates a coupon.
func (r *couponRepository) UpdateCoupon(ctx context.Context, coupon *core.Coupon) error {
	model := couponToSchema(coupon)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)

	return err
}

// DeleteCoupon deletes a coupon.
func (r *couponRepository) DeleteCoupon(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionCoupon)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// IncrementRedemptions increments the redemption count.
func (r *couponRepository) IncrementRedemptions(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.SubscriptionCoupon)(nil)).
		Set("times_redeemed = times_redeemed + 1").
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// CreateRedemption creates a new redemption.
func (r *couponRepository) CreateRedemption(ctx context.Context, redemption *core.CouponRedemption) error {
	model := redemptionToSchema(redemption)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)

	return err
}

// GetRedemption returns a redemption by ID.
func (r *couponRepository) GetRedemption(ctx context.Context, id xid.ID) (*core.CouponRedemption, error) {
	var redemption schema.SubscriptionCouponRedemption

	err := r.db.NewSelect().
		Model(&redemption).
		Relation("Coupon").
		Where("scr.id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return schemaToRedemption(&redemption), nil
}

// GetRedemptionByCouponAndOrg returns a redemption by coupon and org.
func (r *couponRepository) GetRedemptionByCouponAndOrg(ctx context.Context, couponID, orgID xid.ID) (*core.CouponRedemption, error) {
	var redemption schema.SubscriptionCouponRedemption

	err := r.db.NewSelect().
		Model(&redemption).
		Where("coupon_id = ?", couponID).
		Where("organization_id = ?", orgID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return schemaToRedemption(&redemption), nil
}

// ListRedemptions returns all redemptions for a coupon.
func (r *couponRepository) ListRedemptions(ctx context.Context, couponID xid.ID, page, pageSize int) ([]*core.CouponRedemption, int, error) {
	var redemptions []schema.SubscriptionCouponRedemption

	query := r.db.NewSelect().
		Model(&redemptions).
		Where("coupon_id = ?", couponID)

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	err = query.
		Order("redeemed_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*core.CouponRedemption, len(redemptions))
	for i, r := range redemptions {
		result[i] = schemaToRedemption(&r)
	}

	return result, count, nil
}

// ListRedemptionsByOrg returns all redemptions for an organization.
func (r *couponRepository) ListRedemptionsByOrg(ctx context.Context, orgID xid.ID) ([]*core.CouponRedemption, error) {
	var redemptions []schema.SubscriptionCouponRedemption

	err := r.db.NewSelect().
		Model(&redemptions).
		Relation("Coupon").
		Where("organization_id = ?", orgID).
		Order("redeemed_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*core.CouponRedemption, len(redemptions))
	for i, r := range redemptions {
		result[i] = schemaToRedemption(&r)
	}

	return result, nil
}

// CountRedemptionsByOrg counts redemptions by coupon and org.
func (r *couponRepository) CountRedemptionsByOrg(ctx context.Context, couponID, orgID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.SubscriptionCouponRedemption)(nil)).
		Where("coupon_id = ?", couponID).
		Where("organization_id = ?", orgID).
		Count(ctx)

	return count, err
}

// CreatePromotionCode creates a new promotion code.
func (r *couponRepository) CreatePromotionCode(ctx context.Context, code *core.PromotionCode) error {
	model := promotionCodeToSchema(code)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)

	return err
}

// GetPromotionCode returns a promotion code by ID.
func (r *couponRepository) GetPromotionCode(ctx context.Context, id xid.ID) (*core.PromotionCode, error) {
	var code schema.SubscriptionPromotionCode

	err := r.db.NewSelect().
		Model(&code).
		Relation("Coupon").
		Where("spc.id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return schemaToPromotionCode(&code), nil
}

// GetPromotionCodeByCode returns a promotion code by code.
func (r *couponRepository) GetPromotionCodeByCode(ctx context.Context, appID xid.ID, code string) (*core.PromotionCode, error) {
	var promo schema.SubscriptionPromotionCode

	err := r.db.NewSelect().
		Model(&promo).
		Relation("Coupon").
		Where("app_id = ?", appID).
		Where("code = ?", code).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return schemaToPromotionCode(&promo), nil
}

// ListPromotionCodes returns all promotion codes for a coupon.
func (r *couponRepository) ListPromotionCodes(ctx context.Context, couponID xid.ID, page, pageSize int) ([]*core.PromotionCode, int, error) {
	var codes []schema.SubscriptionPromotionCode

	query := r.db.NewSelect().
		Model(&codes).
		Where("coupon_id = ?", couponID)

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	err = query.
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*core.PromotionCode, len(codes))
	for i, c := range codes {
		result[i] = schemaToPromotionCode(&c)
	}

	return result, count, nil
}

// UpdatePromotionCode updates a promotion code.
func (r *couponRepository) UpdatePromotionCode(ctx context.Context, code *core.PromotionCode) error {
	model := promotionCodeToSchema(code)
	model.UpdatedAt = time.Now()
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)

	return err
}

// DeletePromotionCode deletes a promotion code.
func (r *couponRepository) DeletePromotionCode(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.SubscriptionPromotionCode)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// IncrementPromoRedemptions increments the promo code redemption count.
func (r *couponRepository) IncrementPromoRedemptions(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewUpdate().
		Model((*schema.SubscriptionPromotionCode)(nil)).
		Set("times_redeemed = times_redeemed + 1").
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Helper functions

func schemaToCoupon(s *schema.SubscriptionCoupon) *core.Coupon {
	var metadata map[string]any
	if s.Metadata != "" {
		_ = json.Unmarshal([]byte(s.Metadata), &metadata)
	}

	return &core.Coupon{
		ID:                   s.ID,
		AppID:                s.AppID,
		Code:                 s.Code,
		Name:                 s.Name,
		Description:          s.Description,
		Type:                 core.CouponType(s.Type),
		Duration:             core.CouponDuration(s.Duration),
		Status:               core.CouponStatus(s.Status),
		PercentOff:           s.PercentOff,
		AmountOff:            s.AmountOff,
		Currency:             s.Currency,
		TrialDays:            s.TrialDays,
		FreeMonths:           s.FreeMonths,
		DurationMonths:       s.DurationMonths,
		MaxRedemptions:       s.MaxRedemptions,
		MaxRedemptionsPerOrg: s.MaxRedemptionsPerOrg,
		MinPurchaseAmount:    s.MinPurchaseAmount,
		ApplicablePlans:      s.ApplicablePlans,
		ApplicableAddOns:     s.ApplicableAddOns,
		FirstPurchaseOnly:    s.FirstPurchaseOnly,
		ValidFrom:            s.ValidFrom,
		ValidUntil:           s.ValidUntil,
		TimesRedeemed:        s.TimesRedeemed,
		ProviderCouponID:     s.ProviderCouponID,
		Metadata:             metadata,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

func couponToSchema(c *core.Coupon) *schema.SubscriptionCoupon {
	metadata := ""

	if c.Metadata != nil {
		metadataBytes, _ := json.Marshal(c.Metadata)
		metadata = string(metadataBytes)
	}

	return &schema.SubscriptionCoupon{
		ID:                   c.ID,
		AppID:                c.AppID,
		Code:                 c.Code,
		Name:                 c.Name,
		Description:          c.Description,
		Type:                 string(c.Type),
		Duration:             string(c.Duration),
		Status:               string(c.Status),
		PercentOff:           c.PercentOff,
		AmountOff:            c.AmountOff,
		Currency:             c.Currency,
		TrialDays:            c.TrialDays,
		FreeMonths:           c.FreeMonths,
		DurationMonths:       c.DurationMonths,
		MaxRedemptions:       c.MaxRedemptions,
		MaxRedemptionsPerOrg: c.MaxRedemptionsPerOrg,
		MinPurchaseAmount:    c.MinPurchaseAmount,
		ApplicablePlans:      c.ApplicablePlans,
		ApplicableAddOns:     c.ApplicableAddOns,
		FirstPurchaseOnly:    c.FirstPurchaseOnly,
		ValidFrom:            c.ValidFrom,
		ValidUntil:           c.ValidUntil,
		TimesRedeemed:        c.TimesRedeemed,
		ProviderCouponID:     c.ProviderCouponID,
		Metadata:             metadata,
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
	}
}

func schemaToRedemption(s *schema.SubscriptionCouponRedemption) *core.CouponRedemption {
	return &core.CouponRedemption{
		ID:             s.ID,
		AppID:          s.AppID,
		CouponID:       s.CouponID,
		OrganizationID: s.OrganizationID,
		SubscriptionID: s.SubscriptionID,
		DiscountType:   core.CouponType(s.DiscountType),
		DiscountAmount: s.DiscountAmount,
		Currency:       s.Currency,
		RedeemedAt:     s.RedeemedAt,
		ExpiresAt:      s.ExpiresAt,
	}
}

func redemptionToSchema(r *core.CouponRedemption) *schema.SubscriptionCouponRedemption {
	return &schema.SubscriptionCouponRedemption{
		ID:             r.ID,
		AppID:          r.AppID,
		CouponID:       r.CouponID,
		OrganizationID: r.OrganizationID,
		SubscriptionID: r.SubscriptionID,
		DiscountType:   string(r.DiscountType),
		DiscountAmount: r.DiscountAmount,
		Currency:       r.Currency,
		RedeemedAt:     r.RedeemedAt,
		ExpiresAt:      r.ExpiresAt,
	}
}

func schemaToPromotionCode(s *schema.SubscriptionPromotionCode) *core.PromotionCode {
	var coupon *core.Coupon
	if s.Coupon != nil {
		coupon = schemaToCoupon(s.Coupon)
	}

	return &core.PromotionCode{
		ID:              s.ID,
		AppID:           s.AppID,
		CouponID:        s.CouponID,
		Coupon:          coupon,
		Code:            s.Code,
		IsActive:        s.IsActive,
		MaxRedemptions:  s.MaxRedemptions,
		ValidFrom:       s.ValidFrom,
		ValidUntil:      s.ValidUntil,
		RestrictToOrgs:  s.RestrictToOrgs,
		FirstTimeOnly:   s.FirstTimeOnly,
		TimesRedeemed:   s.TimesRedeemed,
		ProviderPromoID: s.ProviderPromoID,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func promotionCodeToSchema(c *core.PromotionCode) *schema.SubscriptionPromotionCode {
	return &schema.SubscriptionPromotionCode{
		ID:              c.ID,
		AppID:           c.AppID,
		CouponID:        c.CouponID,
		Code:            c.Code,
		IsActive:        c.IsActive,
		MaxRedemptions:  c.MaxRedemptions,
		ValidFrom:       c.ValidFrom,
		ValidUntil:      c.ValidUntil,
		RestrictToOrgs:  c.RestrictToOrgs,
		FirstTimeOnly:   c.FirstTimeOnly,
		TimesRedeemed:   c.TimesRedeemed,
		ProviderPromoID: c.ProviderPromoID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}
