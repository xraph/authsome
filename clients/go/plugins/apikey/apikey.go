package apikey

import (
	"context"
	"net/url"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated apikey plugin

// Plugin implements the apikey plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new apikey plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "apikey"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateAPIKey CreateAPIKey handles API key creation
func (p *Plugin) CreateAPIKey(ctx context.Context, req *authsome.CreateAPIKeyRequest) (*authsome.CreateAPIKeyResponse, error) {
	path := "/api-keys/createapikey"
	var result authsome.CreateAPIKeyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RotateAPIKey RotateAPIKey handles API key rotation
func (p *Plugin) RotateAPIKey(ctx context.Context, req *authsome.RotateAPIKeyRequest, id xid.ID) (*authsome.RotateAPIKeyResponse, error) {
	path := "/api-keys/:id/rotate"
	var result authsome.RotateAPIKeyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateAPIKey CreateAPIKey handles POST /api-keys
func (p *Plugin) CreateAPIKey(ctx context.Context, req *authsome.CreateAPIKeyRequest) (*authsome.CreateAPIKeyResponse, error) {
	path := "/api-keys/createapikey"
	var result authsome.CreateAPIKeyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAPIKeys ListAPIKeys handles GET /api-keys
func (p *Plugin) ListAPIKeys(ctx context.Context, req *authsome.ListAPIKeysRequest) error {
	path := "/api-keys/listapikeys"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// GetAPIKey GetAPIKey handles GET /api-keys/:id
func (p *Plugin) GetAPIKey(ctx context.Context, req *authsome.GetAPIKeyRequest, id xid.ID) error {
	path := "/api-keys/:id"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// UpdateAPIKey UpdateAPIKey handles PATCH /api-keys/:id
func (p *Plugin) UpdateAPIKey(ctx context.Context, req *authsome.UpdateAPIKeyRequest, id xid.ID) error {
	path := "/api-keys/:id"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// DeleteAPIKey DeleteAPIKey handles DELETE /api-keys/:id
func (p *Plugin) DeleteAPIKey(ctx context.Context, req *authsome.DeleteAPIKeyRequest, id xid.ID) error {
	path := "/api-keys/:id"
	err := p.client.Request(ctx, "DELETE", path, req, nil, false)
	return err
}

// RotateAPIKey RotateAPIKey handles POST /api-keys/:id/rotate
func (p *Plugin) RotateAPIKey(ctx context.Context, req *authsome.RotateAPIKeyRequest, id xid.ID) (*authsome.RotateAPIKeyResponse, error) {
	path := "/api-keys/:id/rotate"
	var result authsome.RotateAPIKeyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// VerifyAPIKey VerifyAPIKey handles POST /api-keys/verify
func (p *Plugin) VerifyAPIKey(ctx context.Context, req *authsome.VerifyAPIKeyRequest) error {
	path := "/api-keys/verify"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

