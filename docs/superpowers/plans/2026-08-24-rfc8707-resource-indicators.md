# RFC 8707 Resource Indicators Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bind OAuth2 tokens to the resource server they were requested for, so a token minted for service A stops authenticating at service B.

**Architecture:** Clients pass a repeatable `resource` parameter at the three grant entry points. The plugin validates each value against a per-client allowlist and stores the result on the authorization code or device code, then stamps it onto the issued session and, for JWT-format apps, into the `aud` claim. The core auth middleware optionally refuses a token audienced at some other resource.

**Tech Stack:** Go 1.26.0, forge v1.9.10, go-utils v1.1.6, grove v1.6.0, golang-jwt/jwt/v5, testify.

**Spec:** `docs/superpowers/specs/2026-08-24-rfc8707-resource-indicators-design.md`

## Global Constraints

- Go 1.26.0. Module is `github.com/xraph/authsome`.
- Core migration group `authsome`: use version `20260824000040`, name `add_session_audience`. Do **not** use `20260824000001` in this group. Three other plans dated today (DPoP `add_session_dpop_jkt`, agentauth `add_session_agent_principal`, token-exchange `add_session_scopes_and_actors`) each claim `20260824000001` in this same group and already collide with one another. Two migrations sharing a version in one group fails at startup.
- Plugin migration group `authsome-oauth2`: use version `20260824000001`, name `add_oauth2_resources`. This number is reserved for this work; dynamic client registration moved to `20260824000030` in commit `a8489d8` to clear it.
- Every new field is `[]string` and defaults to empty. An empty audience means an unrestricted token, which is what every existing token is.
- Mongo: never write a nil slice to a grove-mapped array field. `toSessionModel` must assign `[]string{}` when the source slice is empty, or the collection's generated `$jsonSchema` rejects the write. See commit `9116564`.
- Backwards compatibility is not optional. A client that sends no `resource` must behave exactly as it does today, and a deployment that configures no expected audience must see no behaviour change.
- Do not add a `[]string` field with a `query:"..."` struct tag anywhere. The binder has no slice case and fails the whole request. See Task 5.
- Test packages: `oauth2provider_test` for handler tests, `package oauth2provider` for unexported helper tests (`dashboard_test.go` already does this).

---

### Task 1: Audience on the session entity and every store

**Files:**
- Modify: `session/session.go` (add field after `Roles`, around line 47)
- Modify: `store/postgres/models.go:222` (model field), `:305` (decode), `:345` (encode)
- Modify: `store/sqlite/models.go:222`, `:305`, `:345`
- Modify: `store/mongo/models.go:209` (model field), `:255` (encode guard), `:330` (decode)
- Modify: `store/postgres/migrations.go`, `store/sqlite/migrations.go`, `store/mongo/migrations.go`
- Test: `store/storetest/storetest.go` (new case + new function)

**Interfaces:**
- Produces: `session.Session.Audience []string`. Every later task reads or writes this field.
- Consumes: nothing.

- [ ] **Step 1: Write the failing conformance test**

In `store/storetest/storetest.go`, add to the `cases` slice at line 47, immediately after the `SessionRolesRoundTrip` entry:

```go
		{"SessionAudienceRoundTrip", testSessionAudienceRoundTrip},
```

Then add this function directly after `testSessionRolesRoundTrip` ends (line 384):

```go
// testSessionAudienceRoundTrip proves a session's granted audience survives
// persistence on every backend.
//
// The audience is what stops a token issued for one resource server from
// authenticating at another, so a backend that drops the field does not fail
// loudly. It silently returns an unrestricted token, which is the exact
// confused-deputy hole RFC 8707 exists to close. The empty case matters just
// as much: a backend that encodes []string as a delimited list hands back a
// single empty-string member, and "" is an audience nobody can match.
func testSessionAudienceRoundTrip(t *testing.T, s store.Store) {
	ctx := context.Background()
	tn := seedTenant(t, s)
	u := seedUser(t, s, tn, "aud-"+suffix(tn.AppID.String())+"@example.com")

	sess := &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 tn.AppID,
		EnvID:                 tn.EnvID,
		UserID:                u.ID,
		Token:                 "tok-aud-" + suffix(tn.AppID.String()),
		RefreshToken:          "ref-aud-" + suffix(tn.AppID.String()),
		FamilyID:              id.NewSessionFamilyID(),
		Audience:              []string{"https://api.example.com", "https://files.example.com"},
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
		{"GetSessionByToken", func() (*session.Session, error) { return s.GetSessionByToken(ctx, sess.Token) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			assert.Equal(t,
				[]string{"https://api.example.com", "https://files.example.com"},
				got.Audience,
				"granted audience did not survive the round trip")
		})
	}

	bare := seedSession(t, s, tn, u.ID,
		"tok-noaud-"+suffix(tn.AppID.String()),
		"ref-noaud-"+suffix(tn.AppID.String()))

	got, err := s.GetSession(ctx, bare.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Audience, "an unaudienced session came back carrying an audience")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./store/memory/ -run 'TestStore/SessionAudienceRoundTrip' -v`

Expected: FAIL to compile, with `unknown field Audience in struct literal of type session.Session`.

- [ ] **Step 3: Add the field to the session entity**

In `session/session.go`, immediately after the `Roles []string` field (line 47), add:

```go
	// Audience holds the resource identifiers this session's access token was
	// granted for (RFC 8707). Empty means unrestricted, which is what every
	// session issued before resource indicators existed carries, and what any
	// client that sends no `resource` parameter still gets.
	//
	// This is the opaque-token half of the `aud` claim. A JWT carries its
	// audience inside the token and is checked without a store read, while an
	// opaque token is only a session lookup key, so the same value has to live
	// here for introspection and for the middleware audience check to see it.
	Audience []string `json:"aud,omitempty"`
```

- [ ] **Step 4: Add the field to the postgres model**

In `store/postgres/models.go`, after the `Roles` field at line 222:

```go
	// Audience is JSON for the same reason Roles is. A resource identifier is
	// a URI and may contain a comma, and these strings are compared to decide
	// whether a token is allowed to authenticate at all.
	Audience json.RawMessage `grove:"audience,type:jsonb"`
```

In the decode function, after the `Roles` block at line 304:

```go
	if len(m.Audience) > 0 {
		_ = json.Unmarshal(m.Audience, &s.Audience) //nolint:errcheck // best-effort decode
	}
```

In the encode function, after line 345:

```go
	m.Audience, _ = json.Marshal(s.Audience) //nolint:errcheck // best-effort encode
```

- [ ] **Step 5: Add the field to the sqlite model**

Apply the identical three edits to `store/sqlite/models.go` at lines 222, 304 and 345. The model shape is the same as postgres.

- [ ] **Step 6: Add the field to the mongo model**

In `store/mongo/models.go`, after the `Roles` field at line 209:

```go
	Audience []string `grove:"audience"                  bson:"audience,omitempty"`
```

In `toSessionModel`, directly after the `m.Roles` assignment at line 255:

```go
	// Always a non-nil slice, for the same reason as Roles above: grove writes
	// every mapped field regardless of the bson omitempty tag, so nil reaches
	// mongo as `audience: null` and the generated $jsonSchema rejects the
	// whole insert.
	m.Audience = append([]string{}, s.Audience...)
```

In `fromSessionModel`, after the `Roles` block at line 330:

```go
	if len(m.Audience) > 0 {
		s.Audience = append([]string(nil), m.Audience...)
	}
```

- [ ] **Step 7: Add the postgres migration**

In `store/postgres/migrations.go`, in the same `init()` block that holds `add_session_roles` (line 1179), register:

```go
		&migrate.Migration{
			Name:    "add_session_audience",
			Version: "20260824000040",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS audience JSONB NOT NULL DEFAULT '[]'::jsonb;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions DROP COLUMN IF EXISTS audience;
`)
				return err
			},
		},
