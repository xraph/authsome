package oidcprovider

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/oidcprovider/deviceflow"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// DefaultConfig returns the default OIDC Provider configuration
func DefaultConfig() Config {
	return Config{
		Issuer: "http://localhost:3001",
	}
}

// Service provides enterprise OIDC Provider operations with org-aware support
type Service struct {
	clientRepo  *repo.OAuthClientRepository
	codeRepo    *repo.AuthorizationCodeRepository
	tokenRepo   *repo.OAuthTokenRepository
	consentRepo *repo.OAuthConsentRepository
	sessionSvc  *session.Service
	userSvc     *user.Service
	jwtService  *JWTService
	jwksService *JWKSService

	// Enterprise services
	registration  *RegistrationService
	introspection *IntrospectionService
	revocation    *RevokeTokenService
	consent       *ConsentService
	discovery     *DiscoveryService
	clientAuth    *ClientAuthenticator

	// Device flow service (RFC 8628)
	deviceFlowService *deviceflow.Service

	config         Config
	rotationTicker *time.Ticker
	rotationDone   chan bool
}

// NewService creates a new OIDC Provider service with default config
func NewService(config Config) *Service {
	// Set default values if not provided
	if config.Issuer == "" {
		config.Issuer = "http://localhost:3001"
	}

	return &Service{
		config: config,
	}
}

// NewServiceWithRepos creates a new OIDC Provider service with repositories
func NewServiceWithRepos(clientRepo *repo.OAuthClientRepository, config Config, db *bun.DB, appID xid.ID, logger interface{ Printf(string, ...interface{}) }) *Service {
	s := NewService(config)
	s.clientRepo = clientRepo

	// Initialize JWKS service
	var jwksService *JWKSService
	var err error

	if config.Keys.PrivateKeyPath != "" && config.Keys.PublicKeyPath != "" {
		// Load keys from files
		jwksService, err = NewJWKSServiceFromFiles(
			config.Keys.PrivateKeyPath,
			config.Keys.PublicKeyPath,
			config.Keys.RotationInterval,
			config.Keys.KeyLifetime,
		)
		if err != nil {
			logger.Printf("Failed to load keys from files: %v", err)
		} else {
			logger.Printf("JWKS service initialized from files")
		}
	}
	
	// If file-based keys failed or weren't configured, use database-backed keys
	if jwksService == nil {
		logger.Printf("Initializing database-backed JWKS service")
		jwksService, err = NewDatabaseJWKSService(db, appID, logger)
		if err != nil {
			logger.Printf("Failed to initialize database JWKS service: %v", err)
			// Fallback to in-memory keys
			logger.Printf("Falling back to in-memory keys (not recommended for production)")
			jwksService, err = NewJWKSService()
			if err != nil {
				logger.Printf("Failed to initialize in-memory JWKS service: %v", err)
				return s
			}
		} else {
			logger.Printf("Database-backed JWKS service initialized")
		}
	}

	s.jwksService = jwksService

	// Initialize JWT service
	jwtService, err := NewJWTService(config.Issuer, jwksService)
	if err != nil {
		logger.Printf("Failed to initialize JWT service: %v", err)
		return s
	}
	s.jwtService = jwtService
	logger.Printf("JWT service initialized")

	return s
}

// SetRepositories configures all required repositories
func (s *Service) SetRepositories(
	clientRepo *repo.OAuthClientRepository,
	codeRepo *repo.AuthorizationCodeRepository,
	tokenRepo *repo.OAuthTokenRepository,
	consentRepo *repo.OAuthConsentRepository,
) {
	s.clientRepo = clientRepo
	s.codeRepo = codeRepo
	s.tokenRepo = tokenRepo
	s.consentRepo = consentRepo

	// Initialize enterprise services
	s.registration = NewRegistrationService(clientRepo, s.config)

	// Create user service adapter for introspection
	var userSvcAdapter UserService
	if s.userSvc != nil {
		userSvcAdapter = &userServiceAdapter{svc: s.userSvc}
	}
	s.introspection = NewIntrospectionService(tokenRepo, clientRepo, userSvcAdapter)
	s.revocation = NewRevokeTokenService(tokenRepo)
	s.consent = NewConsentService(consentRepo, clientRepo)
	s.discovery = NewDiscoveryService(s.config)
	s.clientAuth = NewClientAuthenticator(clientRepo)
}

