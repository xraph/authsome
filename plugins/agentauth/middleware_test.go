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
type stubChecker struct {
	allow map[string]bool
}

func (c *stubChecker) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
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
func TestAuthorize_AgentCannotExceedItsOwner(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:write"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true, // owner may read, but not write
	}})

	err := p.Authorize(context.Background(), sess, "write", "invoice")

	require.Error(t, err, "an agent granted write must still be refused when its owner cannot write")
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

// The agent path fails closed. plugin.PermissionGuard deliberately degrades to
// session-only when no checker exists; for agents that would be a hole,
// because the user gate is the entire security model.
func TestAuthorize_NoPermissionCheckerDenies(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.Error(t, err, "no permission checker must deny, never degrade")
}

// A human session is not this plugin's business and must pass straight through.
func TestAuthorize_NonAgentSessionPassesThrough(t *testing.T) {
	p := agentauth.New()
	sess := &session.Session{ID: id.NewSessionID(), UserID: id.NewUserID()}

	assert.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))
}
