package agentauth

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// This file exists because `go test ./plugins/agentauth/ -race` was, until it
// was added, a vacuous green. The race detector is dynamic: it reports two
// accesses to the same address from different goroutines only when it
// actually observes them execute without a happens-before edge between them.
// It has no static model of the code. Every other test file in this package
// is strictly sequential (no goroutine is ever started), so the detector had
// nothing to observe and `-race` proved exactly as much as running without
// it.
//
// The package has real concurrent machinery behind that green: grantCache's
// RWMutex and generation counter (cache.go), and MemoryStore's RWMutex, whose
// bulk-revocation paths iterate the grant map under a write lock
// (store_memory.go). The tests below put load on both.
//
// A note on what -race alone can and cannot catch here, because it shapes
// TestGrantCache_RevocationSurvivesConcurrentPuts below. The generation
// protocol's correctness depends on put's comparison happening inside the
// write-lock critical section, atomically with the map write. There are two
// distinct ways to break that, and only one of them is a data race:
//
//   - Hoisting `gen != c.gen` above c.mu.Lock() reads c.gen unsynchronized
//     while invalidate and clear write it under the lock. That is a data
//     race, and the hammer below gives the detector the concurrent accesses
//     it needs to see it.
//   - Rewriting the check as `gen != c.generation()` before taking the write
//     lock takes RLock for the read and releases it again. That is perfectly
//     race-clean and still completely broken: invalidate can land in the gap
//     between the check and the write, and the stale grant lands after it.
//     TSan will never report this.
//
// So the second test asserts the security property itself, that a revocation
// once returned is never resurrected, rather than relying on the detector to
// notice. -race catches the first mutation. The invariant catches both.

// Sizing. These tests are meant to run in the ordinary suite on every `go
// test`, not behind a build tag that nobody remembers to pass, so they are
// bounded by a short wall-clock budget and a modest goroutine count rather
// than by a big iteration count.
//
// Measured cost under -race: about 0.35s for the file, which is noise against
// this package's ~46s. The exception is GOMAXPROCS=1, where the per-round
// goroutine churn in TestGrantCache_RevocationSurvivesConcurrentPuts has no
// second processor to overlap with and the file costs about 4s. That is a
// degenerate configuration for a Go test run, since the toolchain defaults
// to NumCPU and CI runners have more than one, so it is not worth trading
// detection strength to optimize. Note there is deliberately no -short knob
// here: see resurrectionRounds.
const (
	// hammerWorkers is per role, not in total: each test starts this many
	// readers, this many writers, and so on.
	hammerWorkers = 4
	// hammerBudget bounds the duration-driven hammers.
	hammerBudget = 150 * time.Millisecond
	// resurrectionRounds is how many separate revocations
	// TestGrantCache_RevocationSurvivesConcurrentPuts stages. Each round is
	// one chance to land an invalidate inside put's critical section, and
	// that window is a few tens of nanoseconds wide, so the count buys
	// detection probability. It is not a timing budget: rounds are gated on
	// observed put attempts, not on sleeps.
	//
	// 128 is sized off measurement, not guessed. Over repeated runs with put's
	// comparison moved out of the critical section (either way of doing it,
	// see the file comment), the worst round the resurrection was first
	// observed on was 13, across both GOMAXPROCS=1 and GOMAXPROCS=8. 128
	// leaves roughly a factor of ten of headroom over that.
	//
	// This is why the test has no `testing.Short()` path. Scaling rounds down
	// is the only thing that would meaningfully speed the test up, and any
	// divisor big enough to matter takes the budget to within a whisker of
	// that worst observed case. A -short run would still report PASS while
	// having become close to a coin flip. A knob that quietly converts a
	// detector into a coin flip is worse than no knob, and the test is fast
	// enough on any multi-core machine not to need one.
	resurrectionRounds = 128
	// putsBeforeRevoke is how many puts must have landed in a round before
	// the revoker fires, so the invalidate is guaranteed to arrive while
	// putters are actively in flight rather than still starting up. Spread
	// across hammerWorkers putters, this is a couple of dozen iterations
	// each: enough to be unambiguously in steady state, and no higher, since
	// every round pays it.
	putsBeforeRevoke = 100
)

