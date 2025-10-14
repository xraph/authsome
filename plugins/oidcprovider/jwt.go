package oidcprovider

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/xid"
)

// JWTService handles JWT token generation and signing for OIDC Provider
type JWTService struct {
	jwksService *JWKSService
	issuer      string
}

// NewJWTService creates a new JWT service with JWKS service for key management
func NewJWTService(issuer string, jwksService *JWKSService) (*JWTService, error) {
	return &JWTService{
		jwksService: jwksService,
		issuer:      issuer,
	}, nil
}

// IDTokenClaims represents the claims for an OIDC ID token
type IDTokenClaims struct {
	jwt.RegisteredClaims
	Nonce           string `json:"nonce,omitempty"`
	AuthTime        int64  `json:"auth_time"`
	SessionState    string `json:"session_state,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Email           string `json:"email,omitempty"`
	EmailVerified   bool   `json:"email_verified,omitempty"`
	Name            string `json:"name,omitempty"`
	GivenName       string `json:"given_name,omitempty"`
	FamilyName      string `json:"family_name,omitempty"`
}

// AccessTokenClaims represents the claims for an access token
type AccessTokenClaims struct {
	jwt.RegisteredClaims
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id"`
	TokenType string `json:"token_type"`
}

// GenerateIDToken creates a signed OIDC ID token
func (j *JWTService) GenerateIDToken(userID, clientID, nonce string, authTime time.Time, userInfo map[string]interface{}) (string, error) {
	now := time.Now()
	
	claims := IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  []string{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), // 1 hour expiry
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        xid.New().String(),
		},
		Nonce:    nonce,
		AuthTime: authTime.Unix(),
	}

	// Add user information to claims
	if email, ok := userInfo["email"].(string); ok {
		claims.Email = email
		claims.EmailVerified = true // Assume verified for now
	}
	if name, ok := userInfo["name"].(string); ok {
		claims.Name = name
	}
	if username, ok := userInfo["username"].(string); ok {
		claims.PreferredUsername = username
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.jwksService.GetCurrentKeyID()

	return token.SignedString(j.jwksService.GetCurrentPrivateKey())
}

// GenerateAccessToken creates a signed access token
func (j *JWTService) GenerateAccessToken(userID, clientID, scope string) (string, error) {
	now := time.Now()
	
	claims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			Audience:  []string{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), // 1 hour expiry
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        xid.New().String(),
		},
		Scope:     scope,
		ClientID:  clientID,
		TokenType: "Bearer",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.jwksService.GetCurrentKeyID()

	return token.SignedString(j.jwksService.GetCurrentPrivateKey())
}

// VerifyToken verifies and parses a JWT token
func (j *JWTService) VerifyToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid kid in token header")
		}
		
		// Get public key for this key ID
		publicKey, err := j.jwksService.GetPublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key for kid %s: %w", kid, err)
		}
		
		return publicKey, nil
	})
}