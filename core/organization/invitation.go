package organization

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Invitation represents an organization invitation entity DTO (Data Transfer Object)
// This is separate from schema.OrganizationInvitation to maintain proper separation of concerns
type Invitation struct {
	ID             xid.ID     `json:"id"`
	OrganizationID xid.ID     `json:"organizationID"`
	Email          string     `json:"email"`
	Role           string     `json:"role"` // owner, admin, member
	InviterID      xid.ID     `json:"inviterID"`
	Token          string     `json:"token"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	AcceptedAt     *time.Time `json:"acceptedAt,omitempty"`
	Status         string     `json:"status"` // pending, accepted, expired, cancelled, declined
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Invitation DTO to a schema.OrganizationInvitation model
func (i *Invitation) ToSchema() *schema.OrganizationInvitation {
	return &schema.OrganizationInvitation{
		ID:             i.ID,
		OrganizationID: i.OrganizationID,
		Email:          i.Email,
		Role:           i.Role,
		InviterID:      i.InviterID,
		Token:          i.Token,
		ExpiresAt:      i.ExpiresAt,
		AcceptedAt:     i.AcceptedAt,
		Status:         i.Status,
		AuditableModel: schema.AuditableModel{
			CreatedAt: i.CreatedAt,
			UpdatedAt: i.UpdatedAt,
			DeletedAt: i.DeletedAt,
		},
	}
}

// FromSchemaInvitation converts a schema.OrganizationInvitation model to Invitation DTO
func FromSchemaInvitation(si *schema.OrganizationInvitation) *Invitation {
	if si == nil {
		return nil
	}
	return &Invitation{
		ID:             si.ID,
		OrganizationID: si.OrganizationID,
		Email:          si.Email,
		Role:           si.Role,
		InviterID:      si.InviterID,
		Token:          si.Token,
		ExpiresAt:      si.ExpiresAt,
		AcceptedAt:     si.AcceptedAt,
		Status:         si.Status,
		CreatedAt:      si.CreatedAt,
		UpdatedAt:      si.UpdatedAt,
		DeletedAt:      si.DeletedAt,
	}
}

// FromSchemaInvitations converts a slice of schema.OrganizationInvitation to Invitation DTOs
func FromSchemaInvitations(invitations []*schema.OrganizationInvitation) []*Invitation {
	result := make([]*Invitation, len(invitations))
	for i, inv := range invitations {
		result[i] = FromSchemaInvitation(inv)
	}
	return result
}
