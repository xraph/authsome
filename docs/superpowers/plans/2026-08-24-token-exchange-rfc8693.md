# RFC 8693 Token Exchange Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the `urn:ietf:params:oauth:grant-type:token-exchange` grant to the oauth2provider plugin so a client can trade a broad credential for a narrow, short-lived, fully audited one.

**Architecture:** Scopes and an actor chain become stored properties of `session.Session`, because an OAuth access token in this codebase *is* a session row. A new policy table in the oauth2provider plugin declares who may exchange for whom. The grant handler intersects requested scopes against a three-way ceiling, caps TTL by the subject's own remaining life, appends an actor to the chain, and writes one security event per attempt whether it succeeds or fails.

**Tech Stack:** Go 1.26, forge router, grove migrations and ORM, postgres + sqlite + mongo + in-memory stores, testify.

**Spec:** `docs/superpowers/specs/2026-08-24-token-exchange-rfc8693-design.md`

## Global Constraints

- Go 1.26.0 (`go.mod`). No new third-party dependencies.
- Run tests with `go test ./...`. Run `make check` (fmt, vet, lint) before every commit.
- Four store drivers must stay in parity: postgres, sqlite, mongo, memory. A change to `session.Session` touches `store/postgres/models.go`, `store/sqlite/models.go`, `store/mongo/models.go` and the memory store.
- Mongo slices must never be written as nil. grove writes every mapped field regardless of the bson `omitempty` tag, so a nil slice reaches mongo as `null` and fails to decode. See commit `9116564`, which fixed exactly this for `Roles`. The same rule applies to `Actors`.
- Mongo has no migration group. `Plugin.MigrationGroups` returns nil for it and the schema is implicit in the models.
- Postgres and sqlite migrations are registered in `init()` blocks with a `Name` and a `Version` timestamp string. Versions must sort after existing ones. Use the `202608240000NN` series.
- Imports are grouped stdlib / third-party / `github.com/xraph/authsome` and formatted with `goimports -local github.com/xraph/authsome`.
- Do not drop the `impersonated_by` column in this plan. It is a later release. Task 4 keeps writing it for rolling-deploy compatibility.

---

### Task 1: Actor type and session scope/actor fields

**Files:**
- Modify: `session/session.go:28` (remove nothing yet; add fields after `Roles`)
- Modify: `store/postgres/models.go:216`, `:303`, `:344`
- Modify: `store/sqlite/models.go:217`, `:304`, `:345`
- Modify: `store/mongo/models.go:205`, `:243`, `:316`
- Modify: `store/postgres/migrations.go` (new migration in the `init()` block)
- Modify: `store/sqlite/migrations.go` (new migration in the `init()` block)
- Test: `session/session_test.go` (create)
- Test: `store/memory/session_actors_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `session.Actor` struct with fields `Subject string`, `Kind string`, `Mode string`, `At time.Time`; constants `session.KindUser`, `session.KindServiceAccount`, `session.KindOAuthClient`, `session.ModeDelegation`, `session.ModeImpersonation`; fields `session.Session.Scopes []string` and `session.Session.Actors []Actor`.

- [ ] **Step 1: Write the failing test**

Create `session/session_test.go`:

```go
package session_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/session"
)

func TestActorJSONRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	in := []session.Actor{
		{Subject: "aoac_abc", Kind: session.KindOAuthClient, Mode: session.ModeDelegation, At: now},
		{Subject: "usr_123", Kind: session.KindUser, Mode: session.ModeImpersonation, At: now},
	}

	raw, err := json.Marshal(in)
	require.NoError(t, err)

	var out []session.Actor
	require.NoError(t, json.Unmarshal(raw, &out))
	assert.Equal(t, in, out)
}

func TestActorJSONFieldNames(t *testing.T) {
	raw, err := json.Marshal(session.Actor{
		Subject: "usr_1", Kind: session.KindUser, Mode: session.ModeDelegation,
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Contains(t, m, "sub")
	assert.Contains(t, m, "kind")
	assert.Contains(t, m, "mode")
}

func TestSessionCarriesScopesAndActors(t *testing.T) {
	s := session.Session{
		Scopes: []string{"invoices:read"},
		Actors: []session.Actor{{Subject: "aoac_x", Kind: session.KindOAuthClient, Mode: session.ModeDelegation}},
	}
	assert.Equal(t, []string{"invoices:read"}, s.Scopes)
	require.Len(t, s.Actors, 1)
	assert.Equal(t, session.ModeDelegation, s.Actors[0].Mode)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./session/ -run TestActor -v`
Expected: FAIL, compile error `undefined: session.Actor`.

- [ ] **Step 3: Add the Actor type and the two session fields**

In `session/session.go`, add these constants and the `Actor` type above the `Session` struct:

```go
// Principal kinds that may appear in an Actor.
const (
	KindUser           = "user"
	KindServiceAccount = "service_account"
	KindOAuthClient    = "oauth_client"
)

// Actor modes. Delegation keeps both parties visible in the issued token;
// impersonation drops the actor from the token but never from this record.
const (
	ModeDelegation    = "delegation"
	ModeImpersonation = "impersonation"
)

// Actor is one party in a delegation chain (RFC 8693 `act`).
type Actor struct {
	// Subject is the principal id: a user, service account or oauth client.
	Subject string `json:"sub"`
	// Kind is one of KindUser, KindServiceAccount, KindOAuthClient.
	Kind string `json:"kind"`
	// Mode is ModeDelegation or ModeImpersonation. It is recorded here even
	// when the issued token omits the `act` claim, because RFC 8693 encodes
	// impersonation by absence and an absent claim cannot feed an audit trail.
	Mode string `json:"mode"`
	// At is when this actor entered the chain.
	At time.Time `json:"at"`
}
```

Inside the `Session` struct, immediately after the `Roles []string` field, add:

```go
	// Scopes holds the OAuth scopes this session was issued with, stamped at
	// issuance. Same trade as Roles: authoritative for what this token may
	// do, stale with respect to anything granted afterwards. Empty means the
	// session was not minted through a scoped grant, which is not the same as
	// "may do anything" — see the ceiling rule in the token exchange handler.
	Scopes []string `json:"scopes,omitempty"`

	// Actors is the delegation chain (RFC 8693 `act`). Actors[0] is the
	// immediate actor and later elements are the parties further back. An
	// empty chain means the principal authenticated directly.
	Actors []Actor `json:"actors,omitempty"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./session/ -v`
Expected: PASS.

- [ ] **Step 5: Add the postgres model columns**

In `store/postgres/models.go`, in `SessionModel` immediately after the `Roles` field:

```go
	// Scopes and Actors are JSON for the same reason Roles is: these values
	// are read back as authorization and audit input, so the encoding must
	// not be able to invent or merge a member.
	Scopes json.RawMessage `grove:"scopes,type:jsonb"`
	Actors json.RawMessage `grove:"actors,type:jsonb"`
```

In `toSession`, immediately after the existing `if len(m.Roles) > 0 { ... }` block:

```go
	if len(m.Scopes) > 0 {
		_ = json.Unmarshal(m.Scopes, &s.Scopes) //nolint:errcheck // best-effort decode
	}
	if len(m.Actors) > 0 {
		_ = json.Unmarshal(m.Actors, &s.Actors) //nolint:errcheck // best-effort decode
	}
```

In `fromSession`, immediately after the existing `m.Roles, _ = json.Marshal(s.Roles)` line:

```go
	// Always encoded for the same reason Roles is: the columns are NOT NULL
	// and json.RawMessage cannot scan a NULL back.
	m.Scopes, _ = json.Marshal(s.Scopes) //nolint:errcheck // best-effort encode
	m.Actors, _ = json.Marshal(s.Actors) //nolint:errcheck // best-effort encode
```

- [ ] **Step 6: Add the sqlite model columns**

Apply exactly the same three edits to `store/sqlite/models.go`. The field tags, the `toSession` decode block and the `fromSession` encode block are identical to Step 5, because the sqlite model already mirrors postgres for `Roles`.

- [ ] **Step 7: Add the mongo model columns**

In `store/mongo/models.go`, in `SessionModel` immediately after the `Roles` field:

```go
	Scopes []string  `grove:"scopes"  bson:"scopes,omitempty"`
	Actors []Actor   `grove:"actors"  bson:"actors,omitempty"`
```

Add the mongo-side actor model above `SessionModel`:

```go
// Actor mirrors session.Actor for mongo. Declared here rather than embedding
// the domain type so the bson field names stay owned by the store layer.
type Actor struct {
	Subject string    `bson:"sub"`
	Kind    string    `bson:"kind"`
	Mode    string    `bson:"mode"`
	At      time.Time `bson:"at"`
}
```

In `fromSession`, alongside the existing non-nil `Roles` assignment:

```go
	// Non-nil for the same reason Roles is: grove writes every mapped field
	// whatever the bson omitempty tag says, so a nil slice reaches mongo as
	// null and fails to decode on the way back. See commit 9116564.
	m.Scopes = append([]string{}, s.Scopes...)
	m.Actors = make([]Actor, 0, len(s.Actors))
	for _, a := range s.Actors {
		m.Actors = append(m.Actors, Actor{Subject: a.Subject, Kind: a.Kind, Mode: a.Mode, At: a.At})
	}
```

In `toSession`, after the roles decode:

```go
	s.Scopes = append([]string{}, m.Scopes...)
	for _, a := range m.Actors {
		s.Actors = append(s.Actors, session.Actor{Subject: a.Subject, Kind: a.Kind, Mode: a.Mode, At: a.At})
	}
```

- [ ] **Step 8: Add the postgres migration**

In `store/postgres/migrations.go`, register a new migration in the same `init()` block that holds `add_session_roles`:

```go
		&migrate.Migration{
			Name:    "add_session_scopes_and_actors",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS actors JSONB NOT NULL DEFAULT '[]'::jsonb;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    DROP COLUMN IF EXISTS scopes,
    DROP COLUMN IF EXISTS actors;
`)
				return err
			},
		},
```

- [ ] **Step 9: Add the sqlite migration**

In `store/sqlite/migrations.go`, in the same `init()` block that holds `add_session_roles`. Sqlite takes one `ADD COLUMN` per statement:

```go
		&migrate.Migration{
			Name:    "add_session_scopes_and_actors",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				if _, err := exec.Exec(ctx,
					`ALTER TABLE authsome_sessions ADD COLUMN scopes TEXT NOT NULL DEFAULT '';`); err != nil {
					return err
				}
				_, err := exec.Exec(ctx,
					`ALTER TABLE authsome_sessions ADD COLUMN actors TEXT NOT NULL DEFAULT '';`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				// sqlite cannot drop a column without rebuilding the table, and
				// leaving two unused columns is cheaper than a rebuild here.
				return nil
			},
		},
```

- [ ] **Step 10: Write the memory-store round-trip test**

Create `store/memory/session_actors_test.go`:

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

func TestSessionScopesAndActorsRoundTrip(t *testing.T) {
	st := memory.New()
	ctx := context.Background()

	sess := &session.Session{
		ID:        id.NewSessionID(),
		AppID:     id.NewAppID(),
		UserID:    id.NewUserID(),
		Token:     "tok-round-trip",
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"invoices:read", "invoices:write"},
		Actors: []session.Actor{
			{Subject: "aoac_client", Kind: session.KindOAuthClient, Mode: session.ModeDelegation},
		},
	}
	require.NoError(t, st.CreateSession(ctx, sess))

	got, err := st.GetSessionByToken(ctx, "tok-round-trip")
	require.NoError(t, err)
	assert.Equal(t, []string{"invoices:read", "invoices:write"}, got.Scopes)
	require.Len(t, got.Actors, 1)
	assert.Equal(t, session.KindOAuthClient, got.Actors[0].Kind)
	assert.Equal(t, session.ModeDelegation, got.Actors[0].Mode)
}
```

- [ ] **Step 11: Run the full suite**