// TestGrantCache_ConcurrentAccessIsRaceFree drives every grantCache method
// from concurrent goroutines at once, which is the load the package's
// production callers actually produce and which no other test in this package
// generates: Authorize runs get/generation/put per request (middleware.go),
// RevokeGrant runs invalidate (grant.go), and the bulk offboarding sweeps run
// clear (lifecycle.go, handlers.go). Under -race this is what gives the
// detector concurrent accesses to observe against c.entries and c.gen.
//
// It asserts no invariant of its own beyond "does not race and does not
// deadlock". The behavioural invariant is
// TestGrantCache_RevocationSurvivesConcurrentPuts' job.
func TestGrantCache_ConcurrentAccessIsRaceFree(t *testing.T) {
	c := newGrantCache(time.Minute)

	// A small fixed key space, deliberately smaller than the worker count, so
	// workers collide on the same map entries instead of each quietly
	// operating on its own key and never overlapping.
	grants := make([]*AgentGrant, 8)
	for i := range grants {
		grants[i] = &AgentGrant{
			ID:        id.NewAgentGrantID(),
			ExpiresAt: time.Now().Add(time.Hour),
			Scopes:    []string{"invoices:read"},
		}
	}

	deadline := time.Now().Add(hammerBudget)
	var wg sync.WaitGroup
	// ops guards against a vacuous pass. A hammer whose goroutines all exit
	// immediately (a deadline computed wrong, a loop condition inverted by a
	// later edit) races nothing and would otherwise report green while
	// testing precisely as much as the sequential suite it was added to fix.
	var ops atomic.Int64

	spawn := func(fn func(i int)) {
		for w := 0; w < hammerWorkers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; time.Now().Before(deadline); i++ {
					fn(w*len(grants) + i)
					ops.Add(1)
				}
			}(w)
		}
	}

	// Readers: the cache-hit path.
	spawn(func(i int) {
		c.get(grants[i%len(grants)].ID)
	})

	// Writers: the cache-miss path, in Authorize's exact order. Capture the
	// generation first, then do the work that can race a revoke, then put.
	// Capturing after the read instead would defeat the protocol, so the
	// ordering here is part of what the test is modelling.
	spawn(func(i int) {
		g := grants[i%len(grants)]
		gen := c.generation()
		runtime.Gosched() // stand in for the store read Authorize does here
		c.put(g, gen)
	})

	// Single-grant revocations.
	spawn(func(i int) {
		c.invalidate(grants[i%len(grants)].ID)
	})

	// Bulk offboarding sweeps. Rarer than the rest, as in production, but
	// clear takes the same write lock and bumps the same counter, so it has
	// to be in the mix rather than assumed equivalent to invalidate.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; time.Now().Before(deadline); i++ {
			if i%64 == 0 {
				c.clear()
			}
			ops.Add(1)
			runtime.Gosched()
		}
	}()

	wg.Wait()

	// A floor low enough never to flake on a loaded CI box, high enough that
	// "the goroutines never actually ran" cannot slip past.
	require.Greater(t, ops.Load(), int64(1000), "the hammer did no meaningful work; it is not exercising the cache concurrently")
}

