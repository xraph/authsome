# Non-human principals implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give authsome a first-class principal abstraction covering human users, AI agents and workloads, with an RFC 8693 actor chain, delegation grants, Warden intersection, and risk-plugin visibility into machine traffic.

**Architecture:** A new `principal` package holds value types only (`Ref`, `Chain`, `Principal`, `Delegation`, `AuthAttempt`) and a `Store` interface. Sessions carry a `Subject` derived from existing fields plus an `Actors` chain, which subsumes the current `ImpersonatedBy` field. Non-human principals are stored in the existing `authsome_service_accounts` table, extended with `kind`, `owner_user_id`, `parent_id` and `expires_at`. Authorization goes through a new chain-aware `Engine.Can`, which intersects the subject's decision with every actor's decision.

**Tech Stack:** Go 1.26, grove ORM and grove/migrate, warden v1.6.0 for authorization, forge for HTTP and scope, testify for assertions, testcontainers for postgres and mongo integration tests.

**Spec:** `docs/superpowers/specs/2026-08-24-non-human-principals-design.md`

## Global constraints

- Go 1.26.0, module `github.com/xraph/authsome`.
- Warden is `github.com/xraph/warden v1.6.0`. Subject kinds available: `SubjectUser`, `SubjectAPIKey`, `SubjectService`, `SubjectServiceAcct`.
- Assertions use `github.com/stretchr/testify` (`assert` for soft checks, `require` where a failure makes the rest of the test meaningless). Match the surrounding file.
- The non-human principal table keeps the name `authsome_service_accounts`. Do not rename it.
- The `impersonated_by` database column is retained in this change. Only the Go struct field is removed. A follow-up change drops the column.
- The `impersonated_by` JSON key on `session.Session` and `impersonatedBy` in `extension/contract` must keep appearing on the wire exactly as they do today.
- ID prefix for delegations is `adel`. Prefixes are declared in `id/id.go`.
- Every store change must pass `store/storetest` conformance on all four backends. Memory runs in normal CI. postgres, sqlite and mongo conformance need `go test -tags integration` and Docker.
- No em dashes in code comments or documentation written by this plan.
- Run `make check` before every commit. It runs fmt, vet and lint.

---

## File structure

**New files**

| Path | Responsibility |
|---|---|
| `principal/principal.go` | `Kind`, `Ref`, `Principal`, and their predicates. No store, no context. |
| `principal/chain.go` | `Chain` and its accessors. |
| `principal/delegation.go` | `Delegation`, `GrantKind`, `Store` interface, `ErrNotFound`. |
| `principal/attempt.go` | `AuthAttempt`, the payload for the new plugin hooks. |
| `principal/context.go` | Context carriers so plugins can read the caller without importing `middleware`. |
| `principal/principal_test.go` | Unit tests for the value types. |
| `principal/chain_test.go` | Unit tests for chain accessors. |
| `store/postgres/principal.go` | postgres service-account and delegation store, replacing the stub file. |
| `store/sqlite/principal.go` | sqlite service-account and delegation store, replacing the stub file. |
| `store/mongo/delegations.go` | mongo delegation store. |
| `engine_principal.go` | Engine principal resolution, `Can`, delegation lifecycle. |
| `engine_principal_test.go` | Tests for `Can` intersection and delegation lifecycle. |
| `engine_token_exchange.go` | Token exchange and ephemeral child minting. |
| `api/principal_handlers.go` | HTTP handlers for exchange and children. |

**Modified files**

| Path | Change |
|---|---|
| `id/id.go` | `PrefixDelegation`, `DelegationID`, `NewDelegationID`, `ParseDelegationID`. |
| `serviceaccount/serviceaccount.go` | New fields on the entity, `Kind`, `OwnerUserID`, `ParentID`, `ExpiresAt`, `OrgID`, `EnvID`. |
| `session/session.go` | `Actors`, `ActorGrant`, `DelegationID`. `ImpersonatedBy` field removed, method added. `MarshalJSON` shim. |
| `store/{postgres,sqlite,mongo}/models.go` | Session model maps the chain. Service-account model gains the new columns. |
| `store/{postgres,sqlite,mongo}/migrations.go` | Tables, columns, index and constraint changes. |
| `store/memory/store.go` | Delegation map and principal filters. |
| `store/store.go` | Aggregate `Store` composes `principal.Store`. |
| `store/storetest/storetest.go` | New conformance cases. |
| `service.go` | `Impersonate`, `StopImpersonation`, `HasPermission` delegating to `Can`. |
| `middleware/auth.go`, `middleware/context.go` | Principal resolution and context carriers. |
| `authprovider/session.go`, `extension/contract/handlers_sessions.go` | Read impersonator through the method. |
| `plugin/plugin.go`, `plugin/registry.go` | Engine widening and the two new hooks. |
| `plugins/apikey/plugin.go` | Fire the new hooks, stamp the principal kind. |
| `plugins/riskengine/plugin.go`, `plugins/impossibletravel/plugin.go`, `plugins/anomaly/plugin.go` | Adopt the principal hook. |
| `engine_session_roles.go` | Compare `principal.Kind` instead of a bare string. |

---

## Task 1: The `principal` value types

**Files:**
- Create: `principal/principal.go`
- Create: `principal/chain.go`
- Test: `principal/principal_test.go`, `principal/chain_test.go`

**Interfaces:**
- Consumes: `github.com/xraph/authsome/id` only.
- Produces: `principal.Kind` with constants `KindUser`, `KindAgent`, `KindWorkload`, `KindService`. `principal.Ref{Kind Kind; ID string}` with `String() string`, `IsZero() bool`, and package function `ParseRef(string) (Ref, error)`. `principal.Principal` struct with `IsExpired(time.Time) bool` and `IsActive(time.Time) bool`. `principal.Chain []Ref` with `Actor() (Ref, bool)`, `Root() (Ref, bool)`, `Depth() int`, `Contains(Ref) bool`.

- [ ] **Step 1: Write the failing tests**

Create `principal/principal_test.go`:

```go
package principal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/principal"
)

func TestRefStringRoundTrips(t *testing.T) {
	r := principal.Ref{Kind: principal.KindAgent, ID: "svc_01h2xcejqtf2nbrexx3vqjhp41"}
	assert.Equal(t, "agent:svc_01h2xcejqtf2nbrexx3vqjhp41", r.String())

	got, err := principal.ParseRef(r.String())
	require.NoError(t, err)
	assert.Equal(t, r, got)
}

func TestParseRefRejectsMalformed(t *testing.T) {
	for _, in := range []string{"", "agent", "agent:", ":svc_1", "nosuchkind:svc_1"} {
		_, err := principal.ParseRef(in)
		assert.Error(t, err, "ParseRef(%q) must fail", in)
	}
}

// An ID containing a colon must not be truncated. Refs are compared to make
// authorization decisions, so a split on the wrong colon would silently
// address a different principal.
func TestParseRefSplitsOnFirstColonOnly(t *testing.T) {
	got, err := principal.ParseRef("workload:svc_1:extra")
	require.NoError(t, err)
	assert.Equal(t, principal.KindWorkload, got.Kind)
	assert.Equal(t, "svc_1:extra", got.ID)
}

func TestZeroRef(t *testing.T) {
	assert.True(t, principal.Ref{}.IsZero())
	assert.False(t, principal.Ref{Kind: principal.KindUser, ID: "ausr_1"}.IsZero())
}

func TestPrincipalExpiry(t *testing.T) {
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	past := at.Add(-time.Hour)
	future := at.Add(time.Hour)

	durable := principal.Principal{Ref: principal.Ref{Kind: principal.KindWorkload, ID: "svc_1"}}
	assert.False(t, durable.IsExpired(at), "a nil ExpiresAt means durable")
	assert.True(t, durable.IsActive(at))

	lapsed := principal.Principal{
		Ref:       principal.Ref{Kind: principal.KindAgent, ID: "svc_2"},
		ExpiresAt: &past,
	}
	assert.True(t, lapsed.IsExpired(at))
	assert.False(t, lapsed.IsActive(at))

	live := principal.Principal{
		Ref:       principal.Ref{Kind: principal.KindAgent, ID: "svc_3"},
		ExpiresAt: &future,
	}
	assert.False(t, live.IsExpired(at))
	assert.True(t, live.IsActive(at))

	disabled := principal.Principal{
		Ref:      principal.Ref{Kind: principal.KindAgent, ID: "svc_4"},
		Disabled: true,
	}
	assert.False(t, disabled.IsActive(at), "a disabled principal is never active")
}
```

Create `principal/chain_test.go`:

```go
package principal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/principal"
)

func TestEmptyChain(t *testing.T) {
	var c principal.Chain
	_, ok := c.Actor()
	assert.False(t, ok)
	_, ok = c.Root()
	assert.False(t, ok)
	assert.Equal(t, 0, c.Depth())
}

// Chains are ordered nearest-caller-first, so Actor is the immediate caller
// and Root is the outermost hop. A multi-hop chain is an ephemeral child
// acting through its registered parent.
func TestChainOrdering(t *testing.T) {
	child := principal.Ref{Kind: principal.KindAgent, ID: "svc_child"}
	parent := principal.Ref{Kind: principal.KindAgent, ID: "svc_parent"}
	c := principal.Chain{child, parent}

	got, ok := c.Actor()
	assert.True(t, ok)
	assert.Equal(t, child, got, "Actor must be the immediate caller")

	got, ok = c.Root()
	assert.True(t, ok)
	assert.Equal(t, parent, got, "Root must be the outermost hop")

	assert.Equal(t, 2, c.Depth())
	assert.True(t, c.Contains(parent))
	assert.False(t, c.Contains(principal.Ref{Kind: principal.KindUser, ID: "ausr_1"}))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./principal/ -v`
Expected: FAIL, the package does not exist yet.

- [ ] **Step 3: Write `principal/principal.go`**

