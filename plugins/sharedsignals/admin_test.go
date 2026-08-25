package sharedsignals

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

// mustParseStreamID recovers the typed id.SSFStreamID from a StreamView's
// string ID. StreamView.ID is a string (it crosses the wire as JSON), but
// the store keys on the typed ID, so tests that go from a CreateStream
// response back into the store need to parse it first.
func mustParseStreamID(t *testing.T, s string) id.SSFStreamID {
	t.Helper()
	parsed, err := id.ParseWithPrefix(s, id.PrefixSSFStream)
	require.NoError(t, err)
	return parsed
}

func newAdminFixture(t *testing.T) (*Plugin, id.AppID, id.EnvironmentID) {
	t.Helper()
	p := New(Config{Audience: "https://authsome.example/ssf"})
	p.store = NewMemoryStore()
	return p, id.NewAppID(), id.NewEnvironmentID()
}

func TestCreateStream_ReturnsSecretsOnceAndStoresHashes(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	got, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta-prod", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/oauth2/v1/keys",
	})
	require.NoError(t, err)

	assert.NotEmpty(t, got.PushURLPath, "the caller needs the plaintext once")
	assert.NotEmpty(t, got.PushToken)

	stored, err := p.store.GetInboundStream(ctx, mustParseStreamID(t, got.Stream.ID))
	require.NoError(t, err)
	assert.Equal(t, HashSecret(got.PushURLPath), stored.PushPathHash)
	assert.Equal(t, HashSecret(got.PushToken), stored.PushTokenHash)
	assert.NotContains(t, stored.PushPathHash, got.PushURLPath,
		"the plaintext must never be persisted")
}

func TestCreateStream_AppliesDefaults(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	got, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/keys",
	})
	require.NoError(t, err)

	stored, err := p.store.GetInboundStream(ctx, mustParseStreamID(t, got.Stream.ID))
	require.NoError(t, err)
	assert.Equal(t, StatusEnabled, stored.Status)
	assert.Equal(t, EnforcementEnforce, stored.EnforcementMode)
	assert.Equal(t, 100, stored.MaxActionsPerHour)
	assert.Equal(t, "https://authsome.example/ssf", stored.Audience)
	assert.Equal(t, []string{caep.FormatIssSub}, stored.AllowedSubjectFormats,
		"iss_sub only unless the operator opts into more")
}

func TestCreateStream_RejectsBadInput(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	_, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Issuer: "", JWKSURI: "https://org.okta.com/keys"})
	require.Error(t, err)

	_, err = p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Issuer: "https://org.okta.com", JWKSURI: ""})
	require.Error(t, err)
}

// An operator pasting a metadata-service URL is still an SSRF, so the same
// check the fetcher uses runs at registration too.
func TestCreateStream_RejectsUnsafeJWKSURI(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	for _, uri := range []string{
		"http://org.okta.com/keys",
		"https://169.254.169.254/latest/meta-data/",
		"https://127.0.0.1/keys",
	} {
		_, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
			Name: "x", Issuer: "https://org.okta.com", JWKSURI: uri,
		})
		require.Error(t, err, "jwks_uri %q must be refused", uri)
	}
}

// Allowing email as a subject format without naming a verified domain would
// let the transmitter name anyone at all.
func TestCreateStream_EmailFormatRequiresVerifiedDomains(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	_, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "x", Issuer: "https://org.okta.com",
		JWKSURI:               "https://org.okta.com/keys",
		AllowedSubjectFormats: []string{caep.FormatEmail},
	})
	require.Error(t, err)

	_, err = p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "x", Issuer: "https://org.okta.com",
		JWKSURI:               "https://org.okta.com/keys",
		AllowedSubjectFormats: []string{caep.FormatEmail},
		VerifiedDomains:       []string{"corp.com"},
	})
	require.NoError(t, err)
}

// A listing must never hand back a hash that could be checked offline
// against a guessed secret.
func TestStreamView_OmitsSecrets(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	created, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/keys",
	})
	require.NoError(t, err)

	stored, err := p.store.GetInboundStream(ctx, mustParseStreamID(t, created.Stream.ID))
	require.NoError(t, err)

	view := toStreamView(stored)
	rec := httptest.NewRecorder()
	require.NoError(t, writeJSONForTest(rec, view))

	body := rec.Body.String()
	assert.NotContains(t, body, stored.PushPathHash)
	assert.NotContains(t, body, stored.PushTokenHash)
	assert.Contains(t, body, "https://org.okta.com")
}

