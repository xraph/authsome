package contexts

import "errors"

// Context-related errors
var (
	// ErrAppContextRequired is returned when app context is required but not found
	ErrAppContextRequired = errors.New("app context is required")

	// ErrEnvironmentContextRequired is returned when environment context is required but not found
	ErrEnvironmentContextRequired = errors.New("environment context is required")

	// ErrOrganizationContextRequired is returned when organization context is required but not found
	ErrOrganizationContextRequired = errors.New("organization context is required")

	// ErrUserContextRequired is returned when user context is required but not found
	ErrUserContextRequired = errors.New("user context is required")
)
