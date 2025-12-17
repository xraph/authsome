package secrets

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated secrets plugin

// Plugin implements the secrets plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new secrets plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "secrets"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// List List handles GET /secrets
func (p *Plugin) List(ctx context.Context) error {
	path := "/list"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// Create Create handles POST /secrets
func (p *Plugin) Create(ctx context.Context) (*authsome.CreateResponse, error) {
	path := "/create"
	var result authsome.CreateResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get Get handles GET /secrets/:id
func (p *Plugin) Get(ctx context.Context) (*authsome.GetResponse, error) {
	path := "/:id"
	var result authsome.GetResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetValue GetValue handles GET /secrets/:id/value
func (p *Plugin) GetValue(ctx context.Context) error {
	path := "/:id/value"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// Update Update handles PUT /secrets/:id
func (p *Plugin) Update(ctx context.Context) (*authsome.UpdateResponse, error) {
	path := "/:id"
	var result authsome.UpdateResponse
	err := p.client.Request(ctx, "PUT", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete Delete handles DELETE /secrets/:id
func (p *Plugin) Delete(ctx context.Context) (*authsome.DeleteResponse, error) {
	path := "/:id"
	var result authsome.DeleteResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByPath GetByPath handles GET /secrets/path/*path
func (p *Plugin) GetByPath(ctx context.Context) (*authsome.GetByPathResponse, error) {
	path := "/path/*path"
	var result authsome.GetByPathResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetVersions GetVersions handles GET /secrets/:id/versions
func (p *Plugin) GetVersions(ctx context.Context) error {
	path := "/:id/versions"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// Rollback Rollback handles POST /secrets/:id/rollback/:version
func (p *Plugin) Rollback(ctx context.Context, req *authsome.RollbackRequest) (*authsome.RollbackResponse, error) {
	path := "/:id/rollback/:version"
	var result authsome.RollbackResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetStats GetStats handles GET /secrets/stats
func (p *Plugin) GetStats(ctx context.Context) error {
	path := "/stats"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetTree GetTree handles GET /secrets/tree
func (p *Plugin) GetTree(ctx context.Context) error {
	path := "/tree"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

