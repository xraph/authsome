# Workload identity federation

Status: design, approved 2026-08-24
Depends on: [non-human principal enforcement](2026-08-24-non-human-principal-enforcement-design.md)

## Why this exists

A GitHub Actions workflow that calls an authsome-protected API has one option
today, and it's a static API key pasted into a repository secret. Same for a
Cloud Run service, same for an EKS pod. Those keys don't rotate, they outlive
the person who created them, and they leak. They're also invisible: an API key
authenticating from a new IP looks exactly like an API key authenticating from
the usual one, so nothing in `plugins/riskengine` or `plugins/anomaly` has
anything to work with.

Every one of those platforms already signs a short-lived OIDC token that says
precisely which workload is running. GitHub tells you the repository, the ref,
the workflow and the run. EKS tells you the namespace and the service account.
The token is free, it expires in minutes, and nobody has to store it.

This plugin trusts those tokens. An org admin registers an external issuer and
writes rules about which claims map to which internal principal, then a workload
presents its platform token and gets back a short-lived authsome credential
scoped to what that rule allows. No secret lives in CI.

## What already exists, and what doesn't

Two things were worth checking before designing anything.

Non-human principals are already here. `serviceaccount.ServiceAccount` is a
first-class entity, `session.Session` carries `PrincipalKind` and
`ServiceAccountID`, and the apikey plugin mints service-account sessions today,
so the credential this plugin issues is that principal and there is nothing
synthetic about it and no fake user row anywhere.

What is missing is the enforcement path, so a service-account session currently
gets rejected by every guard it meets. That's the prerequisite spec and it has
to land first.

RFC 8693 token exchange has been designed, and it does not fit. No code exists
yet, but `2026-08-24-token-exchange-rfc8693-design.md` landed the same day as this
one. Read it before you read the endpoint decision below, because the obvious
question is why this plugin does not simply use that grant, and the answer is
that the grant supports exactly two subject token types and both of them mean
"an authsome session token". An `id_token` gets you `unsupported_token_type`,
external JWTs are listed under out-of-scope, the caller has to present
`client_id` and `client_secret`, and every exchange needs a matching row in
`authsome_oauth2_exchange_policies`. That design starts from a credential you already hold and trades it for a
narrower one, where this plugin starts from a credential somebody else signed
and has to decide whether to believe it at all.

Remote JWKS verification does not exist anywhere in the tree. What we have is
`api/jwks_handler.go`, which publishes our own keys, and `plugins/sso/oidc.go`,
which fetches a discovery document to find endpoints and then never verifies an
ID token signature with what it found. Verifying a token somebody else signed is
new code.

## Decisions

The exchange endpoint lives in this plugin, not on `/v1/oauth/token`.

This is the decision most worth arguing with, so here is the whole argument. The
two exchanges look similar from a distance and differ on every axis that
matters: who authenticates, which is a registered confidential client in one
case and nobody at all in the other; what the subject token is, which is our own
session row in one case and a JWT signed by GitHub in the other; and what
authorizes the trade, which is a policy row in one case and an issuer trust
config plus a claim rule in the other. Merging them produces a single endpoint
carrying two disjoint authorization models, one of which turns client
authentication off. That is the shape of endpoint that eventually gets a CVE
written about it.

There is a practical objection underneath the design one. `oauth2provider` is
registered independently, so mounting this on its token endpoint means workload
identity does not work unless you also enable OAuth2, and the two release cycles
become one. Its token endpoint also assumes registered clients throughout:
`resolveScopes` reads `client.Scopes` and `clientSupportsGrant` checks the
client's grant list (`plugins/oauth2provider/plugin.go:546`). Threading a
secretless caller through that is not a small carve-out.

So this plugin gets `POST /v1/workload/token` in RFC 8693 wire format with no
client authentication, and the boundary between the two endpoints is written
down in both specs so that neither one drifts into the other's territory as
people forget why they were separated.

The rest, briefly. A rule binds to a service account you created beforehand, so
a bad rule grants an existing scoped identity rather than conjuring a new one.
The issued credential is a persisted, revocable session honouring the app's
configured token format. Attribution lives in a plugin-local exchange record
keyed by session ID, which keeps `session.Session` out of it. Claim rules are
validated against compile-time issuer profiles that make the pinning claims
mandatory.

## Data model

