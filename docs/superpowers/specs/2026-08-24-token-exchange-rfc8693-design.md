# Token exchange (RFC 8693) design

Status: draft
Date: 2026-08-24

## What this gives you

Every downstream hop in your system today reuses the original token at full
privilege. A user signs in, your API calls a second service, that service calls
a third, and the same credential travels the whole way carrying everything the
user can do, which means a compromise anywhere along that path is a compromise
of the original session with nothing narrowed. There's no way to trade a broad
credential for a narrow one. Grep for `8693` and you get nothing back.

Token exchange gives you one. A client presents a token it already holds, asks
for less than that token carries, and gets back a short-lived token scoped to
exactly what it asked for, while the original stays where it was.

You also get an answer to a question the current design cannot answer at all,
which is who was actually behind a given call. When agent A does something for
user U, the resulting token records both parties and says which of two things
happened: delegation, where A acted for U and both names survive into the token,
or impersonation, where A became U outright and the token looks like U. Those
are different security events, and you want to be able to tell them apart six
months later when somebody asks. The token is only half of what this feature
produces. The audit record is the other half, and if you are choosing what to
cut, cut the token.

## Decisions

Four questions were settled during brainstorming, and most of the rest of this
document follows from them.

**Scopes become a stored property of a session.** You cannot enforce that a new
token is a subset of an old one without knowing what the old one held, and
`session.Session` carries `Roles` and `PrincipalKind` but nothing about scopes,
so `issueTokens` in `plugins/oauth2provider/plugin.go` writes them into the JWT
claims and the response body and then drops them. On an opaque token they are
gone for good. Scopes get a column, the same way `Roles` did in `ba629bd`.

**`ImpersonatedBy` is absorbed into the actor chain.** The existing field models
admin-acting-as-user, the chain generalises it, and running two mechanisms side
by side means every future auditor has to know about both. So we migrate. This
is the most invasive part of the work and the section below is honest about what
it costs.

**Every exchange needs an explicit policy row.** Holding a subject token is not
authorization to trade it. A new `authsome_oauth2_exchange_policies` table
declares which client may exchange for which principals, in which mode, up to
which scopes, and anything without a matching row is refused. RFC 8693 leaves
this question to the deployment, and the deployment should not answer it with
"whoever holds the token wins".

**An exchanged token keeps the subject's roles.** This one is a compromise.
Scopes and roles are separate enforcement paths here, so a scope-narrowed token
that keeps every role still passes every role-gated route its subject passed.
See "Known limitations" before you rely on the narrowing.

## Calling it

```
POST /v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token=<an authsome session token>
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&scope=invoices:read
&client_id=svc-billing
&client_secret=...
```

```json
{
  "access_token": "...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 300,
  "scope": "invoices:read"
}
```

`issued_token_type` is required by RFC 8693 section 2.2.1. It's `omitempty` on
`TokenResponse`, so the three grants you already have keep their current
response shape.

Add `actor_token` and `actor_token_type` when a second party is doing the
acting:

```
&actor_token=<the agent's own token>
&actor_token_type=urn:ietf:params:oauth:token-type:access_token
```

Two subject token types are supported, and deliberately only two:

| Type | Meaning |
|---|---|
| `urn:ietf:params:oauth:token-type:access_token` | an authsome session token, opaque or JWT |
| `urn:x-authsome:params:oauth:token-type:session` | the same thing, named explicitly |

Both resolve through the same code path, because in this codebase they are the
same object: an OAuth access token is a session row. The `refresh_token`,
`id_token` and SAML types return `unsupported_token_type`, which is a better
answer than a half-built implementation of each.

`requested_token_type` accepts `access_token` or nothing at all. You can send
`audience` and `resource` and they will be recorded in the audit metadata, but
nothing enforces them yet. That's the seam RFC 8707 lands in.

## Changes to session.Session

Two fields arrive and one leaves.

