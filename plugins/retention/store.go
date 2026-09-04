package retention

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when a row does not exist. Every store maps its
// driver's own miss onto this, so callers never branch on a driver error.
var ErrNotFound = errors.New("retention: not found")

// Outbox job states.
//
// Suppressed is deliberately distinct from dead. "We chose not to send this"
// and "we tried and failed" are different answers when somebody audits you,
// and collapsing them loses the only record that a consent gate did its job.
const (
	StatePending    = "pending"
	StateInFlight   = "in_flight"
	StateDone       = "done"
	StateDead       = "dead"
	StateSuppressed = "suppressed"
)

// Outbox job kinds.
const (
	KindContactUpsert = "contact_upsert"
	KindActivityLog   = "activity_log"
)

// purgeClass pairs one terminal state with the cutoff that governs it.
type purgeClass struct {
	State  string
	Before time.Time
}

// purgeClasses is the single definition of what PurgeTerminal sweeps, so all
// four backends delete the same states under the same cutoffs and none of
// them can quietly forget one. Non-terminal states are absent on purpose and
// must stay absent: see PurgeTerminal on Store.
//
// A class whose Before is zero deletes nothing, because no row was created
// before year one. That is what turns a non-positive retention setting into
// "keep everything" without any backend special-casing it.
func purgeClasses(doneBefore, auditBefore time.Time) []purgeClass {
	return []purgeClass{
		{State: StateDone, Before: doneBefore},
		{State: StateDead, Before: auditBefore},
		{State: StateSuppressed, Before: auditBefore},
	}
}

// Job is one unit of pending delivery work.
type Job struct {
	ID             id.RetentionJobID `json:"id"`
	AppID          id.AppID          `json:"app_id"`
	EnvID          id.EnvironmentID  `json:"env_id"`
	UserID         id.UserID         `json:"user_id"`
	Provider       string            `json:"provider"`
	Kind           string            `json:"kind"`
	Payload        map[string]string `json:"payload"`
	IdempotencyKey string            `json:"idempotency_key"`
	State          string            `json:"state"`
	Attempts       int               `json:"attempts"`
	NextAttemptAt  time.Time         `json:"next_attempt_at"`
	InFlightUntil  time.Time         `json:"in_flight_until"`
	LastError      string            `json:"last_error,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`

	// Reclaimed is set by ClaimDue when this job was taken from an expired
	// lease rather than from pending. It is computed at claim time from the
	// row's state before the claim, and it is not a column: GetJob and
	// ListDead always report false.
	//
	// It exists because Attempts is blind to exactly the case that matters.
	// A delivery whose provider call succeeded and whose MarkDone then
	// failed leaves the row in_flight with Attempts unchanged, because
	// nothing got to record a failure. The lease expires, the row comes
	// back, and the only trace that it already went out once is that the
	// claim matched the expired-lease clause instead of the pending one.
	//
	// It cannot be recorded any earlier either. The failure it detects is a
	// store outage, and any marker we would write to warn ourselves goes to
	// the same store that just refused the write. The claim is the first
	// moment anything can see it.
	Reclaimed bool `json:"-"`
}

// Redelivered reports whether this job may already have reached the provider
// once, so a delivery that is not naturally idempotent should check before
// repeating itself.
//
// Two ways in. Attempts > 0 means a previous attempt reported an error, and
// an error is no proof the CRM did nothing: a request that succeeded and
// whose response we failed to read looks identical to one that never landed.
// Reclaimed means the previous attempt never reported anything at all.
func (j *Job) Redelivered() bool { return j.Reclaimed || j.Attempts > 0 }

// ContactRef records where a user landed in a given CRM. It is the only thing
// standing between a thousand logins and a thousand duplicate contacts.
type ContactRef struct {
	ID               id.RetentionRefID `json:"id"`
	AppID            id.AppID          `json:"app_id"`
	EnvID            id.EnvironmentID  `json:"env_id"`
	UserID           id.UserID         `json:"user_id"`
	Provider         string            `json:"provider"`
	RemoteObjectType string            `json:"remote_object_type"`
	RemoteID         string            `json:"remote_id"`
	SyncedAt         time.Time         `json:"synced_at"`
}

// Ref converts the row into the shape providers speak.
func (c *ContactRef) Ref() RemoteRef {
	return RemoteRef{Provider: c.Provider, ObjectType: c.RemoteObjectType, ID: c.RemoteID}
}

