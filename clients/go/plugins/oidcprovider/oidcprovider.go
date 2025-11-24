package oidcprovider

import (
	"context"

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
func (p *Plugin) RegisterClient(ctx context.Context, req *authsome.RegisterClientRequest) error {
	path := "/register"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListClients ListClients lists all OAuth clients for the current app/env/org
func (p *Plugin) ListClients(ctx context.Context) error {
	path := "/listclients"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetClient GetClient retrieves detailed information about an OAuth client
func (p *Plugin) GetClient(ctx context.Context) error {
	path := "/:clientId"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateClient UpdateClient updates an existing OAuth client
func (p *Plugin) UpdateClient(ctx context.Context, req *authsome.UpdateClientRequest) error {
	path := "/:clientId"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// DeleteClient DeleteClient deletes an OAuth client
func (p *Plugin) DeleteClient(ctx context.Context) error {
	path := "/:clientId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// Discovery Discovery handles the OIDC discovery endpoint (.well-known/openid-configuration)
func (p *Plugin) Discovery(ctx context.Context) error {
	path := "/.well-known/openid-configuration"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// JWKS JWKS returns the JSON Web Key Set
func (p *Plugin) JWKS(ctx context.Context) error {
	path := "/jwks"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// Authorize Authorize handles OAuth2/OIDC authorization requests
func (p *Plugin) Authorize(ctx context.Context) error {
	path := "/authorize"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// HandleConsent HandleConsent processes the consent form submission
func (p *Plugin) HandleConsent(ctx context.Context) error {
	path := "/consent"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// Token Token handles the token endpoint
func (p *Plugin) Token(ctx context.Context, req *authsome.TokenRequest) error {
	path := "/token"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// UserInfo UserInfo returns user information based on the access token
func (p *Plugin) UserInfo(ctx context.Context) error {
	path := "/userinfo"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// IntrospectToken IntrospectToken handles token introspection requests
func (p *Plugin) IntrospectToken(ctx context.Context) error {
	path := "/introspect"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// RevokeToken RevokeToken handles token revocation requests
func (p *Plugin) RevokeToken(ctx context.Context) error {
	path := "/revoke"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

