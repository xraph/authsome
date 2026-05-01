// Package secutil provides shared test helpers for authsome security work.
//
// This package is test-only. Production code MUST NOT import it.
package secutil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// testAppID is a stable, deterministic AppID used by NewTestEngine so attack
// tests don't have to care about which app they hit.
const testAppID = "aapp_01jf0000000000000000000000"

// NewTestEngine spins up an authsome engine backed by an in-memory store with
// deterministic test secrets. The engine is started and registered for cleanup
// via t.Cleanup, so tests don't need to call Stop.
//
// Additional options can be supplied to override or extend the defaults.
func NewTestEngine(t *testing.T, opts ...authsome.Option) *authsome.Engine {
	t.Helper()

	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err, "secutil: build warden engine")

	base := []authsome.Option{
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppID),
	}
	all := append(base, opts...)

	eng, err := authsome.NewEngine(all...)
	require.NoError(t, err, "secutil: build authsome engine")

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx), "secutil: start authsome engine")
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })

	return eng
}

// AttackRequest builds an httptest request that DELIBERATELY omits Origin,
// Referer, and Cookie headers. This guarantees attack-replay tests cannot
// accidentally pass because of friendly default headers; callers must add
// any header they actually need to simulate.
func AttackRequest(t *testing.T, method, target string, body []byte) *http.Request {
	t.Helper()

	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rdr)

	// Defensive: strip headers httptest.NewRequest may set, and anything a
	// caller might inadvertently inherit. We want a hostile, minimal request.
	req.Header.Del("Origin")
	req.Header.Del("Referer")
	req.Header.Del("Cookie")
	return req
}

// BufferedChronicle is an in-memory bridge.Chronicle implementation that
// records every event for later inspection. Safe for concurrent use.
type BufferedChronicle struct {
	mu     sync.Mutex
	events []bridge.AuditEvent
}

// NewBufferedChronicle returns an empty BufferedChronicle.
func NewBufferedChronicle() *BufferedChronicle {
	return &BufferedChronicle{}
}

// Record implements bridge.Chronicle by appending a copy of event to the
// internal buffer.
func (c *BufferedChronicle) Record(_ context.Context, event *bridge.AuditEvent) error {
	if event == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, *event)
	return nil
}

// Events returns a copy of all events recorded so far.
func (c *BufferedChronicle) Events() []bridge.AuditEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]bridge.AuditEvent, len(c.events))
	copy(out, c.events)
	return out
}

// AssertAuditEvent fails the test unless EXACTLY ONE event with the given
// action was recorded. If a single match is found and inspect is non-nil,
// inspect is invoked with a pointer to the matched event so callers can
// make additional assertions.
func AssertAuditEvent(t *testing.T, c *BufferedChronicle, action string, inspect func(*bridge.AuditEvent)) {
	t.Helper()
	events := c.Events()
	matches := make([]*bridge.AuditEvent, 0, 1)
	for i := range events {
		if events[i].Action == action {
			matches = append(matches, &events[i])
		}
	}
	if len(matches) != 1 {
		t.Fatalf("secutil: expected exactly 1 audit event with action=%q, got %d (recorded actions: %v)",
			action, len(matches), recordedActions(events))
	}
	if inspect != nil {
		inspect(matches[0])
	}
}

// AssertNoAuditEvent fails the test if any event with the given action was
// recorded.
func AssertNoAuditEvent(t *testing.T, c *BufferedChronicle, action string) {
	t.Helper()
	events := c.Events()
	for i := range events {
		if events[i].Action == action {
			t.Fatalf("secutil: expected no audit event with action=%q, but found one (event=%+v)",
				action, events[i])
		}
	}
}

func recordedActions(events []bridge.AuditEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Action
	}
	return out
}
