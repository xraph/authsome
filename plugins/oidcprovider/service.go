package oidcprovider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"log"
	"strings"
	"time"
)

// Config represents the OIDC Provider configuration
type Config struct {
	// Issuer URL for the OIDC Provider
	Issuer string `json:"issuer"`

	// Key configuration
	Keys struct {
		// Path to RSA private key file (PEM format)
		PrivateKeyPath string `json:"privateKeyPath"`
		// Path to RSA public key file (PEM format)
		PublicKeyPath string `json:"publicKeyPath"`
		// Key rotation settings
		RotationInterval string `json:"rotationInterval"` // e.g., "24h"
		KeyLifetime      string `json:"keyLifetime"`      // e.g., "168h" (7 days)
	} `json:"keys"`

	// Token settings
	Tokens struct {
		AccessTokenExpiry  string `json:"accessTokenExpiry"`  // e.g., "1h"
		IDTokenExpiry      string `json:"idTokenExpiry"`      // e.g., "1h"
		RefreshTokenExpiry string `json:"refreshTokenExpiry"` // e.g., "720h" (30 days)
	} `json:"tokens"`
}

// Service provides OIDC Provider operations
type Service struct {
	clientRepo  *repo.OAuthClientRepository
	codeRepo    *repo.AuthorizationCodeRepository
	tokenRepo   *repo.OAuthTokenRepository
	sessionSvc  *session.Service
	userSvc     *user.Service
	jwtService  *JWTService
	jwksService *JWKSService
	config      Config

	// Key rotation management
	rotationTicker *time.Ticker
	rotationDone   chan bool

	// Legacy in-memory storage for backward compatibility
	codes  map[string]string            // code -> clientID
	cbr    map[string]string            // code -> redirectURI
	tokens map[string]map[string]string // accessToken -> {"sub": userID}
}

// AuthorizeRequest represents an OAuth2/OIDC authorization request
type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
}

// ConsentDecision represents a user's consent decision
type ConsentDecision struct {
	Approved bool
	Scopes   []string
}

func NewService(config Config) *Service {
	// Set default values if not provided
	if config.Issuer == "" {
		config.Issuer = "http://localhost:3001"
	}
	if config.Keys.RotationInterval == "" {
		config.Keys.RotationInterval = "24h"
	}
	if config.Keys.KeyLifetime == "" {
		config.Keys.KeyLifetime = "168h" // 7 days
	}
	if config.Tokens.AccessTokenExpiry == "" {
		config.Tokens.AccessTokenExpiry = "1h"
	}
	if config.Tokens.IDTokenExpiry == "" {
		config.Tokens.IDTokenExpiry = "1h"
	}
	if config.Tokens.RefreshTokenExpiry == "" {
		config.Tokens.RefreshTokenExpiry = "720h" // 30 days
	}

	return &Service{
		config: config,
		codes:  make(map[string]string),
		cbr:    make(map[string]string),
		tokens: make(map[string]map[string]string),
	}
}

func NewServiceWithRepo(clientRepo *repo.OAuthClientRepository, config Config) *Service {
	s := NewService(config)
	s.clientRepo = clientRepo

	// Initialize JWKS service with configuration
	var jwksService *JWKSService
	var err error

	if config.Keys.PrivateKeyPath != "" && config.Keys.PublicKeyPath != "" {
		// Load keys from files
		jwksService, err = NewJWKSServiceFromFiles(config.Keys.PrivateKeyPath, config.Keys.PublicKeyPath, config.Keys.RotationInterval, config.Keys.KeyLifetime)
	} else {
		// Use auto-generated keys (development mode)
		log.Println("No certificate paths configured, using auto-generated keys for development")
		jwksService, err = NewJWKSService()
	}

	if err != nil {
		log.Printf("Failed to initialize JWKS service: %v", err)
		return s
	}
	s.jwksService = jwksService

	// Initialize JWT service
	jwtService, err := NewJWTService(config.Issuer, jwksService)
	if err != nil {
		log.Printf("Failed to initialize JWT service: %v", err)
		return s
	}
	s.jwtService = jwtService

	return s
}

