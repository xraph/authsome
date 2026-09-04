package retention

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime/debug"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
)

// workerDeps is everything the delivery loop needs. It takes function values
// for the three things that would otherwise drag the engine into this file,
// which also makes all of them trivially fakeable in tests.
type workerDeps struct {
	Store     Store
	Providers map[string]Provider
	Logger    log.Logger

	Interval    time.Duration
	Lease       time.Duration
	BatchSize   int
	MaxAttempts int
	BaseBackoff time.Duration

	// DoneRetention, AuditRetention and PurgeInterval drive the outbox
	// sweep. See Config in plugin.go for what the numbers mean and why they
	// are what they are. A negative retention keeps that class forever; a
	// non-positive PurgeInterval switches the sweep off altogether.
	DoneRetention  time.Duration
	AuditRetention time.Duration
	PurgeInterval  time.Duration

	// LoadContact reloads the user at delivery time. The worker does not trust
	// the enqueued payload: anything evaluated on the near side of a queue is
	// a snapshot of a fact that may have moved, and the email address is as
	// mutable as the consent grant.
	LoadContact func(ctx context.Context, j *Job) (*Contact, error)

	// Enabled reports whether retention.enabled is on for the job's app. Nil
	// means enabled, which is what a plugin with no settings manager gets.
	//
	// It is resolved per job rather than once at startup because
	// retention.enabled is ScopeApp: one process serves apps that disagree
	// about it, and it is a dynamic setting, so flipping it has to bite
	// without a restart.
	Enabled func(ctx context.Context, j *Job) bool

	// AllowSend is the consent gate. Nil means always allow. It runs here and
	// not at enqueue so a revocation between login and delivery is honoured.
	//
	// The error return is what separates "we asked and the answer was no"
	// from "we could not ask". Only the first is a deliberate choice, and
	// only a deliberate choice may be recorded as suppressed.
	AllowSend func(ctx context.Context, j *Job) (allowed bool, reason string, err error)
}

// errNoContactCapability means the provider cannot hold contacts at all, so
// this job was never deliverable. deliver routes it to suppressed rather than
// dead, because the provider declared the limitation up front.
var errNoContactCapability = errors.New("retention: provider does not support contacts")

// errLocal marks a failure in our own infrastructure - the store, the engine,
// the settings manager - rather than a verdict from the CRM. Terminal-by-
// default is right for an unclassified CRM response, and wrong for our own
// database: a failover would otherwise dead-letter every job claimed in that
// window, and nothing re-enqueues a dead row.
var errLocal = errors.New("retention: local infrastructure failure")

// localError tags err as ours rather than the CRM's. Both verbs are %w, so
// errors.Is still sees errLocal and everything err already wrapped.
func localError(where string, err error) error {
	return fmt.Errorf("%s: %w: %w", where, errLocal, err)
}

// terminal decides whether a failure should stop the job for good.
//
// An unclassified response from the CRM is terminal on purpose: retried
// forever it is a queue that never drains, and the dead-letter row is the
// thing that gets it looked at. A failure of our own infrastructure is not,
// because a database blip would otherwise destroy work permanently, and
// nothing in this plugin re-enqueues a dead row.
func terminal(err error) bool {
	return !isRetryable(err) && !errors.Is(err, errLocal)
}

type worker struct {
	deps   workerDeps
	ticker *time.Ticker
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once

	// lastPurge gates the outbox sweep down to PurgeInterval while it rides
	// the delivery ticker. Touched only from the run goroutine (and from
	// tests, which drive maybePurge directly and never start the loop), so
	// it needs no lock.
	lastPurge time.Time
}

