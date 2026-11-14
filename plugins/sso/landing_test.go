package sso

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// A browser landing (OIDC redirect or SAML ACS) always ends in a redirect to
// the return URL: a code on success, the MFA ticket when a second factor is
// owed, and an sso_error code when the app refused the sign-up.

const landingFrontend = "https://app.wakflo.test"

// assertingProvider gets past every protocol check and asserts u.
type assertingProvider struct {
	stubProvider
	u *User
}

func (a assertingProvider) HandleCallback(context.Context, map[string]string) (*User, error) {
	cp := *a.u
	return &cp, nil
}

// gateRecorder refuses or records the sign-ups the engine gates.
type gateRecorder struct {
	refuse  bool
	before  []*account.SignUpRequest
	created int
}

func (g *gateRecorder) Name() string { return "gate-recorder" }
func (g *gateRecorder) OnBeforeSignUp(_ context.Context, req *account.SignUpRequest) error {
	g.before = append(g.before, req)
	if g.refuse {
		return errors.New("setup: this workspace is not accepting members")
	}
	return nil
}

func (g *gateRecorder) OnAfterSignUp(context.Context, *user.User, *session.Session) error {
	g.created++
	return nil
}

type landingFixture struct {
	p     *Plugin
	eng   *authsome.Engine
	conn  *Connection
	appID id.AppID
}

func newLandingFixture(t *testing.T, opts ...authsome.Option) *landingFixture {
	t.Helper()
	p := New(Config{PublicBaseURL: "https://api.example.com", AllowedReturnOrigins: []string{landingFrontend}})
	p.SetSSOStore(NewMemoryStore())
	eng := secutil.NewTestEngine(t, append(opts, authsome.WithPlugin(p))...)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	conn := &Connection{ID: id.NewSSOConnectionID(), AppID: appID, Provider: "corp", Protocol: "saml", Domain: "corp.example", DisplayName: "Corp Okta", Active: true, Enforced: true}
	require.NoError(t, p.ssoStore.CreateConnection(context.Background(), conn))
	return &landingFixture{p: p, eng: eng, conn: conn, appID: appID}
}

// land runs authenticateUser for an asserted identity and finishes the
// landing the way the OIDC redirect and the ACS do, returning the redirect.
func (f *landingFixture) land(t *testing.T, returnURL string, asserted *User) *url.URL {
	t.Helper()
	router := forge.NewRouter()
	require.NoError(t, router.GET("/land", func(ctx forge.Context) error {
		result, err := f.p.authenticateUser(ctx, f.appID, assertingProvider{u: asserted}, f.conn, nil)
		return f.p.finishLanding(ctx, returnURL, "test", f.appID, result, err)
	}))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/land", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	return loc
}

func (f *landingFixture) setClientConfig(t *testing.T, cfg appclientconfig.Config) {
	t.Helper()
	cfg.ID = id.NewAppClientConfigID()
	cfg.AppID = f.appID
	require.NoError(t, f.eng.Store().SetAppClientConfig(context.Background(), &cfg))
}

func (f *landingFixture) userCount(t *testing.T) int {
	t.Helper()
	list, err := f.eng.Store().ListUsers(context.Background(), &user.Query{AppID: f.appID})
	require.NoError(t, err)
	return len(list.Users)
}

func corpUser(sub, email string) *User {
	return &User{ProviderUserID: sub, Email: email, FirstName: "Ada"}
}

func TestLanding_MFARedirectsWithTheTicket(t *testing.T) {
	f := newLandingFixture(t)
	tru := true
	f.setClientConfig(t, appclientconfig.Config{MFARequired: &tru})

	loc := f.land(t, landingFrontend+"/sso/callback?ws=acme", corpUser("s-1", "ada@corp.example"))
	assert.Equal(t, "app.wakflo.test", loc.Host)
	assert.Equal(t, "/sso/callback", loc.Path)
	q := loc.Query()
	assert.Equal(t, "acme", q.Get("ws"))
	assert.Empty(t, q.Get("code"), "no session is handed over before the second factor")
	assert.Empty(t, q.Get("sso_error"))
	assert.True(t, q.Has("mfa_methods"))
	ticket := q.Get("mfa_ticket")
	require.NotEmpty(t, ticket)

	payload, err := f.eng.LoadMFATicket(context.Background(), ticket)
	require.NoError(t, err)
	assert.Equal(t, "sso:stub", payload.AuthMethod, "the ticket keeps the auth method")
}

