# OAuth 2.0 Dynamic Client Registration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an MCP client discover this authorization server and register itself, via RFC 7591 registration, RFC 7592 management, RFC 8414 authorization server metadata and RFC 9728 protected resource metadata.

**Architecture:** Four well-known documents move to a new `RootRouteProvider` hook so they are served at the origin root instead of under the extension's mount prefix. `POST /v1/oauth/register` resolves its app from the publishable key (falling back to config), then runs every request through one validation pipeline that clamps grant types, scopes and redirect URIs. RFC 7592 reuses that same pipeline so an update can never widen what registration refused.

**Tech Stack:** Go 1.26, forge router v1.9.10, grove ORM (postgres/sqlite), mongo-driver v2, bcrypt, testify.

**Spec:** `docs/superpowers/specs/2026-08-24-oauth2-dynamic-client-registration-design.md`

## Global Constraints

- Go 1.26.0. Module is `github.com/xraph/authsome`.
- Run tests with `go test ./plugins/oauth2provider/... -v`. Full suite is `make test`. Lint is `make lint`.
- Existing external test package is `oauth2provider_test`. Tests for unexported helpers go in package `oauth2provider` in a separate file.
- Dynamic clients may hold only `authorization_code` and `refresh_token`.
- Default scope allowlist is exactly `openid`, `profile`, `email`, `offline_access`.
- Default registration rate limit is 10 per hour, keyed by client IP.
- `Config.DynamicRegistration` defaults to false. No upgrade may open a public endpoint.
- Registration and RFC 7592 errors use the body `{"error": "...", "error_description": "..."}`, never the forge envelope.
- Never log a raw client secret or registration access token.
- Commit messages contain no `Co-Authored-By` trailer and no AI attribution.

---

### Task 1: RootRouteProvider hook

Plugins only ever receive the grouped router, so `/.well-known/*` currently lands under the mount prefix. This task adds the hook and moves the existing OIDC document onto it. No new endpoints.

**Files:**
- Modify: `plugin/plugin.go` (add interface near `RouteProvider`, around line 346)
- Modify: `plugin/registry.go:116` (entry type), `:160` (field), `:258` (Register), `:540` (emitter)
- Modify: `extension/extension.go:513-527`
- Modify: `plugins/oauth2provider/plugin.go:149-262`
- Test: `plugin/registry_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `plugin.RootRouteProvider` with method `RegisterRootRoutes(router forge.Router) error`; `(*plugin.Registry).RootRouteProviders() []RootRouteProvider`; `(*oauth2provider.Plugin).RegisterRootRoutes(forge.Router) error`.

- [ ] **Step 1: Write the failing registry test**

Append to `plugin/registry_test.go`:

```go
type rootRoutePlugin struct{ called bool }

func (p *rootRoutePlugin) Name() string { return "root-route-test" }
func (p *rootRoutePlugin) RegisterRootRoutes(_ forge.Router) error {
	p.called = true
	return nil
}

func TestRegistry_RootRouteProviders(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	p := &rootRoutePlugin{}
	r.Register(p)

	providers := r.RootRouteProviders()
	require.Len(t, providers, 1)
	require.NoError(t, providers[0].RegisterRootRoutes(nil))
	assert.True(t, p.called)
}

func TestRegistry_RootRouteProvidersExcludesPlainPlugins(t *testing.T) {
	r := plugin.NewRegistry(log.NewNoopLogger())
	r.Register(&rootRoutePlugin{})
	// A plugin that only implements Plugin must not appear.
	assert.Len(t, r.RootRouteProviders(), 1)
}
```

Match the import block already at the top of `registry_test.go`; add `"github.com/xraph/forge"` if it is not there.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./plugin/ -run TestRegistry_RootRoute -v`
Expected: FAIL, `r.RootRouteProviders undefined`.

- [ ] **Step 3: Add the interface**

In `plugin/plugin.go`, directly after the `RouteProvider` interface:

```go
// RootRouteProvider is implemented by plugins that must serve routes at the
// origin root rather than under the extension's mount prefix. Well-known
// discovery documents are the only legitimate use: RFC 8414 and RFC 9728
// define their locations relative to the origin, so a prefixed copy is
// invisible to a client that only knows the host.
type RootRouteProvider interface {
	RegisterRootRoutes(router forge.Router) error
}
```

- [ ] **Step 4: Wire the registry**

In `plugin/registry.go`, next to `routeProviderEntry` (line 116):

```go
type rootRouteProviderEntry struct {
	name string
	hook RootRouteProvider
}
```

Next to the `routeProviders` field (line 160):

```go
	rootRouteProviders     []rootRouteProviderEntry
```

In `Register`, directly after the `RouteProvider` assertion (line 257-259):

```go
	if h, ok := p.(RootRouteProvider); ok {
		r.rootRouteProviders = append(r.rootRouteProviders, rootRouteProviderEntry{name, h})
	}
```

Next to `RouteProviders()` (line 540):

```go
// RootRouteProviders returns all plugins that provide origin-root routes.
func (r *Registry) RootRouteProviders() []RootRouteProvider {
	providers := make([]RootRouteProvider, len(r.rootRouteProviders))
	for i, e := range r.rootRouteProviders {
		providers[i] = e.hook
	}
	return providers
}
```

- [ ] **Step 5: Run the test and confirm it passes**

Run: `go test ./plugin/ -run TestRegistry_RootRoute -v`
Expected: PASS.

- [ ] **Step 6: Wire the extension**

In `extension/extension.go`, inside `if router != nil`, before the grouped `RegisterRoutes` calls at line 518:

```go
			// Origin-root routes first. Well-known discovery documents must
			// answer at the host root, so they cannot go on groupedRouter.
			// In standalone mode router and groupedRouter are the same
			// instance, and registering a path twice panics, so skip the
			// root pass when they coincide.
			if router != groupedRouter {
				for _, rp := range eng.Plugins().RootRouteProviders() {
					if err := rp.RegisterRootRoutes(router); err != nil {
						return fmt.Errorf("authsome: register plugin root routes (%T): %w", rp, err)
					}
				}
			}
```

`groupedRouter` is declared on the line above, so move its declaration up if needed.

- [ ] **Step 7: Move the OIDC document onto the hook**

In `plugins/oauth2provider/plugin.go`, delete the `/.well-known/openid-configuration` registration at lines 225-231 from `RegisterRoutes`, then add:

```go
// RegisterRootRoutes registers discovery documents at the origin root.
// These cannot live on the grouped router: a client that only knows the
// host fetches https://host/.well-known/... with no prefix.
func (p *Plugin) RegisterRootRoutes(router forge.Router) error {
	return p.registerWellKnown(router)
}

// registerWellKnown mounts the discovery documents on the given router.
// Called for the origin root, and again on the grouped router so clients
// configured with a prefixed base URL keep working.
func (p *Plugin) registerWellKnown(router forge.Router) error {
	return router.GET("/.well-known/openid-configuration", p.handleDiscovery,
		forge.WithSummary("OpenID Connect Discovery"),
		forge.WithOperationID("oidcDiscovery"),
		forge.WithTags("OAuth2"),
	)
}
```

Keep the prefixed mirror by calling `p.registerWellKnown(router)` from `RegisterRoutes` where the old block was, but give the mirrored route a distinct operation ID so the OpenAPI spec does not collide:

```go
	// Mirror onto the grouped router so an SDK client whose base URL
	// includes the mount prefix still resolves discovery.
	if err := router.GET("/.well-known/openid-configuration", p.handleDiscovery,
		forge.WithSummary("OpenID Connect Discovery"),
		forge.WithOperationID("oidcDiscoveryPrefixed"),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}
```

Add the compile-time check to the `var` block at the top of `plugin.go`:

```go
	_ plugin.RootRouteProvider = (*Plugin)(nil)
```

- [ ] **Step 8: Run the full oauth2 and plugin suites**

Run: `go test ./plugin/... ./plugins/oauth2provider/... ./extension/... -count=1`
Expected: PASS. If a test asserts the prefixed discovery path, it still passes because of the mirror.

- [ ] **Step 9: Commit**

```bash
git add plugin/plugin.go plugin/registry.go plugin/registry_test.go extension/extension.go plugins/oauth2provider/plugin.go
git commit -m "feat(plugin): add RootRouteProvider for origin-root routes

Plugins only ever received the grouped router, so the OIDC discovery
document was served under the extension mount prefix instead of at the
host root. RFC 8414 and RFC 9728 both define their locations relative to
the origin, so they need a hook that reaches it.

The prefixed path stays registered as a mirror."
```

---

### Task 2: Client model fields and migration

**Files:**
- Modify: `plugins/oauth2provider/models.go:9-22`
- Modify: `plugins/oauth2provider/store_models.go:16-30` (model), `:79-144` (converters)
- Modify: `plugins/oauth2provider/store_mongo.go:40-50`
- Modify: `plugins/oauth2provider/migrations.go`
- Test: `plugins/oauth2provider/store_client_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `OAuth2Client` fields `TokenEndpointAuthMethod string`, `RegistrationTokenHash string`, `DynamicallyRegistered bool`, `ClientSecretExpiresAt time.Time`, `Metadata map[string]any`.

- [ ] **Step 1: Write the failing round-trip test**

Create `plugins/oauth2provider/store_client_test.go`:

```go
package oauth2provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func TestMemoryStore_ClientRoundTripsRegistrationFields(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()
	expires := time.Now().Add(90 * 24 * time.Hour).UTC().Truncate(time.Second)

	want := &oauth2provider.OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   id.NewAppID(),
		Name:                    "Dynamic Client",
		ClientID:                "dyn-1",
		RedirectURIs:            []string{"http://127.0.0.1:9000/cb"},
		Scopes:                  []string{"openid"},
		GrantTypes:              []string{"authorization_code"},
		Public:                  true,
		TokenEndpointAuthMethod: "none",
		RegistrationTokenHash:   "$2a$04$abcdefghijklmnopqrstuv",
		DynamicallyRegistered:   true,
		ClientSecretExpiresAt:   expires,
		Metadata: map[string]any{
			"client_uri":  "https://example.com",
			"software_id": "mcp-cli",
		},
	}
	require.NoError(t, st.CreateClient(ctx, want))

	got, err := st.GetClient(ctx, "dyn-1")
	require.NoError(t, err)
	assert.Equal(t, "none", got.TokenEndpointAuthMethod)
	assert.Equal(t, want.RegistrationTokenHash, got.RegistrationTokenHash)
	assert.True(t, got.DynamicallyRegistered)
	assert.Equal(t, expires, got.ClientSecretExpiresAt.UTC().Truncate(time.Second))
	assert.Equal(t, "mcp-cli", got.Metadata["software_id"])
}

