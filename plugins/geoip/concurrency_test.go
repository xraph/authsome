package geoip

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Cache calls itself thread-safe, and until this file existed nothing checked
// that claim. `go test ./plugins/geoip/ -race` passed because no test in the
// package started a goroutine, and the race detector only reports races it
// watches execute. A green -race run over a sequential suite is worth what a
// run with no flag is worth.
//
// Two properties are covered here. Concurrent Get, Set and expiry must not
// race, and neither Get nor Set may hand out or retain a pointer into the
// cache's own storage. The second is what makes the first true for callers:
// Plugin.Resolve returns whatever Get returns, straight through to anomaly,
// geofence, impossibletravel and vpndetect, so a shared pointer would be
// shared across plugins and goroutines both, and one caller writing to a
// field would be an unsynchronized write against every concurrent reader.

const (
	geoHammerWorkers = 4
	geoHammerBudget  = 150 * time.Millisecond
)

// TestCache_ConcurrentGetAndSet drives Get and Set from concurrent goroutines
// over a key space smaller than the worker count, so workers collide on the
// same entries instead of each quietly owning its own key.
func TestCache_ConcurrentGetAndSet(t *testing.T) {
	c := NewCache(time.Hour)

	ips := make([]string, 8)
	for i := range ips {
		ips[i] = fmt.Sprintf("203.0.113.%d", i)
	}

	deadline := time.Now().Add(geoHammerBudget)
	var wg sync.WaitGroup
	// ops guards against a vacuous pass: a hammer whose goroutines all exit
	// immediately races nothing and would still report green.
	var ops atomic.Int64

	spawn := func(fn func(i int)) {
		for w := 0; w < geoHammerWorkers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; time.Now().Before(deadline); i++ {
					fn(w*len(ips) + i)
					ops.Add(1)
				}
			}(w)
		}
	}

	// Readers.
	spawn(func(i int) {
		c.Get(ips[i%len(ips)])
	})

	// Writers. Each builds its own GeoLocation, as Resolve does with whatever
	// the provider returned.
	spawn(func(i int) {
		ip := ips[i%len(ips)]
		c.Set(ip, &GeoLocation{IP: ip, Country: "US", City: "Denver", Latitude: 39.7, Longitude: -104.9})
	})

	// Readers that then write to what they got back. This is the pattern that
	// makes an uncloned Get a data race rather than merely a stale read, and
	// it is why Get copies.
	spawn(func(i int) {
		if got := c.Get(ips[i%len(ips)]); got != nil {
			got.Country = "GB"
			got.IsVPN = true
		}
	})

	wg.Wait()

	require.Greater(t, ops.Load(), int64(1000),
		"the hammer did no meaningful work; it is not exercising the cache concurrently")
}

// TestCache_GetDoesNotAliasStoredEntry pins that a caller cannot reach into
// the cache through the value Get handed it. Without the copy in Get, the
// second read below sees "GB".
func TestCache_GetDoesNotAliasStoredEntry(t *testing.T) {
	c := NewCache(time.Hour)
	c.Set("1.2.3.4", &GeoLocation{IP: "1.2.3.4", Country: "US", City: "New York"})

	got := c.Get("1.2.3.4")
	require.NotNil(t, got)
	got.Country = "GB"
	got.City = "London"

	fresh := c.Get("1.2.3.4")
	require.NotNil(t, fresh)
	require.Equal(t, "US", fresh.Country, "mutating a location returned by Get must not affect the cached entry")
	require.Equal(t, "New York", fresh.City)
}

// TestCache_SetDoesNotAliasCallerValue is the mirror image: the cache must
// copy on the way in too. Plugin.Resolve hands Set the exact value the
// provider returned and then returns that same value to its caller, so
// without this copy the caller holds a live handle on the cached entry.
func TestCache_SetDoesNotAliasCallerValue(t *testing.T) {
	c := NewCache(time.Hour)
	loc := &GeoLocation{IP: "1.2.3.4", Country: "US", City: "New York"}
	c.Set("1.2.3.4", loc)

	loc.Country = "GB" // mutate the caller's value after Set returned
	loc.City = "London"

	got := c.Get("1.2.3.4")
	require.NotNil(t, got)
	require.Equal(t, "US", got.Country, "mutating the caller's value after Set must not affect the cached entry")
	require.Equal(t, "New York", got.City)
}

// TestResolve_DoesNotAliasAcrossCallers is the property the two tests above
// exist to produce, stated at the level callers actually use. anomaly,
// geofence, impossibletravel and vpndetect all call Resolve and read fields
// off the result. Two of them must never be able to see each other's writes.
func TestResolve_DoesNotAliasAcrossCallers(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", City: "Mountain View", IsVPN: false},
	})

	first := p.Resolve("8.8.8.8")
	require.NotNil(t, first)
	first.IsVPN = true // as a caller doing its own bookkeeping might
	first.Country = "GB"

	second := p.Resolve("8.8.8.8")
	require.NotNil(t, second)
	require.False(t, second.IsVPN, "one caller's write must not be visible to the next")
	require.Equal(t, "US", second.Country)
}

// TestCache_ConcurrentExpiry runs Get and Set against a ttl short enough that
// entries lapse mid-run, so the expiry branch in Get is exercised under load
// rather than only in the sequential ttl test.
func TestCache_ConcurrentExpiry(t *testing.T) {
	c := NewCache(time.Millisecond)

	deadline := time.Now().Add(geoHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < geoHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			ip := fmt.Sprintf("198.51.100.%d", w%4)
			for i := 0; time.Now().Before(deadline); i++ {
				c.Set(ip, &GeoLocation{IP: ip, Country: "US"})
				c.Get(ip)
				ops.Add(1)
			}
		}(w)
	}

	wg.Wait()

	require.Greater(t, ops.Load(), int64(500), "the expiry hammer did no meaningful work")
}
