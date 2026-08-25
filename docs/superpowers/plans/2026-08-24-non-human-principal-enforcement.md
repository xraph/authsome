# Non-human principal enforcement implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make a session whose principal is not a user authenticate successfully through all three enforcement surfaces, so a service account, agent or workload credential reaches a guarded route instead of getting a 401.

**Architecture:** Three independent bugs in three files, each fixed behind its own test, plus one integration test that proves they compose. No new packages, no store changes, no migrations. The `principal` package already exists and supplies the kinds and the context helpers, so this work consumes it and never re-implements it.

**Tech Stack:** Go, `github.com/xraph/forge` v1.4.5, `github.com/golang-jwt/jwt/v5`, `stretchr/testify`, `go.jetify.com/typeid/v2`.

**Spec:** [docs/superpowers/specs/2026-08-24-non-human-principal-enforcement-design.md](../specs/2026-08-24-non-human-principal-enforcement-design.md)

## Global Constraints

- Every fix must leave the human path byte-identical in behaviour. A user session authenticates exactly as it does today. Each task carries its own user-path regression assertion.
- Do not compare `PrincipalKind` against the bare string `"service_account"` in new code. Use `principal.Kind` constants. The bare string appears in existing code and stays there for now.
- Do not add `middleware.WithPrincipal` or `middleware.PrincipalFrom`. Those names are claimed by `docs/superpowers/specs/2026-08-24-non-human-principals-design.md`, which is being implemented in parallel. This plan adds `middleware.PrincipalRefFrom`, which reads through `principal.FromContext` first so it composes with that work when it lands.
- `tokenformat.TokenClaims` is edited by this plan and by the RFC 8693 plan, which adds `Act *ActClaim`. Fields do not collide and all are `omitempty`. If 8693 has already landed when you start, add your fields alongside theirs in one pass.
- Run `go build ./...` before every commit. The repo builds clean at the time of writing.
- Test command for the whole affected surface: `go test ./tokenformat/... ./authprovider/... ./middleware/... -race`

## File structure

| File | Responsibility | Task |
|---|---|---|
| `tokenformat/format.go` | `TokenClaims` gains `PrincipalKind` and `PrincipalID` | 1 |
| `tokenformat/jwt.go` | `customClaims` serialises them as `pk` and `pid` | 1 |
| `tokenformat/jwt_test.go` (new) | Round-trip coverage for the new claims | 1 |
| `authprovider/session.go` | `Authenticate` stops requiring a user; `BridgeToContext` records auth method for every caller | 2 |
| `authprovider/session_test.go` (new) | `SessionGuard` path accepts a non-human session | 2 |
| `middleware/context.go` | `PrincipalRefFrom` helper | 3 |
| `middleware/auth.go` | `RequireAuth` gates on principal; JWT branch stops panicking | 3, 4 |
| `middleware/auth_test.go` | `RequireAuth` coverage | 3 |
| `middleware/auth_jwt_test.go` | Malformed-claim and non-user-subject coverage | 4 |
| `middleware/enforcement_integration_test.go` (new) | The proving test across both guards | 5 |

---

### Task 1: TokenClaims carries the principal kind

A JWT minted for a non-human principal has no way to say so today, so the JWT branch in Task 4 has nothing to branch on. This task is first because Tasks 4 and 5 consume it.

**Files:**
- Modify: `tokenformat/format.go:18-27`
- Modify: `tokenformat/jwt.go:69-76` (`customClaims`), `tokenformat/jwt.go:78-110` (`GenerateAccessToken`), `tokenformat/jwt.go:113-150` (`ValidateAccessToken`)
- Test: `tokenformat/jwt_test.go` (new file, the package has no tests today)

**Interfaces:**
- Consumes: nothing.
- Produces: `tokenformat.TokenClaims` with two new string fields, `PrincipalKind` (JSON `pk`) and `PrincipalID` (JSON `pid`). Tasks 4 and 5 read `claims.PrincipalKind` and compare it against `string(principal.KindUser)` and friends.

- [ ] **Step 1: Write the failing test**

Create `tokenformat/jwt_test.go`:

