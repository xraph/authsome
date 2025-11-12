package apikey

import (
	"context"
	"fmt"
	"strings"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/interfaces"
	"github.com/xraph/forge"
)

// Context keys for API key authentication
type contextKey string

const (
	APIKeyContextKey       contextKey = "api_key"
	APIKeyUserContextKey   contextKey = "api_key_user"
	APIKeyAuthenticatedKey contextKey = "api_key_authenticated"
	APIKeyPermissionsKey   contextKey = "api_key_permissions"
)

// Middleware handles API key authentication
type Middleware struct {
	service     *apikey.Service
	userSvc     *user.Service
	rateLimiter *ratelimit.Service
	config      Config
}

// NewMiddleware creates a new API key middleware
func NewMiddleware(
	service *apikey.Service,
	userSvc *user.Service,
	rateLimiter *ratelimit.Service,
	config Config,
) *Middleware {
	return &Middleware{
		service:     service,
		userSvc:     userSvc,
		rateLimiter: rateLimiter,
		config:      config,
	}
}

// Authenticate attempts to authenticate using an API key from the request
// This middleware is non-blocking - it will set context values if a valid API key is found,
// but will not reject requests without API keys (use RequireAPIKey for that)
func (m *Middleware) Authenticate(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Extract API key from request
		key := m.extractAPIKey(c)
		if key == "" {
			// No API key provided, continue without authentication
			return next(c)
		}

		// Verify the API key
		req := &apikey.VerifyAPIKeyRequest{
			Key:       key,
			IP:        c.Request().RemoteAddr,
			UserAgent: c.Request().Header.Get("User-Agent"),
		}

		response, err := m.service.VerifyAPIKey(c.Request().Context(), req)
		if err != nil || !response.Valid {
			// Invalid API key, continue without authentication
			// Don't return error to allow fallback to other auth methods
			return next(c)
		}

		apiKey := response.APIKey

		// Check IP whitelist if enabled
		if m.config.IPWhitelisting.Enabled && len(apiKey.AllowedIPs) > 0 {
			clientIP := extractClientIP(c.Request().RemoteAddr)
			if !isIPAllowed(clientIP, apiKey.AllowedIPs) {
				if m.config.IPWhitelisting.StrictMode {
					return c.JSON(403, map[string]string{
						"error": "IP address not whitelisted",
						"code":  "IP_NOT_ALLOWED",
					})
				}
				// Non-strict mode: log and continue
				// TODO: Add logging when logger is available
			}
		}

		// Check rate limit if enabled
		if m.rateLimiter != nil && m.config.RateLimiting.Enabled {
			rateLimitKey := fmt.Sprintf("apikey:%s", apiKey.ID.String())
			allowed, err := m.rateLimiter.CheckLimit(c.Request().Context(), rateLimitKey, ratelimit.Rule{
				Window: m.config.RateLimiting.Window,
				Max:    apiKey.RateLimit,
			})
			if err != nil {
				return c.JSON(500, map[string]string{
					"error": "rate limit check failed",
				})
			}
			if !allowed {
				return c.JSON(429, map[string]string{
					"error":   "rate limit exceeded",
					"message": fmt.Sprintf("API key rate limit of %d requests per window exceeded", apiKey.RateLimit),
				})
			}
		}

		// Load user information
		var usr *user.User
		if m.userSvc != nil {
			// UserID is already an xid.ID
			usr, err = m.userSvc.FindByID(c.Request().Context(), apiKey.UserID)
			if err != nil {
				// User not found, but continue with API key auth
				// This allows API keys to work even if user service is unavailable
			}
		}

		// Inject API key, user, and organization context
		ctx := c.Request().Context()
		ctx = context.WithValue(ctx, APIKeyContextKey, apiKey)
		ctx = context.WithValue(ctx, APIKeyAuthenticatedKey, true)
		ctx = context.WithValue(ctx, APIKeyPermissionsKey, apiKey.Scopes)

		// Inject V2 architecture context (App, Environment, Organization)
		ctx = interfaces.SetAppID(ctx, apiKey.AppID)
		if apiKey.EnvironmentID != nil {
			ctx = interfaces.SetEnvironmentID(ctx, *apiKey.EnvironmentID)
		}
		if apiKey.OrganizationID != nil {
			ctx = interfaces.SetOrganizationID(ctx, *apiKey.OrganizationID)
		}
		ctx = interfaces.SetUserID(ctx, apiKey.UserID)

		if usr != nil {
			ctx = context.WithValue(ctx, APIKeyUserContextKey, usr)
			ctx = context.WithValue(ctx, "user", usr) // Standard user context key
		}

		// Update request with new context
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// RequireAPIKey enforces API key authentication
func (m *Middleware) RequireAPIKey(scopes ...string) func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Check if API key authenticated
			authenticated := c.Request().Context().Value(APIKeyAuthenticatedKey)
			if authenticated != true {
				return c.JSON(401, map[string]string{
					"error": "API key authentication required",
					"code":  "MISSING_API_KEY",
				})
			}

			// Check scopes if specified
			if len(scopes) > 0 {
				apiKey := GetAPIKey(c)
				if apiKey == nil {
					return c.JSON(401, map[string]string{
						"error": "API key not found in context",
						"code":  "INVALID_API_KEY",
					})
				}

				// Check if API key has all required scopes (supports wildcards)
				for _, requiredScope := range scopes {
					if !apiKey.HasScopeWildcard(requiredScope) {
						return c.JSON(403, map[string]string{
							"error": fmt.Sprintf("Missing required scope: %s", requiredScope),
							"code":  "INSUFFICIENT_SCOPE",
						})
					}
				}
			}

			return next(c)
		}
	}
}

