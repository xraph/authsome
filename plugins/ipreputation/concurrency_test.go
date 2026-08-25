package ipreputation

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/require"
)

// The reputation cache is shared mutable state on the sign-in path: check
// reads it, setCache writes it, and both run concurrently under any real
// load. Nothing in this package exercised that before, because no test here
// started a goroutine, so `go test -race` had nothing to observe and passed
// for free.
//
// The property worth protecting is a security one. A blocked IP must stay
// blocked for as long as its entry is cached. An uncloned cache breaks that
// in a way no sequential test notices: getCached would hand every caller the
// same *IPReputation, so one of them writing to Score or IsBlacklisted would
// flip the verdict for every later request, and that write would race every
// concurrent read.

const (
	repHammerWorkers = 4
	repHammerBudget  = 150 * time.Millisecond
)

// concurrentProvider is a Provider safe to call from many goroutines. The
// package's existing mockProvider increments a plain int field, which is
// itself a race under concurrent use, so these tests need their own.
//
// It returns the same pointer for a given IP on every call, exactly as a
// provider backed by its own in-memory table would. That is what makes
// setCache's copy load-bearing: without it the cache would hold the
// provider's own struct.
type concurrentProvider struct {
	results map[string]*IPReputation
	calls   atomic.Int64
}

func (c *concurrentProvider) CheckIP(_ context.Context, ip string) (*IPReputation, error) {
	c.calls.Add(1)
	if rep, ok := c.results[ip]; ok {
		return rep, nil
	}
	return &IPReputation{IP: ip, Score: 0}, nil
}

func newConcurrentTestPlugin(p Provider) *Plugin {
	pl := New(Config{
		Provider:       p,
		BlockThreshold: 80,
		WarnThreshold:  50,
		CacheTTL:       time.Hour,
		BlockMessage:   "blocked",
	})
	pl.logger = log.NewNoopLogger()
	return pl
}

// TestCheck_ConcurrentWithCaching runs the real check path from concurrent
// goroutines across a mix of blocked, warned and clean IPs, so cache reads,
// cache writes and provider calls all overlap.
//
// The assertion is the security invariant rather than "did not crash": a
// blocked IP must be refused on every single call. If the cache aliased its
// entries, a concurrent writer could turn that into "usually refused".
func TestCheck_ConcurrentWithCaching(t *testing.T) {
	blocked := map[string]bool{}
	results := map[string]*IPReputation{}
	for i := 0; i < 8; i++ {
		ip := fmt.Sprintf("203.0.113.%d", i)
		score := 10
		if i%2 == 0 {
			score = 95 // over BlockThreshold
			blocked[ip] = true
		}
		results[ip] = &IPReputation{
			IP: ip, Score: score, Source: "test",
			Categories: []string{"scanner", "botnet"},
		}
	}

	provider := &concurrentProvider{results: results}
	p := newConcurrentTestPlugin(provider)

	ips := make([]string, 0, len(results))
	for ip := range results {
		ips = append(ips, ip)
	}

	deadline := time.Now().Add(repHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < repHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			ctx := context.Background()
			for i := 0; time.Now().Before(deadline); i++ {
				ip := ips[(w*len(ips)+i)%len(ips)]
				err := p.check(ctx, ip, "app-1")
				if blocked[ip] && err == nil {
					t.Errorf("blocked IP %s was allowed through", ip)
					return
				}
				if !blocked[ip] && err != nil {
					t.Errorf("clean IP %s was refused: %v", ip, err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}

	// Concurrent readers that write to what they got back. This is the shape
	// that makes an uncloned getCached a data race rather than a stale read.
	for w := 0; w < repHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				if rep := p.getCached(ips[(w+i)%len(ips)]); rep != nil {
					rep.Score = 0
					rep.IsBlacklisted = false
					if len(rep.Categories) > 0 {
						rep.Categories[0] = "clean"
					}
				}
				ops.Add(1)
			}
		}(w)
	}

	wg.Wait()

	require.Greater(t, ops.Load(), int64(1000),
		"the hammer did no meaningful work; it is not exercising the cache concurrently")
}

// TestGetCached_DoesNotAliasStoredEntry pins the read side. Without the copy
// in getCached, zeroing Score on the returned value downgrades the cached
// verdict for every later request until the ttl elapses.
func TestGetCached_DoesNotAliasStoredEntry(t *testing.T) {
	p := newConcurrentTestPlugin(&concurrentProvider{})
	p.setCache("1.2.3.4", &IPReputation{
		IP: "1.2.3.4", Score: 95, IsBlacklisted: true,
		Categories: []string{"scanner"},
	})

	got := p.getCached("1.2.3.4")
	require.NotNil(t, got)
	got.Score = 0
	got.IsBlacklisted = false
	got.Categories[0] = "clean"

	fresh := p.getCached("1.2.3.4")
	require.NotNil(t, fresh)
	require.Equal(t, 95, fresh.Score, "mutating a reputation returned by getCached must not downgrade the cached verdict")
	require.True(t, fresh.IsBlacklisted)
	require.Equal(t, "scanner", fresh.Categories[0], "Categories is a slice; a shallow copy would still alias it")
}

// TestSetCache_DoesNotAliasCallerValue pins the write side. rep arrives
// straight from Provider.CheckIP, and a provider that hands back a struct it
// still owns would otherwise be writing into this cache after the fact.
func TestSetCache_DoesNotAliasCallerValue(t *testing.T) {
	p := newConcurrentTestPlugin(&concurrentProvider{})
	rep := &IPReputation{
		IP: "1.2.3.4", Score: 95, IsBlacklisted: true,
		Categories: []string{"scanner"},
	}
	p.setCache("1.2.3.4", rep)

	rep.Score = 0 // the provider mutates its own struct after setCache returned
	rep.IsBlacklisted = false
	rep.Categories[0] = "clean"

	got := p.getCached("1.2.3.4")
	require.NotNil(t, got)
	require.Equal(t, 95, got.Score, "mutating the caller's value after setCache must not affect the cached entry")
	require.True(t, got.IsBlacklisted)
	require.Equal(t, "scanner", got.Categories[0])
}

// TestCheck_BlockedIPStaysBlockedUnderLoad is the invariant above stated at
// its narrowest: one IP, blocked, hammered from every goroutine at once, with
// other goroutines actively trying to downgrade the cached entry through the
// value getCached returns them. Not one call may be allowed through.
func TestCheck_BlockedIPStaysBlockedUnderLoad(t *testing.T) {
	const ip = "198.51.100.7"
	provider := &concurrentProvider{results: map[string]*IPReputation{
		ip: {IP: ip, Score: 99, IsBlacklisted: true, Categories: []string{"botnet"}},
	}}
	p := newConcurrentTestPlugin(provider)

	deadline := time.Now().Add(repHammerBudget)
	var wg sync.WaitGroup
	var allowed atomic.Int64
	var checks atomic.Int64

	for w := 0; w < repHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for time.Now().Before(deadline) {
				if err := p.check(ctx, ip, "app-1"); err == nil {
					allowed.Add(1)
				}
				checks.Add(1)
			}
		}()
	}

	for w := 0; w < repHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				if rep := p.getCached(ip); rep != nil {
					rep.Score = 0
					rep.IsBlacklisted = false
				}
			}
		}()
	}

	wg.Wait()

	require.Greater(t, checks.Load(), int64(500), "the hammer did no meaningful work")
	require.Zero(t, allowed.Load(), "a blacklisted IP was allowed through %d times; the cached verdict was mutated out from under check", allowed.Load())
}
