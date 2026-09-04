package retention

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// Capability is a bitmask of what a CRM can actually accept. The worker checks
// it before enqueuing, so a provider is never handed a call it would have to
// fake success on.
type Capability uint8

const (
	// CapContacts means the provider can create and update contacts.
	CapContacts Capability = 1 << iota
	// CapActivities means the provider can record an activity against a contact.
	CapActivities
	// CapActivityDedupe means LogActivity is idempotent on
	// Activity.ExternalID: given the same external id twice, the CRM ends up
	// with one activity entry, not two.
	//
	// It exists because delivery is at-least-once and contacts are protected
	// while activities are not. A contact ref makes a redelivered upsert an
	// update. Nothing plays that role for an activity, so a MarkDone that
	// fails after the provider call succeeded leaves the job in_flight, the
	// lease expires, and the CRM gets the same login logged twice.
	//
	// A provider that cannot promise this must not set the bit. The worker
	// then says so out loud when it redelivers, which is the honest answer:
	// the gap is real, and a silent nil would hide it the same way a
	// provider faking success on an unsupported call would.
	CapActivityDedupe
)

// Has reports whether every bit in want is set.
func (c Capability) Has(want Capability) bool { return c&want == want }

// RemoteRef points at a record in the CRM. ObjectType is carried alongside the
// id because Salesforce-shaped CRMs need it before you can address the record.
type RemoteRef struct {
	Provider   string `json:"provider"`
	ObjectType string `json:"object_type"`
	ID         string `json:"id"`
}

// IsZero reports whether the ref names nothing.
func (r RemoteRef) IsZero() bool { return r.ID == "" }

// Contact is the normalized subset of a user that CRMs agree on.
type Contact struct {
	UserID    id.UserID         `json:"user_id"`
	AppID     id.AppID          `json:"app_id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name,omitempty"`
	LastName  string            `json:"last_name,omitempty"`
	Traits    map[string]string `json:"traits,omitempty"`
}

// Activity is one thing the user did, addressed to an existing contact.
type Activity struct {
	Type       string            `json:"type"`
	OccurredAt time.Time         `json:"occurred_at"`
	Properties map[string]string `json:"properties,omitempty"`

	// ExternalID names this activity from our side, deterministically. It is
	// the outbox job's own idempotency key, so every delivery attempt at the
	// same job carries the same value and two different logins never share
	// one. A provider advertising CapActivityDedupe keys on it.
	//
	// Empty means the job carried no idempotency key, which the hooks never
	// produce. A provider must treat that as "cannot deduplicate" and create,
	// rather than matching every activity that also has no external id.
	ExternalID string `json:"external_id,omitempty"`

	// Redelivery says this job has already been through at least one
	// delivery attempt, so the activity may already exist in the CRM. It is
	// the signal a provider needs to decide whether the dedupe check is
	// worth its cost: on a first delivery there is nothing to collide with,
	// and HubSpot's search endpoint is rate limited to five requests per
	// second per account, so checking every time would halve the throughput
	// of the common case to protect the rare one.
	Redelivery bool `json:"redelivery,omitempty"`
}

// Provider is a CRM the retention plugin can mirror auth activity into.
type Provider interface {
	// Name is the provider's unique identifier, e.g. "hubspot".
	Name() string

	// Capabilities reports what this CRM can accept.
	Capabilities() Capability

	// UpsertContact creates or updates the contact and returns its ref.
	UpsertContact(ctx context.Context, c *Contact) (RemoteRef, error)

	// LogActivity records an activity against an existing contact.
	LogActivity(ctx context.Context, ref RemoteRef, a *Activity) error
}

// ProviderError classifies a failure so the worker can decide retry vs
// dead-letter without knowing anything about the CRM's HTTP semantics.
type ProviderError struct {
	Err        error
	Retryable  bool
	RetryAfter time.Duration // honoured when non-zero, e.g. from 429

	// DropRef means the remote record is gone, so the worker should delete
	// the local contact ref and let the retry recreate the contact rather
	// than keep updating something that no longer exists.
	DropRef bool
}

func (e *ProviderError) Error() string {
	if e.Err == nil {
		return "retention: provider error"
	}
	return e.Err.Error()
}

func (e *ProviderError) Unwrap() error { return e.Err }

// Retryable reports whether err asks to be retried, and after how long. An
// error that is not a ProviderError is treated as terminal on purpose: an
// unclassified failure retried forever is a queue that never drains, and the
// dead-letter row is the thing that gets it looked at.
func Retryable(err error) (bool, time.Duration) {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Retryable, pe.RetryAfter
	}
	return false, 0
}