// The registration token hash is a credential. It must never reach a JSON
// response body, the way ClientSecret already does not.
func TestOAuth2Client_RegistrationTokenHashIsNotSerialised(t *testing.T) {
	c := &oauth2provider.OAuth2Client{
		ClientID:              "dyn-1",
		RegistrationTokenHash: "$2a$04$secret",
	}
	b, err := json.Marshal(c)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "secret")
	assert.NotContains(t, string(b), "registration_token_hash")
}
```

Add `"encoding/json"` to the import block.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./plugins/oauth2provider/ -run 'TestMemoryStore_ClientRoundTrips|TestOAuth2Client_Registration' -v`
Expected: FAIL, `unknown field TokenEndpointAuthMethod`.

- [ ] **Step 3: Add the domain fields**

In `plugins/oauth2provider/models.go`, extend `OAuth2Client` after `Public`:

```go
	// TokenEndpointAuthMethod is RFC 7591 token_endpoint_auth_method:
	// "none", "client_secret_basic" or "client_secret_post". It is the
	// source of truth for whether a client is public; Public is derived
	// from it. Two flags that can disagree is a bug waiting to happen.
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`

	// RegistrationTokenHash is the bcrypt hash of the RFC 7592 registration
	// access token. Empty for admin-created clients, which is what makes
	// them unreachable over the 7592 routes.
	RegistrationTokenHash string `json:"-"`

	// DynamicallyRegistered records that this client came in over RFC 7591
	// rather than the admin surface. It gates the 7592 routes and lets the
	// dashboard tell the two populations apart.
	DynamicallyRegistered bool `json:"dynamically_registered"`

	// ClientSecretExpiresAt is RFC 7591 client_secret_expires_at. The zero
	// value means the secret never expires and serialises as 0.
	ClientSecretExpiresAt time.Time `json:"client_secret_expires_at,omitempty"`

	// Metadata holds the RFC 7591 fields that carry no behaviour:
	// client_uri, logo_uri, contacts, tos_uri, policy_uri, software_id,
	// software_version, and anything unrecognised the client sent. They
	// only need to round-trip on a 7592 read, so they do not each earn a
	// column across four backends.
	Metadata map[string]any `json:"metadata,omitempty"`
```

- [ ] **Step 4: Extend the SQL model and converters**

In `store_models.go`, add to `oauth2ClientModel` after `Public`:

```go
	TokenEndpointAuthMethod string          `grove:"token_endpoint_auth_method,notnull"`
	RegistrationTokenHash   string          `grove:"registration_token_hash,notnull"`
	DynamicallyRegistered   bool            `grove:"dynamically_registered,notnull"`
	ClientSecretExpiresAt   *time.Time      `grove:"client_secret_expires_at"`
	Metadata                json.RawMessage `grove:"metadata,type:jsonb"`
```

In `toOAuth2Client`, before the return:

```go
	var metadata map[string]any
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &metadata) //nolint:errcheck // best-effort decode
	}
	var secretExpires time.Time
	if m.ClientSecretExpiresAt != nil {
		secretExpires = *m.ClientSecretExpiresAt
	}
```

and add to the returned struct literal:

```go
		TokenEndpointAuthMethod: m.TokenEndpointAuthMethod,
		RegistrationTokenHash:   m.RegistrationTokenHash,
		DynamicallyRegistered:   m.DynamicallyRegistered,
		ClientSecretExpiresAt:   secretExpires,
		Metadata:                metadata,
```

In `fromOAuth2Client`, before the return:

```go
	metadata, _ := json.Marshal(c.Metadata) //nolint:errcheck // marshaling known types
	if len(metadata) == 0 || string(metadata) == "null" {
		metadata = []byte("{}")
	}
	var secretExpires *time.Time
	if !c.ClientSecretExpiresAt.IsZero() {
		t := c.ClientSecretExpiresAt
		secretExpires = &t
	}
```

and add to the returned struct literal:

```go
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		RegistrationTokenHash:   c.RegistrationTokenHash,
		DynamicallyRegistered:   c.DynamicallyRegistered,
		ClientSecretExpiresAt:   secretExpires,
		Metadata:                metadata,
```

- [ ] **Step 5: Extend the mongo model**

In `store_mongo.go`, add to the client bson struct after `Public`:

```go
	TokenEndpointAuthMethod string         `bson:"token_endpoint_auth_method"`
	RegistrationTokenHash   string         `bson:"registration_token_hash"`
	DynamicallyRegistered   bool           `bson:"dynamically_registered"`
	ClientSecretExpiresAt   *time.Time     `bson:"client_secret_expires_at,omitempty"`
	Metadata                map[string]any `bson:"metadata"`
```

In the mongo `from...` converter, initialise the map explicitly so it marshals as `{}` and never `null`, matching the fix in commit 9116564:

```go
	metadata := c.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
```

Do the same defensive empty-slice init for `RedirectURIs`, `Scopes` and `GrantTypes` if it is not already there.

- [ ] **Step 6: Add the migration**

In `migrations.go`, register on `PostgresMigrations`:

```go
	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "add_dynamic_registration_columns",
			Version: "20260824000001",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients
    ADD COLUMN IF NOT EXISTS token_endpoint_auth_method TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS registration_token_hash    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS dynamically_registered     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS client_secret_expires_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS metadata                   JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_authsome_oauth2_clients_dynamic
    ON authsome_oauth2_clients (app_id)
    WHERE dynamically_registered;
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP INDEX IF EXISTS idx_authsome_oauth2_clients_dynamic;
ALTER TABLE authsome_oauth2_clients
    DROP COLUMN IF EXISTS token_endpoint_auth_method,
    DROP COLUMN IF EXISTS registration_token_hash,
    DROP COLUMN IF EXISTS dynamically_registered,
    DROP COLUMN IF EXISTS client_secret_expires_at,
    DROP COLUMN IF EXISTS metadata;
`)
				return err
			},
		},
	)
```

Register the sqlite variant on `SqliteMigrations` with the same name and version. SQLite has neither `TIMESTAMPTZ` nor `JSONB`, and it takes one `ADD COLUMN` per statement:

```go
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_oauth2_clients ADD COLUMN token_endpoint_auth_method TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_oauth2_clients ADD COLUMN registration_token_hash    TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_oauth2_clients ADD COLUMN dynamically_registered     BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE authsome_oauth2_clients ADD COLUMN client_secret_expires_at   TIMESTAMP;
ALTER TABLE authsome_oauth2_clients ADD COLUMN metadata                   TEXT NOT NULL DEFAULT '{}';
`)
```

The sqlite `Down` can be a no-op returning nil, since older SQLite cannot drop columns. Say so in a comment.

- [ ] **Step 7: Keep the admin create path consistent**

`token_endpoint_auth_method` is the source of truth for whether a client is
public, so the admin path has to write it too. Leaving it empty on
admin-created clients would give the two flags different answers, which is
exactly the disagreement this field exists to prevent.

In `plugins/oauth2provider/plugin.go`, in `handleCreateClient`, add to the
`&OAuth2Client{...}` literal:

```go
		TokenEndpointAuthMethod: authMethodForPublic(req.Public),
```

and add the helper near `clientAllowsGrant`:

```go
// authMethodForPublic maps the admin surface's Public bool onto the RFC 7591
// token_endpoint_auth_method that is the source of truth everywhere else.
func authMethodForPublic(public bool) string {
	if public {
		return "none"
	}
	return "client_secret_basic"
}
```

Add a test for it in `plugins/oauth2provider/store_client_test.go`:

```go
func TestAuthMethodForPublicMatchesTheFlag(t *testing.T) {
	assert.Equal(t, "none", oauth2provider.AuthMethodForPublic(true))
	assert.Equal(t, "client_secret_basic", oauth2provider.AuthMethodForPublic(false))
}
```

The test is in the external package, so export the helper as
`AuthMethodForPublic` and use that name in both places, or move this one
test into the in-package file from Task 4. Pick one and be consistent.

- [ ] **Step 8: Run the tests and confirm they pass**

Run: `go test ./plugins/oauth2provider/ -run 'TestMemoryStore_ClientRoundTrips|TestOAuth2Client_Registration|TestAuthMethodForPublic' -v`
Expected: PASS.

- [ ] **Step 9: Run the full plugin suite**

Run: `go test ./plugins/oauth2provider/... -count=1`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add plugins/oauth2provider/
git commit -m "feat(oauth2): add dynamic registration columns to the client model

Five new fields: token_endpoint_auth_method, registration_token_hash,
dynamically_registered, client_secret_expires_at, and a metadata blob for
the RFC 7591 fields that carry no behaviour.

The blob keeps eight informational fields from each costing a column on
four backends when they only need to round-trip on a 7592 read."
```

---

### Task 3: UpdateClient across four stores

**Files:**
- Modify: `plugins/oauth2provider/store.go:19-24`
- Modify: `plugins/oauth2provider/store_memory.go`
- Modify: `plugins/oauth2provider/store_postgres.go` (client section, around line 88)
- Modify: `plugins/oauth2provider/store_sqlite.go` (same section)
- Modify: `plugins/oauth2provider/store_mongo.go` (client section)
- Test: `plugins/oauth2provider/store_client_test.go` (append)

**Interfaces:**
- Consumes: the `OAuth2Client` fields from Task 2.
- Produces: `Store.UpdateClient(ctx context.Context, c *OAuth2Client) error`. Returns `ErrClientNotFound` when no row matches.

- [ ] **Step 1: Write the failing test**

Append to `plugins/oauth2provider/store_client_test.go`:

```go
func TestMemoryStore_UpdateClient(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	ctx := context.Background()

	c := &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        id.NewAppID(),
		Name:         "Before",
		ClientID:     "dyn-2",
		RedirectURIs: []string{"http://127.0.0.1:9000/cb"},
		Scopes:       []string{"openid"},
		GrantTypes:   []string{"authorization_code"},
	}
	require.NoError(t, st.CreateClient(ctx, c))

	c.Name = "After"
	c.RedirectURIs = []string{"http://127.0.0.1:9100/cb"}
	require.NoError(t, st.UpdateClient(ctx, c))

	got, err := st.GetClient(ctx, "dyn-2")
	require.NoError(t, err)
	assert.Equal(t, "After", got.Name)
	assert.Equal(t, []string{"http://127.0.0.1:9100/cb"}, got.RedirectURIs)
}

func TestMemoryStore_UpdateClientMissing(t *testing.T) {
	st := oauth2provider.NewMemoryStore()
	err := st.UpdateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:       id.NewOAuth2ClientID(),
		ClientID: "nope",
	})
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./plugins/oauth2provider/ -run TestMemoryStore_UpdateClient -v`
Expected: FAIL, `st.UpdateClient undefined`.

- [ ] **Step 3: Add the interface method**

In `store.go`, in the Clients block after `GetClientByID`:

