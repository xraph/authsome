package authsome_test

// An OAuth2 access token is issued to a third party for a set of scopes, so
// the auth middleware parks it until a RequireScope route admits it. In
// opaque mode the middleware finds the session row, sees its ClientID and
// parks it. In JWT mode there is no row lookup, so the token itself has to
// say which client it belongs to. Before it did, a JWT minted by the
// /v1/oauth/token endpoint authenticated as the full user on every route.
//
// This test issues the token through the real grant and presents it to the
// engine's real middleware, because the bug lived at exactly that seam: each
// half had its own passing tests.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/tokenformat"
)

// issueOAuth2JWT runs an authorization-code flow for scope "profile" against
// a plugin initialised from eng, so the token endpoint mints with the app's
// JWT format. It returns the token response and a function that redeems a
// refresh token at the same endpoint.
func issueOAuth2JWT(t *testing.T, eng *authsome.Engine, appID id.AppID) (*oauth2provider.TokenResponse, func(string) *oauth2provider.TokenResponse) {
	t.Helper()
	ctx := context.Background()

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "oauth2-jwt@example.com",
		Password:  "SecureP@ss123",
		FirstName: "JWT Scope",
	})
	require.NoError(t, err)

	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	oauthStore := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(oauthStore)
	require.NoError(t, p.OnInit(ctx, eng))

	hashed, err := bcrypt.GenerateFromPassword([]byte(e2eClientSecret), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, oauthStore.CreateClient(ctx, &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     e2eClientID,
		FirstParty:   true,
		ClientSecret: string(hashed),
		Name:         "Third Party",
		RedirectURIs: []string{e2eRedirectURI},
		Scopes:       []string{"openid", "profile"},
		GrantTypes:   []string{"authorization_code"},
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	q := url.Values{
		"response_type": {"code"},
		"client_id":     {e2eClientID},
		"redirect_uri":  {e2eRedirectURI},
		"scope":         {"profile"},
	}
	authRec := httptest.NewRecorder()
	mux.ServeHTTP(authRec, httptest.NewRequestWithContext(
		middleware.WithUserID(ctx, sess.UserID), http.MethodGet,
		"/v1/oauth/authorize?"+q.Encode(), nil))
	require.Equal(t, http.StatusFound, authRec.Code, "body: %s", authRec.Body.String())
	loc, err := url.Parse(authRec.Header().Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)

	redeem := func(form map[string]string) *oauth2provider.TokenResponse {
		form["client_id"] = e2eClientID
		form["client_secret"] = e2eClientSecret
		body, err := json.Marshal(form)
		require.NoError(t, err)
		tokReq := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/oauth/token", bytes.NewReader(body))
		tokReq.Header.Set("Content-Type", "application/json")
		tokRec := httptest.NewRecorder()
		mux.ServeHTTP(tokRec, tokReq)
		require.Equal(t, http.StatusOK, tokRec.Code, "body: %s", tokRec.Body.String())

		var resp oauth2provider.TokenResponse
		require.NoError(t, json.Unmarshal(tokRec.Body.Bytes(), &resp))
		// Guard the premise: an opaque token would be parked through its
		// session row and this test would pass without exercising the JWT
		// path at all.
		require.True(t, tokenformat.IsJWT(resp.AccessToken), "the app is JWT-configured, so the grant must mint a JWT")
		return &resp
	}

	resp := redeem(map[string]string{
		"grant_type":   "authorization_code",
		"code":         code,
		"redirect_uri": e2eRedirectURI,
	})
	refresh := func(refreshToken string) *oauth2provider.TokenResponse {
		return redeem(map[string]string{"grant_type": "refresh_token", "refresh_token": refreshToken})
	}
	return resp, refresh
}

func TestOAuth2IssuedJWTIsParkedUntilAScopedRouteAdmitsIt(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)
	eng, _ := newTestEngine(t, authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))

	issued, refresh := issueOAuth2JWT(t, eng, testAppID(t))
	require.NotEmpty(t, issued.RefreshToken)

	whoami := func(ctx forge.Context) error {
		if _, ok := middleware.UserFrom(ctx.Context()); !ok {
			return ctx.NoContent(http.StatusUnauthorized)
		}
		return ctx.NoContent(http.StatusOK)
	}
	router := forge.NewRouter()
	router.Use(eng.AuthMiddleware())
	require.NoError(t, router.GET("/untagged", whoami))
	require.NoError(t, router.GET("/profile", whoami, forge.WithMiddleware(middleware.RequireScope("profile"))))
	require.NoError(t, router.GET("/admin", whoami, forge.WithMiddleware(middleware.RequireScope("admin"))))

	call := func(token, path string) int {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	// The refresh grant re-mints through the engine rather than the plugin,
	// so it is a second place the client binding can fall off the token.
	tokens := map[string]string{
		"issued by the code grant":    issued.AccessToken,
		"issued by the refresh grant": refresh(issued.RefreshToken).AccessToken,
	}
	for name, token := range tokens {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, http.StatusUnauthorized, call(token, "/untagged"),
				"a third-party JWT must not authenticate as the user on a route that never declared scopes")
			assert.Equal(t, http.StatusOK, call(token, "/profile"), "a route asking for a granted scope admits the token")
			assert.Equal(t, http.StatusForbidden, call(token, "/admin"), "a route asking for an ungranted scope refuses it")
		})
	}
}
