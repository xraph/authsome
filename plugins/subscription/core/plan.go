package core

import (
	"time"

	"github.com/rs/xid"
)

// Plan represents a subscription plan that organizations can subscribe to.
type Plan struct {
	ID              xid.ID          `json:"id"`
	AppID           xid.ID          `json:"appId"`           // Scoped to app
	Name            string          `json:"name"`            // Display name
	Slug            string          `json:"slug"`            // URL-safe identifier
	Description     string          `json:"description"`     // Marketing description
	Archived        bool            `json:"archived"`        // Whether the plan is archived
	BillingPattern  BillingPattern  `json:"billingPattern"`  // How the plan is billed
	BillingInterval BillingInterval `json:"billingInterval"` // Billing frequency
	BasePrice       int64           `json:"basePrice"`       // Price in cents
	Currency        string          `json:"currency"`        // ISO 4217 currency code
	TrialDays       int             `json:"trialDays"`       // Number of trial days (0 for no trial)
	Features        []PlanFeature   `json:"features"`        // Feature flags and limits
	PriceTiers      []PriceTier     `json:"priceTiers"`      // For tiered/usage pricing
	TierMode        TierMode        `json:"tierMode"`        // How tiers are applied
	Metadata        map[string]any  `json:"metadata"`        // Custom metadata
	IsActive        bool            `json:"isActive"`        // Can be subscribed to
	IsPublic        bool            `json:"isPublic"`        // Visible in public pricing
	DisplayOrder    int             `json:"displayOrder"`    // Order in pricing pages
	ProviderPlanID  string          `json:"providerPlanId"`  // Stripe Product ID
	ProviderPriceID string          `json:"providerPriceId"` // Stripe Price ID
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// PlanFeature represents a feature or limit included in a plan.
type PlanFeature struct {
	Key           string      `json:"key"`                     // Feature identifier (e.g., "max_members")
	Name          string      `json:"name"`                    // Display name
	Description   string      `json:"description"`             // Feature description
	Type          FeatureType `json:"type"`                    // boolean, limit, or unlimited
	Value         any         `json:"value"`                   // The feature value (true, 100, -1 for unlimited)
	Unit          string      `json:"unit,omitempty"`          // Unit of measurement (e.g., "seats", "GB", "API calls")
	IsHighlighted bool        `json:"isHighlighted,omitempty"` // Highlight in pricing comparison
}

// PriceTier represents a pricing tier for tiered or usage-based billing.
type PriceTier struct {
	UpTo       int64 `json:"upTo"`       // Upper bound (-1 for infinite)
	UnitAmount int64 `json:"unitAmount"` // Price per unit in cents
	FlatAmount int64 `json:"flatAmount"` // Flat fee for this tier in cents
}

// NewPlan creates a new Plan with default values.
func NewPlan(appID xid.ID, name, slug string) *Plan {
	now := time.Now()

	return &Plan{
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
		IsActive:        true,
		IsPublic:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// GetFeature returns a feature by key, or nil if not found.
func (p *Plan) GetFeature(key string) *PlanFeature {
	for i := range p.Features {
		if p.Features[i].Key == key {
			return &p.Features[i]
		}
	}

	return nil
}

// HasFeature checks if a plan has a specific feature enabled.
func (p *Plan) HasFeature(key string) bool {
	feature := p.GetFeature(key)
	if feature == nil {
		return false
	}

	switch feature.Type {
	case FeatureTypeBoolean:
		if val, ok := feature.Value.(bool); ok {
			return val
		}
	case FeatureTypeUnlimited:
		return true
	case FeatureTypeLimit:
		if val, ok := feature.Value.(float64); ok {
			return val > 0
		}

		if val, ok := feature.Value.(int); ok {
			return val > 0
		}
	}

	return false
}

// GetFeatureLimit returns the numeric limit for a feature, or -1 if unlimited.
func (p *Plan) GetFeatureLimit(key string) int64 {
	feature := p.GetFeature(key)
	if feature == nil {
		return 0
	}

	switch feature.Type {
	case FeatureTypeUnlimited:
		return -1 // Unlimited
	case FeatureTypeLimit:
		if val, ok := feature.Value.(float64); ok {
			return int64(val)
		}

		if val, ok := feature.Value.(int); ok {
			return int64(val)
		}

		if val, ok := feature.Value.(int64); ok {
			return val
		}
	}

	return 0
}

// SetFeature adds or updates a feature on the plan.
func (p *Plan) SetFeature(key, name string, featureType FeatureType, value any) {
	for i := range p.Features {
		if p.Features[i].Key == key {
			p.Features[i].Name = name
			p.Features[i].Type = featureType
			p.Features[i].Value = value

			return
		}
	}

	p.Features = append(p.Features, PlanFeature{
		Key:   key,
		Name:  name,
		Type:  featureType,
		Value: value,
	})
}

// AddPriceTier adds a pricing tier to the plan.
func (p *Plan) AddPriceTier(upTo, unitAmount, flatAmount int64) {
	p.PriceTiers = append(p.PriceTiers, PriceTier{
		UpTo:       upTo,
		UnitAmount: unitAmount,
		FlatAmount: flatAmount,
	})
}

// CalculateTieredPrice calculates the price for a given quantity using tiered pricing.
func (p *Plan) CalculateTieredPrice(quantity int64) int64 {
	if len(p.PriceTiers) == 0 {
		return p.BasePrice * quantity
	}

	var total int64

	remaining := quantity

	if p.TierMode == TierModeVolume {
		// Volume pricing: find the tier and apply to all units
		for _, tier := range p.PriceTiers {
			if tier.UpTo == -1 || quantity <= tier.UpTo {
				return tier.FlatAmount + (tier.UnitAmount * quantity)
			}
		}
	}

	// Graduated pricing: apply each tier's price to units in that tier
	var previousUpTo int64 = 0

	for _, tier := range p.PriceTiers {
		if remaining <= 0 {
			break
		}

		tierRange := tier.UpTo - previousUpTo
		if tier.UpTo == -1 {
			// Unlimited tier
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

// CreatePlanRequest represents a request to create a new plan.
type CreatePlanRequest struct {
	Name            string          `json:"name"            validate:"required,min=1,max=100"`
	Slug            string          `json:"slug"            validate:"required,min=1,max=50,alphanum"`
	Description     string          `json:"description"     validate:"max=1000"`
	BillingPattern  BillingPattern  `json:"billingPattern"  validate:"required"`
	BillingInterval BillingInterval `json:"billingInterval" validate:"required"`
	BasePrice       int64           `json:"basePrice"       validate:"min=0"`
	Currency        string          `json:"currency"        validate:"len=3"`
	TrialDays       int             `json:"trialDays"       validate:"min=0,max=365"`
	Features        []PlanFeature   `json:"features"`
	PriceTiers      []PriceTier     `json:"priceTiers"`
	TierMode        TierMode        `json:"tierMode"`
	Metadata        map[string]any  `json:"metadata"`
	IsActive        bool            `json:"isActive"`
	IsPublic        bool            `json:"isPublic"`
	DisplayOrder    int             `json:"displayOrder"`
}

// UpdatePlanRequest represents a request to update an existing plan.
type UpdatePlanRequest struct {
	Name         *string        `json:"name,omitempty"         validate:"omitempty,min=1,max=100"`
	Description  *string        `json:"description,omitempty"  validate:"omitempty,max=1000"`
	BasePrice    *int64         `json:"basePrice,omitempty"    validate:"omitempty,min=0"`
	TrialDays    *int           `json:"trialDays,omitempty"    validate:"omitempty,min=0,max=365"`
	Features     []PlanFeature  `json:"features,omitempty"`
	PriceTiers   []PriceTier    `json:"priceTiers,omitempty"`
	TierMode     *TierMode      `json:"tierMode,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	IsActive     *bool          `json:"isActive,omitempty"`
	IsPublic     *bool          `json:"isPublic,omitempty"`
	DisplayOrder *int           `json:"displayOrder,omitempty"`
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}