```go
// Scopes holds the OAuth scopes this session was issued with. Stamped at
// issuance, with the same trade as Roles: authoritative for what this token
// may do, stale with respect to anything granted afterwards.
Scopes []string `json:"scopes,omitempty"`

// Actors is the delegation chain (the RFC 8693 `act` claim). Actors[0] is
// the immediate actor and later elements are the parties further back.
Actors []Actor `json:"actors,omitempty"`
```

```go
// Actor is one party in a delegation chain.
type Actor struct {
    Subject string    `json:"sub"`  // user, service account or oauth client id
    Kind    string    `json:"kind"` // "user" | "service_account" | "oauth_client"
    Mode    string    `json:"mode"` // "delegation" | "impersonation"
    At      time.Time `json:"at"`
}
```

`ImpersonatedBy` goes away. `Engine.Impersonate` at `service.go:2701` writes a
one-element chain in its place, and `StopImpersonation` looks for a chain whose
outermost entry is an impersonation.

### Why Mode lives on the session and not in the token

RFC 8693 encodes the delegation and impersonation distinction by absence.
Delegation emits an `act` claim naming both parties, impersonation emits no
`act` at all so the token simply looks like the subject, and that asymmetry is
the wire format we follow exactly.

An impersonation token carrying no trace of its actor cannot feed an audit
trail, though, and the audit trail is most of why you would build this. So the
work splits: the token follows the RFC and the session row records everything
that happened. `Mode` is how the row stays complete at the moments the wire
format is deliberately silent.

### Migration, in two releases

`add_session_scopes_and_actors` follows the shape of `add_session_roles` at
`store/postgres/migrations.go:1180`. You get JSONB on postgres, TEXT on sqlite,
and nothing at all on mongo, which has no migration group. It adds both columns
and backfills:

```sql
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS actors JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE authsome_sessions
   SET actors = jsonb_build_array(jsonb_build_object(
           'sub',  impersonated_by,
           'kind', 'user',
           'mode', 'impersonation',
           'at',   created_at))
 WHERE impersonated_by <> '' AND actors = '[]'::jsonb;
```

The `impersonated_by` column is **not** dropped here. Dropping it waits for a
later release, after the store models have shipped a read fallback, because if
you add, backfill and drop in a single migration then a rolling deploy leaves
old binaries writing to a column that no longer exists. While both are present
the models read `Actors` first and fall back to `impersonated_by`.

### Coordination with the other session specs

Three drafts in this directory widen `session.Session`, and whoever lands second
pays the merge cost:

- DPoP adds `DPoPJKT`.
- agentauth adds `AgentID`, `GrantID` and `PrincipalKind = "agent"`.
- This one adds `Scopes` and `Actors`, and removes `ImpersonatedBy`.

Only the removal actually conflicts. The agentauth draft cites `ImpersonatedBy`
as its precedent for how an agent rides along on a session, so that paragraph
wants rewriting against `Actors` once this lands. None of agentauth's design
breaks, only its wording.

## The policy table

```sql
CREATE TABLE authsome_oauth2_exchange_policies (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL REFERENCES authsome_apps(id),
    client_id       TEXT NOT NULL,
    subject_kind    TEXT NOT NULL,              -- user | service_account | oauth_client | any
    subject_match   TEXT NOT NULL DEFAULT '*',  -- a principal id, or '*'
    modes           JSONB NOT NULL DEFAULT '["delegation"]',
    max_scopes      JSONB NOT NULL DEFAULT '[]',
    max_ttl_seconds INT  NOT NULL DEFAULT 0,    -- 0 inherits from SessionConfig
    max_chain_depth INT  NOT NULL DEFAULT 1,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (app_id, client_id, subject_kind, subject_match)
);
```

`client_id` is always the authenticated requesting client. When an `actor_token`
comes along its principal becomes the head of the chain, but policy still gates
on the client, because the client is the party that authenticated and the
actor's identity is proven by resolving its token rather than asserted in a
parameter. That's what makes gating on the client alone safe.

Matching is most specific wins, so an exact `subject_match` beats `*` and a
concrete `subject_kind` beats `any`. If no enabled row matches, the exchange is
refused.