// TestGrantCache_RevocationSurvivesConcurrentPuts pins down the generation
// protocol's actual security property under real concurrency: once a
// revocation has landed and invalidate has returned, no put still in flight
// may resurrect the grant.
//
// TestAuthorize_RevokeDuringStoreReadIsNotResurrected in cache_test.go
// already covers this deterministically, by injecting a revoke into the store
// read from a single thread. That test is exact but it proves the check
// happens, not that it happens under the lock. It passes just as green if
// the comparison is hoisted out of the critical section, because with one
// thread there is no window for anything to land in. This one exercises the
// lock scope itself.
//
// Each round models one request racing one revocation, in the ordering the
// real code produces:
//
//	putter:  gen := generation()   ->  read store  ->  put(grant, gen)
//	revoker:                           revoke store ->  invalidate(id)
//
// A putter only calls put if it observed the grant as live, so its generation
// capture necessarily happened before the revoker's store write, and
// therefore before the invalidate that follows it. A correct put, comparing
// generations inside the write lock atomically with the map write, can
// never let such a write land after the bump. So after the revoker returns,
// the entry must be gone and must stay gone, with no flakiness in either
// direction.
func TestGrantCache_RevocationSurvivesConcurrentPuts(t *testing.T) {
	rounds, warmup := resurrectionRounds, putsBeforeRevoke
	// A ttl far longer than the test runs. A short one would let entries
	// lapse on their own and mask a resurrection as a cache miss.
	c := newGrantCache(time.Minute)

	for round := 0; round < rounds; round++ {
		g := &AgentGrant{
			ID:        id.NewAgentGrantID(),
			ExpiresAt: time.Now().Add(time.Hour),
			Scopes:    []string{"invoices:read"},
		}

		// revoked stands in for the store: the revoker sets it before
		// invalidating, exactly as RevokeGrant writes the store before
		// calling p.cache.invalidate.
		var revoked atomic.Bool
		var landed atomic.Int64

		var wg sync.WaitGroup
		for w := 0; w < hammerWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					// Captured before the store read, as Authorize does.
					gen := c.generation()
					if revoked.Load() {
						// The store now says revoked. A real Authorize
						// returns ErrGrantInactive here and caches nothing,
						// so this putter is done.
						return
					}
					c.put(g, gen)
					landed.Add(1)
				}
			}()
		}

		// Wait until puts are demonstrably flowing before revoking, so the
		// invalidate lands in traffic rather than in a lull.
		for landed.Load() < int64(warmup) {
			runtime.Gosched()
		}

		// Guard against a vacuous pass. If put stopped writing entries at all
		// the assertion below would hold for the wrong reason and this test
		// would silently stop testing anything.
		_, ok := c.get(g.ID)
		require.True(t, ok, "round %d: puts are not reaching the cache; the test is not exercising put", round)

		revoked.Store(true)
		c.invalidate(g.ID)

		wg.Wait()

		if _, ok := c.get(g.ID); ok {
			t.Fatalf("round %d: a revoked grant was resurrected by a put whose generation "+
				"predated the invalidate; put's generation check must happen inside its write lock", round)
		}
	}
}

