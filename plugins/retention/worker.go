package retention

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
)

// workerDeps is everything the delivery loop needs. It takes function values
// for the two things that would otherwise drag the engine into this file,
// which also makes both trivially fakeable in tests.
type workerDeps struct {
	Store     Store
	Providers map[string]Provider
	Logger    log.Logger

	Interval    time.Duration
	Lease       time.Duration
	BatchSize   int
	MaxAttempts int
	BaseBackoff time.Duration

	// LoadContact reloads the user at delivery time. The worker does not trust
	// the enqueued payload: anything evaluated on the near side of a queue is
	// a snapshot of a fact that may have moved, and the email address is as
	// mutable as the consent grant.
	LoadContact func(ctx context.Context, j *Job) (*Contact, error)

	// AllowSend is the consent gate. Nil means always allow. It runs here and
	// not at enqueue so a revocation between login and delivery is honoured.
	AllowSend func(ctx context.Context, j *Job) (bool, string)
}

type worker struct {
	deps   workerDeps
	ticker *time.Ticker
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func newWorker(deps workerDeps) *worker {
	if deps.Interval <= 0 {
		deps.Interval = 30 * time.Second
	}
	if deps.BatchSize <= 0 {
		deps.BatchSize = 50
	}
	if deps.MaxAttempts <= 0 {
		deps.MaxAttempts = 8
	}
	if deps.BaseBackoff <= 0 {
		deps.BaseBackoff = 5 * time.Second
	}
	if deps.Lease <= 0 {
		deps.Lease = 2 * time.Minute
	}
	if deps.Logger == nil {
		deps.Logger = log.NewNoopLogger()
	}
	return &worker{deps: deps, done: make(chan struct{})}
}

// start begins the loop on its own ticker.
func (w *worker) start() {
	w.ticker = time.NewTicker(w.deps.Interval)
	w.startWith(w.ticker.C)
}

// startWith begins the loop on a caller-supplied tick channel, so tests drive
// it deterministically instead of sleeping.
func (w *worker) startWith(tick <-chan time.Time) {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	go w.run(ctx, tick)
}

func (w *worker) run(ctx context.Context, tick <-chan time.Time) {
	defer close(w.done)
	for {
		select {
		case <-tick:
			w.runOnce(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// stop halts the loop and waits for a round already in flight, so the process
// does not exit with a CRM request still open. Safe to call more than once and
// safe on a worker that never started.
func (w *worker) stop() {
	w.once.Do(func() {
		if w.cancel != nil {
			w.cancel()
			<-w.done
		}
		if w.ticker != nil {
			w.ticker.Stop()
		}
	})
}

// runOnce claims one batch and delivers it.
func (w *worker) runOnce(ctx context.Context) {
	now := time.Now()
	jobs, err := w.deps.Store.ClaimDue(ctx, w.deps.BatchSize, w.deps.Lease, now)
	if err != nil {
		w.deps.Logger.Warn("retention: claim failed", log.String("error", err.Error()))
		return
	}
	for _, j := range jobs {
		w.deliver(ctx, j)
	}
}

func (w *worker) deliver(ctx context.Context, j *Job) {
	p, ok := w.deps.Providers[j.Provider]
	if !ok {
		// Dead-letter rather than retry. A provider that is not configured
		// will not appear because we waited, and a job that retries forever
		// is a queue that never drains.
		w.fail(ctx, j, fmt.Errorf("provider %q is not configured", j.Provider), true)
		return
	}

	if w.deps.AllowSend != nil {
		allowed, reason := w.deps.AllowSend(ctx, j)
		if !allowed {
			w.suppress(ctx, j, reason)
			return
		}
	}

	contact, err := w.deps.LoadContact(ctx, j)
	if err != nil {
		w.fail(ctx, j, err, !isRetryable(err))
		return
	}

	ref, err := w.ensureRef(ctx, p, j, contact)
	if err != nil {
		w.fail(ctx, j, err, !isRetryable(err))
		return
	}

	if j.Kind == KindActivityLog {
		if !p.Capabilities().Has(CapActivities) {
			// The provider told us up front it cannot do this. Recording it as
			// suppressed keeps "we did not send" distinct from "we failed".
			w.suppress(ctx, j, "provider does not support activities")
			return
		}
		activity := &Activity{
			Type:       j.Payload["activity_type"],
			OccurredAt: j.CreatedAt,
			Properties: j.Payload,
		}
		if err := p.LogActivity(ctx, ref, activity); err != nil {
			w.fail(ctx, j, err, !isRetryable(err))
			return
		}
	}

	if err := w.deps.Store.MarkDone(ctx, j.ID, time.Now()); err != nil {
		w.deps.Logger.Warn("retention: mark done failed", log.String("error", err.Error()))
	}
}

// ensureRef returns the contact's ref, creating the contact when we have not
// seen it before. This is why a sign-in hook does not need to check for a ref:
// the worker heals it here, so a contact deleted upstream or a provider
// enabled after the user existed both recover on the next login.
func (w *worker) ensureRef(ctx context.Context, p Provider, j *Job, c *Contact) (RemoteRef, error) {
	existing, err := w.deps.Store.GetRef(ctx, j.AppID, j.EnvID, j.UserID, j.Provider)
	switch {
	case err == nil && j.Kind != KindContactUpsert:
		return existing.Ref(), nil
	case err != nil && !errors.Is(err, ErrNotFound):
		return RemoteRef{}, err
	}

	if !p.Capabilities().Has(CapContacts) {
		return RemoteRef{}, fmt.Errorf("provider %q cannot upsert contacts", j.Provider)
	}
	ref, err := p.UpsertContact(ctx, c)
	if err != nil {
		return RemoteRef{}, err
	}
	row := &ContactRef{
		ID: id.NewRetentionRefID(), AppID: j.AppID, EnvID: j.EnvID, UserID: j.UserID,
		Provider: j.Provider, RemoteObjectType: ref.ObjectType, RemoteID: ref.ID,
		SyncedAt: time.Now(),
	}
	if existing != nil {
		row.ID = existing.ID
	}
	if err := w.deps.Store.PutRef(ctx, row); err != nil {
		return RemoteRef{}, err
	}
	return ref, nil
}

func isRetryable(err error) bool {
	ok, _ := Retryable(err)
	return ok
}

// fail either defers the job or parks it. terminal short-circuits the retry
// budget for errors that will never succeed.
func (w *worker) fail(ctx context.Context, j *Job, cause error, terminal bool) {
	if terminal || j.Attempts+1 >= w.deps.MaxAttempts {
		if err := w.deps.Store.MarkDead(ctx, j.ID, cause.Error()); err != nil {
			w.deps.Logger.Warn("retention: mark dead failed", log.String("error", err.Error()))
		}
		w.deps.Logger.Warn("retention: job dead-lettered",
			log.String("job_id", j.ID.String()),
			log.String("provider", j.Provider),
			log.String("error", cause.Error()))
		return
	}
	_, after := Retryable(cause)
	if after <= 0 {
		after = w.backoff(j.Attempts)
	}
	if err := w.deps.Store.MarkRetry(ctx, j.ID, time.Now().Add(after), cause.Error()); err != nil {
		w.deps.Logger.Warn("retention: mark retry failed", log.String("error", err.Error()))
	}
}

func (w *worker) suppress(ctx context.Context, j *Job, reason string) {
	if err := w.deps.Store.MarkSuppressed(ctx, j.ID, reason); err != nil {
		w.deps.Logger.Warn("retention: mark suppressed failed", log.String("error", err.Error()))
	}
}

// backoff is exponential on the attempt count, capped so a long-dead CRM does
// not push the next attempt past the point anyone is still watching.
func (w *worker) backoff(attempts int) time.Duration {
	const maxBackoff = 30 * time.Minute
	d := time.Duration(float64(w.deps.BaseBackoff) * math.Pow(2, float64(attempts)))
	if d > maxBackoff || d <= 0 {
		return maxBackoff
	}
	return d
}
