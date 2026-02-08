package secrets

import (
	"context"
	"net/url"

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
func (p *Plugin) List(ctx context.Context, req *authsome.ListRequest) error {
	path := "/secrets"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// Create Create handles POST /secrets
func (p *Plugin) Create(ctx context.Context, req *authsome.CreateRequest) error {
	path := "/secrets"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// Get Get handles GET /secrets/:id
func (p *Plugin) Get(ctx context.Context, req *authsome.GetRequest, id xid.ID) error {
	path := "/secrets/:id"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// GetValue GetValue handles GET /secrets/:id/value
func (p *Plugin) GetValue(ctx context.Context, req *authsome.GetValueRequest, id xid.ID) error {
	path := "/secrets/:id/value"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// Update Update handles PUT /secrets/:id
func (p *Plugin) Update(ctx context.Context, req *authsome.UpdateRequest, id xid.ID) error {
	path := "/secrets/:id"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// Delete Delete handles DELETE /secrets/:id
func (p *Plugin) Delete(ctx context.Context, req *authsome.DeleteRequest, id xid.ID) (*authsome.DeleteResponse, error) {
	path := "/secrets/:id"
	var result authsome.DeleteResponse
	err := p.client.Request(ctx, "DELETE", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetByPath GetByPath handles GET /secrets/path/*path
func (p *Plugin) GetByPath(ctx context.Context) (*authsome.GetByPathResponse, error) {
	path := "/secrets/path/*path"
	var result authsome.GetByPathResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetVersions GetVersions handles GET /secrets/:id/versions
func (p *Plugin) GetVersions(ctx context.Context, req *authsome.GetVersionsRequest, id xid.ID) error {
	path := "/secrets/:id/versions"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

// Rollback Rollback handles POST /secrets/:id/rollback/:version
func (p *Plugin) Rollback(ctx context.Context, req *authsome.RollbackRequest, id xid.ID, version int) error {
	path := "/secrets/:id/rollback/:version"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetStats GetStats handles GET /secrets/stats
func (p *Plugin) GetStats(ctx context.Context) error {
	path := "/secrets/stats"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetTree GetTree handles GET /secrets/tree
func (p *Plugin) GetTree(ctx context.Context, req *authsome.GetTreeRequest) error {
	path := "/secrets/tree"
	err := p.client.Request(ctx, "GET", path, req, nil, false)
	return err
}