```go
	// UpdateClient persists changes to an existing client, matched on ID.
	// Returns ErrClientNotFound when no row matches.
	UpdateClient(ctx context.Context, c *OAuth2Client) error
```

- [ ] **Step 4: Implement for memory**

In `store_memory.go`, after `GetClientByID`:

```go
func (s *MemoryStore) UpdateClient(_ context.Context, c *OAuth2Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.clients[c.ClientID]
	if !ok || existing.ID != c.ID {
		return ErrClientNotFound
	}
	c.UpdatedAt = time.Now()
	s.clients[c.ClientID] = c
	return nil
}
```

- [ ] **Step 5: Implement for postgres and sqlite**

In `store_postgres.go`, after `ListClients`:

```go
func (s *PostgresStore) UpdateClient(ctx context.Context, c *OAuth2Client) error {
	c.UpdatedAt = time.Now()
	m := fromOAuth2Client(c)
	res, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	if err != nil {
		return oauth2PgError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return oauth2PgError(err)
	}
	if n == 0 {
		return ErrClientNotFound
	}
	return nil
}
```

Add the identical method to `store_sqlite.go`, using that file's own receiver name, driver handle and error helper. Read the surrounding methods first and match them exactly rather than copying the postgres identifiers.

- [ ] **Step 6: Implement for mongo**

In `store_mongo.go`, after `ListClients`:

```go
func (s *MongoStore) UpdateClient(ctx context.Context, c *OAuth2Client) error {
	c.UpdatedAt = time.Now()
	m := fromOAuth2ClientMongo(c)
	res, err := s.mdb.Collection(oauth2ClientsColl).ReplaceOne(ctx,
		bson.M{"_id": c.ID.String()}, m)
	if err != nil {
		return oauth2MongoError(err)
	}
	if res.MatchedCount == 0 {
		return ErrClientNotFound
	}
	return nil
}
```

Use whatever the mongo client converter and collection constant are actually named in that file; read them before writing.

- [ ] **Step 7: Run the tests and confirm they pass**

Run: `go test ./plugins/oauth2provider/ -run TestMemoryStore_UpdateClient -v`
Expected: PASS. The compile-time `var _ Store = (*PostgresStore)(nil)` checks prove the other three satisfy the interface.

- [ ] **Step 8: Commit**

```bash
git add plugins/oauth2provider/
git commit -m "feat(oauth2): add UpdateClient to the store interface

RFC 7592 PUT needs it. All four backends return ErrClientNotFound when no
row matches, so the handler can tell a missing client from a failed write."
```

---

### Task 4: Config and the validation pipeline

Pure functions, no HTTP. This is where the security policy lives, so it gets the densest tests.

**Files:**
- Modify: `plugins/oauth2provider/plugin.go:41-64` (Config)
- Create: `plugins/oauth2provider/register_validate.go`
- Test: `plugins/oauth2provider/register_validate_test.go` (create, package `oauth2provider`)

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `Config.DynamicRegistration bool`, `Config.RegistrationAppID string`, `Config.DynamicRegistrationScopes []string`, `Config.RegistrationRateLimit RateLimit`, `Config.ProtectedResources map[string]ProtectedResource`
  - `type RateLimit struct { Limit int; Window time.Duration }`
  - `type ProtectedResource struct { Resource string; ScopesSupported []string }`
  - `func validateRedirectURI(raw string) error`
  - `func clampGrantTypes(requested []string) ([]string, error)`
  - `func clampScopes(requested, allowlist []string) []string`
  - `func regError(status int, code, desc string) *oauthRegError`
  - `type oauthRegError` implementing `error`, `StatusCode() int`, `ResponseBody() any`
  - Constants `errInvalidRedirectURI = "invalid_redirect_uri"`, `errInvalidClientMetadata = "invalid_client_metadata"`, `errAccessDenied = "access_denied"`

- [ ] **Step 1: Write the failing validation tests**

Create `plugins/oauth2provider/register_validate_test.go`:

```go
package oauth2provider

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		ok   bool
	}{
		{"https with host", "https://app.example.com/cb", true},
		{"https with port", "https://app.example.com:8443/cb", true},
		{"loopback v4", "http://127.0.0.1:9000/cb", true},
		{"loopback v4 no port", "http://127.0.0.1/cb", true},
		{"loopback v6", "http://[::1]:9000/cb", true},
		{"private-use scheme", "com.example.app:/callback", true},

		{"empty", "", false},
		{"http non-loopback", "http://app.example.com/cb", false},
		{"localhost by name", "http://localhost:9000/cb", false},
		{"https no host", "https:///cb", false},
		{"fragment", "https://app.example.com/cb#frag", false},
		{"userinfo", "https://user:pw@app.example.com/cb", false},
		{"wildcard host", "https://*.example.com/cb", false},
		{"scheme without dot", "myapp:/callback", false},
		{"javascript scheme", "javascript:alert(1)", false},
		{"not a url", "://////", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedirectURI(tc.uri)
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestClampGrantTypes(t *testing.T) {
	t.Run("defaults to authorization_code when empty", func(t *testing.T) {
		got, err := clampGrantTypes(nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"authorization_code"}, got)
	})

	t.Run("keeps the allowed pair", func(t *testing.T) {
		got, err := clampGrantTypes([]string{"authorization_code", "refresh_token"})
		require.NoError(t, err)
		assert.Equal(t, []string{"authorization_code", "refresh_token"}, got)
	})

	// A dynamic client holding client_credentials gets a session token with
	// no user and no consent step, so this is a rejection and not a drop.
	t.Run("rejects client_credentials", func(t *testing.T) {
		_, err := clampGrantTypes([]string{"authorization_code", "client_credentials"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client_credentials")
	})

	t.Run("rejects the device grant", func(t *testing.T) {
		_, err := clampGrantTypes([]string{deviceCodeGrantType})
		assert.Error(t, err)
	})
}

func TestClampScopes(t *testing.T) {
	allow := []string{"openid", "profile", "email", "offline_access"}

	t.Run("empty request yields the allowlist", func(t *testing.T) {
		assert.Equal(t, allow, clampScopes(nil, allow))
	})

	t.Run("drops what is outside the allowlist", func(t *testing.T) {
		got := clampScopes([]string{"openid", "admin:all", "email"}, allow)
		assert.Equal(t, []string{"openid", "email"}, got)
	})

	t.Run("everything dropped yields empty, not the allowlist", func(t *testing.T) {
		assert.Empty(t, clampScopes([]string{"admin:all"}, allow))
	})
}

// RFC 7591 section 3.2.2 fixes the error body. MCP clients parse these two
// fields and will not find them in the forge envelope.
func TestRegErrorWireShape(t *testing.T) {
	e := regError(http.StatusBadRequest, errInvalidRedirectURI, "loopback only")
	assert.Equal(t, http.StatusBadRequest, e.StatusCode())

	body, ok := e.ResponseBody().(*oauthRegError)
	require.True(t, ok)
	assert.Equal(t, errInvalidRedirectURI, body.Code)
	assert.Equal(t, "loopback only", body.Desc)
}
```

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./plugins/oauth2provider/ -run 'TestValidateRedirectURI|TestClamp|TestRegError' -v`
Expected: FAIL, `undefined: validateRedirectURI`.

- [ ] **Step 3: Add the config types**

In `plugins/oauth2provider/plugin.go`, add above `Config`:

```go
// RateLimit bounds how often an endpoint may be called.
type RateLimit struct {
	Limit  int
	Window time.Duration
}

// ProtectedResource describes one RFC 9728 protected resource.
type ProtectedResource struct {
	Resource        string
	ScopesSupported []string
}
```

and to `Config`:

```go
	// DynamicRegistration enables RFC 7591 registration. Off by default:
	// an upgrade must never open a public registration endpoint on its own.
	DynamicRegistration bool

	// RegistrationAppID is the app dynamic clients belong to when the
	// request carries no resolvable publishable key. Leave empty on a
	// multi-tenant deployment so an unkeyed request is refused rather
	// than pooled into somebody's app.
	RegistrationAppID string

	// DynamicRegistrationScopes is the allowlist a dynamic client's
	// requested scopes are intersected against. Defaults to openid,
	// profile, email and offline_access.
	DynamicRegistrationScopes []string

	// RegistrationRateLimit caps POST /register per client IP.
	// Defaults to 10 per hour.
	RegistrationRateLimit RateLimit

	// ProtectedResources declares additional RFC 9728 resource identifiers,
	// keyed by the path suffix they are served under. AuthSome always
	// describes itself at the unsuffixed path regardless of this map.
	ProtectedResources map[string]ProtectedResource
```

In `New`, after the existing defaults:

```go
	if len(c.DynamicRegistrationScopes) == 0 {
		c.DynamicRegistrationScopes = []string{"openid", "profile", "email", "offline_access"}
	}
	if c.RegistrationRateLimit.Limit == 0 {
		c.RegistrationRateLimit = RateLimit{Limit: 10, Window: time.Hour}
	}
```

Then, still in `New`, default the logger. It is currently set only in
`OnInit`, so a plugin built directly (which is how every test in this
package builds one) carries a nil logger, and the first handler that logs
nil-panics. `OnInit` still overwrites it with the engine's logger.

```go
	p := &Plugin{config: c, logger: log.NewNoopLogger()}
	return p
```

Replace the existing `return &Plugin{config: c}` with those two lines.

- [ ] **Step 4: Write the validation implementation**

Create `plugins/oauth2provider/register_validate.go`:

```go
package oauth2provider

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// RFC 7591 section 3.2.2 error codes.
const (
	errInvalidRedirectURI    = "invalid_redirect_uri"
	errInvalidClientMetadata = "invalid_client_metadata"
	errAccessDenied          = "access_denied"
)

// dynamicGrantTypes is everything a dynamically registered client may hold.
// client_credentials is absent deliberately: issueClientToken mints a real
// session with a nil user, so an open endpoint handing it out would give
// anyone who can reach /register a working service token with no consent
// step. The device grant is absent for the same reason.
var dynamicGrantTypes = map[string]struct{}{
	"authorization_code": {},
	"refresh_token":      {},
}

// oauthRegError is an RFC 7591 section 3.2.2 error. Forge serialises the
// value returned by ResponseBody verbatim for any status below 500, so this
// puts {"error": ..., "error_description": ...} on the wire instead of the
// house envelope. MCP clients parse those two fields and nothing else.
type oauthRegError struct {
	status int
	Code   string `json:"error"`
	Desc   string `json:"error_description,omitempty"`
}

func (e *oauthRegError) Error() string {
	if e.Desc == "" {
		return e.Code
	}
	return e.Code + ": " + e.Desc
}

func (e *oauthRegError) StatusCode() int  { return e.status }
func (e *oauthRegError) ResponseBody() any { return e }

