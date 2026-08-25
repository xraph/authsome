package anomaly

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// p.patterns is shared mutable state written on every authentication, and
// nothing in this package exercised it from more than one goroutine before
// this file. `go test ./plugins/anomaly/ -race` therefore passed without the
// detector observing a single concurrent access.
//
// recordLogin is already written for concurrency, and carefully: it takes the
// write lock, mutates the pattern, then builds a snapshot with
// CountryHistogram copied key by key, and only then unlocks and does the
// analysis and IO. That last part is the load-bearing bit. LoginPattern's
// HourHistogram and DayHistogram are fixed-size arrays, so a plain struct
// copy duplicates them, but CountryHistogram is a map and a plain copy shares
// it. Analysis reading a shared map outside the lock, while another
// goroutine's recordLogin writes it inside, is a data race.
//
// Two things are covered here: that the concurrent path does not race, and
// that no login is lost when goroutines collide on one principal's pattern.

const (
	anomalyHammerWorkers = 4
	anomalyLoginsPerRef  = 60
)

// TestRecordLogin_ConcurrentSamePrincipalLosesNoLogins hammers one principal's
// pattern from every goroutine at once and then checks the arithmetic.
//
// LoginCount is a read-modify-write on shared state. If the increment ever
// escapes the write lock, concurrent logins interleave and the count comes out
// short. Asserting the exact total is what makes that visible: a lost update
// is silent otherwise, since a slightly low login count still looks plausible.
func TestRecordLogin_ConcurrentSamePrincipalLosesNoLogins(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 5, EnableGeoAnomaly: true, EnableTimeAnomaly: true}, usMapping)
	ctx := context.Background()

	ref := principal.Ref{Kind: principal.KindAgent, ID: "svc_hammer"}

	var wg sync.WaitGroup
	for w := 0; w < anomalyHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < anomalyLoginsPerRef; i++ {
				if err := p.recordLogin(ctx, ref, "app-1", session.Session{}.ID, "1.1.1.1"); err != nil {
					t.Errorf("recordLogin: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	want := anomalyHammerWorkers * anomalyLoginsPerRef

	p.mu.RLock()
	pattern, ok := p.patterns[ref.String()]
	require.True(t, ok, "the hammered principal has no pattern at all")
	gotCount := pattern.LoginCount
	gotCountry := pattern.CountryHistogram["US"]
	hourTotal := 0
	for _, n := range pattern.HourHistogram {
		hourTotal += n
	}
	p.mu.RUnlock()

	require.Equal(t, want, gotCount,
		"LoginCount lost updates under concurrency: the increment must happen under the write lock")
	require.Equal(t, want, gotCountry,
		"CountryHistogram lost updates: every one of these logins resolved to US")
	require.Equal(t, want, hourTotal,
		"HourHistogram lost updates")
}

// TestRecordLogin_ConcurrentDistinctPrincipalsStayIsolated runs many
// principals concurrently. Each one's tally must come out exactly right,
// which fails both if the map write races and if two principals ever share a
// pattern, the isolation TestAnomalyKeysByPrincipal pins sequentially.
func TestRecordLogin_ConcurrentDistinctPrincipalsStayIsolated(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 5, EnableGeoAnomaly: true, EnableTimeAnomaly: true}, usMapping)
	ctx := context.Background()

	refs := make([]principal.Ref, 8)
	for i := range refs {
		refs[i] = principal.Ref{Kind: principal.KindAgent, ID: fmt.Sprintf("svc_%d", i)}
	}

	const perRefPerWorker = 20

	var wg sync.WaitGroup
	for w := 0; w < anomalyHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := ctx
			for i := 0; i < perRefPerWorker; i++ {
				for _, ref := range refs {
					if err := p.recordLogin(ctx, ref, "app-1", session.Session{}.ID, "1.1.1.1"); err != nil {
						t.Errorf("recordLogin: %v", err)
						return
					}
				}
			}
		}()
	}
	wg.Wait()

	want := anomalyHammerWorkers * perRefPerWorker

	p.mu.RLock()
	defer p.mu.RUnlock()
	require.Len(t, p.patterns, len(refs), "every principal must have exactly one pattern of its own")
	for _, ref := range refs {
		pattern, ok := p.patterns[ref.String()]
		require.True(t, ok, "principal %s has no pattern", ref.ID)
		require.Equal(t, want, pattern.LoginCount, "principal %s lost logins", ref.ID)
	}
}

// TestRecordLogin_ConcurrentMixedGeography drives two countries at once, so
// CountryHistogram takes concurrent writes on two different keys rather than
// repeatedly on one. A map write racing a map read is the failure the
// snapshot copy in recordLogin exists to prevent, and Go's map
// implementation detects concurrent map read/write itself, so this also
// exercises that path.
func TestRecordLogin_ConcurrentMixedGeography(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 2, EnableGeoAnomaly: true, EnableTimeAnomaly: true}, usMapping)
	ctx := context.Background()

	ref := principal.Ref{Kind: principal.KindAgent, ID: "svc_travel"}
	ips := []string{"1.1.1.1", "2.2.2.2"} // US and GB in usMapping

	var wg sync.WaitGroup
	for w := 0; w < anomalyHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < anomalyLoginsPerRef; i++ {
				ip := ips[(w+i)%len(ips)]
				if err := p.recordLogin(ctx, ref, "app-1", session.Session{}.ID, ip); err != nil {
					t.Errorf("recordLogin: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	want := anomalyHammerWorkers * anomalyLoginsPerRef

	p.mu.RLock()
	defer p.mu.RUnlock()
	pattern := p.patterns[ref.String()]
	require.NotNil(t, pattern)
	require.Equal(t, want, pattern.LoginCount)

	total := 0
	for _, n := range pattern.CountryHistogram {
		total += n
	}
	require.Equal(t, want, total,
		"every login resolved to a country, so the histogram must account for all of them")
}

// TestOnAfterPrincipalAuth_ConcurrentIsRaceFree drives the exported hook
// rather than recordLogin directly, so the path a real caller takes is the
// one under the detector. It asserts no invariant of its own beyond not
// racing and not deadlocking; the arithmetic is the tests above.
func TestOnAfterPrincipalAuth_ConcurrentIsRaceFree(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 5, EnableGeoAnomaly: true, EnableTimeAnomaly: true}, usMapping)
	ctx := context.Background()

	refs := []principal.Ref{
		{Kind: principal.KindAgent, ID: "svc_a"},
		{Kind: principal.KindAgent, ID: "svc_b"},
	}

	deadline := time.Now().Add(150 * time.Millisecond)
	var wg sync.WaitGroup
	for w := 0; w < anomalyHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				ref := refs[(w+i)%len(refs)]
				ip := "1.1.1.1"
				if i%3 == 0 {
					ip = "2.2.2.2"
				}
				if err := p.OnAfterPrincipalAuth(ctx,
					&principal.AuthAttempt{Subject: ref, IPAddress: ip},
					&session.Session{IPAddress: ip}); err != nil {
					t.Errorf("OnAfterPrincipalAuth: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()

	p.mu.RLock()
	defer p.mu.RUnlock()
	require.NotEmpty(t, p.patterns, "the hammer recorded nothing; it is not exercising the plugin")
}
