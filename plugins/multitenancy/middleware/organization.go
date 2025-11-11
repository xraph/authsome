package middleware

import (
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/interfaces"
)

// OrganizationContext is middleware that extracts organization ID from the request
// and injects it into the context for multi-tenant operations
func OrganizationContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgIDStr := extractOrganizationID(r)

		if orgIDStr != "" {
			orgID, _ := xid.FromString(orgIDStr)
			// if err != nil {
			// 	return c.JSON(400, map[string]string{"error": "invalid organization ID"})
			// }
			ctx := interfaces.SetOrganizationID(r.Context(), orgID)
			// ctx := context.WithValue(r.Context(), interfaces.OrganizationContextKey, orgID)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// extractOrganizationID determines the organization from the request
// Priority order:
// 1. X-Organization-ID header (explicit org selection)
// 2. Subdomain extraction (e.g., acme.authsome.dev -> acme)
// 3. JWT token claims (when implemented)
// 4. Default to empty (standalone mode)
func extractOrganizationID(r *http.Request) string {
	// 1. Check for explicit org header
	if orgID := r.Header.Get("X-Organization-ID"); orgID != "" {
		return orgID
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
		// Ignore common non-org subdomains
		if subdomain != "www" && subdomain != "api" && subdomain != "app" {
			return subdomain
		}
	}

	// 3. TODO: Extract from JWT token claims when JWT plugin is integrated
	// if token := extractJWTFromRequest(r); token != "" {
	//     if claims, err := parseJWT(token); err == nil {
	//         if orgID, ok := claims["org_id"].(string); ok {
	//             return orgID
	//         }
	//     }
	// }

	// 4. No organization context (standalone mode)
	return ""
}

// ForgeMiddleware wraps OrganizationContext for use with Forge framework
func ForgeMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return OrganizationContext(next)
	}
}
