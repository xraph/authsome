package organization

import (
	"time"

	"github.com/rs/xid"
)

// Invitation represents an organization invitation
type Invitation struct {
	ID             xid.ID     `json:"id"`
	OrganizationID xid.ID     `json:"organizationId"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	InviterID      xid.ID     `json:"inviterId"`
	Token          string     `json:"token"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	AcceptedAt     *time.Time `json:"acceptedAt"`
	Status         string     `json:"status"` // pending, accepted, expired
	CreatedAt      time.Time  `json:"createdAt"`
}