// TestAuthorize_ConcurrentWithRevocation runs the whole authorization path
// against MemoryStore under concurrent revocation: many Authorize calls
// racing single-grant revokes and a bulk org sweep.
//
// Two things are under test that the cache hammers do not reach. MemoryStore
// itself, whose bulk paths (RevokeGrantsByOrg here) iterate the grant map
// under a write lock while readers hold the read lock. Nothing in this
// package exercised those maps from more than one goroutine before. And the
// composition: Authorize's cache read, store read and cache write are three
// separate steps, and the guarantee has to survive a revocation landing
// between any two of them.
//
// The invariant is the one that matters to a caller: if RevokeGrant (or the
// org sweep) had already returned before an Authorize call began, that call
// must deny. There is no assertion on calls that overlap a revocation.
// Either answer is legitimate for those, and demanding one would be a flaky
// test of an unspecified behaviour.
func TestAuthorize_ConcurrentWithRevocation(t *testing.T) {
	const grantsPerOrg = 8

	store := NewMemoryStore()
	p := New(WithStore(store), WithScope("invoices:read", Grants("read", "invoice")))
	p.SetPermissionChecker(allowAll{})

	// Two orgs: one revoked grant by grant through RevokeGrant, one swept
	// wholesale through OnAfterOrgDelete. The bulk path is not a nicety here.
	// It is the one that invalidates by clear rather than by id, and it is
	// the only path that iterates the store's map under a write lock.
	singleOrg, bulkOrg := id.NewOrgID(), id.NewOrgID()

	type tracked struct {
		grant *AgentGrant
		// revoked is set only after the revocation call has fully returned,
		// which is what makes "was already revoked before this call started"
		// a sound thing for an authorizer to assert on.
		revoked *atomic.Bool
	}

	ctx := t.Context()
	var all []tracked
	for _, orgID := range []id.OrgID{singleOrg, bulkOrg} {
		for i := 0; i < grantsPerOrg; i++ {
			g := &AgentGrant{
				ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
				UserID: id.NewUserID(), OrgID: orgID, Scopes: []string{"invoices:read"},
				ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			require.NoError(t, store.CreateAgentGrant(ctx, g))
			all = append(all, tracked{grant: g, revoked: new(atomic.Bool)})
		}
	}

	var wg sync.WaitGroup
	deadline := time.Now().Add(hammerBudget)

	// Authorizers.
	for w := 0; w < hammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			// Each worker uses its own context: t.Context is safe to share,
			// but a per-goroutine background context keeps this independent
			// of any cancellation the test framework does at teardown.
			c := context.Background()
			for i := 0; time.Now().Before(deadline); i++ {
				tr := all[(w*grantsPerOrg+i)%len(all)]
				// Read the flag first. If it is already set, the revocation
				// returned before this call began, so the answer is pinned.
				wasRevoked := tr.revoked.Load()
				err := p.Authorize(c, agentSession(tr.grant), "read", "invoice")
				if wasRevoked && err == nil {
					// t.Errorf, not t.Fatalf: Fatalf may only be called from
					// the goroutine running the test.
					t.Errorf("grant %s authorized after its revocation had already returned", tr.grant.ID)
					return
				}
			}
		}(w)
	}

	// Single-grant revocations, staggered across the budget so they land in
	// amongst live authorization traffic rather than all at the start.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, tr := range all {
			if tr.grant.OrgID.String() != singleOrg.String() {
				continue
			}
			if err := p.RevokeGrant(context.Background(), tr.grant.ID); err != nil {
				t.Errorf("RevokeGrant(%s): %v", tr.grant.ID, err)
				return
			}
			tr.revoked.Store(true)
			runtime.Gosched()
		}
	}()

	// The bulk sweep: RevokeGrantsByOrg under the store's write lock, then
	// cache.clear.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := p.OnAfterOrgDelete(context.Background(), bulkOrg); err != nil {
			t.Errorf("OnAfterOrgDelete: %v", err)
			return
		}
		for _, tr := range all {
			if tr.grant.OrgID.String() == bulkOrg.String() {
				tr.revoked.Store(true)
			}
		}
	}()

	// Concurrent readers of the store's other locked paths, so the bulk
	// write-locked iteration above has read-lock traffic to contend with.
	wg.Add(1)
	go func() {
		defer wg.Done()
		c := context.Background()
		for i := 0; time.Now().Before(deadline); i++ {
			tr := all[i%len(all)]
			_, _ = store.ListGrantsByUser(c, tr.grant.UserID)
			_, _ = store.GetActiveGrant(c, tr.grant.AgentID, tr.grant.UserID, tr.grant.OrgID)
		}
	}()

	wg.Wait()

	// Every grant was revoked by one path or the other, so once the dust
	// settles nothing may still authorize.
	for _, tr := range all {
		require.True(t, tr.revoked.Load(), "grant %s was never revoked; the test did not run to completion", tr.grant.ID)
		require.ErrorIs(t, p.Authorize(ctx, agentSession(tr.grant), "read", "invoice"), ErrGrantInactive,
			"grant %s still authorizes after the hammer finished", tr.grant.ID)
	}
}