// ──────────────────────────────────────────────────
// IDOR: a stream must only ever be reachable by the app that owns it.
// ──────────────────────────────────────────────────

func strPtr(s string) *string { return &s }

// routerForApp mounts only the two handlers under test on a bare router,
// with a middleware that stamps the given appID onto the request context --
// the same context slot the publishable-key middleware fills in production
// (see requestScope). This isolates the handlers' own IDOR guard from the
// unrelated session-auth machinery that registerAdminRoutes' route group
// wires in front of them.
func routerForApp(p *Plugin, appID id.AppID) http.Handler {
	router := forge.NewRouter()
	router.Use(func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			ctx.WithContext(middleware.WithAppID(ctx.Context(), appID))
			return next(ctx)
		}
	})
	_ = router.PATCH("/streams/:id", p.handleUpdateStream)
	_ = router.DELETE("/streams/:id", p.handleDeleteStream)
	return router.Handler()
}

// TestHandleUpdateStream_CrossAppIsIndistinguishableFromMissing proves the
// IDOR fix: a caller scoped to app B cannot update app A's stream, and the
// failure they see is byte-for-byte the same as updating a stream id that
// was never created at all. A caller able to tell those two cases apart
// (e.g. via a 403 instead of a 404) could enumerate other tenants' stream
// ids one probe at a time.
func TestHandleUpdateStream_CrossAppIsIndistinguishableFromMissing(t *testing.T) {
	ctx := context.Background()
	p, appA, envA := newAdminFixture(t)
	appB := id.NewAppID()

	created, err := p.CreateStream(ctx, appA, envA, CreateStreamRequest{
		Name: "okta-a", Issuer: "https://a.okta.com", JWKSURI: "https://a.okta.com/keys",
	})
	require.NoError(t, err)

	body, err := json.Marshal(UpdateStreamRequest{Name: strPtr("renamed-by-app-b")})
	require.NoError(t, err)

	crossReq := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		"/streams/"+created.Stream.ID, bytes.NewReader(body))
	crossRec := httptest.NewRecorder()
	routerForApp(p, appB).ServeHTTP(crossRec, crossReq)

	missingReq := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		"/streams/"+id.NewSSFStreamID().String(), bytes.NewReader(body))
	missingRec := httptest.NewRecorder()
	routerForApp(p, appB).ServeHTTP(missingRec, missingReq)

	require.Equal(t, http.StatusNotFound, crossRec.Code,
		"app B must not be able to reach app A's stream")
	assert.Equal(t, missingRec.Code, crossRec.Code)
	assert.Equal(t, missingRec.Body.String(), crossRec.Body.String(),
		"a cross-tenant stream must look identical to one that does not exist")

	// The stream itself must be untouched by the cross-app attempt.
	stillA, err := p.store.GetInboundStream(ctx, mustParseStreamID(t, created.Stream.ID))
	require.NoError(t, err)
	assert.Equal(t, "okta-a", stillA.Name)

	// The owning app, in contrast, can update its own stream.
	ownReq := httptest.NewRequestWithContext(context.Background(), http.MethodPatch,
		"/streams/"+created.Stream.ID, bytes.NewReader(body))
	ownRec := httptest.NewRecorder()
	routerForApp(p, appA).ServeHTTP(ownRec, ownReq)
	assert.Equal(t, http.StatusOK, ownRec.Code)
}

