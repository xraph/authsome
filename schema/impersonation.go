package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// ImpersonationSession represents an admin impersonating a user
type ImpersonationSession struct {
	bun.BaseModel `bun:"table:impersonation_sessions,alias:is"`

	// Core fields
	ID              xid.ID     `bun:"id,pk,type:varchar(20)" json:"id"`
	OrganizationID  xid.ID     `bun:"organization_id,notnull,type:varchar(20)" json:"organization_id"`
	ImpersonatorID  xid.ID     `bun:"impersonator_id,notnull,type:varchar(20)" json:"impersonator_id"`     // Admin who is impersonating
	TargetUserID    xid.ID     `bun:"target_user_id,notnull,type:varchar(20)" json:"target_user_id"`       // User being impersonated
	OriginalSession *xid.ID    `bun:"original_session,type:varchar(20)" json:"original_session,omitempty"` // Admin's original session
	NewSessionID    *xid.ID    `bun:"new_session_id,type:varchar(20)" json:"new_session_id,omitempty"`     // New session for impersonation
	SessionToken    string     `bun:"session_token,type:text" json:"-"`                                     // Session token for revocation (not exposed in JSON)

	// Metadata
	Reason       string            `bun:"reason,type:text" json:"reason"`                      // Required: ticket/reason for impersonation
	IPAddress    string            `bun:"ip_address,type:varchar(45)" json:"ip_address"`      // Admin's IP
	UserAgent    string            `bun:"user_agent,type:text" json:"user_agent"`             // Admin's user agent
	Metadata     map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`      // Additional context
	TicketNumber string            `bun:"ticket_number,type:varchar(100)" json:"ticket_number,omitempty"` // Support ticket reference

	// Status and lifecycle
	Active    bool       `bun:"active,notnull,default:true" json:"active"`       // Currently active
	ExpiresAt time.Time  `bun:"expires_at,notnull" json:"expires_at"`            // Auto-logout time
	EndedAt   *time.Time `bun:"ended_at" json:"ended_at,omitempty"`              // When impersonation ended
	EndReason string     `bun:"end_reason,type:varchar(50)" json:"end_reason,omitempty"` // manual, timeout, error

	// Audit trail
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships (for joins)
	Impersonator *User `bun:"rel:belongs-to,join:impersonator_id=id" json:"impersonator,omitempty"`
	TargetUser   *User `bun:"rel:belongs-to,join:target_user_id=id" json:"target_user,omitempty"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}

// IsExpired checks if the impersonation session has expired
func (i *ImpersonationSession) IsExpired() bool {
	return time.Now().UTC().After(i.ExpiresAt)
}

// IsActive checks if the impersonation session is currently active
func (i *ImpersonationSession) IsActive() bool {
	return i.Active && !i.IsExpired() && i.EndedAt == nil
}

// ImpersonationAuditEvent represents a detailed audit log for impersonation events
type ImpersonationAuditEvent struct {
	bun.BaseModel `bun:"table:impersonation_audit,alias:ia"`

	ID                   xid.ID            `bun:"id,pk,type:varchar(20)" json:"id"`
	ImpersonationID      xid.ID            `bun:"impersonation_id,notnull,type:varchar(20)" json:"impersonation_id"`
	OrganizationID       xid.ID            `bun:"organization_id,notnull,type:varchar(20)" json:"organization_id"`
	EventType            string            `bun:"event_type,notnull,type:varchar(50)" json:"event_type"` // started, ended, action_performed, expired
	Action               string            `bun:"action,type:varchar(100)" json:"action,omitempty"`      // Specific action performed during impersonation
	Resource             string            `bun:"resource,type:varchar(255)" json:"resource,omitempty"`  // Resource accessed
	IPAddress            string            `bun:"ip_address,type:varchar(45)" json:"ip_address"`
	UserAgent            string            `bun:"user_agent,type:text" json:"user_agent"`
	Details              map[string]string `bun:"details,type:jsonb" json:"details,omitempty"`
	CreatedAt            time.Time         `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`

	// Relationships
	ImpersonationSession *ImpersonationSession `bun:"rel:belongs-to,join:impersonation_id=id" json:"impersonation_session,omitempty"`
}