Three tables, plugin-local, following how `plugins/sso` owns its own store
rather than joining `store.Store`. Tenancy mirrors sso: `app_id`, `env_id` and
`org_id` on both config tables.

### authsome_workload_issuers

The trust configuration. Identity and tenancy columns, then `profile` (one of
`github-actions`, `gcp`, `eks`, `spiffe`, `generic-oidc`), `issuer_url`,
`jwks_uri`, `audience`, `allowed_algorithms`, `required_claims` for the generic
profile, and `active`.

Two constraints carry weight. `audience` is `NOT NULL` with no default and an
empty value is refused, because an unpinned audience is what lets a token minted
for somebody else's service get replayed at yours. And `issuer_url` is unique
per `(app_id, env_id)`, so you can't register the same issuer twice at different
strictness and have evaluation quietly pick the lenient one.

### authsome_workload_rules

The claim-matching rule: `issuer_id`, `name`, `priority`, `conditions` as a JSON
array of `{claim, op, value}`, `service_account_id`, `granted_scopes`,
`granted_roles`, `session_ttl_seconds` and `active`.

`service_account_id` is `NOT NULL`. There's no such thing as a rule that matches
and binds to nothing.

### authsome_workload_exchanges

Written on every successful exchange. It holds `session_id`, `issuer_id`,
`rule_id`, `service_account_id`, the raw `subject`, a JSON `claims` map of the
profile's attribution claims, the token's `jti`, plus `ip_address`, `issued_at`
and `expires_at`.

The unique index on `(issuer_id, jti)` is doing more work than it looks like.
It's replay prevention: a token presented twice loses to a constraint violation,
and you never size or expire a cache to get that. The rows you need for the
audit trail are the same rows doing the deduplication, so retention becomes a
delete job and not a correctness question. Then indexes on `session_id` for
the audit join and `(app_id, issued_at DESC)` for the log view.

Three new ID prefixes in `id/id.go`, following the four-character convention:
`awid`, `awrl`, `awxc`.

## Token verification

This goes in a new top-level `oidcverify` package, not inside the plugin.
Signature verification is the part of this feature where a bug is silent and
total, and code that sits next to HTTP handlers ends up tested only through the
endpoint. Standalone, it takes claims and keys and returns verified claims, so
its tests are tables of crafted tokens with no server involved.

Top-level, not `internal/`, because `internal/` here holds only
`secutil`, a test helper, while every runtime domain package in this repo
(`tokenformat`, `ceremony`, `apikey`, `serviceaccount`, `bridge`) sits at the
top. There's a plausible second consumer in `plugins/sso`, which doesn't verify
ID token signatures today, but this package is designed for the one caller it
has.

Seven steps, ordered so the cheapest and most restrictive gates run first.

1. Parse the token without verifying it, reading `iss` and `kid` and nothing
   else. Refuse `alg: none`, and refuse any algorithm outside the trust config's
   `allowed_algorithms`.

2. Resolve the trust config by exact `iss` match, scoped to `(app_id, env_id)`.
   This step matters far more than its size suggests, because key resolution
   means an outbound HTTPS fetch, and if an unregistered `iss` could reach that
   fetch then anyone holding a token could make the server request a URL of
   their choosing. Resolving registered trust first means we only ever fetch
   from URLs an admin put there. Exact string equality after trailing-slash
   normalisation, and no prefix logic.

3. Resolve the JWKS URI, from either an explicit `jwks_uri` or discovery at
   `{issuer}/.well-known/openid-configuration`. Three constraints on what comes
   back: HTTPS only, the document's own `issuer` must equal the registered
   issuer, and the `jwks_uri` host must match the issuer host. Skip the last two
   and a discovery document served through a hijacked path points key resolution
   wherever it likes. Use a dedicated `http.Client` with an explicit timeout and
   a response size cap, not `http.DefaultClient` the way `plugins/sso/oidc.go`
   does at line 178.

4. Fetch and cache keys by `(issuer_id, kid)` with a TTL, plus negative caching
   for a `kid` the issuer does not have. An unknown `kid` triggers at most one
   refresh per issuer per cooldown window, so a flood of tokens carrying random
   `kid` values cannot be turned into an amplifier against the issuer or against
   us.

5. Verify the signature.

