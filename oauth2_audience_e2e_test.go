package authsome_test

// End-to-end composition test for RFC 8707: a token the OAuth2 provider
// actually issued, presented to the auth middleware the engine actually
// builds.
//
// Every hop already has its own test. The provider's tests prove a `resource`
// parameter reaches the issued token; the middleware's tests prove an audience
// check refuses a mismatch; the engine's tests prove the resolver is wired.
// None of them ever ran an issued token into the middleware, and both of this
// branch's composition bugs lived at exactly that seam: the resolver reading
// the wrong app id, and a refused JWT being rescued by a session lookup.
//
// So this test issues nothing by hand. It drives /v1/oauth/authorize and
// /v1/oauth/token for real, takes whatever access token comes back, and hands
// that string to eng.AuthMiddleware().

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
	"golang.org/x/crypto/bcrypt"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

const (
	e2eClientID     = "e2e-conf-client"
	e2eClientSecret = "e2e-conf-secret"
	e2eRedirectURI  = "https://app.example.com/cb"
	e2eResourceAPI  = "https://api.example.com"
	e2eResourceElse = "https://files.example.com"
)

// issueOAuth2AccessToken runs a full authorization-code flow against a real
// plugin sharing the engine's store, and returns the access token the token
// endpoint handed back.
//
// The plugin writes its session straight into the same store the engine reads
// from, which is what makes the token the middleware sees the same token the
// grant issued rather than a stand-in built in the test.
func issueOAuth2AccessToken(t *testing.T, eng *authsome.Engine, appID id.AppID, resource string) string {
	t.Helper()

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     "oauth2-e2e@example.com",
		Password:  "SecureP@ss1",
		FirstName: "E2E User",
	})
	require.NoError(t, err)

	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	oauthStore := oauth2provider.NewMemoryStore()
	p.SetOAuth2Store(oauthStore)
	p.SetStore(eng.Store())

	hashed, err := bcrypt.GenerateFromPassword([]byte(e2eClientSecret), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, oauthStore.CreateClient(context.Background(), &oauth2provider.OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		ClientID:     e2eClientID,
		ClientSecret: string(hashed),
		Name:         "E2E Confidential",
		RedirectURIs: []string{e2eRedirectURI},
		Scopes:       []string{"openid", "profile"},
		Resources:    []string{e2eResourceAPI, e2eResourceElse},
		GrantTypes:   []string{"authorization_code"},
	}))

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	// Authorize as the signed-up user, naming the resource.
	q := url.Values{
		"response_type": {"code"},
		"client_id":     {e2eClientID},
		"redirect_uri":  {e2eRedirectURI},
		"resource":      {resource},
	}
	authCtx := middleware.WithUserID(context.Background(), sess.UserID)
	authRec := httptest.NewRecorder()
	mux.ServeHTTP(authRec, httptest.NewRequestWithContext(authCtx, http.MethodGet,
		"/v1/oauth/authorize?"+q.Encode(), nil))
	require.Equal(t, http.StatusFound, authRec.Code, "body: %s", authRec.Body.String())

	loc, err := url.Parse(authRec.Header().Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code, "authorize did not return a code")

	// Redeem the code.
	body, err := json.Marshal(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     e2eClientID,
		"client_secret": e2eClientSecret,
		"redirect_uri":  e2eRedirectURI,
	})
	require.NoError(t, err)
	tokReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/oauth/token", bytes.NewReader(body))
	tokReq.Header.Set("Content-Type", "application/json")
	tokRec := httptest.NewRecorder()
	mux.ServeHTTP(tokRec, tokReq)
	require.Equal(t, http.StatusOK, tokRec.Code, "body: %s", tokRec.Body.String())

	var resp oauth2provider.TokenResponse
	require.NoError(t, json.Unmarshal(tokRec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)

	// Guard the premise. If the grant stopped attaching the audience this test
	// would keep passing while proving nothing, because an unaudienced token
	// passes every audience check by design.
	issued, err := eng.Store().GetSessionByToken(context.Background(), resp.AccessToken)
	require.NoError(t, err)
	require.Equal(t, []string{resource}, issued.Audience,
		"the issued token carries no audience, so the rest of this test would be vacuous")

	return resp.AccessToken
}

// TestOAuth2IssuedTokenMeetsAuthMiddleware is the composition check.
//
// Same flow, same issued token, two deployments: one whose resource identifier
// matches what the token was issued for, one whose does not. The matching
// deployment must authenticate and the other must not. Running both is what
// makes the result meaningful, since a check that refused everything would
// satisfy the rejection half on its own.
func TestOAuth2IssuedTokenMeetsAuthMiddleware(t *testing.T) {
	cases := []struct {
		name               string
		resourceIdentifier string
		wantStatus         int
	}{
		{
			name:               "the deployment answers to the resource the token names",
			resourceIdentifier: e2eResourceAPI,
			wantStatus:         http.StatusOK,
		},
		{
			name:               "the deployment answers to a different resource",
			resourceIdentifier: e2eResourceElse,
			wantStatus:         http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eng, _ := newTestEngine(t)
			appID := testAppID(t)

			token := issueOAuth2AccessToken(t, eng, appID, e2eResourceAPI)
			setResourceIdentifier(t, eng, appID, tc.resourceIdentifier)

			rec := httptest.NewRecorder()
			audienceRouter(eng).ServeHTTP(rec, audienceRequest(token))

			assert.Equal(t, tc.wantStatus, rec.Code,
				"an OAuth2-issued token audienced at %s, presented to a deployment answering to %s",
				e2eResourceAPI, tc.resourceIdentifier)
		})
	}
}
