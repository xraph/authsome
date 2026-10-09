package oauth2provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
)

// A token the provider issues belongs to its client: only that client can
// refresh it at the token endpoint, the session refresh route refuses it,
// and userinfo shows only what its scopes cover.

func newEngineFixture(t *testing.T) (*authsome.Engine, forge.Router, id.AppID) {
	t.Helper()
	eng := secutil.NewTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	st := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(st)
	require.NoError(t, p.OnInit(context.Background(), eng))
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: publicID, Name: "Public", FirstParty: true,
		RedirectURIs: []string{registeredURI}, Scopes: []string{"openid", "profile", "email"},
		GrantTypes: []string{"authorization_code"}, Public: true, TokenEndpointAuthMethod: "none",
	}))
	require.NoError(t, st.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: appID, ClientID: "other-public", Name: "Other", FirstParty: true,
		RedirectURIs: []string{registeredURI}, Scopes: []string{"openid"},
		GrantTypes: []string{"authorization_code"}, Public: true, TokenEndpointAuthMethod: "none",
	}))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return eng, mux, appID
}

func obtainTokens(t *testing.T, mux forge.Router, userID id.UserID, appID id.AppID, scope string) oauth2provider.TokenResponse {
	t.Helper()
	const verifier = "verifier-verifier-verifier-verifier-1234"
	q := baseAuthorizeQuery(publicID)
	q.Set("scope", scope)
	q.Set("code_challenge", s256(verifier))
	q.Set("code_challenge_method", "S256")
	ctx := middleware.WithAppID(middleware.WithUserID(context.Background(), userID), appID)
	req := httptest.NewRequestWithContext(ctx, "GET", "/v1/oauth/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	code := codeFrom(t, rec)
	tok := postToken(t, mux, map[string]string{
		"grant_type": "authorization_code", "code": code, "redirect_uri": registeredURI,
		"client_id": publicID, "code_verifier": verifier,
	})
	require.Equal(t, http.StatusOK, tok.Code, tok.Body.String())
	assert.Equal(t, "no-store", tok.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", tok.Header().Get("Pragma"))
	var out oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(tok.Body.Bytes(), &out))
	require.NotEmpty(t, out.RefreshToken)
	return out
}

func TestRefreshGrant_RotatesOnlyForTheOwningClient(t *testing.T) {
	eng, mux, appID := newEngineFixture(t)
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{AppID: appID, Email: "bound@example.com", Password: "SecureP@ss123", FirstName: "B"})
	require.NoError(t, err)
	first := obtainTokens(t, mux, u.ID, appID, "openid profile")

	sess, err := eng.ResolveSessionByToken(context.Background(), first.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, publicID, sess.ClientID, "the session is stamped with its client")

	// The session refresh route refuses an OAuth2 refresh token.
	_, err = eng.Refresh(context.Background(), first.RefreshToken)
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)

	// Another client cannot redeem it.
	wrong := postToken(t, mux, map[string]string{"grant_type": "refresh_token", "refresh_token": first.RefreshToken, "client_id": "other-public"})
	assert.Equal(t, http.StatusBadRequest, wrong.Code, wrong.Body.String())
	assert.Contains(t, wrong.Body.String(), "invalid_grant")

	// The owning client gets a rotated pair carrying the same scopes.
	rotated := postToken(t, mux, map[string]string{"grant_type": "refresh_token", "refresh_token": first.RefreshToken, "client_id": publicID})
	require.Equal(t, http.StatusOK, rotated.Code, rotated.Body.String())
	var second oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(rotated.Body.Bytes(), &second))
	assert.NotEqual(t, first.AccessToken, second.AccessToken)
	assert.NotEqual(t, first.RefreshToken, second.RefreshToken)
	assert.Equal(t, "openid profile", second.Scope)
	live, err := eng.ResolveSessionByToken(context.Background(), second.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, publicID, live.ClientID)
	_, err = eng.ResolveSessionByToken(context.Background(), first.AccessToken)
	assert.Error(t, err, "the previous access token is retired")

	// Replaying the spent refresh token revokes the family.
	replay := postToken(t, mux, map[string]string{"grant_type": "refresh_token", "refresh_token": first.RefreshToken, "client_id": publicID})
	assert.Equal(t, http.StatusBadRequest, replay.Code)
	_, err = eng.ResolveSessionByToken(context.Background(), second.AccessToken)
	assert.Error(t, err, "a replay revokes the whole family")
}

func TestAuthorize_RefusesASessionFromAnotherApp(t *testing.T) {
	_, mux, _ := newEngineFixture(t)
	q := baseAuthorizeQuery(publicID)
	q.Set("code_challenge", s256("verifier-verifier-verifier-verifier-1234"))
	ctx := middleware.WithAppID(middleware.WithUserID(context.Background(), id.NewUserID()), id.NewAppID())
	req := httptest.NewRequestWithContext(ctx, "GET", "/v1/oauth/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestAuthorize_PlainPKCEIsRefusedForEveryClient(t *testing.T) {
	_, _, mux := newFixture(t)
	q := baseAuthorizeQuery(confidentialID)
	q.Set("code_challenge", "the-verifier-in-the-clear")
	q.Set("code_challenge_method", "plain")
	rec := authorize(t, mux, q)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestUserInfo_ClaimsFollowTheTokenScopes(t *testing.T) {
	eng, mux, appID := newEngineFixture(t)
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{AppID: appID, Email: "claims@example.com", Password: "SecureP@ss123", FirstName: "Claire"})
	require.NoError(t, err)

	call := func(scopes ...string) (int, map[string]any) {
		sess := &session.Session{ID: id.NewSessionID(), AppID: appID, UserID: u.ID, ClientID: publicID, Scopes: scopes}
		ctx := middleware.WithPendingOAuthSession(context.Background(), &middleware.PendingOAuthSession{Session: sess})
		req := httptest.NewRequestWithContext(ctx, "GET", "/v1/oauth/userinfo", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}
	code, _ := call("profile")
	assert.Equal(t, http.StatusForbidden, code, "userinfo needs openid")

	code, body := call("openid")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, u.ID.String(), body["sub"])
	assert.Nil(t, body["email"], "email is not granted")
	assert.Nil(t, body["name"], "profile is not granted")

	code, body = call("openid", "email", "profile")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "claims@example.com", body["email"])
	assert.Equal(t, "Claire", body["name"])
}

func TestDeviceAuthorize_ScopesAreBoundedByTheRegistration(t *testing.T) {
	_, _, mux := newFixture(t)
	form := url.Values{"client_id": {publicID}, "scope": {"openid admin:everything"}}
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/oauth/device/authorize", nil)
	req.URL.RawQuery = form.Encode()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestDiscovery_AdvertisesOnlyWhatExists(t *testing.T) {
	_, _, mux := newFixture(t)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var doc oauth2provider.DiscoveryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	assert.Contains(t, doc.GrantTypesSupported, "refresh_token")
	assert.Equal(t, []string{"S256"}, doc.CodeChallengeMethodsSupported)
}
