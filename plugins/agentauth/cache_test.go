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

	c.put(g, c.generation())
	got, ok := c.get(g.ID)
	require.True(t, ok)
	assert.Equal(t, g.ID.String(), got.ID.String())
}

func TestGrantCache_EntryExpires(t *testing.T) {
	c := newGrantCache(10 * time.Millisecond)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g, c.generation())

	time.Sleep(20 * time.Millisecond)

	_, ok := c.get(g.ID)
	assert.False(t, ok, "a cache entry past its ttl must miss")
}

func TestGrantCache_Invalidate(t *testing.T) {
	c := newGrantCache(time.Minute)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g, c.generation())

	c.invalidate(g.ID)

	_, ok := c.get(g.ID)
	assert.False(t, ok)
}

// get must hand back a copy whose Scopes does not alias the cached entry's
// backing array. AgentGrant.Scopes is a slice, so a shallow struct copy
// shares it: a caller that mutates the grant it got back from get would
// otherwise corrupt what every other reader sees for the rest of the ttl.
func TestGrantCache_GetDoesNotAliasScopes(t *testing.T) {
	c := newGrantCache(time.Minute)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour), Scopes: []string{"invoices:read"}}
	c.put(g, c.generation())

	got, ok := c.get(g.ID)
	require.True(t, ok)
	got.Scopes[0] = "admin:all"

	fresh, ok := c.get(g.ID)
	require.True(t, ok)
	assert.Equal(t, "invoices:read", fresh.Scopes[0], "mutating a grant returned by get must not affect a later get")
}

// put must copy g's Scopes on the way in, not alias the caller's slice.
// Otherwise a caller that mutates the AgentGrant it just handed to put (or
// the store's own return value, which is exactly what Authorize does with
// the grant it just loaded) corrupts the cached entry after the fact.
func TestGrantCache_PutDoesNotAliasScopes(t *testing.T) {
	c := newGrantCache(time.Minute)
	scopes := []string{"invoices:read"}
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour), Scopes: scopes}
	c.put(g, c.generation())

	scopes[0] = "admin:all" // mutate the caller's slice after put returns

	got, ok := c.get(g.ID)
	require.True(t, ok)
	assert.Equal(t, "invoices:read", got.Scopes[0], "mutating the caller's slice after put must not affect the cached entry")
}

// clear is not wired into Authorize or RevokeGrant by this task — it exists
// for Task 11's bulk-revoke sweeps, which have no single grant id to hand
// invalidate — but it must still drop every entry on its own.
func TestGrantCache_Clear(t *testing.T) {
	c := newGrantCache(time.Minute)
	g1 := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	g2 := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g1, c.generation())
	c.put(g2, c.generation())

	c.clear()

	_, ok := c.get(g1.ID)
	assert.False(t, ok)
	_, ok = c.get(g2.ID)
	assert.False(t, ok)
}

// clear must bump the generation exactly as invalidate does, so a put
// already in flight for a grant caught up in a bulk revoke is dropped too,
// not just the entries that existed at the moment clear ran.
func TestGrantCache_ClearBumpsGeneration(t *testing.T) {
	c := newGrantCache(time.Minute)
	gen := c.generation() // captured before clear, as a racing put would
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}

	c.clear()

	c.put(g, gen)
	_, ok := c.get(g.ID)
	assert.False(t, ok, "a put whose generation predates clear must be dropped")
}

// Revocation must be visible immediately, not after the ttl. An explicit
// invalidate on revoke is what makes single-node behavior exact; nothing in
// this repository deletes the sessions a revoked grant issued, so that is
// not a mechanism this cache can lean on (see the comment on grantCacheTTL).
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

// raceStore wraps a Store so its GetAgentGrant call can trigger a full
// RevokeGrant of the very grant it is about to return, deterministically
// reproducing the window Authorize leaves open between reading the store and
// writing the cache: the read is already in flight (and so still carries the
// pre-revocation grant) when the revoke lands, invalidates a cache entry
// that does not exist yet, and returns.
type raceStore struct {
	Store
	p       *Plugin
	grantID id.AgentGrantID
	raced   bool
}

func (s *raceStore) GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error) {
	g, err := s.Store.GetAgentGrant(ctx, grantID)
	if err != nil {
		return nil, err
	}
	if !s.raced && grantID.String() == s.grantID.String() {
		s.raced = true
		if err := s.p.RevokeGrant(ctx, grantID); err != nil {
			return nil, err
		}
	}
	// Return the snapshot this read already captured, before the revoke
	// above landed, exactly as a real store read already in flight would.
	return g, nil
}

// The core of C2: a RevokeGrant landing in the window between Authorize's
// store read and its cache write must never be resurrected by that
// request's own write. Without the generation check, the first Authorize
// call below would cache the pre-revocation grant on its way out, and the
// second call would then read that resurrected entry and wrongly succeed
// for the rest of the ttl.
func TestAuthorize_RevokeDuringStoreReadIsNotResurrected(t *testing.T) {
	store := NewMemoryStore()
	p := New(WithScope("invoices:read", Grants("read", "invoice")))
	userID := id.NewUserID()
	g := &AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(t.Context(), g))
	race := &raceStore{Store: store, grantID: g.ID}
	p.store = race // internal test, package agentauth: swap the store post-construction
	race.p = p
	p.SetPermissionChecker(allowAll{})
	sess := agentSession(g)

	// This call's own store read races the revoke race injects; either
	// outcome for this particular call is a pre-existing, unavoidable race
	// inherent to "read then decide" and not what this test pins down.
	_ = p.Authorize(t.Context(), sess, "read", "invoice")

	// What must hold is the next call: the store now unambiguously says
	// revoked, and nothing the first call did may have cached over that.
	require.ErrorIs(t, p.Authorize(t.Context(), sess, "read", "invoice"), ErrGrantInactive,
		"a revoke that lands mid-request must not be resurrected by that request's own cache write")
}

// countingStore wraps a Store and counts GetAgentGrant calls, so a test can
// pin down that Authorize actually reads through the cache on a hit instead
// of hitting the store on every call.
type countingStore struct {
	Store
	calls int
}

func (s *countingStore) GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error) {
	s.calls++
	return s.Store.GetAgentGrant(ctx, grantID)
}

// I2: nothing else in this package pins that Authorize reads through the
// cache at all — every other test still passes if the cache read is deleted
// and every call falls through to the store. This is the test that would
// catch that.
func TestAuthorize_ReadsGrantFromStoreOnlyOnce(t *testing.T) {
	mem := NewMemoryStore()
	counting := &countingStore{Store: mem}
	p := New(WithStore(counting), WithScope("invoices:read", Grants("read", "invoice")))
	userID := id.NewUserID()
	g := &AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, mem.CreateAgentGrant(t.Context(), g))
	p.SetPermissionChecker(allowAll{})
	sess := agentSession(g)

	require.NoError(t, p.Authorize(t.Context(), sess, "read", "invoice"))
	require.NoError(t, p.Authorize(t.Context(), sess, "read", "invoice"))

	assert.Equal(t, 1, counting.calls, "the second Authorize call must be served from the cache, not the store")
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
