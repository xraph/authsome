# DPoP (RFC 9449) design

Status: approved design, not yet implemented
Date: 2026-08-24

## What this gives you

Every token authsome issues today is a bearer token, which means whoever holds
it can use it, and it means a session token copied out of a log file or lifted
by an XSS payload works exactly as well for the person who stole it as it does
for the person it was issued to. We already spend a lot of effort noticing that
after the fact, through session binding to IP and device,
`plugins/impossibletravel`, `plugins/anomaly` and the refresh replay cascade in
`Engine.Refresh`. None of it stops the first replay, though.

DPoP binds a token to a key pair the client holds. The client signs a small JWT
per request proving it still has the private key, and the server refuses the
token without that proof. A stolen token on its own is useless.

You turn it on per client, per app or globally, and nothing you issued before
you turned it on changes behaviour.

## Decisions

OAuth access tokens and session tokens are the same object here. Look at
`issueTokens` in `plugins/oauth2provider/plugin.go` and you will find that it
builds an `account.Session`, optionally overwrites `sess.Token` with a JWT,
saves it through the ordinary session store and hands `sess.Token` straight
back as the `access_token`, which means there is no separate OAuth token entity
anywhere in the codebase for a confirmation key to hang off. So the key lives
on `session.Session`, and both the OAuth token endpoint and first-party sign-in
stamp it.

Configuration comes in three layers: global, app and client. Global and per-app
ride the existing settings manager. Per-client is a column on `OAuth2Client`.

The replay cache gets its own interface. `ceremony.Store` looked like the right
home and it is not, for one reason: ceremonies run a handful of writes per
sign-in while a `jti` cache takes a write on every single authenticated
request, so anybody who follows the invitation in that package's own doc
comment and plugs in a database-backed ceremony store would quietly buy a round
trip per API call for something that was supposed to be cheap. So we define
`dpop.ReplayCache`, default it to a bounded in-memory implementation, and ship
an adapter over `ceremony.Store` for people who already run a distributed one
and want the cross-instance guarantee.

JWK handling is hand-rolled in `internal/jwkutil`, with no new dependency. We
already hand-write the key-to-JWK direction in `api/jwks_handler.go`, and this
puts both halves in one place where the round trip is testable.

Proof `iat` is accepted from 60 seconds in the past to 30 seconds ahead. The
window is deliberately lopsided, because the two directions mean completely
different things: a proof dated in the future is somebody pre-minting proofs
and deserves almost no slack at all, while a proof dated in the past is
overwhelmingly just a phone with a drifting clock, and tightening that side
buys you very little except support tickets.

## Two invariants

These hold regardless of configuration, and the design's wrong if either one
ever becomes a setting.

Enforcement is a property of the token, not the mode. Once a token is bound,
whether that binding lives in a `cnf` claim or in the session row, presenting
it without a valid proof fails. Always. Mode governs issuance only, and that's
what makes an `optional` mode safe instead of a downgrade oracle an attacker
gets to pick from.

Binding survives refresh. A refresh of a bound session has to re-prove the same
key, and the rotated session inherits the thumbprint. Skip either half and a
stolen refresh token launders itself into a fresh unbound access token, which
makes everything else here decorative.

## Package layout

```
dpop/                      core, sibling of tokenformat/ and ceremony/
  proof.go                 Proof struct, Parse, header and claim types
  validator.go             Validator, Validate(ctx, Proof, Expectation)
  thumbprint.go            JKT(jwk), RFC 7638
  replay.go                ReplayCache interface, bounded memory impl
  ceremonyadapter.go       ReplayCache over ceremony.Store, opt-in
  nonce.go                 stateless HMAC nonce issuer and verifier
  errors.go                typed errors mapped to RFC 6750 and 9449 codes
internal/jwkutil/          JWK to crypto.PublicKey and back
```

