# sharedsignals: OpenID Shared Signals Framework and CAEP

Status: approved design, not yet implemented

Date: 2026-08-24

## The problem

When Okta decides one of your users is compromised, authsome finds out never.
The account is disabled upstream, the security team files the incident, and
that user's authsome sessions keep working until they expire on their own
schedule. Same story for Entra and Google Workspace. Grep the tree for CAEP,
SSF, RISC or continuous access and you get nothing, because the only signal
plumbing we have is `webhook/`, and that only points outward.

The Shared Signals Framework is the standard fix. An identity provider acting
as a Transmitter sends Security Event Tokens to a Receiver over a configured
stream, and CAEP defines the event types that describe an account going bad.
This plugin makes authsome a Receiver, and later a Transmitter, so a
compromise decision made upstream lands here in seconds.

One thing to keep in front of you the whole way through. This endpoint takes
instructions from an outside party about whose sessions to destroy. Signature
validation, transmitter allow-listing and subject resolution are not features
alongside the rest, they are the product. A receiver that accepts too much is
a remote session-kill button with a public URL.

## Scope

Four subsystems, shipped as four milestones. Each one is independently
revertible and only the first is on the critical path.

| Milestone | What lands |
|---|---|
| M1 Receiver core | push endpoint, SET validation, JWKS client, subject resolution, the five core CAEP handlers plus `risk-level-change` and SSF `verification`, action matrix, signal store across four backends, risk contributor, admin CRUD |
| M2 Outbound stream client | discovery of a transmitter's SSF configuration, stream creation, the verification handshake |
| M3 Dashboard | contract manifest and a dashui page showing streams and recent signals |
| M4 Transmitter emit | authsome signs and delivers its own SETs, hosts the stream management API, push and poll delivery |

M1 solves the problem in the first paragraph. If you ship nothing else, a
compromised account upstream still loses its authsome sessions, which is the
whole point of the exercise.

## Architecture

The receiver and the transmitter share a core, and that core knows nothing
about authsome. SET parsing, JWKS handling, subject identifiers and CAEP
payload shapes are pure spec plumbing. They get their own packages with no
authsome imports, so you can test them against spec fixtures alone and review
them as security code without reading plugin wiring. You end up with spec code
you can hand to a reviewer on its own, and plugin code that reads like every
other plugin in the tree.

```
plugins/sharedsignals/
  caep/          event type URIs, typed event payloads, RFC 9493 subject
                 identifiers, SSF complex subjects. No dependencies.
  setjwt/        RFC 8417 parse, validate and sign.
  jwksclient/    fetch, cache, rotate, kid lookup.
  ssfclient/     transmitter discovery and stream management client (M2).

  plugin.go       lifecycle, settings, wiring
  receiver.go     the push endpoint
  dispatch.go     event type to handler routing
  actions.go      the action matrix
  subject.go      subject identifier to authsome user
  risk.go         RiskContributor reading back durable signals
  links.go        subject-link stamping
  transmitter.go  emit path and hosted management API (M4)
  store_memory.go store_postgres.go store_sqlite.go store_mongo.go
  migrations.go   three migrate.Groups
  contract.go dashboard.go dashui/   (M3)
```

Three notes before you start writing any of it.

There is no inbound JWKS client anywhere in this repo. `api/jwks_handler.go`
publishes keys and `tokenformat/jwt.go` validates our own tokens, and neither
helps. You'll write `jwksclient` from scratch on `golang-jwt/jwt/v5`, which is
already a direct dependency. Don't add `go-jose` for this. The extra
supply-chain surface isn't worth the few hundred lines it saves.

Mongo gets a real migration group. `store/mongo/migrations.go` already drives
`mongomigrate.Executor` with `CreateCollection` and `CreateIndexes`, so the
machinery is there. `plugins/sso` skips mongo migrations entirely and that's a
gap, not a pattern to copy. Register three groups, all depending on
`authsome`.

The plugin needs a job queue and `plugin.Engine` doesn't expose one.
`bridge.Dispatcher` exists but no accessor reaches it. Rather than widen the
Engine interface for everybody, add an optional capability interface next to
`LedgerEngineProvider`, and fall back to an owned goroutine worker with
`OnShutdown` when the host has no dispatcher.

## Data model

Five tables, all prefixed `authsome_ssf_`. You'll notice inbound and outbound
streams live in separate tables instead of one table with a direction column.
They share about half their columns and none of their trust properties, and
nobody reviewing the security of this should have to work out which branch of
a discriminator they're reading.

