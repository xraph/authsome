// Package schema defines the database models for the subscription plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainschema "github.com/xraph/authsome/schema"
)

// Feature represents a standalone feature definition that can be linked to plans
type Feature struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_features,alias:sf"`

	ID                xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID             xid.ID                 `json:"appId" bun:"app_id,notnull,type:varchar(20)"`
	Key               string                 `json:"key" bun:"key,notnull"`                         // Unique per app
	Name              string                 `json:"name" bun:"name,notnull"`                       // Display name
	Description       string                 `json:"description" bun:"description"`                 // Feature description
	Type              string                 `json:"type" bun:"type,notnull"`                       // boolean, limit, unlimited, metered, tiered
	Unit              string                 `json:"unit" bun:"unit"`                               // "seats", "GB", "API calls", etc.
	ResetPeriod       string                 `json:"resetPeriod" bun:"reset_period"`                // none, daily, weekly, monthly, yearly, billing_period
	IsPublic          bool                   `json:"isPublic" bun:"is_public,notnull,default:true"` // Show in pricing pages
	DisplayOrder      int                    `json:"displayOrder" bun:"display_order,notnull,default:0"`
	Icon              string                 `json:"icon" bun:"icon"`                             // Icon identifier for UI
	ProviderFeatureID string                 `json:"providerFeatureId" bun:"provider_feature_id"` // Provider sync ID
	LastSyncedAt      *time.Time             `json:"lastSyncedAt" bun:"last_synced_at"`           // Last provider sync time
	Metadata          map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	App   *mainschema.App   `json:"app,omitempty" bun:"rel:belongs-to,join:app_id=id"`
	Tiers []FeatureTier     `json:"tiers,omitempty" bun:"rel:has-many,join:id=feature_id"`
	Plans []PlanFeatureLink `json:"plans,omitempty" bun:"rel:has-many,join:id=feature_id"`
}

// FeatureTier represents a tier within a tiered feature type
type FeatureTier struct {
	bun.BaseModel `bun:"table:subscription_feature_tiers,alias:sft"`

	ID        xid.ID    `json:"id" bun:"id,pk,type:varchar(20)"`
	FeatureID xid.ID    `json:"featureId" bun:"feature_id,notnull,type:varchar(20)"`
	TierOrder int       `json:"tierOrder" bun:"tier_order,notnull,default:0"`
	UpTo      int64     `json:"upTo" bun:"up_to,notnull"` // -1 for unlimited
	Value     string    `json:"value" bun:"value"`        // What's unlocked at this tier (JSON)
	Label     string    `json:"label" bun:"label"`        // Display label for this tier
	CreatedAt time.Time `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Feature *Feature `json:"feature,omitempty" bun:"rel:belongs-to,join:feature_id=id"`
}

// PlanFeatureLink connects features to plans with plan-specific configuration
type PlanFeatureLink struct {
	bun.BaseModel `bun:"table:subscription_plan_feature_links,alias:spfl"`

	ID               xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	PlanID           xid.ID                 `json:"planId" bun:"plan_id,notnull,type:varchar(20)"`
	FeatureID        xid.ID                 `json:"featureId" bun:"feature_id,notnull,type:varchar(20)"`
	Value            string                 `json:"value" bun:"value"`                                        // JSON: limit value, tier, boolean, etc.
	IsBlocked        bool                   `json:"isBlocked" bun:"is_blocked,notnull,default:false"`         // Explicitly blocked for this plan
	IsHighlighted    bool                   `json:"isHighlighted" bun:"is_highlighted,notnull,default:false"` // Highlight in pricing comparison
	OverrideSettings map[string]interface{} `json:"overrideSettings" bun:"override_settings,type:jsonb"`
	CreatedAt        time.Time              `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt        time.Time              `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Plan    *SubscriptionPlan `json:"plan,omitempty" bun:"rel:belongs-to,join:plan_id=id"`
	Feature *Feature          `json:"feature,omitempty" bun:"rel:belongs-to,join:feature_id=id"`
}
