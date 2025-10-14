package types

import "errors"

// Common errors
var (
    // User errors
    ErrUserNotFound          = errors.New("user not found")
    ErrUserAlreadyExists     = errors.New("user already exists")
    ErrEmailAlreadyExists    = errors.New("email already exists")
    ErrInvalidCredentials    = errors.New("invalid credentials")

    // Session errors
    ErrSessionNotFound       = errors.New("session not found")
    ErrSessionExpired        = errors.New("session expired")
    ErrInvalidSession        = errors.New("invalid session")

    // Auth errors
    ErrUnauthorized          = errors.New("unauthorized")
    ErrForbidden             = errors.New("forbidden")
    ErrEmailNotVerified      = errors.New("email not verified")

    // Organization errors
    ErrOrganizationNotFound  = errors.New("organization not found")
    ErrNotOrganizationMember = errors.New("not an organization member")

    // Generic errors
    ErrInvalidInput          = errors.New("invalid input")
    ErrInternalError         = errors.New("internal error")
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