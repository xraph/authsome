package middleware

import (
	"errors"
	"net/http"

	"github.com/xraph/forge"
)

// ResourceMetadataChallenge adds the RFC 9728 section 5.1 resource_metadata
// parameter to the WWW-Authenticate header on a 401.
//
// It is how a client bootstraps discovery from a failed call instead of from
// a URL somebody handed it, which is the path the MCP spec uses. An empty
// metadataURL makes the middleware inert, so a deployment that has not
// configured discovery emits nothing rather than a broken hint.
//
// A handler that already set its own challenge keeps it. The RFC 7592 routes
// set a realm of their own, and clobbering it would lose information the
// client can use.
//
// Detection reads the error the inner handler returned rather than the
// response status: forge's Context does not expose a way to swap or inspect
// the underlying http.ResponseWriter from middleware, and forge's own error
// handler (which runs at the top of the middleware chain, after this
// middleware returns) is what actually turns a 401 error into a written
// response. forge.Unauthorized and friends implement an interface with
// StatusCode() int and ResponseBody() any; errors.As is used instead of a
// bare type assertion so a wrapped 401 still matches.
//
// This only sees errors returned by other MIDDLEWARE further down the
// chain, such as middleware/rbac.go and middleware/auth.go, which is where
// every current 401 in this codebase originates. A route handler's
// returned error is converted to a written response by forge before it is
// ever handed back to enclosing middleware, so this would not observe a
// 401 minted that way. The RFC 7592 registration-management handlers are a
// route-handler case, but they set their own WWW-Authenticate header on
// ctx directly before returning (plugins/oauth2provider/register.go,
// authenticateRegistration), which this middleware's overwrite guard below
// picks up regardless of whether the error itself propagates.
func ResourceMetadataChallenge(metadataURL string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		if metadataURL == "" {
			return next
		}
		return func(ctx forge.Context) error {
			err := next(ctx)

			w := ctx.Response()
			if w.Header().Get("WWW-Authenticate") != "" {
				return err
			}

			var responder interface{ StatusCode() int }
			if !errors.As(err, &responder) || responder.StatusCode() != http.StatusUnauthorized {
				return err
			}

			w.Header().Set("WWW-Authenticate",
				`Bearer resource_metadata="`+metadataURL+`"`)
			return err
		}
	}
}