Core, not a plugin, and it is worth saying why, because this repo puts a lot in
`plugins/`. Core cannot import plugins. If DPoP validation lived in one then
`middleware/auth.go` would have to reach it through a registry or a hook, which
makes enforcement late-bound, and late-bound enforcement means that the day
somebody ships a build where the plugin was never registered, every bound token
in your fleet quietly validates as an ordinary bearer token and nothing
anywhere logs a complaint. The plugin pattern fits features that add endpoints.
DPoP mostly constrains one you already have.

The validator itself is pure. It takes a parsed proof, an `Expectation`, and a
clock. No HTTP types, no store, no engine. Everything stateful gets injected at
construction, so the whole RFC test suite is table-driven with no fixtures.

## Validation sequence

Ordered cheapest first, so a malformed header dies before any crypto runs. The
ordering isn't cosmetic, and two of the steps below sit where they sit for
reasons that aren't obvious from the RFC.

1. Exactly one `DPoP` header, at most 4 KB. Oversized headers are rejected
   before parsing.
2. Well-formed compact JWS, three parts.
3. `typ` is `dpop+jwt`. This is checked before anything reads `alg`, and it's
   the algorithm-confusion firewall.
4. `alg` is one of ES256, RS256, PS256, EdDSA. `none` and every symmetric
   algorithm are rejected unconditionally.
5. `jwk` header present and public only. Reject if `d`, `p`, `q`, `dp`, `dq`,
   `qi` or `k` show up.
6. Parse `jwk` into a `crypto.PublicKey`, with curve and on-curve checks and a
   2048-bit floor on RSA moduli.
7. Verify the signature with that key.
8. `jti`, `htm`, `htu` and `iat` are all present.
9. `htm` matches the request method, case sensitive.
10. `htu` matches the request URI normalised to scheme, host and path. Query
    and fragment stripped.
11. `iat` falls inside the configured window.
12. `nonce` matches, when a nonce is required.
13. `ath` equals base64url(SHA-256(access token)) whenever a token accompanies
    the proof.
14. `JKT(jwk)` equals the bound token's thumbprint, when a bound token
    accompanies the proof. At the token endpoint there is no token yet, so
    this step is skipped and the key is learned instead.
15. `jti` has not been seen inside the window.

Step 15 goes last on purpose. Cache on first sight and an attacker can burn
`jti` values with cheap malformed proofs, which turns into false replay
rejections for the client that owns them.

Step 10 is the one that goes quietly wrong in most implementations you'll read.
`htu` has to match the URL the client thinks it called, but behind a load
balancer `r.Host` and `r.TLS` describe the hop. Get it wrong in one direction
and every proof fails in production, which is loud and survivable. Get it wrong
the other way and you accept proofs minted for a different URL, which defeats
the point. We already solved the shape of this once: `middleware.ClientIP`
honours `X-Forwarded-For` only when the immediate peer is a trusted proxy.
`htu` reconstruction reuses that same trusted-proxy configuration for
`X-Forwarded-Proto` and `X-Forwarded-Host`. There's one notion of trusted proxy
in the codebase and this doesn't add a second.

## Configuration

Global and per-app go through the settings manager, which already resolves
global to app and renders dashboard UI:

```go
SettingDPoPMode = settings.Define("dpop.mode", "off",
    settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
    settings.WithInputType(formconfig.FieldSelect),
    settings.WithOptions(off, optional, required), ...)
SettingDPoPNonceRequired   = settings.Define("dpop.nonce_required", false, ...)
SettingDPoPIatLeewayPast   = settings.Define("dpop.iat_leeway_past_seconds", 60, ...)
SettingDPoPIatLeewayFuture = settings.Define("dpop.iat_leeway_future_seconds", 30, ...)
```

Per-client is `OAuth2Client.DPoPMode string`, added by an `oauth2provider`
migration for Postgres and SQLite. Mongo needs the struct mapping only. Empty
string means inherit, matching how `TokenFormat` already works on
`appsessionconfig`.

Modes: `off` ignores proofs entirely, `optional` binds when the client proves
and issues a bearer token when it doesn't, `required` refuses to issue anything
unbound.

