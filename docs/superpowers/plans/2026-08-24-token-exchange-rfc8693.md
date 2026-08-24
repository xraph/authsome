# RFC 8693 Token Exchange Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Put the RFC 8693 wire protocol on the OAuth2 token endpoint, so an external client can trade a broad credential for a narrow, short-lived, audited one over the standard grant.

**Architecture:** This is a protocol layer, not an authority model. The actor chain, the delegation grant that authorises an exchange, and the `Engine.ExchangeToken` primitive all come from the non-human principals plan. This plan adds what that plan does not have: the `urn:ietf:params:oauth:grant-type:token-exchange` grant at `/v1/oauth/token`, subject and actor token resolution, the `act` JWT claim, a `scopes` column on sessions, and a security event per exchange.

**Tech Stack:** Go 1.26, forge router, grove migrations and ORM, postgres + sqlite + mongo + in-memory stores, testify.

**Spec:** `docs/superpowers/specs/2026-08-24-token-exchange-rfc8693-design.md`

## Hard dependency

**This plan cannot start until [`2026-08-24-non-human-principals.md`](2026-08-24-non-human-principals.md) has landed through at least its Task 14.** That plan owns:

| Thing | Where it comes from |
|---|---|
| `principal.Ref`, `principal.Chain`, `principal.Kind` | their Task 1 |
| `principal.Delegation`, `principal.GrantKind`, `principal.Store` | their Task 2 |
| `session.Session.Actors`, `.ActorGrant`, `.DelegationID`; `ImpersonatedBy()` as a method | their Task 3 |
| The `actors` column across all four drivers | their Tasks 5, 6, 7 |
| `plugin.Engine.ResolvePrincipal`, `.PrincipalStore`, `.Can` | their Task 9 |
| `Engine.ExchangeToken`, `Engine.GrantDelegation`, `intersectScopes` | their Task 14 |
| Impersonation recorded as a delegation grant | their Task 16 |

An earlier draft of this plan defined its own `session.Actor` type, its own `Impersonator()` accessor, and an `authsome_oauth2_exchange_policies` table. All three duplicated that plan with incompatible types. They are gone. The authority for an exchange is a `principal.Delegation`, which is the explicit delegation policy the design calls for, better integrated than a second table would have been.

## Global Constraints

- Go 1.26.0 (`go.mod`). No new third-party dependencies.
- Run tests with `go test ./...`. Run `make check` (fmt, vet, lint) before every commit.
- Four store drivers stay in parity: postgres, sqlite, mongo, memory.
- Mongo slices must never be written as nil. grove writes every mapped field regardless of the bson `omitempty` tag, so a nil slice reaches mongo as `null` and fails to decode. See commit `9116564`.
- Mongo has no migration group; its schema is implicit in the models.
- Migration versions use the `20260824000 06x` series. `...0001` and `...0002` are claimed by the agentauth and DPoP plans, and `...0050` through `...0052` by non-human principals. Do not reuse any of them.
- Imports group stdlib / third-party / `github.com/xraph/authsome`, formatted with `goimports -local github.com/xraph/authsome`.
- `session.Session.PrincipalKind` is `principal.Kind`, not `string`, once the dependency lands. Compare against `principal.KindUser` and friends, never against string literals.

---

### Task 1: Sessions record the scopes they were issued with

**Files:**
- Modify: `session/session.go`
- Modify: `store/postgres/models.go`, `store/sqlite/models.go`, `store/mongo/models.go`
- Modify: `store/postgres/migrations.go`, `store/sqlite/migrations.go`
- Modify: `plugins/oauth2provider/plugin.go:881` (`issueTokens`), `:934` (`issueClientToken`)
- Modify: `engine_token_exchange.go` (the `_ = scopes` line from their Task 14)
- Test: `store/memory/session_scopes_test.go` (create)
- Test: `plugins/oauth2provider/issue_scopes_test.go` (create)

**Interfaces:**
- Consumes: nothing from this plan.
- Produces: `session.Session.Scopes []string`.

This closes a hole the non-human principals plan leaves open on purpose. Its `Engine.ExchangeToken` computes granted scopes and then writes `_ = scopes // recorded on the session by whichever scope field the app uses`, with a note to assign it "once you confirm it". There is no such field today. This task adds it, and Step 8 wires their line up to it.

- [ ] **Step 1: Write the failing test**

Create `store/memory/session_scopes_test.go`:

```go
package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

func TestSessionScopesRoundTrip(t *testing.T) {
	st := memory.New()
	ctx := context.Background()

	require.NoError(t, st.CreateSession(ctx, &session.Session{
		ID:        id.NewSessionID(),
		AppID:     id.NewAppID(),
		UserID:    id.NewUserID(),
		Token:     "tok-scopes",
		Scopes:    []string{"invoices:read", "invoices:write"},
		ExpiresAt: time.Now().Add(time.Hour),
	}))

	got, err := st.GetSessionByToken(ctx, "tok-scopes")
	require.NoError(t, err)
	assert.Equal(t, []string{"invoices:read", "invoices:write"}, got.Scopes)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./store/memory/ -run TestSessionScopesRoundTrip -v`
Expected: FAIL, `unknown field Scopes in struct literal`.

- [ ] **Step 3: Add the field**

In `session/session.go`, immediately after the `Roles []string` field:

```go
	// Scopes holds the OAuth scopes this session was issued with, stamped at
	// issuance. Same trade as Roles: authoritative for what this token may
	// do, stale with respect to anything granted afterwards.
	//
	// Empty does not mean "may do anything". A session minted by password
	// sign-in carries no scopes, and the token exchange grant treats that as
	// "no subject-side ceiling" while still bounding the result by the
	// delegation grant and the client's own registered scopes.
	Scopes []string `json:"scopes,omitempty"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./store/memory/ -run TestSessionScopesRoundTrip -v`
Expected: PASS. The memory store keeps whole structs, so no store code changes for this one.

- [ ] **Step 5: Add the column to the three persistent models**

In `store/postgres/models.go`, in `SessionModel` after `Roles`:

```go
	// JSON for the same reason Roles is: these strings are read back as an
	// authorization decision, so the encoding must not be able to invent or
	// merge a member.
	Scopes json.RawMessage `grove:"scopes,type:jsonb"`
