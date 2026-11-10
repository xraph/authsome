package oauth2provider

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/xraph/forge"
)

// CORS for the endpoints a relying party calls from a browser: discovery,
// the key sets, token, userinfo, revoke and dynamic registration. A
// single-page app on another origin signs users in with PKCE and has to
// read these responses itself.
//
// Credentials are never allowed. None of these endpoints reads a cookie:
// token and revoke authenticate the client, userinfo takes a bearer token,
// and the rest are public. Answering with the caller's origin rather than
// "*", and deleting any Allow-Credentials a host-wide CORS middleware set
// before us, keeps the two from combining into a response browsers refuse.
// The endpoints that do use the session cookie (authorize, consent,
// logout) are navigations, not fetches, and get no CORS headers.

const (
	corsAllowMethods  = "GET, POST, OPTIONS"
	corsAllowHeaders  = "Authorization, Content-Type, DPoP"
	corsExposeHeaders = "WWW-Authenticate, DPoP-Nonce, Cache-Control"
	corsMaxAge        = "600"
)

// corsOriginAllowed reports whether origin may read OIDC responses. With
// no CORSOrigins configured, any origin may.
func (p *Plugin) corsOriginAllowed(origin string) bool {
	if len(p.config.CORSOrigins) == 0 {
		return true
	}
	o := normalizeOrigin(origin)
	for _, allowed := range p.config.CORSOrigins {
		if allowed == "*" || normalizeOrigin(allowed) == o {
			return true
		}
	}
	return false
}

// normalizeOrigin lowercases scheme and host and drops a default port, so
// "https://App.example.com:443" and "https://app.example.com" match.
func normalizeOrigin(raw string) string {
	u, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.ToLower(raw)
	}
	scheme, host := strings.ToLower(u.Scheme), strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		return scheme + "://" + host + ":" + port
	}
	return scheme + "://" + host
}

// corsMiddleware adds the CORS headers for an allowed Origin and answers a
// preflight itself.
func (p *Plugin) corsMiddleware() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			r := ctx.Request()
			h := ctx.Response().Header()
			h.Add("Vary", "Origin")
			origin := r.Header.Get("Origin")
			allowed := origin != "" && p.corsOriginAllowed(origin)
			if allowed {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Del("Access-Control-Allow-Credentials")
				h.Set("Access-Control-Expose-Headers", corsExposeHeaders)
			}
			if r.Method == http.MethodOptions {
				if allowed {
					h.Set("Access-Control-Allow-Methods", corsAllowMethods)
					h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
					h.Set("Access-Control-Max-Age", corsMaxAge)
				}
				return ctx.NoContent(http.StatusNoContent)
			}
			return next(ctx)
		}
	}
}

// corsOpts is the route option for an endpoint a browser may call, or
// nothing when CORS is off.
func (p *Plugin) corsOpts() []forge.RouteOption {
	if p.config.DisableCORS {
		return nil
	}
	return []forge.RouteOption{forge.WithMiddleware(p.corsMiddleware())}
}

// withCORS puts the CORS middleware ahead of opts, so an error from a
// later middleware (a scope check, a rate limit) still carries the headers
// the browser needs to read it.
func (p *Plugin) withCORS(opts ...forge.RouteOption) []forge.RouteOption {
	return append(p.corsOpts(), opts...)
}

// registerPreflight answers OPTIONS on path. It stays out of the OpenAPI
// document, so the generated clients don't grow a method per preflight.
func (p *Plugin) registerPreflight(r forge.Router, path string) error {
	if p.config.DisableCORS {
		return nil
	}
	return r.OPTIONS(path, func(ctx forge.Context) error { return ctx.NoContent(http.StatusNoContent) },
		forge.WithMiddleware(p.corsMiddleware()), forge.WithOpenAPIExclude())
}