6. Validate the claims, now that they are authenticated. `exp` is mandatory and
   a token without one is refused, never defaulted. `nbf` and `iat` get a small
   fixed skew allowance. `aud` must contain the configured audience by exact
   match against each array element, and `iss` is rechecked. Maximum token age
   is enforced separately from `exp`, because some platforms issue long-lived
   OIDC tokens and the exchange window should not be whatever they happened to
   pick.

7. Refuse replays, which the exchange record insert handles as described above.

Step 2 is the one that's easy to misorder. The intuitive pipeline is "verify the
token, then look up who issued it", which feels safer because verification comes
first. But verification needs keys, keys need a fetch, and the fetch destination
comes from the token. Looking up registered trust first rejects an unknown
issuer before anything leaves the process.

## Issuer profiles

Profiles are compile-time, so nobody can edit one into permissiveness through
the admin API.

```go
type Profile struct {
    Name              string
    RequiredClaims    []string          // every rule must pin each of these
    PrefixableClaims  []string          // only these may use the prefix operator
    PrefixDelimiters  map[string]string // claim -> delimiter a prefix must end on
    AttributionClaims []string          // copied into the exchange record
}
```

`github-actions` requires `repository`, allows a prefix on `sub` and `ref`, and
attributes on `repository`, `ref`, `sha`, `workflow`, `run_id`, `actor`,
`environment` and `job_workflow_ref`.

`eks` requires `sub`, which arrives as `system:serviceaccount:<ns>:<name>`, and
allows a prefix on it for namespace-wide rules. It attributes on the
`kubernetes.io` namespace, pod and service-account claims.

`gcp` requires `sub` and attributes on `email` and the compute metadata claims.
`spiffe` requires `sub` and allows a trust-domain prefix.

`generic-oidc` has an empty `RequiredClaims`, and registration refuses it unless
the admin names the pinning claims explicitly. The generic profile is not a way
out of the mechanism.

## Claim matching

Two operators: `exact` and `prefix`. No regex, no globs. Regex in an
authorization matcher is a review problem before it's a ReDoS problem, because
nobody reading a rule list can tell at a glance what a `.*` in the middle of a
pattern admits.

A rule is rejected at write time when any of these hold.

1. A claim in the profile's `RequiredClaims` is missing from the conditions. The
   error names the claim.
2. A condition value is empty, is `*`, or contains `*` or `?` anywhere. The
   error says wildcards aren't supported. Rejecting matters here, because if
   you quietly treat `*` as a literal you get a rule that never fires, and the
   usual response to a rule that never fires is to loosen something else until
   it does.
3. The `prefix` operator is used on a claim outside `PrefixableClaims`.
4. A `prefix` value doesn't end on the delimiter the profile declares for that
   claim.

Rule 4 is the subtle one and it's the real bug class. A prefix of
`repo:acme/api` also matches `repo:acme/api-internal`, so anyone who can create
a repository called `api-internal` in your org inherits the identity you meant
for `api`. Requiring the prefix to end at `:` makes `repo:acme/api:` mean that
repository and nothing else. Same shape for `system:serviceaccount:prod:`, where
`prod` would otherwise also match `production`.

Audience is validated on the trust config, not the rule. Empty is always
refused. For the `github-actions` profile, the default audience GitHub hands out
(`https://github.com/<owner>`) raises a warning on save and a standing warning
in the dashboard, because a value shared with every other service that also
didn't configure one isn't much of an audience. It's a warning and not a
rejection, so the strictness is visible without being in your way.

Active rules for the matched issuer are evaluated by `(priority ASC,
created_at ASC)` and the first full match wins. The matched `rule_id` goes into
the exchange record, so "why did this workload get that account" is answerable
from the trail, not re-derived by hand. The dashboard flags a rule fully shadowed
by a higher-priority one, as a warning, since shadowing is legitimate during a
migration.

There's also a dry-run endpoint, and it's probably the highest-value thing here
per line of code. Submit a real token or a claims JSON and get back which issuer
resolved, which rule matched, which service account it would bind to, and the
scopes and roles it would carry. It mints nothing and writes no exchange record,
but it does write an audit event saying a dry run happened. It's the difference
between an admin verifying a rule and an admin finding out what it does in
production.

## Exchange flow

`POST /v1/workload/token`, no client authentication, accepting form-encoded and
JSON bodies.

