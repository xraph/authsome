package middleware

import (
	"net/http"
	"strings"

	"github.com/xraph/forge"
)

// CSRFConfig governs the cross-site check on cookie sessions.
type CSRFConfig struct {
	// Enabled turns the check on.
	Enabled bool
	// AllowedOrigins are origins (scheme://host[:port], compared case
	// insensitively) whose cross-site cookie requests are accepted.
	AllowedOrigins []string
}

// safeMethods never change state, so a cross-site one carries no risk the
// browser's own same-origin policy does not already cover.
var safeMethods = map[string]bool{
	http.MethodGet: true, http.MethodHead: true, http.MethodOptions: true, http.MethodTrace: true,
}

// CSRF refuses an unsafe request that was authenticated by the session
// cookie unless the browser says it came from the same site, or its
// Origin is one the operator listed. A cookie travels with any request a
// browser makes, including one another site's page provoked; the
// Sec-Fetch-Site header (every current browser sends it) and the Origin
// header (sent on cross-site unsafe requests) are what tell those apart.
// A request with neither header is refused: a browser that sends none is
// older than this check, and a non-browser client does not use cookies.
//
// Requests authenticated by bearer token, DPoP or API key are untouched:
// a credential the script had to attach itself cannot be attached by
// another site's page.
func CSRF(cfg CSRFConfig) forge.Middleware {
	allowed := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[strings.ToLower(strings.TrimRight(o, "/"))] = true
	}
	return func(next forge.Handler) forge.Handler {
		if !cfg.Enabled {
			return next
		}
		return func(ctx forge.Context) error {
			r := ctx.Request()
			if safeMethods[r.Method] {
				return next(ctx)
			}
			if scheme, ok := CredentialSchemeFrom(ctx.Context()); !ok || scheme != schemeCookie {
				return next(ctx)
			}
			if crossSiteAllowed(r, allowed) {
				return next(ctx)
			}
			return forge.NewHTTPError(http.StatusForbidden, "cross_site_request: a cookie session cannot be used from another site")
		}
	}
}

// crossSiteAllowed decides from the fetch metadata and the Origin header.
func crossSiteAllowed(r *http.Request, allowed map[string]bool) bool {
	switch strings.ToLower(r.Header.Get("Sec-Fetch-Site")) {
	case "same-origin", "none":
		return true
	}
	origin := strings.ToLower(strings.TrimRight(r.Header.Get("Origin"), "/"))
	if origin == "" || origin == "null" {
		return false
	}
	if allowed[origin] {
		return true
	}
	// An Origin that is this host, on the scheme the client used, is the
	// same site even when the browser did not say so.
	return origin == requestScheme(r)+"://"+strings.ToLower(r.Host)
}
