package user

import (
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/base"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// USER DTO (Data Transfer Object)
// =============================================================================

// User represents a user entity DTO
// This is separate from schema.User to maintain proper separation of concerns.
type User = base.User

// FromSchemaUser converts a schema.User model to User DTO.
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

// FromSchemaUsers converts a slice of schema.User to User DTOs.
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

// CreateUserRequest represents a create user request.
type CreateUserRequest struct {
	AppID    xid.ID `json:"appId"    validate:"required"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Name     string `json:"name"`
}

// UpdateUserRequest represents an update user request.
type UpdateUserRequest struct {
	Name            *string `json:"name,omitempty"`
	Email           *string `json:"email,omitempty"           validate:"omitempty,email"`
	EmailVerified   *bool   `json:"emailVerified,omitempty"`
	Image           *string `json:"image,omitempty"`
	Username        *string `json:"username,omitempty"`
	DisplayUsername *string `json:"displayUsername,omitempty"`
}

type ListUsersResponse = pagination.PageResponse[*User]
