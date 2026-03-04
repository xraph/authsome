// Example: standalone AuthSome server with memory store and password + social plugins.
//
// Run:
//
//	go run ./_examples/standalone/
//
// Then interact with the API:
//
//	# Health check
//	curl http://localhost:8080/v1/auth/health
//
//	# Sign up
//	curl -X POST http://localhost:8080/v1/auth/signup \
//	  -H 'Content-Type: application/json' \
//	  -d '{"email":"user@example.com","password":"SecureP@ss1","name":"Alice"}'
//
//	# Sign in
//	curl -X POST http://localhost:8080/v1/auth/signin \
//	  -H 'Content-Type: application/json' \
//	  -d '{"email":"user@example.com","password":"SecureP@ss1"}'
//
//	# Get current user (use session_token from sign-in response)
//	curl http://localhost:8080/v1/auth/me \
//	  -H 'Authorization: Bearer <session_token>'
//
//	# OpenAPI spec
//	curl http://localhost:8080/.well-known/authsome/openapi
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/store/memory"
)

func main() {
	logger := log.NewBeautifulLogger("authsome")

	// Create the in-memory store (swap with pgstore.New(db) for production).
	store := memory.New()

	// Build the AuthSome engine with desired plugins.
	engine, err := authsome.NewEngine(
		authsome.WithStore(store),
		authsome.WithLogger(logger),
		authsome.WithDisableMigrate(), // memory store has no migrations

		// Enable the password plugin (email + password authentication).
		authsome.WithPlugin(password.New()),

		// Add more plugins as needed:
		// authsome.WithPlugin(social.New(social.WithGitHub(clientID, secret, callbackURL))),
		// authsome.WithPlugin(magiclink.New()),
		// authsome.WithPlugin(mfa.New()),
		// authsome.WithPlugin(apikey.New()),
	)
	if err != nil {
		logger.Fatal("create engine", log.Error(err))
	}

	// Start the engine (runs migrations if enabled, initializes plugins).
	if err := engine.Start(context.Background()); err != nil {
		logger.Fatal("start engine", log.Error(err))
	}

	// Build the API handler with a Forge router.
	a := api.New(engine)
	router := forge.NewRouter()

	// Apply auth middleware so protected endpoints resolve the current user.
	router.Use(middleware.AuthMiddleware(
		engine.ResolveSessionByToken,
		engine.ResolveUser,
		logger,
	))

	// Register all AuthSome API routes.
	if err := a.RegisterRoutes(router); err != nil {
		logger.Fatal("register routes", log.Error(err))
	}

	addr := ":8080"
	logger.Info("authsome standalone server starting", log.String("addr", addr))

	srv := &http.Server{Addr: addr, Handler: router.Handler()}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("shutting down...")
		_ = engine.Stop(context.Background())
		_ = srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
