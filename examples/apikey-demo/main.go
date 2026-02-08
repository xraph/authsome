package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/apikey"
	apikeyPlugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/forge"
)

func main() {
	// Initialize database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/authsome_dev?sslmode=disable"
	}

	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithTimeout(30*time.Second),
	)
	sqldb := sql.OpenDB(connector)
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("✅ Connected to database")

	// Initialize Forge app
	app := forge.New()

	// Initialize AuthSome
	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithForgeApp(app),
	)

	// Register API key plugin
	plugin := apikeyPlugin.NewPlugin()
	if err := auth.RegisterPlugin(plugin); err != nil {
		log.Fatalf("Failed to register API key plugin: %v", err)
	}
	log.Println("✅ API Key plugin registered")

	// Initialize AuthSome (this calls Init on all plugins)
	ctx := context.Background()
	if err := auth.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize AuthSome: %v", err)
	}
	log.Println("✅ AuthSome initialized")

	// Mount authentication routes
	if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
		log.Fatalf("Failed to mount auth routes: %v", err)
	}
	log.Println("✅ Auth routes mounted at /api/auth")

	// Setup demo routes
	setupDemoRoutes(app.Router(), plugin)

	// Create a demo API key for testing
	go func() {
		time.Sleep(2 * time.Second) // Wait for server to start
		createDemoAPIKey(plugin.Service())
	}()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Server starting on http://localhost:%s", port)
	log.Printf("📖 API Key management: http://localhost:%s/api/auth/api-keys", port)
	log.Printf("🧪 Test endpoints: http://localhost:%s/api/v1/*", port)
	log.Println()
	log.Println("=== Quick Test ===")
	log.Println("After demo key is created, test with:")
	log.Println("  curl -H 'Authorization: ApiKey <your-key>' http://localhost:3000/api/v1/users")
	log.Println()

	if err := app.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// setupDemoRoutes creates example API routes protected by API key auth
func setupDemoRoutes(router forge.Router, plugin *apikeyPlugin.Plugin) {
	// Public endpoints (no auth required)
	router.GET("/", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "API Key Demo Server",
			"version": "1.0.0",
			"endpoints": map[string]string{
				"health":      "GET /health",
				"api_keys":    "POST /api/auth/api-keys (create key)",
				"list_keys":   "GET /api/auth/api-keys?org_id=demo&user_id=demo",
				"users":       "GET /api/v1/users (requires API key)",
				"admin_users": "GET /api/v1/admin/users (requires 'admin' scope)",
				"settings":    "POST /api/v1/settings (requires 'settings:write' permission)",
			},
		})
	})

	router.GET("/health", func(c forge.Context) error {
		return c.JSON(200, map[string]string{"status": "healthy"})
	})

	// API v1 routes - protected by API key
	apiV1 := router.Group("/api/v1")

	// Apply API key authentication middleware
	// Note: Middleware would be applied here if available
	// apiV1.Use(plugin.Middleware())

	// Public API endpoint (auth optional)
	apiV1.GET("/public", func(c forge.Context) error {
		authenticated := apikeyPlugin.IsAuthenticated(c)
		response := map[string]interface{}{
			"message":       "This endpoint is public",
			"authenticated": authenticated,
		}

		if authenticated {
			apiKey := apikeyPlugin.GetAPIKey(c)
			response["api_key_name"] = apiKey.Name
			// response["org_id"] = apiKey.OrgID  // OrgID is now part of V2 architecture
		}

		return c.JSON(200, response)
	})

	// Protected endpoints - require valid API key
	// Note: RequireAPIKey middleware would be applied here if available
	protected := apiV1.Group("")
	// protected.Use(plugin.RequireAPIKey())

	protected.GET("/users", func(c forge.Context) error {
		// Extract API key info from context
		apiKey := apikeyPlugin.GetAPIKey(c)
		orgID := apikeyPlugin.GetOrgID(c)
		user := apikeyPlugin.GetUser(c)
		scopes := apikeyPlugin.GetScopes(c)

		return c.JSON(200, map[string]interface{}{
			"message":      "User list endpoint",
			"org_id":       orgID,
			"api_key_name": apiKey.Name,
			"scopes":       scopes,
			"user_id":      apiKey.UserID,
			"user":         user,
			"data": []map[string]string{
				{"id": "1", "name": "Alice", "email": "alice@example.com"},
				{"id": "2", "name": "Bob", "email": "bob@example.com"},
			},
		})
	})

	protected.POST("/users", func(c forge.Context) error {
		orgID := apikeyPlugin.GetOrgID(c)
		return c.JSON(201, map[string]interface{}{
			"message": "User created",
			"org_id":  orgID,
		})
	})

	// Admin endpoints - require 'admin' scope
	// Note: RequireAPIKey with scope would be applied here if available
	admin := apiV1.Group("/admin")
	// admin.Use(plugin.RequireAPIKey("admin"))

	admin.GET("/users", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Admin user list - requires 'admin' scope",
			"data": []map[string]string{
				{"id": "1", "name": "Alice", "role": "admin"},
				{"id": "2", "name": "Bob", "role": "user"},
			},
		})
	})

	// Settings endpoint - requires specific permission
	// Note: Permission middleware would be applied here if available
	apiV1.POST("/settings", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"message": "Settings updated (permission check disabled for now)",
		})
	})

	// Scoped endpoints examples
	// Note: Scope middleware would be applied here if available
	apiV1.GET("/resources/read", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"message": "Read resources (scope check disabled for now)",
		})
	})

	apiV1.POST("/resources/write", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"message": "Write resources (scope check disabled for now)",
		})
	})

	log.Println("✅ Demo routes configured")
}

// createDemoAPIKey creates a demo API key for testing
func createDemoAPIKey(service *apikey.Service) {
	ctx := context.Background()

	demoAppID := xid.New()
	demoEnvID := xid.New()
	demoUserID := xid.New()

	req := &apikey.CreateAPIKeyRequest{
		AppID:         demoAppID,
		EnvironmentID: demoEnvID,
		UserID:        demoUserID,
		KeyType:       apikey.KeyTypeSecret,
		Name:          "Demo API Key",
		Description:   "Automatically created demo key for testing",
		Scopes:        []string{"users:read", "users:write", "resources:read", "admin"},
		Permissions: map[string]string{
			"settings:write": "all",
		},
		RateLimit: 1000,
		Metadata: map[string]string{
			"created_by": "demo_script",
			"purpose":    "testing",
		},
	}

	key, err := service.CreateAPIKey(ctx, req)
	if err != nil {
		log.Printf("⚠️  Failed to create demo API key: %v", err)
		return
	}

	// Commented out: Organization field from V2 architecture change

}
