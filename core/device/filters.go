package device

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListDevicesFilter represents filter parameters for listing devices.
type ListDevicesFilter struct {
	pagination.PaginationParams

	UserID xid.ID `json:"userId" query:"user_id"`
}
