package organization

import (
	"time"

	"github.com/rs/xid"
)

// Organization represents an organization entity
type Organization struct {
	ID        xid.ID                 `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Logo      string                 `json:"logo"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// CreateOrganizationRequest represents a create organization request
type CreateOrganizationRequest struct {
	Name     string                 `json:"name"`
	Slug     string                 `json:"slug"`
	Logo     string                 `json:"logo"`
	Metadata map[string]interface{} `json:"metadata"`
}

// UpdateOrganizationRequest represents an update organization request
type UpdateOrganizationRequest struct {
	Name     *string                `json:"name"`
	Logo     *string                `json:"logo"`
	Metadata map[string]interface{} `json:"metadata"`
}