// TestHandleDeleteStream_CrossAppIsIndistinguishableFromMissing is the same
// proof for delete: deleting is the more dangerous half of this IDOR, since
// it destroys the thing that authorises session revocation for app A.
func TestHandleDeleteStream_CrossAppIsIndistinguishableFromMissing(t *testing.T) {
	ctx := context.Background()
	p, appA, envA := newAdminFixture(t)
	appB := id.NewAppID()

	created, err := p.CreateStream(ctx, appA, envA, CreateStreamRequest{
		Name: "okta-a", Issuer: "https://a.okta.com", JWKSURI: "https://a.okta.com/keys",
	})
	require.NoError(t, err)

	crossReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete,
		"/streams/"+created.Stream.ID, nil)
	crossRec := httptest.NewRecorder()
	routerForApp(p, appB).ServeHTTP(crossRec, crossReq)

	missingReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete,
		"/streams/"+id.NewSSFStreamID().String(), nil)
	missingRec := httptest.NewRecorder()
	routerForApp(p, appB).ServeHTTP(missingRec, missingReq)

	require.Equal(t, http.StatusNotFound, crossRec.Code,
		"app B must not be able to delete app A's stream")
	assert.Equal(t, missingRec.Code, crossRec.Code)
	assert.Equal(t, missingRec.Body.String(), crossRec.Body.String(),
		"a cross-tenant stream must look identical to one that does not exist")

	// The stream must survive the cross-app delete attempt.
	_, err = p.store.GetInboundStream(ctx, mustParseStreamID(t, created.Stream.ID))
	require.NoError(t, err, "app A's stream must not be deleted by app B")
}

// ──────────────────────────────────────────────────
// Auth enforcement: registerAdminRoutes must actually gate the group, not
// just document it. forge.WithGroupAuth("session") on its own writes
// OpenAPI metadata and enforces nothing at request time -- the group needs
// forge.WithGroupMiddleware(plugin.SessionGuard(engine)...) too, and the
// three mutating routes additionally need a permission check so an ordinary
// signed-in user can't mint themselves a session-revocation capability.
// ──────────────────────────────────────────────────

// fakeAuthEngine wraps stubEngine (see stub_engine_test.go) so these tests
// can exercise the real plugin.SessionGuard / plugin.PermissionGuard code
// paths -- a real auth.Registry actually runs its provider lookup and a
// real middleware.RequirePermission actually calls HasPermission -- rather
// than asserting only that a slice of middleware is non-empty.
type fakeAuthEngine struct {
	stubEngine
	registry   auth.Registry
	defaultApp string
	// permissions maps "userID|action|resource" to the scripted RBAC
	// decision; a missing key means denied, matching a real checker with no
	// grant on file.
	permissions map[string]bool
}

func (e *fakeAuthEngine) AuthRegistry() auth.Registry { return e.registry }
func (e *fakeAuthEngine) DefaultAppID() string        { return e.defaultApp }

func (e *fakeAuthEngine) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	return e.permissions[userID.String()+"|"+action+"|"+resource], nil
}

var _ plugin.PermissionChecker = (*fakeAuthEngine)(nil)

// alwaysSessionProvider authenticates every request unconditionally. It
// exists so a test can get past SessionGuard's real registry-driven check
// without standing up a full session/cookie issuing pipeline, isolating
// what these tests are actually proving: whether the group's middleware is
// wired at all (TestRegisterAdminRoutes_RejectsUnauthenticated uses a
// registry with NO provider registered instead, precisely to exercise the
// rejection path), and whether PermissionGuard gates the write routes once
// a session exists.
type alwaysSessionProvider struct{}

func (alwaysSessionProvider) Name() string                  { return "session" }
func (alwaysSessionProvider) Type() auth.SecuritySchemeType { return auth.SecurityTypeHTTP }

func (alwaysSessionProvider) Authenticate(context.Context, *http.Request) (*auth.AuthContext, error) {
	return &auth.AuthContext{Subject: "test-subject", ProviderName: "session"}, nil
}

func (alwaysSessionProvider) OpenAPIScheme() auth.SecurityScheme { return auth.SecurityScheme{} }

func (alwaysSessionProvider) Middleware() forge.Middleware {
	return func(next forge.Handler) forge.Handler { return next }
}

