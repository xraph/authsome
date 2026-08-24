package agentauth

import (
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// grantCacheTTL bounds how stale a cached grant may be.
//
// The cache is safe against revocation only because Plugin.RevokeGrant
// invalidates the entry explicitly, and put's generation check (see put)
// closes the race where a revoke lands mid-request. Nothing in this
// repository deletes the sessions a grant issued when it is revoked, so
// that is not a backstop this cache can lean on.
//
// The ttl exists for the two paths that do not go through RevokeGrant: a
// revocation performed some other way (a direct store write, a future
// admin path that bypasses this plugin), and a scope narrowed via
// UpdateAgentGrant, which changes what a grant may do without revoking it
// at all and so triggers no invalidation anywhere. The second is the
// dangerous one — it is silent, and until the ttl elapses a cached entry
// keeps authorizing against the wider scope it had when it was read. A
// grant simply ageing past ExpiresAt is not something the ttl needs to
// cover: IsActive runs against a fresh clock over an immutable ExpiresAt on
// every call, cached or not, so that check never goes stale. Authorize also
// never consults org policy, so an org flipping its policy mode is not
// masked by this cache either — only Evaluate and CreateGrant look at
// policy, at consent and grant-creation time, not at authorization time.
const grantCacheTTL = 10 * time.Second

type cachedGrant struct {
	grant    *AgentGrant
	cachedAt time.Time
}

// grantCache caches AgentGrant reads keyed by grant id. generation is a
// counter bumped by invalidate and clear; Authorize captures it with
// generation() before doing the store read that might race a concurrent
// revoke, then passes it to put, so put can tell whether an invalidation
// happened while the read was in flight.
type grantCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]cachedGrant
	gen     uint64
}

func newGrantCache(ttl time.Duration) *grantCache {
	return &grantCache{ttl: ttl, entries: make(map[string]cachedGrant)}
}

func (c *grantCache) get(grantID id.AgentGrantID) (*AgentGrant, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[grantID.String()]
	if !ok || time.Since(e.cachedAt) > c.ttl {
		return nil, false
	}
	return cloneGrant(e.grant), true
}

// generation reports the cache's current generation. A caller about to do
// work whose result it will hand to put — a store read that takes real time
// and so can race a concurrent revoke — captures this first, so put can
// refuse the write if invalidate or clear ran in between.
func (c *grantCache) generation() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gen
}

// put writes g into the cache, unless gen is stale, i.e. invalidate or
// clear bumped the generation after gen was captured.
//
// This is what stops a revoke that lands mid-request from being
// resurrected. Authorize reads the store and only then writes the cache; a
// RevokeGrant that runs in that exact window invalidates an entry that does
// not exist yet, and without this check the request's own write — carrying
// the pre-revocation grant — would land right after and undo the revoke for
// the rest of the ttl. Comparing generations closes that window: a write
// whose generation was captured before the bump can never land after it.
func (c *grantCache) put(g *AgentGrant, gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if gen != c.gen {
		return
	}
	c.entries[g.ID.String()] = cachedGrant{grant: cloneGrant(g), cachedAt: time.Now()}
}

func (c *grantCache) invalidate(grantID id.AgentGrantID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, grantID.String())
	c.gen++
}

// clear drops every cached entry and bumps the generation, so any put
// already in flight for a grant caught up in a bulk revoke is dropped too,
// the same way invalidate protects a single-grant revoke.
//
// This exists for Task 11's offboarding sweeps: RevokeGrantsByUser,
// RevokeGrantsByUserOrg and RevokeGrantsByAgent each revoke many grants by a
// query rather than by id, so there is no single grant id to hand
// invalidate. This task does not call clear from anywhere; wiring it into
// those bulk paths is Task 11's job.
func (c *grantCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cachedGrant)
	c.gen++
}

// cloneGrant deep-copies g, including Scopes. AgentGrant.Scopes is a slice,
// so a shallow `cp := *g` shares the backing array both ways: a cache read
// could hand out a slice the caller then mutates, corrupting the cached
// entry for the rest of its ttl, and a cache write could alias the caller's
// own slice, corrupting the cache if the caller mutates it after put
// returns. store_memory.go deep-copies Scopes for the identical reason on
// every one of its grant paths (Task 2); the cache needs it more, since one
// cached entry backs every authorization decision for that grant for the
// whole ttl, not just one store call.
func cloneGrant(g *AgentGrant) *AgentGrant {
	cp := *g
	cp.Scopes = append([]string(nil), g.Scopes...)
	return &cp
}
