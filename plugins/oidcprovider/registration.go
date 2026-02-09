package oidcprovider

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// RegistrationService handles RFC 7591 dynamic client registration operations.
type RegistrationService struct {
	clientRepo *repo.OAuthClientRepository
	config     Config
}

// NewRegistrationService creates a new client registration service.
func NewRegistrationService(clientRepo *repo.OAuthClientRepository, config Config) *RegistrationService {
	return &RegistrationService{
		clientRepo: clientRepo,
		config:     config,
	}
}

// RegisterClient implements RFC 7591 dynamic client registration.
func (s *RegistrationService) RegisterClient(ctx context.Context, req *ClientRegistrationRequest, appID, envID xid.ID, orgID *xid.ID) (*ClientRegistrationResponse, error) {
	// Validate the registration request
	if err := s.ValidateRegistrationRequest(req); err != nil {
		return nil, err
	}

	// Generate client credentials
	clientID := "client_" + xid.New().String()

	clientSecret, err := s.generateClientSecret()
	if err != nil {
		return nil, errs.InternalError(err)
	}

	// Set defaults for optional fields
	if req.ApplicationType == "" {
		req.ApplicationType = "web"
	}

	if req.TokenEndpointAuthMethod == "" {
		if req.ApplicationType == "native" || req.ApplicationType == "spa" {
			req.TokenEndpointAuthMethod = "none"
			req.RequirePKCE = true // Enforce PKCE for public clients
		} else {
			req.TokenEndpointAuthMethod = "client_secret_basic"
		}
	}

	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "refresh_token"}
	}

	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}

	// Ensure PKCE for public clients
	if req.TokenEndpointAuthMethod == "none" {
		req.RequirePKCE = true
	}

	// Create OAuth client record
	client := &schema.OAuthClient{
		ID:                      xid.New(),
		AppID:                   appID,
		EnvironmentID:           envID,
		OrganizationID:          orgID,
		Name:                    req.ClientName,
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		RedirectURI:             req.RedirectURIs[0], // Legacy single URI
		RedirectURIs:            req.RedirectURIs,
		PostLogoutRedirectURIs:  req.PostLogoutRedirectURIs,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		AllowedScopes:           s.parseScopes(req.Scope),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		ApplicationType:         req.ApplicationType,
		RequirePKCE:             req.RequirePKCE,
		RequireConsent:          req.RequireConsent,
		TrustedClient:           req.TrustedClient,
		LogoURI:                 req.LogoURI,
		PolicyURI:               req.PolicyURI,
		TosURI:                  req.TosURI,
		Contacts:                req.Contacts,
	}

	// Store client in database
	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, errs.DatabaseError("create client", err)
	}

	// Build registration response
	response := &ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientIDIssuedAt:        time.Now().Unix(),
		ClientSecretExpiresAt:   0, // Never expires
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		PostLogoutRedirectURIs:  req.PostLogoutRedirectURIs,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		ApplicationType:         req.ApplicationType,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		LogoURI:                 req.LogoURI,
		PolicyURI:               req.PolicyURI,
		TosURI:                  req.TosURI,
		Contacts:                req.Contacts,
		Scope:                   req.Scope,
	}

	// Don't return client secret for public clients
	if req.TokenEndpointAuthMethod == "none" {
		response.ClientSecret = ""
	}

	return response, nil
}