// RequirePermission enforces specific permissions
func (m *Middleware) RequirePermission(permissions ...string) func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Check if API key authenticated
			authenticated := c.Request().Context().Value(APIKeyAuthenticatedKey)
			if authenticated != true {
				return c.JSON(401, map[string]string{
					"error": "API key authentication required",
					"code":  "MISSING_API_KEY",
				})
			}

			// Get API key from context
			apiKey := GetAPIKey(c)
			if apiKey == nil {
				return c.JSON(401, map[string]string{
					"error": "API key not found in context",
					"code":  "INVALID_API_KEY",
				})
			}

			// Check if API key has all required permissions
			for _, perm := range permissions {
				if !apiKey.HasPermission(perm) {
					return c.JSON(403, map[string]string{
						"error": fmt.Sprintf("Missing required permission: %s", perm),
						"code":  "INSUFFICIENT_PERMISSION",
					})
				}
			}

			return next(c)
		}
	}
}

// extractAPIKey extracts the API key from the request
// Supports multiple extraction methods:
// 1. Authorization: ApiKey <key>
// 2. Authorization: Bearer <key> (if it starts with ak_)
// 3. X-API-Key: <key>
// 4. Query parameter: api_key=<key> (optional, can be disabled in config)
func (m *Middleware) extractAPIKey(c forge.Context) string {
	// Method 1: Authorization header with ApiKey scheme
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		// Check for "ApiKey <key>" format
		if strings.HasPrefix(authHeader, "ApiKey ") {
			return strings.TrimPrefix(authHeader, "ApiKey ")
		}

		// Check for "Bearer <key>" format where key starts with "ak_"
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if strings.HasPrefix(token, "ak_") {
				return token
			}
		}
	}

	// Method 2: X-API-Key header (common convention)
	if apiKey := c.Request().Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	// Method 3: Query parameter (if enabled in config)
	if m.config.AllowQueryParam {
		if apiKey := c.Request().URL.Query().Get("api_key"); apiKey != "" {
			return apiKey
		}
	}

	return ""
}

// GetAPIKey extracts the API key from the request context
func GetAPIKey(c forge.Context) *apikey.APIKey {
	if key := c.Request().Context().Value(APIKeyContextKey); key != nil {
		if apiKey, ok := key.(*apikey.APIKey); ok {
			return apiKey
		}
	}
	return nil
}

// GetUser extracts the user associated with the API key from context
func GetUser(c forge.Context) *user.User {
	if u := c.Request().Context().Value(APIKeyUserContextKey); u != nil {
		if usr, ok := u.(*user.User); ok {
			return usr
		}
	}
	return nil
}

// GetOrgID extracts the organization ID from context (V2 architecture)
// Returns the xid.ID, check with IsNil() before use
func GetOrgID(c forge.Context) string {
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	if orgID.IsNil() {
		return ""
	}
	return orgID.String()
}

// IsAuthenticated checks if the request is authenticated via API key
func IsAuthenticated(c forge.Context) bool {
	return c.Request().Context().Value(APIKeyAuthenticatedKey) == true
}

// GetScopes returns the scopes associated with the authenticated API key
func GetScopes(c forge.Context) []string {
	if scopes := c.Request().Context().Value(APIKeyPermissionsKey); scopes != nil {
		if s, ok := scopes.([]string); ok {
			return s
		}
	}
	return []string{}
}
