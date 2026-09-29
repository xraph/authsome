package sso

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A fake OpenID provider: discovery, a token endpoint that insists on the
// PKCE verifier and echoes the nonce it is told to, and userinfo.
type fakeOP struct {
	srv        *httptest.Server
	discovery  atomic.Int32
	nonceToUse string
	sawVerify  string
}

func unsignedIDToken(claims map[string]any) string {
	payload, _ := json.Marshal(claims)
	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func newFakeOP(t *testing.T) *fakeOP {
	t.Helper()
	op := &fakeOP{}
	// Routes are matched by suffix: each test uses its own issuer path under
	// the server so the discovery cache keys them apart.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/.well-known/openid-configuration"):
			op.discovery.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"authorization_endpoint": op.srv.URL + "/authz",
				"token_endpoint":         op.srv.URL + "/token",
				"userinfo_endpoint":      op.srv.URL + "/me",
			})
		case r.URL.Path == "/token":
			require.NoError(t, r.ParseForm())
			op.sawVerify = r.Form.Get("code_verifier")
			if op.sawVerify == "" {
				http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "at", "token_type": "Bearer", "expires_in": 3600,
				"id_token": unsignedIDToken(map[string]any{"sub": "sub-1", "nonce": op.nonceToUse}),
			})
		case r.URL.Path == "/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": "sub-1", "email": "a@corp.example", "email_verified": "true", "name": "A"})
		default:
			http.NotFound(w, r)
		}
	})
	op.srv = httptest.NewServer(handler)
	t.Cleanup(op.srv.Close)
	return op
}

func TestOIDC_LoginCarriesNonceAndChallenge(t *testing.T) {
	op := newFakeOP(t)
	p := NewOIDCProvider(OIDCConfig{Name: "fake", Issuer: op.srv.URL + "/" + t.Name(), ClientID: "c", RedirectURL: "https://rp.example/cb", HTTPClient: op.srv.Client()})
	pk, ok := p.(pkceProvider)
	require.True(t, ok)
	loginURL, err := pk.LoginURLWithPKCE("state-1", "nonce-1", "verifier-verifier-verifier-verifier-1234567890")
	require.NoError(t, err)
	u, err := url.Parse(loginURL)
	require.NoError(t, err)
	q := u.Query()
	assert.Equal(t, "nonce-1", q.Get("nonce"))
	assert.Equal(t, "S256", q.Get("code_challenge_method"))
	assert.NotEmpty(t, q.Get("code_challenge"))
	assert.NotContains(t, loginURL, "verifier-verifier", "the verifier never leaves the server")
	_, err = pk.LoginURLWithPKCE("s", "", "v")
	assert.Error(t, err, "a login without a nonce is refused")
}

func TestOIDC_CallbackPresentsVerifierAndChecksNonce(t *testing.T) {
	op := newFakeOP(t)
	op.nonceToUse = "nonce-ok"
	p := NewOIDCProvider(OIDCConfig{Name: "fake", Issuer: op.srv.URL + "/" + t.Name(), ClientID: "c", RedirectURL: "https://rp.example/cb", HTTPClient: op.srv.Client()})

	u, err := p.HandleCallback(context.Background(), map[string]string{"code": "code", "nonce": "nonce-ok", "code_verifier": "the-verifier"})
	require.NoError(t, err)
	assert.Equal(t, "the-verifier", op.sawVerify, "the exchange presents the verifier")
	assert.Equal(t, "sub-1", u.ProviderUserID)
	require.NotNil(t, u.EmailVerified)
	assert.True(t, *u.EmailVerified, "a string email_verified is honoured")

	_, err = p.HandleCallback(context.Background(), map[string]string{"code": "code", "nonce": "other-nonce", "code_verifier": "the-verifier"})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "nonce"), "a mismatched nonce refuses the login: %v", err)

	_, err = p.HandleCallback(context.Background(), map[string]string{"code": "code", "nonce": "nonce-ok"})
	require.Error(t, err, "without the verifier the provider refuses the exchange")
}

func TestOIDC_DiscoveryIsCachedPerIssuer(t *testing.T) {
	op := newFakeOP(t)
	op.nonceToUse = "n"
	cfg := OIDCConfig{Name: "fake", Issuer: op.srv.URL + "/" + t.Name(), ClientID: "c", RedirectURL: "https://rp.example/cb", HTTPClient: op.srv.Client()}
	for range 3 {
		p := NewOIDCProvider(cfg)
		_, err := p.LoginURL("s")
		require.NoError(t, err)
		_, err = p.HandleCallback(context.Background(), map[string]string{"code": "code", "nonce": "n", "code_verifier": "v"})
		require.NoError(t, err)
	}
	assert.EqualValues(t, 1, op.discovery.Load(), "one fetch serves every provider built for the issuer")
}

func TestOIDC_HardenedClientRefusesLocalAndPlainIssuers(t *testing.T) {
	p := NewOIDCProvider(OIDCConfig{Name: "prod", Issuer: "http://127.0.0.1:9/" + t.Name(), ClientID: "c"})
	_, err := p.(*oidcProvider).discover()
	require.Error(t, err, "plain http discovery is refused before any dial")
	p = NewOIDCProvider(OIDCConfig{Name: "prod", Issuer: "https://127.0.0.1:9/" + t.Name(), ClientID: "c"})
	_, err = p.(*oidcProvider).discover()
	require.Error(t, err, "a loopback issuer is refused")
}