```

In `toSession`, after the roles decode:

```go
	if len(m.Scopes) > 0 {
		_ = json.Unmarshal(m.Scopes, &s.Scopes) //nolint:errcheck // best-effort decode
	}
```

In `fromSession`, after the roles encode:

```go
	// Always encoded: the column is NOT NULL and json.RawMessage cannot scan
	// a NULL back.
	m.Scopes, _ = json.Marshal(s.Scopes) //nolint:errcheck // best-effort encode
```

Apply the identical three edits to `store/sqlite/models.go`.

For `store/mongo/models.go`, the field is a native array beside `Roles`:

```go
	Scopes []string `grove:"scopes" bson:"scopes,omitempty"`
```

with `m.Scopes = append([]string{}, s.Scopes...)` in `fromSession` (non-nil, per the global constraint) and `s.Scopes = append([]string{}, m.Scopes...)` in `toSession`.

- [ ] **Step 6: Add the migrations**

In `store/postgres/migrations.go`, in the `init()` block that holds `add_session_roles`:

```go
		&migrate.Migration{
			Name:    "add_session_scopes",
			Version: "20260824000060",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS scopes JSONB NOT NULL DEFAULT '[]'::jsonb;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions DROP COLUMN IF EXISTS scopes;
`)
				return err
			},
		},
```

In `store/sqlite/migrations.go`, same name and version:

```go
		&migrate.Migration{
			Name:    "add_session_scopes",
			Version: "20260824000060",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx,
					`ALTER TABLE authsome_sessions ADD COLUMN scopes TEXT NOT NULL DEFAULT '';`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				// sqlite cannot drop a column without rebuilding the table.
				return nil
			},
		},
```

- [ ] **Step 7: Write the failing issuance test**

Create `plugins/oauth2provider/issue_scopes_test.go`:

```go
package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/store/memory"
)

// decodeTokenResponse reads the token endpoint's JSON body. Reused by Task 5.
func decodeTokenResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestAuthCodeGrantStampsScopesOnSession(t *testing.T) {
	p, _, mux := newFixture(t)
	core := memory.New()
	p.SetStore(core)

	q := baseAuthorizeQuery(confidentialID)
	q.Set("scope", "openid profile")
	code := codeFrom(t, authorize(t, mux, q))

	body := decodeTokenResponse(t, postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  registeredURI,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	}))

	sess, err := core.GetSessionByToken(context.Background(), body["access_token"].(string))
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile"}, sess.Scopes)
}
```

`newFixture`, `authorize`, `baseAuthorizeQuery`, `codeFrom` and `postToken` already exist in `authcode_test.go` in this package. Do not redefine them.

- [ ] **Step 8: Stamp scopes everywhere a scoped session is minted**

In `plugins/oauth2provider/plugin.go`, in `issueTokens`, right after the `account.NewSession(...)` error check:

```go
	// Stamp the granted scopes. Without this they exist only in the JWT claims
	// and the response body, so an opaque token loses them and token exchange
	// has no subject-side ceiling to narrow against.
	sess.Scopes = scopes
```

In `issueClientToken`, after its `account.NewSession(...)` error check:

```go
	sess.Scopes = client.Scopes
```

In `engine_token_exchange.go`, replace the placeholder their Task 14 leaves behind:

```go
	// was: _ = scopes
	sess.Scopes = scopes
```

- [ ] **Step 9: Run the tests**

Run: `go test ./store/... ./plugins/oauth2provider/ . -count=1`
Expected: PASS, including their `TestExchangeIntersectsScopes`.

- [ ] **Step 10: Check and commit**

```bash
make check
git add session store plugins/oauth2provider engine_token_exchange.go
git commit -m "feat(session): record the OAuth scopes a session was issued with"
```

---

### Task 2: TokenExchangeTTL through the config layers

**Files:**
- Modify: `account/service.go:165`
- Modify: `appsessionconfig/appsessionconfig.go`
- Modify: `environment/settings.go:196`
- Modify: `api/requests.go:599`
- Modify: `service.go` (the engine's base session config)
- Test: `account/session_config_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `account.SessionConfig.TokenExchangeTTL time.Duration`, resolved per app by `Engine.sessionConfigForApp` at `service.go:783`.

- [ ] **Step 1: Write the failing test**

Create `account/session_config_test.go`:

```go
package account_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appsessionconfig"
)

func TestAppConfigOverridesTokenExchangeTTL(t *testing.T) {
	base := account.SessionConfig{TokenExchangeTTL: 5 * time.Minute}
	secs := 120
	(&appsessionconfig.Config{TokenExchangeTTLSeconds: &secs}).ApplyTo(&base)
	assert.Equal(t, 2*time.Minute, base.TokenExchangeTTL)
}

func TestNilAppConfigLeavesTokenExchangeTTL(t *testing.T) {
	base := account.SessionConfig{TokenExchangeTTL: 5 * time.Minute}
	(&appsessionconfig.Config{}).ApplyTo(&base)
	assert.Equal(t, 5*time.Minute, base.TokenExchangeTTL)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./account/ -run TokenExchangeTTL -v`
Expected: FAIL, `unknown field TokenExchangeTTL`.

- [ ] **Step 3: Add the field to SessionConfig**

In `account/service.go`:

```go
	// TokenExchangeTTL caps tokens minted by the RFC 8693 token exchange
	// grant. Short by design: an exchanged token is meant to be re-minted,
	// not held. Zero means the caller's own default applies.
	TokenExchangeTTL time.Duration
```

- [ ] **Step 4: Add the per-app override**

In `appsessionconfig/appsessionconfig.go`, in `Config` beside the other token overrides:

```go
	TokenExchangeTTLSeconds *int `json:"token_exchange_ttl_seconds,omitempty"`
```

and in `ApplyTo`:

```go
	if c.TokenExchangeTTLSeconds != nil {
		base.TokenExchangeTTL = time.Duration(*c.TokenExchangeTTLSeconds) * time.Second
	}
```

- [ ] **Step 5: Add the per-environment override**

In `environment/settings.go`, in `Settings` beside `TokenTTLSeconds`:

```go
	TokenExchangeTTLSeconds *int `json:"token_exchange_ttl_seconds,omitempty"`
```

and in `ApplySessionOverrides`:

```go
	if s.TokenExchangeTTLSeconds != nil {
		cfg.TokenExchangeTTL = time.Duration(*s.TokenExchangeTTLSeconds) * time.Second
	}
```

- [ ] **Step 6: Expose it and default it**

In `api/requests.go`, in the struct holding `RefreshTokenTTLSeconds` at line 599:

```go
	TokenExchangeTTLSeconds *int `json:"token_exchange_ttl_seconds,omitempty" description:"Token exchange TTL in seconds (nil = inherit)"`
```

Find the handler mapping that struct onto `appsessionconfig.Config` with `grep -rn "RefreshTokenTTLSeconds" --include="*.go" api/` and add the passthrough.

In the engine's base session config (find it with `grep -n "func (e \*Engine) sessionConfig()" service.go`):

```go
	if cfg.TokenExchangeTTL == 0 {
		cfg.TokenExchangeTTL = 5 * time.Minute
	}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./account/ ./environment/ ./api/ -count=1`
Expected: PASS.

- [ ] **Step 8: Check and commit**

```bash
make check
git add account appsessionconfig environment api service.go
git commit -m "feat(config): add per-app TokenExchangeTTL"
```

---

### Task 3: The act claim in tokenformat

**Files:**
- Modify: `tokenformat/format.go:18`
- Modify: `tokenformat/jwt.go:69`, `:78`, `:143`
- Test: `tokenformat/act_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `tokenformat.ActClaim{Subject string, Act *ActClaim}`; `tokenformat.TokenClaims.Act *ActClaim`.

RFC 8693 nests `act` recursively, so the type is self-referential.

- [ ] **Step 1: Write the failing test**

Create `tokenformat/act_test.go`:

```go
package tokenformat_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/tokenformat"
)

func TestActClaimNests(t *testing.T) {
	raw, err := json.Marshal(tokenformat.ActClaim{
		Subject: "workload:svc_outer",
		Act:     &tokenformat.ActClaim{Subject: "user:usr_inner"},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"sub":"workload:svc_outer","act":{"sub":"user:usr_inner"}}`, string(raw))
}

