// Package dpoptest mints DPoP proofs for tests. It lives under internal/ so
// the enforcement tests in middleware, extension and authprovider can share
// one minting path instead of copying the JWS assembly into every package
// that needs to present a proof.
package dpoptest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/internal/jwkutil"
)

// Key generates an ES256 key pair for a test client to hold.
func Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

// MintProof builds a signed DPoP proof. Claims left out of claims are simply
// absent, so a test can produce a proof missing exactly one claim.
func MintProof(t *testing.T, key *ecdsa.PrivateKey, alg string, claims map[string]any) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	// A proof's jwk carries no use or alg member; strip them so the
	// thumbprint the server computes matches the one the client computed.
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": "dpop+jwt", "alg": alg, "jwk": j}
	hb, err := json.Marshal(header)
	require.NoError(t, err)
	cb, err := json.Marshal(claims)
	require.NoError(t, err)

	signing := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(cb)

	method := jwt.GetSigningMethod(alg)
	require.NotNil(t, method, "unknown alg %q", alg)
	sig, err := method.Sign(signing, key)
	require.NoError(t, err)

	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// Thumbprint computes the RFC 7638 thumbprint a proof minted with key will
// carry, so a test can bind a session to it up front.
func Thumbprint(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	j.Use, j.ALG = "", ""
	jkt, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	return jkt
}

// ValidClaims builds the mandatory claim set for a proof covering method
// against htu. Callers add ath themselves, since a proof presented at a
// protected resource needs it and a proof presented at a token endpoint
// does not.
func ValidClaims(method, htu string) map[string]any {
	if method == "" {
		method = http.MethodGet
	}
	return map[string]any{
		"jti": "proof-" + htu,
		"htm": method,
		"htu": htu,
		"iat": time.Now().Unix(),
	}
}
