package middleware

import (
	"net/http"
	"strings"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// AuthMiddleware handles authentication via API keys and sessions
// Following production patterns like Clerk, this middleware supports:
// - API key authentication (pk/sk/rk keys)
// - Session-based authentication (cookies + bearer tokens)
// - Dual authentication (both API key and user session)
type AuthMiddleware struct {
	apiKeySvc  *apikey.Service
	sessionSvc session.ServiceInterface
	userSvc    user.ServiceInterface
	config     AuthMiddlewareConfig
}

// AuthMiddlewareConfig configures the authentication middleware behavior
type AuthMiddlewareConfig struct {
	// Cookie name for session token
	SessionCookieName string

	// Allow unauthenticated requests to pass through
	// If false, middleware will return 401 for unauthenticated requests
	Optional bool

	// Header names to check for API keys
	APIKeyHeaders []string

	// Allow API key in query params (NOT recommended for production)
	AllowAPIKeyInQuery bool

	// Allow query param session tokens (NOT recommended for production)
	AllowSessionInQuery bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(
	apiKeySvc *apikey.Service,
	sessionSvc session.ServiceInterface,
	userSvc user.ServiceInterface,
	config AuthMiddlewareConfig,
) *AuthMiddleware {
	// Set defaults
	if config.SessionCookieName == "" {
		config.SessionCookieName = "authsome_session"
	}
	if len(config.APIKeyHeaders) == 0 {
		config.APIKeyHeaders = []string{
			"Authorization", // Bearer/ApiKey scheme
			"X-API-Key",     // Dedicated header
		}
	}

	return &AuthMiddleware{
		apiKeySvc:  apiKeySvc,
		sessionSvc: sessionSvc,
		userSvc:    userSvc,
		config:     config,
	}
}

// Authenticate is the main middleware function that populates auth context
// This middleware is optional by default - it populates context but doesn't block
func (m *AuthMiddleware) Authenticate(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		ctx := c.Request().Context()

		// Initialize empty auth context
		authCtx := &contexts.AuthContext{
			Method:          contexts.AuthMethodNone,
			IsAuthenticated: false,
			IPAddress:       extractClientIP(c.Request().RemoteAddr),
			UserAgent:       c.Request().Header.Get("User-Agent"),
		}

		// Try API key authentication first
		apiKeyResult := m.tryAPIKeyAuth(c)
		if apiKeyResult.Authenticated {
			authCtx.APIKey = apiKeyResult.APIKey
			authCtx.APIKeyScopes = apiKeyResult.APIKey.GetAllScopes()
			authCtx.AppID = apiKeyResult.APIKey.AppID
			authCtx.EnvironmentID = apiKeyResult.APIKey.EnvironmentID
			authCtx.OrganizationID = apiKeyResult.APIKey.OrganizationID
			authCtx.IsAPIKeyAuth = true
			authCtx.Method = contexts.AuthMethodAPIKey

			// Load RBAC roles and permissions
			authCtx.APIKeyRoles = apiKeyResult.Roles
			authCtx.APIKeyPermissions = apiKeyResult.Permissions
			authCtx.CreatorPermissions = apiKeyResult.CreatorPermissions
		}

		// Try session authentication
		sessionResult := m.trySessionAuth(c)
		if sessionResult.Authenticated {
			authCtx.Session = sessionResult.Session
			authCtx.User = sessionResult.User
			authCtx.IsUserAuth = true

			// Load user RBAC roles and permissions
			authCtx.UserRoles = sessionResult.Roles
			authCtx.UserPermissions = sessionResult.Permissions

			// Update method
			if authCtx.IsAPIKeyAuth {
				authCtx.Method = contexts.AuthMethodBoth
			} else {
				authCtx.Method = contexts.AuthMethodSession
				// Use session context if no API key
				authCtx.AppID = sessionResult.Session.AppID
				if sessionResult.Session.EnvironmentID != nil {
					authCtx.EnvironmentID = *sessionResult.Session.EnvironmentID
				}
				authCtx.OrganizationID = sessionResult.Session.OrganizationID
			}
		}

	// Set authenticated flag
	authCtx.IsAuthenticated = authCtx.IsAPIKeyAuth || authCtx.IsUserAuth

	// Compute effective permissions (union of all applicable permissions)
	authCtx.EffectivePermissions = m.computeEffectivePermissions(authCtx)

	// If not optional and not authenticated, reject with specific error message
	if !m.config.Optional && !authCtx.IsAuthenticated {
		// Check if API key was attempted but failed validation
		apiKeyAttempted := m.extractAPIKey(c)
		if apiKeyAttempted != "" {
			// API key was provided but invalid/expired
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "Invalid or expired API key",
				"code":  "INVALID_API_KEY",
			})
		}
		
		// No authentication provided at all
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "API key required for app identification",
			"code":  "API_KEY_REQUIRED",
		})
	}

		// Store auth context
		ctx = contexts.SetAuthContext(ctx, authCtx)

		// Also set individual context values for backward compatibility
		if !authCtx.AppID.IsNil() {
			ctx = contexts.SetAppID(ctx, authCtx.AppID)
		}
		if !authCtx.EnvironmentID.IsNil() {
			ctx = contexts.SetEnvironmentID(ctx, authCtx.EnvironmentID)
		}
		if authCtx.OrganizationID != nil && !authCtx.OrganizationID.IsNil() {
			ctx = contexts.SetOrganizationID(ctx, *authCtx.OrganizationID)
		}
		if authCtx.User != nil {
			ctx = contexts.SetUserID(ctx, authCtx.User.ID)
		}

		// Update forge context
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// authResult holds the result of an authentication attempt
type authResult struct {
	Authenticated bool
	APIKey        *apikey.APIKey
	Session       *session.Session
	User          *user.User
	Error         error
	// RBAC data
	Roles              []string
	Permissions        []string
	CreatorPermissions []string
}

