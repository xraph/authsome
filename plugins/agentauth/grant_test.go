package agentauth_test

import (
	"context"
	"errors"
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

// stubPolicyStore wraps a real Store but always answers GetOrgPolicy with a
// canned policy, regardless of which org is asked about. It exists to get an
// OrgAgentPolicy carrying a mode that PutOrgPolicy would now refuse to store
// in front of Evaluate/checkPolicy, so the default-deny branch is reachable
// in a test at all.
type stubPolicyStore struct {
	agentauth.Store
	policy *agentauth.OrgAgentPolicy
}

func (s *stubPolicyStore) GetOrgPolicy(_ context.Context, _ id.OrgID) (*agentauth.OrgAgentPolicy, error) {
	return s.policy, nil
}

// erroringAgentLookupStore wraps a real Store but makes GetAgentByClientID
// fail with an arbitrary, non-ErrNotFound error, to prove a genuine backend
// failure denies rather than being misread as "not an agent".
type erroringAgentLookupStore struct {
	agentauth.Store
	err error
}

func (s *erroringAgentLookupStore) GetAgentByClientID(_ context.Context, _ string) (*agentauth.Agent, error) {
	return nil, s.err
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
	assert.Contains(t, err.Error(), "does not allow agent delegation")
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
	assert.Contains(t, err.Error(), "not approved")
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
	assert.Contains(t, err.Error(), "unknown delegation scope")
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
	assert.Contains(t, err.Error(), "not permitted by organization policy")
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
	assert.Contains(t, err.Error(), "does not allow agent delegation")
}

// A consuming org's own block must also be enforced, even when the agent is
// registered under a different, permissive org. Preferring only the agent's
// org (the previous fix for the case above) would mean a blocked org could
// only ever enforce that block against agents it registered itself, never
// against an agent one of its own members brings in from elsewhere — which
// is the opposite of what ModeBlocked is supposed to mean.
func TestEvaluate_SessionOrgBlockDeniesDespitePermissiveAgentOrg(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	agentOrg := id.NewOrgID()
	sessionOrg := id.NewOrgID()
	approvedAgent(t, store, agentOrg, "client_crossorg")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: agentOrg, Mode: agentauth.ModeOpen,
	}))
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: sessionOrg, Mode: agentauth.ModeBlocked,
	}))

	err := p.Evaluate(context.Background(), "client_crossorg", id.NewUserID(), sessionOrg, []string{"invoices:read"})

	require.Error(t, err, "a consuming org's block must be enforced even when the agent's own org is permissive")
	assert.Contains(t, err.Error(), "does not allow agent delegation")
}

// An org with no policy row falls back to open. Changing this default is a
// policy decision, not an implementation detail, so it gets its own test.
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

// A client_id that isn't a registered agent at all is not this gate's
// business — an ordinary OAuth2 client must sail through untouched.
func TestEvaluate_UnknownClientIDAllows(t *testing.T) {
	p := agentauth.New()

	err := p.Evaluate(context.Background(), "not_an_agent_client", id.NewUserID(), id.NewOrgID(), []string{"anything"})

	require.NoError(t, err, "a client that is not a registered agent is not this gate's business")
}

// A policy nobody can interpret must not be treated as permission. A bare
// string PolicyMode has "" as its zero value and nothing upstream guarantees
// the value is one of the three known constants once it leaves the store, so
// the switch that reads it must default to deny, not fall through to allow.
func TestEvaluate_UnrecognizedPolicyModeDenies(t *testing.T) {
	mem := agentauth.NewMemoryStore()
	org := id.NewOrgID()
	approvedAgent(t, mem, org, "client_badmode")
	store := &stubPolicyStore{
		Store:  mem,
		policy: &agentauth.OrgAgentPolicy{OrgID: org, Mode: agentauth.PolicyMode("bogus")},
	}
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)

	err := p.Evaluate(context.Background(), "client_badmode", id.NewUserID(), org, []string{"invoices:read"})

	require.Error(t, err, "a policy nobody can interpret must not be treated as permission")
	assert.Contains(t, err.Error(), "not recognized")
}

