// Package organization defines the organization domain entity and its store interface.
// Organizations are user-created workspaces within an app (Clerk-style).
package organization

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Organization represents a user-created workspace within an app.
type Organization struct {
	ID        id.OrgID         `json:"id"`
	AppID     id.AppID         `json:"app_id"`
	EnvID     id.EnvironmentID `json:"env_id"`
	Name      string           `json:"name"`
	Slug      string           `json:"slug"`
	Logo      string           `json:"logo,omitempty"`
	Metadata  Metadata         `json:"metadata,omitempty"`
	CreatedBy id.UserID        `json:"created_by"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// Metadata holds arbitrary organization metadata as typed key-value pairs.
type Metadata map[string]string

// Member represents a user's membership in an organization.
type Member struct {
	ID        id.MemberID `json:"id"`
	OrgID     id.OrgID    `json:"org_id"`
	UserID    id.UserID   `json:"user_id"`
	Role      MemberRole  `json:"role"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// MemberRole defines the role of an organization member.
type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
)

// Invitation represents a pending organization invitation.
type Invitation struct {
	ID        id.InvitationID  `json:"id"`
	OrgID     id.OrgID         `json:"org_id"`
	Email     string           `json:"email"`
	Role      MemberRole       `json:"role"`
	InviterID id.UserID        `json:"inviter_id"`
	Status    InvitationStatus `json:"status"`
	Token     string           `json:"-"`
	ExpiresAt time.Time        `json:"expires_at"`
	CreatedAt time.Time        `json:"created_at"`
}

// InvitationStatus tracks the state of an invitation.
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationExpired  InvitationStatus = "expired"
	InvitationDeclined InvitationStatus = "declined"
)

// Team represents a team within an organization.
type Team struct {
	ID        id.TeamID `json:"id"`
	OrgID     id.OrgID  `json:"org_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
