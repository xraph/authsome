package sharedsignals

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The loop is driven through an injected tick channel rather than a real
// time.Ticker, so these tests assert on what the refresher does instead of
// on how long a sleep took.
func TestRefresher_RunsARoundOnEveryTick(t *testing.T) {
	var rounds int64
	r := newRefresher(time.Minute, func(context.Context) {
		atomic.AddInt64(&rounds, 1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	tick := make(chan time.Time)
	go r.run(ctx, tick)

	tick <- time.Now()
	tick <- time.Now()

	cancel()
	<-r.done
	assert.Equal(t, int64(2), atomic.LoadInt64(&rounds))
}

// A refresher that outlives the engine keeps an IdP's JWKS endpoint on a
// timer belonging to a process that has finished with it, so stopping has to
// actually wait for the loop to leave rather than just ask it to.
func TestRefresher_StopWaitsForTheLoopToExit(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	r := newRefresher(time.Minute, func(context.Context) {
		close(entered)
		<-release
	})

	tick := make(chan time.Time, 1)
	r.startWith(tick)
	tick <- time.Now()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("the tick never reached a refresh round")
	}

	stopped := make(chan struct{})
	go func() { r.stop(); close(stopped) }()

	select {
	case <-stopped:
		t.Fatal("stop returned while a refresh round was still running")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("stop never returned after the round finished")
	}
}

// OnShutdown runs on a plugin whose OnInit may never have been reached, and
// the registry calls it once per shutdown but nothing stops a host calling
// it twice. Neither may panic on a closed channel.
func TestRefresher_StopIsSafeWhenNeverStartedAndWhenRepeated(t *testing.T) {
	r := newRefresher(time.Minute, func(context.Context) {})
	assert.NotPanics(t, r.stop, "stopping a refresher that never started")

	r2 := newRefresher(time.Minute, func(context.Context) {})
	r2.startWith(make(chan time.Time))
	assert.NotPanics(t, r2.stop)
	assert.NotPanics(t, r2.stop, "stopping twice")
}

// The ticker is the mechanism the design doc promised and the tree never
// grew, so the wiring is worth pinning: an initialised plugin has a running
// refresher, and shutting it down leaves nothing behind.
func TestPlugin_OnInitStartsTheRefresherAndOnShutdownStopsIt(t *testing.T) {
	p := New()
	require.NoError(t, p.OnInit(context.Background(), stubEngine{}))
	require.NotNil(t, p.refresher, "OnInit must start the JWKS refresh ticker")

	require.NoError(t, p.OnShutdown(context.Background()))
	select {
	case <-p.refresher.done:
	case <-time.After(time.Second):
		t.Fatal("OnShutdown left the refresh loop running")
	}
}

// An embedder that does not want a background goroutine has to be able to
// say so, and must not silently get one anyway.
func TestPlugin_ANonPositiveRefreshIntervalStartsNoTicker(t *testing.T) {
	p := New(Config{KeyRefreshInterval: -1})
	require.NoError(t, p.OnInit(context.Background(), stubEngine{}))
	assert.Nil(t, p.refresher, "a disabled interval must not start a goroutine")
	assert.NoError(t, p.OnShutdown(context.Background()))
}

func TestConfig_KeyRefreshIntervalDefaultsToFifteenMinutes(t *testing.T) {
	var c Config
	c.defaults()
	assert.Equal(t, 15*time.Minute, c.KeyRefreshInterval)
}
