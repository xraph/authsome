package dpop

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// ReplayCache records proof jti values so a captured proof cannot be replayed
// inside its freshness window.
//
// Seen atomically records the key and reports whether it was already present.
// Implementations must be safe for concurrent use, and Seen sits on the hot
// path of every authenticated request, so it needs to be cheap.
type ReplayCache interface {
	Seen(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

// MemoryReplayCache is a bounded in-process cache with TTL expiry and
// oldest-first eviction.
//
// In-process means per-instance. Behind a load balancer a proof replayed
// against a different instance is not caught here, which is why the nonce
// exists and why NewCeremonyReplayCache is offered for deployments that run a
// distributed ceremony store.
type MemoryReplayCache struct {
	mu      sync.Mutex
	max     int
	entries map[string]*list.Element
	order   *list.List // front is oldest

	evictions atomic.Uint64
}

type replayEntry struct {
	key       string
	expiresAt time.Time
}

var _ ReplayCache = (*MemoryReplayCache)(nil)

// NewMemoryReplayCache creates a cache holding at most maxEntries.
func NewMemoryReplayCache(maxEntries int) *MemoryReplayCache {
	if maxEntries <= 0 {
		maxEntries = 100_000
	}
	return &MemoryReplayCache{
		max:     maxEntries,
		entries: make(map[string]*list.Element, maxEntries),
		order:   list.New(),
	}
}

// Evictions returns how many entries have been dropped for capacity. A
// non-zero and climbing value means proofs are being accepted that should have
// been checked, so surface it rather than leaving it to be discovered later.
func (c *MemoryReplayCache) Evictions() uint64 { return c.evictions.Load() }

func (c *MemoryReplayCache) Seen(_ context.Context, key string, ttl time.Duration) (bool, error) {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Drop anything expired at the front before deciding. Cheap because the
	// list is ordered by insertion and every entry shares a similar ttl.
	for e := c.order.Front(); e != nil; e = c.order.Front() {
		entry, ok := e.Value.(*replayEntry)
		if !ok || entry.expiresAt.After(now) {
			break
		}
		c.order.Remove(e)
		delete(c.entries, entry.key)
	}

	if existing, found := c.entries[key]; found {
		entry, ok := existing.Value.(*replayEntry)
		if ok && entry.expiresAt.After(now) {
			return true, nil
		}
		c.order.Remove(existing)
		delete(c.entries, key)
	}

	for len(c.entries) >= c.max {
		oldest := c.order.Front()
		if oldest == nil {
			break
		}
		entry, ok := oldest.Value.(*replayEntry)
		c.order.Remove(oldest)
		if ok {
			delete(c.entries, entry.key)
		}
		c.evictions.Add(1)
	}

	c.entries[key] = c.order.PushBack(&replayEntry{key: key, expiresAt: now.Add(ttl)})
	return false, nil
}
