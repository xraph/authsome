package jwt

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/xid"
)

// JWTKey represents a JWT signing key
type JWTKey struct {
	ID          xid.ID    `json:"id"`
	OrgID       string    `json:"org_id"`
	KeyID       string    `json:"key_id"`
	Algorithm   string    `json:"algorithm"`
	KeyType     string    `json:"key_type"`
	Curve       string    `json:"curve,omitempty"`
	PrivateKey  string    `json:"-"` // Never expose in JSON
	PublicKey   string    `json:"public_key"`
	IsActive    bool      `json:"is_active"`
	UsageCount  int64     `json:"usage_count"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID       string                 `json:"user_id"`
	OrgID        string                 `json:"org_id"`
	SessionID    string                 `json:"session_id,omitempty"`
	Scopes       []string               `json:"scopes,omitempty"`
	Permissions  []string               `json:"permissions,omitempty"`
	TokenType    string                 `json:"token_type"` // access, refresh, id
	Audience     []string               `json:"aud,omitempty"`
	Subject      string                 `json:"sub"`
	Issuer       string                 `json:"iss"`
	IssuedAt     *jwt.NumericDate       `json:"iat"`
	ExpiresAt    *jwt.NumericDate       `json:"exp"`
	NotBefore    *jwt.NumericDate       `json:"nbf,omitempty"`
	JwtID        string                 `json:"jti"`
	KeyID        string                 `json:"kid"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	jwt.RegisteredClaims
}

// CreateJWTKeyRequest represents a request to create a JWT key
type CreateJWTKeyRequest struct {
	OrgID     string                 `json:"org_id" validate:"required"`
	Algorithm string                 `json:"algorithm" validate:"required,oneof=RS256 RS384 RS512 ES256 ES384 ES512 HS256 HS384 HS512"`
	KeyType   string                 `json:"key_type" validate:"required,oneof=RSA ECDSA HMAC"`
	Curve     string                 `json:"curve,omitempty" validate:"omitempty,oneof=P-256 P-384 P-521"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateTokenRequest represents a request to generate a JWT token
type GenerateTokenRequest struct {
	UserID      string                 `json:"user_id" validate:"required"`
	OrgID       string                 `json:"org_id" validate:"required"`
	SessionID   string                 `json:"session_id,omitempty"`
	TokenType   string                 `json:"token_type" validate:"required,oneof=access refresh id"`
	Scopes      []string               `json:"scopes,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Audience    []string               `json:"audience,omitempty"`
	ExpiresIn   time.Duration          `json:"expires_in,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateTokenResponse represents the response from token generation
type GenerateTokenResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
	ExpiresIn int64     `json:"expires_in"`
}

// VerifyTokenRequest represents a request to verify a JWT token
type VerifyTokenRequest struct {
	Token     string   `json:"token" validate:"required"`
	OrgID     string   `json:"org_id" validate:"required"`
	Audience  []string `json:"audience,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
}

// VerifyTokenResponse represents the response from token verification
type VerifyTokenResponse struct {
	Valid       bool          `json:"valid"`
	Claims      *TokenClaims  `json:"claims,omitempty"`
	Error       string        `json:"error,omitempty"`
	UserID      string        `json:"user_id,omitempty"`
	OrgID       string        `json:"org_id,omitempty"`
	SessionID   string        `json:"session_id,omitempty"`
	Scopes      []string      `json:"scopes,omitempty"`
	Permissions []string      `json:"permissions,omitempty"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
}

// JWKSResponse represents a JSON Web Key Set response
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	KeyType   string   `json:"kty"`
	KeyID     string   `json:"kid"`
	Use       string   `json:"use"`
	Algorithm string   `json:"alg"`
	N         string   `json:"n,omitempty"`         // RSA modulus
	E         string   `json:"e,omitempty"`         // RSA exponent
	X         string   `json:"x,omitempty"`         // ECDSA x coordinate
	Y         string   `json:"y,omitempty"`         // ECDSA y coordinate
	Curve     string   `json:"crv,omitempty"`       // ECDSA curve
	KeyOps    []string `json:"key_ops,omitempty"`   // Key operations
}

// ListJWTKeysRequest represents a request to list JWT keys
type ListJWTKeysRequest struct {
	OrgID    string `json:"org_id" validate:"required"`
	Page     int    `json:"page" validate:"min=1"`
	PageSize int    `json:"page_size" validate:"min=1,max=100"`
	Active   *bool  `json:"active,omitempty"`
}

// ListJWTKeysResponse represents the response from listing JWT keys
type ListJWTKeysResponse struct {
	Keys       []*JWTKey `json:"keys"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// Repository defines the interface for JWT key storage
type Repository interface {
	// Create creates a new JWT key
	Create(ctx context.Context, key *JWTKey) error

	// FindByID finds a JWT key by ID
	FindByID(ctx context.Context, id string) (*JWTKey, error)

	// FindByKeyID finds a JWT key by key ID and organization
	FindByKeyID(ctx context.Context, keyID, orgID string) (*JWTKey, error)

	// FindByOrgID finds all JWT keys for an organization
	FindByOrgID(ctx context.Context, orgID string, active *bool, offset, limit int) ([]*JWTKey, int64, error)

	// Update updates a JWT key
	Update(ctx context.Context, key *JWTKey) error

	// UpdateUsage updates the usage statistics for a JWT key
	UpdateUsage(ctx context.Context, keyID string) error

	// Deactivate deactivates a JWT key
	Deactivate(ctx context.Context, id string) error

	// Delete soft deletes a JWT key
	Delete(ctx context.Context, id string) error

	// CleanupExpired removes expired JWT keys
	CleanupExpired(ctx context.Context) (int64, error)
}

// IsExpired checks if the JWT key is expired
func (k *JWTKey) IsExpired() bool {
	return k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now())
}

// CanSign checks if the key can be used for signing
func (k *JWTKey) CanSign() bool {
	return k.IsActive && !k.IsExpired() && k.PrivateKey != ""
}

// CanVerify checks if the key can be used for verification
func (k *JWTKey) CanVerify() bool {
	return k.IsActive && !k.IsExpired() && k.PublicKey != ""
}