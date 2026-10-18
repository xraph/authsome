package authsome_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

func TestRetentionSweepIsARootSpanOnlyWhenItHoldsTheLease(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	eng, s := newTestEngine(t, authsome.WithMetricsSink(reg))
	ctx := context.Background()
	appID := signInApp(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "sweep@example.com", Password: "SecureP@ss123", FirstName: "S"})
	require.NoError(t, err)
	old, err := account.NewSession(appID, u.ID, account.SessionConfig{TokenTTL: time.Hour, RefreshTokenTTL: time.Hour})
	require.NoError(t, err)
	old.ExpiresAt = time.Now().Add(-90 * 24 * time.Hour)
	old.RefreshTokenExpiresAt = old.ExpiresAt
	require.NoError(t, s.CreateSession(ctx, old))

	n := len(rec.Ended())
	removed, err := eng.SweepRetention(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, removed, int64(1))
	// The lease is held for half the interval, so a second sweep right away
	// behaves as another replica would: it skips, and starts no span.
	_, err = eng.SweepRetention(ctx)
	require.NoError(t, err)

	sweeps := telemetrytest.Named(spansSince(rec, n), "sweep retention")
	require.Len(t, sweeps, 1)
	assert.False(t, sweeps[0].Parent().IsValid(), "a sweep outside any request is a root span")
	assert.Equal(t, "1", telemetrytest.Attrs(sweeps[0])["authsome.retention.removed"])

	body := scrape()
	assert.Contains(t, body, `forge_authsome_retention_sweeps_attempted_total{result="ok"} 1`)
	assert.Contains(t, body, `forge_authsome_retention_sweeps_attempted_total{result="skipped"} 1`)
	assert.Contains(t, body, `forge_authsome_retention_rows_deleted_total{kind="sessions"} 1`)
}

// spanSpyStore records whether the queries background work makes carry a
// span. grove opens a db.client span under any valid span context, even an
// ended one, so a true here means the query lands in someone else's trace.
type spanSpyStore struct {
	*memory.Store

	mu       sync.Mutex
	seen     map[string][]bool
	hashed   chan struct{}
	leased   chan struct{}
	hashOnce sync.Once
	kvOnce   sync.Once
}

func (s *spanSpyStore) note(ctx context.Context, op string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[op] = append(s.seen[op], trace.SpanContextFromContext(ctx).IsValid())
}

func (s *spanSpyStore) HashLegacyTokens(ctx context.Context, batch int) (int64, error) {
	s.note(ctx, "hash legacy tokens")
	defer s.hashOnce.Do(func() { close(s.hashed) })
	return s.Store.HashLegacyTokens(ctx, batch)
}

func (s *spanSpyStore) KVSetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	s.note(ctx, "retention lease")
	defer s.kvOnce.Do(func() { close(s.leased) })
	return s.Store.KVSetNX(ctx, key, value, ttl)
}

// forge runs Engine.Start inside a "start authsome" span. Nothing the
// engine leaves running may keep that span: not the legacy token sweep and
// not the retention sweeper's lease check on an idle tick.
func TestBackgroundWorkDropsTheBootSpan(t *testing.T) {
	rec := telemetrytest.Spans(t)
	spy := &spanSpyStore{Store: memory.New(), seen: map[string][]bool{}, hashed: make(chan struct{}), leased: make(chan struct{})}
	appID := signInApp(t)
	now := time.Now()
	require.NoError(t, spy.CreateApp(context.Background(), &app.App{
		ID: appID, Name: "Platform", Slug: "platform", PublishableKey: "pk_test_authsome_bootspan",
		IsPlatform: true, CreatedAt: now, UpdatedAt: now,
	}))
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	cfg := testEngineConfig()
	cfg.Session.CleanupInterval = 10 * time.Millisecond
	eng, err := authsome.NewEngine(
		authsome.WithStore(spy),
		authsome.WithWarden(w),
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithDisableMigrate(),
		authsome.WithConfig(cfg),
		authsome.WithAppID(appID.String()),
	)
	require.NoError(t, err)

	bootCtx, boot := telemetry.Start(context.Background(), "start authsome")
	require.NoError(t, eng.Start(bootCtx))
	boot.End()
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })
	secutil.RelaxAuthDefaults(t, eng)

	for _, ch := range []chan struct{}{spy.hashed, spy.leased} {
		select {
		case <-ch:
		case <-time.After(10 * time.Second):
			t.Fatal("background work never ran")
		}
	}
	require.NoError(t, eng.Stop(context.Background()))

	spy.mu.Lock()
	defer spy.mu.Unlock()
	for op, valid := range spy.seen {
		require.NotEmpty(t, valid, op)
		for _, v := range valid {
			assert.False(t, v, "%s ran under the boot span", op)
		}
	}
	for _, s := range telemetrytest.Named(telemetrytest.Authsome(rec), "sweep retention") {
		assert.False(t, s.Parent().IsValid(), "a background sweep is its own root, never a child of the boot span")
	}
}

// sessionSpy records the context GetSession ran under.
type sessionSpy struct {
	*memory.Store
	mu       sync.Mutex
	spanned  []bool
	canceled []bool
}

func (s *sessionSpy) GetSession(ctx context.Context, sid id.SessionID) (*session.Session, error) {
	s.mu.Lock()
	s.spanned = append(s.spanned, trace.SpanContextFromContext(ctx).IsValid())
	s.canceled = append(s.canceled, ctx.Err() != nil)
	s.mu.Unlock()
	return s.Store.GetSession(ctx, sid)
}

// The JWT session check used to run on context.Background(), so its store
// lookup was in no trace at all. It now runs on the request's values, under
// the request's span, and still ignores the request's cancellation exactly
// as before.
func TestJWTSessionCheckRunsUnderTheRequestButNotItsCancellation(t *testing.T) {
	telemetrytest.Spans(t)
	spy := &sessionSpy{Store: memory.New()}
	appID := signInApp(t)
	now := time.Now()
	require.NoError(t, spy.CreateApp(context.Background(), &app.App{
		ID: appID, Name: "Platform", Slug: "platform", PublishableKey: "pk_test_authsome_jwtcheck",
		IsPlatform: true, CreatedAt: now, UpdatedAt: now,
	}))
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(spy), authsome.WithWarden(w), authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithDisableMigrate(), authsome.WithConfig(testEngineConfig()), authsome.WithAppID(appID.String()),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	t.Cleanup(func() { _ = eng.Stop(context.Background()) })

	reqCtx, cancel := context.WithCancel(context.Background())
	reqCtx, span := telemetry.Start(reqCtx, "request")
	cancel()
	_, _ = eng.JWTSessionContextChecker(reqCtx, appID.String(), id.NewSessionID().String())
	span.End()

	spy.mu.Lock()
	defer spy.mu.Unlock()
	require.Len(t, spy.spanned, 1)
	assert.True(t, spy.spanned[0], "the lookup carries the request's span")
	assert.False(t, spy.canceled[0], "and not its cancellation")
}
