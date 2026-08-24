# Non-human principals implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give authsome a first-class principal abstraction covering human users, AI agents and workloads, with an RFC 8693 actor chain, delegation grants, Warden intersection, and risk-plugin visibility into machine traffic.

**Architecture:** A new `principal` package holds value types only (`Ref`, `Chain`, `Principal`, `Delegation`, `AuthAttempt`) and a `Store` interface. Sessions carry a `Subject` derived from existing fields plus an `Actors` chain, which subsumes the current `ImpersonatedBy` field. Non-human principals are stored in the existing `authsome_service_accounts` table, extended with `kind`, `owner_user_id`, `parent_id` and `expires_at`. Authorization goes through a new chain-aware `Engine.Can`, which intersects the subject's decision with every actor's decision.

**Tech Stack:** Go 1.26, grove ORM and grove/migrate, warden v1.6.0 for authorization, forge for HTTP and scope, testify for assertions, testcontainers for postgres and mongo integration tests.

**Spec:** `docs/superpowers/specs/2026-08-24-non-human-principals-design.md`

## Global constraints

**Coordination.** Several designs landed on this repo the same day and some
touch the same tables. Read this before writing a migration.

- Migration versions for this plan are `20260824000050`, `20260824000051` and
  `20260824000052`, in every backend. `20260824000001` through `000003` are
  contested by at least three other plans, and `000030` and `000040` are taken
  by dynamic client registration, DPoP and resource indicators.
- `docs/superpowers/plans/2026-08-24-token-exchange-rfc8693.md` also puts an
  actor chain and a scope list on `session.Session`, and
  `docs/superpowers/plans/2026-08-24-agentauth-delegation.md` also adds agent
  delegation with `PrincipalKind = "agent"`. Whichever lands second must build
  on the first rather than adding a second mechanism. See the note at the end
  of this plan.


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
- Produces: `principal.GrantKind` with `GrantDelegation` and `GrantImpersonation`. `principal.Delegation` with `IsActive(time.Time) bool` and `AllowsScope(string) bool`. `principal.Store` interface with `GetPrincipal`, `ListPrincipals`, `CreateDelegation`, `GetDelegation`, `FindActiveDelegation`, `ListDelegations`, `RevokeDelegation`. `principal.AuthAttempt`. `principal.NewContext`, `principal.FromContext`, `principal.NewActorsContext`, `principal.ActorsFromContext`. `id.DelegationID`, `id.NewDelegationID`, `id.ParseDelegationID`, `id.PrefixDelegation`.

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
			Version: "20260824000050",
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
			Version: "20260824000051",
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
			Version: "20260824000052",
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

Append three migrations at versions `20260824000050`, `20260824000051` and `20260824000052`, mirroring Task 5's DDL with these substitutions:

```sql
-- 20260824000050
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

-- 20260824000051
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

-- 20260824000052
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

Append to `store/mongo/migrations.go` at versions `20260824000050` and
`20260824000051`, following the shape of `create_authsome_service_accounts`
already in that file:

```go
		&migrate.Migration{
			Name:    "create_authsome_delegations",
			Version: "20260824000050",
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

## Task 8: Chain-aware authorization

`HasPermission` keeps its exact signature so `plugin.PermissionChecker` and
every current caller are untouched. It becomes a wrapper.

Where the spec's three-part rule is enforced: Warden covers the subject check
and the actor checks, which is this task. The grant's scope filter is applied
when the session is minted, in Task 14, by intersecting the requested scope
with the grant and with the actor's own scopes. Splitting it that way keeps a
scope lookup off the per-check path.

**Files:**
- Create: `engine_principal.go`
- Create: `engine_principal_test.go`
- Modify: `service.go:1915-1954`
- Modify: `session/session.go`

**Interfaces:**
- Consumes: Task 1 and 2 types, `e.wardenEng`, `e.ensureWardenScope`.
- Produces: `Engine.Can(ctx context.Context, subject principal.Ref, actors principal.Chain, action, resource string) (bool, error)`, `Engine.ResolvePrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error)`, `session.Session.AuthzActors() principal.Chain`.

- [ ] **Step 1: Write the failing test**

Create `engine_principal_test.go`. Use the existing engine test fixtures in `engine_test.go` for construction:

```go
package authsome

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

// Delegation narrows and never widens. Each row is one of the four
// combinations of (subject allowed, actor allowed), and only the case where
// both allow may pass.
func TestCanIntersectsSubjectAndActor(t *testing.T) {
	for _, tc := range []struct {
		name          string
		subjectAllow  bool
		actorAllow    bool
		wantAllowed   bool
	}{
		{"both allow", true, true, true},
		{"actor denied", true, false, false},
		{"subject denied", false, true, false},
		{"both denied", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, userID, agentRef := setupCanFixture(t, tc.subjectAllow, tc.actorAllow)

			got, err := e.Can(context.Background(),
				principal.UserRef(userID),
				principal.Chain{agentRef},
				"read", "document")
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllowed, got)
		})
	}
}

// With no chain the behaviour must be byte-identical to a plain subject check,
// which is what every existing caller relies on.
func TestCanWithEmptyChainIsASingleCheck(t *testing.T) {
	e, userID, _ := setupCanFixture(t, true, false)

	got, err := e.Can(context.Background(), principal.UserRef(userID), nil, "read", "document")
	require.NoError(t, err)
	assert.True(t, got, "an empty chain must not consult any actor")
}

// HasPermission must keep answering exactly as it did, since it is on the
// plugin.PermissionChecker contract.
func TestHasPermissionDelegatesToCan(t *testing.T) {
	e, userID, _ := setupCanFixture(t, true, false)

	got, err := e.HasPermission(context.Background(), userID, "read", "document")
	require.NoError(t, err)
	assert.True(t, got)
}

// A multi-hop chain checks every hop. An ephemeral child acting through a
// parent that has lost the permission must be denied, or revoking the parent
// would leave its children running.
func TestCanChecksEveryHop(t *testing.T) {
	e, userID, childRef, parentRef := setupMultiHopFixture(t,
		true /*subject*/, true /*child*/, false /*parent*/)

	got, err := e.Can(context.Background(),
		principal.UserRef(userID),
		principal.Chain{childRef, parentRef},
		"read", "document")
	require.NoError(t, err)
	assert.False(t, got, "a denied parent must deny the child acting through it")
}
```

Write `setupCanFixture` and `setupMultiHopFixture` in the same file using the
engine construction helper already used by `engine_test.go`, granting or
withholding the `read` permission on `document` per argument by creating a role
in Warden and assigning it to the subject and actor refs.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test . -run 'TestCan|TestHasPermissionDelegates' -v`
Expected: FAIL, `e.Can` undefined.

- [ ] **Step 3: Add `AuthzActors` to the session**

In `session/session.go`:

```go
// AuthzActors returns the actors Warden must independently authorize.
//
// Empty for an impersonation, and that is the point. Impersonating somebody is
// precisely the request to evaluate as them, so the admin's own permissions
// are not intersected in. The gate for impersonation sits on Engine.Impersonate,
// which is where an admin is checked once, rather than on every subsequent
// permission check made while impersonating.
func (s *Session) AuthzActors() principal.Chain {
	if s.ActorGrant == principal.GrantImpersonation {
		return nil
	}
	return s.Actors
}
```

- [ ] **Step 4: Write `engine_principal.go`**

```go
package authsome

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"
	"github.com/xraph/warden"

	"github.com/xraph/authsome/principal"
)

// wardenSubjectKind maps a principal kind onto a Warden subject kind.
//
// All three non-human kinds collapse onto SubjectServiceAcct so role
// assignments live in one namespace and an operator grants a role once rather
// than once per kind. The finer kind rides along in the subject attributes, so
// an ABAC policy can still tell an agent from a CI workload.
func wardenSubjectKind(k principal.Kind) warden.SubjectKind {
	if k == principal.KindUser || k == "" {
		return warden.SubjectUser
	}
	return warden.SubjectServiceAcct
}

// wardenSubject builds the Warden subject for ref.
//
// onBehalfOf is set when ref is an actor rather than the subject, so a policy
// can deny an agent an action it would allow the same agent calling for
// itself.
func wardenSubject(ref principal.Ref, onBehalfOf *principal.Ref) warden.Subject {
	attrs := map[string]any{"principal_kind": string(ref.Kind)}
	if onBehalfOf != nil {
		attrs["on_behalf_of"] = onBehalfOf.String()
		attrs["actor_kind"] = string(ref.Kind)
	}
	return warden.Subject{
		Kind:       wardenSubjectKind(ref.Kind),
		ID:         ref.ID,
		Attributes: attrs,
	}
}

// Can reports whether subject may perform action on resource, given that
// actors are acting on the subject's behalf.
//
// With no actors this is a single Warden check and behaves exactly as
// HasPermission always has. With actors, every party must allow: the subject,
// and each hop of the chain. Delegation can only narrow. The first denial
// short-circuits, so a denied agent costs one check rather than the whole
// chain.
//
// Impersonation does not reach here with a populated chain. Session.AuthzActors
// returns nil for it, because impersonating somebody is the request to
// evaluate as them.
func (e *Engine) Can(
	ctx context.Context, subject principal.Ref, actors principal.Chain, action, resource string,
) (bool, error) {
	ctx = e.ensureWardenScope(ctx)

	allowed, err := e.checkOne(ctx, wardenSubject(subject, nil), action, resource)
	if err != nil || !allowed {
		return false, err
	}

	for _, actor := range actors {
		allowed, err := e.checkOne(ctx, wardenSubject(actor, &subject), action, resource)
		if err != nil {
			return false, err
		}
		if !allowed {
			e.logger.Warn("authsome: Can denied by actor",
				log.String("subject", subject.String()),
				log.String("actor", actor.String()),
				log.String("action", action),
				log.String("resource", resource),
			)
			return false, nil
		}
	}

	return true, nil
}

