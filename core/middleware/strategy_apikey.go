package middleware

import (
	"context"
	"strings"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

// APIKeyStrategy implements authentication via API keys
// Supports multiple extraction methods:
// - Authorization: ApiKey <key>
// - Authorization: Bearer <key> (if key has pk_/sk_/rk_ prefix)
// - X-API-Key: <key>
// - Query param (if enabled, not recommended)
type APIKeyStrategy struct {
	service          *apikey.Service
	allowInQuery     bool
	additionalHeaders []string
}

// NewAPIKeyStrategy creates a new API key authentication strategy
func NewAPIKeyStrategy(service *apikey.Service, allowInQuery bool) *APIKeyStrategy {
	return &APIKeyStrategy{
		service:      service,
		allowInQuery: allowInQuery,
		additionalHeaders: []string{
			"X-API-Key",
		},
	}
}

// ID returns the strategy identifier
func (s *APIKeyStrategy) ID() string {
	return "apikey"
}

// Priority returns the strategy priority (10 = high priority for API keys)
func (s *APIKeyStrategy) Priority() int {
	return 10
}

// Extract attempts to extract an API key from the request
func (s *APIKeyStrategy) Extract(c forge.Context) (interface{}, bool) {
	// Method 1: Authorization header with ApiKey scheme
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		// ApiKey pk_test_xxx or ApiKey sk_test_xxx
		if strings.HasPrefix(authHeader, "ApiKey ") {
			apiKey := strings.TrimPrefix(authHeader, "ApiKey ")
			if apiKey != "" {
				return apiKey, true
			}
		}

		// Bearer pk_test_xxx (if starts with pk_/sk_/rk_)
		// IMPORTANT: Only extract if it's actually an API key, not a JWT
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			token = strings.TrimSpace(token)
			
			// Only accept if it starts with known API key prefixes
			if strings.HasPrefix(token, "pk_") ||
				strings.HasPrefix(token, "sk_") ||
				strings.HasPrefix(token, "rk_") {
				return token, true
			}
			
			// Skip JWTs (3 parts separated by dots)
			// Let the JWT strategy handle them
			if strings.Count(token, ".") == 2 {
				return nil, false
			}
		}
	}

	// Method 2: Additional headers (X-API-Key, etc.)
	for _, header := range s.additionalHeaders {
		if apiKey := c.Request().Header.Get(header); apiKey != "" {
			// Make sure it's an API key, not a JWT
			if strings.HasPrefix(apiKey, "pk_") ||
				strings.HasPrefix(apiKey, "sk_") ||
				strings.HasPrefix(apiKey, "rk_") {
				return apiKey, true
			}
		}
	}

	// Method 3: Query parameter (if enabled, NOT recommended)
	if s.allowInQuery {
		if apiKey := c.Request().URL.Query().Get("api_key"); apiKey != "" {
			return apiKey, true
		}
	}

	return nil, false
}

// Authenticate validates the API key and builds auth context
func (s *APIKeyStrategy) Authenticate(ctx context.Context, credentials interface{}) (*contexts.AuthContext, error) {
	apiKeyStr, ok := credentials.(string)
	if !ok {
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  "invalid credentials type",
		}
	}

	// Verify API key using the service
	req := &apikey.VerifyAPIKeyRequest{
		Key: apiKeyStr,
		// IP and UserAgent can be added if needed
	}
	
	resp, err := s.service.VerifyAPIKey(ctx, req)
	if err != nil {
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  "invalid API key",
			Err:      err,
		}
	}

	// Check if verification was successful and APIKey is not nil
	if !resp.Valid || resp.APIKey == nil {
		errMsg := "invalid API key"
		if resp.Error != "" {
			errMsg = resp.Error
		}
		return nil, &AuthStrategyError{
			Strategy: s.ID(),
			Message:  errMsg,
			Err:      nil,
		}
	}

	// Build auth context
	authCtx := &contexts.AuthContext{
		Method:          contexts.AuthMethodAPIKey,
		IsAuthenticated: true,
		IsAPIKeyAuth:    true,
		APIKey:          resp.APIKey,
		APIKeyScopes:    resp.APIKey.GetAllScopes(),
		AppID:           resp.APIKey.AppID,
		EnvironmentID:   resp.APIKey.EnvironmentID,
		OrganizationID:  resp.APIKey.OrganizationID,
	}

	// Note: RBAC data (roles, permissions) are loaded separately by the middleware
	// after authentication succeeds. This keeps the strategy focused on authentication
	// and allows the middleware to handle authorization concerns.
	authCtx.APIKeyRoles = []string{}
	authCtx.APIKeyPermissions = []string{}
	authCtx.CreatorPermissions = []string{}

	return authCtx, nil
}

// AuthStrategyError represents an authentication strategy error
type AuthStrategyError struct {
	Strategy string
	Message  string
	Err      error
}

func (e *AuthStrategyError) Error() string {
	if e.Err != nil {
		return e.Strategy + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Strategy + ": " + e.Message
}

func (e *AuthStrategyError) Unwrap() error {
	return e.Err
}

