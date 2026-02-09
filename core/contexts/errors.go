package contexts

import "errors"

// Context-related errors.
var (
	// ErrAppContextRequired is returned when app context is required but not found.
	ErrAppContextRequired = errors.New("app context is required")

	// ErrEnvironmentContextRequired is returned when environment context is required but not found.
	ErrEnvironmentContextRequired = errors.New("environment context is required")

	// ErrOrganizationContextRequired is returned when organization context is required but not found.
	ErrOrganizationContextRequired = errors.New("organization context is required")

	// ErrUserContextRequired is returned when user context is required but not found.
	ErrUserContextRequired = errors.New("user context is required")

	// ErrAuthContextRequired is returned when auth context is required but not found.
	ErrAuthContextRequired = errors.New("authentication context is required")

	// ErrUserAuthRequired is returned when user authentication is required.
	ErrUserAuthRequired = errors.New("user authentication is required")

	// ErrAPIKeyRequired is returned when API key authentication is required.
	ErrAPIKeyRequired = errors.New("API key authentication is required")

	// ErrInsufficientScope is returned when API key lacks required scope.
	ErrInsufficientScope = errors.New("insufficient API key scope")

	// ErrInsufficientPermission is returned when lacking required RBAC permission.
	ErrInsufficientPermission = errors.New("insufficient permission")
)
