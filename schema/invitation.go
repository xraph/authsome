package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// InvitationStatus represents the status of an invitation
type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusExpired   InvitationStatus = "expired"
	InvitationStatusCancelled InvitationStatus = "cancelled"
	InvitationStatusDeclined  InvitationStatus = "declined"
)

// IsValid checks if the status is valid
func (s InvitationStatus) IsValid() bool {
	switch s {
	case InvitationStatusPending, InvitationStatusAccepted, InvitationStatusExpired,
		InvitationStatusCancelled, InvitationStatusDeclined:
		return true
	}
	return false
}

// String returns the string representation
func (s InvitationStatus) String() string {
	return string(s)
}

// Invitation represents the app invitation table
type Invitation struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:invitations,alias:inv"`

	ID         xid.ID                 `json:"id" bun:"id,pk,type:varchar(20)"`
	AppID      xid.ID                 `json:"appID" bun:"organization_id,notnull,type:varchar(20)"` // Column still named organization_id for migration compatibility
	Email      string                 `json:"email" bun:"email,notnull"`
	Role       MemberRole             `json:"role" bun:"role,notnull"`
	InviterID  xid.ID                 `json:"inviterID" bun:"inviter_id,notnull,type:varchar(20)"`
	Token      string                 `json:"token" bun:"token,notnull,unique"`
	ExpiresAt  time.Time              `json:"expiresAt" bun:"expires_at,notnull"`
	AcceptedAt *time.Time             `json:"acceptedAt" bun:"accepted_at"`
	Status     InvitationStatus       `json:"status" bun:"status,notnull"`
	Metadata   map[string]interface{} `json:"metadata" bun:"metadata,type:jsonb"`

	// Relations
	App *App `json:"app,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}
