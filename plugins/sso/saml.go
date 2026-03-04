package sso

import (
	"context"
	"fmt"
	"net/url"
)

// SAMLConfig configures a SAML 2.0 identity provider.
type SAMLConfig struct {
	// Name is the unique provider identifier (e.g. "okta-saml", "azure-ad-saml").
	Name string

	// MetadataURL is the URL to the IdP's SAML metadata XML.
	MetadataURL string

	// ACSURL is the Assertion Consumer Service URL (your app's callback).
	ACSURL string

	// EntityID is your Service Provider entity ID.
	EntityID string

	// SignRequests controls whether SAML AuthnRequests should be signed.
	SignRequests bool
}

// samlProvider implements the Provider interface for SAML 2.0.
type samlProvider struct {
	name        string
	metadataURL string
	acsURL      string
	entityID    string
}

// NewSAMLProvider creates a SAML 2.0 SSO provider.
//
// This implementation provides a lightweight SAML flow:
// - LoginURL redirects to the IdP's SSO endpoint
// - HandleCallback processes the SAMLResponse form parameter
//
// For production SAML deployments, consider using a full SAML library
// (e.g. crewjam/saml) for XML signature validation and assertion parsing.
func NewSAMLProvider(cfg SAMLConfig) Provider {
	return &samlProvider{
		name:        cfg.Name,
		metadataURL: cfg.MetadataURL,
		acsURL:      cfg.ACSURL,
		entityID:    cfg.EntityID,
	}
}

func (p *samlProvider) Name() string     { return p.name }
func (p *samlProvider) Protocol() string { return "saml" }

// LoginURL returns the IdP SSO URL with SAMLRequest parameters.
func (p *samlProvider) LoginURL(state string) (string, error) {
	// For SAML, the login URL is typically the IdP's SSO endpoint.
	// In a full implementation, this would construct a SAMLRequest.
	// Here we provide the metadata URL as a reference and the state
	// for relay state tracking.
	if p.metadataURL == "" {
		return "", fmt.Errorf("sso/saml: metadata URL is required")
	}

	// Construct IdP SSO URL using the metadata URL base.
	// The actual SSO endpoint would be parsed from the metadata XML.
	u, err := url.Parse(p.metadataURL)
	if err != nil {
		return "", fmt.Errorf("sso/saml: invalid metadata URL: %w", err)
	}

	// Use the IdP host for the SSO endpoint.
	ssoURL := fmt.Sprintf("%s://%s/saml/sso", u.Scheme, u.Host)

	params := url.Values{
		"SAMLRequest": {""},
		"RelayState":  {state},
	}

	return ssoURL + "?" + params.Encode(), nil
}

// HandleCallback processes the SAML assertion from the IdP.
func (p *samlProvider) HandleCallback(_ context.Context, params map[string]string) (*SSOUser, error) {
	samlResponse := params["SAMLResponse"]
	if samlResponse == "" {
		return nil, fmt.Errorf("sso/saml: missing SAMLResponse")
	}

	// In a production implementation, this would:
	// 1. Base64-decode the SAMLResponse
	// 2. Parse the XML assertion
	// 3. Validate the XML signature against the IdP's certificate
	// 4. Validate assertion conditions (audience, timestamps)
	// 5. Extract the NameID and attributes

	// For now, return an error indicating the SAML response needs to be
	// processed by a full SAML library. This provider serves as the
	// integration point and interface definition.
	return nil, fmt.Errorf("sso/saml: SAML response parsing requires a SAML library " +
		"(e.g. crewjam/saml); configure the provider with full SAML support")
}
