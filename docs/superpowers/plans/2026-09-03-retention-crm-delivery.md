# Retention CRM Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mirror authsome signup and login activity into any CRM, durably and off the request path, with HubSpot as the reference provider.

**Architecture:** Auth hooks write one row to an outbox table and return. A ticker-driven worker claims due rows under a lease, resolves a `Provider` from config, upserts the contact, records the returned remote id in a ref table, then logs the activity. Retry classification lives in the provider contract as `ProviderError`, so the worker decides retry versus dead-letter without knowing any CRM's HTTP semantics.

**Tech Stack:** Go 1.26.0, grove ORM + `grove/migrate`, testify (`assert`/`require`), authsome `plugin` interfaces, `settings.Define` for dynamic config.

**Spec:** `docs/superpowers/specs/2026-09-03-crm-retention-delivery-design.md`

## Global Constraints

- Module path is `github.com/xraph/authsome`. Go directive `go 1.26.0`.
- Package name is `retention`, living at `plugins/retention/`.
- Table prefix is `authsome_retention_`. Migration group name is `authsome-retention`, and every driver group declares `migrate.DependsOn("authsome")`.
- Tests use `github.com/stretchr/testify/require` for fatal assertions and `assert` for non-fatal. No other assertion library.
- Every store method returns `ErrNotFound` (package sentinel) for a missing row, never a driver error.
- Hook methods must never return a non-nil error. A retention failure must not fail a login.
- Hooks perform exactly one store write and no store reads. Any read on the login path is a plan violation.
- Mongo and Postgres conformance runners sit behind the `integration` build tag, matching `plugins/sharedsignals/store_mongo_conformance_test.go`.
- **Task 9 is not for an agentic worker.** It contains a policy decision reserved for Rex. Halt and ask.

---

## File structure

| File | Responsibility |
|---|---|
| `id/id.go` (modify) | Two new ID prefixes and their constructors |
| `plugins/retention/provider.go` | `Provider` interface, `Capability`, `Contact`, `Activity`, `RemoteRef`, `ProviderError` |
| `plugins/retention/store.go` | `Store` interface, sentinel errors, state constants |
| `plugins/retention/store_models.go` | grove row models and mappers |
| `plugins/retention/store_memory.go` | in-memory store |
| `plugins/retention/store_sqlite.go` | sqlite store |
| `plugins/retention/store_postgres.go` | postgres store, including the `SKIP LOCKED` claim |
| `plugins/retention/store_mongo.go` | mongo store |
| `plugins/retention/migrations.go` | per-driver migration groups and DDL |
| `plugins/retention/worker.go` | claim loop, delivery, backoff, dead-letter, consent gate |
| `plugins/retention/plugin.go` | `Plugin` struct, lifecycle, settings, driver store selection |
| `plugins/retention/hooks.go` | `AfterSignUp`, `AfterSignIn`, `AfterSignOut`, `AfterUserUpdate` |
| `plugins/retention/provider_generic.go` | config-driven REST provider and `classifyHTTPError` |
| `plugins/retention/provider_hubspot.go` | HubSpot reference provider |
| `plugins/retention/export.go` | `DataExportContributor` |
| `plugins/consent/plugin.go` (modify) | additive `HasConsent` method |

---

## Task 1: ID prefixes and the provider contract

**Files:**
- Modify: `id/id.go`
- Create: `plugins/retention/provider.go`
- Test: `plugins/retention/provider_test.go`

**Interfaces:**
- Consumes: `id.New`, `id.Prefix`, `id.ParseWithPrefix` from `id/id.go`.
- Produces: `id.NewRetentionJobID() id.ID`, `id.NewRetentionRefID() id.ID`, `id.ParseRetentionJobID(string) (id.ID, error)`, `id.RetentionJobID`, `id.RetentionRefID`. Also `retention.Provider`, `retention.Capability` with `CapContacts`/`CapActivities`, `retention.Contact`, `retention.Activity`, `retention.RemoteRef`, `retention.ProviderError`, and `retention.Retryable(error) (bool, time.Duration)`.

- [ ] **Step 1: Add the ID prefixes**

In `id/id.go`, add to the `Prefix` const block (alongside `PrefixConsent Prefix = "acns"`):

```go
	PrefixRetentionJob    Prefix = "artj"
	PrefixRetentionRef    Prefix = "artr"
```

Add the type aliases next to `ConsentID`:

```go
// RetentionJobID is a type-safe identifier for retention outbox jobs (prefix: "artj").
type RetentionJobID = ID

// RetentionRefID is a type-safe identifier for retention contact refs (prefix: "artr").
type RetentionRefID = ID
```

Add the constructors and parsers next to `NewConsentID`/`ParseConsentID`:

```go
// NewRetentionJobID generates a new unique retention outbox job ID.
func NewRetentionJobID() ID { return New(PrefixRetentionJob) }

// NewRetentionRefID generates a new unique retention contact ref ID.
func NewRetentionRefID() ID { return New(PrefixRetentionRef) }

// ParseRetentionJobID parses a string and validates the "artj" prefix.
func ParseRetentionJobID(s string) (ID, error) { return ParseWithPrefix(s, PrefixRetentionJob) }

// ParseRetentionRefID parses a string and validates the "artr" prefix.
func ParseRetentionRefID(s string) (ID, error) { return ParseWithPrefix(s, PrefixRetentionRef) }
```

- [ ] **Step 2: Write the failing test**

Create `plugins/retention/provider_test.go`:

```go
package retention

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestCapabilityHas(t *testing.T) {
	both := CapContacts | CapActivities
	assert.True(t, both.Has(CapContacts))
	assert.True(t, both.Has(CapActivities))

	contactsOnly := CapContacts
	assert.True(t, contactsOnly.Has(CapContacts))
	assert.False(t, contactsOnly.Has(CapActivities),
		"a contacts-only CRM must not advertise activity logging")
}

func TestRetryableUnwrapsProviderError(t *testing.T) {
	pe := &ProviderError{Err: errors.New("429 slow down"), Retryable: true, RetryAfter: 30 * time.Second}
	ok, after := Retryable(pe)
	require.True(t, ok)
	assert.Equal(t, 30*time.Second, after)
}

func TestRetryableWrappedProviderError(t *testing.T) {
	pe := &ProviderError{Err: errors.New("boom"), Retryable: true}
	ok, after := Retryable(fmt.Errorf("hubspot: %w", pe))
	require.True(t, ok, "Retryable must see through fmt.Errorf wrapping")
	assert.Zero(t, after)
}

func TestRetryablePlainErrorIsTerminal(t *testing.T) {
	ok, after := Retryable(errors.New("unclassified"))
	assert.False(t, ok, "an unclassified error must not be retried forever")
	assert.Zero(t, after)
}

func TestRemoteRefEmpty(t *testing.T) {
	assert.True(t, RemoteRef{}.IsZero())
	assert.False(t, RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "501"}.IsZero())
}

func TestRetentionIDPrefixes(t *testing.T) {
	j := id.NewRetentionJobID()
	_, err := id.ParseRetentionJobID(j.String())
	require.NoError(t, err)

	_, err = id.ParseRetentionRefID(j.String())
	assert.Error(t, err, "a job id must not parse as a ref id")
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./plugins/retention/ -run 'TestCapability|TestRetryable|TestRemoteRef|TestRetentionID' -v`
Expected: FAIL to build, `undefined: CapContacts` and friends.

- [ ] **Step 4: Write the implementation**

Create `plugins/retention/provider.go`:

```go
// Package retention mirrors authsome auth activity into a CRM. Hooks write to
// an outbox and a background worker delivers, so a slow or unavailable CRM
// never shows up as login latency.
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
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./plugins/retention/ ./id/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add id/id.go plugins/retention/provider.go plugins/retention/provider_test.go
git commit -m "feat(retention): add the CRM provider contract"
```

---

## Task 2: Store interface, models and the memory store

**Files:**
- Create: `plugins/retention/store.go`
- Create: `plugins/retention/store_memory.go`
- Test: `plugins/retention/store_conformance_test.go`
- Test: `plugins/retention/store_memory_test.go`

**Interfaces:**
- Consumes: `RemoteRef`, `Capability` from Task 1.
- Produces: `retention.Store` with `Enqueue`, `ClaimDue`, `MarkDone`, `MarkRetry`, `MarkDead`, `MarkSuppressed`, `GetJob`, `ListDead`, `GetRef`, `PutRef`, `DeleteRef`. Plus `Job`, `ContactRef`, state constants `StatePending`/`StateInFlight`/`StateDone`/`StateDead`/`StateSuppressed`, kind constants `KindContactUpsert`/`KindActivityLog`, and `ErrNotFound`. `runStoreConformance(t, factory)` is the shared suite later store tasks reuse.

- [ ] **Step 1: Write the store interface and models**

Create `plugins/retention/store.go`:

```go
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
	ID             id.RetentionJobID     `json:"id"`
	AppID          id.AppID              `json:"app_id"`
	EnvID          id.EnvironmentID      `json:"env_id"`
	UserID         id.UserID             `json:"user_id"`
	Provider       string                `json:"provider"`
	Kind           string                `json:"kind"`
	Payload        map[string]string     `json:"payload"`
	IdempotencyKey string                `json:"idempotency_key"`
	State          string                `json:"state"`
	Attempts       int                   `json:"attempts"`
	NextAttemptAt  time.Time             `json:"next_attempt_at"`
	InFlightUntil  time.Time             `json:"in_flight_until"`
	LastError      string                `json:"last_error,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
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
}
```

- [ ] **Step 2: Write the failing conformance suite**

Create `plugins/retention/store_conformance_test.go`:

```go
package retention

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// storeFactory builds an empty store for one subtest.
type storeFactory func(t *testing.T) Store

