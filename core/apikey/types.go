package apikey

import "github.com/xraph/authsome/core/base"

// KeyType represents the type of API key.
type KeyType = base.KeyType

const (
	// KeyTypePublishable - Frontend-safe, identifies app, limited operations
	// Can be safely exposed in client-side code (browser, mobile apps)
	// Limited to read-only and session creation operations.
	KeyTypePublishable = base.KeyTypePublishable

	// KeyTypeSecret - Backend-only, full administrative privileges
	// Must be kept secret on server-side only
	// Has unrestricted access to all operations.
	KeyTypeSecret = base.KeyTypeSecret

	// KeyTypeRestricted - Backend-only, scoped to specific operations
	// Must be kept secret on server-side
	// Access limited to explicitly granted scopes.
	KeyTypeRestricted = base.KeyTypeRestricted
)

// KeyTypeScopes defines default scopes for each key type
// These are automatically granted based on key type.
var KeyTypeScopes = map[KeyType][]string{
	KeyTypePublishable: {
		"app:identify",    // Can identify which app is making request
		"sessions:create", // Can create user sessions (for auth flows)
		"users:verify",    // Can verify user tokens
		"public:read",     // Can read public data
	},
	KeyTypeSecret: {
		"admin:full", // Full administrative access to everything
	},
	KeyTypeRestricted: {
		// No default scopes - must be explicitly granted
	},
}

// SafePublicScopes defines scopes that are safe for publishable keys
// Only these scopes can be granted to pk_ keys.
var SafePublicScopes = map[string]bool{
	"app:identify":    true,
	"sessions:create": true,
	"sessions:verify": true,
	"users:verify":    true,
	"users:read":      true, // Read-only user data
	"public:read":     true,
	"webhooks:verify": true, // Can verify webhook signatures
}

// IsSafeForPublicKey checks if a scope is safe for publishable keys.
func IsSafeForPublicKey(scope string) bool {
	return SafePublicScopes[scope]
}
