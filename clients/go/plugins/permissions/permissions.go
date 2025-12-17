package permissions

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated permissions plugin

// Plugin implements the permissions plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new permissions plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "permissions"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// MigrateAll MigrateAll migrates all RBAC policies to the permissions system
func (p *Plugin) MigrateAll(ctx context.Context, req *authsome.MigrateAllRequest) error {
	path := "/migrate/all"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// MigrateRoles MigrateRoles migrates role-based permissions to policies
func (p *Plugin) MigrateRoles(ctx context.Context) error {
	path := "/migrate/roles"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// PreviewConversion PreviewConversion previews the conversion of an RBAC policy
func (p *Plugin) PreviewConversion(ctx context.Context, req *authsome.PreviewConversionRequest) error {
	path := "/migrate/preview"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

