package main

import (
	"log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

// Example demonstrating the production-grade authentication context system
// with pk/sk/rk API key support, following Clerk's pattern

func main() {
	// Create Forge app
	app := forge.New()

	// Initialize AuthSome
	auth := authsome.New(
		authsome.WithBasePath("/api/auth"),
		authsome.WithForgeApp(app),
	)

	// Initialize core services
	if err := auth.Initialize(app.Context()); err != nil {
		log.Fatal(err)
	}

	// Mount auth routes
	if err := auth.Mount(app, "/api/auth"); err != nil {
		log.Fatal(err)
	}

	// =============================================================================
	// GLOBAL MIDDLEWARE - Populates auth context for ALL routes
	// =============================================================================
	app.Use(auth.AuthMiddleware())

	// =============================================================================
	// PUBLIC ROUTES - No authentication required
	// =============================================================================
	app.GET("/api/public/status", handleStatus)

	// =============================================================================
	// PROTECTED ROUTES - Require any form of authentication
	// =============================================================================
	protectedGroup := app.Group("/api/protected")
	protectedGroup.Use(auth.RequireAuth())
	{
		// Works with either session OR API key
		protectedGroup.GET("/profile", handleProfile)
	}

	// =============================================================================
	// USER ROUTES - Require user session authentication
	// =============================================================================
	userGroup := app.Group("/api/user")
	userGroup.Use(auth.RequireUser())
	{
		// Only works with user session, not API key
		userGroup.GET("/me", handleGetMe)
		userGroup.PATCH("/me", handleUpdateMe)
		userGroup.GET("/sessions", handleListSessions)
	}

	// =============================================================================
	// BACKEND ROUTES - Require API key authentication
	// =============================================================================
	backendGroup := app.Group("/api/backend")
	backendGroup.Use(auth.RequireAPIKey())
	{
		// Requires any API key (pk/sk/rk)
		backendGroup.GET("/stats", handleStats)
	}

	// =============================================================================
	// ADMIN ROUTES - Require secret API key with admin scope
	// =============================================================================
	adminGroup := app.Group("/api/admin")
	adminGroup.Use(auth.RequireSecretKey())
	adminGroup.Use(auth.RequireAdmin())
	{
		// Only works with sk_ keys that have admin:full scope
		adminGroup.GET("/users", handleListAllUsers)
		adminGroup.DELETE("/users/:id", handleDeleteUser)
		adminGroup.POST("/api-keys", handleCreateAPIKey)
	}

	// =============================================================================
	// SCOPED ROUTES - Require specific API key scopes
	// =============================================================================
	analyticsGroup := app.Group("/api/analytics")
	analyticsGroup.Use(auth.RequireAPIKey())
	analyticsGroup.Use(auth.RequireScope("analytics:write"))
	{
		// Requires API key with "analytics:write" scope
		analyticsGroup.POST("/events", handleTrackEvent)
	}

	// Multi-scope example
	dataGroup := app.Group("/api/data")
	dataGroup.Use(auth.RequireAPIKey())
	dataGroup.Use(auth.RequireAllScopes("data:read", "data:export"))
	{
		// Requires API key with BOTH scopes
		dataGroup.GET("/export", handleExportData)
	}

	// Run server
	log.Println("Server running on :8080")
	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

// =============================================================================
// HANDLER EXAMPLES
// =============================================================================

func handleStatus(c forge.Context) error {
	// Public endpoint - no authentication required
	// But auth context is still available if user is authenticated
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	response := map[string]interface{}{
		"status": "ok",
		"authenticated": authCtx != nil && authCtx.IsAuthenticated,
	}
	
	if authCtx != nil && authCtx.IsAuthenticated {
		response["method"] = string(authCtx.Method)
	}
	
	return c.JSON(200, response)
}

func handleProfile(c forge.Context) error {
	// Works with either API key OR user session
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	if authCtx.IsUserAuth {
		// User is logged in via session
		return c.JSON(200, map[string]interface{}{
			"type": "user",
			"user": authCtx.User,
			"session": authCtx.Session,
		})
	}
	
	if authCtx.IsAPIKeyAuth {
		// Authenticated via API key
		return c.JSON(200, map[string]interface{}{
			"type": "api_key",
			"api_key": map[string]interface{}{
				"name":    authCtx.APIKey.Name,
				"type":    authCtx.APIKey.KeyType,
				"scopes":  authCtx.APIKeyScopes,
			},
		})
	}
	
	return c.JSON(401, map[string]string{"error": "not authenticated"})
}

func handleGetMe(c forge.Context) error {
	// Requires user session (enforced by RequireUser middleware)
	user, _ := contexts.RequireUser(c.Request().Context())
	
	return c.JSON(200, map[string]interface{}{
		"id":       user.ID,
		"email":    user.Email,
		"name":     user.Name,
		"verified": user.EmailVerified,
	})
}

func handleUpdateMe(c forge.Context) error {
	user, _ := contexts.RequireUser(c.Request().Context())
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	// Check if API key also present (dual authentication)
	if authCtx.HasAPIKey() {
		// Log that both session and API key are present
		log.Printf("Dual auth: User %s with API key %s", user.Email, authCtx.APIKey.Name)
	}
	
	// Update logic here...
	
	return c.JSON(200, map[string]interface{}{
		"message": "profile updated",
		"user":    user,
	})
}

func handleListSessions(c forge.Context) error {
	user, _ := contexts.RequireUser(c.Request().Context())
	
	// User can only see their own sessions
	return c.JSON(200, map[string]interface{}{
		"user_id":  user.ID,
		"sessions": []string{}, // Would fetch from session service
	})
}

func handleStats(c forge.Context) error {
	// Requires API key (any type)
	apiKey, _ := contexts.RequireAPIKey(c.Request().Context())
	
	return c.JSON(200, map[string]interface{}{
		"api_key_type": apiKey.KeyType,
		"api_key_name": apiKey.Name,
		"stats":        map[string]int{"users": 100, "sessions": 500},
	})
}

func handleListAllUsers(c forge.Context) error {
	// Requires secret API key with admin privileges
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	if !authCtx.CanPerformAdminOp() {
		return c.JSON(403, map[string]string{
			"error": "admin privileges required",
		})
	}
	
	// Admin operation - list all users
	return c.JSON(200, map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": "1", "email": "user1@example.com"},
			{"id": "2", "email": "user2@example.com"},
		},
	})
}

