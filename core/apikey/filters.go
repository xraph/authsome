package apikey

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListAPIKeysFilter represents filter parameters for listing API keys
// Supports flexible filtering by app, environment, organization, and user
type ListAPIKeysFilter struct {
	pagination.PaginationParams
	AppID          xid.ID  `json:"appId" query:"app_id"`
	EnvironmentID  *xid.ID `json:"environmentId,omitempty" query:"environment_id"`
	OrganizationID *xid.ID `json:"organizationId,omitempty" query:"organization_id"`
	UserID         *xid.ID `json:"userId,omitempty" query:"user_id"`
	Active         *bool   `json:"active,omitempty" query:"active"`
}
