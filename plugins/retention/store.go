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
}

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
