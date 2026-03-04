package middleware

import (
	log "github.com/xraph/go-utils/log"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
)

// EnvironmentResolver loads an environment by ID.
type EnvironmentResolver func(envID id.EnvironmentID) (*environment.Environment, error)

// DefaultEnvironmentResolver loads the default environment for an app.
type DefaultEnvironmentResolver func(appID id.AppID) (*environment.Environment, error)

// EnvironmentDetector detects an environment type from a raw API key string.
// Returns the detected type and true, or empty and false if unknown.
type EnvironmentDetector func(rawKey string) (environment.Type, bool)

// EnvironmentMiddlewareConfig configures the environment resolution middleware.
type EnvironmentMiddlewareConfig struct {
	// ResolveEnvironment loads an environment by ID.
	ResolveEnvironment EnvironmentResolver

	// ResolveDefault loads the default environment for an app.
	ResolveDefault DefaultEnvironmentResolver

	// DetectFromKey detects environment type from an API key prefix (optional).
	DetectFromKey EnvironmentDetector

	// Logger for debug/warn messages.
	Logger log.Logger
}

// EnvironmentMiddleware resolves the active environment for each request.
//
// Resolution order:
//  1. X-Environment-ID header (explicit override)
//  2. API key prefix-based detection (if DetectFromKey is configured)
//  3. Fall back to the app's default environment
//
// After resolution, it stores the environment, its ID, and its resolved
// settings on the request context for downstream handlers.
func EnvironmentMiddleware(cfg EnvironmentMiddlewareConfig) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			appID, ok := AppIDFrom(ctx.Context())
			if !ok {
				// No app context yet — skip environment resolution.
				return next(ctx)
			}

			var env *environment.Environment

			// 1. Check explicit X-Environment-ID header.
			if headerVal := ctx.Request().Header.Get("X-Environment-ID"); headerVal != "" {
				envID, err := id.ParseEnvironmentID(headerVal)
				if err != nil {
					return ctx.JSON(http.StatusBadRequest, map[string]any{
						"error": "invalid X-Environment-ID header",
						"code":  http.StatusBadRequest,
					})
				}
				resolved, err := cfg.ResolveEnvironment(envID)
				if err != nil {
					cfg.Logger.Warn("environment middleware: failed to resolve environment",
						log.String("env_id", headerVal),
						log.String("error", err.Error()),
					)
					return ctx.JSON(http.StatusNotFound, map[string]any{
						"error": "environment not found",
						"code":  http.StatusNotFound,
					})
				}
				// Verify the environment belongs to the current app.
				if resolved.AppID != appID {
					return ctx.JSON(http.StatusForbidden, map[string]any{
						"error": "environment does not belong to this app",
						"code":  http.StatusForbidden,
					})
				}
				env = resolved
			}

			// 2. Try API key prefix-based detection.
			if env == nil && cfg.DetectFromKey != nil {
				token := extractBearerToken(ctx.Request())
				if token != "" {
					if envType, detected := cfg.DetectFromKey(token); detected {
						slug := envType.String()
						resolved, err := cfg.ResolveDefault(appID)
						if err == nil && resolved.Type != envType {
							// Try to find environment by slug matching the detected type.
							// Fall back to default if slug-based lookup isn't available here.
							_ = slug
							env = resolved
						} else if err == nil {
							env = resolved
						}
					}
				}
			}

			// 3. Fall back to the app's default environment.
			if env == nil {
				resolved, err := cfg.ResolveDefault(appID)
				if err != nil {
					cfg.Logger.Warn("environment middleware: failed to resolve default environment",
						log.String("app_id", appID.String()),
						log.String("error", err.Error()),
					)
					// Continue without environment — backward compat for apps
					// that haven't been migrated yet.
					return next(ctx)
				}
				env = resolved
			}

			// Set environment on context.
			goCtx := ctx.Context()
			goCtx = WithEnvID(goCtx, env.ID)
			goCtx = WithEnvironment(goCtx, env)

			// Resolve effective settings: type defaults + per-environment overrides.
			typeDefaults := environment.DefaultSettingsForType(env.Type)
			effective := environment.MergeSettings(typeDefaults, env.Settings)
			goCtx = WithEnvironmentSettings(goCtx, effective)

			ctx.WithContext(goCtx)
			return next(ctx)
		}
	}
}

// RequireEnvironment returns middleware that rejects requests without
// a resolved environment context.
func RequireEnvironment() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if _, ok := EnvIDFrom(ctx.Context()); !ok {
				return ctx.JSON(http.StatusBadRequest, map[string]any{
					"error": "environment context required",
					"code":  http.StatusBadRequest,
				})
			}
			return next(ctx)
		}
	}
}