```go
// Package principal defines the caller abstraction shared by human users,
// AI agents and workloads. It holds value types only: no store access, no
// HTTP, no engine. Everything above it in the import graph (session, store,
// middleware, plugin) can depend on this package without cycles.
package principal

import (
	"fmt"
	"strings"
	"time"

	"github.com/xraph/authsome/id"
)

// Kind is the sort of caller a principal is.
type Kind string

const (
	// KindUser is a human with an account.
	KindUser Kind = "user"
	// KindAgent is an AI agent or MCP client acting under a registration.
	KindAgent Kind = "agent"
	// KindWorkload is a machine caller with no human behind it: a CI job, a
	// cron, a service calling another service.
	KindWorkload Kind = "workload"
	// KindService is the kind sessions written before agents and workloads
	// existed carry. Retained so those rows keep resolving.
	KindService Kind = "service_account"
)

// IsHuman reports whether k denotes a person.
func (k Kind) IsHuman() bool { return k == KindUser }

// Valid reports whether k is a kind this package knows.
func (k Kind) Valid() bool {
	switch k {
	case KindUser, KindAgent, KindWorkload, KindService:
		return true
	default:
		return false
	}
}

// Ref is the addressable identity of a principal. It is comparable, so it can
// be a map key and can be compared with ==, and it is cheap enough to put on a
// context and to serialize into a token claim.
type Ref struct {
	Kind Kind   `json:"kind"`
	ID   string `json:"id"`
}

// String renders the ref as "kind:id".
func (r Ref) String() string {
	if r.IsZero() {
		return ""
	}
	return string(r.Kind) + ":" + r.ID
}

// IsZero reports whether r addresses nothing.
func (r Ref) IsZero() bool { return r.Kind == "" || r.ID == "" }

// ParseRef parses the "kind:id" form produced by Ref.String.
//
// It splits on the first colon only. A TypeID does not contain one today, but
// refs are compared to make authorization decisions, and splitting on the last
// colon would let an id with a colon in it address a different principal than
// the one that was written.
func ParseRef(s string) (Ref, error) {
	kindStr, idStr, found := strings.Cut(s, ":")
	if !found {
		return Ref{}, fmt.Errorf("principal: parse ref %q: missing kind separator", s)
	}
	if idStr == "" {
		return Ref{}, fmt.Errorf("principal: parse ref %q: empty id", s)
	}
	kind := Kind(kindStr)
	if !kind.Valid() {
		return Ref{}, fmt.Errorf("principal: parse ref %q: unknown kind %q", s, kindStr)
	}
	return Ref{Kind: kind, ID: idStr}, nil
}

// UserRef builds a ref for a human user.
func UserRef(userID id.UserID) Ref {
	return Ref{Kind: KindUser, ID: userID.String()}
}

// Principal is a resolved caller carrying everything an authorization decision
// needs, so callers do not go back to the store mid-decision.
type Principal struct {
	Ref

	AppID  id.AppID
	OrgID  id.OrgID
	EnvID  id.EnvironmentID
	Name   string
	Scopes []string
	Roles  []string

	// Owner is the principal answerable for this one. Nil for users, who are
	// answerable for themselves.
	Owner *Ref
	// Parent is the registered principal that minted this one. Set only on
	// ephemeral children.
	Parent *Ref
	// ExpiresAt is a hard cutoff. Nil means durable.
	ExpiresAt *time.Time
	Disabled  bool
}

// IsExpired reports whether p has passed its cutoff at time at.
func (p *Principal) IsExpired(at time.Time) bool {
	return p.ExpiresAt != nil && at.After(*p.ExpiresAt)
}

// IsActive reports whether p may authenticate at time at.
func (p *Principal) IsActive(at time.Time) bool {
	return !p.Disabled && !p.IsExpired(at)
}

// IsEphemeral reports whether p was minted by another principal.
func (p *Principal) IsEphemeral() bool { return p.Parent != nil }
```

- [ ] **Step 4: Write `principal/chain.go`**

```go
package principal

// Chain is an actor chain, ordered nearest-caller-first.
//
// It follows RFC 8693. The session's subject is who the request is for, and
// the chain is who is doing the acting on their behalf. Element 0 is the
// immediate caller. A chain of length two is an ephemeral child acting
// through its registered parent. An empty chain means the subject is calling
// directly, which is the ordinary sign-in case.
type Chain []Ref

// Actor returns the immediate caller.
func (c Chain) Actor() (Ref, bool) {
	if len(c) == 0 {
		return Ref{}, false
	}
	return c[0], true
}

// Root returns the outermost hop, the principal furthest from the subject.
func (c Chain) Root() (Ref, bool) {
	if len(c) == 0 {
		return Ref{}, false
	}
	return c[len(c)-1], true
}

// Depth returns how many actors stand between the caller and the subject.
func (c Chain) Depth() int { return len(c) }

// Contains reports whether r appears anywhere in the chain.
func (c Chain) Contains(r Ref) bool {
	for _, got := range c {
		if got == r {
			return true
		}
	}
	return false
}

// Prepend returns a chain with r as the new immediate caller. The receiver is
// not modified, so a chain read off a session can be extended without the
// extension leaking back into the session.
func (c Chain) Prepend(r Ref) Chain {
	out := make(Chain, 0, len(c)+1)
	out = append(out, r)
	out = append(out, c...)
	return out
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./principal/ -v`
Expected: PASS, all cases.

- [ ] **Step 6: Commit**

```bash
make check
git add principal/
git commit -m "feat(principal): add principal ref, chain and kind value types"
```

---

## Task 2: Delegation entity, store interface, and the `adel` id prefix

**Files:**
- Create: `principal/delegation.go`
- Create: `principal/attempt.go`
- Create: `principal/context.go`
- Create: `principal/delegation_test.go`
- Modify: `id/id.go`

**Interfaces:**
- Consumes: Task 1's `Ref`, `Chain`, `Kind`.
- Produces: `principal.GrantKind` with `GrantDelegation` and `GrantImpersonation`. `principal.Delegation` with `IsActive(time.Time) bool` and `AllowsScope(string) bool`. `principal.Store` interface with `CreateDelegation`, `GetDelegation`, `FindActiveDelegation`, `ListDelegationsForSubject`, `ListDelegationsForActor`, `RevokeDelegation`, plus the principal read side `GetPrincipal` and `ListPrincipals`. `principal.AuthAttempt`. `principal.NewContext`, `principal.FromContext`, `principal.NewActorsContext`, `principal.ActorsFromContext`. `id.DelegationID`, `id.NewDelegationID`, `id.ParseDelegationID`, `id.PrefixDelegation`.

- [ ] **Step 1: Write the failing test**

Create `principal/delegation_test.go`:

```go
package principal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

func TestDelegationIsActive(t *testing.T) {
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	past := at.Add(-time.Hour)
	future := at.Add(time.Hour)

	base := principal.Delegation{
		ID:        id.NewDelegationID(),
		Actor:     principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		Subject:   principal.Ref{Kind: principal.KindUser, ID: "ausr_1"},
		GrantKind: principal.GrantDelegation,
	}

	assert.True(t, base.IsActive(at), "no expiry and no revocation means active")

	expired := base
	expired.ExpiresAt = &past
	assert.False(t, expired.IsActive(at))

	live := base
	live.ExpiresAt = &future
	assert.True(t, live.IsActive(at))

	// A revoked grant stays dead even when its expiry is still in the future.
	revoked := live
	revoked.RevokedAt = &past
	assert.False(t, revoked.IsActive(at), "revocation must beat a future expiry")
}

// An empty scope list on a grant means the grant places no scope restriction
// of its own. Narrowing then comes from the actor's own scopes alone. An
// empty list must not be read as "deny everything", which would make every
// grant created without an explicit scope useless.
func TestDelegationAllowsScope(t *testing.T) {
	unrestricted := principal.Delegation{}
	assert.True(t, unrestricted.AllowsScope("repo:write"))

	narrow := principal.Delegation{Scopes: []string{"repo:read", "issues:read"}}
	assert.True(t, narrow.AllowsScope("repo:read"))
	assert.False(t, narrow.AllowsScope("repo:write"))
}

func TestDelegationIDPrefix(t *testing.T) {
	d := id.NewDelegationID()
	parsed, err := id.ParseDelegationID(d.String())
	assert.NoError(t, err)
	assert.Equal(t, d.String(), parsed.String())

	_, err = id.ParseDelegationID(id.NewUserID().String())
	assert.Error(t, err, "a user id must not parse as a delegation id")
}

func TestPrincipalContextRoundTrip(t *testing.T) {
	ctx := context.Background()

	_, ok := principal.FromContext(ctx)
	assert.False(t, ok, "a bare context carries no principal")

	p := &principal.Principal{Ref: principal.Ref{Kind: principal.KindAgent, ID: "svc_1"}}
	ctx = principal.NewContext(ctx, p)
	got, ok := principal.FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, p, got)

	chain := principal.Chain{{Kind: principal.KindAgent, ID: "svc_1"}}
	ctx = principal.NewActorsContext(ctx, chain)
	gotChain, ok := principal.ActorsFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, chain, gotChain)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./principal/ -run 'Delegation|Context' -v`
Expected: FAIL, undefined `principal.Delegation` and `id.NewDelegationID`.

- [ ] **Step 3: Add the id prefix**

In `id/id.go`, add to the prefix const block next to `PrefixServiceAccount`:

```go
	PrefixDelegation      Prefix = "adel"
```

Add the type alias next to `ServiceAccountID`:

```go
// DelegationID is a type-safe identifier for delegation grants (prefix: "adel").
// A delegation records that one principal may act on behalf of another.
type DelegationID = ID
```

Add the constructor next to `NewServiceAccountID`:

```go
// NewDelegationID generates a new unique delegation ID.
func NewDelegationID() ID { return New(PrefixDelegation) }
```

Add the parser next to `ParseServiceAccountID`:

```go
// ParseDelegationID parses a string and validates the "adel" prefix.
func ParseDelegationID(s string) (ID, error) { return ParseWithPrefix(s, PrefixDelegation) }
```

- [ ] **Step 4: Write `principal/delegation.go`**

