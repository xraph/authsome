package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
)

func approvedAgent(t *testing.T, s agentauth.Store, orgID id.OrgID, clientID string) *agentauth.Agent {
	t.Helper()
	a := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: id.NewAppID(), OrgID: orgID,
		ClientID: clientID, Name: "Test Agent",
		Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAgent(context.Background(), a))
	return a
}

func TestEvaluate_BlockedOrgRefusesEvenApprovedAgent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_blocked")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeBlocked,
	}))

	err := p.Evaluate(context.Background(), "client_blocked", id.NewUserID(), org, []string{"invoices:read"})

	require.Error(t, err, "a blocked org must refuse consent even for an approved agent")
}

func TestEvaluate_AllowlistRefusesPendingAgent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	a := approvedAgent(t, store, org, "client_pending")
	a.Status = agentauth.StatusPending
	require.NoError(t, store.UpdateAgent(context.Background(), a))
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeAllowlist,
	}))

	err := p.Evaluate(context.Background(), "client_pending", id.NewUserID(), org, []string{"invoices:read"})

	require.Error(t, err)
}

func TestEvaluate_UnmappedScopeRefusedAtConsent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_open")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	err := p.Evaluate(context.Background(), "client_open", id.NewUserID(), org, []string{"invoices:delete"})

	require.Error(t, err, "a scope with no warden mapping must never reach a stored grant")
}

func TestEvaluate_ScopeOutsideOrgCeilingRefused(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
		agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_ceiling")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, AllowedScopes: []string{"invoices:read"},
	}))

	err := p.Evaluate(context.Background(), "client_ceiling", id.NewUserID(), org, []string{"invoices:write"})

	require.Error(t, err)
}

func TestEvaluate_OpenOrgAllowsMappedScope(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_ok")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	err := p.Evaluate(context.Background(), "client_ok", id.NewUserID(), org, []string{"invoices:read"})

	require.NoError(t, err)
}

// An org with no policy row falls back to open. Changing this default is a
// policy decision, not an implementation detail, so it gets its own test.
// The org that registered an agent governs it, even when the consenting
// session carries no org context of its own. Keying policy off the session's
// org alone would let a member of a blocked org authorize the agent simply by
// signing in without an active organization.
func TestEvaluate_AgentOrgGovernsWhenSessionHasNoOrg(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_orgowned")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeBlocked,
	}))

	// Zero org id, as an app-scoped session would produce.
	err := p.Evaluate(context.Background(), "client_orgowned", id.NewUserID(), id.OrgID{}, []string{"invoices:read"})

	require.Error(t, err, "the agent's own org policy must apply when the session carries no org")
}

func TestEvaluate_MissingPolicyDefaultsToOpen(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_nopolicy")

	err := p.Evaluate(context.Background(), "client_nopolicy", id.NewUserID(), org, []string{"invoices:read"})

	require.NoError(t, err)
}

func TestCreateGrant_ClampsTTLToOrgCeiling(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store), agentauth.WithDefaultGrantTTL(90*24*time.Hour))
	org := id.NewOrgID()
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, MaxGrantTTL: 24 * time.Hour,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"}, RequestedTTL: 365 * 24 * time.Hour,
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), g.ExpiresAt, time.Minute,
		"the org ceiling must win over both the request and the plugin default")
}

func TestCreateGrant_RejectsZeroUser(t *testing.T) {
	p := agentauth.New()

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), OrgID: id.NewOrgID(),
		Scopes: []string{"invoices:read"},
	})

	require.Error(t, err, "a grant with no delegating human must be impossible to create")
}
