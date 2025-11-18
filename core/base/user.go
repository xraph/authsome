package base

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// USER DTO (Data Transfer Object)
// =============================================================================

// User represents a user entity DTO
// This is separate from schema.User to maintain proper separation of concerns
type User struct {
	ID              xid.ID     `json:"id"`
	AppID           xid.ID     `json:"appId"`
	Email           string     `json:"email"`
	EmailVerified   bool       `json:"emailVerified"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt,omitempty"`
	Name            string     `json:"name"`
	Image           string     `json:"image,omitempty"`
	PasswordHash    string     `json:"-"` // Never expose in JSON
	Username        string     `json:"username"`
	DisplayUsername string     `json:"displayUsername,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the User DTO to a schema.User model
func (u *User) ToSchema() *schema.User {
	return &schema.User{
		ID:              u.ID,
		AppID:           &u.AppID,
		Email:           u.Email,
		EmailVerified:   u.EmailVerified,
		EmailVerifiedAt: u.EmailVerifiedAt,
		Name:            u.Name,
		Image:           u.Image,
		PasswordHash:    u.PasswordHash,
		Username:        u.Username,
		DisplayUsername: u.DisplayUsername,
		AuditableModel: schema.AuditableModel{
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
			DeletedAt: u.DeletedAt,
		},
	}
}
