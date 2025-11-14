package impersonation

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListSessionsFilter represents filter parameters for listing impersonation sessions
type ListSessionsFilter struct {
	pagination.PaginationParams
	AppID              xid.ID  `json:"appId" query:"app_id"`
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty" query:"user_organization_id"`
	ImpersonatorID     *xid.ID `json:"impersonatorId,omitempty" query:"impersonator_id"`
	TargetUserID       *xid.ID `json:"targetUserId,omitempty" query:"target_user_id"`
	ActiveOnly         *bool   `json:"activeOnly,omitempty" query:"active_only"`
}

// ListAuditEventsFilter represents filter parameters for listing audit events
type ListAuditEventsFilter struct {
	pagination.PaginationParams
	AppID              xid.ID     `json:"appId" query:"app_id"`
	UserOrganizationID *xid.ID    `json:"userOrganizationId,omitempty" query:"user_organization_id"`
	ImpersonationID    *xid.ID    `json:"impersonationId,omitempty" query:"impersonation_id"`
	ImpersonatorID     *xid.ID    `json:"impersonatorId,omitempty" query:"impersonator_id"`
	TargetUserID       *xid.ID    `json:"targetUserId,omitempty" query:"target_user_id"`
	EventType          *string    `json:"eventType,omitempty" query:"event_type"`
	Since              *time.Time `json:"since,omitempty" query:"since"`
	Until              *time.Time `json:"until,omitempty" query:"until"`
}
