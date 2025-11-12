package impersonation

import (
	"time"

	"github.com/rs/xid"
)

// StartRequest represents a request to start impersonation
// Updated for V2 architecture: App → Environment → Organization
type StartRequest struct {
	AppID              xid.ID  `json:"app_id" validate:"required"`
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"` // Optional: user-created org context
	ImpersonatorID     xid.ID  `json:"impersonator_id" validate:"required"`
	TargetUserID       xid.ID  `json:"target_user_id" validate:"required"`
	Reason             string  `json:"reason" validate:"required,min=10,max=500"` // Required: minimum 10 chars
	TicketNumber       string  `json:"ticket_number,omitempty" validate:"max=100"`
	IPAddress          string  `json:"ip_address,omitempty"`
	UserAgent          string  `json:"user_agent,omitempty"`
	DurationMinutes    int     `json:"duration_minutes,omitempty"` // Custom duration, uses config default if not provided
}

// StartResponse represents the response after starting impersonation
type StartResponse struct {
	ImpersonationID xid.ID    `json:"impersonation_id"`
	SessionID       xid.ID    `json:"session_id"`
	SessionToken    string    `json:"session_token"`
	ExpiresAt       time.Time `json:"expires_at"`
	Message         string    `json:"message"`
}

// EndRequest represents a request to end impersonation
type EndRequest struct {
	ImpersonationID    xid.ID  `json:"impersonation_id" validate:"required"`
	AppID              xid.ID  `json:"app_id" validate:"required"`
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"`
	ImpersonatorID     xid.ID  `json:"impersonator_id" validate:"required"`
	Reason             string  `json:"reason,omitempty"` // manual, timeout, error
}

// EndResponse represents the response after ending impersonation
type EndResponse struct {
	Success         bool      `json:"success"`
	ImpersonationID xid.ID    `json:"impersonation_id"`
	EndedAt         time.Time `json:"ended_at"`
	Message         string    `json:"message"`
}

// ListRequest represents a request to list impersonation sessions
type ListRequest struct {
	AppID              xid.ID  `json:"app_id" validate:"required"`
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"` // Filter by user-created org
	ImpersonatorID     *xid.ID `json:"impersonator_id,omitempty"`      // Filter by impersonator
	TargetUserID       *xid.ID `json:"target_user_id,omitempty"`       // Filter by target user
	ActiveOnly         bool    `json:"active_only"`                    // Only show active sessions
	Limit              int     `json:"limit"`
	Offset             int     `json:"offset"`
}

// ListResponse represents the response for list requests
type ListResponse struct {
	Sessions []*SessionInfo `json:"sessions"`
	Total    int            `json:"total"`
	Limit    int            `json:"limit"`
	Offset   int            `json:"offset"`
}

// SessionInfo represents impersonation session information
type SessionInfo struct {
	ID                 xid.ID     `json:"id"`
	AppID              xid.ID     `json:"app_id"`
	UserOrganizationID *xid.ID    `json:"user_organization_id,omitempty"`
	ImpersonatorID     xid.ID     `json:"impersonator_id"`
	TargetUserID       xid.ID     `json:"target_user_id"`
	Reason             string     `json:"reason"`
	TicketNumber       string     `json:"ticket_number,omitempty"`
	Active             bool       `json:"active"`
	ExpiresAt          time.Time  `json:"expires_at"`
	EndedAt            *time.Time `json:"ended_at,omitempty"`
	EndReason          string     `json:"end_reason,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Enriched data
	ImpersonatorEmail string `json:"impersonator_email,omitempty"`
	ImpersonatorName  string `json:"impersonator_name,omitempty"`
	TargetEmail       string `json:"target_email,omitempty"`
	TargetName        string `json:"target_name,omitempty"`
}

// AuditListRequest represents a request to list audit events
type AuditListRequest struct {
	AppID              xid.ID     `json:"app_id" validate:"required"`
	UserOrganizationID *xid.ID    `json:"user_organization_id,omitempty"` // Filter by user-created org
	ImpersonationID    *xid.ID    `json:"impersonation_id,omitempty"`     // Filter by impersonation session
	ImpersonatorID     *xid.ID    `json:"impersonator_id,omitempty"`      // Filter by impersonator
	TargetUserID       *xid.ID    `json:"target_user_id,omitempty"`       // Filter by target user
	EventType          string     `json:"event_type,omitempty"`           // Filter by event type
	Since              *time.Time `json:"since,omitempty"`
	Until              *time.Time `json:"until,omitempty"`
	Limit              int        `json:"limit"`
	Offset             int        `json:"offset"`
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID                 xid.ID            `json:"id"`
	ImpersonationID    xid.ID            `json:"impersonation_id"`
	AppID              xid.ID            `json:"app_id"`
	UserOrganizationID *xid.ID           `json:"user_organization_id,omitempty"`
	EventType          string            `json:"event_type"`
	Action             string            `json:"action,omitempty"`
	Resource           string            `json:"resource,omitempty"`
	IPAddress          string            `json:"ip_address"`
	UserAgent          string            `json:"user_agent"`
	Details            map[string]string `json:"details,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
}

// GetRequest represents a request to get an impersonation session
type GetRequest struct {
	ImpersonationID    xid.ID  `json:"impersonation_id" validate:"required"`
	AppID              xid.ID  `json:"app_id" validate:"required"`
	UserOrganizationID *xid.ID `json:"user_organization_id,omitempty"`
}

// VerifyRequest represents a request to verify if a session is impersonating
type VerifyRequest struct {
	SessionID xid.ID `json:"session_id" validate:"required"`
}

// VerifyResponse represents the response for verification
type VerifyResponse struct {
	IsImpersonating    bool       `json:"is_impersonating"`
	ImpersonationID    *xid.ID    `json:"impersonation_id,omitempty"`
	AppID              *xid.ID    `json:"app_id,omitempty"`
	UserOrganizationID *xid.ID    `json:"user_organization_id,omitempty"`
	ImpersonatorID     *xid.ID    `json:"impersonator_id,omitempty"`
	TargetUserID       *xid.ID    `json:"target_user_id,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}
