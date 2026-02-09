package apikey_test

import (
	"context"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/apikey"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/forge"
)

// Example: Basic plugin registration.
func ExamplePlugin_basic() {
	// Initialize database (example)
	var db *bun.DB // Your database connection

	// Create Forge app
	app := forge.New()

	// Create AuthSome instance
	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithForgeApp(app),
	)

	// Register API key plugin
	plugin := apikeyPlugin.NewPlugin()
	_ = auth.RegisterPlugin(plugin)

	// Initialize (this will call plugin Init)
	_ = auth.Initialize(context.Background())

	// Mount to router
	_ = auth.Mount(app.Router(), "/api/auth")

	// Now API key authentication is available
}

// Example: Protect routes with API key middleware.
func ExamplePlugin_protectRoutes() {
	// This example demonstrates how to extract API key context from requests
	// For actual middleware setup, see the plugin documentation

	// When a request is authenticated with an API key, you can extract context:

	// In your handler:
	handler := func(c forge.Context) error {
		// Extract API key context (V2 Architecture)
		key := apikeyPlugin.GetAPIKey(c)                 // The full API key data
		orgID := apikeyPlugin.GetOrgID(c)                // User org ID (may be empty string)
		user := apikeyPlugin.GetUser(c)                  // Associated user
		scopes := apikeyPlugin.GetScopes(c)              // API key scopes
		authenticated := apikeyPlugin.IsAuthenticated(c) // Check if authenticated

		// V2 Architecture context is available in the key:
		if key != nil {
			_ = key.AppID          // Platform app ID
			_ = key.EnvironmentID  // Environment ID (dev/staging/prod)
			_ = key.OrganizationID // User-created org ID (may be nil)
			_ = key.Scopes         // Key scopes
			_ = key.Permissions    // Key permissions
		}

		_ = orgID
		_ = user
		_ = scopes
		_ = authenticated

		return c.JSON(200, map[string]string{"status": "ok"})
	}

	_ = handler

	// Middleware is configured through the plugin's Middleware() method
	// See plugin documentation for setup details
}

// Example: Create and use API keys programmatically.
func ExamplePlugin_createKey() {
	var ctx context.Context

	var service *apikey.Service // From plugin.Service()

	// V2 Architecture: App → Environment → Organization (optional)
	appID := xid.New()  // Platform app (required)
	envID := xid.New()  // Environment: dev/staging/prod (required)
	orgID := xid.New()  // User org (optional - can be nil)
	userID := xid.New() // User creating the key

	// Create an API key
	req := &apikey.CreateAPIKeyRequest{
		AppID:         appID,  // Platform app (required)
		EnvironmentID: envID,  // Environment (required)
		OrgID:         &orgID, // User-created org (optional)
		UserID:        userID, // User who owns the key
		Name:          "Production API Key",
		Description:   "Key for prod server",
		KeyType:       apikey.KeyTypeSecret, // sk (secret), pk (publishable), rk (restricted)
		Scopes:        []string{"users:read", "users:write"},
		RateLimit:     5000,
	}

	key, err := service.CreateAPIKey(ctx, req)
	if err != nil {
		panic(err)
	}

	// IMPORTANT: Store key.Key securely - it won't be shown again!
	// key.Key = "sk_prod_abc123.secret_token"

	_ = key
}

// TestAPIKeyAuthentication tests the full authentication flow.
func TestAPIKeyAuthentication(t *testing.T) {
	// This is a placeholder test showing the structure
	// Actual implementation would need database setup
	t.Skip("Integration test - requires database setup")

	// Setup
	var (
		db  *bun.DB
		ctx = context.Background()
	)

	// Create Forge app
	app := forge.New()

	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithForgeApp(app),
	)

	plugin := apikeyPlugin.NewPlugin()
	err := auth.RegisterPlugin(plugin)
	require.NoError(t, err)

	err = auth.Initialize(ctx)
	require.NoError(t, err)

	// V2 Architecture context
	appID := xid.New()  // Platform app
	envID := xid.New()  // Environment
	orgID := xid.New()  // User org (optional)
	userID := xid.New() // User

	// Create API key
	service := plugin.Service()
	key, err := service.CreateAPIKey(ctx, &apikey.CreateAPIKeyRequest{
		AppID:         appID,  // Platform app (required)
		EnvironmentID: envID,  // Environment (required)
		OrgID:         &orgID, // User-created org (optional)
		UserID:        userID, // User who owns the key
		Name:          "Test Key",
		KeyType:       apikey.KeyTypeSecret, // sk/pk/rk
		Scopes:        []string{"test:read"},
		RateLimit:     100,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, key.Key)
	assert.Contains(t, key.Key, ".", "Key should have format: prefix.secret")

	// Verify API key
	resp, err := service.VerifyAPIKey(ctx, &apikey.VerifyAPIKeyRequest{
		Key: key.Key,
	})
	require.NoError(t, err)
	assert.True(t, resp.Valid)
	assert.Equal(t, appID, resp.APIKey.AppID, "AppID should match")
	assert.Equal(t, envID, resp.APIKey.EnvironmentID, "EnvironmentID should match")
	assert.NotNil(t, resp.APIKey.OrganizationID, "OrgID should be set")
	assert.Equal(t, orgID, *resp.APIKey.OrganizationID, "OrgID should match")
}

// TestRateLimiting tests rate limit enforcement.
func TestRateLimiting(t *testing.T) {
	t.Skip("Integration test - requires rate limiter setup")

	// Rate limiting is enforced automatically in middleware
	// when a rate limiter service is available
	// Rate limits are per-key and tracked per API key ID
}

// TestMultiTenancy tests the V2 architecture: App → Environment → Organization.
func TestMultiTenancy(t *testing.T) {
	t.Skip("Integration test - requires multi-tenancy setup")

	// V2 Architecture enforces 3-tier isolation:
	// 1. AppID: Platform tenant (like a SaaS customer)
	// 2. EnvironmentID: Dev/Staging/Production within the app
	// 3. OrganizationID: End-user workspaces within the environment (optional)
	//
	// API keys are scoped to all three levels:
	// - AppID (required): Isolates platform tenants
	// - EnvironmentID (required): Isolates environments
	// - OrganizationID (optional): If set, restricts access to that org only
	//
	// Example key prefixes:
	// - sk_prod_xyz123 = Secret key for production environment
	// - pk_dev_abc456  = Publishable key for development environment
	// - rk_test_def789 = Restricted key for test environment
}
