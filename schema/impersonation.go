package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// ImpersonationSession represents an admin impersonating a user
// Updated for V2 architecture: App → Environment → Organization
type ImpersonationSession struct {
	AuditableModel
	bun.BaseModel `bun:"table:impersonation_sessions,alias:is"`

	// Core fields
	AppID           xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appID"`                       // Platform app (required)
	EnvironmentID   *xid.ID `bun:"environment_id,type:varchar(20)" json:"environmentID,omitempty"`     // Environment (optional)
	OrganizationID  *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"`   // User-created organization (optional)
	ImpersonatorID  xid.ID  `bun:"impersonator_id,notnull,type:varchar(20)" json:"impersonatorID"`     // Admin who is impersonating
	TargetUserID    xid.ID  `bun:"target_user_id,notnull,type:varchar(20)" json:"targetUserID"`        // User being impersonated
	OriginalSession *xid.ID `bun:"original_session,type:varchar(20)" json:"originalSession,omitempty"` // Admin's original session
	NewSessionID    *xid.ID `bun:"new_session_id,type:varchar(20)" json:"newSessionID,omitempty"`      // New session for impersonation
	SessionToken    string  `bun:"session_token,type:text" json:"-"`                                   // Session token for revocation (not exposed in JSON)

	// Metadata
	Reason       string            `bun:"reason,type:text" json:"reason"`                                 // Required: ticket/reason for impersonation
	IPAddress    string            `bun:"ip_address,type:varchar(45)" json:"ip_address"`                  // Admin's IP
	UserAgent    string            `bun:"user_agent,type:text" json:"user_agent"`                         // Admin's user agent
	Metadata     map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`                  // Additional context
	TicketNumber string            `bun:"ticket_number,type:varchar(100)" json:"ticket_number,omitempty"` // Support ticket reference

	// Status and lifecycle
	Active    bool       `bun:"active,notnull,default:true" json:"active"`               // Currently active
	ExpiresAt time.Time  `bun:"expires_at,notnull" json:"expires_at"`                    // Auto-logout time
	EndedAt   *time.Time `bun:"ended_at" json:"ended_at,omitempty"`                      // When impersonation ended
	EndReason string     `bun:"end_reason,type:varchar(50)" json:"end_reason,omitempty"` // manual, timeout, error

	// Relationships (for joins)
	Impersonator *User         `bun:"rel:belongs-to,join:impersonator_id=id" json:"impersonator,omitempty"`
	TargetUser   *User         `bun:"rel:belongs-to,join:target_user_id=id" json:"targetUser,omitempty"`
	App          *App          `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`                         // Platform app
	Environment  *Environment  `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`         // Environment (optional)
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`       // User-created org (optional)
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
// Updated for V2 architecture: App → Environment → Organization
type ImpersonationAuditEvent struct {
	bun.BaseModel `bun:"table:impersonation_audit,alias:ia"`

	ID              xid.ID            `bun:"id,pk,type:varchar(20)" json:"id"`
	ImpersonationID xid.ID            `bun:"impersonation_id,notnull,type:varchar(20)" json:"impersonationID"`
	AppID           xid.ID            `bun:"app_id,notnull,type:varchar(20)" json:"appID"`                     // Platform app (required)
	EnvironmentID   *xid.ID           `bun:"environment_id,type:varchar(20)" json:"environmentID,omitempty"`   // Environment (optional)
	OrganizationID  *xid.ID           `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // User-created organization (optional)
	EventType       string            `bun:"event_type,notnull,type:varchar(50)" json:"eventType"`             // started, ended, action_performed, expired
	Action          string            `bun:"action,type:varchar(100)" json:"action,omitempty"`                 // Specific action performed during impersonation
	Resource        string            `bun:"resource,type:varchar(255)" json:"resource,omitempty"`             // Resource accessed
	IPAddress       string            `bun:"ip_address,type:varchar(45)" json:"ipAddress"`
	UserAgent       string            `bun:"user_agent,type:text" json:"userAgent"`
	Details         map[string]string `bun:"details,type:jsonb" json:"details,omitempty"`
	CreatedAt       time.Time         `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`

	// Relationships
	ImpersonationSession *ImpersonationSession `bun:"rel:belongs-to,join:impersonation_id=id" json:"impersonationSession,omitempty"`
	App                  *App                  `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Environment          *Environment          `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`
	Organization         *Organization         `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}
