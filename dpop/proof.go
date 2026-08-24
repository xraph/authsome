package dpop

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/xraph/authsome/internal/jwkutil"
)

// MaxProofBytes caps the DPoP header before any parsing happens. A proof is a
// few hundred bytes; anything larger is either broken or an attempt to make us
// do expensive work on garbage.
const MaxProofBytes = 4096

// allowedAlgs is the complete set of proof algorithms.
//
// This is an allow-list and not a deny-list on purpose. RFC 9449 forbids
// symmetric algorithms because the jwk header is attacker-supplied: with HS256
// the "public key" in the header would be the very secret used to verify the
// signature, so any proof would verify against itself.
var allowedAlgs = map[string]bool{
	"ES256": true,
	"RS256": true,
	"PS256": true,
	"EdDSA": true,
}

// SupportedAlgs returns the advertised algorithm list for server metadata.
func SupportedAlgs() []string { return []string{"ES256", "RS256", "PS256", "EdDSA"} }

// Proof is a parsed and signature-verified DPoP proof. Holding one means the
// signature checked out; it says nothing yet about whether the proof matches
// the request it arrived on. That is Validator's job.
type Proof struct {
	Raw string

	Typ string
	Alg string
	JWK json.RawMessage

	PublicKey crypto.PublicKey
	JKT       string

	JTI      string
	HTM      string
	HTU      string
	Nonce    string
	ATH      string
	IssuedAt time.Time
}

type proofHeader struct {
	Typ string          `json:"typ"`
	Alg string          `json:"alg"`
	JWK json.RawMessage `json:"jwk"`
}

type proofClaims struct {
	JTI   string `json:"jti"`
	HTM   string `json:"htm"`
	HTU   string `json:"htu"`
	IAT   *int64 `json:"iat"`
	Nonce string `json:"nonce,omitempty"`
	ATH   string `json:"ath,omitempty"`
}

// Parse decodes a compact DPoP proof and verifies its signature against the
// public key carried in its own jwk header.
//
// Self-signed verification looks circular and is not. The signature proves the
// sender holds the private key for the jwk they supplied; binding that key to
// a token is what makes the claim mean anything, and that check happens in
// Validator against the token's thumbprint.
func Parse(raw string) (*Proof, error) {
	if raw == "" || len(raw) > MaxProofBytes {
		return nil, fmt.Errorf("%w: length %d", ErrMalformedProof, len(raw))
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: expected 3 segments, got %d", ErrMalformedProof, len(parts))
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: decode header: %w", ErrMalformedProof, err)
	}
	var h proofHeader
	if err := json.Unmarshal(headerBytes, &h); err != nil {
		return nil, fmt.Errorf("%w: decode header json: %w", ErrMalformedProof, err)
	}

	// typ before alg. A caller who reads alg first and dispatches on it has
	// already let a plain JWT into the proof code path.
	if h.Typ != "dpop+jwt" {
		return nil, fmt.Errorf("%w: typ is %q, want dpop+jwt", ErrMalformedProof, h.Typ)
	}
	if !allowedAlgs[h.Alg] {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedAlg, h.Alg)
	}
	if len(h.JWK) == 0 {
		return nil, fmt.Errorf("%w: missing jwk header", ErrMalformedProof)
	}

	pub, jwkDecoded, err := jwkutil.ParseJSON(h.JWK)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedProof, err)
	}

	// Look the method up from our allow-list, never from the token's own
	// header string, so GetSigningMethod cannot hand back an HMAC method.
	method := jwt.GetSigningMethod(h.Alg)
	if method == nil {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedAlg, h.Alg)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: decode signature: %w", ErrMalformedProof, err)
	}
	if err := method.Verify(parts[0]+"."+parts[1], sig, pub); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBadSignature, err)
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: decode claims: %w", ErrMalformedProof, err)
	}
	var c proofClaims
	if err := json.Unmarshal(claimBytes, &c); err != nil {
		return nil, fmt.Errorf("%w: decode claims json: %w", ErrMalformedProof, err)
	}
	if c.JTI == "" || c.HTM == "" || c.HTU == "" || c.IAT == nil {
		return nil, fmt.Errorf("%w: missing one of jti, htm, htu, iat", ErrMalformedProof)
	}

	jkt, err := jwkutil.Thumbprint(jwkDecoded)
	if err != nil {
		return nil, fmt.Errorf("%w: thumbprint: %w", ErrMalformedProof, err)
	}

	return &Proof{
		Raw: raw, Typ: h.Typ, Alg: h.Alg, JWK: h.JWK,
		PublicKey: pub, JKT: jkt,
		JTI: c.JTI, HTM: c.HTM, HTU: c.HTU,
		Nonce: c.Nonce, ATH: c.ATH,
		IssuedAt: time.Unix(*c.IAT, 0),
	}, nil
}
