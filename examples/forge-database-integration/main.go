package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/xraph/authsome"
	"github.com/xraph/forge"
	forgedb "github.com/xraph/forge/extensions/database"
)

func main() {
	log.Println("🚀 Starting AuthSome with Forge Database Extension...")

	// Initialize Forge app
	app := forge.NewApp(forge.AppConfig{
		Name:        "authsome-forge-db-demo",
		Version:     "1.0.0",
		Environment: "development",
		HTTPAddress: ":8080",
	})

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "file:authsome_forge_db.db?cache=shared&_fk=1"
	}

	// Configure Forge database extension
	dbExt := forgedb.NewExtension(
		forgedb.WithDatabase(forgedb.DatabaseConfig{
			Name: "default",
			Type: forgedb.TypeSQLite,
			DSN:  dbURL,
		}),
	)

	// Register the database extension
	if err := app.RegisterExtension(dbExt); err != nil {
		log.Fatalf("Failed to register database extension: %v", err)
	}
	log.Println("✅ Forge database extension registered")

	// Initialize AuthSome using Forge's database extension (Method 1: Direct DatabaseManager)
	// Get the manager from DI after extension registration
	ctx := context.Background()

	// Start the app to initialize extensions
	go func() {
		if err := app.Start(ctx); err != nil {
			log.Fatalf("Failed to start app: %v", err)
		}
	}()

	// Wait for extensions to initialize
	// In production, use proper lifecycle management

	// Method 1: Using WithDatabaseFromForge() - Recommended
	auth1 := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithDatabaseFromForge(),
	)

	if err := auth1.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize AuthSome (Method 1): %v", err)
	}
	log.Println("✅ AuthSome initialized using WithDatabaseFromForge()")

	// Method 2: Using DatabaseManager directly
	manager, err := authsome.ResolveDatabaseManager(app.Container())
	if err != nil {
		log.Fatalf("Failed to resolve database manager: %v", err)
	}

	auth2 := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithDatabaseManager(manager, "default"),
	)

	if err := auth2.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize AuthSome (Method 2): %v", err)
	}
	log.Println("✅ AuthSome initialized using WithDatabaseManager()")

	// Method 3: Traditional approach (backwards compatible)
	db, err := manager.SQL("default")
	if err != nil {
		log.Fatalf("Failed to get SQL database: %v", err)
	}

	auth3 := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithDatabase(db),
	)

	if err := auth3.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize AuthSome (Method 3): %v", err)
	}
	log.Println("✅ AuthSome initialized using traditional WithDatabase()")

	// Mount authentication routes (using first instance)
	if err := auth1.Mount(app.Router(), "/api/auth"); err != nil {
		log.Fatalf("Failed to mount auth routes: %v", err)
	}
	log.Println("✅ Auth routes mounted at /api/auth")

	// Display integration methods
	_ = strings.Repeat("=", 60) // separator for visual formatting

	// Run migrations

	// TODO: Add migration runner here

	// In a real app, you would start the server here
	// if err := app.Run(); err != nil {
	// 	log.Fatalf("Server failed: %v", err)
	// }

}
