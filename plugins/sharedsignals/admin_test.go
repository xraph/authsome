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

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
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
