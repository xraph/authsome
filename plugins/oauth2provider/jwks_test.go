package oauth2provider_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/tokenformat"
)

type jwksDoc struct {
	Keys []map[string]any `json:"keys"`
}

// jwksFixture boots the plugin against a real engine so OnInit builds the
// key manager. The token encryptor lets the deployment issuer mint a managed
// key.
func jwksFixture(t *testing.T, cfg oauth2provider.Config, opts ...authsome.Option) forge.Router {
	t.Helper()
	enc, err := bridge.NewAESGCMEncryptor(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	eng := secutil.NewTestEngine(t, append([]authsome.Option{authsome.WithTokenEncryptor(enc)}, opts...)...)
	if cfg.Issuer == "" {
		cfg.Issuer = "https://auth.example.com"
	}
	p := oauth2provider.New(cfg)
	p.SetOAuth2Store(oauth2provider.NewMemoryStore())
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return mux
}

func getPath(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func fetchJWKS(t *testing.T, h http.Handler) jwksDoc {
	t.Helper()
	rec := getPath(t, h, "/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var doc jwksDoc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	return doc
}

func kids(doc jwksDoc) []string {
	out := make([]string, 0, len(doc.Keys))
	for _, k := range doc.Keys {
		out = append(out, k["kid"].(string))
	}
	return out
}

func rsaJWTFormat(t *testing.T, kid string) *tokenformat.JWT {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	f, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodRS256, SigningKey: k, VerifyKey: &k.PublicKey, KeyID: kid,
	})
	require.NoError(t, err)
	return f
}

func TestJWKS_ServedAtAdvertisedURI(t *testing.T) {
	mux := jwksFixture(t, oauth2provider.Config{})

	var disc oauth2provider.DiscoveryResponse
	rec := getPath(t, mux, "/.well-known/openid-configuration")
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &disc))
	assert.Equal(t, "https://auth.example.com/v1/oauth/jwks", disc.JWKSURI)

	u, err := url.Parse(disc.JWKSURI)
	require.NoError(t, err)
	rec = getPath(t, mux, u.Path)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var doc jwksDoc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	require.Len(t, doc.Keys, 1)
	assert.Equal(t, "RS256", doc.Keys[0]["alg"])
	assert.Equal(t, "sig", doc.Keys[0]["use"])
}

func TestJWKS_IncludesAsymmetricAccessTokenKeys(t *testing.T) {
	mux := jwksFixture(t, oauth2provider.Config{},
		authsome.WithJWTFormat("app-a", rsaJWTFormat(t, "at-key")))

	got := kids(fetchJWKS(t, mux))
	require.Len(t, got, 2)
	assert.Contains(t, got, "at-key")
}

func TestJWKS_NeverPublishesHMAC(t *testing.T) {
	hmacFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256, SigningKey: []byte("0123456789abcdef0123456789abcdef"), KeyID: "hmac-key",
	})
	require.NoError(t, err)
	mux := jwksFixture(t, oauth2provider.Config{}, authsome.WithJWTFormat("app-a", hmacFmt))

	doc := fetchJWKS(t, mux)
	assert.Len(t, doc.Keys, 1)
	assert.NotContains(t, kids(doc), "hmac-key")
}

func TestJWKS_CacheControl(t *testing.T) {
	mux := jwksFixture(t, oauth2provider.Config{})
	rec := getPath(t, mux, "/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public, max-age=300", rec.Header().Get("Cache-Control"))
}

func TestJWKS_DeduplicatesByKID(t *testing.T) {
	f := rsaJWTFormat(t, "same")
	mux := jwksFixture(t, oauth2provider.Config{},
		authsome.WithJWTFormat("app-a", f), authsome.WithJWTFormat("app-b", f))
	got := kids(fetchJWKS(t, mux))
	assert.Len(t, got, 2, "the managed key plus one copy of the shared access-token key")
}

func discoveryDoc(t *testing.T, mux http.Handler) map[string]any {
	t.Helper()
	rec := getPath(t, mux, "/.well-known/openid-configuration")
	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

func TestDiscovery_AdvertisesLoadedAlgs(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(k)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	mux := jwksFixture(t, oauth2provider.Config{IDTokenSigningKey: pemKey})
	assert.Equal(t, []any{"ES256"}, discoveryDoc(t, mux)["id_token_signing_alg_values_supported"])
}

func TestDiscovery_FallsBackToRS256WithoutAKey(t *testing.T) {
	// No encryptor and no operator key: nothing can sign, but an empty list
	// would make clients refuse discovery.
	eng := secutil.NewTestEngine(t)
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(oauth2provider.NewMemoryStore())
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	assert.Equal(t, []any{"RS256"}, discoveryDoc(t, mux)["id_token_signing_alg_values_supported"])
}

func TestDiscovery_ClaimsAndResponseModes(t *testing.T) {
	mux := jwksFixture(t, oauth2provider.Config{})
	got := discoveryDoc(t, mux)

	claims, ok := got["claims_supported"].([]any)
	require.True(t, ok)
	assert.Contains(t, claims, "sub")
	assert.Contains(t, claims, "email_verified")
	assert.Contains(t, claims, "nonce")
	assert.Equal(t, []any{"query"}, got["response_modes_supported"])
	assert.NotContains(t, got, "introspection_endpoint")
}

// failingKeyStore fails every signing key read and write while broken is
// set, the way a database outage would.
type failingKeyStore struct {
	oauth2provider.Store
	broken atomic.Bool
}

var errKeyStoreDown = errors.New("signing key store is down")

func (s *failingKeyStore) ListSigningKeys(ctx context.Context, scope string) ([]*oauth2provider.SigningKey, error) {
	if s.broken.Load() {
		return nil, errKeyStoreDown
	}
	return s.Store.ListSigningKeys(ctx, scope)
}

func (s *failingKeyStore) CreateSigningKey(ctx context.Context, k *oauth2provider.SigningKey) error {
	if s.broken.Load() {
		return errKeyStoreDown
	}
	return s.Store.CreateSigningKey(ctx, k)
}

// When the id_token keys can't be read the JWKS still serves the access
// token keys, uncached, and picks the id_token keys up again as soon as the
// store recovers.
func TestJWKS_StoreFailureServesAccessTokenKeys(t *testing.T) {
	enc, err := bridge.NewAESGCMEncryptor(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	eng := secutil.NewTestEngine(t, authsome.WithTokenEncryptor(enc),
		authsome.WithJWTFormat("app-a", rsaJWTFormat(t, "at-key")))
	st := &failingKeyStore{Store: oauth2provider.NewMemoryStore()}
	p := oauth2provider.New(oauth2provider.Config{Issuer: "https://auth.example.com"})
	p.SetOAuth2Store(st)
	require.NoError(t, p.OnInit(context.Background(), eng))
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	st.broken.Store(true)
	rec := getPath(t, mux, "/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"), "a partial key set must not be cached")
	var doc jwksDoc
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	assert.Equal(t, []string{"at-key"}, kids(doc))

	st.broken.Store(false)
	rec = getPath(t, mux, "/v1/oauth/jwks")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "public, max-age=300", rec.Header().Get("Cache-Control"))
	doc = jwksDoc{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	assert.Len(t, doc.Keys, 2, "the id_token key is back, so the degraded result was not memoized")
	assert.Contains(t, kids(doc), "at-key")
}
