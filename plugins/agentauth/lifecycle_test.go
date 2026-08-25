package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// alwaysChecker stands in for the engine's RBAC checker in tests that only
// need the user gate to stay out of the way.
type alwaysChecker struct{ allow bool }

func (c *alwaysChecker) HasPermission(_ context.Context, _ id.UserID, _, _ string) (bool, error) {
	return c.allow, nil
}

func grantFor(t *testing.T, s agentauth.Store, userID id.UserID, orgID id.OrgID) *agentauth.AgentGrant {
	t.Helper()
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: orgID, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAgentGrant(context.Background(), g))
	return g
}

func assertRevoked(t *testing.T, s agentauth.Store, gid id.AgentGrantID) {
	t.Helper()
	g, err := s.GetAgentGrant(context.Background(), gid)
	require.NoError(t, err)
	assert.NotNil(t, g.RevokedAt, "grant %s should be revoked", gid)
}

// Banning a user must disarm every agent acting for them. Killing sessions
// alone is not enough: the next refresh would mint a fresh one, because the
// grant is what keeps authorizing.
func TestOnAfterUserUpdate_BannedUserLosesGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	g := grantFor(t, store, userID, id.NewOrgID())

	require.NoError(t, p.OnAfterUserUpdate(context.Background(), &user.User{ID: userID, Banned: true}))

	assertRevoked(t, store, g.ID)
}

func TestOnAfterUserUpdate_ActiveUserKeepsGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	g := grantFor(t, store, userID, id.NewOrgID())

	require.NoError(t, p.OnAfterUserUpdate(context.Background(), &user.User{ID: userID, Banned: false}))

	got, err := store.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Nil(t, got.RevokedAt, "an ordinary profile update must not disarm agents")
}

func TestOnAfterUserDelete_RevokesGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	g := grantFor(t, store, userID, id.NewOrgID())

	require.NoError(t, p.OnAfterUserDelete(context.Background(), userID))

	assertRevoked(t, store, g.ID)
}

// TestOnAfterUserUpdate_BannedUserGrantIsRevokedThroughAuthorizeEvenWarm
// proves the cache half of the sweep, not just the store half.
// assertRevoked above reads straight from the store, so it can't tell
// p.cache.clear() apart from a no-op: a mutation that deletes the clear()
// call in revokeUserGrantsBulk still leaves every store-only test in this
// file passing. Warming the cache with one Authorize call before the ban,
// then calling Authorize again after, makes a missing clear() visible:
// without it, the second call is a cache hit against the pre-ban entry and
// keeps authorizing for the rest of the cache ttl even though the grant is
// revoked in the store.
func TestOnAfterUserUpdate_BannedUserGrantIsRevokedThroughAuthorizeEvenWarm(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	p.SetPermissionChecker(&alwaysChecker{allow: true})

	userID := id.NewUserID()
	agentID := id.NewAgentID()
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: agentID,
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), g))
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: g.AppID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: agentID, GrantID: g.ID,
	}

	// Warm the cache: this Authorize call reads the store and caches the
	// still-active grant.
	require.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))

	require.NoError(t, p.OnAfterUserUpdate(context.Background(), &user.User{ID: userID, Banned: true}))

	err := p.Authorize(context.Background(), sess, "read", "invoice")
	assert.ErrorIs(t, err, agentauth.ErrGrantInactive,
		"a warm cache entry must not let a banned user's agent keep authorizing")
}

// Leaving one org must not disarm the user's agents in their other orgs.
func TestOnBeforeMemberRemove_RevokesOnlyThatOrg(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	leaving, staying := id.NewOrgID(), id.NewOrgID()
	gone := grantFor(t, store, userID, leaving)
	kept := grantFor(t, store, userID, staying)

	require.NoError(t, p.OnBeforeMemberRemove(context.Background(), &organization.Member{
		ID: id.NewMemberID(), OrgID: leaving, UserID: userID,
	}))

	assertRevoked(t, store, gone.ID)
	survivor, err := store.GetAgentGrant(context.Background(), kept.ID)
	require.NoError(t, err)
	assert.Nil(t, survivor.RevokedAt, "grants in the user's other orgs must survive")
}
