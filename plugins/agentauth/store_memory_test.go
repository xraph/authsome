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

func newGrant(t *testing.T, userID id.UserID, orgID id.OrgID, expires time.Time) *agentauth.AgentGrant {
	t.Helper()
	return &agentauth.AgentGrant{
		ID:        id.NewAgentGrantID(),
		AppID:     id.NewAppID(),
		AgentID:   id.NewAgentID(),
		UserID:    userID,
		OrgID:     orgID,
		Scopes:    []string{"invoices:read"},
		ExpiresAt: expires,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestMemoryStore_GetAgentGrant_RoundTrips(t *testing.T) {
	s := agentauth.NewMemoryStore()
	g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))

	require.NoError(t, s.CreateAgentGrant(context.Background(), g))

	got, err := s.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, g.UserID.String(), got.UserID.String())
	assert.Equal(t, []string{"invoices:read"}, got.Scopes)
}

func TestMemoryStore_GetAgentGrant_NotFound(t *testing.T) {
	s := agentauth.NewMemoryStore()

	_, err := s.GetAgentGrant(context.Background(), id.NewAgentGrantID())

	require.ErrorIs(t, err, agentauth.ErrNotFound)
}

// Offboarding revokes by user, so the store has to be able to find every
// grant a person issued regardless of which agent holds it.
func TestMemoryStore_RevokeGrantsByUser(t *testing.T) {
	s := agentauth.NewMemoryStore()
	victim, bystander := id.NewUserID(), id.NewUserID()
	org := id.NewOrgID()
	g1 := newGrant(t, victim, org, time.Now().Add(time.Hour))
	g2 := newGrant(t, victim, org, time.Now().Add(time.Hour))
	g3 := newGrant(t, bystander, org, time.Now().Add(time.Hour))
	for _, g := range []*agentauth.AgentGrant{g1, g2, g3} {
		require.NoError(t, s.CreateAgentGrant(context.Background(), g))
	}

	require.NoError(t, s.RevokeGrantsByUser(context.Background(), victim))

	for _, gid := range []id.AgentGrantID{g1.ID, g2.ID} {
		got, err := s.GetAgentGrant(context.Background(), gid)
		require.NoError(t, err)
		assert.NotNil(t, got.RevokedAt, "the victim's grants must be revoked")
	}
	survivor, err := s.GetAgentGrant(context.Background(), g3.ID)
	require.NoError(t, err)
	assert.Nil(t, survivor.RevokedAt, "another user's grant must survive")
}

// Removing someone from one org must not disarm their agents everywhere else.
func TestMemoryStore_RevokeGrantsByUserOrg_ScopedToThatOrg(t *testing.T) {
	s := agentauth.NewMemoryStore()
	user := id.NewUserID()
	leaving, staying := id.NewOrgID(), id.NewOrgID()
	gone := newGrant(t, user, leaving, time.Now().Add(time.Hour))
	kept := newGrant(t, user, staying, time.Now().Add(time.Hour))
	require.NoError(t, s.CreateAgentGrant(context.Background(), gone))
	require.NoError(t, s.CreateAgentGrant(context.Background(), kept))

	require.NoError(t, s.RevokeGrantsByUserOrg(context.Background(), user, leaving))

	g, err := s.GetAgentGrant(context.Background(), gone.ID)
	require.NoError(t, err)
	assert.NotNil(t, g.RevokedAt)
	k, err := s.GetAgentGrant(context.Background(), kept.ID)
	require.NoError(t, err)
	assert.Nil(t, k.RevokedAt, "grants in the user's other orgs must survive")
}

func TestAgentGrant_IsActive(t *testing.T) {
	now := time.Now()
	revoked := now.Add(-time.Minute)

	tests := []struct {
		name  string
		grant agentauth.AgentGrant
		want  bool
	}{
		{"live", agentauth.AgentGrant{ExpiresAt: now.Add(time.Hour)}, true},
		{"expired", agentauth.AgentGrant{ExpiresAt: now.Add(-time.Hour)}, false},
		{"revoked", agentauth.AgentGrant{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.grant.IsActive(now))
		})
	}
}

// Copy safety test: reading should return a deep copy that can't mutate stored state.
func TestMemoryStore_GetAgentGrant_CopySafety_ReadPath(t *testing.T) {
	s := agentauth.NewMemoryStore()
	g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))

	require.NoError(t, s.CreateAgentGrant(context.Background(), g))

	got1, err := s.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	require.Equal(t, "invoices:read", got1.Scopes[0])

	// Mutate the returned grant
	got1.Scopes[0] = "admin:*"

	// Read again and verify mutation did not affect stored state
	got2, err := s.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, "invoices:read", got2.Scopes[0], "stored scope must not be mutated")
}

// Copy safety test: writing should not share the caller's slice.
func TestMemoryStore_CreateAgentGrant_CopySafety_WritePath(t *testing.T) {
	s := agentauth.NewMemoryStore()
	g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))

	require.NoError(t, s.CreateAgentGrant(context.Background(), g))

	// Mutate the caller's grant after creating it
	g.Scopes[0] = "admin:*"

	// Read from store and verify mutation did not affect stored state
	got, err := s.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, "invoices:read", got.Scopes[0], "stored scope must not be affected by caller mutation")
}