Resolution is monotonic. Order the modes `off < optional < required`, then:

```
effective = max(appMode, clientMode)
```

A client cannot configure itself below its app. This is the one place worth
being deliberately inflexible, because when you set an app to `required` you
are stating a mandate for every client underneath it, and a per-client field
with the power to quietly undo that would turn the strongest setting in the
system into a suggestion any client operator is free to decline. If you have a
legacy client that cannot cope, move the app back to `optional`. That is
explicit, visible and audited.

## Binding and storage

`session.Session` gains one field:

```go
// DPoPJKT is the RFC 7638 thumbprint of the public key this session is
// bound to (RFC 9449). Empty means an unbound bearer session.
DPoPJKT string `json:"dpop_jkt,omitempty"`
```

It sits in the JSON and is not hidden. A thumbprint is public by construction,
and "which of my sessions are bound?" is exactly the question you want a
session list to answer.

The migration follows `add_session_roles` in `store/postgres/migrations.go`:
`TEXT NOT NULL DEFAULT ''`. Every existing row backfills to unbound and
authorises what it authorised yesterday. No index, since lookups are still by
token.

JWT-format tokens need the binding inside the token as well, because a
stateless validator has no session row to consult and cannot enforce anything
it cannot read. `tokenformat.TokenClaims` gains `DPoPJKT`, and `customClaims`
gains `Confirmation *Confirmation` serialised as `cnf`, holding the standard
`{"jkt": "..."}`.

Issuance touches `issueTokens` and `issueClientToken` in the OAuth2 plugin,
which between them cover the authorization code, client credentials and device
code grants, plus the first-party sign-in handler. Same shape in each:

- Proof present and mode isn't `off`: validate against
  `Expectation{Method: POST, URL: <token endpoint>, ExpectedJKT: ""}`, then set
  `sess.DPoPJKT`. `ExpectedJKT` is empty because at issuance we learn the key
  instead of checking it.
- Mode is `required` and no proof: `400 invalid_dpop_proof`.
- Mode is `optional` and no proof: issue an ordinary bearer session, unchanged.

`token_type` in the response becomes `"DPoP"` when the token is bound. It's
easy to miss, and skipping it quietly breaks every spec-compliant client you
have, because that field is how a client decides whether to attach proofs at
all.

`Engine.Refresh` takes a DPoP expectation through `RefreshOpts`. When
`sess.DPoPJKT` is non-empty the refresh must carry a valid proof for that
thumbprint, and the rotated session inherits it.

That refresh path already has an elaborate replay cascade: hash the presented
token, check the revoked set, `MarkRefreshTokenReplayed` as an atomic
conditional upgrade, revoke the `FamilyID`, alert once. DPoP doesn't replace
it, it changes what it means. Today the cascade fires after a leaked token has
already been redeemed once, so what you get is detection and containment after
the fact, whereas with binding an attacker holding the token but not the key
cannot redeem it at all, and the cascade stops being your primary defence and
becomes a tripwire.

## Middleware enforcement

`extractBearerToken` in `middleware/auth.go` becomes:

```go
func extractCredential(r *http.Request, cookieName string) (scheme, token string)
// "dpop" | "bearer" | "cookie" | ""
```

All three middleware variants call it, and all three call a shared
`enforceDPoP` at one point: right after the existing IP and device binding
checks, before context population.

The rule:

```
boundJKT := sess.DPoPJKT      (opaque)  or  claims.cnf.jkt  (JWT)

boundJKT == ""  ->  no requirement. Proceed exactly as today.
boundJKT != ""  ->  scheme must be "dpop", proof must be present, valid,
                    and JKT(proof.jwk) == boundJKT, with ath over the
                    presented token. Otherwise 401.
```

That first branch is the whole migration story. An unbound token costs one
string comparison against the empty string and then takes the identical path
through the middleware it takes today, with the same session lookup, the same
context population and the same downstream behaviour, which is what lets you
roll this out across a live fleet without a flag day.

