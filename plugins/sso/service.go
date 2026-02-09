package sso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	oidcsvc "github.com/xraph/authsome/plugins/sso/oidc"
	samsvc "github.com/xraph/authsome/plugins/sso/saml"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Service provides SSO operations (registration, callbacks, metadata).
type Service struct {
	mu        sync.RWMutex
	providers map[string]*schema.SSOProvider // cache keyed by ProviderID
	repo      *repo.SSOProviderRepository
	saml      *samsvc.Service
	oidc      *oidcsvc.Service

	// User provisioning dependencies
	userSvc    user.ServiceInterface
	sessionSvc session.ServiceInterface
	config     Config

	// OIDC state storage for PKCE and nonce
	stateStore *StateStore
}

func NewService(r *repo.SSOProviderRepository, cfg Config, userSvc user.ServiceInterface, sessionSvc session.ServiceInterface) *Service {
	// initialize SAML service with default dev URLs
	s := &Service{
		providers:  make(map[string]*schema.SSOProvider),
		repo:       r,
		saml:       samsvc.NewService(),
		oidc:       oidcsvc.NewService(),
		userSvc:    userSvc,
		sessionSvc: sessionSvc,
		config:     cfg,
		stateStore: NewStateStore(), // Initialize OIDC state store
	}
	_ = s.saml.NewServiceProvider("authsome-sp", "http://localhost/api/auth/sso/saml2/callback/default", "http://localhost/api/auth/sso/saml2/sp/metadata")

	return s
}

func (s *Service) RegisterProvider(ctx context.Context, p *schema.SSOProvider) error {
	// persist via repository and update cache
	if p.ID.IsZero() {
		p.ID = xid.New()
	}

	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}

	p.UpdatedAt = time.Now()
	if s.repo != nil {
		if err := s.repo.Upsert(ctx, p); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.providers[p.ProviderID] = p
	s.mu.Unlock()

	return nil
}

func (s *Service) GetProvider(ctx context.Context, providerID string) (*schema.SSOProvider, error) {
	// Try cache first (cache key includes tenant context)
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	cacheKey := fmt.Sprintf("%s:%s:%s:%s", appID.String(), envID.String(), orgID.String(), providerID)

	s.mu.RLock()
	p, ok := s.providers[cacheKey]
	s.mu.RUnlock()

	if ok {
		return p, nil
	}

	// Fetch from database with tenant filtering
	if s.repo != nil {
		dbp, err := s.repo.FindByProviderID(ctx, providerID)
		if err != nil {
			return nil, err
		}

		if dbp != nil {
			// Cache the result
			s.mu.Lock()
			s.providers[cacheKey] = dbp
			s.mu.Unlock()

			return dbp, nil
		}
	}

	return nil, errs.NotFound("SSO provider not found")
}

// SPMetadata returns a minimal placeholder SP metadata string.
func (s *Service) SPMetadata() string {
	if s.saml != nil {
		return s.saml.Metadata()
	}

	return "<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\" entityID=\"authsome-sp\"></EntityDescriptor>"
}

// InitiateSAMLLogin generates an AuthnRequest and returns the redirect URL.
func (s *Service) InitiateSAMLLogin(idpURL, relayState string) (string, string, error) {
	if s.saml == nil {
		return "", "", errs.InternalServerErrorWithMessage("SAML service not configured")
	}

	return s.saml.GenerateAuthnRequest(idpURL, relayState)
}

// ValidateSAMLResponse performs full SAML response validation.
func (s *Service) ValidateSAMLResponse(b64Response, expectedIssuer, relayState string) (*samsvc.SAMLAssertion, error) {
	if s.saml == nil {
		return nil, errs.InternalServerErrorWithMessage("SAML service not configured")
	}

	return s.saml.ParseAndValidateResponse(b64Response, expectedIssuer, relayState, nil)
}

// GeneratePKCEChallenge generates PKCE challenge for OIDC flow.
func (s *Service) GeneratePKCEChallenge() (*oidcsvc.PKCEChallenge, error) {
	return s.oidc.GeneratePKCEChallenge()
}

// ExchangeOIDCCode exchanges authorization code for tokens with PKCE support.
func (s *Service) ExchangeOIDCCode(ctx context.Context, provider *schema.SSOProvider, code, redirectURI, codeVerifier string) (*oidcsvc.OIDCTokenResponse, error) {
	if provider.Type != "oidc" {
		return nil, fmt.Errorf("provider %s is not an OIDC provider", provider.ProviderID)
	}

	if provider.OIDCIssuer == "" {
		return nil, errs.BadRequest("missing OIDC issuer in provider config")
	}

	if provider.OIDCClientID == "" {
		return nil, errs.BadRequest("missing OIDC client ID in provider config")
	}

	// Fetch token endpoint from OIDC discovery
	discovery, err := s.oidc.FetchDiscovery(ctx, provider.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}

	return s.oidc.ExchangeCodeForTokens(ctx, discovery.TokenEndpoint, provider.OIDCClientID, provider.OIDCClientSecret, code, redirectURI, codeVerifier)
}