// Copy safety test for OrgAgentPolicy.AllowedScopes.
func TestMemoryStore_OrgAgentPolicy_CopySafety(t *testing.T) {
	s := agentauth.NewMemoryStore()
	org := id.NewOrgID()
	p := &agentauth.OrgAgentPolicy{
		OrgID:         org,
		Mode:          agentauth.ModeAllowlist,
		MaxGrantTTL:   time.Hour,
		AllowedScopes: []string{"invoices:read", "users:read"},
	}

	require.NoError(t, s.PutOrgPolicy(context.Background(), p))

	got1, err := s.GetOrgPolicy(context.Background(), org)
	require.NoError(t, err)
	require.Equal(t, "invoices:read", got1.AllowedScopes[0])

	// Mutate the returned policy
	got1.AllowedScopes[0] = "admin:*"

	// Read again and verify mutation did not affect stored state
	got2, err := s.GetOrgPolicy(context.Background(), org)
	require.NoError(t, err)
	assert.Equal(t, "invoices:read", got2.AllowedScopes[0], "stored allowed_scopes must not be mutated")
}

// PutOrgPolicy must refuse a policy with a mode that isn't one of the three
// known constants. Evaluate and CreateGrant treat an unrecognized mode as a
// deny, but the safer invariant is that bad data can never be written in the
// first place — a partial update that only touches MaxGrantTTL and re-Puts
// the struct must not be able to carry a garbled Mode into the store.
func TestMemoryStore_PutOrgPolicy_RejectsUnrecognizedMode(t *testing.T) {
	s := agentauth.NewMemoryStore()

	err := s.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: id.NewOrgID(), Mode: agentauth.PolicyMode("bogus"),
	})

	require.Error(t, err, "a policy nobody can interpret must be impossible to store")
}

// The zero value of PolicyMode ("") must be refused for the same reason.
func TestMemoryStore_PutOrgPolicy_RejectsZeroValueMode(t *testing.T) {
	s := agentauth.NewMemoryStore()

	err := s.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{OrgID: id.NewOrgID()})

	require.Error(t, err, "an unset mode must be impossible to store")
}

// RevokeGrantsByAgent with specific org should only revoke grants in that org.
func TestMemoryStore_RevokeGrantsByAgent_WithOrg(t *testing.T) {
	s := agentauth.NewMemoryStore()
	agent := id.NewAgentID()
	org1 := id.NewOrgID()
	org2 := id.NewOrgID()
	user := id.NewUserID()

	g1 := newGrant(t, user, org1, time.Now().Add(time.Hour))
	g1.AgentID = agent
	g2 := newGrant(t, user, org2, time.Now().Add(time.Hour))
	g2.AgentID = agent

	require.NoError(t, s.CreateAgentGrant(context.Background(), g1))
	require.NoError(t, s.CreateAgentGrant(context.Background(), g2))

	// Revoke the agent's grants only in org1
	require.NoError(t, s.RevokeGrantsByAgent(context.Background(), agent, org1))

	// g1 should be revoked
	revoked, err := s.GetAgentGrant(context.Background(), g1.ID)
	require.NoError(t, err)
	assert.NotNil(t, revoked.RevokedAt, "grant in specified org must be revoked")

	// g2 should survive
	survivor, err := s.GetAgentGrant(context.Background(), g2.ID)
	require.NoError(t, err)
	assert.Nil(t, survivor.RevokedAt, "grant in different org must not be revoked")
}

// RevokeGrantsByAgent with zero-value org should revoke the agent's grants everywhere.
func TestMemoryStore_RevokeGrantsByAgent_AllOrgs(t *testing.T) {
	s := agentauth.NewMemoryStore()
	agent := id.NewAgentID()
	org1 := id.NewOrgID()
	org2 := id.NewOrgID()
	user := id.NewUserID()

	g1 := newGrant(t, user, org1, time.Now().Add(time.Hour))
	g1.AgentID = agent
	g2 := newGrant(t, user, org2, time.Now().Add(time.Hour))
	g2.AgentID = agent

	require.NoError(t, s.CreateAgentGrant(context.Background(), g1))
	require.NoError(t, s.CreateAgentGrant(context.Background(), g2))

	// Revoke the agent's grants across all orgs (zero-value orgID)
	require.NoError(t, s.RevokeGrantsByAgent(context.Background(), agent, id.OrgID{}))

	// Both grants should be revoked
	r1, err := s.GetAgentGrant(context.Background(), g1.ID)
	require.NoError(t, err)
	assert.NotNil(t, r1.RevokedAt, "grant in org1 must be revoked")

	r2, err := s.GetAgentGrant(context.Background(), g2.ID)
	require.NoError(t, err)
	assert.NotNil(t, r2.RevokedAt, "grant in org2 must be revoked")
}
