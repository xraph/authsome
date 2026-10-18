package api_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/middleware"
	apikeyplugin "github.com/xraph/authsome/plugins/apikey"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// randomHex returns n random bytes as hex, so the probe values below are
// recognisable in a failure message without being fixed strings a secret
// scanner would flag.
func randomHex(t *testing.T, n int) string {
	t.Helper()
	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return hex.EncodeToString(b)
}

// TestTelemetryCarriesNoCredentialsOrPersonalData drives sign-in (wrong
// password, unknown identifier, the MFA gate and the TOTP challenge),
// refresh, API key authentication and the OAuth2 token endpoint, all with
// values that are easy to spot, and then checks that none of them reached
// a span authsome started or the metric scrape.
//
// forge's own server span is left out on purpose: it records url.path and
// user_agent.original, which is forge's to redact, not authsome's.
func TestTelemetryCarriesNoCredentialsOrPersonalData(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)

	probe := randomHex(t, 6)
	email := "privacy-" + probe + "@example.com"
	unknownEmail := "nobody-" + probe + "@example.com"
	password := "Pw-" + probe + "-Aa1!"
	wrongPassword := "Wrong-" + probe + "-Aa1!"
	clientIP := "203.0.113.77"
	remoteIP := "198.51.100.23"
	userAgent := "PrivacyProbe/" + probe
	clientID := "client-" + probe
	clientSecret := "cs-" + randomHex(t, 16)

	mfaStore := mfa.NewMemoryStore()
	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfaStore)

	oauthStore := oauth2provider.NewMemoryStore()
	oauthPlugin := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	oauthPlugin.SetOAuth2Store(oauthStore)

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(
		authsome.WithConfig(testConfig()),
		authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithMetricsSink(reg),
		authsome.WithPlugin(mfaPlugin),
		authsome.WithPlugin(apikeyplugin.New()),
		authsome.WithPlugin(oauthPlugin),
	)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })
	secutil.RelaxAuthDefaults(t, eng)
	oauthPlugin.SetStore(s)

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	rootRouter := forge.NewRouter()
	a := api.New(eng, rootRouter)
	require.NoError(t, a.RegisterRoutes(rootRouter))
	require.NoError(t, mfaPlugin.RegisterRoutes(rootRouter))
	require.NoError(t, oauthPlugin.RegisterRoutes(rootRouter))
	// A route behind the auth middleware, so an API key goes through the
	// strategy the way it does in production.
	guarded := forge.NewRouter()
	guarded.Use(middleware.AuthMiddlewareWithStrategies(eng.ResolveSessionByToken, eng.ResolveUser, eng.Strategies(), eng.Logger()))
	require.NoError(t, guarded.GET("/whoami", func(c forge.Context) error {
		if _, ok := middleware.UserIDFrom(c.Context()); !ok {
			return c.NoContent(http.StatusUnauthorized)
		}
		return c.NoContent(http.StatusNoContent)
	}))
	router := withTestKey(rootRouter.Handler())

	// send runs one request with the probe's address and user agent.
	send := func(h http.Handler, method, path string, body any, hdr map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		var rdr *bytes.Reader
		if body != nil {
			b, mErr := json.Marshal(body)
			require.NoError(t, mErr)
			rdr = bytes.NewReader(b)
		} else {
			rdr = bytes.NewReader(nil)
		}
		req := httptest.NewRequestWithContext(ctx, method, path, rdr)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("X-Forwarded-For", clientIP)
		req.RemoteAddr = remoteIP + ":4242"
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		return out
	}

	// Sign up, then put a verified TOTP factor on the account and require
	// MFA for the app.
	_, _, _ = signUp(t, eng, email, password)
	u, err := s.GetUserByEmail(ctx, appID, id.Nil, email)
	require.NoError(t, err)
	totpKey, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "PrivacyProbe", AccountName: email})
	require.NoError(t, err)
	require.NoError(t, mfaStore.CreateEnrollment(ctx, &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: totpKey.Secret(), Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	tru := true
	require.NoError(t, s.SetAppClientConfig(ctx, &appclientconfig.Config{ID: id.NewAppClientConfigID(), AppID: appID, MFARequired: &tru}))

	// Sign-in: a wrong password, an unknown identifier, then the right
	// password, which the MFA gate answers with a ticket.
	assert.Equal(t, http.StatusUnauthorized, send(router, http.MethodPost, "/v1/signin", map[string]string{"email": email, "password": wrongPassword}, nil).Code)
	assert.Equal(t, http.StatusUnauthorized, send(router, http.MethodPost, "/v1/signin", map[string]string{"email": unknownEmail, "password": wrongPassword}, nil).Code)
	gateRes := send(router, http.MethodPost, "/v1/signin", map[string]string{"email": email, "password": password}, nil)
	require.Equal(t, http.StatusForbidden, gateRes.Code, gateRes.Body.String())
	var gate map[string]any
	require.NoError(t, json.NewDecoder(gateRes.Body).Decode(&gate))
	ticket, _ := gate["mfa_ticket"].(string)
	require.NotEmpty(t, ticket)

	// MFA: a wrong code, then the right one.
	code, err := mfa.GenerateTOTPCode(totpKey.Secret())
	require.NoError(t, err)
	wrongCode := "000000"
	if code == wrongCode {
		wrongCode = "111111"
	}
	assert.Equal(t, http.StatusUnauthorized, send(router, http.MethodPost, "/v1/mfa/challenge", map[string]string{"mfa_ticket": ticket, "code": wrongCode}, nil).Code)
	chRes := send(router, http.MethodPost, "/v1/mfa/challenge", map[string]string{"mfa_ticket": ticket, "code": code}, nil)
	require.Equal(t, http.StatusOK, chRes.Code, chRes.Body.String())
	var session map[string]any
	require.NoError(t, json.NewDecoder(chRes.Body).Decode(&session))
	accessToken, _ := session["session_token"].(string)
	refreshToken, _ := session["refresh_token"].(string)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// Refresh with the token the challenge returned.
	refRes := send(router, http.MethodPost, "/v1/refresh", map[string]string{"refresh_token": refreshToken}, nil)
	require.Equal(t, http.StatusOK, refRes.Code, refRes.Body.String())
	var refreshed map[string]any
	require.NoError(t, json.NewDecoder(refRes.Body).Decode(&refreshed))
	newAccess, _ := refreshed["session_token"].(string)
	newRefresh, _ := refreshed["refresh_token"].(string)

	// API key: a good key and a guess.
	rawKey, keyHash, keyPrefix, err := apikey.GenerateKey()
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, s.CreateAPIKey(ctx, &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: appID, UserID: u.ID, Name: "probe", KeyHash: keyHash, KeyPrefix: keyPrefix,
		CreatedAt: now, UpdatedAt: now,
	}))
	assert.Equal(t, http.StatusNoContent, send(guarded, http.MethodGet, "/whoami", nil,
		map[string]string{"Authorization": "Bearer " + rawKey, "X-App-ID": testAppIDStr}).Code)
	guess := rawKey[:len(rawKey)-4] + "zzzz"
	assert.NotEqual(t, http.StatusNoContent, send(guarded, http.MethodGet, "/whoami", nil,
		map[string]string{"Authorization": "Bearer " + guess, "X-App-ID": testAppIDStr}).Code)

	// OAuth2: client credentials with the right secret and a wrong one.
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, oauthStore.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: clientID, FirstParty: true,
		ClientSecret: string(hashedSecret), Name: "Probe", RedirectURIs: []string{"https://probe.example.com/cb"},
		Scopes: []string{"openid"}, GrantTypes: []string{"client_credentials"},
	}))
	tokRes := send(router, http.MethodPost, "/v1/oauth/token",
		map[string]string{"grant_type": "client_credentials", "client_id": clientID, "client_secret": clientSecret}, nil)
	require.Equal(t, http.StatusOK, tokRes.Code, tokRes.Body.String())
	var issued map[string]any
	require.NoError(t, json.NewDecoder(tokRes.Body).Decode(&issued))
	oauthAccess, _ := issued["access_token"].(string)
	assert.Equal(t, http.StatusUnauthorized, send(router, http.MethodPost, "/v1/oauth/token",
		map[string]string{"grant_type": "client_credentials", "client_id": clientID, "client_secret": clientSecret + "x"}, nil).Code)

	// The flows above produced the telemetry they should have.
	spans := telemetrytest.Authsome(rec)
	names := telemetrytest.Names(spans)
	for _, want := range []string{"verify password", "issue session", "refresh session", "authenticate api key", "verify client secret", "hash password"} {
		assert.Contains(t, names, want)
	}
	body := scrape()
	for _, want := range []string{
		`forge_authsome_signin_attempts_total{method="password",result="invalid"} 2`,
		`forge_authsome_signin_attempts_total{method="password",result="mfa_required"} 1`,
		`forge_authsome_mfa_verifications_total{flow="signin",method="totp",result="invalid"} 1`,
		`forge_authsome_mfa_verifications_total{flow="signin",method="totp",result="ok"} 1`,
		`forge_authsome_sessions_refreshed_total{result="ok"} 1`,
		`forge_authsome_apikey_authentications_total{result="ok"} 1`,
		`forge_authsome_apikey_authentications_total{result="invalid"} 1`,
		`forge_authsome_oauth2_tokens_requested_total{grant_type="client_credentials",result="ok"} 1`,
		`forge_authsome_oauth2_tokens_requested_total{grant_type="client_credentials",result="invalid_client"} 1`,
	} {
		assert.Contains(t, body, want)
	}

	// And none of the values reached any of it.
	secrets := map[string]string{
		"email":                  email,
		"unknown email":          unknownEmail,
		"email local part":       "privacy-" + probe,
		"password":               password,
		"wrong password":         wrongPassword,
		"client IP":              clientIP,
		"remote IP":              remoteIP,
		"user agent":             userAgent,
		"mfa ticket":             ticket,
		"totp code":              code,
		"totp secret":            totpKey.Secret(),
		"access token":           accessToken,
		"refresh token":          refreshToken,
		"refreshed access":       newAccess,
		"refreshed refresh":      newRefresh,
		"api key":                rawKey,
		"api key prefix":         keyPrefix,
		"api key guess":          guess,
		"oauth2 client id":       clientID,
		"oauth2 client secret":   clientSecret,
		"oauth2 access token":    oauthAccess,
		"user id":                u.ID.String(),
		"app id":                 testAppIDStr,
		"probe":                  probe,
		"publishable key":        testPublishableKey,
		"identifier-hash marker": "identifier_hash",
	}
	for _, s := range spans {
		text := telemetrytest.Text(s)
		for what, v := range secrets {
			if v == "" {
				continue
			}
			assert.False(t, strings.Contains(text, v), "%s reached span %q:\n%s", what, s.Name(), text)
		}
	}
	for what, v := range secrets {
		if v == "" {
			continue
		}
		assert.False(t, strings.Contains(body, v), "%s reached the metric scrape", what)
	}
}