```go
package tokenformat_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/tokenformat"
)

func newTestJWT(t *testing.T) *tokenformat.JWT {
	t.Helper()
	j, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-at-least-32-bytes-ok"),
	})
	require.NoError(t, err)
	return j
}

func TestJWT_RoundTripsPrincipalClaims(t *testing.T) {
	j := newTestJWT(t)

	token, err := j.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:        "svc_01hqxw000000000000000000",
		AppID:         "aapp_01hqxw000000000000000000",
		SessionID:     "ases_01hqxw000000000000000000",
		PrincipalKind: "workload",
		PrincipalID:   "svc_01hqxw000000000000000000",
		ExpiresAt:     time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	got, err := j.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "workload", got.PrincipalKind)
	assert.Equal(t, "svc_01hqxw000000000000000000", got.PrincipalID)
	assert.Equal(t, "svc_01hqxw000000000000000000", got.UserID, "sub carries the principal id")
}

func TestJWT_UserTokenOmitsPrincipalClaims(t *testing.T) {
	j := newTestJWT(t)

	token, err := j.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:    "ausr_01hqxw000000000000000000",
		AppID:     "aapp_01hqxw000000000000000000",
		SessionID: "ases_01hqxw000000000000000000",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	got, err := j.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Empty(t, got.PrincipalKind, "a user token must be unchanged on the wire")
	assert.Empty(t, got.PrincipalID)
	assert.Equal(t, "ausr_01hqxw000000000000000000", got.UserID)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./tokenformat/ -run TestJWT_ -v`

Expected: compile failure, `unknown field PrincipalKind in struct literal of type tokenformat.TokenClaims`.

- [ ] **Step 3: Add the fields to TokenClaims**

In `tokenformat/format.go`, extend the struct:

```go
type TokenClaims struct {
	UserID    string   `json:"sub"`
	AppID     string   `json:"app_id"`
	EnvID     string   `json:"env_id,omitempty"`
	OrgID     string   `json:"org_id,omitempty"`
	SessionID string   `json:"sid"`
	Scopes    []string `json:"scopes,omitempty"`
	IssuedAt  time.Time
	ExpiresAt time.Time

	// PrincipalKind names the kind of caller this token belongs to, using the
	// values in the principal package: "user", "agent", "workload" or
	// "service_account". Empty means user, so tokens minted before this field
	// existed keep validating.
	//
	// It is a string and not principal.Kind because tokenformat sits below
	// principal in the import graph and a claims struct should stay free of
	// domain dependencies.
	PrincipalKind string `json:"pk,omitempty"`

	// PrincipalID is the caller's ID, duplicating what Subject carries. It is
	// here so a consumer can read the principal without deciding whether sub
	// means a user, which is exactly the ambiguity that made the JWT auth path
	// panic on a non-human token.
	PrincipalID string `json:"pid,omitempty"`
}
```

- [ ] **Step 4: Serialise the fields in customClaims**

In `tokenformat/jwt.go`, add to the `customClaims` struct:

```go
type customClaims struct {
	jwt.RegisteredClaims
	AppID         string   `json:"app_id,omitempty"`
	EnvID         string   `json:"env_id,omitempty"`
	OrgID         string   `json:"org_id,omitempty"`
	SessionID     string   `json:"sid,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	PrincipalKind string   `json:"pk,omitempty"`
	PrincipalID   string   `json:"pid,omitempty"`
}
```

In `GenerateAccessToken`, add the two fields to the `customClaims` literal:

```go
		SessionID:     claims.SessionID,
		Scopes:        claims.Scopes,
		PrincipalKind: claims.PrincipalKind,
		PrincipalID:   claims.PrincipalID,
	}
```

In `ValidateAccessToken`, add them to the returned `TokenClaims`:

```go
	return &TokenClaims{
		UserID:        claims.Subject,
		AppID:         claims.AppID,
		EnvID:         claims.EnvID,
		OrgID:         claims.OrgID,
		SessionID:     claims.SessionID,
		Scopes:        claims.Scopes,
		IssuedAt:      issuedAt,
		ExpiresAt:     expiresAt,
		PrincipalKind: claims.PrincipalKind,
		PrincipalID:   claims.PrincipalID,
	}, nil
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./tokenformat/ -run TestJWT_ -v`

Expected: PASS, both tests.

- [ ] **Step 6: Confirm nothing else broke**

Run: `go build ./... && go test ./tokenformat/... ./middleware/... -race`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add tokenformat/format.go tokenformat/jwt.go tokenformat/jwt_test.go
git commit -m "feat(tokenformat): carry the principal kind in access token claims"
```

