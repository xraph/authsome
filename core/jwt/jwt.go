package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// JWT KEY DTO (Data Transfer Object)
// =============================================================================

// JWTKey represents a JWT signing key DTO
// This is separate from schema.JWTKey to maintain proper separation of concerns
type JWTKey struct {
	ID            xid.ID                 `json:"id"`
	AppID         xid.ID                 `json:"appId"`
	IsPlatformKey bool                   `json:"isPlatformKey"`
	KeyID         string                 `json:"keyId"`
	Algorithm     string                 `json:"algorithm"`
	KeyType       string                 `json:"keyType"`
	Curve         string                 `json:"curve,omitempty"`
	PrivateKey    string                 `json:"-"` // Never expose in JSON
	PublicKey     string                 `json:"publicKey"`
	IsActive      bool                   `json:"isActive"`
	UsageCount    int64                  `json:"usageCount"`
	LastUsedAt    *time.Time             `json:"lastUsedAt,omitempty"`
	ExpiresAt     *time.Time             `json:"expiresAt,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the JWTKey DTO to a schema.JWTKey model
func (k *JWTKey) ToSchema() *schema.JWTKey {
	// Convert metadata
	metadata := make(map[string]string)
	for key, val := range k.Metadata {
		if str, ok := val.(string); ok {
			metadata[key] = str
		}
	}

	return &schema.JWTKey{
		ID:            k.ID,
		AppID:         k.AppID,
		IsPlatformKey: k.IsPlatformKey,
		KeyID:         k.KeyID,
		Algorithm:     k.Algorithm,
		KeyType:       k.KeyType,
		Curve:         k.Curve,
		PrivateKey:    []byte(k.PrivateKey),
		PublicKey:     []byte(k.PublicKey),
		Active:        k.IsActive,
		UsageCount:    k.UsageCount,
		LastUsedAt:    k.LastUsedAt,
		ExpiresAt:     k.ExpiresAt,
		Metadata:      metadata,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
		DeletedAt:     k.DeletedAt,
	}
}

// FromSchemaJWTKey converts a schema.JWTKey model to JWTKey DTO
func FromSchemaJWTKey(sk *schema.JWTKey) *JWTKey {
	if sk == nil {
		return nil
	}

	// Convert metadata
	metadata := make(map[string]interface{})
	for key, val := range sk.Metadata {
		metadata[key] = val
	}

	return &JWTKey{
		ID:            sk.ID,
		AppID:         sk.AppID,
		IsPlatformKey: sk.IsPlatformKey,
		KeyID:         sk.KeyID,
		Algorithm:     sk.Algorithm,
		KeyType:       sk.KeyType,
		Curve:         sk.Curve,
		PrivateKey:    string(sk.PrivateKey),
		PublicKey:     string(sk.PublicKey),
		IsActive:      sk.Active,
		UsageCount:    sk.UsageCount,
		LastUsedAt:    sk.LastUsedAt,
		ExpiresAt:     sk.ExpiresAt,
		Metadata:      metadata,
		CreatedAt:     sk.CreatedAt,
		UpdatedAt:     sk.UpdatedAt,
		DeletedAt:     sk.DeletedAt,
	}
}

// FromSchemaJWTKeys converts a slice of schema.JWTKey to JWTKey DTOs
func FromSchemaJWTKeys(keys []*schema.JWTKey) []*JWTKey {
	result := make([]*JWTKey, len(keys))
	for i, k := range keys {
		result[i] = FromSchemaJWTKey(k)
	}
	return result
}

// =============================================================================
// JWT KEY METHODS
// =============================================================================

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

// =============================================================================
// TOKEN CLAIMS
// =============================================================================

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID      string                 `json:"userId"`
	AppID       string                 `json:"appId"`
	SessionID   string                 `json:"sessionId,omitempty"`
	Scopes      []string               `json:"scopes,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	TokenType   string                 `json:"tokenType"` // access, refresh, id
	Audience    []string               `json:"aud,omitempty"`
	Subject     string                 `json:"sub"`
	Issuer      string                 `json:"iss"`
	IssuedAt    *jwt.NumericDate       `json:"iat"`
	ExpiresAt   *jwt.NumericDate       `json:"exp"`
	NotBefore   *jwt.NumericDate       `json:"nbf,omitempty"`
	JwtID       string                 `json:"jti"`
	KeyID       string                 `json:"kid"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	jwt.RegisteredClaims
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateJWTKeyRequest represents a request to create a JWT key
type CreateJWTKeyRequest struct {
	AppID         xid.ID                 `json:"appId" validate:"required"`
	IsPlatformKey bool                   `json:"isPlatformKey"`
	Algorithm     string                 `json:"algorithm" validate:"required,oneof=RS256 RS384 RS512 ES256 ES384 ES512 HS256 HS384 HS512"`
	KeyType       string                 `json:"keyType" validate:"required,oneof=RSA ECDSA HMAC"`
	Curve         string                 `json:"curve,omitempty" validate:"omitempty,oneof=P-256 P-384 P-521"`
	ExpiresAt     *time.Time             `json:"expiresAt,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateTokenRequest represents a request to generate a JWT token
type GenerateTokenRequest struct {
	UserID      string                 `json:"userId" validate:"required"`
	AppID       xid.ID                 `json:"appId" validate:"required"`
	SessionID   string                 `json:"sessionId,omitempty"`
	TokenType   string                 `json:"tokenType" validate:"required,oneof=access refresh id"`
	Scopes      []string               `json:"scopes,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Audience    []string               `json:"audience,omitempty"`
	ExpiresIn   time.Duration          `json:"expiresIn,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateTokenResponse represents the response from token generation
type GenerateTokenResponse struct {
	Token     string    `json:"token"`
	TokenType string    `json:"tokenType"`
	ExpiresAt time.Time `json:"expiresAt"`
	ExpiresIn int64     `json:"expiresIn"`
}

// VerifyTokenRequest represents a request to verify a JWT token
type VerifyTokenRequest struct {
	Token     string   `json:"token" validate:"required"`
	AppID     xid.ID   `json:"appId" validate:"required"`
	Audience  []string `json:"audience,omitempty"`
	TokenType string   `json:"tokenType,omitempty"`
}

// VerifyTokenResponse represents the response from token verification
type VerifyTokenResponse struct {
	Valid       bool         `json:"valid"`
	Claims      *TokenClaims `json:"claims,omitempty"`
	Error       string       `json:"error,omitempty"`
	UserID      string       `json:"userId,omitempty"`
	AppID       string       `json:"appId,omitempty"`
	SessionID   string       `json:"sessionId,omitempty"`
	Scopes      []string     `json:"scopes,omitempty"`
	Permissions []string     `json:"permissions,omitempty"`
	ExpiresAt   *time.Time   `json:"expiresAt,omitempty"`
}

// JWKSResponse represents a JSON Web Key Set response
type JWKSResponse = pagination.PageResponse[JWK]

// JWK represents a JSON Web Key
type JWK struct {
	KeyType   string   `json:"kty"`
	KeyID     string   `json:"kid"`
	Use       string   `json:"use"`
	Algorithm string   `json:"alg"`
	N         string   `json:"n,omitempty"`       // RSA modulus
	E         string   `json:"e,omitempty"`       // RSA exponent
	X         string   `json:"x,omitempty"`       // ECDSA x coordinate
	Y         string   `json:"y,omitempty"`       // ECDSA y coordinate
	Curve     string   `json:"crv,omitempty"`     // ECDSA curve
	KeyOps    []string `json:"key_ops,omitempty"` // Key operations
}

type ListJWTKeysResponse = pagination.PageResponse[*JWTKey]
