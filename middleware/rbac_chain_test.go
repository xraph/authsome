package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// recordingChecker satisfies both PermissionChecker and ChainPermissionChecker
// so a test can see which one RequirePermission actually reached for, and with
// what.
type recordingChecker struct {
	allow bool
	err   error

	canCalls  int
	hasCalls  int
	gotSubj   principal.Ref
	gotActors principal.Chain
}

func (c *recordingChecker) HasPermission(_ context.Context, _ id.UserID, _, _ string) (bool, error) {
	c.hasCalls++
	return c.allow, c.err
}

func (c *recordingChecker) Can(
	_ context.Context, subject principal.Ref, actors principal.Chain, _, _ string,
) (bool, error) {
	c.canCalls++
	c.gotSubj = subject
	c.gotActors = actors
	return c.allow, c.err
}

// userOnlyChecker implements PermissionChecker and nothing else, standing in
// for a consumer that never grew a chain-aware method.
type userOnlyChecker struct {
	calls int
}

func (c *userOnlyChecker) HasPermission(_ context.Context, _ id.UserID, _, _ string) (bool, error) {
	c.calls++
	return true, nil
}

// serveGuarded runs one request through AuthMiddleware and then the guard,
// with sess resolved from the bearer token. Using the real auth middleware
// rather than stuffing the context by hand is the point: the bug this pins
// lived in what the middleware puts on the context, not in the guard alone.
func serveGuarded(t *testing.T, sess *session.Session, guard forge.Middleware) int {
	t.Helper()
	code, _ := serveGuardedCapturingActors(t, sess, guard)
	return code
}

// serveGuardedCapturingActors is serveGuarded plus the actor chain the auth
// middleware left on the context, which is what setPrincipalContext writes and
// what any consumer reading middleware.ActorsFrom will see.
func serveGuardedCapturingActors(
	t *testing.T, sess *session.Session, guard forge.Middleware,
) (int, principal.Chain) {
	t.Helper()

	var ctxActors principal.Chain

	auth := middleware.AuthMiddleware(
		func(token string) (*session.Session, error) {
			if sess != nil && token == sess.Token {
				return sess, nil
			}
			return nil, errors.New("invalid token")
		},
		func(uid string) (*user.User, error) {
			parsed, err := id.ParseUserID(uid)
			if err != nil {
				return nil, err
			}
			return &user.User{ID: parsed}, nil
		},
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	router.Use(auth)
	router.Use(guard)
	router.GET("/guarded", func(ctx forge.Context) error {
		ctxActors, _ = middleware.ActorsFrom(ctx.Context())
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
	if sess != nil {
		req.Header.Set("Authorization", "Bearer "+sess.Token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec.Code, ctxActors
}

func delegatedSession(agent principal.Ref) *session.Session {
	return &session.Session{
		ID:            id.NewSessionID(),
		AppID:         id.NewAppID(),
		UserID:        id.NewUserID(),
		Token:         "delegated-token",
		PrincipalKind: principal.KindUser,
		Actors:        principal.Chain{agent},
		ActorGrant:    principal.GrantDelegation,
		DelegationID:  id.NewDelegationID(),
	}
}

// The guard on a delegated session must evaluate the agent as well as the
// human. Checking the human alone is the whole hole: an agent that exchanged a
// narrow grant would arrive holding the human's entire authority.
func TestGuardEvaluatesTheActorChainOnADelegatedSession(t *testing.T) {
	agent := principal.Ref{Kind: principal.KindAgent, ID: "svc_agent"}
	sess := delegatedSession(agent)
	checker := &recordingChecker{allow: true}

	code := serveGuarded(t, sess, middleware.RequirePermission(checker, "read", "document"))

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, 1, checker.canCalls, "a chain-aware checker must be reached through Can")
	assert.Zero(t, checker.hasCalls, "the user-only path must not be taken when the checker is chain-aware")
	assert.Equal(t, principal.UserRef(sess.UserID), checker.gotSubj,
		"the subject stays the human the request is for")
	assert.Equal(t, principal.Chain{agent}, checker.gotActors,
		"the agent must reach the check, or the grant's narrowing is thrown away")
}

// The denial half. Can returning false must be a 403 and not a pass-through.
func TestGuardRefusesWhenTheChainCheckDenies(t *testing.T) {
	agent := principal.Ref{Kind: principal.KindAgent, ID: "svc_agent"}
	checker := &recordingChecker{allow: false}

	code := serveGuarded(t, delegatedSession(agent),
		middleware.RequirePermission(checker, "read", "document"))

	assert.Equal(t, http.StatusForbidden, code)
	assert.Equal(t, 1, checker.canCalls)
}

// Impersonation is the documented exception, and it must survive the wiring.
//
// Session.AuthzActors returns nil for an impersonation, so the admin's own
// permissions are never intersected in: impersonating somebody is precisely
// the request to evaluate as them. If the context carried sess.Actors instead,
// the admin would be able to do LESS than the person they are standing in for,
// which inverts the feature rather than tightening it.
func TestGuardWithholdsTheAdminDuringImpersonation(t *testing.T) {
	admin := id.NewUserID()
	sess := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         id.NewAppID(),
		UserID:        id.NewUserID(),
		Token:         "impersonated-token",
		PrincipalKind: principal.KindUser,
	}
	sess.SetImpersonatedBy(admin)
	require.NotEmpty(t, sess.Actors, "sanity: the session must actually carry the admin")

	checker := &recordingChecker{allow: true}
	code, ctxActors := serveGuardedCapturingActors(t, sess,
		middleware.RequirePermission(checker, "read", "document"))

	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, principal.UserRef(sess.UserID), checker.gotSubj,
		"an impersonated request evaluates as the target")
	assert.Empty(t, ctxActors,
		"setPrincipalContext must write AuthzActors, not Actors: the admin on the context is what a chain-aware guard would intersect in, inverting impersonation")
	assert.Empty(t, checker.gotActors,
		"the impersonating admin must not be intersected into the check")

	// The same property, read straight off the context the middleware built.
	// This is what setPrincipalContext writes, and writing sess.Actors there
	// is what would break the assertion above.
	assert.Empty(t, middleware.AuthzActorsFrom(middleware.WithSession(context.Background(), sess)),
		"AuthzActorsFrom must apply the impersonation exception")
}

// An unauthenticated request must still be a 401, not a check against a zero
// principal. The chain-aware path must be no more permissive at the door than
// the user-only one it replaced.
func TestGuardStillRefusesAnUnauthenticatedRequest(t *testing.T) {
	checker := &recordingChecker{allow: true}

	code := serveGuarded(t, nil, middleware.RequirePermission(checker, "read", "document"))

	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Zero(t, checker.canCalls, "nothing should be checked for a caller that does not exist")
}

// A checker that only knows HasPermission keeps the behavior it always had.
func TestGuardFallsBackToHasPermissionForAUserOnlyChecker(t *testing.T) {
	checker := &userOnlyChecker{}
	sess := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         id.NewAppID(),
		UserID:        id.NewUserID(),
		Token:         "plain-token",
		PrincipalKind: principal.KindUser,
	}

	code := serveGuarded(t, sess, middleware.RequirePermission(checker, "read", "document"))

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, 1, checker.calls)
}