func regError(status int, code, desc string) *oauthRegError {
	return &oauthRegError{status: status, Code: code, Desc: desc}
}

// validateRedirectURI reports whether a redirect URI may be registered.
//
// Runtime matching is already exact-string in resolveRedirectURI, so this is
// only about what may be recorded in the first place. Three shapes pass:
// https with a real host; http on the loopback literals, which is how
// desktop and CLI clients receive a code (RFC 8252 section 7.3); and a
// private-use scheme containing a dot, which is the RFC 8252 section 7.1
// reverse-domain convention.
//
// http://localhost by name is refused on purpose. Its resolution depends on
// the client host's DNS and hosts file, so it is not the guaranteed-local
// target the literal IP is, and RFC 8252 recommends the IP for that reason.
func validateRedirectURI(raw string) error {
	if raw == "" {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must not be empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			fmt.Sprintf("redirect_uri %q is not a valid URI", raw))
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must not contain a fragment")
	}
	if u.User != nil {
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must not contain userinfo")
	}

	switch u.Scheme {
	case "https":
		if u.Hostname() == "" {
			return regError(http.StatusBadRequest, errInvalidRedirectURI,
				"https redirect_uri requires a host")
		}
		if strings.Contains(u.Hostname(), "*") {
			return regError(http.StatusBadRequest, errInvalidRedirectURI,
				"redirect_uri must not contain a wildcard")
		}
		return nil

	case "http":
		host := u.Hostname()
		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			return nil
		}
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"http redirect_uri is only allowed on the loopback literals 127.0.0.1 and [::1]")

	case "":
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			"redirect_uri must be absolute")

	default:
		// Private-use scheme. RFC 8252 section 7.1 wants a reverse-domain
		// name the app controls, so require a dot. A bare "myapp:" scheme
		// is trivially squattable by another app on the same device.
		if strings.Contains(u.Scheme, ".") {
			return nil
		}
		return regError(http.StatusBadRequest, errInvalidRedirectURI,
			fmt.Sprintf("scheme %q is not allowed; use https, loopback http, or a private-use scheme containing a dot", u.Scheme))
	}
}

// clampGrantTypes limits a registration to the grants a dynamic client may
// hold. An empty request defaults to authorization_code, matching the admin
// path. A request naming a forbidden grant is rejected rather than trimmed:
// the client is asking for a capability, not expressing a preference, and
// silently handing back less would leave it broken in a confusing way.
func clampGrantTypes(requested []string) ([]string, error) {
	if len(requested) == 0 {
		return []string{"authorization_code"}, nil
	}
	out := make([]string, 0, len(requested))
	for _, g := range requested {
		if _, ok := dynamicGrantTypes[g]; !ok {
			return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
				fmt.Sprintf("grant_type %q is not available to dynamically registered clients", g))
		}
		out = append(out, g)
	}
	return out, nil
}

// clampScopes intersects requested scopes with the allowlist, dropping
// anything outside it. RFC 7591 section 2 lets the server substitute, and
// dropping beats erroring because clients tend to ask for a broad set
// optimistically; the response echoes what was actually granted. An empty
// request yields the full allowlist.
func clampScopes(requested, allowlist []string) []string {
	if len(requested) == 0 {
		return append([]string(nil), allowlist...)
	}
	allowed := make(map[string]struct{}, len(allowlist))
	for _, s := range allowlist {
		allowed[s] = struct{}{}
	}
	out := make([]string, 0, len(requested))
	for _, s := range requested {
		if _, ok := allowed[s]; ok {
			out = append(out, s)
		}
	}
	return out
}
```

- [ ] **Step 5: Run the tests and confirm they pass**

Run: `go test ./plugins/oauth2provider/ -run 'TestValidateRedirectURI|TestClamp|TestRegError' -v`
Expected: PASS, all sub-tests.

- [ ] **Step 6: Commit**

```bash
git add plugins/oauth2provider/register_validate.go plugins/oauth2provider/register_validate_test.go plugins/oauth2provider/plugin.go
git commit -m "feat(oauth2): add the dynamic registration policy pipeline

Redirect URIs are limited to https, the loopback literals, and dotted
private-use schemes. Grants are clamped to authorization_code and
refresh_token, with anything else rejected rather than trimmed. Scopes are
intersected against an allowlist and silently dropped, which RFC 7591
section 2 permits.

Errors carry the RFC 7591 section 3.2.2 body via forge's HTTPResponder
hook, so MCP clients get the two fields they parse."
```

---

### Task 5: POST /register

**Files:**
- Create: `plugins/oauth2provider/register.go`
- Modify: `plugins/oauth2provider/plugin.go` (RegisterRoutes)
- Test: `plugins/oauth2provider/register_test.go` (create, package `oauth2provider_test`)

**Interfaces:**
- Consumes: `validateRedirectURI`, `clampGrantTypes`, `clampScopes`, `regError` and the error constants from Task 4; the `OAuth2Client` fields from Task 2.
- Produces:
  - `type RegisterClientRequest` with JSON fields `redirect_uris []string`, `client_name string`, `grant_types []string`, `response_types []string`, `scope string`, `token_endpoint_auth_method string`, `client_uri`, `logo_uri`, `tos_uri`, `policy_uri`, `software_id`, `software_version string`, `contacts []string`
  - `type RegisterClientResponse` with `client_id`, `client_secret`, `client_id_issued_at`, `client_secret_expires_at`, `registration_access_token`, `registration_client_uri`, `redirect_uris`, `grant_types`, `scope`, `token_endpoint_auth_method`, `client_name`
  - `func (p *Plugin) handleRegisterClient(ctx forge.Context, req *RegisterClientRequest) (*RegisterClientResponse, error)`
  - `func (p *Plugin) resolveRegistrationAppID(ctx forge.Context) (id.AppID, error)`

- [ ] **Step 1: Write the failing tests**

Create `plugins/oauth2provider/register_test.go`. Build the plugin the way `newFixture` does, add `Config{DynamicRegistration: true, RegistrationAppID: <app>.String(), Issuer: "https://auth.example.com"}`, and drive the handler through a `httptest` request the way `authcode_test.go` already does.

```go
package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

func postRegister(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRegister_HappyPathPublicClient(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)

	rec := postRegister(t, router, `{
		"client_name": "MCP CLI",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none",
		"grant_types": ["authorization_code", "refresh_token"],
		"scope": "openid profile email"
	}`)

	require.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.NotEmpty(t, got["client_id"])
	assert.NotEmpty(t, got["registration_access_token"])
	assert.Contains(t, got["registration_client_uri"], got["client_id"])
	// A public client gets no secret.
	assert.Empty(t, got["client_secret"])
	assert.Equal(t, "openid profile email", got["scope"])
}

func TestRegister_ConfidentialClientGetsSecretOnce(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)

	rec := postRegister(t, router, `{
		"client_name": "Server",
		"redirect_uris": ["https://app.example.com/cb"],
		"token_endpoint_auth_method": "client_secret_post"
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	secret, _ := got["client_secret"].(string)
	require.NotEmpty(t, secret)

	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	// Stored hashed, never in the clear.
	assert.NotEqual(t, secret, stored.ClientSecret)
	assert.NotEmpty(t, stored.RegistrationTokenHash)
	assert.True(t, stored.DynamicallyRegistered)
}

func TestRegister_DisabledReturns404(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, false)
	rec := postRegister(t, router, `{"redirect_uris":["https://app.example.com/cb"]}`)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRegister_RejectsClientCredentials(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"grant_types": ["client_credentials"]
	}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
	assert.Contains(t, got["error_description"], "client_credentials")
}

func TestRegister_DropsScopesOutsideAllowlist(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"scope": "openid admin:all email"
	}`)

	require.Equal(t, http.StatusCreated, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "openid email", got["scope"])
}

