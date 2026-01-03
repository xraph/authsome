package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/xraph/authsome"
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/database"
)

func main() {
	// Create Forge application
	app := forge.New()

	// Database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/authsome_custom_schema?sslmode=disable"
		log.Printf("Using default DATABASE_URL: %s", dbURL)
	}

	// Initialize Forge database extension
	dbExt := database.New(app, database.Config{
		Driver: "postgres",
		DSN:    dbURL,
	})

	// Initialize database extension
	if err := dbExt.Initialize(app.Context()); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create AuthSome instance with custom schema
	auth := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithMode(authsome.ModeStandalone),
		authsome.WithDatabaseManager(dbExt.Manager(), "default"),
		authsome.WithDatabaseSchema("auth"), // 🔑 Custom schema for all auth tables
		authsome.WithBasePath("/api/auth"),
		authsome.WithSecret("your-secret-key-change-in-production"),
	)

	// Register plugins (they will also use the custom schema)
	if err := auth.RegisterPlugin(multitenancy.New()); err != nil {
		log.Fatalf("Failed to register multitenancy plugin: %v", err)
	}

	// Initialize AuthSome (creates schema and runs migrations)
	ctx := context.Background()
	if err := auth.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize AuthSome: %v", err)
	}

	// Mount AuthSome routes
	if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
		log.Fatalf("Failed to mount AuthSome: %v", err)
	}

	// Add health check endpoint to verify schema
	app.Router().GET("/health", func(c *forge.Context) error {
		db := auth.GetDB()
		if db == nil {
			return c.JSON(500, map[string]string{"status": "error", "message": "database not available"})
		}

		// Check if we can query the auth schema
		var count int
		err := db.NewSelect().
			Table("users").
			ColumnExpr("COUNT(*)").
			Scan(ctx, &count)

		if err != nil {
			return c.JSON(500, map[string]interface{}{
				"status":  "error",
				"message": fmt.Sprintf("database error: %v", err),
			})
		}

		return c.JSON(200, map[string]interface{}{
			"status": "ok",
			"schema": "auth",
			"users":  count,
		})
	})

	// Add schema info endpoint
	app.Router().GET("/schema-info", func(c *forge.Context) error {
		db := auth.GetDB()
		if db == nil {
			return c.JSON(500, map[string]string{"error": "database not available"})
		}

		// Query PostgreSQL to show which schema contains auth tables
		type SchemaInfo struct {
			SchemaName string `bun:"schemaname"`
			TableName  string `bun:"tablename"`
		}

		var tables []SchemaInfo
		err := db.NewRaw(`
			SELECT schemaname, tablename 
			FROM pg_tables 
			WHERE schemaname IN ('auth', 'public')
			ORDER BY schemaname, tablename
		`).Scan(ctx, &tables)

		if err != nil {
			return c.JSON(500, map[string]interface{}{
				"error": fmt.Sprintf("failed to query schema info: %v", err),
			})
		}

		// Group by schema
		schemaMap := make(map[string][]string)
		for _, t := range tables {
			schemaMap[t.SchemaName] = append(schemaMap[t.SchemaName], t.TableName)
		}

		return c.JSON(200, map[string]interface{}{
			"configured_schema": "auth",
			"schemas":           schemaMap,
			"message":           "All AuthSome tables should be in 'auth' schema",
		})
	})

	// Root endpoint
	app.Router().GET("/", func(c *forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "AuthSome Custom Schema Example",
			"endpoints": map[string]string{
				"health":      "GET /health",
				"schema_info": "GET /schema-info",
				"auth":        "* /api/auth/*",
			},
			"schema": "auth",
		})
	})

	// Start server


	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
