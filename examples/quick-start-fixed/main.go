package main

import (
	"log"
	"os"

	authext "github.com/xraph/authsome/extension"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/forge"
	forgedb "github.com/xraph/forge/extensions/database"
)

func main() {
	log.Println("🚀 Starting AuthSome with Forge Extensions (Fixed)...")

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "file:authsome_quick.db?cache=shared&_fk=1"
	}

	// Create Forge app
	app := forge.NewApp(forge.AppConfig{
		Name:        "authsome-quick-start",
		Version:     "1.0.0",
		Environment: "development",
		HTTPAddress: ":8080",
	})

	// ✅ FIX: Always provide database configuration programmatically
	// This prevents the "ConfigManager not registered" error
	app.RegisterExtension(forgedb.NewExtension(
		forgedb.WithDatabase(forgedb.DatabaseConfig{
			Name: "default",
			Type: forgedb.TypeSQLite,
			DSN:  dbURL,
		}),
	))
	log.Println("✅ Database extension registered with config")

	// Register AuthSome extension
	app.RegisterExtension(authext.NewExtension(
		authext.WithBasePath("/api/auth"),
		authext.WithSecret("demo-secret-key-change-in-production"),
		authext.WithTrustedOrigins([]string{
			"http://localhost:3000",
			"http://localhost:8080",
		}),
		authext.WithPlugins(
			dashboard.NewPlugin(),
		),
	))
	log.Println("✅ AuthSome extension registered")

	// Add routes
	app.Router().GET("/", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Welcome to AuthSome",
			"auth":    "/api/auth",
			"health":  "/health",
		})
	})

	log.Println("🌟 Server starting on http://localhost:8080")
	log.Println("🔐 Auth API: http://localhost:8080/api/auth")
	log.Println("📊 Dashboard: http://localhost:8080/api/auth/dashboard")

	// Run the app
	if err := app.Run(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
