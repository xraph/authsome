package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/dpoptest"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/store/memory"

	"golang.org/x/oauth2"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// routablePlugin is the subset these tests need: something the engine accepts
// as a plugin and that can hang its own HTTP routes off the shared router.
type routablePlugin interface {
	plugin.Plugin
	RegisterRoutes(forge.Router) error
}

// pluginDPoPFixture wires an engine with one plugin registered and returns it
// alongside a router carrying both the first-party API routes and the plugin's
// own, so a test can drive a plugin sign-in path over HTTP against a real
// engine. The plugin packages' own test helpers stub the engine out, which
// means they exercise the fallback mint rather than IssueSession — the very
// path this change is about.
func pluginDPoPFixture(t *testing.T, p routablePlugin) (*authsome.Engine, http.Handler, id.AppID) {
	t.Helper()

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithPlugin(p),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	rootRouter := forge.NewRouter()
	a := api.New(eng, rootRouter)
	require.NoError(t, a.RegisterRoutes(rootRouter))
	require.NoError(t, p.RegisterRoutes(rootRouter))

	return eng, withTestKey(rootRouter.Handler()), appID
}

// TestMagicLinkVerify_UnderRequiredMode_BindsSession pins that the magic-link
// path resolves a binding instead of minting an exempt session.
//
// Magic link reads like a browser link-click, which would make a proof
// impossible. It isn't: /v1/magic-link/verify is a POST from the SDK carrying
// the token in the body, so the client holding the DPoP key is the caller and
// can prove possession like any other first-party route.
func TestMagicLinkVerify_UnderRequiredMode_BindsSession(t *testing.T) {
	t.Parallel()

	mlPlugin := magiclink.New()
	eng, router, appID := pluginDPoPFixture(t, mlPlugin)

	const email = "magiclink-dpop@example.com"
	_, _, _ = signUp(t, eng, email, "SecureP@ss1")
	u, err := eng.Store().GetUserByEmail(context.Background(), appID, id.Nil, email)
	require.NoError(t, err)

	// Seed the link directly; /send only exists to email this token out.
	v, err := account.NewVerification(context.Background(), appID, u.ID,
		magiclink.VerificationTypeMagicLink, 10*time.Minute)
	require.NoError(t, err)
	require.NoError(t, eng.Store().CreateVerification(context.Background(), v))

	setSignInDPoPMode(t, eng, "required")

	const endpoint = "http://example.com/v1/magic-link/verify"
	key := dpoptest.Key(t)
	jkt := dpoptest.Thumbprint(t, key)
	proof := dpoptest.MintProof(t, key, "ES256",
		dpoptest.ValidClaims(http.MethodPost, endpoint))

	body, _ := json.Marshal(map[string]string{"token": v.Token})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/magic-link/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DPoP", proof)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"a proof-carrying magic-link verify must succeed; body=%s", rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	tok, _ := resp["session_token"].(string)
	require.NotEmpty(t, tok)

	sess, err := eng.Store().GetSessionByToken(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, jkt, sess.DPoPJKT, "the magic-link session must be bound to the proven key")
}

// TestMagicLinkVerify_UnderRequiredMode_RefusesWithoutProof is the other half:
// the same route with no proof must not quietly hand back a session that the
// middleware will never demand anything for.
func TestMagicLinkVerify_UnderRequiredMode_RefusesWithoutProof(t *testing.T) {
	t.Parallel()

	mlPlugin := magiclink.New()
	eng, router, appID := pluginDPoPFixture(t, mlPlugin)

	const email = "magiclink-dpop-unbound@example.com"
	_, _, _ = signUp(t, eng, email, "SecureP@ss1")
	u, err := eng.Store().GetUserByEmail(context.Background(), appID, id.Nil, email)
	require.NoError(t, err)

	v, err := account.NewVerification(context.Background(), appID, u.ID,
		magiclink.VerificationTypeMagicLink, 10*time.Minute)
	require.NoError(t, err)
	require.NoError(t, eng.Store().CreateVerification(context.Background(), v))

	setSignInDPoPMode(t, eng, "required")

	body, _ := json.Marshal(map[string]string{"token": v.Token})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/magic-link/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"mode=required must refuse an unbound magic-link session; body=%s", rec.Body.String())
}

// TestVerifyEmail_UnderRequiredMode_BindsAutoLoginSession covers the
// email-verification auto-login. Signup deliberately withholds a session until
// the address is verified, so /v1/verify-email is a real sign-in surface, and
// it was minting unbound sessions like the plugin paths.
//
// This one has a quieter failure mode than the others: the handler treats
// auto-login as best-effort and falls back to a plain "email verified" status
// when issuance fails. So a missing binding never produced an error a client
// would notice, it just produced an exempt session — or, after the gate, no
// session at all. Only the positive case proves the wiring is there.
func TestVerifyEmail_UnderRequiredMode_BindsAutoLoginSession(t *testing.T) {
	t.Parallel()

	eng, router, appID := pluginDPoPFixture(t, magiclink.New())

	const email = "verify-email-dpop@example.com"
	_, _, _ = signUp(t, eng, email, "SecureP@ss1")
	u, err := eng.Store().GetUserByEmail(context.Background(), appID, id.Nil, email)
	require.NoError(t, err)

	// Seed the OTP the signup email would have carried.
	const otp = "424242"
	require.NoError(t, eng.Store().CreateVerification(context.Background(), &account.Verification{
		ID:        id.NewVerificationID(),
		AppID:     appID,
		UserID:    u.ID,
		Token:     otp,
		Type:      account.VerificationEmail,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}))

	setSignInDPoPMode(t, eng, "required")

	const endpoint = "http://example.com/v1/verify-email"
	key := dpoptest.Key(t)
	jkt := dpoptest.Thumbprint(t, key)
	proof := dpoptest.MintProof(t, key, "ES256",
		dpoptest.ValidClaims(http.MethodPost, endpoint))

	body, _ := json.Marshal(map[string]string{"code": otp, "email": email})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/verify-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DPoP", proof)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	tok, _ := resp["session_token"].(string)
	require.NotEmpty(t, tok,
		"a proof-carrying verify-email must auto-login, not fall back to a bare status")

	sess, err := eng.Store().GetSessionByToken(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, jkt, sess.DPoPJKT, "the auto-login session must be bound to the proven key")
}

