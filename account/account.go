// Package account defines the account lifecycle domain (signup, signin, verification, password reset).
package account

import (
	"time"

	"github.com/xraph/authsome/id"
)

// Verification represents an email or phone verification token.
type Verification struct {
	ID        id.VerificationID `json:"id"`
	AppID     id.AppID          `json:"app_id"`
	EnvID     id.EnvironmentID  `json:"env_id"`
	UserID    id.UserID         `json:"user_id"`
	Token     string            `json:"-"`
	Type      VerificationType  `json:"type"`
	ExpiresAt time.Time         `json:"expires_at"`
	Consumed  bool              `json:"consumed"`
	CreatedAt time.Time         `json:"created_at"`
}

// VerificationType identifies the kind of verification.
type VerificationType string

const (
	VerificationEmail VerificationType = "email"
	VerificationPhone VerificationType = "phone"
)

// PasswordReset represents a password reset token.
type PasswordReset struct {
	ID        id.PasswordResetID `json:"id"`
	AppID     id.AppID           `json:"app_id"`
	EnvID     id.EnvironmentID   `json:"env_id"`
	UserID    id.UserID          `json:"user_id"`
	Token     string             `json:"-"`
	ExpiresAt time.Time          `json:"expires_at"`
	Consumed  bool               `json:"consumed"`
	CreatedAt time.Time          `json:"created_at"`
}

// SignUpRequest is the input for account registration.
type SignUpRequest struct {
	AppID     id.AppID          `json:"app_id"`
	EnvID     id.EnvironmentID  `json:"env_id"`
	Email     string            `json:"email"`
	Password  string            `json:"password"`
	FirstName string            `json:"first_name,omitempty"`
	LastName  string            `json:"last_name,omitempty"`
	Username  string            `json:"username,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"` // custom form fields
	IPAddress string            `json:"ip_address,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
}

// SignInRequest is the input for account authentication.
type SignInRequest struct {
	AppID     id.AppID         `json:"app_id"`
	EnvID     id.EnvironmentID `json:"env_id"`
	Email     string           `json:"email,omitempty"`
	Username  string           `json:"username,omitempty"`
	Password  string           `json:"password"`
	IPAddress string           `json:"ip_address,omitempty"`
	UserAgent string           `json:"user_agent,omitempty"`
}
