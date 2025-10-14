package organization

import (
	"time"

	"github.com/rs/xid"
)

// Member represents an organization member
type Member struct {
	ID             xid.ID    `json:"id"`
	OrganizationID xid.ID    `json:"organizationId"`
	UserID         xid.ID    `json:"userId"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// InviteMemberRequest represents an invite member request
type InviteMemberRequest struct {
	OrganizationID xid.ID `json:"organizationId"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	InviterID      xid.ID `json:"inviterId"`
}
