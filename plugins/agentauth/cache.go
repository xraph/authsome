package agentauth

import (
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// grantCacheTTL bounds how stale a cached grant may be.
//
// Revocation does not rely on this. Revoking a grant deletes the sessions it
// issued, and session resolution runs on every request before the grant is
// ever consulted, so a stale entry is unreachable. What the ttl actually
// covers is the slower stuff: a grant ageing past ExpiresAt, or an org
// flipping its policy to blocked. Multi-node deployments inherit the same
// bounded staleness.
const grantCacheTTL = 10 * time.Second

type cachedGrant struct {
	grant    *AgentGrant
	cachedAt time.Time
}

type grantCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]cachedGrant
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
	cp := *e.grant
	return &cp, true
}

func (c *grantCache) put(g *AgentGrant) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := *g
	c.entries[g.ID.String()] = cachedGrant{grant: &cp, cachedAt: time.Now()}
}

func (c *grantCache) invalidate(grantID id.AgentGrantID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, grantID.String())
}
