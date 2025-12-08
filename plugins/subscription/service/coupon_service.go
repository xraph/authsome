package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/repository"
)

// CouponService handles coupon and discount operations
type CouponService struct {
	repo    repository.CouponRepository
	subRepo repository.SubscriptionRepository
}

// NewCouponService creates a new coupon service
func NewCouponService(repo repository.CouponRepository, subRepo repository.SubscriptionRepository) *CouponService {
	return &CouponService{
		repo:    repo,
		subRepo: subRepo,
	}
}

// CreateCoupon creates a new coupon
func (s *CouponService) CreateCoupon(ctx context.Context, appID xid.ID, req *core.CreateCouponRequest) (*core.Coupon, error) {
	// Validate coupon type and values
	if err := s.validateCouponRequest(req); err != nil {
		return nil, err
	}

	// Check for duplicate code
	existing, _ := s.repo.GetCouponByCode(ctx, appID, strings.ToUpper(req.Code))
	if existing != nil {
		return nil, core.ErrCouponCodeExists
	}

	coupon := &core.Coupon{
		ID:                   xid.New(),
		AppID:                appID,
		Code:                 strings.ToUpper(req.Code),
		Name:                 req.Name,
		Description:          req.Description,
		Type:                 req.Type,
		Duration:             req.Duration,
		Status:               core.CouponStatusActive,
		PercentOff:           req.PercentOff,
		AmountOff:            req.AmountOff,
		Currency:             req.Currency,
		TrialDays:            req.TrialDays,
		FreeMonths:           req.FreeMonths,
		DurationMonths:       req.DurationMonths,
		MaxRedemptions:       req.MaxRedemptions,
		MaxRedemptionsPerOrg: req.MaxRedemptionsPerOrg,
		MinPurchaseAmount:    req.MinPurchaseAmount,
		ApplicablePlans:      req.ApplicablePlans,
		ApplicableAddOns:     req.ApplicableAddOns,
		FirstPurchaseOnly:    req.FirstPurchaseOnly,
		ValidFrom:            req.ValidFrom,
		ValidUntil:           req.ValidUntil,
		TimesRedeemed:        0,
		Metadata:             req.Metadata,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if coupon.ValidFrom.IsZero() {
		coupon.ValidFrom = time.Now()
	}

	if err := s.repo.CreateCoupon(ctx, coupon); err != nil {
		return nil, fmt.Errorf("failed to create coupon: %w", err)
	}

	return coupon, nil
}

// GetCoupon returns a coupon by ID
func (s *CouponService) GetCoupon(ctx context.Context, id xid.ID) (*core.Coupon, error) {
	return s.repo.GetCoupon(ctx, id)
}

// GetCouponByCode returns a coupon by code
func (s *CouponService) GetCouponByCode(ctx context.Context, appID xid.ID, code string) (*core.Coupon, error) {
	return s.repo.GetCouponByCode(ctx, appID, strings.ToUpper(code))
}

// ListCoupons returns all coupons for an app
func (s *CouponService) ListCoupons(ctx context.Context, appID xid.ID, status *core.CouponStatus, page, pageSize int) ([]*core.Coupon, int, error) {
	return s.repo.ListCoupons(ctx, appID, status, page, pageSize)
}

// UpdateCoupon updates a coupon
func (s *CouponService) UpdateCoupon(ctx context.Context, id xid.ID, req *core.UpdateCouponRequest) (*core.Coupon, error) {
	coupon, err := s.repo.GetCoupon(ctx, id)
	if err != nil {
		return nil, err
	}
	if coupon == nil {
		return nil, core.ErrCouponNotFound
	}

	if req.Name != nil {
		coupon.Name = *req.Name
	}
	if req.Description != nil {
		coupon.Description = *req.Description
	}
	if req.MaxRedemptions != nil {
		coupon.MaxRedemptions = *req.MaxRedemptions
	}
	if req.MaxRedemptionsPerOrg != nil {
		coupon.MaxRedemptionsPerOrg = *req.MaxRedemptionsPerOrg
	}
	if len(req.ApplicablePlans) > 0 {
		coupon.ApplicablePlans = req.ApplicablePlans
	}
	if len(req.ApplicableAddOns) > 0 {
		coupon.ApplicableAddOns = req.ApplicableAddOns
	}
	if req.ValidUntil != nil {
		coupon.ValidUntil = req.ValidUntil
	}
	if req.Status != nil {
		coupon.Status = *req.Status
	}
	if req.Metadata != nil {
		coupon.Metadata = req.Metadata
	}

	coupon.UpdatedAt = time.Now()

	if err := s.repo.UpdateCoupon(ctx, coupon); err != nil {
		return nil, fmt.Errorf("failed to update coupon: %w", err)
	}

	return coupon, nil
}

// ArchiveCoupon archives a coupon
func (s *CouponService) ArchiveCoupon(ctx context.Context, id xid.ID) error {
	coupon, err := s.repo.GetCoupon(ctx, id)
	if err != nil {
		return err
	}
	if coupon == nil {
		return core.ErrCouponNotFound
	}

	coupon.Status = core.CouponStatusArchived
	coupon.UpdatedAt = time.Now()

	return s.repo.UpdateCoupon(ctx, coupon)
}

// ValidateCoupon validates a coupon code
func (s *CouponService) ValidateCoupon(ctx context.Context, appID xid.ID, req *core.ValidateCouponRequest) (*core.ValidateCouponResponse, error) {
	coupon, err := s.repo.GetCouponByCode(ctx, appID, strings.ToUpper(req.Code))
	if err != nil {
		return nil, err
	}
	if coupon == nil {
		return &core.ValidateCouponResponse{
			Valid:   false,
			Message: "Coupon not found",
			Errors:  []string{core.ErrCouponNotFound.Message},
		}, nil
	}

	errs := s.validateCouponUsage(ctx, coupon, req)
	if len(errs) > 0 {
		return &core.ValidateCouponResponse{
			Valid:   false,
			Coupon:  coupon,
			Message: errs[0],
			Errors:  errs,
		}, nil
	}

	discountAmount := coupon.CalculateDiscount(req.PurchaseAmount)

	return &core.ValidateCouponResponse{
		Valid:           true,
		Coupon:          coupon,
		DiscountAmount:  discountAmount,
		DiscountPercent: coupon.PercentOff,
		Message:         "Coupon is valid",
	}, nil
}

// RedeemCoupon redeems a coupon for a subscription
func (s *CouponService) RedeemCoupon(ctx context.Context, appID xid.ID, req *core.RedeemCouponRequest) (*core.CouponRedemption, error) {
	coupon, err := s.repo.GetCouponByCode(ctx, appID, strings.ToUpper(req.Code))
	if err != nil {
		return nil, err
	}
	if coupon == nil {
		return nil, core.ErrCouponNotFound
	}

	// Validate coupon can be redeemed
	validateReq := &core.ValidateCouponRequest{
		Code:           req.Code,
		OrganizationID: req.OrganizationID,
	}
	errs := s.validateCouponUsage(ctx, coupon, validateReq)
	if len(errs) > 0 {
		return nil, fmt.Errorf(errs[0])
	}

	// Get subscription to calculate discount
	sub, err := s.subRepo.FindByID(ctx, req.SubscriptionID)
	if err != nil {
		return nil, fmt.Errorf("subscription not found")
	}

	discountAmount := coupon.CalculateDiscount(sub.Plan.BasePrice)

	// Create redemption record
	redemption := &core.CouponRedemption{
		ID:             xid.New(),
		AppID:          appID,
		CouponID:       coupon.ID,
		OrganizationID: req.OrganizationID,
		SubscriptionID: req.SubscriptionID,
		DiscountType:   coupon.Type,
		DiscountAmount: discountAmount,
		Currency:       coupon.Currency,
		RedeemedAt:     time.Now(),
	}

	// Set expiry based on duration
	if coupon.Duration == core.CouponDurationRepeating && coupon.DurationMonths > 0 {
		expiresAt := time.Now().AddDate(0, coupon.DurationMonths, 0)
		redemption.ExpiresAt = &expiresAt
	}

	if err := s.repo.CreateRedemption(ctx, redemption); err != nil {
		return nil, fmt.Errorf("failed to create redemption: %w", err)
	}

	// Increment redemption count
	if err := s.repo.IncrementRedemptions(ctx, coupon.ID); err != nil {
		// Log but don't fail
	}

	return redemption, nil
}

// ListRedemptions lists redemptions for a coupon
func (s *CouponService) ListRedemptions(ctx context.Context, couponID xid.ID, page, pageSize int) ([]*core.CouponRedemption, int, error) {
	return s.repo.ListRedemptions(ctx, couponID, page, pageSize)
}

// ListOrgRedemptions lists all redemptions for an organization
func (s *CouponService) ListOrgRedemptions(ctx context.Context, orgID xid.ID) ([]*core.CouponRedemption, error) {
	return s.repo.ListRedemptionsByOrg(ctx, orgID)
}

// CreatePromotionCode creates a promotion code for a coupon
func (s *CouponService) CreatePromotionCode(ctx context.Context, appID xid.ID, req *core.CreatePromotionCodeRequest) (*core.PromotionCode, error) {
	// Verify coupon exists
	coupon, err := s.repo.GetCoupon(ctx, req.CouponID)
	if err != nil {
		return nil, err
	}
	if coupon == nil {
		return nil, core.ErrCouponNotFound
	}

	code := &core.PromotionCode{
		ID:             xid.New(),
		AppID:          appID,
		CouponID:       req.CouponID,
		Code:           strings.ToUpper(req.Code),
		IsActive:       true,
		MaxRedemptions: req.MaxRedemptions,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
		RestrictToOrgs: req.RestrictToOrgs,
		FirstTimeOnly:  req.FirstTimeOnly,
		TimesRedeemed:  0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if code.ValidFrom.IsZero() {
		code.ValidFrom = time.Now()
	}

	if err := s.repo.CreatePromotionCode(ctx, code); err != nil {
		return nil, fmt.Errorf("failed to create promotion code: %w", err)
	}

	return code, nil
}

// ListPromotionCodes lists promotion codes for a coupon
func (s *CouponService) ListPromotionCodes(ctx context.Context, couponID xid.ID, page, pageSize int) ([]*core.PromotionCode, int, error) {
	return s.repo.ListPromotionCodes(ctx, couponID, page, pageSize)
}

// Helper methods

func (s *CouponService) validateCouponRequest(req *core.CreateCouponRequest) error {
	switch req.Type {
	case core.CouponTypePercentage:
		if req.PercentOff <= 0 || req.PercentOff > 100 {
			return fmt.Errorf("percent off must be between 0 and 100")
		}
	case core.CouponTypeFixedAmount:
		if req.AmountOff <= 0 {
			return fmt.Errorf("amount off must be positive")
		}
		if req.Currency == "" {
			return fmt.Errorf("currency is required for fixed amount coupons")
		}
	case core.CouponTypeTrialExtension:
		if req.TrialDays <= 0 {
			return fmt.Errorf("trial days must be positive")
		}
	case core.CouponTypeFreeMonths:
		if req.FreeMonths <= 0 {
			return fmt.Errorf("free months must be positive")
		}
	}

	if req.Duration == core.CouponDurationRepeating && req.DurationMonths <= 0 {
		return fmt.Errorf("duration months required for repeating coupons")
	}

	return nil
}

func (s *CouponService) validateCouponUsage(ctx context.Context, coupon *core.Coupon, req *core.ValidateCouponRequest) []string {
	var errs []string

	// Check if coupon is valid
	if !coupon.IsValid() {
		if coupon.Status != core.CouponStatusActive {
			errs = append(errs, core.ErrCouponExpired.Message)
		}
		now := time.Now()
		if now.Before(coupon.ValidFrom) {
			errs = append(errs, core.ErrCouponNotYetValid.Message)
		}
		if coupon.ValidUntil != nil && now.After(*coupon.ValidUntil) {
			errs = append(errs, core.ErrCouponExpired.Message)
		}
	}

	// Check max redemptions
	if coupon.MaxRedemptions > 0 && coupon.TimesRedeemed >= coupon.MaxRedemptions {
		errs = append(errs, core.ErrCouponMaxRedemptions.Message)
	}

	// Check org-specific limits
	if coupon.MaxRedemptionsPerOrg > 0 {
		count, _ := s.repo.CountRedemptionsByOrg(ctx, coupon.ID, req.OrganizationID)
		if count >= coupon.MaxRedemptionsPerOrg {
			errs = append(errs, core.ErrCouponAlreadyUsed.Message)
		}
	}

	// Check minimum purchase
	if coupon.MinPurchaseAmount > 0 && req.PurchaseAmount < coupon.MinPurchaseAmount {
		errs = append(errs, core.ErrCouponMinPurchase.Message)
	}

	// Check plan applicability
	if len(coupon.ApplicablePlans) > 0 && req.PlanSlug != "" {
		found := false
		for _, plan := range coupon.ApplicablePlans {
			if plan == req.PlanSlug {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, core.ErrCouponNotApplicable.Message)
		}
	}

	// Check first purchase only
	if coupon.FirstPurchaseOnly {
		existingRedemptions, _ := s.repo.ListRedemptionsByOrg(ctx, req.OrganizationID)
		if len(existingRedemptions) > 0 {
			errs = append(errs, core.ErrCouponFirstPurchase.Message)
		}
	}

	return errs
}