---

### Task 2: SessionProvider stops requiring a user

This is the 401 that hits every route registered with `forge.WithGroupAuth("session")` or `plugin.SessionGuard`.

**Files:**
- Modify: `authprovider/session.go:102-130` (`Authenticate`), `authprovider/session.go:195-205` (`BridgeToContext`)
- Test: `authprovider/session_test.go` (new file, the package has no tests today)

**Interfaces:**
- Consumes: `principal.Kind` constants from `github.com/xraph/authsome/principal`.
- Produces: an `*auth.AuthContext` whose `Subject` is the service account ID and whose `Claims` carry `principal_kind` and `principal_id`, for a session with `PrincipalKind == "service_account"`. `SessionData.User` is nil in that case, which `BridgeToContext` already tolerates.

- [ ] **Step 1: Write the failing test**

Create `authprovider/session_test.go`:

```go
package authprovider_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

func TestSessionProvider_ServiceAccountAuthenticates(t *testing.T) {
	appID := id.NewAppID()
	svcID := id.NewServiceAccountID()
	sess := &session.Session{
		ID:               id.NewSessionID(),
		AppID:            appID,
		PrincipalKind:    "service_account",
		ServiceAccountID: svcID,
		Roles:            []string{"deployer"},
		Token:            "machine-token",
	}

	p := authprovider.NewSessionProvider(
		func(token string) (*session.Session, error) {
			if token == "machine-token" {
				return sess, nil
			}
			return nil, errors.New("not found")
		},
		func(_ string) (*user.User, error) {
			return nil, errors.New("resolveUser must not be called for a service account")
		},
		log.NewNoopLogger(),
	)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/x", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer machine-token")

	authCtx, err := p.Authenticate(context.Background(), req)
	require.NoError(t, err, "a service-account session must authenticate")
	assert.Equal(t, svcID.String(), authCtx.Subject)
	assert.Equal(t, []string{"deployer"}, authCtx.Roles)
	assert.Equal(t, "service_account", authCtx.Claims["principal_kind"])
	assert.Equal(t, svcID.String(), authCtx.Claims["principal_id"])

	data, ok := authCtx.Data.(*authprovider.SessionData)
	require.True(t, ok)
	assert.Nil(t, data.User, "there is no user behind a service account")
	assert.Equal(t, sess.ID, data.Session.ID)
}

func TestSessionProvider_UserPathUnchanged(t *testing.T) {
	appID := id.NewAppID()
	userID := id.NewUserID()
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  appID,
		UserID: userID,
		Roles:  []string{"admin"},
		Token:  "human-token",
	}
	u := &user.User{ID: userID, AppID: appID, Email: "a@b.com"}

	p := authprovider.NewSessionProvider(
		func(string) (*session.Session, error) { return sess, nil },
		func(idStr string) (*user.User, error) {
			require.Equal(t, userID.String(), idStr)
			return u, nil
		},
		log.NewNoopLogger(),
	)

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/x", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer human-token")

	authCtx, err := p.Authenticate(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, userID.String(), authCtx.Subject)
	assert.Equal(t, "a@b.com", authCtx.Claims["email"])

	data, ok := authCtx.Data.(*authprovider.SessionData)
	require.True(t, ok)
	require.NotNil(t, data.User)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./authprovider/ -run TestSessionProvider_ -v`

Expected: `TestSessionProvider_ServiceAccountAuthenticates` FAILS with `auth.ErrAuthenticationFailed`, because `resolveUser("")` returns the error the fake was told to return. `TestSessionProvider_UserPathUnchanged` passes already.

- [ ] **Step 3: Branch before the user lookup**