// Store persists outbox jobs and contact refs.
type Store interface {
	// Enqueue inserts a pending job. Inserting a job whose IdempotencyKey
	// already exists is a no-op and returns nil, so a double-fired hook
	// cannot produce two deliveries.
	Enqueue(ctx context.Context, j *Job) error

	// ClaimDue atomically moves up to limit due pending jobs to in_flight and
	// returns them, setting InFlightUntil to now.Add(lease).
	//
	// A job is due when state is pending and NextAttemptAt is at or before
	// now, OR when state is in_flight and InFlightUntil has passed. That
	// second clause is what recovers work from a process that died mid
	// delivery; without it those rows are invisible to every later claim and
	// the user behind them silently stops syncing.
	//
	// A job claimed through that second clause comes back with Reclaimed
	// set, which is the only signal anywhere that it may already have been
	// delivered once. See Job.Reclaimed.
	ClaimDue(ctx context.Context, limit int, lease time.Duration, now time.Time) ([]*Job, error)

	// MarkDone completes a job.
	MarkDone(ctx context.Context, jobID id.RetentionJobID, now time.Time) error

	// MarkRetry returns a job to pending with an incremented attempt count.
	MarkRetry(ctx context.Context, jobID id.RetentionJobID, nextAttemptAt time.Time, lastErr string) error

	// MarkDeferred returns a job to pending at nextAttemptAt WITHOUT
	// incrementing the attempt count, and records reason in the same field
	// MarkRetry uses for its last error.
	//
	// It exists for the case where nothing failed and nothing was decided:
	// delivery is switched off for the job's app, so the job goes back on
	// the queue untouched. MarkRetry would spend an attempt, and a long
	// enough disable would then burn the whole budget for jobs that never
	// failed, so the first real error after re-enabling would dead-letter
	// them immediately.
	MarkDeferred(ctx context.Context, jobID id.RetentionJobID, nextAttemptAt time.Time, reason string) error

	// MarkDead parks a job permanently after too many attempts.
	MarkDead(ctx context.Context, jobID id.RetentionJobID, lastErr string) error

	// MarkSuppressed records that the job was deliberately not delivered.
	MarkSuppressed(ctx context.Context, jobID id.RetentionJobID, reason string) error

	// PurgeTerminal deletes terminal rows that are older than their cutoff
	// and returns how many went. `done` rows are compared against
	// doneBefore; `dead` and `suppressed` against auditBefore, which is the
	// longer of the two because those two are the audit trail.
	//
	// Only terminal rows are eligible. A `pending` or `in_flight` row is
	// never deleted however old it is: age there means a stuck job, and
	// deleting it destroys work nobody knows is missing.
	//
	// The comparison is on CreatedAt, because there is no completed_at
	// column and adding one to buy a few hours of precision against a
	// window measured in months is not worth the migration. A job that
	// spent its whole retry budget dying therefore ages out about 1.7
	// hours "early" relative to the moment it actually died.
	//
	// A zero cutoff means "delete nothing in that class", which is how a
	// non-positive retention setting switches the sweep off. Every backend
	// gets that for free: nothing was created before year one.
	//
	// Deleting a `done` row releases its idempotency key. See the
	// replay-window note in the Data model section of
	// docs/superpowers/specs/2026-09-03-crm-retention-delivery-design.md
	// for why that is safe at these distances and not at shorter ones.
	PurgeTerminal(ctx context.Context, doneBefore, auditBefore time.Time) (int, error)

	// GetJob fetches one job. Returns ErrNotFound when absent.
	GetJob(ctx context.Context, jobID id.RetentionJobID) (*Job, error)

	// ListDead returns dead-lettered jobs for an app, newest first.
	ListDead(ctx context.Context, appID id.AppID, limit int) ([]*Job, error)

	// GetRef returns the contact ref. Returns ErrNotFound when absent.
	GetRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		userID id.UserID, provider string) (*ContactRef, error)

	// PutRef inserts or updates the contact ref for the unique tuple.
	PutRef(ctx context.Context, r *ContactRef) error

	// DeleteRef removes the contact ref, so the next attempt recreates the
	// contact. Deleting an absent ref is not an error.
	DeleteRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
		userID id.UserID, provider string) error

	// ListRefsForUser returns every CRM ref held for one user across all
	// providers, for the data export. Returns an empty slice, not an error,
	// when the user has none.
	ListRefsForUser(ctx context.Context, userID id.UserID) ([]*ContactRef, error)
}
