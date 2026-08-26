package jwksclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The EC branch of publicKey assembles the point and hands it to
// ecdsa.ParseUncompressedPublicKey rather than assigning X and Y onto a
// struct literal. That is not a style preference. A struct literal accepts a
// point that is not on the curve and accepts the point at infinity; parsing
// rejects both. These coordinates arrive from someone else's JWKS endpoint,
// and a key we accept is a key we will verify session-revocation tokens
// with, so the validation is the point of the code.
//
// Nothing exercised it until this file. Deleting the parse check left the
// whole suite green.

// ecCoords returns the raw X and Y coordinates of a public key, taken from
// the uncompressed SEC 1 encoding (0x04 || X || Y) so the test never touches
// the deprecated X and Y fields.
func ecCoords(t *testing.T, pub *ecdsa.PublicKey) (x, y []byte) {
	t.Helper()
	b, err := pub.Bytes()
	require.NoError(t, err)
	require.Equal(t, byte(4), b[0], "expected an uncompressed point")
	n := (len(b) - 1) / 2
	return b[1 : 1+n], b[1+n:]
}

func ecKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return k
}

const ecTestKID = "kid-ec"

func ecJWK(x, y []byte) jwk {
	return jwk{
		KTY: "EC", KID: ecTestKID, CRV: "P-256",
		X: base64.RawURLEncoding.EncodeToString(x),
		Y: base64.RawURLEncoding.EncodeToString(y),
	}
}

// The positive control. Without it, an implementation that refused every key
// would satisfy all the rejection tests below.
func TestPublicKey_ECValidKeyParsesAndVerifies(t *testing.T) {
	priv := ecKey(t)
	x, y := ecCoords(t, &priv.PublicKey)

	got, err := ecJWK(x, y).publicKey()
	require.NoError(t, err)

	pub, ok := got.(*ecdsa.PublicKey)
	require.True(t, ok, "expected an *ecdsa.PublicKey")

	// Prove it is the right key, not merely a non-nil one.
	digest := sha256.Sum256([]byte("a security event token"))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	require.NoError(t, err)
	assert.True(t, ecdsa.VerifyASN1(pub, digest[:], sig),
		"the parsed key must verify a signature from its own private half")
}

// A point that is not on the curve. Flipping one bit of Y moves it off P-256,
// because for a given X only Y and its negation lie on the curve.
func TestPublicKey_ECRejectsPointOffCurve(t *testing.T) {
	priv := ecKey(t)
	x, y := ecCoords(t, &priv.PublicKey)

	offCurve := make([]byte, len(y))
	copy(offCurve, y)
	offCurve[len(offCurve)-1] ^= 0x01

	_, err := ecJWK(x, offCurve).publicKey()
	require.Error(t, err,
		"a point that is not on the curve must be refused, not turned into a usable key")
	assert.Contains(t, err.Error(), "P-256")
}

// The point at infinity encodes as all-zero coordinates. A struct literal
// accepts it; parsing does not.
func TestPublicKey_ECRejectsPointAtInfinity(t *testing.T) {
	zero := make([]byte, 32)

	_, err := ecJWK(zero, zero).publicKey()
	require.Error(t, err, "the point at infinity must be refused")
}

// A coordinate wider than the curve is rejected before the point is even
// assembled, so an oversized value cannot shift the bytes either side of it.
func TestPublicKey_ECRejectsOversizedCoordinate(t *testing.T) {
	priv := ecKey(t)
	x, y := ecCoords(t, &priv.PublicKey)

	tooLong := append([]byte{0x01}, x...) // 33 bytes on a 32 byte curve

	_, err := ecJWK(tooLong, y).publicKey()
	require.Error(t, err, "a coordinate longer than the curve must be refused")
	assert.Contains(t, err.Error(), "longer than curve")
}

// An unsupported curve name is refused rather than guessed at.
func TestPublicKey_ECRejectsUnknownCurve(t *testing.T) {
	priv := ecKey(t)
	x, y := ecCoords(t, &priv.PublicKey)

	k := ecJWK(x, y)
	k.CRV = "P-224"

	_, err := k.publicKey()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported curve")
}

// End to end: an off-curve key served by a real endpoint must never come back
// out of Key. The fetch loop skips a key it cannot use, so the caller sees a
// miss rather than a usable key built from coordinates nobody validated.
func TestKey_ECOffCurveKeyNeverBecomesUsable(t *testing.T) {
	priv := ecKey(t)
	x, y := ecCoords(t, &priv.PublicKey)
	offCurve := make([]byte, len(y))
	copy(offCurve, y)
	offCurve[len(offCurve)-1] ^= 0x01

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"keys":[{"kty":"EC","kid":"kid-ec","crv":"P-256","x":%q,"y":%q}]}`,
			base64.RawURLEncoding.EncodeToString(x),
			base64.RawURLEncoding.EncodeToString(offCurve))
	}))
	defer srv.Close()

	c := New(testOptions(srv))
	got, err := c.Key(context.Background(), srv.URL, "kid-ec")
	require.Error(t, err, "an off-curve key must not be handed to a caller")
	assert.Nil(t, got)
}
