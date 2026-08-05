package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"testing"
)

// RFC 7518 §6.2.1.2 requires an EC coordinate to be the full field size of the
// curve, left-padded with zeros. big.Int.Bytes() returns the minimal
// representation, so a coordinate whose leading byte is zero used to serialise
// one byte short — a JWK that strict verifiers reject.
//
// The failure is data-dependent (~1 key in 256 per coordinate), which is why
// these cases are constructed rather than generated: a random key almost
// always has a full-width coordinate and passes either way.
func TestECCoordinateIsFixedLength(t *testing.T) {
	cases := []struct {
		name    string
		bitSize int
		value   *big.Int
		wantLen int
	}{
		{"P-256 full width", 256, new(big.Int).Lsh(big.NewInt(1), 255), 32},
		{"P-256 one leading zero byte", 256, new(big.Int).Lsh(big.NewInt(1), 247), 32},
		{"P-256 many leading zeros", 256, big.NewInt(1), 32},
		{"P-256 zero", 256, big.NewInt(0), 32},
		{"P-384 one leading zero byte", 384, new(big.Int).Lsh(big.NewInt(1), 375), 48},
		{"P-521 odd field size", 521, big.NewInt(1), 66},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enc := ecCoordinate(c.value, c.bitSize)
			raw, err := base64.RawURLEncoding.DecodeString(enc)
			if err != nil {
				t.Fatalf("not valid base64url: %v", err)
			}
			if len(raw) != c.wantLen {
				t.Errorf("encoded to %d bytes, want %d — RFC 7518 requires the full field size",
					len(raw), c.wantLen)
			}
			// Padding must not change the value it represents.
			if got := new(big.Int).SetBytes(raw); got.Cmp(c.value) != 0 {
				t.Errorf("round-trip changed the value: got %s, want %s", got, c.value)
			}
		})
	}
}

// Real key material: every coordinate must come out at the curve's field
// size, whatever the generated key happens to be.
func TestGeneratedKeyCoordinateLengths(t *testing.T) {
	for _, curve := range []struct {
		name    string
		c       elliptic.Curve
		wantLen int
	}{
		{"P-256", elliptic.P256(), 32},
		{"P-384", elliptic.P384(), 48},
		{"P-521", elliptic.P521(), 66},
	} {
		t.Run(curve.name, func(t *testing.T) {
			bits := curve.c.Params().BitSize
			for i := 0; i < 50; i++ {
				priv, err := ecdsa.GenerateKey(curve.c, rand.Reader)
				if err != nil {
					t.Fatalf("generate: %v", err)
				}
				for label, v := range map[string]*big.Int{"x": priv.X, "y": priv.Y} {
					raw, err := base64.RawURLEncoding.DecodeString(ecCoordinate(v, bits))
					if err != nil {
						t.Fatalf("%s is not valid base64url: %v", label, err)
					}
					if len(raw) != curve.wantLen {
						t.Fatalf("%s encoded to %d bytes, want %d", label, len(raw), curve.wantLen)
					}
				}
			}
		})
	}
}
