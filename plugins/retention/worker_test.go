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

	"github.com/xraph/authsome/id"
)

// fakeProvider records calls and returns scripted results.
type fakeProvider struct {
	mu          sync.Mutex
	caps        Capability
	upsertErr   []error // consumed one per call
	activityErr []error // consumed one per call
	activity    []*Activity
	upserts     int
	refID       string
}

func (f *fakeProvider) Name() string             { return "fake" }
func (f *fakeProvider) Capabilities() Capability { return f.caps }

func (f *fakeProvider) UpsertContact(_ context.Context, _ *Contact) (RemoteRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.upserts++
	if len(f.upsertErr) > 0 {
		err := f.upsertErr[0]
		f.upsertErr = f.upsertErr[1:]
		if err != nil {
			return RemoteRef{}, err
		}
	}
	rid := f.refID
	if rid == "" {
		rid = "501"
	}
	return RemoteRef{Provider: "fake", ObjectType: "contact", ID: rid}, nil
}

func (f *fakeProvider) LogActivity(_ context.Context, _ RemoteRef, a *Activity) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.activity = append(f.activity, a)
	if len(f.activityErr) > 0 {
		err := f.activityErr[0]
		f.activityErr = f.activityErr[1:]
		if err != nil {
			return err
		}
	}
	return nil
}

func newTestWorker(t *testing.T, s Store, p Provider) *worker {
	t.Helper()
	return newWorker(workerDeps{
		Store:       s,
		Providers:   map[string]Provider{"fake": p},
		Logger:      log.NewNoopLogger(),
		Lease:       time.Minute,
		BatchSize:   10,
		MaxAttempts: 3,
		BaseBackoff: time.Second,
		LoadContact: func(_ context.Context, j *Job) (*Contact, error) {
			return &Contact{UserID: j.UserID, AppID: j.AppID, Email: "a@example.com"}, nil
		},
	})
}

func enqueued(t *testing.T, s Store, kind, key string) *Job {
	t.Helper()
	j := &Job{
		ID: id.NewRetentionJobID(), AppID: id.NewAppID(), UserID: id.NewUserID(),
		Provider: "fake", Kind: kind, IdempotencyKey: key,
		Payload:       map[string]string{"activity_type": "logged_in"},
		State:         StatePending,
		NextAttemptAt: time.Now().Add(-time.Second),
		CreatedAt:     time.Now(),
	}
	require.NoError(t, s.Enqueue(context.Background(), j))
	return j
}

func TestWorkerUpsertsContactAndStoresRef(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "k1")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDone, stored.State)

	ref, err := s.GetRef(ctx, j.AppID, j.EnvID, j.UserID, "fake")
	require.NoError(t, err)
	assert.Equal(t, "501", ref.RemoteID)
}

func TestWorkerActivityUpsertsFirstWhenRefMissing(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindActivityLog, "k2")

	newTestWorker(t, s, p).runOnce(ctx)

	assert.Equal(t, 1, p.upserts, "a missing ref must be created inside the same job")
	require.Len(t, p.activity, 1)
	assert.Equal(t, "logged_in", p.activity[0].Type)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDone, stored.State)
}

func TestWorkerSuppressesWhenProviderCannotHoldContacts(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapActivities} // no CapContacts
	j := enqueued(t, s, KindContactUpsert, "k3a")

	newTestWorker(t, s, p).runOnce(ctx)

	assert.Zero(t, p.upserts, "a provider without CapContacts must not be called")
	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateSuppressed, stored.State,
		"a statically-impossible delivery is suppressed, not dead-lettered: "+
			"dead means we tried and failed, and we never tried")
}

func TestWorkerSkipsActivityWhenProviderLacksCapability(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts} // no CapActivities
	j := enqueued(t, s, KindActivityLog, "k3")

	newTestWorker(t, s, p).runOnce(ctx)

	assert.Empty(t, p.activity, "a provider without CapActivities must not be called")
	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateSuppressed, stored.State,
		"an unsupported activity is suppressed, not failed and not silently done")
}

