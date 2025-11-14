package webhook

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Webhook represents a webhook subscription (DTO)
type Webhook struct {
	ID            xid.ID            `json:"id"`
	AppID         xid.ID            `json:"appID"`
	EnvironmentID xid.ID            `json:"environmentID"`
	URL           string            `json:"url"`
	Events        []string          `json:"events"`
	Secret        string            `json:"-"` // Never expose in JSON
	Enabled       bool              `json:"enabled"`
	MaxRetries    int               `json:"maxRetries"`
	RetryBackoff  string            `json:"retryBackoff"` // "exponential" or "linear"
	Headers       map[string]string `json:"headers,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	LastDelivery  *time.Time        `json:"lastDelivery,omitempty"`
	FailureCount  int               `json:"failureCount"`
}

// ToSchema converts Webhook DTO to schema.Webhook
func (w *Webhook) ToSchema() *schema.Webhook {
	return &schema.Webhook{
		ID:            w.ID,
		AppID:         w.AppID,
		EnvironmentID: w.EnvironmentID,
		URL:           w.URL,
		Events:        w.Events,
		Secret:        w.Secret,
		Enabled:       w.Enabled,
		MaxRetries:    w.MaxRetries,
		RetryBackoff:  w.RetryBackoff,
		Headers:       w.Headers,
		LastDelivery:  w.LastDelivery,
		FailureCount:  w.FailureCount,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

// FromSchemaWebhook converts schema.Webhook to Webhook DTO
func FromSchemaWebhook(w *schema.Webhook) *Webhook {
	if w == nil {
		return nil
	}
	return &Webhook{
		ID:            w.ID,
		AppID:         w.AppID,
		EnvironmentID: w.EnvironmentID,
		URL:           w.URL,
		Events:        w.Events,
		Secret:        w.Secret,
		Enabled:       w.Enabled,
		MaxRetries:    w.MaxRetries,
		RetryBackoff:  w.RetryBackoff,
		Headers:       w.Headers,
		LastDelivery:  w.LastDelivery,
		FailureCount:  w.FailureCount,
		CreatedAt:     w.CreatedAt,
		UpdatedAt:     w.UpdatedAt,
	}
}

// FromSchemaWebhooks converts multiple schema.Webhook to Webhook DTOs
func FromSchemaWebhooks(webhooks []*schema.Webhook) []*Webhook {
	result := make([]*Webhook, len(webhooks))
	for i, w := range webhooks {
		result[i] = FromSchemaWebhook(w)
	}
	return result
}

// Event represents a webhook event (DTO)
type Event struct {
	ID            xid.ID                 `json:"id"`
	AppID         xid.ID                 `json:"appID"`
	EnvironmentID xid.ID                 `json:"environmentID"`
	Type          string                 `json:"type"`
	Data          map[string]interface{} `json:"data"`
	OccurredAt    time.Time              `json:"occurredAt"`
	CreatedAt     time.Time              `json:"createdAt"`
}

// ToSchema converts Event DTO to schema.Event
func (e *Event) ToSchema() *schema.Event {
	return &schema.Event{
		ID:            e.ID,
		AppID:         e.AppID,
		EnvironmentID: e.EnvironmentID,
		Type:          e.Type,
		Data:          e.Data,
		OccurredAt:    e.OccurredAt,
		CreatedAt:     e.CreatedAt,
	}
}

// FromSchemaEvent converts schema.Event to Event DTO
func FromSchemaEvent(e *schema.Event) *Event {
	if e == nil {
		return nil
	}
	return &Event{
		ID:            e.ID,
		AppID:         e.AppID,
		EnvironmentID: e.EnvironmentID,
		Type:          e.Type,
		Data:          e.Data,
		OccurredAt:    e.OccurredAt,
		CreatedAt:     e.CreatedAt,
	}
}

// FromSchemaEvents converts multiple schema.Event to Event DTOs
func FromSchemaEvents(events []*schema.Event) []*Event {
	result := make([]*Event, len(events))
	for i, e := range events {
		result[i] = FromSchemaEvent(e)
	}
	return result
}

// Delivery represents a webhook delivery attempt (DTO)
type Delivery struct {
	ID          xid.ID     `json:"id"`
	WebhookID   xid.ID     `json:"webhookID"`
	EventID     xid.ID     `json:"eventID"`
	Attempt     int        `json:"attempt"`
	Status      string     `json:"status"` // "pending", "delivered", "failed", "retrying"
	StatusCode  int        `json:"statusCode,omitempty"`
	Response    string     `json:"response,omitempty"`
	Error       string     `json:"error,omitempty"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// ToSchema converts Delivery DTO to schema.Delivery
func (d *Delivery) ToSchema() *schema.Delivery {
	statusCode := d.StatusCode
	errorStr := d.Error
	return &schema.Delivery{
		ID:            d.ID,
		WebhookID:     d.WebhookID,
		EventID:       d.EventID,
		AttemptNumber: d.Attempt,
		Status:        d.Status,
		StatusCode:    &statusCode,
		ResponseBody:  []byte(d.Response),
		Error:         &errorStr,
		DeliveredAt:   d.DeliveredAt,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// FromSchemaDelivery converts schema.Delivery to Delivery DTO
func FromSchemaDelivery(d *schema.Delivery) *Delivery {
	if d == nil {
		return nil
	}
	delivery := &Delivery{
		ID:          d.ID,
		WebhookID:   d.WebhookID,
		EventID:     d.EventID,
		Attempt:     d.AttemptNumber,
		Status:      d.Status,
		DeliveredAt: d.DeliveredAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if d.StatusCode != nil {
		delivery.StatusCode = *d.StatusCode
	}
	if d.ResponseBody != nil {
		delivery.Response = string(d.ResponseBody)
	}
	if d.Error != nil {
		delivery.Error = *d.Error
	}
	return delivery
}

// FromSchemaDeliveries converts multiple schema.Delivery to Delivery DTOs
func FromSchemaDeliveries(deliveries []*schema.Delivery) []*Delivery {
	result := make([]*Delivery, len(deliveries))
	for i, d := range deliveries {
		result[i] = FromSchemaDelivery(d)
	}
	return result
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	AppID         xid.ID            `json:"appID" validate:"required"`
	EnvironmentID xid.ID            `json:"environmentID" validate:"required"`
	URL           string            `json:"url" validate:"required,url"`
	Events        []string          `json:"events" validate:"required,min=1"`
	MaxRetries    int               `json:"maxRetries" validate:"min=0,max=10"`
	RetryBackoff  string            `json:"retryBackoff" validate:"oneof=exponential linear"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// UpdateWebhookRequest represents a request to update a webhook
type UpdateWebhookRequest struct {
	URL          *string           `json:"url,omitempty" validate:"omitempty,url"`
	Events       []string          `json:"events,omitempty" validate:"omitempty,min=1"`
	Enabled      *bool             `json:"enabled,omitempty"`
	MaxRetries   *int              `json:"maxRetries,omitempty" validate:"omitempty,min=0,max=10"`
	RetryBackoff *string           `json:"retryBackoff,omitempty" validate:"omitempty,oneof=exponential linear"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// ListWebhooksResponse is a type alias for paginated response
type ListWebhooksResponse = pagination.PageResponse[*Webhook]

// ListEventsResponse is a type alias for paginated response
type ListEventsResponse = pagination.PageResponse[*Event]

// ListDeliveriesResponse is a type alias for paginated response
type ListDeliveriesResponse = pagination.PageResponse[*Delivery]

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
