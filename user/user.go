// Package user defines the user domain entity and its store interface.
package user

import (
	"strings"
	"time"

	"github.com/xraph/authsome/id"
)

// User represents an authenticated identity within an application.
type User struct {
	ID                id.UserID        `json:"id"`
	AppID             id.AppID         `json:"app_id"`
	EnvID             id.EnvironmentID `json:"env_id"`
	Email             string           `json:"email"`
	EmailVerified     bool             `json:"email_verified"`
	FirstName         string           `json:"first_name"`
	LastName          string           `json:"last_name"`
	Image             string           `json:"image,omitempty"`
	Username          string           `json:"username,omitempty"`
	DisplayUsername   string           `json:"display_username,omitempty"`
	Phone             string           `json:"phone,omitempty"`
	PhoneVerified     bool             `json:"phone_verified"`
	PasswordHash      string           `json:"-"`
	PasswordChangedAt *time.Time       `json:"password_changed_at,omitempty"`
	Banned            bool             `json:"banned"`
	BanReason         string           `json:"ban_reason,omitempty"`
	BanExpires        *time.Time       `json:"ban_expires,omitempty"`
	Metadata          Metadata         `json:"metadata,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	DeletedAt         *time.Time       `json:"deleted_at,omitempty"`
}

// Name returns the full name by joining FirstName and LastName.
func (u *User) Name() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Metadata holds arbitrary user metadata as typed key-value pairs.
type Metadata map[string]string

// Query holds query parameters for listing users.
type Query struct {
	AppID  id.AppID         `json:"app_id"`
	EnvID  id.EnvironmentID `json:"env_id,omitempty"`
	Email  string           `json:"email,omitempty"`
	Cursor string           `json:"cursor,omitempty"`
	Limit  int              `json:"limit,omitempty"`
}

// List is a paginated list of users.
type List struct {
	Users      []*User `json:"users"`
	NextCursor string  `json:"next_cursor,omitempty"`
	Total      int     `json:"total"`
}