Scheme matching is strict for credentials that arrive in the `Authorization`
header. A bound token presented as `Authorization: Bearer` is rejected even
with a valid `DPoP` header alongside it. RFC 9449 section 7.1 specifies the
`DPoP` scheme, and since bound tokens only exist after somebody opts in,
strictness costs no compatibility.

Three edges worth naming:

- Cookie transport. Enforcement follows the token, not the transport, so a
  bound session in a cookie still needs a proof. What it does not need is the
  `DPoP` authorization scheme: section 7.1 is a rule about the `Authorization`
  header, a cookie is not one, and a browser cannot set a scheme on a cookie.
  So a bound session presented by cookie is honoured when a valid proof
  accompanies it and refused when one does not, with `ath` over the session
  token exactly as on the header path.

  This used to say the case never fired, on the grounds that browser sign-in
  never binds. That stopped being true when `handleSignIn` and `handleSignUp`
  started resolving a binding through `dpopBindingForRequest`: under
  `mode=required` every first-party sign-in now mints a bound session and sets
  a cookie for it. The engine's cookie-to-header bridge rewrites that cookie
  into `Authorization: Bearer <token>` before the inner middleware runs, so
  strict scheme matching turned every such session into a permanent 401. The
  bridge records the value it wrote, and the enforcement path treats a bridged
  cookie as the cookie it is.

  A client that only holds an `HttpOnly` cookie and never sees its own token
  cannot compute `ath` and so cannot use a bound session. That is not a gap:
  under `mode=required` the client already had to mint a proof to sign in, and
  the sign-in response hands it the session token.
- API keys with the `ask_` prefix go through `tryStrategyAuth` and are
  untouched. They are not issued by a token endpoint and carry no `cnf`.
- `ath` is mandatory at a protected resource. A proof valid in every other
  respect but missing `ath` is rejected, because without it a proof captured
  from one request replays against any other endpoint.

`AuthMiddleware` is normally non-blocking, passing unresolvable tokens through
so the route can decide. It already breaks that convention for IP and device
mismatches, returning `forge.Unauthorized` outright, and DPoP inherits the same
treatment. Pass-through means "I could not identify you, maybe this route is
public". A binding violation means "I identified you and the binding failed",
which is a positive signal that something is wrong, and degrading that to
anonymous would hand an attacker a way to strip the proof off a bound token and
fall through to any endpoint in your API that tolerates anonymous access.

## The other two paths to an identity

`middleware/auth.go` is not the only place a token turns into an authenticated
request. Two other paths resolve a credential, and a rule that holds in one
place and not the others is not an invariant, so both apply the same check.

Client mode is the one that was genuinely open. A service running with
`WithClientMode` has no engine and no session table, so it validates tokens by
calling the identity server's `/v1/introspect` and using what comes back. The
introspection response now carries `cnf` with the `jkt` whenever the token is
bound, which is what RFC 9449 section 7.3 asks for. Without it the calling
service has no way to find out that the token in its hand is bound to a key,
and it accepts a stolen one. The client middleware reads that claim and runs
the same `enforceDPoP` the engine path runs. None of the checking moves to the
identity server, because the request never goes there.

So the service validates the proof itself, with its own `dpop.Validator` and
its own replay cache, built by the extension on first use. It mints no tokens,
so it has no nonce secret and never demands a nonce. If you assemble the
middleware yourself and pass no validator, a bound token is refused rather than
admitted, matching what the engine path does when its validator is missing.

The second path is `authprovider.SessionProvider`, registered with the forge
auth registry and reached by plugin authz, organization, waitlist and consent.
In a normal deployment the global middleware rejects a bad presentation long
before this runs, so it was closed in practice. It was closed by accident of
chain ordering, though, and it opens the moment somebody assembles a chain
without the global middleware. The provider now checks the binding itself. It
answers with an auth context rather than a response, so it cannot write an RFC
9449 challenge and returns `ErrInvalidCredentials` instead. The client learns
less from that than it would from the middleware, which is a reason to keep the
middleware in front of it.

