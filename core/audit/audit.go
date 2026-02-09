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
// This is separate from schema.AuditEvent to maintain proper separation of concerns.
type Event struct {
	ID             xid.ID      `json:"id"`
	AppID          xid.ID      `json:"appId"`
	OrganizationID *xid.ID     `json:"organizationId,omitempty"`
	EnvironmentID  *xid.ID     `json:"environmentId,omitempty"`
	UserID         *xid.ID     `json:"userId,omitempty"`
	Action         string      `json:"action"`
	Resource       string      `json:"resource"`
	Source         AuditSource `json:"source"`
	IPAddress      string      `json:"ipAddress,omitempty"`
	UserAgent      string      `json:"userAgent,omitempty"`
	Metadata       string      `json:"metadata,omitempty"` // JSON string or plain text
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

// ToSchema converts the Event DTO to a schema.AuditEvent model.
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
		ID:             e.ID,
		AppID:          e.AppID,
		OrganizationID: e.OrganizationID,
		EnvironmentID:  e.EnvironmentID,
		UserID:         e.UserID,
		Action:         e.Action,
		Resource:       e.Resource,
		Source:         string(e.Source),
		IPAddress:      e.IPAddress,
		UserAgent:      e.UserAgent,
		Metadata:       e.Metadata,
	}
}

// FromSchemaEvent converts a schema.AuditEvent model to Event DTO.
func FromSchemaEvent(ae *schema.AuditEvent) *Event {
	if ae == nil {
		return nil
	}

	return &Event{
		ID:             ae.ID,
		AppID:          ae.AppID,
		OrganizationID: ae.OrganizationID,
		EnvironmentID:  ae.EnvironmentID,
		UserID:         ae.UserID,
		Action:         ae.Action,
		Resource:       ae.Resource,
		Source:         AuditSource(ae.Source),
		IPAddress:      ae.IPAddress,
		UserAgent:      ae.UserAgent,
		Metadata:       ae.Metadata,
		CreatedAt:      ae.CreatedAt,
		UpdatedAt:      ae.UpdatedAt,
	}
}

// FromSchemaEvents converts a slice of schema.AuditEvent to Event DTOs.
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

// CreateEventRequest represents a request to create an audit event.
type CreateEventRequest struct {
	AppID          xid.ID       `json:"appId,omitempty"`          // Optional - will be read from context if not provided
	OrganizationID *xid.ID      `json:"organizationId,omitempty"` // Optional - user-created organization
	EnvironmentID  *xid.ID      `json:"environmentId,omitempty"`  // Optional - will be read from context if not provided
	UserID         *xid.ID      `json:"userId,omitempty"`
	Action         string       `json:"action"                   validate:"required"`
	Resource       string       `json:"resource"                 validate:"required"`
	Source         *AuditSource `json:"source,omitempty"` // Optional - defaults to 'application' if not provided
	IPAddress      string       `json:"ipAddress,omitempty"`
	UserAgent      string       `json:"userAgent,omitempty"`
	Metadata       string       `json:"metadata,omitempty"`
}

// CreateEventResponse represents the response after creating an audit event.
type CreateEventResponse struct {
	Event *Event `json:"event"`
}

// GetEventRequest represents a request to get an audit event by ID.
type GetEventRequest struct {
	ID xid.ID `json:"id" validate:"required"`
}

// GetEventResponse represents the response for getting an audit event.
type GetEventResponse struct {
	Event *Event `json:"event"`
}

// ListEventsResponse represents a paginated list of audit events.
type ListEventsResponse = pagination.PageResponse[*Event]

// =============================================================================
// COUNT REQUEST/RESPONSE
// =============================================================================

// CountEventsRequest represents a request to count audit events.
type CountEventsRequest struct {
	Filter *ListEventsFilter `json:"filter,omitempty"`
}

// CountEventsResponse represents the response for counting audit events.
type CountEventsResponse struct {
	Count int64 `json:"count"`
}

// =============================================================================
// DELETE REQUEST/RESPONSE
// =============================================================================

// DeleteOlderThanRequest represents a request to delete old audit events.
type DeleteOlderThanRequest struct {
	Before time.Time     `json:"before"           validate:"required"`
	Filter *DeleteFilter `json:"filter,omitempty"`
}

// DeleteOlderThanResponse represents the response for deleting audit events.
type DeleteOlderThanResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}

// =============================================================================
// STATISTICS REQUEST/RESPONSE
// =============================================================================

// GetStatisticsByActionRequest represents a request to get action statistics.
type GetStatisticsByActionRequest struct {
	Filter *StatisticsFilter `json:"filter,omitempty"`
}

// GetStatisticsByActionResponse represents the response for action statistics.
type GetStatisticsByActionResponse struct {
	Statistics []*ActionStatistic `json:"statistics"`
	Total      int64              `json:"total"`
}

// GetStatisticsByResourceRequest represents a request to get resource statistics.
type GetStatisticsByResourceRequest struct {
	Filter *StatisticsFilter `json:"filter,omitempty"`
}

// GetStatisticsByResourceResponse represents the response for resource statistics.
type GetStatisticsByResourceResponse struct {
	Statistics []*ResourceStatistic `json:"statistics"`
	Total      int64                `json:"total"`
}

// GetStatisticsByUserRequest represents a request to get user statistics.
type GetStatisticsByUserRequest struct {
	Filter *StatisticsFilter `json:"filter,omitempty"`
}

// GetStatisticsByUserResponse represents the response for user statistics.
type GetStatisticsByUserResponse struct {
	Statistics []*UserStatistic `json:"statistics"`
	Total      int64            `json:"total"`
}

// =============================================================================
// OLDEST EVENT REQUEST/RESPONSE
// =============================================================================

// GetOldestEventRequest represents a request to get the oldest event.
type GetOldestEventRequest struct {
	Filter *ListEventsFilter `json:"filter,omitempty"`
}

// GetOldestEventResponse represents the response for the oldest event.
type GetOldestEventResponse struct {
	Event *Event `json:"event,omitempty"`
}