func TestJWTRoundTripsActClaim(t *testing.T) {
	f := newTestJWT(t)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID: "usr_1", AppID: "aapp_1", SessionID: "ases_1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
		Act: &tokenformat.ActClaim{Subject: "workload:svc_actor"},
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	require.NotNil(t, out.Act)
	assert.Equal(t, "workload:svc_actor", out.Act.Subject)
}

func TestJWTOmitsActWhenNil(t *testing.T) {
	f := newTestJWT(t)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID: "usr_1", AppID: "aapp_1", SessionID: "ases_1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Nil(t, out.Act)
}
```

Add a `newTestJWT(t *testing.T) tokenformat.Format` helper in this file that builds a JWT format with a test signing key. Read `tokenformat/jwt.go` for the real constructor name and config shape, and follow whatever an existing test in that package already does if one exists.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tokenformat/ -run TestAct -v`
Expected: FAIL, `undefined: tokenformat.ActClaim`.

- [ ] **Step 3: Add ActClaim and the TokenClaims field**

In `tokenformat/format.go`, above `TokenClaims`:

```go
// ActClaim is the RFC 8693 `act` claim: the party acting on behalf of the
// subject. It nests, so a chain of delegations is a chain of ActClaims with
// the immediate actor outermost. Subject carries a principal.Ref in its
// "kind:id" string form.
type ActClaim struct {
	Subject string    `json:"sub"`
	Act     *ActClaim `json:"act,omitempty"`
}
```

and to `TokenClaims`:

```go
	// Act is present for a delegated token and nil otherwise. Impersonation
	// emits no act claim at all (RFC 8693 section 1.1), which is why the full
	// record lives on the session row rather than in the token.
	Act *ActClaim `json:"act,omitempty"`
```

- [ ] **Step 4: Wire it through the JWT format**

In `tokenformat/jwt.go`, add `Act *ActClaim` with tag `json:"act,omitempty"` to `customClaims`, set `Act: claims.Act` on the `jwtClaims` literal in `GenerateAccessToken`, and add `Act: claims.Act` to the `&TokenClaims{...}` returned by `ValidateAccessToken`.

- [ ] **Step 5: Emit it where sessions are minted**

The plugin does not mint the exchanged session after this rebase; `Engine.ExchangeToken` does. Add the emission there, in `engine_token_exchange.go`, where the JWT is generated:

```go
	// Delegation names both parties. Impersonation emits no act claim at all
	// (RFC 8693 section 1.1); the chain on the session row is the record.
	if sess.ActorGrant == principal.GrantDelegation {
		claims.Act = actChainToClaim(sess.Actors)
	}
```

and add the helper beside `intersectScopes`:

```go
// actChainToClaim renders an actor chain as a nested RFC 8693 act claim, the
// immediate actor outermost. An empty chain yields nil, which omits the claim.
func actChainToClaim(chain principal.Chain) *tokenformat.ActClaim {
	var head, tail *tokenformat.ActClaim
	for _, ref := range chain {
		node := &tokenformat.ActClaim{Subject: ref.String()}
		if head == nil {
			head, tail = node, node
			continue
		}
		tail.Act = node
		tail = node
	}
	return head
}
```

Extend their `TestExchangeMintsADelegatedSession` with an assertion that a JWT-format app puts the actor in `act`, rather than writing a separate test for it.

- [ ] **Step 6: Run the tests**

Run: `go test ./tokenformat/ . -count=1`
Expected: PASS.

- [ ] **Step 7: Check and commit**

```bash
make check
git add tokenformat engine_token_exchange.go
git commit -m "feat(tokenformat): add the RFC 8693 act claim and emit it on delegated tokens"
```

---

### Task 4: SecurityEvents on the plugin Engine interface

**Files:**
- Modify: `plugin/plugin.go`
- Test: `plugin/security_events_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `plugin.Engine.SecurityEvents() securityevent.Store`.

The concrete engine already has this method at `engine.go:831`, so the interface addition is satisfied without touching the engine. Their Task 9 widens the same interface with the principal methods; if it has landed, add this alongside rather than reordering what it wrote.

- [ ] **Step 1: Write the failing test**

Create `plugin/security_events_test.go`:

```go
package plugin_test

