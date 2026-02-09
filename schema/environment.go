package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// Environment represents an isolated data context within an App (dev, prod, staging, etc.)
type Environment struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:environments,alias:env"`

	ID        xid.ID         `bun:"id,pk,type:varchar(20)"           json:"id"`
	AppID     xid.ID         `bun:"app_id,notnull,type:varchar(20)"  json:"appID"` // Foreign key to App
	Name      string         `bun:"name,notnull"                     json:"name"`
	Slug      string         `bun:"slug,notnull"                     json:"slug"`      // dev, prod, staging
	Type      string         `bun:"type,notnull"                     json:"type"`      // development, production, staging, preview
	Status    string         `bun:"status,notnull,default:'active'"  json:"status"`    // active, inactive
	Config    map[string]any `bun:"config,type:jsonb"                json:"config"`    // Environment-specific configuration
	IsDefault bool           `bun:"is_default,notnull,default:false" json:"isDefault"` // Is this the default environment for the app

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// Environment types.
const (
	EnvironmentTypeDevelopment = "development"
	EnvironmentTypeProduction  = "production"
	EnvironmentTypeStaging     = "staging"
	EnvironmentTypePreview     = "preview"
	EnvironmentTypeTest        = "test"
)

// Environment status.
const (
	EnvironmentStatusActive   = "active"
	EnvironmentStatusInactive = "inactive"
)

// IsProduction checks if this is a production environment.
func (e *Environment) IsProduction() bool {
	return e.Type == EnvironmentTypeProduction
}

// IsDevelopment checks if this is a development environment.
func (e *Environment) IsDevelopment() bool {
	return e.Type == EnvironmentTypeDevelopment
}

// CanDelete checks if this environment can be deleted.
func (e *Environment) CanDelete() bool {
	// Cannot delete default environment
	if e.IsDefault {
		return false
	}
	// Cannot delete production without confirmation
	if e.IsProduction() {
		return false
	}

	return true
}

// EnvironmentPromotion represents a promotion from one environment to another.
type EnvironmentPromotion struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:environment_promotions,alias:ep"`

	ID           xid.ID     `bun:"id,pk,type:varchar(20)"                 json:"id"`
	AppID        xid.ID     `bun:"app_id,notnull,type:varchar(20)"        json:"appID"`
	SourceEnvID  xid.ID     `bun:"source_env_id,notnull,type:varchar(20)" json:"sourceEnvId"`
	TargetEnvID  xid.ID     `bun:"target_env_id,notnull,type:varchar(20)" json:"targetEnvId"`
	PromotedBy   xid.ID     `bun:"promoted_by,notnull,type:varchar(20)"   json:"promotedBy"`  // User who performed promotion
	IncludeData  bool       `bun:"include_data,notnull,default:false"     json:"includeData"` // Whether to copy data or just schema
	Status       string     `bun:"status,notnull"                         json:"status"`      // pending, in_progress, completed, failed
	ErrorMessage string     `bun:"error_message"                          json:"errorMessage"`
	CompletedAt  *time.Time `bun:"completed_at"                           json:"completedAt"`

	// Relations
	App       *App         `bun:"rel:belongs-to,join:app_id=id"`
	SourceEnv *Environment `bun:"rel:belongs-to,join:source_env_id=id"`
	TargetEnv *Environment `bun:"rel:belongs-to,join:target_env_id=id"`
}

// Promotion status.
const (
	PromotionStatusPending    = "pending"
	PromotionStatusInProgress = "in_progress"
	PromotionStatusCompleted  = "completed"
	PromotionStatusFailed     = "failed"
)