// ValidateOIDCIDToken validates an OIDC ID token.
func (s *Service) ValidateOIDCIDToken(ctx context.Context, provider *schema.SSOProvider, idToken, nonce string) (*oidcsvc.OIDCUserInfo, error) {
	if provider.Type != "oidc" {
		return nil, fmt.Errorf("provider %s is not an OIDC provider", provider.ProviderID)
	}

	if provider.OIDCIssuer == "" {
		return nil, errs.BadRequest("missing OIDC issuer in provider config")
	}

	if provider.OIDCClientID == "" {
		return nil, errs.BadRequest("missing OIDC client ID in provider config")
	}

	// Construct JWKS URL from issuer
	jwksURL := provider.OIDCIssuer + "/.well-known/jwks.json"

	// Validate the ID token using JWKS
	claims, err := s.oidc.ValidateIDToken(ctx, idToken, jwksURL, provider.OIDCIssuer, provider.OIDCClientID, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}

	// Extract user info from claims
	userInfo := &oidcsvc.OIDCUserInfo{}
	if sub, ok := (*claims)["sub"].(string); ok {
		userInfo.Sub = sub
	}

	if name, ok := (*claims)["name"].(string); ok {
		userInfo.Name = name
	}

	if email, ok := (*claims)["email"].(string); ok {
		userInfo.Email = email
	}

	if emailVerified, ok := (*claims)["email_verified"].(bool); ok {
		userInfo.EmailVerified = emailVerified
	}

	if givenName, ok := (*claims)["given_name"].(string); ok {
		userInfo.GivenName = givenName
	}

	if familyName, ok := (*claims)["family_name"].(string); ok {
		userInfo.FamilyName = familyName
	}

	if picture, ok := (*claims)["picture"].(string); ok {
		userInfo.Picture = picture
	}

	if preferredUsername, ok := (*claims)["preferred_username"].(string); ok {
		userInfo.PreferredUsername = preferredUsername
	}

	return userInfo, nil
}

// GetOIDCUserInfo fetches user information from userinfo endpoint.
func (s *Service) GetOIDCUserInfo(ctx context.Context, provider *schema.SSOProvider, accessToken string) (*oidcsvc.OIDCUserInfo, error) {
	if provider.Type != "oidc" {
		return nil, fmt.Errorf("provider %s is not an OIDC provider", provider.ProviderID)
	}

	if provider.OIDCIssuer == "" {
		return nil, errs.BadRequest("missing OIDC issuer in provider config")
	}

	// Fetch userinfo endpoint from OIDC discovery
	discovery, err := s.oidc.FetchDiscovery(ctx, provider.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}

	// Some providers might not have userinfo endpoint
	if discovery.UserinfoEndpoint == "" {
		return nil, errs.BadRequest("provider does not support userinfo endpoint")
	}

	return s.oidc.GetUserInfo(ctx, discovery.UserinfoEndpoint, accessToken)
}

// =============================================================================
// OIDC LOGIN INITIATION
// =============================================================================

// InitiateOIDCLogin generates an OIDC authorization URL with PKCE.
func (s *Service) InitiateOIDCLogin(
	ctx context.Context,
	provider *schema.SSOProvider,
	redirectURI, state, nonce string,
) (string, *oidcsvc.PKCEChallenge, error) {
	if provider.Type != "oidc" {
		return "", nil, errs.BadRequest("provider is not OIDC")
	}

	if provider.OIDCIssuer == "" {
		return "", nil, errs.BadRequest("OIDC issuer not configured")
	}

	if provider.OIDCClientID == "" {
		return "", nil, errs.BadRequest("OIDC client ID not configured")
	}

	// Fetch authorization endpoint from OIDC discovery
	discovery, err := s.oidc.FetchDiscovery(ctx, provider.OIDCIssuer)
	if err != nil {
		return "", nil, errs.Wrap(err, errs.CodeOIDCError, "failed to fetch OIDC discovery", 500)
	}

	// Generate PKCE challenge for secure authorization code flow
	pkce, err := s.oidc.GeneratePKCEChallenge()
	if err != nil {
		return "", nil, errs.Wrap(err, errs.CodeOIDCError, "failed to generate PKCE challenge", 500)
	}

	// Use configured redirect URI if not provided
	if redirectURI == "" {
		redirectURI = provider.OIDCRedirectURI
	}

	// Default scope for OIDC
	scope := "openid profile email"

	// Build authorization URL using discovered endpoint
	authURL := fmt.Sprintf(
		"%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&state=%s&nonce=%s&code_challenge=%s&code_challenge_method=S256",
		discovery.AuthorizationEndpoint,
		provider.OIDCClientID,
		redirectURI,
		scope,
		state,
		nonce,
		pkce.CodeChallenge,
	)

	return authURL, pkce, nil
}

// =============================================================================
// JIT USER PROVISIONING
// =============================================================================

