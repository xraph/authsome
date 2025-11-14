package impersonation

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// IMPERSONATION SESSION DTO (Data Transfer Object)
// =============================================================================

// ImpersonationSession represents an impersonation session DTO
// This is separate from schema.ImpersonationSession to maintain proper separation of concerns
type ImpersonationSession struct {
	ID                 xid.ID     `json:"id"`
	AppID              xid.ID     `json:"appId"`
	UserOrganizationID *xid.ID    `json:"userOrganizationId,omitempty"`
	ImpersonatorID     xid.ID     `json:"impersonatorId"`
	TargetUserID       xid.ID     `json:"targetUserId"`
	NewSessionID       *xid.ID    `json:"newSessionId,omitempty"`
	SessionToken       string     `json:"-"` // Never expose in JSON
	Reason             string     `json:"reason"`
	TicketNumber       string     `json:"ticketNumber,omitempty"`
	IPAddress          string     `json:"ipAddress,omitempty"`
	UserAgent          string     `json:"userAgent,omitempty"`
	Active             bool       `json:"active"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	EndedAt            *time.Time `json:"endedAt,omitempty"`
	EndReason          string     `json:"endReason,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the ImpersonationSession DTO to a schema.ImpersonationSession model
func (s *ImpersonationSession) ToSchema() *schema.ImpersonationSession {
	return &schema.ImpersonationSession{
		ID:                 s.ID,
		AppID:              s.AppID, // Maps to organization_id column in DB
		UserOrganizationID: s.UserOrganizationID,
		ImpersonatorID:     s.ImpersonatorID,
		TargetUserID:       s.TargetUserID,
		NewSessionID:       s.NewSessionID,
		SessionToken:       s.SessionToken,
		Reason:             s.Reason,
		TicketNumber:       s.TicketNumber,
		IPAddress:          s.IPAddress,
		UserAgent:          s.UserAgent,
		Active:             s.Active,
		ExpiresAt:          s.ExpiresAt,
		EndedAt:            s.EndedAt,
		EndReason:          s.EndReason,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}

// FromSchemaImpersonationSession converts a schema.ImpersonationSession model to ImpersonationSession DTO
func FromSchemaImpersonationSession(ss *schema.ImpersonationSession) *ImpersonationSession {
	if ss == nil {
		return nil
	}

	return &ImpersonationSession{
		ID:                 ss.ID,
		AppID:              ss.AppID, // Maps from organization_id column in DB
		UserOrganizationID: ss.UserOrganizationID,
		ImpersonatorID:     ss.ImpersonatorID,
		TargetUserID:       ss.TargetUserID,
		NewSessionID:       ss.NewSessionID,
		SessionToken:       ss.SessionToken,
		Reason:             ss.Reason,
		TicketNumber:       ss.TicketNumber,
		IPAddress:          ss.IPAddress,
		UserAgent:          ss.UserAgent,
		Active:             ss.Active,
		ExpiresAt:          ss.ExpiresAt,
		EndedAt:            ss.EndedAt,
		EndReason:          ss.EndReason,
		CreatedAt:          ss.CreatedAt,
		UpdatedAt:          ss.UpdatedAt,
	}
}

// FromSchemaImpersonationSessions converts a slice of schema.ImpersonationSession to ImpersonationSession DTOs
func FromSchemaImpersonationSessions(sessions []*schema.ImpersonationSession) []*ImpersonationSession {
	result := make([]*ImpersonationSession, len(sessions))
	for i, s := range sessions {
		result[i] = FromSchemaImpersonationSession(s)
	}
	return result
}

// =============================================================================
// IMPERSONATION SESSION METHODS
// =============================================================================

// IsActive checks if the impersonation session is active
func (s *ImpersonationSession) IsActive() bool {
	return s.Active && time.Now().UTC().Before(s.ExpiresAt) && s.EndedAt == nil
}

// IsExpired checks if the session has expired
func (s *ImpersonationSession) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsEnded checks if the session has been manually ended
func (s *ImpersonationSession) IsEnded() bool {
	return s.EndedAt != nil
}

// =============================================================================
// AUDIT EVENT DTO
// =============================================================================

// AuditEvent represents an impersonation audit event DTO
type AuditEvent struct {
	ID                 xid.ID            `json:"id"`
	ImpersonationID    xid.ID            `json:"impersonationId"`
	AppID              xid.ID            `json:"appId"`
	UserOrganizationID *xid.ID           `json:"userOrganizationId,omitempty"`
	EventType          string            `json:"eventType"`
	Action             string            `json:"action,omitempty"`
	Resource           string            `json:"resource,omitempty"`
	IPAddress          string            `json:"ipAddress"`
	UserAgent          string            `json:"userAgent"`
	Details            map[string]string `json:"details,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
}

// ToSchema converts the AuditEvent DTO to a schema.ImpersonationAuditEvent model
func (e *AuditEvent) ToSchema() *schema.ImpersonationAuditEvent {
	return &schema.ImpersonationAuditEvent{
		ID:                 e.ID,
		ImpersonationID:    e.ImpersonationID,
		AppID:              e.AppID, // Maps to organization_id column in DB
		UserOrganizationID: e.UserOrganizationID,
		EventType:          e.EventType,
		Action:             e.Action,
		Resource:           e.Resource,
		IPAddress:          e.IPAddress,
		UserAgent:          e.UserAgent,
		Details:            e.Details,
		CreatedAt:          e.CreatedAt,
	}
}

// FromSchemaAuditEvent converts a schema.ImpersonationAuditEvent model to AuditEvent DTO
func FromSchemaAuditEvent(se *schema.ImpersonationAuditEvent) *AuditEvent {
	if se == nil {
		return nil
	}

	return &AuditEvent{
		ID:                 se.ID,
		ImpersonationID:    se.ImpersonationID,
		AppID:              se.AppID, // Maps from organization_id column in DB
		UserOrganizationID: se.UserOrganizationID,
		EventType:          se.EventType,
		Action:             se.Action,
		Resource:           se.Resource,
		IPAddress:          se.IPAddress,
		UserAgent:          se.UserAgent,
		Details:            se.Details,
		CreatedAt:          se.CreatedAt,
	}
}

// FromSchemaAuditEvents converts a slice of schema.ImpersonationAuditEvent to AuditEvent DTOs
func FromSchemaAuditEvents(events []*schema.ImpersonationAuditEvent) []*AuditEvent {
	result := make([]*AuditEvent, len(events))
	for i, e := range events {
		result[i] = FromSchemaAuditEvent(e)
	}
	return result
}

// =============================================================================
// SESSION INFO DTO (Enriched with user data)
// =============================================================================

// SessionInfo represents enriched impersonation session information
type SessionInfo struct {
	ID                 xid.ID     `json:"id"`
	AppID              xid.ID     `json:"appId"`
	UserOrganizationID *xid.ID    `json:"userOrganizationId,omitempty"`
	ImpersonatorID     xid.ID     `json:"impersonatorId"`
	TargetUserID       xid.ID     `json:"targetUserId"`
	Reason             string     `json:"reason"`
	TicketNumber       string     `json:"ticketNumber,omitempty"`
	Active             bool       `json:"active"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	EndedAt            *time.Time `json:"endedAt,omitempty"`
	EndReason          string     `json:"endReason,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`

	// Enriched data
	ImpersonatorEmail string `json:"impersonatorEmail,omitempty"`
	ImpersonatorName  string `json:"impersonatorName,omitempty"`
	TargetEmail       string `json:"targetEmail,omitempty"`
	TargetName        string `json:"targetName,omitempty"`
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// StartRequest represents a request to start impersonation
type StartRequest struct {
	AppID              xid.ID  `json:"appId" validate:"required"`
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty"`
	ImpersonatorID     xid.ID  `json:"impersonatorId" validate:"required"`
	TargetUserID       xid.ID  `json:"targetUserId" validate:"required"`
	Reason             string  `json:"reason" validate:"required,min=10,max=500"`
	TicketNumber       string  `json:"ticketNumber,omitempty" validate:"max=100"`
	IPAddress          string  `json:"ipAddress,omitempty"`
	UserAgent          string  `json:"userAgent,omitempty"`
	DurationMinutes    int     `json:"durationMinutes,omitempty"`
}

// StartResponse represents the response after starting impersonation
type StartResponse struct {
	ImpersonationID xid.ID    `json:"impersonationId"`
	SessionID       xid.ID    `json:"sessionId"`
	SessionToken    string    `json:"sessionToken"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Message         string    `json:"message"`
}

// EndRequest represents a request to end impersonation
type EndRequest struct {
	ImpersonationID    xid.ID  `json:"impersonationId" validate:"required"`
	AppID              xid.ID  `json:"appId" validate:"required"`
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty"`
	ImpersonatorID     xid.ID  `json:"impersonatorId" validate:"required"`
	Reason             string  `json:"reason,omitempty"`
}

// EndResponse represents the response after ending impersonation
type EndResponse struct {
	Success         bool      `json:"success"`
	ImpersonationID xid.ID    `json:"impersonationId"`
	EndedAt         time.Time `json:"endedAt"`
	Message         string    `json:"message"`
}

// GetRequest represents a request to get an impersonation session
type GetRequest struct {
	ImpersonationID    xid.ID  `json:"impersonationId" validate:"required"`
	AppID              xid.ID  `json:"appId" validate:"required"`
	UserOrganizationID *xid.ID `json:"userOrganizationId,omitempty"`
}

// VerifyRequest represents a request to verify if a session is impersonating
type VerifyRequest struct {
	SessionID xid.ID `json:"sessionId" validate:"required"`
}

// VerifyResponse represents the response for verification
type VerifyResponse struct {
	IsImpersonating    bool       `json:"isImpersonating"`
	ImpersonationID    *xid.ID    `json:"impersonationId,omitempty"`
	AppID              *xid.ID    `json:"appId,omitempty"`
	UserOrganizationID *xid.ID    `json:"userOrganizationId,omitempty"`
	ImpersonatorID     *xid.ID    `json:"impersonatorId,omitempty"`
	TargetUserID       *xid.ID    `json:"targetUserId,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
}

// ListSessionsResponse represents paginated session response
type ListSessionsResponse = pagination.PageResponse[*ImpersonationSession]

// ListAuditEventsResponse represents paginated audit events response
type ListAuditEventsResponse = pagination.PageResponse[*AuditEvent]
