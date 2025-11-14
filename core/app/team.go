package app

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Team represents a team within an app DTO (Data Transfer Object)
type Team struct {
	ID          xid.ID                 `json:"id"`
	AppID       xid.ID                 `json:"appId"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Team DTO to a schema.Team model
func (t *Team) ToSchema() *schema.Team {
	return &schema.Team{
		ID:          t.ID,
		AppID:       t.AppID,
		Name:        t.Name,
		Description: t.Description,
		Metadata:    t.Metadata,
		AuditableModel: schema.AuditableModel{
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
			DeletedAt: t.DeletedAt,
		},
	}
}

// FromSchemaTeam converts a schema.Team model to Team DTO
func FromSchemaTeam(st *schema.Team) *Team {
	if st == nil {
		return nil
	}
	return &Team{
		ID:          st.ID,
		AppID:       st.AppID,
		Name:        st.Name,
		Description: st.Description,
		Metadata:    st.Metadata,
		CreatedAt:   st.CreatedAt,
		UpdatedAt:   st.UpdatedAt,
		DeletedAt:   st.DeletedAt,
	}
}

// FromSchemaTeams converts a slice of schema.Team to Team DTOs
func FromSchemaTeams(teams []*schema.Team) []*Team {
	result := make([]*Team, len(teams))
	for i, t := range teams {
		result[i] = FromSchemaTeam(t)
	}
	return result
}

// TeamMember represents a team member DTO (Data Transfer Object)
// Note: Role is on the Member entity, not TeamMember
type TeamMember struct {
	ID       xid.ID `json:"id"`
	TeamID   xid.ID `json:"teamId"`
	MemberID xid.ID `json:"memberId"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the TeamMember DTO to a schema.TeamMember model
func (tm *TeamMember) ToSchema() *schema.TeamMember {
	return &schema.TeamMember{
		ID:       tm.ID,
		TeamID:   tm.TeamID,
		MemberID: tm.MemberID,
		AuditableModel: schema.AuditableModel{
			CreatedAt: tm.CreatedAt,
			UpdatedAt: tm.UpdatedAt,
			DeletedAt: tm.DeletedAt,
		},
	}
}

// FromSchemaTeamMember converts a schema.TeamMember model to TeamMember DTO
func FromSchemaTeamMember(stm *schema.TeamMember) *TeamMember {
	if stm == nil {
		return nil
	}
	return &TeamMember{
		ID:        stm.ID,
		TeamID:    stm.TeamID,
		MemberID:  stm.MemberID,
		CreatedAt: stm.CreatedAt,
		UpdatedAt: stm.UpdatedAt,
		DeletedAt: stm.DeletedAt,
	}
}

// FromSchemaTeamMembers converts a slice of schema.TeamMember to TeamMember DTOs
func FromSchemaTeamMembers(members []*schema.TeamMember) []*TeamMember {
	result := make([]*TeamMember, len(members))
	for i, m := range members {
		result[i] = FromSchemaTeamMember(m)
	}
	return result
}

// CreateTeamRequest represents a create team request
type CreateTeamRequest struct {
	AppID       xid.ID                 `json:"appId"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTeamRequest represents an update team request
type UpdateTeamRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateMemberRequest represents an update member request
type UpdateMemberRequest struct {
	Role   *string `json:"role,omitempty"`
	Status *string `json:"status,omitempty"`
}
