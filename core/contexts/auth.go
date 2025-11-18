package contexts

import (
	"context"

	"github.com/rs/xid"
	base "github.com/xraph/authsome/core/base"
)

// AuthMethod indicates how the request was authenticated
type AuthMethod string

const (
	AuthMethodNone    AuthMethod = "none"
	AuthMethodSession AuthMethod = "session"
	AuthMethodAPIKey  AuthMethod = "apikey"
	AuthMethodBoth    AuthMethod = "both"
)

// AuthContext holds complete authentication state for a request
// This provides a unified view of both API key (app) authentication
// and user session authentication, following production patterns like Clerk
type AuthContext struct {
	// Platform/App Authentication (via API key)
	APIKey       *base.APIKey `json:"apiKey,omitempty"`
	APIKeyScopes []string     `json:"apiKeyScopes,omitempty"`

	// End-User Authentication (via session/bearer token)
	Session *base.Session `json:"session,omitempty"`
	User    *base.User    `json:"user,omitempty"`

	// Resolved Context (from either API key or session)
	AppID          xid.ID  `json:"appID"`
	EnvironmentID  xid.ID  `json:"environmentID"`
	OrganizationID *xid.ID `json:"organizationID,omitempty"`

	// Authentication Metadata
	Method          AuthMethod `json:"method"`
	IsAuthenticated bool       `json:"isAuthenticated"`
	IsAPIKeyAuth    bool       `json:"isAPIKeyAuth"`
	IsUserAuth      bool       `json:"isUserAuth"`

	// Security Metadata
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent"`

	// RBAC Integration (Hybrid Approach)
	APIKeyRoles        []string `json:"apiKeyRoles,omitempty"`        // Roles assigned to API key
	APIKeyPermissions  []string `json:"apiKeyPermissions,omitempty"`  // Permissions from API key roles
	CreatorPermissions []string `json:"creatorPermissions,omitempty"` // Permissions from key creator (if delegated)
	UserRoles          []string `json:"userRoles,omitempty"`          // Roles from session user
	UserPermissions    []string `json:"userPermissions,omitempty"`    // Permissions from session user roles

	// Effective (computed) permissions - union of all applicable permissions
	EffectivePermissions []string `json:"effectivePermissions,omitempty"`
}

// Context key for storing auth context
type authContextKey struct{}

// =============================================================================
// CONTEXT STORAGE AND RETRIEVAL
// =============================================================================

// SetAuthContext stores the auth context in the request context
func SetAuthContext(ctx context.Context, ac *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, ac)
}

// GetAuthContext retrieves the auth context from the request context
func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	ac, ok := ctx.Value(authContextKey{}).(*AuthContext)
	return ac, ok
}

// RequireAuthContext retrieves auth context or returns error
func RequireAuthContext(ctx context.Context) (*AuthContext, error) {
	ac, ok := GetAuthContext(ctx)
	if !ok || ac == nil {
		return nil, ErrAuthContextRequired
	}
	return ac, nil
}

// RequireUser ensures a user is authenticated
func RequireUser(ctx context.Context) (*base.User, error) {
	ac, err := RequireAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	if ac.User == nil {
		return nil, ErrUserAuthRequired
	}
	return ac.User, nil
}

// RequireAPIKey ensures an API key is present
func RequireAPIKey(ctx context.Context) (*base.APIKey, error) {
	ac, err := RequireAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	if ac.APIKey == nil {
		return nil, ErrAPIKeyRequired
	}
	return ac.APIKey, nil
}

// GetUser safely retrieves the user from context (returns nil if not present)
func GetUser(ctx context.Context) *base.User {
	ac, ok := GetAuthContext(ctx)
	if !ok || ac == nil {
		return nil
	}
	return ac.User
}

// GetAPIKey safely retrieves the API key from context (returns nil if not present)
func GetAPIKey(ctx context.Context) *base.APIKey {
	ac, ok := GetAuthContext(ctx)
	if !ok || ac == nil {
		return nil
	}
	return ac.APIKey
}

// GetSession safely retrieves the session from context (returns nil if not present)
func GetSession(ctx context.Context) *base.Session {
	ac, ok := GetAuthContext(ctx)
	if !ok || ac == nil {
		return nil
	}
	return ac.Session
}

// =============================================================================
// AUTH CONTEXT HELPER METHODS
// =============================================================================

// HasAPIKey returns true if authenticated via API key
func (ac *AuthContext) HasAPIKey() bool {
	return ac.APIKey != nil
}

// HasSession returns true if authenticated via user session
func (ac *AuthContext) HasSession() bool {
	return ac.Session != nil && ac.User != nil
}

// HasScope checks if the API key has a specific scope
func (ac *AuthContext) HasScope(scope string) bool {
	if ac.APIKey == nil {
		return false
	}
	return ac.APIKey.HasScope(scope)
}