func TestWorkerRetriesRetryableErrorWithBackoff(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{
		caps:      CapContacts | CapActivities,
		upsertErr: []error{&ProviderError{Err: errors.New("503"), Retryable: true}},
	}
	j := enqueued(t, s, KindContactUpsert, "k4")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State)
	assert.Equal(t, 1, stored.Attempts)
	assert.True(t, stored.NextAttemptAt.After(time.Now()), "the retry must be deferred")
}

func TestWorkerHonoursRetryAfter(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{
		caps:      CapContacts | CapActivities,
		upsertErr: []error{&ProviderError{Err: errors.New("429"), Retryable: true, RetryAfter: time.Hour}},
	}
	j := enqueued(t, s, KindContactUpsert, "k5")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.True(t, stored.NextAttemptAt.After(time.Now().Add(50*time.Minute)),
		"RetryAfter must win over the computed backoff")
}

func TestWorkerDeadLettersTerminalError(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{
		caps:      CapContacts | CapActivities,
		upsertErr: []error{errors.New("400 invalid email")},
	}
	j := enqueued(t, s, KindContactUpsert, "k6")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDead, stored.State, "an unclassified error is terminal")
	assert.Contains(t, stored.LastError, "400")
}

// The name is about the attempt budget, not about how many deliveries this
// test performs: the forced MarkRetry calls below inflate the attempt count
// on purpose, so the job reaches the budget sooner than the number of
// provider calls suggests.
func TestWorkerDeadLettersOnceTheAttemptBudgetIsSpent(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	retry := func() error { return &ProviderError{Err: errors.New("503"), Retryable: true} }
	p := &fakeProvider{
		caps:      CapContacts | CapActivities,
		upsertErr: []error{retry(), retry(), retry()},
	}
	j := enqueued(t, s, KindContactUpsert, "k7")
	w := newTestWorker(t, s, p)

	for i := 0; i < 3; i++ {
		// Force the job due again without waiting out the backoff.
		require.NoError(t, s.MarkRetry(ctx, j.ID, time.Now().Add(-time.Second), "forced due"))
		w.runOnce(ctx)
	}

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDead, stored.State, "MaxAttempts must eventually stop the retries")
}

func TestWorkerSuppressesWhenAllowSendSaysNo(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "k8")

	w := newTestWorker(t, s, p)
	w.deps.AllowSend = func(_ context.Context, _ *Job) (bool, string, error) {
		return false, "no marketing consent", nil
	}
	w.runOnce(ctx)

	assert.Zero(t, p.upserts, "a suppressed job must never reach the provider")
	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateSuppressed, stored.State)
	assert.Equal(t, "no marketing consent", stored.LastError)
}

func TestWorkerUnknownProviderIsDeadLettered(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID := id.NewAppID()
	j := &Job{
		ID: id.NewRetentionJobID(), AppID: appID, UserID: id.NewUserID(),
		Provider: "gone", Kind: KindContactUpsert, IdempotencyKey: "k9",
		State: StatePending, NextAttemptAt: time.Now().Add(-time.Second), CreatedAt: time.Now(),
	}
	require.NoError(t, s.Enqueue(ctx, j))

	newTestWorker(t, s, &fakeProvider{caps: CapContacts}).runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDead, stored.State,
		"a job naming a provider that is gone must not spin forever")

	dead, err := s.ListDead(ctx, appID, 10)
	require.NoError(t, err)
	assert.Len(t, dead, 1)
}

func TestWorkerTickDrivesRunOnce(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "k11")

	w := newTestWorker(t, s, p)
	tick := make(chan time.Time)
	w.startWith(tick)
	t.Cleanup(w.stop)

	tick <- time.Now()
	require.Eventually(t, func() bool {
		stored, err := s.GetJob(ctx, j.ID)
		return err == nil && stored.State == StateDone
	}, 2*time.Second, 10*time.Millisecond)
}
