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

// ConfiguredAudience returns the app-wide default "aud" this format stamps on
// tokens whose claims carry no audience of their own. Empty when unset.
//
// Callers need it so a session row can record the same audience the minted
// token actually carries. When the row says nothing and the claim says
// "https://api.example.com", the two disagree, and any check that reads one
// after the other gets two different answers about the same credential.
func (j *JWT) ConfiguredAudience() string { return j.config.Audience }

// Confirmation is the RFC 7800 cnf claim. Only the jkt member is used.
type Confirmation struct {
	JKT string `json:"jkt,omitempty"`
}

// customClaims embeds jwt.RegisteredClaims and adds our custom fields.
type customClaims struct {
	jwt.RegisteredClaims
	AppID     string   `json:"app_id,omitempty"`
	EnvID     string   `json:"env_id,omitempty"`
	OrgID     string   `json:"org_id,omitempty"`
	SessionID string   `json:"sid,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	// Act is the RFC 8693 delegation claim. Omitted entirely for an
	// impersonated token, which is how the RFC encodes that case.
	Act *ActClaim `json:"act,omitempty"`
	// PrincipalKind and PrincipalID name the caller when it is not a user.
	// Both stay absent on a human token.
	PrincipalKind string `json:"pk,omitempty"`
	PrincipalID   string `json:"pid,omitempty"`
	// Confirmation is a pointer with omitempty so an unbound token carries no
	// cnf member at all, rather than an empty object. Anything reading these
	// tokens treats the presence of cnf as the signal that a proof is required.
	Confirmation *Confirmation `json:"cnf,omitempty"`
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
		AppID:         claims.AppID,
		EnvID:         claims.EnvID,
		OrgID:         claims.OrgID,
		SessionID:     claims.SessionID,
		Scopes:        claims.Scopes,
		Act:           claims.Act,
		PrincipalKind: claims.PrincipalKind,
		PrincipalID:   claims.PrincipalID,
	}

	if claims.DPoPJKT != "" {
		jwtClaims.Confirmation = &Confirmation{JKT: claims.DPoPJKT}
	}

	if j.config.Issuer != "" {
		jwtClaims.Issuer = j.config.Issuer
	}
	// A per-token audience is the resource the client actually asked for, so
	// it wins. The configured value is an app-wide default from before
	// resource indicators existed.
	switch {
	case len(claims.Audience) > 0:
		jwtClaims.Audience = jwt.ClaimStrings(claims.Audience)
	case j.config.Audience != "":
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
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
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
		UserID:        claims.Subject,
		AppID:         claims.AppID,
		EnvID:         claims.EnvID,
		OrgID:         claims.OrgID,
		SessionID:     claims.SessionID,
		Scopes:        claims.Scopes,
		Audience:      []string(claims.Audience),
		DPoPJKT:       confirmationJKT(claims.Confirmation),
		Act:           claims.Act,
		PrincipalKind: claims.PrincipalKind,
		PrincipalID:   claims.PrincipalID,
		IssuedAt:      issuedAt,
		ExpiresAt:     expiresAt,
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

// HMACKey returns the symmetric signing key when this JWT format uses an
// HMAC algorithm (HS256/HS384/HS512). The second return is false for RSA or
// ECDSA configurations, where no symmetric key exists.
//
// This is intended for callers that need a process-wide symmetric secret
// to derive other HMAC keys from (e.g. the dashboard nonce signer). Never
// expose the returned bytes over the network.
func (j *JWT) HMACKey() ([]byte, bool) {
	if k, ok := j.config.SigningKey.([]byte); ok {
		return k, true
	}
	return nil, false
}

// confirmationJKT reads the thumbprint out of a cnf claim, tolerating both an
// absent claim and a present but empty one.
func confirmationJKT(c *Confirmation) string {
	if c == nil {
		return ""
	}
	return c.JKT
}
