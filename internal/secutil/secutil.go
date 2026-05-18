// Package secutil provides shared test helpers for authsome security work.
//
// This package is test-only. Production code MUST NOT import it.
package secutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
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

	// Seed the platform app at testAppID BEFORE engine.Start so bootstrap
	// adopts it via slug "platform" and engine.SignUp's app-existence
	// check (added alongside the publishable-key fix) succeeds for the
	// constant testAppID used throughout the test suite.
	parsedAppID, parseErr := id.ParseAppID(testAppID)
	require.NoError(t, parseErr, "secutil: parse testAppID")
	now := time.Now()
	require.NoError(t, s.CreateApp(context.Background(), &app.App{
		ID:             parsedAppID,
		Name:           "Platform",
		Slug:           "platform",
		PublishableKey: "pk_test_authsome_secutil_default",
		IsPlatform:     true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}), "secutil: seed platform app")

	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err, "secutil: build warden engine")

	all := make([]authsome.Option, 0, 4+len(opts))
	all = append(all,
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppID),
	)
	all = append(all, opts...)

	eng, err := authsome.NewEngine(all...)
	require.NoError(t, err, "secutil: build authsome engine")

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx), "secutil: start authsome engine")
	t.Cleanup(func() { _ = eng.Stop(context.Background()) }) //nolint:errcheck // cleanup best-effort

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
	req := httptest.NewRequestWithContext(context.Background(), method, target, rdr)

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

// AttachChronicle replaces the engine's chronicle with the given buffered
// chronicle so a test can assert against recorded events. Call sites that
// emit audit via plugin-cached references should re-resolve from the engine
// after this call (see organization.Plugin.SetChronicleForTest).
func AttachChronicle(t *testing.T, eng *authsome.Engine, ch *BufferedChronicle) {
	t.Helper()
	require.NotNil(t, eng, "secutil: AttachChronicle: nil engine")
	require.NotNil(t, ch, "secutil: AttachChronicle: nil chronicle")
	eng.SetChronicle(ch)
}

// InitTestNonceSigner installs a deterministic process-wide nonce signer so
// tests that exercise GenerateScopedNonce / ConsumeScopedNonce code paths
// don't have to wire one themselves.
func InitTestNonceSigner(t *testing.T) {
	t.Helper()
	require.NoError(t, dashboard.InitNonceSigner([]byte("secutil-test-nonce-signer-secret-32bytes!")),
		"secutil: init nonce signer")
}

// RelaxAuthDefaults disables the production-secure auth defaults that would
// otherwise break the bulk of authsome's pre-existing tests:
//
//   - SettingRequireEmailVerification (default true since Phase 2A) is
//     overridden to false at the global scope so signed-up users can sign
//     in immediately without going through the verification flow.
//
// Tests that specifically exercise these gates (verification-required,
// captcha-required, etc.) should write their own per-app overrides on top
// of this baseline; this helper covers the "I just want a usable engine
// for a sign-in test" case.
//
// Centralised here so additions to the secure-by-default set don't require
// touching every test bootstrap individually.
func RelaxAuthDefaults(t *testing.T, eng *authsome.Engine) {
	t.Helper()
	require.NotNil(t, eng, "secutil: RelaxAuthDefaults: nil engine")
	mgr := eng.Settings()
	if mgr == nil {
		return
	}
	require.NoError(t,
		mgr.Set(context.Background(),
			"auth.require_email_verification",
			json.RawMessage(`false`),
			settings.ScopeGlobal, "", "", "", "test-bootstrap"),
		"secutil: relax auth.require_email_verification")
}

// InjectStoreFault makes the named store method return err on its next
// call against the underlying memory store. Test-only.
func InjectStoreFault(t *testing.T, eng *authsome.Engine, method string, err error) {
	t.Helper()
	memStore, ok := eng.Store().(*memory.Store)
	if !ok {
		t.Fatalf("InjectStoreFault requires the memory store; got %T", eng.Store())
	}
	memStore.InjectOneShotFault(method, err)
}

// afterOrgDeleteTestPlugin is a minimal plugin that fires fn whenever the
// AfterOrgDelete hook is emitted. It exists so tests can observe the hook
// without booting an entire downstream plugin.
type afterOrgDeleteTestPlugin struct {
	name string
	fn   func(context.Context, id.OrgID) error
}

func (p *afterOrgDeleteTestPlugin) Name() string { return p.name }
func (p *afterOrgDeleteTestPlugin) OnAfterOrgDelete(ctx context.Context, orgID id.OrgID) error {
	return p.fn(ctx, orgID)
}

// onAfterOrgDeleteSeq is a process-wide counter used to give each
// OnAfterOrgDelete test plugin a unique name. The plugin registry has no
// Unregister entry point at time of writing, so the registered plugin
// leaks for the remainder of the engine's lifetime — but the engine itself
// is torn down by NewTestEngine's t.Cleanup, so the leak is bounded to the
// test.
var onAfterOrgDeleteSeq int64
var onAfterOrgDeleteMu sync.Mutex

// OnAfterOrgDelete registers an additional AfterOrgDelete hook for the
// test. The registered plugin leaks for the engine's lifetime (no
// Unregister API exists yet), which is acceptable because the engine is
// scoped to the test via NewTestEngine.
func OnAfterOrgDelete(t *testing.T, eng *authsome.Engine, fn func(context.Context, id.OrgID) error) {
	t.Helper()
	onAfterOrgDeleteMu.Lock()
	onAfterOrgDeleteSeq++
	name := "secutil-after-org-delete-" + itoa(onAfterOrgDeleteSeq)
	onAfterOrgDeleteMu.Unlock()
	eng.Plugins().Register(&afterOrgDeleteTestPlugin{name: name, fn: fn})
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func recordedActions(events []bridge.AuditEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Action
	}
	return out
}