// HasAnyScopeOf checks if the API key has any of the specified scopes
func (ac *AuthContext) HasAnyScopeOf(scopes ...string) bool {
	if ac.APIKey == nil {
		return false
	}
	for _, scope := range scopes {
		if ac.APIKey.HasScope(scope) {
			return true
		}
	}
	return false
}

// HasAllScopesOf checks if the API key has all of the specified scopes
func (ac *AuthContext) HasAllScopesOf(scopes ...string) bool {
	if ac.APIKey == nil {
		return false
	}
	for _, scope := range scopes {
		if !ac.APIKey.HasScope(scope) {
			return false
		}
	}
	return true
}

// RequireScope ensures the API key has a specific scope
func (ac *AuthContext) RequireScope(scope string) error {
	if !ac.HasScope(scope) {
		return ErrInsufficientScope
	}
	return nil
}

// IsAdmin returns true if the API key has admin privileges
func (ac *AuthContext) IsAdmin() bool {
	return ac.HasScope("admin:full")
}

// CanPerformAdminOp returns true if can perform admin operations
// Must have secret key with admin scope
func (ac *AuthContext) CanPerformAdminOp() bool {
	return ac.HasAPIKey() && ac.APIKey.IsSecret() && ac.IsAdmin()
}

// GetUserOrAPIKeyUser returns the session user or nil
// In production auth systems, the session user takes precedence
func (ac *AuthContext) GetUserOrAPIKeyUser() *base.User {
	// Always prefer session user over API key creator
	if ac.User != nil {
		return ac.User
	}
	// API key authentication doesn't directly provide a user
	// The user would need to be loaded separately if needed
	return nil
}

// GetEffectiveOrgID returns the organization ID to use for the request
// Priority: Session org > API key org
func (ac *AuthContext) GetEffectiveOrgID() *xid.ID {
	// Priority 1: Session organization (user's current org)
	if ac.Session != nil && ac.Session.OrganizationID != nil {
		return ac.Session.OrganizationID
	}
	// Priority 2: API key organization (app/key scoped org)
	if ac.APIKey != nil && ac.APIKey.OrganizationID != nil {
		return ac.APIKey.OrganizationID
	}
	return nil
}

// GetEffectiveAppID returns the app ID to use for the request
// Priority: API key app > Session app
func (ac *AuthContext) GetEffectiveAppID() xid.ID {
	// Priority 1: API key app (explicit app context)
	if ac.HasAPIKey() {
		return ac.APIKey.AppID
	}
	// Priority 2: Session app
	if ac.HasSession() {
		return ac.Session.AppID
	}
	// Fallback to stored app ID
	return ac.AppID
}

// GetEffectiveEnvironmentID returns the environment ID to use
// Priority: API key env > Session env
func (ac *AuthContext) GetEffectiveEnvironmentID() xid.ID {
	// Priority 1: API key environment
	if ac.HasAPIKey() {
		return ac.APIKey.EnvironmentID
	}
	// Priority 2: Session environment
	if ac.HasSession() && ac.Session.EnvironmentID != nil {
		return *ac.Session.EnvironmentID
	}
	// Fallback to stored environment ID
	return ac.EnvironmentID
}

// IsPublishableKey returns true if authenticated with a publishable key
func (ac *AuthContext) IsPublishableKey() bool {
	return ac.HasAPIKey() && ac.APIKey.IsPublishable()
}

// IsSecretKey returns true if authenticated with a secret key
func (ac *AuthContext) IsSecretKey() bool {
	return ac.HasAPIKey() && ac.APIKey.IsSecret()
}

// IsRestrictedKey returns true if authenticated with a restricted key
func (ac *AuthContext) IsRestrictedKey() bool {
	return ac.HasAPIKey() && ac.APIKey.IsRestricted()
}

// CanAccessUserData checks if the context can access data for a specific user
// Returns true if:
// - The authenticated user is the target user, OR
// - The API key has admin privileges
func (ac *AuthContext) CanAccessUserData(targetUserID xid.ID) bool {
	// Admin API keys can access any user data
	if ac.CanPerformAdminOp() {
		return true
	}
	// Regular users can only access their own data
	if ac.User != nil && ac.User.ID == targetUserID {
		return true
	}
	return false
}

// CanAccessOrgData checks if the context can access data for a specific org
// Returns true if:
// - The user belongs to the org, OR
// - The API key is scoped to the org, OR
// - The API key has admin privileges
func (ac *AuthContext) CanAccessOrgData(targetOrgID xid.ID) bool {
	// Admin API keys can access any org data
	if ac.CanPerformAdminOp() {
		return true
	}
	// Check if session is scoped to this org
	if ac.Session != nil && ac.Session.OrganizationID != nil {
		if *ac.Session.OrganizationID == targetOrgID {
			return true
		}
	}
	// Check if API key is scoped to this org
	if ac.APIKey != nil && ac.APIKey.OrganizationID != nil {
		if *ac.APIKey.OrganizationID == targetOrgID {
			return true
		}
	}
	return false
}

