package magiclink

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated magiclink plugin

// Plugin implements the magiclink plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new magiclink plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "magiclink"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

func (p *Plugin) Send(ctx context.Context, req *authsome.SendRequest) error {
	path := "/magic-link/send"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

func (p *Plugin) Verify(ctx context.Context) error {
	path := "/magic-link/verify"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

