package core

import (
	"time"

	"github.com/rs/xid"
)

// CouponType represents the type of discount a coupon provides
type CouponType string

const (
	CouponTypePercentage     CouponType = "percentage"      // Percentage off (e.g., 20% off)
	CouponTypeFixedAmount    CouponType = "fixed_amount"    // Fixed amount off (e.g., $10 off)
	CouponTypeTrialExtension CouponType = "trial_extension" // Extend trial period
	CouponTypeFreeMonths     CouponType = "free_months"     // Free months
)

// CouponDuration defines how long a coupon applies
type CouponDuration string

const (
	CouponDurationOnce      CouponDuration = "once"      // Apply once
	CouponDurationRepeating CouponDuration = "repeating" // Apply for X months
	CouponDurationForever   CouponDuration = "forever"   // Apply forever
)

// CouponStatus represents the status of a coupon
type CouponStatus string

const (
	CouponStatusActive   CouponStatus = "active"
	CouponStatusExpired  CouponStatus = "expired"
	CouponStatusArchived CouponStatus = "archived"
)

// Coupon represents a discount coupon
type Coupon struct {
	ID          xid.ID         `json:"id"`
	AppID       xid.ID         `json:"appId"`
	Code        string         `json:"code"`        // Unique coupon code (e.g., "SUMMER2024")
	Name        string         `json:"name"`        // Display name
	Description string         `json:"description"` // Optional description
	Type        CouponType     `json:"type"`        // Type of discount
	Duration    CouponDuration `json:"duration"`    // How long it applies
	Status      CouponStatus   `json:"status"`

	// Discount values (depending on type)
	PercentOff float64 `json:"percentOff"` // Percentage off (0-100)
	AmountOff  int64   `json:"amountOff"`  // Fixed amount off (in smallest currency unit)
	Currency   string  `json:"currency"`   // Currency for fixed amount
	TrialDays  int     `json:"trialDays"`  // Additional trial days
	FreeMonths int     `json:"freeMonths"` // Number of free months

	// Duration settings
	DurationMonths int `json:"durationMonths"` // For repeating duration

	// Restrictions
	MaxRedemptions       int      `json:"maxRedemptions"`       // Max total redemptions (0 = unlimited)
	MaxRedemptionsPerOrg int      `json:"maxRedemptionsPerOrg"` // Max per organization
	MinPurchaseAmount    int64    `json:"minPurchaseAmount"`    // Minimum purchase amount
	ApplicablePlans      []string `json:"applicablePlans"`      // Plan slugs this applies to (empty = all)
	ApplicableAddOns     []string `json:"applicableAddOns"`     // Add-on slugs this applies to
	FirstPurchaseOnly    bool     `json:"firstPurchaseOnly"`    // Only for first purchase

	// Validity
	ValidFrom  time.Time  `json:"validFrom"`
	ValidUntil *time.Time `json:"validUntil"`

	// Usage tracking
	TimesRedeemed int `json:"timesRedeemed"` // Number of times redeemed

	// Provider integration
	ProviderCouponID string `json:"providerCouponId"` // Coupon ID in payment provider

	// Metadata
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// CouponRedemption tracks when a coupon is used
type CouponRedemption struct {
	ID             xid.ID `json:"id"`
	AppID          xid.ID `json:"appId"`
	CouponID       xid.ID `json:"couponId"`
	OrganizationID xid.ID `json:"organizationId"`
	SubscriptionID xid.ID `json:"subscriptionId"`

	// Applied discount
	DiscountType   CouponType `json:"discountType"`
	DiscountAmount int64      `json:"discountAmount"` // Actual amount discounted
	Currency       string     `json:"currency"`

	RedeemedAt time.Time  `json:"redeemedAt"`
	ExpiresAt  *time.Time `json:"expiresAt"` // When the discount stops applying
}

// PromotionCode represents a promotion code that references a coupon
// This allows multiple codes to reference the same underlying coupon
type PromotionCode struct {
	ID       xid.ID  `json:"id"`
	AppID    xid.ID  `json:"appId"`
	CouponID xid.ID  `json:"couponId"`
	Coupon   *Coupon `json:"coupon,omitempty"`
	Code     string  `json:"code"` // Unique promotion code
	IsActive bool    `json:"isActive"`

	// Restrictions (override coupon settings)
	MaxRedemptions int        `json:"maxRedemptions"` // Max for this specific code
	ValidFrom      time.Time  `json:"validFrom"`
	ValidUntil     *time.Time `json:"validUntil"`

	// Customer restrictions
	RestrictToOrgs []string `json:"restrictToOrgs"` // Org IDs that can use this
	FirstTimeOnly  bool     `json:"firstTimeOnly"`

	// Usage tracking
	TimesRedeemed int `json:"timesRedeemed"`

	// Provider integration
	ProviderPromoID string `json:"providerPromoId"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateCouponRequest is used to create a new coupon
type CreateCouponRequest struct {
	Code                 string                 `json:"code" validate:"required,min=3,max=50"`
	Name                 string                 `json:"name" validate:"required"`
	Description          string                 `json:"description"`
	Type                 CouponType             `json:"type" validate:"required"`
	Duration             CouponDuration         `json:"duration" validate:"required"`
	PercentOff           float64                `json:"percentOff"`
	AmountOff            int64                  `json:"amountOff"`
	Currency             string                 `json:"currency"`
	TrialDays            int                    `json:"trialDays"`
	FreeMonths           int                    `json:"freeMonths"`
	DurationMonths       int                    `json:"durationMonths"`
	MaxRedemptions       int                    `json:"maxRedemptions"`
	MaxRedemptionsPerOrg int                    `json:"maxRedemptionsPerOrg"`
	MinPurchaseAmount    int64                  `json:"minPurchaseAmount"`
	ApplicablePlans      []string               `json:"applicablePlans"`
	ApplicableAddOns     []string               `json:"applicableAddOns"`
	FirstPurchaseOnly    bool                   `json:"firstPurchaseOnly"`
	ValidFrom            time.Time              `json:"validFrom"`
	ValidUntil           *time.Time             `json:"validUntil"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// UpdateCouponRequest is used to update a coupon
type UpdateCouponRequest struct {
	Name                 *string                `json:"name"`
	Description          *string                `json:"description"`
	MaxRedemptions       *int                   `json:"maxRedemptions"`
	MaxRedemptionsPerOrg *int                   `json:"maxRedemptionsPerOrg"`
	ApplicablePlans      []string               `json:"applicablePlans"`
	ApplicableAddOns     []string               `json:"applicableAddOns"`
	ValidUntil           *time.Time             `json:"validUntil"`
	Status               *CouponStatus          `json:"status"`
	Metadata             map[string]interface{} `json:"metadata"`
}

// ValidateCouponRequest is used to validate a coupon code
type ValidateCouponRequest struct {
	Code           string `json:"code" validate:"required"`
	OrganizationID xid.ID `json:"organizationId" validate:"required"`
	PlanSlug       string `json:"planSlug"`
	AddOnSlug      string `json:"addOnSlug"`
	PurchaseAmount int64  `json:"purchaseAmount"`
}

// ValidateCouponResponse contains coupon validation result
type ValidateCouponResponse struct {
	Valid           bool     `json:"valid"`
	Coupon          *Coupon  `json:"coupon,omitempty"`
	DiscountAmount  int64    `json:"discountAmount"`
	DiscountPercent float64  `json:"discountPercent"`
	Message         string   `json:"message"`
	Errors          []string `json:"errors,omitempty"`
}

// RedeemCouponRequest is used to redeem a coupon
type RedeemCouponRequest struct {
	Code           string `json:"code" validate:"required"`
	OrganizationID xid.ID `json:"organizationId" validate:"required"`
	SubscriptionID xid.ID `json:"subscriptionId" validate:"required"`
}

// AppliedDiscount represents a discount applied to an invoice line item
type AppliedDiscount struct {
	CouponID       xid.ID     `json:"couponId"`
	CouponCode     string     `json:"couponCode"`
	Type           CouponType `json:"type"`
	OriginalAmount int64      `json:"originalAmount"`
	DiscountAmount int64      `json:"discountAmount"`
	FinalAmount    int64      `json:"finalAmount"`
	Description    string     `json:"description"`
}

// CreatePromotionCodeRequest is used to create a promotion code
type CreatePromotionCodeRequest struct {
	CouponID       xid.ID     `json:"couponId" validate:"required"`
	Code           string     `json:"code" validate:"required,min=3,max=50"`
	MaxRedemptions int        `json:"maxRedemptions"`
	ValidFrom      time.Time  `json:"validFrom"`
	ValidUntil     *time.Time `json:"validUntil"`
	RestrictToOrgs []string   `json:"restrictToOrgs"`
	FirstTimeOnly  bool       `json:"firstTimeOnly"`
}

// CouponError represents a coupon-related error
type CouponError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Common coupon errors
var (
	ErrCouponNotFound       = &CouponError{Code: "COUPON_NOT_FOUND", Message: "coupon not found"}
	ErrCouponExpired        = &CouponError{Code: "COUPON_EXPIRED", Message: "coupon has expired"}
	ErrCouponNotYetValid    = &CouponError{Code: "COUPON_NOT_YET_VALID", Message: "coupon is not yet valid"}
	ErrCouponMaxRedemptions = &CouponError{Code: "COUPON_MAX_REDEMPTIONS", Message: "coupon has reached maximum redemptions"}
	ErrCouponNotApplicable  = &CouponError{Code: "COUPON_NOT_APPLICABLE", Message: "coupon is not applicable to this plan"}
	ErrCouponMinPurchase    = &CouponError{Code: "COUPON_MIN_PURCHASE", Message: "purchase amount does not meet minimum"}
	ErrCouponAlreadyUsed    = &CouponError{Code: "COUPON_ALREADY_USED", Message: "coupon has already been used by this organization"}
	ErrCouponFirstPurchase  = &CouponError{Code: "COUPON_FIRST_PURCHASE", Message: "coupon is only valid for first purchase"}
	ErrCouponCodeExists     = &CouponError{Code: "COUPON_CODE_EXISTS", Message: "coupon code already exists"}
)

func (e *CouponError) Error() string {
	return e.Message
}

// IsValid checks if a coupon is currently valid (not considering usage limits)
func (c *Coupon) IsValid() bool {
	if c.Status != CouponStatusActive {
		return false
	}
	now := time.Now()
	if now.Before(c.ValidFrom) {
		return false
	}
	if c.ValidUntil != nil && now.After(*c.ValidUntil) {
		return false
	}
	return true
}

// CanRedeem checks if a coupon can be redeemed (considering usage limits)
func (c *Coupon) CanRedeem() bool {
	if !c.IsValid() {
		return false
	}
	if c.MaxRedemptions > 0 && c.TimesRedeemed >= c.MaxRedemptions {
		return false
	}
	return true
}

// CalculateDiscount calculates the discount amount for a given price
func (c *Coupon) CalculateDiscount(originalAmount int64) int64 {
	switch c.Type {
	case CouponTypePercentage:
		return int64(float64(originalAmount) * c.PercentOff / 100)
	case CouponTypeFixedAmount:
		if c.AmountOff > originalAmount {
			return originalAmount
		}
		return c.AmountOff
	default:
		return 0
	}
}
