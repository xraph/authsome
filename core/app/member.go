package app

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Member represents an app member DTO (Data Transfer Object)
// This is separate from schema.Member to maintain proper separation of concerns.
type Member struct {
	ID       xid.ID       `json:"id"`
	AppID    xid.ID       `json:"appId"`
	UserID   xid.ID       `json:"userId"`
	Role     MemberRole   `json:"role"`
	Status   MemberStatus `json:"status"`
	JoinedAt time.Time    `json:"joinedAt"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Member DTO to a schema.Member model.
func (m *Member) ToSchema() *schema.Member {
	return &schema.Member{
		ID:       m.ID,
		AppID:    m.AppID,
		UserID:   m.UserID,
		Role:     m.Role,
		Status:   m.Status,
		JoinedAt: m.JoinedAt,
		AuditableModel: schema.AuditableModel{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			DeletedAt: m.DeletedAt,
		},
	}
}

// FromSchemaMember converts a schema.Member model to Member DTO.
func FromSchemaMember(sm *schema.Member) *Member {
	if sm == nil {
		return nil
	}

	return &Member{
		ID:        sm.ID,
		AppID:     sm.AppID,
		UserID:    sm.UserID,
		Role:      sm.Role,
		Status:    sm.Status,
		JoinedAt:  sm.JoinedAt,
		CreatedAt: sm.CreatedAt,
		UpdatedAt: sm.UpdatedAt,
		DeletedAt: sm.DeletedAt,
	}
}

// FromSchemaMembers converts a slice of schema.Member to Member DTOs.
func FromSchemaMembers(members []*schema.Member) []*Member {
	result := make([]*Member, len(members))
	for i, m := range members {
		result[i] = FromSchemaMember(m)
	}

	return result
}

// InviteMemberRequest represents an invite member request.
type InviteMemberRequest struct {
	AppID     xid.ID         `json:"appId"`
	Email     string         `json:"email"`
	Role      string         `json:"role"`
	InviterID xid.ID         `json:"inviterId"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