```

- [ ] **Step 8: Add the sqlite migration**

In `store/sqlite/migrations.go`, in the same `init()` block:

```go
		&migrate.Migration{
			Name:    "add_session_audience",
			Version: "20260824000040",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions ADD COLUMN audience TEXT NOT NULL DEFAULT '[]';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions DROP COLUMN audience;
`)
				return err
			},
		},
```

- [ ] **Step 9: Add the mongo migration**

In `store/mongo/migrations.go`, following the shape of `add_session_principal_identity` at line 878. Mongo adds no column, but an existing deployment's generated validator has to be told about the new field or every write fails validation:

```go
		&migrate.Migration{
			Name:    "add_session_audience",
			Version: "20260824000040",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				mexec, ok := exec.(*mongomigrate.Executor)
				if !ok {
					return fmt.Errorf("expected mongomigrate executor, got %T", exec)
				}
				// The collection's $jsonSchema is generated from sessionModel,
				// so an existing deployment's validator does not know the
				// audience field and rejects every document carrying it.
				return mexec.RefreshValidator(ctx, (*sessionModel)(nil))
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				return nil
			},
		},
```

- [ ] **Step 10: Run the conformance suite**

Run: `go test ./store/memory/ ./store/sqlite/ -run 'TestStore/SessionAudienceRoundTrip' -v`

Expected: PASS on both. Postgres and mongo skip unless their env vars are set, which is expected locally.

- [ ] **Step 11: Run the whole store suite for regressions**

Run: `go test ./store/... ./session/...`

Expected: PASS.

- [ ] **Step 12: Commit**

```bash
git add session/session.go store/postgres/models.go store/postgres/migrations.go \
  store/sqlite/models.go store/sqlite/migrations.go \
  store/mongo/models.go store/mongo/migrations.go store/storetest/storetest.go
git commit -m "feat(session): carry the granted audience on a session"
```

---

### Task 2: Audience on token claims and in the JWT

**Files:**
- Modify: `tokenformat/format.go:19-29` (TokenClaims)
- Modify: `tokenformat/jwt.go:66-73` (customClaims), `:75-105` (generate), `:107-155` (validate)
- Test: `tokenformat/jwt_audience_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `tokenformat.TokenClaims.Audience []string`. `GenerateAccessToken` honours it; `ValidateAccessToken` returns it.

- [ ] **Step 1: Write the failing test**

Create `tokenformat/jwt_audience_test.go`:

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

func newTestJWT(t *testing.T, configuredAudience string) *tokenformat.JWT {
	t.Helper()
	j, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-key-not-a-real-secret-000000"),
		Issuer:        "https://auth.example.com",
		Audience:      configuredAudience,
	})
	require.NoError(t, err)
	return j
}

func TestJWT_Audience(t *testing.T) {
	base := tokenformat.TokenClaims{
		UserID:    "user_1",
		AppID:     "app_1",
		SessionID: "sess_1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tests := []struct {
		name             string
		configured       string
		perToken         []string
		wantAudience     []string
	}{
		{
			name:         "per-token audience wins over configured",
			configured:   "https://legacy.example.com",
			perToken:     []string{"https://api.example.com"},
			wantAudience: []string{"https://api.example.com"},
		},
		{
			name:         "configured audience is the fallback",
			configured:   "https://legacy.example.com",
			perToken:     nil,
			wantAudience: []string{"https://legacy.example.com"},
		},
		{
			name:         "no audience anywhere stays empty",
			configured:   "",
			perToken:     nil,
			wantAudience: nil,
		},
		{
			name:         "multiple resources round trip as an array",
			configured:   "",
			perToken:     []string{"https://api.example.com", "https://files.example.com"},
			wantAudience: []string{"https://api.example.com", "https://files.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := newTestJWT(t, tt.configured)

			claims := base
			claims.Audience = tt.perToken

			token, err := j.GenerateAccessToken(claims)
			require.NoError(t, err)

			got, err := j.ValidateAccessToken(token)
			require.NoError(t, err)
			assert.Equal(t, tt.wantAudience, got.Audience)
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./tokenformat/ -run TestJWT_Audience -v`

Expected: FAIL to compile, with `unknown field Audience in struct literal of type tokenformat.TokenClaims`.

- [ ] **Step 3: Add Audience to TokenClaims**

In `tokenformat/format.go`, inside `TokenClaims`, after the `Scopes` field:

```go
	// Audience holds the resource identifiers this token was granted for
	// (RFC 8707), emitted as the `aud` claim. Empty means unrestricted.
	//
	// This overrides JWTConfig.Audience when set. The config value is a single
	// static audience for a whole app, which predates resource indicators and
	// stays as the fallback so existing deployments are unaffected.
	Audience []string `json:"aud,omitempty"`
```

- [ ] **Step 4: Emit the claim**

In `tokenformat/jwt.go`, replace the audience block inside `GenerateAccessToken` (currently lines 100 to 102):

```go
	// A per-token audience is the resource the client actually asked for, so
	// it wins. The configured value is an app-wide default from before
	// resource indicators existed.
	switch {
	case len(claims.Audience) > 0:
		jwtClaims.Audience = jwt.ClaimStrings(claims.Audience)
	case j.config.Audience != "":
		jwtClaims.Audience = jwt.ClaimStrings{j.config.Audience}
	}
```

- [ ] **Step 5: Return the claim on validate**

In `tokenformat/jwt.go`, in the struct literal `ValidateAccessToken` returns, add after `Scopes`:

```go
		Audience:  []string(claims.Audience),
```

Note: `claims.Audience` here is the embedded `jwt.RegisteredClaims.Audience`, of type `jwt.ClaimStrings`. The conversion yields nil for an absent claim, which is what the "no audience anywhere" case asserts.

- [ ] **Step 6: Run the test**

Run: `go test ./tokenformat/ -run TestJWT_Audience -v`

Expected: PASS, four subtests.

- [ ] **Step 7: Run the package for regressions**

Run: `go test ./tokenformat/`

Expected: PASS. `ValidateAccessToken` must not start rejecting tokens on audience mismatch. The authorization server validates its own signature; deciding whether an audience is acceptable belongs to the resource server, which is Task 11.

- [ ] **Step 8: Commit**

```bash
git add tokenformat/format.go tokenformat/jwt.go tokenformat/jwt_audience_test.go
git commit -m "feat(tokenformat): emit a per-token aud claim"
```

---

### Task 3: Refresh carries the audience forward

**Files:**
- Modify: `service.go:581-592` (the JWT regeneration block inside the refresh path)
- Test: `refresh_audience_test.go` (create, at repo root, `package authsome_test`)

**Interfaces:**
- Consumes: `session.Session.Audience` (Task 1), `tokenformat.TokenClaims.Audience` (Task 2).
- Produces: nothing new. This closes a hole rather than adding surface.

- [ ] **Step 1: Understand what breaks without this**

`account.RefreshSession` mints a fresh opaque token, then `service.go` re-derives a JWT for apps configured for JWT format. That re-derivation builds a brand new `tokenformat.TokenClaims` and does not copy the audience, so the first refresh converts a token bound to one resource into an unrestricted one. The opaque path is already safe because `RotateSession` writes the whole `sessionModel` through `fromSession`, which includes the field from Task 1.

- [ ] **Step 2: Write the failing test**

Create `refresh_audience_test.go` at the repo root, modelled on `refresh_jwt_test.go`:

```go
package authsome_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/tokenformat"
)

// TestRefreshPreservesAudience proves a refresh does not widen a token.
//
// A token bound to one resource server must stay bound across rotation. If the
// audience is dropped on refresh the client keeps working and nobody notices,
// while the token silently becomes valid everywhere, which is a worse hole
// than the one resource indicators close.
//
// The opaque half already passes before the fix, because RotateSession writes
// the whole model. The JWT half fails, because service.go rebuilds the claims
// from scratch and never copies the audience across.
func TestRefreshPreservesAudience(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)

	eng, st := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "aud-refresh@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Aud User",
	})
	require.NoError(t, err)

	// Bind the session to one resource, the way issueTokens does.
	sess.Audience = []string{"https://api.example.com"}
	require.NoError(t, st.UpdateSession(ctx, sess))

	refreshed, err := eng.Refresh(ctx, sess.RefreshToken)
	require.NoError(t, err)

	assert.Equal(t, []string{"https://api.example.com"}, refreshed.Audience,
		"the rotated session lost its audience")

	claims, err := jwtFmt.ValidateAccessToken(refreshed.Token)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://api.example.com"}, claims.Audience,
		"the regenerated JWT lost its aud claim, so the refresh widened the token")
}
```

If `newTestEngine` returns only the engine in this package, fetch the store the same way the neighbouring tests do, and use whichever update method the store exposes (`UpdateSession` or `RotateSession`) to persist the audience before refreshing.

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test . -run TestRefreshPreservesAudience -v`

Expected: FAIL on the `aud` claim assertion. The returned session carries the audience but the regenerated JWT does not.

- [ ] **Step 4: Carry the audience into the regenerated JWT**

In `service.go`, in the `tokenformat.TokenClaims` literal at line 582, add:

```go
			Audience:  sess.Audience,
```

so the block reads:

```go
	tokFmt := e.TokenFormatForApp(sess.AppID.String())
	if tokFmt.Name() == "jwt" {
		jwtToken, genErr := tokFmt.GenerateAccessToken(tokenformat.TokenClaims{
			UserID:    sess.UserID.String(),
			AppID:     sess.AppID.String(),
			SessionID: sess.ID.String(),
			// Without this the first refresh turns a token bound to one
			// resource server into an unrestricted one, silently.
			Audience:  sess.Audience,
			IssuedAt:  sess.UpdatedAt,
			ExpiresAt: sess.ExpiresAt,
		})
```

- [ ] **Step 5: Run the test**

Run: `go test . -run TestRefreshPreservesAudience -v`

Expected: PASS.

- [ ] **Step 6: Run the refresh suite for regressions**

Run: `go test . -run 'TestRefresh' -v`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add service.go refresh_audience_test.go
git commit -m "fix(session): keep the audience across a refresh"
```

---

### Task 4: Resources on the OAuth2 models and stores

**Files:**
- Modify: `plugins/oauth2provider/models.go` (OAuth2Client, AuthorizationCode, DeviceCode)
- Modify: `plugins/oauth2provider/store_models.go` (three model structs, six converters)
- Modify: `plugins/oauth2provider/store_mongo.go:40-65,197+` (three bson structs and their converters)
- Modify: `plugins/oauth2provider/migrations.go` (postgres and sqlite groups)
- Test: `plugins/oauth2provider/store_resources_test.go` (create, `package oauth2provider`)

**Interfaces:**
- Consumes: nothing.
- Produces: `OAuth2Client.Resources []string`, `AuthorizationCode.Resources []string`, `DeviceCode.Resources []string`, all persisted on every backend.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/store_resources_test.go`:

```go
package oauth2provider

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// TestMemoryStore_ResourcesRoundTrip pins the resource allowlist and the
// per-code grant to storage. A backend that drops either one fails open: the
// client appears to have no allowlist (so every request is rejected as
// invalid_target) or the code carries no resources (so the issued token comes
// back unrestricted).
func TestMemoryStore_ResourcesRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	appID := id.NewAppID()
	want := []string{"https://api.example.com", "https://files.example.com"}

	client := &OAuth2Client{
		ID:        id.NewOAuth2ClientID(),
		AppID:     appID,
		Name:      "test",
		ClientID:  "client-abc",
		Scopes:    []string{"openid"},
		Resources: want,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateClient(ctx, client))

	gotClient, err := s.GetClient(ctx, "client-abc")
	require.NoError(t, err)
	assert.Equal(t, want, gotClient.Resources)

	code := &AuthorizationCode{
		ID:        id.NewAuthCodeID(),
		Code:      "code-abc",
		ClientID:  "client-abc",
		UserID:    id.NewUserID(),
		AppID:     appID,
		Scopes:    []string{"openid"},
		Resources: []string{"https://api.example.com"},
		ExpiresAt: time.Now().Add(time.Minute),
		CreatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAuthCode(ctx, code))

	gotCode, err := s.GetAuthCode(ctx, "code-abc")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://api.example.com"}, gotCode.Resources)

	dc := &DeviceCode{
		ID:         id.NewDeviceCodeID(),
		DeviceCode: "dev-abc",
		UserCode:   "BCDF-GHJK",
		ClientID:   "client-abc",
		AppID:      appID,
		Scopes:     []string{"openid"},
		Resources:  []string{"https://files.example.com"},
		Status:     DeviceCodeStatusPending,
		ExpiresAt:  time.Now().Add(time.Minute),
		CreatedAt:  time.Now(),
	}
	require.NoError(t, s.CreateDeviceCode(ctx, dc))

	gotDC, err := s.GetDeviceCodeByDeviceCode(ctx, "dev-abc")
	require.NoError(t, err)
	assert.Equal(t, []string{"https://files.example.com"}, gotDC.Resources)
}
```

If `NewMemoryStore` has a different constructor name in `store_memory.go`, use the existing one rather than renaming it.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestMemoryStore_ResourcesRoundTrip -v`

Expected: FAIL to compile, with `unknown field Resources`.

- [ ] **Step 3: Add the domain fields**

In `plugins/oauth2provider/models.go`, add to `OAuth2Client` after `Scopes`:

```go
	// Resources is the allowlist of resource identifiers this client may
	// request a token for (RFC 8707). Empty means the client may target
	// nothing, so a request naming any resource is rejected with
	// invalid_target. Deny by default: an empty list is the state every
	// existing client is in, and a client that never sends `resource` is
	// unaffected either way.
	Resources []string `json:"resources"`
```

Add to `AuthorizationCode` after `Scopes`:

```go
	// Resources is the audience the user authorized this code for. The token
	// endpoint may narrow it further but never widen it.
	Resources []string `json:"resources"`
```

Add to `DeviceCode` after `Scopes`:

```go
	// Resources is the audience this device authorization was granted for.
	Resources []string `json:"resources"`
```

- [ ] **Step 4: Add the SQL model fields and converters**

In `plugins/oauth2provider/store_models.go`:

Add to `oauth2ClientModel` after `Scopes`:

```go
	Resources json.RawMessage `grove:"resources,type:jsonb"`
```

Add to `authCodeModel` after `Scopes`:

```go
	Resources json.RawMessage `grove:"resources,type:jsonb"`
```

Add to `deviceCodeModel` after `Scopes`:

```go
	Resources json.RawMessage `grove:"resources,type:jsonb"`
```

In each of the three `to*` converters, decode alongside the existing scopes decode:

```go
	var resources []string
	if len(m.Resources) > 0 {
		_ = json.Unmarshal(m.Resources, &resources) //nolint:errcheck // best-effort decode
	}
```

and set `Resources: resources` in the returned struct.

In each of the three `from*` converters, encode alongside scopes:

```go
	resources, _ := json.Marshal(c.Resources) //nolint:errcheck // marshaling known types
	if len(resources) == 0 {
		resources = []byte("[]")
	}
```

and set `Resources: resources` in the returned model. Use the receiver name each function already uses (`c` for client and auth code, `dc` for device code).

- [ ] **Step 5: Add the mongo fields**

In `plugins/oauth2provider/store_mongo.go`, add to each of the three bson structs, next to the existing `Scopes` line:

```go
	Resources []string `bson:"resources"`
```

Then set the field in both directions in the converters that sit beside each struct. These structs are hand-written bson, not grove-mapped with a generated validator, so a nil slice is safe here and no `RefreshValidator` migration is needed.

- [ ] **Step 6: Add the plugin migrations**

In `plugins/oauth2provider/migrations.go`, register in the `PostgresMigrations` group:

```go
		&migrate.Migration{
			Name:    "add_oauth2_resources",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients
    ADD COLUMN IF NOT EXISTS resources JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE authsome_oauth2_auth_codes
    ADD COLUMN IF NOT EXISTS resources JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE authsome_oauth2_device_codes
    ADD COLUMN IF NOT EXISTS resources JSONB NOT NULL DEFAULT '[]'::jsonb;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients DROP COLUMN IF EXISTS resources;
ALTER TABLE authsome_oauth2_auth_codes DROP COLUMN IF EXISTS resources;
ALTER TABLE authsome_oauth2_device_codes DROP COLUMN IF EXISTS resources;
`)
				return err
			},
		},
