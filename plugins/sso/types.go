package sso

import (
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
)

// =============================================================================
// REQUEST TYPES
// =============================================================================

// RegisterProviderRequest represents a request to register a new SSO provider
type RegisterProviderRequest struct {
	ProviderID string `json:"providerId" validate:"required"`
	Type       string `json:"type" validate:"required,oneof=saml oidc"`
	Domain     string `json:"domain"`

	// Attribute mapping from user fields to SSO attribute names
	AttributeMapping map[string]string `json:"attributeMapping"`

	// SAML configuration
	SAMLEntryPoint string `json:"samlEntryPoint"`
	SAMLIssuer     string `json:"samlIssuer"`
	SAMLCert       string `json:"samlCert"`

	// OIDC configuration
	OIDCClientID     string `json:"oidcClientID"`
	OIDCClientSecret string `json:"oidcClientSecret"`
	OIDCIssuer       string `json:"oidcIssuer"`
	OIDCRedirectURI  string `json:"oidcRedirectURI"`
}

// OIDCLoginRequest represents a request to initiate OIDC login
type OIDCLoginRequest struct {
	RedirectURI string `json:"redirectUri"`
	State       string `json:"state"`
	Nonce       string `json:"nonce"`
	Scope       string `json:"scope"` // Optional custom scope
}

// SAMLLoginRequest represents a request to initiate SAML login
type SAMLLoginRequest struct {
	RelayState string `json:"relayState"`
}

// DiscoverProviderRequest represents a request to discover SSO provider by email
type DiscoverProviderRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// =============================================================================
// RESPONSE TYPES
// =============================================================================

// SSOAuthResponse represents a successful SSO authentication response
type SSOAuthResponse struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
	Token   string           `json:"token"`
}

// ProviderRegisteredResponse represents a successful provider registration
type ProviderRegisteredResponse struct {
	ProviderID string `json:"providerId"`
	Type       string `json:"type"`
	Status     string `json:"status"`
}

// OIDCLoginResponse represents the response to OIDC login initiation
type OIDCLoginResponse struct {
	AuthURL    string `json:"authUrl"`
	State      string `json:"state"`
	Nonce      string `json:"nonce"`
	ProviderID string `json:"providerId"`
}

// SAMLLoginResponse represents the response to SAML login initiation
type SAMLLoginResponse struct {
	RedirectURL string `json:"redirectUrl"`
	RequestID   string `json:"requestId"`
	ProviderID  string `json:"providerId"`
}

// MetadataResponse represents SAML SP metadata
type MetadataResponse struct {
	Metadata string `json:"metadata"`
}

// ProviderDiscoveredResponse represents the result of provider discovery
type ProviderDiscoveredResponse struct {
	Found      bool   `json:"found"`
	ProviderID string `json:"providerId,omitempty"`
	Type       string `json:"type,omitempty"`
}

// ProviderListResponse represents a list of SSO providers
type ProviderListResponse struct {
	Providers []ProviderInfo `json:"providers"`
	Total     int            `json:"total"`
}

// ProviderInfo represents basic SSO provider information
type ProviderInfo struct {
	ProviderID string `json:"providerId"`
	Type       string `json:"type"`
	Domain     string `json:"domain,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// ProviderDetailResponse represents detailed SSO provider information
type ProviderDetailResponse struct {
	ProviderID       string            `json:"providerId"`
	Type             string            `json:"type"`
	Domain           string            `json:"domain,omitempty"`
	AttributeMapping map[string]string `json:"attributeMapping,omitempty"`

	// SAML info (without sensitive data)
	SAMLEntryPoint string `json:"samlEntryPoint,omitempty"`
	SAMLIssuer     string `json:"samlIssuer,omitempty"`
	HasSAMLCert    bool   `json:"hasSamlCert,omitempty"`

	// OIDC info (without sensitive data)
	OIDCClientID    string `json:"oidcClientID,omitempty"`
	OIDCIssuer      string `json:"oidcIssuer,omitempty"`
	OIDCRedirectURI string `json:"oidcRedirectURI,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
