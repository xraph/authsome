package api

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/internal/jwkutil"
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

// jwtToJWK converts a JWT format's public key into the JWKS response shape.
// The conversion itself lives in internal/jwkutil so the DPoP proof path and
// this endpoint cannot drift apart on padding or curve naming.
func jwtToJWK(jwtFmt *tokenformat.JWT) *JWK {
	pub := jwtFmt.PublicKey()
	if pub == nil {
		return nil // HMAC keys must never be published
	}
	j, err := jwkutil.Encode(pub, jwtFmt.KeyID(), jwtFmt.Algorithm())
	if err != nil {
		return nil
	}
	// For ECDSA keys, generate a thumbprint-based KID if one isn't provided.
	result := &JWK{
		KTY: j.KTY, Use: j.Use, KID: j.KID, ALG: j.ALG,
		N: j.N, E: j.E, CRV: j.CRV, X: j.X, Y: j.Y,
	}
	if result.KID == "" && result.CRV != "" {
		// Generate a thumbprint-based KID for ECDSA keys.
		h := sha256.Sum256([]byte(result.X + result.Y))
		result.KID = base64.RawURLEncoding.EncodeToString(h[:8])
	}
	return result
}
