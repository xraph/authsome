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

	ID        xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID     xid.ID                 `json:"appID" bun:"app_id,notnull,type:varchar(20)"` // Foreign key to App
	Name      string                 `json:"name" bun:"name,notnull"`
	Slug      string                 `json:"slug" bun:"slug,notnull"`                          // dev, prod, staging
	Type      string                 `json:"type" bun:"type,notnull"`                          // development, production, staging, preview
	Status    string                 `json:"status" bun:"status,notnull,default:'active'"`     // active, inactive
	Config    map[string]interface{} `json:"config" bun:"config,type:jsonb"`                   // Environment-specific configuration
	IsDefault bool                   `json:"isDefault" bun:"is_default,notnull,default:false"` // Is this the default environment for the app

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id"`
}

// Environment types
const (
	EnvironmentTypeDevelopment = "development"
	EnvironmentTypeProduction  = "production"
	EnvironmentTypeStaging     = "staging"
	EnvironmentTypePreview     = "preview"
	EnvironmentTypeTest        = "test"
)

// Environment status
const (
	EnvironmentStatusActive   = "active"
	EnvironmentStatusInactive = "inactive"
)

// IsProduction checks if this is a production environment
func (e *Environment) IsProduction() bool {
	return e.Type == EnvironmentTypeProduction
}

// IsDevelopment checks if this is a development environment
func (e *Environment) IsDevelopment() bool {
	return e.Type == EnvironmentTypeDevelopment
}

// CanDelete checks if this environment can be deleted
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

// EnvironmentPromotion represents a promotion from one environment to another
type EnvironmentPromotion struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:environment_promotions,alias:ep"`

	ID           xid.ID     `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID        xid.ID     `json:"appID" bun:"app_id,notnull,type:varchar(20)"`
	SourceEnvID  xid.ID     `json:"sourceEnvId" bun:"source_env_id,notnull,type:varchar(20)"`
	TargetEnvID  xid.ID     `json:"targetEnvId" bun:"target_env_id,notnull,type:varchar(20)"`
	PromotedBy   xid.ID     `json:"promotedBy" bun:"promoted_by,notnull,type:varchar(20)"` // User who performed promotion
	IncludeData  bool       `json:"includeData" bun:"include_data,notnull,default:false"`  // Whether to copy data or just schema
	Status       string     `json:"status" bun:"status,notnull"`                           // pending, in_progress, completed, failed
	ErrorMessage string     `json:"errorMessage" bun:"error_message"`
	CompletedAt  *time.Time `json:"completedAt" bun:"completed_at"`

	// Relations
	App       *App         `bun:"rel:belongs-to,join:app_id=id"`
	SourceEnv *Environment `bun:"rel:belongs-to,join:source_env_id=id"`
	TargetEnv *Environment `bun:"rel:belongs-to,join:target_env_id=id"`
}

// Promotion status
const (
	PromotionStatusPending    = "pending"
	PromotionStatusInProgress = "in_progress"
	PromotionStatusCompleted  = "completed"
	PromotionStatusFailed     = "failed"
)
