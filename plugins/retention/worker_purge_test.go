package retention

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"
)

// purgeCall records one PurgeTerminal invocation's cutoffs.
type purgeCall struct {
	done  time.Time
	audit time.Time
}

// purgeSpyStore is a MemoryStore that records what the sweep asked for, and
// can be made to fail so the delivery loop's tolerance of a failed purge is
// assertable rather than assumed.
type purgeSpyStore struct {
	*MemoryStore

	mu    sync.Mutex
	calls []purgeCall
	err   error
}

func newPurgeSpy() *purgeSpyStore {
	return &purgeSpyStore{MemoryStore: NewMemoryStore()}
}

func (s *purgeSpyStore) PurgeTerminal(ctx context.Context, done, audit time.Time) (int, error) {
	s.mu.Lock()
	s.calls = append(s.calls, purgeCall{done: done, audit: audit})
	failWith := s.err
	s.mu.Unlock()
	if failWith != nil {
		return 0, failWith
	}
	return s.MemoryStore.PurgeTerminal(ctx, done, audit)
}

func (s *purgeSpyStore) recorded() []purgeCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]purgeCall, len(s.calls))
	copy(out, s.calls)
	return out
}

// newPurgeWorker builds a worker whose sweep settings are the subject of the
// test. Interval stays short because these tests drive maybePurge and
// runOnce directly; nothing here starts the loop or sleeps.
func newPurgeWorker(s Store, doneRet, auditRet, purgeEvery time.Duration) *worker {
	return newWorker(workerDeps{
		Store:          s,
		Providers:      map[string]Provider{},
		Logger:         log.NewNoopLogger(),
		Interval:       time.Second,
		Lease:          time.Minute,
		BatchSize:      10,
		DoneRetention:  doneRet,
		AuditRetention: auditRet,
		PurgeInterval:  purgeEvery,
		LoadContact: func(_ context.Context, j *Job) (*Contact, error) {
			return &Contact{UserID: j.UserID, AppID: j.AppID, Email: "a@example.com"}, nil
		},
	})
}

func TestWorkerSweepRunsFarLessOftenThanDelivery(t *testing.T) {
	ctx := context.Background()
	spy := newPurgeSpy()
	w := newPurgeWorker(spy, 30*24*time.Hour, 180*24*time.Hour, time.Hour)

	now := time.Now()
	// newWorker starts the clock at construction, so the first sweep is one
	// interval away rather than on the first tick. A fleet that restarts
	// every few minutes must not sweep the whole table on every rollout.
	w.maybePurge(ctx, now)
	assert.Empty(t, spy.recorded(), "the sweep does not run on the first tick")

	w.maybePurge(ctx, now.Add(59*time.Minute))
	assert.Empty(t, spy.recorded(), "still inside the interval")

	w.maybePurge(ctx, now.Add(time.Hour+time.Second))
	require.Len(t, spy.recorded(), 1, "one sweep once the interval has passed")

	// Ticks keep arriving every 30 seconds. None of them may sweep again.
	for i := 1; i <= 10; i++ {
		w.maybePurge(ctx, now.Add(time.Hour+time.Second+time.Duration(i)*30*time.Second))
	}
	assert.Len(t, spy.recorded(), 1, "delivery ticks must not each trigger a sweep")

	w.maybePurge(ctx, now.Add(2*time.Hour+2*time.Second))
	assert.Len(t, spy.recorded(), 2)
}

func TestWorkerSweepPassesTheConfiguredWindows(t *testing.T) {
	ctx := context.Background()
	spy := newPurgeSpy()
	w := newPurgeWorker(spy, 30*24*time.Hour, 180*24*time.Hour, time.Hour)

	now := time.Now()
	w.maybePurge(ctx, now.Add(2*time.Hour))

	calls := spy.recorded()
	require.Len(t, calls, 1)
	at := now.Add(2 * time.Hour)
	assert.WithinDuration(t, at.Add(-30*24*time.Hour), calls[0].done, time.Second)
	assert.WithinDuration(t, at.Add(-180*24*time.Hour), calls[0].audit, time.Second,
		"the audit trail gets the longer window, not the same one")
}

func TestWorkerSweepDisabledPerClassAndOverall(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("a negative done retention keeps done rows forever", func(t *testing.T) {
		spy := newPurgeSpy()
		w := newPurgeWorker(spy, -1, 180*24*time.Hour, time.Hour)
		w.maybePurge(ctx, now.Add(2*time.Hour))

		calls := spy.recorded()
		require.Len(t, calls, 1)
		assert.True(t, calls[0].done.IsZero(), "a disabled class is a zero cutoff, which reaches nothing")
		assert.False(t, calls[0].audit.IsZero())
	})

	t.Run("both negative skips the store entirely", func(t *testing.T) {
		spy := newPurgeSpy()
		w := newPurgeWorker(spy, -1, -1, time.Hour)
		w.maybePurge(ctx, now.Add(2*time.Hour))
		assert.Empty(t, spy.recorded(), "nothing to delete means no round trip")
	})

	t.Run("a non-positive interval switches the sweep off", func(t *testing.T) {
		spy := newPurgeSpy()
		w := newPurgeWorker(spy, 30*24*time.Hour, 180*24*time.Hour, -1)
		for i := range 10 {
			w.maybePurge(ctx, now.Add(time.Duration(i)*24*time.Hour))
		}
		assert.Empty(t, spy.recorded())
	})
}

// A failed purge is a disk problem for tomorrow. Letting it stop the claim
// would turn it into a sync outage today, so runOnce must deliver anyway.
func TestWorkerSweepFailureDoesNotStopDelivery(t *testing.T) {
	ctx := context.Background()
	spy := newPurgeSpy()
	spy.err = errors.New("delete failed")

	p := &fakeProvider{caps: CapContacts | CapActivities}
	w := newWorker(workerDeps{
		Store:         spy,
		Providers:     map[string]Provider{"fake": p},
		Logger:        log.NewNoopLogger(),
		Interval:      time.Second,
		Lease:         time.Minute,
		BatchSize:     10,
		MaxAttempts:   3,
		BaseBackoff:   time.Second,
		PurgeInterval: time.Nanosecond, // due on the very next tick
		LoadContact: func(_ context.Context, j *Job) (*Contact, error) {
			return &Contact{UserID: j.UserID, AppID: j.AppID, Email: "a@example.com"}, nil
		},
	})

	j := enqueued(t, spy, KindActivityLog, "purge-fails")
	w.runOnce(ctx)

	require.NotEmpty(t, spy.recorded(), "the sweep was attempted")
	stored, err := spy.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDone, stored.State, "delivery carries on through a failed purge")
}

// The sweep is wired from Config through to the worker, so an operator who
// sets the fields gets them rather than the defaults.
func TestPluginConfigDefaultsForRetentionWindows(t *testing.T) {
	var c Config
	c.defaults()
	assert.Equal(t, 30*24*time.Hour, c.DoneRetention)
	assert.Equal(t, 180*24*time.Hour, c.AuditRetention)
	assert.Equal(t, time.Hour, c.PurgeInterval)

	// Negative is a real value, not an unset one: it is the only way to say
	// "keep everything", so defaults() must leave it alone.
	custom := Config{DoneRetention: -1, AuditRetention: -1, PurgeInterval: -1}
	custom.defaults()
	assert.Equal(t, time.Duration(-1), custom.DoneRetention)
	assert.Equal(t, time.Duration(-1), custom.AuditRetention)
	assert.Equal(t, time.Duration(-1), custom.PurgeInterval)
}