`authsome_ssf_inbound_streams` holds one row per identity provider you trust,
and carries `push_path_hash` (SHA-256 of the secret URL segment, unique
index), `push_token_hash`, `issuer`, `jwks_uri`, `audience`,
`allowed_event_types`, `allowed_subject_formats`, `verified_domains`,
`action_overrides`, `enforcement_mode`, `status`, `max_actions_per_hour` and
`last_verified_at`.

The join key that makes `iss_sub` work is `authsome_ssf_subject_links`, which
maps `(app_id, env_id, issuer, subject)` to a user, unique on that tuple, with
a `source` of `sso`, `social`, `scim` or `manual`.

Two jobs land on one row in `authsome_ssf_received_events`. It is the replay
guard, unique on `(stream_id, jti)`, and it is the audit trail, carrying
`outcome`, `action_taken`, `resolved_user_id` and `error`. One table because
the dedupe row and the what-happened row have the same key and the same
lifetime, so splitting them buys nothing.

Durable risk state lives in `authsome_ssf_signals`: `user_id`, `event_type`,
`severity`, `expires_at`. That is the table that makes an event arriving at
02:00 visible to a sign-in at 08:00.

Then `authsome_ssf_outbound_streams` and `authsome_ssf_deliveries`, both M4.
The `authorization_header` is encrypted through `engine.TokenEncryptor()`,
same as sso client secrets. The deliveries table is the retry queue for push
and the unacked buffer for poll.

There is no signing-key table, and that is deliberate. The transmitter signs
with a key from `Config`, mirroring `tokenformat.JWTConfig`, and publishes it
at `/.well-known/ssf-jwks.json`. Rotation is out of v1. An operator restarts
with a new key and the SETs already delivered stay delivered. That drops a
table, a rotation state machine and a key-encryption path, and it can come
back later if anyone actually needs it.

Secrets follow the `apikey` pattern. The push URL segment and the bearer token
are shown once at stream creation and stored only as hashes, so a stolen
database doesn't hand anyone a working push endpoint.

## The receive pipeline

`POST /v1/ssf/streams/{push_path}/events` with `Content-Type:
application/secevent+jwt`. Gates run cheapest and least trusting first.

| Step | Check | On failure |
|---|---|---|
| 1 | per-IP rate limit, body through `io.LimitReader` at 64 KiB | 429 or 400 `invalid_request` |
| 2 | find the stream by `SHA-256(push_path)`, running a dummy compare on a miss | 404, generic body |
| 3 | stream status is `enabled` | 403 `access_denied` |
| 4 | `subtle.ConstantTimeCompare` on the bearer token hash | 401 `authentication_failed` |
| 5 | parse the JWS header without verifying, for `kid` and `alg` only | 400 `invalid_request` |
| 6 | `alg` in RS256/384/512, ES256/384/512, EdDSA. No `none`, no HMAC | 400 `invalid_key` |
| 7 | `typ` header is `secevent+jwt` | 400 `invalid_request` |
| 8 | resolve `kid` through this stream's `jwks_uri` | 400 `invalid_key` |
| 9 | verify the signature | 400 `invalid_key` |
| 10 | `iss` equals `stream.issuer` exactly, `aud` contains our audience | 400 `invalid_issuer` or `invalid_audience` |
| 11 | `jti` present and length-capped, `iat` inside `[now-24h, now+5m]` | 400 `invalid_request` |
| 12 | insert `authsome_ssf_received_events` unique on `(stream_id, jti)` | conflict returns 202, it's a replay |
| 13 | per event: resolve subject, check the allow-list, act, write a signal | recorded, not fatal |
| 14 | 202 Accepted, empty body | |

Two semantics here are worth defending, because both look wrong at first
glance.

A subject we can't resolve returns 202, not an error. Erroring would tell the
transmitter which of its users have authsome accounts, which is a user
enumeration oracle you're handing to the one party you're least able to audit.
It would also put the transmitter into a permanent retry loop over an event
that will never succeed. The row records `outcome: unresolved` and we move on.

The dedupe row commits before any action runs, which makes the row the ledger.
Processing is inline under a hard deadline. If the deadline trips or an action
fails, the row stays `pending` and a dispatcher job picks it up. That's
at-least-once delivery with the unique constraint making retries safe.

Errors use the RFC 8935 body exactly, `{"err": ..., "description": ...}`, and
the description never echoes anything the caller sent. If you find yourself
wanting to put the offending value in there while debugging an integration,
put it in the audit record instead, where it is scoped to an operator who
already has access rather than returned to whoever sent the token.

## Security

Three independent gates stand between an outside party and a dead session: the
unguessable path segment, the bearer token, and the signature. Breaking one of
them gets you nothing, and you would need all three.

