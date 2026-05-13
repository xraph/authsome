package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/app"
)

// PublishableKeyHeader is the canonical request header carrying a `pk_*`
// publishable key. Frontends set this once on every request so the server
// can route signup, signin, and other public-auth calls to the correct
// app without an explicit `app_id` field in the body.
const PublishableKeyHeader = "X-Publishable-Key"

// PublishableKeyQuery is the query-string fallback for GET endpoints
// where setting a custom header is awkward (mirrors the existing
// /v1/client-config?key=… convention).
const PublishableKeyQuery = "publishable_key"

// AppResolver resolves a publishable key to its owning app. The engine
// satisfies this interface via Engine.ResolveAppByPublicKey; defining it
// locally keeps the middleware package free of an import cycle on the
// authsome root package.
type AppResolver interface {
	ResolveAppByPublicKey(ctx context.Context, publicKey string) (*app.App, error)
}

// PublishableKeyMiddleware extracts a publishable key from the request,
// resolves it to an *app.App via the supplied resolver, and stashes both
// the App and its AppID on the request context (WithApp + WithAppID).
//
// On success: handlers can read the App via AppFrom or the ID via AppIDFrom.
// On a missing key: the middleware is a no-op — handlers decide whether
// the absence is an error (signup/signin reject; some endpoints don't need
// an app context).
// On an unknown key: the request still proceeds with no context AppID set,
// so the handler's resolver returns its own well-shaped 400. The middleware
// itself never aborts — that keeps composition with rate-limit / captcha /
// auth middlewares predictable.
//
// resolver must be non-nil; passing nil panics at construction time so a
// misconfigured server fails loudly during startup rather than silently
// admitting every request to the platform app.
func PublishableKeyMiddleware(resolver AppResolver, logger log.Logger) forge.Middleware {
	if resolver == nil {
		panic("middleware: PublishableKeyMiddleware requires a non-nil AppResolver")
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			key := extractPublishableKey(ctx.Request())
			if key == "" {
				return next(ctx)
			}

			a, err := resolver.ResolveAppByPublicKey(ctx.Context(), key)
			if err != nil || a == nil {
				if logger != nil {
					// Never log the key itself, only its prefix for
					// correlation (pk_live_, pk_test_, pk_stg_).
					logger.Debug("publishable key: resolve failed",
						log.String("prefix", publishableKeyPrefix(key)),
						log.String("path", ctx.Request().URL.Path))
				}
				return next(ctx)
			}

			goCtx := ctx.Context()
			goCtx = WithApp(goCtx, a)
			goCtx = WithAppID(goCtx, a.ID)
			ctx.WithContext(goCtx)
			return next(ctx)
		}
	}
}

// extractPublishableKey reads the publishable key from the canonical
// header first, then the query-string fallback. Returns "" when neither
// is present.
func extractPublishableKey(r *http.Request) string {
	if v := r.Header.Get(PublishableKeyHeader); v != "" {
		return strings.TrimSpace(v)
	}
	if v := r.URL.Query().Get(PublishableKeyQuery); v != "" {
		return strings.TrimSpace(v)
	}
	return ""
}

// publishableKeyPrefix returns the marker prefix of a key (pk_live_,
// pk_test_, pk_stg_) for use in debug logs without leaking the secret
// portion of the key.
func publishableKeyPrefix(key string) string {
	for _, p := range []string{"pk_live_", "pk_test_", "pk_stg_"} {
		if strings.HasPrefix(key, p) {
			return p
		}
	}
	return "unknown"
}