func handleDeleteUser(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	// Double-check admin privileges
	if !authCtx.CanPerformAdminOp() {
		return c.JSON(403, map[string]string{
			"error": "admin privileges required",
		})
	}
	
	userID := c.Param("id")
	
	// Delete user logic here...
	log.Printf("Admin deleted user: %s", userID)
	
	return c.JSON(200, map[string]interface{}{
		"message": "user deleted",
		"user_id": userID,
	})
}

func handleCreateAPIKey(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	// Parse request
	var req struct {
		Name        string                `json:"name"`
		KeyType     apikey.KeyType       `json:"keyType"`
		Scopes      []string              `json:"scopes"`
		Description string                `json:"description"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "invalid request body",
		})
	}
	
	// Validate key type
	if !req.KeyType.IsValid() {
		return c.JSON(400, map[string]string{
			"error": "invalid key type: must be pk, sk, or rk",
		})
	}
	
	// For publishable keys, ensure only safe scopes
	if req.KeyType == apikey.KeyTypePublishable {
		for _, scope := range req.Scopes {
			if !apikey.IsSafeForPublicKey(scope) {
				return c.JSON(400, map[string]interface{}{
					"error": "invalid scope for publishable key",
					"scope": scope,
				})
			}
		}
	}
	
	return c.JSON(201, map[string]interface{}{
		"message": "API key created",
		"key_type": req.KeyType,
		"example_prefix": string(req.KeyType) + "_prod_abc123xyz",
	})
}

func handleTrackEvent(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	// Verify scope (middleware already checked, this is extra validation)
	if !authCtx.HasScope("analytics:write") {
		return c.JSON(403, map[string]string{
			"error": "insufficient scope",
		})
	}
	
	return c.JSON(201, map[string]interface{}{
		"message": "event tracked",
		"api_key": authCtx.APIKey.Name,
	})
}

func handleExportData(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())
	
	// Verify multiple scopes
	if !authCtx.HasAllScopesOf("data:read", "data:export") {
		return c.JSON(403, map[string]interface{}{
			"error": "insufficient scopes",
			"required": []string{"data:read", "data:export"},
			"current": authCtx.APIKeyScopes,
		})
	}
	
	return c.JSON(200, map[string]interface{}{
		"message": "data export started",
		"export_id": "export_123",
	})
}