// tryAPIKeyAuth attempts to authenticate via API key
func (m *AuthMiddleware) tryAPIKeyAuth(c forge.Context) authResult {
	ctx := c.Request().Context()

	// Extract API key from request
	key := m.extractAPIKey(c)
	if key == "" {
		return authResult{Authenticated: false}
	}

	// Verify the API key
	req := &apikey.VerifyAPIKeyRequest{
		Key:       key,
		IP:        extractClientIP(c.Request().RemoteAddr),
		UserAgent: c.Request().Header.Get("User-Agent"),
	}

	response, err := m.apiKeySvc.VerifyAPIKey(ctx, req)
	if err != nil || !response.Valid {
		return authResult{Authenticated: false, Error: err}
	}

	result := authResult{
		Authenticated: true,
		APIKey:        response.APIKey,
	}

	// Load RBAC roles and permissions for the API key
	orgID := response.APIKey.OrganizationID

	// Get API key's own roles
	roles, err := m.apiKeySvc.GetRoles(ctx, response.APIKey.ID, orgID)
	if err == nil && roles != nil {
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}
		result.Roles = roleNames
	}

	// Get API key's permissions (through roles)
	permissions, err := m.apiKeySvc.GetPermissions(ctx, response.APIKey.ID, orgID)
	if err == nil && permissions != nil {
		permStrings := make([]string, len(permissions))
		for i, perm := range permissions {
			permStrings[i] = perm.Action + ":" + perm.Resource
		}
		result.Permissions = permStrings
	}

	// If delegation enabled, load creator's permissions
	if response.APIKey.DelegateUserPermissions {
		effectivePerms, err := m.apiKeySvc.GetEffectivePermissions(ctx, response.APIKey.ID, orgID)
		if err == nil && effectivePerms != nil {
			creatorPerms := []string{}
			for _, perm := range effectivePerms.Permissions {
				if perm.Source == "creator" {
					creatorPerms = append(creatorPerms, perm.Action+":"+perm.Resource)
				}
			}
			result.CreatorPermissions = creatorPerms
		}
	}

	return result
}

