package main

import (
	"log"

	"github.com/xraph/authsome"
)

func main() {
	// Example 1: Use default middleware config (backwards compatible)
	auth1 := authsome.New(
		authsome.WithSecret("my-secret-key"),
	)
	_ = auth1

	// Example 2: Customize auth middleware configuration
	auth2 := authsome.New(
		authsome.WithSecret("my-secret-key"),
		authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
			SessionCookieName:   "my_custom_session",
			Optional:            false, // Require authentication for all requests
			AllowAPIKeyInQuery:  false, // Security: don't allow API keys in query params
			AllowSessionInQuery: false, // Security: don't allow session tokens in query params
			APIKeyHeaders:       []string{"Authorization", "X-API-Key", "X-Custom-Key"},
			Context: authsome.ContextConfig{
				AutoDetectFromAPIKey: true,  // Infer app/env from API key
				AutoDetectFromConfig: false, // Don't auto-detect from config
				AppIDHeader:          "X-App-ID",
				EnvironmentIDHeader:  "X-Environment-ID",
			},
		}),
	)
	_ = auth2

	// Example 3: Partial config - only override what you need
	auth3 := authsome.New(
		authsome.WithSecret("my-secret-key"),
		authsome.WithSessionCookieName("my_session"), // This gets used by middleware if not in AuthMiddlewareConfig
		authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
			Optional: true, // Allow unauthenticated requests
			Context: authsome.ContextConfig{
				AutoDetectFromAPIKey: true,
				AutoDetectFromConfig: true, // Enable auto-detect for standalone mode
			},
		}),
	)
	_ = auth3

	// Example 4: Security-first configuration
	auth4 := authsome.New(
		authsome.WithSecret("my-secret-key"),
		authsome.WithAuthMiddlewareConfig(authsome.AuthMiddlewareConfig{
			SessionCookieName:   "secure_session",
			Optional:            false, // Require auth for all routes
			AllowAPIKeyInQuery:  false, // Never allow in query params
			AllowSessionInQuery: false, // Never allow in query params
			Context: authsome.ContextConfig{
				AutoDetectFromAPIKey: true, // Most secure pattern
			},
		}),
	)
	_ = auth4

	log.Println("All authentication instances configured successfully")
	log.Println("The middleware config is now customizable via functional options!")
}
