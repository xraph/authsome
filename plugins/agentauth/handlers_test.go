package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/store"
)

// A user must be able to see every agent acting for them, and only theirs.
func TestListMyGrants_ShowsOnlyTheCallersGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	me, someoneElse := id.NewUserID(), id.NewUserID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: id.NewAppID(), ClientID: "client_x", Name: "Invoice Reader",
		Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusApproved,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgent(context.Background(), agent))
	mine := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: agent.AppID, AgentID: agent.ID, UserID: me,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	theirs := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: agent.AppID, AgentID: agent.ID, UserID: someoneElse,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), mine))
	require.NoError(t, store.CreateAgentGrant(context.Background(), theirs))

	resp, err := p.ListMyGrants(context.Background(), me)

	require.NoError(t, err)
	require.Len(t, resp.Grants, 1)
	assert.Equal(t, mine.ID.String(), resp.Grants[0].ID)
	assert.Equal(t, "Invoice Reader", resp.Grants[0].AgentName)
}

// A user must not be able to revoke somebody else's delegation.
func TestRevokeMyGrant_RefusesAnotherUsersGrant(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	me, owner := id.NewUserID(), id.NewUserID()
	theirs := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: owner,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), theirs))

	err := p.RevokeMyGrant(context.Background(), me, theirs.ID)

	require.Error(t, err)
	got, gerr := store.GetAgentGrant(context.Background(), theirs.ID)
	require.NoError(t, gerr)
	assert.Nil(t, got.RevokedAt, "the grant must survive an unauthorized revoke attempt")
}

// The security property that matters most: a grant id must never be an
// authorization, so the endpoint must not let a caller distinguish "this
// grant exists but isn't yours" from "this grant doesn't exist at all".
func TestRevokeMyGrant_SameResponseForNotYoursAndNotFound(t *testing.T) {
	memStore := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(memStore))
	me, owner := id.NewUserID(), id.NewUserID()
	theirs := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: owner,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, memStore.CreateAgentGrant(context.Background(), theirs))

	notYoursErr := p.RevokeMyGrant(context.Background(), me, theirs.ID)
	notFoundErr := p.RevokeMyGrant(context.Background(), me, id.NewAgentGrantID())

	require.Error(t, notYoursErr)
	require.Error(t, notFoundErr)
	assert.Equal(t, notFoundErr.Error(), notYoursErr.Error(),
		"a grant id must never be an authorization: both responses must be indistinguishable")
}

func TestRevokeMyGrant_RevokesOwnGrant(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	me := id.NewUserID()
	mine := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: me,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), mine))

	require.NoError(t, p.RevokeMyGrant(context.Background(), me, mine.ID))

	got, err := store.GetAgentGrant(context.Background(), mine.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt)
}

// The critical gap this task closes: revoking a grant must kill the session
// it issued, or an agent session (which carries the delegating human's
// UserID) keeps authenticating as that human on any route not guarded by
// agentauth.Authorize until it separately expires.
func TestRevokeMyGrant_KillsTheSessionItIssued(t *testing.T) {
	p, grant, eng := issuanceSetup(t, &recordingHooks{})

	sess, err := p.IssueAgentSession(context.Background(), grant, agentauth.IssueMeta{})
	require.NoError(t, err)
	_, err = eng.store.GetSession(context.Background(), sess.ID)
	require.NoError(t, err, "the session must exist before revocation")

	require.NoError(t, p.RevokeMyGrant(context.Background(), grant.UserID, grant.ID))

	_, err = eng.store.GetSession(context.Background(), sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound,
		"a session issued under a revoked grant must not keep resolving")
}

// Blocking an agent must disarm it across the org in one action.
func TestBlockAgent_RevokesItsGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	org := id.NewOrgID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: id.NewAppID(), OrgID: org, ClientID: "client_bad",
		Name: "Rogue", Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusApproved,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgent(context.Background(), agent))
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), g))

	require.NoError(t, p.SetAgentStatus(context.Background(), agent.ID, org, agentauth.StatusBlocked))

	got, err := store.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt, "blocking an agent must revoke the grants it holds")
}