```go
package principal

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when a principal or delegation does not exist.
var ErrNotFound = errors.New("principal: not found")

// GrantKind distinguishes the two ways one principal comes to act for another.
type GrantKind string

const (
	// GrantDelegation is an agent or workload acting for a user who granted
	// it that authority. Both parties are checked, so the decision narrows.
	GrantDelegation GrantKind = "delegation"
	// GrantImpersonation is an admin acting as a user. The actor is not
	// independently checked, because impersonating somebody is precisely the
	// request to evaluate as them. The gate sits on the Impersonate call.
	GrantImpersonation GrantKind = "impersonation"
)

// Delegation records that Actor may act on behalf of Subject.
//
// It is the durable, revocable, auditable half of the actor chain. The chain
// on a session says who is acting. The grant says they were allowed to.
type Delegation struct {
	ID    id.DelegationID `json:"id"`
	AppID id.AppID        `json:"app_id"`
	OrgID id.OrgID        `json:"org_id,omitempty"`

	Actor   Ref `json:"actor"`
	Subject Ref `json:"subject"`

	GrantKind GrantKind `json:"grant_kind"`
	// Scopes narrows what the actor may do while acting. An empty list places
	// no restriction of its own, leaving the actor's own scopes as the only
	// limit.
	Scopes []string `json:"scopes,omitempty"`
	// GrantedBy is who consented. For a delegation that is normally the
	// subject; for an impersonation it is the admin who initiated it.
	GrantedBy Ref `json:"granted_by"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// IsActive reports whether d may be exercised at time at.
func (d *Delegation) IsActive(at time.Time) bool {
	if d.RevokedAt != nil {
		return false
	}
	return d.ExpiresAt == nil || !at.After(*d.ExpiresAt)
}

// AllowsScope reports whether scope falls inside d's scope filter.
func (d *Delegation) AllowsScope(scope string) bool {
	if len(d.Scopes) == 0 {
		return true
	}
	for _, s := range d.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// DelegationQuery filters a delegation listing.
type DelegationQuery struct {
	AppID       id.AppID
	OrgID       id.OrgID
	Actor       *Ref
	Subject     *Ref
	GrantKind   GrantKind
	ActiveOnly  bool
	ActiveAsOf  time.Time
	Limit       int
}

// Query filters a principal listing.
type Query struct {
	AppID      id.AppID
	Kind       Kind
	OwnerUser  *id.UserID
	Parent     *Ref
	ActiveOnly bool
	ActiveAsOf time.Time
	Limit      int
}

// Store is the persistence interface for principals and their delegations.
//
// The principal read side reads the same rows the serviceaccount store
// writes. It is separate because callers resolving a caller mid-request want a
// Principal and not a ServiceAccount, and because a user principal is
// assembled from a different table entirely.
type Store interface {
	// GetPrincipal resolves any principal by ref. Users are assembled from
	// the user table; every other kind comes from the service accounts table.
	GetPrincipal(ctx context.Context, ref Ref) (*Principal, error)
	// ListPrincipals returns principals matching q.
	ListPrincipals(ctx context.Context, q *Query) ([]*Principal, error)

	// CreateDelegation stores a new grant.
	CreateDelegation(ctx context.Context, d *Delegation) error
	// GetDelegation returns a grant by ID.
	GetDelegation(ctx context.Context, delID id.DelegationID) (*Delegation, error)
	// FindActiveDelegation returns the live grant letting actor act for
	// subject under grantKind, or ErrNotFound.
	FindActiveDelegation(ctx context.Context, appID id.AppID, actor, subject Ref, grantKind GrantKind) (*Delegation, error)
	// ListDelegations returns grants matching q.
	ListDelegations(ctx context.Context, q *DelegationQuery) ([]*Delegation, error)
	// RevokeDelegation marks a grant revoked at the given time. Revoking an
	// already-revoked grant is not an error.
	RevokeDelegation(ctx context.Context, delID id.DelegationID, at time.Time) error
}
```

- [ ] **Step 5: Write `principal/attempt.go`**

```go
package principal

import (
	"time"

	"github.com/xraph/authsome/id"
)

// AuthAttempt describes a credential being turned into a session.
//
// It is the payload the risk plugins receive for machine callers. Sign-in
// carries an account.SignInRequest, which has an email and a password on it
// and means nothing for an agent. This is the machine-side equivalent: enough
// for a risk contributor to score, with nothing on it that only a human has.
type AuthAttempt struct {
	Subject Ref
	Actors  Chain

	AppID id.AppID
	EnvID id.EnvironmentID
	OrgID id.OrgID

	// CredentialKind is how the caller authenticated: "api_key",
	// "token_exchange" or "jwt".
	CredentialKind string
	// CredentialID identifies the specific credential, so a verdict can be
	// cached against it and a compromised one can be traced.
	CredentialID string

	IPAddress string
	UserAgent string

	// Ephemeral is true when the subject was minted by another principal
	// rather than registered.
	Ephemeral bool

	At time.Time
}
```

- [ ] **Step 6: Write `principal/context.go`**

```go
package principal

import "context"

// Context keys are unexported struct types so no other package can collide
// with them, which is the same convention middleware/context.go uses.
type principalCtxKey struct{}
type actorsCtxKey struct{}

// NewContext returns ctx carrying p as the resolved caller.
func NewContext(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, p)
}

// FromContext returns the resolved caller.
//
// This lives here rather than only in middleware so a plugin can read the
// caller without importing middleware, which would pull in forge and the
// whole HTTP surface.
func FromContext(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(principalCtxKey{}).(*Principal)
	return p, ok
}

// NewActorsContext returns ctx carrying the actor chain.
func NewActorsContext(ctx context.Context, c Chain) context.Context {
	return context.WithValue(ctx, actorsCtxKey{}, c)
}

// ActorsFromContext returns the actor chain, if the caller is acting for
// somebody else.
func ActorsFromContext(ctx context.Context) (Chain, bool) {
	c, ok := ctx.Value(actorsCtxKey{}).(Chain)
	return c, ok
}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./principal/ ./id/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
make check
git add principal/ id/id.go
git commit -m "feat(principal): add delegation grants, auth attempts and context carriers"
```

---

## Task 3: Session carries the actor chain, `ImpersonatedBy` becomes a method

This is the compile-breaking task. Nothing persists a chain yet. The four store
models keep round-tripping impersonation through the existing
`impersonated_by` column, so behaviour is unchanged and no migration is needed.

**Files:**
- Modify: `session/session.go`
- Modify: `service.go:2701`, `service.go:2738`, `service.go:2746`
- Modify: `middleware/auth.go:222-223`, `middleware/auth.go:630-631`
- Modify: `authprovider/session.go:188-189`
- Modify: `extension/contract/handlers_sessions.go:175-176`
- Modify: `store/postgres/models.go`, `store/sqlite/models.go`, `store/mongo/models.go`
- Modify: `engine_session_roles.go`
- Test: `session/session_test.go` (create)

**Interfaces:**
- Consumes: Task 1's `principal.Kind`, `principal.Ref`, `principal.Chain`; Task 2's `principal.GrantKind`, `id.DelegationID`.
- Produces: `session.Session.Actors principal.Chain`, `session.Session.ActorGrant principal.GrantKind`, `session.Session.DelegationID id.DelegationID`. `session.Session.PrincipalKind` retyped from `string` to `principal.Kind`. Methods `Subject() principal.Ref`, `ImpersonatedBy() id.UserID`, `SetImpersonatedBy(id.UserID)`, `IsHumanPrincipal() bool`, `MarshalJSON`, `UnmarshalJSON`.

- [ ] **Step 1: Write the failing test**

Create `session/session_test.go`:

```go
package session_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// Impersonation is now one shape of actor chain rather than its own field.
// Setting it must produce a chain, and reading it back must find the admin.
func TestImpersonationRoundTripsThroughTheChain(t *testing.T) {
	admin := id.NewUserID()
	target := id.NewUserID()

	s := &session.Session{UserID: target}
	assert.True(t, s.ImpersonatedBy().IsNil(), "a plain session has no impersonator")

	s.SetImpersonatedBy(admin)
	assert.Equal(t, admin.String(), s.ImpersonatedBy().String())
	assert.Equal(t, principal.GrantImpersonation, s.ActorGrant)

	actor, ok := s.Actors.Actor()
	require.True(t, ok)
	assert.Equal(t, principal.Ref{Kind: principal.KindUser, ID: admin.String()}, actor)
}

// A delegation chain is not impersonation. An agent acting for a user must
// not surface as that user's impersonator, or every audit record and every
// admin banner reads the delegation as an admin takeover.
func TestDelegationChainIsNotImpersonation(t *testing.T) {
	s := &session.Session{
		UserID:     id.NewUserID(),
		ActorGrant: principal.GrantDelegation,
		Actors:     principal.Chain{{Kind: principal.KindAgent, ID: "svc_1"}},
	}
	assert.True(t, s.ImpersonatedBy().IsNil(), "a delegation must not read as impersonation")
}

func TestSubjectDerivesFromPrincipalKind(t *testing.T) {
	uid := id.NewUserID()
	human := &session.Session{UserID: uid}
	assert.Equal(t, principal.Ref{Kind: principal.KindUser, ID: uid.String()}, human.Subject(),
		"an unset PrincipalKind means user")
	assert.True(t, human.IsHumanPrincipal())

	svcID := id.NewServiceAccountID()
	machine := &session.Session{PrincipalKind: principal.KindService, ServiceAccountID: svcID}
	assert.Equal(t, principal.Ref{Kind: principal.KindService, ID: svcID.String()}, machine.Subject())
	assert.False(t, machine.IsHumanPrincipal())

	agent := &session.Session{PrincipalKind: principal.KindAgent, ServiceAccountID: svcID}
	assert.Equal(t, principal.Ref{Kind: principal.KindAgent, ID: svcID.String()}, agent.Subject())
	assert.False(t, agent.IsHumanPrincipal())
}

// The wire format must not move. Removing the struct field would silently drop
// impersonated_by from every serialized session.
func TestImpersonatedByStaysOnTheWire(t *testing.T) {
	admin := id.NewUserID()
	s := &session.Session{UserID: id.NewUserID()}
	s.SetImpersonatedBy(admin)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, admin.String(), decoded["impersonated_by"])

	plain, err := json.Marshal(&session.Session{UserID: id.NewUserID()})
	require.NoError(t, err)
	var plainDecoded map[string]any
	require.NoError(t, json.Unmarshal(plain, &plainDecoded))
	_, present := plainDecoded["impersonated_by"]
	assert.False(t, present, "an unimpersonated session must omit the key, as it does today")
}

