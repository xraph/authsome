package extension

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"
)

// registerClientAPIProxy mounts a same-origin reverse proxy that forwards
// admin API calls from the dashboard host (this process, in client mode) to
// the upstream authsome service at PortalURL.
//
// Why: Forge's dashboard remote-contributor proxy only forwards the
// contributor protocol routes (/_forge/dashboard/{manifest,pages,widgets,
// settings}). The HTML pages it returns are loaded into the dashboard
// host's origin, but they call admin APIs at <basePath>/v1/admin/... via
// same-origin fetch. Without a local handler for that path the dashboard
// host returns 404 and every admin XHR (settings save, app client-config
// toggle, etc.) breaks in client mode.
//
// This proxy reuses authsome's PortalURL — the operator already
// configured it via WithClientMode(portalURL). It forwards the user's auth
// token from the dashboard's auth_token cookie as Authorization: Bearer so
// upstream's existing auth + permission middleware authenticates the call.
// The browser only ever sees same-origin requests, so no CORS pre-flight
// is needed.
//
// Mount path: <BasePath>/v1/  (default /authsome/v1/). Restricted to /v1/
// rather than the full <BasePath>/ to avoid colliding with the auth pages
// (/login, /callback) and the contributor protocol (/_forge/dashboard/...)
// that the upstream serves under the same BasePath in non-client setups.
//
// No-ops cleanly when client mode isn't active or PortalURL is unset.
func (e *Extension) registerClientAPIProxy(router forge.Router) error {
	if !e.clientMode || e.config.PortalURL == "" || router == nil {
		return nil
	}

	mountPrefix, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		return err
	}

	if err := router.Handle(mountPrefix, proxy); err != nil {
		return fmt.Errorf("authsome: register client API proxy at %s: %w", mountPrefix, err)
	}

	if logger := e.Logger(); logger != nil {
		logger.Info("authsome: registered client-mode API proxy",
			log.String("mount", mountPrefix),
			log.String("portal_url", e.config.PortalURL),
		)
	}
	return nil
}

// buildClientAPIProxy constructs the mount prefix and ReverseProxy used by
// registerClientAPIProxy. Split out so unit tests can exercise the proxy
// behaviour without standing up a forge router.
func (e *Extension) buildClientAPIProxy() (string, *httputil.ReverseProxy, error) {
	upstream, err := url.Parse(e.config.PortalURL)
	if err != nil {
		return "", nil, fmt.Errorf("authsome: parse PortalURL %q for client API proxy: %w", e.config.PortalURL, err)
	}
	if upstream.Scheme == "" || upstream.Host == "" {
		return "", nil, fmt.Errorf("authsome: PortalURL %q must include scheme and host for client API proxy", e.config.PortalURL)
	}

	basePath := strings.TrimRight(e.config.BasePath, "/")
	if basePath == "" {
		basePath = "/authsome"
	}
	// Trailing "/*" intentionally: forge's BunRouterAdapter.Mount has a
	// dual-register quirk (registers both the exact path and a wildcard
	// when the input lacks "/*"), and bunrouter then panics with
	// "routes \"/authsome/v1/\" and \"/authsome/v1/*filepath\" can't
	// both handle GET". Passing "/*" routes through the wildcard-only
	// branch.
	mountPrefix := basePath + "/v1/*"

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Forward to upstream, preserving the inbound path + query.
			// The path already begins with <basePath>/v1/... and upstream
			// expects exactly the same prefix because authsome registers
			// its API under <BasePath>/v1 on its own router too.
			req.URL.Scheme = upstream.Scheme
			req.URL.Host = upstream.Host
			req.Host = upstream.Host

			// Promote the dashboard's auth_token cookie to Authorization:
			// Bearer when the caller hasn't already supplied one. extractToken
			// (auth_pages.go) is the canonical reader the rest of the
			// extension uses for the same cookie name. The token is the
			// upstream session token; upstream's middleware accepts both
			// header and cookie, but stripping the cookie below means the
			// header is what carries it across origins safely.
			if req.Header.Get("Authorization") == "" {
				if tok := extractToken(req); tok != "" {
					req.Header.Set("Authorization", "Bearer "+tok)
				}
			}

			// Drop the inbound Cookie — those cookies were set on the
			// dashboard host's domain and are meaningless (or worse,
			// confusing) to upstream. The Authorization header above
			// carries the auth context.
			req.Header.Del("Cookie")

			// Forge's tenant resolver and CSRF middleware key on these
			// when present; preserve the originals so upstream sees the
			// real client, not the dashboard host loopback.
			if req.Header.Get("X-Forwarded-Host") == "" {
				if h := req.Host; h != "" {
					req.Header.Set("X-Forwarded-Host", h)
				}
			}
			if req.Header.Get("X-Forwarded-Proto") == "" {
				proto := "http"
				if req.TLS != nil {
					proto = "https"
				}
				req.Header.Set("X-Forwarded-Proto", proto)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, perr error) {
			if logger := e.Logger(); logger != nil {
				logger.Warn("authsome: client API proxy: upstream call failed",
					log.String("portal_url", e.config.PortalURL),
					log.String("error", perr.Error()),
				)
			}
			http.Error(w, "upstream authsome unreachable", http.StatusBadGateway)
		},
	}
	return mountPrefix, proxy, nil
}
