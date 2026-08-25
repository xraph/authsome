// Package jwkutil converts between JSON Web Keys and Go public keys, and
// computes RFC 7638 thumbprints.
//
// Both directions live here on purpose. The encode direction serves the JWKS
// endpoint and the parse direction serves DPoP proof validation, and keeping
// them together makes the round trip testable as a single property.
package jwkutil

import (
	"crypto"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

// Sentinel errors.
var (
	ErrUnsupportedKey     = errors.New("jwkutil: unsupported key type")
	ErrPrivateKeyMaterial = errors.New("jwkutil: jwk contains private key material")
	ErrWeakKey            = errors.New("jwkutil: key below minimum strength")
)

// minRSABits is the smallest RSA modulus we accept. Below this the key is
// weak enough that accepting it would undermine the proof it signs.
const minRSABits = 2048

// JWK is a JSON Web Key. Only the public members are modelled; private
// members are detected during parsing and rejected.
type JWK struct {
	KTY string `json:"kty"`
	Use string `json:"use,omitempty"`
	KID string `json:"kid,omitempty"`
	ALG string `json:"alg,omitempty"`
	N   string `json:"n,omitempty"`   // RSA modulus
	E   string `json:"e,omitempty"`   // RSA exponent
	CRV string `json:"crv,omitempty"` // EC or OKP curve
	X   string `json:"x,omitempty"`   // EC x coordinate, or OKP public key
	Y   string `json:"y,omitempty"`   // EC y coordinate
}

// privateMembers are the JWK fields that only ever appear on a private key.
// A proof carrying any of them is malformed at best and an attempt to have us
// import an attacker-chosen private key at worst.
var privateMembers = []string{"d", "p", "q", "dp", "dq", "qi", "k", "oth"}

// Thumbprint returns the base64url-encoded SHA-256 RFC 7638 thumbprint.
//
// The hash covers only the required members, in lexicographic order, with no
// whitespace. Optional members such as kid must not affect the result, or a
// key that later gains a kid would stop matching tokens bound to it.
func Thumbprint(j *JWK) (string, error) {
	var canonical string
	switch j.KTY {
	case "EC":
		if j.CRV == "" || j.X == "" || j.Y == "" {
			return "", fmt.Errorf("%w: EC key missing crv, x or y", ErrUnsupportedKey)
		}
		canonical = fmt.Sprintf(`{"crv":%s,"kty":"EC","x":%s,"y":%s}`,
			quote(j.CRV), quote(j.X), quote(j.Y))
	case "RSA":
		if j.N == "" || j.E == "" {
			return "", fmt.Errorf("%w: RSA key missing n or e", ErrUnsupportedKey)
		}
		canonical = fmt.Sprintf(`{"e":%s,"kty":"RSA","n":%s}`, quote(j.E), quote(j.N))
	case "OKP":
		if j.CRV == "" || j.X == "" {
			return "", fmt.Errorf("%w: OKP key missing crv or x", ErrUnsupportedKey)
		}
		canonical = fmt.Sprintf(`{"crv":%s,"kty":"OKP","x":%s}`, quote(j.CRV), quote(j.X))
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedKey, j.KTY)
	}

	sum := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// quote JSON-encodes a string member. Using encoding/json rather than adding
// quotes by hand keeps any character needing an escape correct.
func quote(s string) string {
	b, _ := json.Marshal(s) //nolint:errcheck // marshalling a string cannot fail
	return string(b)
}

// ParseJSON converts a raw JWK into a public key, rejecting anything that
// carries private key material or fails a strength or well-formedness check.
// It returns the decoded JWK alongside the key so callers can compute a
// thumbprint without unmarshalling twice.
func ParseJSON(raw json.RawMessage) (crypto.PublicKey, *JWK, error) {
	// Check for private members against the generic object first. The typed
	// struct below has no fields for them, so unmarshalling into it would
	// discard the evidence and leave us importing an attacker's key material
	// as though it were an ordinary public key.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, nil, fmt.Errorf("jwkutil: decode jwk: %w", err)
	}
	for _, m := range privateMembers {
		if _, found := probe[m]; found {
			return nil, nil, fmt.Errorf("%w: member %q", ErrPrivateKeyMaterial, m)
		}
	}

	var j JWK
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, nil, fmt.Errorf("jwkutil: decode jwk: %w", err)
	}

	switch j.KTY {
	case "EC":
		pub, err := parseEC(&j)
		return pub, &j, err
	case "RSA":
		pub, err := parseRSA(&j)
		return pub, &j, err
	case "OKP":
		pub, err := parseOKP(&j)
		return pub, &j, err
	default:
		return nil, nil, fmt.Errorf("%w: %q", ErrUnsupportedKey, j.KTY)
	}
}

