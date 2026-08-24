# Non-human principals in the authsome core

Status: approved design, not yet implemented
Date: 2026-08-24

## The problem

authsome knows about two kinds of caller. There's a human, a `user.User` holding
a `session.Session`. And there's a static credential, an `apikey.APIKey`. That's
it.

An AI agent is neither. Nor is an MCP client, or a CI job that deploys on merge.
Today you have two options for those callers and both are wrong: you can mint a
user row, in which case the thing pollutes your user counts, receives
verification email it will never read, and has nobody answerable for it, or you
can hand it a long-lived API key, in which case you give up user attribution and
expiry semantics and the traffic becomes invisible to every risk plugin you
run.

Say that last one plainly, because it is a live gap and not a matter of taste.
Static API key traffic reaches `strategy.Authenticate` and never touches
`OnBeforeSignIn`, which means riskengine, impossibletravel, ipreputation,
anomaly, geofence and vpndetect score exactly none of it. Six plugins, zero
coverage, on the credential type most likely to end up committed to a public
repository.

## What already exists

A previous change (`6716455`) laid real groundwork, and this design builds on it
rather than around it.

You already have a `serviceaccount` package with an entity and a store
interface, sessions that carry `PrincipalKind` and `ServiceAccountID` with
migrations landed on postgres, sqlite and mongo, API keys that carry a
`ServiceAccountID`, and an apikey plugin that already mints a synthetic
service-account session when it sees one.

What is missing is the abstraction over all of it. `"service_account"` is a
bare string compared in five separate places, there is no way to say "agent A is
acting for user U", and `plugin.Engine` has no principal-shaped method on it. On
top of that, postgres and sqlite return `not implemented` from all five
service-account store methods, so the feature you think you shipped is dead on
the two backends most people deploy.

## The `principal` package

Three value types, no store dependency, sitting below `session` and `user` in
the import graph.

```go
package principal

type Kind string

const (
    KindUser     Kind = "user"
    KindAgent    Kind = "agent"           // AI agent, MCP client
    KindWorkload Kind = "workload"        // CI job, cron, service-to-service
    KindService  Kind = "service_account" // what already exists on disk
)

// Ref is an addressable identity: kind plus id. Comparable, cheap to put in a
// context, trivially serializable into a token claim.
type Ref struct {
    Kind Kind   `json:"kind"`
    ID   string `json:"id"`
}

// Principal is a resolved caller with everything authorization needs.
type Principal struct {
    Ref
    AppID     id.AppID
    OrgID     id.OrgID
    Name      string
    Scopes    []string
    Roles     []string
    Owner     *Ref       // who is answerable for this principal; nil for users
    Parent    *Ref       // registered parent, set on ephemeral children
    ExpiresAt *time.Time // hard TTL; nil for durable principals
    Disabled  bool
}
```

## The actor chain

Here is the part that turns `ImpersonatedBy` into a special case of the general
mechanism instead of a cousin sitting beside it. Two fields carry everything you
need, and they follow RFC 8693 so you are reading a vocabulary that already has
a specification behind it.

`Subject` is the principal the request is for. That's the RFC's `sub`. `Actors`
is who is doing the acting, ordered nearest-caller-first, which is the RFC's
nested `act`.

```go
type Chain []Ref

func (c Chain) Actor() (Ref, bool)   // c[0], the immediate caller
func (c Chain) Depth() int
func (c Chain) Contains(r Ref) bool
```

Every case falls out of those two fields:

| Case | Subject | Actors |
|---|---|---|
| Ordinary sign-in | `{user, U}` | `[]` |
| CI workload, no human | `{workload, W}` | `[]` |
| Agent A acting for U in org O | `{user, U}` | `[{agent, A}]` |
| Admin impersonating U | `{user, U}` | `[{user, admin}]` |
| Ephemeral agent A' under A, for U | `{user, U}` | `[{agent, A'}, {agent, A}]` |

`Session.ImpersonatedBy()` becomes the last actor of kind `user` reached through
an impersonation grant. Same behaviour you have now. One expression, and no
second mechanism to keep in sync.

Putting the delegating user in `Subject` rather than the agent is what buys
backwards compatibility almost for free. On an on-behalf-of session,
`session.UserID` still means "the human this request is for". Every consumer
that reads it keeps producing correct answers without knowing agents exist: the
audit records, the org scoping, the `forge.NewOrgScope` call. Put the agent in
`Subject` instead and you'd be updating all of them.

## Warden

A principal is a Warden subject in its own right, which is what lets a CI
workload hold a real role with no user standing behind it and no fake user row
invented to carry the grant. When an actor chain is present the decision
narrows, and narrowing is the only direction it can move.

One new engine method. The existing one stays byte-identical in signature, so
`plugin.PermissionChecker` and every current caller are untouched:

```go
// New.
func (e *Engine) Can(ctx context.Context, subject principal.Ref,
    actors principal.Chain, action, resource string) (bool, error)

