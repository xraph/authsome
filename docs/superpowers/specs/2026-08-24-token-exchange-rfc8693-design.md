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

**The actor chain comes from the non-human principals design.** An earlier
draft of this spec defined its own `session.Actor` type and folded
`ImpersonatedBy` into it. That duplicated
`2026-08-24-non-human-principals-design.md`, which models the same thing more
thoroughly as `principal.Chain` plus a per-session `ActorGrant`. This spec now
depends on that one rather than competing with it, and the sections below say
which side owns what.

**The authority for an exchange is a delegation grant.** Holding a subject
token is not permission to trade it. A `principal.Delegation` names an actor, a
subject, a grant kind and a scope filter, and an exchange with no live matching
grant is refused. An earlier draft built a separate
`authsome_oauth2_exchange_policies` table for this. The delegation record is
the same idea with a lifecycle, a revocation surface and a listing API already
attached, so the table is gone.

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

One field, and it is the only core schema change this design owns.

```go
// Scopes holds the OAuth scopes this session was issued with. Stamped at
// issuance, with the same trade as Roles: authoritative for what this token
// may do, stale with respect to anything granted afterwards.
Scopes []string `json:"scopes,omitempty"`
```

`Actors`, `ActorGrant` and `DelegationID` all arrive with the non-human
principals work, along with `ImpersonatedBy` becoming a method and
`PrincipalKind` being retyped to `principal.Kind`. Nothing here touches them.

The `scopes` column follows `add_session_roles` at
`store/postgres/migrations.go:1180`: JSONB on postgres, TEXT on sqlite, a
native array on mongo, which has no migration group.

### Why this field has to exist

`Engine.ExchangeToken` in the non-human principals plan computes the granted
scope set and then writes `_ = scopes`, with a note to assign it "to whichever
field the session carries scopes on once you confirm it". There is no such
field today, so the value is computed and dropped.

That matters beyond tidiness. Without stored scopes there is no subject-side
ceiling on the second hop, so a token narrowed to one scope could be exchanged
back up to everything its client is registered for. The column closes that.

### Delegation, impersonation, and where the mode lives

RFC 8693 encodes the distinction by absence. Delegation emits an `act` claim
naming both parties, impersonation emits no `act` at all so the token simply
looks like the subject, and that asymmetry is the wire format we follow
exactly.

An impersonation token carrying no trace of its actor cannot feed an audit
trail, though, and the audit trail is most of why you would build this. The
non-human principals design already solves it: the mode is a per-session
`ActorGrant`, and the chain is on the row whichever mode applied. The token
follows the RFC and the row records what happened.

One consequence worth stating. Because `ActorGrant` is one value per session
rather than one per hop, a chain cannot record that hop one was a delegation
and hop two an impersonation. Each hop mints its own session, so the mode of
each is recorded on the session that hop produced and in that hop's audit
event. What you lose is reading a mixed-mode history off a single row, and the
audit trail is the right place to reconstruct it anyway.

## The scope ceiling

This part took the most thought, and it starts with a problem the obvious
design walks straight into.

A password sign-in produces a session with no scopes on it. If you decide empty
means unrestricted, every legacy session in your database becomes a universal
ceiling and the subset rule turns decorative. If you decide empty means nothing,
then downgrading an existing session into a scoped service token never works,
because the ceiling is empty on the very first hop. Both readings fail a stated
requirement.

The answer is that three bounds apply and no single one of them has to carry
the whole job:

```
granted ⊆ client.Scopes ∩ delegation.Scopes ∩ (subject.Scopes if non-empty else ⊤)
```

The OAuth plugin owns the first and third, because they are cheap and local and
an obviously bad request should never reach a grant lookup. The engine owns the
second, in `intersectScopes`, because the delegation record is its to read.

A scopeless session gets a ceiling from the client registration and the
delegation grant, both authored by an administrator. A scoped subject token can
never widen past what it holds. The top element is safe only because it can
never be the outermost bound: the other two are finite and both mandatory.

One deliberate deviation: `scope` is required. The RFC makes it optional and
lets the server choose a default, but the entire point of the feature is asking
for less than you hold, and a caller who omits `scope` is asking the server to
guess on their behalf. An empty `scope` returns `invalid_scope`.

## The flow

1. Authenticate the client. Confidential only, bcrypt against `ClientSecret`,
   the same as `client_credentials`. Public clients cannot exchange, since they
   cannot keep a secret and this is a privilege operation.
2. `clientSupportsGrant(client, tokenExchangeGrantType)`, or
   `unauthorized_client`.
3. Resolve the client to its principal. `Engine.ExchangeToken` takes a
   `principal.Ref` actor and `principal.Kind` has no `oauth_client` member, so
   `OAuth2Client` carries a `PrincipalID` and a client without one is refused.
4. Resolve `subject_token`. Invalid or expired gives you `invalid_grant`.
5. The subject's `AppID` must equal the client's `AppID`. Skip this and a client
   in one app can launder a session out of another.
6. Resolve `actor_token` if present, with the same app check. Its principal
   replaces the client as the acting party, and it is proven by resolution
   rather than asserted in a parameter.