In `authprovider/session.go`, replace the block starting at the `// 4. Resolve user from session` comment (currently line 102) so the non-human case returns before `resolveUser` runs:

```go
	// 4. A caller that is not a user has no user row to resolve. Return the
	//    principal directly.
	//
	//    Without this branch resolveUser("") fails and every route behind
	//    forge.WithGroupAuth("session") answers 401 to a credential that is
	//    otherwise entirely valid, which is what made service accounts
	//    unusable on guarded routes.
	if sess.PrincipalKind != "" && sess.PrincipalKind != string(principal.KindUser) {
		principalID := sess.ServiceAccountID.String()
		return &auth.AuthContext{
			Subject:      principalID,
			ProviderName: "session",
			Roles:        sess.Roles,
			Data:         &SessionData{Session: sess},
			Claims: map[string]any{
				"principal_kind": sess.PrincipalKind,
				"principal_id":   principalID,
			},
		}, nil
	}

	// 5. Resolve user from session
	u, err := p.resolveUser(sess.UserID.String())
```

Add the import:

```go
	"github.com/xraph/authsome/principal"
```

- [ ] **Step 4: Confirm SessionData has no other consumer**

The spec's risk section asks you to audit this before shipping the branch,
because `SessionData.User` can now be nil where callers previously assumed it
was not.

Run: `grep -rn "SessionData" --include="*.go" . | grep -v authprovider/session.go`

Expected: only `webauthn.SessionData` hits in `plugins/passkey/`, which is an
unrelated type from the webauthn library. `BridgeToContext` is the only
consumer of `authprovider.SessionData`, and it already guards on
`data.User != nil`. If this grep shows anything else dereferencing `.User`,
fix it here before continuing.

- [ ] **Step 5: Record the auth method for every caller**

In `BridgeToContext`, `WithAuthMethod` currently sits inside the `if data.User != nil` guard, so a machine request reaches the handler with no auth method recorded. Move it out:

```go
	if data.User != nil {
		goCtx = authmw.WithUser(goCtx, data.User)
	}
	goCtx = authmw.WithAuthMethod(goCtx, "session")

	ctx.WithContext(goCtx)
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./authprovider/ -run TestSessionProvider_ -v`

Expected: PASS, both tests.

- [ ] **Step 7: Confirm nothing else broke**