// TestRegisterAdminRoutes_RejectsUnauthenticated reproduces, and proves the
// fix for, exactly what the review found live: POST /v1/ssf/admin/streams
// with no Authorization header used to return 201 with a working
// push_url_path and push_token, because WithGroupAuth("session") alone
// enforces nothing. Every admin route must now reject a request that
// carries no session at all.
func TestRegisterAdminRoutes_RejectsUnauthenticated(t *testing.T) {
	ctx := context.Background()
	appID := id.NewAppID()

	p := New(Config{Audience: "https://authsome.example/ssf"})
	p.store = NewMemoryStore()
	p.engine = &fakeAuthEngine{
		// No provider registered: exactly the "nobody is signed in" case.
		registry:   auth.NewRegistry(nil, forge.NewNoopLogger()),
		defaultApp: appID.String(),
	}

	// Structural backstop: if SessionGuard ever regresses to returning nil
	// for an engine that does expose an AuthRegistry, the live assertions
	// below would still catch it (nil middleware wired in means everything
	// passes through), but pin the precondition directly too.
	require.NotEmpty(t, plugin.SessionGuard(p.engine), "SessionGuard must produce middleware for a real registry")

	router := forge.NewRouter()
	require.NoError(t, p.registerAdminRoutes(router))
	h := router.Handler()

	createBody := []byte(`{"issuer":"https://org.okta.com","jwks_uri":"https://org.okta.com/keys"}`)
	someStreamID := id.NewSSFStreamID().String()

	cases := []struct {
		name, method, path string
		body               []byte
	}{
		{"create", http.MethodPost, "/v1/ssf/admin/streams", createBody},
		{"list", http.MethodGet, "/v1/ssf/admin/streams", nil},
		{"update", http.MethodPatch, "/v1/ssf/admin/streams/" + someStreamID, []byte(`{"name":"x"}`)},
		{"delete", http.MethodDelete, "/v1/ssf/admin/streams/" + someStreamID, nil},
	}
	for _, c := range cases {
		req := httptest.NewRequestWithContext(ctx, c.method, c.path, bytes.NewReader(c.body))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code,
			"%s (%s %s) must be rejected without a session", c.name, c.method, c.path)
	}

	// The rejected create must not have registered a stream.
	streams, err := p.store.ListInboundStreams(ctx, appID)
	require.NoError(t, err)
	assert.Empty(t, streams, "an unauthenticated create must not have gone through")
}

