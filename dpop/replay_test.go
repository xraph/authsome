package dpop_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/dpop"
)

func TestMemoryReplayCache_FirstSightThenReplay(t *testing.T) {
	c := dpop.NewMemoryReplayCache(128)
	ctx := context.Background()

	seen, err := c.Seen(ctx, "jkt:jti-1", time.Minute)
	require.NoError(t, err)
	assert.False(t, seen, "first sight must not report a replay")

	seen, err = c.Seen(ctx, "jkt:jti-1", time.Minute)
	require.NoError(t, err)
	assert.True(t, seen, "second sight must report a replay")
}

func TestMemoryReplayCache_ExpiredEntryIsNotAReplay(t *testing.T) {
	c := dpop.NewMemoryReplayCache(128)
	ctx := context.Background()

	_, err := c.Seen(ctx, "jkt:jti-1", time.Nanosecond)
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)

	seen, err := c.Seen(ctx, "jkt:jti-1", time.Minute)
	require.NoError(t, err)
	assert.False(t, seen, "an entry past its ttl must not block a fresh proof")
}

// TestMemoryReplayCache_BoundedAndCounted proves the cache cannot grow without
// limit and that overflow is counted. A cache that silently drops entries
// degrades to no replay protection while still reporting success, which is the
// worst of both outcomes.
func TestMemoryReplayCache_BoundedAndCounted(t *testing.T) {
	c := dpop.NewMemoryReplayCache(16)
	ctx := context.Background()

	for i := range 64 {
		_, err := c.Seen(ctx, fmt.Sprintf("jkt:jti-%d", i), time.Minute)
		require.NoError(t, err)
	}
	assert.Positive(t, c.Evictions(), "overflow must be counted, not hidden")

	// Independently derived expectation, not read back from the cache under
	// test: 64 unique keys into a capacity-16 cache with no expiry in play
	// evicts exactly one oldest entry per insert past capacity, so the count
	// must be exactly 64-16=48. A cache that evicted the wrong number, or
	// evicted without actually bounding, or counted without evicting, would
	// pass "Positive" but fail this.
	assert.Equal(t, uint64(48), c.Evictions(), "eviction count must match capacity math exactly, not just be nonzero")
}

// TestMemoryReplayCache_ConcurrentSeenIsAtomic exercises the property the
// brief calls out explicitly: two concurrent calls with the same key must
// not both report "not seen". A lock acquired around only part of the
// check-then-record sequence would let this race window open under -race,
// even though no individual field access is unsynchronized. Sequential
// tests alone never hit that window, so this is here to make the race
// detector run actually exercise the concurrent path.
func TestMemoryReplayCache_ConcurrentSeenIsAtomic(t *testing.T) {
	c := dpop.NewMemoryReplayCache(128)
	ctx := context.Background()

	const callers = 50
	results := make([]bool, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := range callers {
		go func(i int) {
			defer wg.Done()
			seen, err := c.Seen(ctx, "jkt:shared-jti", time.Minute)
			require.NoError(t, err)
			results[i] = seen
		}(i)
	}
	wg.Wait()

	notSeen := 0
	for _, seen := range results {
		if !seen {
			notSeen++
		}
	}
	assert.Equal(t, 1, notSeen, "exactly one concurrent caller must be the first to see the key")
}

func TestCeremonyReplayCache(t *testing.T) {
	c := dpop.NewCeremonyReplayCache(ceremony.NewMemory())
	ctx := context.Background()

	seen, err := c.Seen(ctx, "jkt:jti-1", time.Minute)
	require.NoError(t, err)
	assert.False(t, seen)

	seen, err = c.Seen(ctx, "jkt:jti-1", time.Minute)
	require.NoError(t, err)
	assert.True(t, seen)
}
