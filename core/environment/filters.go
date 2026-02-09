package environment

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListEnvironmentsFilter represents filter parameters for listing environments.
type ListEnvironmentsFilter struct {
	pagination.PaginationParams

	AppID     xid.ID  `json:"appId"               query:"app_id"`
	Type      *string `json:"type,omitempty"      query:"type"`
	Status    *string `json:"status,omitempty"    query:"status"`
	IsDefault *bool   `json:"isDefault,omitempty" query:"is_default"`
}

// ListPromotionsFilter represents filter parameters for listing promotions.
type ListPromotionsFilter struct {
	pagination.PaginationParams

	AppID       xid.ID  `json:"appId"                 query:"app_id"`
	SourceEnvID *xid.ID `json:"sourceEnvId,omitempty" query:"source_env_id"`
	TargetEnvID *xid.ID `json:"targetEnvId,omitempty" query:"target_env_id"`
	Status      *string `json:"status,omitempty"      query:"status"`
	PromotedBy  *xid.ID `json:"promotedBy,omitempty"  query:"promoted_by"`
}