func TestRegister_RejectsBadRedirectURI(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{"redirect_uris":["http://evil.example.com/cb"]}`)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_redirect_uri", got["error"])
}

func TestRegister_RequiresRedirectURIs(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{"client_name":"No URIs"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// With no publishable key on the request and no configured fallback, there
// is no app to attach the client to and the request must be refused rather
// than pooled into somebody's tenant.
func TestRegister_NoAppResolvesTo403(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: true,
		// RegistrationAppID deliberately unset.
	})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	router := newTestRouter(t, p)

	rec := postRegister(t, router, `{"redirect_uris":["https://app.example.com/cb"]}`)
	require.Equal(t, http.StatusForbidden, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "access_denied", got["error"])
}

func TestRegister_RoundTripsInformationalMetadata(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	rec := postRegister(t, router, `{
		"redirect_uris": ["https://app.example.com/cb"],
		"client_uri": "https://example.com",
		"software_id": "mcp-cli",
		"software_version": "2.1.0",
		"contacts": ["ops@example.com"]
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	stored, err := st.GetClient(t.Context(), got["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, "mcp-cli", stored.Metadata["software_id"])
	assert.Equal(t, "https://example.com", stored.Metadata["client_uri"])
}
```

Write two helpers in the same file. `newTestRouter(t, p)` builds a `forge` router and calls `p.RegisterRoutes(router)`, mirroring however `authcode_test.go` already stands a router up; read that file and reuse its approach rather than inventing a second one. `newRegistrationFixture(t, enabled bool)` returns `(*oauth2provider.Plugin, oauth2provider.Store, http.Handler, id.AppID)`, creating an `id.NewAppID()` and passing it as `RegistrationAppID`.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./plugins/oauth2provider/ -run TestRegister_ -v`
Expected: FAIL, no route registered so every case 404s (and the 404 case passes for the wrong reason, which the next steps fix).

- [ ] **Step 3: Write the request and response types**

Create `plugins/oauth2provider/register.go`:

```go
package oauth2provider

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// RegisterClientRequest is an RFC 7591 client registration request.
type RegisterClientRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// RegisterClientResponse is an RFC 7591 client information response. It is
// also what the RFC 7592 read and update endpoints return.
type RegisterClientResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RegistrationAccessToken string   `json:"registration_access_token,omitempty"`
	RegistrationClientURI   string   `json:"registration_client_uri,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}
```

- [ ] **Step 4: Write the app resolver and the handler**

Append to `register.go`:

```go
// resolveRegistrationAppID picks the app a new dynamic client belongs to.
//
// RFC 7591 has no field for it and a body field would let any caller name
// any tenant, so it comes off the transport: the publishable key middleware
// resolves X-Publishable-Key (or ?publishable_key=) onto the context. That
// middleware never aborts, so an unresolvable key looks exactly like a
// missing one here, and the config fallback is what a single-tenant
// deployment sets so a stock MCP client can register with no key at all.
func (p *Plugin) resolveRegistrationAppID(ctx forge.Context) (id.AppID, error) {
	if appID, ok := middleware.AppIDFrom(ctx.Context()); ok {
		return appID, nil
	}
	if p.config.RegistrationAppID != "" {
		appID, err := id.ParseAppID(p.config.RegistrationAppID)
		if err != nil {
			return id.AppID{}, forge.InternalError(
				fmt.Errorf("oauth2: RegistrationAppID is not a valid app id: %w", err))
		}
		return appID, nil
	}
	return id.AppID{}, regError(http.StatusForbidden, errAccessDenied,
		"registration requires a publishable key on this deployment")
}

// issuerURL returns the configured issuer, or the localhost default the
// discovery handler already falls back to.
func (p *Plugin) issuerURL() string {
	if p.config.Issuer != "" {
		return p.config.Issuer
	}
	return "https://localhost"
}

func (p *Plugin) registrationClientURI(clientID string) string {
	return p.issuerURL() + "/v1/oauth/register/" + clientID
}

// metadataFromRequest collects the RFC 7591 fields that carry no behaviour.
// They only have to round-trip on a 7592 read, so they live in one blob
// instead of eight columns across four backends.
func metadataFromRequest(req *RegisterClientRequest) map[string]any {
	m := map[string]any{}
	for k, v := range map[string]string{
		"client_uri":       req.ClientURI,
		"logo_uri":         req.LogoURI,
		"tos_uri":          req.TOSURI,
		"policy_uri":       req.PolicyURI,
		"software_id":      req.SoftwareID,
		"software_version": req.SoftwareVersion,
	} {
		if v != "" {
			m[k] = v
		}
	}
	if len(req.Contacts) > 0 {
		m["contacts"] = req.Contacts
	}
	return m
}

func (p *Plugin) handleRegisterClient(ctx forge.Context, req *RegisterClientRequest) (*RegisterClientResponse, error) {
	if !p.config.DynamicRegistration {
		return nil, forge.NotFound("dynamic client registration is not enabled")
	}

	appID, err := p.resolveRegistrationAppID(ctx)
	if err != nil {
		return nil, err
	}

	if len(req.RedirectURIs) == 0 {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			"redirect_uris is required")
	}
	for _, u := range req.RedirectURIs {
		if err := validateRedirectURI(u); err != nil {
			return nil, err
		}
	}

	grantTypes, err := clampGrantTypes(req.GrantTypes)
	if err != nil {
		return nil, err
	}
	scopes := clampScopes(strings.Fields(req.Scope), p.config.DynamicRegistrationScopes)

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "client_secret_basic"
	}
	switch authMethod {
	case "none", "client_secret_basic", "client_secret_post":
	default:
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("token_endpoint_auth_method %q is not supported", authMethod))
	}
	isPublic := authMethod == "none"

	clientIDStr, err := generateSecureToken(16)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_id: %w", err))
	}

	var rawSecret, hashedSecret string
	if !isPublic {
		rawSecret, err = generateSecureToken(32)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: generate client_secret: %w", err))
		}
		h, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if err != nil {
			return nil, forge.InternalError(fmt.Errorf("oauth2: hash client_secret: %w", err))
		}
		hashedSecret = string(h)
	}

	rawRegToken, err := generateSecureToken(32)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: generate registration token: %w", err))
	}
	regHash, err := bcrypt.GenerateFromPassword([]byte(rawRegToken), bcrypt.DefaultCost)
	if err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: hash registration token: %w", err))
	}

	now := time.Now()
	client := &OAuth2Client{
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   appID,
		Name:                    req.ClientName,
		ClientID:                clientIDStr,
		ClientSecret:            hashedSecret,
		RedirectURIs:            req.RedirectURIs,
		Scopes:                  scopes,
		GrantTypes:              grantTypes,
		Public:                  isPublic,
		TokenEndpointAuthMethod: authMethod,
		RegistrationTokenHash:   string(regHash),
		DynamicallyRegistered:   true,
		Metadata:                metadataFromRequest(req),
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := p.oauth2Store.CreateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: create dynamic client: %w", err))
	}

	p.logger.Info("oauth2: dynamic client registered",
		log.String("client_id", client.ClientID),
		log.String("app_id", appID.String()),
		log.String("client_name", client.Name))

	resp := p.clientInfoResponse(client)
	resp.ClientSecret = rawSecret
	resp.RegistrationAccessToken = rawRegToken
	return resp, nil
}

// clientInfoResponse renders the RFC 7591 client information response. It
// omits both credentials: the caller adds the raw secret and registration
// token on the one response that is allowed to carry them.
func (p *Plugin) clientInfoResponse(c *OAuth2Client) *RegisterClientResponse {
	var secretExpires int64
	if !c.ClientSecretExpiresAt.IsZero() {
		secretExpires = c.ClientSecretExpiresAt.Unix()
	}
	str := func(k string) string {
		v, _ := c.Metadata[k].(string)
		return v
	}
	var contacts []string
	if raw, ok := c.Metadata["contacts"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				contacts = append(contacts, s)
			}
		}
	} else if raw, ok := c.Metadata["contacts"].([]string); ok {
		contacts = raw
	}

	return &RegisterClientResponse{
		ClientID:                c.ClientID,
		ClientIDIssuedAt:        c.CreatedAt.Unix(),
		ClientSecretExpiresAt:   secretExpires,
		RegistrationClientURI:   p.registrationClientURI(c.ClientID),
		RedirectURIs:            c.RedirectURIs,
		GrantTypes:              c.GrantTypes,
		ResponseTypes:           []string{"code"},
		Scope:                   strings.Join(c.Scopes, " "),
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		ClientName:              c.Name,
		ClientURI:               str("client_uri"),
		LogoURI:                 str("logo_uri"),
		TOSURI:                  str("tos_uri"),
		PolicyURI:               str("policy_uri"),
		SoftwareID:              str("software_id"),
		SoftwareVersion:         str("software_version"),
		Contacts:                contacts,
	}
}
```

Add `log "github.com/xraph/go-utils/log"` to the import block.

- [ ] **Step 5: Register the route with its rate limit**

In `plugins/oauth2provider/plugin.go`, inside `RegisterRoutes` after the device authorize route:

```go
	// RFC 7591 dynamic client registration. Unauthenticated by design, so
	// it is rate limited by IP and returns 404 unless explicitly enabled.
	regOpts := []forge.RouteOption{
		forge.WithSummary("Register OAuth2 client"),
		forge.WithDescription("RFC 7591 dynamic client registration."),
		forge.WithOperationID("oauth2RegisterClient"),
		forge.WithRequestSchema(RegisterClientRequest{}),
		forge.WithResponseSchema(http.StatusCreated, "Client registered", RegisterClientResponse{}),
		forge.WithErrorResponses(),
	}
	if p.engine != nil {
		if rl := p.engine.RateLimiter(); rl != nil {
			regOpts = append(regOpts, forge.WithMiddleware(middleware.RateLimit(rl, middleware.RateLimitConfig{
				Limit:  p.config.RegistrationRateLimit.Limit,
				Window: p.config.RegistrationRateLimit.Window,
			})))
		}
	}
	if err := g.POST("/register", p.handleRegisterClient, regOpts...); err != nil {
		return err
	}
```

If `forge.WithMiddleware` is not the right route option name, read the other route registrations and the forge router package for the correct one, then use that. Do not silently drop the rate limit.

`p.engine.RateLimiter()` needs to exist on the `plugin.Engine` interface. Check `plugin/plugin.go` around line 121; if `RateLimiter()` is absent, add it to the interface (the concrete `*Engine` already has it at `engine.go:822`).

Also set the created status. If forge derives 201 from the response schema option above, nothing more is needed; if it defaults to 200, find the route option that sets the success status and add it, and assert the code in the test.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test ./plugins/oauth2provider/ -run TestRegister_ -v`
Expected: PASS, all nine cases.

- [ ] **Step 7: Run the full plugin suite**

Run: `go test ./plugins/oauth2provider/... -count=1 && make lint`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add plugins/oauth2provider/
git commit -m "feat(oauth2): add RFC 7591 dynamic client registration

POST /v1/oauth/register, off unless Config.DynamicRegistration is set. The
app comes from the publishable key with a config fallback, so a stock MCP
client can register on a single-tenant deployment and a multi-tenant one
refuses an unkeyed request instead of pooling it.

Rate limited by IP. The registration access token is bcrypt hashed and
returned exactly once."
```

---

### Task 6: RFC 7592 read, update and delete

**Files:**
- Modify: `plugins/oauth2provider/register.go`
- Modify: `plugins/oauth2provider/plugin.go` (RegisterRoutes)
- Test: `plugins/oauth2provider/register_manage_test.go` (create)

**Interfaces:**
- Consumes: everything Task 5 produced, plus `Store.UpdateClient` from Task 3.
- Produces:
  - `type RegistrationRequest struct { ClientID string \`path:"clientId"\` }`
  - `type UpdateRegistrationRequest` embedding the RFC 7591 fields plus `ClientID string \`path:"clientId"\`` and body `client_id` / `client_secret`
  - `func (p *Plugin) authenticateRegistration(ctx forge.Context, clientID string) (*OAuth2Client, error)`
  - Handlers `handleReadRegistration`, `handleUpdateRegistration`, `handleDeleteRegistration`

- [ ] **Step 1: Write the failing tests**

Create `plugins/oauth2provider/register_manage_test.go`. Register a client through `POST /register` first, keep the returned token, then exercise the three routes.

```go
package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registerOne registers a client and returns its id and registration token.
func registerOne(t *testing.T, router http.Handler) (clientID, regToken string) {
	t.Helper()
	rec := postRegister(t, router, `{
		"client_name": "Managed",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"token_endpoint_auth_method": "none",
		"scope": "openid profile"
	}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got["client_id"].(string), got["registration_access_token"].(string)
}

func manageReq(t *testing.T, router http.Handler, method, clientID, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, "/v1/oauth/register/"+clientID, nil)
	} else {
		r = httptest.NewRequest(method, "/v1/oauth/register/"+clientID, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, r)
	return rec
}

func TestManage_ReadReturnsRegistration(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodGet, clientID, token, "")
	require.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, clientID, got["client_id"])
	assert.Equal(t, "Managed", got["client_name"])
	// The token is never re-issued on a read.
	assert.Empty(t, got["registration_access_token"])
}

