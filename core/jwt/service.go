package jwt

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/internal/crypto"
)

// Config holds JWT service configuration
type Config struct {
	EncryptionKey string `json:"encryption_key"`
	DefaultTTL    string `json:"default_ttl"`
}

// Service handles JWT operations
type Service struct {
	repo     Repository
	auditSvc *audit.Service
	config   Config
}

// NewService creates a new JWT service
func NewService(config Config, repo Repository, auditSvc *audit.Service) *Service {
	return &Service{
		repo:     repo,
		auditSvc: auditSvc,
		config:   config,
	}
}

// CreateJWTKey creates a new JWT signing key
func (s *Service) CreateJWTKey(ctx context.Context, req *CreateJWTKeyRequest) (*JWTKey, error) {
	// Generate key pair based on algorithm
	privateKeyBytes, publicKeyBytes, err := s.generateKeyPair(req.Algorithm, req.KeyType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate IDs
	id := xid.New()
	keyID := xid.New().String()

	// Encrypt private key for storage
	encryptedPrivateKey, err := crypto.Encrypt(string(privateKeyBytes), s.config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Parse default TTL
	defaultTTL, err := time.ParseDuration(s.config.DefaultTTL)
	if err != nil {
		defaultTTL = 24 * time.Hour // Default to 24 hours if parsing fails
	}

	// Set expiration if not provided
	expiresAt := req.ExpiresAt
	if expiresAt == nil {
		expiry := time.Now().Add(defaultTTL)
		expiresAt = &expiry
	}

	// Create JWT key record
	jwtKey := &JWTKey{
		ID:          id,
		OrgID:       req.OrgID,
		KeyID:       keyID,
		Algorithm:   req.Algorithm,
		KeyType:     req.KeyType,
		PrivateKey:  encryptedPrivateKey,
		PublicKey:   string(publicKeyBytes),
		IsActive:    true,
		ExpiresAt:   expiresAt,
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	if err := s.repo.Create(ctx, jwtKey); err != nil {
		return nil, fmt.Errorf("failed to create JWT key: %w", err)
	}

	// Audit log
	if s.auditSvc != nil {
		userID := xid.NilID()
		s.auditSvc.Log(ctx, &userID, "jwt_key", "create", fmt.Sprintf(`{"key_id":"%s","algorithm":"%s"}`, keyID, req.Algorithm), req.OrgID, "")
	}

	return jwtKey, nil
}

// GenerateToken creates a new JWT token
func (s *Service) GenerateToken(ctx context.Context, req *GenerateTokenRequest) (*GenerateTokenResponse, error) {
	// Find an active signing key for the organization
	keys, _, err := s.repo.FindByOrgID(ctx, req.OrgID, &[]bool{true}[0], 0, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to find signing key: %w", err)
	}
	
	if len(keys) == 0 {
		return nil, fmt.Errorf("no active signing key found for organization")
	}
	
	signingKey := keys[0]

	// Check if key is active and not expired
	if !signingKey.IsActive || signingKey.IsExpired() {
		return nil, fmt.Errorf("signing key is not active or expired")
	}

	// Decrypt private key
	decryptedPrivateKey, err := crypto.Decrypt(signingKey.PrivateKey, s.config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Parse private key
	privateKey, err := s.parsePrivateKey(decryptedPrivateKey, signingKey.Algorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Calculate expiration
	expiresIn := req.ExpiresIn
	if expiresIn == 0 {
		switch req.TokenType {
		case "access":
			expiresIn = 1 * time.Hour
		case "refresh":
			expiresIn = 24 * time.Hour
		case "id":
			expiresIn = 1 * time.Hour
		default:
			expiresIn = 1 * time.Hour
		}
	}
	
	now := time.Now()
	expiresAt := now.Add(expiresIn)

	// Create claims
	claims := &TokenClaims{
		UserID:      req.UserID,
		OrgID:       req.OrgID,
		SessionID:   req.SessionID,
		TokenType:   req.TokenType,
		Scopes:      req.Scopes,
		Permissions: req.Permissions,
		Audience:    req.Audience,
		Subject:     req.UserID,
		Issuer:      fmt.Sprintf("authsome:%s", req.OrgID),
		IssuedAt:    jwt.NewNumericDate(now),
		ExpiresAt:   jwt.NewNumericDate(expiresAt),
		JwtID:       xid.New().String(),
		KeyID:       signingKey.KeyID,
		Metadata:    req.Metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   req.UserID,
			Issuer:    fmt.Sprintf("authsome:%s", req.OrgID),
			Audience:  req.Audience,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        xid.New().String(),
		},
	}

	// Create token
	token := jwt.NewWithClaims(s.getSigningMethod(signingKey.Algorithm), claims)
	token.Header["kid"] = signingKey.KeyID

	// Sign token
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Update usage statistics
	if err := s.repo.UpdateUsage(ctx, signingKey.KeyID); err != nil {
		// Log error but don't fail the request
		fmt.Printf("failed to update key usage: %v\n", err)
	}

	return &GenerateTokenResponse{
		Token:     tokenString,
		TokenType: req.TokenType,
		ExpiresAt: expiresAt,
		ExpiresIn: int64(expiresIn.Seconds()),
	}, nil
}

// VerifyToken verifies a JWT token
func (s *Service) VerifyToken(ctx context.Context, req *VerifyTokenRequest) (*VerifyTokenResponse, error) {
	// Parse token to get kid from header
	token, err := jwt.Parse(req.Token, func(token *jwt.Token) (interface{}, error) {
		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header")
		}

		// Find verification key
		verificationKey, err := s.repo.FindByKeyID(ctx, kid, req.OrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to find verification key: %w", err)
		}

		// Check if key is active and not expired
		if !verificationKey.IsActive || verificationKey.IsExpired() {
			return nil, fmt.Errorf("verification key is not active or expired")
		}

		// Parse public key
		return s.parsePublicKey(verificationKey.PublicKey, verificationKey.Algorithm)
	})

	if err != nil {
		return &VerifyTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	if !token.Valid {
		return &VerifyTokenResponse{
			Valid: false,
			Error: "invalid token",
		}, nil
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &VerifyTokenResponse{
			Valid: false,
			Error: "invalid claims",
		}, nil
	}

	// Validate audience if specified
	if len(req.Audience) > 0 {
		tokenAud := getStringSliceClaim(claims, "aud")
		if !hasIntersection(tokenAud, req.Audience) {
			return &VerifyTokenResponse{
				Valid: false,
				Error: "invalid audience",
			}, nil
		}
	}

	// Validate token type if specified
	if req.TokenType != "" {
		tokenType := getStringClaim(claims, "token_type")
		if tokenType != req.TokenType {
			return &VerifyTokenResponse{
				Valid: false,
				Error: "invalid token type",
			}, nil
		}
	}

	// Build response
	response := &VerifyTokenResponse{
		Valid:       true,
		UserID:      getStringClaim(claims, "user_id"),
		OrgID:       getStringClaim(claims, "org_id"),
		SessionID:   getStringClaim(claims, "session_id"),
		Scopes:      getStringSliceClaim(claims, "scopes"),
		Permissions: getStringSliceClaim(claims, "permissions"),
		ExpiresAt:   getTimeClaim(claims, "exp"),
		Claims: &TokenClaims{
			UserID:      getStringClaim(claims, "user_id"),
			OrgID:       getStringClaim(claims, "org_id"),
			SessionID:   getStringClaim(claims, "session_id"),
			TokenType:   getStringClaim(claims, "token_type"),
			Scopes:      getStringSliceClaim(claims, "scopes"),
			Permissions: getStringSliceClaim(claims, "permissions"),
			Audience:    getStringSliceClaim(claims, "aud"),
			Subject:     getStringClaim(claims, "sub"),
			Issuer:      getStringClaim(claims, "iss"),
			JwtID:       getStringClaim(claims, "jti"),
			KeyID:       getStringClaim(claims, "kid"),
		},
	}

	return response, nil
}

// GetJWKS returns the JSON Web Key Set for an organization
func (s *Service) GetJWKS(ctx context.Context, orgID string) (*JWKSResponse, error) {
	// Find all active keys for the organization
	keys, _, err := s.repo.FindByOrgID(ctx, orgID, &[]bool{true}[0], 0, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to find keys: %w", err)
	}

	jwks := &JWKSResponse{
		Keys: make([]JWK, 0, len(keys)),
	}

	for _, key := range keys {
		if key.IsActive && !key.IsExpired() {
			jwk, err := s.convertToJWK(key)
			if err != nil {
				continue // Skip invalid keys
			}
			jwks.Keys = append(jwks.Keys, *jwk)
		}
	}

	return jwks, nil
}

// ListJWTKeys lists JWT keys for an organization
func (s *Service) ListJWTKeys(ctx context.Context, req *ListJWTKeysRequest) (*ListJWTKeysResponse, error) {
	offset := (req.Page - 1) * req.PageSize
	keys, total, err := s.repo.FindByOrgID(ctx, req.OrgID, req.Active, offset, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to find keys: %w", err)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ListJWTKeysResponse{
		Keys:       keys,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CleanupExpired removes expired JWT keys
func (s *Service) CleanupExpired(ctx context.Context) (int64, error) {
	return s.repo.CleanupExpired(ctx)
}

// generateKeyPair generates a key pair based on algorithm and key type
func (s *Service) generateKeyPair(algorithm, keyType string) ([]byte, []byte, error) {
	switch keyType {
	case "RSA":
		// Generate RSA key pair
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, err
		}

		// Encode private key
		privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		})

		// Encode public key
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			return nil, nil, err
		}
		publicKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		})

		return privateKeyPEM, publicKeyPEM, nil

	case "ECDSA":
		// For now, use Ed25519 for simplicity
		fallthrough
	case "HMAC":
		fallthrough
	default:
		// Generate Ed25519 key pair
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}

		// Encode private key
		privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, nil, err
		}
		privateKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyBytes,
		})

		// Encode public key
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return nil, nil, err
		}
		publicKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		})

		return privateKeyPEM, publicKeyPEM, nil
	}
}

