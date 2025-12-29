package social

import (
	"context"
	"net/url"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated social plugin

// Plugin implements the social plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new social plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "social"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// SignIn SignIn initiates OAuth flow for sign-in
POST /api/auth/signin/social
func (p *Plugin) SignIn(ctx context.Context, req *authsome.SignInRequest) (*authsome.SignInResponse, error) {
	path := "/signin/social"
	var result authsome.SignInResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Callback Callback handles OAuth provider callback
GET /api/auth/callback/:provider
func (p *Plugin) Callback(ctx context.Context, provider string) (*authsome.CallbackResponse, error) {
	path := "/callback/:provider"
	var result authsome.CallbackResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// LinkAccount LinkAccount links a social provider to the current user
POST /api/auth/account/link
func (p *Plugin) LinkAccount(ctx context.Context, req *authsome.LinkAccountRequest) (*authsome.LinkAccountResponse, error) {
	path := "/account/link"
	var result authsome.LinkAccountResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UnlinkAccount UnlinkAccount unlinks a social provider from the current user
DELETE /api/auth/account/unlink/:provider
func (p *Plugin) UnlinkAccount(ctx context.Context, provider string) error {
	path := "/account/unlink/:provider"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ListProviders ListProviders returns available OAuth providers
GET /api/auth/providers
func (p *Plugin) ListProviders(ctx context.Context) (*authsome.ListProvidersResponse, error) {
	path := "/providers"
	var result authsome.ListProvidersResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

