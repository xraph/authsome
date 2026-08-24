# Auth security hardening — design

**Status:** draft
**Date:** 2026-05-01
**Owner:** Rex Raphael

## Context

Recent feature work (social plugin frontend/redirect URL state, dashboard org delete, members-tab user resolution) shipped without a corresponding security pass. A two-pronged audit found 6 immediately exploitable issues plus 12 layered-defense gaps. This spec covers a defense-in-depth response across the auth surface — the "Slice C" scope agreed during brainstorming.

The work is split into six phases that can ship as independent PRs. Each phase stands alone (no dependencies on later phases) so we can pause or reorder if priorities change.

## Goals

- Close every issue an attacker can exploit against today's deployment.
- Move from "trust the caller" to "trust an explicit allowlist" in OAuth/redirect handling.
- Bind every CSRF-sensitive form to a session, statelessly.
- Make destructive actions transactional and audited.
- Encrypt OAuth provider tokens at rest behind a pluggable interface so production can drop in a real KMS later.
- Sign outbound webhooks; rotate SCIM and session secrets.
- Tests are the deliverable, not a side-effect: every audit finding closes with a dedicated regression test (TDD; failing first) and an attack-replay test where the threat is exploitable. See "Testing strategy" below for the discipline, types, coverage gates, and CI requirements.

## Non-goals

