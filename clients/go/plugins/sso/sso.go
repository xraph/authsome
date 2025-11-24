package sso

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated sso plugin

// Plugin implements the sso plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new sso plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "sso"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// RegisterProvider RegisterProvider registers a new SSO provider (SAML or OIDC)
func (p *Plugin) RegisterProvider(ctx context.Context, req *authsome.RegisterProviderRequest) (*authsome.RegisterProviderResponse, error) {
	path := "/provider/register"
	var result authsome.RegisterProviderResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SAMLSPMetadata SAMLSPMetadata returns Service Provider metadata
func (p *Plugin) SAMLSPMetadata(ctx context.Context) (*authsome.SAMLSPMetadataResponse, error) {
	path := "/saml2/sp/metadata"
	var result authsome.SAMLSPMetadataResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SAMLLogin SAMLLogin initiates SAML authentication by generating AuthnRequest
func (p *Plugin) SAMLLogin(ctx context.Context, req *authsome.SAMLLoginRequest) (*authsome.SAMLLoginResponse, error) {
	path := "/saml2/login/:providerId"
	var result authsome.SAMLLoginResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SAMLCallback SAMLCallback handles SAML response callback and provisions user
func (p *Plugin) SAMLCallback(ctx context.Context) (*authsome.SAMLCallbackResponse, error) {
	path := "/saml2/callback/:providerId"
	var result authsome.SAMLCallbackResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// OIDCLogin OIDCLogin initiates OIDC authentication flow with PKCE
func (p *Plugin) OIDCLogin(ctx context.Context, req *authsome.OIDCLoginRequest) (*authsome.OIDCLoginResponse, error) {
	path := "/oidc/login/:providerId"
	var result authsome.OIDCLoginResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// OIDCCallback OIDCCallback handles OIDC callback and provisions user
func (p *Plugin) OIDCCallback(ctx context.Context) (*authsome.OIDCCallbackResponse, error) {
	path := "/oidc/callback/:providerId"
	var result authsome.OIDCCallbackResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

