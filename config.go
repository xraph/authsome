package authsome

import (
	forgedb "github.com/xraph/forge/extensions/database"
)

// Mode represents the operation mode
type Mode int

const (
	// ModeStandalone represents single-tenant mode
	ModeStandalone Mode = iota
	// ModeSaaS represents multi-tenant mode
	ModeSaaS
)

// Config represents the root configuration
type Config struct {
	// Mode determines if running in standalone or SaaS mode
	Mode Mode

	// BasePath is the base path for auth routes
	BasePath string

	// TrustedOrigins for CORS
	TrustedOrigins []string

	// Secret for signing tokens
	Secret string

	// RBACEnforce toggles handler-level RBAC enforcement (off by default)
	RBACEnforce bool

	// Database configuration - support for Forge database extension
	// DatabaseManager is the Forge database extension manager
	DatabaseManager *forgedb.DatabaseManager
	// DatabaseManagerName is the name of the database to use from the manager
	DatabaseManagerName string
	// UseForgeDI indicates whether to resolve database from Forge DI container
	UseForgeDI bool

	// DatabaseSchema specifies the PostgreSQL schema for AuthSome tables
	// Default: "" (uses database default, typically "public")
	// Example: "auth" will store all tables in the auth schema
	// Note: This is NOT for multi-tenancy, just organizational separation
	DatabaseSchema string
}
