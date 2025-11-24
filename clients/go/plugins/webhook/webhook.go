package webhook

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated webhook plugin

// Plugin implements the webhook plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new webhook plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "webhook"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// Create Create a webhook
func (p *Plugin) Create(ctx context.Context, req *authsome.CreateRequest) (*authsome.CreateResponse, error) {
	path := "/api/auth/webhooks"
	var result authsome.CreateResponse
	err := p.client.Request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// List List webhooks
func (p *Plugin) List(ctx context.Context) (*authsome.ListResponse, error) {
	path := "/api/auth/webhooks"
	var result authsome.ListResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Update Update a webhook
func (p *Plugin) Update(ctx context.Context, req *authsome.UpdateRequest) (*authsome.UpdateResponse, error) {
	path := "/api/auth/webhooks/update"
	var result authsome.UpdateResponse
	err := p.client.Request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete Delete a webhook
func (p *Plugin) Delete(ctx context.Context, req *authsome.DeleteRequest) (*authsome.DeleteResponse, error) {
	path := "/api/auth/webhooks/delete"
	var result authsome.DeleteResponse
	err := p.client.Request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