```

And in the `SqliteMigrations` group, one `ADD COLUMN` per statement:

```go
		&migrate.Migration{
			Name:    "add_oauth2_resources",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				for _, stmt := range []string{
					`ALTER TABLE authsome_oauth2_clients ADD COLUMN resources TEXT NOT NULL DEFAULT '[]';`,
					`ALTER TABLE authsome_oauth2_auth_codes ADD COLUMN resources TEXT NOT NULL DEFAULT '[]';`,
					`ALTER TABLE authsome_oauth2_device_codes ADD COLUMN resources TEXT NOT NULL DEFAULT '[]';`,
				} {
					if _, err := exec.Exec(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				for _, stmt := range []string{
					`ALTER TABLE authsome_oauth2_clients DROP COLUMN resources;`,
					`ALTER TABLE authsome_oauth2_auth_codes DROP COLUMN resources;`,
					`ALTER TABLE authsome_oauth2_device_codes DROP COLUMN resources;`,
				} {
					if _, err := exec.Exec(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			},
		},
```

- [ ] **Step 7: Confirm the memory store needs no change**

`store_memory.go` holds the domain structs directly, so the new fields persist without edits. Read the file to confirm before moving on. If it copies fields one by one, add the three assignments.

- [ ] **Step 8: Run the test**

Run: `go test ./plugins/oauth2provider/ -run TestMemoryStore_ResourcesRoundTrip -v`

Expected: PASS.

- [ ] **Step 9: Run the package**

Run: `go test ./plugins/oauth2provider/`

Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add plugins/oauth2provider/models.go plugins/oauth2provider/store_models.go \
  plugins/oauth2provider/store_mongo.go plugins/oauth2provider/migrations.go \
  plugins/oauth2provider/store_resources_test.go
git commit -m "feat(oauth2): persist requested resources on clients, codes and device codes"
```

---

### Task 5: The resourceParams helper

**Files:**
- Create: `plugins/oauth2provider/resource.go`
- Test: `plugins/oauth2provider/resource_test.go` (`package oauth2provider`)

**Interfaces:**
- Consumes: nothing.
- Produces: `func resourceParams(r *http.Request) []string`.

- [ ] **Step 1: Understand why the struct binder cannot do this**

forge binds request structs through `go-utils/http.Ctx.BindRequest`. Two independent blockers:

1. `bindQueryParam` reads one value via `c.Query(name)`, which is `url.Values.Get`, so `?resource=a&resource=b` yields only `a`.
2. `setFieldValue` switches on String, Int, Uint, Float and Bool. A `[]string` hits the default branch and records `unsupported field type: slice`, which fails the entire request with a 400.

So a `[]string` tagged `query:"resource"` does not lose data quietly. It breaks `/authorize` for every caller. Read the raw request instead.

- [ ] **Step 2: Write the failing test**

Create `plugins/oauth2provider/resource_test.go`:

```go
package oauth2provider

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceParams(t *testing.T) {
	tests := []struct {
		name string
		req  func() *http.Request
		want []string
	}{
		{
			name: "single query value",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet,
					"/authorize?resource=https%3A%2F%2Fapi.example.com", nil)
			},
			want: []string{"https://api.example.com"},
		},
		{
			name: "repeated query values are all kept",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet,
					"/authorize?resource=https%3A%2F%2Fa.example.com&resource=https%3A%2F%2Fb.example.com", nil)
			},
			want: []string{"https://a.example.com", "https://b.example.com"},
		},
		{
			name: "absent parameter yields nothing",
			req: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/authorize?client_id=abc", nil)
			},
			want: nil,
		},
		{
			name: "repeated form values on a POST",
			req: func() *http.Request {
				body := "grant_type=authorization_code&resource=https%3A%2F%2Fa.example.com&resource=https%3A%2F%2Fb.example.com"
				r := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(body))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			},
			want: []string{"https://a.example.com", "https://b.example.com"},
		},
		{
			name: "query and form values are both collected on a POST",
			req: func() *http.Request {
				body := "resource=https%3A%2F%2Fb.example.com"
				r := httptest.NewRequest(http.MethodPost,
					"/token?resource=https%3A%2F%2Fa.example.com", strings.NewReader(body))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			},
			want: []string{"https://a.example.com", "https://b.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resourceParams(tt.req()))
		})
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestResourceParams -v`

Expected: FAIL to compile, with `undefined: resourceParams`.

- [ ] **Step 4: Write the helper**

Create `plugins/oauth2provider/resource.go`:

```go
package oauth2provider

import (
	"net/http"
)

// resourceParams extracts the repeatable RFC 8707 resource parameter.
//
// This cannot go through the struct binder. go-utils' bindQueryParam reads a
// single value via url.Values.Get, and setFieldValue has no reflect.Slice
// case, so a []string field tagged query:"resource" does not merely lose the
// second value, it fails the whole request with "unsupported field type".
// Reading the raw request is the only way to honour a parameter the RFC
// defines as repeatable.
//
// Query values come first so a GET works without touching the body. For a POST
// the form is parsed too, because RFC 6749 sends token-endpoint parameters as
// application/x-www-form-urlencoded. ParseForm merges the query string into
// r.Form, so only r.PostForm is read here to avoid returning each query value
// twice.
func resourceParams(r *http.Request) []string {
	if r == nil {
		return nil
	}

	var out []string
	out = append(out, r.URL.Query()["resource"]...)

	if r.Method == http.MethodPost {
		// A parse failure means an unreadable body, which the handler will
		// reject on its own terms. There is nothing to add here.
		_ = r.ParseForm() //nolint:errcheck // handler rejects a malformed body
		out = append(out, r.PostForm["resource"]...)
	}

	return out
}
```

- [ ] **Step 5: Run the test**

Run: `go test ./plugins/oauth2provider/ -run TestResourceParams -v`

Expected: PASS, five subtests.

- [ ] **Step 6: Commit**

```bash
git add plugins/oauth2provider/resource.go plugins/oauth2provider/resource_test.go
git commit -m "feat(oauth2): read the repeatable resource parameter off the raw request"
```

---

### Task 6: Validating a requested resource

**Files:**
- Modify: `plugins/oauth2provider/resource.go` (add `resolveResources`)
- Test: `plugins/oauth2provider/resource_test.go` (add `TestResolveResources`)

**Interfaces:**
- Consumes: `OAuth2Client.Resources` (Task 4), `newOAuth2Error` (exists, `plugin.go:422`).
- Produces: `func resolveResources(client *OAuth2Client, requested []string) ([]string, error)`.

- [ ] **Step 1: Write the failing test**

Append to `plugins/oauth2provider/resource_test.go`:

```go
func TestResolveResources(t *testing.T) {
	allowed := &OAuth2Client{
		Resources: []string{"https://api.example.com", "https://files.example.com"},
	}
	noAllowlist := &OAuth2Client{}

	tests := []struct {
		name      string
		client    *OAuth2Client
		requested []string
		want      []string
		wantErr   bool
	}{
		{
			name:      "no resource requested yields no audience",
			client:    allowed,
			requested: nil,
			want:      nil,
		},
		{
			name:      "a registered resource is granted",
			client:    allowed,
			requested: []string{"https://api.example.com"},
			want:      []string{"https://api.example.com"},
		},
		{
			name:      "two registered resources are both granted",
			client:    allowed,
			requested: []string{"https://api.example.com", "https://files.example.com"},
			want:      []string{"https://api.example.com", "https://files.example.com"},
		},
		{
			name:      "duplicates collapse and order is preserved",
			client:    allowed,
			requested: []string{"https://files.example.com", "https://api.example.com", "https://files.example.com"},
			want:      []string{"https://files.example.com", "https://api.example.com"},
		},
		{
			name:      "an unregistered resource is rejected",
			client:    allowed,
			requested: []string{"https://evil.example.com"},
			wantErr:   true,
		},
		{
			name:      "an empty allowlist rejects any request",
			client:    noAllowlist,
			requested: []string{"https://api.example.com"},
			wantErr:   true,
		},
		{
			name:      "an empty allowlist still allows an empty request",
			client:    noAllowlist,
			requested: nil,
			want:      nil,
		},
		{
			name:      "a relative URI is rejected",
			client:    allowed,
			requested: []string{"/api"},
			wantErr:   true,
		},
		{
			name:      "a URI carrying a fragment is rejected",
			client:    allowed,
			requested: []string{"https://api.example.com#section"},
			wantErr:   true,
		},
		{
			name:      "an empty string is rejected",
			client:    allowed,
			requested: []string{""},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveResources(tt.client, tt.requested)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid_target")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

Add `"github.com/stretchr/testify/require"` to the file's imports.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestResolveResources -v`

Expected: FAIL to compile, with `undefined: resolveResources`.

- [ ] **Step 3: Write the validator**

Append to `plugins/oauth2provider/resource.go`, and add `"fmt"`, `"net/url"` and `"github.com/xraph/authsome/apitypes"` only if needed (the error helper lives in `plugin.go`, same package, so no new import for it):

```go
// resolveResources validates the requested resource indicators against the
// client's allowlist and returns the audience to grant.
//
// This mirrors resolveScopes, with one deliberate difference. An empty scope
// request yields the client's whole registered set, because a scope names
// something the client already holds. An empty resource request yields
// nothing, because widening a token to every resource a client may target is
// the opposite of what RFC 8707 is for.
//
// A client with an empty allowlist may target nothing. That is the state every
// client registered before this existed is in, and it is why the deny is safe:
// such a client has never sent a resource parameter, so it never reaches the
// rejection.
func resolveResources(client *OAuth2Client, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}

	allowed := make(map[string]struct{}, len(client.Resources))
	for _, r := range client.Resources {
		allowed[r] = struct{}{}
	}

	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))

	for _, raw := range requested {
		if raw == "" {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				"resource must not be empty")
		}

		// RFC 8707 section 2: the value MUST be an absolute URI and MUST NOT
		// carry a fragment. A fragment never reaches a server, so two values
		// differing only after the # would name the same resource while
		// comparing as different audiences.
		u, err := url.Parse(raw)
		if err != nil || !u.IsAbs() {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				fmt.Sprintf("resource %q is not an absolute URI", raw))
		}
		if u.Fragment != "" || strings.Contains(raw, "#") {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				fmt.Sprintf("resource %q must not include a fragment", raw))
		}

		if _, ok := allowed[raw]; !ok {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				fmt.Sprintf("resource %q is not registered for this client", raw))
		}

		if _, dup := seen[raw]; dup {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}

	return out, nil
}
```

Imports for `resource.go` become `fmt`, `net/http`, `net/url`, `strings`.

- [ ] **Step 4: Check the error helper carries the code**

Read `newOAuth2Error` at `plugin.go:422` and `OAuth2HTTPError.Error()` at `:412`. The test asserts `err.Error()` contains `invalid_target`. If `Error()` returns only the description, assert on `ResponseBody()` instead of changing the helper, since other handlers depend on its current shape.

- [ ] **Step 5: Run the test**

Run: `go test ./plugins/oauth2provider/ -run TestResolveResources -v`

Expected: PASS, ten subtests.

- [ ] **Step 6: Commit**

```bash
git add plugins/oauth2provider/resource.go plugins/oauth2provider/resource_test.go
git commit -m "feat(oauth2): validate requested resources against the client allowlist"
```

---

### Task 7: Admin registration of the resource allowlist

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:329-348` (CreateClientRequest, CreateClientResponse), `:759-835` (handleCreateClient)
- Modify: `plugins/oauth2provider/dashboard.go:153`, `:232`
- Test: `plugins/oauth2provider/client_resources_test.go` (create, `package oauth2provider_test`)

**Interfaces:**
- Consumes: `OAuth2Client.Resources` (Task 4), `resolveResources` validation rules (Task 6).
- Produces: `CreateClientRequest.Resources []string`, `CreateClientResponse.Resources []string`.

- [ ] **Step 1: Understand why this task exists**

Deny by default means a client with an empty allowlist can target nothing. Without a way to populate the allowlist, the whole feature is unreachable, so registration comes before the flows that depend on it.

- [ ] **Step 2: Write the failing test**

Create `plugins/oauth2provider/client_resources_test.go`. Build the handler harness the way `authcode_test.go` does. Assert:

1. Creating a client with `resources: ["https://api.example.com"]` returns those resources in the response and stores them.
2. Creating a client with `resources: ["not-a-uri"]` returns 400.
3. Creating a client with `resources: ["https://api.example.com#frag"]` returns 400.
4. Creating a client with no `resources` field succeeds and stores an empty allowlist.

Case 4 is the backwards-compatibility guard: every existing caller omits the field.

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestCreateClientResources -v`

Expected: FAIL, with the request field unknown and resources absent from the response.

- [ ] **Step 4: Add the request and response fields**

In `plugin.go`, add to `CreateClientRequest` after `Scopes`:

```go
	Resources    []string `json:"resources,omitempty"`
```

Add to `CreateClientResponse` after `Scopes`:

```go
	Resources    []string `json:"resources"`
```

- [ ] **Step 5: Validate and store on create**

In `handleCreateClient`, after the `scopes` default block (around line 800), add:

```go
	// Validate the allowlist at registration so a malformed entry is caught
	// here rather than turning every later authorize request into an opaque
	// invalid_target. Same rules resolveResources applies at request time.
	for _, raw := range req.Resources {
		u, parseErr := url.Parse(raw)
		if raw == "" || parseErr != nil || !u.IsAbs() {
			return nil, forge.BadRequest(fmt.Sprintf("resource %q is not an absolute URI", raw))
		}
		if u.Fragment != "" || strings.Contains(raw, "#") {
			return nil, forge.BadRequest(fmt.Sprintf("resource %q must not include a fragment", raw))
		}
	}
```

Set `Resources: req.Resources` in the `OAuth2Client` literal, and `Resources: client.Resources` in the `CreateClientResponse` literal.

`plugin.go` already imports `fmt`, `net/url` and `strings`.

- [ ] **Step 6: Surface the field in the dashboard**

In `dashboard.go`, add `Resources: c.Resources` alongside the `Scopes` assignment at line 154, and `Resources: resources` alongside line 233. Parse the form input the same way the existing scopes input is parsed in that handler.

- [ ] **Step 7: Run the tests**

Run: `go test ./plugins/oauth2provider/ -run 'TestCreateClientResources|TestDashboard' -v`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add plugins/oauth2provider/plugin.go plugins/oauth2provider/dashboard.go \
  plugins/oauth2provider/client_resources_test.go
git commit -m "feat(oauth2): register a resource allowlist on a client"
```

---

### Task 8: The authorization endpoint accepts resource

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:434-527` (handleAuthorize), `:149-270` (route registration for `/authorize`)
- Test: `plugins/oauth2provider/authorize_resource_test.go` (create, `package oauth2provider_test`)

**Interfaces:**
- Consumes: `resourceParams` (Task 5), `resolveResources` (Task 6), `AuthorizationCode.Resources` (Task 4).
- Produces: authorization codes carrying `Resources`.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/authorize_resource_test.go`. It reuses the fixtures already defined in `authcode_test.go` in this same package: `newFixture`, `authorize`, `baseAuthorizeQuery`, `codeFrom` and the `confidentialID` / `registeredURI` constants.

```go
package oauth2provider_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

const (
	resAPI   = "https://api.example.com"
	resFiles = "https://files.example.com"
	resOther = "https://other.example.com"
)

// grantResources gives the confidential fixture client an allowlist. The
// fixture registers no resources, which is the deny-by-default state every
// existing client is in.
func grantResources(t *testing.T, st oauth2provider.Store, resources ...string) {
	t.Helper()
	c, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)
	c.Resources = resources
	require.NoError(t, st.CreateClient(context.Background(), c))
}

func TestAuthorizeResource(t *testing.T) {
	t.Run("a registered resource lands on the code", func(t *testing.T) {
		_, st, mux := newFixture(t)
		grantResources(t, st, resAPI)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)

		code := codeFrom(t, authorize(t, mux, q))

		stored, err := st.GetAuthCode(context.Background(), code)
		require.NoError(t, err)
		assert.Equal(t, []string{resAPI}, stored.Resources)
	})

	t.Run("two resources both land on the code", func(t *testing.T) {
		_, st, mux := newFixture(t)
		grantResources(t, st, resAPI, resFiles)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)
		q.Add("resource", resFiles)

		code := codeFrom(t, authorize(t, mux, q))

		stored, err := st.GetAuthCode(context.Background(), code)
		require.NoError(t, err)
		assert.Equal(t, []string{resAPI, resFiles}, stored.Resources,
			"a repeated resource parameter lost a value, which means it went through the struct binder")
	})

	t.Run("an unregistered resource is refused", func(t *testing.T) {
		_, st, mux := newFixture(t)
		grantResources(t, st, resAPI)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resOther)

		rec := authorize(t, mux, q)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
	})

	t.Run("an empty allowlist refuses any resource", func(t *testing.T) {
		_, _, mux := newFixture(t)

		q := baseAuthorizeQuery(confidentialID)
		q.Add("resource", resAPI)

		rec := authorize(t, mux, q)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid_target")
	})

	// The regression guard. Every client that exists today sends no resource
	// parameter and must be completely unaffected.
	t.Run("no resource parameter still authorizes", func(t *testing.T) {
		_, st, mux := newFixture(t)

		code := codeFrom(t, authorize(t, mux, baseAuthorizeQuery(confidentialID)))

		stored, err := st.GetAuthCode(context.Background(), code)
		require.NoError(t, err)
		assert.Empty(t, stored.Resources)
	})
}
```

If `CreateClient` on the memory store rejects a duplicate ID, add an `UpdateClient` to the `Store` interface in a preceding step, or build the fixture client with its resources already set instead of mutating it.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestAuthorizeResource -v`

