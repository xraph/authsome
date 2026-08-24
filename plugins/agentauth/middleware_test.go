package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
)

// stubChecker stands in for the engine's Warden-backed PermissionChecker.
// calls counts invocations of HasPermission so ordering tests can assert the
// user gate was never reached, not just that it returned the right thing.
type stubChecker struct {
	allow map[string]bool
	calls int
}

func (c *stubChecker) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	c.calls++
	return c.allow[userID.String()+"|"+action+"|"+resource], nil
}

func agentSetup(t *testing.T, scopes []string, expires time.Time) (*agentauth.Plugin, *agentauth.MemoryStore, *session.Session, id.UserID) {
	t.Helper()
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
		agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
	)
	userID := id.NewUserID()
	agentID := id.NewAgentID()
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: agentID,
		UserID: userID, OrgID: id.NewOrgID(), Scopes: scopes,
		ExpiresAt: expires, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), g))
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: g.AppID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: agentID, GrantID: g.ID,
	}
	return p, store, sess, userID
}

// The intersection property. If this ever passes, the feature is broken.
// Asserting the specific sentinel (rather than just "an error") matters here:
// a loose require.Error would also pass on a store failure, a wrongly-denying
// scope gate, or a nil checker — none of which demonstrate the intersection.
func TestAuthorize_AgentCannotExceedItsOwner(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:write"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true, // owner may read, but not write
	}})

	err := p.Authorize(context.Background(), sess, "write", "invoice")

	require.ErrorIs(t, err, agentauth.ErrUserNotPermitted, "an agent granted write must still be refused when its owner cannot write")
}

func TestAuthorize_AllowsWhenScopeAndOwnerBothPermit(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	require.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))
}

// Revoking a permission from the owner narrows every agent acting for them on
// the very next request. This is what proves agent authorization never rides
// the stamped-roles fast path on the session.
func TestAuthorize_OwnerLosingPermissionNarrowsAgentImmediately(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	checker := &stubChecker{allow: map[string]bool{userID.String() + "|read|invoice": true}}
	p.SetPermissionChecker(checker)
	require.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))

	checker.allow[userID.String()+"|read|invoice"] = false

	require.Error(t, p.Authorize(context.Background(), sess, "read", "invoice"))
}

func TestAuthorize_MissingScopeIsInsufficientScope(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|write|invoice": true,
	}})

	err := p.Authorize(context.Background(), sess, "write", "invoice")

	require.ErrorIs(t, err, agentauth.ErrInsufficientScope)
}

// The scope gate must run before the user gate, and it must actually be the
// reason a request stops there: this is the one case that separates the two
// orderings, a scope the grant does not cover, combined with an owner who
// would be denied anyway. A test that only checks the returned error cannot
// tell "scope gate ran first" from "user gate ran first and also failed", so
// this asserts the checker was never called.
func TestAuthorize_ScopeGateRunsBeforeUserGate(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	checker := &stubChecker{allow: map[string]bool{}} // owner permitted nothing at all
	p.SetPermissionChecker(checker)

	err := p.Authorize(context.Background(), sess, "write", "invoice") // grant only covers read

	require.ErrorIs(t, err, agentauth.ErrInsufficientScope)
	assert.Equal(t, 0, checker.calls, "the user gate must never run once the scope gate has already failed")
}

func TestAuthorize_ExpiredGrantIsRefused(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(-time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

func TestAuthorize_RevokedGrantIsRefused(t *testing.T) {
	p, store, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})
	require.NoError(t, store.RevokeAgentGrant(context.Background(), sess.GrantID))

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// The confused-deputy case: a session for one principal (different user,
// different agent, different app) points its GrantID at a grant that
// belongs to someone else entirely. Authorize must refuse to decide against
// a grant that does not belong to the session presenting it, regardless of
// what that grant would otherwise permit.
func TestAuthorize_GrantMustBelongToSessionPrincipal(t *testing.T) {
	p, _, ownerSess, ownerUserID := agentSetup(t, []string{"invoices:write"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		ownerUserID.String() + "|write|invoice": true,
	}})

	attackerSess := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         id.NewAppID(),  // different app
		UserID:        id.NewUserID(), // different user
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       id.NewAgentID(), // different agent
		GrantID:       ownerSess.GrantID,
	}

	err := p.Authorize(context.Background(), attackerSess, "write", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive, "a grant must never authorize a session it wasn't issued to")
}

// The agent path fails closed. plugin.PermissionGuard deliberately degrades to
// session-only when no checker exists; for agents that would be a hole,
// because the user gate is the entire security model. Asserting the specific
// sentinel matters: ErrNoPermissionChecker is exported precisely so this can
// be checked exactly, not just "an error of some kind".
func TestAuthorize_NoPermissionCheckerDenies(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrNoPermissionChecker, "no permission checker must deny, never degrade")
}

// A session that is missing entirely carries no permission. Symmetric with
// the no-permission-checker case: both are "the thing needed to decide is
// absent", and both must deny.
func TestAuthorize_NilSessionDenies(t *testing.T) {
	p := agentauth.New()

	err := p.Authorize(context.Background(), nil, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// A session is recognized as an agent by PrincipalKind, but a session
// carrying agent markers (AgentID or GrantID) under a different or missing
// PrincipalKind is internally inconsistent, and inconsistency here must
// deny rather than fall through to pass-through: the fallback direction for
// "is this an agent" is allow, so any misrecognition is a full bypass.
func TestAuthorize_AgentMarkersWithoutAgentKindDenies(t *testing.T) {
	p := agentauth.New()
	sess := &session.Session{
		ID:      id.NewSessionID(),
		UserID:  id.NewUserID(),
		AgentID: id.NewAgentID(),
		GrantID: id.NewAgentGrantID(),
		// PrincipalKind left as the zero value, inconsistent with the agent
		// markers above.
	}

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// A human session is not this plugin's business and must pass straight
// through, for every non-agent PrincipalKind, not just the empty string.
func TestAuthorize_NonAgentSessionPassesThrough(t *testing.T) {
	for _, kind := range []string{"", session.PrincipalKindUser, session.PrincipalKindServiceAccount} {
		t.Run(kind, func(t *testing.T) {
			p := agentauth.New()
			sess := &session.Session{ID: id.NewSessionID(), UserID: id.NewUserID(), PrincipalKind: kind}

			assert.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))
		})
	}
}

// An empty action or resource is never a real authorization question. Without
// this guard a scope misregistered as Grants("", "") would match a route call
// that (by bug) supplies neither.
func TestAuthorize_EmptyActionOrResourceDenies(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	require.Error(t, p.Authorize(context.Background(), sess, "", "invoice"))
	require.Error(t, p.Authorize(context.Background(), sess, "read", ""))
}
