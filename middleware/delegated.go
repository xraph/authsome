package middleware

import (
	"context"
	"errors"

	"github.com/xraph/forge"
)

// ErrDelegatedSession is returned when a session that acts for a user without
// being that user (an agent session, or a session minted by token exchange)
// asks for something only the user's own session may do: minting a credential
// in the user's name, approving an OAuth2 client, impersonating, changing the
// password or the second factor.
var ErrDelegatedSession = errors.New("authsome: an agent or delegated session cannot do this")

// DelegatedSessionMessage is the 403 body DenyDelegatedSession answers with.
const DelegatedSessionMessage = "this endpoint needs the user's own session; an agent or delegated session cannot use it"

// RefuseDelegated returns ErrDelegatedSession when the session resolved onto
// ctx is delegated (see session.Session.IsDelegated), and nil otherwise,
// including when there is no session at all. A method that mints a
// credential calls it so a caller that skipped the HTTP route, and with it
// DenyDelegatedSession, is still refused.
func RefuseDelegated(ctx context.Context) error {
	if sess, ok := SessionFrom(ctx); ok && sess != nil && sess.IsDelegated() {
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