import (
	"testing"

	"github.com/xraph/authsome/plugin"
)

// Plugins write security events directly rather than through the hook bus.
// The bus bridge at engine.go:526 never sets AppID, and securityevent.Query
// filters on it, so events recorded that way are written but unqueryable.
func TestEngineInterfaceExposesSecurityEvents(t *testing.T) {
	var e plugin.Engine
	if e != nil {
		_ = e.SecurityEvents
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugin/ -run TestEngineInterfaceExposesSecurityEvents -v`
Expected: FAIL to compile, the method is not on the interface.

- [ ] **Step 3: Add the method**

In `plugin/plugin.go`, beside `APIKeyStore()`:

```go
	// SecurityEvents returns the queryable security event store, or nil when
	// the engine was built without one.
	SecurityEvents() securityevent.Store
```

Add the `github.com/xraph/authsome/securityevent` import.

- [ ] **Step 4: Run the tests**

Run: `go test ./plugin/ ./plugins/... -count=1`
Expected: PASS. Any plugin test with its own `Engine` double will fail to compile until it gains the method; add `func (e *fake) SecurityEvents() securityevent.Store { return nil }` to each.

- [ ] **Step 5: Check and commit**

```bash
make check
git add plugin plugins
git commit -m "feat(plugin): expose the security event store to plugins"
```

---

### Task 5: The RFC 8693 grant

**Files:**
- Create: `plugins/oauth2provider/token_exchange.go`
- Modify: `plugins/oauth2provider/models.go` (`OAuth2Client.PrincipalID`)
- Modify: `plugins/oauth2provider/store_models.go` and the four store files
- Modify: `plugins/oauth2provider/migrations.go`
- Modify: `plugins/oauth2provider/plugin.go:289`, `:346` area, `:580`, `:746`
- Test: `plugins/oauth2provider/token_exchange_test.go` (create)

**Interfaces:**
- Consumes: `Engine.ExchangeToken` and `ExchangeRequest` (their Task 14), `principal.Ref`, `principal.KindWorkload` (their Task 1), `session.Session.Scopes` (Task 1), `session.Session.Subject()` (their Task 3).
- Produces: `tokenExchangeGrantType`; `narrowRequestedScopes`; `p.handleTokenExchangeGrant`; `OAuth2Client.PrincipalID id.ServiceAccountID`.

**The one design decision this task makes.** `Engine.ExchangeToken` takes a `principal.Ref` actor, and `principal.Kind` has no `oauth_client` member. So an OAuth2 client cannot act until it is linked to a principal. `OAuth2Client` gains a nullable `PrincipalID id.ServiceAccountID`, and a client without one is refused the grant with a clear error. That keeps a single authority model: the delegation grant decides, and the OAuth client is just one way of presenting as its actor.

**Two things to confirm against their landed code before writing any of this.** First, whether `ExchangeRequest` lives in the root `authsome` package; if it does, the plugin cannot reference it without an import cycle, so move the type into `principal` and leave an alias behind. Second, whether `ExchangeToken` is on `plugin.Engine`; if not, add it there in Step 5 with a comment naming the OAuth grant as its caller. The code below assumes both resolved in favour of `principal.ExchangeRequest` and a `plugin.Engine` method.

- [ ] **Step 1: Add the client-to-principal link**

In `plugins/oauth2provider/models.go`, in `OAuth2Client`:

```go
	// PrincipalID links this client to the non-human principal it acts as.
	// Required for the token exchange grant: Engine.ExchangeToken takes a
	// principal.Ref actor and principal.Kind has no oauth_client member, so an
	// unlinked client has no identity to act with. Zero for clients that only
	// use the other grants.
	PrincipalID id.ServiceAccountID `json:"principal_id,omitempty"`
```

In `plugins/oauth2provider/store_models.go`, add to `OAuth2ClientModel`:

```go
	PrincipalID string `grove:"principal_id" bson:"principal_id,omitempty"`
```

Map it in both directions the way the existing optional id fields in that file are mapped: guarded on non-zero when writing, guarded on non-empty and parsed with `id.ParseServiceAccountID` when reading.

In `plugins/oauth2provider/migrations.go`, in the postgres block:

```go
		&migrate.Migration{
			Name:    "add_oauth2_client_principal_id",
			Version: "20260824000061",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients
    ADD COLUMN IF NOT EXISTS principal_id TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx,
					`ALTER TABLE authsome_oauth2_clients DROP COLUMN IF EXISTS principal_id;`)
				return err
			},
		},
```

and in the sqlite block, the same name and version with `ALTER TABLE authsome_oauth2_clients ADD COLUMN principal_id TEXT NOT NULL DEFAULT '';` and a `Down` that returns nil.

- [ ] **Step 2: Write the failing test**

Create `plugins/oauth2provider/token_exchange_test.go` in package `oauth2provider_test`.

Name every test `TestRFC8693_*`. `authcode_test.go` already owns `TestTokenExchange_RejectsMismatchedRedirectURI` and `TestTokenExchange_RejectsMismatchedClientID`, which are about redeeming an authorization code, so a shared prefix would make `-run` ambiguous between two unrelated features.

```go
package oauth2provider_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/securityevent"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/tokenformat"
)

const (
	exchangeGrant   = "urn:ietf:params:oauth:grant-type:token-exchange"
	accessTokenType = "urn:ietf:params:oauth:token-type:access_token"
	xchgClientID    = "svc-exchange"
	xchgSecret      = "svc-exchange-secret"
)

// recordingEvents captures what the plugin writes. Task 6 asserts on it.
type recordingEvents struct{ events []*securityevent.Event }

func (r *recordingEvents) RecordSecurityEvent(_ context.Context, e *securityevent.Event) error {
	r.events = append(r.events, e)
	return nil
}

func (r *recordingEvents) QuerySecurityEvents(_ context.Context, _ *securityevent.Query) ([]*securityevent.Event, string, error) {
	return r.events, "", nil
}

