package core

import (
	"time"

	"github.com/rs/xid"
)

// Feature represents a standalone feature definition that can be linked to plans.
type Feature struct {
	ID                xid.ID         `json:"id"`
	AppID             xid.ID         `json:"appId"`
	Key               string         `json:"key"`               // Unique per app
	Name              string         `json:"name"`              // Display name
	Description       string         `json:"description"`       // Feature description
	Type              FeatureType    `json:"type"`              // boolean, limit, unlimited, metered, tiered
	Unit              string         `json:"unit"`              // "seats", "GB", "API calls", etc.
	ResetPeriod       ResetPeriod    `json:"resetPeriod"`       // When usage resets
	IsPublic          bool           `json:"isPublic"`          // Show in pricing pages
	DisplayOrder      int            `json:"displayOrder"`      // Order in UI
	Icon              string         `json:"icon"`              // Icon identifier for UI
	ProviderFeatureID string         `json:"providerFeatureId"` // Provider sync ID (e.g., Stripe product feature ID)
	LastSyncedAt      *time.Time     `json:"lastSyncedAt"`      // Last provider sync time
	Metadata          map[string]any `json:"metadata"`
	Tiers             []FeatureTier  `json:"tiers,omitempty"` // For tiered features
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// FeatureTier represents a tier within a tiered feature type.
type FeatureTier struct {
	ID        xid.ID `json:"id"`
	FeatureID xid.ID `json:"featureId"`
	TierOrder int    `json:"tierOrder"`
	UpTo      int64  `json:"upTo"`  // -1 for unlimited
	Value     string `json:"value"` // What's unlocked at this tier
	Label     string `json:"label"` // Display label for this tier
}

// PlanFeatureLink connects features to plans with plan-specific configuration.
type PlanFeatureLink struct {
	ID               xid.ID         `json:"id"`
	PlanID           xid.ID         `json:"planId"`
	FeatureID        xid.ID         `json:"featureId"`
	Value            string         `json:"value"`         // JSON: limit value, tier, boolean, etc.
	IsBlocked        bool           `json:"isBlocked"`     // Explicitly blocked for this plan
	IsHighlighted    bool           `json:"isHighlighted"` // Highlight in pricing comparison
	OverrideSettings map[string]any `json:"overrideSettings,omitempty"`
	Feature          *Feature       `json:"feature,omitempty"` // Loaded feature
}

// OrganizationFeatureUsage tracks feature usage per organization.
type OrganizationFeatureUsage struct {
	ID             xid.ID         `json:"id"`
	OrganizationID xid.ID         `json:"organizationId"`
	FeatureID      xid.ID         `json:"featureId"`
	FeatureKey     string         `json:"featureKey"` // Denormalized for convenience
	CurrentUsage   int64          `json:"currentUsage"`
	PeriodStart    time.Time      `json:"periodStart"`
	PeriodEnd      time.Time      `json:"periodEnd"`
	LastReset      time.Time      `json:"lastReset"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// FeatureUsageLog represents an audit entry for feature usage changes.
type FeatureUsageLog struct {
	ID             xid.ID         `json:"id"`
	OrganizationID xid.ID         `json:"organizationId"`
	FeatureID      xid.ID         `json:"featureId"`
	FeatureKey     string         `json:"featureKey"`
	Action         string         `json:"action"` // consume, grant, reset, adjust
	Quantity       int64          `json:"quantity"`
	PreviousUsage  int64          `json:"previousUsage"`
	NewUsage       int64          `json:"newUsage"`
	ActorID        *xid.ID        `json:"actorId,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	IdempotencyKey string         `json:"idempotencyKey,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
}

// FeatureGrant provides additional feature access beyond the plan.
type FeatureGrant struct {
	ID             xid.ID           `json:"id"`
	OrganizationID xid.ID           `json:"organizationId"`
	FeatureID      xid.ID           `json:"featureId"`
	FeatureKey     string           `json:"featureKey"` // Denormalized for convenience
	GrantType      FeatureGrantType `json:"grantType"`  // addon, override, promotion, trial, manual
	Value          int64            `json:"value"`      // Additional quota
	ExpiresAt      *time.Time       `json:"expiresAt,omitempty"`
	SourceType     string           `json:"sourceType,omitempty"` // addon, coupon, promotion, etc.
	SourceID       *xid.ID          `json:"sourceId,omitempty"`   // AddOn ID, Promotion ID, etc.
	Reason         string           `json:"reason,omitempty"`
	IsActive       bool             `json:"isActive"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
	CreatedAt      time.Time        `json:"createdAt"`
	UpdatedAt      time.Time        `json:"updatedAt"`
}

// FeatureAccess represents the complete access state for a feature.
type FeatureAccess struct {
	Feature      *Feature `json:"feature"`
	HasAccess    bool     `json:"hasAccess"`
	IsBlocked    bool     `json:"isBlocked"`
	Limit        int64    `json:"limit"` // -1 for unlimited, 0 for no access
	CurrentUsage int64    `json:"currentUsage"`
	Remaining    int64    `json:"remaining"`    // -1 for unlimited
	GrantedExtra int64    `json:"grantedExtra"` // Extra from grants
	PlanValue    string   `json:"planValue"`    // The raw value from plan
}

// NewFeature creates a new Feature with default values.
func NewFeature(appID xid.ID, key, name string, featureType FeatureType) *Feature {
	now := time.Now()

	return &Feature{
		ID:           xid.New(),
		AppID:        appID,
		Key:          key,
		Name:         name,
		Type:         featureType,
		ResetPeriod:  ResetPeriodBillingCycle,
		IsPublic:     true,
		DisplayOrder: 0,
		Metadata:     make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateFeatureRequest represents a request to create a new feature.
type CreateFeatureRequest struct {
	Key          string         `json:"key"                validate:"required,min=1,max=100"`
	Name         string         `json:"name"               validate:"required,min=1,max=100"`
	Description  string         `json:"description"        validate:"max=1000"`
	Type         FeatureType    `json:"type"               validate:"required"`
	Unit         string         `json:"unit"               validate:"max=50"`
	ResetPeriod  ResetPeriod    `json:"resetPeriod"`
	IsPublic     bool           `json:"isPublic"`
	DisplayOrder int            `json:"displayOrder"`
	Icon         string         `json:"icon"               validate:"max=100"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Tiers        []FeatureTier  `json:"tiers,omitempty"` // For tiered features
}

// UpdateFeatureRequest represents a request to update an existing feature.
type UpdateFeatureRequest struct {
	Name         *string        `json:"name,omitempty"         validate:"omitempty,min=1,max=100"`
	Description  *string        `json:"description,omitempty"  validate:"omitempty,max=1000"`
	Unit         *string        `json:"unit,omitempty"         validate:"omitempty,max=50"`
	ResetPeriod  *ResetPeriod   `json:"resetPeriod,omitempty"`
	IsPublic     *bool          `json:"isPublic,omitempty"`
	DisplayOrder *int           `json:"displayOrder,omitempty"`
	Icon         *string        `json:"icon,omitempty"         validate:"omitempty,max=100"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Tiers        []FeatureTier  `json:"tiers,omitempty"`
}

// LinkFeatureRequest represents a request to link a feature to a plan.
type LinkFeatureRequest struct {
	FeatureID        xid.ID         `json:"featureId"                  validate:"required"`
	Value            string         `json:"value"`         // The limit/value for this plan
	IsBlocked        bool           `json:"isBlocked"`     // Explicitly block this feature
	IsHighlighted    bool           `json:"isHighlighted"` // Highlight in pricing UI
	OverrideSettings map[string]any `json:"overrideSettings,omitempty"`
}

// UpdateLinkRequest represents a request to update a feature-plan link.
type UpdateLinkRequest struct {
	Value            *string        `json:"value,omitempty"`
	IsBlocked        *bool          `json:"isBlocked,omitempty"`
	IsHighlighted    *bool          `json:"isHighlighted,omitempty"`
	OverrideSettings map[string]any `json:"overrideSettings,omitempty"`
}

// ConsumeFeatureRequest represents a request to consume feature quota.
type ConsumeFeatureRequest struct {
	OrganizationID xid.ID         `json:"organizationId"           validate:"required"`
	FeatureKey     string         `json:"featureKey"               validate:"required"`
	Quantity       int64          `json:"quantity"                 validate:"required,min=1"`
	IdempotencyKey string         `json:"idempotencyKey,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// GrantFeatureRequest represents a request to grant additional feature quota.
type GrantFeatureRequest struct {
	OrganizationID xid.ID           `json:"organizationId"       validate:"required"`
	FeatureKey     string           `json:"featureKey"           validate:"required"`
	GrantType      FeatureGrantType `json:"grantType"            validate:"required"`
	Value          int64            `json:"value"                validate:"required,min=1"`
	ExpiresAt      *time.Time       `json:"expiresAt,omitempty"`
	SourceType     string           `json:"sourceType,omitempty"`
	SourceID       *xid.ID          `json:"sourceId,omitempty"`
	Reason         string           `json:"reason,omitempty"`
	Metadata       map[string]any   `json:"metadata,omitempty"`
}

// FeatureUsageResponse represents the response for feature usage queries.
type FeatureUsageResponse struct {
	FeatureKey   string    `json:"featureKey"`
	FeatureName  string    `json:"featureName"`
	FeatureType  string    `json:"featureType"`
	CurrentUsage int64     `json:"currentUsage"`
	Limit        int64     `json:"limit"`     // -1 for unlimited
	Remaining    int64     `json:"remaining"` // -1 for unlimited
	PeriodStart  time.Time `json:"periodStart"`
	PeriodEnd    time.Time `json:"periodEnd"`
	GrantedExtra int64     `json:"grantedExtra"` // Extra quota from grants
}

// PublicFeature represents a feature for public API (pricing pages).
type PublicFeature struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Unit         string `json:"unit,omitempty"`
	Icon         string `json:"icon,omitempty"`
	DisplayOrder int    `json:"displayOrder"`
}

// PublicPlanFeature represents a feature within a plan for public API.
type PublicPlanFeature struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Unit          string `json:"unit,omitempty"`
	Value         any    `json:"value"` // Parsed value (boolean, number, or tier info)
	IsHighlighted bool   `json:"isHighlighted"`
	IsBlocked     bool   `json:"isBlocked"`
	DisplayOrder  int    `json:"displayOrder"`
}