// Existing signature preserved, delegates with an empty chain.
func (e *Engine) HasPermission(ctx context.Context, userID id.UserID,
    action, resource string) (bool, error)
```

Evaluation runs in order:

1. No actors, so a single `warden.Check` on the subject. Identical to today.
2. Actors present, so allow only if `Check(subject)` allows, and every actor's
   `Check` allows, and the action falls inside the delegation grant's scope
   filter. First deny short-circuits.
3. Impersonation is the documented exception. Actors reached through an
   `impersonation` grant are not independently checked, because impersonating
   somebody is precisely the request to evaluate as them. The admin-side gate
   stays where it is, on the `Impersonate` call itself.

That asymmetry is the one place "delegation can only narrow" doesn't hold, so it
gets a comment at the call site as well as this paragraph.

Subject kinds map onto warden v1.6.0 like this. `KindUser` becomes
`warden.SubjectUser`. `KindAgent`, `KindWorkload` and `KindService` all become
`warden.SubjectServiceAcct`, with the finer kind carried in
`Subject.Attributes["principal_kind"]`.

Collapsing three kinds onto one warden subject keeps role assignments in a
single namespace, so an operator grants `ci-deployer` once and not once per
kind. You do not lose the distinction. `Attributes` also carries `on_behalf_of`,
`actor_kind`, `delegation_id` and `ephemeral`, which is enough for you to write
"deny when `actor_kind` is agent and the resource is billing" as an ABAC policy
without warden needing a new feature to support it.

## Resolving a principal on a request

Two entry points converge today and both stay. Bearer and cookie sessions go
through `trySessionAuth` into `setSessionContext`, API keys go through
`tryStrategyAuth`, and both of them now derive `Subject` and `Actors` before
calling a single helper that puts the resolved principal on the context where
everything downstream can find it.

`*user.User` keeps being set whenever the subject is a user. `UserFrom(ctx)` and
every handler reading it are untouched.

```go
// middleware/context.go, mirroring the existing With/From convention.
func WithPrincipal(ctx context.Context, p *principal.Principal) context.Context
func PrincipalFrom(ctx context.Context) (*principal.Principal, bool)
func WithActors(ctx context.Context, c principal.Chain) context.Context
func ActorsFrom(ctx context.Context) (principal.Chain, bool)
```

`WithImpersonator` stays exactly as it is, fed from `Actors` now that the struct
field is gone. Middleware takes a
`PrincipalResolver func(principal.Ref) (*principal.Principal, error)` alongside
the existing `UserResolver`, so it still doesn't import the engine.

## `plugin.Engine`

Three methods go on the core interface rather than behind an optional capability
interface. Plugins are supposed to reason about non-human callers, and burying
that behind a type assertion makes it undiscoverable.

```go
// ── Principals ──

// ResolvePrincipal resolves any caller, human or otherwise, by ref.
ResolvePrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error)

// PrincipalStore returns the principal and delegation store.
PrincipalStore() principal.Store

// Can is the chain-aware authorization check. An empty chain is an ordinary
// single-subject check.
Can(ctx context.Context, subject principal.Ref, actors principal.Chain,
    action, resource string) (bool, error)
