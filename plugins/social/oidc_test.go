package social_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/xraph/authsome/plugins/social"
	"github.com/xraph/authsome/plugins/social/socialtest"
)

func newOIDC(t *testing.T, iss *socialtest.Issuer, scopes ...string) social.Provider {
	t.Helper()
	prov, err := social.NewOIDCProvider(context.Background(), social.OIDCConfig{
		ProviderConfig: social.ProviderConfig{ClientID: iss.ClientID, ClientSecret: "s", Scopes: scopes},
		Name:           "muono",
		Issuer:         iss.URL,
		HTTPClient:     iss.Client(),
	})
	require.NoError(t, err)
	return prov
}

func tokenWith(idToken string) *oauth2.Token {
	tok := &oauth2.Token{AccessToken: "issuer-access-token", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	if idToken == "" {
		return tok
	}
	return tok.WithExtra(map[string]any{"id_token": idToken})
}

func TestOIDCProvider_Config(t *testing.T) {
	iss := socialtest.NewIssuer(t, "client-1")

	prov := newOIDC(t, iss)
	assert.Equal(t, "muono", prov.Name())
	cfg := prov.OAuth2Config()
	assert.Equal(t, iss.URL+"/authorize", cfg.Endpoint.AuthURL)
	assert.Equal(t, iss.URL+"/token", cfg.Endpoint.TokenURL)
	assert.Equal(t, []string{"openid", "email", "profile"}, cfg.Scopes)

	// openid is required for an id_token, so it is added when left out.
	assert.Equal(t, []string{"openid", "email"}, newOIDC(t, iss, "email").OAuth2Config().Scopes)
	assert.Equal(t, []string{"email", "openid"}, newOIDC(t, iss, "email", "openid").OAuth2Config().Scopes)
}

func TestNewOIDCProvider_Errors(t *testing.T) {
	iss := socialtest.NewIssuer(t, "client-1")
	_, err := social.NewOIDCProvider(context.Background(), social.OIDCConfig{Name: "google", Issuer: iss.URL, HTTPClient: iss.Client()})
	assert.ErrorContains(t, err, "built-in")

	_, err = social.NewOIDCProvider(context.Background(), social.OIDCConfig{Name: "muono", Issuer: iss.URL + "/nope", HTTPClient: iss.Client()})
	assert.ErrorContains(t, err, "discover")

	// The default client will not call a loopback address, which is where
	// every httptest server lives.
	_, err = social.NewOIDCProvider(context.Background(), social.OIDCConfig{Name: "muono", Issuer: iss.URL})
	assert.ErrorContains(t, err, "private, loopback or link-local")
}

func TestOIDCProvider_FetchUser(t *testing.T) {
	iss := socialtest.NewIssuer(t, "client-1")
	prov := newOIDC(t, iss)
	ctx := context.Background()

	t.Run("profile from the id_token", func(t *testing.T) {
		u, err := prov.FetchUser(ctx, tokenWith(iss.Sign(map[string]any{
			"sub": "u-42", "email": "ada@example.com", "email_verified": true,
			"given_name": "Ada", "family_name": "Lovelace", "picture": "https://example.com/a.png",
		})))
		require.NoError(t, err)
		assert.Equal(t, &social.ProviderUser{
			ProviderUserID: "u-42", Email: "ada@example.com", EmailVerified: true,
			FirstName: "Ada", LastName: "Lovelace", AvatarURL: "https://example.com/a.png",
		}, u)
	})

	t.Run("name split and string email_verified", func(t *testing.T) {
		u, err := prov.FetchUser(ctx, tokenWith(iss.Sign(map[string]any{
			"email": "g@example.com", "email_verified": "true", "name": "Grace Brewster Hopper",
		})))
		require.NoError(t, err)
		assert.True(t, u.EmailVerified)
		assert.Equal(t, "Grace", u.FirstName)
		assert.Equal(t, "Brewster Hopper", u.LastName)
	})

	t.Run("unrecognised email_verified is unverified", func(t *testing.T) {
		u, err := prov.FetchUser(ctx, tokenWith(iss.Sign(map[string]any{"email": "x@example.com", "email_verified": 1})))
		require.NoError(t, err)
		assert.False(t, u.EmailVerified)
	})

	t.Run("userinfo fills a missing email", func(t *testing.T) {
		iss.SetUserInfo(map[string]any{
			"sub": "issuer-user-1", "email": "info@example.com", "email_verified": true,
			"given_name": "Info", "family_name": "User", "picture": "https://example.com/i.png",
		})
		t.Cleanup(func() { iss.SetUserInfo(nil) })
		u, err := prov.FetchUser(ctx, tokenWith(iss.Sign(nil)))
		require.NoError(t, err)
		assert.Equal(t, "info@example.com", u.Email)
		assert.True(t, u.EmailVerified)
		assert.Equal(t, "Info", u.FirstName)
		assert.Equal(t, "https://example.com/i.png", u.AvatarURL)
	})

	t.Run("userinfo for another subject is ignored", func(t *testing.T) {
		iss.SetUserInfo(map[string]any{"sub": "someone-else", "email": "other@example.com", "email_verified": true})
		t.Cleanup(func() { iss.SetUserInfo(nil) })
		u, err := prov.FetchUser(ctx, tokenWith(iss.Sign(nil)))
		require.NoError(t, err)
		assert.Empty(t, u.Email)
	})

	t.Run("failed userinfo leaves the email empty", func(t *testing.T) {
		u, err := prov.FetchUser(ctx, tokenWith(iss.Sign(nil)))
		require.NoError(t, err)
		assert.Empty(t, u.Email)
	})

	for name, tok := range map[string]*oauth2.Token{
		"no id_token":        tokenWith(""),
		"unpublished key":    tokenWith(iss.SignWithOtherKey(nil)),
		"wrong audience":     tokenWith(iss.Sign(map[string]any{"aud": "someone-else"})),
		"wrong issuer":       tokenWith(iss.Sign(map[string]any{"iss": "https://evil.example.com"})),
		"expired":            tokenWith(iss.Sign(map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})),
		"malformed id_token": tokenWith("not.a.jwt"),
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := prov.FetchUser(ctx, tok)
			assert.Error(t, err)
		})
	}
}

func TestOIDCProvider_UsesItsOwnClient(t *testing.T) {
	iss := socialtest.NewIssuer(t, "client-1")
	prov := newOIDC(t, iss)
	hc, ok := prov.(interface{ HTTPClient() *http.Client })
	require.True(t, ok)
	assert.Same(t, iss.Client(), hc.HTTPClient())
}

func TestValidateOIDCNameAndIssuer(t *testing.T) {
	for _, ok := range []string{"muono", "a", "my-idp-2"} {
		assert.NoError(t, social.ValidateOIDCName(ok), ok)
	}
	for _, bad := range []string{"", "Muono", "-x", "has space", "oidc", "github", "this-name-is-far-too-long-for-an-id"} {
		assert.Error(t, social.ValidateOIDCName(bad), bad)
	}
	assert.NoError(t, social.ValidateOIDCIssuer("https://id.example.com/realms/main"))
	for _, bad := range []string{"http://id.example.com", "https://", "id.example.com", "https://id.example.com?x=1",
		"https://id.example.com#f", "https://u:p@id.example.com", "://"} {
		assert.Error(t, social.ValidateOIDCIssuer(bad), bad)
	}
}