Expected: FAIL. The code is created with no resources.

- [ ] **Step 3: Resolve resources in the handler**

In `handleAuthorize`, after the `resolveScopes` block (line 469) and before the PKCE checks, add:

```go
	// RFC 8707. Read off the raw request because the struct binder rejects a
	// []string query field outright; see resourceParams.
	resources, err := resolveResources(client, resourceParams(ctx.Request()))
	if err != nil {
		return nil, err
	}
```

Then set `Resources: resources` in the `AuthorizationCode` literal at line 500.

- [ ] **Step 4: Document the parameter in OpenAPI**

In the `/authorize` route registration, add:

```go
		forge.WithParameter("resource", "query",
			"RFC 8707 resource indicator. Repeatable. Absolute URI, no fragment.",
			false, "https://api.example.com"),
```

`WithParameter` likely cannot express `style: form, explode: true` for an array, so the generated SDKs will show a single string. Leave that for the SDK regeneration pass; the server honours repeats regardless.

- [ ] **Step 5: Run the tests**

Run: `go test ./plugins/oauth2provider/ -run 'TestAuthorize' -v`

Expected: PASS, including the pre-existing authorize tests.

- [ ] **Step 6: Commit**

```bash
git add plugins/oauth2provider/plugin.go plugins/oauth2provider/authorize_resource_test.go
git commit -m "feat(oauth2): accept resource indicators at the authorization endpoint"
```

