// Package schema defines the database models for the subscription plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainschema "github.com/xraph/authsome/schema"
)

// OrganizationFeatureUsage tracks feature usage per organization.
type OrganizationFeatureUsage struct {
	bun.BaseModel `bun:"table:subscription_org_feature_usage,alias:sofu"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	OrganizationID xid.ID         `bun:"organization_id,notnull,type:varchar(20)"              json:"organizationId"`
	FeatureID      xid.ID         `bun:"feature_id,notnull,type:varchar(20)"                   json:"featureId"`
	CurrentUsage   int64          `bun:"current_usage,notnull,default:0"                       json:"currentUsage"`
	PeriodStart    time.Time      `bun:"period_start,notnull"                                  json:"periodStart"`
	PeriodEnd      time.Time      `bun:"period_end,notnull"                                    json:"periodEnd"`
	LastReset      time.Time      `bun:"last_reset,notnull"                                    json:"lastReset"`
	Metadata       map[string]any `bun:"metadata,type:jsonb"                                   json:"metadata"`
	CreatedAt      time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time      `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Relations
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
	Feature      *Feature                 `bun:"rel:belongs-to,join:feature_id=id"      json:"feature,omitempty"`
}

// FeatureUsageLog provides an audit trail for feature usage changes.
type FeatureUsageLog struct {
	bun.BaseModel `bun:"table:subscription_feature_usage_logs,alias:sful"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"                                json:"id"`
	OrganizationID xid.ID         `bun:"organization_id,notnull,type:varchar(20)"              json:"organizationId"`
	FeatureID      xid.ID         `bun:"feature_id,notnull,type:varchar(20)"                   json:"featureId"`
	Action         string         `bun:"action,notnull"                                        json:"action"` // consume, grant, reset, adjust
	Quantity       int64          `bun:"quantity,notnull"                                      json:"quantity"`
	PreviousUsage  int64          `bun:"previous_usage,notnull,default:0"                      json:"previousUsage"`
	NewUsage       int64          `bun:"new_usage,notnull,default:0"                           json:"newUsage"`
	ActorID        *xid.ID        `bun:"actor_id,type:varchar(20)"                             json:"actorId"`
	Reason         string         `bun:"reason"                                                json:"reason"`
	IdempotencyKey string         `bun:"idempotency_key"                                       json:"idempotencyKey"`
	Metadata       map[string]any `bun:"metadata,type:jsonb"                                   json:"metadata"`
	CreatedAt      time.Time      `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
	Feature      *Feature                 `bun:"rel:belongs-to,join:feature_id=id"      json:"feature,omitempty"`
}

// FeatureGrant provides additional feature access beyond the plan.
type FeatureGrant struct {
	mainschema.AuditableModel `bun:",inline"`
	bun.BaseModel             `bun:"table:subscription_feature_grants,alias:sfg"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"                   json:"id"`
	OrganizationID xid.ID         `bun:"organization_id,notnull,type:varchar(20)" json:"organizationId"`
	FeatureID      xid.ID         `bun:"feature_id,notnull,type:varchar(20)"      json:"featureId"`
	GrantType      string         `bun:"grant_type,notnull"                       json:"grantType"` // addon, override, promotion, trial, manual
	Value          int64          `bun:"value,notnull"                            json:"value"`     // Additional quota
	ExpiresAt      *time.Time     `bun:"expires_at"                               json:"expiresAt"`
	SourceType     string         `bun:"source_type"                              json:"sourceType"` // addon, coupon, promotion, etc.
	SourceID       *xid.ID        `bun:"source_id,type:varchar(20)"               json:"sourceId"`   // AddOn ID, Promotion ID, etc.
	Reason         string         `bun:"reason"                                   json:"reason"`
	IsActive       bool           `bun:"is_active,notnull,default:true"           json:"isActive"`
	Metadata       map[string]any `bun:"metadata,type:jsonb"                      json:"metadata"`

	// Relations
	Organization *mainschema.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
	Feature      *Feature                 `bun:"rel:belongs-to,join:feature_id=id"      json:"feature,omitempty"`
}

// FeatureUsageAction defines the type of usage action.
type FeatureUsageAction string

const (
	// FeatureUsageActionConsume decrements the available quota.
	FeatureUsageActionConsume FeatureUsageAction = "consume"
	// FeatureUsageActionGrant adds to the quota.
	FeatureUsageActionGrant FeatureUsageAction = "grant"
	// FeatureUsageActionReset resets the usage counter.
	FeatureUsageActionReset FeatureUsageAction = "reset"
	// FeatureUsageActionAdjust manually adjusts the usage.
	FeatureUsageActionAdjust FeatureUsageAction = "adjust"
)

// FeatureGrantType defines the type of feature grant.
type FeatureGrantType string

const (
	// FeatureGrantTypeAddon is a grant from an add-on purchase.
	FeatureGrantTypeAddon FeatureGrantType = "addon"
	// FeatureGrantTypeOverride is a manual override.
	FeatureGrantTypeOverride FeatureGrantType = "override"
	// FeatureGrantTypePromotion is a promotional grant.
	FeatureGrantTypePromotion FeatureGrantType = "promotion"
	// FeatureGrantTypeTrial is a trial grant.
	FeatureGrantTypeTrial FeatureGrantType = "trial"
	// FeatureGrantTypeManual is a manually added grant.
	FeatureGrantTypeManual FeatureGrantType = "manual"
)
