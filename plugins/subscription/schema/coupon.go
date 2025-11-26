package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SubscriptionCoupon represents a coupon in the database
type SubscriptionCoupon struct {
	bun.BaseModel `bun:"table:subscription_coupons,alias:scp"`

	ID                  xid.ID     `bun:"id,pk,type:char(20)"`
	AppID               xid.ID     `bun:"app_id,notnull,type:char(20)"`
	Code                string     `bun:"code,notnull,unique"`
	Name                string     `bun:"name,notnull"`
	Description         string     `bun:"description"`
	Type                string     `bun:"type,notnull"`     // percentage, fixed_amount, trial_extension, free_months
	Duration            string     `bun:"duration,notnull"` // once, repeating, forever
	Status              string     `bun:"status,notnull,default:'active'"`
	PercentOff          float64    `bun:"percent_off"`
	AmountOff           int64      `bun:"amount_off"`
	Currency            string     `bun:"currency"`
	TrialDays           int        `bun:"trial_days"`
	FreeMonths          int        `bun:"free_months"`
	DurationMonths      int        `bun:"duration_months"`
	MaxRedemptions      int        `bun:"max_redemptions"`
	MaxRedemptionsPerOrg int       `bun:"max_redemptions_per_org"`
	MinPurchaseAmount   int64      `bun:"min_purchase_amount"`
	ApplicablePlans     []string   `bun:"applicable_plans,array"`
	ApplicableAddOns    []string   `bun:"applicable_addons,array"`
	FirstPurchaseOnly   bool       `bun:"first_purchase_only,notnull,default:false"`
	ValidFrom           time.Time  `bun:"valid_from,notnull"`
	ValidUntil          *time.Time `bun:"valid_until"`
	TimesRedeemed       int        `bun:"times_redeemed,notnull,default:0"`
	ProviderCouponID    string     `bun:"provider_coupon_id"`
	Metadata            string     `bun:"metadata,type:jsonb"`
	CreatedAt           time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt           time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
}

// SubscriptionCouponRedemption represents a coupon redemption in the database
type SubscriptionCouponRedemption struct {
	bun.BaseModel `bun:"table:subscription_coupon_redemptions,alias:scr"`

	ID             xid.ID    `bun:"id,pk,type:char(20)"`
	AppID          xid.ID    `bun:"app_id,notnull,type:char(20)"`
	CouponID       xid.ID    `bun:"coupon_id,notnull,type:char(20)"`
	OrganizationID xid.ID    `bun:"organization_id,notnull,type:char(20)"`
	SubscriptionID xid.ID    `bun:"subscription_id,notnull,type:char(20)"`
	DiscountType   string    `bun:"discount_type,notnull"`
	DiscountAmount int64     `bun:"discount_amount,notnull"`
	Currency       string    `bun:"currency,notnull"`
	RedeemedAt     time.Time `bun:"redeemed_at,notnull,default:current_timestamp"`
	ExpiresAt      *time.Time `bun:"expires_at"`

	// Relations
	Coupon *SubscriptionCoupon `bun:"rel:belongs-to,join:coupon_id=id"`
}

// SubscriptionPromotionCode represents a promotion code in the database
type SubscriptionPromotionCode struct {
	bun.BaseModel `bun:"table:subscription_promotion_codes,alias:spc"`

	ID              xid.ID     `bun:"id,pk,type:char(20)"`
	AppID           xid.ID     `bun:"app_id,notnull,type:char(20)"`
	CouponID        xid.ID     `bun:"coupon_id,notnull,type:char(20)"`
	Code            string     `bun:"code,notnull,unique"`
	IsActive        bool       `bun:"is_active,notnull,default:true"`
	MaxRedemptions  int        `bun:"max_redemptions"`
	ValidFrom       time.Time  `bun:"valid_from,notnull"`
	ValidUntil      *time.Time `bun:"valid_until"`
	RestrictToOrgs  []string   `bun:"restrict_to_orgs,array"`
	FirstTimeOnly   bool       `bun:"first_time_only,notnull,default:false"`
	TimesRedeemed   int        `bun:"times_redeemed,notnull,default:0"`
	ProviderPromoID string     `bun:"provider_promo_id"`
	CreatedAt       time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:current_timestamp"`

	// Relations
	Coupon *SubscriptionCoupon `bun:"rel:belongs-to,join:coupon_id=id"`
}

