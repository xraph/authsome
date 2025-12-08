package main

import (
	"context"
	"log"

	"github.com/xraph/authsome/core/middleware"
	"github.com/xraph/authsome/extension"
	"github.com/xraph/forge"
)

func main() {
	// Example 1: Extension with default middleware config
	app1 := forge.New()
	ext1 := extension.NewExtension(
		extension.WithBasePath("/auth"),
		extension.WithSecret("my-secret-key"),
	)
	if err := app1.RegisterExtension(ext1); err != nil {
		log.Fatal(err)
	}
	_ = app1

	// Example 2: Extension with custom middleware config
	app2 := forge.New()
	ext2 := extension.NewExtension(
		extension.WithBasePath("/api/auth"),
		extension.WithSecret("my-secret-key"),
		extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
			SessionCookieName:   "my_custom_session",
			Optional:            false, // Require authentication for all requests
			AllowAPIKeyInQuery:  false, // Security best practice
			AllowSessionInQuery: false, // Security best practice
			APIKeyHeaders:       []string{"Authorization", "X-API-Key", "X-Custom-Key"},
			Context: middleware.ContextConfig{
				AutoDetectFromAPIKey: true,  // Infer app/env from API key
				AutoDetectFromConfig: false, // Don't auto-detect from config
				AppIDHeader:          "X-App-ID",
				EnvironmentIDHeader:  "X-Environment-ID",
			},
		}),
	)
	if err := app2.RegisterExtension(ext2); err != nil {
		log.Fatal(err)
	}
	_ = app2

	// Example 3: Extension with partial middleware config
	app3 := forge.New()
	ext3 := extension.NewExtension(
		extension.WithBasePath("/auth"),
		extension.WithSecret("my-secret-key"),
		extension.WithSessionCookieName("my_session"),
		extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
			Optional: true, // Allow unauthenticated requests
			Context: middleware.ContextConfig{
				AutoDetectFromAPIKey: true,
				AutoDetectFromConfig: true, // Enable auto-detect for standalone mode
			},
		}),
	)
	if err := app3.RegisterExtension(ext3); err != nil {
		log.Fatal(err)
	}
	_ = app3

	// Example 4: Security-first configuration
	app4 := forge.New()
	ext4 := extension.NewExtension(
		extension.WithBasePath("/auth"),
		extension.WithSecret("production-secret-key"),
		extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
			SessionCookieName:   "secure_session",
			Optional:            false, // Require auth
			AllowAPIKeyInQuery:  false, // Never in production
			AllowSessionInQuery: false, // Never in production
			Context: middleware.ContextConfig{
				AutoDetectFromAPIKey: true,
			},
		}),
	)
	if err := app4.RegisterExtension(ext4); err != nil {
		log.Fatal(err)
	}
	_ = app4

	log.Println("All extension instances configured successfully")
	log.Println("The middleware config is now customizable via extension options!")
}

// Example of running the extension with custom middleware config
func runExample() {
	app := forge.New()

	// Create extension with custom middleware config
	authExt := extension.NewExtension(
		extension.WithBasePath("/api/auth"),
		extension.WithSecret("my-secret-key"),
		extension.WithAuthMiddlewareConfig(middleware.AuthMiddlewareConfig{
			SessionCookieName: "app_session",
			Optional:          true,
			Context: middleware.ContextConfig{
				AutoDetectFromAPIKey: true,
				AppIDHeader:          "X-App-ID",
				EnvironmentIDHeader:  "X-Environment-ID",
			},
		}),
	)

	// Register and start
	if err := app.RegisterExtension(authExt); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Now all AuthSome routes use the custom middleware configuration
	log.Println("Extension started with custom middleware config")
	log.Println("Base path:", authExt.GetBasePath())

	// Your app routes here...
	app.Router().GET("/", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"message": "Hello from Forge with AuthSome extension!",
		})
	})

	log.Println("Server is ready. Authentication middleware configured.")
}