7. Apply the client bound and the subject bound.
8. Hand off to `Engine.ExchangeToken`, which finds the delegation grant, applies
   its scope filter, bounds the TTL and builds the chain.
9. Write the audit record.

Steps 3 and 5 appear nowhere in RFC 8693. The RFC assumes a single
authorization server in one trust domain issuing stateless tokens, while this
codebase is multi-tenant by `AppID` and its principals are first-class rows.
Specs written against a simpler deployment model tend to leave exactly those
gaps for you to find.

### Requesting impersonation

You cannot. The RFC gives no parameter for it, and an earlier draft of this
spec invented a namespaced one defaulting to delegation. That is gone.

The mode is a property of the delegation grant, authored through
`Engine.GrantDelegation` by somebody with the authority to author it. A caller
presenting a token cannot ask to be upgraded from delegation to impersonation,
which is a better place for that decision to live than a request body.

### TTL

`account.SessionConfig` gains `TokenExchangeTTL`, defaulting to five minutes,
and it rides the three layers `sessionConfigForApp` at `service.go:783` already
walks: the engine default, then `store.AppSessionConfig.ApplyTo`, then
environment settings. Your admin surface is one more field on the request struct
at `api/requests.go:599`.

The exchanged session must also never outlive the grant that authorised it or
the subject it came from. A token that survives the credential it was minted
from is an escalation in the time dimension, and it slips past review easily
because each individual bound looks short on its own. The grant bound is the
engine's and already has a test. The subject bound belongs beside it.

Exchanged tokens get no refresh token. You re-exchange when you need another,
which keeps the subject as the only durable credential in the picture.

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

Much smaller than the earlier draft, because the invasive half moved to its
proper owner.

Core: `session/session.go` for the one field, the three `store/*/models.go`,
`store/postgres/migrations.go` and `store/sqlite/migrations.go`,
`account/service.go` and the two config override layers, `api/requests.go`,
`tokenformat/format.go` and `tokenformat/jwt.go` for the `act` claim,
`plugin/plugin.go` for `SecurityEvents()`, and `engine_token_exchange.go` to
assign the scopes it already computes and to emit `act`.

Plugin: a new `token_exchange.go` holding the handler and the two local scope
bounds, plus `plugin.go` for dispatch, request and response fields and the
discovery advertisement, `models.go` and `store_models.go` for the client
principal link, the four store files, and `migrations.go`.

## Sequencing

Six commits, each independently green, and all of them after the non-human
principals work has landed through its delegation and exchange task.

1. The `scopes` column, stamped at issuance, wired into the engine's exchange.
2. `TokenExchangeTTL` through the config layers.
3. The `act` claim, defined and emitted on delegated tokens.
4. `SecurityEvents()` on the Engine interface.
5. The grant itself, the client principal link, dispatch and discovery.
6. The audit events.

## Testing

The protocol layer carries these claims, and each gets a test that asserts the
engine is never reached when the refusal is local:

- A scope the subject does not hold is refused.
- A scope the client is not registered for is refused, even when the subject
  holds it. Both local bounds have to bite independently.
- An empty `scope` is refused.
- A subject from a different app is refused.
- A public client is refused.
- A client with no linked principal is refused, with a message that says so.
- An unsupported `subject_token_type` returns `unsupported_token_type`.
- An engine refusal, meaning no live grant, surfaces as `invalid_grant` and
  never leaks a token.
- The response carries `issued_token_type` and no refresh token.
- Discovery advertises the grant.
- Every refusal above writes a security event with the right `denial_reason`.

The grant filter, the TTL bound and the chain are the engine's, and the
non-human principals plan already tests them. Do not re-test them here.

## Coordination with the other designs

Five drafts in this directory now touch the same few files, and whoever lands
second pays the merge cost. Worth knowing before you start.

`session.Session` is widened by non-human principals (`Actors`, `ActorGrant`,
`DelegationID`, and `PrincipalKind` retyped), by DPoP (`DPoPJKT`), by agentauth
(`AgentID`, `GrantID`) and by this one (`Scopes`). Only the retype is a
breaking change to existing readers, and it belongs to non-human principals.

`tokenformat.TokenClaims` is widened by this design (`Act`) and by non-human
principal enforcement (`pk`, `ServiceAccountID`). Every field is `omitempty`
and nothing conflicts.

Migration versions collide if nobody coordinates. The agentauth and DPoP plans
both claim `20260824000001` against `authsome_sessions`, non-human principals
takes `20260824000050` through `...0052`, and this design takes `...0060` and
`...0061`. Pick from a free range rather than appending to the lowest one.

## Known limitations

Exchanged tokens keep their subject's roles. Scope narrowing is real for
anything that checks scopes and completely invisible to anything that checks
roles, so a token you think of as downgraded still passes every role-gated route
its subject passed, which is a surprising result if you have not read this far.
Narrowing both axes needs a scope-to-role mapping that doesn't exist in this
codebase yet. The follow-up is a role filter on the delegation grant, beside
the scope filter it already carries, and until that ships this limitation
belongs in the user-facing docs and not only here.

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
governed by a delegation grant. A subject token signed by GitHub or Google or an EKS
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
