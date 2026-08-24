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
	"github.com/xraph/authsome/user"
)

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