// SetRepositories configures the service with all required repositories
func (s *Service) SetRepositories(clientRepo *repo.OAuthClientRepository, codeRepo *repo.AuthorizationCodeRepository, tokenRepo *repo.OAuthTokenRepository) {
	s.clientRepo = clientRepo
	s.codeRepo = codeRepo
	s.tokenRepo = tokenRepo
}

// SetSessionService configures the session service for user authentication checks
func (s *Service) SetSessionService(sessionSvc *session.Service) {
	s.sessionSvc = sessionSvc
}

// SetUserService sets the user service for the OIDC Provider
func (s *Service) SetUserService(userSvc *user.Service) {
	s.userSvc = userSvc
}

// RegisterClient persists a new OAuth client
func (s *Service) RegisterClient(ctx context.Context, name, redirectURI string) (*schema.OAuthClient, error) {
	c := &schema.OAuthClient{
		ID:           xid.New(),
		Name:         name,
		ClientID:     xid.New().String(),
		ClientSecret: xid.New().String(),
		RedirectURI:  redirectURI,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if s.clientRepo != nil {
		if err := s.clientRepo.Create(ctx, c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// ValidateClient checks client_id and redirect uri
func (s *Service) ValidateClient(ctx context.Context, clientID, redirectURI string) bool {
	if s.clientRepo == nil {
		return clientID != "" && redirectURI != ""
	}
	c, err := s.clientRepo.FindByClientID(ctx, clientID)
	if err != nil || c == nil {
		return false
	}
	return c.RedirectURI == redirectURI
}

// ValidateAuthorizeRequest validates an OAuth2/OIDC authorization request
func (s *Service) ValidateAuthorizeRequest(ctx context.Context, req *AuthorizeRequest) error {
	// Validate required parameters
	if req.ClientID == "" {
		return fmt.Errorf("missing client_id")
	}
	if req.RedirectURI == "" {
		return fmt.Errorf("missing redirect_uri")
	}
	if req.ResponseType != "code" {
		return fmt.Errorf("unsupported response_type: %s", req.ResponseType)
	}

	// Validate client and redirect URI
	if !s.ValidateClient(ctx, req.ClientID, req.RedirectURI) {
		return fmt.Errorf("invalid client or redirect URI")
	}

	// Validate PKCE if present
	if req.CodeChallenge != "" {
		if req.CodeChallengeMethod != "S256" && req.CodeChallengeMethod != "plain" {
			return fmt.Errorf("unsupported code_challenge_method: %s", req.CodeChallengeMethod)
		}
	}

	return nil
}

// CheckUserSession checks if the user has a valid session
func (s *Service) CheckUserSession(ctx context.Context, sessionToken string) (*session.Session, error) {
	if s.sessionSvc == nil {
		return nil, fmt.Errorf("session service not configured")
	}

	sess, err := s.sessionSvc.FindByToken(ctx, sessionToken)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	return sess, nil
}

// GenerateAuthorizationCode generates a secure authorization code
func (s *Service) GenerateAuthorizationCode() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate code: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateAuthorizationCode creates and stores an authorization code
func (s *Service) CreateAuthorizationCode(ctx context.Context, req *AuthorizeRequest, userID xid.ID) (*schema.AuthorizationCode, error) {
	code, err := s.GenerateAuthorizationCode()
	if err != nil {
		return nil, err
	}

	authCode := &schema.AuthorizationCode{
		ID:                  xid.New(),
		Code:                code,
		ClientID:            req.ClientID,
		UserID:              userID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(10 * time.Minute), // 10 minute expiry
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if s.codeRepo != nil {
		if err := s.codeRepo.Create(ctx, authCode); err != nil {
			return nil, fmt.Errorf("failed to store authorization code: %w", err)
		}
	} else {
		// Fallback to in-memory storage
		s.codes[code] = req.ClientID
		s.cbr[code] = req.RedirectURI
	}

	return authCode, nil
}

// ValidateAuthorizationCode validates and retrieves an authorization code
func (s *Service) ValidateAuthorizationCode(ctx context.Context, code, clientID, redirectURI, codeVerifier string) (*schema.AuthorizationCode, error) {
	var authCode *schema.AuthorizationCode
	var err error

	if s.codeRepo != nil {
		authCode, err = s.codeRepo.FindByCode(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("authorization code not found")
		}
	} else {
		// Fallback to in-memory storage
		if storedClientID, exists := s.codes[code]; !exists || storedClientID != clientID {
			return nil, fmt.Errorf("invalid authorization code")
		}
		if storedRedirectURI, exists := s.cbr[code]; !exists || storedRedirectURI != redirectURI {
			return nil, fmt.Errorf("invalid redirect URI")
		}
		// Create a mock auth code for in-memory mode
		authCode = &schema.AuthorizationCode{
			Code:        code,
			ClientID:    clientID,
			RedirectURI: redirectURI,
			ExpiresAt:   time.Now().Add(10 * time.Minute),
		}
	}

	// Validate the authorization code
	if !authCode.IsValid() {
		return nil, fmt.Errorf("authorization code expired or already used")
	}

	if authCode.ClientID != clientID {
		return nil, fmt.Errorf("client ID mismatch")
	}

	if authCode.RedirectURI != redirectURI {
		return nil, fmt.Errorf("redirect URI mismatch")
	}

	// Validate PKCE if present
	if authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			return nil, fmt.Errorf("code verifier required for PKCE")
		}

		if !s.validatePKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier) {
			return nil, fmt.Errorf("invalid PKCE code verifier")
		}
	}

	return authCode, nil
}

// validatePKCE validates PKCE code challenge and verifier
func (s *Service) validatePKCE(challenge, method, verifier string) bool {
	switch method {
	case "plain":
		return challenge == verifier
	case "S256":
		hash := sha256.Sum256([]byte(verifier))
		return challenge == base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
	default:
		return false
	}
}

// MarkCodeAsUsed marks an authorization code as used
func (s *Service) MarkCodeAsUsed(ctx context.Context, code string) error {
	if s.codeRepo != nil {
		return s.codeRepo.MarkAsUsed(ctx, code)
	} else {
		// Remove from in-memory storage
		delete(s.codes, code)
		delete(s.cbr, code)
	}
	return nil
}

// IssueCode generates and stores an authorization code for a client
func (s *Service) IssueCode(clientID, redirectURI string) string {
	code := xid.New().String()
	s.codes[code] = clientID
	s.cbr[code] = redirectURI
	return code
}

// ExchangeCode returns an access token if code is valid
func (s *Service) ExchangeCode(code string) (string, bool) {
	clientID, ok := s.codes[code]
	if !ok {
		return "", false
	}
	token := xid.New().String()
	s.tokens[token] = map[string]string{"client_id": clientID}
	delete(s.codes, code)
	delete(s.cbr, code)
	return token, true
}

// TokenResponse represents the response from the token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ExchangeCodeForTokens exchanges an authorization code for JWT tokens
func (s *Service) ExchangeCodeForTokens(ctx context.Context, authCode *schema.AuthorizationCode, userInfo map[string]interface{}) (*TokenResponse, error) {
	if s.jwtService == nil {
		return nil, fmt.Errorf("JWT service not initialized")
	}

	// Generate access token
	accessToken, err := s.jwtService.GenerateAccessToken(
		authCode.UserID.String(),
		authCode.ClientID,
		authCode.Scope,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate ID token if openid scope is requested
	var idToken string
	if containsScope(authCode.Scope, "openid") {
		idToken, err = s.jwtService.GenerateIDToken(
			authCode.UserID.String(),
			authCode.ClientID,
			authCode.Nonce,
			authCode.CreatedAt,
			userInfo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ID token: %w", err)
		}
	}

	// Generate refresh token (simple implementation)
	refreshToken := xid.New().String()

	// Store tokens in database
	refreshExpiresAt := time.Now().Add(24 * time.Hour) // 24 hours

	oauthToken := &schema.OAuthToken{
		ID:               xid.New(),
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ClientID:         authCode.ClientID,
		UserID:           authCode.UserID,
		Scope:            authCode.Scope,
		ExpiresAt:        time.Now().Add(time.Hour), // 1 hour
		RefreshExpiresAt: &refreshExpiresAt,
	}

	if err := s.tokenRepo.Create(ctx, oauthToken); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour in seconds
		RefreshToken: refreshToken,
		IDToken:      idToken,
		Scope:        authCode.Scope,
	}, nil
}

// GetJWKS returns the JSON Web Key Set for token verification
func (s *Service) GetJWKS() (*JWKS, error) {
	if s.jwksService == nil {
		return nil, fmt.Errorf("JWKS service not initialized")
	}
	return s.jwksService.GetJWKS()
}

// containsScope checks if a scope string contains a specific scope
func containsScope(scopes, target string) bool {
	for _, scope := range strings.Split(scopes, " ") {
		if scope == target {
			return true
		}
	}
	return false
}

// StartKeyRotation begins automatic key rotation in the background
func (s *Service) StartKeyRotation() {
	if s.jwksService == nil {
		log.Println("JWKS service not initialized, skipping key rotation")
		return
	}

	// Check every hour for key rotation
	s.rotationTicker = time.NewTicker(1 * time.Hour)
	s.rotationDone = make(chan bool)

	go func() {
		for {
			select {
			case <-s.rotationTicker.C:
				if s.jwksService.ShouldRotate() {
					log.Println("Rotating OIDC Provider keys...")
					if err := s.jwksService.RotateKeys(); err != nil {
						log.Printf("Failed to rotate keys: %v", err)
					} else {
						log.Println("Successfully rotated OIDC Provider keys")
					}
				}
			case <-s.rotationDone:
				return
			}
		}
	}()

	log.Println("Started automatic key rotation for OIDC Provider")
}

// StopKeyRotation stops the automatic key rotation
func (s *Service) StopKeyRotation() {
	if s.rotationTicker != nil {
		s.rotationTicker.Stop()
	}
	if s.rotationDone != nil {
		close(s.rotationDone)
	}
	log.Println("Stopped automatic key rotation for OIDC Provider")
}

// ValidateAccessToken validates an access token and returns the associated OAuth token
func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (*schema.OAuthToken, error) {
	if s.tokenRepo == nil {
		return nil, fmt.Errorf("token repository not initialized")
	}

	token, err := s.tokenRepo.FindByAccessToken(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to find token: %w", err)
	}

	if !token.IsValid() {
		return nil, fmt.Errorf("token is invalid or expired")
	}

	return token, nil
}

// GetUserInfoFromToken retrieves user information based on an access token and requested scopes
func (s *Service) GetUserInfoFromToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	// Validate the access token
	token, err := s.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	// Get user information
	if s.userSvc == nil {
		return nil, fmt.Errorf("user service not initialized")
	}

	user, err := s.userSvc.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Build user info response based on scopes
	userInfo := map[string]interface{}{
		"sub": user.ID.String(),
	}

	// Parse scopes from token
	scopes := strings.Split(token.Scope, " ")

	// Add profile information if profile scope is requested
	if containsScope(token.Scope, "profile") {
		if user.Name != "" {
			userInfo["name"] = user.Name
		}
		if user.Username != "" {
			userInfo["preferred_username"] = user.Username
		}
		if user.DisplayUsername != "" {
			userInfo["display_username"] = user.DisplayUsername
		}
		if user.Image != "" {
			userInfo["picture"] = user.Image
		}
	}

	// Add email information if email scope is requested
	if containsScope(token.Scope, "email") {
		if user.Email != "" {
			userInfo["email"] = user.Email
			userInfo["email_verified"] = user.EmailVerified
		}
	}

	// Add additional standard claims based on scopes
	for _, scope := range scopes {
		switch scope {
		case "openid":
			// Already included sub
		case "profile":
			// Already handled above
		case "email":
			// Already handled above
		case "phone":
			// Already handled above
		case "address":
			// Could add address information if available in user schema
		}
	}

	return userInfo, nil
}
