package oidcprovider

import (
	"context"
	"net/url"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated oidcprovider plugin

// Plugin implements the oidcprovider plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new oidcprovider plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "oidcprovider"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// RegisterClient RegisterClient handles dynamic client registration (admin only)
func (p *Plugin) RegisterClient(ctx context.Context, req *authsome.RegisterClientRequest) (*authsome.RegisterClientResponse, error) {
	path := "/oauth2/register"
	var result authsome.RegisterClientResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListClients ListClients lists all OAuth clients for the current app/env/org
func (p *Plugin) ListClients(ctx context.Context) (*authsome.ListClientsResponse, error) {
	path := "/oauth2/clients"
	var result authsome.ListClientsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetClient GetClient retrieves detailed information about an OAuth client
func (p *Plugin) GetClient(ctx context.Context, clientId string) (*authsome.GetClientResponse, error) {
	path := "/oauth2/clients/:clientId"
	var result authsome.GetClientResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateClient UpdateClient updates an existing OAuth client
func (p *Plugin) UpdateClient(ctx context.Context, req *authsome.UpdateClientRequest, clientId string) (*authsome.UpdateClientResponse, error) {
	path := "/oauth2/clients/:clientId"
	var result authsome.UpdateClientResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteClient DeleteClient deletes an OAuth client
func (p *Plugin) DeleteClient(ctx context.Context, clientId string) error {
	path := "/oauth2/clients/:clientId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// Discovery Discovery handles the OIDC discovery endpoint (.well-known/openid-configuration)
func (p *Plugin) Discovery(ctx context.Context) (*authsome.DiscoveryResponse, error) {
	path := "/oauth2/.well-known/openid-configuration"
	var result authsome.DiscoveryResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// JWKS JWKS returns the JSON Web Key Set
func (p *Plugin) JWKS(ctx context.Context) (*authsome.JWKSResponse, error) {
	path := "/oauth2/jwks"
	var result authsome.JWKSResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Authorize Authorize handles OAuth2/OIDC authorization requests
func (p *Plugin) Authorize(ctx context.Context) error {
	path := "/oauth2/authorize"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// HandleConsent HandleConsent processes the consent form submission
func (p *Plugin) HandleConsent(ctx context.Context, req *authsome.HandleConsentRequest) error {
	path := "/oauth2/consent"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// Token Token handles the token endpoint
func (p *Plugin) Token(ctx context.Context, req *authsome.TokenRequest) (*authsome.TokenResponse, error) {
	path := "/oauth2/token"
	var result authsome.TokenResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UserInfo UserInfo returns user information based on the access token
func (p *Plugin) UserInfo(ctx context.Context) (*authsome.UserInfoResponse, error) {
	path := "/oauth2/userinfo"
	var result authsome.UserInfoResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// IntrospectToken IntrospectToken handles token introspection requests
func (p *Plugin) IntrospectToken(ctx context.Context, req *authsome.IntrospectTokenRequest) (*authsome.IntrospectTokenResponse, error) {
	path := "/oauth2/introspect"
	var result authsome.IntrospectTokenResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeToken RevokeToken handles token revocation requests
func (p *Plugin) RevokeToken(ctx context.Context, req *authsome.RevokeTokenRequest) error {
	path := "/oauth2/revoke"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// DeviceAuthorize DeviceAuthorize initiates the device authorization flow
func (p *Plugin) DeviceAuthorize(ctx context.Context, req *authsome.DeviceAuthorizeRequest) (*authsome.DeviceAuthorizeResponse, error) {
	path := "/oauth2/device_authorization"
	var result authsome.DeviceAuthorizeResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeviceCodeEntry DeviceCodeEntry shows the device code entry form
func (p *Plugin) DeviceCodeEntry(ctx context.Context) (*authsome.DeviceCodeEntryResponse, error) {
	path := "/oauth2/device"
	var result authsome.DeviceCodeEntryResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeviceVerify DeviceVerify verifies the user code and shows the consent screen
func (p *Plugin) DeviceVerify(ctx context.Context, req *authsome.DeviceVerifyRequest) (*authsome.DeviceVerifyResponse, error) {
	path := "/oauth2/device/verify"
	var result authsome.DeviceVerifyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeviceAuthorizeDecision DeviceAuthorizeDecision handles the user's authorization decision
func (p *Plugin) DeviceAuthorizeDecision(ctx context.Context, req *authsome.DeviceAuthorizeDecisionRequest) (*authsome.DeviceAuthorizeDecisionResponse, error) {
	path := "/oauth2/device/authorize"
	var result authsome.DeviceAuthorizeDecisionResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