func newJob(appID id.AppID, userID id.UserID, kind, key string, due time.Time) *Job {
	return &Job{
		ID:             id.NewRetentionJobID(),
		AppID:          appID,
		UserID:         userID,
		Provider:       "fake",
		Kind:           kind,
		Payload:        map[string]string{"email": "a@example.com"},
		IdempotencyKey: key,
		State:          StatePending,
		NextAttemptAt:  due,
		CreatedAt:      due,
	}
}

// runStoreConformance is the single suite every backend must satisfy.
func runStoreConformance(t *testing.T, newStore storeFactory) {
	ctx := context.Background()
	base := time.Now().UTC().Truncate(time.Second)

	t.Run("enqueue then claim", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindContactUpsert, "k1", base)
		require.NoError(t, s.Enqueue(ctx, j))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, j.ID.String(), got[0].ID.String())
		assert.Equal(t, StateInFlight, got[0].State)
		assert.Equal(t, "a@example.com", got[0].Payload["email"])
	})

	t.Run("enqueue is idempotent on key", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "dupe", base)))
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "dupe", base)))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		assert.Len(t, got, 1, "the same idempotency key must not enqueue twice")
	})

	t.Run("claim skips jobs not yet due", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "later", base.Add(time.Hour))))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("claim does not re-take a live lease", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "lease", base)))

		first, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, first, 1)

		second, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(30*time.Second))
		require.NoError(t, err)
		assert.Empty(t, second, "a job under a live lease must not be claimed twice")
	})

	t.Run("claim reclaims an expired lease", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "expire", base)))

		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)

		again, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(2*time.Minute))
		require.NoError(t, err)
		require.Len(t, again, 1, "a crashed worker's job must come back after the lease expires")
	})

	t.Run("mark done removes the job from the queue", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "done", base)
		require.NoError(t, s.Enqueue(ctx, j))
		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.NoError(t, s.MarkDone(ctx, j.ID, base))

		stored, err := s.GetJob(ctx, j.ID)
		require.NoError(t, err)
		assert.Equal(t, StateDone, stored.State)

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("mark retry increments attempts and defers", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		j := newJob(appID, userID, KindActivityLog, "retry", base)
		require.NoError(t, s.Enqueue(ctx, j))
		_, err := s.ClaimDue(ctx, 10, time.Minute, base)
		require.NoError(t, err)
		require.NoError(t, s.MarkRetry(ctx, j.ID, base.Add(10*time.Second), "429"))

		stored, err := s.GetJob(ctx, j.ID)
		require.NoError(t, err)
		assert.Equal(t, StatePending, stored.State)
		assert.Equal(t, 1, stored.Attempts)
		assert.Equal(t, "429", stored.LastError)

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(11*time.Second))
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, 1, got[0].Attempts, "attempts must survive the reclaim")
	})

	t.Run("dead and suppressed are terminal and distinguishable", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		dead := newJob(appID, userID, KindActivityLog, "dead", base)
		supp := newJob(appID, userID, KindActivityLog, "supp", base)
		require.NoError(t, s.Enqueue(ctx, dead))
		require.NoError(t, s.Enqueue(ctx, supp))
		require.NoError(t, s.MarkDead(ctx, dead.ID, "400 invalid email"))
		require.NoError(t, s.MarkSuppressed(ctx, supp.ID, "no marketing consent"))

		got, err := s.ClaimDue(ctx, 10, time.Minute, base.Add(time.Hour))
		require.NoError(t, err)
		assert.Empty(t, got, "terminal jobs must never be claimed again")

		listed, err := s.ListDead(ctx, appID, 10)
		require.NoError(t, err)
		require.Len(t, listed, 1, "suppressed must not show up as dead-lettered")
		assert.Equal(t, dead.ID.String(), listed[0].ID.String())

		storedSupp, err := s.GetJob(ctx, supp.ID)
		require.NoError(t, err)
		assert.Equal(t, StateSuppressed, storedSupp.State)
		assert.Equal(t, "no marketing consent", storedSupp.LastError)
	})

	t.Run("claim honours limit and takes oldest first", func(t *testing.T) {
		s := newStore(t)
		appID, userID := id.NewAppID(), id.NewUserID()
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "a", base.Add(-2*time.Hour))))
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "b", base.Add(-time.Hour))))
		require.NoError(t, s.Enqueue(ctx, newJob(appID, userID, KindActivityLog, "c", base)))

		got, err := s.ClaimDue(ctx, 2, time.Minute, base)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "a", got[0].IdempotencyKey)
		assert.Equal(t, "b", got[1].IdempotencyKey)
	})

	t.Run("get job missing returns ErrNotFound", func(t *testing.T) {
		s := newStore(t)
		_, err := s.GetJob(ctx, id.NewRetentionJobID())
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("ref put get delete round trip", func(t *testing.T) {
		s := newStore(t)
		appID, envID, userID := id.NewAppID(), id.EnvironmentID{}, id.NewUserID()

		_, err := s.GetRef(ctx, appID, envID, userID, "hubspot")
		assert.ErrorIs(t, err, ErrNotFound)

		r := &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "501", SyncedAt: base,
		}
		require.NoError(t, s.PutRef(ctx, r))

		got, err := s.GetRef(ctx, appID, envID, userID, "hubspot")
		require.NoError(t, err)
		assert.Equal(t, "501", got.RemoteID)
		assert.Equal(t, RemoteRef{Provider: "hubspot", ObjectType: "contact", ID: "501"}, got.Ref())

		r.RemoteID = "777"
		require.NoError(t, s.PutRef(ctx, r), "PutRef must upsert, not conflict")
		got, err = s.GetRef(ctx, appID, envID, userID, "hubspot")
		require.NoError(t, err)
		assert.Equal(t, "777", got.RemoteID)

		require.NoError(t, s.DeleteRef(ctx, appID, envID, userID, "hubspot"))
		_, err = s.GetRef(ctx, appID, envID, userID, "hubspot")
		assert.ErrorIs(t, err, ErrNotFound)
		require.NoError(t, s.DeleteRef(ctx, appID, envID, userID, "hubspot"),
			"deleting an absent ref is not an error")
	})

	t.Run("refs are isolated per provider", func(t *testing.T) {
		s := newStore(t)
		appID, envID, userID := id.NewAppID(), id.EnvironmentID{}, id.NewUserID()
		require.NoError(t, s.PutRef(ctx, &ContactRef{
			ID: id.NewRetentionRefID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "1", SyncedAt: base,
		}))
		_, err := s.GetRef(ctx, appID, envID, userID, "generic")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}
```

Create `plugins/retention/store_memory_test.go`:

```go
package retention

import "testing"

func TestMemoryStoreConformance(t *testing.T) {
	runStoreConformance(t, func(_ *testing.T) Store { return NewMemoryStore() })
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./plugins/retention/ -run TestMemoryStoreConformance -v`
Expected: FAIL to build, `undefined: NewMemoryStore`.

- [ ] **Step 4: Implement the memory store**

Create `plugins/retention/store_memory.go`:

```go
package retention

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is a process-local Store, used in tests and when no database is
// configured. Nothing here survives a restart, which for this plugin means a
// pending backlog is lost rather than delayed.
type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	keys map[string]string // idempotency key -> job id
	refs map[string]*ContactRef
}

// NewMemoryStore builds an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*Job),
		keys: make(map[string]string),
		refs: make(map[string]*ContactRef),
	}
}

var _ Store = (*MemoryStore)(nil)

func refKey(appID id.AppID, envID id.EnvironmentID, userID id.UserID, provider string) string {
	return appID.String() + "|" + envID.String() + "|" + userID.String() + "|" + provider
}

func cloneJob(j *Job) *Job {
	out := *j
	out.Payload = make(map[string]string, len(j.Payload))
	for k, v := range j.Payload {
		out.Payload[k] = v
	}
	return &out
}

func (s *MemoryStore) Enqueue(_ context.Context, j *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j.IdempotencyKey != "" {
		if _, dupe := s.keys[j.IdempotencyKey]; dupe {
			return nil
		}
		s.keys[j.IdempotencyKey] = j.ID.String()
	}
	if j.State == "" {
		j.State = StatePending
	}
	s.jobs[j.ID.String()] = cloneJob(j)
	return nil
}

func (s *MemoryStore) ClaimDue(_ context.Context, limit int, lease time.Duration, now time.Time) ([]*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var due []*Job
	for _, j := range s.jobs {
		switch j.State {
		case StatePending:
			if !j.NextAttemptAt.After(now) {
				due = append(due, j)
			}
		case StateInFlight:
			if j.InFlightUntil.Before(now) {
				due = append(due, j)
			}
		}
	}
	sort.Slice(due, func(a, b int) bool {
		if due[a].NextAttemptAt.Equal(due[b].NextAttemptAt) {
			return due[a].CreatedAt.Before(due[b].CreatedAt)
		}
		return due[a].NextAttemptAt.Before(due[b].NextAttemptAt)
	})
	if limit > 0 && len(due) > limit {
		due = due[:limit]
	}

	out := make([]*Job, 0, len(due))
	for _, j := range due {
		j.State = StateInFlight
		j.InFlightUntil = now.Add(lease)
		out = append(out, cloneJob(j))
	}
	return out, nil
}

