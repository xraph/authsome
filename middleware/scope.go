package middleware

import (
	"context"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/session"
)

// OAuth2 access tokens are sessions that carry a ClientID. Such a session is
// not an ordinary login: it was issued to a third party for a set of scopes,
// and it must only reach handlers that were written to check those scopes.
// The auth middleware therefore parks it instead of installing it, and a
// route opts in with RequireScope, which admits the session when it holds
// the scopes the route names. A route nobody tagged sees no session at all,
// which is the safe failure: a resource that forgot to declare its scopes
// refuses OAuth2 tokens rather than honouring them unconditionally.

// PendingOAuthSession is an OAuth2-issued session the auth middleware
// resolved but has not admitted to the request.
type PendingOAuthSession struct {
	Session *session.Session
	// installed is the request context with the session and everything the
	// middleware resolved alongside it (user, principal, scope) in place,
	// ready to be adopted once a route admits the session.
	installed context.Context
}

// WithPendingOAuthSession parks p on ctx. The auth middleware calls it; tests
// call it to stand in for the middleware.
func WithPendingOAuthSession(ctx context.Context, p *PendingOAuthSession) context.Context {
	return context.WithValue(ctx, ctxKeyPendingOAuth, p)
}

// PendingOAuthSessionFrom returns the parked OAuth2 session, if any.
func PendingOAuthSessionFrom(ctx context.Context) (*PendingOAuthSession, bool) {
	p, ok := ctx.Value(ctxKeyPendingOAuth).(*PendingOAuthSession)
	return p, ok && p != nil
}

// parkOAuthSession keeps an OAuth2-issued session out of the request
// context until a RequireScope route admits it. base is the context before
// the middleware installed anything; installed is the fully built one. Any
// other session is returned installed, as before.
func parkOAuthSession(base, installed context.Context) context.Context {
	sess, ok := SessionFrom(installed)
	if !ok || sess == nil || sess.ClientID == "" {
		return installed
	}
	return WithPendingOAuthSession(base, &PendingOAuthSession{Session: sess, installed: installed})
}

// HasScopes reports whether sess holds every scope in scopes.
func HasScopes(sess *session.Session, scopes ...string) bool {
	if sess == nil {
		return false
	}
	have := make(map[string]bool, len(sess.Scopes))
	for _, s := range sess.Scopes {
		have[s] = true
	}
	for _, s := range scopes {
		if !have[s] {
			return false
		}
	}
	return true
}

// RequireScope admits a parked OAuth2 session to the route when it holds
// every named scope, and answers 403 insufficient_scope when it does not
// (RFC 6750 section 3.1). With no scopes named, any OAuth2 session is
// admitted. An ordinary session, or no session, passes through untouched:
// the route's own authentication rules still apply to those.
func RequireScope(scopes ...string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			pending, ok := PendingOAuthSessionFrom(ctx.Context())
			if !ok {
				return next(ctx)
			}
			if !HasScopes(pending.Session, scopes...) {
				ctx.SetHeader("WWW-Authenticate", `Bearer error="insufficient_scope"`)
				return forge.NewHTTPError(403, "the token does not hold the scope this resource requires") //nolint:mnd // HTTP status
			}
			if pending.installed != nil {
				ctx.WithContext(pending.installed)
			} else {
				// A pending session placed directly (tests, custom
				// middleware) carries no prepared context; install the
				// identity the session itself names.
				goCtx := WithSession(ctx.Context(), pending.Session)
				goCtx = WithSessionID(goCtx, pending.Session.ID)
				goCtx = WithAppID(goCtx, pending.Session.AppID)
				if !pending.Session.UserID.IsNil() {
					goCtx = WithUserID(goCtx, pending.Session.UserID)
				}
				ctx.WithContext(goCtx)
			}
			return next(ctx)
		}
	}
}
