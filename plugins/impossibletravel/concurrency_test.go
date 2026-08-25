package impossibletravel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// The plugin holds two pieces of shared mutable state behind one RWMutex:
// lastLogins, written on every authentication, and events, appended to
// whenever an alert fires. Neither was ever touched from a second goroutine
// by this package's tests, so `go test -race` here proved nothing.
//
// One design note worth recording, because it is what makes the current code
// correct and it is not obvious from reading recordLocation alone.
// recordLocation reads prev out of the map under RLock, releases the lock,
// and only then reads prev.LoginAt, prev.Latitude and prev.Longitude. That is
// safe solely because a *LoginLocation is never mutated once stored: an
// update replaces the map entry with a freshly built pointer rather than
// writing through the old one. Add a single in-place field update anywhere
// and those reads become races against every concurrent recordLocation.
// TestRecordLocation_ConcurrentSamePrincipal is what would catch it.

const (
	travelHammerWorkers = 4
	travelHammerBudget  = 150 * time.Millisecond
)

// TestRecordLocation_ConcurrentSamePrincipal drives the real path from
// several goroutines against one principal, cycling through cities far enough
// apart to trip the speed threshold, so alert handling (which takes the write
// lock to append) runs concurrently with the map reads and writes.
//
// No assertion is made about how many alerts fire. recordLocation reads prev,
// computes, then writes current without holding the lock across all three, so
// which login is "previous" under concurrency is genuinely nondeterministic.
// That is inherent to the design, not a defect this test should pin.
func TestRecordLocation_ConcurrentSamePrincipal(t *testing.T) {
	p := newTestPlugin(Config{
		MaxSpeedKmH:    900,
		MinDistanceKm:  100,
		LookbackWindow: time.Hour,
	}, defaultMapping)
	ctx := context.Background()

	ref := principal.Ref{Kind: principal.KindAgent, ID: "svc_traveller"}
	ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

	deadline := time.Now().Add(travelHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < travelHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				ip := ips[(w+i)%len(ips)]
				if err := p.recordLocation(ctx, ref, "app-1", id.NewSessionID(), ip); err != nil {
					t.Errorf("recordLocation: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}

	// Concurrent readers of the alert log, which takes the read lock while
	// the writers above are appending under the write lock.
	for w := 0; w < travelHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				_ = p.RecordedEvents()
				ops.Add(1)
			}
		}()
	}

	wg.Wait()

	require.Greater(t, ops.Load(), int64(500),
		"the hammer did no meaningful work; it is not exercising the plugin concurrently")
}

// TestHandleAlert_ConcurrentAppendsLoseNothing pins the append. events is a
// slice grown from several goroutines, and an append that escapes the write
// lock silently drops entries: two goroutines read the same length, both
// write to that index, and one alert vanishes. A missing security alert is
// exactly the kind of loss that leaves no trace, so the count is asserted
// exactly rather than approximately.
func TestHandleAlert_ConcurrentAppendsLoseNothing(t *testing.T) {
	p := newTestPlugin(Config{MaxSpeedKmH: 900, MinDistanceKm: 100}, defaultMapping)
	ctx := context.Background()

	const alertsPerWorker = 100

	var wg sync.WaitGroup
	for w := 0; w < travelHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < alertsPerWorker; i++ {
				p.handleAlert(ctx, "app-1", &TravelAlert{
					Principal: principal.Ref{Kind: principal.KindAgent, ID: fmt.Sprintf("svc_%d", w)},
					SessionID: id.NewSessionID(),
					RiskLevel: "critical",
				})
			}
		}(w)
	}
	wg.Wait()

	got := p.RecordedEvents()
	require.Len(t, got, travelHammerWorkers*alertsPerWorker,
		"alerts were lost: the append to events must happen under the write lock")
}

