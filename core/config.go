package core

import (
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	forgedb "github.com/xraph/forge/extensions/database"
)

// Config represents the root configuration
type Config struct {
	// RequireEmailVerification requires email verification for all users
	RequireEmailVerification bool `json:"requireEmailVerification" yaml:"requireEmailVerification"`

	// BasePath is the base path for auth routes
	BasePath string `json:"basePath" yaml:"basePath"`

	// CORS configuration
	CORSEnabled    bool     `json:"corsEnabled" yaml:"corsEnabled"`       // Enable/disable CORS middleware (default: false)
	TrustedOrigins []string `json:"trustedOrigins" yaml:"trustedOrigins"` // Allowed origins for CORS

	// Secret for signing tokens
	Secret string `json:"secret" yaml:"secret"`

	// RBACEnforce toggles handler-level RBAC enforcement (off by default)
	RBACEnforce bool `json:"rbacEnforce" yaml:"rbacEnforce"`

	// SessionCookieName is the name of the session cookie (default: "authsome_session")
	// DEPRECATED: Use SessionCookie.Name instead. Kept for backward compatibility.
	SessionCookieName string `json:"sessionCookieName" yaml:"sessionCookieName"`

	// SessionCookie configures cookie-based session management
	// When enabled, authentication responses will automatically set session cookies
	// Apps can override this configuration via their metadata
	SessionCookie session.CookieConfig `json:"sessionCookie" yaml:"sessionCookie"`

	// SessionConfig configures session behavior (TTL, sliding window, refresh tokens)
	SessionConfig session.Config `json:"sessionConfig" yaml:"sessionConfig"`

	// UserConfig configures user service behavior (password requirements, etc.)
	UserConfig user.Config `json:"userConfig" yaml:"userConfig"`

	// Database configuration - support for Forge database extension
	// DatabaseManager is the Forge database extension manager
	DatabaseManager *forgedb.DatabaseManager `json:"databaseManager" yaml:"databaseManager"`
	// DatabaseManagerName is the name of the database to use from the manager
	DatabaseManagerName string `json:"databaseManagerName" yaml:"databaseManagerName"`
	// UseForgeDI indicates whether to resolve database from Forge DI container
	UseForgeDI bool `json:"useForgeDi" yaml:"useForgeDi"`

	// DatabaseSchema specifies the PostgreSQL schema for AuthSome tables
	// Default: "" (uses database default, typically "public")
	// Example: "auth" will store all tables in the auth schema
	// Note: This is NOT for multi-tenancy, just organizational separation
	DatabaseSchema string `json:"databaseSchema" yaml:"databaseSchema"`
}
