package webhook

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// Webhook represents a webhook subscription
type Webhook struct {
	ID             xid.ID            `json:"id"`
	OrganizationID string            `json:"organization_id"`
	URL            string            `json:"url"`
	Events         []string          `json:"events"`
	Secret         string            `json:"-"` // Never expose in JSON
	Enabled        bool              `json:"enabled"`
	MaxRetries     int               `json:"max_retries"`
	RetryBackoff   string            `json:"retry_backoff"` // "exponential" or "linear"
	Headers        map[string]string `json:"headers,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastDelivery   *time.Time        `json:"last_delivery,omitempty"`
	FailureCount   int               `json:"failure_count"`
}

// Event represents a webhook event
type Event struct {
	ID             xid.ID                 `json:"id"`
	Type           string                 `json:"type"`
	OrganizationID string                 `json:"organization_id"`
	Data           map[string]interface{} `json:"data"`
	OccurredAt     time.Time              `json:"occurred_at"`
	CreatedAt      time.Time              `json:"created_at"`
}

// Delivery represents a webhook delivery attempt
type Delivery struct {
	ID          xid.ID     `json:"id"`
	WebhookID   xid.ID     `json:"webhook_id"`
	EventID     xid.ID     `json:"event_id"`
	Attempt     int        `json:"attempt"`
	Status      string     `json:"status"` // "pending", "delivered", "failed", "retrying"
	StatusCode  int        `json:"status_code,omitempty"`
	Response    string     `json:"response,omitempty"`
	Error       string     `json:"error,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	OrganizationID string            `json:"organization_id" validate:"required"`
	URL            string            `json:"url" validate:"required,url"`
	Events         []string          `json:"events" validate:"required,min=1"`
	MaxRetries     int               `json:"max_retries" validate:"min=0,max=10"`
	RetryBackoff   string            `json:"retry_backoff" validate:"oneof=exponential linear"`
	Headers        map[string]string `json:"headers,omitempty"`
}

// UpdateWebhookRequest represents a request to update a webhook
type UpdateWebhookRequest struct {
	URL          *string           `json:"url,omitempty" validate:"omitempty,url"`
	Events       []string          `json:"events,omitempty" validate:"omitempty,min=1"`
	Enabled      *bool             `json:"enabled,omitempty"`
	MaxRetries   *int              `json:"max_retries,omitempty" validate:"omitempty,min=0,max=10"`
	RetryBackoff *string           `json:"retry_backoff,omitempty" validate:"omitempty,oneof=exponential linear"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// ListWebhooksRequest represents a request to list webhooks
type ListWebhooksRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
	Page           int    `json:"page" validate:"min=1"`
	PageSize       int    `json:"page_size" validate:"min=1,max=100"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

// ListWebhooksResponse represents the response from listing webhooks
type ListWebhooksResponse struct {
	Webhooks   []*Webhook `json:"webhooks"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// ListDeliveriesRequest represents a request to list webhook deliveries
type ListDeliveriesRequest struct {
	WebhookID xid.ID `json:"webhook_id" validate:"required"`
	Page      int    `json:"page" validate:"min=1"`
	PageSize  int    `json:"page_size" validate:"min=1,max=100"`
	Status    string `json:"status,omitempty" validate:"omitempty,oneof=pending delivered failed retrying"`
}

// ListDeliveriesResponse represents the response from listing deliveries
type ListDeliveriesResponse struct {
	Deliveries []*Delivery `json:"deliveries"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// Repository defines the interface for webhook storage operations
type Repository interface {
	// Webhook operations
	Create(ctx context.Context, webhook *Webhook) error
	FindByID(ctx context.Context, id xid.ID) (*Webhook, error)
	FindByOrgID(ctx context.Context, orgID string, enabled *bool, offset, limit int) ([]*Webhook, int64, error)
	FindByOrgAndEvent(ctx context.Context, orgID, eventType string) ([]*Webhook, error)
	Update(ctx context.Context, webhook *Webhook) error
	Delete(ctx context.Context, id xid.ID) error
	UpdateFailureCount(ctx context.Context, id xid.ID, count int) error
	UpdateLastDelivery(ctx context.Context, id xid.ID, timestamp time.Time) error

	// Event operations
	CreateEvent(ctx context.Context, event *Event) error
	FindEventByID(ctx context.Context, id xid.ID) (*Event, error)
	ListEvents(ctx context.Context, orgID string, offset, limit int) ([]*Event, int64, error)

	// Delivery operations
	CreateDelivery(ctx context.Context, delivery *Delivery) error
	FindDeliveryByID(ctx context.Context, id xid.ID) (*Delivery, error)
	FindDeliveriesByWebhook(ctx context.Context, webhookID xid.ID, status string, offset, limit int) ([]*Delivery, int64, error)
	UpdateDelivery(ctx context.Context, delivery *Delivery) error
	FindPendingDeliveries(ctx context.Context, limit int) ([]*Delivery, error)
}

// EventType constants for webhook events
const (
	EventUserCreated       = "user.created"
	EventUserUpdated       = "user.updated"
	EventUserDeleted       = "user.deleted"
	EventUserLogin         = "user.login"
	EventUserLogout        = "user.logout"
	EventSessionCreated    = "session.created"
	EventSessionRevoked    = "session.revoked"
	EventOrgCreated        = "organization.created"
	EventOrgUpdated        = "organization.updated"
	EventMemberAdded       = "member.added"
	EventMemberRemoved     = "member.removed"
	EventMemberRoleChanged = "member.role_changed"
)

// AllEventTypes returns all available event types
func AllEventTypes() []string {
	return []string{
		EventUserCreated,
		EventUserUpdated,
		EventUserDeleted,
		EventUserLogin,
		EventUserLogout,
		EventSessionCreated,
		EventSessionRevoked,
		EventOrgCreated,
		EventOrgUpdated,
		EventMemberAdded,
		EventMemberRemoved,
		EventMemberRoleChanged,
	}
}

// IsValidEventType checks if an event type is valid
func IsValidEventType(eventType string) bool {
	for _, validType := range AllEventTypes() {
		if eventType == validType {
			return true
		}
	}
	return false
}

// DeliveryStatus constants
const (
	DeliveryStatusPending   = "pending"
	DeliveryStatusDelivered = "delivered"
	DeliveryStatusFailed    = "failed"
	DeliveryStatusRetrying  = "retrying"
)

// RetryBackoff constants
const (
	RetryBackoffExponential = "exponential"
	RetryBackoffLinear      = "linear"
)