func (e *Engine) checkOne(
	ctx context.Context, sub warden.Subject, action, resource string,
) (bool, error) {
	result, err := e.wardenEng.Check(ctx, &warden.CheckRequest{
		Subject:  sub,
		Action:   warden.Action{Name: action},
		Resource: warden.Resource{Type: resource},
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// ResolvePrincipal resolves any caller by ref.
func (e *Engine) ResolvePrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	p, err := e.store.GetPrincipal(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("authsome: resolve principal %s: %w", ref, err)
	}
	return p, nil
}

// PrincipalStore returns the principal and delegation store.
func (e *Engine) PrincipalStore() principal.Store { return e.store }
```

- [ ] **Step 5: Rewrite `HasPermission` as a wrapper**

Replace the body of `HasPermission` in `service.go:1915`, keeping the signature
and the existing denial logging block:

```go
// HasPermission checks whether a user has a specific permission.
//
// Preserved as-is for plugin.PermissionChecker and for every caller that has a
// user ID and no chain. It is Can with an empty chain.
func (e *Engine) HasPermission(ctx context.Context, userID id.UserID, action, resource string) (bool, error) {
	allowed, err := e.Can(ctx, principal.UserRef(userID), nil, action, resource)
	if err != nil {
		e.logger.Warn("authsome: HasPermission error",
			log.String("user_id", userID.String()),
			log.String("action", action),
			log.String("resource", resource),
			log.String("error", err.Error()),
		)
		return false, err
	}
	if !allowed {
		// Keep the existing tenant and scope diagnostics verbatim: they are
		// what operators use to work out which app a denial came from.
		forgeAppID := ""
		forgeOrgID := ""
		if s, ok := forge.ScopeFrom(ctx); ok {
			forgeAppID = s.AppID()
			forgeOrgID = s.OrgID()
		}
		e.logger.Warn("authsome: HasPermission denied",
			log.String("user_id", userID.String()),
			log.String("action", action),
			log.String("resource", resource),
			log.String("forge_app_id", forgeAppID),
			log.String("forge_org_id", forgeOrgID),
			log.String("platform_app_id", e.PlatformAppID().String()),
		)
	}
	return allowed, nil
}
```

The `decision` and `reason` log fields go away, because `Can` returns a boolean
rather than a `CheckResult`. If those fields matter to you, have `checkOne`
return the `*warden.CheckResult` and log them there instead, where they are
available for every hop and not only the subject.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test . -run 'TestCan|TestHasPermission' -v && go test ./...`
Expected: PASS. Every existing permission test must still pass, since the
empty-chain path is the old path.

- [ ] **Step 7: Commit**

```bash
make check
git add -A
git commit -m "feat(authz): add chain-aware Can, intersecting subject with actors

A principal is a Warden subject in its own right, so a workload holds a role
with no user behind it. When a chain is present every hop must allow, which
means delegation can only narrow. HasPermission keeps its signature and
becomes Can with an empty chain."
```

---

## Task 9: Widen `plugin.Engine`

**Files:**
- Modify: `plugin/plugin.go:52-143`
- Modify: `plugins/apikey/plugin_test.go:47`

**Interfaces:**
- Consumes: Task 8's `Engine.Can`, `Engine.ResolvePrincipal`, `Engine.PrincipalStore`.
- Produces: three new methods on the `plugin.Engine` interface.

- [ ] **Step 1: Write the failing test**

Add to `plugin/plugin_test.go` (create if absent):

```go
package plugin_test

import (
	"testing"

	"github.com/xraph/authsome/plugin"
)

// The three principal methods are on the core interface rather than an
// optional capability interface. Plugins are meant to reason about non-human
// callers, and a type assertion makes that undiscoverable.
func TestEngineInterfaceExposesPrincipalMethods(t *testing.T) {
	var e plugin.Engine
	if e != nil {
		_ = e.ResolvePrincipal
		_ = e.PrincipalStore
		_ = e.Can
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./plugin/ -run TestEngineInterfaceExposesPrincipalMethods -v`
Expected: FAIL to compile, the methods are not on the interface.

- [ ] **Step 3: Add the methods to the interface**

In `plugin/plugin.go`, after the `// ── User / session resolution ──` block:

```go
	// ── Principals ──

	// ResolvePrincipal resolves any caller, human or otherwise, by ref.
	// Use this rather than ResolveUser when a plugin must work for agents and
	// workloads as well as people.
	ResolvePrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error)

	// PrincipalStore returns the principal and delegation store.
	PrincipalStore() principal.Store

	// Can is the chain-aware authorization check. Pass an empty chain for an
	// ordinary single-subject check.
	Can(ctx context.Context, subject principal.Ref, actors principal.Chain,
		action, resource string) (bool, error)
```

Add `github.com/xraph/authsome/principal` to the imports.

- [ ] **Step 4: Update the one in-repo mock**

`plugins/apikey/plugin_test.go` declares `var _ plugin.Engine = (*mockEngine)(nil)`.
Add the three methods to `mockEngine`:

```go
func (m *mockEngine) ResolvePrincipal(_ context.Context, _ principal.Ref) (*principal.Principal, error) {
	return nil, principal.ErrNotFound
}

func (m *mockEngine) PrincipalStore() principal.Store { return nil }

func (m *mockEngine) Can(_ context.Context, _ principal.Ref, _ principal.Chain, _, _ string) (bool, error) {
	return false, nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go build ./... && go test ./plugin/ ./plugins/apikey/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
make check
git add -A
git commit -m "feat(plugin): expose principal resolution and chain-aware authz on Engine

Out-of-tree implementers of plugin.Engine will need the three new methods."
```

---

## Task 10: Resolve the principal onto the request context

**Files:**
- Modify: `middleware/context.go`
- Modify: `middleware/auth.go` (`setSessionContext`, `tryStrategyAuth`)
- Test: `middleware/principal_test.go` (create)

**Interfaces:**
- Consumes: Task 3's `Session.Subject()`, `Session.Actors`; Task 2's context carriers.
- Produces: `middleware.WithPrincipal`, `middleware.PrincipalFrom`, `middleware.WithActors`, `middleware.ActorsFrom`, `middleware.PrincipalResolver`, and `middleware.setPrincipalContext(goCtx context.Context, sess *session.Session, resolve PrincipalResolver, logger log.Logger) context.Context`.

- [ ] **Step 1: Write the failing test**

Create `middleware/principal_test.go`:

```go
package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
)

func TestPrincipalContextCarriers(t *testing.T) {
	ctx := context.Background()

	_, ok := middleware.PrincipalFrom(ctx)
	assert.False(t, ok)

	p := &principal.Principal{Ref: principal.Ref{Kind: principal.KindWorkload, ID: "svc_ci"}}
	ctx = middleware.WithPrincipal(ctx, p)

	got, ok := middleware.PrincipalFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, p, got)

	// The same value must be readable through the principal package, so a
	// plugin can get the caller without importing middleware.
	viaPackage, ok := principal.FromContext(ctx)
	require.True(t, ok, "middleware and principal must share one context key")
	assert.Equal(t, p, viaPackage)
}

func TestActorsContextCarrier(t *testing.T) {
	ctx := context.Background()
	chain := principal.Chain{{Kind: principal.KindAgent, ID: "svc_a"}}
	ctx = middleware.WithActors(ctx, chain)

	got, ok := middleware.ActorsFrom(ctx)
	require.True(t, ok)
	assert.Equal(t, chain, got)

	viaPackage, ok := principal.ActorsFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, chain, viaPackage)
}

func TestImpersonatorStillLandsOnContext(t *testing.T) {
	admin := id.NewUserID()
	ctx := middleware.WithImpersonator(context.Background(), admin)

	got, ok := middleware.ImpersonatorFrom(ctx)
	require.True(t, ok, "the impersonator carrier must keep working")
	assert.Equal(t, admin.String(), got.String())
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./middleware/ -run 'Principal|Actors|Impersonator' -v`
Expected: FAIL, `middleware.WithPrincipal` undefined.

- [ ] **Step 3: Add the carriers**

In `middleware/context.go`, following the existing `With`/`From` convention.
They delegate to the `principal` package rather than defining their own keys,
so a plugin reading through `principal.FromContext` sees the same value:

```go
// WithPrincipal returns ctx carrying the resolved caller.
func WithPrincipal(ctx context.Context, p *principal.Principal) context.Context {
	return principal.NewContext(ctx, p)
}

// PrincipalFrom returns the resolved caller.
func PrincipalFrom(ctx context.Context) (*principal.Principal, bool) {
	return principal.FromContext(ctx)
}

// WithActors returns ctx carrying the actor chain.
func WithActors(ctx context.Context, c principal.Chain) context.Context {
	return principal.NewActorsContext(ctx, c)
}

// ActorsFrom returns the actor chain.
func ActorsFrom(ctx context.Context) (principal.Chain, bool) {
	return principal.ActorsFromContext(ctx)
}
```

Add the resolver type next to `UserResolver` in `middleware/auth.go`:

```go
// PrincipalResolver resolves a caller by ref. Middleware takes this as a
// function rather than an engine so it does not import the engine package.
type PrincipalResolver func(principal.Ref) (*principal.Principal, error)
```

- [ ] **Step 4: Populate the context from both auth paths**

Add one helper to `middleware/auth.go`:

```go
// setPrincipalContext resolves the session's subject and puts it, and the
// actor chain, on the context.
//
// A resolution failure is logged and passed over rather than failing the
// request. The session already authenticated the caller; this is enrichment,
// and refusing the request over it would turn a principal-store blip into an
// outage on traffic that is otherwise fine.
func setPrincipalContext(
	goCtx context.Context, sess *session.Session, resolve PrincipalResolver, logger log.Logger,
) context.Context {
	if len(sess.Actors) > 0 {
		goCtx = WithActors(goCtx, sess.Actors)
	}
	if resolve == nil {
		return goCtx
	}
	ref := sess.Subject()
	if ref.IsZero() {
		return goCtx
	}
	p, err := resolve(ref)
	if err != nil {
		logger.Warn("auth middleware: failed to resolve principal",
			log.String("principal", ref.String()),
			log.String("error", err.Error()),
		)
		return goCtx
	}
	return WithPrincipal(goCtx, p)
}
```

Call it from `setSessionContext` immediately after the impersonator block, and
from `tryStrategyAuth` immediately after the session block. Both already hold
`goCtx` and `sess`. Thread a `PrincipalResolver` through
`AuthMiddlewareWithStrategies` and `AuthMiddleware` alongside the existing
`UserResolver`, defaulting to nil so a caller that has not wired one keeps
working.

`*user.User` continues to be set for human subjects, so `UserFrom(ctx)` and
every handler reading it are untouched.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./middleware/ ./... `
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
make check
git add -A
git commit -m "feat(middleware): resolve the caller as a principal on both auth paths"
```

---

## Task 11: The principal auth hooks

**Files:**
- Modify: `plugin/plugin.go`
- Modify: `plugin/registry.go`
- Test: `plugin/registry_test.go`

**Interfaces:**
- Consumes: Task 2's `principal.AuthAttempt`.
- Produces: `plugin.BeforePrincipalAuth`, `plugin.AfterPrincipalAuth`, `Registry.EmitBeforePrincipalAuth(ctx, *principal.AuthAttempt) error`, `Registry.EmitAfterPrincipalAuth(ctx, *principal.AuthAttempt, *session.Session)`.

- [ ] **Step 1: Write the failing test**

Add to `plugin/registry_test.go`:

```go
// A denying plugin must stop the authentication, the same way EmitBeforeSignIn
// does. This is the whole reason these are typed hooks and not hook.Bus
// events: Bus.Emit logs handler errors and returns nothing, so it cannot deny.
func TestEmitBeforePrincipalAuthStopsOnFirstError(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())

	first := &recordingPrincipalHook{name: "first"}
	denier := &denyingPrincipalHook{name: "denier"}
	last := &recordingPrincipalHook{name: "last"}
	require.NoError(t, r.Register(first))
	require.NoError(t, r.Register(denier))
	require.NoError(t, r.Register(last))

	err := r.EmitBeforePrincipalAuth(context.Background(), &principal.AuthAttempt{
		Subject:        principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		CredentialKind: "api_key",
	})
	require.Error(t, err)
	assert.True(t, first.called, "hooks before the denier must have run")
	assert.False(t, last.called, "hooks after the denier must not run")
}

func TestEmitAfterPrincipalAuthRunsEveryHook(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	a := &recordingPrincipalHook{name: "a"}
	b := &recordingPrincipalHook{name: "b"}
	require.NoError(t, r.Register(a))
	require.NoError(t, r.Register(b))

	r.EmitAfterPrincipalAuth(context.Background(),
		&principal.AuthAttempt{Subject: principal.Ref{Kind: principal.KindWorkload, ID: "svc_2"}},
		&session.Session{})

	assert.True(t, a.afterCalled)
	assert.True(t, b.afterCalled)
}
```

Write `recordingPrincipalHook` and `denyingPrincipalHook` in the same file,
implementing `Name()`, `OnBeforePrincipalAuth` and `OnAfterPrincipalAuth`,
following the existing test doubles in that file.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./plugin/ -run PrincipalAuth -v`
Expected: FAIL, `EmitBeforePrincipalAuth` undefined.

- [ ] **Step 3: Declare the hooks**

In `plugin/plugin.go`, after the sign-out hooks:

```go
// ──────────────────────────────────────────────────
// Principal auth hooks (non-human callers)
// ──────────────────────────────────────────────────

// BeforePrincipalAuth is called before a credential becomes a session for a
// caller that did not go through sign-in: an API key, a token exchange, a
// workload JWT.
//
// Returning an error denies the authentication. This is the machine-side
// counterpart to BeforeSignIn, and it exists because static API key traffic
// reaches strategy.Authenticate and never fires the sign-in hooks, so every
// risk plugin was blind to it.
type BeforePrincipalAuth interface {
	OnBeforePrincipalAuth(ctx context.Context, a *principal.AuthAttempt) error
}

// AfterPrincipalAuth is called once a non-human caller has a session. Errors
// are logged and do not fail the request, matching the other After hooks.
type AfterPrincipalAuth interface {
	OnAfterPrincipalAuth(ctx context.Context, a *principal.AuthAttempt, s *session.Session) error
}
```

- [ ] **Step 4: Wire the registry**

In `plugin/registry.go`, add the entry types next to `beforeSignInEntry`:

```go
type beforePrincipalAuthEntry struct {
	name string
	hook BeforePrincipalAuth
}
type afterPrincipalAuthEntry struct {
	name string
	hook AfterPrincipalAuth
}
```

Add the slices to `Registry`:

```go
	beforePrincipalAuth []beforePrincipalAuthEntry
	afterPrincipalAuth  []afterPrincipalAuthEntry
```

Add the registration branches next to the `BeforeSignIn` branch:

```go
	if h, ok := p.(BeforePrincipalAuth); ok {
		r.beforePrincipalAuth = append(r.beforePrincipalAuth, beforePrincipalAuthEntry{name, h})
	}
	if h, ok := p.(AfterPrincipalAuth); ok {
		r.afterPrincipalAuth = append(r.afterPrincipalAuth, afterPrincipalAuthEntry{name, h})
	}
```

Add the emitters next to `EmitBeforeSignIn`:

```go
// EmitBeforePrincipalAuth notifies all plugins that implement
// BeforePrincipalAuth, stopping at the first error.
func (r *Registry) EmitBeforePrincipalAuth(ctx context.Context, a *principal.AuthAttempt) error {
	for _, e := range r.beforePrincipalAuth {
		if err := e.hook.OnBeforePrincipalAuth(ctx, a); err != nil {
			return fmt.Errorf("plugin %s: %w", e.name, err)
		}
	}
	return nil
}

// EmitAfterPrincipalAuth notifies all plugins that implement
// AfterPrincipalAuth. Errors are logged, never propagated.
func (r *Registry) EmitAfterPrincipalAuth(ctx context.Context, a *principal.AuthAttempt, s *session.Session) {
	for _, e := range r.afterPrincipalAuth {
		if err := e.hook.OnAfterPrincipalAuth(ctx, a, s); err != nil {
			r.logger.Warn("plugin hook error",
				log.String("plugin", e.name),
				log.String("hook", "OnAfterPrincipalAuth"),
				log.String("error", err.Error()),
			)
		}
	}
}
```

Match the exact error-wrapping and logging style the neighbouring emitters use
in that file.

- [ ] **Step 5: Add the bus action constant**

In `hook/hook.go`, add to the action constants:

```go
	// ActionPrincipalAuth fires after a non-human caller has authenticated.
	// The typed BeforePrincipalAuth hook is what denies; this is the audit
	// and relay signal, so Chronicle picks up machine auth without any
	// subscriber implementing a plugin interface.
	ActionPrincipalAuth = "auth.principal"
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./plugin/ ./hook/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
make check
git add -A
git commit -m "feat(plugin): add BeforePrincipalAuth and AfterPrincipalAuth hooks"
```

---

## Task 12: Fire the hooks from API-key auth, with a cached verdict

This is the task that closes the risk gap. It also contains the one decision
this plan deliberately leaves to you: the cache eviction policy. See Step 4.

**Files:**
- Create: `plugin_principalauth.go`
- Create: `plugin_principalauth_test.go`
- Modify: `plugins/apikey/plugin.go` (`apikeyStrategy.Authenticate`)
- Modify: `plugins/apikey/contract.go`
- Modify: `engine.go` (wire the emitter into the apikey strategy at init)

**Interfaces:**
- Consumes: Task 11's emitters, Task 2's `AuthAttempt`.
- Produces: `authsome.principalAuthGate` with `Authorize(ctx context.Context, a *principal.AuthAttempt) error`, and `apikey.PrincipalAuthGate` as the narrow interface the plugin depends on:

```go
// PrincipalAuthGate scores a machine caller and may deny it. The apikey
// plugin holds this rather than the engine so it does not import authsome.
type PrincipalAuthGate interface {
	Authorize(ctx context.Context, a *principal.AuthAttempt) error
	Observe(ctx context.Context, a *principal.AuthAttempt, s *session.Session)
}
```

- [ ] **Step 1: Write the failing test**

Create `plugin_principalauth_test.go`. It needs `log "github.com/xraph/go-utils/log"` in its imports for `log.NewNoopLogger()`:

```go
package authsome

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/principal"
)

func attempt(credID, ip string) *principal.AuthAttempt {
	return &principal.AuthAttempt{
		Subject:        principal.Ref{Kind: principal.KindService, ID: "svc_1"},
		CredentialKind: "api_key",
		CredentialID:   credID,
		IPAddress:      ip,
		At:             time.Now(),
	}
}

// A denying plugin must deny the request. Without this the six risk plugins
// see machine traffic but cannot act on it, which is worse than not seeing it.
func TestPrincipalAuthGateDenies(t *testing.T) {
	denied := errors.New("blocked by risk")
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		return denied
	}, nil, time.Minute, log.NewNoopLogger())

	err := g.Authorize(context.Background(), attempt("akey_1", "1.2.3.4"))
	assert.ErrorIs(t, err, denied)
}

