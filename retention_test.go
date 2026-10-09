package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// sweepSpy is a plugin that records what the engine's sweeper asked of it.
type sweepSpy struct {
	calls   int
	batch   int
	cutoffs map[string]time.Time
}

func (*sweepSpy) Name() string { return "sweep-spy" }

func (s *sweepSpy) SweepRetention(_ context.Context, cutoff func(string) time.Time, batch int) (int64, error) {
	s.calls++
	s.batch = batch
	s.cutoffs = map[string]time.Time{
		store.RetentionAuthCodes:   cutoff(store.RetentionAuthCodes),
		store.RetentionDeviceCodes: cutoff(store.RetentionDeviceCodes),
	}
	return 2, nil
}

// The sweeper removes what has expired, leaves live rows, asks plugins to
// sweep with cutoffs derived from the configuration, and runs once per
// lease: a second call while the lease is held does nothing.
func TestSweepRetention(t *testing.T) {
	cfg := testEngineConfig()
	cfg.Retention.DeviceCodesDays = -1 // kept forever
	cfg.Retention.BatchSize = 7
	spy := &sweepSpy{}
	eng, st := newTestEngine(t, authsome.WithConfig(cfg), authsome.WithPlugin(spy))
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	seed := func(name string, expires time.Time) *session.Session {
		s := &session.Session{
			ID: id.NewSessionID(), AppID: appID, UserID: id.NewUserID(), Token: name, RefreshToken: name + "-r",
			ExpiresAt: expires, RefreshTokenExpiresAt: expires, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, st.CreateSession(ctx, s))
		return s
	}
	dead := seed("dead", time.Now().Add(-31*24*time.Hour))
	recent := seed("recent", time.Now().Add(-time.Hour)) // expired, but inside the 30 day window
	live := seed("live", time.Now().Add(time.Hour))

	n, err := eng.SweepRetention(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 3, n, "one session plus the two rows the plugin reported")
	_, err = st.GetSession(ctx, dead.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	for _, kept := range []*session.Session{recent, live} {
		_, err = st.GetSession(ctx, kept.ID)
		assert.NoError(t, err, "a row inside its retention window stays")
	}

	require.Equal(t, 1, spy.calls)
	assert.Equal(t, 7, spy.batch)
	assert.WithinDuration(t, time.Now().Add(-24*time.Hour), spy.cutoffs[store.RetentionAuthCodes], time.Minute, "auth codes default to one day")
	assert.True(t, spy.cutoffs[store.RetentionDeviceCodes].IsZero(), "a negative window means never")

	seed("dead-again", time.Now().Add(-31*24*time.Hour))
	n, err = eng.SweepRetention(ctx)
	require.NoError(t, err)
	assert.Zero(t, n, "the lease is still held, so nothing runs")
	assert.Equal(t, 1, spy.calls)

	require.NoError(t, st.KVDelete(ctx, "authsome:retention:lease"))
	n, err = eng.SweepRetention(ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 3, n, "with the lease released the sweep runs again")
}

// Stop ends the sweeper and returns only once it has, so no sweep runs
// against a store that is shutting down.
func TestStopEndsRetentionSweeper(t *testing.T) {
	cfg := testEngineConfig()
	cfg.Session.CleanupInterval = 10 * time.Millisecond
	eng, _ := newTestEngine(t, authsome.WithConfig(cfg))
	time.Sleep(30 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, eng.Stop(context.Background()))
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return; the sweeper goroutine is stuck")
	}
}