```

Only two types implement `plugin.Engine`, the engine itself and one mock in the
apikey plugin tests, so widening it costs one test double in-repo. Out-of-tree
implementers will break, and that goes in the changelog.

`principal.FromContext(ctx)` also lives in the `principal` package, reading the
same context key, so a plugin can get at the caller without importing
`middleware`.

## Making machine traffic visible to the risk plugins

A new hook pair on the registry, following the `beforeSignIn` entry pattern that
`plugin/registry.go` already uses:

```go
type BeforePrincipalAuth interface {
    OnBeforePrincipalAuth(ctx context.Context, a *principal.AuthAttempt) error
}
type AfterPrincipalAuth interface {
    OnAfterPrincipalAuth(ctx context.Context, a *principal.AuthAttempt, s *session.Session) error
}
```

```go
type AuthAttempt struct {
    Subject        Ref
    Actors         Chain
    AppID          id.AppID
    EnvID          id.EnvironmentID
    OrgID          id.OrgID
    CredentialKind string // "api_key" | "token_exchange" | "jwt"
    CredentialID   string // akey_… | adel_…
    IPAddress      string
    UserAgent      string
    Ephemeral      bool
    At             time.Time
}
```

`EmitBeforePrincipalAuth` returns the first error and denies, matching
`EmitBeforeSignIn`. It fires wherever a credential becomes a session: the apikey
strategy, token exchange, and JIT child minting.

The verdict is cached in the engine, keyed by credential ID and client IP, with
a settings-driven TTL defaulting to five minutes, so a chatty agent pays for one
geo and reputation lookup per credential per source IP and not one per call.
That is the whole reason you can afford this on your highest-volume traffic.

It has to be a typed plugin hook and not a `hook.Bus` event. `Bus.Emit` logs
handler errors and returns nothing, so it structurally cannot deny, while
riskengine denies by returning an error. Route machine traffic through the bus
and you would hand the risk plugins visibility with no ability to act on it,
which is worse than the gap you started with for the simple reason that it
looks fixed.

A `hook.ActionPrincipalAuth` bus event fires after the decision, so Chronicle
and the relay pick up machine auth in the audit trail without subscribing to a
typed hook.

What each of the six plugins actually needs:

- riskengine implements `BeforePrincipalAuth` and builds the same
  `RiskRequest{IPAddress, UserAgent, AppID}` it builds today. Its five
  contributors need no changes, because that struct never carried a user.
- ipreputation, geofence and vpndetect are contributors only, reached through
  riskengine. Zero changes.
- impossibletravel and anomaly key their in-memory history by `u.ID.String()`.
  Their map key becomes `Ref.String()`, and `LoginLocation.UserID` and
  `LoginPattern.UserID` become a `principal.Ref`. Contained to those two files.

## Getting a delegated credential

One core endpoint, RFC 8693 shaped, under the existing base path:

```
POST {basePath}/token/exchange
  subject_token          the agent's own credential (ask_… API key)
  requested_subject      ausr_…, who it wants to act for
  scope                  requested subset
  → session token whose Subject is the user and Actors is [the agent]
```

The engine finds an active, unexpired delegation grant matching actor, subject
and org, intersects the requested scope with both the grant's scopes and the
actor's own scopes, fires `BeforePrincipalAuth`, and mints a real persisted
session whose TTL is bounded by the grant's expiry. No grant, or a revoked one,
is a 403, because the endpoint has no path that creates authority and can only
exercise a grant that already exists.

Ephemeral children get a sibling endpoint, `POST {basePath}/principals/{id}/children`,
authenticated as the parent. It returns a short-lived principal and a
credential, with the TTL capped by the parent's own expiry and the scopes
validated as a subset of the parent's.

Agents keep authenticating with API keys. `apikey.APIKey` already has
`ExpiresAt`, `Revoked` and `ServiceAccountID`, so there's no new credential
primitive here. What changes is that the key now names a principal of a specific
kind, and that using it fires the risk hooks.

## Schema

`authsome_service_accounts` keeps its name. Renaming the mongo collection buys
nothing and costs a data migration. It becomes the storage for every non-human
principal, with new columns:

| Column | Type | Purpose |
|---|---|---|
| `kind` | TEXT NOT NULL DEFAULT `'service_account'` | `agent`, `workload` or `service_account` |
| `owner_user_id` | TEXT NOT NULL DEFAULT `''` | the human answerable for it |
| `parent_id` | TEXT NOT NULL DEFAULT `''` | registered parent, set on ephemeral children |
| `expires_at` | TIMESTAMPTZ NULL | hard TTL; NULL means durable |
| `org_id`, `env_id` | TEXT NOT NULL DEFAULT `''` | tenant binding |

`active` and `scopes` already exist and keep their current meaning.

`authsome_delegations` is new, in all four backends:

```
id, app_id, org_id
actor_kind,   actor_id        -- who may act
subject_kind, subject_id      -- on whose behalf
grant_kind                    -- 'delegation' | 'impersonation'
scopes, granted_by
expires_at, revoked_at, created_at, updated_at