// A repeat call with the same credential and IP inside the TTL must not
// re-run the contributor chain. A chatty agent would otherwise pay for a geo
// and reputation lookup on every single call.
func TestPrincipalAuthGateCachesTheVerdict(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return nil
	}, nil, time.Minute, log.NewNoopLogger())

	ctx := context.Background()
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	assert.Equal(t, 1, calls, "the second call must be served from the cache")
}

// A different source IP is a different verdict. The same key used from a new
// country is exactly what impossibletravel and geofence exist to catch, so it
// must not ride the cached allow.
func TestPrincipalAuthGateKeysOnIP(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return nil
	}, nil, time.Minute, log.NewNoopLogger())

	ctx := context.Background()
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "5.6.7.8")))
	assert.Equal(t, 2, calls, "a new source IP must be scored fresh")
}

// A denial must not be cached as long as an allow, or a transient block
// outlives the condition that caused it. Denials are re-evaluated every time.
func TestPrincipalAuthGateDoesNotCacheDenials(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return errors.New("blocked")
	}, nil, time.Minute, log.NewNoopLogger())

	ctx := context.Background()
	_ = g.Authorize(ctx, attempt("akey_1", "1.2.3.4"))
	_ = g.Authorize(ctx, attempt("akey_1", "1.2.3.4"))
	assert.Equal(t, 2, calls, "a denial must be re-evaluated, not cached")
}