`max_chain_depth` earns its place because every hop appends an actor. Leave it
uncapped and chains grow without bound, with every session row and every audit
record carrying the weight. The default of 1 permits a single hop, so a subject
that has never been exchanged can be exchanged once and the result cannot be
exchanged again until somebody raises the number.

`max_ttl_seconds` of 0 means this row does not constrain the TTL, so it drops
out of the minimum below instead of clamping it to zero. Read the same way as
the `0 inherits` convention `store.AppSessionConfig` already uses.

Admin CRUD lands at `/v1/admin/oauth/exchange-policies` behind
`plugin.AdminGuard(p.engine, "manage", "oauth2_exchange_policy")`, which is a
different permission from `oauth2_client` on purpose. Anyone who can author
exchange policy can authorise impersonation, and that is a larger power than
registering a client.

## The scope ceiling

This part took the most thought, and it starts with a problem that the obvious
design walks straight into.

A password sign-in produces a session with no scopes on it. If you decide empty
means unrestricted, every legacy session in your database becomes a universal
ceiling and the subset rule turns decorative. If you decide empty means nothing,
then downgrading an existing session into a scoped service token never works,
because the ceiling is empty on the very first hop. Both readings fail a stated
requirement, so the answer has to come from somewhere other than the subject.

The policy table is where it comes from. Give each row a `max_scopes` and the
ceiling collapses into one expression:

```
ceiling = client.Scopes ∩ policy.MaxScopes ∩ (subject.Scopes if non-empty else ⊤)
```

then `requested ⊆ ceiling`, or `invalid_scope`.

A scopeless session now gets a ceiling that an administrator wrote down, and a
scoped subject token can never widen past what it holds. Both cases fall out of
the same line of code.

That top element is only safe because it can never be the outermost bound.
`client.Scopes` and `policy.MaxScopes` are both finite and both mandatory, so
the intersection stays finite and stays administrator-authored no matter what
the subject looks like. A ceiling rule with a top element in it usually deserves
suspicion, and this one survives the suspicion.

One deliberate deviation: `scope` is required. The RFC makes it optional and
lets the server choose a default, but the entire point of the feature is asking
for less than you hold, and a caller who omits `scope` is asking the server to
guess on their behalf. An empty `scope` returns `invalid_scope`.

## The flow

1. Authenticate the client. Confidential only, bcrypt against `ClientSecret`,
   the same as `client_credentials`. Public clients cannot exchange, since they
   cannot keep a secret and this is a privilege operation.
2. `clientSupportsGrant(client, tokenExchangeGrantType)`, or `unauthorized_client`.
3. Resolve `subject_token` through `Engine.ResolveSessionByToken`. Invalid or
   expired gives you `invalid_grant`.
4. The subject's `AppID` must equal the client's `AppID`. Skip this and a client
   in one app can launder a session out of another.
5. Resolve `actor_token` if present, with the same app check.
6. Look up policy on `(app, client, subject kind, subject id)`. Missing or
   disabled means refuse.
7. Apply the ceiling.
8. Check `len(subject.Actors) + 1 <= policy.MaxChainDepth`.
9. Compute the TTL, mint the session, write the audit record.

Steps 4 and 8 appear nowhere in RFC 8693, which assumes a single authorization
server in one trust domain issuing stateless tokens, while this codebase is
multi-tenant by `AppID` and persists its chains. Specs written against a simpler
deployment model tend to leave exactly those two gaps for you to find.

### Requesting impersonation

The RFC gives you no parameter for this, since it encodes the distinction by
absence. Follow that literally and the unmarked default becomes the more
privileged of the two modes, which is a bad default to ship to anybody.

Because the requesting client here is always authenticated and always a genuine
actor, we always record it. Mode comes from a namespaced `authsome_act_mode`
parameter that defaults to `delegation` and is checked against the policy's
`modes` list. Section 2.1 explicitly allows additional parameters.

### TTL

```
ttl = min(policy.MaxTTL, sessCfg.TokenExchangeTTL, time.Until(subject.ExpiresAt))
```

