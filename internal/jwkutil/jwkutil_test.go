package jwkutil_test

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
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
// point that is not on P-256. A struct literal will happily hold such a point,
// which is why parseEC builds the key through ParseUncompressedPublicKey: the
// on-curve check lives inside the constructor and cannot be skipped.
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
// type: encoding a key and parsing the result back gives the same
// crypto.PublicKey the encoder started from.
//
// This does NOT catch a coordinate padding regression: the
// comparison is on decoded *ecdsa.PublicKey structs, whose X and Y are
// big.Int, and big.Int.SetBytes normalises away any leading zero byte on the
// way in. An unpadded coordinate and a correctly padded one decode to the
// same big.Int, so this test passes either way. See
// TestEncode_PadsShortCoordinate, which asserts on the encoded wire form
// instead and is the one that actually catches that regression.
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

// TestEncode_PadsShortCoordinate exercises the case TestRoundTrip cannot: an
// EC coordinate whose raw big-endian bytes are shorter than the curve's field
// size because its top byte is zero.
//
// k = 43 is a fixed scalar, not one found by generating keys until one
// happened to qualify: multiplying the P-256 base point by it is deterministic,
// so this test cannot flake. It was chosen by an offline search (multiply the
// base point by 1, 2, 3, ... and check the coordinates) for a scalar whose
// public key has a Y coordinate under 2^248, i.e. one whose 32-byte big-endian
// form starts with a zero byte. k = 43 is the first such scalar; its Y is 31
// raw bytes, not 32. The assertion below checks BitLen at test time too, so a
// change to the elliptic implementation that shifted the result would fail
// loudly here rather than silently stop testing anything.
//
// Unlike TestRoundTrip, this asserts on the encoded wire form: the raw byte
// length behind the base64url x and y in the resulting JWK. That is what a
// third-party verifier actually parses, and what Encode must keep at exactly
// 32 bytes for P-256 regardless of leading zeros.
func TestEncode_PadsShortCoordinate(t *testing.T) {
	// k=43 on P-256 lands on a point whose Y is short enough to need padding,
	// which is the whole reason this fixture exists. Deriving it through
	// crypto/ecdh rather than ScalarBaseMult gets the same point without the
	// deprecated low-level call, and Bytes() hands back SEC 1 uncompressed
	// form: 0x04, then X and Y at fixed width.
	scalar := make([]byte, 32)
	scalar[31] = 43

	priv, err := ecdh.P256().NewPrivateKey(scalar)
	require.NoError(t, err)

	point := priv.PublicKey().Bytes()
	require.Len(t, point, 65)

	yBits := new(big.Int).SetBytes(point[33:]).BitLen()
	require.LessOrEqualf(t, yBits, 248,
		"fixture scalar k=43 no longer produces a short Y coordinate (BitLen=%d); pick a new scalar", yBits)

	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
	require.NoError(t, err)
	j, err := jwkutil.Encode(pub, "", "ES256")
	require.NoError(t, err)

	xBytes, err := base64.RawURLEncoding.DecodeString(j.X)
	require.NoError(t, err)
	yBytes, err := base64.RawURLEncoding.DecodeString(j.Y)
	require.NoError(t, err)

	assert.Len(t, xBytes, 32, "encoded x must be exactly the P-256 field size")
	assert.Len(t, yBytes, 32, "encoded y must be exactly the P-256 field size, even with a leading zero byte")
}

// TestThumbprint_QuoteInMemberCannotCollide is the attack the canonicalization
// has to survive. A JWK member is base64url in any honest key, so a double
// quote never appears by accident, but nothing stops a caller sending one.
//
// If the canonical JSON were assembled by concatenation, a crafted x could
// close its own string and open the next member, so two different keys could
// hash to one thumbprint. A thumbprint is a DPoP key binding, so a collision
// there means a token bound to one key is accepted for another.
func TestThumbprint_QuoteInMemberCannotCollide(t *testing.T) {
	crafted, err := jwkutil.Thumbprint(&jwkutil.JWK{
		KTY: "EC", CRV: "P-256", X: `a","y":"b`, Y: "c",
	})
	require.NoError(t, err)

	honest, err := jwkutil.Thumbprint(&jwkutil.JWK{
		KTY: "EC", CRV: "P-256", X: "a", Y: "b",
	})
	require.NoError(t, err)

	assert.NotEqual(t, honest, crafted,
		"a member containing a quote must not be able to forge another key's thumbprint")
}
