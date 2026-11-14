package authsome_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/social/socialtest"
	"github.com/xraph/authsome/plugins/waitlist"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// A browser that signs in through a social provider comes back to the
// frontend URL it started from, whatever the outcome: a second factor owed,
// or a sign-up the app refused, is handed back in the query rather than as a
// JSON body the browser would render.

const kinetaFrontend = "https://app.wakflo.test"

type kinetaFlow struct {
	eng   *authsome.Engine
	iss   *socialtest.Issuer
	appID id.AppID
	serve func(method, target, body string) *httptest.ResponseRecorder
}

func newKinetaFlow(t *testing.T, opts ...authsome.Option) *kinetaFlow {
	t.Helper()
	ctx := context.Background()
	iss := socialtest.NewIssuer(t, "kineta-client")
	sp := social.New(social.Config{HTTPClient: iss.Client()})
	sp.SetOAuthStore(social.NewMemoryStore())
	eng, _ := newTestEngine(t, append([]authsome.Option{authsome.WithPlugin(sp)}, opts...)...)
	appID := testAppID(t)

	raw, err := json.Marshal([]social.ProviderSetting{{
		Name: "kineta", Type: social.ProviderTypeOIDC, Issuer: iss.URL, DisplayName: "Kineta",
		ClientID: "kineta-client", ClientSecret: "kineta-secret", Enabled: true,
	}})
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, social.SettingSocialProviders.Def.Key, raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
	allowed, err := json.Marshal(kinetaFrontend)
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, social.SettingAllowedFrontendURLs.Def.Key, allowed,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	router := forge.NewRouter()
	require.NoError(t, sp.RegisterRoutes(router))
	scoped := forge.WithScope(ctx, forge.NewAppScope(appID.String()))
	return &kinetaFlow{eng: eng, iss: iss, appID: appID, serve: func(method, target, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(scoped, method, target, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}}
}

// signIn runs a start and a GET callback for the given id_token claims. A
// non-empty redirectURL is passed to start with the allowlisted frontend.
func (f *kinetaFlow) signIn(t *testing.T, redirectURL string, claims map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	target := "/v1/social/kineta"
	if redirectURL != "" {
		target += "?" + url.Values{"frontend_url": {kinetaFrontend}, "redirect_url": {redirectURL}}.Encode()
	}
	rec := f.serve(http.MethodPost, target, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var start struct {
		AuthURL string `json:"auth_url"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &start))
	state := socialtest.AuthorizeURLParam(start.AuthURL, "state")
	c := map[string]any{"nonce": socialtest.AuthorizeURLParam(start.AuthURL, "nonce")}
	for k, v := range claims {
		c[k] = v
	}
	f.iss.SetClaims(c)
	return f.serve(http.MethodGet, "/v1/social/kineta/callback?code=issuer-code&state="+url.QueryEscape(state), "")
}

func (f *kinetaFlow) setClientConfig(t *testing.T, cfg appclientconfig.Config) {
	t.Helper()
	cfg.ID = id.NewAppClientConfigID()
	cfg.AppID = f.appID
	require.NoError(t, f.eng.Store().SetAppClientConfig(context.Background(), &cfg))
}

func (f *kinetaFlow) users(t *testing.T) []*user.User {
	t.Helper()
	list, err := f.eng.Store().ListUsers(context.Background(), &user.Query{AppID: f.appID})
	require.NoError(t, err)
	return list.Users
}

func kinetaClaims(sub, email string) map[string]any {
	return map[string]any{"sub": sub, "email": email, "email_verified": true, "given_name": "Ada"}
}

func redirectedTo(t *testing.T, rec *httptest.ResponseRecorder) *url.URL {
	t.Helper()
	require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
	loc, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	return loc
}

func TestSocialCallback_MFARedirectsToTheFrontendWithTheTicket(t *testing.T) {
	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfa.NewMemoryStore())
	f := newKinetaFlow(t, authsome.WithPlugin(mfaPlugin))
	tru := true
	f.setClientConfig(t, appclientconfig.Config{MFARequired: &tru})

	rec := f.signIn(t, kinetaFrontend+"/sign-in/done?next=%2Fflows", kinetaClaims("kineta-1", "ada@wakflo.test"))
	loc := redirectedTo(t, rec)
	assert.Equal(t, "app.wakflo.test", loc.Host)
	assert.Equal(t, "/sign-in/done", loc.Path)
	q := loc.Query()
	assert.Equal(t, "/flows", q.Get("next"), "the redirect URL's own query is kept")
	assert.Equal(t, "totp", q.Get("mfa_methods"))
	ticket := q.Get("mfa_ticket")
	require.NotEmpty(t, ticket)
	assert.Empty(t, rec.Result().Cookies(), "no session cookie before the second factor")
	assert.Empty(t, loc.Fragment)

	payload, err := f.eng.LoadMFATicket(context.Background(), ticket)
	require.NoError(t, err)
	assert.Equal(t, "social:kineta", payload.AuthMethod, "the ticket keeps the auth method for the session minted after the challenge")
}

func TestSocialCallback_MFAWithoutAFrontendURLKeepsTheJSONAnswer(t *testing.T) {
	f := newKinetaFlow(t)
	tru := true
	f.setClientConfig(t, appclientconfig.Config{MFARequired: &tru})

	// A flow started without a frontend URL has nowhere trusted to send
	// the browser, so the 403 ticket envelope stands.
	rec := f.signIn(t, "", kinetaClaims("kineta-2", "bo@wakflo.test"))
	require.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "mfa_required", body["type"])
	assert.NotEmpty(t, body["mfa_ticket"])
}

func TestSocialCallback_MFARedirectHonoursTheAllowlist(t *testing.T) {
	f := newKinetaFlow(t)
	tru := true
	f.setClientConfig(t, appclientconfig.Config{MFARequired: &tru})

	// An absolute redirect to another origin is refused at start; the
	// browser goes to the allowlisted frontend_url instead.
	rec := f.signIn(t, "https://evil.example/steal", kinetaClaims("kineta-3", "cy@wakflo.test"))
	loc := redirectedTo(t, rec)
	assert.Equal(t, "app.wakflo.test", loc.Host)
	assert.NotEmpty(t, loc.Query().Get("mfa_ticket"))
}

func TestSocialCallback_SignUpDisabledRedirectsAndCreatesNoUser(t *testing.T) {
	f := newKinetaFlow(t)
	off := false
	f.setClientConfig(t, appclientconfig.Config{SignupEnabled: &off})

	loc := redirectedTo(t, f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-4", "di@wakflo.test")))
	assert.Equal(t, "app.wakflo.test", loc.Host)
	assert.Equal(t, authsome.SignUpRefusedDisabled, loc.Query().Get("error"))
	assert.Empty(t, f.users(t), "a refused sign-up writes no user")
}

func TestSocialCallback_WaitlistRefusalRedirectsWithItsCode(t *testing.T) {
	f := newKinetaFlow(t, authsome.WithPlugin(waitlist.New(waitlist.Config{Enabled: true})))

	loc := redirectedTo(t, f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-5", "ed@wakflo.test")))
	assert.Equal(t, authsome.SignUpRefusedWaitlist, loc.Query().Get("error"))
	assert.Empty(t, f.users(t))
}

// signUpRecorder refuses or records every sign-up the engine gates.
type signUpRecorder struct {
	refuse  bool
	before  []*account.SignUpRequest
	created []id.UserID
}

func (r *signUpRecorder) Name() string { return "signup-recorder" }
func (r *signUpRecorder) OnBeforeSignUp(_ context.Context, req *account.SignUpRequest) error {
	r.before = append(r.before, req)
	if r.refuse {
		return errors.New("setup: workspace is not accepting members")
	}
	return nil
}

func (r *signUpRecorder) OnAfterSignUp(_ context.Context, u *user.User, _ *session.Session) error {
	r.created = append(r.created, u.ID)
	return nil
}

func TestSocialCallback_SignUpHooksSeeTheFirstSignInOnly(t *testing.T) {
	rec := &signUpRecorder{}
	f := newKinetaFlow(t, authsome.WithPlugin(rec))

	resp := f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-6", "fay@wakflo.test"))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	require.Len(t, rec.before, 1)
	assert.Equal(t, "fay@wakflo.test", rec.before[0].Email)
	assert.Equal(t, "social:kineta", rec.before[0].AuthMethod)
	assert.Empty(t, rec.before[0].Password)
	users := f.users(t)
	require.Len(t, users, 1)
	assert.Equal(t, []id.UserID{users[0].ID}, rec.created, "after-sign-up hooks fire for the new account")

	// The same identity signing in again is not a sign-up, so a gate that
	// now refuses everyone doesn't stop it.
	rec.refuse = true
	resp = f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-6", "fay@wakflo.test"))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Len(t, rec.before, 1, "an existing account is never gated")

	// A new identity is.
	loc := redirectedTo(t, f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-7", "gus@wakflo.test")))
	assert.Equal(t, authsome.SignUpRefusedHook, loc.Query().Get("error"))
	assert.Len(t, f.users(t), 1)
}

func TestSocialCallback_ExistingUserIsNotBlockedBySignUpGates(t *testing.T) {
	f := newKinetaFlow(t, authsome.WithPlugin(waitlist.New(waitlist.Config{Enabled: true})))
	ctx := context.Background()
	// Created before the waitlist mattered: written straight to the store
	// with a verified email the provider identity links to.
	u := &user.User{ID: id.NewUserID(), AppID: f.appID, Email: "hal@wakflo.test", EmailVerified: true}
	require.NoError(t, f.eng.Store().CreateUserWithPrimaryEmail(ctx, u, user.NewPrimaryEmail(u, "password")))
	off := false
	f.setClientConfig(t, appclientconfig.Config{SignupEnabled: &off})

	resp := f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-8", "hal@wakflo.test"))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), "Login successful")
	assert.Len(t, f.users(t), 1)
}

var handoffCodeRE = regexp.MustCompile(`code(?:=|\\u003D)([0-9a-f]{64})`)

// An app that keeps the session tokens on its own origin gets a one-time
// code on the redirect and trades it, once, for a rotated token pair.
func TestSocialCallback_BrowserSuccessHandsOverAOneTimeCode(t *testing.T) {
	f := newKinetaFlow(t)

	resp := f.signIn(t, kinetaFrontend+"/sign-in/done?next=%2Fflows", kinetaClaims("kineta-9", "ivy@wakflo.test"))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.Contains(t, resp.Body.String(), "Login successful", "the popup-closing page is kept")
	var cookieToken string
	for _, c := range resp.Result().Cookies() {
		if c.Value != "" {
			cookieToken = c.Value
		}
	}
	require.NotEmpty(t, cookieToken, "cookie-mode clients still get the session cookie")
	m := handoffCodeRE.FindStringSubmatch(resp.Body.String())
	require.Len(t, m, 2, "the redirect carries code=: %s", resp.Body.String())
	code := m[1]

	exchange := func(code string) *httptest.ResponseRecorder {
		return f.serve(http.MethodPost, "/v1/social/exchange", `{"code":"`+code+`"}`)
	}
	rec := exchange(code)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body struct {
		SessionToken string         `json:"session_token"`
		RefreshToken string         `json:"refresh_token"`
		ExpiresAt    string         `json:"expires_at"`
		Provider     string         `json:"provider"`
		IsNewUser    bool           `json:"is_new_user"`
		User         map[string]any `json:"user"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.NotEmpty(t, body.SessionToken)
	assert.NotEmpty(t, body.RefreshToken)
	assert.NotEmpty(t, body.ExpiresAt)
	assert.Equal(t, "kineta", body.Provider)
	assert.True(t, body.IsNewUser)
	assert.Equal(t, "ivy@wakflo.test", body.User["email"])
	assert.NotEqual(t, cookieToken, body.SessionToken, "the exchange rotates the session")

	_, err := f.eng.ResolveSessionByToken(context.Background(), body.SessionToken)
	require.NoError(t, err, "the exchanged token authenticates")

	assert.Equal(t, http.StatusBadRequest, exchange(code).Code, "a code redeems once")
	assert.Equal(t, http.StatusBadRequest, exchange(strings.Repeat("0", 64)).Code, "an unknown code is refused")
}

func TestSocialCallback_CodeIsBoundToItsApp(t *testing.T) {
	f := newKinetaFlow(t)
	resp := f.signIn(t, kinetaFrontend+"/sign-in", kinetaClaims("kineta-10", "jo@wakflo.test"))
	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	m := handoffCodeRE.FindStringSubmatch(resp.Body.String())
	require.Len(t, m, 2)

	sp, ok := f.eng.Plugin("social").(*social.Plugin)
	require.True(t, ok)
	router := forge.NewRouter()
	require.NoError(t, sp.RegisterRoutes(router))
	other := middleware.WithAppID(context.Background(), id.NewAppID())
	req := httptest.NewRequestWithContext(other, http.MethodPost, "/v1/social/exchange", strings.NewReader(`{"code":"`+m[1]+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