func newWorker(deps workerDeps) *worker {
	if deps.Interval <= 0 {
		deps.Interval = 30 * time.Second
	}
	if deps.BatchSize <= 0 {
		deps.BatchSize = 50
	}
	if deps.MaxAttempts <= 0 {
		deps.MaxAttempts = 12
	}
	if deps.BaseBackoff <= 0 {
		deps.BaseBackoff = 5 * time.Second
	}
	if deps.Lease <= 0 {
		deps.Lease = 2 * time.Minute
	}
	if deps.DoneRetention == 0 {
		deps.DoneRetention = 30 * 24 * time.Hour
	}
	if deps.AuditRetention == 0 {
		deps.AuditRetention = 180 * 24 * time.Hour
	}
	if deps.PurgeInterval == 0 {
		deps.PurgeInterval = time.Hour
	}
	if deps.Logger == nil {
		deps.Logger = log.NewNoopLogger()
	}
	// lastPurge starts at now, not at the zero time, so the first sweep
	// happens one PurgeInterval into the process rather than on the first
	// tick. A fleet that restarts often would otherwise run a full-table
	// sweep on every rollout, which is the one moment you least want the
	// database doing extra work.
	return &worker{deps: deps, done: make(chan struct{}), lastPurge: time.Now()}
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

// runOnce claims one batch and delivers it, sweeping expired terminal rows
// first on the ticks where that is due.
func (w *worker) runOnce(ctx context.Context) {
	now := time.Now()
	w.maybePurge(ctx, now)
	jobs, err := w.deps.Store.ClaimDue(ctx, w.deps.BatchSize, w.deps.Lease, now)
	if err != nil {
		w.deps.Logger.Warn("retention: claim failed", log.String("error", err.Error()))
		return
	}
	for _, j := range jobs {
		w.deliver(ctx, j)
	}
}

// maybePurge sweeps expired terminal rows, at most once per PurgeInterval.
//
// It rides the delivery ticker rather than taking a goroutine of its own.
// A second goroutine would need its own lifecycle, its own stop, and its
// own answer for what happens when it overlaps a delivery round; sharing
// this one costs a comparison per tick and inherits all of that for free.
// The gate is what keeps a sweep off the other 119 ticks in the hour.
//
// A sweep that finds a very large backlog holds this goroutine for its
// duration and delays that round's deliveries. That is accepted rather than
// batched: after the first sweep the working set is one interval's worth of
// terminal rows, and the queue it delays is already asynchronous by design.
//
// Errors are logged, never escalated. Failing to prune is a disk problem
// somebody can look at tomorrow; refusing to deliver because pruning failed
// would turn it into a sync outage today.
func (w *worker) maybePurge(ctx context.Context, now time.Time) {
	if w.deps.PurgeInterval <= 0 || now.Sub(w.lastPurge) < w.deps.PurgeInterval {
		return
	}
	w.lastPurge = now

	// A zero cutoff reaches nothing, which is how a negative retention
	// setting means "keep this class forever". See purgeClasses.
	var doneBefore, auditBefore time.Time
	if w.deps.DoneRetention > 0 {
		doneBefore = now.Add(-w.deps.DoneRetention)
	}
	if w.deps.AuditRetention > 0 {
		auditBefore = now.Add(-w.deps.AuditRetention)
	}
	if doneBefore.IsZero() && auditBefore.IsZero() {
		return
	}

	removed, err := w.deps.Store.PurgeTerminal(ctx, doneBefore, auditBefore)
	if err != nil {
		w.deps.Logger.Warn("retention: outbox purge failed",
			log.String("error", err.Error()))
		return
	}
	if removed > 0 {
		w.deps.Logger.Info("retention: purged expired outbox rows",
			log.Int("removed", removed))
	}
}

// deliver runs one job with a recover around it, so that no provider, present
// or future, can take the auth process down.
//
// A panic on this goroutine kills the whole binary, and the outbox exists
// precisely so a misbehaving CRM integration cannot reach the login path. The
// job is dead-lettered rather than retried: a panic is a bug, not a
// transient, and running the same bug again on the same row panics again.
// The rest of the batch carries on.
func (w *worker) deliver(ctx context.Context, j *Job) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		w.deps.Logger.Error("retention: delivery panicked, job dead-lettered",
			log.String("job_id", j.ID.String()),
			log.String("provider", j.Provider),
			log.String("panic", fmt.Sprint(r)),
			log.String("stack", string(debug.Stack())))
		if err := w.deps.Store.MarkDead(ctx, j.ID,
			fmt.Sprintf("panic during delivery: %v", r)); err != nil {
			w.deps.Logger.Warn("retention: mark dead failed", log.String("error", err.Error()))
		}
	}()
	w.deliverOne(ctx, j)
}