// trySessionAuth attempts to authenticate via session token
func (m *AuthMiddleware) trySessionAuth(c forge.Context) authResult {
	ctx := c.Request().Context()

	// Try cookie first
	sessionToken := m.extractSessionFromCookie(c)

	// Fallback to Authorization: Bearer header (if not an API key)
	if sessionToken == "" {
		sessionToken = m.extractSessionFromBearer(c)
	}

	// Fallback to query param (if enabled, not recommended)
	if sessionToken == "" && m.config.AllowSessionInQuery {
		sessionToken = c.Request().URL.Query().Get("session_token")
	}

	if sessionToken == "" {
		return authResult{Authenticated: false}
	}

	// Validate session
	sess, err := m.sessionSvc.FindByToken(ctx, sessionToken)
	if err != nil || sess == nil {
		return authResult{Authenticated: false, Error: err}
	}

	// Load user
	usr, err := m.userSvc.FindByID(ctx, sess.UserID)
	if err != nil || usr == nil {
		return authResult{Authenticated: false, Error: err}
	}

	result := authResult{
		Authenticated: true,
		Session:       sess,
		User:          usr,
	}

	// TODO: Load user RBAC roles and permissions
	// This requires access to RBAC service, which should be added to AuthMiddleware
	// For now, leave empty - can be populated when RBAC service is integrated
	result.Roles = []string{}
	result.Permissions = []string{}

	return result
}

// extractAPIKey extracts API key from request using multiple methods
func (m *AuthMiddleware) extractAPIKey(c forge.Context) string {
	// Method 1: Authorization header with ApiKey scheme
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		// ApiKey pk_test_xxx or ApiKey sk_test_xxx
		if strings.HasPrefix(authHeader, "ApiKey ") {
			return strings.TrimPrefix(authHeader, "ApiKey ")
		}

		// Bearer pk_test_xxx (if starts with pk_/sk_/rk_)
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if strings.HasPrefix(token, "pk_") ||
				strings.HasPrefix(token, "sk_") ||
				strings.HasPrefix(token, "rk_") {
				return token
			}
		}
	}

	// Method 2: X-API-Key header
	if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Method 3: Query parameter (if enabled, NOT recommended)
	if m.config.AllowAPIKeyInQuery {
		if apiKey := c.Request().URL.Query().Get("api_key"); apiKey != "" {
			return apiKey
		}
	}

	return ""
}

// extractSessionFromCookie extracts session token from cookie
func (m *AuthMiddleware) extractSessionFromCookie(c forge.Context) string {
	cookie, err := c.Request().Cookie(m.config.SessionCookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}

// extractSessionFromBearer extracts session token from Bearer header
// Only if it doesn't look like an API key
func (m *AuthMiddleware) extractSessionFromBearer(c forge.Context) string {
	authHeader := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Don't treat API keys as session tokens
	if strings.HasPrefix(token, "pk_") ||
		strings.HasPrefix(token, "sk_") ||
		strings.HasPrefix(token, "rk_") {
		return ""
	}

	return token
}

// =============================================================================
// CONVENIENCE MIDDLEWARE FUNCTIONS
// =============================================================================

// RequireAuth middleware that rejects unauthenticated requests
func (m *AuthMiddleware) RequireAuth(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		authCtx, ok := contexts.GetAuthContext(c.Request().Context())
		if !ok || !authCtx.IsAuthenticated {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "authentication required",
				"code":  "AUTHENTICATION_REQUIRED",
			})
		}
		return next(c)
	}
}

// RequireUser middleware that requires a logged-in user (session)
func (m *AuthMiddleware) RequireUser(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		authCtx, ok := contexts.GetAuthContext(c.Request().Context())
		if !ok || !authCtx.IsUserAuth {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "user authentication required",
				"code":  "USER_AUTH_REQUIRED",
			})
		}
		return next(c)
	}
}

// RequireAPIKey middleware that requires an API key
func (m *AuthMiddleware) RequireAPIKey(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		authCtx, ok := contexts.GetAuthContext(c.Request().Context())
		if !ok || !authCtx.IsAPIKeyAuth {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "API key required",
				"code":  "API_KEY_REQUIRED",
			})
		}
		return next(c)
	}
}

// RequireScope middleware that requires a specific API key scope
func (m *AuthMiddleware) RequireScope(scope string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.HasScope(scope) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "insufficient API key scope",
					"code":     "INSUFFICIENT_SCOPE",
					"required": scope,
					"current":  authCtx.APIKeyScopes,
				})
			}
			return next(c)
		}
	}
}

// RequireAnyScope middleware that requires any of the specified scopes
func (m *AuthMiddleware) RequireAnyScope(scopes ...string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.HasAnyScopeOf(scopes...) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "insufficient API key scope",
					"code":     "INSUFFICIENT_SCOPE",
					"required": "any of: " + strings.Join(scopes, ", "),
					"current":  authCtx.APIKeyScopes,
				})
			}
			return next(c)
		}
	}
}

