package environment

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
)

// MiddlewareConfig holds configuration for environment middleware.
type MiddlewareConfig struct {
	HeaderName        string   `json:"headerName"`
	QueryParamName    string   `json:"queryParamName"`
	SubdomainEnabled  bool     `json:"subdomainEnabled"`
	DefaultToDevEnv   bool     `json:"defaultToDevEnv"`
	AllowedSubdomains []string `json:"allowedSubdomains"`
}

// Middleware extracts environment context from the request.
type Middleware struct {
	service *Service
	config  MiddlewareConfig
}

// NewMiddleware creates a new environment middleware.
func NewMiddleware(service *Service, config MiddlewareConfig) *Middleware {
	// Set defaults
	if config.HeaderName == "" {
		config.HeaderName = "X-Environment"
	}

	if config.QueryParamName == "" {
		config.QueryParamName = "env"
	}

	if len(config.AllowedSubdomains) == 0 {
		config.AllowedSubdomains = []string{"dev", "prod", "staging", "preview"}
	}

	return &Middleware{
		service: service,
		config:  config,
	}
}

// Handler returns the middleware handler function.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get app ID from context (set by multitenancy middleware)
		appID, ok := contexts.GetAppID(ctx)
		if !ok {
			// No app in context, skip environment extraction
			next.ServeHTTP(w, r)

			return
		}

		// Extract environment slug from request
		envSlug := m.extractEnvironmentSlug(r)

		// Get environment from database
		var (
			env *Environment
			err error
		)

		if envSlug != "" {
			env, err = m.service.GetEnvironmentBySlug(ctx, appID, envSlug)
		} else if m.config.DefaultToDevEnv {
			// Default to dev environment
			env, err = m.service.GetDefaultEnvironment(ctx, appID)
		}

		// If environment found, add to context
		if err == nil && env != nil {
			ctx = contexts.SetEnvironmentID(ctx, env.ID)
			// Also store full environment in context for additional metadata access
			ctx = SetEnvironment(ctx, env)
		}

		// Continue with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractEnvironmentSlug extracts environment slug from various sources
// Priority: Header > Query Param > Subdomain > Default.
func (m *Middleware) extractEnvironmentSlug(r *http.Request) string {
	// 1. Check X-Environment header
	if envSlug := r.Header.Get(m.config.HeaderName); envSlug != "" {
		return envSlug
	}

	// 2. Check query parameter
	if envSlug := r.URL.Query().Get(m.config.QueryParamName); envSlug != "" {
		return envSlug
	}

	// 3. Check subdomain (if enabled)
	if m.config.SubdomainEnabled {
		if envSlug := m.extractFromSubdomain(r.Host); envSlug != "" {
			return envSlug
		}
	}

	// 4. Return empty (will use default)
	return ""
}

// extractFromSubdomain extracts environment from subdomain
// Example: dev.myapp.com → "dev".
func (m *Middleware) extractFromSubdomain(host string) string {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Split by dots
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	// First part is potential environment
	subdomain := parts[0]

	// Check if it's an allowed environment subdomain
	if slices.Contains(m.config.AllowedSubdomains, subdomain) {
		return subdomain
	}

	return ""
}

// ContextKey type for environment context keys.
type ContextKey string

const (
	// EnvironmentKey is the context key for full environment object.
	EnvironmentKey ContextKey = "environment"
)

// GetEnvironmentID retrieves environment ID from context.
func GetEnvironmentID(ctx context.Context) (xid.ID, bool) {
	return contexts.GetEnvironmentID(ctx)
}

// SetEnvironmentID sets environment ID in context.
func SetEnvironmentID(ctx context.Context, envID xid.ID) context.Context {
	return contexts.SetEnvironmentID(ctx, envID)
}

// GetEnvironment retrieves full environment from context.
func GetEnvironment(ctx context.Context) any {
	if env := ctx.Value(EnvironmentKey); env != nil {
		return env
	}

	return nil
}

// SetEnvironment sets full environment in context.
func SetEnvironment(ctx context.Context, env any) context.Context {
	return context.WithValue(ctx, EnvironmentKey, env)
}