// SetSessionService configures the session service
func (s *Service) SetSessionService(sessionSvc *session.Service) {
	s.sessionSvc = sessionSvc
}

// SetUserService sets the user service
func (s *Service) SetUserService(userSvc *user.Service) {
	s.userSvc = userSvc
}

// SetDeviceFlowService sets the device flow service
func (s *Service) SetDeviceFlowService(deviceFlowSvc *deviceflow.Service) {
	s.deviceFlowService = deviceFlowSvc
}

// =============================================================================
// CONTEXT HELPERS
// =============================================================================

// ExtractContext extracts app, env, and org context from request context
func (s *Service) ExtractContext(ctx context.Context) (appID, envID xid.ID, orgID *xid.ID, err error) {
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		return xid.NilID(), xid.NilID(), nil, errs.BadRequest("app context required")
	}

	envID, ok = contexts.GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		return xid.NilID(), xid.NilID(), nil, errs.BadRequest("environment context required")
	}

	// Org context is optional
	orgIDVal, ok := contexts.GetOrganizationID(ctx)
	if ok && !orgIDVal.IsNil() {
		orgID = &orgIDVal
	}

	return appID, envID, orgID, nil
}

// =============================================================================
// AUTHORIZATION FLOW
// =============================================================================

// ValidateAuthorizeRequest validates an OAuth2/OIDC authorization request
func (s *Service) ValidateAuthorizeRequest(ctx context.Context, req *AuthorizeRequest) error {
	// Extract context
	appID, envID, orgID, err := s.ExtractContext(ctx)
	if err != nil {
		return err
	}

	// Validate required parameters
	if req.ClientID == "" {
		return errs.RequiredField("client_id")
	}
	if req.RedirectURI == "" {
		return errs.RequiredField("redirect_uri")
	}
	if req.ResponseType != "code" {
		return errs.BadRequest("unsupported response_type: " + req.ResponseType)
	}

	// Validate client with org hierarchy
	client, err := s.clientRepo.FindByClientIDWithContext(ctx, appID, envID, orgID, req.ClientID)
	if err != nil {
		return errs.DatabaseError("find client", err)
	}
	if client == nil {
		return errs.NotFound("invalid client_id")
	}

	// Validate redirect URI
	if err := s.validateRedirectURI(client, req.RedirectURI); err != nil {
		return err
	}

	// Validate PKCE if present
	if req.CodeChallenge != "" {
		if req.CodeChallengeMethod != "S256" && req.CodeChallengeMethod != "plain" {
			return errs.BadRequest("unsupported code_challenge_method")
		}
	} else if client.RequirePKCE {
		return errs.BadRequest("PKCE required for this client")
	}

	return nil
}

// validateRedirectURI checks if redirect URI is allowed for client
func (s *Service) validateRedirectURI(client *schema.OAuthClient, redirectURI string) error {
	// Check against all registered redirect URIs
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return nil
		}
	}

	// Also check legacy single URI for backward compatibility
	if client.RedirectURI == redirectURI {
		return nil
	}

	return errs.BadRequest("redirect_uri not registered for this client")
}