`grant_type` must be exactly
`urn:ietf:params:oauth:grant-type:token-exchange`. `subject_token` carries the
platform token. `subject_token_type` must be
`urn:ietf:params:oauth:token-type:jwt` or the `id_token` variant. `scope` is
optional and space-delimited. `requested_token_type`, if present, must be the
`access_token` type.

App context comes from the `X-Publishable-Key` header, matching
`middleware.PublishableKeyHeader` and the rest of the product. A publishable key
is public by definition, so requiring one in CI costs nothing and keeps trust
lookup scoped to one app, so we never search every registered issuer.

The response is the 8693 success shape: `access_token`, `issued_token_type`,
`token_type` of `Bearer`, `expires_in` and `scope`. There is no `refresh_token`,
which 8693 allows you to omit. A refresh token sitting in a CI environment is a
long-lived static credential with extra steps. A job that needs another
credential exchanges again.

### Write the exchange record before the session

This ordering is what makes the unique `(issuer_id, jti)` index actually prevent
replay, so don't reverse it. Generate the session ID up front, which is free
because typeids are caller-generated. Write the exchange record carrying that
session ID. Then create the session.

Insert the session first and a replayed token mints a second session before it
loses the race on the record. Insert the record first and a replay fails the
constraint having minted nothing. The failure mode of this order is a claimed
`jti` whose session creation then failed, stranding one token, and that's the
safe direction: the caller requests a fresh platform token and retries.

### The session

`PrincipalKind` of `"service_account"`, `ServiceAccountID` from the matched
rule, nil `UserID`, and `Roles` from the rule's `granted_roles`. Because `Roles`
is already populated, `roleStampingStore.CreateSession` leaves it alone
(`engine_session_roles.go:187`), which is exactly the case that comment
describes.

TTL comes from the rule, clamped to a plugin setting with a 15 minute default
and a 1 hour ceiling. `RefreshToken` is empty and `RefreshTokenExpiresAt` is
zero.

Then the token format branch, mirroring `service.go:810`. Opaque by default, and
when `TokenFormatForApp` says `jwt`, generate one with `sub` set to the service
account ID and the `pk` and `sa` claims from the prerequisite spec.

Three checks fail closed, each with its own audit reason. The bound service
account must exist, must be `Active`, and must belong to the same `AppID` as the
request. A rule pointing at a deactivated or cross-app account is a
configuration error and must not authenticate.

Requested scopes must be a subset of `rule.granted_scopes` intersected with the
service account's own `Scopes`. Downscoping is honoured. Upscoping is refused
and not quietly clamped, because quiet clamping means a job believes it has
permissions it doesn't and then fails somewhere far less legible.

## Attribution

Every exchange writes a `bridge.AuditEvent`, success or failure. Action
`workload.token.exchange`, `ActorID` the service account, `ResourceID` the
matched rule, and `Metadata` carrying the profile's attribution claims alongside
issuer, rule name, session ID and `jti`. `AuditEvent.Metadata` is already
`map[string]string` (`bridge/chronicle.go:26`), so nothing in core has to
change.

The trail then reads as the workload itself: actor `svc_...`, metadata
`repository=acme/api`, `ref=refs/heads/main`, `workflow=deploy`, `run_id=…`,
`actor=…`. That's the deploy job of repo X on branch main, and it's queryable
and not free text.

Failures are audited too, with a specific reason and a graded severity: unknown
issuer, signature invalid, audience mismatch, no rule matched, replay detected.
Replay is `SeverityCritical`. No-rule-matched is `SeverityWarning`. Each also
goes onto the hook bus following the apikey pattern
(`plugins/apikey/plugin.go:686`), so `riskengine` and `anomaly` can subscribe.

That closes the original complaint. A static key in CI is invisible. A federated
exchange is an audited, subscribable event with the workload's identity on it.

## Plugin surface

Files mirror `plugins/sso`: `doc.go`, `plugin.go`, `contract.go`, `profiles.go`,
`matching.go`, `exchange.go`, `migrations.go`, `store_models.go`, `store.go`
plus `store_memory.go`, `store_postgres.go`, `store_sqlite.go` and
`store_mongo.go`, then `dashboard.go` and `dashui/`.

The plugin implements `OnInit`, `RouteProvider`, `MigrationProvider` and
`SettingsProvider`. It does not implement `StrategyProvider`, because the
credential it issues is an ordinary session and the existing session path
resolves it once the prerequisite spec lands.

