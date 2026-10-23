package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/xraph/forge"
	"github.com/xraph/forge/interceptors"
)

// ErrDelegatedSession is returned when a session that acts for a user without
// being that user (an agent session, or a session minted by token exchange)
// asks for something only the user's own session may do: minting a credential
// in the user's name, approving an OAuth2 client, impersonating, changing the
// password or the second factor.
var ErrDelegatedSession = errors.New("authsome: an agent or delegated session cannot do this")

// DelegatedSessionMessage is the 403 body DenyDelegatedSession answers with.
const DelegatedSessionMessage = "this endpoint needs the user's own session; an agent or delegated session cannot use it"

// delegatedCallerKey marks a request whose caller was found to be delegated
// without a session row to say so: a client-mode service learns it from
// introspection, and the JWT path from the token's claims.
type delegatedCallerKey struct{}

// withDelegatedCaller marks ctx as carrying a delegated caller.
func withDelegatedCaller(ctx context.Context) context.Context {
	return context.WithValue(ctx, delegatedCallerKey{}, true)
}

// RefuseDelegated returns ErrDelegatedSession when the caller on ctx is
// delegated, and nil otherwise, including when there is no caller at all. It
// reads the session the auth middleware resolved (see
// session.Session.IsDelegated), and the marker the client-mode and JWT paths
// leave when they have no session row: introspection's delegated field, or a
// JWT's act claim. A method that mints a credential calls it so a caller that
// skipped the HTTP route, and with it DenyDelegatedSession, is still refused.
func RefuseDelegated(ctx context.Context) error {
	if sess, ok := SessionFrom(ctx); ok && sess != nil && sess.IsDelegated() {
		return ErrDelegatedSession
	}
	if marked, ok := ctx.Value(delegatedCallerKey{}).(bool); ok && marked {
		return ErrDelegatedSession
	}
	return nil
}

// DenyDelegatedSession refuses a request whose resolved session acts for a
// user without being that user, with a 403, before the handler runs.
//
// The auth middleware puts an agent session's delegating user on the context
// through WithUserID, so any handler that reads UserIDFrom sees the agent as
// that user. Mount this on every route that mints something in the user's
// name, or the agent walks away with a credential its grant never covered and
// that outlives it.
//
// It has to run after whatever resolves the session. Forge wraps group
// middleware from the last index inward, so index 0 runs first: in a group
// guarded by plugin.SessionGuard or plugin.AdminGuard, append this after the
// guard, never before it. Run first, it sees no session and lets everything
// through. A route on a group without a guard relies on the global auth
// middleware, which has resolved the session before any route middleware
// runs. With no session on the context this passes the request on and leaves
// the 401 to the handler.
func DenyDelegatedSession() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if RefuseDelegated(ctx.Context()) != nil {
				return forge.Forbidden(DelegatedSessionMessage)
			}
			return next(ctx)
		}
	}
}

// AgentSafeMetadataKey is the route metadata key AgentSafe sets.
const AgentSafeMetadataKey = "authsome.agent_safe"

// DelegatedWriteInterceptorName names the interceptor GuardDelegatedWrites
// installs.
const DelegatedWriteInterceptorName = "authsome.deny_delegated_writes"

// DelegatedWriteMessage is the 403 body the write guard answers with.
const DelegatedWriteMessage = "an agent or delegated session cannot use this endpoint; " +
	"an agent acts through its grant, and this route is not marked agent-safe"

// AgentSafe marks a write route as one a delegated session may call. It is
// for routes where the caller can only touch its own credential (ending its
// own session, revoking its own token) and for routes that enforce the
// agent's grant themselves, such as one behind agentauth's Guard. Anything
// that acts on what the user owns stays unmarked.
func AgentSafe() forge.RouteOption {
	return forge.WithMetadata(AgentSafeMetadataKey, true)
}

// IsAgentSafe reports whether route carries the AgentSafe marker.
func IsAgentSafe(route forge.RouteInfo) bool {
	marked, ok := route.Metadata[AgentSafeMetadataKey].(bool)
	return ok && marked
}

// readOnlyMethod reports whether method is one HTTP defines as safe.
func readOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

// DelegatedWriteInterceptor refuses a delegated caller (see RefuseDelegated)
// on any request that isn't GET, HEAD or OPTIONS, unless the route is marked
// AgentSafe. It is the deny-by-default rule for every authsome route: an
// agent's authority comes from its grant, so it reads what its user can read
// and writes only where a route says so.
//
// It is an interceptor rather than a middleware because interceptors see the
// matched route's metadata, which is where AgentSafe lives, and run after
// every middleware, so the session guard of the route's group has already
// resolved the caller.
func DelegatedWriteInterceptor() interceptors.Interceptor {
	return interceptors.NewInterceptor(DelegatedWriteInterceptorName,
		func(ctx forge.Context, route forge.RouteInfo) interceptors.InterceptorResult {
			if readOnlyMethod(ctx.Request().Method) || IsAgentSafe(route) {
				return interceptors.Allow()
			}
			if RefuseDelegated(ctx.Context()) != nil {
				return interceptors.Block(forge.Forbidden(DelegatedWriteMessage))
			}
			return interceptors.Allow()
		})
}

// GuardDelegatedWrites returns a group of router, with no prefix, that runs
// DelegatedWriteInterceptor on every route registered through it or through
// any group made from it. The API registers its own routes this way, and the
// Forge extension hands it to every plugin's RegisterRoutes, so a route a
// plugin adds is refused to delegated sessions until someone marks it
// AgentSafe. If you mount authsome's plugins on a router yourself, pass them
// this.
//
// Create it after any Use calls on router that the routes should get: a group
// copies its parent's middleware when it is made.
func GuardDelegatedWrites(router forge.Router) forge.Router {
	return router.Group("", interceptors.WithGroupInterceptor(DelegatedWriteInterceptor()))
}
