package apikey_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/apikey"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/forge"
)

// Example: Basic plugin registration
func ExamplePlugin_basic() {
	// Initialize database (example)
	var db *bun.DB // Your database connection

	// Create AuthSome instance
	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithMode(authsome.ModeStandalone),
	)

	// Register API key plugin
	plugin := apikeyPlugin.NewPlugin()
	auth.Use(plugin)

	// Initialize (this will call plugin Init)
	_ = auth.Initialize(context.Background())

	// Mount to router
	app := forge.New()
	_ = auth.Mount(app.Router(), "/api/auth")

	// Now API key authentication is available
}

// Example: Protect routes with API key middleware
func ExamplePlugin_protectRoutes() {
	var auth *authsome.Auth         // Initialized AuthSome instance
	var apikey *apikeyPlugin.Plugin // Initialized plugin

	app := forge.New()
	router := app.Router()

	// Option 1: Global middleware (optional auth)
	router.Use(apikey.Middleware())

	// Option 2: Require API key for specific group
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(apikey.RequireAPIKey())

	apiGroup.GET("/users", func(c forge.Context) error {
		// Extract API key info
		key := apikeyPlugin.GetAPIKey(c)
		orgID := apikeyPlugin.GetOrgID(c)
		user := apikeyPlugin.GetUser(c)

		_ = key
		_ = orgID
		_ = user

		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Option 3: Require specific scopes
	adminGroup := router.Group("/api/v1/admin")
	adminGroup.Use(apikey.RequireAPIKey("admin"))

	// Option 4: Require specific permissions
	router.POST("/api/v1/settings", func(c forge.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	}).Use(apikey.RequirePermission("settings:write"))

	_ = auth
}

// Example: Create and use API keys programmatically
func ExamplePlugin_createKey() {
	var ctx context.Context
	var service *apikey.Service // From plugin.Service()

	// Create an API key
	req := &apikey.CreateAPIKeyRequest{
		OrgID:       "org_abc123",
		UserID:      "user_xyz789",
		Name:        "Production API Key",
		Description: "Key for prod server",
		Scopes:      []string{"users:read", "users:write"},
		RateLimit:   5000,
	}

	key, err := service.CreateAPIKey(ctx, req)
	if err != nil {
		panic(err)
	}

	// IMPORTANT: Store key.Key securely - it won't be shown again!
	// key.Key = "ak_abc123_xyz789.secret_token"

	_ = key
}

// TestAPIKeyAuthentication tests the full authentication flow
func TestAPIKeyAuthentication(t *testing.T) {
	// This is a placeholder test showing the structure
	// Actual implementation would need database setup

	t.Skip("Integration test - requires database setup")

	// Setup
	var db *bun.DB
	var ctx = context.Background()

	auth := authsome.New(
		authsome.WithDatabase(db),
	)

	plugin := apikeyPlugin.NewPlugin()
	err := auth.Use(plugin)
	require.NoError(t, err)

	err = auth.Initialize(ctx)
	require.NoError(t, err)

	// Create API key
	service := plugin.Service()
	key, err := service.CreateAPIKey(ctx, &apikey.CreateAPIKeyRequest{
		OrgID:     "test_org",
		UserID:    "test_user",
		Name:      "Test Key",
		Scopes:    []string{"test:read"},
		RateLimit: 100,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, key.Key)

	// Verify API key
	resp, err := service.VerifyAPIKey(ctx, &apikey.VerifyAPIKeyRequest{
		Key: key.Key,
	})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, "test_org", resp.APIKey.OrgID)
}

// TestRateLimiting tests rate limit enforcement
func TestRateLimiting(t *testing.T) {
	t.Skip("Integration test - requires rate limiter setup")

	// Rate limiting is enforced automatically in middleware
	// when a rate limiter service is available
}

// TestMultiTenancy tests organization context injection
func TestMultiTenancy(t *testing.T) {
	t.Skip("Integration test - requires multi-tenancy plugin")

	// API keys automatically inject organization context
	// The organization ID from the API key is used to scope all operations
}
