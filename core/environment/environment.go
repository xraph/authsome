package environment

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// ENVIRONMENT DTO (Data Transfer Object)
// =============================================================================

// Environment represents an environment DTO
// This is separate from schema.Environment to maintain proper separation of concerns.
type Environment struct {
	ID        xid.ID         `json:"id"`
	AppID     xid.ID         `json:"appId"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Config    map[string]any `json:"config,omitempty"`
	IsDefault bool           `json:"isDefault"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Environment DTO to a schema.Environment model.
func (e *Environment) ToSchema() *schema.Environment {
	return &schema.Environment{
		ID:        e.ID,
		AppID:     e.AppID,
		Name:      e.Name,
		Slug:      e.Slug,
		Type:      e.Type,
		Status:    e.Status,
		Config:    e.Config,
		IsDefault: e.IsDefault,
		AuditableModel: schema.AuditableModel{
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: e.DeletedAt,
		},
	}
}

// FromSchemaEnvironment converts a schema.Environment model to Environment DTO.
func FromSchemaEnvironment(se *schema.Environment) *Environment {
	if se == nil {
		return nil
	}

	return &Environment{
		ID:        se.ID,
		AppID:     se.AppID,
		Name:      se.Name,
		Slug:      se.Slug,
		Type:      se.Type,
		Status:    se.Status,
		Config:    se.Config,
		IsDefault: se.IsDefault,
		CreatedAt: se.CreatedAt,
		UpdatedAt: se.UpdatedAt,
		DeletedAt: se.DeletedAt,
	}
}

// FromSchemaEnvironments converts a slice of schema.Environment to Environment DTOs.
func FromSchemaEnvironments(envs []*schema.Environment) []*Environment {
	result := make([]*Environment, len(envs))
	for i, e := range envs {
		result[i] = FromSchemaEnvironment(e)
	}

	return result
}

// =============================================================================
// ENVIRONMENT METHODS
// =============================================================================

// IsProduction checks if the environment is production.
func (e *Environment) IsProduction() bool {
	return e.Type == schema.EnvironmentTypeProduction
}

// IsDevelopment checks if the environment is development.
func (e *Environment) IsDevelopment() bool {
	return e.Type == schema.EnvironmentTypeDevelopment
}

// IsActive checks if the environment is active.
func (e *Environment) IsActive() bool {
	return e.Status == schema.EnvironmentStatusActive
}

// IsDeleted checks if the environment is soft-deleted.
func (e *Environment) IsDeleted() bool {
	return e.DeletedAt != nil
}

// =============================================================================
// PROMOTION DTO
// =============================================================================

// Promotion represents an environment promotion DTO.
type Promotion struct {
	ID           xid.ID     `json:"id"`
	AppID        xid.ID     `json:"appId"`
	SourceEnvID  xid.ID     `json:"sourceEnvId"`
	TargetEnvID  xid.ID     `json:"targetEnvId"`
	PromotedBy   xid.ID     `json:"promotedBy"`
	Status       string     `json:"status"`
	IncludeData  bool       `json:"includeData"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// ToSchema converts the Promotion DTO to a schema.EnvironmentPromotion model.
func (p *Promotion) ToSchema() *schema.EnvironmentPromotion {
	return &schema.EnvironmentPromotion{
		ID:           p.ID,
		AppID:        p.AppID,
		SourceEnvID:  p.SourceEnvID,
		TargetEnvID:  p.TargetEnvID,
		PromotedBy:   p.PromotedBy,
		Status:       p.Status,
		IncludeData:  p.IncludeData,
		ErrorMessage: p.ErrorMessage,
		CompletedAt:  p.CompletedAt,
		AuditableModel: schema.AuditableModel{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
	}
}

// FromSchemaPromotion converts a schema.EnvironmentPromotion model to Promotion DTO.
func FromSchemaPromotion(sp *schema.EnvironmentPromotion) *Promotion {
	if sp == nil {
		return nil
	}

	return &Promotion{
		ID:           sp.ID,
		AppID:        sp.AppID,
		SourceEnvID:  sp.SourceEnvID,
		TargetEnvID:  sp.TargetEnvID,
		PromotedBy:   sp.PromotedBy,
		Status:       sp.Status,
		IncludeData:  sp.IncludeData,
		ErrorMessage: sp.ErrorMessage,
		CompletedAt:  sp.CompletedAt,
		CreatedAt:    sp.CreatedAt,
		UpdatedAt:    sp.UpdatedAt,
	}
}

// FromSchemaPromotions converts a slice of schema.EnvironmentPromotion to Promotion DTOs.
func FromSchemaPromotions(promotions []*schema.EnvironmentPromotion) []*Promotion {
	result := make([]*Promotion, len(promotions))
	for i, p := range promotions {
		result[i] = FromSchemaPromotion(p)
	}

	return result
}

// =============================================================================
// PROMOTION METHODS
// =============================================================================

// IsPending checks if the promotion is pending.
func (p *Promotion) IsPending() bool {
	return p.Status == schema.PromotionStatusPending
}

// IsInProgress checks if the promotion is in progress.
func (p *Promotion) IsInProgress() bool {
	return p.Status == schema.PromotionStatusInProgress
}

// IsCompleted checks if the promotion is completed.
func (p *Promotion) IsCompleted() bool {
	return p.Status == schema.PromotionStatusCompleted
}

// IsFailed checks if the promotion failed.
func (p *Promotion) IsFailed() bool {
	return p.Status == schema.PromotionStatusFailed
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateEnvironmentRequest represents the request to create an environment.
type CreateEnvironmentRequest struct {
	AppID  xid.ID         `json:"appId"            validate:"required"`
	Name   string         `json:"name"             validate:"required,min=1,max=100"`
	Slug   string         `json:"slug"             validate:"required,min=1,max=50,alphanum"`
	Type   string         `json:"type"             validate:"required,oneof=development staging production preview test"`
	Config map[string]any `json:"config,omitempty"`
}

// UpdateEnvironmentRequest represents the request to update an environment.
type UpdateEnvironmentRequest struct {
	Name   *string        `json:"name,omitempty"   validate:"omitempty,min=1,max=100"`
	Status *string        `json:"status,omitempty" validate:"omitempty,oneof=active inactive maintenance"`
	Config map[string]any `json:"config,omitempty"`
	Type   *string        `json:"type,omitempty"   validate:"omitempty,oneof=development staging production preview test"`
}

// PromoteEnvironmentRequest represents the request to promote an environment.
type PromoteEnvironmentRequest struct {
	SourceEnvID xid.ID         `json:"sourceEnvId"      validate:"required"`
	TargetName  string         `json:"targetName"       validate:"required,min=1,max=100"`
	TargetSlug  string         `json:"targetSlug"       validate:"required,min=1,max=50,alphanum"`
	TargetType  string         `json:"targetType"       validate:"required,oneof=development staging production preview test"`
	IncludeData bool           `json:"includeData"`
	PromotedBy  xid.ID         `json:"promotedBy"       validate:"required"`
	Config      map[string]any `json:"config,omitempty"`
}

// ListEnvironmentsResponse represents paginated environment response.
type ListEnvironmentsResponse = pagination.PageResponse[*Environment]

// ListPromotionsResponse represents paginated promotion response.
type ListPromotionsResponse = pagination.PageResponse[*Promotion]
