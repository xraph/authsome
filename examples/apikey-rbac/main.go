package main

import (
	"context"
	"fmt"
	"log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

func main() {
	// Initialize AuthSome
	auth := authsome.New()

	if err := auth.Initialize(context.Background()); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Create Forge app
	app := forge.New()

	// Mount AuthSome
	auth.Mount(app.Router(), "/auth")

	// Apply global authentication middleware
	// Note: AuthMiddleware would be applied here if available in the current API
	// app.Router().Use(auth.AuthMiddleware())

	// =============================================================================
	// EXAMPLE 1: RBAC-only permission check (strict)
	// =============================================================================
	// Note: RBAC permission middleware would be used here if available
	app.Router().GET("/api/users",
		handleListUsers,
	)

	// =============================================================================
	// EXAMPLE 2: Flexible check - accepts scope OR RBAC (recommended)
	// =============================================================================
	// Note: Permission middleware would be used here if available
	app.Router().POST("/api/users", handleCreateUser)

	// =============================================================================
	// EXAMPLE 3: Multiple permission options
	// =============================================================================
	// Note: Permission middleware would be used here if available
	app.Router().GET("/api/dashboard", handleDashboard)

	// =============================================================================
	// EXAMPLE 4: Runtime permission check
	// =============================================================================
	// Note: Permission checks would be done in handler if available
	app.Router().DELETE("/api/users/:id", handleDeleteUser)

	// =============================================================================
	// EXAMPLE 5: API Key role management endpoints
	// =============================================================================
	// Note: User authentication would be required here if available
	app.Router().POST("/api/api-keys/:id/roles", handleAssignRole)

	app.Router().GET("/api/api-keys/:id/permissions", handleGetEffectivePermissions)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// =============================================================================
// EXAMPLE HANDLERS
// =============================================================================

func handleListUsers(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())

	return c.JSON(200, map[string]interface{}{
		"message": "List users",
		"auth": map[string]interface{}{
			"method":               string(authCtx.Method),
			"apiKeyRoles":          authCtx.APIKeyRoles,
			"apiKeyPermissions":    authCtx.APIKeyPermissions,
			"effectivePermissions": authCtx.EffectivePermissions,
		},
	})
}

func handleCreateUser(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())

	// The middleware already checked CanAccess("create", "users")
	// This means either:
	// 1. API key has "users:create" scope, OR
	// 2. API key has RBAC permission for create:users

	return c.JSON(201, map[string]interface{}{
		"message": "User created",
		"auth": map[string]interface{}{
			"method":            string(authCtx.Method),
			"hasScope":          authCtx.HasScope("users:create"),
			"hasRBACPermission": authCtx.HasRBACPermission("create", "users"),
		},
	})
}

func handleDashboard(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())

	return c.JSON(200, map[string]interface{}{
		"message": "Dashboard data",
		"auth": map[string]interface{}{
			"authenticated":        authCtx.IsAuthenticated,
			"effectivePermissions": authCtx.EffectivePermissions,
			"delegating":           authCtx.IsDelegatingCreatorPermissions(),
		},
	})
}

func handleDeleteUser(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())

	// Runtime permission check with detailed response
	if !authCtx.CanAccess("delete", "users") {
		return c.JSON(403, map[string]interface{}{
			"error":           "Access denied",
			"reason":          "Missing delete:users permission",
			"yourPermissions": authCtx.EffectivePermissions,
		})
	}

	userID := c.Param("id")

	// Additional check: admins can delete anyone, regular users only themselves
	if !authCtx.IsAdmin() {
		if authCtx.User == nil || authCtx.User.ID.String() != userID {
			return c.JSON(403, map[string]interface{}{
				"error":  "Access denied",
				"reason": "Can only delete your own user",
			})
		}
	}

	return c.JSON(200, map[string]interface{}{
		"message": fmt.Sprintf("User %s deleted", userID),
	})
}

func handleAssignRole(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())

	// Only authenticated users can assign roles
	if !authCtx.IsUserAuth {
		return c.JSON(401, map[string]interface{}{
			"error": "User authentication required",
		})
	}

	keyID := c.Param("id")

	var body struct {
		RoleID string `json:"roleID"`
	}
	if err := c.BindJSON(&body); err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// NOTE: In production, call the API key service here:
	// err := apikeyService.AssignRole(ctx, keyID, roleID, orgID, userID)

	return c.JSON(200, map[string]interface{}{
		"message": fmt.Sprintf("Role %s assigned to API key %s", body.RoleID, keyID),
	})
}

func handleGetEffectivePermissions(c forge.Context) error {
	authCtx, _ := contexts.GetAuthContext(c.Request().Context())

	keyID := c.Param("id")

	// NOTE: In production, call the API key service here:
	// effectivePerms, err := apikeyService.GetEffectivePermissions(ctx, keyID, orgID)

	// For demonstration, return the current auth context permissions
	return c.JSON(200, map[string]interface{}{
		"apiKeyID": keyID,
		"effective": map[string]interface{}{
			"scopes":               authCtx.APIKeyScopes,
			"apiKeyRoles":          authCtx.APIKeyRoles,
			"apiKeyPermissions":    authCtx.APIKeyPermissions,
			"creatorPermissions":   authCtx.CreatorPermissions,
			"effectivePermissions": authCtx.EffectivePermissions,
			"delegating":           authCtx.IsDelegatingCreatorPermissions(),
			"impersonating":        authCtx.IsImpersonating(),
		},
	})
}

// =============================================================================
// UTILITY FUNCTIONS FOR DEMONSTRATION
// =============================================================================

// DemonstratePermissionPatterns shows different permission check patterns
func DemonstratePermissionPatterns(authCtx *contexts.AuthContext) {

	// Pattern 1: Legacy scope check

	// Pattern 2: RBAC permission check

	// Pattern 3: Flexible check (scope OR RBAC)

	// Pattern 4: Multiple permission check

	// Pattern 5: All permissions required

	// Check delegation
	if authCtx.IsDelegatingCreatorPermissions() {

	}

	// Check impersonation
	if authCtx.IsImpersonating() {

	}
}

// DemonstrateScopeMapping shows scope-to-RBAC conversion
func DemonstrateScopeMapping() {

	scopes := []string{
		"users:read",
		"users:write",
		"sessions:create",
		"admin:full",
	}

	for _, scope := range scopes {
		action, resource := apikey.MapScopeToRBAC(scope)

	}

	// Suggest role based on scopes
	suggested := apikey.GenerateSuggestedRole(scopes)

}