All three call one implementation. `checkDPoP` holds the rule and touches no
response, `enforceDPoP` wraps it for the middleware and writes the challenge,
and `EnforceDPoPForRequest` wraps it for callers that only get to say yes or
no. Two copies of a rule this important would drift, and the copy nobody reads
is the one that drifts first.

## Nonce

Stateless HMAC, built the same way as `dashboard/nonce.go`, keyed from
`Engine.NonceSecret()` with its own info string `authsome:dpop:nonce-v1`. Same
domain separation the dashboard nonce already uses. The value is `b64(ts) "."
b64(HMAC(secret, ts || jkt))`, bound to the thumbprint so a nonce minted for
one client is useless to another.

`Engine.NonceSecret()` returns nil when there is no HMAC JWT key and no
`AUTHSOME_DASHBOARD_NONCE_SECRET`, which the dashboard handles by logging a
warning and leaving its signer uninitialised. DPoP must not copy that. If an
app has nonces switched on and no secret can be derived, that is a
misconfiguration and it has to fail closed with a clear error.

Where it fails is the setting, not startup. `initDPoP` runs before any app has
resolved `dpop.nonce_required`, so refusing to start there would refuse every
deployment that has no HMAC JWT key, and almost none of those asked for nonces.
So `initDPoP` logs a warning naming what would break and carries on, and
`Engine.DPoPNonceRequiredForApp` answers true when it finds the setting on with
no signer behind it, logging an error that names the app. Every DPoP request
for that app is then refused, which is loud and takes one config change to fix.
Answering false instead would leave the control switched on in the dashboard,
switched off in reality, and nothing anywhere saying so.

The alternative to both is a per-process random secret, which mints nonces no
other instance can verify and turns a security feature into an intermittent
outage that only shows up once you scale past one replica.

One thing diverges sharply from the file we are copying, and it wants a comment
in the code because the neighbour invites the mistake. **DPoP nonces are not
single use.** `nonceSigner.Consume` marks a nonce used and refuses replays,
which is exactly wrong here, because a client legitimately reuses a single
nonce for every request it makes during that nonce's lifetime. We reuse the
construction, not the type. Call `Consume` and the feature breaks on the second
request.

Rotation is soft. Past half the TTL, successful responses carry a fresh
`DPoP-Nonce` and the client swaps it in. No flag day.

Nonces are off by default and switch on per app, so nobody pays the extra round
trip until they want the guarantee.

## Errors

| Where | Status | Response |
|---|---|---|
| Token endpoint, nonce needed | 400 | `{"error":"use_dpop_nonce"}` plus `DPoP-Nonce` |
| Resource, nonce needed | 401 | `WWW-Authenticate: DPoP error="use_dpop_nonce"` plus `DPoP-Nonce` |
| Resource, bad or absent proof | 401 | `WWW-Authenticate: DPoP error="invalid_token", algs="ES256 RS256 PS256 EdDSA"` |
| Token endpoint, bad proof | 400 | `{"error":"invalid_dpop_proof"}` |

Three new audit actions, because the signal differs a lot between them.
`dpop_proof_invalid` is noisy and usually means a broken client.
`dpop_proof_replayed` is rare and interesting. `dpop_key_mismatch` is a
structurally valid proof for the wrong key, which has no benign explanation the
way an IP change does, so it goes out at `SeverityWarning` next to
`refresh_token_replayed`.

## Metadata

`DiscoveryResponse` gains `dpop_signing_alg_values_supported` carrying the four
algorithms, advertised unconditionally. `handleDiscovery` is not app scoped and
has no way to know the caller's app, and a server that can validate ES256
proofs supports ES256 whoever is asking. Per-client mode is discovered at
registration.