// exchangeEngine implements only the plugin.Engine methods the grant touches.
// Embedding the interface satisfies the type; anything else stays nil and
// panics loudly if the path ever reaches it.
type exchangeEngine struct {
	plugin.Engine
	core   store.Store
	events *recordingEvents
	cfg    account.SessionConfig

	// lastExchange records what the handler asked for, so a test can assert on
	// the translation from HTTP parameters into an ExchangeRequest.
	lastExchange *principal.ExchangeRequest
	issued       *session.Session
	exchangeErr  error
}

func (e *exchangeEngine) Store() store.Store                  { return e.core }
func (e *exchangeEngine) Logger() log.Logger                  { return log.NewNoopLogger() }
func (e *exchangeEngine) SecurityEvents() securityevent.Store { return e.events }

func (e *exchangeEngine) ResolveSessionByToken(token string) (*session.Session, error) {
	return e.core.GetSessionByToken(context.Background(), token)
}

func (e *exchangeEngine) SessionConfigForApp(_ context.Context, _ id.AppID, _ ...id.EnvironmentID) account.SessionConfig {
	return e.cfg
}

func (e *exchangeEngine) TokenFormatForApp(_ string) tokenformat.Format { return tokenformat.Opaque{} }

func (e *exchangeEngine) ExchangeToken(_ context.Context, req *principal.ExchangeRequest) (*session.Session, error) {
	e.lastExchange = req
	return e.issued, e.exchangeErr
}

type xchgFixture struct {
	plugin *oauth2provider.Plugin
	oauth  oauth2provider.Store
	core   store.Store
	engine *exchangeEngine
	mux    forge.Router
	appID  id.AppID
	events *recordingEvents
}

// newExchangeFixture registers one confidential client holding scopes a and b,
// registered for the exchange grant and linked to a principal.
func newExchangeFixture(t *testing.T, ttl time.Duration) *xchgFixture {
	t.Helper()
	ctx := context.Background()

	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	oauth := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(oauth)

	core := memory.New()
	events := &recordingEvents{}
	eng := &exchangeEngine{
		core:   core,
		events: events,
		cfg:    account.SessionConfig{TokenTTL: time.Hour, TokenExchangeTTL: ttl},
	}
	// OnInit keeps the pre-set oauth2Store, so engine.DB() is never reached.
	require.NoError(t, p.OnInit(ctx, eng))

	hashed, err := bcrypt.GenerateFromPassword([]byte(xchgSecret), bcrypt.MinCost)
	require.NoError(t, err)

	appID := id.NewAppID()
	require.NoError(t, oauth.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     xchgClientID,
		ClientSecret: string(hashed),
		Name:         "Exchange client",
		Scopes:       []string{"a", "b"},
		GrantTypes:   []string{exchangeGrant},
		PrincipalID:  id.NewServiceAccountID(),
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	return &xchgFixture{plugin: p, oauth: oauth, core: core, engine: eng, mux: mux, appID: appID, events: events}
}

func (f *xchgFixture) seedSubject(t *testing.T, appID id.AppID, scopes []string, life time.Duration) string {
	t.Helper()
	tok := "subject-" + id.NewSessionID().String()
	require.NoError(t, f.core.CreateSession(context.Background(), &session.Session{
		ID:        id.NewSessionID(),
		AppID:     appID,
		UserID:    id.NewUserID(),
		Token:     tok,
		Scopes:    scopes,
		ExpiresAt: time.Now().Add(life),
	}))
	return tok
}

// grantSucceeds makes the stub engine return a plausible delegated session.
func (f *xchgFixture) grantSucceeds(scopes []string, life time.Duration) {
	f.engine.issued = &session.Session{
		ID:         id.NewSessionID(),
		AppID:      f.appID,
		UserID:     id.NewUserID(),
		Token:      "issued-" + id.NewSessionID().String(),
		Scopes:     scopes,
		ActorGrant: principal.GrantDelegation,
		Actors:     principal.Chain{{Kind: principal.KindWorkload, ID: "svc_actor"}},
		ExpiresAt:  time.Now().Add(life),
	}
	f.engine.exchangeErr = nil
}

func (f *xchgFixture) exchange(t *testing.T, extra map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{
		"grant_type":    exchangeGrant,
		"client_id":     xchgClientID,
		"client_secret": xchgSecret,
	}
	for k, v := range extra {
		body[k] = v
	}
	return postToken(t, f.mux, body)
}

func TestRFC8693_ReturnsIssuedTokenTypeAndNoRefreshToken(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.grantSucceeds([]string{"a"}, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, time.Hour)

	body := decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	assert.Equal(t, accessTokenType, body["issued_token_type"])
	assert.Equal(t, "a", body["scope"])
	assert.NotEmpty(t, body["access_token"])
	// Re-exchange rather than refresh, so the subject stays the only durable
	// credential.
	assert.Empty(t, body["refresh_token"])
}

func TestRFC8693_PassesActorSubjectAndScopesToTheEngine(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.grantSucceeds([]string{"a"}, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, time.Hour)

	decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	require.NotNil(t, f.engine.lastExchange)
	assert.Equal(t, f.appID, f.engine.lastExchange.AppID)
	assert.Equal(t, []string{"a"}, f.engine.lastExchange.Scopes)
	assert.Equal(t, principal.KindWorkload, f.engine.lastExchange.Actor.Kind)
	assert.Equal(t, principal.KindUser, f.engine.lastExchange.RequestedSubject.Kind)
}

func TestRFC8693_RefusesScopeTheSubjectDoesNotHold(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.grantSucceeds([]string{"b"}, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "b",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange, "the engine must not be reached on a subject-side refusal")
}

func TestRFC8693_RefusesScopeTheClientIsNotRegisteredFor(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a", "c"}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "c", // held by the subject, not registered to the client
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange)
}

func TestRFC8693_RefusesEmptyScope(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_RefusesCrossApp(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, id.NewAppID(), []string{"a"}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, f.engine.lastExchange)
}

