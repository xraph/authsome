package user

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
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

// FromSchemaUser converts a schema.User model to User DTO
func FromSchemaUser(su *schema.User) *User {
	if su == nil {
		return nil
	}

	var appID xid.ID
	if su.AppID != nil {
		appID = *su.AppID
	}

	return &User{
		ID:              su.ID,
		AppID:           appID,
		Email:           su.Email,
		EmailVerified:   su.EmailVerified,
		EmailVerifiedAt: su.EmailVerifiedAt,
		Name:            su.Name,
		Image:           su.Image,
		PasswordHash:    su.PasswordHash,
		Username:        su.Username,
		DisplayUsername: su.DisplayUsername,
		CreatedAt:       su.CreatedAt,
		UpdatedAt:       su.UpdatedAt,
		DeletedAt:       su.DeletedAt,
	}
}

// FromSchemaUsers converts a slice of schema.User to User DTOs
func FromSchemaUsers(users []*schema.User) []*User {
	result := make([]*User, len(users))
	for i, u := range users {
		result[i] = FromSchemaUser(u)
	}
	return result
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateUserRequest represents a create user request
type CreateUserRequest struct {
	AppID    xid.ID `json:"appId" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Name     string `json:"name"`
}

// UpdateUserRequest represents an update user request
type UpdateUserRequest struct {
	Name            *string `json:"name,omitempty"`
	Email           *string `json:"email,omitempty" validate:"omitempty,email"`
	EmailVerified   *bool   `json:"emailVerified,omitempty"`
	Image           *string `json:"image,omitempty"`
	Username        *string `json:"username,omitempty"`
	DisplayUsername *string `json:"displayUsername,omitempty"`
}

type ListUsersResponse = pagination.PageResponse[*User]
