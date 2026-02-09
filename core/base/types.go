package base

// KeyType represents the type of API key.
type KeyType string

const (
	// KeyTypePublishable - Frontend-safe, identifies app, limited operations
	// Can be safely exposed in client-side code (browser, mobile apps)
	// Limited to read-only and session creation operations.
	KeyTypePublishable KeyType = "pk"

	// KeyTypeSecret - Backend-only, full administrative privileges
	// Must be kept secret on server-side only
	// Has unrestricted access to all operations.
	KeyTypeSecret KeyType = "sk"

	// KeyTypeRestricted - Backend-only, scoped to specific operations
	// Must be kept secret on server-side
	// Access limited to explicitly granted scopes.
	KeyTypeRestricted KeyType = "rk"
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

// IsBackendOnly returns true if key must be used server-side only.
func (kt KeyType) IsBackendOnly() bool {
	return kt == KeyTypeSecret || kt == KeyTypeRestricted
}

// IsPublic returns true if key can be safely exposed in frontend.
func (kt KeyType) IsPublic() bool {
	return kt == KeyTypePublishable
}

// String returns the string representation of the key type.
func (kt KeyType) String() string {
	return string(kt)
}

// IsValid checks if the key type is valid.
func (kt KeyType) IsValid() bool {
	switch kt {
	case KeyTypePublishable, KeyTypeSecret, KeyTypeRestricted:
		return true
	}

	return false
}

// GetDefaultScopes returns the default scopes for this key type.
func (kt KeyType) GetDefaultScopes() []string {
	if scopes, ok := KeyTypeScopes[kt]; ok {
		return scopes
	}

	return []string{}
}

// IsSafeForPublicKey checks if a scope is safe for publishable keys.
func IsSafeForPublicKey(scope string) bool {
	return SafePublicScopes[scope]
}