// A genuine store failure must deny, not collapse into the not-found branch
// (which allows) that sits right above it in Evaluate.
func TestEvaluate_StoreErrorDenies(t *testing.T) {
	mem := agentauth.NewMemoryStore()
	store := &erroringAgentLookupStore{Store: mem, err: errors.New("boom: connection reset")}
	p := agentauth.New(agentauth.WithStore(store))

	err := p.Evaluate(context.Background(), "client_whatever", id.NewUserID(), id.NewOrgID(), []string{"invoices:read"})

	require.Error(t, err, "a genuine store error must deny, not be mistaken for an unregistered client")
}

func TestCreateGrant_ClampsTTLToOrgCeiling(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithDefaultGrantTTL(90*24*time.Hour),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_ttlceiling")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, MaxGrantTTL: 24 * time.Hour,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"}, RequestedTTL: 365 * 24 * time.Hour,
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), g.ExpiresAt, time.Minute,
		"the org ceiling must win over both the request and the plugin default")
}

// The reviewer confirmed the fold arithmetic itself is right: a request
// shorter than every ceiling must win. This pins that down explicitly rather
// than only ever exercising the "ceiling is tightest" branch.
func TestCreateGrant_RequestedTTLShorterThanBothCapsWins(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithDefaultGrantTTL(90*24*time.Hour),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_shortreq")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, MaxGrantTTL: 24 * time.Hour,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"}, RequestedTTL: time.Hour,
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), g.ExpiresAt, time.Minute,
		"a request shorter than every ceiling must win")
}

// A zero requested TTL means "use the plugin default", when nothing shorter
// applies.
func TestCreateGrant_ZeroRequestedTTLUsesPluginDefault(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithDefaultGrantTTL(48*time.Hour),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_zeroreq")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"},
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(48*time.Hour), g.ExpiresAt, time.Minute)
}

// A non-positive plugin default (e.g. WithDefaultGrantTTL(0), a
// misconfiguration) must not produce a grant that is born expired. It should
// fall back to the package default instead of silently denying every future
// request through IsActive.
func TestCreateGrant_NonPositiveDefaultTTLFallsBackToPackageDefault(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithDefaultGrantTTL(0),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_deadttl")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"},
	})

	require.NoError(t, err)
	assert.True(t, g.ExpiresAt.After(time.Now().Add(time.Hour)),
		"a non-positive default TTL must not produce a grant that is born expired")
}

// A zero plugin default combined with a real org ceiling must still clamp to
// that ceiling, not silently expand past it. Flooring the *result* of the
// fold instead of the *base* let a zero default skip both fold comparisons
// (0 is not less than anything positive) and return the unclamped package
// default, bypassing the org ceiling entirely — the exact regression the
// previous fix round introduced while fixing the "born expired" bug.
func TestCreateGrant_ZeroDefaultStillClampsToOrgCeiling(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithDefaultGrantTTL(0),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_zerodefault_ceiling")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, MaxGrantTTL: time.Hour,
	}))

	// No RequestedTTL, so the only thing standing between a zero default and
	// the full (floored) package default is the org ceiling.
	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"},
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), g.ExpiresAt, time.Minute,
		"a zero plugin default must not bypass a real org ceiling")
}

// CreateGrant must derive the TTL ceiling from the agent's actual org, not
// merely from whatever OrgID the caller happens to pass — a zero caller OrgID
// must not bypass an org ceiling that genuinely applies to this agent.
func TestCreateGrant_UsesAgentOrgCeilingRegardlessOfRequestOrgID(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithDefaultGrantTTL(90*24*time.Hour),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_orgceiling")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, MaxGrantTTL: time.Hour,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID:  id.OrgID{}, // zero, as if the caller trusted nothing about org context
		Scopes: []string{"invoices:read"}, RequestedTTL: 365 * 24 * time.Hour,
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), g.ExpiresAt, time.Minute,
		"the agent's own org ceiling must apply even when the caller passes a zero org id")
}

