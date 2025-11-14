package sso

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
	oidcsvc "github.com/xraph/authsome/plugins/sso/oidc"
	samsvc "github.com/xraph/authsome/plugins/sso/saml"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Service provides SSO operations (registration, callbacks, metadata)
type Service struct {
	mu        sync.RWMutex
	providers map[string]*schema.SSOProvider // cache keyed by ProviderID
	repo      *repo.SSOProviderRepository
	saml      *samsvc.Service
	oidc      *oidcsvc.Service
}

func NewService(r *repo.SSOProviderRepository) *Service {
	// initialize SAML service with default dev URLs
	s := &Service{
		providers: make(map[string]*schema.SSOProvider),
		repo:      r,
		saml:      samsvc.NewService(),
		oidc:      oidcsvc.NewService(),
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

func (s *Service) GetProvider(providerID string) (*schema.SSOProvider, bool) {
	s.mu.RLock()
	p, ok := s.providers[providerID]
	s.mu.RUnlock()
	if ok {
		return p, true
	}
	if s.repo != nil {
		if dbp, err := s.repo.FindByProviderID(context.Background(), providerID); err == nil && dbp != nil {
			s.mu.Lock()
			s.providers[providerID] = dbp
			s.mu.Unlock()
			return dbp, true
		}
	}
	return nil, false
}

// SPMetadata returns a minimal placeholder SP metadata string
func (s *Service) SPMetadata() string {
	if s.saml != nil {
		return s.saml.Metadata()
	}
	return "<EntityDescriptor xmlns=\"urn:oasis:names:tc:SAML:2.0:metadata\" entityID=\"authsome-sp\"></EntityDescriptor>"
}

// InitiateSAMLLogin generates an AuthnRequest and returns the redirect URL
func (s *Service) InitiateSAMLLogin(idpURL, relayState string) (string, string, error) {
	if s.saml == nil {
		return "", "", fmt.Errorf("SAML service not configured")
	}
	return s.saml.GenerateAuthnRequest(idpURL, relayState)
}

// ValidateSAMLResponse performs full SAML response validation
func (s *Service) ValidateSAMLResponse(b64Response, expectedIssuer, relayState string) (*samsvc.SAMLAssertion, error) {
	if s.saml == nil {
		return nil, fmt.Errorf("SAML service not configured")
	}
	return s.saml.ParseAndValidateResponse(b64Response, expectedIssuer, relayState, nil)
}

// GeneratePKCEChallenge generates PKCE challenge for OIDC flow
func (s *Service) GeneratePKCEChallenge() (*oidcsvc.PKCEChallenge, error) {
	return s.oidc.GeneratePKCEChallenge()
}

// ExchangeOIDCCode exchanges authorization code for tokens with PKCE support
func (s *Service) ExchangeOIDCCode(ctx context.Context, provider *schema.SSOProvider, code, redirectURI, codeVerifier string) (*oidcsvc.OIDCTokenResponse, error) {
	if provider.Type != "oidc" {
		return nil, fmt.Errorf("provider %s is not an OIDC provider", provider.ProviderID)
	}

	// For now, construct token endpoint from issuer
	// TODO: Fetch from OIDC discovery endpoint
	tokenEndpoint := provider.OIDCIssuer + "/token"
	if provider.OIDCIssuer == "" {
		return nil, fmt.Errorf("missing OIDC issuer in provider config")
	}

	if provider.OIDCClientID == "" {
		return nil, fmt.Errorf("missing OIDC client ID in provider config")
	}

	return s.oidc.ExchangeCodeForTokens(ctx, tokenEndpoint, provider.OIDCClientID, provider.OIDCClientSecret, code, redirectURI, codeVerifier)
}

// ValidateOIDCIDToken validates an OIDC ID token
func (s *Service) ValidateOIDCIDToken(ctx context.Context, provider *schema.SSOProvider, idToken, nonce string) (*oidcsvc.OIDCUserInfo, error) {
	if provider.Type != "oidc" {
		return nil, fmt.Errorf("provider %s is not an OIDC provider", provider.ProviderID)
	}

	if provider.OIDCIssuer == "" {
		return nil, fmt.Errorf("missing OIDC issuer in provider config")
	}

	if provider.OIDCClientID == "" {
		return nil, fmt.Errorf("missing OIDC client ID in provider config")
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

// GetOIDCUserInfo fetches user information from userinfo endpoint
func (s *Service) GetOIDCUserInfo(ctx context.Context, provider *schema.SSOProvider, accessToken string) (*oidcsvc.OIDCUserInfo, error) {
	if provider.Type != "oidc" {
		return nil, fmt.Errorf("provider %s is not an OIDC provider", provider.ProviderID)
	}

	// For now, construct userinfo endpoint from issuer
	// TODO: Fetch from OIDC discovery endpoint
	userinfoEndpoint := provider.OIDCIssuer + "/userinfo"
	if provider.OIDCIssuer == "" {
		return nil, fmt.Errorf("missing OIDC issuer in provider config")
	}

	return s.oidc.GetUserInfo(ctx, userinfoEndpoint, accessToken)
}