func TestRFC8693_RefusesPublicClient(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	require.NoError(t, f.oauth.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: f.appID, ClientID: "pub-exchange",
		Name: "Public", Scopes: []string{"a"}, GrantTypes: []string{exchangeGrant},
		Public: true, PrincipalID: id.NewServiceAccountID(),
	}))
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	rec := postToken(t, f.mux, map[string]string{
		"grant_type": exchangeGrant, "client_id": "pub-exchange",
		"client_secret": "anything", "subject_token": subject,
		"subject_token_type": accessTokenType, "scope": "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_RefusesClientWithNoPrincipal(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	hashed, err := bcrypt.GenerateFromPassword([]byte("s"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, f.oauth.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: f.appID, ClientID: "unlinked",
		ClientSecret: string(hashed), Name: "Unlinked", Scopes: []string{"a"},
		GrantTypes: []string{exchangeGrant}, // no PrincipalID
	}))
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	rec := postToken(t, f.mux, map[string]string{
		"grant_type": exchangeGrant, "client_id": "unlinked", "client_secret": "s",
		"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "principal")
}

func TestRFC8693_RefusesUnsupportedSubjectTokenType(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": "urn:ietf:params:oauth:token-type:refresh_token",
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported_token_type")
}

func TestRFC8693_SurfacesEngineRefusalAsInvalidGrant(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.engine.exchangeErr = assert.AnError // no live delegation grant
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "access_token")
}

