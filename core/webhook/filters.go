package webhook

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// ListWebhooksFilter represents filter parameters for listing webhooks
type ListWebhooksFilter struct {
	pagination.PaginationParams
	AppID         xid.ID  `json:"appId" query:"app_id"`
	EnvironmentID xid.ID  `json:"environmentId" query:"environment_id"`
	Enabled       *bool   `json:"enabled,omitempty" query:"enabled"`
	Event         *string `json:"event,omitempty" query:"event"` // Filter by specific event type
}

// ListEventsFilter represents filter parameters for listing events
type ListEventsFilter struct {
	pagination.PaginationParams
	AppID         xid.ID  `json:"appId" query:"app_id"`
	EnvironmentID xid.ID  `json:"environmentId" query:"environment_id"`
	Type          *string `json:"type,omitempty" query:"type"` // Filter by event type
}

// ListDeliveriesFilter represents filter parameters for listing deliveries
type ListDeliveriesFilter struct {
	pagination.PaginationParams
	WebhookID xid.ID  `json:"webhookId" query:"webhook_id"`
	Status    *string `json:"status,omitempty" query:"status"` // Filter by delivery status
}
