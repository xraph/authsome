package organization

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Team represents an organization team entity DTO (Data Transfer Object)
// This is separate from schema.OrganizationTeam to maintain proper separation of concerns
type Team struct {
	ID             xid.ID                 `json:"id"`
	OrganizationID xid.ID                 `json:"organizationID"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	// Provisioning tracking
	ProvisionedBy *string `json:"provisionedBy,omitempty"` // e.g., "scim"
	ExternalID    *string `json:"externalID,omitempty"`    // External system ID
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Team DTO to a schema.OrganizationTeam model
func (t *Team) ToSchema() *schema.OrganizationTeam {
	return &schema.OrganizationTeam{
		ID:             t.ID,
		OrganizationID: t.OrganizationID,
		Name:           t.Name,
		Description:    t.Description,
		Metadata:       t.Metadata,
		ProvisionedBy:  t.ProvisionedBy,
		ExternalID:     t.ExternalID,
		AuditableModel: schema.AuditableModel{
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
			DeletedAt: t.DeletedAt,
		},
	}
}

// FromSchemaTeam converts a schema.OrganizationTeam model to Team DTO
func FromSchemaTeam(st *schema.OrganizationTeam) *Team {
	if st == nil {
		return nil
	}
	return &Team{
		ID:             st.ID,
		OrganizationID: st.OrganizationID,
		Name:           st.Name,
		Description:    st.Description,
		Metadata:       st.Metadata,
		ProvisionedBy:  st.ProvisionedBy,
		ExternalID:     st.ExternalID,
		CreatedAt:      st.CreatedAt,
		UpdatedAt:      st.UpdatedAt,
		DeletedAt:      st.DeletedAt,
	}
}

// FromSchemaTeams converts a slice of schema.OrganizationTeam to Team DTOs
func FromSchemaTeams(teams []*schema.OrganizationTeam) []*Team {
	result := make([]*Team, len(teams))
	for i, t := range teams {
		result[i] = FromSchemaTeam(t)
	}
	return result
}

// TeamMember represents a team member entity DTO
type TeamMember struct {
	ID       xid.ID    `json:"id"`
	TeamID   xid.ID    `json:"teamID"`
	MemberID xid.ID    `json:"memberID"` // References OrganizationMember
	JoinedAt time.Time `json:"joinedAt"`
	// User info (populated when listing)
	User *UserInfo `json:"user,omitempty"`
	// Provisioning tracking
	ProvisionedBy *string `json:"provisionedBy,omitempty"` // e.g., "scim"
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the TeamMember DTO to a schema.OrganizationTeamMember model
func (tm *TeamMember) ToSchema() *schema.OrganizationTeamMember {
	return &schema.OrganizationTeamMember{
		ID:            tm.ID,
		TeamID:        tm.TeamID,
		MemberID:      tm.MemberID,
		JoinedAt:      tm.JoinedAt,
		ProvisionedBy: tm.ProvisionedBy,
		AuditableModel: schema.AuditableModel{
			CreatedAt: tm.CreatedAt,
			UpdatedAt: tm.UpdatedAt,
			DeletedAt: tm.DeletedAt,
		},
	}
}

// FromSchemaTeamMember converts a schema.OrganizationTeamMember model to TeamMember DTO
func FromSchemaTeamMember(stm *schema.OrganizationTeamMember) *TeamMember {
	if stm == nil {
		return nil
	}
	return &TeamMember{
		ID:            stm.ID,
		TeamID:        stm.TeamID,
		MemberID:      stm.MemberID,
		JoinedAt:      stm.JoinedAt,
		ProvisionedBy: stm.ProvisionedBy,
		CreatedAt:     stm.CreatedAt,
		UpdatedAt:     stm.UpdatedAt,
		DeletedAt:     stm.DeletedAt,
	}
}

// FromSchemaTeamMembers converts a slice of schema.OrganizationTeamMember to TeamMember DTOs
func FromSchemaTeamMembers(teamMembers []*schema.OrganizationTeamMember) []*TeamMember {
	result := make([]*TeamMember, len(teamMembers))
	for i, tm := range teamMembers {
		result[i] = FromSchemaTeamMember(tm)
	}
	return result
}

// CreateTeamRequest represents a request to create a team
type CreateTeamRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTeamRequest represents a request to update a team
type UpdateTeamRequest struct {
	Name        *string                `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