Run: `go test ./session/... ./store/... -count=1`
Expected: PASS. The memory store keeps whole structs, so no memory-store code change is needed for this test to pass.

- [ ] **Step 12: Check and commit**

```bash
make check
git add session/session.go session/session_test.go store/postgres/models.go store/postgres/migrations.go store/sqlite/models.go store/sqlite/migrations.go store/mongo/models.go store/memory/session_actors_test.go
git commit -m "feat(session): carry OAuth scopes and a delegation actor chain"
```

---

### Task 2: Stamp scopes at issuance

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:881` (`issueTokens`), `:934` (`issueClientToken`)
- Test: `plugins/oauth2provider/issue_scopes_test.go` (create)

**Interfaces:**
- Consumes: `session.Session.Scopes` from Task 1.
- Produces: every session minted by the OAuth2 plugin carries its granted scopes, which is the ceiling the exchange handler reads in Task 10.

Without this, an exchanged token has no recorded scopes and a second-hop exchange has no ceiling to narrow against.

- [ ] **Step 1: Write the failing test**

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

// decodeTokenResponse reads the token endpoint's JSON body.
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

	rec := postToken(t, mux, map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  registeredURI,
		"client_id":     confidentialID,
		"client_secret": confidentialSecret,
	})
	body := decodeTokenResponse(t, rec)

	sess, err := core.GetSessionByToken(context.Background(), body["access_token"].(string))
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", "profile"}, sess.Scopes)
}
```

`newFixture`, `authorize`, `baseAuthorizeQuery`, `codeFrom` and `postToken` all already exist in `authcode_test.go` in the same package. Do not redefine them. `decodeTokenResponse` is new; put it in this file and reuse it from Task 10.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestAuthCodeGrantStampsScopes -v`
Expected: FAIL, `sess.Scopes` is nil against the expected two entries.

- [ ] **Step 3: Stamp scopes in issueTokens**

In `plugins/oauth2provider/plugin.go`, inside `issueTokens`, immediately after the `sess, err := account.NewSession(...)` error check:

```go
	// Stamp the granted scopes on the session. Without this the scopes only
	// exist in the JWT claims and the response body, so an opaque token loses
	// them and token exchange has no ceiling to enforce a subset against.
	sess.Scopes = scopes
```

- [ ] **Step 4: Stamp scopes in issueClientToken**

Inside `issueClientToken`, immediately after its `account.NewSession(...)` error check:

```go
	// Client credentials tokens carry the client's registered scopes.
	sess.Scopes = client.Scopes
```

- [ ] **Step 5: Run the tests**

Run: `go test ./plugins/oauth2provider/ -count=1`
Expected: PASS.

- [ ] **Step 6: Check and commit**

```bash
make check
git add plugins/oauth2provider/plugin.go plugins/oauth2provider/issue_scopes_test.go
git commit -m "feat(oauth2): stamp granted scopes on issued sessions"
```

---

### Task 3: Backfill impersonated_by into the actor chain

**Files:**
- Modify: `store/postgres/migrations.go`
- Modify: `store/sqlite/migrations.go`
- Modify: `store/postgres/models.go` (`toSession` fallback)
- Modify: `store/sqlite/models.go` (`toSession` fallback)
- Modify: `store/mongo/models.go` (`toSession` fallback)
- Test: `store/postgres/backfill_actors_test.go` (create)

**Interfaces:**
- Consumes: `session.Actor` from Task 1.
- Produces: a read fallback so a row written by an older binary still resolves to a one-element impersonation chain. Task 4 depends on this being in place before `ImpersonatedBy` leaves the domain type.

- [ ] **Step 1: Write the failing test**

Create `store/postgres/backfill_actors_test.go`:

```go
package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/session"
)

// A row written before the actors column existed carries impersonated_by and
// an empty actors array. It must still read back as an impersonation chain.
func TestToSessionFallsBackToImpersonatedBy(t *testing.T) {
	m := &SessionModel{
		ID:             "ases_00000000000000000000000000",
		AppID:          "aapp_00000000000000000000000000",
		EnvID:          "aenv_00000000000000000000000000",
		UserID:         "usr_00000000000000000000000000",
		ImpersonatedBy: "usr_11111111111111111111111111",
		Token:          "t",
		CreatedAt:      time.Now(),
	}

	s, err := toSession(m)
	require.NoError(t, err)
	require.Len(t, s.Actors, 1)
	assert.Equal(t, "usr_11111111111111111111111111", s.Actors[0].Subject)
	assert.Equal(t, session.KindUser, s.Actors[0].Kind)
	assert.Equal(t, session.ModeImpersonation, s.Actors[0].Mode)
}

// When actors is populated it wins; the legacy column is not re-appended.
func TestToSessionPrefersActorsOverImpersonatedBy(t *testing.T) {
	m := &SessionModel{
		ID:             "ases_00000000000000000000000000",
		AppID:          "aapp_00000000000000000000000000",
		EnvID:          "aenv_00000000000000000000000000",
		UserID:         "usr_00000000000000000000000000",
		ImpersonatedBy: "usr_11111111111111111111111111",
		Actors:         []byte(`[{"sub":"aoac_x","kind":"oauth_client","mode":"delegation"}]`),
		Token:          "t",
		CreatedAt:      time.Now(),
	}

	s, err := toSession(m)
	require.NoError(t, err)
	require.Len(t, s.Actors, 1)
	assert.Equal(t, "aoac_x", s.Actors[0].Subject)
}
```

The id strings above must be valid for `id.ParseSessionID` and friends. If the parser rejects the placeholder shape, generate real ones with `id.NewSessionID().String()` inside the test and use those instead.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./store/postgres/ -run TestToSessionFallsBack -v`
Expected: FAIL, `s.Actors` is empty.

- [ ] **Step 3: Add the fallback to the postgres model**

In `store/postgres/models.go`, replace the existing `if m.ImpersonatedBy != "" { ... }` block in `toSession` with:

```go
	if m.ImpersonatedBy != "" {
		impID, err := id.ParseUserID(m.ImpersonatedBy)
		if err != nil {
			return nil, err
		}
		// Legacy read path. Rows written before the actors column existed, or
		// by an older binary during a rolling deploy, carry the impersonator
		// here and nothing in actors. Actors wins when both are present.
		if len(s.Actors) == 0 {
			s.Actors = []session.Actor{{
				Subject: impID.String(),
				Kind:    session.KindUser,
				Mode:    session.ModeImpersonation,
				At:      m.CreatedAt,
			}}
		}
	}
```

This block must run **after** the actors decode added in Task 1, so move it below that decode if it currently sits above.

- [ ] **Step 4: Add the same fallback to sqlite and mongo**

Apply the identical change to `store/sqlite/models.go`. For `store/mongo/models.go` the guard reads the decoded slice the same way:

```go
	if m.ImpersonatedBy != "" {
		impID, err := id.ParseUserID(m.ImpersonatedBy)
		if err != nil {
			return nil, err
		}
		if len(s.Actors) == 0 {
			s.Actors = []session.Actor{{
				Subject: impID.String(),
				Kind:    session.KindUser,
				Mode:    session.ModeImpersonation,
				At:      m.CreatedAt,
			}}
		}
	}
```

- [ ] **Step 5: Add the postgres backfill migration**

In `store/postgres/migrations.go`:

```go
		&migrate.Migration{
			Name:    "backfill_session_actors_from_impersonated_by",
			Version: "20260824000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
UPDATE authsome_sessions
   SET actors = jsonb_build_array(jsonb_build_object(
           'sub',  impersonated_by,
           'kind', 'user',
           'mode', 'impersonation',
           'at',   created_at))
 WHERE impersonated_by <> '' AND actors = '[]'::jsonb;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
UPDATE authsome_sessions SET actors = '[]'::jsonb
 WHERE impersonated_by <> '';
`)
				return err
			},
		},
```

- [ ] **Step 6: Add the sqlite backfill migration**

Sqlite has no `jsonb_build_object`, so build the JSON with string concatenation:

```go
		&migrate.Migration{
			Name:    "backfill_session_actors_from_impersonated_by",
			Version: "20260824000002",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
UPDATE authsome_sessions
   SET actors = '[{"sub":"' || impersonated_by ||
                '","kind":"user","mode":"impersonation"}]'
 WHERE impersonated_by <> '' AND (actors = '' OR actors = '[]');
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx,
					`UPDATE authsome_sessions SET actors = '[]' WHERE impersonated_by <> '';`)
				return err
			},
		},
```

- [ ] **Step 7: Run the tests**

Run: `go test ./store/... -count=1`
Expected: PASS.

- [ ] **Step 8: Check and commit**

```bash
make check
git add store/postgres store/sqlite store/mongo
git commit -m "feat(store): backfill impersonated_by into the session actor chain"
```

---

### Task 4: Remove ImpersonatedBy from the domain type

**Files:**
- Modify: `session/session.go:28`
- Modify: `service.go:2701`, `:2738`, `:2746`
- Modify: `middleware/auth.go:222-224`, `:630-632`
- Modify: `authprovider/session.go:188-190`
- Modify: `extension/contract/handlers_sessions.go:175-177`
- Modify: `store/postgres/models.go`, `store/sqlite/models.go`, `store/mongo/models.go` (`fromSession` write-compat)
- Test: `impersonate_actors_test.go` (create, package `authsome`)

**Interfaces:**
- Consumes: `session.Actor`, `session.ModeImpersonation` from Task 1; the read fallback from Task 3.
- Produces: `session.Session.Impersonator() (id.UserID, bool)`, returning the outermost impersonation actor. All former `ImpersonatedBy` readers call this.

This is the invasive task. Everything it touches is existing behaviour, and the test proves the behaviour is unchanged.

- [ ] **Step 1: Write the failing test**

Create `impersonate_actors_test.go` at the repo root (package `authsome`, matching `service.go`):

```go
package authsome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

func TestImpersonatorReturnsOutermostImpersonationActor(t *testing.T) {
	admin := id.NewUserID()
	s := &session.Session{
		Actors: []session.Actor{
			{Subject: admin.String(), Kind: session.KindUser, Mode: session.ModeImpersonation, At: time.Now()},
		},
	}

	got, ok := s.Impersonator()
	require.True(t, ok)
	assert.Equal(t, admin, got)
}

func TestImpersonatorIgnoresDelegationChains(t *testing.T) {
	s := &session.Session{
		Actors: []session.Actor{
			{Subject: "aoac_x", Kind: session.KindOAuthClient, Mode: session.ModeDelegation},
		},
	}

	_, ok := s.Impersonator()
	assert.False(t, ok)
}

func TestImpersonatorEmptyChain(t *testing.T) {
	_, ok := (&session.Session{}).Impersonator()
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test . -run TestImpersonator -v`
Expected: FAIL, `s.Impersonator undefined`.

- [ ] **Step 3: Add the Impersonator accessor**

In `session/session.go`, below the `Session` struct:

```go
// Impersonator returns the user id of the outermost impersonating actor and
// whether one exists. A delegation-only chain reports false, because
// delegation and impersonation are different security events and the callers
// of this method are the ones that only care about the second.
func (s *Session) Impersonator() (id.UserID, bool) {
	for _, a := range s.Actors {
		if a.Mode != ModeImpersonation || a.Kind != KindUser {
			continue
		}
		uid, err := id.ParseUserID(a.Subject)
		if err != nil {
			continue
		}
		return uid, true
	}
	return id.Nil, false
}
```

- [ ] **Step 4: Run the accessor test**

Run: `go test . -run TestImpersonator -v`
Expected: PASS.

- [ ] **Step 5: Remove the field and update Impersonate**

Delete the `ImpersonatedBy id.UserID` line from the `Session` struct in `session/session.go`.

In `service.go`, in `Impersonate`, replace `sess.ImpersonatedBy = adminID` with:

```go
	sess.Actors = []session.Actor{{
		Subject: adminID.String(),
		Kind:    session.KindUser,
		Mode:    session.ModeImpersonation,
		At:      time.Now(),
	}}
```

In `StopImpersonation`, replace the guard and the audit call:

```go
	adminID, ok := sess.Impersonator()
	if !ok {
		return fmt.Errorf("authsome: session is not an impersonation session")
	}

	if err := e.store.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("authsome: stop impersonation: delete session: %w", err)
	}

	e.audit(ctx, bridge.SeverityInfo, bridge.OutcomeSuccess, "stop_impersonation", "session", sessionID.String(), adminID.String(), sess.AppID.String(), "admin", map[string]string{
		"target_user_id": sess.UserID.String(),
	})