// String returns a human-readable representation of the auth context
func (ac *AuthContext) String() string {
	if !ac.IsAuthenticated {
		return "Unauthenticated"
	}

	var parts []string
	if ac.IsAPIKeyAuth {
		keyType := "API Key"
		if ac.APIKey != nil {
			keyType = string(ac.APIKey.KeyType) + " API Key"
		}
		parts = append(parts, keyType)
	}
	if ac.IsUserAuth && ac.User != nil {
		parts = append(parts, "User: "+ac.User.Email)
	}

	if len(parts) == 0 {
		return "Authenticated (unknown method)"
	}

	result := parts[0]
	if len(parts) > 1 {
		result += " + " + parts[1]
	}
	return result
}

// =============================================================================
// RBAC PERMISSION CHECKING
// =============================================================================

// HasRBACPermission checks if the auth context has a specific RBAC permission
// Permission format: "action:resource" (e.g., "view:users", "edit:posts")
func (ac *AuthContext) HasRBACPermission(action, resource string) bool {
	permString := action + ":" + resource

	// Check effective permissions (precomputed)
	for _, perm := range ac.EffectivePermissions {
		if perm == permString || perm == "*:*" {
			return true
		}
		// Wildcard matching: "view:*" or "*:users"
		if matchWildcardPermission(perm, action, resource) {
			return true
		}
	}

	return false
}

// CanAccess checks if the auth context can perform an action on a resource
// This is the main permission check method that combines:
// 1. Legacy scope strings (e.g., "users:read")
// 2. RBAC permissions (e.g., action="view", resource="users")
// 3. Delegated permissions (from creator)
// 4. User session permissions
func (ac *AuthContext) CanAccess(action, resource string) bool {
	// Method 1: Check legacy scopes
	scopeString := resource + ":" + action
	if ac.HasScope(scopeString) {
		return true
	}

	// Admin scope grants everything
	if ac.HasScope("admin:full") {
		return true
	}

	// Method 2: Check RBAC permissions
	if ac.HasRBACPermission(action, resource) {
		return true
	}

	return false
}

// HasAnyPermission checks if context has any of the specified permissions
func (ac *AuthContext) HasAnyPermission(permissions ...string) bool {
	for _, perm := range permissions {
		// Format: "action:resource"
		parts := splitPermission(perm)
		if len(parts) == 2 {
			if ac.CanAccess(parts[0], parts[1]) {
				return true
			}
		}
	}
	return false
}

// HasAllPermissions checks if context has all of the specified permissions
func (ac *AuthContext) HasAllPermissions(permissions ...string) bool {
	for _, perm := range permissions {
		parts := splitPermission(perm)
		if len(parts) == 2 {
			if !ac.CanAccess(parts[0], parts[1]) {
				return false
			}
		}
	}
	return true
}

// RequireRBACPermission ensures the context has a specific RBAC permission
func (ac *AuthContext) RequireRBACPermission(action, resource string) error {
	if !ac.HasRBACPermission(action, resource) {
		return ErrInsufficientPermission
	}
	return nil
}

// RequireCanAccess ensures the context can access (scopes OR RBAC)
func (ac *AuthContext) RequireCanAccess(action, resource string) error {
	if !ac.CanAccess(action, resource) {
		return ErrInsufficientPermission
	}
	return nil
}

// IsDelegatingCreatorPermissions returns true if API key is delegating creator's permissions
func (ac *AuthContext) IsDelegatingCreatorPermissions() bool {
	return ac.HasAPIKey() && ac.APIKey.DelegateUserPermissions
}

// IsImpersonating returns true if API key is impersonating a user
func (ac *AuthContext) IsImpersonating() bool {
	return ac.HasAPIKey() && ac.APIKey.ImpersonateUserID != nil
}

// GetImpersonatedUserID returns the user ID being impersonated (if any)
func (ac *AuthContext) GetImpersonatedUserID() *xid.ID {
	if ac.IsImpersonating() {
		return ac.APIKey.ImpersonateUserID
	}
	return nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// matchWildcardPermission checks if a permission pattern matches action/resource
// Patterns: "*:*" (all), "view:*" (all resources), "*:users" (all actions on users)
func matchWildcardPermission(permPattern, action, resource string) bool {
	parts := splitPermission(permPattern)
	if len(parts) != 2 {
		return false
	}

	permAction := parts[0]
	permResource := parts[1]

	// Full wildcard
	if permAction == "*" && permResource == "*" {
		return true
	}

	// Action wildcard
	if permAction == "*" && permResource == resource {
		return true
	}

	// Resource wildcard
	if permAction == action && permResource == "*" {
		return true
	}

	return false
}

// splitPermission splits a permission string into action and resource
// Format: "action:resource" -> ["action", "resource"]
func splitPermission(perm string) []string {
	result := []string{}
	for i := 0; i < len(perm); i++ {
		if perm[i] == ':' {
			result = append(result, perm[:i], perm[i+1:])
			break
		}
	}
	return result
}
