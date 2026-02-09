package jwt

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListJWTKeysFilter represents filter parameters for listing JWT keys.
type ListJWTKeysFilter struct {
	pagination.PaginationParams

	AppID         xid.ID `json:"appId"                   query:"app_id"`
	IsPlatformKey *bool  `json:"isPlatformKey,omitempty" query:"is_platform_key"`
	Active        *bool  `json:"active,omitempty"        query:"active"`
}