```

- [ ] **Step 6: Update the middleware, authprovider and contract readers**

`middleware/auth.go` at both sites (lines 222 and 630), replace the two-line guard with:

```go
			if impersonator, ok := sess.Impersonator(); ok {
				goCtx = WithImpersonator(goCtx, impersonator)
			}
```

Match the surrounding indentation at each site; the second one sits inside `setSessionContext` and is one tab shallower.

`authprovider/session.go:188`:

```go
	if impersonator, ok := data.Session.Impersonator(); ok {
		goCtx = authmw.WithImpersonator(goCtx, impersonator)
	}
```

`extension/contract/handlers_sessions.go:175`:

```go
		if impersonator, ok := s.Impersonator(); ok {
			d.ImpersonatedBy = impersonator.String()
		}
```

The `ImpersonatedBy` field on the contract DTO stays. It is the wire format for the sessions API and renaming it would break clients.

- [ ] **Step 7: Keep writing the legacy column**

In all three of `store/postgres/models.go`, `store/sqlite/models.go` and `store/mongo/models.go`, replace the `if s.ImpersonatedBy.Prefix() != "" { ... }` block in `fromSession` with:

```go
	// Still written even though the domain type no longer has the field. An
	// older binary in a rolling deploy reads this column and nothing else, so
	// dropping the write here would blind it to impersonation. The column
	// goes away in a later release, after every binary reads actors.
	if impersonator, ok := s.Impersonator(); ok {
		m.ImpersonatedBy = impersonator.String()
	}
