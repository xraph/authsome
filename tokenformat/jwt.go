package tokenformat

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig configures the JWT token format.
type JWTConfig struct {
	// SigningMethod is the JWT signing algorithm (e.g., jwt.SigningMethodRS256,
	// jwt.SigningMethodES256, jwt.SigningMethodHS256).
	SigningMethod jwt.SigningMethod

	// SigningKey is the private key used to sign tokens.
	// - RSA: *rsa.PrivateKey
	// - ECDSA: *ecdsa.PrivateKey
	// - HMAC: []byte
	SigningKey any

	// VerifyKey is the public key used to verify tokens.
	// For HMAC, this should be the same as SigningKey.
	// - RSA: *rsa.PublicKey
	// - ECDSA: *ecdsa.PublicKey
	// - HMAC: []byte
	VerifyKey any

	// KeyID is the "kid" header value for key rotation support.
	KeyID string

	// Issuer is the "iss" claim (e.g., "https://auth.example.com").
	Issuer string

	// Audience is the "aud" claim (e.g., "api-service").
	Audience string
}

// JWT generates and validates JWT access tokens.
type JWT struct {
	config JWTConfig
}

// Compile-time check.
var _ Format = (*JWT)(nil)

// NewJWT creates a new JWT token format with the given configuration.
func NewJWT(cfg JWTConfig) (*JWT, error) {
	if cfg.SigningMethod == nil {
		return nil, errors.New("tokenformat: signing method required")
	}
	if cfg.SigningKey == nil {
		return nil, errors.New("tokenformat: signing key required")
	}
	if cfg.VerifyKey == nil {
		// For HMAC, verify key == signing key.
		cfg.VerifyKey = cfg.SigningKey
	}
	return &JWT{config: cfg}, nil
}

func (j *JWT) Name() string { return "jwt" }

// customClaims embeds jwt.RegisteredClaims and adds our custom fields.
type customClaims struct {
	jwt.RegisteredClaims
	AppID     string   `json:"app_id,omitempty"`
	EnvID     string   `json:"env_id,omitempty"`
	OrgID     string   `json:"org_id,omitempty"`
	SessionID string   `json:"sid,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
}

func (j *JWT) GenerateAccessToken(claims TokenClaims) (string, error) {
	now := time.Now()
	jwtClaims := customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(claims.ExpiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		AppID:     claims.AppID,
		EnvID:     claims.EnvID,
		OrgID:     claims.OrgID,
		SessionID: claims.SessionID,
		Scopes:    claims.Scopes,
	}

	if j.config.Issuer != "" {
		jwtClaims.Issuer = j.config.Issuer
	}
	if j.config.Audience != "" {
		jwtClaims.Audience = jwt.ClaimStrings{j.config.Audience}
	}

	token := jwt.NewWithClaims(j.config.SigningMethod, jwtClaims)
	if j.config.KeyID != "" {
		token.Header["kid"] = j.config.KeyID
	}

	signed, err := token.SignedString(j.config.SigningKey)
	if err != nil {
		return "", fmt.Errorf("tokenformat: sign jwt: %w", err)
	}
	return signed, nil
}

func (j *JWT) ValidateAccessToken(tokenStr string) (*TokenClaims, error) {
	claims := &customClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		// Verify the signing method matches.
		if token.Method.Alg() != j.config.SigningMethod.Alg() {
			return nil, fmt.Errorf("tokenformat: unexpected signing method: %s", token.Method.Alg())
		}
		return j.config.VerifyKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrUnsignedToken
	}

	issuedAt := time.Time{}
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Time
	}
	expiresAt := time.Time{}
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &TokenClaims{
		UserID:    claims.Subject,
		AppID:     claims.AppID,
		EnvID:     claims.EnvID,
		OrgID:     claims.OrgID,
		SessionID: claims.SessionID,
		Scopes:    claims.Scopes,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}, nil
}

// IsJWT returns true if the token string appears to be a JWT (has 3 dot-separated parts).
func IsJWT(token string) bool {
	return strings.Count(token, ".") == 2
}

// PublicKey returns the public key for JWKS exposure.
func (j *JWT) PublicKey() any {
	switch k := j.config.VerifyKey.(type) {
	case *rsa.PublicKey:
		return k
	case *ecdsa.PublicKey:
		return k
	default:
		return nil // HMAC keys should not be exposed
	}
}

// KeyID returns the configured key ID for JWKS.
func (j *JWT) KeyID() string { return j.config.KeyID }

// Algorithm returns the signing algorithm name.
func (j *JWT) Algorithm() string { return j.config.SigningMethod.Alg() }
