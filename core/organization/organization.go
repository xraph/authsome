package organization

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/schema"
)

// Organization represents an organization entity DTO (Data Transfer Object)
// This is separate from schema.Organization to maintain proper separation of concerns.
type Organization struct {
	ID            xid.ID         `json:"id"`
	AppID         xid.ID         `json:"appID"`
	EnvironmentID xid.ID         `json:"environmentID"`
	Name          string         `json:"name"`
	Slug          string         `json:"slug"`
	Logo          string         `json:"logo,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedBy     xid.ID         `json:"createdBy"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Organization DTO to a schema.Organization model.
func (o *Organization) ToSchema() *schema.Organization {
	return &schema.Organization{
		ID:            o.ID,
		AppID:         o.AppID,
		EnvironmentID: o.EnvironmentID,
		Name:          o.Name,
		Slug:          o.Slug,
		Logo:          o.Logo,
		Metadata:      o.Metadata,
		CreatedBy:     o.CreatedBy,
		AuditableModel: schema.AuditableModel{
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
			DeletedAt: o.DeletedAt,
		},
	}
}

// FromSchemaOrganization converts a schema.Organization model to Organization DTO.
func FromSchemaOrganization(so *schema.Organization) *Organization {
	if so == nil {
		return nil
	}

	return &Organization{
		ID:            so.ID,
		AppID:         so.AppID,
		EnvironmentID: so.EnvironmentID,
		Name:          so.Name,
		Slug:          so.Slug,
		Logo:          so.Logo,
		Metadata:      so.Metadata,
		CreatedBy:     so.CreatedBy,
		CreatedAt:     so.CreatedAt,
		UpdatedAt:     so.UpdatedAt,
		DeletedAt:     so.DeletedAt,
	}
}

// FromSchemaOrganizations converts a slice of schema.Organization to Organization DTOs.
func FromSchemaOrganizations(orgs []*schema.Organization) []*Organization {
	result := make([]*Organization, len(orgs))
	for i, o := range orgs {
		result[i] = FromSchemaOrganization(o)
	}

	return result
}

// CreateOrganizationRequest represents a create organization request.
type CreateOrganizationRequest struct {
	Name               string                             `json:"name"                         validate:"required,min=1,max=100"`
	Slug               string                             `json:"slug"                         validate:"required,min=1,max=100,slug"`
	Logo               *string                            `json:"logo,omitempty"`
	Metadata           map[string]any                     `json:"metadata,omitempty"`
	RoleTemplateIDs    []xid.ID                           `json:"roleTemplateIDs,omitempty"`    // Role templates to bootstrap (empty = all)
	RoleCustomizations map[xid.ID]*rbac.RoleCustomization `json:"roleCustomizations,omitempty"` // Customizations for role templates
}

// UpdateOrganizationRequest represents an update organization request.
type UpdateOrganizationRequest struct {
	Name     *string        `json:"name,omitempty"     validate:"omitempty,min=1,max=100"`
	Logo     *string        `json:"logo,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}
