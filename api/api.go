// Package api provides Forge-style HTTP handlers for the AuthSome engine.
// Routes are registered with struct-based request/response binding and full OpenAPI metadata.
package api

import (
	"net/http"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/sdkgen/openapi"
)

// API wires all AuthSome HTTP handlers together.
type API struct {
	engine     *authsome.Engine
	router     forge.Router
	rootRouter forge.Router
}

// New creates an API from an Engine and an optional Forge router.
func New(engine *authsome.Engine, router ...forge.Router) *API {
	a := &API{engine: engine}
	if len(router) > 0 {
		a.router = router[0]
		a.rootRouter = router[0]
	}
	return a
}

// Handler returns the fully assembled http.Handler with all routes.
// If route registration fails, it logs the error and returns a handler
// that responds with 503 Service Unavailable.
func (a *API) Handler() http.Handler {
	if a.router == nil {
		a.router = forge.NewRouter()
	}
	if err := a.RegisterRoutes(a.router); err != nil {
		a.engine.Logger().Error("authsome: register routes failed", log.String("error", err.Error()))
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "authsome: service unavailable", http.StatusServiceUnavailable)
		})
	}
	return a.router.Handler()
}

// RegisterRoutes registers all AuthSome API routes into the given Forge router
// with full OpenAPI metadata.
func (a *API) RegisterRoutes(router forge.Router) error {
	// Capture the router so handleOpenAPI can use its dynamic spec.
	a.router = router

	// Well-known and JWKS routes must be registered on the root router
	// (not the grouped router) so they appear at /.well-known/* instead
	// of being nested under the extension group prefix.
	rootRouter := a.rootRouter
	if rootRouter == nil {
		rootRouter = router
	}

	// Routes registered on the root router (not nested under extension group).
	rootRegisterers := []func(forge.Router) error{
		a.registerWellKnownRoutes,
		a.registerJWKSRoutes,
	}
	for _, fn := range rootRegisterers {
		if err := fn(rootRouter); err != nil {
			return err
		}
	}

	// Routes registered on the grouped router (nested under extension group).
	registerers := []func(forge.Router) error{
		a.registerAuthRoutes,
		a.registerPasswordRoutes,
		a.registerUserRoutes,
		a.registerSessionRoutes,
		a.registerDeviceRoutes,
		a.registerWebhookRoutes,
		a.registerRBACRoutes,
		a.registerEnvironmentRoutes,
		a.registerAdminRoutes,
		a.registerAppSessionConfigRoutes,
		a.registerAppClientConfigRoutes,
		a.registerClientConfigRoutes,
		a.registerAuthMethodRoutes,
		a.registerBulkRoutes,
		a.registerSecurityEventRoutes,
		a.registerHealthRoutes,
		a.registerSettingsRoutes,
		a.registerIntrospectRoutes,
	}
	for _, fn := range registerers {
		if err := fn(router); err != nil {
			return err
		}
	}
	return nil
}

// ──────────────────────────────────────────────────
// Well-known routes
// ──────────────────────────────────────────────────

func (a *API) registerWellKnownRoutes(router forge.Router) error {
	g := router.Group("/.well-known/authsome", forge.WithGroupTags("well-known"))

	if err := g.GET("/manifest", a.handleManifest,
		forge.WithSummary("Get AuthSome manifest"),
		forge.WithDescription("Returns service manifest with version, base path, and available endpoints."),
		forge.WithOperationID("getManifest"),
		forge.WithResponseSchema(http.StatusOK, "Manifest", map[string]any{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.GET("/openapi", a.handleOpenAPI,
		forge.WithSummary("Get OpenAPI specification"),
		forge.WithDescription("Returns the OpenAPI 3.0 specification for the AuthSome API."),
		forge.WithOperationID("getOpenAPI"),
		forge.WithResponseSchema(http.StatusOK, "OpenAPI spec", map[string]any{}),
		forge.WithErrorResponses(),
	)
}

func (a *API) handleManifest(ctx forge.Context, _ *struct{}) (*map[string]any, error) { //nolint:gocritic // Forge requires pointer return type for handler detection
	manifest := map[string]any{
		"version":   "0.5.0",
		"base_path": a.engine.Config().BasePath,
		"endpoints": []map[string]string{
			{"method": "POST", "path": "/signup", "auth": "none"},
			{"method": "POST", "path": "/signin", "auth": "none"},
			{"method": "POST", "path": "/signout", "auth": "session"},
			{"method": "POST", "path": "/refresh", "auth": "none"},
			{"method": "POST", "path": "/forgot-password", "auth": "none"},
			{"method": "POST", "path": "/reset-password", "auth": "none"},
			{"method": "POST", "path": "/change-password", "auth": "session"},
			{"method": "POST", "path": "/verify-email", "auth": "none"},
			{"method": "GET", "path": "/me", "auth": "session"},
			{"method": "PATCH", "path": "/me", "auth": "session"},
			{"method": "GET", "path": "/sessions", "auth": "session"},
			{"method": "DELETE", "path": "/sessions/{id}", "auth": "session"},
			{"method": "GET", "path": "/devices", "auth": "session"},
			{"method": "DELETE", "path": "/devices/{id}", "auth": "session"},
		},
	}
	return nil, ctx.JSON(http.StatusOK, manifest)
}

func (a *API) handleOpenAPI(ctx forge.Context, _ *struct{}) (*map[string]any, error) { //nolint:gocritic // Forge requires pointer return type for handler detection
	// Prefer the Forge router's dynamically-generated spec when available.
	// This spec is built from the actual registered routes and their OpenAPI
	// metadata, so it always reflects the true API surface.
	if a.router != nil {
		if spec := a.router.OpenAPISpec(); spec != nil {
			return nil, ctx.JSON(http.StatusOK, spec)
		}
	}

	// Fallback: hardcoded generator (for standalone mode without OpenAPI-enabled router).
	var enabledPlugins []string
	for _, p := range a.engine.Plugins().Plugins() {
		enabledPlugins = append(enabledPlugins, p.Name())
	}

	gen := openapi.NewGenerator(openapi.GeneratorConfig{
		Title:          "AuthSome API",
		Description:    "Authentication API powered by AuthSome",
		Version:        "0.5.0",
		BasePath:       a.engine.Config().BasePath,
		EnabledPlugins: enabledPlugins,
	})

	spec := gen.Generate()
	return nil, ctx.JSON(http.StatusOK, spec)
}

// ──────────────────────────────────────────────────
// Health routes
// ──────────────────────────────────────────────────

func (a *API) registerHealthRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("health"))

	return g.GET("/health", a.handleHealth,
		forge.WithSummary("Health check"),
		forge.WithDescription("Returns service health status and database connectivity."),
		forge.WithOperationID("getHealth"),
		forge.WithResponseSchema(http.StatusOK, "Healthy", HealthResponse{}),
		forge.WithErrorResponses(),
	)
}

func (a *API) handleHealth(ctx forge.Context, _ *struct{}) (*HealthResponse, error) {
	if err := a.engine.Store().Ping(ctx.Context()); err != nil {
		resp := &HealthResponse{Status: "unhealthy", Error: err.Error()}
		return nil, ctx.JSON(http.StatusServiceUnavailable, resp)
	}
	resp := &HealthResponse{Status: "healthy"}
	return nil, ctx.JSON(http.StatusOK, resp)
}
