package authsome_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/social/socialtest"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// An app adds another OpenID Connect issuer (here a stand-in for a second
// authsome deployment) as a sign-in button, and a user signs in with it:
// the button carries the app's label, start sends the browser to the
// issuer's authorize endpoint, and the callback verifies the id_token and
// signs the user in.
func TestSocial_GenericOIDCProviderSignsIn(t *testing.T) {
	ctx := context.Background()
	iss := socialtest.NewIssuer(t, "muono-client")
	sp := social.New(social.Config{HTTPClient: iss.Client()})
	sp.SetOAuthStore(social.NewMemoryStore()) // a real deployment wires this from its database
	eng, st := newTestEngine(t, authsome.WithPlugin(sp))
	appID := testAppID(t)

	raw, err := json.Marshal([]social.ProviderSetting{{
		Name: "muono", Type: social.ProviderTypeOIDC, Issuer: iss.URL, DisplayName: "Muono",
		ClientID: "muono-client", ClientSecret: "muono-secret", Enabled: true,
	}})
	require.NoError(t, err)
	require.NoError(t, eng.Settings().Set(ctx, social.SettingSocialProviders.Def.Key, raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))

	cfg := eng.ClientConfig(ctx, appID)
	require.NotNil(t, cfg.Social)
	assert.Contains(t, cfg.Social.Providers, authsome.ClientConfigSocialProvider{ID: "muono", Name: "Muono"})

	router := forge.NewRouter()
	require.NoError(t, sp.RegisterRoutes(router))
	scoped := forge.WithScope(ctx, forge.NewAppScope(appID.String()))
	serve := func(method, target string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(scoped, method, target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	start := func() (state, nonce string) {
		rec := serve(http.MethodPost, "/v1/social/muono")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var body struct {
			AuthURL string `json:"auth_url"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.True(t, strings.HasPrefix(body.AuthURL, iss.URL+"/authorize?"), body.AuthURL)
		assert.Equal(t, "muono-client", socialtest.AuthorizeURLParam(body.AuthURL, "client_id"))
		assert.Contains(t, socialtest.AuthorizeURLParam(body.AuthURL, "scope"), "openid")
		assert.NotEmpty(t, socialtest.AuthorizeURLParam(body.AuthURL, "code_challenge"))
		return socialtest.AuthorizeURLParam(body.AuthURL, "state"), socialtest.AuthorizeURLParam(body.AuthURL, "nonce")
	}
	callback := func(state string) *httptest.ResponseRecorder {
		return serve(http.MethodGet, "/v1/social/muono/callback?code=issuer-code&state="+url.QueryEscape(state))
	}

	state, nonce := start()
	require.NotEmpty(t, nonce)
	iss.SetClaims(map[string]any{
		"sub": "muono-user-7", "nonce": nonce, "email": "ada@example.com", "email_verified": true,
		"given_name": "Ada", "family_name": "Lovelace",
	})
	rec := callback(state)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "Login successful")
	assert.NotEmpty(t, rec.Result().Cookies(), "a session cookie is set")
	assert.Equal(t, int32(1), iss.TokenRequests.Load())

	users, err := st.ListUsers(ctx, &user.Query{AppID: appID})
	require.NoError(t, err)
	require.Len(t, users.Users, 1)
	u := users.Users[0]
	assert.Equal(t, "ada@example.com", u.Email)
	assert.Equal(t, "Ada", u.FirstName)
	assert.Equal(t, "Lovelace", u.LastName)

	// The same issuer identity signs in to the same account next time,
	// even with the email gone from the id_token.
	state, nonce = start()
	iss.SetClaims(map[string]any{"sub": "muono-user-7", "nonce": nonce})
	rec = callback(state)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	users, err = st.ListUsers(ctx, &user.Query{AppID: appID})
	require.NoError(t, err)
	assert.Len(t, users.Users, 1)

	// An id_token minted for another flow's nonce is refused.
	state, _ = start()
	iss.SetClaims(map[string]any{"sub": "muono-user-7", "nonce": "someone-elses-nonce"})
	assert.Equal(t, http.StatusBadRequest, callback(state).Code)

	// So is one the issuer did not sign.
	state, nonce = start()
	iss.SetClaims(map[string]any{"sub": "muono-user-7", "nonce": nonce, "aud": "another-client"})
	assert.NotEqual(t, http.StatusOK, callback(state).Code)

	// Discovery ran once for all of the above.
	assert.Equal(t, int32(1), iss.DiscoveryHits.Load())

	// Another app does not see the provider.
	other := id.NewAppID()
	cfgOther := eng.ClientConfig(ctx, other)
	if cfgOther.Social != nil {
		for _, p := range cfgOther.Social.Providers {
			assert.NotEqual(t, "muono", p.ID)
		}
	}
}