Public route: `POST /v1/workload/token`.

Admin routes under `SessionGuard` plus `PermissionGuard`: CRUD on
`/v1/workload/issuers` and `/v1/workload/issuers/{id}/rules`, `GET
/v1/workload/exchanges` for the log view, and `POST /v1/workload/dry-run`.

Settings via `SettingsProvider`, following the sso pattern: default and maximum
session TTL, JWKS cache TTL, discovery timeout, maximum token age, clock skew
allowance, and exchange record retention in days.

## Testing

`oidcverify` gets table tests over crafted tokens signed with a locally
generated key: `alg: none`, an unlisted algorithm, expired, future `nbf`,
missing `exp`, wrong audience, wrong issuer, unknown `kid`, tampered signature.
No HTTP server, just claims and keys.

Matching gets per-profile tables covering an omitted required claim, every
wildcard form, a prefix on a non-prefixable claim, and explicitly the
`repo:acme/api` against `repo:acme/api-internal` delimiter case. That last one
is the bug this design exists to prevent, so it gets a test with the org name
spelled out.

Stores get the conformance suite across all four backends, mirroring
`plugins/sso/store_conformance_test.go`.

Exchange tests: a replayed token returns 400 and mints nothing, a deactivated
service account is refused, a cross-app service account is refused, downscoping
works, upscoping is refused, and TTL clamping holds at both ends.

Then the one that proves the two specs compose. Exchange a token, call a
`SessionGuard`-protected route with the result, assert 200. It fails today for
reasons that have nothing to do with this plugin, which is why the prerequisite
spec exists.

## Phasing

Full parity with `plugins/sso` is a lot of surface, so build it in order and
keep each phase shippable.

0. Non-human principal enforcement (the prerequisite spec).
1. The `oidcverify` package, with its table tests.
2. Data model, four store backends, migrations, conformance tests.
3. Profiles, matching and write-time validation.
4. The exchange endpoint, attribution and audit.
5. Admin API and dry-run.
6. Dashboard and `dashui`.
7. SDK regeneration across `sdkgen/spec.json` and the Go, TypeScript and Dart
   clients.

Phase 7 is real work, not a footnote. Those clients are generated, and
they're already carrying uncommitted changes in the working tree, so plan on
regenerating deliberately, and not at the end of a long day.

## Relationship to the other specs landing this week

Six designs went into `docs/superpowers/specs/` on 2026-08-24 and three of them
touch this one. None of the three is implemented yet, so the seams below are
cheap to agree on now and expensive to discover later.

Token exchange, RFC 8693. Two endpoints, and the boundary between them is the
subject token. If the subject token is an authsome session and the caller is a
registered client narrowing what it already holds, that is `/v1/oauth/token` and
it is governed by an exchange policy row. If the subject token was signed by
somebody else and the caller holds no secret at all, that is
`/v1/workload/token` and it is governed by an issuer trust config and a claim
rule. Neither endpoint should ever grow the other's case. A workload that wants
a narrower credential exchanges here first, then narrows there, and the audit
trail shows both hops.

Agent delegation. That spec adds `PrincipalKind = "agent"` and is explicit
that autonomous agents with no human behind them stay service accounts, which is
exactly the principal this plugin issues. The two do not overlap. They do share
the enforcement prerequisite, so the `principalFrom` helper in that spec is
written for any non-user principal and not for service accounts specifically.

DPoP. The 8693 draft already writes down the rule that an exchange must
never turn a bound token into an unbound one. The same rule applies here with
one wrinkle worth naming: a workload's platform token is not DPoP-bound and
never will be, since GitHub and GCP do not issue bound tokens, so an exchange
here starts unbound by definition. If DPoP lands with a per-app requirement,
this endpoint needs to either accept a `dpop_jkt` parameter and bind the issued
session to it, or be explicitly exempted. Whichever of the two ships second owns
that decision and its test.

## Open items

The GitHub default-audience check is a warning by decision. If it turns out
people ignore it, promoting it to a rejection is a one-line change plus a
migration path for existing rows. Make that call with usage data, not today.

Exchange record retention has a setting but no enforcement job in this spec.
Until one exists the table grows without bound, which is fine at first and won't
stay fine. Worth a follow-up once phase 4 is running somewhere real.