`account.SessionConfig` gains `TokenExchangeTTL`, defaulting to five minutes,
and it rides the three layers that `sessionConfigForApp` at `service.go:783`
already walks: the engine default, then `store.AppSessionConfig.ApplyTo`, then
environment settings. Your admin surface is one more field on the request struct
at `api/requests.go:599`. No new configuration mechanism appears anywhere.

The third clamp matters more than it looks. A token that outlives the credential
it was minted from is an escalation in the time dimension, and it slips past
review easily because each individual TTL in the expression looks perfectly
short on its own.

Exchanged tokens get no refresh token. `RefreshTokenTTL` is zero and you
re-exchange when you need another, which keeps the subject as the only durable
credential in the picture.

### What the new session carries

`AppID` comes from the client. `EnvID` is inherited from the subject and not
from the app default, so an exchanged token stays in the environment it came
out of. `UserID`, `PrincipalKind` and `ServiceAccountID` are copied from the
subject, because both modes keep the subject as `sub`. `Scopes` holds the
granted set, `Actors` holds the new actor prepended to the subject's chain, and
`Roles` is copied straight across with the caveat in "Known limitations".

On JWT-format apps, `tokenformat.TokenClaims` gains an `Act *ActClaim` and
`customClaims` at `tokenformat/jwt.go:69` serialises it as `act`, nested the way
the RFC describes, emitted for delegation and omitted for impersonation. That
struct has no extension map today, so this is a real edit to a third core
package and not a drive-by.

## Audit

`plugin.Engine` gains `SecurityEvents() securityevent.Store`. The concrete
engine already has that method at `engine.go:831`, so the interface addition is
one line and is already satisfied. The plugin simply cannot see it today.

The plugin writes directly instead of going through the hook bus. The bridge at
`engine.go:526` builds its `securityevent.Event` from only `Action`, `Outcome`,
`Metadata` and `CreatedAt`, never setting `AppID`, and since
`securityevent.Query` filters on `AppID`, everything recorded down that path is
written but unqueryable. That's a pre-existing bug affecting every plugin's
security events, and it deserves its own diff rather than being smuggled into
this one.

One `oauth2.token_exchange` event per attempt, success or failure, with `AppID`
and `UserID` populated properly and metadata carrying `client_id`,
`subject_kind`, `subject_principal_id`, `subject_session_id`, `actor_kind`,
`actor_principal_id`, `act_mode`, `requested_scopes`, `granted_scopes`,
`chain_depth`, `policy_id`, `issued_session_id` and `expires_in`.

Failures carry a `denial_reason` drawn from a closed set: `no_policy`,
`mode_not_allowed`, `scope_escalation`, `chain_too_deep`, `cross_app`,
`invalid_subject`, `unsupported_token_type`. Those records are worth more to you
than the successful ones, because `scope_escalation` and `cross_app` are attack
signatures and not user error, and keeping the vocabulary closed is what lets
you alert on them instead of grepping for them after the fact.

## What gets touched

This is bigger than the plugin, mostly on account of the `ImpersonatedBy`
unification.

Core: `session/session.go`, `account/service.go`, `service.go` for both
impersonation functions, `middleware/auth.go` at lines 222 and 630,
`authprovider/session.go:188`, `extension/contract/handlers_sessions.go`, the
three `store/*/models.go`, `store/postgres/migrations.go`,
`store/sqlite/migrations.go`, `tokenformat/format.go`, `tokenformat/jwt.go` and
`plugin/plugin.go`.

Plugin: a new `token_exchange.go` holding the handler along with the ceiling,
chain and policy logic, plus `plugin.go` for dispatch, request and response
fields, discovery and admin routes, then `models.go`, `store.go`,
`store_memory.go`, `store_postgres.go`, `store_sqlite.go`, `store_mongo.go` and
`migrations.go`.

Add the grant to `GrantTypesSupported` in `handleDiscovery` at
`plugins/oauth2provider/plugin.go:746` while you are in there. It's one line
and it's the sort of thing nobody notices until a conformance test does.

## Sequencing

Four commits, each independently green. The first one carries the risk and
shouldn't be tangled up with feature work.

1. Session schema. `Scopes` and `Actors`, the migration and backfill, and
   `ImpersonatedBy` removed across every consumer, with no new behaviour at all.
