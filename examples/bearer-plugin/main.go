package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xraph/authsome"
	bearerplugin "github.com/xraph/authsome/plugins/bearer"
	"github.com/xraph/forge"
)

func main() {
	// Create logger
	logger := log.Default()
	logger.Println("Starting Bearer Plugin Example...")

	// Create Forge app
	app := forge.New()

	// Create AuthSome instance
	auth := authsome.New(
		authsome.WithForgeApp(app),
		authsome.WithBasePath("/api/auth"),
	)

	// Register Bearer plugin with custom configuration
	bearerPlugin := bearerplugin.NewPlugin(
		bearerplugin.WithTokenPrefix("Bearer"), // Default, but shown for clarity
		bearerplugin.WithValidateIssuer(false),
	)

	if err := auth.RegisterPlugin(bearerPlugin); err != nil {
		log.Fatalf("Failed to register bearer plugin: %v", err)
	}

	logger.Println("Bearer plugin registered successfully")

	// Initialize AuthSome
	if err := auth.Initialize(context.Background()); err != nil {
		log.Fatalf("Failed to initialize AuthSome: %v", err)
	}

	logger.Println("AuthSome initialized successfully")

	// Mount routes
	if err := auth.Mount(app.Router(), "/api/auth"); err != nil {
		log.Fatalf("Failed to mount AuthSome routes: %v", err)
	}

	// Create a protected route that requires bearer authentication
	api := app.Router().Group("/api")
	{
		// The auth middleware will automatically try bearer token authentication
		api.GET("/protected", protectedHandler,
			forge.WithMiddleware(auth.AuthMiddleware()),
		)
	}

	logger.Println("Routes mounted successfully")
	logger.Println("")
	logger.Println("Bearer token authentication is now available!")
	logger.Println("The bearer plugin registers a strategy that extracts tokens from:")
	logger.Println("  - Authorization: Bearer <token>")
	logger.Println("")
	logger.Println("Example usage:")
	logger.Println("  1. Sign in to get a session token:")
	logger.Println("     POST /api/auth/signin")
	logger.Println("     { \"email\": \"user@example.com\", \"password\": \"password123\" }")
	logger.Println("")
	logger.Println("  2. Use the session token as a bearer token:")
	logger.Println("     GET /api/protected")
	logger.Println("     Authorization: Bearer <session_token>")
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

func protectedHandler(c forge.Context) error {
	return c.JSON(200, map[string]interface{}{
		"message": "You are authenticated via bearer token!",
		"user":    "John Doe", // In real app, get from auth context
	})
}