func (w *worker) deliverOne(ctx context.Context, j *Job) {
	if w.deps.Enabled != nil && !w.deps.Enabled(ctx, j) {
		// Neither dead nor done. The operator turned delivery off, most
		// likely mid-incident, and the work should still be there when they
		// turn it back on: "dead" would claim we tried and failed, "done"
		// would claim we delivered, and nothing in this plugin re-enqueues
		// either. Deferring spends no attempt budget because nothing failed.
		// Spending one would mean a long enough disable quietly burns the
		// whole budget, and the first real error after re-enabling kills a
		// job that never actually failed.
		w.deferJob(ctx, j, "delivery disabled for this app (retention.enabled)")
		return
	}

	p, ok := w.deps.Providers[j.Provider]
	if !ok {
		// Dead-letter rather than retry. A provider that is not configured
		// will not appear because we waited, and a job that retries forever
		// is a queue that never drains.
		w.fail(ctx, j, fmt.Errorf("provider %q is not configured", j.Provider), true)
		return
	}

	if w.deps.AllowSend != nil {
		allowed, reason, err := w.deps.AllowSend(ctx, j)
		switch {
		case err != nil:
			// We could not ask, so we have not chosen anything. Recording
			// that as suppressed would write "we deliberately did not send
			// this" into the audit trail for a lookup that never completed.
			w.fail(ctx, j, err, terminal(err))
			return
		case !allowed:
			w.suppress(ctx, j, reason)
			return
		}
	}

	contact, err := w.deps.LoadContact(ctx, j)
	if err != nil {
		w.fail(ctx, j, err, terminal(err))
		return
	}

	ref, hadRef, err := w.ensureRef(ctx, p, j, contact)
	if err != nil {
		if errors.Is(err, errNoContactCapability) {
			// Symmetric with the CapActivities check below. The provider told
			// us up front it cannot do this, so the right record is a
			// deliberate skip, not a failed delivery. Dead-lettering here
			// would put "we tried and failed" in the audit trail for
			// something we never attempted.
			w.suppress(ctx, j, "provider does not support contacts")
			return
		}
		w.dropRefIfNeeded(ctx, j, hadRef, err)
		w.fail(ctx, j, err, terminal(err))
		return
	}

	if j.Kind == KindActivityLog {
		if !p.Capabilities().Has(CapActivities) {
			// The provider told us up front it cannot do this. Recording it as
			// suppressed keeps "we did not send" distinct from "we failed".
			w.suppress(ctx, j, "provider does not support activities")
			return
		}
		// Delivery is at-least-once, and the contact ref that makes a
		// repeated upsert harmless has no counterpart for activities. So
		// the job's own idempotency key travels with the activity, and a
		// provider that can key on it says so with CapActivityDedupe.
		activity := &Activity{
			Type:       j.Payload["activity_type"],
			OccurredAt: j.CreatedAt,
			Properties: j.Payload,
			ExternalID: j.IdempotencyKey,
			Redelivery: j.Redelivered(),
		}
		if activity.Redelivery && !p.Capabilities().Has(CapActivityDedupe) {
			// Said out loud rather than swallowed. The provider cannot
			// promise the CRM will end up with one entry, and a silent
			// retry here is how somebody finds three "logged in" notes on a
			// contact and has no idea where they came from.
			w.deps.Logger.Warn(
				"retention: redelivering an activity to a provider that cannot deduplicate; "+
					"the CRM may end up with a duplicate entry",
				log.String("job_id", j.ID.String()),
				log.String("provider", j.Provider),
				log.String("activity_type", activity.Type),
				log.Int("attempts", j.Attempts),
				log.Bool("reclaimed", j.Reclaimed))
		}
		if err := p.LogActivity(ctx, ref, activity); err != nil {
			w.dropRefIfNeeded(ctx, j, hadRef, err)
			w.fail(ctx, j, err, terminal(err))
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
//
// The second return, hadRef, tells the caller whether a ref already existed
// in the store when this attempt started, as opposed to being created just
// now. That distinction is what lets dropRefIfNeeded tell a stale reference
// (drop it) from a transient failure on a brand-new create (nothing to
// orphan, so leave it alone).
func (w *worker) ensureRef(ctx context.Context, p Provider, j *Job, c *Contact) (RemoteRef, bool, error) {
	existing, err := w.deps.Store.GetRef(ctx, j.AppID, j.EnvID, j.UserID, j.Provider)
	hadRef := err == nil
	switch {
	case err == nil && j.Kind != KindContactUpsert:
		return existing.Ref(), hadRef, nil
	case err != nil && !errors.Is(err, ErrNotFound):
		// Our store, not the CRM. Retry inside the normal budget instead of
		// dead-lettering a whole claimed batch because the database blinked.
		return RemoteRef{}, false, localError("retention: read contact ref", err)
	}

	if !p.Capabilities().Has(CapContacts) {
		return RemoteRef{}, hadRef, errNoContactCapability
	}
	ref, err := p.UpsertContact(ctx, c)
	if err != nil {
		return RemoteRef{}, hadRef, err
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
		// This one is worse than a lost job if we give up: the contact
		// already exists in the CRM and the ref row does not, so the next
		// login creates a second contact and the dedup spine is defeated.
		return RemoteRef{}, hadRef, localError("retention: write contact ref", err)
	}
	return ref, hadRef, nil
}

// dropRefIfNeeded deletes the local contact ref when the CRM reports the
// record is gone (ProviderError.DropRef) and this attempt actually held a
// ref going in. hadRef == false means either there was nothing to drop, or
// the ref that just failed is the one this very attempt was about to create
// — a transient 404 there must not orphan a ref that does not exist yet. A
// failed delete is logged, not escalated: it is our own store, and the job
// is about to be retried anyway.
func (w *worker) dropRefIfNeeded(ctx context.Context, j *Job, hadRef bool, err error) {
	if !hadRef {
		return
	}
	var pe *ProviderError
	if !errors.As(err, &pe) || !pe.DropRef {
		return
	}
	if delErr := w.deps.Store.DeleteRef(ctx, j.AppID, j.EnvID, j.UserID, j.Provider); delErr != nil {
		w.deps.Logger.Warn("retention: drop ref failed",
			log.String("job_id", j.ID.String()),
			log.String("provider", j.Provider),
			log.String("error", delErr.Error()))
	}
}

func isRetryable(err error) bool {
	ok, _ := Retryable(err)
	return ok
}

// fail either defers the job or parks it. isTerminal short-circuits the retry
// budget for errors that will never succeed.
func (w *worker) fail(ctx context.Context, j *Job, cause error, isTerminal bool) {
	if isTerminal || j.Attempts+1 >= w.deps.MaxAttempts {
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

// deferJob puts a claimed job back without spending an attempt, for the case
// where nothing failed and nothing was decided: we are simply not delivering
// right now. It comes back up roughly one tick later.
func (w *worker) deferJob(ctx context.Context, j *Job, reason string) {
	if err := w.deps.Store.MarkDeferred(ctx, j.ID, time.Now().Add(w.deps.Interval), reason); err != nil {
		w.deps.Logger.Warn("retention: mark deferred failed", log.String("error", err.Error()))
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