func TestSessionJSONRoundTrip(t *testing.T) {
	admin := id.NewUserID()
	s := &session.Session{UserID: id.NewUserID()}
	s.SetImpersonatedBy(admin)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var back session.Session
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, admin.String(), back.ImpersonatedBy().String())
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./session/ -v`
Expected: FAIL, `s.ImpersonatedBy` is a field and not a method, and `SetImpersonatedBy` is undefined.

- [ ] **Step 3: Rewrite `session/session.go`**

Replace the `ImpersonatedBy` field, retype `PrincipalKind`, and add the new fields and methods. The full set of changes to the struct:

```go
// Remove this line entirely:
//     ImpersonatedBy        id.UserID          `json:"impersonated_by,omitempty"`

// Retype this field. It was `string`.
	PrincipalKind principal.Kind `json:"principal_kind,omitempty"`

// Add after ServiceAccountID:

	// Actors is the chain of principals acting on the subject's behalf,
	// ordered nearest-caller-first. Empty on an ordinary session, where the
	// subject is calling directly.
	//
	// This subsumes the old ImpersonatedBy field. An impersonation is a chain
	// of one user actor with ActorGrant set to impersonation, which is why
	// ImpersonatedBy is now derived rather than stored.
	Actors principal.Chain `json:"actors,omitempty"`

	// ActorGrant records which sort of grant put the actors on this session.
	// Empty when Actors is empty.
	ActorGrant principal.GrantKind `json:"actor_grant,omitempty"`

	// DelegationID is the grant this session was minted against. Zero for an
	// ordinary session.
	DelegationID id.DelegationID `json:"delegation_id,omitempty"`
```

Add the methods:

```go
// Subject returns the principal this session is for.
//
// An empty PrincipalKind means "user", which is what every row written before
// the field existed carries. Normalizing it on read would make a legacy row
// indistinguishable from one deliberately stamped, so the zero value is
// interpreted here and not rewritten in the store.
func (s *Session) Subject() principal.Ref {
	switch s.PrincipalKind {
	case "", principal.KindUser:
		return principal.Ref{Kind: principal.KindUser, ID: s.UserID.String()}
	default:
		return principal.Ref{Kind: s.PrincipalKind, ID: s.ServiceAccountID.String()}
	}
}

// IsHumanPrincipal reports whether a person owns this session.
func (s *Session) IsHumanPrincipal() bool {
	return s.PrincipalKind == "" || s.PrincipalKind == principal.KindUser
}

// ImpersonatedBy returns the admin acting as this session's user, or the zero
// ID when nobody is.
//
// Derived from Actors rather than stored. The grant kind is checked first: an
// agent acting for a user is also a session with two principals, and reporting
// that as impersonation would put an admin-takeover banner and an admin-
// severity audit record on ordinary delegated traffic.
func (s *Session) ImpersonatedBy() id.UserID {
	if s.ActorGrant != principal.GrantImpersonation {
		return id.Nil
	}
	for i := len(s.Actors) - 1; i >= 0; i-- {
		if s.Actors[i].Kind != principal.KindUser {
			continue
		}
		uid, err := id.ParseUserID(s.Actors[i].ID)
		if err != nil {
			continue
		}
		return uid
	}
	return id.Nil
}

// SetImpersonatedBy marks this session as adminID acting as its user.
func (s *Session) SetImpersonatedBy(adminID id.UserID) {
	if adminID.IsNil() {
		return
	}
	s.Actors = principal.Chain{{Kind: principal.KindUser, ID: adminID.String()}}
	s.ActorGrant = principal.GrantImpersonation
}

// MarshalJSON keeps impersonated_by on the wire now that it is no longer a
// struct field. Consumers outside this repository read that key, and the
// chain is an addition for them, not a replacement.
func (s Session) MarshalJSON() ([]byte, error) {
	type alias Session
	out := struct {
		alias
		ImpersonatedBy string `json:"impersonated_by,omitempty"`
	}{alias: alias(s)}
	if imp := s.ImpersonatedBy(); !imp.IsNil() {
		out.ImpersonatedBy = imp.String()
	}
	return json.Marshal(out)
}

