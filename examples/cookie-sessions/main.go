package main

import (
	"context"
	"fmt"
	"log"

	"github.com/xraph/authsome"
	"github.com/xraph/forge"
)

func main() {
	// Create Forge application
	app := forge.New()

	// Note: Database setup depends on your Forge configuration
	// Ensure database extension is properly registered before using AuthSome

	// Create AuthSome instance with cookie support using functional options
	auth := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithBasePath("/auth"),
		authsome.WithSecret("your-super-secret-key-change-in-production"),
		authsome.WithCORSEnabled(true),
		authsome.WithTrustedOrigins([]string{
			"http://localhost:3000",
			"http://localhost:8080",
		}),
		// Cookie configuration using functional options
		authsome.WithSessionCookieEnabled(true),
		authsome.WithSessionCookieName("authsome_session"),
		// Or use WithGlobalCookieConfig for full configuration:
		// authsome.WithGlobalCookieConfig(session.CookieConfig{
		//     Enabled:  true,
		//     Name:     "authsome_session",
		//     Path:     "/",
		//     HttpOnly: true,
		//     SameSite: "Lax",
		// }),
	)

	// Initialize AuthSome
	ctx := context.Background()
	if err := auth.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize AuthSome: %v", err)
	}

	// Mount routes
	if err := auth.Mount(app.Router(), "/auth"); err != nil {
		log.Fatalf("Failed to mount AuthSome: %v", err)
	}

	// Add a test route to verify cookie authentication
	app.Router().GET("/api/me", func(c forge.Context) error {
		// The AuthSome middleware will populate the auth context from the cookie
		user := c.Get("user")
		if user == nil {
			return c.JSON(401, map[string]string{
				"error": "Not authenticated",
			})
		}

		return c.JSON(200, map[string]interface{}{
			"message": "Cookie authentication successful!",
			"user":    user,
		})
	})

	// Example: Configure per-app cookie settings
	app.Router().GET("/example/app-cookie-config", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Per-app cookie configuration example",
			"info":    "Use PUT /auth/apps/:appId/cookie-config to customize cookies per app",
			"example": map[string]interface{}{
				"enabled":  true,
				"name":     "custom_session",
				"path":     "/",
				"httpOnly": true,
				"secure":   true,
				"sameSite": "Strict",
				"domain":   ".example.com",
			},
		})
	})

	// Start server
	port := 8080

	fmt.Printf("🍪 Cookie Sessions Example\n")
	fmt.Printf("   Server: http://localhost:%d\n", port)
	fmt.Printf("   SignUp: POST http://localhost:%d/auth/signup\n", port)
	fmt.Printf("   SignIn: POST http://localhost:%d/auth/signin\n", port)
	fmt.Printf("   Test:   GET  http://localhost:%d/api/me (with cookie)\n\n", port)
	fmt.Printf("💡 Cookies are automatically set on successful authentication!\n\n")

	// Start the Forge app (use your actual Forge server start method)
	log.Printf("Example server ready on port %d", port)
	log.Println("Note: This is a skeleton example. Adapt to your Forge setup.")
}