// CreateAuthorizationCode creates and stores an authorization code with full context
func (s *Service) CreateAuthorizationCode(ctx context.Context, req *AuthorizeRequest, userID xid.ID, sessionID xid.ID) (*schema.AuthorizationCode, error) {
	appID, envID, orgID, err := s.ExtractContext(ctx)
	if err != nil {
		return nil, err
	}

	code, err := s.GenerateAuthorizationCode()
	if err != nil {
		return nil, errs.InternalError(err)
	}

	authCode := &schema.AuthorizationCode{
		AppID:               appID,
		EnvironmentID:       envID,
		OrganizationID:      orgID,
		SessionID:           &sessionID,
		Code:                code,
		ClientID:            req.ClientID,
		UserID:              userID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ConsentGranted:      true, // Set by consent flow
		ConsentScopes:       req.Scope,
		AuthTime:            time.Now(),
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}

	if err := s.codeRepo.Create(ctx, authCode); err != nil {
		return nil, errs.DatabaseError("create authorization code", err)
	}

	return authCode, nil
}

// GenerateAuthorizationCode generates a secure authorization code
func (s *Service) GenerateAuthorizationCode() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// =============================================================================
// TOKEN EXCHANGE
// =============================================================================

// ValidateAuthorizationCode validates and retrieves an authorization code
func (s *Service) ValidateAuthorizationCode(ctx context.Context, code, clientID, redirectURI, codeVerifier string) (*schema.AuthorizationCode, error) {
	authCode, err := s.codeRepo.FindByCode(ctx, code)
	if err != nil {
		return nil, errs.DatabaseError("find authorization code", err)
	}
	if authCode == nil {
		return nil, errs.NotFound("invalid authorization code")
	}

	// Validate the authorization code
	if !authCode.IsValid() {
		return nil, errs.BadRequest("authorization code expired or already used")
	}

	if authCode.ClientID != clientID {
		return nil, errs.BadRequest("client ID mismatch")
	}

	if authCode.RedirectURI != redirectURI {
		return nil, errs.BadRequest("redirect URI mismatch")
	}

	// Validate PKCE if present
	if authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			return nil, errs.BadRequest("code verifier required for PKCE")
		}

		if !s.validatePKCE(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier) {
			return nil, errs.BadRequest("invalid PKCE code verifier")
		}
	}

	return authCode, nil
}

// validatePKCE validates PKCE code challenge and verifier with timing-safe comparison
func (s *Service) validatePKCE(challenge, method, verifier string) bool {
	switch method {
	case "plain":
		return challenge == verifier
	case "S256":
		hash := sha256.Sum256([]byte(verifier))
		computed := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
		return challenge == computed
	default:
		return false
	}
}

// MarkCodeAsUsed marks an authorization code as used
func (s *Service) MarkCodeAsUsed(ctx context.Context, code string) error {
	return s.codeRepo.MarkAsUsed(ctx, code)
}

