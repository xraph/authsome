package core

import (
	"time"

	"github.com/rs/xid"
)

// AddOn represents an optional add-on that can be attached to subscriptions
type AddOn struct {
	ID              xid.ID          `json:"id"`
	AppID           xid.ID          `json:"appId"`           // Scoped to app
	Name            string          `json:"name"`            // Display name
	Slug            string          `json:"slug"`            // URL-safe identifier
	Description     string          `json:"description"`     // Marketing description
	BillingPattern  BillingPattern  `json:"billingPattern"`  // How the add-on is billed
	BillingInterval BillingInterval `json:"billingInterval"` // Billing frequency
	Price           int64           `json:"price"`           // Price in cents
	Currency        string          `json:"currency"`        // ISO 4217 currency code
	Features        []PlanFeature   `json:"features"`        // Features provided by this add-on
	PriceTiers      []PriceTier     `json:"priceTiers"`      // For usage-based add-ons
	TierMode        TierMode        `json:"tierMode"`        // How tiers are applied
	Metadata        map[string]any  `json:"metadata"`        // Custom metadata
	IsActive        bool            `json:"isActive"`        // Available for purchase
	IsPublic        bool            `json:"isPublic"`        // Visible in public listing
	DisplayOrder    int             `json:"displayOrder"`    // Display order
	ProviderPriceID string          `json:"providerPriceId"` // Stripe Price ID
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`

	// Constraints
	RequiresPlanIDs []xid.ID `json:"requiresPlanIds"` // Only available with these plans
	ExcludesPlanIDs []xid.ID `json:"excludesPlanIds"` // Not available with these plans
	MaxQuantity     int      `json:"maxQuantity"`     // Maximum quantity per subscription (0 = unlimited)
}

// NewAddOn creates a new AddOn with default values
func NewAddOn(appID xid.ID, name, slug string) *AddOn {
	now := time.Now()
	return &AddOn{
		ID:              xid.New(),
		AppID:           appID,
		Name:            name,
		Slug:            slug,
		BillingPattern:  BillingPatternFlat,
		BillingInterval: BillingIntervalMonthly,
		Currency:        DefaultCurrency,
		TierMode:        TierModeGraduated,
		Features:        []PlanFeature{},
		PriceTiers:      []PriceTier{},
		Metadata:        make(map[string]any),
		RequiresPlanIDs: []xid.ID{},
		ExcludesPlanIDs: []xid.ID{},
		IsActive:        true,
		IsPublic:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// GetFeature returns a feature by key, or nil if not found
func (a *AddOn) GetFeature(key string) *PlanFeature {
	for i := range a.Features {
		if a.Features[i].Key == key {
			return &a.Features[i]
		}
	}
	return nil
}

// IsAvailableForPlan checks if this add-on is available for a specific plan
func (a *AddOn) IsAvailableForPlan(planID xid.ID) bool {
	// Check exclusions first
	for _, excludedID := range a.ExcludesPlanIDs {
		if excludedID == planID {
			return false
		}
	}

	// If requires list is empty, available for all
	if len(a.RequiresPlanIDs) == 0 {
		return true
	}

	// Check if plan is in requires list
	for _, requiredID := range a.RequiresPlanIDs {
		if requiredID == planID {
			return true
		}
	}

	return false
}

// CalculatePrice calculates the price for a given quantity
func (a *AddOn) CalculatePrice(quantity int) int64 {
	if len(a.PriceTiers) == 0 {
		return a.Price * int64(quantity)
	}

	// Use tiered pricing logic similar to Plan
	var total int64
	remaining := int64(quantity)

	if a.TierMode == TierModeVolume {
		for _, tier := range a.PriceTiers {
			if tier.UpTo == -1 || int64(quantity) <= tier.UpTo {
				return tier.FlatAmount + (tier.UnitAmount * int64(quantity))
			}
		}
	}

	var previousUpTo int64 = 0
	for _, tier := range a.PriceTiers {
		if remaining <= 0 {
			break
		}

		tierRange := tier.UpTo - previousUpTo
		if tier.UpTo == -1 {
			total += tier.FlatAmount + (tier.UnitAmount * remaining)
			break
		}

		unitsInTier := min(remaining, tierRange)
		total += tier.FlatAmount + (tier.UnitAmount * unitsInTier)
		remaining -= unitsInTier
		previousUpTo = tier.UpTo
	}

	return total
}

// CreateAddOnRequest represents a request to create an add-on
type CreateAddOnRequest struct {
	Name            string          `json:"name" validate:"required,min=1,max=100"`
	Slug            string          `json:"slug" validate:"required,min=1,max=50,alphanum"`
	Description     string          `json:"description" validate:"max=1000"`
	BillingPattern  BillingPattern  `json:"billingPattern" validate:"required"`
	BillingInterval BillingInterval `json:"billingInterval" validate:"required"`
	Price           int64           `json:"price" validate:"min=0"`
	Currency        string          `json:"currency" validate:"len=3"`
	Features        []PlanFeature   `json:"features"`
	PriceTiers      []PriceTier     `json:"priceTiers"`
	TierMode        TierMode        `json:"tierMode"`
	Metadata        map[string]any  `json:"metadata"`
	IsActive        bool            `json:"isActive"`
	IsPublic        bool            `json:"isPublic"`
	DisplayOrder    int             `json:"displayOrder"`
	RequiresPlanIDs []xid.ID        `json:"requiresPlanIds"`
	ExcludesPlanIDs []xid.ID        `json:"excludesPlanIds"`
	MaxQuantity     int             `json:"maxQuantity"`
}

// UpdateAddOnRequest represents a request to update an add-on
type UpdateAddOnRequest struct {
	Name            *string        `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description     *string        `json:"description,omitempty" validate:"omitempty,max=1000"`
	Price           *int64         `json:"price,omitempty" validate:"omitempty,min=0"`
	Features        []PlanFeature  `json:"features,omitempty"`
	PriceTiers      []PriceTier    `json:"priceTiers,omitempty"`
	TierMode        *TierMode      `json:"tierMode,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	IsActive        *bool          `json:"isActive,omitempty"`
	IsPublic        *bool          `json:"isPublic,omitempty"`
	DisplayOrder    *int           `json:"displayOrder,omitempty"`
	RequiresPlanIDs []xid.ID       `json:"requiresPlanIds,omitempty"`
	ExcludesPlanIDs []xid.ID       `json:"excludesPlanIds,omitempty"`
	MaxQuantity     *int           `json:"maxQuantity,omitempty"`
}

// AttachAddOnRequest represents a request to attach an add-on to a subscription
type AttachAddOnRequest struct {
	SubscriptionID xid.ID `json:"subscriptionId" validate:"required"`
	AddOnID        xid.ID `json:"addOnId" validate:"required"`
	Quantity       int    `json:"quantity" validate:"min=1"`
}