// ProvisionUser finds or creates a user from SSO assertion
// Implements Just-in-Time (JIT) user provisioning.
func (s *Service) ProvisionUser(
	ctx context.Context,
	email string,
	attributes map[string][]string,
	provider *schema.SSOProvider,
) (*user.User, error) {
	if email == "" {
		return nil, errs.RequiredField("email")
	}

	// Try to find existing user by email within app scope
	appID, _ := contexts.GetAppID(ctx)

	usr, err := s.userSvc.FindByAppAndEmail(ctx, appID, email)
	if err == nil && usr != nil {
		// User exists - update attributes if configured
		if s.config.UpdateAttributes {
			s.applyAttributeMapping(usr, attributes, provider)
			// Update user in database
			updateReq := &user.UpdateUserRequest{
				Name: &usr.Name,
			}
			if _, err := s.userSvc.Update(ctx, usr, updateReq); err != nil {
				// Log error but don't fail authentication
			}
		}

		return usr, nil
	}

	// User not found - check if auto-provisioning is enabled
	if !s.config.AutoProvision {
		return nil, errs.UserNotFound()
	}

	// Create new user from SSO attributes
	createReq := s.buildCreateUserRequest(ctx, email, attributes, provider)

	// Create user via user service
	createdUser, err := s.userSvc.Create(ctx, createReq)
	if err != nil {
		return nil, errs.Wrap(err, errs.CodeInternalError, "failed to provision user from SSO", 500)
	}

	return createdUser, nil
}

// applyAttributeMapping maps SSO attributes to user fields based on provider configuration.
func (s *Service) applyAttributeMapping(usr *user.User, attributes map[string][]string, provider *schema.SSOProvider) {
	// Use provider-specific attribute mapping if available, otherwise use config default
	mapping := provider.AttributeMapping
	if len(mapping) == 0 {
		mapping = s.config.AttributeMapping
	}

	// Apply attribute mapping
	for userField, ssoAttr := range mapping {
		if values, ok := attributes[ssoAttr]; ok && len(values) > 0 {
			value := values[0] // Take first value

			switch userField {
			case "name":
				usr.Name = value
			case "email":
				// Don't override email - it's the identifier
			case "image":
				usr.Image = value
				// Add more field mappings as needed
			}
		}
	}
}

// buildCreateUserRequest creates a CreateUserRequest from SSO attributes.
func (s *Service) buildCreateUserRequest(
	ctx context.Context,
	email string,
	attributes map[string][]string,
	provider *schema.SSOProvider,
) *user.CreateUserRequest {
	// Extract context for multi-tenant scoping
	appID, _ := contexts.GetAppID(ctx)

	// Build create request with basic info
	req := &user.CreateUserRequest{
		AppID:    appID,
		Email:    email,
		Password: generateRandomPassword(), // Generate random password for SSO users
		Name:     "",
	}

	// Extract name from attributes
	// Try common attribute names
	nameAttrs := []string{
		"name",
		"displayName",
		"cn",
		"commonName",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
	}

	for _, attr := range nameAttrs {
		if values, ok := attributes[attr]; ok && len(values) > 0 {
			req.Name = values[0]

			break
		}
	}

	// If no name found, use email prefix
	if req.Name == "" {
		if idx := strings.Index(email, "@"); idx > 0 {
			req.Name = email[:idx]
		} else {
			req.Name = email
		}
	}

	return req
}

// generateRandomPassword generates a secure random password for SSO users.
func generateRandomPassword() string {
	// SSO users don't need to know their password
	// Generate a strong random password
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return xid.New().String() + xid.New().String()
	}

	return base64.URLEncoding.EncodeToString(bytes)
}

// CreateSSOSession creates a session after successful SSO authentication.
func (s *Service) CreateSSOSession(
	ctx context.Context,
	userID xid.ID,
	provider *schema.SSOProvider,
) (*session.Session, string, error) {
	if s.sessionSvc == nil {
		return nil, "", errs.InternalServerErrorWithMessage("session service not available")
	}

	// Extract app context
	appID, _ := contexts.GetAppID(ctx)

	// Extract OrganizationID from context (optional)
	var organizationID *xid.ID
	if orgID, ok := contexts.GetOrganizationID(ctx); ok && !orgID.IsNil() {
		organizationID = &orgID
	}

	// Extract EnvironmentID from context (optional)
	var environmentID *xid.ID
	if envID, ok := contexts.GetEnvironmentID(ctx); ok && !envID.IsNil() {
		environmentID = &envID
	}

	// Create session request
	createReq := &session.CreateSessionRequest{
		AppID:          appID,
		EnvironmentID:  environmentID,
		OrganizationID: organizationID,
		UserID:         userID,
		IPAddress:      "", // Would be extracted from HTTP request
		UserAgent:      "", // Would be extracted from HTTP request
	}

	// Create session via session service
	sess, err := s.sessionSvc.Create(ctx, createReq)
	if err != nil {
		return nil, "", errs.Wrap(err, errs.CodeInternalError, "failed to create SSO session", 500)
	}

	// Session token is in the Token field
	token := sess.Token
	if token == "" {
		// Generate a secure random token if not provided by session service
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			return nil, "", errs.Wrap(err, errs.CodeInternalError, "failed to generate session token", 500)
		}

		token = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return sess, token, nil
}