```

- [ ] **Step 8: Find every remaining reference**

Run: `grep -rn "ImpersonatedBy" --include="*.go" . | grep -v "_test.go"`
Expected: only `store/*/models.go` (the `SessionModel` field and the write-compat block) and `extension/contract/handlers_sessions.go` (the DTO field). Any other hit is a site you missed; fix it before moving on.

- [ ] **Step 9: Run the full suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 10: Check and commit**

```bash
make check
git add -A
git commit -m "refactor(session): fold ImpersonatedBy into the actor chain"
```

---

### Task 5: TokenExchangeTTL through the config layers

**Files:**
- Modify: `account/service.go:165`
- Modify: `appsessionconfig/appsessionconfig.go` (the `Config` struct and `ApplyTo`)
- Modify: `environment/settings.go:196`
- Modify: `api/requests.go:599`
- Test: `account/session_config_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `account.SessionConfig.TokenExchangeTTL time.Duration`, resolved per app by the existing `Engine.sessionConfigForApp` at `service.go:783`.

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

In `account/service.go`, in the `SessionConfig` struct:

```go
type SessionConfig struct {
	TokenTTL           time.Duration
	RefreshTokenTTL    time.Duration
	MaxActiveSessions  int
	RotateRefreshToken bool

	// TokenExchangeTTL caps tokens minted by the RFC 8693 token exchange
	// grant. Short by design: an exchanged token is meant to be re-minted,
	// not held. Zero means the caller's own default applies.
	TokenExchangeTTL time.Duration
}
```

- [ ] **Step 4: Add the per-app override**

In `appsessionconfig/appsessionconfig.go`, add to the `Config` struct beside the other token overrides:

```go
	TokenExchangeTTLSeconds *int `json:"token_exchange_ttl_seconds,omitempty"`
```

and to `ApplyTo`, beside the other clauses:

```go
	if c.TokenExchangeTTLSeconds != nil {
		base.TokenExchangeTTL = time.Duration(*c.TokenExchangeTTLSeconds) * time.Second
	}
```

- [ ] **Step 5: Add the per-environment override**

In `environment/settings.go`, add the field to the `Settings` struct beside `TokenTTLSeconds` and `RefreshTokenTTLSeconds`:

```go
	TokenExchangeTTLSeconds *int `json:"token_exchange_ttl_seconds,omitempty"`
```

and to `ApplySessionOverrides`:

```go
	if s.TokenExchangeTTLSeconds != nil {
		cfg.TokenExchangeTTL = time.Duration(*s.TokenExchangeTTLSeconds) * time.Second
	}
```

- [ ] **Step 6: Expose it on the admin request struct**

In `api/requests.go`, in the struct that already carries `RefreshTokenTTLSeconds` at line 599:

```go
	TokenExchangeTTLSeconds *int `json:"token_exchange_ttl_seconds,omitempty" description:"Token exchange TTL in seconds (nil = inherit)"`
```

Then find the handler that maps that struct onto `appsessionconfig.Config` and add the passthrough there. Locate it with:

`grep -rn "RefreshTokenTTLSeconds" --include="*.go" api/`

- [ ] **Step 7: Set the default**

In the engine's base session config (find it with `grep -n "func (e \*Engine) sessionConfig()" service.go`), set the default when unset:

```go
	if cfg.TokenExchangeTTL == 0 {
		cfg.TokenExchangeTTL = 5 * time.Minute
	}
```

- [ ] **Step 8: Run the tests**

Run: `go test ./account/ ./environment/ ./api/ -count=1`
Expected: PASS.

- [ ] **Step 9: Check and commit**

```bash
make check
git add account appsessionconfig environment api service.go
git commit -m "feat(config): add per-app TokenExchangeTTL"
```

---

### Task 6: The act claim in tokenformat

**Files:**
- Modify: `tokenformat/format.go:18`
- Modify: `tokenformat/jwt.go:69`, `:78`, `:143`
- Test: `tokenformat/act_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `tokenformat.ActClaim` with fields `Subject string` (json `sub`) and `Act *ActClaim` (json `act`, nested); `tokenformat.TokenClaims.Act *ActClaim`.

RFC 8693 nests `act` recursively: `{"act": {"sub": "A", "act": {"sub": "B"}}}`. The type is self-referential to match.

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
		Subject: "aoac_outer",
		Act:     &tokenformat.ActClaim{Subject: "usr_inner"},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"sub":"aoac_outer","act":{"sub":"usr_inner"}}`, string(raw))
}

func TestJWTRoundTripsActClaim(t *testing.T) {
	f, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		Secret: []byte("test-secret-at-least-32-bytes-long!!"),
	})
	require.NoError(t, err)

	now := time.Now()
	tok, err := f.GenerateAccessToken(tokenformat.TokenClaims{
		UserID:    "usr_1",
		AppID:     "aapp_1",
		SessionID: "ases_1",
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Minute),
		Act:       &tokenformat.ActClaim{Subject: "aoac_client"},
	})
	require.NoError(t, err)

	out, err := f.ValidateAccessToken(tok)
	require.NoError(t, err)
	require.NotNil(t, out.Act)
	assert.Equal(t, "aoac_client", out.Act.Subject)
}

func TestJWTOmitsActWhenNil(t *testing.T) {
	f, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		Secret: []byte("test-secret-at-least-32-bytes-long!!"),
	})
	require.NoError(t, err)

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

Read `tokenformat/jwt.go` first and match the real constructor name and config-struct shape. If the constructor is not `NewJWT(JWTConfig{Secret: ...})`, use whatever the package actually exports and keep the rest of the test as written.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./tokenformat/ -run TestAct -v`
Expected: FAIL, `undefined: tokenformat.ActClaim`.

- [ ] **Step 3: Add ActClaim and the TokenClaims field**

In `tokenformat/format.go`, above `TokenClaims`:

```go
// ActClaim is the RFC 8693 `act` claim: the party acting on behalf of the
// subject. It nests, so a chain of delegations is a chain of ActClaims with
// the immediate actor outermost.
type ActClaim struct {
	Subject string    `json:"sub"`
	Act     *ActClaim `json:"act,omitempty"`
}
```

and to `TokenClaims`:

```go
	// Act is present for a delegated token and nil otherwise. Impersonation
	// deliberately emits no act claim (RFC 8693 section 1.1), which is why the
	// full record lives on the session row rather than in the token.
	Act *ActClaim `json:"act,omitempty"`
```

- [ ] **Step 4: Wire it through the JWT format**

In `tokenformat/jwt.go`, add to `customClaims`:

```go
	Act *ActClaim `json:"act,omitempty"`
```

In `GenerateAccessToken`, set it on the `jwtClaims` literal:

```go
		Act: claims.Act,
```

In `ValidateAccessToken`, add it to the returned `&TokenClaims{...}`:

```go
		Act: claims.Act,
```

- [ ] **Step 5: Run the tests**

Run: `go test ./tokenformat/ -count=1`
Expected: PASS.

- [ ] **Step 6: Check and commit**

```bash
make check
git add tokenformat
git commit -m "feat(tokenformat): add the RFC 8693 act claim"
```

---

### Task 7: The exchange policy model and memory store

**Files:**
- Modify: `id/id.go:52` (prefix), `:161` (type alias), `:347` (constructor), `:467` (parser)
- Modify: `plugins/oauth2provider/models.go`
- Modify: `plugins/oauth2provider/store.go`
- Modify: `plugins/oauth2provider/store_memory.go`
- Test: `plugins/oauth2provider/exchange_policy_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `id.ExchangePolicyID`, `id.NewExchangePolicyID() ID`, `id.ParseExchangePolicyID(string) (ID, error)`, prefix `"aoxp"`.
  - `oauth2provider.ExchangePolicy` struct.
  - `Store` methods `CreateExchangePolicy(ctx, *ExchangePolicy) error`, `ListExchangePolicies(ctx, id.AppID) ([]*ExchangePolicy, error)`, `DeleteExchangePolicy(ctx, id.ExchangePolicyID) error`, `MatchExchangePolicy(ctx, appID id.AppID, clientID, subjectKind, subjectID string) (*ExchangePolicy, error)`.
  - `ErrExchangePolicyNotFound`.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/exchange_policy_test.go`:

```go
package oauth2provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func TestMatchExchangePolicyPrefersExactSubject(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	wildcard := &oauth2provider.ExchangePolicy{
		ID: id.NewExchangePolicyID(), AppID: appID, ClientID: "svc",
		SubjectKind: "any", SubjectMatch: "*",
		Modes: []string{"delegation"}, MaxScopes: []string{"a"},
		MaxChainDepth: 1, Enabled: true,
	}
	exact := &oauth2provider.ExchangePolicy{
		ID: id.NewExchangePolicyID(), AppID: appID, ClientID: "svc",
		SubjectKind: "user", SubjectMatch: "usr_42",
		Modes: []string{"delegation", "impersonation"}, MaxScopes: []string{"a", "b"},
		MaxChainDepth: 3, Enabled: true,
	}
	require.NoError(t, st.CreateExchangePolicy(ctx, wildcard))
	require.NoError(t, st.CreateExchangePolicy(ctx, exact))

	got, err := st.MatchExchangePolicy(ctx, appID, "svc", "user", "usr_42")
	require.NoError(t, err)
	assert.Equal(t, exact.ID, got.ID)
}

func TestMatchExchangePolicyFallsBackToWildcard(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	require.NoError(t, st.CreateExchangePolicy(ctx, &oauth2provider.ExchangePolicy{
		ID: id.NewExchangePolicyID(), AppID: appID, ClientID: "svc",
		SubjectKind: "any", SubjectMatch: "*",
		Modes: []string{"delegation"}, MaxChainDepth: 1, Enabled: true,
	}))

	got, err := st.MatchExchangePolicy(ctx, appID, "svc", "user", "usr_99")
	require.NoError(t, err)
	assert.Equal(t, "any", got.SubjectKind)
}

func TestMatchExchangePolicySkipsDisabled(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	require.NoError(t, st.CreateExchangePolicy(ctx, &oauth2provider.ExchangePolicy{
		ID: id.NewExchangePolicyID(), AppID: appID, ClientID: "svc",
		SubjectKind: "any", SubjectMatch: "*", Enabled: false,
	}))

	_, err := st.MatchExchangePolicy(ctx, appID, "svc", "user", "usr_1")
	assert.ErrorIs(t, err, oauth2provider.ErrExchangePolicyNotFound)
}

func TestMatchExchangePolicyIsScopedToAppAndClient(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	require.NoError(t, st.CreateExchangePolicy(ctx, &oauth2provider.ExchangePolicy{
		ID: id.NewExchangePolicyID(), AppID: appID, ClientID: "svc",
		SubjectKind: "any", SubjectMatch: "*", Enabled: true,
	}))

	_, err := st.MatchExchangePolicy(ctx, id.NewAppID(), "svc", "user", "usr_1")
	assert.ErrorIs(t, err, oauth2provider.ErrExchangePolicyNotFound)

	_, err = st.MatchExchangePolicy(ctx, appID, "other-client", "user", "usr_1")
	assert.ErrorIs(t, err, oauth2provider.ErrExchangePolicyNotFound)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestMatchExchangePolicy -v`
Expected: FAIL, `undefined: oauth2provider.ExchangePolicy`.

- [ ] **Step 3: Add the id type**

In `id/id.go`, add the prefix beside `PrefixOAuth2Client`:

```go
	PrefixExchangePolicy  Prefix = "aoxp"
```

the alias beside `OAuth2ClientID`:

```go
// ExchangePolicyID is a type-safe identifier for token exchange policies (prefix: "aoxp").
type ExchangePolicyID = ID
```

the constructor beside `NewOAuth2ClientID`:

```go
// NewExchangePolicyID generates a new unique token exchange policy ID.
func NewExchangePolicyID() ID { return New(PrefixExchangePolicy) }
```

and the parser beside `ParseOAuth2ClientID`:

```go
// ParseExchangePolicyID parses a string and validates the "aoxp" prefix.
func ParseExchangePolicyID(s string) (ID, error) { return ParseWithPrefix(s, PrefixExchangePolicy) }
```

- [ ] **Step 4: Add the model**

In `plugins/oauth2provider/models.go`:

```go
// Subject kinds an ExchangePolicy may match. SubjectKindAny matches every kind.
const (
	SubjectKindAny = "any"
)

// ExchangePolicy declares that one client may exchange tokens for one class of
// subject. There is no implicit permission: an exchange with no matching
// enabled policy is refused.
type ExchangePolicy struct {
	ID    id.ExchangePolicyID `json:"id"`
	AppID id.AppID            `json:"app_id"`

	// ClientID is the authenticated requesting client. When an actor_token is
	// presented its principal heads the chain, but policy still gates on the
	// client, because the client is the party that authenticated.
	ClientID string `json:"client_id"`

	// SubjectKind is a session.Kind* value or SubjectKindAny.
	SubjectKind string `json:"subject_kind"`
	// SubjectMatch is a principal id, or "*" for any principal of that kind.
	SubjectMatch string `json:"subject_match"`

	// Modes lists the permitted act modes: session.ModeDelegation and/or
	// session.ModeImpersonation.
	Modes []string `json:"modes"`

	// MaxScopes bounds what this policy may ever grant. It is also the ceiling
	// used when the subject session carries no scopes of its own.
	MaxScopes []string `json:"max_scopes"`

	// MaxTTLSeconds of 0 means this policy does not constrain the TTL, so it
	// drops out of the minimum rather than clamping it to zero.
	MaxTTLSeconds int `json:"max_ttl_seconds"`

	// MaxChainDepth bounds len(subject.Actors)+1. The default of 1 permits a
	// single hop.
	MaxChainDepth int `json:"max_chain_depth"`

	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Specificity ranks a policy for match resolution. Higher wins.
func (p *ExchangePolicy) Specificity() int {
	n := 0
	if p.SubjectMatch != "*" {
		n += 2
	}
	if p.SubjectKind != SubjectKindAny {
		n++
	}
	return n
}

// AllowsMode reports whether this policy permits the given act mode.
func (p *ExchangePolicy) AllowsMode(mode string) bool {
	for _, m := range p.Modes {
		if m == mode {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Extend the Store interface**

In `plugins/oauth2provider/store.go`, add the error:

```go
	ErrExchangePolicyNotFound = errors.New("oauth2: exchange policy not found")
```

and the methods to the `Store` interface:

```go
	// Token exchange policies (RFC 8693)
	CreateExchangePolicy(ctx context.Context, p *ExchangePolicy) error
	ListExchangePolicies(ctx context.Context, appID id.AppID) ([]*ExchangePolicy, error)
	DeleteExchangePolicy(ctx context.Context, id id.ExchangePolicyID) error
	// MatchExchangePolicy returns the most specific enabled policy for this
	// (app, client, subject) triple, or ErrExchangePolicyNotFound. An exact
	// subject_match outranks "*", and a concrete subject_kind outranks "any".
	MatchExchangePolicy(ctx context.Context, appID id.AppID, clientID, subjectKind, subjectID string) (*ExchangePolicy, error)
```

- [ ] **Step 6: Implement them in the memory store**

In `plugins/oauth2provider/store_memory.go`, add a map to the struct (beside the existing client and code maps) and the four methods:

```go
	exchangePolicies map[string]*ExchangePolicy
```

Initialise it in `NewMemoryStore` alongside the other maps, then:

```go
func (s *MemoryStore) CreateExchangePolicy(_ context.Context, p *ExchangePolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exchangePolicies[p.ID.String()] = p
	return nil
}

func (s *MemoryStore) ListExchangePolicies(_ context.Context, appID id.AppID) ([]*ExchangePolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ExchangePolicy, 0, len(s.exchangePolicies))
	for _, p := range s.exchangePolicies {
		if p.AppID == appID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *MemoryStore) DeleteExchangePolicy(_ context.Context, policyID id.ExchangePolicyID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.exchangePolicies, policyID.String())
	return nil
}

func (s *MemoryStore) MatchExchangePolicy(_ context.Context, appID id.AppID, clientID, subjectKind, subjectID string) (*ExchangePolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *ExchangePolicy
	for _, p := range s.exchangePolicies {
		if !p.Enabled || p.AppID != appID || p.ClientID != clientID {
			continue
		}
		if p.SubjectKind != SubjectKindAny && p.SubjectKind != subjectKind {
			continue
		}
		if p.SubjectMatch != "*" && p.SubjectMatch != subjectID {
			continue
		}
		if best == nil || p.Specificity() > best.Specificity() {
			best = p
		}
	}
	if best == nil {
		return nil, ErrExchangePolicyNotFound
	}
	return best, nil
}
```

Match the existing mutex field name in that file. If it uses `sync.RWMutex` under a different name, use that name.

- [ ] **Step 7: Run the tests**

Run: `go test ./plugins/oauth2provider/ ./id/ -count=1`
Expected: PASS.

- [ ] **Step 8: Check and commit**

```bash
make check
git add id plugins/oauth2provider
git commit -m "feat(oauth2): add the token exchange policy model and memory store"
```

---

### Task 8: Policy persistence for postgres, sqlite and mongo

**Files:**
- Modify: `plugins/oauth2provider/store_models.go`
- Modify: `plugins/oauth2provider/store_postgres.go`
- Modify: `plugins/oauth2provider/store_sqlite.go`
- Modify: `plugins/oauth2provider/store_mongo.go`
- Modify: `plugins/oauth2provider/migrations.go`
- Test: `plugins/oauth2provider/exchange_policy_model_test.go` (create)

**Interfaces:**
- Consumes: `ExchangePolicy` and the four `Store` methods from Task 7.
- Produces: nothing new. This brings the three persistent drivers up to the interface Task 7 widened, so the package compiles again for them.

After Task 7 the three SQL and mongo stores no longer satisfy `Store`. This task fixes that.

- [ ] **Step 1: Confirm the compile break**

Run: `go build ./plugins/oauth2provider/`
Expected: FAIL, `*PostgresStore does not implement Store (missing method CreateExchangePolicy)`.

- [ ] **Step 2: Write the failing model test**

Create `plugins/oauth2provider/exchange_policy_model_test.go`:

```go
package oauth2provider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestExchangePolicyModelRoundTrip(t *testing.T) {
	in := &ExchangePolicy{
		ID: id.NewExchangePolicyID(), AppID: id.NewAppID(), ClientID: "svc",
		SubjectKind: "user", SubjectMatch: "usr_1",
		Modes: []string{"delegation"}, MaxScopes: []string{"a", "b"},
		MaxTTLSeconds: 120, MaxChainDepth: 2, Enabled: true,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}

	out, err := toExchangePolicy(fromExchangePolicy(in))
	require.NoError(t, err)
	assert.Equal(t, in.ID, out.ID)
	assert.Equal(t, in.Modes, out.Modes)
	assert.Equal(t, in.MaxScopes, out.MaxScopes)
	assert.Equal(t, in.MaxChainDepth, out.MaxChainDepth)
	assert.True(t, out.Enabled)
}
```

- [ ] **Step 3: Add the store model**

In `plugins/oauth2provider/store_models.go`, following the shape of the existing `OAuth2ClientModel`:

```go
// ExchangePolicyModel is the persisted form of ExchangePolicy.
type ExchangePolicyModel struct {
	ID            string          `grove:"id,pk"                bson:"_id"`
	AppID         string          `grove:"app_id,notnull"       bson:"app_id"`
	ClientID      string          `grove:"client_id,notnull"    bson:"client_id"`
	SubjectKind   string          `grove:"subject_kind,notnull" bson:"subject_kind"`
	SubjectMatch  string          `grove:"subject_match"        bson:"subject_match"`
	Modes         json.RawMessage `grove:"modes,type:jsonb"     bson:"modes"`
	MaxScopes     json.RawMessage `grove:"max_scopes,type:jsonb" bson:"max_scopes"`
	MaxTTLSeconds int             `grove:"max_ttl_seconds"      bson:"max_ttl_seconds"`
	MaxChainDepth int             `grove:"max_chain_depth"      bson:"max_chain_depth"`
	Enabled       bool            `grove:"enabled"              bson:"enabled"`
	CreatedAt     time.Time       `grove:"created_at,notnull"   bson:"created_at"`
	UpdatedAt     time.Time       `grove:"updated_at,notnull"   bson:"updated_at"`
}

func fromExchangePolicy(p *ExchangePolicy) *ExchangePolicyModel {
	m := &ExchangePolicyModel{
		ID:            p.ID.String(),
		AppID:         p.AppID.String(),
		ClientID:      p.ClientID,
		SubjectKind:   p.SubjectKind,
		SubjectMatch:  p.SubjectMatch,
		MaxTTLSeconds: p.MaxTTLSeconds,
		MaxChainDepth: p.MaxChainDepth,
		Enabled:       p.Enabled,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
	// Never nil: these columns are NOT NULL and json.RawMessage cannot scan a
	// NULL back, the same constraint the session Roles column carries.
	if p.Modes == nil {
		p.Modes = []string{}
	}
	if p.MaxScopes == nil {
		p.MaxScopes = []string{}
	}
	m.Modes, _ = json.Marshal(p.Modes)         //nolint:errcheck // best-effort encode
	m.MaxScopes, _ = json.Marshal(p.MaxScopes) //nolint:errcheck // best-effort encode
	return m
}

func toExchangePolicy(m *ExchangePolicyModel) (*ExchangePolicy, error) {
	policyID, err := id.ParseExchangePolicyID(m.ID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	p := &ExchangePolicy{
		ID:            policyID,
		AppID:         appID,
		ClientID:      m.ClientID,
		SubjectKind:   m.SubjectKind,
		SubjectMatch:  m.SubjectMatch,
		MaxTTLSeconds: m.MaxTTLSeconds,
		MaxChainDepth: m.MaxChainDepth,
		Enabled:       m.Enabled,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if len(m.Modes) > 0 {
		_ = json.Unmarshal(m.Modes, &p.Modes) //nolint:errcheck // best-effort decode
	}
	if len(m.MaxScopes) > 0 {
		_ = json.Unmarshal(m.MaxScopes, &p.MaxScopes) //nolint:errcheck // best-effort decode
	}
	return p, nil
}
```

- [ ] **Step 4: Implement the four methods on each SQL store**

In `plugins/oauth2provider/store_postgres.go` and `store_sqlite.go`, follow the exact query-builder idiom the existing `CreateClient` and `ListClients` methods in that same file use. The shape is:

- `CreateExchangePolicy`: insert `fromExchangePolicy(p)`.
- `ListExchangePolicies`: select where `app_id = ?`, map each row through `toExchangePolicy`.
- `DeleteExchangePolicy`: delete where `id = ?`.
- `MatchExchangePolicy`: select where `app_id = ? AND client_id = ? AND enabled = true`, then resolve the winner **in Go** using `Specificity()`, exactly as the memory store does in Task 7 Step 6.

Resolve specificity in Go rather than in SQL. The ranking rule is one function, and duplicating it across three dialects is three chances for the drivers to disagree about which policy applies.

In `store_mongo.go`, the same four methods against the `authsome_oauth2_exchange_policies` collection, following the idiom the existing mongo client methods in that file use.

- [ ] **Step 5: Add the postgres migration**

In `plugins/oauth2provider/migrations.go`, in the `PostgresMigrations` block:

```go
		&migrate.Migration{
			Name:    "create_oauth2_exchange_policies",
			Version: "20260824000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth2_exchange_policies (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL REFERENCES authsome_apps(id),
    client_id       TEXT NOT NULL,
    subject_kind    TEXT NOT NULL,
    subject_match   TEXT NOT NULL DEFAULT '*',
    modes           JSONB NOT NULL DEFAULT '["delegation"]',
    max_scopes      JSONB NOT NULL DEFAULT '[]',
    max_ttl_seconds INT NOT NULL DEFAULT 0,
    max_chain_depth INT NOT NULL DEFAULT 1,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (app_id, client_id, subject_kind, subject_match)
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_exchange_policies_lookup
    ON authsome_oauth2_exchange_policies (app_id, client_id)
    WHERE enabled;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx,
					`DROP TABLE IF EXISTS authsome_oauth2_exchange_policies;`)
				return err
			},
		},
```

- [ ] **Step 6: Add the sqlite migration**

Same table in the `SqliteMigrations` block, with `TEXT` for the JSON columns, `INTEGER` for the ints, `BOOLEAN NOT NULL DEFAULT 1` for `enabled`, `TIMESTAMP` for the times, and no partial index (sqlite supports partial indexes but the plain one is fine here):

```go
		&migrate.Migration{
			Name:    "create_oauth2_exchange_policies",
			Version: "20260824000003",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_oauth2_exchange_policies (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL,
    client_id       TEXT NOT NULL,
    subject_kind    TEXT NOT NULL,
    subject_match   TEXT NOT NULL DEFAULT '*',
    modes           TEXT NOT NULL DEFAULT '["delegation"]',
    max_scopes      TEXT NOT NULL DEFAULT '[]',
    max_ttl_seconds INTEGER NOT NULL DEFAULT 0,
    max_chain_depth INTEGER NOT NULL DEFAULT 1,
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    UNIQUE (app_id, client_id, subject_kind, subject_match)
);

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_exchange_policies_lookup
    ON authsome_oauth2_exchange_policies (app_id, client_id);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx,
					`DROP TABLE IF EXISTS authsome_oauth2_exchange_policies;`)
				return err
			},
		},
```

- [ ] **Step 7: Run build and tests**

Run: `go build ./... && go test ./plugins/oauth2provider/ -count=1`
Expected: build succeeds, tests PASS.

- [ ] **Step 8: Check and commit**

```bash
make check
git add plugins/oauth2provider
git commit -m "feat(oauth2): persist exchange policies across all four drivers"
```

---

### Task 9: Admin CRUD for exchange policies

**Files:**
- Modify: `plugins/oauth2provider/plugin.go` (route registration around `:236`, handlers after `handleDeleteClient` at `:860`, request/response types after `:346`)
- Test: `plugins/oauth2provider/exchange_policy_admin_test.go` (create)

**Interfaces:**
- Consumes: the `Store` methods from Task 7.
- Produces: `POST /v1/admin/oauth/exchange-policies`, `GET /v1/admin/oauth/exchange-policies`, `DELETE /v1/admin/oauth/exchange-policies/:policyId`, guarded by `plugin.AdminGuard(p.engine, "manage", "oauth2_exchange_policy")`.

The permission is deliberately not `oauth2_client`. Anyone who can author exchange policy can authorise impersonation, which is a larger power than registering a client.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/exchange_policy_admin_test.go`:

```go
package oauth2provider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

func TestCreateExchangePolicyRejectsUnknownMode(t *testing.T) {
	p, _, _ := newFixture(t)
	_, err := p.CreateExchangePolicyForTest(&oauth2provider.CreateExchangePolicyRequest{
		ClientID: "svc", SubjectKind: "user", SubjectMatch: "*",
		Modes: []string{"teleportation"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mode")
}

func TestCreateExchangePolicyDefaultsChainDepthAndMode(t *testing.T) {
	p, _, _ := newFixture(t)
	got, err := p.CreateExchangePolicyForTest(&oauth2provider.CreateExchangePolicyRequest{
		ClientID: "svc", SubjectKind: "user", SubjectMatch: "*",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"delegation"}, got.Modes)
	assert.Equal(t, 1, got.MaxChainDepth)
	assert.True(t, got.Enabled)
}
```

`CreateExchangePolicyForTest` is a thin exported wrapper the implementation adds so the validation logic is testable without standing up the full admin-guarded router. Define it in `plugin.go` next to the handler.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestCreateExchangePolicy -v`
Expected: FAIL, `undefined: oauth2provider.CreateExchangePolicyRequest`.

- [ ] **Step 3: Add the request and response types**

In `plugins/oauth2provider/plugin.go`, in the request/response block:

```go
// CreateExchangePolicyRequest creates a token exchange policy.
type CreateExchangePolicyRequest struct {
	ClientID      string   `json:"client_id"`
	SubjectKind   string   `json:"subject_kind"`
	SubjectMatch  string   `json:"subject_match,omitempty"`
	Modes         []string `json:"modes,omitempty"`
	MaxScopes     []string `json:"max_scopes,omitempty"`
	MaxTTLSeconds int      `json:"max_ttl_seconds,omitempty"`
	MaxChainDepth int      `json:"max_chain_depth,omitempty"`
}

// ExchangePolicyResponse is one policy as returned by the admin API.
type ExchangePolicyResponse struct {
	ID            string   `json:"id"`
	ClientID      string   `json:"client_id"`
	SubjectKind   string   `json:"subject_kind"`
	SubjectMatch  string   `json:"subject_match"`
	Modes         []string `json:"modes"`
	MaxScopes     []string `json:"max_scopes"`
	MaxTTLSeconds int      `json:"max_ttl_seconds"`
	MaxChainDepth int      `json:"max_chain_depth"`
	Enabled       bool     `json:"enabled"`
}

// ListExchangePoliciesRequest is empty; the app comes from the session.
type ListExchangePoliciesRequest struct{}

// ListExchangePoliciesResponse wraps the policy list.
type ListExchangePoliciesResponse struct {
	Policies []ExchangePolicyResponse `json:"policies"`
}

// DeleteExchangePolicyRequest identifies a policy to delete.
type DeleteExchangePolicyRequest struct {
	PolicyID string `path:"policyId"`
}

// DeleteExchangePolicyResponse reports the deletion.
type DeleteExchangePolicyResponse struct {
	Status string `json:"status"`
}
```

- [ ] **Step 4: Add validation and the handlers**

```go
// validateExchangePolicyRequest applies defaults and rejects unknown values.
func validateExchangePolicyRequest(req *CreateExchangePolicyRequest) (*ExchangePolicy, error) {
	if req.ClientID == "" {
		return nil, forge.BadRequest("client_id required")
	}
	switch req.SubjectKind {
	case session.KindUser, session.KindServiceAccount, session.KindOAuthClient, SubjectKindAny:
	default:
		return nil, forge.BadRequest("subject_kind must be user, service_account, oauth_client or any")
	}

	modes := req.Modes
	if len(modes) == 0 {
		// Delegation by default. Impersonation is the more privileged mode and
		// has to be asked for explicitly.
		modes = []string{session.ModeDelegation}
	}
	for _, m := range modes {
		if m != session.ModeDelegation && m != session.ModeImpersonation {
			return nil, forge.BadRequest(fmt.Sprintf("unknown mode %q", m))
		}
	}

	match := req.SubjectMatch
	if match == "" {
		match = "*"
	}
	depth := req.MaxChainDepth
	if depth <= 0 {
		depth = 1
	}
	scopes := req.MaxScopes
	if scopes == nil {
		scopes = []string{}
	}

	now := time.Now()
	return &ExchangePolicy{
		ID:            id.NewExchangePolicyID(),
		ClientID:      req.ClientID,
		SubjectKind:   req.SubjectKind,
		SubjectMatch:  match,
		Modes:         modes,
		MaxScopes:     scopes,
		MaxTTLSeconds: req.MaxTTLSeconds,
		MaxChainDepth: depth,
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// CreateExchangePolicyForTest exposes validation without the admin router.
func (p *Plugin) CreateExchangePolicyForTest(req *CreateExchangePolicyRequest) (*ExchangePolicy, error) {
	return validateExchangePolicyRequest(req)
}

func toExchangePolicyResponse(p *ExchangePolicy) ExchangePolicyResponse {
	return ExchangePolicyResponse{
		ID:            p.ID.String(),
		ClientID:      p.ClientID,
		SubjectKind:   p.SubjectKind,
		SubjectMatch:  p.SubjectMatch,
		Modes:         p.Modes,
		MaxScopes:     p.MaxScopes,
		MaxTTLSeconds: p.MaxTTLSeconds,
		MaxChainDepth: p.MaxChainDepth,
		Enabled:       p.Enabled,
	}
}

func (p *Plugin) handleCreateExchangePolicy(ctx forge.Context, req *CreateExchangePolicyRequest) (*ExchangePolicyResponse, error) {
	policy, err := validateExchangePolicyRequest(req)
	if err != nil {
		return nil, err
	}
	appID, err := p.appIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	policy.AppID = appID

	if err := p.oauth2Store.CreateExchangePolicy(ctx.Context(), policy); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create exchange policy: %w", err))
	}
	resp := toExchangePolicyResponse(policy)
	return &resp, nil
}

func (p *Plugin) handleListExchangePolicies(ctx forge.Context, _ *ListExchangePoliciesRequest) (*ListExchangePoliciesResponse, error) {
	appID, err := p.appIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	policies, err := p.oauth2Store.ListExchangePolicies(ctx.Context(), appID)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: list exchange policies: %w", err))
	}
	out := make([]ExchangePolicyResponse, 0, len(policies))
	for _, pol := range policies {
		out = append(out, toExchangePolicyResponse(pol))
	}
	return &ListExchangePoliciesResponse{Policies: out}, nil
}

func (p *Plugin) handleDeleteExchangePolicy(ctx forge.Context, req *DeleteExchangePolicyRequest) (*DeleteExchangePolicyResponse, error) {
	policyID, err := id.ParseExchangePolicyID(req.PolicyID)
	if err != nil {
		return nil, forge.BadRequest("invalid policy ID")
	}
	if err := p.oauth2Store.DeleteExchangePolicy(ctx.Context(), policyID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: delete exchange policy: %w", err))
	}
	return &DeleteExchangePolicyResponse{Status: "deleted"}, nil
}
```

`appIDFromContext` resolves the app from the authenticated session. `handleCreateClient` at `plugin.go:759` already does this; extract whatever it uses into a small `appIDFromContext(ctx forge.Context) (id.AppID, error)` helper and call it from both, rather than copying the lines.

- [ ] **Step 5: Register the routes**

In `RegisterRoutes`, after the existing client routes, add a second admin group:

```go
	// Exchange policy admin. A separate permission from oauth2_client: whoever
	// can author exchange policy can authorise impersonation.
	xchg := router.Group("/v1/admin/oauth",
		forge.WithGroupTags("OAuth2 Admin"),
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.AdminGuard(p.engine, "manage", "oauth2_exchange_policy")...),
	)

	if err := xchg.POST("/exchange-policies", p.handleCreateExchangePolicy,
		forge.WithSummary("Create token exchange policy"),
		forge.WithOperationID("createOAuth2ExchangePolicy"),
		forge.WithRequestSchema(CreateExchangePolicyRequest{}),
		forge.WithResponseSchema(http.StatusCreated, "Policy created", ExchangePolicyResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := xchg.GET("/exchange-policies", p.handleListExchangePolicies,
		forge.WithSummary("List token exchange policies"),
		forge.WithOperationID("listOAuth2ExchangePolicies"),
		forge.WithResponseSchema(http.StatusOK, "Policies", ListExchangePoliciesResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := xchg.DELETE("/exchange-policies/:policyId", p.handleDeleteExchangePolicy,
		forge.WithSummary("Delete token exchange policy"),
		forge.WithOperationID("deleteOAuth2ExchangePolicy"),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}
```

The existing `RegisterRoutes` ends with `return admin.DELETE(...)`. Insert this block before that final return and change the final return to a guarded `if err := ...; err != nil { return err }` followed by `return nil`, so the new group is reached.

- [ ] **Step 6: Run the tests**

Run: `go test ./plugins/oauth2provider/ -count=1`
Expected: PASS.

- [ ] **Step 7: Check and commit**

```bash
make check
git add plugins/oauth2provider
git commit -m "feat(oauth2): admin CRUD for token exchange policies"
```

---

### Task 10: The token exchange grant

**Files:**
- Create: `plugins/oauth2provider/token_exchange.go`
- Modify: `plugins/oauth2provider/plugin.go:289` (`TokenRequest`), `:346` area (`TokenResponse`), `:580` (`handleToken`), `:746` (`handleDiscovery`)
- Test: `plugins/oauth2provider/token_exchange_test.go` (create)

**Interfaces:**
- Consumes: `ExchangePolicy` and `MatchExchangePolicy` (Task 7), `session.Actor` and `session.Session.Scopes` (Task 1), `account.SessionConfig.TokenExchangeTTL` (Task 5), `tokenformat.ActClaim` (Task 6).
- Produces: `tokenExchangeGrantType` constant; `resolveCeiling(requested, clientScopes, policyScopes, subjectScopes []string) ([]string, error)`; `p.handleTokenExchangeGrant(ctx forge.Context, req *TokenRequest) (*TokenResponse, error)`.

- [ ] **Step 1: Write the failing ceiling test**

Create `plugins/oauth2provider/token_exchange_test.go`:

```go
package oauth2provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCeiling(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		client    []string
		policy    []string
		subject   []string
		want      []string
		wantErr   bool
	}{
		{
			name:      "subset of every bound is granted",
			requested: []string{"a"},
			client:    []string{"a", "b"}, policy: []string{"a", "b"}, subject: []string{"a", "b"},
			want: []string{"a"},
		},
		{
			name:      "scope the subject lacks is refused",
			requested: []string{"b"},
			client:    []string{"a", "b"}, policy: []string{"a", "b"}, subject: []string{"a"},
			wantErr: true,
		},
		{
			name:      "scope the policy excludes is refused",
			requested: []string{"b"},
			client:    []string{"a", "b"}, policy: []string{"a"}, subject: []string{"a", "b"},
			wantErr: true,
		},
		{
			name:      "scope the client is not registered for is refused",
			requested: []string{"b"},
			client:    []string{"a"}, policy: []string{"a", "b"}, subject: []string{"a", "b"},
			wantErr: true,
		},
		{
			name:      "scopeless subject falls back to client and policy bounds",
			requested: []string{"a"},
			client:    []string{"a", "b"}, policy: []string{"a"}, subject: nil,
			want: []string{"a"},
		},
		{
			name:      "scopeless subject is still bounded by policy",
			requested: []string{"b"},
			client:    []string{"a", "b"}, policy: []string{"a"}, subject: nil,
			wantErr: true,
		},
		{
			name:      "empty request is refused",
			requested: nil,
			client:    []string{"a"}, policy: []string{"a"}, subject: []string{"a"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCeiling(tt.requested, tt.client, tt.policy, tt.subject)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestResolveCeiling -v`
Expected: FAIL, `undefined: resolveCeiling`.

- [ ] **Step 3: Create token_exchange.go with the ceiling**

```go
package oauth2provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
)

// tokenExchangeGrantType is the IANA grant type for RFC 8693.
const tokenExchangeGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"

// Token type URNs. Only the two access-token forms are supported; both resolve
// to a session, because in this codebase an OAuth access token is a session row.
const (
	tokenTypeAccessToken  = "urn:ietf:params:oauth:token-type:access_token"
	tokenTypeSession      = "urn:x-authsome:params:oauth:token-type:session"
	tokenTypeRefreshToken = "urn:ietf:params:oauth:token-type:refresh_token"
	tokenTypeIDToken      = "urn:ietf:params:oauth:token-type:id_token"
)

// actModeParam is the namespaced parameter that selects impersonation. RFC 8693
// encodes the distinction by absence, which would make the unmarked default the
// more privileged mode. Section 2.1 permits additional parameters, so we ask.
const actModeParam = "authsome_act_mode"

// denial reasons recorded on failed exchanges. Closed set: these are alerted on.
const (
	denyNoPolicy         = "no_policy"
	denyModeNotAllowed   = "mode_not_allowed"
	denyScopeEscalation  = "scope_escalation"
	denyChainTooDeep     = "chain_too_deep"
	denyCrossApp         = "cross_app"
	denyInvalidSubject   = "invalid_subject"
	denyUnsupportedToken = "unsupported_token_type"
)

// resolveCeiling intersects the requested scopes against every bound and
// returns the granted set.
//
//	ceiling = client ∩ policy ∩ (subject if non-empty else ⊤)
//
// The top element for a scopeless subject is safe only because client and
// policy are both finite and both mandatory, so the intersection stays finite
// and administrator-authored. A session minted by password sign-in carries no
// scopes, and treating that as "unrestricted" would make the subset rule
// decorative, while treating it as "nothing" would make the feature useless on
// the first hop.
func resolveCeiling(requested, clientScopes, policyScopes, subjectScopes []string) ([]string, error) {
	// Required, unlike elsewhere. RFC 8693 makes scope optional and lets the
	// server choose, but the point of an exchange is asking for less than you
	// hold, and an omitted scope asks the server to guess.
	if len(requested) == 0 {
		return nil, fmt.Errorf("scope is required for token exchange")
	}

	inClient := toSet(clientScopes)
	inPolicy := toSet(policyScopes)
	inSubject := toSet(subjectScopes)
	subjectBounds := len(subjectScopes) > 0

	granted := make([]string, 0, len(requested))
	for _, s := range requested {
		if _, ok := inClient[s]; !ok {
			return nil, fmt.Errorf("scope %q is not registered for this client", s)
		}
		if _, ok := inPolicy[s]; !ok {
			return nil, fmt.Errorf("scope %q exceeds the exchange policy", s)
		}
		if subjectBounds {
			if _, ok := inSubject[s]; !ok {
				return nil, fmt.Errorf("scope %q is not held by the subject token", s)
			}
		}
		granted = append(granted, s)
	}
	return granted, nil
}

func toSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}
	return out
}
```

- [ ] **Step 4: Run the ceiling test**

Run: `go test ./plugins/oauth2provider/ -run TestResolveCeiling -v`
Expected: PASS, all seven cases.

- [ ] **Step 5: Commit the ceiling**

```bash
make check
git add plugins/oauth2provider/token_exchange.go plugins/oauth2provider/token_exchange_test.go
git commit -m "feat(oauth2): add the token exchange scope ceiling"
```

- [ ] **Step 6: Add the request and response fields**

In `plugins/oauth2provider/plugin.go`, add to `TokenRequest`:

```go
	// RFC 8693 token exchange.
	SubjectToken       string `json:"subject_token,omitempty" form:"subject_token"`
	SubjectTokenType   string `json:"subject_token_type,omitempty" form:"subject_token_type"`
	ActorToken         string `json:"actor_token,omitempty" form:"actor_token"`
	ActorTokenType     string `json:"actor_token_type,omitempty" form:"actor_token_type"`
	RequestedTokenType string `json:"requested_token_type,omitempty" form:"requested_token_type"`
	Scope              string `json:"scope,omitempty" form:"scope"`
	// Audience and Resource are accepted and audited but not yet enforced.
	// That is the seam RFC 8707 resource indicators land in.
	Audience string `json:"audience,omitempty" form:"audience"`
	Resource string `json:"resource,omitempty" form:"resource"`
	// ActMode selects delegation (default) or impersonation.
	ActMode string `json:"authsome_act_mode,omitempty" form:"authsome_act_mode"`
```

and to `TokenResponse`:

```go
	// IssuedTokenType is required by RFC 8693 section 2.2.1 and omitted by
	// every other grant.
	IssuedTokenType string `json:"issued_token_type,omitempty"`
```

- [ ] **Step 7: Write the failing handler test**

Create `plugins/oauth2provider/token_exchange_grant_test.go` in package `oauth2provider_test`.

Name every test `TestRFC8693_*`, not `TestTokenExchange*`. `authcode_test.go` already owns `TestTokenExchange_RejectsMismatchedRedirectURI` and `TestTokenExchange_RejectsMismatchedClientID`, which are about redeeming an authorization code. A shared prefix makes `go test -run TestTokenExchange` ambiguous between two unrelated features.

```go
package oauth2provider_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/oauth2provider"
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

// recordingEvents captures what the plugin writes. Task 11 asserts on it.
type recordingEvents struct{ events []*securityevent.Event }

func (r *recordingEvents) RecordSecurityEvent(_ context.Context, e *securityevent.Event) error {
	r.events = append(r.events, e)
	return nil
}

func (r *recordingEvents) QuerySecurityEvents(_ context.Context, _ *securityevent.Query) ([]*securityevent.Event, string, error) {
	return r.events, "", nil
}

// exchangeEngine implements only the plugin.Engine methods the exchange path
// touches. Embedding the interface satisfies the type; anything the path does
// not call stays nil and would panic loudly if that ever changed.
type exchangeEngine struct {
	plugin.Engine
	core   store.Store
	events *recordingEvents
	cfg    account.SessionConfig
}

func (e *exchangeEngine) Store() store.Store   { return e.core }
func (e *exchangeEngine) Logger() log.Logger   { return log.NewNoopLogger() }
func (e *exchangeEngine) SecurityEvents() securityevent.Store { return e.events }

func (e *exchangeEngine) ResolveSessionByToken(token string) (*session.Session, error) {
	return e.core.GetSessionByToken(context.Background(), token)
}

func (e *exchangeEngine) SessionConfigForApp(_ context.Context, _ id.AppID, _ ...id.EnvironmentID) account.SessionConfig {
	return e.cfg
}

func (e *exchangeEngine) TokenFormatForApp(_ string) tokenformat.Format {
	return tokenformat.Opaque{}
}

type xchgFixture struct {
	plugin *oauth2provider.Plugin
	oauth  oauth2provider.Store
	core   store.Store
	mux    forge.Router
	appID  id.AppID
	events *recordingEvents
}

// newExchangeFixture registers one confidential client holding scopes a and b
// and registered for the exchange grant. No policy and no subject session are
// created; each test adds exactly what it needs.
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
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	return &xchgFixture{plugin: p, oauth: oauth, core: core, mux: mux, appID: appID, events: events}
}

// seedPolicy creates one enabled wildcard policy for the exchange client.
func (f *xchgFixture) seedPolicy(t *testing.T, maxScopes, modes []string, depth int) *oauth2provider.ExchangePolicy {
	t.Helper()
	pol := &oauth2provider.ExchangePolicy{
		ID:            id.NewExchangePolicyID(),
		AppID:         f.appID,
		ClientID:      xchgClientID,
		SubjectKind:   oauth2provider.SubjectKindAny,
		SubjectMatch:  "*",
		Modes:         modes,
		MaxScopes:     maxScopes,
		MaxChainDepth: depth,
		Enabled:       true,
	}
	require.NoError(t, f.oauth.CreateExchangePolicy(context.Background(), pol))
	return pol
}

// seedSubject inserts a subject session and returns its token.
func (f *xchgFixture) seedSubject(t *testing.T, appID id.AppID, scopes []string, actors []session.Actor, life time.Duration) string {
	t.Helper()
	tok := "subject-" + id.NewSessionID().String()
	require.NoError(t, f.core.CreateSession(context.Background(), &session.Session{
		ID:        id.NewSessionID(),
		AppID:     appID,
		UserID:    id.NewUserID(),
		Token:     tok,
		Scopes:    scopes,
		Actors:    actors,
		ExpiresAt: time.Now().Add(life),
	}))
	return tok
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

func TestRFC8693_NarrowsScope(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a", "b"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, nil, time.Hour)

	body := decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	assert.Equal(t, "a", body["scope"])
	assert.Equal(t, accessTokenType, body["issued_token_type"])
	assert.NotEmpty(t, body["access_token"])
	// No refresh token: re-exchange instead, so the subject stays the only
	// durable credential.
	assert.Empty(t, body["refresh_token"])
}

func TestRFC8693_RefusesScopeEscalation(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a", "b"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "b",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "access_token")
}

func TestRFC8693_RefusesEmptyScope(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_RefusesWithoutPolicy(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "access_token")
}

func TestRFC8693_RefusesCrossApp(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	// Subject belongs to a different app than the client.
	subject := f.seedSubject(t, id.NewAppID(), []string{"a"}, nil, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.NotContains(t, rec.Body.String(), "access_token")
}

func TestRFC8693_RefusesImpersonationWithoutPolicyMode(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
		"authsome_act_mode":  session.ModeImpersonation,
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_RefusesChainTooDeep(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	// Depth 1 permits one hop, so a subject that already carries an actor is
	// at the cap and cannot be exchanged again.
	subject := f.seedSubject(t, f.appID, []string{"a"}, []session.Actor{
		{Subject: "aoac_prior", Kind: session.KindOAuthClient, Mode: session.ModeDelegation},
	}, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_ClampsTTLToSubjectRemainingLife(t *testing.T) {
	// Config allows five minutes; the subject has four left.
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, 4*time.Minute)

	body := decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	expiresIn := int(body["expires_in"].(float64))
	assert.LessOrEqual(t, expiresIn, 240)
	assert.Greater(t, expiresIn, 200)
}

func TestRFC8693_AppendsActorToChain(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	body := decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	issued, err := f.core.GetSessionByToken(context.Background(), body["access_token"].(string))
	require.NoError(t, err)
	require.Len(t, issued.Actors, 1)
	assert.Equal(t, xchgClientID, issued.Actors[0].Subject)
	assert.Equal(t, session.KindOAuthClient, issued.Actors[0].Kind)
	assert.Equal(t, session.ModeDelegation, issued.Actors[0].Mode)
	assert.Equal(t, []string{"a"}, issued.Scopes)
}

func TestRFC8693_RefusesPublicClient(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	require.NoError(t, f.oauth.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:         id.NewOAuth2ClientID(),
		AppID:      f.appID,
		ClientID:   "pub-exchange",
		Name:       "Public",
		Scopes:     []string{"a"},
		GrantTypes: []string{exchangeGrant},
		Public:     true,
	}))
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	rec := postToken(t, f.mux, map[string]string{
		"grant_type":         exchangeGrant,
		"client_id":          "pub-exchange",
		"client_secret":      "anything",
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRFC8693_RefusesUnsupportedSubjectTokenType(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	rec := f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": "urn:ietf:params:oauth:token-type:refresh_token",
		"scope":              "a",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported_token_type")
}
```

Add `"net/http/httptest"` and `"golang.org/x/crypto/bcrypt"` to the import block above. If `plugin.Engine` has since gained a method the exchange path calls, the embedded-interface double will nil-panic with a clear stack; implement that method on `exchangeEngine` rather than widening the test.

- [ ] **Step 8: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run TestTokenExchange -v`
Expected: FAIL for every case.

- [ ] **Step 9: Implement the handler**

Append to `plugins/oauth2provider/token_exchange.go`:

```go
// resolveExchangeSubject resolves a subject or actor token to its session and
// derives the principal kind and id used for policy matching.
func (p *Plugin) resolveExchangeSubject(token, tokenType string) (*session.Session, string, string, error) {
	switch tokenType {
	case tokenTypeAccessToken, tokenTypeSession:
	case tokenTypeRefreshToken, tokenTypeIDToken:
		return nil, "", "", forge.BadRequest("unsupported_token_type")
	default:
		return nil, "", "", forge.BadRequest("unsupported_token_type")
	}
	if p.engine == nil {
		return nil, "", "", forge.InternalError(fmt.Errorf("oauth2: no engine"))
	}

	sess, err := p.engine.ResolveSessionByToken(token)
	if err != nil || sess == nil {
		return nil, "", "", forge.BadRequest("invalid_grant")
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, "", "", forge.BadRequest("invalid_grant")
	}

	kind := sess.PrincipalKind
	if kind == "" {
		// Empty means user, for rows written before the column existed.
		kind = session.KindUser
	}
	principal := sess.UserID.String()
	if kind == session.KindServiceAccount {
		principal = sess.ServiceAccountID.String()
	}
	return sess, kind, principal, nil
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
	if err := bcrypt.CompareHashAndPassword([]byte(client.ClientSecret), []byte(req.ClientSecret)); err != nil {
		return nil, forge.Unauthorized("invalid client_secret")
	}

	// 2. The client must be registered for this grant.
	if !p.clientSupportsGrant(client, tokenExchangeGrantType) {
		return nil, forge.BadRequest("unauthorized_client")
	}

	// 3. Resolve the subject.
	if req.SubjectToken == "" || req.SubjectTokenType == "" {
		return nil, forge.BadRequest("subject_token and subject_token_type required")
	}
	if req.RequestedTokenType != "" && req.RequestedTokenType != tokenTypeAccessToken {
		return nil, forge.BadRequest("unsupported requested_token_type")
	}
	subject, subjectKind, subjectPrincipal, err := p.resolveExchangeSubject(req.SubjectToken, req.SubjectTokenType)
	if err != nil {
		p.recordExchange(ctx, client, nil, nil, "", nil, nil, "", denyInvalidSubject, err)
		return nil, err
	}

	// 4. Cross-app exchange is refused. Without this a client in one app
	// launders a session out of another.
	if subject.AppID != client.AppID {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, nil, "", nil, nil, "", denyCrossApp, e)
		return nil, e
	}

	// 5. Resolve the actor token if one was sent. Its principal is proven by
	// resolution, not asserted in a parameter, which is what lets policy gate
	// on the client alone.
	actorSubject := client.ClientID
	actorKind := session.KindOAuthClient
	if req.ActorToken != "" {
		actorSess, aKind, aPrincipal, aErr := p.resolveExchangeSubject(req.ActorToken, req.ActorTokenType)
		if aErr != nil {
			p.recordExchange(ctx, client, subject, nil, "", nil, nil, "", denyInvalidSubject, aErr)
			return nil, aErr
		}
		if actorSess.AppID != client.AppID {
			e := forge.BadRequest("invalid_grant")
			p.recordExchange(ctx, client, subject, nil, "", nil, nil, "", denyCrossApp, e)
			return nil, e
		}
		actorSubject, actorKind = aPrincipal, aKind
	}

	// 6. Policy. Deny by default.
	policy, err := p.oauth2Store.MatchExchangePolicy(ctx.Context(), client.AppID, client.ClientID, subjectKind, subjectPrincipal)
	if err != nil {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, nil, "", nil, nil, "", denyNoPolicy, e)
		return nil, e
	}

	mode := req.ActMode
	if mode == "" {
		mode = session.ModeDelegation
	}
	if mode != session.ModeDelegation && mode != session.ModeImpersonation {
		return nil, forge.BadRequest("invalid authsome_act_mode")
	}
	if !policy.AllowsMode(mode) {
		e := forge.BadRequest("invalid_grant")
		p.recordExchange(ctx, client, subject, policy, mode, nil, nil, "", denyModeNotAllowed, e)
		return nil, e
	}

	// 7. Scope ceiling.
	requested := strings.Fields(req.Scope)
	granted, err := resolveCeiling(requested, client.Scopes, policy.MaxScopes, subject.Scopes)
	if err != nil {
		e := forge.BadRequest(fmt.Sprintf("invalid_scope: %s", err.Error()))
		p.recordExchange(ctx, client, subject, policy, mode, requested, nil, "", denyScopeEscalation, e)
		return nil, e
	}

	// 8. Chain depth. Every hop appends an actor, so this has to be bounded.
	if len(subject.Actors)+1 > policy.MaxChainDepth {
		e := forge.BadRequest("invalid_grant: delegation chain too deep")
		p.recordExchange(ctx, client, subject, policy, mode, requested, nil, "", denyChainTooDeep, e)
		return nil, e
	}

	// 9. Mint.
	resp, issuedID, err := p.issueExchangedToken(ctx.Context(), client, subject, policy, mode, actorSubject, actorKind, granted)
	if err != nil {
		return nil, err
	}
	p.recordExchange(ctx, client, subject, policy, mode, requested, granted, issuedID, "", nil)
	return resp, nil
}

// issueExchangedToken mints the narrowed session.
func (p *Plugin) issueExchangedToken(
	ctx context.Context,
	client *OAuth2Client,
	subject *session.Session,
	policy *ExchangePolicy,
	mode, actorSubject, actorKind string,
	granted []string,
) (*TokenResponse, string, error) {
	sessCfg := account.SessionConfig{TokenTTL: 5 * time.Minute}
	if p.engine != nil {
		sessCfg = p.engine.SessionConfigForApp(ctx, client.AppID)
	}

	ttl := sessCfg.TokenExchangeTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if policy.MaxTTLSeconds > 0 {
		if capped := time.Duration(policy.MaxTTLSeconds) * time.Second; capped < ttl {
			ttl = capped
		}
	}
	// Never outlive the subject. A token that survives the credential it came
	// from is an escalation in the time dimension, and it is easy to miss
	// because each individual TTL looks short.
	if remaining := time.Until(subject.ExpiresAt); remaining < ttl {
		ttl = remaining
	}
	if ttl <= 0 {
		return nil, "", forge.BadRequest("invalid_grant: subject token has expired")
	}

	// No refresh token. Re-exchange instead, which keeps the subject as the
	// only durable credential.
	cfg := account.SessionConfig{TokenTTL: ttl, RefreshTokenTTL: 0}

	sess, err := account.NewSession(client.AppID, subject.UserID, cfg)
	if err != nil {
		return nil, "", forge.InternalError(fmt.Errorf("oauth2: create exchanged session: %w", err))
	}
	sess.RefreshToken = ""
	sess.RefreshTokenExpiresAt = time.Time{}
	// Inherit the subject's environment rather than the app default, so an
	// exchanged token stays where it came from.
	sess.EnvID = subject.EnvID
	sess.OrgID = subject.OrgID
	sess.PrincipalKind = subject.PrincipalKind
	sess.ServiceAccountID = subject.ServiceAccountID
	sess.Scopes = granted
	// Roles copied verbatim. Scope narrowing is real for scope-gated code and
	// invisible to role-gated code; see the known limitation in the design doc.
	sess.Roles = subject.Roles
	sess.Actors = append([]session.Actor{{
		Subject: actorSubject,
		Kind:    actorKind,
		Mode:    mode,
		At:      time.Now(),
	}}, subject.Actors...)

	if p.engine != nil {
		tokFmt := p.engine.TokenFormatForApp(client.AppID.String())
		if tokFmt.Name() == "jwt" {
			claims := tokenformat.TokenClaims{
				UserID:    subject.UserID.String(),
				AppID:     client.AppID.String(),
				SessionID: sess.ID.String(),
				Scopes:    granted,
				IssuedAt:  sess.CreatedAt,
				ExpiresAt: sess.ExpiresAt,
			}
			// Delegation names both parties. Impersonation emits no act claim
			// at all (RFC 8693 section 1.1); the session row keeps the record.
			if mode == session.ModeDelegation {
				claims.Act = actChainToClaim(sess.Actors)
			}
			jwtToken, genErr := tokFmt.GenerateAccessToken(claims)
			if genErr != nil {
				return nil, "", forge.InternalError(fmt.Errorf("oauth2: generate JWT: %w", genErr))
			}
			sess.Token = jwtToken
		}
	}

	if err := p.store.CreateSession(ctx, sess); err != nil {
		return nil, "", forge.InternalError(fmt.Errorf("oauth2: save exchanged session: %w", err))
	}

	return &TokenResponse{
		AccessToken:     sess.Token,
		IssuedTokenType: tokenTypeAccessToken,
		TokenType:       "Bearer",
		ExpiresIn:       int(time.Until(sess.ExpiresAt).Seconds()),
		Scope:           strings.Join(granted, " "),
	}, sess.ID.String(), nil
}

// actChainToClaim renders an actor chain as a nested RFC 8693 act claim, the
// immediate actor outermost. Impersonation entries are skipped: the RFC has no
// representation for them and the session row is the record.
func actChainToClaim(actors []session.Actor) *tokenformat.ActClaim {
	var head, tail *tokenformat.ActClaim
	for _, a := range actors {
		if a.Mode != session.ModeDelegation {
			continue
		}
		node := &tokenformat.ActClaim{Subject: a.Subject}
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

Add `"golang.org/x/crypto/bcrypt"` to the imports of `token_exchange.go`. `id` is imported for the audit call in Task 11; if the compiler reports it unused at this step, add it in Task 11 instead.

- [ ] **Step 10: Add a temporary no-op recordExchange**

Task 11 fills in the body. Declare the **final** signature now so Task 11 changes no call sites:

```go
// recordExchange writes one security event per exchange attempt.
// Body implemented in Task 11; the signature is final.
func (p *Plugin) recordExchange(
	_ forge.Context,
	_ *OAuth2Client,
	_ *session.Session,
	_ *ExchangePolicy,
	_ string, // mode
	_ []string, // requested scopes
	_ []string, // granted scopes, nil on failure
	_ string, // issued session id, "" on failure
	_ string, // denial reason, "" on success
	_ error,
) {
}
```

The Step 9 code already calls this signature at every site: failures pass `nil, ""` for granted scopes and issued session id, and the success call passes `granted, issuedID` after the mint. Nothing here needs changing later, which is the point of declaring the full signature now rather than widening it in Task 11.

- [ ] **Step 11: Wire dispatch and discovery**

In `handleToken` at `plugin.go:580`, add a case:

```go
	case tokenExchangeGrantType:
		return p.handleTokenExchangeGrant(ctx, req)
```

In `handleDiscovery` at `plugin.go:746`, add the grant to the advertised list:

```go
		GrantTypesSupported: []string{
			"authorization_code",
			"client_credentials",
			"urn:ietf:params:oauth:grant-type:device_code",
			tokenExchangeGrantType,
		},
```

- [ ] **Step 12: Run the tests**

Run: `go test ./plugins/oauth2provider/ -count=1 -v`
Expected: PASS, including every `TestTokenExchange*` case.

- [ ] **Step 13: Check and commit**

```bash
make check
git add plugins/oauth2provider
git commit -m "feat(oauth2): add the RFC 8693 token exchange grant"
```

---

### Task 11: Security events for every exchange

**Files:**
- Modify: `plugin/plugin.go:103` area (the `Engine` interface)
- Modify: `plugins/oauth2provider/token_exchange.go` (replace the no-op `recordExchange`)
- Test: `plugins/oauth2provider/token_exchange_audit_test.go` (create)

**Interfaces:**
- Consumes: `securityevent.Store`, the denial-reason constants from Task 10.
- Produces: `plugin.Engine.SecurityEvents() securityevent.Store`. The concrete engine already has this method at `engine.go:831`, so the interface addition is satisfied without touching the engine.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/token_exchange_audit_test.go`:

`recordingEvents`, `newExchangeFixture`, `seedPolicy`, `seedSubject` and `exchange` all come from `token_exchange_grant_test.go` in Task 10. Do not redefine them.

```go
package oauth2provider_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

func TestRFC8693_SuccessWritesSecurityEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	pol := f.seedPolicy(t, []string{"a", "b"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a", "b"}, nil, time.Hour)

	decodeTokenResponse(t, f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	}))

	require.Len(t, f.events.events, 1)
	ev := f.events.events[0]
	assert.Equal(t, "oauth2.token_exchange", ev.Action)
	assert.Equal(t, "success", ev.Outcome)
	// AppID must be set. The hook-bus bridge does not populate it, which is
	// why the plugin writes to the store directly.
	assert.Equal(t, f.appID, ev.AppID)
	assert.Equal(t, "a", ev.Metadata["granted_scopes"])
	assert.Equal(t, session.ModeDelegation, ev.Metadata["act_mode"])
	assert.Equal(t, "1", ev.Metadata["chain_depth"])
	assert.Equal(t, pol.ID.String(), ev.Metadata["policy_id"])
	assert.NotEmpty(t, ev.Metadata["issued_session_id"])
	assert.NotContains(t, ev.Metadata, "denial_reason")
}

func TestRFC8693_ScopeEscalationWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a", "b"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

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
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, id.NewAppID(), []string{"a"}, nil, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "failure", f.events.events[0].Outcome)
	assert.Equal(t, "cross_app", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_MissingPolicyWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "failure", f.events.events[0].Outcome)
	assert.Equal(t, "no_policy", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_ModeNotAllowedWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, nil, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
		"authsome_act_mode":  session.ModeImpersonation,
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "mode_not_allowed", f.events.events[0].Metadata["denial_reason"])
}

func TestRFC8693_ChainTooDeepWritesFailureEvent(t *testing.T) {
	f := newExchangeFixture(t, 5*time.Minute)
	f.seedPolicy(t, []string{"a"}, []string{session.ModeDelegation}, 1)
	subject := f.seedSubject(t, f.appID, []string{"a"}, []session.Actor{
		{Subject: "aoac_prior", Kind: session.KindOAuthClient, Mode: session.ModeDelegation},
	}, time.Hour)

	f.exchange(t, map[string]string{
		"subject_token":      subject,
		"subject_token_type": accessTokenType,
		"scope":              "a",
	})

	require.Len(t, f.events.events, 1)
	assert.Equal(t, "chain_too_deep", f.events.events[0].Metadata["denial_reason"])
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run WritesEvent -v`
Expected: FAIL, no events recorded.

- [ ] **Step 3: Add SecurityEvents to the Engine interface**

In `plugin/plugin.go`, in the `Engine` interface beside `APIKeyStore()`:

```go
	// SecurityEvents returns the queryable security event store, or nil when
	// the engine was built without one. Plugins write here directly rather
	// than through the hook bus: the bus bridge does not populate AppID, and
	// securityevent.Query filters on it.
	SecurityEvents() securityevent.Store
```

Add the import for `github.com/xraph/authsome/securityevent`.

- [ ] **Step 4: Implement recordExchange**

Replace the no-op in `plugins/oauth2provider/token_exchange.go`:

```go
// recordExchange writes one security event per exchange attempt. Failures
// matter more than successes here: scope_escalation and cross_app are attack
// signatures, and the closed denial-reason vocabulary is what makes them
// alertable instead of greppable.
func (p *Plugin) recordExchange(
	ctx forge.Context,
	client *OAuth2Client,
	subject *session.Session,
	policy *ExchangePolicy,
	mode string,
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

	meta := map[string]string{
		"client_id": client.ClientID,
		"act_mode":  mode,
	}
	if len(requested) > 0 {
		meta["requested_scopes"] = strings.Join(requested, " ")
	}
	if len(granted) > 0 {
		meta["granted_scopes"] = strings.Join(granted, " ")
	}
	if issuedSessionID != "" {
		meta["issued_session_id"] = issuedSessionID
	}
	if policy != nil {
		meta["policy_id"] = policy.ID.String()
	}

	var userID id.UserID
	if subject != nil {
		userID = subject.UserID
		meta["subject_session_id"] = subject.ID.String()
		meta["chain_depth"] = fmt.Sprintf("%d", len(subject.Actors)+1)
		kind := subject.PrincipalKind
		if kind == "" {
			kind = session.KindUser
		}
		meta["subject_kind"] = kind
		meta["subject_principal_id"] = subject.UserID.String()
		if kind == session.KindServiceAccount {
			meta["subject_principal_id"] = subject.ServiceAccountID.String()
		}
	}
	if denialReason != "" {
		meta["denial_reason"] = denialReason
	}

	_ = events.RecordSecurityEvent(ctx.Context(), &securityevent.Event{
		AppID:     client.AppID,
		UserID:    userID,
		Action:    "oauth2.token_exchange",
		Outcome:   outcome,
		Metadata:  meta,
		CreatedAt: time.Now(),
	}) //nolint:errcheck // audit is best-effort; it must not fail the exchange
}
```

Add the `securityevent` and `id` imports to `token_exchange.go`.

- [ ] **Step 5: Verify no call site changed**

The signature was declared in full in Task 10 Step 10, so filling in the body should not touch `handleTokenExchangeGrant` at all.

Run: `git diff --stat plugins/oauth2provider/token_exchange.go`
Expected: the only changes are inside `recordExchange` and the import block. If `handleTokenExchangeGrant` shows up in the diff, a call site drifted from the declared signature; reconcile it against Task 10 Step 10 rather than changing the signature again.

- [ ] **Step 6: Run the tests**

Run: `go test ./plugins/oauth2provider/ ./plugin/ -count=1`
Expected: PASS.

- [ ] **Step 7: Run the whole suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 8: Check and commit**

```bash
make check
git add plugin plugins/oauth2provider
git commit -m "feat(oauth2): write a security event for every token exchange"
```

---

## Self-review notes

Checked against the spec:

- Scopes as a session column: Task 1, stamped at issuance in Task 2.
- Actor chain and the `Mode` field: Task 1.
- `ImpersonatedBy` unification with backfill and a two-release drop: Tasks 3 and 4. The column drop is deliberately absent, as the spec requires.
- Policy table, deny by default, most-specific-wins: Tasks 7, 8, 9.
- Ceiling with the scopeless-subject fallback: Task 10, seven table cases.
- Required `scope`: Task 10, `resolveCeiling` first branch and its test case.
- TTL clamped by policy, config, and the subject's remaining life: Tasks 5 and 10.
- No refresh token on exchanged sessions: Task 10, `issueExchangedToken`.
- `act` claim emitted for delegation and omitted for impersonation: Tasks 6 and 10.
- `issued_token_type` in the response: Task 10.
- Discovery advertisement: Task 10 Step 11.
- Security event per attempt with the closed denial-reason set: Task 11.
- Store parity across four drivers: Tasks 1, 3, 8.

Two things the spec mentions that this plan deliberately does **not** do, both flagged there as separate work: fixing the hook bridge at `engine.go:526` that drops `AppID`, and the eventual `impersonated_by` column drop.