UNIQUE (app_id, actor_kind, actor_id, subject_kind, subject_id, grant_kind)
       WHERE revoked_at IS NULL
INDEX  (subject_kind, subject_id)   -- what is allowed to act for me
INDEX  (actor_kind, actor_id)       -- what may I act for
```

`authsome_sessions` gains `actors` and `delegation_id`. That's JSONB on
postgres, TEXT holding JSON on sqlite, and an array on mongo.

The existing session CHECK constraint hardcodes `principal_kind IN ('', 'user')`
or `'service_account'`, so it needs relaxing to admit `agent` and `workload`.
The mongo equivalent is a `collMod` schema refresh, and
`store/mongo/migrations.go` already has a worked precedent for that.

The RFC 8693 orientation pays off here too. A delegated session still carries
`principal_kind = 'user'` and a real `user_id`, so the constraint keeps holding
for the common case and only widens for standalone agent and workload sessions.
Had the agent been the subject, every delegated session would violate the
constraint and the migration would be rewriting live rows.

## What happens to `impersonated_by`

The Go struct field goes away. `Session.ImpersonatedBy()` becomes a method
deriving from `Actors`, and the 31 in-repo references move over to it. Nothing
in the SDKs or the generated spec touches it, so the blast radius is the
repository.

The database column stays for one release, written from the chain by the model
layer on the way down. It is the one field in this change that an operator
plausibly queries directly in SQL during a security review, and a migration that
backfills `actors` and drops the source column in a single step leaves you with
no cheap rollback if the backfill turns out to be wrong. So this change
backfills `actors` from `impersonated_by` and leaves the column standing as a
derived projection. A follow-up migration drops it.

The JSON tag `impersonated_by` on `session.Session` is preserved through a
marshaling shim, and `impersonatedBy` in the extension contract is unchanged.
The wire format does not move.

## Per-backend work

| Backend | Work |
|---|---|
| postgres | Create `authsome_service_accounts` from scratch. Create `authsome_delegations`. Session columns, relax CHECK. Implement all five stubbed service-account methods plus the delegation store. |
| sqlite | Same as postgres, adapted for its timestamp handling. `store/sqlite/migrations_timestamps.go` is the precedent. |
| mongo | Add fields to `serviceAccountModel`. New `delegations` collection with indexes. `collMod` for the session schema. The store methods are already real, so extend rather than build. |
| memory | Two maps and a delegation index. No migration. |

The postgres and sqlite service-account implementations are unplanned scope that
surfaced during design, and they're a prerequisite either way. They stay in this
change.

## Testing

Most of the value lands in `store/storetest/storetest.go`, because each backend
already has a `conformance_test.go` running it. New cases cover all four
automatically: principal round-trip per kind, an ephemeral child with
`parent_id` and expiry, delegation create and lookup and revoke, actor-chain
round-trip, and the expiry filter excluding a lapsed principal.

Beyond that:

Warden intersection gets a table-driven test over the four combinations of agent
allowed and user allowed, plus impersonation evaluating as the subject alone.

Risk exposure gets three. An API-key request fires `BeforePrincipalAuth`, a
denying plugin produces a 401, and a second request inside the TTL hits the
cache without re-running contributors.

Compatibility gets four. The existing impersonation e2e passes unchanged,
`ImpersonatedBy()` returns the admin for a session from `Engine.Impersonate`,
and a legacy row with an empty `principal_kind` resolves to a user principal.
The fourth matters more than it looks: `roleStampingStore.shouldStamp` has to
keep skipping non-human sessions, because that branch exists precisely because a
service-account session carries a zero `UserID` with nothing to look up, and
breaking it starts erroring session creation for every machine caller you
have.

## Deliberately out of scope

No agent-specific metadata. Model name, vendor, tool manifest belong to whoever
builds an MCP integration on top, not in the auth core.

No new credential primitive. `apikey` already has expiry and revocation.

No dashboard UI in this change.
