package organization

import (
	"time"

	"github.com/rs/xid"
)

// Team represents a team within an organization
type Team struct {
	ID             xid.ID    `json:"id"`
	OrganizationID xid.ID    `json:"organizationId"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TeamMember represents a team member
type TeamMember struct {
	ID        xid.ID    `json:"id"`
	TeamID    xid.ID    `json:"teamId"`
	MemberID  xid.ID    `json:"memberId"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateTeamRequest represents a create team request
type CreateTeamRequest struct {
	OrganizationID xid.ID `json:"organizationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}
