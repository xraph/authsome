package dpop_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/internal/jwkutil"
)

// mintProof builds a signed DPoP proof, always under ES256, the only alg
// these tests need to exercise the parse path. Fields left empty are omitted,
// so a test can produce a proof missing exactly one claim.
func mintProof(t *testing.T, key *ecdsa.PrivateKey, typ string, claims map[string]any) string {
	t.Helper()

	const alg = "ES256"

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	// A proof's jwk carries no use or alg member; strip them so the
	// thumbprint the client computes matches the one the server computes.
	j.Use, j.ALG = "", ""

	header := map[string]any{"typ": typ, "alg": alg, "jwk": j}
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

func testKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

func validClaims() map[string]any {
	return map[string]any{
		"jti": "proof-1",
		"htm": "POST",
		"htu": "https://auth.example.com/v1/oauth/token",
		"iat": time.Now().Unix(),
	}
}

func TestParse_Valid(t *testing.T) {
	key := testKey(t)
	p, err := dpop.Parse(mintProof(t, key, "dpop+jwt", validClaims()))
	require.NoError(t, err)

	assert.Equal(t, "ES256", p.Alg)
	assert.Equal(t, "POST", p.HTM)
	assert.Equal(t, "proof-1", p.JTI)
	assert.NotEmpty(t, p.JKT)
}

// TestParse_RejectsWrongTyp is the algorithm-confusion firewall. A plain JWT
// presented as a proof must not be accepted merely because it verifies.
func TestParse_RejectsWrongTyp(t *testing.T) {
	key := testKey(t)
	_, err := dpop.Parse(mintProof(t, key, "JWT", validClaims()))
	assert.ErrorIs(t, err, dpop.ErrMalformedProof)
}

// TestParse_RejectsSymmetricAndNone verifies no symmetric algorithm or "none"
// survives. With a symmetric alg the jwk header would be attacker-supplied key
// material used to verify the attacker's own signature.
func TestParse_RejectsSymmetricAndNone(t *testing.T) {
	for _, alg := range []string{"HS256", "HS384", "HS512", "none", "NONE", ""} {
		t.Run(alg, func(t *testing.T) {
			key := testKey(t)
			raw := mintProof(t, key, "dpop+jwt", validClaims())
			parts := strings.Split(raw, ".")
			hb, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": alg, "jwk": map[string]string{"kty": "oct", "k": "AAAA"}})
			require.NoError(t, err)
			parts[0] = base64.RawURLEncoding.EncodeToString(hb)

			_, err = dpop.Parse(strings.Join(parts, "."))
			assert.Error(t, err)
		})
	}
}

func TestParse_RejectsTamperedSignature(t *testing.T) {
	key := testKey(t)
	raw := mintProof(t, key, "dpop+jwt", validClaims())
	parts := strings.Split(raw, ".")
	claims := validClaims()
	claims["jti"] = "swapped"
	cb, err := json.Marshal(claims)
	require.NoError(t, err)
	parts[1] = base64.RawURLEncoding.EncodeToString(cb)

	_, err = dpop.Parse(strings.Join(parts, "."))
	assert.ErrorIs(t, err, dpop.ErrBadSignature)
}

func TestParse_RejectsOversized(t *testing.T) {
	_, err := dpop.Parse(strings.Repeat("a", dpop.MaxProofBytes+1))
	assert.ErrorIs(t, err, dpop.ErrMalformedProof)
}

func TestParse_RequiresMandatoryClaims(t *testing.T) {
	for _, missing := range []string{"jti", "htm", "htu", "iat"} {
		t.Run(missing, func(t *testing.T) {
			key := testKey(t)
			claims := validClaims()
			delete(claims, missing)
			_, err := dpop.Parse(mintProof(t, key, "dpop+jwt", claims))
			assert.ErrorIs(t, err, dpop.ErrMalformedProof)
		})
	}
}

// TestParse_RejectsPrivateKeyInJWK stops a proof from carrying private members
// in its jwk header.
func TestParse_RejectsPrivateKeyInJWK(t *testing.T) {
	key := testKey(t)
	raw := mintProof(t, key, "dpop+jwt", validClaims())
	parts := strings.Split(raw, ".")

	j, err := jwkutil.Encode(&key.PublicKey, "", "")
	require.NoError(t, err)
	poisoned := map[string]any{
		"kty": j.KTY, "crv": j.CRV, "x": j.X, "y": j.Y, "d": "c3RvbGVu",
	}
	hb, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": poisoned})
	require.NoError(t, err)
	parts[0] = base64.RawURLEncoding.EncodeToString(hb)

	_, err = dpop.Parse(strings.Join(parts, "."))
	assert.Error(t, err)
}