---

### Task 9: Issuance stamps the audience

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:881-932` (issueTokens), `:934-968` (issueClientToken)
- Test: covered by Task 10's end-to-end assertions plus a direct unit assertion here

**Interfaces:**
- Consumes: `session.Session.Audience` (Task 1), `tokenformat.TokenClaims.Audience` (Task 2).
- Produces: `issueTokens(ctx, client, userID, appID, scopes, resources []string)` and `issueClientToken(ctx, client, resources []string)`. Both signatures gain a trailing `resources []string`. Every caller must be updated in the same commit or the package will not build.

- [ ] **Step 1: Change the issueTokens signature and body**

Replace the signature at line 881:

```go
func (p *Plugin) issueTokens(ctx context.Context, _ *OAuth2Client, userID id.UserID, appID id.AppID, scopes []string, resources []string) (*TokenResponse, error) {
```

After `sess` is created and its `EnvID` bound, add:

```go
	// The opaque half of the audience. An opaque access token carries no
	// claims, so this is the only place introspection and the middleware
	// audience check can read it from.
	sess.Audience = resources
```

In the JWT block, add to the `tokenformat.TokenClaims` literal:

```go
				Audience:  resources,
```

- [ ] **Step 2: Change issueClientToken the same way**

Replace the signature at line 934:

```go
func (p *Plugin) issueClientToken(ctx context.Context, client *OAuth2Client, resources []string) (*TokenResponse, error) {
```

After the `EnvID` binding, add `sess.Audience = resources`.

`issueClientToken` does not currently generate a JWT even for JWT-format apps, which is a pre-existing gap. Do not fix it here. Stamping the session is enough for introspection and for the opaque middleware path, and widening the change would tangle two concerns.

- [ ] **Step 3: Update every caller**

Four call sites, all in `plugin.go`:

- `handleAuthorizationCodeGrant` (line 668): `p.issueTokens(ctx.Context(), client, authCode.UserID, authCode.AppID, authCode.Scopes, authCode.Resources)`. Task 10 narrows this further.
- `handleClientCredentialsGrant` (line 690): `p.issueClientToken(ctx.Context(), client, nil)`. Task 10 wires the real value.
- `handleDeviceCodeGrant` (line 1110): `p.issueTokens(ctx.Context(), client, dc.UserID, dc.AppID, dc.Scopes, dc.Resources)`.
- Any other call the compiler flags.

- [ ] **Step 4: Verify the package builds**

Run: `go build ./plugins/oauth2provider/`

Expected: success. A failure here names a caller you missed.

- [ ] **Step 5: Run the package**

Run: `go test ./plugins/oauth2provider/`

Expected: PASS. The authorize test from Task 8 now proves the code carries resources, and the device path carries them through.

- [ ] **Step 6: Commit**

```bash
git add plugins/oauth2provider/plugin.go
git commit -m "feat(oauth2): stamp the granted audience onto issued tokens"
```

---

### Task 10: The token endpoint narrows, and the remaining grants

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:288-297` (TokenRequest), `:592-669` (auth code grant), `:671-690` (client credentials), `:972-1040` (device authorize), `:1042-1112` (device code grant)
- Test: `plugins/oauth2provider/token_resource_test.go` (create, `package oauth2provider_test`)

**Interfaces:**
- Consumes: everything from Tasks 4 through 9.
- Produces: `narrowResources(granted, requested []string) ([]string, error)` in `resource.go`.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/token_resource_test.go`. Assert:

1. Authorize with two resources, then redeem with one: the token's audience is the one requested at the token endpoint.
2. Authorize with one resource, then redeem asking for a different one: 400 `invalid_target`.
3. Authorize with two resources, then redeem with no `resource`: the token carries both.
4. Client credentials with a registered resource: the session carries it.
5. Device authorize with a resource, then poll after approval: the token carries it.
6. A JSON token request carrying `"resource": ["https://api.example.com"]` works, since that path binds through `encoding/json` rather than `resourceParams`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestTokenResource -v`

Expected: FAIL on the narrowing cases.

- [ ] **Step 3: Add the narrowing helper**

Append to `plugins/oauth2provider/resource.go`:

```go
// narrowResources restricts an already-granted audience to the subset the
// token request asked for (RFC 8707 section 2.2).
//
// A token request may narrow what the user authorized but must never widen it.
// An empty request inherits the whole granted set, which is what a client that
// only sends `resource` at the authorization endpoint does.
func narrowResources(granted, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return granted, nil
	}

	grantedSet := make(map[string]struct{}, len(granted))
	for _, g := range granted {
		grantedSet[g] = struct{}{}
	}

	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))
	for _, r := range requested {
		if _, ok := grantedSet[r]; !ok {
			return nil, newOAuth2Error(http.StatusBadRequest, "invalid_target",
				fmt.Sprintf("resource %q was not granted by this authorization", r))
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out, nil
}
```

- [ ] **Step 4: Add the JSON field to TokenRequest**

In `plugin.go`, add to `TokenRequest`:

```go
	// No form tag and no query tag. The binder cannot decode a slice from
	// either, so the form-encoded case is handled by resourceParams; this
	// field only carries the JSON body case, which BindJSON decodes natively.
	Resource []string `json:"resource,omitempty"`
```

- [ ] **Step 5: Add a request-level resource reader**

Append to `resource.go`:

```go
// tokenRequestResources collects the resource indicators from a token request.
//
// A form-encoded request populates resourceParams and leaves the JSON field
// empty; a JSON request does the reverse, so in practice only one of the two
// ever carries values. If both do, the raw request wins and the two sets are
// never concatenated.
func tokenRequestResources(r *http.Request, req *TokenRequest) []string {
	if raw := resourceParams(r); len(raw) > 0 {
		return raw
	}
	return req.Resource
}
```

- [ ] **Step 6: Narrow in the authorization code grant**

In `handleAuthorizationCodeGrant`, after the code is consumed (line 665) and before issuance:

```go
	resources, err := narrowResources(authCode.Resources, tokenRequestResources(ctx.Request(), req))
	if err != nil {
		return nil, err
	}

	return p.issueTokens(ctx.Context(), client, authCode.UserID, authCode.AppID, authCode.Scopes, resources)
```

- [ ] **Step 7: Wire client credentials**

In `handleClientCredentialsGrant`, before issuance:

```go
	// There is no prior authorization to narrow against here, so the client's
	// own allowlist is the only bound.
	resources, err := resolveResources(client, tokenRequestResources(ctx.Request(), req))
	if err != nil {
		return nil, err
	}

	return p.issueClientToken(ctx.Context(), client, resources)
```

- [ ] **Step 8: Wire the device authorization endpoint**

In `handleDeviceAuthorize`, resolve against the client and store on the device code:

```go
	resources, err := resolveResources(client, resourceParams(ctx.Request()))
	if err != nil {
		return nil, err
	}
```

Set `Resources: resources` in the `DeviceCode` literal.

- [ ] **Step 9: Narrow in the device code grant**

In `handleDeviceCodeGrant`, in the `DeviceCodeStatusAuthorized` branch, replace the issuance call:

```go
		resources, resErr := narrowResources(dc.Resources, tokenRequestResources(ctx.Request(), req))
		if resErr != nil {
			return nil, resErr
		}

		return p.issueTokens(ctx.Context(), client, dc.UserID, dc.AppID, dc.Scopes, resources)
```

- [ ] **Step 10: Document the parameter on the three POST routes**

Add the same `forge.WithParameter("resource", "query", ...)` used in Task 8 to the `/token` and `/device/authorize` registrations, with the description noting it may also be sent in the form body.

- [ ] **Step 11: Run the tests**

Run: `go test ./plugins/oauth2provider/ -v`

Expected: PASS, including every pre-existing test in the package.

- [ ] **Step 12: Commit**

```bash
git add plugins/oauth2provider/plugin.go plugins/oauth2provider/resource.go \
  plugins/oauth2provider/token_resource_test.go
git commit -m "feat(oauth2): honour resource indicators on every grant"
```

---

### Task 11: Middleware refuses a token audienced elsewhere

**Files:**
- Modify: `middleware/auth.go:132-149` (SessionBindingConfig), `:467+` (trySessionAuth), `:352+` (tryJWTAuth)
- Test: `middleware/auth_audience_test.go` (create, `package middleware_test`)

**Interfaces:**
- Consumes: `session.Session.Audience` (Task 1), `tokenformat.TokenClaims.Audience` (Task 2).
- Produces: `middleware.SessionBindingConfig.ExpectedAudienceResolver func(context.Context) []string`, and `func audienceAllowed(tokenAudience, expected []string) bool`.

- [ ] **Step 1: Write the failing test**

Create `middleware/auth_audience_test.go`. Model the harness on `auth_test.go` and `auth_jwt_test.go`. Cover both paths with the same table:

```go
	tests := []struct {
		name           string
		tokenAudience  []string
		expected       []string
		wantAuthorized bool
	}{
		{
			name:           "no resolver configured, audienced token still passes",
			tokenAudience:  []string{"https://other.example.com"},
			expected:       nil,
			wantAuthorized: true,
		},
		{
			name:           "unaudienced token passes an audience check",
			tokenAudience:  nil,
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "matching audience passes",
			tokenAudience:  []string{"https://api.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "one of several audiences matching is enough",
			tokenAudience:  []string{"https://files.example.com", "https://api.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: true,
		},
		{
			name:           "disjoint audience is refused",
			tokenAudience:  []string{"https://other.example.com"},
			expected:       []string{"https://api.example.com"},
			wantAuthorized: false,
		},
	}
```

Run the table twice, once against an opaque session token and once against a JWT, asserting the handler saw an authenticated user or did not.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./middleware/ -run TestAuthMiddleware_Audience -v`

Expected: FAIL to compile, with `unknown field ExpectedAudienceResolver`.

- [ ] **Step 3: Add the config field**

In `middleware/auth.go`, inside `SessionBindingConfig`:

```go
	// ExpectedAudienceResolver returns the resource identifiers this
	// deployment answers to for the current request's app. Nil, or an empty
	// result, disables the check.
	//
	// Per app rather than per process: two apps in one deployment are two
	// different resources, and a token minted for one must not authenticate at
	// the other. This runs per request on the same path as
	// CookieNameResolver, so it should read a cached setting rather than hit
	// the database.
	ExpectedAudienceResolver func(context.Context) []string
```

- [ ] **Step 4: Add the comparison helper**

In `middleware/auth.go`, near the other unexported helpers:

```go
// audienceAllowed reports whether a token may be used at this resource.
//
// An empty expected set means the deployment has not declared what it answers
// to, so no check is possible. An empty token audience means an unrestricted
// token, which every token issued before RFC 8707 support carries, so it
// passes. Anything else has to intersect.
func audienceAllowed(tokenAudience, expected []string) bool {
	if len(expected) == 0 || len(tokenAudience) == 0 {
		return true
	}
	want := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		want[e] = struct{}{}
	}
	for _, a := range tokenAudience {
		if _, ok := want[a]; ok {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Enforce on the opaque path**

In `trySessionAuth`, after the cross-tenant guard and before the IP binding check:

```go
	if bindCfg.ExpectedAudienceResolver != nil {
		expected := bindCfg.ExpectedAudienceResolver(ctx.Context())
		if !audienceAllowed(sess.Audience, expected) {
			logger.Warn("auth middleware: session audience mismatch",
				log.String("session_id", sess.ID.String()),
			)
			return false
		}
	}
```

Do not log the audience values themselves. They are resource URIs supplied by the caller and land in logs verbatim.

- [ ] **Step 6: Enforce on the JWT path**

In `tryJWTAuth`, after the `requestAppIDMismatch` guard:

```go
	if bindCfg.ExpectedAudienceResolver != nil {
		expected := bindCfg.ExpectedAudienceResolver(ctx.Context())
		if !audienceAllowed(claims.Audience, expected) {
			logger.Warn("auth middleware: JWT audience mismatch",
				log.String("session_id", claims.SessionID),
			)
			return false
		}
	}
```

- [ ] **Step 7: Run the tests**

Run: `go test ./middleware/ -run TestAuthMiddleware_Audience -v`

Expected: PASS, ten subtests (five cases on each path).

- [ ] **Step 8: Run the middleware package**

Run: `go test ./middleware/`

Expected: PASS. Every existing test leaves `ExpectedAudienceResolver` nil, so behaviour is unchanged.

- [ ] **Step 9: Commit**

```bash
git add middleware/auth.go middleware/auth_audience_test.go
git commit -m "feat(middleware): refuse a token audienced at another resource"
```

---

### Task 12: Advertise support and expose the audience

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:311-327` (DiscoveryResponse), `:731-757` (handleDiscovery)
- Modify: `api/requests.go:666-675` (IntrospectResponse)
- Modify: `api/introspect_handler.go:52-64` (JWT branch), `:88-101` (opaque branch)
- Test: `plugins/oauth2provider/discovery_resource_test.go`, `api/introspect_audience_test.go` (create)

**Interfaces:**
- Consumes: `session.Session.Audience` (Task 1), `tokenformat.TokenClaims.Audience` (Task 2).
- Produces: `DiscoveryResponse.ResourceIndicatorsSupported bool`, `IntrospectResponse.Audience []string`.

- [ ] **Step 1: Write the failing tests**

For discovery, assert `GET /.well-known/oauth-authorization-server` returns `resource_indicators_supported: true`. Use the existing discovery test as the model.

For introspection, assert that a JWT carrying `aud` and an opaque session carrying `Audience` both introspect with the same `aud` array, and that a token with no audience omits the field.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./plugins/oauth2provider/ -run TestDiscoveryResourceIndicators -v && go test ./api/ -run TestIntrospectAudience -v`

Expected: FAIL, fields absent.

- [ ] **Step 3: Add the discovery field**

In `plugin.go`, add to `DiscoveryResponse`:

```go
	// ResourceIndicatorsSupported advertises RFC 8707.
	//
	// This name is not registered. RFC 8707 registers the `resource`
	// parameter and the `invalid_target` error and defines no discovery
	// metadata at all, and the RFC 8414 IANA registry has no entry for it.
	// It is the convention that came out of the MCP ecosystem and it is what
	// clients look for, so do not read it as standardised.
	ResourceIndicatorsSupported bool `json:"resource_indicators_supported"`
```

Set `ResourceIndicatorsSupported: true` in the literal `handleDiscovery` returns.

- [ ] **Step 4: Add the introspection field**

In `api/requests.go`, add to `IntrospectResponse` after `SessionID`:

```go
	Audience  []string        `json:"aud,omitempty" description:"Resource identifiers this token is valid for (RFC 8707)"`
```

- [ ] **Step 5: Populate it on both paths**

In `api/introspect_handler.go`, add `Audience: claims.Audience,` to the JWT branch's `IntrospectResponse` literal, and `Audience: sess.Audience,` to the opaque branch's literal.

- [ ] **Step 6: Run the tests**

Run: `go test ./plugins/oauth2provider/ ./api/ -run 'TestDiscoveryResourceIndicators|TestIntrospectAudience' -v`

Expected: PASS.

- [ ] **Step 7: Run the package suites**

Run: `go test ./plugins/oauth2provider/ ./api/ ./middleware/`

Expected: PASS. The full-tree run happens at the end of Task 13.

- [ ] **Step 8: Commit**

```bash
git add plugins/oauth2provider/plugin.go plugins/oauth2provider/discovery_resource_test.go \
  api/requests.go api/introspect_handler.go api/introspect_audience_test.go
git commit -m "feat(oauth2): advertise resource indicators and expose aud on introspection"
```

---

### Task 13: Wire the expected audience from app settings

**Files:**
- Modify: `session_settings.go:233-246` (add the setting), `:383` (register it)
- Modify: `engine.go:246-250` (bindCfg), and add `resolveExpectedAudience` beside `resolveSessionCookieName` at `:346`
- Test: `engine_audience_test.go` (create, `package authsome_test`)

**Interfaces:**
- Consumes: `middleware.SessionBindingConfig.ExpectedAudienceResolver` (Task 11).
- Produces: `SettingResourceIdentifier`, and `func (e *Engine) resolveExpectedAudience(ctx context.Context) []string`.

- [ ] **Step 1: Understand why this task exists**

Task 11 added the hook but nothing fills it, so the check never runs. This is the task that makes the enforcement reachable, and it is the last piece of the spec.

- [ ] **Step 2: Define the setting**

In `session_settings.go`, in the `var` block that holds `SettingCookieName` (line 235):

```go
	// SettingResourceIdentifier is the resource identifier this app answers to
	// as an OAuth2 resource server (RFC 8707). Empty disables the audience
	// check, which is the default and preserves existing behaviour.
	//
	// Per app rather than per deployment: two apps in one deployment are two
	// different resources, and a token minted for one must not authenticate at
	// the other.
	SettingResourceIdentifier = settings.Define("session.resource_identifier", "",
		settings.WithDisplayName("Resource Identifier"),
		settings.WithDescription("Absolute URI this app answers to as an OAuth2 resource server. Empty disables audience checking."),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldText),
		settings.WithHelpText("When set, a token whose aud names a different resource is refused. Example: https://api.example.com"),
		settings.WithOrder(160),
	)
```

Pick an `WithOrder` value that does not collide with the ones already in that category, and move it to a more fitting category than "Cookie Configuration" if one exists in the file.

- [ ] **Step 3: Register the setting**

In `session_settings.go`, beside the existing registration at line 383:

```go
	if err := settings.RegisterTyped(m, "session", SettingResourceIdentifier); err != nil {
		return err
	}
```

- [ ] **Step 4: Write the resolver**

In `engine.go`, directly after `resolveSessionCookieName` (line 360):

```go
// resolveExpectedAudience returns the resource identifiers this deployment
// answers to for the request's app, or nil to disable the check.
//
// This mirrors resolveSessionCookieName exactly, including the fail-open on a
// settings error. Failing closed here would turn a settings outage into a
// total authentication outage, and an audience check is a second line of
// defence behind the token's own signature.
func (e *Engine) resolveExpectedAudience(ctx context.Context) []string {
	mgr := e.Settings()
	if mgr == nil {
		return nil
	}
	opts := settings.ResolveOpts{}
	if appID, ok := middleware.AppIDFrom(ctx); ok {
		opts.AppID = appID.String()
	}
	identifier, err := settings.Get(ctx, mgr, SettingResourceIdentifier, opts)
	if err != nil || identifier == "" {
		return nil
	}
	return []string{identifier}
}
```

- [ ] **Step 5: Wire it into the middleware config**

In `engine.go`, in `buildAuthMiddleware` at line 246:

```go
	bindCfg := middleware.SessionBindingConfig{
		CookieNameResolver:       e.resolveSessionCookieName,
		JWTSessionChecker:        e.jwtSessionChecker,
		ExpectedAudienceResolver: e.resolveExpectedAudience,
	}
```

- [ ] **Step 6: Write the test**

Create `engine_audience_test.go` (`package authsome_test`). Using `newTestEngine` and `testAppID` as `refresh_jwt_test.go` does:

1. With the setting unset, sign up, stamp the session with `Audience: []string{"https://other.example.com"}`, and assert a request carrying that token still authenticates. Unset means no check.
2. Set `session.resource_identifier` to `https://api.example.com` for the app, then assert a token audienced at `https://other.example.com` no longer authenticates, and one audienced at `https://api.example.com` does.
3. With the setting set, assert a session carrying no audience still authenticates. This is the regression guard for every existing session.

- [ ] **Step 7: Run the tests**

Run: `go test . -run TestEngineExpectedAudience -v`

Expected: PASS.

- [ ] **Step 8: Run the full suite**

Run: `go test ./...`

Expected: PASS. Every app leaves the new setting empty by default, so nothing changes for an existing deployment.

- [ ] **Step 9: Commit**

```bash
git add session_settings.go engine.go engine_audience_test.go
git commit -m "feat(auth): resolve the expected token audience from app settings"
```

---

## Verification

After Task 12, confirm the whole feature end to end before calling it done.

- [ ] `go build ./...` succeeds.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` is clean.
- [ ] Point `AUTHSOME_MONGO_URI` at a live mongo and run `go test ./store/mongo/ -run 'TestStore/SessionAudienceRoundTrip' -v`. This is the only way the null-array trap surfaces, and skipping it is how `9116564` reached main.
- [ ] Confirm no `[]string` field anywhere in the diff carries a `query:` or `form:` struct tag.
- [ ] Confirm migration versions: `20260824000040` in the core group, `20260824000001` in `authsome-oauth2`, and neither duplicated. Run `grep -rn '20260824' store/*/migrations.go plugins/*/migrations.go` and read the result.
