package audit

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// AUDIT EVENT DTO (Data Transfer Object)
// =============================================================================

// Event represents an audit trail record DTO
// This is separate from schema.AuditEvent to maintain proper separation of concerns
type Event struct {
	ID        xid.ID    `json:"id"`
	UserID    *xid.ID   `json:"userId,omitempty"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
	Metadata  string    `json:"metadata,omitempty"` // JSON string or plain text
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToSchema converts the Event DTO to a schema.AuditEvent model
func (e *Event) ToSchema() *schema.AuditEvent {
	// Ensure auditable fields satisfy NOT NULL constraints
	createdBy := e.ID
	if e.UserID != nil {
		createdBy = *e.UserID
	}

	return &schema.AuditEvent{
		AuditableModel: schema.AuditableModel{
			CreatedBy: createdBy,
			UpdatedBy: createdBy,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
		},
		ID:        e.ID,
		UserID:    e.UserID,
		Action:    e.Action,
		Resource:  e.Resource,
		IPAddress: e.IPAddress,
		UserAgent: e.UserAgent,
		Metadata:  e.Metadata,
	}
}

// FromSchemaEvent converts a schema.AuditEvent model to Event DTO
func FromSchemaEvent(ae *schema.AuditEvent) *Event {
	if ae == nil {
		return nil
	}

	return &Event{
		ID:        ae.ID,
		UserID:    ae.UserID,
		Action:    ae.Action,
		Resource:  ae.Resource,
		IPAddress: ae.IPAddress,
		UserAgent: ae.UserAgent,
		Metadata:  ae.Metadata,
		CreatedAt: ae.CreatedAt,
		UpdatedAt: ae.UpdatedAt,
	}
}

// FromSchemaEvents converts a slice of schema.AuditEvent to Event DTOs
func FromSchemaEvents(events []*schema.AuditEvent) []*Event {
	result := make([]*Event, len(events))
	for i, e := range events {
		result[i] = FromSchemaEvent(e)
	}
	return result
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateEventRequest represents a request to create an audit event
type CreateEventRequest struct {
	UserID    *xid.ID `json:"userId,omitempty"`
	Action    string  `json:"action" validate:"required"`
	Resource  string  `json:"resource" validate:"required"`
	IPAddress string  `json:"ipAddress,omitempty"`
	UserAgent string  `json:"userAgent,omitempty"`
	Metadata  string  `json:"metadata,omitempty"`
}

// CreateEventResponse represents the response after creating an audit event
type CreateEventResponse struct {
	Event *Event `json:"event"`
}

// GetEventRequest represents a request to get an audit event by ID
type GetEventRequest struct {
	ID xid.ID `json:"id" validate:"required"`
}

// GetEventResponse represents the response for getting an audit event
type GetEventResponse struct {
	Event *Event `json:"event"`
}

// ListEventsResponse represents a paginated list of audit events
type ListEventsResponse = pagination.PageResponse[*Event]