func TestRFC8693_DiscoveryAdvertisesTheGrant(t *testing.T) {
	_, _, mux := newFixture(t)
	req := httptest.NewRequest("GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Contains(t, rec.Body.String(), exchangeGrant)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestRFC8693 -v`
Expected: FAIL, `unsupported grant_type` on every case that reaches the endpoint.

- [ ] **Step 4: Add the request and response fields**

In `plugins/oauth2provider/plugin.go`, add to `TokenRequest`:

```go
	// RFC 8693 token exchange.
	SubjectToken       string `json:"subject_token,omitempty" form:"subject_token"`
	SubjectTokenType   string `json:"subject_token_type,omitempty" form:"subject_token_type"`
	ActorToken         string `json:"actor_token,omitempty" form:"actor_token"`
	ActorTokenType     string `json:"actor_token_type,omitempty" form:"actor_token_type"`
	RequestedTokenType string `json:"requested_token_type,omitempty" form:"requested_token_type"`
	Scope              string `json:"scope,omitempty" form:"scope"`
	// Audience and Resource are accepted and audited but not enforced. That is
	// the seam RFC 8707 resource indicators land in.
	Audience string `json:"audience,omitempty" form:"audience"`
	Resource string `json:"resource,omitempty" form:"resource"`
```

and to `TokenResponse`:

```go
	// IssuedTokenType is required by RFC 8693 section 2.2.1 and omitted by
	// every other grant.
	IssuedTokenType string `json:"issued_token_type,omitempty"`
```

There is deliberately no request parameter selecting impersonation. The mode comes from the delegation grant's `GrantKind`, which an administrator authored through `Engine.GrantDelegation`. That is stricter than an earlier draft of this plan, which had a namespaced `authsome_act_mode` parameter, and it removes a way for a caller to ask for the more privileged mode.

- [ ] **Step 5: Write the handler**

Create `plugins/oauth2provider/token_exchange.go`:

```go
package oauth2provider

import (
	"fmt"
	"strings"
	"time"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// tokenExchangeGrantType is the IANA grant type for RFC 8693.
const tokenExchangeGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"

// Supported token type URNs. Both access-token forms resolve to a session,
// because in this codebase an OAuth access token is a session row.
const (
	tokenTypeAccessToken = "urn:ietf:params:oauth:token-type:access_token"
	tokenTypeSession     = "urn:x-authsome:params:oauth:token-type:session"
)

// Denial reasons recorded on failed exchanges. Closed set: these are alerted
// on, so the vocabulary must not drift.
const (
	denyNoGrant          = "no_grant"
	denyScopeEscalation  = "scope_escalation"
	denyCrossApp         = "cross_app"
	denyInvalidSubject   = "invalid_subject"
	denyUnsupportedToken = "unsupported_token_type"
	denyNoPrincipal      = "client_has_no_principal"
)

// errUnsupportedTokenType is returned for a token type this build does not
// resolve. Sentinel rather than a string match so the denial reason and the
// HTTP body cannot drift apart.
var errUnsupportedTokenType = forge.BadRequest("unsupported_token_type")

// resolveExchangeToken resolves a subject or actor token to its session.
func (p *Plugin) resolveExchangeToken(token, tokenType string) (*session.Session, error) {
	if tokenType != tokenTypeAccessToken && tokenType != tokenTypeSession {
		return nil, errUnsupportedTokenType
	}
	if p.engine == nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: no engine"))
	}
	sess, err := p.engine.ResolveSessionByToken(token)
	if err != nil || sess == nil {
		return nil, forge.BadRequest("invalid_grant")
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, forge.BadRequest("invalid_grant")
	}
	return sess, nil
}

// narrowRequestedScopes applies the two bounds this layer owns: the subject
// token's own scopes and the client's registered scopes.
//
// The delegation grant is the third bound and belongs to the engine, which
// applies it in intersectScopes. Checking the two cheap local bounds first
// means an obviously bad request never reaches the grant lookup.
//
// A subject with no scopes imposes no subject-side ceiling. A password
// sign-in produces exactly that, and treating it as "nothing" would make the
// session-downgrade case impossible on the first hop, while treating it as
// "everything" is safe only because the client bound and the grant bound both
// still apply.
func narrowRequestedScopes(requested, clientScopes, subjectScopes []string) ([]string, error) {
	// Required, unlike elsewhere. RFC 8693 makes scope optional and lets the
	// server choose, but the point of an exchange is asking for less than you
	// hold, and an omitted scope asks the server to guess.
	if len(requested) == 0 {
		return nil, fmt.Errorf("scope is required for token exchange")
	}

	inClient := toScopeSet(clientScopes)
	inSubject := toScopeSet(subjectScopes)
	subjectBounds := len(subjectScopes) > 0

	out := make([]string, 0, len(requested))
	for _, s := range requested {
		if _, ok := inClient[s]; !ok {
			return nil, fmt.Errorf("scope %q is not registered for this client", s)
		}
		if subjectBounds {
			if _, ok := inSubject[s]; !ok {
				return nil, fmt.Errorf("scope %q is not held by the subject token", s)
			}
		}
		out = append(out, s)
	}
	return out, nil
}

func toScopeSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}

func (p *Plugin) handleTokenExchangeGrant(ctx forge.Context, req *TokenRequest) (*TokenResponse, error) {
	// 1. Client authentication. Confidential only: a public client cannot keep
	// a secret and this is a privilege operation.
	if req.ClientID == "" || req.ClientSecret == "" {
		return nil, forge.BadRequest("client_id and client_secret required")
	}
	client, err := p.oauth2Store.GetClient(ctx.Context(), req.ClientID)
	if err != nil {
		return nil, forge.Unauthorized("invalid client")
	}
	if client.Public {
		return nil, forge.BadRequest("token exchange not allowed for public clients")
	}
	if cmpErr := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret)); cmpErr != nil {
		return nil, forge.Unauthorized("invalid client_secret")
	}
	if !p.clientSupportsGrant(client, tokenExchangeGrantType) {
		return nil, forge.BadRequest("unauthorized_client")
	}

	// 2. The client must have a principal to act as.
	if client.PrincipalID.IsNil() {
		e := forge.BadRequest("this client has no linked principal and cannot exchange tokens")
		p.recordExchange(ctx, client, nil, nil, nil, "", denyNoPrincipal, e)
		return nil, e
	}
	actor := principal.Ref{Kind: principal.KindWorkload, ID: client.PrincipalID.String()}

	// 3. Resolve the subject.
	if req.SubjectToken == "" || req.SubjectTokenType == "" {
		return nil, forge.BadRequest("subject_token and subject_token_type required")
	}
	if req.RequestedTokenType != "" && req.RequestedTokenType != tokenTypeAccessToken {
		return nil, forge.BadRequest("unsupported requested_token_type")
	}
	subject, err := p.resolveExchangeToken(req.SubjectToken, req.SubjectTokenType)
	if err != nil {
		reason := denyInvalidSubject
		if errors.Is(err, errUnsupportedTokenType) {
			reason = denyUnsupportedToken
		}
		p.recordExchange(ctx, client, nil, nil, nil, "", reason, err)
		return nil, err
	}

	// 4. Cross-app exchange is refused. Without this a client in one app
	// launders a session out of another.
	if subject.AppID != client.AppID {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, nil, nil, "", denyCrossApp, e)
		return nil, e
	}

	// 5. An actor token, when present, replaces the client as the acting
	// party. Its identity is proven by resolving it, never asserted.
	if req.ActorToken != "" {
		actorSess, aErr := p.resolveExchangeToken(req.ActorToken, req.ActorTokenType)
		if aErr != nil {
			reason := denyInvalidSubject
			if errors.Is(aErr, errUnsupportedTokenType) {
				reason = denyUnsupportedToken
			}
			p.recordExchange(ctx, client, subject, nil, nil, "", reason, aErr)
			return nil, aErr
		}
		if actorSess.AppID != client.AppID {
			e := forge.BadRequest("invalid_grant")
			p.recordExchange(ctx, client, subject, nil, nil, "", denyCrossApp, e)
			return nil, e
		}
		actor = actorSess.Subject()
	}

	// 6. The two bounds this layer owns.
	requested := strings.Fields(req.Scope)
	narrowed, err := narrowRequestedScopes(requested, client.Scopes, subject.Scopes)
	if err != nil {
		e := forge.BadRequest(fmt.Sprintf("invalid_scope: %s", err.Error()))
		p.recordExchange(ctx, client, subject, requested, nil, "", denyScopeEscalation, e)
		return nil, e
	}

	// 7. The engine owns the grant lookup, the grant scope filter, the TTL
	// bound and the actor chain. A refusal here means no live delegation
	// authorises this actor to act for this subject.
	issued, err := p.engine.ExchangeToken(ctx.Context(), &principal.ExchangeRequest{
		AppID:            client.AppID,
		Actor:            actor,
		RequestedSubject: subject.Subject(),
		Scopes:           narrowed,
		IPAddress:        ctx.Request().RemoteAddr,
		UserAgent:        ctx.Request().UserAgent(),
		CredentialID:     client.ClientID,
	})
	if err != nil {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, requested, nil, "", denyNoGrant, e)
		return nil, e
	}

	p.recordExchange(ctx, client, subject, requested, issued.Scopes, issued.ID.String(), "", nil)

	return &TokenResponse{
		AccessToken:     issued.Token,
		IssuedTokenType: tokenTypeAccessToken,
		TokenType:       "Bearer",
		ExpiresIn:       int(time.Until(issued.ExpiresAt).Seconds()),
		Scope:           strings.Join(issued.Scopes, " "),
	}, nil
}

// recordExchange writes one security event per attempt.
// Body implemented in Task 6; the signature is final.
func (p *Plugin) recordExchange(
	_ forge.Context,
	_ *OAuth2Client,
	_ *session.Session,
	_ []string, // requested scopes
	_ []string, // granted scopes, nil on failure
	_ string, // issued session id, "" on failure
	_ string, // denial reason, "" on success
	_ error,
) {
}
```

Add `"errors"` to the import block.

- [ ] **Step 6: Wire dispatch and discovery**

In `handleToken` at `plugin.go:580`:

```go
	case tokenExchangeGrantType:
		return p.handleTokenExchangeGrant(ctx, req)
```

In `handleDiscovery` at `plugin.go:746`:

```go
		GrantTypesSupported: []string{
			"authorization_code",
			"client_credentials",
			"urn:ietf:params:oauth:grant-type:device_code",
			tokenExchangeGrantType,
		},
```

- [ ] **Step 7: Run the tests**

Run: `go test ./plugins/oauth2provider/ -count=1 -v`
Expected: PASS, every `TestRFC8693_*` case.

- [ ] **Step 8: Check and commit**

```bash
make check
git add plugins/oauth2provider plugin principal
git commit -m "feat(oauth2): add the RFC 8693 token exchange grant"
```

---

### Task 6: Security events for every exchange

**Files:**
- Modify: `plugins/oauth2provider/token_exchange.go` (fill in `recordExchange`)
- Test: `plugins/oauth2provider/token_exchange_audit_test.go` (create)

**Interfaces:**
- Consumes: `plugin.Engine.SecurityEvents()` (Task 4), the denial constants and the `recordExchange` signature (Task 5).
- Produces: nothing new.

`recordingEvents`, `newExchangeFixture`, `seedSubject`, `grantSucceeds` and `exchange` come from Task 5. Do not redefine them.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/token_exchange_audit_test.go`:

```go
package oauth2provider_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestRFC8693_SuccessWritesSecurityEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.grantSucceeds([]string{"a"}, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, time.Hour)

	decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	require.Len(t, f.events.events, 1)
	ev := f.events.events[0]
	assert.Equal(t, "oauth2.token_exchange", ev.Action)
	assert.Equal(t, "success", ev.Outcome)
	// AppID must be set. The hook-bus bridge at engine.go:526 does not set it,
	// which is why the plugin writes to the store directly.
	assert.Equal(t, f.appID, ev.AppID)
	assert.Equal(t, "a", ev.Metadata["granted_scopes"])
	assert.Equal(t, xchgClientID, ev.Metadata["client_id"])
	assert.NotEmpty(t, ev.Metadata["issued_session_id"])
	assert.NotEmpty(t, ev.Metadata["subject_session_id"])
	assert.NotContains(t, ev.Metadata, "denial_reason")
}

func TestRFC8693_ScopeEscalationWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "b",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "failure", f.events.events[0].Outcome)
	assert.Equal(t, "scope_escalation", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_CrossAppWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, id.NewAppID(), []string{"a"}, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "cross_app", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_MissingGrantWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.engine.exchangeErr = assert.AnError
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "failure", f.events.events[0].Outcome)
	assert.Equal(t, "no_grant", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_UnsupportedTokenTypeWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": "urn:ietf:params:oauth:token-type:refresh_token",
		"scope":              "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "unsupported_token_type", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_UnlinkedClientWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	hashed, err := bcrypt.GenerateFromPassword([]byte("s"), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, f.oauth.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: f.appID, ClientID: "unlinked2",
		ClientSecret: string(hashed), Name: "Unlinked", Scopes: []string{"a"},
		GrantTypes: []string{exchangeGrant},
	}))
	subject := f.seedSubject(t, f.appID, []string{"a"}, time.Hour)

	postToken(t, f.mux, map[string]string{
		"grant_type": exchangeGrant, "client_id": "unlinked2", "client_secret": "s",
		"subject_token": subject, "subject_token_type": accessTokenType, "scope": "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "client_has_no_principal", f.events.events[0].Metadata["denial_reason"])
}
```

Add the `context`, `bcrypt` and `oauth2provider` imports that last case needs.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run WritesSecurityEvent -v`
Expected: FAIL, no events recorded.

- [ ] **Step 3: Fill in recordExchange**

Replace the empty body in `plugins/oauth2provider/token_exchange.go`:

```go
func (p *Plugin) recordExchange(
	ctx forge.Context,
	client *OAuth2Client,
	subject *session.Session,
	requested []string,
	granted []string,
	issuedSessionID string,
	denialReason string,
	cause error,
) {
	if p.engine == nil {
		return
	}
	events := p.engine.SecurityEvents()
	if events == nil {
		return
	}

	outcome := "success"
	if denialReason != "" || cause != nil {
		outcome = "failure"
	}

	meta := map[string]string{"client_id": client.ClientID}
	if len(requested) > 0 {
		meta["requested_scopes"] = strings.Join(requested, " ")
	}
	if len(granted) > 0 {
		meta["granted_scopes"] = strings.Join(granted, " ")
	}
	if issuedSessionID != "" {
		meta["issued_session_id"] = issuedSessionID
	}
	if denialReason != "" {
		meta["denial_reason"] = denialReason
	}

	var userID id.UserID
	if subject != nil {
		userID = subject.UserID
		ref := subject.Subject()
		meta["subject_session_id"] = subject.ID.String()
		meta["subject_kind"] = string(ref.Kind)
		meta["subject_principal_id"] = ref.ID
		meta["chain_depth"] = fmt.Sprintf("%d", len(subject.Actors)+1)
	}

	_ = events.RecordSecurityEvent(ctx.Context(), &securityevent.Event{
		AppID:     client.AppID,
		UserID:    userID,
		Action:    "oauth2.token_exchange",
		Outcome:   outcome,
		Metadata:  meta,
		IPAddress: ctx.Request().RemoteAddr,
		UserAgent: ctx.Request().UserAgent(),
		CreatedAt: time.Now(),
	}) //nolint:errcheck // audit is best-effort and must not fail the exchange
}
```

Add the `id` and `securityevent` imports.

- [ ] **Step 4: Verify no call site changed**

The signature was declared in full in Task 5 Step 5, so filling in the body should not touch `handleTokenExchangeGrant`.

Run: `git diff --stat plugins/oauth2provider/token_exchange.go`
Expected: changes confined to `recordExchange` and the import block.

- [ ] **Step 5: Run the whole suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 6: Check and commit**

```bash
make check
git add plugins/oauth2provider
git commit -m "feat(oauth2): write a security event for every token exchange"
```

---

## Self-review notes

Spec coverage, against `docs/superpowers/specs/2026-08-24-token-exchange-rfc8693-design.md` as amended for this rebase:

- Session scopes stamped at issuance: Task 1, which also closes the `_ = scopes` gap their Task 14 leaves.
- Actor chain, and the delegation versus impersonation distinction: **not this plan.** Owned by non-human principals Tasks 3 and 16.
- Authority for an exchange: **not this plan.** `principal.Delegation`, their Tasks 2 and 14.
- Required `scope`: Task 5, `narrowRequestedScopes` first branch, tested by `TestRFC8693_RefusesEmptyScope`.
- Subject-side and client-side scope bounds: Task 5, one test each, both asserting the engine is never reached.
- Cross-app refusal: Task 5, with its own denial reason.
- TTL: Task 2 supplies the config value. The subject-life bound is the engine's, covered by their `TestExchangeBoundsSessionTTLByGrantExpiry`.
- `act` claim: Task 3 defines it and emits it from the engine's minting path, guarded on `GrantDelegation`.
- `issued_token_type`, discovery advertisement, no refresh token: Task 5.
- Security event per attempt with a closed denial-reason set: Task 6, one test per reason.

Deliberately not done here, both flagged in the spec as separate work: fixing the hook bridge at `engine.go:526` that drops `AppID`, and dropping the `impersonated_by` column.
