package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xraph/authsome"
	jwtplugin "github.com/xraph/authsome/plugins/jwt"
	"github.com/xraph/forge"
)

func main() {
	// Create logger
	logger := log.Default()
	logger.Println("Starting JWT Plugin Example...")

	// Create Forge app
	app := forge.New()

	// Configure database (using SQLite for simplicity)
	// Note: Database configuration would typically be set via config files or environment variables
	// For demonstration purposes, assuming database is already configured

	// Create AuthSome instance
	auth := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithBasePath("/api/auth"),
	)

	// Register JWT plugin with custom configuration
	jwtPlugin := jwtplugin.NewPlugin(
		jwtplugin.WithIssuer("jwt-example-app"),
		jwtplugin.WithAccessExpiry(3600),     // 1 hour
		jwtplugin.WithRefreshExpiry(2592000), // 30 days
		jwtplugin.WithSigningAlgorithm("HS256"),
		jwtplugin.WithIncludeAppIDClaim(true),
	)

	if err := auth.RegisterPlugin(jwtPlugin); err != nil {
		log.Fatalf("Failed to register JWT plugin: %v", err)
	}

	logger.Println("JWT plugin registered successfully")

	// Initialize AuthSome
	if err := auth.Initialize(context.Background()); err != nil {
		log.Fatalf("Failed to initialize AuthSome: %v", err)
	}

	logger.Println("AuthSome initialized successfully")

	// Mount routes
	if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
		log.Fatalf("Failed to mount AuthSome routes: %v", err)
	}

	logger.Println("Routes mounted successfully")
	logger.Println("")
	logger.Println("Available JWT endpoints:")
	logger.Println("  POST   /api/auth/jwt/keys         - Create JWT signing key")
	logger.Println("  GET    /api/auth/jwt/keys         - List JWT signing keys")
	logger.Println("  POST   /api/auth/jwt/generate     - Generate JWT token")
	logger.Println("  POST   /api/auth/jwt/verify       - Verify JWT token")
	logger.Println("  GET    /api/auth/jwt/jwks         - Get JWKS")
	logger.Println("")

	// Start server in a goroutine
	go func() {
		logger.Println("Starting server on :8080...")
		if err := app.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	if err := app.Stop(context.Background()); err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}

	logger.Println("Server stopped gracefully")
}