Run: `go build ./... && go test ./authprovider/... ./middleware/... ./plugins/... -race`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add authprovider/session.go authprovider/session_test.go
git commit -m "fix(authprovider): authenticate sessions whose principal is not a user"
```

---

### Task 3: RequireAuth gates on the principal

`middleware.RequireAuth()` checks `UserFrom(ctx)`, which is never populated for a machine caller, so the nine `api/*_handlers.go` groups built on it answer 401.

**Files:**
- Modify: `middleware/context.go` (add `PrincipalRefFrom`)
- Modify: `middleware/auth.go:658-690` (`RequireAuth`)
- Test: `middleware/auth_test.go` (append)

**Interfaces:**
- Consumes: `principal.Ref`, `principal.Kind`, `principal.UserRef`, `principal.FromContext` from Task 0 dependencies (the `principal` package, already on disk).
- Produces: `middleware.PrincipalRefFrom(ctx context.Context) (principal.Ref, bool)`. Task 5 asserts through `RequireAuth` behaviour and does not call this directly.

- [ ] **Step 1: Write the failing test**

Append to `middleware/auth_test.go`:

```go
func TestRequireAuth_WithServiceAccountSession(t *testing.T) {
	sess := &session.Session{
		ID:               id.NewSessionID(),
		AppID:            id.NewAppID(),
		PrincipalKind:    "service_account",
		ServiceAccountID: id.NewServiceAccountID(),
	}

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			ctx.WithContext(middleware.WithSession(ctx.Context(), sess))
			return next(ctx)
		}
	})
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code,
		"a machine principal is authenticated even though there is no user")
}

func TestRequireAuth_SessionWithoutPrincipalStillRejected(t *testing.T) {
	// An empty PrincipalKind means "user" for backwards compatibility, and a
	// user session with no resolved user is not an authenticated caller. This
	// guards against the fix widening into "any session at all will do".
	sess := &session.Session{
		ID:    id.NewSessionID(),
		AppID: id.NewAppID(),
	}

	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			ctx.WithContext(middleware.WithSession(ctx.Context(), sess))
			return next(ctx)
		}
	})
	router.Use(middleware.RequireAuth())
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./middleware/ -run TestRequireAuth_ -v`

Expected: `TestRequireAuth_WithServiceAccountSession` FAILS with 401. The other three `TestRequireAuth_*` tests pass.

- [ ] **Step 3: Add the PrincipalRefFrom helper**

Append to `middleware/context.go`:

```go
// PrincipalRefFrom returns the authenticated caller as a principal ref,
// whatever kind it is.
//
// It reads three sources in order, which is what lets it work today and keep
// working once the principal package is wired all the way through the
// middleware. First a principal already resolved onto the context, which is
// what the principal package's own middleware will set. Then a user, which is
// every human request today. Then a session carrying a non-user PrincipalKind,
// which is how service accounts, agents and workloads arrive before that
// wiring exists.
//
// An empty PrincipalKind means user, matching the field's documented
// backwards-compatible default, so a session without a resolved user is not a
// caller and this returns false for it.
func PrincipalRefFrom(ctx context.Context) (principal.Ref, bool) {
	if p, ok := principal.FromContext(ctx); ok && p != nil {
		return p.Ref, true
	}

	if u, ok := UserFrom(ctx); ok && u != nil {
		return principal.UserRef(u.ID), true
	}

	if s, ok := SessionFrom(ctx); ok && s != nil {
		kind := principal.Kind(s.PrincipalKind)
		if kind != "" && kind != principal.KindUser {
			return principal.Ref{Kind: kind, ID: s.ServiceAccountID.String()}, true
		}
	}

	return principal.Ref{}, false
}
```

Add the import to `middleware/context.go`:

```go
	"github.com/xraph/authsome/principal"
```

- [ ] **Step 4: Gate RequireAuth on the principal**

In `middleware/auth.go`, change the guard inside `RequireAuth`:

```go
			if _, ok := PrincipalRefFrom(ctx.Context()); !ok {
```

Leave the whole response body, including the `authDebugEnabled()` branch, exactly as it is. Update the doc comment above `RequireAuth`:

```go
// RequireAuth returns a forge middleware that rejects unauthenticated requests.
//
// Authenticated means a resolved principal of any kind, not specifically a
// user. A machine caller carries no *user.User, so gating on one turned away
// every service account, agent and workload credential.
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./middleware/ -run TestRequireAuth_ -v`

Expected: PASS, all five tests including the two pre-existing ones.

- [ ] **Step 6: Confirm nothing else broke**

Run: `go build ./... && go test ./... -race 2>&1 | grep -E "^(FAIL|ok .*(api|middleware|authprovider))" | head -20`

Expected: no `FAIL` lines.

- [ ] **Step 7: Commit**

```bash
git add middleware/context.go middleware/auth.go middleware/auth_test.go
git commit -m "fix(middleware): gate RequireAuth on the principal and not the user"
```

---

### Task 4: The JWT path stops panicking on a non-user subject

`id.MustParse(claims.UserID)` panics when `sub` is empty because `id.Parse("")` returns an error. There are five `MustParse` calls in that branch, all on claim values, and all of them panic on anything malformed.

**Files:**
- Modify: `middleware/auth.go:420-460` (the context-building block in `tryJWTAuth`)
- Test: `middleware/auth_jwt_test.go` (append)

**Interfaces:**
- Consumes: `tokenformat.TokenClaims.PrincipalKind` from Task 1.
- Produces: no new exported symbols.

- [ ] **Step 1: Write the failing test**

Append to `middleware/auth_jwt_test.go`:

```go
func TestJWTAuth_ServiceAccountSubjectDoesNotPanic(t *testing.T) {
	appID := id.NewAppID()
	sessID := id.NewSessionID()
	svcID := id.NewServiceAccountID()

	validator := &mockJWTValidator{
		claims: &tokenformat.TokenClaims{
			UserID:        svcID.String(),
			AppID:         appID.String(),
			SessionID:     sessID.String(),
			PrincipalKind: "service_account",
			PrincipalID:   svcID.String(),
		},
	}

	mw := middleware.AuthMiddlewareWithJWT(
		func(string) (*session.Session, error) { return nil, errors.New("not found") },
		func(string) (*user.User, error) {
			t.Fatal("resolveUser must not be called for a service-account JWT")
			return nil, nil
		},
		&mockStrategyAuth{},
		validator,
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.GET("/test", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer a.b.c")
	rec := httptest.NewRecorder()

	require.NotPanics(t, func() { router.ServeHTTP(rec, req) })
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJWTAuth_MalformedClaimIDsAreRefusedNotPanicked(t *testing.T) {
	valid := func() *tokenformat.TokenClaims {
		return &tokenformat.TokenClaims{
			UserID:    id.NewUserID().String(),
			AppID:     id.NewAppID().String(),
			SessionID: id.NewSessionID().String(),
		}
	}

	cases := []struct {
		name   string
		mutate func(c *tokenformat.TokenClaims)
	}{
		{"app_id", func(c *tokenformat.TokenClaims) { c.AppID = "not-a-typeid" }},
		{"sub", func(c *tokenformat.TokenClaims) { c.UserID = "not-a-typeid" }},
		{"sid", func(c *tokenformat.TokenClaims) { c.SessionID = "not-a-typeid" }},
		{"env_id", func(c *tokenformat.TokenClaims) { c.EnvID = "not-a-typeid" }},
		{"org_id", func(c *tokenformat.TokenClaims) { c.OrgID = "not-a-typeid" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := valid()
			tc.mutate(claims)

			mw := middleware.AuthMiddlewareWithJWT(
				func(string) (*session.Session, error) { return nil, errors.New("not found") },
				func(string) (*user.User, error) { return nil, errors.New("not found") },
				&mockStrategyAuth{},
				&mockJWTValidator{claims: claims},
				log.NewNoopLogger(),
			)

			router := forge.NewRouter()
			router.Use(mw)
			router.Use(middleware.RequireAuth())
			router.GET("/test", func(ctx forge.Context) error {
				return ctx.NoContent(http.StatusOK)
			})

			req, err := http.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer a.b.c")
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() { router.ServeHTTP(rec, req) })
			assert.Equal(t, http.StatusUnauthorized, rec.Code,
				"a token carrying a malformed %s is refused, not honoured", tc.name)
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./middleware/ -run "TestJWTAuth_(ServiceAccountSubject|MalformedClaimIDs)" -v`

Expected: panics surfacing as `require.NotPanics` failures, with messages like `id: must parse "not-a-typeid"`.

- [ ] **Step 3: Replace the panicking parses**

In `middleware/auth.go`, replace the context-building block that currently runs from `appID := id.MustParse(claims.AppID)` through the `resolveUser` call:

```go
	goCtx := ctx.Context()

	// Build a virtual session from JWT claims (no DB record needed).
	//
	// These values arrive inside a token, so they are attacker-influenced even
	// though a valid signature is required to reach here. MustParse panics on
	// anything malformed, and it panicked outright on a non-human token, whose
	// sub is a service account id and whose UserID claim was previously empty.
	// Refuse the token instead.
	appID, err := id.Parse(claims.AppID)
	if err != nil {
		logger.Warn("auth middleware: JWT carries a malformed app_id",
			log.String("error", err.Error()),
		)
		return false
	}
	goCtx = WithAppID(goCtx, appID)

	subjectID, err := id.Parse(claims.UserID)
	if err != nil {
		logger.Warn("auth middleware: JWT carries a malformed sub",
			log.String("error", err.Error()),
		)
		return false
	}

	isUser := claims.PrincipalKind == "" ||
		claims.PrincipalKind == string(principal.KindUser)
	if isUser {
		goCtx = WithUserID(goCtx, subjectID)
	}

	goCtx = WithAuthMethod(goCtx, "jwt")

	if claims.SessionID != "" {
		sessionID, sidErr := id.Parse(claims.SessionID)
		if sidErr != nil {
			logger.Warn("auth middleware: JWT carries a malformed sid",
				log.String("error", sidErr.Error()),
			)
			return false
		}
		goCtx = WithSessionID(goCtx, sessionID)
	}

	if claims.EnvID != "" {
		envID, envErr := id.Parse(claims.EnvID)
		if envErr != nil {
			logger.Warn("auth middleware: JWT carries a malformed env_id",
				log.String("error", envErr.Error()),
			)
			return false
		}
		goCtx = WithEnvID(goCtx, envID)
	}

	if claims.OrgID != "" {
		orgID, orgErr := id.Parse(claims.OrgID)
		if orgErr != nil {
			logger.Warn("auth middleware: JWT carries a malformed org_id",
				log.String("error", orgErr.Error()),
			)
			return false
		}
		goCtx = WithOrgID(goCtx, orgID)
		goCtx = forge.WithScope(goCtx, forge.NewOrgScope(claims.AppID, claims.OrgID))
	} else {
		goCtx = forge.WithScope(goCtx, forge.NewAppScope(claims.AppID))
	}

	// A machine caller has no user row. Put the principal on the context and
	// stop, so RequireAuth sees a caller via PrincipalRefFrom.
	if !isUser {
		goCtx = principal.NewContext(goCtx, &principal.Principal{
			Ref:   principal.Ref{Kind: principal.Kind(claims.PrincipalKind), ID: subjectID.String()},
			AppID: appID,
			Scopes: claims.Scopes,
		})
		ctx.WithContext(goCtx)
		return true
	}

	// Resolve user from claims.
	u, err := resolveUser(claims.UserID)
	if err != nil {
		logger.Debug("auth middleware: JWT user resolution failed",
			log.String("user_id", claims.UserID),
			log.String("error", err.Error()),
		)
		ctx.WithContext(goCtx)
		return true // Authenticated via JWT even if user lookup fails
	}
	goCtx = WithUser(goCtx, u)

	ctx.WithContext(goCtx)
	return true
}
```

Add the import to `middleware/auth.go`:

```go
	"github.com/xraph/authsome/principal"
```

Note on the malformed-claim test expecting 401: a malformed `sub` now returns `false` from the JWT branch, so no principal reaches the context and `RequireAuth` rejects. That is the behaviour change the test pins.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./middleware/ -run "TestJWTAuth_" -v`

Expected: PASS, including the pre-existing `TestJWTAuth_SessionChecker_*` tests.

- [ ] **Step 5: Confirm nothing else broke**

Run: `go build ./... && go test ./... -race 2>&1 | grep -E "^FAIL" | head`

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add middleware/auth.go middleware/auth_jwt_test.go
git commit -m "fix(middleware): refuse malformed JWT claim ids instead of panicking"
```

---

### Task 5: The proving test

Tasks 2, 3 and 4 each fix one surface behind a unit test. This task is the spec's actual deliverable: a credential that goes all the way through.

**Files:**
- Test: `middleware/enforcement_integration_test.go` (new)

**Interfaces:**
- Consumes: everything from Tasks 1 through 4.
- Produces: nothing. This is the gate.

- [ ] **Step 1: Write the failing test**

Create `middleware/enforcement_integration_test.go`:

```go
package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// machineSession builds the credential a workload or service account holds:
// a real session row with no user behind it.
func machineSession(kind string) *session.Session {
	return &session.Session{
		ID:               id.NewSessionID(),
		AppID:            id.NewAppID(),
		PrincipalKind:    kind,
		ServiceAccountID: id.NewServiceAccountID(),
		Roles:            []string{"deployer"},
		Token:            "machine-token",
	}
}

// TestEnforcement_MachineCredentialReachesGuardedRoute is the deliverable of
// the non-human principal enforcement spec. Before that work a machine
// credential authenticated, carried roles, resolved from its token, and then
// got a 401 from every guard it met.
func TestEnforcement_MachineCredentialReachesGuardedRoute(t *testing.T) {
	for _, kind := range []string{"service_account", "workload", "agent"} {
		t.Run(kind, func(t *testing.T) {
			sess := machineSession(kind)

			mw := middleware.AuthMiddleware(
				func(token string) (*session.Session, error) {
					if token == "machine-token" {
						return sess, nil
					}
					return nil, errors.New("invalid")
				},
				func(string) (*user.User, error) {
					return nil, errors.New("no user behind a machine principal")
				},
				log.NewNoopLogger(),
			)

			router := forge.NewRouter()
			router.Use(mw)
			router.Use(middleware.RequireAuth())
			router.GET("/guarded", func(ctx forge.Context) error {
				s, ok := middleware.SessionFrom(ctx.Context())
				require.True(t, ok, "the session must reach the handler")
				assert.Equal(t, kind, s.PrincipalKind)
				return ctx.NoContent(http.StatusOK)
			})

			req, err := http.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer machine-token")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code,
				"a %s credential must reach a RequireAuth-guarded route", kind)
		})
	}
}

// TestEnforcement_HumanCredentialUnchanged is the regression guard, and it
// matters more than the cases above.
func TestEnforcement_HumanCredentialUnchanged(t *testing.T) {
	userID := id.NewUserID()
	appID := id.NewAppID()
	sess := &session.Session{
		ID:     id.NewSessionID(),
		AppID:  appID,
		UserID: userID,
		Token:  "human-token",
	}
	u := &user.User{ID: userID, AppID: appID, Email: "human@example.com"}

	mw := middleware.AuthMiddleware(
		func(string) (*session.Session, error) { return sess, nil },
		func(idStr string) (*user.User, error) {
			if idStr == userID.String() {
				return u, nil
			}
			return nil, errors.New("not found")
		},
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	router.GET("/guarded", func(ctx forge.Context) error {
		got, ok := middleware.UserFrom(ctx.Context())
		require.True(t, ok, "UserFrom must still work for humans")
		assert.Equal(t, "human@example.com", got.Email)
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer human-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestEnforcement_AnonymousStillRejected pins the boundary, so the fix cannot
// quietly widen into "anything with a session gets in".
func TestEnforcement_AnonymousStillRejected(t *testing.T) {
	mw := middleware.AuthMiddleware(
		func(string) (*session.Session, error) { return nil, errors.New("invalid") },
		func(string) (*user.User, error) { return nil, errors.New("not found") },
		log.NewNoopLogger(),
	)

	router := forge.NewRouter()
	router.Use(mw)
	router.Use(middleware.RequireAuth())
	router.GET("/guarded", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req, err := http.NewRequestWithContext(context.Background(), "GET", "/guarded", nil)
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
```

- [ ] **Step 2: Run the test**

Run: `go test ./middleware/ -run TestEnforcement_ -v`

Expected: PASS. If Tasks 2 through 4 are done, this passes on the first run. If it fails, the failing subtest names which surface is still rejecting.

- [ ] **Step 3: Run the full suite with the race detector**

Run: `go test ./... -race 2>&1 | grep -E "^(FAIL|panic)" | head`

Expected: no output.

- [ ] **Step 4: Commit**

```bash
git add middleware/enforcement_integration_test.go
git commit -m "test(middleware): prove a machine credential reaches a guarded route"
```

---

## Notes for the executor

**The `SessionGuard` path is covered by Task 2, not Task 5.** Task 5 drives `middleware.AuthMiddleware` plus `RequireAuth`, because building a full forge auth registry in a middleware test pulls in the engine. The registry path is `authprovider.SessionProvider`, which Task 2 tests directly at its own boundary. Together they cover both guards named in the spec.

**Backends.** Every test here uses fakes and touches no store, so all of it runs anywhere. If you want to exercise this against a real store, use `secutil.NewTestEngine`, which is memory-backed. Postgres and sqlite return `not implemented` from all five service-account store methods, so on those two backends you cannot create the principal at all until the `principal` package work lands its store implementations. That is tracked in `docs/superpowers/specs/2026-08-24-non-human-principals-design.md`.

**If the principal middleware wiring lands first.** Should `middleware.PrincipalFrom` and `middleware.WithPrincipal` already exist when you start Task 3, do not add `PrincipalRefFrom` alongside them. Fold the three-source lookup into their helper and gate `RequireAuth` on that. The behaviour this plan pins does not change; only the name does.
