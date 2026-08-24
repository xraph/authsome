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
