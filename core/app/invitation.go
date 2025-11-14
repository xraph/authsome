package app

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// Invitation represents an app invitation DTO (Data Transfer Object)
type Invitation struct {
	ID        xid.ID                 `json:"id"`
	AppID     xid.ID                 `json:"appId"`
	Email     string                 `json:"email"`
	Role      MemberRole             `json:"role"`
	Token     string                 `json:"token"`
	InviterID xid.ID                 `json:"inviterId"`
	Status    InvitationStatus       `json:"status"`
	ExpiresAt time.Time              `json:"expiresAt"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Invitation DTO to a schema.Invitation model
func (inv *Invitation) ToSchema() *schema.Invitation {
	return &schema.Invitation{
		ID:        inv.ID,
		AppID:     inv.AppID,
		Email:     inv.Email,
		Role:      inv.Role,
		Token:     inv.Token,
		InviterID: inv.InviterID,
		Status:    inv.Status,
		ExpiresAt: inv.ExpiresAt,
		Metadata:  inv.Metadata,
		AuditableModel: schema.AuditableModel{
			CreatedAt: inv.CreatedAt,
			UpdatedAt: inv.UpdatedAt,
			DeletedAt: inv.DeletedAt,
		},
	}
}

// FromSchemaInvitation converts a schema.Invitation model to Invitation DTO
func FromSchemaInvitation(si *schema.Invitation) *Invitation {
	if si == nil {
		return nil
	}
	return &Invitation{
		ID:        si.ID,
		AppID:     si.AppID,
		Email:     si.Email,
		Role:      si.Role,
		Token:     si.Token,
		InviterID: si.InviterID,
		Status:    si.Status,
		ExpiresAt: si.ExpiresAt,
		Metadata:  si.Metadata,
		CreatedAt: si.CreatedAt,
		UpdatedAt: si.UpdatedAt,
		DeletedAt: si.DeletedAt,
	}
}

// FromSchemaInvitations converts a slice of schema.Invitation to Invitation DTOs
func FromSchemaInvitations(invitations []*schema.Invitation) []*Invitation {
	result := make([]*Invitation, len(invitations))
	for i, inv := range invitations {
		result[i] = FromSchemaInvitation(inv)
	}
	return result
}