- Replacing the dashboard auth middleware (we'll lean on it; not redesign it).
- Migrating to a different store, ORM, or session backend.
- Multi-region/HA hardening of the in-memory ceremony store (we namespace state by app and document the recommendation; full HA is out of scope).
- Bug-bounty or pen-test-driven discovery — this scope is the audit findings only.

## Threat model summary

Three attacker classes informed scope:

1. **Unauthenticated external attacker** — phishing redirects, OAuth code interception, webhook spoofing.
2. **Logged-in low-privilege user** — privilege escalation via destructive actions on resources they don't own.
3. **External site loaded by an admin** — CSRF against destructive dashboard actions.

DB-compromise scenarios are partially in scope (token encryption at rest); host compromise and side-channel attacks are not.

## Testing strategy

Tests are not an afterthought for this work — they are the deliverable. A security fix without a test that proves the threat is blocked is not done.

### Discipline

- **TDD per item.** Every numbered design item below ships with its tests written *before* the implementation. The expected workflow per item: (1) write the failing test that exercises the vulnerable code path or threat, (2) confirm the test fails for the right reason on `main`, (3) implement the fix, (4) confirm the test now passes. The `superpowers:test-driven-development` skill applies.
- **No green-without-failing-first.** A test that has never failed is not a test, it's a fixture. PR review checks that each new test was committed in a state that fails against the unfixed code.
- **One test per finding minimum.** Every audit finding (C1–C6, H1–H4, plus the medium/low items folded into phases 5/6) gets at least one dedicated test named after the finding so future regressions are easy to attribute.

### Test types per phase

Each phase below specifies which of these apply; this is the menu:

| Type | Purpose |
|---|---|
| **Unit** | Pure-function behaviour: `sanitizeRedirectURL`, nonce HMAC verify, AES-GCM round-trip, webhook signature compute, etc. Table-driven Go tests in the same package. |
| **Integration** | Multi-component flows through a real store (memory backend by default, plus at least one SQL backend for the destructive flows): org-delete cascade rollback, OAuth state across handleStart→handleCallback, SCIM token rotation. |
| **Attack-replay** | Each critical finding (C1–C6) ships with a test named `TestAttack_<finding>` that constructs the original exploit (e.g. POSTs a hostile `redirect_url` with no Origin header) and asserts the request is refused. These are the regression sentinels. |
| **Fuzz** | `sanitizeRedirectURL`, `sanitizeFrontendURL`, nonce verify, and the new `Encryptor.Decrypt` get Go fuzz harnesses (`go test -fuzz`). 5-minute soak in CI per fuzz target on a nightly job. |
| **Property** | Where it fits cheaply (HMAC round-trip, encryption round-trip with random AAD), use `testing/quick` for property-based assertions. |
| **Benchmark** | Phase 4 batch lookup and the nonce verify hot path get `Benchmark*` tests so we catch perf regressions. |

### Coverage gates

- Touched packages must end at **≥ 80% statement coverage** for the files modified in this work (measured per-file, not per-package, so unrelated old code doesn't dilute the signal).
- Every new exported function from this work has at least one test covering both happy path and at least one error path.
- The five files most central to the audit (`plugins/social/plugin.go`, `dashboard/nonce.go`, `plugins/organization/{service,dashboard}.go`, `bridge/aesgcm.go`) target **≥ 90%** statement coverage.
- Coverage is enforced in CI via `go test -coverprofile=cover.out` + a small script that fails the build if any of the listed files drops below the threshold.

### Standing test infrastructure to build once, reuse

These helpers live in a new `internal/secutil` (or similar) package and are shared across all phase tests:

- `secutil.NewTestEngine(t, opts...)` — spins up an authsome engine backed by the memory store with a known fixed encryption key, deterministic nonce secret, and the social plugin pre-registered with a fake provider. Cleanup via `t.Cleanup`.
- `secutil.FakeOAuthProvider` — implements the `social.Provider` interface; lets a test drive `OAuth2Config()`, `FetchUser()`, and assert which `redirect_uri` and PKCE params were passed.
- `secutil.AttackRequest(t, method, path, body, opts...)` — request builder that defaults to *no* `Origin`/`Referer`/`Cookie` so attack-replay tests can't accidentally pass because the test rig sent a friendly origin.
- `secutil.AssertAuditEvent(t, chronicle, action, expected)` — drains a buffered chronicle and asserts the right `bridge.AuditEvent` was recorded.
- `secutil.AssertNoAuditEvent(t, chronicle, action)` — for negative paths (e.g. rejected delete must NOT log a successful-delete event).

Building this infra is part of phase 1's deliverables (so subsequent phases can lean on it).

### CI

- `go test ./...` plus `go test -race ./...` on every PR.
- Nightly: `go test -fuzz=Fuzz... -fuzztime=5m` per fuzz target.
- Per-PR: coverage gate script.
- Failures in any of the above block merge.

## Phases

Each phase lists items, the design for each, files touched, and verification (with explicit test names).

---

### Phase 1 — Stop the bleeding (immediately exploitable)

**Goal:** ship one PR that closes every audit finding flagged "exploitable today."

#### 1.1 Open redirect in `sanitizeRedirectURL`

When `requestOrigin` is empty, the current implementation lets any absolute URL through ([plugins/social/plugin.go:1027](../../../plugins/social/plugin.go)). An attacker calling `/v1/social/google` server-side with no `Origin`/`Referer` and a hostile `redirect_url` gets redirected post-callback to their domain.

**Design:**
- Reject absolute URLs when no trusted origin is available. Allow only relative paths in that case.
- The "trusted origin" comes from (in order): the app's allowlist (1.2), the caller's `frontend_url` *if it's on the allowlist*, the `Origin`/`Referer` headers *if they match the allowlist*. Falling off all three → relative paths only.

**Files:** plugins/social/plugin.go (rewrite of `sanitizeRedirectURL` + `sanitizeFrontendURL`).

#### 1.2 Per-app frontend-URL allowlist

Caller-supplied `frontend_url` is currently the trust authority with no validation. Add an app-scoped dynamic setting `auth.allowed_frontend_urls` (CSV of origins, e.g. `https://app.example.com,https://staging.example.com`).

**Design:**
- Register the setting in `plugins/social` (or a new `auth-core` plugin if shared) using the existing `settings.Define` API. Type: string CSV. Scope: app-level. Default: empty (no allowlist → reject all absolute redirects, allow only relative).
- New helper `isAllowedOrigin(appID, candidateURL) bool` — parses the candidate, looks up the setting via `p.settings.Get(ctx, ResolveOpts{AppID: appID})`, splits CSV, matches host case-insensitively (no path).
- `handleStart` sanitises `frontend_url` and `redirect_url` against the app's allowlist; both are rejected if the host isn't listed (with relative-path fallback for `redirect_url`).
- Dashboard UI: surface the setting in the existing settings editor (no custom UI needed).

**Files:** plugins/social/plugin.go, plugins/social/settings.go (new), plugins/social/plugin_test.go.

#### 1.3 HMAC-signed dashboard nonce bound to session

Replace `dashboard/nonce.go`'s global map with a stateless HMAC token.

**Design:**
- New shape: `nonce = base64url(timestamp || HMAC-SHA256(serverSecret, sessionID || actionScope || timestamp))`.
- `GenerateNonce(sessionID, scope string) string` — emits the token.
- `ConsumeNonce(sessionID, scope, nonceStr string) bool` — verifies HMAC, checks timestamp ≤ 10 min old, **and** rejects replays via a small in-memory replay-set (`map[sha256(nonce)]expiry`, cleaned on each call). The replay-set is per-instance; cross-replica replay within the same 10-min window is acceptable risk for now (documented).
- `serverSecret`: pulled from a new top-level `dashboard.NonceSecret []byte` plumbed via the engine config (default: derived from app's signing key on first init, persisted in `app_client_configs.metadata`).
- All call sites (`handleDashboardCreateOrg`, org delete, user delete, user update, user ban, role changes, etc.) updated to pass the session ID + a string scope ("org.delete", "user.delete", etc.).

**Files:** dashboard/nonce.go (rewrite), dashboard/contributor.go (call sites), plugins/organization/dashboard.go (call sites), plus templ files that include nonces.

**Migration:** old in-flight nonces become invalid on deploy → users see "form expired" and re-submit. Acceptable.

#### 1.4 Authz check on org delete

`renderOrgDetail` action="delete" calls `p.DeleteOrganization` after consuming the nonce, but never verifies the caller can delete this specific org.

**Design:**
- Resolve caller's user ID from the dashboard auth context (`dashboard.UserIDFromContext`).
- Allow delete if (a) caller is `org.CreatedBy` or has `Owner` role on the org, OR (b) caller is an app-level admin (re-use existing `engine.HasPermission(ctx, userID, "org.delete", orgID.String())` if available; otherwise look up the member record and check role).
- On reject: render the org detail with `data.Error = "You don't have permission to delete this organization."` instead of deleting.
- Same authz pattern applied to other newly-shipped destructive paths discovered during implementation.

**Files:** plugins/organization/dashboard.go, plugins/organization/service.go (authz helper).

#### 1.5 Transactional cascade delete

Wrap the cascade in a single store transaction. If any step fails, the whole thing rolls back and `EmitAfterOrgDelete` is **not** called.

**Design:**
- Add `WithTx(ctx, fn func(tx Store) error) error` to `organization.Store` if not already present (postgres: `pg.RunInTx`; sqlite: `sdb.RunInTx`; mongo: session txn; memory: a no-op snapshot/rollback).
- If the underlying composite store doesn't support transactions, the in-service implementation falls back to best-effort with a warning log — no behaviour change for the memory store, but real backends always run in a tx.
- After commit: `EmitAfterOrgDelete` runs in the original (non-tx) ctx so external side-effects (subscription cancel, SCIM cleanup) aren't rolled back if they fail. Their own failures are already best-effort.

**Files:** organization/store.go, store/{postgres,sqlite,memory,mongo}/store.go, plugins/organization/service.go.

#### 1.6 Audit log destructive actions

Every destructive dashboard action records a chronicle event before the action runs. The infrastructure exists already: `bridge.Chronicle` interface with `Record(ctx, *bridge.AuditEvent)` ([bridge/chronicle.go:13](../../../bridge/chronicle.go)) and `bridge.AuditEvent` shape with `Action`, `Severity`, `ActorID`, `ResourceID`, `Metadata` fields. We just need to call it consistently.

**Design:**
- `chronicle.Record(ctx, &bridge.AuditEvent{Action: "org.delete", ResourceID: orgID.String(), ActorID: callerUserID.String(), Severity: bridge.SeverityCritical, Metadata: map[string]string{"slug": org.Slug, "app_id": appID.String()}})` immediately before the cascade.
- Same for: user delete, user ban/unban, user update, role change, app delete (when added), and any future destructive flow.
- A small helper `(c *Contributor).recordAudit(...)` plus a per-plugin equivalent so call sites are one line.

**Files:** dashboard/audit.go (new helper), call sites across dashboard/contributor.go and plugins/organization/dashboard.go.

#### Phase 1 verification

**Unit tests** (`plugins/social/sanitize_test.go`, table-driven):

- `TestSanitizeRedirectURL_NoOriginRejectsAbsolute` — empty origin + absolute URL returns `""`.
- `TestSanitizeRedirectURL_NoOriginAllowsRelative` — empty origin + `/dashboard` returns `/dashboard`.
- `TestSanitizeRedirectURL_AllowlistMatch` — host on allowlist passes.
- `TestSanitizeRedirectURL_AllowlistMismatch` — host not on allowlist returns `""`.
- `TestSanitizeRedirectURL_SchemeInjection` — `javascript:`, `data:`, `file:` all rejected.
- `TestSanitizeRedirectURL_CredentialsRejected` — `https://user:pass@host/` rejected.
- `TestSanitizeRedirectURL_CaseInsensitiveHost`, `_PortPreserved`, `_TrailingSlashIgnored`, `_IDNHomograph` (Punycode shenanigans).
- `TestSanitizeFrontendURL_RequiresAbsoluteHTTP` — relative paths and non-http schemes rejected.
- `FuzzSanitizeRedirectURL`, `FuzzSanitizeFrontendURL` — never panic, never return a URL with a scheme other than `http`/`https` or starting with `/`.

**Attack-replay test** (`plugins/social/plugin_test.go`):

- `TestAttack_OpenRedirect_NoOrigin` — POST `/v1/social/google` with no Origin/Referer header and `redirect_url=https://attacker.example`. Assert the response does NOT contain `attacker.example` in the OAuth `state` lookup or the eventual redirect.
- `TestAttack_FrontendURL_NotAllowlisted` — POST with `frontend_url=https://attacker.example` against an app whose `auth.allowed_frontend_urls` doesn't list it. Assert `frontend_url` in the stored state is empty.

**Unit tests for nonce** (`dashboard/nonce_test.go`):

- `TestNonce_ValidRoundTrip`, `TestNonce_Replay` (second consume returns false), `TestNonce_WrongSession`, `TestNonce_WrongScope`, `TestNonce_Expired` (>10 min), `TestNonce_TamperedHMAC`, `TestNonce_ConcurrentConsume` (race on `t -race`), `TestNonce_TimestampInFuture` (clock skew tolerance).
- `FuzzNonceVerify` — never panic on malformed input, never return true on random bytes.

**Attack-replay** (`dashboard/nonce_test.go`):

- `TestAttack_CSRF_StolenNonce` — generate a nonce as user A, attempt to consume as user B → must fail.

**Integration test** (`plugins/organization/dashboard_test.go` — new file):

- `TestOrgDelete_RequiresOwner` — non-owner caller hits the delete handler → assert org still exists, no audit event recorded, response surfaces error.
- `TestOrgDelete_OwnerSucceeds` — owner caller → org gone, audit event recorded, AfterOrgDelete hook fired.
- `TestOrgDelete_AppAdminSucceeds` — non-member with app-admin permission → succeeds.
- `TestOrgDelete_TransactionalRollback` — inject a failure into `DeleteTeam` (test double) → assert org/members/invitations are still present and `EmitAfterOrgDelete` was NOT called. Use the secutil helpers to assert "no audit event for completion" while a "delete attempted" event may exist.
- `TestOrgDelete_AuditEventShape` — successful delete records `Action="org.delete"`, `Severity=Critical`, `ActorID` and `ResourceID` populated, metadata includes `slug` and `app_id`.

**Attack-replay** (cross-cutting):

- `TestAttack_OrgDelete_CSRFForgedPost` — simulate a forged POST from a hostile origin against an admin's session cookie; nonce check must reject (validates both 1.3 and 1.4 together).

---

### Phase 2 — OAuth flow hardening

#### 2.1 PKCE (S256)

**Design:**
- Generate a 32-byte verifier in `handleStart`, derive `code_challenge = base64url(sha256(verifier))`.
- Store `code_verifier` in the OAuth state alongside `frontend_url`/`redirect_url`.
- Pass `code_challenge` and `code_challenge_method=S256` as extra `oauth2.AuthCodeOption`s in `AuthCodeURL`.
- On callback, pass `oauth2.SetAuthURLParam("code_verifier", verifier)` to `Exchange`.
- Providers that don't support PKCE silently ignore the extra params (RFC 7636 §4.4) — no provider-specific gating needed.

#### 2.2 OIDC nonce

For providers known to issue OIDC ID tokens (Google, Apple, Microsoft):
- Generate a random `nonce`, include in `AuthCodeURL`, store in state.
- After token exchange, decode the ID token (no signature verification yet — that's a separate hardening item) and confirm the `nonce` claim matches.
- If a provider doesn't return an ID token, skip the check (logged at debug level).

#### 2.3 State key namespacing by app

**Design:**
- Change ceremony key from `social:state:<state>` to `social:state:<appID>:<state>`.
- Callback validates: read app from request scope (or default), build the key, fetch. State that was stored under app A can't be read in app B's callback.
- Migration: deploys overlap; old in-flight states (10-min TTL) become unreadable → user sees "invalid state" and re-tries. Acceptable.

#### 2.4 Rate limiting `/v1/social/*`

**Design:**
- Use existing forge security extension's rate limiter middleware.
- `POST /v1/social/:provider`: 10 req / min / IP, 60 / hour / IP.
- `GET /v1/social/:provider/callback`: 30 / min / IP (legitimate callbacks may retry on transient browser issues).
- Configurable per-app via dynamic settings for tuning.

#### Phase 2 verification

**PKCE** (`plugins/social/pkce_test.go`):

- `TestPKCE_VerifierRoundTripsThroughState` — verifier stored in state during `handleStart`, retrieved during callback.
- `TestPKCE_ChallengeMethodIsS256` — assert `code_challenge_method=S256` in the auth URL query.
- `TestPKCE_ChallengeMatchesVerifier` — assert `base64url(sha256(verifier)) == challenge`.
- `TestPKCE_ExchangeIncludesVerifier` — fake provider asserts `code_verifier` is sent on the token exchange request.
- `TestPKCE_VerifierEntropy` — generated verifier is at least 43 bytes base64url, distinct across 1000 iterations.

**Attack-replay**:

- `TestAttack_CodeInterception_RequiresVerifier` — fake provider rejects exchange that's missing `code_verifier`; assert authsome's exchange would fail (i.e. the provider check actually triggers).

**OIDC nonce** (`plugins/social/oidc_test.go`):

- `TestOIDCNonce_StoredInState`, `TestOIDCNonce_MatchesIDTokenClaim`, `TestOIDCNonce_MismatchRejected`, `TestOIDCNonce_MissingFromIDTokenWhenExpected` — synthesize ID tokens (header.payload.signature) with controlled `nonce` claims; we don't yet verify the signature so this is a JSON-shape test.
- `TestOIDCNonce_AbsentForNonOIDCProvider` — provider that doesn't issue ID tokens skips the check without erroring.

**State namespace** (`plugins/social/state_namespace_test.go`):

- `TestState_KeyNamespacedByApp` — state stored under app A is unreadable when callback resolves to app B.
- `TestAttack_StateCrossTenantReplay` — start flow on app A, attempt to consume the state on app B's callback URL → BadRequest.

**Rate limiting** (`plugins/social/ratelimit_test.go` — integration via `httptest`):

- `TestRateLimit_StartEndpoint_60PerMinPerIP` — 11th request from same IP in 60s window returns 429.
- `TestRateLimit_DifferentIPs_Independent` — IP A's bucket doesn't drain IP B's quota.
- `TestRateLimit_CallbackTolerant` — callback bucket accepts up to 30/min (browser retry budget).
- `TestRateLimit_PerAppTuning` — overriding the app setting raises/lowers the limit.

---

### Phase 3 — Token encryption at rest

#### 3.1 `bridge.Encryptor` interface + AES-GCM default

**Design:**

```go
// bridge/encryptor.go
type Encryptor interface {
    Encrypt(ctx context.Context, plaintext []byte, aad []byte) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte, aad []byte) ([]byte, error)
    KeyID() string  // for envelope/rotation
}
```

- Default impl `bridge/aesgcm.go`: 256-bit key from env `AUTHSOME_ENCRYPTION_KEY` (base64). Random 12-byte nonce prefixed to ciphertext. AAD = `appID || userID || tokenType` for domain separation.
- Engine init: read env, construct `aesgcmEncryptor`, expose via `engine.Encryptor()`.
- Production drops in a KMS-backed implementation by setting `engine.SetEncryptor(kms.New(...))` before start.
- Dev mode (key absent): warns loudly and uses a process-local random key — won't decrypt across restarts. Test fixtures use a known fixed key.

#### 3.2 Encrypt OAuth provider tokens

**Design:**
- `OAuthConnection.AccessToken` and `RefreshToken` fields stay as `string` in the model but storage layer encrypts on write, decrypts on read.
- Store layer: implement `setOAuthTokens(ctx, conn, plaintextAccess, plaintextRefresh)` and `getOAuthTokens(...)` helpers; the public Connection struct exposes plaintext only after decryption.
- AAD includes `userID || provider || "oauth"` to bind ciphertext to its row — moving a row in the DB invalidates decryption.
- Migration: add nullable `access_token_v2` and `refresh_token_v2` BLOB columns. Backfill job re-encrypts existing rows on first read (lazy). After 30-day grace, drop the v1 columns.

#### Phase 3 verification

**Unit tests** (`bridge/aesgcm_test.go`):

- `TestAESGCM_RoundTrip` — encrypt then decrypt returns the original bytes.
- `TestAESGCM_AADBindsCiphertext` — encrypting with AAD `A` then decrypting with AAD `B` fails.
- `TestAESGCM_TamperedCiphertextFails` — flipping any byte of the ciphertext (or nonce prefix or tag) makes decrypt fail.
- `TestAESGCM_NonceUniqueness` — 100k encrypts → no nonce collision (Go's `crypto/rand`).
- `TestAESGCM_KeyIDStable` — `KeyID()` returns a stable value derived from the key (e.g. first 8 bytes of `sha256(key)` hex).
- `TestAESGCM_KeyMissingDevWarning` — constructor with empty key in dev mode logs a warning and uses a process-local key (with a marker `KeyID()` that prod code can detect).
- `TestAESGCM_KeyMissingProdRefused` — constructor with empty key + `RequireEncryption=true` returns an error.
- `FuzzAESGCMDecrypt` — random ciphertext+AAD inputs never panic, always cleanly error.
- `TestAESGCM_RoundTripProperty` — `testing/quick` property: for any plaintext + AAD, `Decrypt(Encrypt(p, aad), aad) == p`.

**Integration tests** (`plugins/social/encryption_test.go`):

- `TestOAuthConnection_TokensEncryptedAtRest` — create a connection, query the raw DB column, assert the bytes are not the plaintext access/refresh token.
- `TestOAuthConnection_TokensRoundTrip` — create then retrieve via the public API; plaintext matches.
- `TestOAuthConnection_AADBoundToRow` — manually swap the ciphertext between two rows; both reads must fail with a decryption error (proves the AAD includes row identity).
- `TestOAuthConnection_LazyMigration` — insert a v1 (plaintext) row, retrieve, assert the row was upgraded in-place to v2 (ciphertext).
- `TestOAuthConnection_FeatureFlagOff` — with `AUTHSOME_ENCRYPT_OAUTH_TOKENS=false`, behaviour is the existing plaintext path (regression guard during rollout).

---

### Phase 4 — DoS + integrity

#### 4.1 Batch user lookup

Add `engine.GetUsers(ctx, []id.UserID) (map[id.UserID]*user.User, error)` and a corresponding store method. Replace `loadMemberUsers`'s loop with a single batched call.

**Design:**
- Postgres/sqlite: `WHERE id IN (...)` with chunking at 1000 IDs.
- Memory/mongo: equivalent.
- Errors on individual rows are tolerated (partial map returned + warning log).

#### 4.2 Webhook HMAC signing

**Design:**
- `bridge.Webhook` (or webhook delivery worker) signs the JSON body: `X-Authsome-Signature: sha256=<hex>` where the HMAC key is per-webhook `Webhook.Secret` (already exists on the model).
- Add `X-Authsome-Timestamp: <unix>` and include the timestamp in the signed payload (`timestamp.body`) to prevent replay.
- Receiver-side helper in `sdk/go/webhook_verify.go`: `Verify(secret, body, sigHeader, tsHeader, maxAge time.Duration) error`.
- Existing webhooks without a secret get a generated one on first delivery and the dashboard surfaces it.

#### Phase 4 verification

**Batch user lookup** (`service_test.go` plus per-store backend tests):

- `TestGetUsers_Empty` — empty input returns empty map, no DB call.
- `TestGetUsers_Mixed` — request 5 IDs, 3 exist in DB → returned map has exactly those 3 entries.
- `TestGetUsers_Chunking` — request 2500 IDs, assert at most 3 underlying queries (chunk size 1000).
- `TestGetUsers_Deduplication` — duplicate IDs in input collapse to one DB lookup.
- `BenchmarkLoadMemberUsers_1000Members` — loadMemberUsers on a 1000-member org issues exactly one batch call (instrumented store counts queries).

**Regression guard**:

- `TestOrgDetail_DoesNotN1` — render org detail for a 1000-member org; assert query count ≤ 5 (org + members + teams + invitations + 1 batch user load).

**Webhook signing** (`webhook/sign_test.go` + `sdk/go/webhook_verify_test.go`):

- `TestWebhookSign_HeaderShape` — signature header matches the documented format (`sha256=<hex>` + timestamp).
- `TestWebhookSign_StableForSameInput` — same body+secret+timestamp produces same signature.
- `TestWebhookVerify_ValidSignaturePasses`.
- `TestWebhookVerify_TamperedBodyFails` — flip one byte in body → verify fails.
- `TestWebhookVerify_WrongSecretFails`.
- `TestWebhookVerify_ReplayBeyondMaxAgeFails` — `now - timestamp > maxAge` → verify fails.
- `TestWebhookVerify_FutureTimestampRejected` — clock-skew tolerance bounded.
- `TestWebhookVerify_ConstantTimeCompare` — assert signature compare uses `hmac.Equal` (statically, via a code reference) — covered by code review, not a runtime test.

**Attack-replay**:

- `TestAttack_WebhookReplay` — capture a valid delivery, replay it 24 hours later → receiver-side verify rejects.
- `TestAttack_WebhookForgery` — attacker without the secret tries to construct a delivery → verify rejects.

---

### Phase 5 — Secret rotation

#### 5.1 SCIM per-config secret + rotation UI

**Design:**
- `SCIMConfig` already has a `BearerToken` field. Add `BearerTokenHash` (SHA-256 hex). Store the hash; never persist plaintext after first creation (return it once on create, like API keys).
- Dashboard SCIM settings panel: "Rotate token" button → generates new token, displays once, stores new hash.
- Auth middleware compares `sha256(headerToken)` to the stored hash.
- Migration: existing plaintext tokens get hashed on first successful auth (lazy).

#### 5.2 Refresh-token rotation + replay detection

The codebase already has `account.RefreshSession` with a `RotateRefreshToken` flag plumbed through `engine.config.Session.ShouldRotateRefreshToken()` ([account/service.go:184](../../../account/service.go), [service.go:618](../../../service.go)). Two gaps:

1. Rotation isn't necessarily on by default for new app configs — confirm and force on.
2. There's no replay detection. Once a refresh token rotates, using the *old* token should not just fail silently — it should revoke the entire session family (RFC 6819 §5.2.2.3).

**Design:**
- Audit `Session.ShouldRotateRefreshToken` defaults; flip to true if not already.
- On rotation, store the old refresh-token hash in a `revoked_refresh_tokens` table (or a Redis set with TTL = original token lifetime). New table fields: `(token_hash, session_id, revoked_at)`.
- `engine.Refresh` checks the revoked set first; if hit, revoke the session AND emit an audit event (`token.replay_detected`, severity `critical`) so SOC tooling can alert.
- Add `(*Store).IsRefreshTokenRevoked(ctx, hash) (bool, error)` and `MarkRefreshTokenRevoked(ctx, hash, sessID, ttl) error` across all four backends.

#### Phase 5 verification

**SCIM rotation** (`plugins/scim/rotation_test.go`):

- `TestSCIMToken_HashedAtRest` — after `CreateConfig`, the DB row's `bearer_token_hash` is non-empty and the plaintext is NOT in any persisted column.
- `TestSCIMToken_PlaintextWorksOnce` — first use of the freshly-created plaintext token authenticates; second use fails (plaintext not stored).
- Alternative if returning plaintext on every read isn't the design: `TestSCIMToken_HashAuth_Succeeds` and `TestSCIMToken_HashAuth_WrongTokenFails`.
- `TestSCIMToken_RotateInvalidatesOld` — rotate; old plaintext returns 401, new plaintext authenticates.
- `TestSCIMToken_LazyHashMigration` — pre-existing plaintext-only row authenticates on first request and the hash is populated for subsequent reads.

**Attack-replay**:

- `TestAttack_SCIMTokenReplayAfterRotation` — capture token, rotate, replay → 401.

**Refresh-token replay** (`account/refresh_replay_test.go` + `service_test.go`):

- `TestRefresh_RotationOnByDefault` — default `Session.ShouldRotateRefreshToken()` is true for new app configs.
- `TestRefresh_OldTokenRevokesFamily` — issue session, refresh once (→ token T2 with old token T1 revoked), then attempt to refresh with T1 → returns error AND the session is revoked AND no token T3 was issued.
- `TestRefresh_FamilyRevocationAuditEvent` — replay path emits `bridge.AuditEvent{Action: "token.replay_detected", Severity: SeverityCritical}`.
- `TestRefresh_RevokedTokenSetTTL` — the revoked-token entry expires when the original token would have expired (no unbounded growth).
- `TestRefresh_ConcurrentRefreshIsSafe` — two concurrent refresh calls with the same token: one succeeds, the other detects replay and revokes (use `t -race`).

---

### Phase 6 — Polish

#### 6.1 Cookie hardening

**Design:**
- Default session cookie: `SameSite=Strict`, `Secure=true`, `HttpOnly=true`, `Path=/`, name prefixed `__Host-` when no `Domain` attribute is set.
- Configurable per-app (some SPAs need `Lax` for cross-origin top-level navs).

#### 6.2 Argon2 transparent rehash on login

Verify the comment at `account/argon2.go` is actually wired: after a successful password verify, if the stored hash's params don't match the current config, rehash and persist. Add a unit test to lock the behaviour.

#### 6.3 Login error message uniformity

Audit `plugins/password`, `plugins/social`, `plugins/passkey`, `plugins/magiclink` for failure messages. Replace specific messages ("user not found", "wrong password", "email domain not allowed") with `"authentication failed"` at the API boundary; keep specifics in server logs.

#### Phase 6 verification

**Cookie hardening** (`account/cookie_test.go` or wherever `setSessionCookie` lives):

- `TestSessionCookie_DefaultsAreSecureStrictHttpOnly` — fresh response cookie has `Secure=true`, `HttpOnly=true`, `SameSite=Strict`, `Path=/`.
- `TestSessionCookie_HostPrefixWhenNoDomain` — when no `Domain` configured, cookie name is `__Host-…`.
- `TestSessionCookie_PerAppOverride` — app config `cookie_samesite=lax` is honoured.
- `TestSessionCookie_NotSecureInDevMode` — TLS-disabled dev mode degrades gracefully (warning logged).

**Argon2 transparent rehash** (`account/argon2_rehash_test.go`):

- `TestArgon2_RehashOnLoginWhenParamsChange` — store a hash with old params, log in successfully, assert the stored hash's params now match current config.
- `TestArgon2_NoRehashWhenParamsMatch` — same params → DB row untouched.
- `TestArgon2_BcryptToArgon2Migration` — login against a bcrypt hash succeeds and the row is upgraded to argon2.

**Error message uniformity** (`plugins/{password,social,passkey,magiclink}/error_uniformity_test.go`):

- `TestAuthError_UnknownUserAndWrongPasswordIdentical` — both produce `{"error": "authentication failed"}`.
- `TestAuthError_PluginConsistency` — every plugin's auth-failure response uses the same error string and HTTP status.
- Server-side: `TestAuthError_DetailsInLogs` — failures still log the specific reason (so SOC tooling can distinguish without exposing it to attackers).

---

## Critical files (across all phases)

- `plugins/social/plugin.go` — phases 1, 2, 3
- `plugins/social/settings.go` (new) — phase 1
- `dashboard/nonce.go` — phase 1
- `dashboard/contributor.go` and per-plugin dashboard handlers — phases 1, 6
- `plugins/organization/{service,dashboard}.go` — phase 1
- `organization/store.go` and the four store backends — phase 1
- `dashboard/audit.go` (new) — phase 1
- `bridge/encryptor.go`, `bridge/aesgcm.go` (new) — phase 3
- `plugins/social/store_models.go` and migrations — phase 3
- `plugins/scim/{plugin,service}.go` and migrations — phase 5
- `account/session_refresh.go` (or equivalent) — phase 5
- `webhook/` and outbound delivery — phase 4
- `sdk/go/webhook_verify.go` (new) — phase 4

## Migration / rollout

- Phase 1 invalidates in-flight nonces and OAuth states on deploy → users see "form expired"/"invalid state" and retry. Acceptable for a single deploy window.
- Phase 3 ships behind a feature flag (`AUTHSOME_ENCRYPT_OAUTH_TOKENS`); enabled in staging first, then prod after a soak.
- Phase 5 SCIM token migration is lazy (hash-on-first-use); no downtime.
- Each phase ships as its own PR with its own commit history; nothing forces them into a single deploy.

## Verification (overall)

Each phase's verification block above is the test inventory for that phase — those tests are part of the phase's acceptance criteria, not a follow-up. Cross-cutting requirements:

- **Per-PR gate:** `go test ./...` and `go test -race ./...` must pass; the coverage-gate script must pass; for any phase that adds a fuzz target, a one-time short fuzz run (`-fuzztime=30s`) must pass during PR CI.
- **Nightly:** longer fuzz runs (`-fuzztime=5m` per target) on a scheduled CI job.
- **End-to-end:** each phase adds at least one test to `extension/dashboard_actions_test.go` (or a sibling integration suite) that exercises the user-visible flow against the memory store engine.
- **Acceptance review:** before each phase's PR is merged, the reviewer walks through the phase's verification list and confirms every named test exists, has a meaningful assertion, and demonstrably failed at least once during development (visible in the commit history of the branch).
- **Out of scope:** the pre-existing `extension/extension.go:191` build break (`WatchRemoteContributor`) is unrelated and tracked separately.

## Open questions for execution

- Does `engine.HasPermission` already exist and cover org-scoped resources, or do we need a thin shim? (Resolved during phase 1 implementation.)
- Should phase 3's KMS-readiness include a config knob to require encryption (`require_oauth_token_encryption=true`) so prod can refuse to start without it? Default off; recommend on for prod docs.
- Webhook v2 signature header naming — `X-Authsome-Signature` vs reusing GitHub's `X-Hub-Signature-256` for ecosystem familiarity. Decide during phase 4.
