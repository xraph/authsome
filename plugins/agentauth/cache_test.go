package agentauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

func TestGrantCache_HitAndMiss(t *testing.T) {
	c := newGrantCache(time.Minute)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}

	_, ok := c.get(g.ID)
	assert.False(t, ok, "an empty cache must miss")

	c.put(g)
	got, ok := c.get(g.ID)
	require.True(t, ok)
	assert.Equal(t, g.ID.String(), got.ID.String())
}

func TestGrantCache_EntryExpires(t *testing.T) {
	c := newGrantCache(10 * time.Millisecond)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g)

	time.Sleep(20 * time.Millisecond)

	_, ok := c.get(g.ID)
	assert.False(t, ok, "a cache entry past its ttl must miss")
}

func TestGrantCache_Invalidate(t *testing.T) {
	c := newGrantCache(time.Minute)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g)

	c.invalidate(g.ID)

	_, ok := c.get(g.ID)
	assert.False(t, ok)
}

// Revocation must be visible immediately, not after the ttl. Session deletion
// is the primary invalidation point, but an explicit invalidate on revoke is
// what makes single-node behavior exact.
func TestAuthorize_RevocationBeatsTheCache(t *testing.T) {
	store := NewMemoryStore()
	p := New(WithStore(store), WithScope("invoices:read", Grants("read", "invoice")))
	userID := id.NewUserID()
	g := &AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(t.Context(), g))
	p.SetPermissionChecker(allowAll{})
	sess := agentSession(g)
	require.NoError(t, p.Authorize(t.Context(), sess, "read", "invoice")) // warms the cache

	require.NoError(t, p.RevokeGrant(t.Context(), g.ID))

	require.ErrorIs(t, p.Authorize(t.Context(), sess, "read", "invoice"), ErrGrantInactive)
}

// allowAll stands in for a permission checker that never refuses, so tests in
// this file that only care about the cache/grant path are not also
// exercising the user gate.
type allowAll struct{}

func (allowAll) HasPermission(_ context.Context, _ id.UserID, _, _ string) (bool, error) {
	return true, nil
}

// agentSession builds the session an agent grant would present at request
// time, bound to the grant by UserID, AgentID and AppID as Authorize requires.
func agentSession(g *AgentGrant) *session.Session {
	return &session.Session{
		ID:            id.NewSessionID(),
		AppID:         g.AppID,
		UserID:        g.UserID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       g.AgentID,
		GrantID:       g.ID,
	}
}
