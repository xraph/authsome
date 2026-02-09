package session

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// ListSessionsFilter represents filter parameters for listing sessions.
type ListSessionsFilter struct {
	pagination.PaginationParams

	AppID          xid.ID  `json:"appId"                    query:"app_id"`
	EnvironmentID  *xid.ID `json:"environmentId,omitempty"  query:"environment_id"`
	OrganizationID *xid.ID `json:"organizationId,omitempty" query:"organization_id"`
	UserID         *xid.ID `json:"userId,omitempty"         query:"user_id"`
	Active         *bool   `json:"active,omitempty"         query:"active"` // Filter by expired/active
}
