package socialtest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Issuer is a minimal OpenID Connect issuer for tests: discovery, JWKS, a
// token endpoint that returns a signed id_token, and userinfo. It serves
// TLS, so use Client() for calls to it.
type Issuer struct {
	Server   *httptest.Server
	URL      string
	ClientID string

	key   *rsa.PrivateKey
	other *rsa.PrivateKey

	mu       sync.Mutex
	claims   map[string]any
	userInfo map[string]any

	// DiscoveryHits counts fetches of the discovery document.
	DiscoveryHits atomic.Int32
	// TokenRequests counts calls to the token endpoint.
	TokenRequests atomic.Int32
}

// NewIssuer starts an issuer whose tokens are for clientID. It stops when
// the test ends.
func NewIssuer(t *testing.T, clientID string) *Issuer {
	t.Helper()
	iss := &Issuer{ClientID: clientID, key: mustKey(t), other: mustKey(t)}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		iss.DiscoveryHits.Add(1)
		writeJSON(w, map[string]any{
			"issuer":                                iss.URL,
			"authorization_endpoint":                iss.URL + "/authorize",
			"token_endpoint":                        iss.URL + "/token",
			"jwks_uri":                              iss.URL + "/jwks",
			"userinfo_endpoint":                     iss.URL + "/userinfo",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		pub := iss.key.PublicKey
		writeJSON(w, map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "k1",
			"n": b64(pub.N.Bytes()), "e": b64(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		iss.TokenRequests.Add(1)
		iss.mu.Lock()
		claims := iss.claims
		iss.mu.Unlock()
		writeJSON(w, map[string]any{
			"access_token": "issuer-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     iss.Sign(claims),
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer issuer-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		iss.mu.Lock()
		info := iss.userInfo
		iss.mu.Unlock()
		if info == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, info)
	})
	iss.Server = httptest.NewTLSServer(mux)
	iss.URL = iss.Server.URL
	t.Cleanup(iss.Server.Close)
	return iss
}

// Client trusts the issuer's certificate.
func (i *Issuer) Client() *http.Client { return i.Server.Client() }

// SetClaims sets the claims the token endpoint's next id_tokens carry, on
// top of the defaults Sign adds.
func (i *Issuer) SetClaims(claims map[string]any) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.claims = claims
}

// SetUserInfo sets the userinfo response. Nil answers 404.
func (i *Issuer) SetUserInfo(info map[string]any) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.userInfo = info
}

// Sign returns an RS256 id_token with iss, aud, sub, iat and exp filled in
// unless claims sets them.
func (i *Issuer) Sign(claims map[string]any) string { return i.sign(i.key, claims) }

// SignWithOtherKey signs with a key the JWKS does not publish.
func (i *Issuer) SignWithOtherKey(claims map[string]any) string { return i.sign(i.other, claims) }

func (i *Issuer) sign(key *rsa.PrivateKey, claims map[string]any) string {
	now := time.Now()
	full := map[string]any{
		"iss": i.URL, "aud": i.ClientID, "sub": "issuer-user-1",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
	for k, v := range claims {
		full[k] = v
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "k1"}) //nolint:errcheck // fixed shape
	payload, _ := json.Marshal(full)                                                        //nolint:errcheck // test claims
	signing := b64(header) + "." + b64(payload)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	return signing + "." + b64(sig)
}

// AuthorizeURLParam reads one query parameter from an authorization URL
// the plugin built, such as nonce or client_id.
func AuthorizeURLParam(authURL, name string) string {
	u, err := url.Parse(authURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(name)
}

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v) //nolint:errcheck // test server
}