// A grant created with a zero caller OrgID under an org-registered agent must
// still be stored under that agent's org, not org-less — otherwise the org's
// own offboarding sweep (RevokeGrantsByUserOrg) can never find it, and the
// grant survives its delegating user leaving the org. The second assertion
// below is the one that actually matters: it proves the property, not just
// the field.
func TestCreateGrant_StoresAgentOrgWhenCallerOrgIsZero(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_orphanorg")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))
	userID := id.NewUserID()

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: userID,
		OrgID:  id.OrgID{}, // caller passes no org context
		Scopes: []string{"invoices:read"},
	})
	require.NoError(t, err)
	assert.Equal(t, org.String(), g.OrgID.String(),
		"a grant under an org-registered agent must never be stored org-less")

	require.NoError(t, store.RevokeGrantsByUserOrg(context.Background(), userID, org))
	got, err := store.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.False(t, got.IsActive(time.Now()),
		"the org's own offboarding sweep must find and revoke a grant stored under it")
}

// An unmapped scope must never reach a stored grant. This must hold as a
// property of CreateGrant itself, not only when a caller happens to run
// Evaluate first — CreateGrant is exported with no enforced caller.
func TestCreateGrant_RejectsUnmappedScope(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_cgunmapped")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"totally:unmapped"},
	})

	require.Error(t, err, "an unmapped scope must never reach a stored grant")
	assert.Contains(t, err.Error(), "unknown delegation scope")
}

func TestCreateGrant_RejectsScopeOutsideOrgCeiling(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
		agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_cgceiling")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, AllowedScopes: []string{"invoices:read"},
	}))

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:write"},
	})

	require.Error(t, err, "a scope outside the org ceiling must never reach a stored grant")
	assert.Contains(t, err.Error(), "not permitted by organization policy")
}

func TestCreateGrant_RejectsWhenPolicyBlocked(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_cgblocked")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeBlocked,
	}))

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"},
	})

	require.Error(t, err, "a blocked org's policy must be enforced by CreateGrant too, not only by Evaluate")
	assert.Contains(t, err.Error(), "does not allow agent delegation")
}

func TestCreateGrant_RejectsBlockedAgent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	org := id.NewOrgID()
	agent := approvedAgent(t, store, org, "client_cgagentblocked")
	agent.Status = agentauth.StatusBlocked
	require.NoError(t, store.UpdateAgent(context.Background(), agent))

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(), OrgID: org,
	})

	require.Error(t, err, "a blocked agent must never receive a new grant")
	assert.Contains(t, err.Error(), "blocked")
}

func TestCreateGrant_RejectsUnknownAgent(t *testing.T) {
	p := agentauth.New()

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: id.NewUserID(), OrgID: id.NewOrgID(),
	})

	require.Error(t, err, "a grant cannot be created for an agent that doesn't exist")
}

func TestCreateGrant_RejectsZeroUser(t *testing.T) {
	p := agentauth.New()

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), OrgID: id.NewOrgID(),
		Scopes: []string{"invoices:read"},
	})

	require.Error(t, err, "a grant with no delegating human must be impossible to create")
	assert.Contains(t, err.Error(), "delegating user")
}

// id.UserID is a type alias for id.ID, so the compiler cannot distinguish it
// from an OrgID or AgentID at the call site. A grant's UserID must actually
// carry the user prefix, not merely be non-nil.
func TestCreateGrant_RejectsNonUserPrefixedUserID(t *testing.T) {
	p := agentauth.New()

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: id.NewOrgID(), OrgID: id.NewOrgID(),
		Scopes: []string{"invoices:read"},
	})

	require.Error(t, err, "a UserID that is actually an org id must be rejected, not merely checked for nil")
	assert.Contains(t, err.Error(), "delegating user")
}
