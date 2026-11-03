package oauth2provider

import (
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrSigningKeyConflict is returned by CreateSigningKey when a key with the
// same (Scope, Generation) already exists.
var ErrSigningKeyConflict = errors.New("oauth2: signing key generation already exists")

// SigningKey is one id_token signing key. Keys are never deleted: a retired
// key keeps its public half in the JWKS until RetiresAt passes, so tokens
// signed just before a rotation still verify.
type SigningKey struct {
	ID id.OAuth2SigningKeyID
	// Scope is "" for the deployment issuer, else the app id.
	Scope string
	Alg   string // "RS256"
	// PrivateKey is sealed with engine.TokenEncryptor().
	PrivateKey string
	// PublicJWK is the JSON of one JWK, exactly as served.
	PublicJWK  []byte
	Generation int64
	CreatedAt  time.Time
	// RetiresAt is nil while the key is active or not yet retired.
	RetiresAt *time.Time
}