Nothing about which keys to trust ever comes out of the token. The URL finds
the stream row, and that row pins `issuer`, `jwks_uri`, `audience`, `app_id`
and `env_id`. The token's `iss` gets checked against the stream, it never
selects one. `kid` picks among that stream's keys and no others. A genuine
Okta SET replayed at another tenant's push URL dies at step 10.

The JWKS client is the one component that makes outbound requests because of
inbound traffic, so it's hardened accordingly: HTTPS only, 5 second timeout,
256 KiB response cap, 20 key cap, no cross-host redirects, single-flight,
negative caching, and a minimum refetch interval per stream so a flood of
unknown-`kid` tokens can't drive our fetch rate. Keys also refresh on a
background timer, which means a real IdP key rotation gets noticed without
waiting for traffic to trigger it. When an admin registers a stream we
validate the `jwks_uri` against private, loopback and link-local ranges,
because an admin pasting `169.254.169.254` is still an SSRF even when the
admin meant well.

Then there's the case where the signature is perfectly valid and the events
are still wrong, which no amount of crypto helps with. Every stream carries
`max_actions_per_hour`. Cross it and the stream flips to `paused`, we write a
`SeverityCritical` audit event and fire a
`security.ssf.circuit_breaker_tripped` relay event. A transmitter that
suddenly wants ten thousand sessions gone is either compromised or
misconfigured, and both of those want the same response, which is to stop and
wake somebody up. Correctness gates handle forged events. The breaker bounds
authentic ones.

`enforcement_mode` defaults to `enforce`, with `observe` available per stream
when you want to watch a new integration before trusting it. Every accepted
and rejected event goes to `bridge.Chronicle`, at `SeverityCritical` when
sessions died.

## Subject resolution

Accept both `sub_id` and `subject`. CAEP 1.0 final says `sub_id`, but Okta and
Google both ship `subject` today, and this is verified against Okta's
published SET payloads rather than guessed. Prefer `sub_id` when a token
carries both.

Resolution runs in this order:

1. An object with no `format` member is a complex subject. Take `user` for
   identity and `session` for a targeted revocation. If the transmitter's
   configuration declares a `critical_subject_members` entry we can't process,
   discard the event, which is what SSF requires.
2. `iss_sub` requires `iss` to equal `stream.issuer`, because one IdP doesn't
   get to assert subjects belonging to another. Then look up
   `authsome_ssf_subject_links`.
3. `opaque` uses the same table, keyed on `stream.issuer` plus `id`.
4. `email` resolves only when `email` is in `allowed_subject_formats` and the
   domain is in `verified_domains`. Call `GetUserByAnyEmail`, and then check
   that the matched address is verified on that user.
5. `phone_number` gets the same gating, normalized to E.164, verified required.
6. `aliases` resolves every member. If two members land on different users,
   reject the whole event.
7. `account`, `uri`, `did` and anything unrecognised get recorded and ignored.

Step 4's verified check is the one people skip, so here's why it's there.
Users can attach secondary email addresses to their accounts, so if you match
on any address without checking verification, someone attaches `ceo@corp.com`
to their own account and from then on every session-revoked event meant for
the CEO resolves to the attacker instead, killing the attacker's own sessions
at no cost to them while the sessions you were trying to protect stay up.
Check verification.

Every lookup is scoped to the stream's `app_id` and `env_id`. There is no
cross-app path.

Populating the link table splits by source. Social already persists
`authsome_oauth_connections (provider, provider_user_id, user_id)`, so those
links are derivable with a provider-to-issuer map and no new writes at all.
SSO keeps no per-user subject anywhere, so we add one: this plugin exposes
`LinkSubject(...)`, and sso calls it when the plugin is registered, using the
`engine.Plugin(name)` plus interface assertion idiom the repo already uses for
risk contributors.

## Actions

| Event | Default |
|---|---|
| `session-revoked` | revoke every session the user has in that app and env. If a `session` subject member resolved, revoke only that one |
| `credential-change` | `revoke` and `delete` revoke all sessions, `create` writes a signal |
| `token-claims-change` | revoke all sessions |
| `assurance-level-change` | `decrease` writes a high severity signal, `increase` a low one |
| `device-compliance-change` | `not-compliant` writes a high severity signal |
| `risk-level-change` | signal, severity taken from the level |
| SSF `verification` | match `state` against the pending challenge, stamp `last_verified_at`, no user action |

`token-claims-change` revokes because of how sessions work here.
`session.Roles` is stamped at issue time and never re-resolved, which the
field comment on `session.Session` is explicit about. A claims change upstream
can't reach an existing session, so the only way to apply it is to end the
session and let the next one pick up current roles.

