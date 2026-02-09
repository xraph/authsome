package webhook

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Repository defines the interface for webhook storage operations
// Following ISP - works with schema types.
type Repository interface {
	// Webhook operations
	CreateWebhook(ctx context.Context, webhook *schema.Webhook) error
	FindWebhookByID(ctx context.Context, id xid.ID) (*schema.Webhook, error)
	ListWebhooks(ctx context.Context, filter *ListWebhooksFilter) (*pagination.PageResponse[*schema.Webhook], error)
	FindWebhooksByAppAndEvent(ctx context.Context, appID xid.ID, envID xid.ID, eventType string) ([]*schema.Webhook, error)
	UpdateWebhook(ctx context.Context, webhook *schema.Webhook) error
	DeleteWebhook(ctx context.Context, id xid.ID) error
	UpdateFailureCount(ctx context.Context, id xid.ID, count int) error
	UpdateLastDelivery(ctx context.Context, id xid.ID, timestamp time.Time) error

	// Event operations
	CreateEvent(ctx context.Context, event *schema.Event) error
	FindEventByID(ctx context.Context, id xid.ID) (*schema.Event, error)
	ListEvents(ctx context.Context, filter *ListEventsFilter) (*pagination.PageResponse[*schema.Event], error)

	// Delivery operations
	CreateDelivery(ctx context.Context, delivery *schema.Delivery) error
	FindDeliveryByID(ctx context.Context, id xid.ID) (*schema.Delivery, error)
	ListDeliveries(ctx context.Context, filter *ListDeliveriesFilter) (*pagination.PageResponse[*schema.Delivery], error)
	UpdateDelivery(ctx context.Context, delivery *schema.Delivery) error
	FindPendingDeliveries(ctx context.Context, limit int) ([]*schema.Delivery, error)
}
