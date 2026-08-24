package jwkutil_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/internal/jwkutil"
)

// TestThumbprint_RFC7638Vector checks our thumbprint against the worked
// example in RFC 7638 Section 3.1. An external oracle, so a bug in our
// canonicalisation cannot agree with itself and pass.
func TestThumbprint_RFC7638Vector(t *testing.T) {
	// n and want copied verbatim from RFC 7638 Section 3.1.
	const n = "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw"
	const want = "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"

	j := &jwkutil.JWK{KTY: "RSA", N: n, E: "AQAB"}
	got, err := jwkutil.Thumbprint(j)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// TestThumbprint_IgnoresOptionalMembers verifies that kid, use and alg do not
// change the thumbprint. RFC 7638 hashes only the required members, so a key
// that gains a kid must keep the same jkt or every bound token breaks.
func TestThumbprint_IgnoresOptionalMembers(t *testing.T) {
	base := &jwkutil.JWK{KTY: "RSA", N: "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw", E: "AQAB"}
	decorated := &jwkutil.JWK{KTY: "RSA", N: base.N, E: "AQAB", KID: "abc", Use: "sig", ALG: "RS256"}

	a, err := jwkutil.Thumbprint(base)
	require.NoError(t, err)
	b, err := jwkutil.Thumbprint(decorated)
	require.NoError(t, err)
	assert.Equal(t, a, b)
}

// TestParseJSON_RejectsPrivateMembers is the important one. A proof whose jwk
// header smuggles a private member must be refused outright, never silently
// parsed as though only the public half were present.
func TestParseJSON_RejectsPrivateMembers(t *testing.T) {
	cases := []string{"d", "p", "q", "dp", "dq", "qi", "k", "oth"}
	for _, member := range cases {
		t.Run(member, func(t *testing.T) {
			raw := json.RawMessage(`{"kty":"EC","crv":"P-256","x":"AA","y":"BB","` + member + `":"leak"}`)
			_, _, err := jwkutil.ParseJSON(raw)
			assert.ErrorIs(t, err, jwkutil.ErrPrivateKeyMaterial)
		})
	}
}

// TestParseJSON_RejectsOffCurvePoint guards against an attacker supplying a
// point that is not on P-256. Go's elliptic package will happily hold such a
// point in a struct; only an explicit IsOnCurve check refuses it.
func TestParseJSON_RejectsOffCurvePoint(t *testing.T) {
	raw := json.RawMessage(`{"kty":"EC","crv":"P-256",` +
		`"x":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",` +
		`"y":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`)
	_, _, err := jwkutil.ParseJSON(raw)
	assert.Error(t, err)
}

// TestParseJSON_RejectsWeakRSA refuses a modulus below 2048 bits.
func TestParseJSON_RejectsWeakRSA(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)
	j, err := jwkutil.Encode(&key.PublicKey, "", "RS256")
	require.NoError(t, err)
	raw, err := json.Marshal(j)
	require.NoError(t, err)

	_, _, err = jwkutil.ParseJSON(raw)
	assert.ErrorIs(t, err, jwkutil.ErrWeakKey)
}

// TestRoundTrip proves Encode and ParseJSON agree for every supported key
// type. Testing them as a pair catches a coordinate padding bug that testing
// either alone would miss.
func TestRoundTrip(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	edPub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cases := []struct {
		name string
		pub  crypto.PublicKey
		alg  string
	}{
		{"ES256", &ecKey.PublicKey, "ES256"},
		{"RS256", &rsaKey.PublicKey, "RS256"},
		{"EdDSA", edPub, "EdDSA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			j, err := jwkutil.Encode(tc.pub, "", tc.alg)
			require.NoError(t, err)
			raw, err := json.Marshal(j)
			require.NoError(t, err)

			got, _, err := jwkutil.ParseJSON(raw)
			require.NoError(t, err)
			assert.Equal(t, tc.pub, got)
		})
	}
}