// RequireAllScopes middleware that requires all of the specified scopes
func (m *AuthMiddleware) RequireAllScopes(scopes ...string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.HasAllScopesOf(scopes...) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "insufficient API key scope",
					"code":     "INSUFFICIENT_SCOPE",
					"required": "all of: " + strings.Join(scopes, ", "),
					"current":  authCtx.APIKeyScopes,
				})
			}
			return next(c)
		}
	}
}

// RequireSecretKey middleware that requires a secret (sk_) API key
func (m *AuthMiddleware) RequireSecretKey(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		authCtx, ok := contexts.GetAuthContext(c.Request().Context())
		if !ok || !authCtx.IsSecretKey() {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "secret API key required",
				"code":  "SECRET_KEY_REQUIRED",
			})
		}
		return next(c)
	}
}

// RequirePublishableKey middleware that requires a publishable (pk_) API key
func (m *AuthMiddleware) RequirePublishableKey(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		authCtx, ok := contexts.GetAuthContext(c.Request().Context())
		if !ok || !authCtx.IsPublishableKey() {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "publishable API key required",
				"code":  "PUBLISHABLE_KEY_REQUIRED",
			})
		}
		return next(c)
	}
}

// RequireAdmin middleware that requires admin privileges
func (m *AuthMiddleware) RequireAdmin(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		authCtx, ok := contexts.GetAuthContext(c.Request().Context())
		if !ok || !authCtx.CanPerformAdminOp() {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "admin privileges required",
				"code":  "ADMIN_REQUIRED",
			})
		}
		return next(c)
	}
}

// =============================================================================
// RBAC-AWARE MIDDLEWARE (Hybrid Approach)
// =============================================================================

// RequireRBACPermission middleware that requires a specific RBAC permission
// Checks only RBAC permissions (not legacy scopes)
func (m *AuthMiddleware) RequireRBACPermission(action, resource string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.HasRBACPermission(action, resource) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "insufficient RBAC permission",
					"code":     "INSUFFICIENT_PERMISSION",
					"required": action + ":" + resource,
				})
			}
			return next(c)
		}
	}
}

// RequireCanAccess middleware that checks if auth context can access a resource
// This is flexible - accepts EITHER legacy scopes OR RBAC permissions
// Recommended for backward compatibility
func (m *AuthMiddleware) RequireCanAccess(action, resource string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.CanAccess(action, resource) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "access denied",
					"code":     "ACCESS_DENIED",
					"required": action + ":" + resource,
				})
			}
			return next(c)
		}
	}
}

// RequireAnyPermission middleware that requires any of the specified permissions
func (m *AuthMiddleware) RequireAnyPermission(permissions ...string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.HasAnyPermission(permissions...) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "insufficient permission",
					"code":     "INSUFFICIENT_PERMISSION",
					"required": "any of: " + strings.Join(permissions, ", "),
				})
			}
			return next(c)
		}
	}
}

// RequireAllPermissions middleware that requires all of the specified permissions
func (m *AuthMiddleware) RequireAllPermissions(permissions ...string) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			authCtx, ok := contexts.GetAuthContext(c.Request().Context())
			if !ok || !authCtx.HasAllPermissions(permissions...) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":    "insufficient permission",
					"code":     "INSUFFICIENT_PERMISSION",
					"required": "all of: " + strings.Join(permissions, ", "),
				})
			}
			return next(c)
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// computeEffectivePermissions computes the union of all applicable permissions
// Priority:
// 1. API key's own permissions
// 2. If delegation enabled: creator's permissions
// 3. User session permissions
func (m *AuthMiddleware) computeEffectivePermissions(authCtx *contexts.AuthContext) []string {
	permissionSet := make(map[string]bool)

	// Add API key permissions
	for _, perm := range authCtx.APIKeyPermissions {
		permissionSet[perm] = true
	}

	// Add creator permissions (if delegated)
	for _, perm := range authCtx.CreatorPermissions {
		permissionSet[perm] = true
	}

	// Add user session permissions
	for _, perm := range authCtx.UserPermissions {
		permissionSet[perm] = true
	}

	// Convert set to slice
	result := make([]string, 0, len(permissionSet))
	for perm := range permissionSet {
		result = append(result, perm)
	}

	return result
}

// extractClientIP extracts the real client IP from the request
func extractClientIP(remoteAddr string) string {
	// Remove port if present
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}
	return remoteAddr
}
