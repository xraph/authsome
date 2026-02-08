package types

import "github.com/xraph/go-utils/errs"

// Common errors
var (
	// User errors
	ErrUserNotFound       = errs.New("user not found")
	ErrUserAlreadyExists  = errs.New("user already exists")
	ErrEmailAlreadyExists = errs.New("email already exists")
	ErrInvalidCredentials = errs.New("invalid credentials")

	// Session errors
	ErrSessionNotFound = errs.New("session not found")
	ErrSessionExpired  = errs.New("session expired")
	ErrInvalidSession  = errs.New("invalid session")

	// Auth errors
	ErrUnauthorized     = errs.New("unauthorized")
	ErrForbidden        = errs.New("forbidden")
	ErrEmailNotVerified = errs.New("email not verified")

	// Organization errors
	ErrOrganizationNotFound  = errs.New("organization not found")
	ErrNotOrganizationMember = errs.New("not an organization member")

	// Generic errors
	ErrInvalidInput  = errs.New("invalid input")
	ErrInternalError = errs.New("internal error")
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