// ExchangeCodeForTokens exchanges an authorization code for JWT tokens
func (s *Service) ExchangeCodeForTokens(ctx context.Context, authCode *schema.AuthorizationCode, userInfo map[string]interface{}) (*TokenResponse, error) {
	if s.jwtService == nil {
		return nil, errs.InternalError(fmt.Errorf("JWT service not initialized"))
	}

	// Generate JTI for token
	jti := "jti_" + xid.New().String()

	// Generate access token
	accessToken, err := s.jwtService.GenerateAccessToken(
		authCode.UserID.String(),
		authCode.ClientID,
		authCode.Scope,
	)
	if err != nil {
		return nil, errs.InternalError(fmt.Errorf("failed to generate access token: %w", err))
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
			return nil, errs.InternalError(fmt.Errorf("failed to generate ID token: %w", err))
		}
	}

	// Generate refresh token
	refreshToken := "refresh_" + xid.New().String()

	// Store tokens in database
	refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	oauthToken := &schema.OAuthToken{
		AppID:            authCode.AppID,
		EnvironmentID:    authCode.EnvironmentID,
		OrganizationID:   authCode.OrganizationID,
		SessionID:        authCode.SessionID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		IDToken:          idToken,
		TokenType:        "Bearer",
		TokenClass:       "access_token",
		ClientID:         authCode.ClientID,
		UserID:           authCode.UserID,
		Scope:            authCode.Scope,
		JTI:              jti,
		Issuer:           s.config.Issuer,
		AuthTime:         &authCode.AuthTime,
		ExpiresAt:        time.Now().Add(time.Hour), // 1 hour
		RefreshExpiresAt: &refreshExpiresAt,
	}

	if err := s.tokenRepo.Create(ctx, oauthToken); err != nil {
		return nil, errs.DatabaseError("create token", err)
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

// GenerateTokensForDeviceCode generates tokens for a device code grant
func (s *Service) GenerateTokensForDeviceCode(ctx context.Context, deviceCode *schema.DeviceCode, client *schema.OAuthClient) (*TokenResponse, error) {
	if s.jwtService == nil {
		return nil, errs.InternalError(fmt.Errorf("JWT service not initialized"))
	}

	// Get user info for ID token
	user, err := s.userSvc.FindByID(ctx, *deviceCode.UserID)
	if err != nil {
		return nil, errs.DatabaseError("find user", err)
	}

	userInfo := map[string]interface{}{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
	}

	// Generate JTI for token
	jti := "jti_" + xid.New().String()

	// Generate access token
	accessToken, err := s.jwtService.GenerateAccessToken(
		deviceCode.UserID.String(),
		deviceCode.ClientID,
		deviceCode.Scope,
	)
	if err != nil {
		return nil, errs.InternalError(fmt.Errorf("failed to generate access token: %w", err))
	}

	// Generate ID token if openid scope is requested
	var idToken string
	if containsScope(deviceCode.Scope, "openid") {
		idToken, err = s.jwtService.GenerateIDToken(
			deviceCode.UserID.String(),
			deviceCode.ClientID,
			"", // nonce not applicable for device flow
			deviceCode.CreatedAt,
			userInfo,
		)
		if err != nil {
			return nil, errs.InternalError(fmt.Errorf("failed to generate ID token: %w", err))
		}
	}

	// Generate refresh token
	refreshToken := "refresh_" + xid.New().String()

	// Store tokens in database
	refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	oauthToken := &schema.OAuthToken{
		AppID:            deviceCode.AppID,
		EnvironmentID:    deviceCode.EnvironmentID,
		OrganizationID:   deviceCode.OrganizationID,
		SessionID:        deviceCode.SessionID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		IDToken:          idToken,
		TokenType:        "Bearer",
		TokenClass:       "access_token",
		ClientID:         deviceCode.ClientID,
		UserID:           *deviceCode.UserID,
		Scope:            deviceCode.Scope,
		JTI:              jti,
		Issuer:           s.config.Issuer,
		AuthTime:         &deviceCode.CreatedAt,
		ExpiresAt:        time.Now().Add(time.Hour), // 1 hour
		RefreshExpiresAt: &refreshExpiresAt,
	}

	if err := s.tokenRepo.Create(ctx, oauthToken); err != nil {
		return nil, errs.DatabaseError("create token", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour in seconds
		RefreshToken: refreshToken,
		IDToken:      idToken,
		Scope:        deviceCode.Scope,
	}, nil
}

// =============================================================================
// JWKS & DISCOVERY
// =============================================================================

// GetJWKS returns the JSON Web Key Set for token verification
func (s *Service) GetJWKS() (*JWKS, error) {
	if s.jwksService == nil {
		return nil, errs.InternalError(fmt.Errorf("JWKS service not initialized"))
	}
	return s.jwksService.GetJWKS()
}

// =============================================================================
// KEY ROTATION
// =============================================================================

// StartKeyRotation begins automatic key rotation in the background
func (s *Service) StartKeyRotation() {
	if s.jwksService == nil {
		log.Println("JWKS service not initialized, skipping key rotation")
		return
	}

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

// =============================================================================
// USER INFO
// =============================================================================

// GetUserInfoFromToken retrieves user information based on an access token
func (s *Service) GetUserInfoFromToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	// Validate the access token
	token, err := s.tokenRepo.FindByAccessToken(ctx, accessToken)
	if err != nil {
		return nil, errs.DatabaseError("find token", err)
	}
	if token == nil {
		return nil, errs.UnauthorizedWithMessage("invalid access token")
	}

	if !token.IsValid() {
		return nil, errs.UnauthorizedWithMessage("token expired or revoked")
	}

	// Get user information
	if s.userSvc == nil {
		return nil, errs.InternalError(fmt.Errorf("user service not initialized"))
	}

	user, err := s.userSvc.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, errs.DatabaseError("find user", err)
	}

	// Build user info response based on scopes
	userInfo := map[string]interface{}{
		"sub": user.ID.String(),
	}

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

	return userInfo, nil
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

// userServiceAdapter adapts user.Service to UserService interface
type userServiceAdapter struct {
	svc *user.Service
}

func (a *userServiceAdapter) FindByID(ctx context.Context, userID xid.ID) (interface{}, error) {
	// FindByID returns a user object which we convert to interface{}
	return a.svc.FindByID(ctx, userID)
}

// =============================================================================
// Configuration and Status Getters
// =============================================================================

// GetConfig returns the service configuration as interface{} to avoid import cycles
func (s *Service) GetConfig() interface{} {
	return s.config
}

// GetDeviceFlowService returns the device flow service if enabled as interface{}
func (s *Service) GetDeviceFlowService() interface{} {
	return s.deviceFlowService
}

// GetConfigTyped returns the typed configuration (for internal use)
func (s *Service) GetConfigTyped() Config {
	return s.config
}

// deviceFlowServiceAdapter adapts deviceflow.Service to the interface expected by bridge
type deviceFlowServiceAdapter struct {
	svc *deviceflow.Service
}

func (a *deviceFlowServiceAdapter) CleanupExpiredCodes(ctx context.Context) (int, error) {
	if a.svc == nil {
		return 0, nil
	}
	return a.svc.CleanupExpiredCodes(ctx)
}

func (a *deviceFlowServiceAdapter) CleanupOldConsumedCodes(ctx context.Context, olderThan time.Duration) (int, error) {
	if a.svc == nil {
		return 0, nil
	}
	return a.svc.CleanupOldConsumedCodes(ctx, olderThan)
}

// GetCurrentKeyID returns the current JWT signing key ID
func (s *Service) GetCurrentKeyID() (string, error) {
	if s.jwksService == nil {
		return "", fmt.Errorf("JWKS service not initialized")
	}
	keyID := s.jwksService.GetCurrentKeyID()
	return keyID, nil
}

// GetLastKeyRotation returns the last key rotation time
func (s *Service) GetLastKeyRotation() time.Time {
	if s.jwksService == nil {
		return time.Time{}
	}
	return s.jwksService.GetLastRotation()
}

// RotateKeys manually triggers a JWT key rotation
func (s *Service) RotateKeys() error {
	if s.jwksService == nil {
		return fmt.Errorf("JWKS service not initialized")
	}
	return s.jwksService.RotateKeys()
}

// serviceAdapter wraps Service to provide bridge-compatible interface
type serviceAdapter struct {
	service *Service
}

// newServiceAdapter creates a new service adapter for bridge
func newServiceAdapter(service *Service) *serviceAdapter {
	return &serviceAdapter{service: service}
}

func (a *serviceAdapter) GetConfig() interface{} {
	return a.service.GetConfig()
}

func (a *serviceAdapter) GetCurrentKeyID() (string, error) {
	return a.service.GetCurrentKeyID()
}

func (a *serviceAdapter) GetLastKeyRotation() time.Time {
	return a.service.GetLastKeyRotation()
}

func (a *serviceAdapter) RotateKeys() error {
	return a.service.RotateKeys()
}

func (a *serviceAdapter) GetDeviceFlowService() interface{} {
	return a.service.GetDeviceFlowService()
}