// TestRegisterAdminRoutes_MutatingRoutesRequirePermission proves the second
// half of the fix: SessionGuard alone would let any signed-in end user of
// the app register, update or delete a stream -- and a stream is what
// authorises session revocation, so that's a privilege escalation. Create,
// update and delete must additionally require the "manage"/permStreamResource
// permission; list must not, since it carries no writeGuard.
func TestRegisterAdminRoutes_MutatingRoutesRequirePermission(t *testing.T) {
	ctx := context.Background()
	registry := auth.NewRegistry(nil, forge.NewNoopLogger())
	require.NoError(t, registry.Register(alwaysSessionProvider{}))

	allowedUser := id.NewUserID()
	deniedUser := id.NewUserID()
	appID := id.NewAppID()

	engine := &fakeAuthEngine{
		registry:   registry,
		defaultApp: appID.String(),
		permissions: map[string]bool{
			allowedUser.String() + "|manage|" + permStreamResource: true,
		},
	}

	// Structural backstop, same reasoning as the unauthenticated test above.
	require.NotEmpty(t, plugin.SessionGuard(engine))
	require.NotEmpty(t, plugin.PermissionGuard(engine, "manage", permStreamResource),
		"PermissionGuard must produce middleware for an engine implementing PermissionChecker")

	p := New(Config{Audience: "https://authsome.example/ssf"})
	p.store = NewMemoryStore()
	p.engine = engine

	router := forge.NewRouter()
	// Stand-in for the engine-wide, non-blocking AuthMiddleware that in
	// production runs ahead of every route group and bridges a resolved
	// session onto middleware.WithUserID (see
	// authprovider.SessionProvider.Middleware / BridgeToContext).
	// registerAdminRoutes' own SessionGuard re-validates that some session
	// exists via the registry above, but the generic multi-provider
	// registry path it uses does not itself populate UserIDFrom -- that
	// bridging is a separate, globally-applied concern in production. This
	// reads a per-request test header instead, matching the same
	// direct-context-injection pattern plugins/organization/authz_test.go's
	// orgReq already uses in this codebase to isolate authorization logic
	// from session/cookie plumbing.
	router.Use(func(next forge.Handler) forge.Handler {
		return func(c forge.Context) error {
			if raw := c.Request().Header.Get("X-Test-User-Id"); raw != "" {
				if uid, uerr := id.ParseUserID(raw); uerr == nil {
					c.WithContext(middleware.WithUserID(c.Context(), uid))
				}
			}
			return next(c)
		}
	})
	require.NoError(t, p.registerAdminRoutes(router))
	h := router.Handler()

	doAs := func(userID id.UserID, method, path string, body []byte) int {
		req := httptest.NewRequestWithContext(ctx, method, path, bytes.NewReader(body))
		req.Header.Set("X-Test-User-Id", userID.String())
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	createBody := []byte(`{"issuer":"https://org.okta.com","jwks_uri":"https://org.okta.com/keys"}`)

	assert.Equal(t, http.StatusForbidden,
		doAs(deniedUser, http.MethodPost, "/v1/ssf/admin/streams", createBody),
		"a session without the permission must not be able to create a stream")
	assert.Equal(t, http.StatusOK,
		doAs(deniedUser, http.MethodGet, "/v1/ssf/admin/streams", nil),
		"list carries no permission gate, only the session requirement")

	require.Equal(t, http.StatusCreated,
		doAs(allowedUser, http.MethodPost, "/v1/ssf/admin/streams", createBody))

	streams, err := p.store.ListInboundStreams(ctx, appID)
	require.NoError(t, err)
	require.Len(t, streams, 1, "only the allowed user's create should have gone through")
	streamPath := "/v1/ssf/admin/streams/" + streams[0].ID.String()

	assert.Equal(t, http.StatusForbidden,
		doAs(deniedUser, http.MethodPatch, streamPath, []byte(`{"name":"renamed"}`)),
		"a session without the permission must not be able to update a stream")
	assert.Equal(t, http.StatusForbidden,
		doAs(deniedUser, http.MethodDelete, streamPath, nil),
		"a session without the permission must not be able to delete a stream")

	assert.Equal(t, http.StatusOK,
		doAs(allowedUser, http.MethodPatch, streamPath, []byte(`{"name":"renamed"}`)),
		"the permitted user must still be able to update its own stream")
}

// ──────────────────────────────────────────────────
// Update-state validation: a PATCH must be validated against its fully
// merged RESULTING state, not just the fields it happened to touch.
// ──────────────────────────────────────────────────

// TestUpdateStream_CannotBypassVerifiedDomainsGateViaPartialPatch proves the
// fix: a PATCH carrying only verified_domains: [] must not be allowed to
// leave a stream with the email format active and zero verified domains --
// the reviewer's live proof of exactly this bypass.
func TestUpdateStream_CannotBypassVerifiedDomainsGateViaPartialPatch(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	created, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI:               "https://org.okta.com/keys",
		AllowedSubjectFormats: []string{caep.FormatEmail},
		VerifiedDomains:       []string{"corp.com"},
	})
	require.NoError(t, err)
	streamID := mustParseStreamID(t, created.Stream.ID)

	_, err = p.UpdateStream(ctx, streamID, UpdateStreamRequest{
		VerifiedDomains: []string{},
	})
	require.Error(t, err,
		"clearing verified_domains while the email format is still active must be refused")

	stored, err := p.store.GetInboundStream(ctx, streamID)
	require.NoError(t, err)
	assert.Equal(t, []string{"corp.com"}, stored.VerifiedDomains,
		"a rejected patch must not have partially applied")
}

// TestUpdateStream_RejectsUnsafeJWKSURI proves an update cannot introduce a
// jwks_uri the create path would have refused.
func TestUpdateStream_RejectsUnsafeJWKSURI(t *testing.T) {
	ctx := context.Background()
	p, appID, envID := newAdminFixture(t)

	created, err := p.CreateStream(ctx, appID, envID, CreateStreamRequest{
		Name: "okta", Issuer: "https://org.okta.com",
		JWKSURI: "https://org.okta.com/keys",
	})
	require.NoError(t, err)
	streamID := mustParseStreamID(t, created.Stream.ID)

	unsafe := "https://169.254.169.254/latest/meta-data/"
	_, err = p.UpdateStream(ctx, streamID, UpdateStreamRequest{JWKSURI: &unsafe})
	require.Error(t, err, "jwks_uri %q must be refused on update just as it would be on create", unsafe)

	stored, err := p.store.GetInboundStream(ctx, streamID)
	require.NoError(t, err)
	assert.Equal(t, "https://org.okta.com/keys", stored.JWKSURI,
		"a rejected patch must not have partially applied")
}