Revocation goes through `Engine.RevokeSession` one session at a time, not
`store.DeleteUserSessions`, so `AfterSessionRevoke` hooks, the hook bus and
the outbound relay all fire. You pay N calls for that and it is worth it.
`plugin.Engine` doesn't expose the method, so add one optional capability
interface beside `LedgerEngineProvider`:

```go
type SessionRevoker interface {
    RevokeSession(ctx context.Context, sessionID id.SessionID) error
}
```

`*authsome.Engine` already has exactly that method, so it satisfies the
interface for free and `service.go` doesn't change.

## Feeding the risk engine

Step-up on next sign-in needs no new mechanism. A high severity signal makes
our `RiskContributor` return a high score, riskengine crosses its medium
threshold, and its decision comes back `challenge`. Expressing step-up through
riskengine instead of a new per-user flag is what feeding the risk plugin
family ought to mean.

There's a bug in the way, though. In `plugins/riskengine/plugin.go`,
`OnBeforeSignIn` builds `RiskRequest{IPAddress, UserAgent, AppID}` and never
sets `UserID`. The field exists, the audit metadata reads it, and it's empty
every single time. Every contributor shipped so far is IP-based so nobody
noticed, but it means any user-scoped contributor gets handed nothing. M1
includes a small contained fix: carry the sign-in identifier into
`RiskRequest` and resolve `UserID` where we can. Do this before you write the
contributor, or you'll spend an afternoon debugging a scorer that is working
perfectly against an empty input. That fixes the path for every future
contributor, not just this one.

Score is the maximum severity across unexpired signals for that user, decayed
linearly over the signal TTL. Default weight is 2.0, because a CAEP event from
an IdP that watched the account get taken over is worth a great deal more than
an IP heuristic.

## Outbound client and transmitter

M2 is `ssfclient`. Fetch `.well-known/ssf-configuration`, POST the
configuration endpoint to create a stream with `delivery: {method:
"urn:ietf:rfc:8935", endpoint_url, authorization_header}`, POST the
verification endpoint with a random `state`, then wait for the verification
event to arrive at our own receiver and match it. Store the transmitter's
`stream_id` on our row. Vendors differ from each other and from the spec here,
so write this one lenient and loud. You want a log line naming exactly which
field came back in a shape the spec does not describe, not a parse error
thirty frames deep.

M4 hosts the transmitter half: `.well-known/ssf-configuration`,
`.well-known/ssf-jwks.json`, the configuration, status, verify, add-subject
and remove-subject endpoints, and an emit path that subscribes to our own hook
bus for session revocation, user deletion and credential changes, turning them
into signed CAEP SETs. Delivery runs through `authsome_ssf_deliveries` with
exponential backoff for push, and the same rows serve as the unacked buffer
for poll receivers. Receivers authenticate to our management API through the
existing `apikey` plugin and `auth.Registry`.

## Testing

TDD throughout. The security tests are the deliverable rather than a
follow-up, so you write them first. They are named here so the plan can point
at them:

`alg: none`, HMAC signed using the IdP's public key as the secret, wrong
`iss`, wrong `aud`, `iat` too old, `iat` in the future, missing `jti`,
replayed `jti`, unknown `kid`, a `kid` belonging to another stream's JWKS, a
valid SET replayed at another tenant's push URL, a subject from another app,
an email outside `verified_domains`, an email matching an unverified secondary
address, `aliases` members that disagree, an oversized body, an oversized JWKS
document, and a circuit breaker trip.

On top of that: fixture SETs shaped like real Okta output, meaning `subject`
and not `sub_id`; a store conformance suite across memory, postgres, sqlite
and mongo following `plugins/sso/store_conformance_test.go`; and one
end-to-end test with a fake transmitter, an `httptest` server holding a real
keypair and serving its own JWKS, that pushes a `session-revoked` and asserts
the session is gone from the store afterwards.

## Changes outside the plugin

Small and contained, listed here so review can find them:

- `plugin/plugin.go` gains `SessionRevoker` and `DispatcherProvider`, both
  optional capability interfaces alongside the existing `LedgerEngineProvider`.
- `plugins/riskengine/plugin.go` populates the sign-in identifier and `UserID`
  on `RiskRequest`.
- `plugins/sso` calls `LinkSubject` after a successful OIDC sign-in when the
  sharedsignals plugin is registered.

## Out of scope for v1

Transmitter signing key rotation. Poll-based receiving, meaning authsome
polling somebody else's transmitter, since every IdP named in the brief
supports push. The `account`, `uri` and `did` subject formats. RISC event
types beyond the CAEP set, which are a straightforward addition later because
the dispatch table is data.