func parseEC(j *JWK) (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	var ecdhCurve ecdh.Curve
	switch j.CRV {
	case "P-256":
		curve = elliptic.P256()
		ecdhCurve = ecdh.P256()
	case "P-384":
		curve = elliptic.P384()
		ecdhCurve = ecdh.P384()
	case "P-521":
		curve = elliptic.P521()
		ecdhCurve = ecdh.P521()
	default:
		return nil, fmt.Errorf("%w: curve %q", ErrUnsupportedKey, j.CRV)
	}

	x, err := decodeBigInt(j.X)
	if err != nil {
		return nil, fmt.Errorf("jwkutil: decode x: %w", err)
	}
	y, err := decodeBigInt(j.Y)
	if err != nil {
		return nil, fmt.Errorf("jwkutil: decode y: %w", err)
	}

	// The jwk header this decodes is entirely attacker-controlled (it comes
	// straight off a DPoP proof), so the point it names has to be validated,
	// not trusted. crypto/ecdh.NewPublicKey is the maintained way to do that:
	// it rejects a point that is not on the curve, and it also rejects the
	// identity/infinity encoding, which a bare on-curve check does not cover.
	// Build the uncompressed SEC1 form (0x04 || X || Y, each field-size
	// padded) and let NewPublicKey be the judge.
	bitSize := curve.Params().BitSize
	sec1 := make([]byte, 0, 1+2*fieldSize(bitSize))
	sec1 = append(sec1, 0x04)
	sec1 = append(sec1, paddedCoordinate(x, bitSize)...)
	sec1 = append(sec1, paddedCoordinate(y, bitSize)...)
	if _, ecdhErr := ecdhCurve.NewPublicKey(sec1); ecdhErr != nil {
		return nil, fmt.Errorf("%w: point is not on curve %s: %w", ErrUnsupportedKey, j.CRV, ecdhErr)
	}

	// golang-jwt verifies ES* signatures against *ecdsa.PublicKey, so build
	// that from the same, now-validated, x and y rather than anything ecdh
	// hands back.
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func parseRSA(j *JWK) (*rsa.PublicKey, error) {
	n, err := decodeBigInt(j.N)
	if err != nil {
		return nil, fmt.Errorf("jwkutil: decode n: %w", err)
	}
	if n.BitLen() < minRSABits {
		return nil, fmt.Errorf("%w: RSA modulus is %d bits, minimum %d", ErrWeakKey, n.BitLen(), minRSABits)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return nil, fmt.Errorf("jwkutil: decode e: %w", err)
	}
	if len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, fmt.Errorf("%w: RSA exponent length %d", ErrUnsupportedKey, len(eBytes))
	}
	var buf [8]byte
	copy(buf[8-len(eBytes):], eBytes)
	e := binary.BigEndian.Uint64(buf[:])
	if e < 3 || e > 1<<31 {
		return nil, fmt.Errorf("%w: RSA exponent %d out of range", ErrUnsupportedKey, e)
	}
	return &rsa.PublicKey{N: n, E: int(e)}, nil
}

func parseOKP(j *JWK) (ed25519.PublicKey, error) {
	if j.CRV != "Ed25519" {
		return nil, fmt.Errorf("%w: OKP curve %q", ErrUnsupportedKey, j.CRV)
	}
	b, err := base64.RawURLEncoding.DecodeString(j.X)
	if err != nil {
		return nil, fmt.Errorf("jwkutil: decode x: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: Ed25519 key is %d bytes, want %d", ErrUnsupportedKey, len(b), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(b), nil
}

func decodeBigInt(s string) (*big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("empty value")
	}
	return new(big.Int).SetBytes(b), nil
}

// Encode converts a public key into a JWK. kid and alg are optional.
func Encode(pub crypto.PublicKey, kid, alg string) (*JWK, error) {
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		bits := k.Curve.Params().BitSize
		var crv string
		switch bits {
		case 256:
			crv = "P-256"
		case 384:
			crv = "P-384"
		case 521:
			crv = "P-521"
		default:
			return nil, fmt.Errorf("%w: EC curve with %d bits", ErrUnsupportedKey, bits)
		}
		return &JWK{
			KTY: "EC", Use: "sig", KID: kid, ALG: alg, CRV: crv,
			// k.X and k.Y: the JWK wire format needs the affine coordinates as
			// big-endian integers, and this key is our own (the JWKS endpoint's),
			// never attacker input. Moving off the affine fields here would
			// change published JWKS output and every existing RFC 7638
			// thumbprint DPoP has bound a token to.
			X: coordinate(k.X, bits), Y: coordinate(k.Y, bits), //nolint:staticcheck // SA1019: see comment above, our own key, wire format needs affine ints
		}, nil

	case *rsa.PublicKey:
		eBytes := big.NewInt(int64(k.E)).Bytes()
		return &JWK{
			KTY: "RSA", Use: "sig", KID: kid, ALG: alg,
			N: base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
			E: base64.RawURLEncoding.EncodeToString(eBytes),
		}, nil

	case ed25519.PublicKey:
		return &JWK{
			KTY: "OKP", Use: "sig", KID: kid, ALG: alg, CRV: "Ed25519",
			X: base64.RawURLEncoding.EncodeToString(k),
		}, nil

	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedKey, pub)
	}
}

// fieldSize returns the byte length of a curve's field elements.
func fieldSize(bitSize int) int {
	return (bitSize + 7) / 8
}

// paddedCoordinate renders v as big-endian bytes, left-padded to the curve's
// field size.
//
// big.Int.Bytes() drops leading zero bytes, so a coordinate whose high byte is
// zero would serialise one byte short. coordinate (the JWK encoder) and
// parseEC (the SEC1 point it validates a jwk header against) both need that
// padding, so it lives here once rather than twice.
func paddedCoordinate(v *big.Int, bitSize int) []byte {
	size := fieldSize(bitSize)
	b := v.Bytes()
	if len(b) < size {
		padded := make([]byte, size)
		copy(padded[size-len(b):], b)
		b = padded
	}
	return b
}

// coordinate renders an EC coordinate left-padded to the curve's field size.
//
// A coordinate whose high byte is zero would otherwise serialise one byte
// short and produce a JWK that strict verifiers reject. That happens for
// roughly 1 key in 256 per coordinate, which survives casual testing and then
// fails intermittently in production.
func coordinate(v *big.Int, bitSize int) string {
	return base64.RawURLEncoding.EncodeToString(paddedCoordinate(v, bitSize))
}