// TestRecordedEvents_DoesNotAliasInternalSlice pins that the caller gets a
// snapshot rather than a window onto live state. RecordedEvents copies, and
// TravelAlert is all value types today (LoginLocation included), so the copy
// is a deep one. Add a slice or map field to TravelAlert and this stops being
// true, which is what the second half of this test would catch.
func TestRecordedEvents_DoesNotAliasInternalSlice(t *testing.T) {
	p := newTestPlugin(Config{MaxSpeedKmH: 900, MinDistanceKm: 100}, defaultMapping)
	ctx := context.Background()

	p.handleAlert(ctx, "app-1", &TravelAlert{
		Principal: principal.Ref{Kind: principal.KindAgent, ID: "svc_a"},
		RiskLevel: "critical",
		ToLocation: LoginLocation{
			Principal: principal.Ref{Kind: principal.KindAgent, ID: "svc_a"},
			City:      "London", Country: "GB",
		},
	})

	first := p.RecordedEvents()
	require.Len(t, first, 1)
	first[0].RiskLevel = "medium"
	first[0].ToLocation.City = "Nowhere"
	grown := append(first, TravelAlert{RiskLevel: "forged"})
	require.Len(t, grown, 2, "the caller's own view may grow")

	second := p.RecordedEvents()
	require.Len(t, second, 1, "appending to the returned slice must not grow the plugin's own log")
	require.Equal(t, "critical", second[0].RiskLevel, "mutating the returned slice must not rewrite the recorded alert")
	require.Equal(t, "London", second[0].ToLocation.City)
}

// TestRecordLocation_ConcurrentDistinctPrincipalsStayIsolated runs many
// principals at once. Each keeps its own last-known location, so after the
// hammer every one of them must have exactly one entry and no principal may
// have picked up another's.
func TestRecordLocation_ConcurrentDistinctPrincipalsStayIsolated(t *testing.T) {
	p := newTestPlugin(Config{
		MaxSpeedKmH:    900,
		MinDistanceKm:  100,
		LookbackWindow: time.Hour,
	}, defaultMapping)
	ctx := context.Background()

	refs := make([]principal.Ref, 8)
	for i := range refs {
		refs[i] = principal.Ref{Kind: principal.KindAgent, ID: fmt.Sprintf("svc_%d", i)}
	}

	// Each principal is pinned to one IP, so its stored location is
	// predictable no matter how the goroutines interleave.
	ipFor := func(i int) string {
		return []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}[i%3]
	}
	cityFor := func(i int) string {
		return []string{"New York", "London", "Sydney"}[i%3]
	}

	var wg sync.WaitGroup
	for w := 0; w < travelHammerWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := 0; round < 20; round++ {
				for i, ref := range refs {
					if err := p.recordLocation(ctx, ref, "app-1", id.NewSessionID(), ipFor(i)); err != nil {
						t.Errorf("recordLocation: %v", err)
						return
					}
				}
			}
		}()
	}
	wg.Wait()

	p.mu.RLock()
	defer p.mu.RUnlock()
	require.Len(t, p.lastLogins, len(refs), "every principal must have exactly one last-login entry")
	for i, ref := range refs {
		got, ok := p.lastLogins[ref.String()]
		require.True(t, ok, "principal %s has no recorded location", ref.ID)
		require.Equal(t, cityFor(i), got.City,
			"principal %s picked up another principal's location", ref.ID)
		require.Equal(t, ref.String(), got.Principal.String())
	}
}

// TestOnAfterSignIn_ConcurrentIsRaceFree drives the exported hook, so the
// path a real caller takes is the one the detector watches.
func TestOnAfterSignIn_ConcurrentIsRaceFree(t *testing.T) {
	p := newTestPlugin(Config{
		MaxSpeedKmH:    900,
		MinDistanceKm:  100,
		LookbackWindow: time.Hour,
	}, defaultMapping)
	ctx := context.Background()

	users := make([]*user.User, 4)
	for i := range users {
		users[i] = &user.User{ID: id.NewUserID(), AppID: id.NewAppID()}
	}
	ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

	deadline := time.Now().Add(travelHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < travelHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				u := users[(w+i)%len(users)]
				ip := ips[(w+i)%len(ips)]
				if err := p.OnAfterSignIn(ctx, u, &session.Session{ID: id.NewSessionID(), IPAddress: ip}); err != nil {
					t.Errorf("OnAfterSignIn: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(200), "the hammer did no meaningful work")
	p.mu.RLock()
	defer p.mu.RUnlock()
	require.NotEmpty(t, p.lastLogins, "the hammer recorded nothing")
}