// ValidateRegistrationRequest validates a client registration request per RFC 7591.
func (s *RegistrationService) ValidateRegistrationRequest(req *ClientRegistrationRequest) error {
	// Validate required fields
	if req.ClientName == "" {
		return errs.RequiredField("client_name")
	}

	if len(req.RedirectURIs) == 0 {
		return errs.RequiredField("redirect_uris")
	}

	// Validate redirect URIs
	for _, uri := range req.RedirectURIs {
		if err := s.validateRedirectURI(uri, req.ApplicationType); err != nil {
			return err
		}
	}

	// Validate post-logout redirect URIs
	for _, uri := range req.PostLogoutRedirectURIs {
		if _, err := url.Parse(uri); err != nil {
			return errs.InvalidInput("post_logout_redirect_uris", "must be valid URLs")
		}
	}

	// Validate application type
	if req.ApplicationType != "" {
		validTypes := []string{"web", "native", "spa"}
		if !contains(validTypes, req.ApplicationType) {
			return errs.InvalidInput("application_type", "must be one of: web, native, spa")
		}
	}

	// Validate token endpoint auth method
	if req.TokenEndpointAuthMethod != "" {
		validMethods := []string{"client_secret_basic", "client_secret_post", "none"}
		if !contains(validMethods, req.TokenEndpointAuthMethod) {
			return errs.InvalidInput("token_endpoint_auth_method", "must be one of: client_secret_basic, client_secret_post, none")
		}

		// 'none' method only allowed for public clients
		if req.TokenEndpointAuthMethod == "none" {
			if req.ApplicationType == "web" {
				return errs.InvalidInput("token_endpoint_auth_method", "'none' not allowed for web applications")
			}
		}
	}

	// Validate grant types
	if len(req.GrantTypes) > 0 {
		validGrantTypes := []string{"authorization_code", "refresh_token", "client_credentials", "implicit"}
		for _, gt := range req.GrantTypes {
			if !contains(validGrantTypes, gt) {
				return errs.InvalidInput("grant_types", "invalid grant type: "+gt)
			}
		}

		// Validate grant type compatibility
		if contains(req.GrantTypes, "implicit") && contains(req.GrantTypes, "authorization_code") {
			return errs.InvalidInput("grant_types", "cannot mix implicit and authorization_code grant types")
		}
	}

	// Validate response types
	if len(req.ResponseTypes) > 0 {
		validResponseTypes := []string{"code", "token", "id_token", "code token", "code id_token", "token id_token", "code token id_token"}
		for _, rt := range req.ResponseTypes {
			if !contains(validResponseTypes, rt) {
				return errs.InvalidInput("response_types", "invalid response type: "+rt)
			}
		}

		// Ensure response types are compatible with grant types
		if contains(req.ResponseTypes, "token") || contains(req.ResponseTypes, "id_token") {
			if !contains(req.GrantTypes, "implicit") {
				return errs.InvalidInput("response_types", "token/id_token response types require implicit grant")
			}
		}
	}

	// Validate URLs
	if req.LogoURI != "" {
		if _, err := url.Parse(req.LogoURI); err != nil {
			return errs.InvalidInput("logo_uri", "must be a valid URL")
		}
	}

	if req.PolicyURI != "" {
		if _, err := url.Parse(req.PolicyURI); err != nil {
			return errs.InvalidInput("policy_uri", "must be a valid URL")
		}
	}

	if req.TosURI != "" {
		if _, err := url.Parse(req.TosURI); err != nil {
			return errs.InvalidInput("tos_uri", "must be a valid URL")
		}
	}

	return nil
}

// validateRedirectURI validates a redirect URI based on application type.
func (s *RegistrationService) validateRedirectURI(uri, applicationType string) error {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return errs.InvalidInput("redirect_uris", "must be valid URLs")
	}

	// Must be absolute URL
	if !parsedURL.IsAbs() {
		return errs.InvalidInput("redirect_uris", "must be absolute URLs")
	}

	// Application type specific validation
	switch applicationType {
	case "web":
		// Web apps must use HTTPS (except localhost for development)
		if parsedURL.Scheme != "https" {
			if parsedURL.Host != "localhost" && !strings.HasPrefix(parsedURL.Host, "localhost:") &&
				parsedURL.Host != "127.0.0.1" && !strings.HasPrefix(parsedURL.Host, "127.0.0.1:") {
				return errs.InvalidInput("redirect_uris", "web applications must use HTTPS")
			}
		}
		// No fragments allowed
		if parsedURL.Fragment != "" {
			return errs.InvalidInput("redirect_uris", "fragments not allowed in redirect URIs")
		}

	case "native":
		// Native apps can use custom schemes or localhost
		if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
			// If using HTTP(S), must be localhost
			if parsedURL.Host != "localhost" && !strings.HasPrefix(parsedURL.Host, "localhost:") &&
				parsedURL.Host != "127.0.0.1" && !strings.HasPrefix(parsedURL.Host, "127.0.0.1:") {
				return errs.InvalidInput("redirect_uris", "native apps can only use localhost for HTTP(S) URIs")
			}
		}

	case "spa":
		// SPAs must use HTTPS (except localhost)
		if parsedURL.Scheme != "https" {
			if parsedURL.Host != "localhost" && !strings.HasPrefix(parsedURL.Host, "localhost:") &&
				parsedURL.Host != "127.0.0.1" && !strings.HasPrefix(parsedURL.Host, "127.0.0.1:") {
				return errs.InvalidInput("redirect_uris", "single-page applications must use HTTPS")
			}
		}
	}

	return nil
}

// generateClientSecret generates a cryptographically secure client secret.
func (s *RegistrationService) generateClientSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return "secret_" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// parseScopes converts a space-separated scope string to a slice.
func (s *RegistrationService) parseScopes(scope string) []string {
	if scope == "" {
		return nil
	}

	return strings.Fields(scope)
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {

	return slices.Contains(slice, value)
}