// UnmarshalJSON accepts either representation: a payload carrying only the
// legacy impersonated_by key rebuilds the chain from it, and one carrying
// actors is taken as written.
func (s *Session) UnmarshalJSON(data []byte) error {
	type alias Session
	in := struct {
		*alias
		ImpersonatedBy string `json:"impersonated_by,omitempty"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if len(s.Actors) == 0 && in.ImpersonatedBy != "" {
		uid, err := id.ParseUserID(in.ImpersonatedBy)
		if err != nil {
			return fmt.Errorf("session: parse impersonated_by: %w", err)
		}
		s.SetImpersonatedBy(uid)
	}
	return nil
}
```

Add `encoding/json`, `fmt` and `github.com/xraph/authsome/principal` to the imports.

- [ ] **Step 4: Run the session tests to verify they pass**

Run: `go test ./session/ -v`
Expected: PASS.

- [ ] **Step 5: Fix every in-repo consumer**

Run `go build ./...` and work through the failures. The complete set:

`service.go:2701`, inside `Impersonate`:

```go
	sess.SetImpersonatedBy(adminID)
```

`service.go:2738`, inside `StopImpersonation`:

```go
	if sess.ImpersonatedBy().Prefix() == "" {
		return fmt.Errorf("authsome: session is not an impersonation session")
	}
```

`service.go:2746`, the audit actor argument:

```go
	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "stop_impersonation", "session", sessionID.String(), sess.ImpersonatedBy().String(), sess.AppID.String(), "admin", map[string]string{
```

`middleware/auth.go:222` and `middleware/auth.go:630`, both instances:

```go
			if imp := sess.ImpersonatedBy(); imp.Prefix() != "" {
				goCtx = WithImpersonator(goCtx, imp)
			}
```

`authprovider/session.go:188`:

```go
	if imp := data.Session.ImpersonatedBy(); imp.Prefix() != "" {
		goCtx = authmw.WithImpersonator(goCtx, imp)
	}
```

`extension/contract/handlers_sessions.go:175`:

```go
		if imp := s.ImpersonatedBy(); !imp.IsNil() {
			d.ImpersonatedBy = imp.String()
		}
```

`middleware/auth.go:570`, which compared a bare string:

```go
	isServiceAccount := result.Session != nil && !result.Session.IsHumanPrincipal()
```

- [ ] **Step 6: Update the three SQL and document models**

In each of `store/postgres/models.go`, `store/sqlite/models.go` and `store/mongo/models.go`, the model struct keeps its `ImpersonatedBy string` field and its column tag. Only the conversions change.

In `toSession` (model to domain), replace the assignment to the removed field:

```go
	if m.ImpersonatedBy != "" {
		impID, err := id.ParseUserID(m.ImpersonatedBy)
		if err != nil {
			return nil, err
		}
		s.SetImpersonatedBy(impID)
	}
```

In `fromSession` (domain to model), replace the read of the removed field:

```go
	if imp := s.ImpersonatedBy(); imp.Prefix() != "" {
		m.ImpersonatedBy = imp.String()
	}
```

Also retype the `PrincipalKind` conversions, since the domain field is now `principal.Kind` while the column stays `string`:

```go
	// model to domain
	PrincipalKind: principal.Kind(m.PrincipalKind),

	// domain to model
	PrincipalKind: string(s.PrincipalKind),
```

Add `github.com/xraph/authsome/principal` to each file's imports.

- [ ] **Step 7: Update `engine_session_roles.go`**

Delete the `principalKindServiceAccount` constant and its doc comment. It named one kind, and there are now three non-human ones. Replace both comparisons:

```go
// in shouldRestamp
	case !sess.IsHumanPrincipal():
		return false

// in shouldStamp
	case !sess.IsHumanPrincipal():
		// Authorized by scope, and UserID is the zero value here.
		return false
```

- [ ] **Step 8: Update the apikey plugin's synthetic session**

`plugins/apikey/plugin.go:573`, which wrote a bare string:

```go
			PrincipalKind:    principal.KindService,
```

Add `github.com/xraph/authsome/principal` to that file's imports.

- [ ] **Step 9: Update the storetest fixtures that write a bare string**

In `store/storetest/storetest.go`, the two fixtures at the `PrincipalKind` assignments become typed:

```go
		PrincipalKind:         principal.KindUser,
		...
		PrincipalKind:         principal.KindService,
```

and the two assertions:

```go
			assert.Equal(t, principal.KindUser, got.PrincipalKind, "PrincipalKind must survive the round trip")
			...
			assert.Equal(t, principal.KindService, got.PrincipalKind, "PrincipalKind must survive the round trip")
```

Add the import.

- [ ] **Step 10: Run the full suite**

Run: `go build ./... && go test ./... `
Expected: PASS. In particular `go test ./store/memory/ -run TestConformance -v` must still pass every existing case, and any impersonation test in `service_test.go` and `e2e_test.go` must pass unchanged.

- [ ] **Step 11: Commit**

```bash
make check
git add -A
git commit -m "refactor(session): derive ImpersonatedBy from an actor chain

The field becomes a method reading Actors, so impersonation stops being a
special case beside the general two-principal session and becomes one shape
of it. The impersonated_by column and JSON key are unchanged: the store
models write the column from the chain, and a MarshalJSON shim keeps the key
on the wire."
```

---

## Task 4: Extend the principal entity, implement memory, add conformance cases

After this task, `go test ./store/memory/` is green and normal CI stays green.
The postgres, sqlite and mongo conformance suites are integration-tagged and
will fail on the new cases until Tasks 5, 6 and 7 land. That window is expected
and is called out in each of those tasks.

**Files:**
- Modify: `serviceaccount/serviceaccount.go`
- Modify: `store/store.go`
- Modify: `store/memory/store.go`
- Modify: `store/storetest/storetest.go`
- Create: `store/postgres/principal.go`, `store/sqlite/principal.go`, `store/mongo/delegations.go` (stubs at this stage)

**Interfaces:**
- Consumes: Task 2's `principal.Store`, `principal.Delegation`, `principal.Query`, `principal.DelegationQuery`.
- Produces: `serviceaccount.ServiceAccount` gains `Kind principal.Kind`, `OwnerUserID id.UserID`, `ParentID id.ServiceAccountID`, `ExpiresAt *time.Time`, `OrgID id.OrgID`, `EnvID id.EnvironmentID`, and a `ToPrincipal() *principal.Principal` method. `store.Store` composes `principal.Store`. Conformance case names `PrincipalRoundTrip`, `EphemeralPrincipalExpiry`, `DelegationLifecycle`, `SessionActorChainRoundTrip`.

- [ ] **Step 1: Extend the entity**

In `serviceaccount/serviceaccount.go`, add to `ServiceAccount`:

```go
	// Kind is which sort of non-human principal this is. Rows written before
	// the column existed carry the empty string, which reads as
	// principal.KindService.
	Kind principal.Kind `json:"kind,omitempty"`
	// OwnerUserID is the human answerable for this principal. Zero for a
	// workload that nobody owns personally, such as a CI runner.
	OwnerUserID id.UserID `json:"owner_user_id,omitempty"`
	// ParentID is the registered principal that minted this one. Set only on
	// ephemeral children.
	ParentID id.ServiceAccountID `json:"parent_id,omitempty"`
	// ExpiresAt is a hard cutoff. Nil means durable.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	OrgID     id.OrgID          `json:"org_id,omitempty"`
	EnvID     id.EnvironmentID  `json:"env_id,omitempty"`
```

Add the conversion, which is the single place the empty-kind fallback is decided:

```go
// ToPrincipal renders svc as a resolved principal.
//
// An empty Kind reads as service_account, matching rows written before the
// column existed. That fallback lives here so the four store backends cannot
// each pick a different one.
func (svc *ServiceAccount) ToPrincipal() *principal.Principal {
	kind := svc.Kind
	if kind == "" {
		kind = principal.KindService
	}
	p := &principal.Principal{
		Ref:       principal.Ref{Kind: kind, ID: svc.ID.String()},
		AppID:     svc.AppID,
		OrgID:     svc.OrgID,
		EnvID:     svc.EnvID,
		Name:      svc.Name,
		Scopes:    svc.Scopes,
		ExpiresAt: svc.ExpiresAt,
		Disabled:  !svc.Active,
	}
	if !svc.OwnerUserID.IsNil() {
		owner := principal.UserRef(svc.OwnerUserID)
		p.Owner = &owner
	}
	if !svc.ParentID.IsNil() {
		parent := principal.Ref{Kind: kind, ID: svc.ParentID.String()}
		p.Parent = &parent
	}
	return p
}
```

Add `github.com/xraph/authsome/principal` to the imports. Extend `serviceaccount.Query` with `Kind principal.Kind` and `ActiveOnly bool`.

- [ ] **Step 2: Compose the store interface**

In `store/store.go`, add to the `Store` interface next to `serviceaccount.Store`:

```go
	principal.Store
```

Add the import. Run `go build ./...` and confirm the four backends now fail to compile, which is the signal the interface took.

- [ ] **Step 3: Write the failing conformance cases**

In `store/storetest/storetest.go`, register four cases in `RunConformance`:

```go
		{"PrincipalRoundTrip", testPrincipalRoundTrip},
		{"EphemeralPrincipalExpiry", testEphemeralPrincipalExpiry},
		{"DelegationLifecycle", testDelegationLifecycle},
		{"SessionActorChainRoundTrip", testSessionActorChainRoundTrip},
```

Append the cases:

```go
// seedPrincipal creates a non-human principal of the given kind.
func seedPrincipal(t *testing.T, s store.Store, tn tenant, kind principal.Kind, name string) *serviceaccount.ServiceAccount {
	t.Helper()
	svc := &serviceaccount.ServiceAccount{
		ID:        id.NewServiceAccountID(),
		AppID:     tn.AppID,
		EnvID:     tn.EnvID,
		Kind:      kind,
		Name:      name,
		Scopes:    []string{"repo:read"},
		Active:    true,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateServiceAccount(context.Background(), svc))
	return svc
}

// testPrincipalRoundTrip pins that the kind and owner survive persistence and
// that GetPrincipal resolves a stored row into the same ref it was written
// under. A backend that drops Kind hands back a principal that every
// authorization decision then treats as a plain service account.
func testPrincipalRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	owner := seedUser(t, s, tn, "owner@test.com")

	svc := seedPrincipal(t, s, tn, principal.KindAgent, "agent-one")
	svc.OwnerUserID = owner.ID
	require.NoError(t, s.UpdateServiceAccount(ctx, svc))

	got, err := s.GetPrincipal(ctx, principal.Ref{Kind: principal.KindAgent, ID: svc.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, principal.KindAgent, got.Kind)
	assert.Equal(t, svc.ID.String(), got.ID)
	assert.Equal(t, tn.AppID.String(), got.AppID.String())
	require.NotNil(t, got.Owner, "the owning user must survive the round trip")
	assert.Equal(t, principal.UserRef(owner.ID), *got.Owner)
	assert.True(t, got.IsActive(now()))

	// A user ref resolves too, out of the user table rather than this one.
	gotUser, err := s.GetPrincipal(ctx, principal.UserRef(owner.ID))
	require.NoError(t, err)
	assert.Equal(t, principal.KindUser, gotUser.Kind)
	assert.Equal(t, owner.ID.String(), gotUser.ID)

	// A row written with no kind at all reads as a service account, which is
	// what every row predating the column is.
	legacy := &serviceaccount.ServiceAccount{
		ID: id.NewServiceAccountID(), AppID: tn.AppID, Name: "legacy",
		Active: true, CreatedAt: now(), UpdatedAt: now(),
	}
	require.NoError(t, s.CreateServiceAccount(ctx, legacy))
	gotLegacy, err := s.GetPrincipal(ctx, principal.Ref{Kind: principal.KindService, ID: legacy.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, principal.KindService, gotLegacy.Kind)
}

// testEphemeralPrincipalExpiry pins the JIT-minted child contract: the parent
// link survives, and a lapsed child is excluded from an active-only listing
// rather than silently continuing to authenticate.
func testEphemeralPrincipalExpiry(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	parent := seedPrincipal(t, s, tn, principal.KindAgent, "parent-agent")

	past := now().Add(-time.Hour)
	future := now().Add(time.Hour)

	lapsed := seedPrincipal(t, s, tn, principal.KindAgent, "lapsed-child")
	lapsed.ParentID = parent.ID
	lapsed.ExpiresAt = &past
	require.NoError(t, s.UpdateServiceAccount(ctx, lapsed))

	live := seedPrincipal(t, s, tn, principal.KindAgent, "live-child")
	live.ParentID = parent.ID
	live.ExpiresAt = &future
	require.NoError(t, s.UpdateServiceAccount(ctx, live))

	gotLapsed, err := s.GetPrincipal(ctx, principal.Ref{Kind: principal.KindAgent, ID: lapsed.ID.String()})
	require.NoError(t, err, "an expired principal must still be readable, so callers can report why")
	require.NotNil(t, gotLapsed.Parent)
	assert.Equal(t, parent.ID.String(), gotLapsed.Parent.ID)
	assert.False(t, gotLapsed.IsActive(now()), "an expired principal must not read as active")

	active, err := s.ListPrincipals(ctx, &principal.Query{
		AppID: tn.AppID, Kind: principal.KindAgent, ActiveOnly: true, ActiveAsOf: now(),
	})
	require.NoError(t, err)
	ids := make([]string, 0, len(active))
	for _, p := range active {
		ids = append(ids, p.ID)
	}
	assert.Contains(t, ids, live.ID.String())
	assert.Contains(t, ids, parent.ID.String())
	assert.NotContains(t, ids, lapsed.ID.String(), "an active-only listing must exclude the lapsed child")
}

// testDelegationLifecycle pins create, lookup and revoke. The revoke half is
// the one that matters: a grant that stays findable after revocation is a
// credential nobody can take away.
func testDelegationLifecycle(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "delegator@test.com")
	agent := seedPrincipal(t, s, tn, principal.KindAgent, "delegated-agent")

	actor := principal.Ref{Kind: principal.KindAgent, ID: agent.ID.String()}
	subject := principal.UserRef(u.ID)

	d := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     tn.AppID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantDelegation,
		Scopes:    []string{"repo:read"},
		GrantedBy: subject,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
	require.NoError(t, s.CreateDelegation(ctx, d))

	got, err := s.GetDelegation(ctx, d.ID)
	require.NoError(t, err)
	assert.Equal(t, actor, got.Actor)
	assert.Equal(t, subject, got.Subject)
	assert.Equal(t, []string{"repo:read"}, got.Scopes)
	assert.True(t, got.IsActive(now()))

	found, err := s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantDelegation)
	require.NoError(t, err)
	assert.Equal(t, d.ID.String(), found.ID.String())

	// The wrong grant kind must not match. Impersonation and delegation are
	// evaluated differently, so crossing them changes the decision.
	_, err = s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantImpersonation)
	assert.ErrorIs(t, err, principal.ErrNotFound)

	listed, err := s.ListDelegations(ctx, &principal.DelegationQuery{AppID: tn.AppID, Subject: &subject})
	require.NoError(t, err)
	assert.Len(t, listed, 1, "the subject must be able to see what may act for them")

	require.NoError(t, s.RevokeDelegation(ctx, d.ID, now()))
	_, err = s.FindActiveDelegation(ctx, tn.AppID, actor, subject, principal.GrantDelegation)
	assert.ErrorIs(t, err, principal.ErrNotFound, "a revoked grant must stop being findable")

	afterRevoke, err := s.GetDelegation(ctx, d.ID)
	require.NoError(t, err, "a revoked grant stays readable for audit")
	assert.NotNil(t, afterRevoke.RevokedAt)

	// Revoking twice is not an error. Revocation is the thing you want to
	// succeed on a retry.
	assert.NoError(t, s.RevokeDelegation(ctx, d.ID, now()))
}

// testSessionActorChainRoundTrip pins that the chain survives every session
// read path, not only the by-id one. Middleware resolves by token and refresh
// resolves by refresh token, and a chain lost on either of those is an
// authorization decision made against the wrong set of principals.
func testSessionActorChainRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "chained@test.com")

	delID := id.NewDelegationID()
	chain := principal.Chain{
		{Kind: principal.KindAgent, ID: "svc_child"},
		{Kind: principal.KindAgent, ID: "svc_parent"},
	}
	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                u.ID,
		PrincipalKind:         principal.KindUser,
		Token:                 "tok-chain",
		RefreshToken:          "rtok-chain",
		FamilyID:              id.NewSessionFamilyID(),
		Actors:                chain,
		ActorGrant:            principal.GrantDelegation,
		DelegationID:          delID,
		ExpiresAt:             now().Add(time.Hour),
		RefreshTokenExpiresAt: now().Add(24 * time.Hour),
		CreatedAt:             now(),
		UpdatedAt:             now(),
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	for _, tc := range []struct {
		name string
		get  func() (*session.Session, error)
	}{
		{"GetSession", func() (*session.Session, error) { return s.GetSession(ctx, sess.ID) }},
		{"GetSessionByToken", func() (*session.Session, error) { return s.GetSessionByToken(ctx, "tok-chain") }},
		{"GetSessionByRefreshToken", func() (*session.Session, error) { return s.GetSessionByRefreshToken(ctx, "rtok-chain") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			assert.Equal(t, chain, got.Actors, "the actor chain must survive the round trip")
			assert.Equal(t, principal.GrantDelegation, got.ActorGrant)
			assert.Equal(t, delID.String(), got.DelegationID.String())
			assert.True(t, got.ImpersonatedBy().IsNil(), "a delegation must not read back as impersonation")
		})
	}

	// The impersonation shape must round-trip too, through whichever column
	// the backend uses for it.
	admin := seedUser(t, s, tn, "admin@test.com")
	imp := seedSession(t, s, tn, u.ID, "tok-imp", "rtok-imp")
	imp.SetImpersonatedBy(admin.ID)
	require.NoError(t, s.UpdateSession(ctx, imp))

	gotImp, err := s.GetSession(ctx, imp.ID)
	require.NoError(t, err)
	assert.Equal(t, admin.ID.String(), gotImp.ImpersonatedBy().String())
}
```

Add `serviceaccount` and `principal` to the storetest imports.

- [ ] **Step 4: Run the conformance suite to verify it fails**

Run: `go test ./store/memory/ -run TestConformance -v`
Expected: FAIL to compile, `store.Store` is not satisfied by `*memory.Store`.

- [ ] **Step 5: Implement the memory backend**

In `store/memory/store.go`, add the field to the struct next to `serviceAccounts` and initialize it in `New()`:

```go
	delegations map[string]*principal.Delegation
	...
		delegations: make(map[string]*principal.Delegation),
```

Append the implementation:

```go
// ──────────────────────────────────────────────────
// Principal Store
// ──────────────────────────────────────────────────

func (s *Store) GetPrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error) {
	if ref.Kind == principal.KindUser {
		uid, err := id.ParseUserID(ref.ID)
		if err != nil {
			return nil, principal.ErrNotFound
		}
		u, err := s.GetUser(ctx, uid)
		if err != nil {
			return nil, principal.ErrNotFound
		}
		return &principal.Principal{
			Ref:      ref,
			AppID:    u.AppID,
			EnvID:    u.EnvID,
			Name:     u.Name(),
			Disabled: false,
		}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	svc, ok := s.serviceAccounts[ref.ID]
	if !ok {
		return nil, principal.ErrNotFound
	}
	return svc.ToPrincipal(), nil
}

func (s *Store) ListPrincipals(_ context.Context, q *principal.Query) ([]*principal.Principal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*principal.Principal, 0, len(s.serviceAccounts))
	for _, svc := range s.serviceAccounts {
		if !q.AppID.IsNil() && svc.AppID.String() != q.AppID.String() {
			continue
		}
		p := svc.ToPrincipal()
		if q.Kind != "" && p.Kind != q.Kind {
			continue
		}
		if q.OwnerUser != nil && (p.Owner == nil || p.Owner.ID != q.OwnerUser.String()) {
			continue
		}
		if q.Parent != nil && (p.Parent == nil || *p.Parent != *q.Parent) {
			continue
		}
		if q.ActiveOnly && !p.IsActive(q.ActiveAsOf) {
			continue
		}
		out = append(out, p)
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (s *Store) CreateDelegation(_ context.Context, d *principal.Delegation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	d.UpdatedAt = d.CreatedAt
	// One live grant per (app, actor, subject, kind), matching the partial
	// unique index the SQL backends carry.
	for _, existing := range s.delegations {
		if existing.AppID.String() == d.AppID.String() &&
			existing.Actor == d.Actor &&
			existing.Subject == d.Subject &&
			existing.GrantKind == d.GrantKind &&
			existing.RevokedAt == nil {
			return store.ErrConflict
		}
	}
	s.delegations[d.ID.String()] = d
	return nil
}

func (s *Store) GetDelegation(_ context.Context, delID id.DelegationID) (*principal.Delegation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.delegations[delID.String()]
	if !ok {
		return nil, principal.ErrNotFound
	}
	return d, nil
}

func (s *Store) FindActiveDelegation(
	_ context.Context, appID id.AppID, actor, subject principal.Ref, grantKind principal.GrantKind,
) (*principal.Delegation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	at := time.Now()
	for _, d := range s.delegations {
		if d.AppID.String() != appID.String() ||
			d.Actor != actor || d.Subject != subject || d.GrantKind != grantKind {
			continue
		}
		if !d.IsActive(at) {
			continue
		}
		return d, nil
	}
	return nil, principal.ErrNotFound
}

func (s *Store) ListDelegations(_ context.Context, q *principal.DelegationQuery) ([]*principal.Delegation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*principal.Delegation, 0, len(s.delegations))
	for _, d := range s.delegations {
		if !q.AppID.IsNil() && d.AppID.String() != q.AppID.String() {
			continue
		}
		if q.Actor != nil && d.Actor != *q.Actor {
			continue
		}
		if q.Subject != nil && d.Subject != *q.Subject {
			continue
		}
		if q.GrantKind != "" && d.GrantKind != q.GrantKind {
			continue
		}
		if q.ActiveOnly && !d.IsActive(q.ActiveAsOf) {
			continue
		}
		out = append(out, d)
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (s *Store) RevokeDelegation(_ context.Context, delID id.DelegationID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.delegations[delID.String()]
	if !ok {
		return principal.ErrNotFound
	}
	// Revoking twice is not an error, and it must not move the timestamp.
	// Revocation is the operation you most want to succeed on a retry.
	if d.RevokedAt == nil {
		revoked := at
		d.RevokedAt = &revoked
		d.UpdatedAt = at
	}
	return nil
}
```

The memory session store keeps whole `*session.Session` values in a map, so `Actors` round-trips with no further change.

- [ ] **Step 6: Add temporary stubs to the three SQL and document backends**

Replace the contents of `store/postgres/service_accounts.go` and `store/sqlite/service_accounts.go` (Tasks 5 and 6 supersede these) and create `store/mongo/delegations.go`, each with the six delegation methods plus `GetPrincipal` and `ListPrincipals` returning `fmt.Errorf("<backend>: <method>: not implemented")`, matching the stub style already in those two files.

- [ ] **Step 7: Run the memory conformance suite to verify it passes**

Run: `go test ./store/memory/ -run TestConformance -v`
Expected: PASS, including the four new cases.

Run: `go test ./...`
Expected: PASS. Integration conformance for postgres, sqlite and mongo is not run without `-tags integration`, so this window does not turn CI red.

- [ ] **Step 8: Commit**

```bash
make check
git add -A
git commit -m "feat(store): add principal and delegation persistence to the memory backend

Extends the service-account entity with kind, owner, parent and expiry, adds
the delegation grant store to the aggregate interface, and covers all of it
with cross-backend conformance cases. postgres, sqlite and mongo carry stubs
until their own tasks."
```

---

## Task 5: PostgreSQL principals and delegations

The postgres backend has no service-account table and returns `not implemented`
from all five service-account methods. This task builds the table, adds the
delegation table and the session columns, and implements both stores.

**Files:**
- Modify: `store/postgres/migrations.go`
- Modify: `store/postgres/models.go`
- Modify: `store/postgres/store.go`
- Replace: `store/postgres/principal.go` (delete `store/postgres/service_accounts.go`)

**Interfaces:**
- Consumes: Task 4's `serviceaccount.ServiceAccount` fields, `ToPrincipal()`, and the conformance cases.
- Produces: postgres implementations of `serviceaccount.Store` and `principal.Store`. Models `ServiceAccountModel` and `DelegationModel`.

- [ ] **Step 1: Run the conformance suite to verify it fails**

Run: `go test -tags integration ./store/postgres/ -run TestConformance -v`
Expected: FAIL on `PrincipalRoundTrip`, `EphemeralPrincipalExpiry`, `DelegationLifecycle`, `SessionActorChainRoundTrip`, and on the existing service-account cases, all with `not implemented`. Requires Docker.

- [ ] **Step 2: Add the migrations**

Append to the migration list in `store/postgres/migrations.go`:

```go
		// Migration: the non-human principal table. postgres never had one:
		// the store methods were stubs, so no rows exist to preserve and this
		// creates the current shape directly rather than in two steps.
		&migrate.Migration{
			Name:    "create_authsome_service_accounts",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_service_accounts (
    id            TEXT PRIMARY KEY,
    app_id        TEXT NOT NULL REFERENCES authsome_apps(id) ON DELETE CASCADE,
    env_id        TEXT NOT NULL DEFAULT '',
    org_id        TEXT NOT NULL DEFAULT '',
    kind          TEXT NOT NULL DEFAULT 'service_account',
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    scopes        JSONB,
    owner_user_id TEXT NOT NULL DEFAULT '',
    parent_id     TEXT NOT NULL DEFAULT '',
    expires_at    TIMESTAMPTZ,
    active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT authsome_service_accounts_kind_check
        CHECK (kind IN ('agent', 'workload', 'service_account'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_service_accounts_app_name
    ON authsome_service_accounts (app_id, name);
CREATE INDEX IF NOT EXISTS idx_authsome_service_accounts_owner
    ON authsome_service_accounts (owner_user_id)
    WHERE owner_user_id <> '';
-- Ephemeral children are reaped by parent and by expiry, so both are indexed.
CREATE INDEX IF NOT EXISTS idx_authsome_service_accounts_parent
    ON authsome_service_accounts (parent_id)
    WHERE parent_id <> '';
CREATE INDEX IF NOT EXISTS idx_authsome_service_accounts_expires
    ON authsome_service_accounts (expires_at)
    WHERE expires_at IS NOT NULL;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_service_accounts`)
				return err
			},
		},

		// Migration: delegation grants. The partial unique index is what makes
		// revocation work: a revoked row keeps its identity for audit while
		// freeing the slot for a fresh grant between the same two principals.
		&migrate.Migration{
			Name:    "create_authsome_delegations",
			Version: "20260824000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_delegations (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL REFERENCES authsome_apps(id) ON DELETE CASCADE,
    org_id          TEXT NOT NULL DEFAULT '',
    actor_kind      TEXT NOT NULL,
    actor_id        TEXT NOT NULL,
    subject_kind    TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    grant_kind      TEXT NOT NULL,
    scopes          JSONB,
    granted_by_kind TEXT NOT NULL DEFAULT '',
    granted_by_id   TEXT NOT NULL DEFAULT '',
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT authsome_delegations_grant_kind_check
        CHECK (grant_kind IN ('delegation', 'impersonation'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_delegations_live
    ON authsome_delegations (app_id, actor_kind, actor_id, subject_kind, subject_id, grant_kind)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_authsome_delegations_subject
    ON authsome_delegations (app_id, subject_kind, subject_id);
CREATE INDEX IF NOT EXISTS idx_authsome_delegations_actor
    ON authsome_delegations (app_id, actor_kind, actor_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `DROP TABLE IF EXISTS authsome_delegations`)
				return err
			},
		},

		// Migration: the actor chain on sessions, and the widened principal
		// check.
		//
		// The existing check admits only '' , 'user' and 'service_account'. A
		// delegated session still carries 'user' with a real user_id, because
		// the subject is the human and the agent is in the chain, so the check
		// only widens for standalone agent and workload sessions.
		//
		// impersonated_by stays. It is backfilled into actors here and dropped
		// in a later change, once the backfill has proven itself.
		&migrate.Migration{
			Name:    "add_session_actor_chain",
			Version: "20260824000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS actors        JSONB,
    ADD COLUMN IF NOT EXISTS actor_grant   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS delegation_id TEXT NOT NULL DEFAULT '';

UPDATE authsome_sessions
   SET actors = jsonb_build_array(jsonb_build_object('kind', 'user', 'id', impersonated_by)),
       actor_grant = 'impersonation'
 WHERE impersonated_by <> ''
   AND actors IS NULL;

ALTER TABLE authsome_sessions
    DROP CONSTRAINT IF EXISTS authsome_sessions_principal_check;
ALTER TABLE authsome_sessions
    ADD CONSTRAINT authsome_sessions_principal_check CHECK (
        (principal_kind IN ('service_account', 'agent', 'workload')
             AND service_account_id <> '' AND user_id = '')
        OR (principal_kind IN ('', 'user')
             AND user_id <> '' AND service_account_id = '')
    );

CREATE INDEX IF NOT EXISTS idx_authsome_sessions_delegation_id
    ON authsome_sessions (delegation_id)
    WHERE delegation_id <> '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				// Rows written for an agent or workload subject violate the
				// narrower check, so they go before it is restored. They are
				// unreachable under the old schema anyway.
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_sessions_delegation_id;
DELETE FROM authsome_sessions WHERE principal_kind IN ('agent', 'workload');

ALTER TABLE authsome_sessions
    DROP CONSTRAINT IF EXISTS authsome_sessions_principal_check;
ALTER TABLE authsome_sessions
    ADD CONSTRAINT authsome_sessions_principal_check CHECK (
        (principal_kind = 'service_account'
             AND service_account_id <> '' AND user_id = '')
        OR (principal_kind IN ('', 'user')
             AND user_id <> '' AND service_account_id = '')
    );

ALTER TABLE authsome_sessions
    DROP COLUMN IF EXISTS delegation_id,
    DROP COLUMN IF EXISTS actor_grant,
    DROP COLUMN IF EXISTS actors;
`)
				return err
			},
		},
```

- [ ] **Step 3: Add the models**

In `store/postgres/models.go`, add to `SessionModel` next to `Roles`:

```go
	// Actors is JSON for the same reason Roles is: a chain element carries a
	// kind and an id, and both are compared to make authorization decisions,
	// so the encoding must not be able to invent or merge a member.
	Actors       json.RawMessage `grove:"actors,type:jsonb"`
	ActorGrant   string          `grove:"actor_grant"`
	DelegationID string          `grove:"delegation_id"`
```

Wire them in `toSession` and `fromSession` alongside the existing `Roles` handling, unmarshalling into `principal.Chain` and marshalling back. An empty chain writes SQL NULL rather than `[]`, matching how `Roles` is handled.

Add the two new models:

```go
type ServiceAccountModel struct {
	grove.BaseModel `grove:"table:authsome_service_accounts,alias:sa"`

	ID          string `grove:"id,pk"`
	AppID       string `grove:"app_id,notnull"`
	EnvID       string `grove:"env_id"`
	OrgID       string `grove:"org_id"`
	// Kind is empty on no row postgres writes, but stays nullable-tolerant so
	// a row inserted by hand without it still reads as a service account.
	Kind        string          `grove:"kind"`
	Name        string          `grove:"name,notnull"`
	Description string          `grove:"description"`
	Scopes      json.RawMessage `grove:"scopes,type:jsonb"`
	OwnerUserID string          `grove:"owner_user_id"`
	ParentID    string          `grove:"parent_id"`
	ExpiresAt   *time.Time      `grove:"expires_at"`
	Active      bool            `grove:"active,notnull"`
	CreatedAt   time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time       `grove:"updated_at,notnull,default:now()"`
}

type DelegationModel struct {
	grove.BaseModel `grove:"table:authsome_delegations,alias:dl"`

	ID            string          `grove:"id,pk"`
	AppID         string          `grove:"app_id,notnull"`
	OrgID         string          `grove:"org_id"`
	ActorKind     string          `grove:"actor_kind,notnull"`
	ActorID       string          `grove:"actor_id,notnull"`
	SubjectKind   string          `grove:"subject_kind,notnull"`
	SubjectID     string          `grove:"subject_id,notnull"`
	GrantKind     string          `grove:"grant_kind,notnull"`
	Scopes        json.RawMessage `grove:"scopes,type:jsonb"`
	GrantedByKind string          `grove:"granted_by_kind"`
	GrantedByID   string          `grove:"granted_by_id"`
	ExpiresAt     *time.Time      `grove:"expires_at"`
	RevokedAt     *time.Time      `grove:"revoked_at"`
	CreatedAt     time.Time       `grove:"created_at,notnull,default:now()"`
	UpdatedAt     time.Time       `grove:"updated_at,notnull,default:now()"`
}
```

Write `toServiceAccount`, `fromServiceAccount`, `toDelegation` and `fromDelegation` alongside the existing converters in the same file, following the `toAPIKey`/`fromAPIKey` shape: parse every ID with the typed parser and return the parse error rather than swallowing it.

- [ ] **Step 4: Implement the stores**

Delete `store/postgres/service_accounts.go` and create `store/postgres/principal.go` with all five `serviceaccount.Store` methods and all six `principal.Store` methods, following the query style of `CreateAPIKey` and `GetAPIKeyByPrefix` in `store/postgres/store.go`. The two whose behaviour is not obvious from the interface:

```go
// FindActiveDelegation resolves the live grant letting actor act for subject.
//
// Active is evaluated in SQL rather than by loading and filtering in Go: this
// runs on the authentication path for every delegated request, and the partial
// unique index means at most one row can match.
func (s *Store) FindActiveDelegation(
	ctx context.Context, appID id.AppID, actor, subject principal.Ref, grantKind principal.GrantKind,
) (*principal.Delegation, error) {
	m := new(DelegationModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("actor_kind = ?", string(actor.Kind)).
		Where("actor_id = ?", actor.ID).
		Where("subject_kind = ?", string(subject.Kind)).
		Where("subject_id = ?", subject.ID).
		Where("grant_kind = ?", string(grantKind)).
		Where("revoked_at IS NULL").
		Where("(expires_at IS NULL OR expires_at > NOW())").
		Scan(ctx)
	if err != nil {
		if errors.Is(pgError(err), store.ErrNotFound) {
			return nil, principal.ErrNotFound
		}
		return nil, pgError(err)
	}
	return toDelegation(m)
}

// RevokeDelegation marks a grant revoked. The revoked_at guard makes a repeat
// call a no-op rather than moving the timestamp, so a retried revocation
// cannot rewrite when the revocation actually happened.
func (s *Store) RevokeDelegation(ctx context.Context, delID id.DelegationID, at time.Time) error {
	res, err := s.pg.NewUpdate((*DelegationModel)(nil)).
		Set("revoked_at = ?", at).
		Set("updated_at = ?", at).
		Where("id = ?", delID.String()).
		Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return pgError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Either already revoked or absent. Distinguish, because a caller
		// revoking a grant that never existed has a different problem from one
		// retrying.
		if _, getErr := s.GetDelegation(ctx, delID); getErr != nil {
			return getErr
		}
	}
	return nil
}
```

`GetPrincipal` branches on `ref.Kind`: `principal.KindUser` reads `authsome_users` and builds a user principal, everything else reads `authsome_service_accounts` and returns `ToPrincipal()`. `ListPrincipals` applies `q.ActiveOnly` as `active = TRUE AND (expires_at IS NULL OR expires_at > ?)` with `q.ActiveAsOf`.

- [ ] **Step 5: Run the conformance suite to verify it passes**

Run: `go test -tags integration ./store/postgres/ -run TestConformance -v`
Expected: PASS, every case.

- [ ] **Step 6: Commit**

```bash
make check
git add -A
git commit -m "feat(store/postgres): implement principal and delegation persistence

postgres previously returned not implemented from every service-account
method and had no table for them, so the non-human principal path did not
work on it at all. Adds both tables, the session actor chain, and the widened
principal check."
```

---

## Task 6: SQLite principals and delegations

Same shape as Task 5. The differences that matter: sqlite stores timestamps as
TEXT, has no partial unique index on an expression but does support partial
indexes with a WHERE clause, and has no `jsonb_build_array`, so the
impersonation backfill is written with string concatenation.

**Files:**
- Modify: `store/sqlite/migrations.go`, `store/sqlite/models.go`, `store/sqlite/store.go`
- Replace: `store/sqlite/principal.go` (delete `store/sqlite/service_accounts.go`)

**Interfaces:**
- Consumes: Task 4's entity and conformance cases.
- Produces: sqlite implementations of `serviceaccount.Store` and `principal.Store`.

- [ ] **Step 1: Run the conformance suite to verify it fails**

Run: `go test -tags integration ./store/sqlite/ -run TestConformance -v`
Expected: FAIL with `not implemented` on the service-account and principal cases.

- [ ] **Step 2: Add the migrations**

Append three migrations at versions `20260824000001`, `20260824000002` and `20260824000003`, mirroring Task 5's DDL with these substitutions:

```sql
-- 20260824000001
CREATE TABLE IF NOT EXISTS authsome_service_accounts (
    id            TEXT PRIMARY KEY,
    app_id        TEXT NOT NULL,
    env_id        TEXT NOT NULL DEFAULT '',
    org_id        TEXT NOT NULL DEFAULT '',
    kind          TEXT NOT NULL DEFAULT 'service_account',
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    scopes        TEXT NOT NULL DEFAULT '',
    owner_user_id TEXT NOT NULL DEFAULT '',
    parent_id     TEXT NOT NULL DEFAULT '',
    expires_at    TEXT,
    active        INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now')),
    CHECK (kind IN ('agent', 'workload', 'service_account'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_service_accounts_app_name
    ON authsome_service_accounts (app_id, name);
CREATE INDEX IF NOT EXISTS idx_authsome_service_accounts_parent
    ON authsome_service_accounts (parent_id) WHERE parent_id <> '';
CREATE INDEX IF NOT EXISTS idx_authsome_service_accounts_expires
    ON authsome_service_accounts (expires_at) WHERE expires_at IS NOT NULL;

-- 20260824000002
CREATE TABLE IF NOT EXISTS authsome_delegations (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL,
    org_id          TEXT NOT NULL DEFAULT '',
    actor_kind      TEXT NOT NULL,
    actor_id        TEXT NOT NULL,
    subject_kind    TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    grant_kind      TEXT NOT NULL,
    scopes          TEXT NOT NULL DEFAULT '',
    granted_by_kind TEXT NOT NULL DEFAULT '',
    granted_by_id   TEXT NOT NULL DEFAULT '',
    expires_at      TEXT,
    revoked_at      TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    CHECK (grant_kind IN ('delegation', 'impersonation'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_delegations_live
    ON authsome_delegations (app_id, actor_kind, actor_id, subject_kind, subject_id, grant_kind)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_authsome_delegations_subject
    ON authsome_delegations (app_id, subject_kind, subject_id);
CREATE INDEX IF NOT EXISTS idx_authsome_delegations_actor
    ON authsome_delegations (app_id, actor_kind, actor_id);

-- 20260824000003
ALTER TABLE authsome_sessions ADD COLUMN actors TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sessions ADD COLUMN actor_grant TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sessions ADD COLUMN delegation_id TEXT NOT NULL DEFAULT '';

UPDATE authsome_sessions
   SET actors = '[{"kind":"user","id":"' || impersonated_by || '"}]',
       actor_grant = 'impersonation'
 WHERE impersonated_by <> '' AND actors = '';

CREATE INDEX IF NOT EXISTS idx_authsome_sessions_delegation_id
    ON authsome_sessions (delegation_id) WHERE delegation_id <> '';
```

sqlite cannot drop or add a table constraint in place, so the session principal
check is not altered here. Confirm before writing the migration whether
`authsome_sessions` in `store/sqlite/migrations.go` carries a principal CHECK
at all. If it does, the widening needs the twelve-step table rebuild sqlite
requires, and that rebuild goes in this migration. If it does not, no
constraint work is needed and this migration is only the three columns, the
backfill and the index.

Follow `store/sqlite/migrations_timestamps.go` for how timestamps are written
and read, so `expires_at` and `revoked_at` compare correctly in SQL.

- [ ] **Step 3: Add the models and implement the stores**

Mirror Task 5's `ServiceAccountModel` and `DelegationModel` with sqlite types:
`Scopes string` holding JSON, `ExpiresAt` and `RevokedAt` as the timestamp type
the other sqlite models in this file use, and `Active bool`.

Delete `store/sqlite/service_accounts.go` and create `store/sqlite/principal.go`
with the same eleven methods as Task 5. `FindActiveDelegation` uses the sqlite
timestamp comparison in place of `NOW()`:

```go
		Where("revoked_at IS NULL").
		Where("(expires_at IS NULL OR expires_at > ?)", formatTimestamp(time.Now())).
```

using whichever helper `store/sqlite/migrations_timestamps.go` and the existing
models already use to format a timestamp for comparison.

- [ ] **Step 4: Run the conformance suite to verify it passes**

Run: `go test -tags integration ./store/sqlite/ -run TestConformance -v`
Expected: PASS, every case.

- [ ] **Step 5: Commit**

```bash
make check
git add -A
git commit -m "feat(store/sqlite): implement principal and delegation persistence"
```

---

## Task 7: Mongo principals and delegations

Mongo already implements the service-account store, so this task extends rather
than builds.

**Files:**
- Modify: `store/mongo/migrations.go`, `store/mongo/models.go`, `store/mongo/store.go`, `store/mongo/service_accounts.go`
- Replace: `store/mongo/delegations.go`

**Interfaces:**
- Consumes: Task 4's entity and conformance cases.
- Produces: mongo implementation of `principal.Store`, extended `serviceAccountModel`, collection constant `colDelegations = "authsome_delegations"`.

- [ ] **Step 1: Run the conformance suite to verify it fails**

Run: `go test -tags integration ./store/mongo/ -run TestConformance -v`
Expected: FAIL on the principal and delegation cases, and on `PrincipalRoundTrip` where the new fields are dropped.

- [ ] **Step 2: Extend the models**

In `store/mongo/models.go`, add to `serviceAccountModel`:

```go
	EnvID       string     `grove:"env_id"        bson:"env_id,omitempty"`
	OrgID       string     `grove:"org_id"        bson:"org_id,omitempty"`
	Kind        string     `grove:"kind"          bson:"kind,omitempty"`
	OwnerUserID string     `grove:"owner_user_id" bson:"owner_user_id,omitempty"`
	ParentID    string     `grove:"parent_id"     bson:"parent_id,omitempty"`
	ExpiresAt   *time.Time `grove:"expires_at"    bson:"expires_at,omitempty"`
```

Update `toServiceAccountModel` and its inverse to carry them. Add to the session
model:

```go
	Actors       []actorModel `grove:"actors"        bson:"actors,omitempty"`
	ActorGrant   string       `grove:"actor_grant"   bson:"actor_grant,omitempty"`
	DelegationID string       `grove:"delegation_id" bson:"delegation_id,omitempty"`
```

with

```go
// actorModel is one hop of a session's actor chain. Stored as a subdocument
// array rather than an encoded string so a chain element can be queried on
// directly, which is what an operator asking "what did this agent touch"
// needs.
type actorModel struct {
	Kind string `bson:"kind"`
	ID   string `bson:"id"`
}
```

Add `delegationModel` mirroring Task 5's `DelegationModel` with bson tags, and
`colDelegations = "authsome_delegations"` to the collection constants in
`store/mongo/store.go`, plus its entry in the collection registry near
`colServiceAccounts`.

- [ ] **Step 3: Add the migrations**

Append to `store/mongo/migrations.go` at versions `20260824000001` and
`20260824000002`, following the shape of `create_authsome_service_accounts`
already in that file:

```go
		&migrate.Migration{
			Name:    "create_authsome_delegations",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				if err := mexec.CreateCollection(ctx, (*delegationModel)(nil)); err != nil {
					return err
				}
				return mexec.CreateIndexes(ctx, colDelegations, []mongo.IndexModel{
					{
						// Partial on revoked_at missing rather than sparse:
						// grove writes every mapped field whatever the bson
						// omitempty tag says, so this mirrors how
						// add_session_principal_identity indexes
						// service_account_id.
						Keys: bson.D{
							{Key: "app_id", Value: 1},
							{Key: "actor_kind", Value: 1}, {Key: "actor_id", Value: 1},
							{Key: "subject_kind", Value: 1}, {Key: "subject_id", Value: 1},
							{Key: "grant_kind", Value: 1},
						},
						Options: options.Index().SetUnique(true).
							SetPartialFilterExpression(bson.D{{Key: "revoked_at", Value: bson.D{{Key: "$exists", Value: false}}}}),
					},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "subject_kind", Value: 1}, {Key: "subject_id", Value: 1}}},
					{Keys: bson.D{{Key: "app_id", Value: 1}, {Key: "actor_kind", Value: 1}, {Key: "actor_id", Value: 1}}},
				})
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				return mexec.DropCollection(ctx, colDelegations)
			},
		},
```

The second migration refreshes the session and service-account validators via
`collMod` and backfills the chain, following
`add_session_principal_identity` in the same file for the `RefreshValidator`
call. The backfill sets `actors` and `actor_grant` on every session document
where `impersonated_by` is a non-empty string.

- [ ] **Step 4: Implement the delegation store**

Replace `store/mongo/delegations.go` with the six `principal.Store` delegation
methods plus `GetPrincipal` and `ListPrincipals`, following the query style of
`store/mongo/service_accounts.go`. `CreateDelegation` maps a duplicate-key
error to `store.ErrConflict` the way `CreateServiceAccount` already does.
`FindActiveDelegation` filters on `revoked_at` missing and `expires_at` either
missing or in the future.

- [ ] **Step 5: Run the conformance suite to verify it passes**

Run: `go test -tags integration ./store/mongo/ -run TestConformance -v`
Expected: PASS, every case.

Run: `go test -tags integration ./store/...`
Expected: PASS across all four backends. This is the first point where every
backend agrees on the principal contract.

- [ ] **Step 6: Commit**

```bash
make check
git add -A
git commit -m "feat(store/mongo): add delegation grants and principal fields"
```

---