func TestManage_WrongTokenIs401(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, _ := registerOne(t, router)

	rec := manageReq(t, router, http.MethodGet, clientID, "not-the-token", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer")
}

func TestManage_MissingTokenIs401(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, _ := registerOne(t, router)
	rec := manageReq(t, router, http.MethodGet, clientID, "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// A token is scoped to the client it was issued for. Presenting client A's
// token against client B must fail even though the token itself is valid.
func TestManage_TokenFromAnotherClientIs401(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	_, tokenA := registerOne(t, router)
	clientB, _ := registerOne(t, router)

	rec := manageReq(t, router, http.MethodGet, clientB, tokenA, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// Admin-created clients have no registration token hash, so they are not
// reachable over 7592 even if the client_id is known.
func TestManage_AdminCreatedClientIsUnreachable(t *testing.T) {
	_, st, router, appID := newRegistrationFixture(t, true)
	require.NoError(t, st.CreateClient(t.Context(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     "admin-made",
		Name:         "Admin",
		RedirectURIs: []string{"https://app.example.com/cb"},
		GrantTypes:   []string{"authorization_code"},
	}))

	rec := manageReq(t, router, http.MethodGet, "admin-made", "anything", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestManage_UpdateChangesRedirectURIs(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"client_name": "Renamed",
		"redirect_uris": ["http://127.0.0.1:9500/cb"],
		"token_endpoint_auth_method": "none"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", stored.Name)
	assert.Equal(t, []string{"http://127.0.0.1:9500/cb"}, stored.RedirectURIs)
}

// The whole point of running updates through the same pipeline: an update
// must not be able to buy a capability registration refused.
func TestManage_UpdateCannotWidenGrants(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"grant_types": ["authorization_code", "client_credentials"]
	}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.NotContains(t, stored.GrantTypes, "client_credentials")
}

func TestManage_UpdateCannotWidenScopes(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "`+clientID+`",
		"redirect_uris": ["http://127.0.0.1:9000/cb"],
		"scope": "openid admin:all"
	}`)
	require.Equal(t, http.StatusOK, rec.Code)

	stored, err := st.GetClient(t.Context(), clientID)
	require.NoError(t, err)
	assert.NotContains(t, stored.Scopes, "admin:all")
}

func TestManage_UpdateRejectsMismatchedClientID(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodPut, clientID, token, `{
		"client_id": "some-other-id",
		"redirect_uris": ["http://127.0.0.1:9000/cb"]
	}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "invalid_client_metadata", got["error"])
}

func TestManage_Delete(t *testing.T) {
	_, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	rec := manageReq(t, router, http.MethodDelete, clientID, token, "")
	require.Equal(t, http.StatusNoContent, rec.Code)

	_, err := st.GetClient(t.Context(), clientID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}

// Turning registration off closes the door to new clients. It must not
// strand the ones that came in while it was open: an operator still needs
// DELETE, and a client still needs to see that it was revoked.
func TestManage_StillWorksWhenRegistrationDisabled(t *testing.T) {
	p, st, router, _ := newRegistrationFixture(t, true)
	clientID, token := registerOne(t, router)

	p.SetDynamicRegistrationForTest(false)

	rec := manageReq(t, router, http.MethodGet, clientID, token, "")
	assert.Equal(t, http.StatusOK, rec.Code)

	rec = manageReq(t, router, http.MethodDelete, clientID, token, "")
	require.Equal(t, http.StatusNoContent, rec.Code)
	_, err := st.GetClient(t.Context(), clientID)
	assert.ErrorIs(t, err, oauth2provider.ErrClientNotFound)
}
```

Add the imports the file needs (`id`, `oauth2provider`).

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./plugins/oauth2provider/ -run TestManage_ -v`
Expected: FAIL, routes not registered and `SetDynamicRegistrationForTest` undefined.

- [ ] **Step 3: Add the test seam**

In `plugins/oauth2provider/plugin.go`, next to `SetOAuth2Store`:

```go
// SetDynamicRegistrationForTest toggles RFC 7591 registration after
// construction. Tests use it to prove the 7592 routes keep working once
// registration itself is closed.
func (p *Plugin) SetDynamicRegistrationForTest(enabled bool) {
	p.config.DynamicRegistration = enabled
}
```

- [ ] **Step 4: Write the authenticator and handlers**

Append to `plugins/oauth2provider/register.go`:

```go
// RegistrationRequest addresses one registration by its client_id.
type RegistrationRequest struct {
	ClientID string `path:"clientId"`
}

// UpdateRegistrationRequest is an RFC 7592 client update. The body repeats
// the full registration; anything omitted is cleared, per RFC 7592
// section 2.2, which specifies a replacement and not a merge.
type UpdateRegistrationRequest struct {
	ClientID string `path:"clientId"`

	BodyClientID     string   `json:"client_id,omitempty"`
	BodyClientSecret string   `json:"client_secret,omitempty"`
	RedirectURIs     []string `json:"redirect_uris"`
	ClientName       string   `json:"client_name,omitempty"`
	GrantTypes       []string `json:"grant_types,omitempty"`
	Scope            string   `json:"scope,omitempty"`

	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// authenticateRegistration resolves the client named in the path and checks
// the bearer registration access token against its stored hash.
//
// Every failure returns the same 401 so the endpoint does not tell an
// unauthenticated caller which client_ids exist. Admin-created clients have
// an empty hash and are rejected before the compare, which is what keeps
// them off these routes entirely.
func (p *Plugin) authenticateRegistration(ctx forge.Context, clientID string) (*OAuth2Client, error) {
	unauthorized := func() error {
		ctx.Response().Header().Set("WWW-Authenticate", `Bearer realm="registration"`)
		return regError(http.StatusUnauthorized, "invalid_token",
			"a valid registration access token is required")
	}

	raw := ctx.Request().Header.Get("Authorization")
	const prefix = "Bearer "
	if len(raw) <= len(prefix) || !strings.EqualFold(raw[:len(prefix)], prefix) {
		return nil, unauthorized()
	}
	token := strings.TrimSpace(raw[len(prefix):])
	if token == "" {
		return nil, unauthorized()
	}

	client, err := p.oauth2Store.GetClient(ctx.Context(), clientID)
	if err != nil {
		return nil, unauthorized()
	}
	if !client.DynamicallyRegistered || client.RegistrationTokenHash == "" {
		return nil, unauthorized()
	}
	if err := bcrypt.CompareHashAndPassword(
		[]byte(client.RegistrationTokenHash), []byte(token)); err != nil {
		return nil, unauthorized()
	}
	return client, nil
}

func (p *Plugin) handleReadRegistration(ctx forge.Context, req *RegistrationRequest) (*RegisterClientResponse, error) {
	client, err := p.authenticateRegistration(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	// The registration access token is not rotated on read. RFC 7592
	// permits rotation, and it strands any client that does not persist
	// the new value, which in practice is most of them.
	return p.clientInfoResponse(client), nil
}

func (p *Plugin) handleUpdateRegistration(ctx forge.Context, req *UpdateRegistrationRequest) (*RegisterClientResponse, error) {
	client, err := p.authenticateRegistration(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}

	// RFC 7592 section 2.2: a client_id in the body must match the one
	// being updated, and a client_secret must match the current one.
	if req.BodyClientID != "" && req.BodyClientID != client.ClientID {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			"client_id in the body does not match the registration being updated")
	}
	if req.BodyClientSecret != "" {
		if bcrypt.CompareHashAndPassword(
			[]byte(client.ClientSecret), []byte(req.BodyClientSecret)) != nil {
			return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
				"client_secret in the body does not match the current secret")
		}
	}

	if len(req.RedirectURIs) == 0 {
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			"redirect_uris is required")
	}
	for _, u := range req.RedirectURIs {
		if err := validateRedirectURI(u); err != nil {
			return nil, err
		}
	}

	// Same clamps as registration. An update must not be able to buy a
	// capability that registration refused.
	grantTypes, err := clampGrantTypes(req.GrantTypes)
	if err != nil {
		return nil, err
	}
	scopes := clampScopes(strings.Fields(req.Scope), p.config.DynamicRegistrationScopes)

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = client.TokenEndpointAuthMethod
	}
	switch authMethod {
	case "none", "client_secret_basic", "client_secret_post":
	default:
		return nil, regError(http.StatusBadRequest, errInvalidClientMetadata,
			fmt.Sprintf("token_endpoint_auth_method %q is not supported", authMethod))
	}

	client.Name = req.ClientName
	client.RedirectURIs = req.RedirectURIs
	client.GrantTypes = grantTypes
	client.Scopes = scopes
	client.TokenEndpointAuthMethod = authMethod
	client.Public = authMethod == "none"
	client.Metadata = metadataFromRequest(&RegisterClientRequest{
		ClientURI:       req.ClientURI,
		LogoURI:         req.LogoURI,
		TOSURI:          req.TOSURI,
		PolicyURI:       req.PolicyURI,
		SoftwareID:      req.SoftwareID,
		SoftwareVersion: req.SoftwareVersion,
		Contacts:        req.Contacts,
	})

	if err := p.oauth2Store.UpdateClient(ctx.Context(), client); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: update registration: %w", err))
	}
	return p.clientInfoResponse(client), nil
}

func (p *Plugin) handleDeleteRegistration(ctx forge.Context, req *RegistrationRequest) (*apitypes.Empty, error) {
	client, err := p.authenticateRegistration(ctx, req.ClientID)
	if err != nil {
		return nil, err
	}
	if err := p.oauth2Store.DeleteClient(ctx.Context(), client.ID); err != nil {
		return nil, forge.InternalError(fmt.Errorf("oauth2: delete registration: %w", err))
	}
	p.logger.Info("oauth2: dynamic client deleted",
		log.String("client_id", client.ClientID))
	return &apitypes.Empty{}, nil
}
```

Add `"github.com/xraph/authsome/apitypes"` to the imports.

- [ ] **Step 5: Register the three routes**

In `RegisterRoutes`, after the `/register` route. These stay live regardless of `Config.DynamicRegistration`, and they share a looser limit keyed by client_id:

```go
	// RFC 7592 registration management. Registered unconditionally: turning
	// registration off must not strand clients that already registered.
	manageOpts := func(extra ...forge.RouteOption) []forge.RouteOption {
		base := []forge.RouteOption{forge.WithErrorResponses()}
		if p.engine != nil {
			if rl := p.engine.RateLimiter(); rl != nil {
				base = append(base, forge.WithMiddleware(middleware.RateLimit(rl, middleware.RateLimitConfig{
					Limit:  p.config.RegistrationRateLimit.Limit * 6,
					Window: p.config.RegistrationRateLimit.Window,
					KeyFunc: func(c forge.Context) string {
						return "oauth2-reg-manage:" + c.Param("clientId")
					},
				})))
			}
		}
		return append(base, extra...)
	}

	if err := g.GET("/register/:clientId", p.handleReadRegistration, manageOpts(
		forge.WithSummary("Read OAuth2 client registration"),
		forge.WithOperationID("oauth2ReadRegistration"),
		forge.WithResponseSchema(http.StatusOK, "Client information", RegisterClientResponse{}),
	)...); err != nil {
		return err
	}

	if err := g.PUT("/register/:clientId", p.handleUpdateRegistration, manageOpts(
		forge.WithSummary("Update OAuth2 client registration"),
		forge.WithOperationID("oauth2UpdateRegistration"),
		forge.WithRequestSchema(UpdateRegistrationRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Client information", RegisterClientResponse{}),
	)...); err != nil {
		return err
	}

	if err := g.DELETE("/register/:clientId", p.handleDeleteRegistration, manageOpts(
		forge.WithSummary("Delete OAuth2 client registration"),
		forge.WithOperationID("oauth2DeleteRegistration"),
		forge.WithNoContentResponse(),
	)...); err != nil {
		return err
	}
```

Confirm `c.Param` is the right accessor for a path parameter on `forge.Context`; read another handler that reads one, and match it.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test ./plugins/oauth2provider/ -run TestManage_ -v`
Expected: PASS, all twelve cases.

- [ ] **Step 7: Run the full suite and lint**

Run: `go test ./plugins/oauth2provider/... -count=1 && make lint`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add plugins/oauth2provider/
git commit -m "feat(oauth2): add RFC 7592 registration management

GET, PUT and DELETE on /v1/oauth/register/{client_id}, authenticated by the
registration access token. Updates run the same clamps registration does, so
a client cannot buy a grant or scope that registration refused.

Admin-created clients have no token hash and stay unreachable. The routes
survive Config.DynamicRegistration being turned off, so closing the door to
new clients does not strand the ones already through it."
```

---

### Task 7: RFC 8414 and RFC 9728 metadata

**Files:**
- Create: `plugins/oauth2provider/metadata.go`
- Modify: `plugins/oauth2provider/plugin.go` (`registerWellKnown`, `handleDiscovery`, `DiscoveryResponse`)
- Test: `plugins/oauth2provider/metadata_test.go` (create)

**Interfaces:**
- Consumes: `Config.DynamicRegistration`, `Config.ProtectedResources`, `p.issuerURL()` from Task 5.
- Produces:
  - `type AuthServerMetadata` with all the `DiscoveryResponse` fields plus `RegistrationEndpoint string \`json:"registration_endpoint,omitempty"\``
  - `type ProtectedResourceMetadata struct { Resource string; AuthorizationServers []string; ScopesSupported []string; BearerMethodsSupported []string }`
  - `func (p *Plugin) buildAuthServerMetadata() *AuthServerMetadata`
  - `func (p *Plugin) handleAuthServerMetadata(forge.Context, *DiscoveryRequest) (*AuthServerMetadata, error)`
  - `func (p *Plugin) handleProtectedResourceMetadata(forge.Context, *DiscoveryRequest) (*ProtectedResourceMetadata, error)`
  - `func (p *Plugin) handleScopedProtectedResourceMetadata(forge.Context, *ProtectedResourceRequest) (*ProtectedResourceMetadata, error)`

- [ ] **Step 1: Write the failing tests**

Create `plugins/oauth2provider/metadata_test.go`:

```go
package oauth2provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/oauth2provider"
)

func getJSON(t *testing.T, router http.Handler, path string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return rec.Code, got
}

func TestMetadata_AuthServerDocument(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)

	code, got := getJSON(t, router, "/.well-known/oauth-authorization-server")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "https://auth.example.com", got["issuer"])
	assert.Equal(t, "https://auth.example.com/v1/oauth/token", got["token_endpoint"])
	assert.Equal(t, "https://auth.example.com/v1/oauth/register", got["registration_endpoint"])
}

// Advertising an endpoint that 404s sends a client down a dead path and
// produces a worse error than not advertising it.
func TestMetadata_NoRegistrationEndpointWhenDisabled(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, false)

	code, got := getJSON(t, router, "/.well-known/oauth-authorization-server")
	require.Equal(t, http.StatusOK, code)
	_, present := got["registration_endpoint"]
	assert.False(t, present)
}

// One builder feeds both documents, so they cannot drift apart.
func TestMetadata_OIDCAndAuthServerAgree(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)

	_, as := getJSON(t, router, "/.well-known/oauth-authorization-server")
	_, oidc := getJSON(t, router, "/.well-known/openid-configuration")

	for _, k := range []string{
		"issuer", "authorization_endpoint", "token_endpoint", "jwks_uri",
		"registration_endpoint", "grant_types_supported",
		"code_challenge_methods_supported",
	} {
		assert.Equal(t, as[k], oidc[k], "field %q differs between the two documents", k)
	}
}

func TestMetadata_ProtectedResourceDocument(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)

	code, got := getJSON(t, router, "/.well-known/oauth-protected-resource")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "https://auth.example.com", got["resource"])
	assert.Equal(t, []any{"https://auth.example.com"}, got["authorization_servers"])
	assert.Equal(t, []any{"header"}, got["bearer_methods_supported"])
}

func TestMetadata_ConfiguredExtraResource(t *testing.T) {
	p := oauth2provider.New(oauth2provider.Config{
		Issuer:              "https://auth.example.com",
		DynamicRegistration: true,
		ProtectedResources: map[string]oauth2provider.ProtectedResource{
			"mcp": {
				Resource:        "https://mcp.example.com",
				ScopesSupported: []string{"openid", "profile"},
			},
		},
	})
	p.SetOAuth2Store(oauth2provider.NewMemoryStore())
	router := newTestRouter(t, p)

	code, got := getJSON(t, router, "/.well-known/oauth-protected-resource/mcp")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "https://mcp.example.com", got["resource"])
	assert.Equal(t, []any{"https://auth.example.com"}, got["authorization_servers"])
	assert.Equal(t, []any{"openid", "profile"}, got["scopes_supported"])
}

func TestMetadata_UnknownExtraResourceIs404(t *testing.T) {
	_, _, router, _ := newRegistrationFixture(t, true)
	code, _ := getJSON(t, router, "/.well-known/oauth-protected-resource/nope")
	assert.Equal(t, http.StatusNotFound, code)
}
```

`newTestRouter` must call both `RegisterRoutes` and `RegisterRootRoutes` so the well-known paths resolve in the test. Update the helper from Task 5 accordingly and note it there.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./plugins/oauth2provider/ -run TestMetadata_ -v`
Expected: FAIL, the two new paths 404.

- [ ] **Step 3: Write the metadata types and builder**

Create `plugins/oauth2provider/metadata.go`:

```go
package oauth2provider

import (
	"net/http"

	"github.com/xraph/forge"
)

// AuthServerMetadata is the RFC 8414 authorization server metadata document.
// It carries every field the OIDC discovery document does, plus the RFC 7591
// registration endpoint, and one builder produces both so they cannot drift.
type AuthServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	DeviceAuthorizationEndpoint       string   `json:"device_authorization_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	JWKSURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// ProtectedResourceMetadata is the RFC 9728 protected resource metadata
// document. An MCP client fetches this first and follows
// authorization_servers to the RFC 8414 document.
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

// ProtectedResourceRequest addresses one configured resource by the path
// suffix it is served under (RFC 9728 section 3.1).
type ProtectedResourceRequest struct {
	Path string `path:"resourcePath"`
}

func (p *Plugin) buildAuthServerMetadata() *AuthServerMetadata {
	issuer := p.issuerURL()

	m := &AuthServerMetadata{
		Issuer:                      issuer,
		AuthorizationEndpoint:       issuer + "/v1/oauth/authorize",
		TokenEndpoint:               issuer + "/v1/oauth/token",
		UserinfoEndpoint:            issuer + "/v1/oauth/userinfo",
		RevocationEndpoint:          issuer + "/v1/oauth/revoke",
		DeviceAuthorizationEndpoint: issuer + "/v1/oauth/device/authorize",
		JWKSURI:                     issuer + "/.well-known/jwks.json",
		ResponseTypesSupported:      []string{"code"},
		GrantTypesSupported: []string{
			"authorization_code",
			"client_credentials",
			deviceCodeGrantType,
		},
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256", "ES256"},
		ScopesSupported:                  []string{"openid", "profile", "email", "phone"},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post", "client_secret_basic", "none",
		},
		CodeChallengeMethodsSupported: []string{"S256", "plain"},
	}

	// Only advertise registration when it will actually answer. Pointing a
	// client at an endpoint that 404s produces a worse failure than not
	// mentioning it at all.
	if p.config.DynamicRegistration {
		m.RegistrationEndpoint = issuer + "/v1/oauth/register"
	}
	return m
}

func (p *Plugin) handleAuthServerMetadata(_ forge.Context, _ *DiscoveryRequest) (*AuthServerMetadata, error) {
	return p.buildAuthServerMetadata(), nil
}

func (p *Plugin) handleProtectedResourceMetadata(_ forge.Context, _ *DiscoveryRequest) (*ProtectedResourceMetadata, error) {
	issuer := p.issuerURL()
	return &ProtectedResourceMetadata{
		Resource:               issuer,
		AuthorizationServers:   []string{issuer},
		ScopesSupported:        []string{"openid", "profile", "email", "phone"},
		BearerMethodsSupported: []string{"header"},
	}, nil
}

func (p *Plugin) handleScopedProtectedResourceMetadata(_ forge.Context, req *ProtectedResourceRequest) (*ProtectedResourceMetadata, error) {
	res, ok := p.config.ProtectedResources[req.Path]
	if !ok {
		return nil, forge.NotFound("no protected resource is declared at that path")
	}
	return &ProtectedResourceMetadata{
		Resource:               res.Resource,
		AuthorizationServers:   []string{p.issuerURL()},
		ScopesSupported:        res.ScopesSupported,
		BearerMethodsSupported: []string{"header"},
	}, nil
}
```

- [ ] **Step 4: Have the OIDC document reuse the builder**

In `plugin.go`, replace the body of `handleDiscovery` so it derives from the same source:

```go
func (p *Plugin) handleDiscovery(_ forge.Context, _ *DiscoveryRequest) (*DiscoveryResponse, error) {
	m := p.buildAuthServerMetadata()
	return &DiscoveryResponse{
		Issuer:                            m.Issuer,
		AuthorizationEndpoint:             m.AuthorizationEndpoint,
		TokenEndpoint:                     m.TokenEndpoint,
		UserinfoEndpoint:                  m.UserinfoEndpoint,
		RevocationEndpoint:                m.RevocationEndpoint,
		DeviceAuthorizationEndpoint:       m.DeviceAuthorizationEndpoint,
		RegistrationEndpoint:              m.RegistrationEndpoint,
		JWKSURI:                           m.JWKSURI,
		ResponseTypesSupported:            m.ResponseTypesSupported,
		GrantTypesSupported:               m.GrantTypesSupported,
		SubjectTypesSupported:             m.SubjectTypesSupported,
		IDTokenSigningAlgValuesSupported:  m.IDTokenSigningAlgValuesSupported,
		ScopesSupported:                   m.ScopesSupported,
		TokenEndpointAuthMethodsSupported: m.TokenEndpointAuthMethodsSupported,
		CodeChallengeMethodsSupported:     m.CodeChallengeMethodsSupported,
	}, nil
}
```

Add the field to `DiscoveryResponse`:

```go
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
```

- [ ] **Step 5: Mount the three new documents**

Extend `registerWellKnown` from Task 1 so all four are registered together:

```go
func (p *Plugin) registerWellKnown(router forge.Router) error {
	if err := router.GET("/.well-known/openid-configuration", p.handleDiscovery,
		forge.WithSummary("OpenID Connect Discovery"),
		forge.WithOperationID("oidcDiscovery"),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	if err := router.GET("/.well-known/oauth-authorization-server", p.handleAuthServerMetadata,
		forge.WithSummary("OAuth2 Authorization Server Metadata"),
		forge.WithDescription("RFC 8414 authorization server metadata."),
		forge.WithOperationID("oauth2AuthServerMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", AuthServerMetadata{}),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	if err := router.GET("/.well-known/oauth-protected-resource", p.handleProtectedResourceMetadata,
		forge.WithSummary("OAuth2 Protected Resource Metadata"),
		forge.WithDescription("RFC 9728 protected resource metadata."),
		forge.WithOperationID("oauth2ProtectedResourceMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", ProtectedResourceMetadata{}),
		forge.WithTags("OAuth2"),
	); err != nil {
		return err
	}

	return router.GET("/.well-known/oauth-protected-resource/:resourcePath", p.handleScopedProtectedResourceMetadata,
		forge.WithSummary("OAuth2 Protected Resource Metadata (scoped)"),
		forge.WithOperationID("oauth2ScopedProtectedResourceMetadata"),
		forge.WithResponseSchema(http.StatusOK, "Metadata", ProtectedResourceMetadata{}),
		forge.WithTags("OAuth2"),
	)
}
```

The prefixed mirror in `RegisterRoutes` keeps registering only `/.well-known/openid-configuration`, with the `oidcDiscoveryPrefixed` operation ID. Do not mirror the 8414 and 9728 documents: a prefixed copy of either is not discoverable and would only add duplicate OpenAPI entries.

- [ ] **Step 6: Run the tests and confirm they pass**

Run: `go test ./plugins/oauth2provider/ -run TestMetadata_ -v`
Expected: PASS, all six cases.

- [ ] **Step 7: Run everything**

Run: `go test ./plugins/oauth2provider/... ./plugin/... ./extension/... -count=1 && make lint`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add plugins/oauth2provider/
git commit -m "feat(oauth2): add RFC 8414 and RFC 9728 metadata documents

An MCP client fetches the protected resource document, follows
authorization_servers to the RFC 8414 document, and reads
registration_endpoint from it. Only the OIDC document existed before, so
that chain broke on its second hop.

One builder feeds the 8414 and OIDC documents so they cannot drift, and
registration_endpoint appears only when registration is actually enabled."
```

---

### Task 8: The 401 resource metadata challenge

**Files:**
- Create: `middleware/resource_metadata.go`
- Create: `middleware/resource_metadata_test.go`
- Modify: `option.go` (new engine option, near `WithRateLimiter` at line 235)
- Modify: `engine.go` (config field and accessor, near line 822)
- Modify: `api/api.go:89` (wire the middleware)

**Interfaces:**
- Consumes: nothing from earlier tasks. The URL is a plain string an operator configures.
- Produces: `middleware.ResourceMetadataChallenge(metadataURL string) forge.Middleware`; `authsome.WithProtectedResourceMetadataURL(url string) Option`; `(*Engine).ProtectedResourceMetadataURL() string`.

- [ ] **Step 1: Write the failing middleware test**

Create `middleware/resource_metadata_test.go`:

```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

const metaURL = "https://auth.example.com/.well-known/oauth-protected-resource"

func TestResourceMetadataChallenge_SetsHeaderOn401(t *testing.T) {
	h := middleware.ResourceMetadataChallenge(metaURL)(func(ctx forge.Context) error {
		ctx.Response().WriteHeader(http.StatusUnauthorized)
		return nil
	})

	rec := httptest.NewRecorder()
	_ = h(newTestContext(httptest.NewRequest(http.MethodGet, "/v1/me", nil), rec))

	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), `resource_metadata="`+metaURL+`"`)
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer")
}

func TestResourceMetadataChallenge_LeavesSuccessAlone(t *testing.T) {
	h := middleware.ResourceMetadataChallenge(metaURL)(func(ctx forge.Context) error {
		ctx.Response().WriteHeader(http.StatusOK)
		return nil
	})

	rec := httptest.NewRecorder()
	_ = h(newTestContext(httptest.NewRequest(http.MethodGet, "/v1/me", nil), rec))
	assert.Empty(t, rec.Header().Get("WWW-Authenticate"))
}

// An empty URL means the operator has not configured discovery, so the
// middleware must be inert rather than emitting a broken header.
func TestResourceMetadataChallenge_InertWhenUnset(t *testing.T) {
	h := middleware.ResourceMetadataChallenge("")(func(ctx forge.Context) error {
		ctx.Response().WriteHeader(http.StatusUnauthorized)
		return nil
	})

	rec := httptest.NewRecorder()
	_ = h(newTestContext(httptest.NewRequest(http.MethodGet, "/v1/me", nil), rec))
	assert.Empty(t, rec.Header().Get("WWW-Authenticate"))
}

// A handler that already set its own challenge, like the RFC 7592 routes,
// keeps it. The header is not appended twice.
func TestResourceMetadataChallenge_DoesNotOverwrite(t *testing.T) {
	h := middleware.ResourceMetadataChallenge(metaURL)(func(ctx forge.Context) error {
		ctx.Response().Header().Set("WWW-Authenticate", `Bearer realm="registration"`)
		ctx.Response().WriteHeader(http.StatusUnauthorized)
		return nil
	})

	rec := httptest.NewRecorder()
	_ = h(newTestContext(httptest.NewRequest(http.MethodGet, "/v1/oauth/register/x", nil), rec))
	assert.Equal(t, `Bearer realm="registration"`, rec.Header().Get("WWW-Authenticate"))
}
```

`middleware` already has tests that stand a `forge.Context` up (see `middleware/ratelimit_test.go`). Reuse that construction and name the helper `newTestContext` if one does not already exist; if the package has its own helper, use it and drop this one.

- [ ] **Step 2: Run the tests and confirm they fail**

Run: `go test ./middleware/ -run TestResourceMetadataChallenge -v`
Expected: FAIL, `undefined: middleware.ResourceMetadataChallenge`.

- [ ] **Step 3: Write the middleware**

Create `middleware/resource_metadata.go`:

```go
package middleware

import (
	"net/http"

	"github.com/xraph/forge"
)

// ResourceMetadataChallenge adds the RFC 9728 section 5.1 resource_metadata
// parameter to the WWW-Authenticate header on a 401.
//
// It is how a client bootstraps discovery from a failed call instead of from
// a URL somebody handed it, which is the path the MCP spec uses. An empty
// metadataURL makes the middleware inert, so a deployment that has not
// configured discovery emits nothing rather than a broken hint.
//
// A handler that already set its own challenge keeps it. The RFC 7592 routes
// set a realm of their own, and clobbering it would lose information the
// client can use.
func ResourceMetadataChallenge(metadataURL string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		if metadataURL == "" {
			return next
		}
		return func(ctx forge.Context) error {
			err := next(ctx)

			w := ctx.Response()
			if w.Header().Get("WWW-Authenticate") != "" {
				return err
			}
			if statusOf(ctx, err) != http.StatusUnauthorized {
				return err
			}
			w.Header().Set("WWW-Authenticate",
				`Bearer resource_metadata="`+metadataURL+`"`)
			return err
		}
	}
}
```

`statusOf` has to read the status the response is going out with. Two sources: the recorded status on `ctx.Response()` if forge's writer exposes one, and the `StatusCode()` of an error implementing forge's responder interface. Read how another middleware in this package inspects a response status, and implement `statusOf` the same way. If the writer records nothing, wrap it in a small `statusCapturingWriter` inside this file and install it before calling `next`, which is the standard approach and keeps the fix local.

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `go test ./middleware/ -run TestResourceMetadataChallenge -v`
Expected: PASS, all four cases.

- [ ] **Step 5: Add the engine option**

In `option.go`, next to `WithRateLimiter`:

```go
// WithProtectedResourceMetadataURL sets the RFC 9728 metadata URL advertised
// in the WWW-Authenticate header on a 401. Leave it unset to emit no hint.
// It should be the origin-root path the oauth2provider plugin serves, for
// example https://auth.example.com/.well-known/oauth-protected-resource.
func WithProtectedResourceMetadataURL(url string) Option {
	return func(c *Config) {
		c.ProtectedResourceMetadataURL = url
	}
}
```

Add `ProtectedResourceMetadataURL string` to the engine config struct, and in `engine.go` next to `RateLimiter()`:

```go
// ProtectedResourceMetadataURL returns the configured RFC 9728 metadata URL.
func (e *Engine) ProtectedResourceMetadataURL() string {
	return e.config.ProtectedResourceMetadataURL
}
```

- [ ] **Step 6: Wire it into the API router**

In `api/api.go`, after the `PublishableKeyMiddleware` line at 89:

```go
	// RFC 9728 section 5.1: tell an unauthenticated caller where the
	// protected resource metadata lives, so an MCP client can bootstrap
	// discovery from a 401 rather than needing the URL up front. Inert
	// unless WithProtectedResourceMetadataURL was set.
	router.Use(middleware.ResourceMetadataChallenge(a.engine.ProtectedResourceMetadataURL()))
```

- [ ] **Step 7: Run the full suite**

Run: `go test ./... -count=1 && make lint`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add middleware/resource_metadata.go middleware/resource_metadata_test.go option.go engine.go api/api.go
git commit -m "feat(middleware): advertise protected resource metadata on 401

RFC 9728 section 5.1. An MCP client that hits a 401 reads
resource_metadata off the WWW-Authenticate header and starts discovery
from there, which is how it finds the server without being handed a URL.

Inert unless WithProtectedResourceMetadataURL is set, and it never
overwrites a challenge a handler already wrote."
```

---

## Verification

After Task 8, confirm the whole chain by hand against a running server with `DynamicRegistration` on:

```bash
curl -s http://localhost:8080/.well-known/oauth-protected-resource | jq
curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq .registration_endpoint
curl -s -X POST http://localhost:8080/v1/oauth/register \
  -H 'Content-Type: application/json' \
  -d '{"client_name":"probe","redirect_uris":["http://127.0.0.1:9000/cb"],"token_endpoint_auth_method":"none"}' | jq
```

The third call returns a `client_id`, a `registration_access_token` and a `registration_client_uri`. Fetch that URI with the token as a bearer header and you should get the same registration back.
