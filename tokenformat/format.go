// Package tokenformat provides pluggable token generation strategies.
// Access tokens can be opaque (hex) or JWT; refresh tokens are always opaque.
package tokenformat

import (
	"errors"
	"time"
)

// Sentinel errors.
var (
	ErrInvalidToken  = errors.New("tokenformat: invalid token")
	ErrTokenExpired  = errors.New("tokenformat: token expired")
	ErrUnsignedToken = errors.New("tokenformat: unsigned or tampered token")
)

// TokenClaims carries the identity payload embedded in an access token.
type TokenClaims struct {
	UserID    string   `json:"sub"`
	AppID     string   `json:"app_id"`
	EnvID     string   `json:"env_id,omitempty"`
	OrgID     string   `json:"org_id,omitempty"`
	SessionID string   `json:"sid"`
	Scopes    []string `json:"scopes,omitempty"`
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Format generates and validates access tokens. Refresh tokens are always
// opaque (must be revocable via the store) so they are not part of this
// interface.
type Format interface {
	// Name returns the format identifier ("opaque" or "jwt").
	Name() string

	// GenerateAccessToken produces a token string from claims.
	GenerateAccessToken(claims TokenClaims) (string, error)

	// ValidateAccessToken parses and validates a token, returning the
	// embedded claims. Returns ErrInvalidToken or ErrTokenExpired on
	// failure.
	ValidateAccessToken(token string) (*TokenClaims, error)
}
