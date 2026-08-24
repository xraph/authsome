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
// chain, such as middleware/rbac.go and middleware/auth.go. A route
// handler's returned error is converted to a written response by forge
// inside its own handler-conversion wrapper before it is ever handed back
// to enclosing middleware, so this middleware cannot observe a 401 minted
// that way at all, whether or not the handler set its own header first.
//
// A route handler that wants the hint on its own 401s has to set the
// header itself before returning. plugins/oauth2provider/register.go's
// authenticateRegistration already did this, for a different reason (it
// sets its own realm on ctx before returning). plugins/oauth2provider
// handleUserInfo (GET /v1/oauth/userinfo) does the same for this hint
// specifically, because AuthMiddleware only soft-resolves the bearer token
// (middleware/auth.go) rather than rejecting, so handleUserInfo's own
// forge.Unauthorized is the actual point of enforcement and this
// middleware never sees it. handleToken's client-authentication 401s
// (POST /v1/oauth/token) are also route-handler errors this middleware
// cannot see, and are left without the header deliberately rather than
// fixed the same way; see the comment on handleToken for why.
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
