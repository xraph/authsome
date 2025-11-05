package user

import (
	"github.com/rs/xid"
	"time"
)

// User represents a user entity
type User struct {
	ID              xid.ID
	Email           string
	EmailVerified   bool
	EmailVerifiedAt *time.Time
	Name            string
	Image           string
	PasswordHash    string
	Username        string
	DisplayUsername string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateUserRequest represents a create user request
type CreateUserRequest struct {
	Email    string
	Password string
	Name     string
}

// UpdateUserRequest represents an update user request
type UpdateUserRequest struct {
	Name            *string
	Email           *string
	EmailVerified   *bool
	Image           *string
	Username        *string
	DisplayUsername *string
}