Known gap: only `/.well-known/openid-configuration` exists here, and RFC 8414's
`/.well-known/oauth-authorization-server` does not, so a client that goes
looking in the place RFC 8414 tells it to look will not find any of this.
Adding that endpoint is RFC 8414 work and belongs in its own change.

## TypeScript SDK

`sdk/typescript/src/client.ts` is generated. It says so on line one. The
template it comes from is `sdkgen/typescript/templates/client.ts.tmpl` and it
carries two `request<T>` implementations, both hardcoding `Bearer`, so the
edits go to the template.

The crypto does not go in the template. Two hundred lines of WebCrypto inside a
Go `text/template` is unmaintainable and impossible to test on its own. Add a
mostly-literal `dpop.ts.tmpl` rendering to `src/dpop.ts`, which costs four
lines in the explicit template list in `sdkgen/typescript/generator.go`, and
have the client template import it. The crypto stays ordinary TypeScript with
ordinary tests.

What `src/dpop.ts` does:

- `generateKey({name: "ECDSA", namedCurve: "P-256"}, false, ["sign"])`. That
  `false` is the entire point. Non-extractable means an XSS on your page walks
  away with a token it has no way to use.
- Stores the key in IndexedDB. This is forced, not preferred. A `CryptoKey` is
  structured-cloneable but not serialisable, so localStorage physically cannot
  hold a non-extractable key handle, and if you reach for it anyway the only
  way to make the code compile is to mark the key extractable, which throws
  away the entire benefit you came for.
- Exports the public JWK with `exportKey("jwk", publicKey)`, which works fine
  while the private half stays sealed.
- Signs. WebCrypto hands back ES256 signatures as raw `R||S`, which is already
  JWS format, so there is no DER unwrapping to do.
- Computes `ath` as base64url(SHA-256(token)).
- Retries once on `use_dpop_nonce`. Once, not a loop. A server that's stuck
  re-challenging should surface as an error, and not as a hot loop hammering
  your own auth server.
- One key pair per origin, created lazily at first sign-in, shared across tabs
  through IndexedDB, destroyed on logout.

## Testing

Primitives:

- RFC 7638 Appendix A.1 vector for the thumbprint,
  `NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs`. An external oracle, not our
  own output fed back in.
- `jwkutil` round trip across EC, RSA and OKP.
- Validator: one table case per rejection branch in the fifteen steps, plus
  injected-clock cases sitting exactly on the 60 and 30 second boundaries.

Behaviour:

- Bound token with no proof, 401. Wrong key, 401. Right proof under the
  `Bearer` scheme, 401. Replayed `jti`, 401.
- Bound session refreshed without a proof is refused. With a proof, the rotated
  session keeps the same thumbprint.
- Nonce challenge, then success on retry.
- `jkt` persistence goes into `storetest.RunConformance` so all three drivers
  prove it. Mongo especially, since the recent empty-array-versus-null fix is a
  fresh reminder that its zero-value handling isn't something to assume.

The migration proof:

`middleware/auth_test.go`, `refresh_replay_test.go` and the OAuth2 suites have
to pass untouched. Treat that as the deliverable in its own right. If any
existing test needs editing to accommodate DPoP then the "unbound tokens are
unaffected" invariant has already been broken somewhere, and the diff to those
files is your alarm.

## Out of scope

- The Go and Dart SDKs. Each needs its own key storage design, keychain or
  secure enclave or encrypted file, with different threat models.
- RFC 8414 authorization server metadata, noted above.
- DPoP for API keys.
- RFC 7591 dynamic client registration, 8707 resource indicators and 8693 token
  exchange. Each has its own design dated today under
  `docs/superpowers/specs/`. DPoP composes with all three and depends on none
  of them, so they can land in any order. What they do share is tables: 8707
  and the agentauth delegation design both add columns to `authsome_sessions`,
  and 7591 and 8707 both add columns to `authsome_oauth2_clients`. Whoever
  implements second takes the union of the columns and a fresh migration
  version, because two migrations sharing a version inside one group fail at
  startup.