func (s *MemoryStore) set(jobID id.RetentionJobID, fn func(*Job)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[jobID.String()]
	if !ok {
		return ErrNotFound
	}
	fn(j)
	return nil
}

func (s *MemoryStore) MarkDone(_ context.Context, jobID id.RetentionJobID, _ time.Time) error {
	return s.set(jobID, func(j *Job) { j.State = StateDone; j.LastError = "" })
}

func (s *MemoryStore) MarkRetry(_ context.Context, jobID id.RetentionJobID, next time.Time, lastErr string) error {
	return s.set(jobID, func(j *Job) {
		j.State = StatePending
		j.Attempts++
		j.NextAttemptAt = next
		j.InFlightUntil = time.Time{}
		j.LastError = lastErr
	})
}

func (s *MemoryStore) MarkDead(_ context.Context, jobID id.RetentionJobID, lastErr string) error {
	return s.set(jobID, func(j *Job) { j.State = StateDead; j.LastError = lastErr })
}

func (s *MemoryStore) MarkSuppressed(_ context.Context, jobID id.RetentionJobID, reason string) error {
	return s.set(jobID, func(j *Job) { j.State = StateSuppressed; j.LastError = reason })
}

func (s *MemoryStore) GetJob(_ context.Context, jobID id.RetentionJobID) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[jobID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneJob(j), nil
}

func (s *MemoryStore) ListDead(_ context.Context, appID id.AppID, limit int) ([]*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Job
	for _, j := range s.jobs {
		if j.State == StateDead && j.AppID.String() == appID.String() {
			out = append(out, cloneJob(j))
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].CreatedAt.After(out[b].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetRef(_ context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) (*ContactRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.refs[refKey(appID, envID, userID, provider)]
	if !ok {
		return nil, ErrNotFound
	}
	out := *r
	return &out, nil
}

func (s *MemoryStore) PutRef(_ context.Context, r *ContactRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := *r
	s.refs[refKey(r.AppID, r.EnvID, r.UserID, r.Provider)] = &out
	return nil
}

func (s *MemoryStore) DeleteRef(_ context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refs, refKey(appID, envID, userID, provider))
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./plugins/retention/ -run TestMemoryStoreConformance -v`
Expected: PASS, every subtest green.

- [ ] **Step 6: Commit**

```bash
git add plugins/retention/store.go plugins/retention/store_memory.go \
        plugins/retention/store_conformance_test.go plugins/retention/store_memory_test.go
git commit -m "feat(retention): add the outbox store interface and memory backend"
```

---

## Task 3: Migrations and the SQLite store

**Files:**
- Create: `plugins/retention/migrations.go`
- Create: `plugins/retention/store_models.go`
- Create: `plugins/retention/store_sqlite.go`
- Test: `plugins/retention/store_sqlite_test.go`

**Interfaces:**
- Consumes: `Store`, `Job`, `ContactRef`, `runStoreConformance` from Task 2.
- Produces: `PostgresMigrations`, `SqliteMigrations`, `MongoMigrations` (all `*migrate.Group`), `NewSqliteStore(*grove.DB) *SqliteStore`, and the shared models `jobModel`/`contactRefModel` with mappers `fromJob`/`toJob`/`fromRef`/`toRef`.

- [ ] **Step 1: Write the migrations**

Create `plugins/retention/migrations.go`. Copy the driver-group and DDL layout from `plugins/retention`'s sibling `plugins/sharedsignals/migrations.go`, which is the pattern of record. Both SQL groups use the same DDL because grove's sqlite and postgres dialects accept it:

```go
package retention

import (
	"github.com/xraph/grove/migrate"
)

// Migration groups, one per driver. Both depend on the core authsome group
// because the outbox references authsome_apps.
var (
	PostgresMigrations = migrate.NewGroup("authsome-retention", migrate.DependsOn("authsome"))
	SqliteMigrations   = migrate.NewGroup("authsome-retention", migrate.DependsOn("authsome"))
	MongoMigrations    = migrate.NewGroup("authsome-retention", migrate.DependsOn("authsome"))
)

// Mongo collection names.
const (
	colOutbox     = "authsome_retention_outbox"
	colContactRef = "authsome_retention_contact_ref"
)

const sqlSchema = `
CREATE TABLE IF NOT EXISTS authsome_retention_contact_ref (
    id                 TEXT PRIMARY KEY,
    app_id             TEXT NOT NULL,
    env_id             TEXT NOT NULL DEFAULT '',
    user_id            TEXT NOT NULL,
    provider           TEXT NOT NULL,
    remote_object_type TEXT NOT NULL DEFAULT '',
    remote_id          TEXT NOT NULL,
    synced_at          TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_retention_ref
    ON authsome_retention_contact_ref (app_id, env_id, user_id, provider);

CREATE TABLE IF NOT EXISTS authsome_retention_outbox (
    id               TEXT PRIMARY KEY,
    app_id           TEXT NOT NULL,
    env_id           TEXT NOT NULL DEFAULT '',
    user_id          TEXT NOT NULL,
    provider         TEXT NOT NULL,
    kind             TEXT NOT NULL,
    payload          TEXT NOT NULL DEFAULT '{}',
    idempotency_key  TEXT NOT NULL DEFAULT '',
    state            TEXT NOT NULL DEFAULT 'pending',
    attempts         INTEGER NOT NULL DEFAULT 0,
    next_attempt_at  TIMESTAMP NOT NULL,
    in_flight_until  TIMESTAMP NULL,
    last_error       TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS ix_retention_outbox_due
    ON authsome_retention_outbox (state, next_attempt_at);
CREATE UNIQUE INDEX IF NOT EXISTS ux_retention_outbox_key
    ON authsome_retention_outbox (idempotency_key)
    WHERE idempotency_key != '';
`

func init() {
	PostgresMigrations.Register(&migrate.Migration{
		Name: "0001_retention_tables",
		Up:   migrate.SQL(sqlSchema),
	})
	SqliteMigrations.Register(&migrate.Migration{
		Name: "0001_retention_tables",
		Up:   migrate.SQL(sqlSchema),
	})
}
```

**Before writing this file, read `plugins/sharedsignals/migrations.go` in full and match its actual registration call shape.** The `migrate.Migration` / `migrate.SQL` names above must be replaced with whatever that file uses; grove's API is the authority, not this plan. The mongo index registration also follows that file, using `mongomigrate` to create a unique index on `(app_id, env_id, user_id, provider)` for `colContactRef` and on `idempotency_key` for `colOutbox`.

- [ ] **Step 2: Write the models**

Create `plugins/retention/store_models.go`, following the grove tag style of `plugins/consent/store_models.go`:

```go
package retention

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/id"
)

type contactRefModel struct {
	grove.BaseModel `grove:"table:authsome_retention_contact_ref,alias:rtr"`

	ID               string    `grove:"id,pk"`
	AppID            string    `grove:"app_id,notnull"`
	EnvID            string    `grove:"env_id,notnull"`
	UserID           string    `grove:"user_id,notnull"`
	Provider         string    `grove:"provider,notnull"`
	RemoteObjectType string    `grove:"remote_object_type,notnull"`
	RemoteID         string    `grove:"remote_id,notnull"`
	SyncedAt         time.Time `grove:"synced_at,notnull"`
}

type jobModel struct {
	grove.BaseModel `grove:"table:authsome_retention_outbox,alias:rto"`

	ID             string       `grove:"id,pk"`
	AppID          string       `grove:"app_id,notnull"`
	EnvID          string       `grove:"env_id,notnull"`
	UserID         string       `grove:"user_id,notnull"`
	Provider       string       `grove:"provider,notnull"`
	Kind           string       `grove:"kind,notnull"`
	Payload        string       `grove:"payload,notnull"`
	IdempotencyKey string       `grove:"idempotency_key,notnull"`
	State          string       `grove:"state,notnull"`
	Attempts       int          `grove:"attempts,notnull"`
	NextAttemptAt  time.Time    `grove:"next_attempt_at,notnull"`
	InFlightUntil  sql.NullTime `grove:"in_flight_until"`
	LastError      string       `grove:"last_error,notnull"`
	CreatedAt      time.Time    `grove:"created_at,notnull"`
}

func fromJob(j *Job) *jobModel {
	payload, _ := json.Marshal(j.Payload)
	m := &jobModel{
		ID: j.ID.String(), AppID: j.AppID.String(), EnvID: j.EnvID.String(),
		UserID: j.UserID.String(), Provider: j.Provider, Kind: j.Kind,
		Payload: string(payload), IdempotencyKey: j.IdempotencyKey,
		State: j.State, Attempts: j.Attempts,
		NextAttemptAt: j.NextAttemptAt, LastError: j.LastError, CreatedAt: j.CreatedAt,
	}
	if !j.InFlightUntil.IsZero() {
		m.InFlightUntil = sql.NullTime{Time: j.InFlightUntil, Valid: true}
	}
	return m
}

func toJob(m *jobModel) (*Job, error) {
	jobID, err := id.ParseRetentionJobID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	userID, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	j := &Job{
		ID: jobID, AppID: appID, UserID: userID, Provider: m.Provider, Kind: m.Kind,
		IdempotencyKey: m.IdempotencyKey, State: m.State, Attempts: m.Attempts,
		NextAttemptAt: m.NextAttemptAt, LastError: m.LastError, CreatedAt: m.CreatedAt,
	}
	if m.InFlightUntil.Valid {
		j.InFlightUntil = m.InFlightUntil.Time
	}
	if m.EnvID != "" {
		envID, err := id.ParseEnvironmentID(m.EnvID)
		if err != nil {
			return nil, err
		}
		j.EnvID = envID
	}
	j.Payload = make(map[string]string)
	if m.Payload != "" {
		if err := json.Unmarshal([]byte(m.Payload), &j.Payload); err != nil {
			return nil, err
		}
	}
	return j, nil
}
```

Write `fromRef`/`toRef` the same way, with no JSON column to unmarshal.

- [ ] **Step 3: Write the failing sqlite conformance runner**

Create `plugins/retention/store_sqlite_test.go`, mirroring `newSQLiteConformanceStore` in `plugins/sharedsignals/store_conformance_test.go`:

```go
package retention

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

func TestSqliteStoreConformance(t *testing.T) {
	runStoreConformance(t, func(t *testing.T) Store {
		t.Helper()
		ctx := context.Background()
		dsn := "file:" + filepath.Join(t.TempDir(), "retention-conformance.db") + "?cache=shared"
		sdb := sqlitedriver.New()
		require.NoError(t, sdb.Open(ctx, dsn))
		db, err := grove.Open(sdb)
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })

		// The core migrations satisfy the group's DependsOn("authsome").
		require.NoError(t, sqlitestore.New(db).Migrate(ctx, SqliteMigrations))
		return NewSqliteStore(db)
	})
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./plugins/retention/ -run TestSqliteStoreConformance -v`
Expected: FAIL to build, `undefined: NewSqliteStore`.

- [ ] **Step 5: Implement the sqlite store**

Create `plugins/retention/store_sqlite.go` using the grove query idiom from `plugins/sharedsignals/store_sqlite.go` (`s.sdb.NewInsert(...)`, `s.sdb.NewSelect(m).Where("id = ?", ...).Scan(ctx)`, `s.sdb.NewUpdate(...)`). Two methods need care and are given in full:

```go
// Enqueue inserts a pending job, treating a duplicate idempotency key as
// success. The unique index is what makes this safe under concurrency; a
// read-then-write check would race two hooks firing at once.
func (s *SqliteStore) Enqueue(ctx context.Context, j *Job) error {
	if j.State == "" {
		j.State = StatePending
	}
	if _, err := s.sdb.NewInsert(fromJob(j)).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return sqlErr(err)
	}
	return nil
}

