package authsome

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// roleStamper resolves the role slugs a user holds within an app.
//
// Slugs rather than names or IDs: GetRoleBySlug and ListUsersWithRole already
// address roles that way, and a slug is what a route declares in
// forge.WithAnyRole("admin").
type roleStamper func(ctx context.Context, appID id.AppID, userID id.UserID) ([]string, error)

// roleStampingStore records the principal's roles on a session as it is
// persisted.
//
// Sessions are issued from six places (Engine.IssueSession, two paths in
// service.go, the oauth2provider and magiclink plugins, and the synthetic
// sessions the apikey plugin builds), and every one of them ends at
// store.CreateSession. Stamping here rather than at each issue site is what
// stops the seventh path from silently shipping sessions with no roles on
// them.
//
// The cost of this seam is that a persistence call now makes an RBAC lookup.
// That is deliberate: the alternative is an RBAC lookup on every authenticated
// request, which is the thing stamping exists to avoid.
type roleStampingStore struct {
	store.Store

	stamp  roleStamper
	logger log.Logger
}

// newRoleStampingStore wraps inner so that every session created through it
// carries the roles its principal held at that moment.
//
// The embedded interface means every other method of the aggregate Store
// passes through untouched, so a backend gaining new methods does not need
// this type updated.
func newRoleStampingStore(inner store.Store, stamp roleStamper, logger log.Logger) store.Store {
	if inner == nil || stamp == nil {
		return inner
	}

	if logger == nil {
		logger = log.NewNoopLogger()
	}

	return &roleStampingStore{Store: inner, stamp: stamp, logger: logger}
}

// Unwrap returns the store this one delegates to.
//
// The engine decorates the store it was given, so an accessor like
// Engine.Store() no longer hands back the exact value the caller passed in.
// Anything that needs the backend itself (a test asserting how the engine was
// wired, a caller reaching for a backend-specific method) goes through
// UnwrapStore rather than type-asserting and finding a decorator.
func (s *roleStampingStore) Unwrap() store.Store {
	return s.Store
}

// UnwrapStore returns the backend behind whatever decorators the engine
// installed over it, or s itself when there are none.
//
// Takes and returns any rather than store.Store because the engine hands the
// same backend out through several narrower interfaces (APIKeyStore returns an
// apikey.Store, for one), and every one of them can be sitting behind the same
// decorator.
func UnwrapStore(s any) any {
	for {
		inner, ok := s.(interface{ Unwrap() store.Store })
		if !ok {
			return s
		}

		s = inner.Unwrap()
	}
}

// CreateSession stamps roles onto sess, then persists it.
//
// A session that already carries roles is left alone, so a caller that knows
// better (impersonation, a test fixture, a plugin minting a session for a
// principal it has already resolved) keeps control of what it wrote.
func (s *roleStampingStore) CreateSession(ctx context.Context, sess *session.Session) error {
	if s.shouldStamp(sess) {
		roles, err := s.stamp(ctx, sess.AppID, sess.UserID)
		if err != nil {
			// The sign-in fails. Chosen rather than defaulted into.
			//
			// The alternative, logging and continuing with no roles, is
			// tempting because it keeps authentication working during an RBAC
			// outage, and an empty role set denies rather than allows, so it
			// fails closed. What rules it out is that this stamp is
			// persisted. A session issued with no roles stays that way for
			// its whole lifetime, so a momentary lookup failure would hand
			// someone a session that signs in fine and cannot reach any
			// route that declares a role, for days, with nothing in the UI
			// to explain it and no way to recover but signing out.
			//
			// Failing here is transient by comparison: the user retries and
			// gets a correct session. An RBAC store that cannot answer is
			// usually the same database that is about to fail the session
			// insert two lines below anyway.
			return fmt.Errorf("authsome: resolve session roles: %w", err)
		}

		sess.Roles = roles
	}

	return s.Store.CreateSession(ctx, sess)
}

// RotateSession re-resolves roles as a refresh rotates the session.
//
// Refresh is the moment stamped roles stop being stale. Without this, roles
// are fixed at sign-in and a grant or revocation waits for the user to sign
// out and back in, which on a long-lived session can be days. Re-stamping
// bounds it to the refresh interval instead, and a refresh is rare enough
// that the extra RBAC lookup does not matter the way a per-request one would.
//
// account.RefreshSession mutates the loaded session in place, so sess arrives
// here already carrying the roles it was issued with. That is exactly the
// fallback the error branch below needs.
func (s *roleStampingStore) RotateSession(
	ctx context.Context, sess *session.Session, expectedToken string,
) (bool, error) {
	if s.shouldRestamp(sess) {
		roles, err := s.stamp(ctx, sess.AppID, sess.UserID)
		if err != nil {
			// Keep the roles the session already holds, unlike CreateSession
			// above, which fails the sign-in. The two differ because here
			// there is something to fall back to: roles resolved successfully
			// when this session was issued. Refusing the rotation would log
			// the user out over a transient lookup failure, and continuing
			// with slightly stale roles is a smaller harm than that.
			s.logger.Warn("authsome: session role refresh failed, keeping the roles the session already carried",
				log.String("session_id", sess.ID.String()),
				log.String("error", err.Error()),
			)
		} else {
			sess.Roles = roles
		}
	}

	return s.Store.RotateSession(ctx, sess, expectedToken)
}

// shouldRestamp reports whether sess is one this store should re-resolve roles
// for as it rotates.
//
// Unlike shouldStamp it does not skip a session that already carries roles:
// replacing those is the entire point of re-stamping on refresh.
func (s *roleStampingStore) shouldRestamp(sess *session.Session) bool {
	switch {
	case sess == nil:
		return false
	case !sess.IsHumanPrincipal():
		return false
	case sess.AppID.IsNil() || sess.UserID.IsNil():
		return false
	default:
		return true
	}
}

// shouldStamp reports whether sess is one this store should resolve roles for.
func (s *roleStampingStore) shouldStamp(sess *session.Session) bool {
	switch {
	case sess == nil:
		return false
	case len(sess.Roles) > 0:
		// Already resolved by the caller.
		return false
	case !sess.IsHumanPrincipal():
		// Authorized by scope, and UserID is the zero value here.
		return false
	case sess.AppID.IsNil() || sess.UserID.IsNil():
		// Nothing to scope a lookup by. Reached by fixtures and by the
		// synthetic sessions the apikey plugin builds.
		return false
	default:
		return true
	}
}

// sessionRoleSlugs is the engine's roleStamper: the role slugs userID holds
// within appID.
//
// Scoped to the session's own app rather than the platform app, because a
// user's roles differ per app and the session records which one it was issued
// for.
func (e *Engine) sessionRoleSlugs(ctx context.Context, appID id.AppID, userID id.UserID) ([]string, error) {
	roles, err := e.ListUserRolesInApp(ctx, appID, userID)
	if err != nil {
		return nil, err
	}

	slugs := make([]string, 0, len(roles))

	for _, role := range roles {
		if role != nil && role.Slug != "" {
			slugs = append(slugs, role.Slug)
		}
	}

	if len(slugs) == 0 {
		// nil rather than an empty non-nil slice, so a session with no roles
		// serializes as an absent field rather than [].
		return nil, nil
	}

	return slugs, nil
}
