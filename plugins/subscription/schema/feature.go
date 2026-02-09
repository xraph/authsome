// Package schema defines the database models for the subscription plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainschema "github.com/xraph/authsome/schema"
)

// Feature represents a standalone feature definition that can be linked to plans.
type Feature struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_features,alias:sf"`

	ID                xid.ID         `bun:"id,pk,type:varchar(20)"          json:"id"`
	AppID             xid.ID         `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	Key               string         `bun:"key,notnull"                     json:"key"`         // Unique per app
	Name              string         `bun:"name,notnull"                    json:"name"`        // Display name
	Description       string         `bun:"description"                     json:"description"` // Feature description
	Type              string         `bun:"type,notnull"                    json:"type"`        // boolean, limit, unlimited, metered, tiered
	Unit              string         `bun:"unit"                            json:"unit"`        // "seats", "GB", "API calls", etc.
	ResetPeriod       string         `bun:"reset_period"                    json:"resetPeriod"` // none, daily, weekly, monthly, yearly, billing_period
	IsPublic          bool           `bun:"is_public,notnull,default:true"  json:"isPublic"`    // Show in pricing pages
	DisplayOrder      int            `bun:"display_order,notnull,default:0" json:"displayOrder"`
	Icon              string         `bun:"icon"                            json:"icon"`              // Icon identifier for UI
	ProviderFeatureID string         `bun:"provider_feature_id"             json:"providerFeatureId"` // Provider sync ID
	LastSyncedAt      *time.Time     `bun:"last_synced_at"                  json:"lastSyncedAt"`      // Last provider sync time
	Metadata          map[string]any `bun:"metadata,type:jsonb"             json:"metadata"`

	// Relations
	App   *mainschema.App   `bun:"rel:belongs-to,join:app_id=id"   json:"app,omitempty"`
	Tiers []FeatureTier     `bun:"rel:has-many,join:id=feature_id" json:"tiers,omitempty"`
	Plans []PlanFeatureLink `bun:"rel:has-many,join:id=feature_id" json:"plans,omitempty"`
}

// FeatureTier represents a tier within a tiered feature type.
type FeatureTier struct {
	bun.BaseModel `bun:"table:subscription_feature_tiers,alias:sft"`

	ID        xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	FeatureID xid.ID    `bun:"feature_id,notnull,type:varchar(20)"                   json:"featureId"`
	TierOrder int       `bun:"tier_order,notnull,default:0"                          json:"tierOrder"`
	UpTo      int64     `bun:"up_to,notnull"                                         json:"upTo"`  // -1 for unlimited
	Value     string    `bun:"value"                                                 json:"value"` // What's unlocked at this tier (JSON)
	Label     string    `bun:"label"                                                 json:"label"` // Display label for this tier
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Feature *Feature `bun:"rel:belongs-to,join:feature_id=id" json:"feature,omitempty"`
}

// PlanFeatureLink connects features to plans with plan-specific configuration.
type PlanFeatureLink struct {
	bun.BaseModel `bun:"table:subscription_plan_feature_links,alias:spfl"`

	ID               xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	PlanID           xid.ID         `bun:"plan_id,notnull,type:varchar(20)"                      json:"planId"`
	FeatureID        xid.ID         `bun:"feature_id,notnull,type:varchar(20)"                   json:"featureId"`
	Value            string         `bun:"value"                                                 json:"value"`         // JSON: limit value, tier, boolean, etc.
	IsBlocked        bool           `bun:"is_blocked,notnull,default:false"                      json:"isBlocked"`     // Explicitly blocked for this plan
	IsHighlighted    bool           `bun:"is_highlighted,notnull,default:false"                  json:"isHighlighted"` // Highlight in pricing comparison
	OverrideSettings map[string]any `bun:"override_settings,type:jsonb"                          json:"overrideSettings"`
	CreatedAt        time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time      `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relations
	Plan    *SubscriptionPlan `bun:"rel:belongs-to,join:plan_id=id"    json:"plan,omitempty"`
	Feature *Feature          `bun:"rel:belongs-to,join:feature_id=id" json:"feature,omitempty"`
}
