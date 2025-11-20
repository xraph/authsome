package responses

import (
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
)

// Base response types used across handlers and plugins

// ErrorResponse is an alias to errs.AuthsomeError for consistency across the codebase.
// Instead of using &ErrorResponse{Error: "message"}, use the errs package constructors:
//   - errs.New(code, message, httpStatus)
//   - errs.BadRequest(message)
//   - errs.Unauthorized()
//   - errs.NotFound(message)
//   - etc.
//
// See internal/errs/errors.go for all available error constructors.
type ErrorResponse = errs.AuthsomeError

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}

// StatusResponse represents a status response
type StatusResponse struct {
	Status string `json:"status"`
}

// SuccessResponse represents a success boolean response
type SuccessResponse struct {
	Success bool `json:"success"`
}

// Auth-specific responses

// AuthResponse represents a successful authentication response with user, session, and token
type AuthResponse struct {
	User         *user.User       `json:"user"`
	Session      *session.Session `json:"session"`
	Token        string           `json:"token"`
	RequireTwoFA bool             `json:"requireTwofa,omitempty"`
}

// TwoFARequiredResponse indicates that two-factor authentication is required
type TwoFARequiredResponse struct {
	User         *user.User `json:"user"`
	RequireTwoFA bool       `json:"requireTwofa"`
	DeviceID     string     `json:"deviceId,omitempty"`
}

// VerifyResponse represents a verification response (used by emailotp, magiclink, phone plugins)
// Uses interface{} for flexibility across different plugin implementations
type VerifyResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token"`
}

// SessionResponse represents a session query response
type SessionResponse struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
}