// ClaimDue claims due rows with a conditional update per row. The SELECT is
// only a candidate list: the UPDATE re-checks the due predicate and the row is
// kept only when it actually changed something, so two callers racing on the
// same candidate cannot both claim it.
//
// A plain unconditional "UPDATE ... WHERE id = ?" would let both win. SQLite
// being single-writer does not help, because nothing stops two readers seeing
// the same row before either writes. The idiom here is the one already used in
// store/sqlite/refresh_replay.go:137 — conditional update plus RowsAffected.
func (s *SqliteStore) ClaimDue(ctx context.Context, limit int, lease time.Duration,
	now time.Time) ([]*Job, error) {
	q := s.sdb.NewSelect(&models).
		Where("(state = ? AND next_attempt_at <= ?) OR (state = ? AND in_flight_until < ?)",
			StatePending, now, StateInFlight, now).
		OrderExpr("next_attempt_at ASC, created_at ASC")
	// limit <= 0 means no limit, matching the memory store.
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []*jobModel
	if err := q.Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}

	until := now.Add(lease)
	out := make([]*Job, 0, len(models))
	for _, m := range models {
		res, err := s.sdb.NewUpdate((*jobModel)(nil)).
			Set("state = ?", StateInFlight).
			Set("in_flight_until = ?", until).
			Where("id = ?", m.ID).
			Where("(state = ? AND next_attempt_at <= ?) OR (state = ? AND in_flight_until < ?)",
				StatePending, now, StateInFlight, now).
			Exec(ctx)
		if err != nil {
			return nil, sqlErr(err)
		}
		n, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
		if n == 0 {
			// Somebody else claimed it between our select and our update.
			continue
		}
		j, err := toJob(m)
		if err != nil {
			return nil, err
		}
		j.State = StateInFlight
		j.InFlightUntil = until
		out = append(out, j)
	}
	return out, nil
}
```

Add the two small helpers this file needs, mapping driver errors onto package sentinels:

```go
// sqlErr maps a driver miss onto ErrNotFound so callers never see sql.ErrNoRows.
func sqlErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// isUniqueViolation reports whether err is a unique-constraint failure, which
// Enqueue treats as a duplicate rather than a fault.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}
```

Implement the remaining `Store` methods as direct `NewUpdate`/`NewSelect`/`NewDelete` calls. `PutRef` upserts: select by the unique tuple, insert when `ErrNotFound`, otherwise update `remote_object_type`, `remote_id` and `synced_at`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./plugins/retention/ -v`
Expected: PASS, both memory and sqlite conformance suites green against the same cases.

- [ ] **Step 7: Commit**

```bash
git add plugins/retention/migrations.go plugins/retention/store_models.go \
        plugins/retention/store_sqlite.go plugins/retention/store_sqlite_test.go
git commit -m "feat(retention): add migrations and the sqlite outbox store"
```

---

## Task 4: Postgres and Mongo stores

**Files:**
- Create: `plugins/retention/store_postgres.go`
- Create: `plugins/retention/store_mongo.go`
- Test: `plugins/retention/store_postgres_conformance_test.go`
- Test: `plugins/retention/store_mongo_conformance_test.go`

**Interfaces:**
- Consumes: `runStoreConformance`, the models and mappers from Task 3.
- Produces: `NewPostgresStore(*grove.DB) *PostgresStore`, `NewMongoStore(*grove.DB) *MongoStore`.

- [ ] **Step 1: Write the failing conformance runners**

Create both test files behind the `//go:build integration` tag. Copy the harness shapes from the files that actually exist on this branch:

- Postgres: `plugins/agentauth/store_conformance_postgres_test.go` — testcontainers, `pgmodule.Run(ctx, "postgres:16-alpine", ...)`, then `pgdriver.New()` and `pgstore "github.com/xraph/authsome/store/postgres"` to migrate. Requires Docker.
- Mongo: `plugins/sharedsignals/store_mongo_conformance_test.go` — `AUTHSOME_MONGO_URI` from the environment, `mongodriver`, `mongostore "github.com/xraph/authsome/store/mongo"`, skipping when the variable is unset.

Each is a thin wrapper calling `runStoreConformance` with a factory that migrates the right group and returns the store.

**Both files must declare `package retention`, not `package retention_test`.** `runStoreConformance` is unexported and lives in the internal test package, so an external test package cannot call it. Note that the agentauth harness you are copying uses `package agentauth_test` — do not copy that part.

- [ ] **Step 2: Run to verify they fail**

Run: `go test -tags integration ./plugins/retention/ -run 'TestPostgresStoreConformance' -v`
Expected: FAIL to build, `undefined: NewPostgresStore`.

- [ ] **Step 3: Implement the Postgres store**

Postgres shares the models and every mapper with sqlite, so most methods are the same calls against `s.pdb`. `ClaimDue` is the one real difference and must be a single statement:

```go
// ClaimDue claims a batch in one statement. SKIP LOCKED is doing real work
// here: without it two instances claiming at once block each other and take
// turns where they should be taking disjoint batches, so throughput collapses
// back to single-instance under exactly the load that made you run two.
//
// The second OR clause reclaims rows whose lease expired, which is how work
// from a process that died mid-delivery comes back at all.
const claimSQL = `
UPDATE authsome_retention_outbox
   SET state = 'in_flight', in_flight_until = $1
 WHERE id IN (
       SELECT id FROM authsome_retention_outbox
        WHERE (state = 'pending'   AND next_attempt_at <= $2)
           OR (state = 'in_flight' AND in_flight_until < $2)
        ORDER BY next_attempt_at ASC, created_at ASC
        LIMIT $3
        FOR UPDATE SKIP LOCKED
       )
RETURNING id, app_id, env_id, user_id, provider, kind, payload,
          idempotency_key, state, attempts, next_attempt_at,
          in_flight_until, last_error, created_at`
```

Scan the returned rows into `jobModel` values and map them through `toJob`.

- [ ] **Step 4: Implement the Mongo store**

Mongo shares neither models nor mappers, so it gets its own bson documents. `ClaimDue` loops `FindOneAndUpdate` with the same due-or-expired filter, sorted by `next_attempt_at`, until it has `limit` documents or the filter stops matching. That is atomic per document, which is what the contract needs. `Enqueue` relies on the unique index on `idempotency_key` and treats a duplicate-key write error as success.

- [ ] **Step 5: Run both suites**