// parsePrivateKey parses a private key based on algorithm
func (s *Service) parsePrivateKey(keyData, algorithm string) (interface{}, error) {
	block, _ := pem.Decode([]byte(keyData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if strings.HasPrefix(algorithm, "RS") {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	
	return x509.ParsePKCS8PrivateKey(block.Bytes)
}

// parsePublicKey parses a public key based on algorithm
func (s *Service) parsePublicKey(keyData, algorithm string) (interface{}, error) {
	block, _ := pem.Decode([]byte(keyData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParsePKIXPublicKey(block.Bytes)
}

// getSigningMethod returns the signing method for an algorithm
func (s *Service) getSigningMethod(algorithm string) jwt.SigningMethod {
	switch algorithm {
	case "RS256":
		return jwt.SigningMethodRS256
	case "RS384":
		return jwt.SigningMethodRS384
	case "RS512":
		return jwt.SigningMethodRS512
	case "ES256":
		return jwt.SigningMethodES256
	case "ES384":
		return jwt.SigningMethodES384
	case "ES512":
		return jwt.SigningMethodES512
	default:
		return jwt.SigningMethodEdDSA
	}
}

// convertToJWK converts a JWT key to JWK format
func (s *Service) convertToJWK(key *JWTKey) (*JWK, error) {
	publicKey, err := s.parsePublicKey(key.PublicKey, key.Algorithm)
	if err != nil {
		return nil, err
	}

	jwk := &JWK{
		KeyType:   key.KeyType,
		KeyID:     key.KeyID,
		Use:       "sig",
		Algorithm: key.Algorithm,
		KeyOps:    []string{"verify"},
	}

	switch pub := publicKey.(type) {
	case *rsa.PublicKey:
		jwk.N = string(pub.N.Bytes())
		jwk.E = strconv.Itoa(pub.E)
	case ed25519.PublicKey:
		jwk.X = string(pub)
		jwk.Curve = "Ed25519"
	}

	return jwk, nil
}

// Helper functions
func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

func getStringSliceClaim(claims jwt.MapClaims, key string) []string {
	if val, ok := claims[key].([]interface{}); ok {
		result := make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	if val, ok := claims[key].(string); ok {
		return []string{val}
	}
	return nil
}

func getTimeClaim(claims jwt.MapClaims, key string) *time.Time {
	if val, ok := claims[key].(float64); ok {
		t := time.Unix(int64(val), 0)
		return &t
	}
	return nil
}

func hasIntersection(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}