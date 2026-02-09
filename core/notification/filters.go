package notification

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// =============================================================================
// FILTER TYPES FOR PAGINATION
// =============================================================================

// ListTemplatesFilter represents filter parameters for listing notification templates.
type ListTemplatesFilter struct {
	pagination.PaginationParams

	AppID    xid.ID            `json:"appId"              query:"app_id"`
	Type     *NotificationType `json:"type,omitempty"     query:"type"`
	Language *string           `json:"language,omitempty" query:"language"`
	Active   *bool             `json:"active,omitempty"   query:"active"`
}

// ListNotificationsFilter represents filter parameters for listing notifications.
type ListNotificationsFilter struct {
	pagination.PaginationParams

	AppID     xid.ID              `json:"appId"               query:"app_id"`
	Type      *NotificationType   `json:"type,omitempty"      query:"type"`
	Status    *NotificationStatus `json:"status,omitempty"    query:"status"`
	Recipient *string             `json:"recipient,omitempty" query:"recipient"`
}