// An expired entry is re-scored.
func TestPrincipalAuthGateExpiresEntries(t *testing.T) {
	var calls int
	g := newPrincipalAuthGate(func(_ context.Context, _ *principal.AuthAttempt) error {
		calls++
		return nil
	}, nil, time.Nanosecond, log.NewNoopLogger())

	ctx := context.Background()
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	time.Sleep(time.Millisecond)
	require.NoError(t, g.Authorize(ctx, attempt("akey_1", "1.2.3.4")))
	assert.Equal(t, 2, calls)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test . -run TestPrincipalAuthGate -v`
Expected: FAIL, `newPrincipalAuthGate` undefined.

- [ ] **Step 3: Write the gate**

Create `plugin_principalauth.go`:

```go
package authsome

import (
	"context"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// principalAuthGate runs the BeforePrincipalAuth chain for machine callers and
// caches the verdict.
//
// The cache is what makes scoring machine traffic affordable. Sign-in happens
// once a day per human; an agent authenticates on every call, and the risk
// contributors do geo lookups and reputation lookups. Keyed on credential and
// source IP together, so the same key appearing from a new address is scored
// fresh rather than riding an earlier allow.
type principalAuthGate struct {
	authorize func(context.Context, *principal.AuthAttempt) error
	observe   func(context.Context, *principal.AuthAttempt, *session.Session)
	ttl       time.Duration
	logger    log.Logger

	mu      sync.Mutex
	entries map[string]time.Time // cache key -> when the allow expires
}

func newPrincipalAuthGate(
	authorize func(context.Context, *principal.AuthAttempt) error,
	observe func(context.Context, *principal.AuthAttempt, *session.Session),
	ttl time.Duration,
	logger log.Logger,
) *principalAuthGate {
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	return &principalAuthGate{
		authorize: authorize,
		observe:   observe,
		ttl:       ttl,
		logger:    logger,
		entries:   make(map[string]time.Time),
	}
}

func cacheKey(a *principal.AuthAttempt) string {
	return a.CredentialID + "|" + a.IPAddress
}

// Authorize scores a, denying if any plugin does.
//
// Only allows are cached. A denial is re-evaluated every time, because the
// condition behind it (a reputation listing, a travel impossibility) can clear
// within the TTL, and a cached denial would keep a caller locked out after the
// reason had gone.
func (g *principalAuthGate) Authorize(ctx context.Context, a *principal.AuthAttempt) error {
	if g.authorize == nil {
		return nil
	}

	key := cacheKey(a)
	now := a.At
	if now.IsZero() {
		now = time.Now()
	}

	g.mu.Lock()
	expires, cached := g.entries[key]
	g.mu.Unlock()
	if cached && now.Before(expires) {
		return nil
	}

	if err := g.authorize(ctx, a); err != nil {
		return err
	}

	g.mu.Lock()
	g.entries[key] = now.Add(g.ttl)
	g.evictLocked(now)
	g.mu.Unlock()
	return nil
}

// Observe runs the after hooks and emits the bus event.
func (g *principalAuthGate) Observe(ctx context.Context, a *principal.AuthAttempt, s *session.Session) {
	if g.observe != nil {
		g.observe(ctx, a, s)
	}
}
```

- [ ] **Step 4: YOUR CONTRIBUTION, the eviction policy**

`evictLocked(now time.Time)` is stubbed in the file above and needs writing.
The trade-off is real and it is yours to make, so the plan does not decide it.

The map grows one entry per (credential, source IP) pair. For a fleet of agents
behind rotating egress addresses that is unbounded, and this map lives for the
process lifetime on the authentication hot path.

Options, roughly:

- Sweep expired entries on every write. Simple and keeps the map honest, but
  it is O(n) under the lock on a path that runs per request.
- Sweep only when the map crosses a size threshold. Amortizes the cost, at the
  price of a periodic latency spike and a bounded overshoot.
- Hard cap with random or oldest-first eviction once full. Bounded memory
  guaranteed, but a busy credential can be evicted by a burst of one-off ones
  and lose its cached allow.
- A time-bucketed two-map rotation. Cheap eviction, coarse expiry.

Write it in `plugin_principalauth.go`:

```go
// evictLocked removes entries that can no longer serve a hit. The caller holds
// g.mu.
//
// This runs on the authentication path for every machine caller, so what it
// costs matters as much as what it reclaims.
func (g *principalAuthGate) evictLocked(now time.Time) {
	// TODO: implement the eviction policy.
}
```

Add a test to `plugin_principalauth_test.go` pinning whatever bound you chose,
so the next person cannot quietly regress it. If you picked a hard cap, assert
the map never exceeds it after inserting well past the cap.

- [ ] **Step 5: Wire the gate into the engine**

In `engine.go`, where the apikey plugin is initialized, build the gate from the
registry and hand it to the plugin:

```go
	gate := newPrincipalAuthGate(
		e.plugins.EmitBeforePrincipalAuth,
		func(ctx context.Context, a *principal.AuthAttempt, s *session.Session) {
			e.plugins.EmitAfterPrincipalAuth(ctx, a, s)
			e.hooks.Emit(ctx, &hook.Event{
				Action:     hook.ActionPrincipalAuth,
				Resource:   hook.ResourceSession,
				ResourceID: s.ID.String(),
				ActorID:    a.Subject.ID,
				Tenant:     a.AppID.String(),
				Metadata: map[string]string{
					"principal_kind":  string(a.Subject.Kind),
					"credential_kind": a.CredentialKind,
					"credential_id":   a.CredentialID,
					"ip":              a.IPAddress,
				},
			})
		},
		e.principalAuthTTL,
		e.logger,
	)
```

Add `principalAuthTTL time.Duration` to the `Engine` struct with a five-minute
default, and a `WithPrincipalAuthTTL(d time.Duration) Option` in `option.go`
following the shape of the other options in that file.

- [ ] **Step 6: Fire from the API-key strategy**

In `plugins/apikey/plugin.go`, add `gate PrincipalAuthGate` to `apikeyStrategy`
and call it in `Authenticate` after the key is verified and before the
synthetic session is returned. Both branches, the service-account one and the
user-bound one, go through it:

```go
	subject := principal.Ref{Kind: principal.KindService, ID: key.ServiceAccountID.String()}
	if key.ServiceAccountID.IsNil() {
		subject = principal.Ref{Kind: principal.KindUser, ID: key.UserID.String()}
	}
	att := &principal.AuthAttempt{
		Subject:        subject,
		AppID:          key.AppID,
		EnvID:          key.EnvID,
		CredentialKind: "api_key",
		CredentialID:   key.ID.String(),
		IPAddress:      clientIP(r),
		UserAgent:      r.UserAgent(),
		At:             now,
	}
	if s.gate != nil {
		if err := s.gate.Authorize(ctx, att); err != nil {
			return nil, fmt.Errorf("apikey: %w", err)
		}
	}
```

and after the synthetic session is built, in both branches:

```go
	if s.gate != nil {
		s.gate.Observe(ctx, att, syntheticSession)
	}
```

A user-bound API key goes through the gate too. It is machine traffic whoever
it is billed to, and it is just as invisible to the sign-in hooks.

Write `clientIP(r *http.Request) string` in the same file if the plugin does
not already have one, reading `X-Forwarded-For` then `X-Real-IP` then
`r.RemoteAddr`, matching whatever `middleware.clientIPFromRequest` does so the
two agree.

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test . ./plugins/apikey/ -v`
Expected: PASS, including a new case in `plugins/apikey/plugin_test.go`
asserting that a gate returning an error makes `Authenticate` fail.

- [ ] **Step 8: Commit**

```bash
make check
git add -A
git commit -m "feat(apikey): score machine callers through the principal auth hooks

Static API key traffic reached strategy.Authenticate and fired none of the
sign-in hooks, so all six risk plugins were blind to it. The verdict is
cached per credential and source IP so a chatty agent does not pay for a geo
lookup on every call."
```

---

## Task 13: The risk plugins adopt the hook

All six need work. Three of them (`ipreputation`, `geofence`, `vpndetect`)
already have their logic extracted into
`check(ctx context.Context, ipAddress, appID string) error`, so for those this
is a three-line method each. `riskengine` aggregates externally-registered
contributors and needs its own entry point. `impossibletravel` and `anomaly`
record history keyed by user and need re-keying by principal.

Note on `RiskRequest`: it carries `UserID string`, which is empty for a machine
caller. This task adds a `Principal string` field rather than stuffing a
service-account id into `UserID`, because an externally-registered contributor
reading `UserID` would otherwise silently start receiving something that is not
a user id.

**Files:**
- Modify: `plugins/riskengine/plugin.go`
- Modify: `plugins/ipreputation/plugin.go`
- Modify: `plugins/geofence/plugin.go`
- Modify: `plugins/vpndetect/plugin.go`
- Modify: `plugins/impossibletravel/plugin.go`
- Modify: `plugins/anomaly/plugin.go`
- Test: the existing `plugin_test.go` in each

**Interfaces:**
- Consumes: Task 11's `plugin.BeforePrincipalAuth` and `plugin.AfterPrincipalAuth`.
- Produces: `riskengine.RiskRequest.Principal string` (a `principal.Ref` rendered as `kind:id`). `impossibletravel.LoginLocation.Principal principal.Ref` replacing its `UserID` field, and `impossibletravel.Plugin.recordLocation(ctx context.Context, ref principal.Ref, ipAddress string) error`. `anomaly.LoginPattern.Principal principal.Ref` replacing its `UserID` field, and `anomaly.Plugin.recordLogin(ctx context.Context, ref principal.Ref, ipAddress string) error`.

- [ ] **Step 1: Write the failing tests**

In `plugins/ipreputation/plugin_test.go`, and the same shape in
`plugins/geofence/plugin_test.go` and `plugins/vpndetect/plugin_test.go` with
each one's own blocking condition:

```go
// A machine caller from a bad IP must be denied, exactly as a person is.
// This is the gap: API-key traffic never reached OnBeforeSignIn.
func TestIPReputationBlocksPrincipalAuth(t *testing.T) {
	p := ipreputation.New(ipreputation.Config{Provider: blockingProvider{}})
	require.NoError(t, p.OnInit(context.Background(), newTestEngine(t)))

	err := p.OnBeforePrincipalAuth(context.Background(), &principal.AuthAttempt{
		Subject:        principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		AppID:          id.NewAppID(),
		IPAddress:      badIP,
		CredentialKind: "api_key",
	})
	assert.Error(t, err)
}

// An attempt with no IP must pass rather than fail closed, matching what
// check already does for sign-in. A blank IP is a deployment detail, not a
// signal.
func TestIPReputationAllowsAttemptWithNoIP(t *testing.T) {
	p := ipreputation.New(ipreputation.Config{Provider: blockingProvider{}})
	require.NoError(t, p.OnInit(context.Background(), newTestEngine(t)))

	assert.NoError(t, p.OnBeforePrincipalAuth(context.Background(), &principal.AuthAttempt{
		Subject: principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		AppID:   id.NewAppID(),
	}))
}
```

In `plugins/riskengine/plugin_test.go`:

```go
func TestRiskEngineBlocksPrincipalAuth(t *testing.T) {
	p := riskengine.New()
	require.NoError(t, p.OnInit(context.Background(), newTestEngine(t)))
	p.AddContributor(alwaysHighRisk{})

	err := p.OnBeforePrincipalAuth(context.Background(), &principal.AuthAttempt{
		Subject:        principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		AppID:          id.NewAppID(),
		IPAddress:      "1.2.3.4",
		CredentialKind: "api_key",
	})
	assert.Error(t, err, "a high-risk machine caller must be denied")
}

// With no contributors the hook is a no-op, matching OnBeforeSignIn.
// Otherwise installing riskengine alone would deny every machine caller.
func TestRiskEngineAllowsWithNoContributors(t *testing.T) {
	p := riskengine.New()
	require.NoError(t, p.OnInit(context.Background(), newTestEngine(t)))

	assert.NoError(t, p.OnBeforePrincipalAuth(context.Background(), &principal.AuthAttempt{
		Subject: principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
	}))
}

// The contributor must be able to tell a machine caller from a person.
// UserID stays empty for machines rather than being filled with a
// service-account id, which an existing contributor would misread.
func TestRiskEngineSetsPrincipalNotUserID(t *testing.T) {
	p := riskengine.New()
	require.NoError(t, p.OnInit(context.Background(), newTestEngine(t)))
	spy := &capturingContributor{}
	p.AddContributor(spy)

	require.NoError(t, p.OnBeforePrincipalAuth(context.Background(), &principal.AuthAttempt{
		Subject:   principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		AppID:     id.NewAppID(),
		IPAddress: "1.2.3.4",
	}))

	require.NotNil(t, spy.last)
	assert.Equal(t, "agent:svc_1", spy.last.Principal)
	assert.Empty(t, spy.last.UserID, "a machine caller has no user id")
}
```

Write `capturingContributor` in that file, implementing `Name()` and
`EvaluateRisk` and storing the last `*RiskRequest` it saw.

In `plugins/impossibletravel/plugin_test.go`:

```go
// History is keyed by principal, not by user. Two agents must not share one
// travel history, and an agent must not inherit a user's.
func TestImpossibleTravelKeysByPrincipal(t *testing.T) {
	p := impossibletravel.New(impossibletravel.Config{
		MaxSpeedKmH:    900,
		MinDistanceKm:  100,
		LookbackWindow: time.Hour,
	})
	require.NoError(t, p.OnInit(context.Background(), newTestEngineWithGeoIP(t)))
	ctx := context.Background()

	agentA := principal.Ref{Kind: principal.KindAgent, ID: "svc_a"}
	agentB := principal.Ref{Kind: principal.KindAgent, ID: "svc_b"}

	require.NoError(t, p.OnAfterPrincipalAuth(ctx,
		&principal.AuthAttempt{Subject: agentA, IPAddress: londonIP},
		&session.Session{IPAddress: londonIP}))

	// Agent B's first sighting is in Sydney. With a shared history this would
	// score as London to Sydney in no time at all; with per-principal history
	// it is simply a first login and produces no event.
	require.NoError(t, p.OnAfterPrincipalAuth(ctx,
		&principal.AuthAttempt{Subject: agentB, IPAddress: sydneyIP},
		&session.Session{IPAddress: sydneyIP}))

	assert.Empty(t, p.RecordedEvents(), "one agent's history must not contaminate another's")
}
```

`RecordedEvents()` does not exist yet. Add it to `impossibletravel.Plugin` as a
test-visible accessor returning the impossible-travel events the plugin has
raised, or assert against whatever the plugin already exposes for that in its
existing tests. Check `plugins/impossibletravel/plugin_test.go` first and reuse
its existing assertion mechanism rather than adding a second one.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./plugins/riskengine/ ./plugins/ipreputation/ ./plugins/geofence/ ./plugins/vpndetect/ ./plugins/impossibletravel/ ./plugins/anomaly/ -v`
Expected: FAIL, `OnBeforePrincipalAuth` undefined on all six.

- [ ] **Step 3: The three IP-based plugins**

Each already has `check(ctx, ipAddress, appID string) error`, so each gets the
same three-line method. In `plugins/ipreputation/plugin.go`:

```go
// OnBeforePrincipalAuth checks the IP reputation of a machine caller.
//
// The same check sign-in gets. API-key traffic reaches strategy.Authenticate
// and fires none of the sign-in hooks, so without this a leaked key works from
// any address in the world.
func (p *Plugin) OnBeforePrincipalAuth(ctx context.Context, a *principal.AuthAttempt) error {
	return p.check(ctx, a.IPAddress, a.AppID.String())
}
```

Add the identical method to `plugins/geofence/plugin.go` and
`plugins/vpndetect/plugin.go`, changing only the doc comment to name what that
plugin checks. Add `github.com/xraph/authsome/principal` to each import block.

- [ ] **Step 4: riskengine**

Add the field to `RiskRequest` in `plugins/riskengine/plugin.go`:

```go
type RiskRequest struct {
	IPAddress string
	UserAgent string
	AppID     string
	// UserID is the human this request is for, empty for a machine caller.
	UserID string
	// Principal is the caller rendered as "kind:id". Set for every request,
	// including sign-in, so a contributor has one field it can always key on.
	// UserID is left empty rather than filled with a service-account id,
	// because an existing contributor reading it expects a user.
	Principal string
}
```

Set `Principal` in the existing `OnBeforeSignIn` as well, so contributors see
it on both paths:

```go
	riskReq := &RiskRequest{
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		AppID:     req.AppID.String(),
		Principal: principal.Ref{Kind: principal.KindUser, ID: req.Email}.String(),
	}
```

If `account.SignInRequest` carries no user id at that point, which is likely
since the user has not been resolved yet, leave `Principal` empty on the
sign-in path and say so in the field comment rather than putting an email in an
id position. Confirm before writing it.

Add the hook:

```go
// OnBeforePrincipalAuth scores a machine caller and blocks above the high
// threshold, exactly as OnBeforeSignIn does for a person.
func (p *Plugin) OnBeforePrincipalAuth(ctx context.Context, a *principal.AuthAttempt) error {
	if len(p.contributors) == 0 {
		return nil
	}

	riskReq := &RiskRequest{
		IPAddress: a.IPAddress,
		UserAgent: a.UserAgent,
		AppID:     a.AppID.String(),
		Principal: a.Subject.String(),
	}

	assessment := p.evaluate(ctx, riskReq)
	p.lastAssessment = assessment
	p.auditAssessment(ctx, riskReq, assessment)

	if assessment.Decision == "block" {
		return fmt.Errorf("riskengine: %s", p.config.BlockMessage)
	}
	return nil
}
```

- [ ] **Step 5: impossibletravel**

In `plugins/impossibletravel/plugin.go`, change `LoginLocation.UserID id.UserID`
to `Principal principal.Ref`, and the `lastLogins` map key from
`u.ID.String()` to `ref.String()`. Extract the body of `OnAfterSignIn` into
`recordLocation` and have both hooks call it:

```go
// OnAfterSignIn records a person's login location.
func (p *Plugin) OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error {
	return p.recordLocation(ctx, principal.UserRef(u.ID), s.IPAddress)
}

// OnAfterPrincipalAuth records a machine caller's location.
//
// Keyed by principal rather than user. An agent has no user, and two agents
// sharing one history would score each other's movements as travel, which on
// a fleet spread across regions means a permanent false positive.
func (p *Plugin) OnAfterPrincipalAuth(ctx context.Context, a *principal.AuthAttempt, s *session.Session) error {
	ip := a.IPAddress
	if ip == "" && s != nil {
		ip = s.IPAddress
	}
	return p.recordLocation(ctx, a.Subject, ip)
}
```

`recordLocation` takes the whole existing body of `OnAfterSignIn` with
`userKey` replaced by `ref.String()` and `current.UserID` replaced by
`current.Principal = ref`. Anywhere the body reaches for `u.ID` for an audit
or event field, use `ref.String()`.

- [ ] **Step 6: anomaly**

Apply exactly the same shape to `plugins/anomaly/plugin.go`:
`LoginPattern.UserID` becomes `Principal principal.Ref`, the `patterns` map key
becomes `ref.String()`, the body of `OnAfterSignIn` moves into
`recordLogin(ctx context.Context, ref principal.Ref, ipAddress string) error`,
and both hooks call it, with `OnAfterPrincipalAuth` falling back to the
session IP the same way impossibletravel does.

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./plugins/... -v`
Expected: PASS, all six risk plugins plus everything else under `plugins/`.

- [ ] **Step 8: Verify the gap is actually closed**

Add one end-to-end assertion in `plugins/apikey/plugin_test.go` that ties Task
12 to this one, since that is the behaviour the whole change exists for:

```go
// The regression test for the original gap. Before this change an API key
// authenticated with all six risk plugins installed and none of them ran.
func TestAPIKeyAuthRunsTheRiskChain(t *testing.T) {
	blocked := false
	gate := gateFunc(func(_ context.Context, _ *principal.AuthAttempt) error {
		blocked = true
		return errors.New("blocked by risk")
	})
	s := newAPIKeyStrategy(t, gate)

	_, err := s.Authenticate(context.Background(), requestWithValidKey(t))
	assert.Error(t, err)
	assert.True(t, blocked, "API key auth must run the principal auth chain")
}
```

- [ ] **Step 9: Commit**

```bash
make check
git add -A
git commit -m "feat(risk): score non-human callers across all six risk plugins

ipreputation, geofence and vpndetect get the same IP check sign-in gets.
riskengine gains a Principal field on RiskRequest so a contributor can tell a
machine caller from a person without UserID being filled with something that
is not a user id. impossibletravel and anomaly key history by principal, so
two agents do not share one travel or login history."
```

---

## Task 14: Delegation lifecycle and token exchange

**Files:**
- Modify: `engine_principal.go`
- Create: `engine_token_exchange.go`
- Create: `engine_token_exchange_test.go`
- Create: `api/principal_handlers.go`
- Modify: `api/api.go` (add the registerer to `rootRegisterers`)

**Interfaces:**
- Consumes: Task 4's `principal.Store`, Task 8's `Can`, Task 12's gate.
- Produces:

```go
func (e *Engine) GrantDelegation(ctx context.Context, appID id.AppID, actor, subject principal.Ref,
    scopes []string, grantedBy principal.Ref, expiresAt *time.Time) (*principal.Delegation, error)
func (e *Engine) RevokeDelegation(ctx context.Context, delID id.DelegationID) error
func (e *Engine) ListDelegationsForSubject(ctx context.Context, appID id.AppID, subject principal.Ref) ([]*principal.Delegation, error)
func (e *Engine) ExchangeToken(ctx context.Context, req *ExchangeRequest) (*session.Session, error)

type ExchangeRequest struct {
    AppID            id.AppID
    Actor            principal.Ref
    RequestedSubject principal.Ref
    Scopes           []string
    IPAddress        string
    UserAgent        string
    CredentialID     string
}
```

- [ ] **Step 1: Write the failing test**

Create `engine_token_exchange_test.go`:

```go
package authsome

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/principal"
)

// Exchange without a grant must fail. The endpoint exercises authority
// somebody already gave; it has no path that creates any.
func TestExchangeWithoutGrantIsRefused(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)

	_, err := e.ExchangeToken(context.Background(), &ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	assert.Error(t, err, "no grant means no exchange")
}

func TestExchangeMintsADelegatedSession(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	_, err := e.GrantDelegation(ctx, appID, agent, userRef,
		[]string{"repo:read"}, userRef, nil)
	require.NoError(t, err)

	sess, err := e.ExchangeToken(ctx, &ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
		Scopes: []string{"repo:read"},
	})
	require.NoError(t, err)

	// The subject is the human. This is what keeps session.UserID meaning
	// "the person this request is for" for every existing consumer.
	assert.Equal(t, userRef.ID, sess.UserID.String())
	assert.Equal(t, principal.KindUser, sess.PrincipalKind)
	assert.Equal(t, principal.Chain{agent}, sess.Actors)
	assert.Equal(t, principal.GrantDelegation, sess.ActorGrant)
	assert.False(t, sess.DelegationID.IsNil())
	assert.True(t, sess.ImpersonatedBy().IsNil(), "a delegation is not impersonation")
}

// A scope the grant does not carry must not survive the exchange. This is
// where the spec's scope filter is enforced.
func TestExchangeIntersectsScopes(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	_, err := e.GrantDelegation(ctx, appID, agent, userRef, []string{"repo:read"}, userRef, nil)
	require.NoError(t, err)

	_, err = e.ExchangeToken(ctx, &ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
		Scopes: []string{"repo:write"},
	})
	assert.Error(t, err, "a scope outside the grant must be refused, not silently dropped")
}

// A revoked grant stops working immediately, which is the entire point of
// storing grants rather than asserting the chain per request.
func TestExchangeRefusesRevokedGrant(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	d, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, nil)
	require.NoError(t, err)
	require.NoError(t, e.RevokeDelegation(ctx, d.ID))

	_, err = e.ExchangeToken(ctx, &ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	assert.Error(t, err)
}

// The session's lifetime cannot outlive the grant's. A one-hour grant that
// mints a thirty-day session is a grant nobody actually revoked.
func TestExchangeBoundsSessionTTLByGrantExpiry(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	soon := time.Now().Add(5 * time.Minute)
	_, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, &soon)
	require.NoError(t, err)

	sess, err := e.ExchangeToken(ctx, &ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	require.NoError(t, err)
	assert.False(t, sess.ExpiresAt.After(soon), "the session must not outlive the grant")
}
```

Write `setupExchangeFixture(t)` returning an engine, an app ID, an agent ref
and a user ref, using the engine construction helper `engine_test.go` already
uses and seeding one agent principal and one user.

- [ ] **Step 2: Run to verify it fails**

Run: `go test . -run TestExchange -v`
Expected: FAIL, `ExchangeToken` undefined.

- [ ] **Step 3: Add the delegation lifecycle to `engine_principal.go`**

```go
// GrantDelegation records that actor may act for subject.
//
// grantedBy is who consented and is recorded for audit. The caller is
// responsible for having checked that grantedBy is entitled to consent: for an
// ordinary delegation that means grantedBy is the subject or an admin over it,
// and the API layer enforces it before calling here.
func (e *Engine) GrantDelegation(
	ctx context.Context, appID id.AppID, actor, subject principal.Ref,
	scopes []string, grantedBy principal.Ref, expiresAt *time.Time,
) (*principal.Delegation, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	if appID.IsNil() {
		return nil, fmt.Errorf("authsome: app_id is required")
	}
	if actor.IsZero() || subject.IsZero() {
		return nil, fmt.Errorf("authsome: actor and subject are required")
	}
	if actor == subject {
		// Acting for yourself is not a delegation, and storing one would put a
		// chain on a session that has no second principal on it.
		return nil, fmt.Errorf("authsome: a principal cannot be delegated to itself")
	}

	now := time.Now()
	d := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     appID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantDelegation,
		Scopes:    scopes,
		GrantedBy: grantedBy,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := e.store.CreateDelegation(ctx, d); err != nil {
		return nil, fmt.Errorf("authsome: grant delegation: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionDelegationGrant,
		Resource:   hook.ResourceSession,
		ResourceID: d.ID.String(),
		ActorID:    grantedBy.ID,
		Tenant:     appID.String(),
		Metadata: map[string]string{
			"actor":   actor.String(),
			"subject": subject.String(),
		},
	})
	return d, nil
}

// RevokeDelegation ends a grant.
func (e *Engine) RevokeDelegation(ctx context.Context, delID id.DelegationID) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	if err := e.store.RevokeDelegation(ctx, delID, time.Now()); err != nil {
		return fmt.Errorf("authsome: revoke delegation: %w", err)
	}
	return nil
}

// ListDelegationsForSubject returns what may act for subject, so a person can
// see and revoke the agents holding authority over their account.
func (e *Engine) ListDelegationsForSubject(
	ctx context.Context, appID id.AppID, subject principal.Ref,
) ([]*principal.Delegation, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	return e.store.ListDelegations(ctx, &principal.DelegationQuery{
		AppID: appID, Subject: &subject, ActiveOnly: true, ActiveAsOf: time.Now(),
	})
}
```

Add `hook.ActionDelegationGrant = "principal.delegation.grant"` and
`hook.ActionDelegationRevoke = "principal.delegation.revoke"` to the action
constants, and emit the revoke event from `RevokeDelegation` the same way.

- [ ] **Step 4: Write `engine_token_exchange.go`**

```go
package authsome

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// ExchangeRequest asks for a session in which Actor acts for RequestedSubject.
type ExchangeRequest struct {
	AppID            id.AppID
	Actor            principal.Ref
	RequestedSubject principal.Ref
	Scopes           []string
	IPAddress        string
	UserAgent        string
	// CredentialID identifies the credential the actor authenticated with, so
	// the risk verdict caches against it.
	CredentialID string
}

// ExchangeToken mints a session in which the actor acts on the subject's
// behalf, against a grant that already exists.
//
// It never creates authority. Without a live grant matching (app, actor,
// subject) this fails, which is what makes revocation meaningful.
func (e *Engine) ExchangeToken(ctx context.Context, req *ExchangeRequest) (*session.Session, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}

	grant, err := e.store.FindActiveDelegation(
		ctx, req.AppID, req.Actor, req.RequestedSubject, principal.GrantDelegation)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: no active delegation for %s acting as %s: %w",
			req.Actor, req.RequestedSubject, err)
	}

	actorPrincipal, err := e.store.GetPrincipal(ctx, req.Actor)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: resolve actor: %w", err)
	}
	now := time.Now()
	if !actorPrincipal.IsActive(now) {
		return nil, fmt.Errorf("authsome: exchange: actor %s is disabled or expired", req.Actor)
	}

	// Scope narrowing. Refuse rather than silently drop: an agent that asked
	// for repo:write and got a session without it fails later, somewhere far
	// from the cause, and reads as a bug in the agent.
	scopes, err := intersectScopes(req.Scopes, grant, actorPrincipal.Scopes)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: %w", err)
	}

	att := &principal.AuthAttempt{
		Subject:        req.RequestedSubject,
		Actors:         principal.Chain{req.Actor},
		AppID:          req.AppID,
		CredentialKind: "token_exchange",
		CredentialID:   req.CredentialID,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		Ephemeral:      actorPrincipal.IsEphemeral(),
		At:             now,
	}
	if err := e.plugins.EmitBeforePrincipalAuth(ctx, att); err != nil {
		return nil, fmt.Errorf("authsome: exchange: %w", err)
	}

	userID, err := id.ParseUserID(req.RequestedSubject.ID)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: subject must be a user: %w", err)
	}

	cfg := e.sessionConfigForApp(ctx, req.AppID)
	// A session must not outlive the grant that justified it.
	if grant.ExpiresAt != nil {
		if remaining := time.Until(*grant.ExpiresAt); remaining < cfg.TokenTTL {
			cfg.TokenTTL = remaining
			cfg.RefreshTokenTTL = remaining
		}
	}

	sess, err := e.newSession(req.AppID, userID, cfg)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: create session: %w", err)
	}
	sess.PrincipalKind = principal.KindUser
	sess.Actors = principal.Chain{req.Actor}
	sess.ActorGrant = principal.GrantDelegation
	sess.DelegationID = grant.ID
	sess.IPAddress = req.IPAddress
	sess.UserAgent = req.UserAgent
	_ = scopes // recorded on the session by whichever scope field the app uses

	if err := e.store.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("authsome: exchange: store session: %w", err)
	}

	e.plugins.EmitAfterPrincipalAuth(ctx, att, sess)
	return sess, nil
}

// intersectScopes returns the scopes an exchanged session may carry.
//
// Every requested scope must be inside both the grant's filter and the actor's
// own scopes. Asking for one that is not is an error rather than a quiet
// removal.
func intersectScopes(requested []string, grant *principal.Delegation, actorScopes []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	actorHas := make(map[string]bool, len(actorScopes))
	for _, s := range actorScopes {
		actorHas[s] = true
	}
	out := make([]string, 0, len(requested))
	for _, s := range requested {
		if !grant.AllowsScope(s) {
			return nil, fmt.Errorf("scope %q is outside the delegation grant", s)
		}
		if len(actorScopes) > 0 && !actorHas[s] {
			return nil, fmt.Errorf("scope %q is outside the actor's own scopes", s)
		}
		out = append(out, s)
	}
	return out, nil
}
```

Replace the `_ = scopes` line with an assignment to whichever field the session
carries scopes on once you confirm it. If sessions have no scope field, drop
the variable and let `intersectScopes` serve purely as the gate, which is the
role that matters.

- [ ] **Step 5: Add the HTTP surface**

Create `api/principal_handlers.go` with a registerer following
`registerHealthRoutes` in `api/api.go`:

```go
func (a *API) registerPrincipalRoutes(router forge.Router) error {
	g := router.Group("/v1", forge.WithGroupTags("principals"))

	if err := g.POST("/token/exchange", a.handleTokenExchange,
		forge.WithSummary("Exchange a credential for a delegated session"),
		forge.WithDescription("Mints a session in which the calling principal acts on behalf of another, against an existing delegation grant. Returns 403 when no live grant matches."),
		forge.WithOperationID("exchangeToken"),
		forge.WithResponseSchema(http.StatusOK, "Delegated session", TokenExchangeResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.GET("/principals/me/delegations", a.handleListMyDelegations,
		forge.WithSummary("List what may act on your behalf"),
		forge.WithOperationID("listMyDelegations"),
		forge.WithResponseSchema(http.StatusOK, "Delegations", DelegationListResponse{}),
		forge.WithErrorResponses(),
	)
}
```

Add `a.registerPrincipalRoutes` to the `rootRegisterers` slice at
`api/api.go:100`. The handler reads the calling principal with
`middleware.PrincipalFrom(ctx.Context())`, refuses when absent, and passes
`req.RequestedSubject` through `id.ParseUserID` before calling
`ExchangeToken`. Define `TokenExchangeRequest`, `TokenExchangeResponse` and
`DelegationListResponse` in `api/requests.go` alongside the existing request
types, and never put the minted token anywhere but the response body.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test . ./api/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
make check
git add -A
git commit -m "feat(principal): add delegation grants and RFC 8693 token exchange

An agent exchanges its own credential for a session acting as a user, against
a grant that already exists. Scope narrowing is enforced here, at mint time,
so the per-check path stays free of a scope lookup."
```

---

## Task 15: Ephemeral children

**Files:**
- Modify: `engine_principal.go`
- Modify: `api/principal_handlers.go`
- Test: `engine_principal_test.go`

**Interfaces:**
- Consumes: Task 4's `ParentID` and `ExpiresAt`, Task 14's handlers.
- Produces:

```go
func (e *Engine) MintChildPrincipal(ctx context.Context, parentID id.ServiceAccountID,
    name string, scopes []string, ttl time.Duration) (*serviceaccount.ServiceAccount, *apikey.APIKey, string, error)
func (e *Engine) ReapExpiredPrincipals(ctx context.Context, appID id.AppID) (int, error)
```

- [ ] **Step 1: Write the failing test**

```go
// A child may not out-scope its parent. Otherwise minting a child is a
// privilege escalation with an extra step.
func TestMintChildRefusesScopesTheParentLacks(t *testing.T) {
	e, _, parent := setupParentFixture(t, []string{"repo:read"})

	_, _, _, err := e.MintChildPrincipal(context.Background(), parent.ID,
		"child", []string{"repo:write"}, time.Hour)
	assert.Error(t, err)
}

// A child may not outlive its parent, or revoking the parent leaves its
// children running.
func TestMintChildCapsTTLByParentExpiry(t *testing.T) {
	e, _, parent := setupParentFixtureExpiring(t, 5*time.Minute)

	child, _, _, err := e.MintChildPrincipal(context.Background(), parent.ID,
		"child", nil, 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, child.ExpiresAt)
	assert.False(t, child.ExpiresAt.After(*parent.ExpiresAt))
}

func TestMintChildRecordsTheParent(t *testing.T) {
	e, appID, parent := setupParentFixture(t, nil)

	child, key, secret, err := e.MintChildPrincipal(context.Background(), parent.ID,
		"child", nil, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, parent.ID.String(), child.ParentID.String())
	assert.Equal(t, appID.String(), child.AppID.String())
	assert.NotEmpty(t, secret, "the secret is returned once and never stored")
	assert.Equal(t, child.ID.String(), key.ServiceAccountID.String())
}

func TestReapRemovesExpiredChildrenOnly(t *testing.T) {
	e, appID, parent := setupParentFixture(t, nil)
	ctx := context.Background()

	lapsed, _, _, err := e.MintChildPrincipal(ctx, parent.ID, "lapsed", nil, -time.Hour)
	require.NoError(t, err)

	n, err := e.ReapExpiredPrincipals(ctx, appID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = e.GetServiceAccount(ctx, lapsed.ID)
	assert.Error(t, err, "the lapsed child must be gone")
	_, err = e.GetServiceAccount(ctx, parent.ID)
	assert.NoError(t, err, "the durable parent must survive the reaper")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test . -run 'TestMintChild|TestReap' -v`
Expected: FAIL, `MintChildPrincipal` undefined.

- [ ] **Step 3: Implement**

```go
// MintChildPrincipal creates a short-lived principal under a registered
// parent, with its own API key.
//
// This is what makes per-task agents workable: one durable registration, N
// ephemeral instances, each with its own identity for attribution and its own
// credential to revoke. The two caps are what keep it from being an escalation:
// a child's scopes are a subset of its parent's, and a child's expiry never
// passes its parent's.
//
// The secret is returned once and is not stored.
func (e *Engine) MintChildPrincipal(
	ctx context.Context, parentID id.ServiceAccountID, name string, scopes []string, ttl time.Duration,
) (*serviceaccount.ServiceAccount, *apikey.APIKey, string, error) {
	if err := e.requireStarted(); err != nil {
		return nil, nil, "", err
	}
	if ttl <= 0 {
		return nil, nil, "", fmt.Errorf("authsome: ttl must be positive")
	}

	parent, err := e.store.GetServiceAccount(ctx, parentID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: get parent: %w", err)
	}
	if !parent.Active {
		return nil, nil, "", fmt.Errorf("authsome: mint child: parent is inactive")
	}
	if parent.ParentID.Prefix() != "" {
		// One level only. A tree of ephemeral principals is a revocation
		// problem nobody can reason about, and nothing needs it.
		return nil, nil, "", fmt.Errorf("authsome: mint child: an ephemeral principal cannot mint children")
	}

	if err := requireScopeSubset(scopes, parent.Scopes); err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: %w", err)
	}

	now := time.Now()
	expires := now.Add(ttl)
	if parent.ExpiresAt != nil && expires.After(*parent.ExpiresAt) {
		expires = *parent.ExpiresAt
	}

	kind := parent.Kind
	if kind == "" {
		kind = principal.KindService
	}
	child := &serviceaccount.ServiceAccount{
		ID:          id.NewServiceAccountID(),
		AppID:       parent.AppID,
		EnvID:       parent.EnvID,
		OrgID:       parent.OrgID,
		Kind:        kind,
		Name:        name,
		ParentID:    parent.ID,
		OwnerUserID: parent.OwnerUserID,
		Scopes:      scopes,
		ExpiresAt:   &expires,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := e.store.CreateServiceAccount(ctx, child); err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: %w", err)
	}

	key, secret, err := e.CreateServiceAccountAPIKey(ctx, child.ID, name, scopes, &expires)
	if err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: create key: %w", err)
	}
	return child, key, secret, nil
}

// requireScopeSubset refuses any scope the parent does not itself hold. A
// parent with no scopes at all places no restriction, matching how an empty
// scope list is read everywhere else in this package.
func requireScopeSubset(child, parent []string) error {
	if len(parent) == 0 || len(child) == 0 {
		return nil
	}
	has := make(map[string]bool, len(parent))
	for _, s := range parent {
		has[s] = true
	}
	for _, s := range child {
		if !has[s] {
			return fmt.Errorf("scope %q is outside the parent's scopes", s)
		}
	}
	return nil
}

// ReapExpiredPrincipals deletes lapsed ephemeral children and returns how many
// went.
//
// Only children are reaped. A durable principal with an expiry set is one an
// operator chose to time-limit, and deleting it out from under them would
// destroy a registration they can see in the dashboard.
func (e *Engine) ReapExpiredPrincipals(ctx context.Context, appID id.AppID) (int, error) {
	if err := e.requireStarted(); err != nil {
		return 0, err
	}
	all, err := e.store.ListPrincipals(ctx, &principal.Query{AppID: appID})
	if err != nil {
		return 0, fmt.Errorf("authsome: reap: list principals: %w", err)
	}
	now := time.Now()
	reaped := 0
	for _, p := range all {
		if p.Parent == nil || !p.IsExpired(now) {
			continue
		}
		svcID, parseErr := id.ParseServiceAccountID(p.ID)
		if parseErr != nil {
			continue
		}
		if delErr := e.store.DeleteServiceAccount(ctx, svcID); delErr != nil {
			e.logger.Warn("authsome: reap: delete failed",
				log.String("principal", p.Ref.String()),
				log.String("error", delErr.Error()),
			)
			continue
		}
		reaped++
	}
	return reaped, nil
}
```

- [ ] **Step 4: Add the endpoint**

In `api/principal_handlers.go`, add to `registerPrincipalRoutes`:

```go
	if err := g.POST("/principals/:id/children", a.handleMintChild,
		forge.WithSummary("Mint an ephemeral child principal"),
		forge.WithDescription("Creates a short-lived principal under the calling parent, with its own credential. Scopes must be a subset of the parent's and the TTL is capped by the parent's expiry."),
		forge.WithOperationID("mintChildPrincipal"),
		forge.WithResponseSchema(http.StatusCreated, "Child principal", MintChildResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
```

The handler reads the calling principal from the context and refuses when the
path id does not match it, so a parent can only mint under itself.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test . ./api/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
make check
git add -A
git commit -m "feat(principal): mint ephemeral children under a registered parent

One durable registration, N per-task instances, each with its own identity and
credential. Scopes are capped by the parent's and the expiry never passes the
parent's, so minting a child cannot escalate."
```

---

## Task 16: Impersonation becomes a delegation grant

The last piece of the subsume. `Engine.Impersonate` already stamps the chain
(Task 3). This gives it a grant row, so an impersonation appears in the same
listing and revocation surface as everything else.

**Files:**
- Modify: `service.go:2679-2750`
- Test: `service_test.go`

**Interfaces:**
- Consumes: Task 14's grant lifecycle.
- Produces: no new exported API. `Impersonate` and `StopImpersonation` keep their signatures.

- [ ] **Step 1: Write the failing test**

Add to `service_test.go`:

```go
// An impersonation now leaves a grant behind, so it shows up wherever
// delegations do and can be revoked the same way.
func TestImpersonateRecordsAGrant(t *testing.T) {
	e, appID, admin, target := setupImpersonationFixture(t)
	ctx := context.Background()

	_, sess, err := e.Impersonate(ctx, admin.ID, target.ID)
	require.NoError(t, err)
	require.False(t, sess.DelegationID.IsNil(), "the session must name its grant")

	d, err := e.PrincipalStore().GetDelegation(ctx, sess.DelegationID)
	require.NoError(t, err)
	assert.Equal(t, principal.GrantImpersonation, d.GrantKind)
	assert.Equal(t, principal.UserRef(admin.ID), d.Actor)
	assert.Equal(t, principal.UserRef(target.ID), d.Subject)
	assert.Equal(t, appID.String(), d.AppID.String())
}

// The behaviour every existing consumer depends on must be unchanged.
func TestImpersonateStillReadsAsImpersonation(t *testing.T) {
	e, _, admin, target := setupImpersonationFixture(t)

	_, sess, err := e.Impersonate(context.Background(), admin.ID, target.ID)
	require.NoError(t, err)

	assert.Equal(t, admin.ID.String(), sess.ImpersonatedBy().String())
	assert.Equal(t, target.ID.String(), sess.UserID.String())
	assert.Empty(t, sess.AuthzActors(), "impersonation evaluates as the target alone")
}

// Ending an impersonation must revoke the grant as well as the session, or
// the admin keeps standing authority to start a new one silently.
func TestStopImpersonationRevokesTheGrant(t *testing.T) {
	e, _, admin, target := setupImpersonationFixture(t)
	ctx := context.Background()

	_, sess, err := e.Impersonate(ctx, admin.ID, target.ID)
	require.NoError(t, err)
	require.NoError(t, e.StopImpersonation(ctx, sess.ID))

	d, err := e.PrincipalStore().GetDelegation(ctx, sess.DelegationID)
	require.NoError(t, err)
	assert.NotNil(t, d.RevokedAt, "stopping must revoke the grant")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test . -run TestImpersonat -v`
Expected: FAIL, `sess.DelegationID` is nil because no grant is written.

- [ ] **Step 3: Write the grant in `Impersonate`**

In `service.go`, between creating the session and storing it:

```go
	now := time.Now()
	// Impersonation is a delegation with its own grant kind, so it appears in
	// the same listing and revocation surface as every other way one principal
	// comes to act for another. The one thing that stays special is how it is
	// authorized: Session.AuthzActors returns nil for it, so the admin's own
	// permissions are not intersected in.
	expires := now.Add(cfg.TokenTTL)
	grant := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     u.AppID,
		Actor:     principal.UserRef(adminID),
		Subject:   principal.UserRef(targetID),
		GrantKind: principal.GrantImpersonation,
		GrantedBy: principal.UserRef(adminID),
		ExpiresAt: &expires,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := e.store.CreateDelegation(ctx, grant); err != nil {
		return nil, nil, fmt.Errorf("authsome: impersonate: record grant: %w", err)
	}

	sess.SetImpersonatedBy(adminID)
	sess.DelegationID = grant.ID
```

- [ ] **Step 4: Revoke in `StopImpersonation`**

After the session is deleted:

```go
	if !sess.DelegationID.IsNil() {
		// Best effort. The session is already gone, so the impersonation has
		// ended whatever happens here, and failing the call would report an
		// unstopped impersonation that has in fact stopped.
		if err := e.store.RevokeDelegation(ctx, sess.DelegationID, time.Now()); err != nil {
			e.logger.Warn("authsome: stop impersonation: revoke grant failed",
				log.String("delegation_id", sess.DelegationID.String()),
				log.String("error", err.Error()),
			)
		}
	}
```

Note that a repeat `Impersonate` for the same admin and target while the first
is live hits the partial unique index and returns `store.ErrConflict`. Decide
whether that is the behaviour you want: it prevents an admin holding two
concurrent impersonation sessions for one user. If you want to allow it,
reuse the existing live grant instead of creating a second one, by calling
`FindActiveDelegation` first and only creating when it returns
`principal.ErrNotFound`.

- [ ] **Step 5: Run the full suite**

Run: `go test ./... && go test -tags integration ./store/...`
Expected: PASS everywhere, including the existing impersonation e2e coverage.

- [ ] **Step 6: Commit**

```bash
make check
git add -A
git commit -m "feat(auth): record impersonation as a delegation grant

Completes the subsume. An impersonation now appears in the same listing and
revocation surface as any other delegation, while keeping its one special
property: it evaluates as the target alone rather than intersecting the
admin's permissions in."
```

---

## Verification

After Task 16, the whole feature is in. Confirm before calling it done:

```bash
go build ./...
go test ./...
go test -tags integration ./store/...
make check
```

Every one of those must pass. The integration run is the one that matters most
here, because it is the only thing that proves postgres and sqlite actually
implement what they previously stubbed.

---

## Overlap with the sibling designs

Two other plans written the same day cover part of this ground. This is not a
merge conflict you can resolve by taking both.

**`2026-08-24-token-exchange-rfc8693.md`** puts an actor chain and a scope list
on `session.Session` and implements RFC 8693 inside the `oauth2provider`
plugin. Task 3 here puts the same actor chain on the same struct, and Task 14
implements the same exchange in the core engine.

**`2026-08-24-agentauth-delegation.md`** models an agent as an OAuth client
registered in `oauth2provider`, joined to a delegating user by an `AgentGrant`
row, with `PrincipalKind = "agent"` and the human left in `Session.UserID`.
That last part matches this design exactly. What differs is where the agent
identity lives: an OAuth client there, a row in the principal table here.

Pick one owner for each shared piece before starting:

- The actor chain on `session.Session` should be added once. This plan's Task 3
  is the more general version, since it also subsumes `ImpersonatedBy`. If the
  token-exchange plan lands first, Task 3 becomes a much smaller change that
  adds `ActorGrant` and the `ImpersonatedBy` method on top of the chain that
  already exists.
- The delegation grant table and the `AgentGrant` table are the same table
  under two names. Running both gives you two places to revoke an agent's
  authority and no way to know you have missed one.
- Token exchange should have one endpoint. Whether it lives in the core engine
  (Task 14) or in `oauth2provider` is a real choice: the plugin gets you OAuth
  client authentication for free, the core engine works for callers who are not
  OAuth clients at all, such as a CI workload holding an API key.
