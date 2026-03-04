package tokenformat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Opaque generates 64-character hex tokens (the default AuthSome format).
// Validation always returns ErrInvalidToken because opaque tokens must be
// resolved via the session store — they carry no embedded claims.
type Opaque struct{}

// Compile-time check.
var _ Format = (*Opaque)(nil)

func (Opaque) Name() string { return "opaque" }

func (Opaque) GenerateAccessToken(_ TokenClaims) (string, error) {
	return generateSecureToken(32)
}

func (Opaque) ValidateAccessToken(_ string) (*TokenClaims, error) {
	return nil, ErrInvalidToken
}

// GenerateRefreshToken creates an opaque refresh token.
// This is a standalone function because refresh tokens are always opaque
// regardless of the access token format.
func GenerateRefreshToken() (string, error) {
	return generateSecureToken(32)
}

func generateSecureToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("tokenformat: generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