func TestLanding_MFARedirectHonoursTheAllowlist(t *testing.T) {
	f := newLandingFixture(t)
	tru := true
	f.setClientConfig(t, appclientconfig.Config{MFARequired: &tru})

	// A return URL off the allowlist falls back to the default landing.
	loc := f.land(t, "https://evil.example/steal", corpUser("s-2", "bo@corp.example"))
	assert.Equal(t, "app.wakflo.test", loc.Host)
	assert.Equal(t, "/sso/callback", loc.Path)
	assert.NotEmpty(t, loc.Query().Get("mfa_ticket"))
}

func TestLanding_SignUpDisabledRedirectsAndCreatesNoUser(t *testing.T) {
	f := newLandingFixture(t)
	off := false
	f.setClientConfig(t, appclientconfig.Config{SignupEnabled: &off})

	loc := f.land(t, landingFrontend+"/sso/callback", corpUser("s-3", "cy@corp.example"))
	assert.Equal(t, authsome.SignUpRefusedDisabled, loc.Query().Get("sso_error"))
	assert.Empty(t, loc.Query().Get("code"))
	assert.Equal(t, 0, f.userCount(t))
}

func TestLanding_HookRefusalRedirectsAndExistingUsersPass(t *testing.T) {
	g := &gateRecorder{}
	f := newLandingFixture(t, authsome.WithPlugin(g))

	// A first sign-in is a sign-up: the hooks see it, with the auth method.
	loc := f.land(t, landingFrontend+"/sso/callback", corpUser("s-4", "di@corp.example"))
	require.NotEmpty(t, loc.Query().Get("code"), loc.String())
	require.Len(t, g.before, 1)
	assert.Equal(t, "di@corp.example", g.before[0].Email)
	assert.Equal(t, "sso:stub", g.before[0].AuthMethod)
	assert.Equal(t, 1, g.created, "after-sign-up hooks fire")

	// The gate now refuses everyone. The account that exists still signs in.
	g.refuse = true
	loc = f.land(t, landingFrontend+"/sso/callback", corpUser("s-4", "di@corp.example"))
	assert.NotEmpty(t, loc.Query().Get("code"), "an existing account is never gated: %s", loc)
	assert.Len(t, g.before, 1)

	// A new one is refused, and nothing is written.
	loc = f.land(t, landingFrontend+"/sso/callback", corpUser("s-5", "ed@corp.example"))
	assert.Equal(t, authsome.SignUpRefusedHook, loc.Query().Get("sso_error"))
	assert.Equal(t, 1, f.userCount(t))
}

func TestDiscover(t *testing.T) {
	f := newLandingFixture(t)
	ctx := context.Background()
	inactive := &Connection{ID: id.NewSSOConnectionID(), AppID: f.appID, Provider: "old", Protocol: "oidc", Domain: "old.example", Active: false}
	require.NoError(t, f.p.ssoStore.CreateConnection(ctx, inactive))

	d, err := f.p.DiscoverByEmail(ctx, f.appID, "  Ada@Corp.Example ")
	require.NoError(t, err)
	assert.Equal(t, Discovery{
		ConnectionID: f.conn.ID.String(), Provider: "corp", Name: "Corp Okta",
		Protocol: "saml", Domain: "corp.example", Enforced: true,
	}, *d)
	for _, email := range []string{"bo@nowhere.example", "cy@old.example", "not-an-email", "dangling@"} {
		_, err = f.p.DiscoverByEmail(ctx, f.appID, email)
		assert.ErrorIs(t, err, ErrNoConnectionForDomain, email)
	}
	_, err = f.p.DiscoverByEmail(ctx, id.NewAppID(), "ada@corp.example")
	assert.ErrorIs(t, err, ErrNoConnectionForDomain, "another app's connection is not found")

	router := forge.NewRouter()
	require.NoError(t, f.p.RegisterRoutes(router))
	get := func(email string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(middleware.WithAppID(ctx, f.appID), http.MethodGet,
			"/v1/sso/discover?email="+url.QueryEscape(email), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	rec := get("ada@corp.example")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, map[string]any{
		"connection_id": f.conn.ID.String(), "provider": "corp", "name": "Corp Okta",
		"protocol": "saml", "domain": "corp.example", "enforced": true,
	}, body)

	for _, email := range []string{"bo@nowhere.example", "cy@old.example"} {
		rec = get(email)
		require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
		var miss map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &miss))
		assert.Equal(t, "no sso connection for this domain", miss["error"])
	}
}
