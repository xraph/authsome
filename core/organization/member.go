package organization

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// UserInfo contains basic user information for display purposes
type UserInfo struct {
	ID              xid.ID `json:"id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Image           string `json:"image"`
	Username        string `json:"username,omitempty"`
	DisplayUsername string `json:"displayUsername,omitempty"`
}

// Member represents an organization member entity DTO (Data Transfer Object)
// This is separate from schema.OrganizationMember to maintain proper separation of concerns
type Member struct {
	ID             xid.ID    `json:"id"`
	OrganizationID xid.ID    `json:"organizationID"`
	UserID         xid.ID    `json:"userID"`
	Role           string    `json:"role"`   // owner, admin, member
	Status         string    `json:"status"` // active, suspended, pending
	JoinedAt       time.Time `json:"joinedAt"`
	// User info (populated when listing)
	User *UserInfo `json:"user,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Member DTO to a schema.OrganizationMember model
func (m *Member) ToSchema() *schema.OrganizationMember {
	return &schema.OrganizationMember{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		UserID:         m.UserID,
		Role:           m.Role,
		Status:         m.Status,
		JoinedAt:       m.JoinedAt,
		AuditableModel: schema.AuditableModel{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			DeletedAt: m.DeletedAt,
		},
	}
}

// FromSchemaMember converts a schema.OrganizationMember model to Member DTO
func FromSchemaMember(sm *schema.OrganizationMember) *Member {
	if sm == nil {
		return nil
	}
	return &Member{
		ID:             sm.ID,
		OrganizationID: sm.OrganizationID,
		UserID:         sm.UserID,
		Role:           sm.Role,
		Status:         sm.Status,
		JoinedAt:       sm.JoinedAt,
		CreatedAt:      sm.CreatedAt,
		UpdatedAt:      sm.UpdatedAt,
		DeletedAt:      sm.DeletedAt,
	}
}

// FromSchemaMembers converts a slice of schema.OrganizationMember to Member DTOs
func FromSchemaMembers(members []*schema.OrganizationMember) []*Member {
	result := make([]*Member, len(members))
	for i, m := range members {
		result[i] = FromSchemaMember(m)
	}
	return result
}

// UpdateMemberRequest represents an update member request
type UpdateMemberRequest struct {
	Role   *string `json:"role,omitempty" validate:"omitempty,oneof=owner admin member"`
	Status *string `json:"status,omitempty" validate:"omitempty,oneof=active suspended pending"`
}

// InviteMemberRequest represents a request to invite a member to an organization
type InviteMemberRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=owner admin member"`
}