// TestPhoneVerify_UnderRequiredMode_BindsSession pins the phone-OTP path.
// Same shape as magic link: /v1/phone/verify is a POST from the SDK, so a
// proof is available and the session it mints must carry one.
func TestPhoneVerify_UnderRequiredMode_BindsSession(t *testing.T) {
	t.Parallel()

	phonePlugin := phone.New()
	eng, router, appID := pluginDPoPFixture(t, phonePlugin)

	// Seed the challenge /start would have written and texted out. The
	// ceremony store is the engine's, which is what the plugin picked up when
	// the engine wired it.
	const number = "+15550001111"
	const code = "654321"
	challenge, _ := json.Marshal(map[string]any{
		"code":       code,
		"phone":      number,
		"app_id":     appID.String(),
		"expires_at": time.Now().Add(5 * time.Minute),
	})
	require.NoError(t, eng.CeremonyStore().Set(context.Background(),
		fmt.Sprintf("phoneauth:%s:%s", appID.String(), number), challenge, 5*time.Minute))

	setSignInDPoPMode(t, eng, "required")

	const endpoint = "http://example.com/v1/phone/verify"
	key := dpoptest.Key(t)
	jkt := dpoptest.Thumbprint(t, key)
	proof := dpoptest.MintProof(t, key, "ES256",
		dpoptest.ValidClaims(http.MethodPost, endpoint))

	body, _ := json.Marshal(map[string]string{
		"phone": number, "code": code, "app_id": appID.String(),
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/phone/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DPoP", proof)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code,
		"a proof-carrying phone verify must succeed; body=%s", rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	tok, _ := resp["session_token"].(string)
	require.NotEmpty(t, tok)

	sess, err := eng.Store().GetSessionByToken(context.Background(), tok)
	require.NoError(t, err)
	assert.Equal(t, jkt, sess.DPoPJKT, "the phone-OTP session must be bound to the proven key")
}

// dpopMockProvider is the smallest social.Provider that gets a callback past
// state validation. Its token endpoint is unreachable on purpose: if the
// refusal did not fire first, the callback would die in the code exchange, and
// the test would see that instead of the 400 it asserts.
type dpopMockProvider struct{}

func (dpopMockProvider) Name() string { return "google" }

func (dpopMockProvider) OAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"openid", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.invalid/auth",
			TokenURL: "https://example.invalid/token",
		},
	}
}

func (dpopMockProvider) FetchUser(_ context.Context, _ *oauth2.Token) (*social.ProviderUser, error) {
	return &social.ProviderUser{ProviderUserID: "p-1", Email: "social-dpop@example.com"}, nil
}

// TestSocialCallback_UnderRequiredMode_Refuses pins the decision that social
// sign-in and mode=required are not a supported combination.
//
// At a social callback the request is the identity provider redirecting the
// user agent, so the client holding the DPoP key is not the caller and has no
// opportunity to present a proof. There is no version of this path that can
// mint a bound session. Issuing an unbound one instead would mean anyone able
// to choose the sign-in method could opt out of the app's own mandate, so the
// callback refuses and the operator's escape hatch is lowering the app to
// optional.
//
// It refuses before the code exchange, not after: IssueSession's gate would
// catch this either way, but only once the single-use authorization code had
// already been redeemed and burned.
func TestSocialCallback_UnderRequiredMode_Refuses(t *testing.T) {
	t.Parallel()

	socialPlugin := social.New(social.Config{
		Providers:         []social.Provider{dpopMockProvider{}},
		SessionTokenTTL:   time.Hour,
		SessionRefreshTTL: 24 * time.Hour,
	})
	socialPlugin.SetOAuthStore(social.NewMemoryStore())
	eng, router, _ := pluginDPoPFixture(t, socialPlugin)

	// Start the flow so the callback has a state it will accept.
	startRec := httptest.NewRecorder()
	startReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/social/google", nil)
	router.ServeHTTP(startRec, startReq)
	require.Equal(t, http.StatusOK, startRec.Code, "body=%s", startRec.Body.String())
	var startResp map[string]string
	require.NoError(t, json.NewDecoder(startRec.Body).Decode(&startResp))
	parsed, err := url.Parse(startResp["auth_url"])
	require.NoError(t, err)
	state := parsed.Query().Get("state")
	require.NotEmpty(t, state)

	setSignInDPoPMode(t, eng, "required")

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"/v1/social/google/callback?state="+state+"&code=abc", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code,
		"a required-mode app must refuse the social callback; body=%s", rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "invalid_dpop_proof", resp["error"],
		"the refusal must be the DPoP one, not a downstream token-exchange failure")
}
