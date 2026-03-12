package api

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"math/big"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/tokenformat"
)

// JWKSResponse is the JSON Web Key Set response.
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a single JSON Web Key.
type JWK struct {
	KTY string `json:"kty"`
	Use string `json:"use"`
	KID string `json:"kid,omitempty"`
	ALG string `json:"alg,omitempty"`
	N   string `json:"n,omitempty"`   // RSA modulus
	E   string `json:"e,omitempty"`   // RSA exponent
	CRV string `json:"crv,omitempty"` // EC curve
	X   string `json:"x,omitempty"`   // EC x coordinate
	Y   string `json:"y,omitempty"`   // EC y coordinate
}

// registerJWKSRoutes registers the JWKS endpoint if JWT formats are configured.
func (a *API) registerJWKSRoutes(router forge.Router) error {
	jwtFormats := a.engine.JWTFormats()
	if len(jwtFormats) == 0 && !a.engine.HasJWT() {
		return nil // No JWT configured, skip JWKS endpoint
	}

	return router.GET("/.well-known/jwks.json", a.handleJWKS,
		forge.WithSummary("JSON Web Key Set"),
		forge.WithOperationID("getJWKS"),
		forge.WithResponseSchema(http.StatusOK, "JWKS", JWKSResponse{}),
		forge.WithTags("JWT"),
	)
}

// handleJWKS serves the public keys for JWT verification.
func (a *API) handleJWKS(_ forge.Context, _ *struct{}) (*JWKSResponse, error) {
	var keys []JWK

	// Collect keys from per-app JWT formats.
	for _, jwtFmt := range a.engine.JWTFormats() {
		if jwk := jwtToJWK(jwtFmt); jwk != nil {
			keys = append(keys, *jwk)
		}
	}

	// Collect from default format if JWT.
	if defaultFmt := a.engine.DefaultTokenFormat(); defaultFmt != nil && defaultFmt.Name() == "jwt" {
		if jwtFmt, ok := defaultFmt.(*tokenformat.JWT); ok {
			if jwk := jwtToJWK(jwtFmt); jwk != nil {
				keys = append(keys, *jwk)
			}
		}
	}

	return &JWKSResponse{Keys: keys}, nil
}

func jwtToJWK(jwtFmt *tokenformat.JWT) *JWK {
	pub := jwtFmt.PublicKey()
	if pub == nil {
		return nil // HMAC keys are not exposed
	}

	kid := jwtFmt.KeyID()
	alg := jwtFmt.Algorithm()

	switch k := pub.(type) {
	case *rsa.PublicKey:
		return &JWK{
			KTY: "RSA",
			Use: "sig",
			KID: kid,
			ALG: alg,
			N:   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes()),
		}
	case *ecdsa.PublicKey:
		jwk := &JWK{
			KTY: "EC",
			Use: "sig",
			KID: kid,
			ALG: alg,
			X:   base64.RawURLEncoding.EncodeToString(k.X.Bytes()),
			Y:   base64.RawURLEncoding.EncodeToString(k.Y.Bytes()),
		}
		switch k.Curve.Params().BitSize {
		case 256:
			jwk.CRV = "P-256"
		case 384:
			jwk.CRV = "P-384"
		case 521:
			jwk.CRV = "P-521"
		}
		if kid == "" {
			// Generate a thumbprint-based KID for ECDSA keys.
			h := sha256.Sum256([]byte(jwk.X + jwk.Y))
			jwk.KID = base64.RawURLEncoding.EncodeToString(h[:8])
		}
		return jwk
	default:
		return nil
	}
}