Run: `go test -tags integration ./plugins/retention/ -v`
Expected: PASS. If Docker or `AUTHSOME_MONGO_URI` is unavailable the runners skip, matching the sharedsignals harness. A skip is not a pass; note it in the task report.

- [ ] **Step 6: Commit**

```bash
git add plugins/retention/store_postgres.go plugins/retention/store_mongo.go \
        plugins/retention/store_postgres_conformance_test.go \
        plugins/retention/store_mongo_conformance_test.go
git commit -m "feat(retention): add postgres and mongo outbox stores"
```

---

## Task 5: The delivery worker

**Files:**
- Create: `plugins/retention/worker.go`
- Test: `plugins/retention/worker_test.go`

**Interfaces:**
- Consumes: `Store`, `Job`, `ContactRef`, `Provider`, `ProviderError`, `Retryable`, `Capability`.
- Produces: `newWorker(deps workerDeps) *worker`, `(*worker).start()`, `(*worker).startWith(tick <-chan time.Time)`, `(*worker).stop()`, `(*worker).runOnce(ctx context.Context)`, and `workerDeps{Store, Providers map[string]Provider, Logger, Interval, Lease, BatchSize, MaxAttempts, BaseBackoff, LoadContact func(context.Context, *Job) (*Contact, error), AllowSend func(context.Context, *Job) (bool, string)}`.

The `startWith` seam is copied deliberately from `plugins/sharedsignals/refresh.go:43`. Tests push ticks by hand and never sleep.

- [ ] **Step 1: Write the failing tests**

Create `plugins/retention/worker_test.go`:

```go
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
	mu        sync.Mutex
	caps      Capability
	upsertErr []error // consumed one per call
	activity  []*Activity
	upserts   int
	refID     string
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

func TestWorkerDeadLettersAfterMaxAttempts(t *testing.T) {
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
	w.deps.AllowSend = func(_ context.Context, _ *Job) (bool, string) {
		return false, "no marketing consent"
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./plugins/retention/ -run TestWorker -v`
Expected: FAIL to build, `undefined: newWorker`.

- [ ] **Step 3: Implement the worker**

Create `plugins/retention/worker.go`:

```go
package retention

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"
)

// errNoContactCapability means the provider cannot hold contacts at all, so
// this job was never deliverable. deliver routes it to suppressed rather than
// dead, because the provider declared the limitation up front.
var errNoContactCapability = errors.New("retention: provider does not support contacts")

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
		if errors.Is(err, errNoContactCapability) {
			// Symmetric with the CapActivities check below. The provider told
			// us up front it cannot do this, so the right record is a
			// deliberate skip, not a failed delivery. Dead-lettering here
			// would put "we tried and failed" in the audit trail for
			// something we never attempted.
			w.suppress(ctx, j, "provider does not support contacts")
			return
		}
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
		return RemoteRef{}, errNoContactCapability
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
```

Add `"github.com/xraph/authsome/id"` to the import block, which `ensureRef` needs.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./plugins/retention/ -run TestWorker -v -race`
Expected: PASS, no data races.

- [ ] **Step 5: Commit**

```bash
git add plugins/retention/worker.go plugins/retention/worker_test.go
git commit -m "feat(retention): add the outbox delivery worker"
```

---

## Task 6: Plugin lifecycle, settings and store selection

**Files:**
- Create: `plugins/retention/plugin.go`
- Test: `plugins/retention/plugin_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1 to 5.
- Produces: `retention.New(cfg ...Config) *Plugin`, `Config`, `ProviderConfig`, `(*Plugin).SetStore(Store)`, `(*Plugin).RegisterProvider(Provider)`, and the settings `SettingEnabled`, `SettingRequireConsent`, `SettingConsentPurpose`, `SettingTrackSignOut`. Both config structs are defined here, in full, because Tasks 7, 9, 10 and 11 all name them:

```go
// ProviderConfig configures one CRM destination. Type selects the
// implementation: "hubspot" for the vendor provider, "generic" for the
// config-driven REST one.
type ProviderConfig struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Token       string            `json:"token,omitempty"`
	AuthType    string            `json:"auth_type,omitempty"`    // "bearer" (default) or "header"
	AuthHeader  string            `json:"auth_header,omitempty"`  // used when AuthType is "header"
	BaseURL     string            `json:"base_url,omitempty"`     // overridden in tests
	ContactURL  string            `json:"contact_url,omitempty"`  // generic only
	ActivityURL string            `json:"activity_url,omitempty"` // generic only; empty means no CapActivities
	FieldMap    map[string]string `json:"field_map,omitempty"`    // generic only
}

// Config is the plugin's static configuration. Everything here has a working
// default except Providers, and an empty Providers list makes every hook a
// no-op rather than an error.
type Config struct {
	Providers    []ProviderConfig `json:"providers"`
	TickInterval time.Duration    `json:"tick_interval"` // default 30s
	Lease        time.Duration    `json:"lease"`         // default 2m
	BaseBackoff  time.Duration    `json:"base_backoff"`  // default 5s
	BatchSize    int              `json:"batch_size"`    // default 50
	MaxAttempts  int              `json:"max_attempts"`  // default 8
}

// defaults fills the zero values, matching the Config.defaults() convention in
// plugins/sharedsignals/plugin.go.
func (c *Config) defaults() {
	if c.TickInterval <= 0 {
		c.TickInterval = 30 * time.Second
	}
	if c.Lease <= 0 {
		c.Lease = 2 * time.Minute
	}
	if c.BaseBackoff <= 0 {
		c.BaseBackoff = 5 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 8
	}
}
```

`buildProviders` maps each `ProviderConfig` onto an implementation by `Type`,
returning an error for an unknown type so a typo in config fails at startup
rather than dead-lettering every job later. Call it from `OnInit`, never from
`New`: Task 7's tests construct a plugin with a `Providers` entry and inject
fake providers directly, so a `New` that tried to build them would fail before
those tests could run.

Until Tasks 9 and 10 land, every `Type` is unknown and `buildProviders` errors
for all of them. That is expected at this point in the plan. Both of this
task's store tests pass an empty `Providers` list so they do not depend on
providers existing.

- [ ] **Step 1: Write the failing test**

Create `plugins/retention/plugin_test.go`:

```go
package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugin"
)

func TestPluginName(t *testing.T) {
	assert.Equal(t, "retention", New().Name())
}

func TestPluginImplementsExpectedInterfaces(t *testing.T) {
	p := New()
	assert.Implements(t, (*plugin.Plugin)(nil), p)
	assert.Implements(t, (*plugin.OnInit)(nil), p)
	assert.Implements(t, (*plugin.OnShutdown)(nil), p)
	assert.Implements(t, (*plugin.MigrationProvider)(nil), p)
	assert.Implements(t, (*plugin.SettingsProvider)(nil), p)
	// Hook interfaces are asserted in hooks_test.go (Task 7), where the
	// methods exist.
}

func TestPluginFallsBackToMemoryStore(t *testing.T) {
	p := New()
	require.NoError(t, p.OnInit(context.Background(), &stubEngine{}))
	assert.NotNil(t, p.store)
	require.NoError(t, p.OnShutdown(context.Background()))
}

func TestPluginShutdownIsSafeWithoutInit(t *testing.T) {
	assert.NoError(t, New().OnShutdown(context.Background()),
		"OnShutdown may run on a plugin whose OnInit never completed")
}

func TestMigrationGroupsPerDriver(t *testing.T) {
	p := New()
	assert.Len(t, p.MigrationGroups("pg"), 1)
	assert.Len(t, p.MigrationGroups("sqlite"), 1)
	assert.Len(t, p.MigrationGroups("mongo"), 1)
	assert.Empty(t, p.MigrationGroups("cassandra"))
}
```

Create `plugins/retention/stub_engine_test.go` by copying `plugins/sharedsignals/stub_engine_test.go` and renaming the type to `stubEngine`. That file already satisfies `plugin.Engine` with nil-returning methods, which is exactly what these tests need. Per the house convention, plugin fixtures wire no real `Engine`, so anything needing real session behaviour belongs in `api/` rather than here.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/retention/ -run TestPlugin -v`
Expected: FAIL to build, `undefined: New`.

- [ ] **Step 3: Implement the plugin**

Create `plugins/retention/plugin.go`. Match `plugins/sharedsignals/plugin.go` exactly for the compile-time interface checks, the `settings.Define` block, the driver switch in `OnInit`, and `MigrationGroups`. The specifics:

```go
// Compile-time interface checks.
var (
	_ plugin.Plugin            = (*Plugin)(nil)
	_ plugin.OnInit            = (*Plugin)(nil)
	_ plugin.OnShutdown        = (*Plugin)(nil)
	_ plugin.MigrationProvider = (*Plugin)(nil)
	_ plugin.SettingsProvider  = (*Plugin)(nil)
)

// The four hook interface checks live in hooks.go (Task 7), not here. A
// compile-time assertion for a method that does not exist yet would stop this
// task from building at all.

