package main

import (
	"log"
	"os"
	"strings"

	"github.com/xraph/authsome"
	authext "github.com/xraph/authsome/extension"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/forge"
	forgedb "github.com/xraph/forge/extensions/database"
)

func main() {
	log.Println("🚀 Starting AuthSome Forge Extension Demo...")

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "file:authsome_extension.db?cache=shared&_fk=1"
	}

	// Create Forge app
	app := forge.NewApp(forge.AppConfig{
		Name:        "authsome-extension-demo",
		Version:     "1.0.0",
		Environment: "development",
		HTTPAddress: ":8080",
	})

	// Register Forge database extension
	app.RegisterExtension(forgedb.NewExtension(
		forgedb.WithDatabase(forgedb.DatabaseConfig{
			Name: "default",
			Type: forgedb.TypeSQLite,
			DSN:  dbURL,
		}),
	))
	log.Println("✅ Database extension registered")

	// Register AuthSome extension - that's it!
	// Option 1: Minimal setup (uses defaults)
	// app.RegisterExtension(authext.NewExtension())

	// Option 2: With configuration and plugins
	app.RegisterExtension(authext.NewExtension(
		authext.WithMode(authsome.ModeStandalone),
		authext.WithBasePath("/api/auth"),
		authext.WithSecret("demo-secret-key-change-in-production"),
		authext.WithRBACEnforcement(false),
		authext.WithPlugins(
			dashboard.NewPlugin(),
		),
	))
	log.Println("✅ AuthSome extension registered with dashboard plugin")

	// Add your app routes
	app.Router().GET("/", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Welcome to AuthSome + Forge",
			"auth":    "/api/auth",
			"health":  "/health",
		})
	})

	app.Router().GET("/protected", func(c forge.Context) error {
		// This would typically use authentication middleware
		return c.JSON(200, map[string]string{
			"message": "This is a protected route",
		})
	})

	// Display information
	separator := strings.Repeat("=", 60)
	log.Println("\n" + separator)
	log.Println("AuthSome Forge Extension - Running")
	log.Println(separator)
	log.Println("\n📍 Endpoints:")
	log.Println("  🏠 Home:          http://localhost:8080/")
	log.Println("  🔐 Auth API:      http://localhost:8080/api/auth")
	log.Println("  📊 Dashboard:     http://localhost:8080/api/auth/dashboard")
	log.Println("  💚 Health:        http://localhost:8080/health")
	log.Println("\n🔑 Features enabled:")
	log.Println("  ✅ Dashboard Plugin")
	log.Println("  ✅ Full Authentication API")
	log.Println("  ✅ Session Management")
	log.Println("\n" + separator + "\n")

	// Run the app
	if err := app.Run(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
