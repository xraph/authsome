package user

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListUsersFilter represents filter parameters for listing users
type ListUsersFilter struct {
	pagination.PaginationParams
	AppID         xid.ID  `json:"appId" query:"app_id"`
	EmailVerified *bool   `json:"emailVerified,omitempty" query:"email_verified"`
	Search        *string `json:"search,omitempty" query:"search"`
}

// CountUsersFilter represents filter parameters for counting users
type CountUsersFilter struct {
	AppID        xid.ID     `json:"appId"`
	CreatedSince *time.Time `json:"createdSince,omitempty"`
}
