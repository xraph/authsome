package retention

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// These cover the three ways a delivery can fail that are not the CRM's
// verdict: our own store, our own engine, and a provider that panics. All of
// them used to end with a dead row, and nothing in this plugin re-enqueues a
// dead row.

// refFailingStore fails one ref method and delegates everything else, so a
// test can drop a database blip into the middle of a delivery.
type refFailingStore struct {
	Store
	getErr error
	putErr error
}

func (s refFailingStore) GetRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) (*ContactRef, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.Store.GetRef(ctx, appID, envID, userID, provider)
}

func (s refFailingStore) PutRef(ctx context.Context, r *ContactRef) error {
	if s.putErr != nil {
		return s.putErr
	}
	return s.Store.PutRef(ctx, r)
}

func TestWorkerRetriesWhenTheRefLookupFails(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryStore()
	s := refFailingStore{Store: mem, getErr: errors.New("connection reset by peer")}
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, mem, KindContactUpsert, "local1")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := mem.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State,
		"a failover in our own database must not permanently destroy the job")
	assert.Equal(t, 1, stored.Attempts, "it spends one attempt, like any other retry")
}

func TestWorkerRetriesWhenTheRefWriteFails(t *testing.T) {
	ctx := context.Background()
	mem := NewMemoryStore()
	s := refFailingStore{Store: mem, putErr: errors.New("connection reset by peer")}
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, mem, KindContactUpsert, "local2")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := mem.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State,
		"the contact already exists in the CRM and the ref row does not, so giving "+
			"up here would let the next login create a second contact")
}

func TestWorkerRetriesWhenLoadingTheUserFails(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "local3")

	w := newTestWorker(t, s, p)
	w.deps.LoadContact = func(_ context.Context, _ *Job) (*Contact, error) {
		return nil, localError("retention: load user", errors.New("db down"))
	}
	w.runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State)
	assert.Zero(t, p.upserts, "nothing reached the CRM")
}

func TestWorkerRetriesWhenTheConsentLookupFails(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "local4")

	w := newTestWorker(t, s, p)
	w.deps.AllowSend = func(_ context.Context, _ *Job) (bool, string, error) {
		return false, "", localError("retention: consent lookup", errors.New("consent store down"))
	}
	w.runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State,
		"a lookup we could not complete is not a choice we made, so it is not suppressed")
}

func TestWorkerStillDeadLettersAnUnclassifiedCRMError(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{
		caps:      CapContacts | CapActivities,
		upsertErr: []error{errors.New("418 teapot")},
	}
	j := enqueued(t, s, KindContactUpsert, "local5")

	newTestWorker(t, s, p).runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDead, stored.State,
		"tagging our own failures as local must not soften the CRM's verdicts")
}

// ──────────────────────────────────────────────────
// retention.enabled
// ──────────────────────────────────────────────────

func TestWorkerDefersWhenDeliveryIsDisabledForTheApp(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "off1")

	w := newTestWorker(t, s, p)
	w.deps.Enabled = func(context.Context, *Job) bool { return false }
	w.runOnce(ctx)

	assert.Zero(t, p.upserts, "the operator turned delivery off")
	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StatePending, stored.State,
		"the work has to still be there when they turn it back on")
	assert.Zero(t, stored.Attempts, "nothing failed, so nothing may be charged to the budget")
	assert.True(t, stored.NextAttemptAt.After(time.Now()), "and it is not re-claimed immediately")
}

func TestWorkerDeliversAgainOnceDeliveryIsReEnabled(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{caps: CapContacts | CapActivities}
	j := enqueued(t, s, KindContactUpsert, "off2")

	w := newTestWorker(t, s, p)
	enabled := false
	w.deps.Enabled = func(context.Context, *Job) bool { return enabled }
	w.runOnce(ctx)

	enabled = true
	// Force the deferred job due again rather than waiting out the interval.
	require.NoError(t, s.MarkDeferred(ctx, j.ID, time.Now().Add(-time.Second), ""))
	w.runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDone, stored.State, "the backlog drains once the switch goes back on")
	assert.Zero(t, stored.Attempts, "the disabled window cost it nothing")
}

func TestDeliveryEnabledDefaultsToOnWithoutASettingsManager(t *testing.T) {
	// A plugin whose OnInit ran under an engine with no settings manager has
	// a nil enabledPolicy. A feature gate nobody can read must not silently
	// stop a working integration.
	assert.True(t, New().deliveryEnabled(context.Background(), &Job{}))
}

// ──────────────────────────────────────────────────
// A panicking provider must not take the process down
// ──────────────────────────────────────────────────

// panickingProvider panics on its first upsert and behaves afterwards, so one
// test can assert both halves: the panicking job is parked, and the rest of
// the batch is still delivered.
type panickingProvider struct {
	calls int
}

func (p *panickingProvider) Name() string             { return "fake" }
func (p *panickingProvider) Capabilities() Capability { return CapContacts | CapActivities }

func (p *panickingProvider) UpsertContact(context.Context, *Contact) (RemoteRef, error) {
	p.calls++
	if p.calls == 1 {
		panic("classifyHTTPError: policy not yet decided")
	}
	return RemoteRef{Provider: "fake", ObjectType: "contact", ID: "501"}, nil
}

func (p *panickingProvider) LogActivity(context.Context, RemoteRef, *Activity) error { return nil }

func TestWorkerRecoversFromAPanickingProviderAndFinishesTheBatch(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	appID := id.NewAppID()
	now := time.Now()

	first := &Job{
		ID: id.NewRetentionJobID(), AppID: appID, UserID: id.NewUserID(),
		Provider: "fake", Kind: KindContactUpsert, IdempotencyKey: "panic1",
		Payload: map[string]string{}, State: StatePending,
		NextAttemptAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour),
	}
	second := &Job{
		ID: id.NewRetentionJobID(), AppID: appID, UserID: id.NewUserID(),
		Provider: "fake", Kind: KindContactUpsert, IdempotencyKey: "panic2",
		Payload: map[string]string{}, State: StatePending,
		NextAttemptAt: now.Add(-time.Minute), CreatedAt: now.Add(-time.Minute),
	}
	require.NoError(t, s.Enqueue(ctx, first))
	require.NoError(t, s.Enqueue(ctx, second))

	// No recover in the test itself: if the worker ever stops swallowing
	// this, the panic unwinds through runOnce and fails the test loudly.
	// In production it would take the auth process with it.
	newTestWorker(t, s, &panickingProvider{}).runOnce(ctx)

	dead, err := s.GetJob(ctx, first.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDead, dead.State, "a panic is a bug, not a transient")
	assert.Contains(t, dead.LastError, "panic")

	done, err := s.GetJob(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, StateDone, done.State, "one bad job must not cost the rest of the batch")
}
