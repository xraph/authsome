package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// InvitationStatus represents the status of an invitation.
type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusExpired   InvitationStatus = "expired"
	InvitationStatusCancelled InvitationStatus = "cancelled"
	InvitationStatusDeclined  InvitationStatus = "declined"
)

// IsValid checks if the status is valid.
func (s InvitationStatus) IsValid() bool {
	switch s {
	case InvitationStatusPending, InvitationStatusAccepted, InvitationStatusExpired,
		InvitationStatusCancelled, InvitationStatusDeclined:
		return true
	}

	return false
}

// String returns the string representation.
func (s InvitationStatus) String() string {
	return string(s)
}

// Invitation represents the app invitation table.
type Invitation struct {
	AuditableModel `bun:",inline"`
	bun.BaseModel  `bun:"table:invitations,alias:inv"`

	ID         xid.ID           `bun:"id,pk,type:varchar(20)"              json:"id"`
	AppID      xid.ID           `bun:"app_id,notnull,type:varchar(20)"     json:"appID"`
	Email      string           `bun:"email,notnull"                       json:"email"`
	Role       MemberRole       `bun:"role,notnull"                        json:"role"`
	InviterID  xid.ID           `bun:"inviter_id,notnull,type:varchar(20)" json:"inviterID"`
	Token      string           `bun:"token,notnull,unique"                json:"token"`
	ExpiresAt  time.Time        `bun:"expires_at,notnull"                  json:"expiresAt"`
	AcceptedAt *time.Time       `bun:"accepted_at"                         json:"acceptedAt"`
	Status     InvitationStatus `bun:"status,notnull"                      json:"status"`
	Metadata   map[string]any   `bun:"metadata,type:jsonb"                 json:"metadata"`

	// Relations
	App *App `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
}
