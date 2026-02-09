package middleware

import (
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
)

// AppContext is middleware that extracts app ID from the request
// and injects it into the context for multi-tenant operations.
func AppContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appIDStr := extractAppID(r)

		if appIDStr != "" {
			if appID, err := xid.FromString(appIDStr); err == nil {
				ctx := contexts.SetAppID(r.Context(), appID)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// extractAppID determines the app from the request
// Priority order:
// 1. X-App-ID header (explicit app selection)
// 2. Subdomain extraction (e.g., acme.authsome.dev -> acme)
// 3. JWT token claims (when implemented)
// 4. Default to empty (standalone mode).
func extractAppID(r *http.Request) string {
	// 1. Check for explicit app header
	if appID := r.Header.Get("X-App-Id"); appID != "" {
		return appID
	}

	// 2. Extract from subdomain
	host := r.Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx > 0 {
		host = host[:idx]
	}

	// Extract subdomain
	if idx := strings.Index(host, "."); idx > 0 {
		subdomain := host[:idx]
		// Ignore common non-app subdomains
		if subdomain != "www" && subdomain != "api" && subdomain != "app" {
			return subdomain
		}
	}

	// 3. TODO: Extract from JWT token claims when JWT plugin is integrated
	// if token := extractJWTFromRequest(r); token != "" {
	//     if claims, err := parseJWT(token); err == nil {
	//         if appID, ok := claims["app_id"].(string); ok {
	//             return appID
	//         }
	//     }
	// }

	// 4. No app context (standalone mode)
	return ""
}

// ForgeMiddleware wraps AppContext for use with Forge framework.
func ForgeMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return AppContext(next)
	}
}