var (
	// SettingEnabled turns delivery off without dropping queued work.
	SettingEnabled = settings.Define("retention.enabled", true,
		settings.WithDisplayName("CRM Retention Sync Enabled"),
		settings.WithDescription("Mirror signup and login activity into the configured CRM"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(10),
	)

	// SettingRequireConsent gates delivery on an active consent grant.
	//
	// Defaults to false on purpose. Fail-closed reads as the responsible
	// default, and would be right if purposes were a fixed enum. They are
	// free text, so fail-closed means a fresh install watches the plugin do
	// nothing and files a bug.
	SettingRequireConsent = settings.Define("retention.require_consent", false,
		settings.WithDisplayName("Require Consent Before Sync"),
		settings.WithDescription("Only send to the CRM when the user has an active grant for the purpose below"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Needs the consent plugin registered. With this on, a user with no grant is never sent."),
		settings.WithOrder(20),
	)

	// SettingConsentPurpose is the purpose string the gate checks.
	SettingConsentPurpose = settings.Define("retention.consent_purpose", "marketing",
		settings.WithDisplayName("Consent Purpose"),
		settings.WithDescription("The consent purpose that authorises CRM sync"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(30),
	)

	// SettingTrackSignOut records sign-out as an activity. Off by default
	// because most CRMs do not want the noise.
	SettingTrackSignOut = settings.Define("retention.track_sign_out", false,
		settings.WithDisplayName("Track Sign-Out"),
		settings.WithDescription("Log a sign-out activity alongside sign-in"),
		settings.WithCategory("Retention"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithOrder(40),
	)
)
```

`New` must set `logger` to `log.NewNoopLogger()` so the hooks can log an
enqueue failure before `OnInit` has ever run. A nil logger there would turn a
swallowed store error into a panic inside a login.

The `Plugin` struct holds `config Config`, `store Store`, `providers map[string]Provider`, `engine plugin.Engine`, `logger log.Logger`, `settingsMgr *settings.Manager`, `consent consentChecker` and `worker *worker`.

`OnInit` captures the engine references, runs the same driver switch as sharedsignals (`pg`/`sqlite`/`mongo`, falling back to `NewMemoryStore()` with a Warn naming the impact: a pending backlog is process-local and dies with the process), builds providers from `p.config.Providers`, then starts the worker with `LoadContact` wired to the engine and `AllowSend` left nil.

`AllowSend` is nil at this task on purpose. The worker's contract from Task 5
already documents nil as "always allow", and the consent gate that fills it in
does not exist until Task 8. Wiring `p.allowSend` here would reference a method
this task does not define, and the package would not build. Task 8 replaces the
nil and resolves the consent plugin in the same change. `OnShutdown` calls `p.worker.stop()` when non-nil and returns nil, so it is safe on a plugin whose `OnInit` never ran.

`DeclareSettings` registers the four settings via `settings.RegisterTyped(m, "retention", ...)`, matching `plugins/social/plugin.go:214`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./plugins/retention/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add plugins/retention/plugin.go plugins/retention/plugin_test.go plugins/retention/stub_engine_test.go
git commit -m "feat(retention): add plugin lifecycle, settings and store selection"
```

---

## Task 7: Auth hooks

**Files:**
- Create: `plugins/retention/hooks.go`
- Test: `plugins/retention/hooks_test.go`

**Interfaces:**
- Consumes: `Store`, `Job`, `KindContactUpsert`, `KindActivityLog`, `Plugin` from Task 6.
- Produces: `(*Plugin).AfterSignUp`, `AfterSignIn`, `AfterSignOut`, `AfterUserUpdate`, and `idempotencyKey(parts ...string) string`.

Read the exact signatures from `plugin/plugin.go:245-330` before writing, and match them. They are the contract; this plan does not restate them because a stale copy here would be worse than a lookup.

This task also owns the four hook interface assertions that Task 6 deliberately
left out, because this is where the methods start existing. Add to `hooks.go`:

```go
// Compile-time interface checks for the hooks this file implements.
var (
	_ plugin.AfterSignUp     = (*Plugin)(nil)
	_ plugin.AfterSignIn     = (*Plugin)(nil)
	_ plugin.AfterSignOut    = (*Plugin)(nil)
	_ plugin.AfterUserUpdate = (*Plugin)(nil)
)
```

and add the matching runtime assertion to `hooks_test.go`:

```go
func TestPluginImplementsHookInterfaces(t *testing.T) {
	p := New()
	assert.Implements(t, (*plugin.AfterSignUp)(nil), p)
	assert.Implements(t, (*plugin.AfterSignIn)(nil), p)
	assert.Implements(t, (*plugin.AfterSignOut)(nil), p)
	assert.Implements(t, (*plugin.AfterUserUpdate)(nil), p)
}
```

That needs `"github.com/xraph/authsome/plugin"` in the test imports.

- [ ] **Step 1: Write the failing tests**

Create `plugins/retention/hooks_test.go`:

```go
package retention

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
)

// failingStore fails every write, to prove a hook still returns nil.
type failingStore struct{ Store }

func (failingStore) Enqueue(context.Context, *Job) error { return errors.New("db down") }

func newHookPlugin(s Store) *Plugin {
	p := New(Config{Providers: []ProviderConfig{{Name: "fake", Type: "generic"}}})
	p.store = s
	// enqueueFor logs on a store error, and these tests exercise that path,
	// so the logger must be set here rather than left to OnInit.
	p.logger = log.NewNoopLogger()
	p.providers = map[string]Provider{"fake": &fakeProvider{caps: CapContacts | CapActivities}}
	return p
}

func TestAfterSignUpEnqueuesContactAndActivity(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterSignUpFor(ctx, appID, id.EnvironmentID{}, userID))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 2)

	kinds := map[string]bool{jobs[0].Kind: true, jobs[1].Kind: true}
	assert.True(t, kinds[KindContactUpsert])
	assert.True(t, kinds[KindActivityLog])
}

func TestAfterSignInEnqueuesActivityOnly(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := newHookPlugin(s)
	appID, userID := id.NewAppID(), id.NewUserID()

	require.NoError(t, p.afterSignInFor(ctx, appID, id.EnvironmentID{}, userID))

	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	require.Len(t, jobs, 1, "sign-in must not read the ref table to decide on an upsert")
	assert.Equal(t, KindActivityLog, jobs[0].Kind)
	assert.Equal(t, "logged_in", jobs[0].Payload["activity_type"])
}

func TestHookSwallowsStoreErrors(t *testing.T) {
	ctx := context.Background()
	p := newHookPlugin(failingStore{Store: NewMemoryStore()})
	err := p.afterSignInFor(ctx, id.NewAppID(), id.EnvironmentID{}, id.NewUserID())
	assert.NoError(t, err, "a retention failure must never fail a login")
}

func TestHookNoOpWithoutProviders(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := New()
	p.store = s
	p.providers = map[string]Provider{}

	require.NoError(t, p.afterSignInFor(ctx, id.NewAppID(), id.EnvironmentID{}, id.NewUserID()))
	jobs, err := s.ClaimDue(ctx, 10, 0, timeNow())
	require.NoError(t, err)
	assert.Empty(t, jobs, "no configured provider means no queued work")
}

func TestIdempotencyKeyIsStableAndDistinct(t *testing.T) {
	a := idempotencyKey("hubspot", "ausr_1", "logged_in", "2026-09-03T10:00:00Z")
	b := idempotencyKey("hubspot", "ausr_1", "logged_in", "2026-09-03T10:00:00Z")
	c := idempotencyKey("hubspot", "ausr_1", "logged_in", "2026-09-03T10:00:01Z")
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
}
```

`afterSignUpFor` / `afterSignInFor` are small internal helpers taking plain ids, so the tests do not have to build a `forge.Context`. The exported hook methods unwrap the engine's context and call them, which keeps the untestable part down to one line each.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./plugins/retention/ -run 'TestAfter|TestHook|TestIdempotency' -v`
Expected: FAIL to build, `undefined: (*Plugin).afterSignUpFor`.

- [ ] **Step 3: Implement the hooks**

Create `plugins/retention/hooks.go`. Every exported hook wraps one internal helper, logs nothing on the happy path, and returns nil unconditionally:

```go
// enqueueFor writes one job per configured provider. It is the only thing a
// hook does. No reads: a lookup here would put a query on the login path, and
// the whole point of the outbox is that a login writes one row and gets out.
func (p *Plugin) enqueueFor(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, kind, activityType string) {
	if p.store == nil || len(p.providers) == 0 {
		return
	}
	now := time.Now()
	stamp := now.UTC().Format(time.RFC3339Nano)
	for name := range p.providers {
		j := &Job{
			ID: id.NewRetentionJobID(), AppID: appID, EnvID: envID, UserID: userID,
			Provider: name, Kind: kind,
			Payload:        map[string]string{"activity_type": activityType},
			IdempotencyKey: idempotencyKey(name, userID.String(), kind+":"+activityType, stamp),
			State:          StatePending,
			NextAttemptAt:  now,
			CreatedAt:      now,
		}
		if err := p.store.Enqueue(ctx, j); err != nil {
			// Swallowed on purpose. This runs in the login path and a CRM
			// bookkeeping miss must never turn into a failed sign-in.
			p.logger.Warn("retention: enqueue failed",
				log.String("provider", name),
				log.String("kind", kind),
				log.String("error", err.Error()))
		}
	}
}

// idempotencyKey is a stable hash of the parts that make one delivery unique,
// so a hook that fires twice for the same event enqueues once.
func idempotencyKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
```

`afterSignUpFor` calls `enqueueFor` twice, with `KindContactUpsert`/`"signed_up"` and `KindActivityLog`/`"signed_up"`. `afterSignInFor` calls it once with `KindActivityLog`/`"logged_in"`. `afterSignOutFor` calls it once with `"logged_out"`, guarded on `SettingTrackSignOut`. `afterUserUpdateFor` calls it once with `KindContactUpsert`/`"profile_updated"`. Add a `timeNow()` test helper returning `time.Now().Add(time.Minute)` so `ClaimDue` in the tests sees everything as due.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./plugins/retention/ -v -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add plugins/retention/hooks.go plugins/retention/hooks_test.go
git commit -m "feat(retention): enqueue CRM work from the auth hooks"
```

---

## Task 8: Consent gate

**Files:**
- Modify: `plugins/consent/plugin.go`
- Create: `plugins/retention/consent.go`
- Test: `plugins/consent/has_consent_test.go`
- Test: `plugins/retention/consent_test.go`

**Interfaces:**
- Consumes: `consent.Store.GetConsent` at `plugins/consent/consent.go:45`, `Plugin.AllowSend` wiring from Task 6.
- Produces: `consent.(*Plugin).HasConsent(ctx, userID, appID, purpose) (bool, error)`, `retention.consentChecker` interface, `(*Plugin).allowSend(ctx, *Job) (bool, string)`.

- [ ] **Step 1: Write the failing consent test**

Create `plugins/consent/has_consent_test.go`:

```go
package consent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestHasConsent(t *testing.T) {
	ctx := context.Background()
	p := &Plugin{}
	p.SetConsentStore(NewMemoryStore())
	userID, appID := id.NewUserID(), id.NewAppID()

	ok, err := p.HasConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	assert.False(t, ok, "no record means no consent")

	require.NoError(t, p.store.GrantConsent(ctx, &Consent{
		ID: id.NewConsentID(), UserID: userID, AppID: appID,
		Purpose: "marketing", Granted: true, Version: "1", GrantedAt: time.Now(),
	}))

	ok, err = p.HasConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = p.HasConsent(ctx, userID, appID, "analytics")
	require.NoError(t, err)
	assert.False(t, ok, "a grant for one purpose is not a grant for another")

	require.NoError(t, p.store.RevokeConsent(ctx, userID, appID, "marketing"))
	ok, err = p.HasConsent(ctx, userID, appID, "marketing")
	require.NoError(t, err)
	assert.False(t, ok, "a revoked grant reports false")
}

func TestHasConsentWithoutStore(t *testing.T) {
	ok, err := (&Plugin{}).HasConsent(context.Background(),
		id.NewUserID(), id.NewAppID(), "marketing")
	require.NoError(t, err)
	assert.False(t, ok, "an unconfigured store must not report consent")
}
```

Check the exact name of the consent memory-store constructor in `plugins/consent/store_memory.go` and use it; `NewMemoryStore` above is the expected name, not a verified one.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./plugins/consent/ -run TestHasConsent -v`
Expected: FAIL to build, `p.HasConsent undefined`.

- [ ] **Step 3: Implement HasConsent**

Add to `plugins/consent/plugin.go`, next to the other exported methods:

```go
// HasConsent reports whether the user currently has an active grant for
// purpose. A missing record, a revoked grant and an unconfigured store all
// report false, so a caller gating an outbound send never has to distinguish
// "no" from "do not know" before deciding not to send.
func (p *Plugin) HasConsent(ctx context.Context, userID id.UserID,
	appID id.AppID, purpose string) (bool, error) {
	if p.store == nil {
		return false, nil
	}
	c, err := p.store.GetConsent(ctx, userID, appID, purpose)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return c != nil && c.Granted && c.RevokedAt.IsZero(), nil
}
```

Check `plugins/consent/consent.go` for the package's actual not-found sentinel and for whether `Consent.RevokedAt` is a `time.Time` or a `*time.Time`, and adjust the final condition to match.

- [ ] **Step 4: Write the failing retention gate test**

Create `plugins/retention/consent_test.go`:

```go
package retention

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

type stubConsent struct {
	granted bool
	err     error
	calls   int
}

func (s *stubConsent) HasConsent(context.Context, id.UserID, id.AppID, string) (bool, error) {
	s.calls++
	return s.granted, s.err
}

func TestAllowSendPassesWhenGateDisabled(t *testing.T) {
	p := New()
	p.requireConsent = false
	p.consent = &stubConsent{granted: false}

	ok, _ := p.allowSend(context.Background(), &Job{})
	assert.True(t, ok, "with the gate off, consent is not consulted")
	assert.Zero(t, p.consent.(*stubConsent).calls)
}

func TestAllowSendBlocksWithoutGrant(t *testing.T) {
	p := New()
	p.requireConsent = true
	p.consentPurpose = "marketing"
	p.consent = &stubConsent{granted: false}

	ok, reason := p.allowSend(context.Background(), &Job{})
	assert.False(t, ok)
	assert.Contains(t, reason, "marketing")
}

func TestAllowSendPassesWithGrant(t *testing.T) {
	p := New()
	p.requireConsent = true
	p.consentPurpose = "marketing"
	p.consent = &stubConsent{granted: true}

	ok, _ := p.allowSend(context.Background(), &Job{})
	assert.True(t, ok)
}

func TestAllowSendBlocksWhenGateOnButConsentUnavailable(t *testing.T) {
	p := New()
	p.requireConsent = true
	p.consentPurpose = "marketing"
	p.consent = nil // consent plugin not registered

	ok, reason := p.allowSend(context.Background(), &Job{})
	assert.False(t, ok, "asking for a gate you cannot evaluate must not send")
	assert.Contains(t, reason, "unavailable")
}

func TestAllowSendBlocksOnLookupError(t *testing.T) {
	p := New()
	p.requireConsent = true
	p.consent = &stubConsent{err: assert.AnError}

	ok, _ := p.allowSend(context.Background(), &Job{})
	assert.False(t, ok, "a failed lookup must not be read as consent")
}
```

- [ ] **Step 5: Run to verify it fails**

Run: `go test ./plugins/retention/ -run TestAllowSend -v`
Expected: FAIL to build, `undefined: (*Plugin).allowSend`.

- [ ] **Step 6: Implement the gate**

Create `plugins/retention/consent.go`:

```go
package retention

import (
	"context"

	"github.com/xraph/authsome/id"
)

// consentChecker is the slice of the consent plugin this one needs. Keeping it
// narrow and local means consent stays an optional dependency, resolved the
// way anomaly, geofence, impossibletravel and vpndetect all resolve geoip.
type consentChecker interface {
	HasConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) (bool, error)
}

// allowSend is the worker's AllowSend. It runs at delivery, not at enqueue, so
// a user who revokes between login and send has that revocation honoured.
//
// Every uncertain branch answers "do not send". Turning the gate on is an
// explicit request for consent to be established before data leaves, and
// treating an error or a missing consent plugin as permission would quietly
// defeat that.
func (p *Plugin) allowSend(ctx context.Context, j *Job) (bool, string) {
	if !p.requireConsent {
		return true, ""
	}
	if p.consent == nil {
		return false, "consent required but the consent plugin is unavailable"
	}
	purpose := p.consentPurpose
	if purpose == "" {
		purpose = "marketing"
	}
	granted, err := p.consent.HasConsent(ctx, j.UserID, j.AppID, purpose)
	if err != nil {
		return false, "consent lookup failed: " + err.Error()
	}
	if !granted {
		return false, "no active consent for purpose " + purpose
	}
	return true, ""
}
```

Add `requireConsent bool`, `consentPurpose string` and `consent consentChecker` to the `Plugin` struct, and in `OnInit` resolve them:

```go
	if cp := engine.Plugin("consent"); cp != nil {
		if checker, ok := cp.(consentChecker); ok {
			p.consent = checker
		}
	}
```

Read `requireConsent` and `consentPurpose` from the settings manager in `OnInit`, then pass `p.allowSend` as the worker's `AllowSend`, replacing the nil that Task 6 left there. Both changes land in `OnInit` in this task: resolving the consent plugin and filling in the gate are the same change, and neither compiles before this task defines `consentChecker` and `allowSend`.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./plugins/retention/ ./plugins/consent/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add plugins/consent/plugin.go plugins/consent/has_consent_test.go \
        plugins/retention/consent.go plugins/retention/consent_test.go
git commit -m "feat(retention): gate CRM delivery on consent at send time"
```

---

## Task 9: Generic HTTP provider and retry classification

> **STOP. This task is not for an agentic worker.**
> Step 3 is a policy decision reserved for Rex. If you are a subagent, complete Steps 1 and 2, then halt and report that Step 3 needs him.

**Files:**
- Create: `plugins/retention/provider_generic.go`
- Test: `plugins/retention/provider_generic_test.go`

**Interfaces:**
- Consumes: `Provider`, `ProviderError`, `Contact`, `Activity`, `RemoteRef` from Task 1, and `ProviderConfig` from Task 6.
- Produces: `NewGenericProvider(ProviderConfig) (*GenericProvider, error)` and `classifyHTTPError(resp *http.Response, body []byte, err error) *ProviderError`. `ProviderConfig` is defined in Task 6; this task only reads `ContactURL`, `ActivityURL`, `AuthType`, `AuthHeader`, `Token` and `FieldMap` from it.

- [ ] **Step 1: Write the provider and its transport tests**

Create `plugins/retention/provider_generic.go` with `NewGenericProvider`, `Name`, `Capabilities` (returns `CapContacts`, plus `CapActivities` only when `ActivityURL` is set), `UpsertContact` and `LogActivity`. Both calls marshal a body by applying `FieldMap` to the contact or activity, POST it with the configured auth header, and return `classifyHTTPError(resp, body, err)` when the status is not 2xx. `UpsertContact` reads the remote id out of the response using the `FieldMap["remote_id"]` JSON path, defaulting to `id`.

Test the transport half against `httptest.NewServer`: a 200 returns the parsed ref, the auth header arrives as configured, `FieldMap` renames fields, and a provider with no `ActivityURL` reports no `CapActivities`.

- [ ] **Step 2: Write the classification signature and run the suite red**

Add to `provider_generic.go`:

```go
// classifyHTTPError maps a CRM's HTTP response onto a ProviderError so the
// worker can decide retry vs dead-letter. resp may be nil when the request
// never completed (dial failure, timeout).
func classifyHTTPError(resp *http.Response, body []byte, err error) *ProviderError {
	panic("classifyHTTPError: policy not yet decided, see Task 9 Step 3")
}
```

Run: `go test ./plugins/retention/ -run TestGenericProvider -v`
Expected: the transport tests PASS; anything reaching the classifier panics.

- [ ] **Step 3: Rex decides the classification policy, then writes tests and body together**

This one function decides whether a bad CRM response means try again soon, wait exactly this long, or give up for good. Wrong one way and you hammer a rate-limited API until it bans you. Wrong the other way and a transient 503 permanently drops a customer's sync.

Cases to rule on, each of which changes behaviour:

| Case | The question |
|---|---|
| `resp == nil`, `err != nil` | Dial failure or timeout. Retryable, and with what delay? |
| 429 with `Retry-After` | Parse and honour it verbatim, or clamp it to a ceiling? |
| 429 without `Retry-After` | Fall back to the computed backoff, or impose a floor? |
| 500, 502, 503, 504 | Retryable. All of them, or is 500 different from 503? |
| 400, 422 | Bad payload. Dead-letter immediately, or retry once in case it is transient validation? |
| 401 | Dead credential, or a token about to be refreshed by something else? |
| 403 | Permission revoked (terminal), or rate-limit-adjacent for some CRMs? |
| 404 on update | Contact deleted upstream. Terminal, or a signal to drop the ref so the next attempt recreates it? |
| 409 | Concurrent write. Retryable? |

The 404 row is worth a second look: answering "drop the ref" turns a permanent failure into a self-healing one, but needs `classifyHTTPError` to signal that intent back to the worker, which means a field on `ProviderError` this plan has not defined. If you want that, say so and it becomes a small amendment rather than an improvisation.

Write a table-driven test enumerating your decisions, then the body that satisfies it. Remove the `panic`.

- [ ] **Step 4: Run the full suite**

Run: `go test ./plugins/retention/ -v -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add plugins/retention/provider_generic.go plugins/retention/provider_generic_test.go
git commit -m "feat(retention): add the config-driven generic CRM provider"
```

---

## Task 10: HubSpot provider

**Files:**
- Create: `plugins/retention/provider_hubspot.go`
- Test: `plugins/retention/provider_hubspot_test.go`

**Interfaces:**
- Consumes: `Provider` from Task 1, `ProviderConfig` from Task 6, `classifyHTTPError` from Task 9. HubSpot reads `Token` and `BaseURL`.
- Produces: `NewHubSpotProvider(ProviderConfig) (*HubSpotProvider, error)`.

**Confirm every endpoint path, the auth header shape and the search-by-email call against HubSpot's current API docs before writing this file.** The spec says so explicitly, and this plan is not a substitute: vendor APIs move and neither document tracks them. Fetch the docs, then write what they say.

- [ ] **Step 1: Write the failing tests**

Create `plugins/retention/provider_hubspot_test.go`, driving the provider against `httptest.NewServer` with a `BaseURL` override so no test touches the network. Cover: `Capabilities()` reports `CapContacts|CapActivities`; `UpsertContact` on a new email creates and returns the new object id; `UpsertContact` for a known email updates rather than duplicating; a 429 with `Retry-After: 30` comes back as a `ProviderError` with `Retryable` true and `RetryAfter` 30s; `LogActivity` posts an engagement associated with the contact id.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./plugins/retention/ -run TestHubSpot -v`
Expected: FAIL to build, `undefined: NewHubSpotProvider`.

- [ ] **Step 3: Implement the provider**

Give the struct a `BaseURL` field defaulting to the documented production host, so tests can point it at `httptest`. Auth is a bearer private-app token read from `ProviderConfig.Token`. Every non-2xx response goes through `classifyHTTPError`, so HubSpot inherits the classification policy rather than restating it.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./plugins/retention/ -v -race`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add plugins/retention/provider_hubspot.go plugins/retention/provider_hubspot_test.go
git commit -m "feat(retention): add the HubSpot reference provider"
```

---

## Task 11: Data export, wiring and docs

**Files:**
- Create: `plugins/retention/export.go`
- Create: `plugins/retention/doc.go`
- Modify: `testutil/server.go`
- Test: `plugins/retention/export_test.go`

**Interfaces:**
- Consumes: `plugin.DataExportContributor` at `plugin/plugin.go:442`, `Store.GetRef`.
- Produces: `(*Plugin).ExportUserData(ctx, userID) (string, any, error)`.

- [ ] **Step 1: Write the failing test**

Create `plugins/retention/export_test.go`:

```go
package retention

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestExportUserDataIncludesCRMRefs(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	p := New()
	p.store = s

	appID, userID := id.NewAppID(), id.NewUserID()
	require.NoError(t, s.PutRef(ctx, &ContactRef{
		ID: id.NewRetentionRefID(), AppID: appID, UserID: userID,
		Provider: "hubspot", RemoteObjectType: "contact", RemoteID: "501",
		SyncedAt: time.Now(),
	}))

	category, data, err := p.ExportUserData(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "retention", category)
	assert.NotNil(t, data, "a user must be able to see which CRMs hold their record")
}

func TestExportUserDataEmptyWithoutStore(t *testing.T) {
	category, _, err := New().ExportUserData(context.Background(), id.NewUserID())
	require.NoError(t, err)
	assert.Equal(t, "retention", category)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./plugins/retention/ -run TestExport -v`
Expected: FAIL to build, `p.ExportUserData undefined`.

- [ ] **Step 3: Implement the export**

Read the exact `DataExportContributor` signature at `plugin/plugin.go:442` and `plugins/consent/plugin.go:164` for a working example, then create `plugins/retention/export.go` returning the category `"retention"` and the user's contact refs. Add a `ListRefsForUser` method to `Store` and implement it in all four backends, extending the conformance suite with a case for it. A nil store returns an empty payload and no error.

- [ ] **Step 4: Write doc.go and wire the plugin in**

Create `plugins/retention/doc.go` following `plugins/consent/doc.go`, showing registration:

```go
//	    authsome.WithPlugin(retention.New(retention.Config{
//	        Providers: []retention.ProviderConfig{{
//	            Name: "hubspot", Type: "hubspot", Token: os.Getenv("HUBSPOT_TOKEN"),
//	        }},
//	        TickInterval: 15 * time.Second,
//	    })),
```

Add `authsome.WithPlugin(retention.New())` to `testutil/server.go` alongside the other plugins so the integration harness exercises registration.

- [ ] **Step 5: Run the full suite**

Run: `go test ./... 2>&1 | tail -30`
Expected: PASS across the repo, with no new failures in packages this plan did not touch.

- [ ] **Step 6: Commit**

```bash
git add plugins/retention/export.go plugins/retention/export_test.go \
        plugins/retention/doc.go plugins/retention/store*.go testutil/server.go
git commit -m "feat(retention): export CRM refs and register the plugin"
```

---

## Self-review notes

Checked against the spec section by section.

- Provider contract, `Capabilities`, `ProviderError`, `RemoteRef`: Task 1.
- Both tables, four backends, one conformance suite: Tasks 2 to 4.
- Claim with `SKIP LOCKED`, lease reclaim, backoff, `RetryAfter`, dead-letter, user reload at delivery: Task 5.
- Worker heals a missing ref inside the activity job, and hooks perform no reads: Tasks 5 and 7.
- Hook wiring for signup, sign-in, sign-out and user update: Task 7.
- Consent checked at delivery, `require_consent` defaulting false, additive `HasConsent`, `suppressed` state: Tasks 2, 6 and 8.
- Generic and HubSpot providers: Tasks 9 and 10.
- Settings through `SettingsProvider`, `DataExportContributor`: Tasks 6 and 11.
- Out of scope in the spec and absent here: lifecycle state, scoring, the dormancy sweeper, rules, actions, the dashboard, and erasure after a post-sync revocation.

Two things a reader should know rather than discover.

`ListRefsForUser` appears first in Task 11 but extends the `Store` interface defined in Task 2, so Task 11 touches all four backends and the conformance suite again. It sits there because the export is the only caller and splitting it earlier would have added a method nothing used.

Task 9 Step 3 is the one deliberate hole in the plan, and it is a decision rather than a placeholder. The scaffolding, the tests around it and the enumerated cases are all here; the policy is Rex's, and the `panic` in Step 2 makes it impossible to ship the task by accident.
