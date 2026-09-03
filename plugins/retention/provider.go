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
