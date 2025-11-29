// Package schema defines the database models for the subscription plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainschema "github.com/xraph/authsome/schema"
)

// OrganizationFeatureUsage tracks feature usage per organization
type OrganizationFeatureUsage struct {
	bun.BaseModel `bun:"table:subscription_org_feature_usage,alias:sofu"`

	ID             xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID                 `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	FeatureID      xid.ID                 `json:"featureId" bun:"feature_id,notnull,type:varchar(20)"`
	CurrentUsage   int64                  `json:"currentUsage" bun:"current_usage,notnull,default:0"`
	PeriodStart    time.Time              `json:"periodStart" bun:"period_start,notnull"`
	PeriodEnd      time.Time              `json:"periodEnd" bun:"period_end,notnull"`
	LastReset      time.Time              `json:"lastReset" bun:"last_reset,notnull"`
	Metadata       map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt      time.Time              `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt      time.Time              `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Feature      *Feature                 `json:"feature,omitempty" bun:"rel:belongs-to,join:feature_id=id"`
}

// FeatureUsageLog provides an audit trail for feature usage changes
type FeatureUsageLog struct {
	bun.BaseModel `bun:"table:subscription_feature_usage_logs,alias:sful"`

	ID              xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID  xid.ID                 `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	FeatureID       xid.ID                 `json:"featureId" bun:"feature_id,notnull,type:varchar(20)"`
	Action          string                 `json:"action" bun:"action,notnull"`        // consume, grant, reset, adjust
	Quantity        int64                  `json:"quantity" bun:"quantity,notnull"`
	PreviousUsage   int64                  `json:"previousUsage" bun:"previous_usage,notnull,default:0"`
	NewUsage        int64                  `json:"newUsage" bun:"new_usage,notnull,default:0"`
	ActorID         *xid.ID                `json:"actorId" bun:"actor_id,type:varchar(20)"`
	Reason          string                 `json:"reason" bun:"reason"`
	IdempotencyKey  string                 `json:"idempotencyKey" bun:"idempotency_key"`
	Metadata        map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`
	CreatedAt       time.Time              `json:"createdAt" bun:"created_at,nullzero,notnull,default:current_timestamp"`

	// Relations
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Feature      *Feature                 `json:"feature,omitempty" bun:"rel:belongs-to,join:feature_id=id"`
}

// FeatureGrant provides additional feature access beyond the plan
type FeatureGrant struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_feature_grants,alias:sfg"`

	ID             xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	OrganizationID xid.ID                 `json:"organizationId" bun:"organization_id,notnull,type:varchar(20)"`
	FeatureID      xid.ID                 `json:"featureId" bun:"feature_id,notnull,type:varchar(20)"`
	GrantType      string                 `json:"grantType" bun:"grant_type,notnull"` // addon, override, promotion, trial, manual
	Value          int64                  `json:"value" bun:"value,notnull"`          // Additional quota
	ExpiresAt      *time.Time             `json:"expiresAt" bun:"expires_at"`
	SourceType     string                 `json:"sourceType" bun:"source_type"`       // addon, coupon, promotion, etc.
	SourceID       *xid.ID                `json:"sourceId" bun:"source_id,type:varchar(20)"` // AddOn ID, Promotion ID, etc.
	Reason         string                 `json:"reason" bun:"reason"`
	IsActive       bool                   `json:"isActive" bun:"is_active,notnull,default:true"`
	Metadata       map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	Organization *mainschema.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Feature      *Feature                 `json:"feature,omitempty" bun:"rel:belongs-to,join:feature_id=id"`
}

// FeatureUsageAction defines the type of usage action
type FeatureUsageAction string

const (
	// FeatureUsageActionConsume decrements the available quota
	FeatureUsageActionConsume FeatureUsageAction = "consume"
	// FeatureUsageActionGrant adds to the quota
	FeatureUsageActionGrant FeatureUsageAction = "grant"
	// FeatureUsageActionReset resets the usage counter
	FeatureUsageActionReset FeatureUsageAction = "reset"
	// FeatureUsageActionAdjust manually adjusts the usage
	FeatureUsageActionAdjust FeatureUsageAction = "adjust"
)

// FeatureGrantType defines the type of feature grant
type FeatureGrantType string

const (
	// FeatureGrantTypeAddon is a grant from an add-on purchase
	FeatureGrantTypeAddon FeatureGrantType = "addon"
	// FeatureGrantTypeOverride is a manual override
	FeatureGrantTypeOverride FeatureGrantType = "override"
	// FeatureGrantTypePromotion is a promotional grant
	FeatureGrantTypePromotion FeatureGrantType = "promotion"
	// FeatureGrantTypeTrial is a trial grant
	FeatureGrantTypeTrial FeatureGrantType = "trial"
	// FeatureGrantTypeManual is a manually added grant
	FeatureGrantTypeManual FeatureGrantType = "manual"
)