2. `SecurityEvents()` on the Engine interface, `TokenExchangeTTL` threaded
   through the config layers, `Act` on `TokenClaims`.
3. The policy table, store methods across four drivers, admin CRUD.
4. The grant itself, discovery, tests.

## Testing

`token_exchange_test.go` is table-driven against the memory store, following
`authcode_test.go`.

These carry the security claims and get written first, failing:

- Request a scope the subject does not hold, expect `invalid_scope`. This is the
  subset property, and if it ever passes the feature is broken.
- Request a scope the subject does hold but the policy's `max_scopes` excludes,
  expect refusal. Both bounds have to bite independently of each other.
- Exchange a subject from a different app, expect a `cross_app` refusal.
- Exchange with no policy row at all, expect refusal, because deny by default is
  a claim and claims get tested.
- Ask for impersonation against a delegation-only policy, expect refusal.
- Build a chain to `max_chain_depth + 1`, expect refusal.
- Exchange a subject with four minutes left against a five-minute config TTL and
  expect four minutes back.
- A delegation exchange on a JWT app emits `act` and the impersonation
  equivalent emits none, while both write a session row with a complete chain.
- Every failure above writes a security event with the right `denial_reason`.

Store parity: session round-trip for `Scopes` and `Actors` across pg, sqlite,
mongo and memory, plus a backfill test proving an `impersonated_by` row lands as
a one-element impersonation chain.

## Known limitations

Exchanged tokens keep their subject's roles. Scope narrowing is real for
anything that checks scopes and completely invisible to anything that checks
roles, so a token you think of as downgraded still passes every role-gated route
its subject passed, which is a surprising result if you have not read this far.
Narrowing both axes needs a scope-to-role mapping that doesn't exist in this
codebase yet. The follow-up is a `granted_roles` column on the policy row, and
until that ships this limitation belongs in the user-facing docs and not only
here.

Stamped scopes go stale the same way stamped roles do. Widening a session's
scopes after issuance does not reach an existing token and narrowing them does
not either, so a revocation still needs the session revoked to take effect. The
comment on `Session.Roles` already documents that trade, and this extends it.

## Out of scope

- RFC 8707 resource indicators. You can send `audience` and `resource` and they
  get parsed and audited. Binding on them is separate work.
- DPoP, which composes rather than conflicts. The composition does have one rule
  worth writing down now: an exchange must never turn a bound token into an
  unbound one, so if the subject carries a `DPoPJKT` then the exchanged token
  has to be bound too. That is the same laundering failure the DPoP draft
  already calls out for refresh, and whichever of the two lands second owns the
  test for it.
- The `refresh_token`, `id_token` and SAML subject token types.
- Exchanging API keys, which are not sessions and never reach this endpoint.
- Externally-signed tokens, which belong to workload identity federation. See
  the boundary note below.

## Boundary with workload identity federation

Added 2026-08-24 by the session designing
`2026-08-24-workload-identity-federation-design.md`, so this spec carries a
record of it.

That plugin also does an RFC 8693 shaped exchange, on its own endpoint at
`/v1/workload/token`, and the two are deliberately separate. The line between
them is the subject token. A subject token that is an authsome session, sent by
a registered client narrowing what it already holds, is this endpoint and is
governed by a policy row. A subject token signed by GitHub or Google or an EKS
OIDC provider, sent by a caller holding no secret at all, is that endpoint and
is governed by an issuer trust config and a claim rule. Neither endpoint should
grow the other's case, because doing so puts two disjoint authorization models
behind one URL and makes client authentication conditional on the token type.

A workload that wants a narrower credential exchanges there first and narrows
here second, and both hops land in the audit trail.

One merge to watch. That spec's prerequisite,
`2026-08-24-non-human-principal-enforcement-design.md`, adds `pk` and
`ServiceAccountID` fields to `tokenformat.TokenClaims`, which is the same struct
this spec adds `Act *ActClaim` to. Nothing conflicts and every field is
`omitempty`. Whoever implements second should add both sets in one pass.
