package audit

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// ListEventsFilter defines filters for listing audit events with pagination
type ListEventsFilter struct {
	pagination.PaginationParams

	// Filter by user
	UserID *xid.ID `json:"userId,omitempty" query:"user_id"`

	// Filter by action
	Action *string `json:"action,omitempty" query:"action"`

	// Filter by resource
	Resource *string `json:"resource,omitempty" query:"resource"`

	// Filter by IP address
	IPAddress *string `json:"ipAddress,omitempty" query:"ip_address"`

	// Time range filters
	Since *time.Time `json:"since,omitempty" query:"since"`
	Until *time.Time `json:"until,omitempty" query:"until"`

	// Sort order (default: created_at DESC)
	SortBy    *string `json:"sortBy,omitempty" query:"sort_by"`       // created_at, action, resource
	SortOrder *string `json:"sortOrder,omitempty" query:"sort_order"` // asc, desc
}
