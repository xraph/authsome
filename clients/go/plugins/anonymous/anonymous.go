package anonymous

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated anonymous plugin

// Plugin implements the anonymous plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new anonymous plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "anonymous"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// SignIn SignIn creates a guest user and session
func (p *Plugin) SignIn(ctx context.Context) (*authsome.SignInResponse, error) {
	path := "/anonymous/signin"
	var result authsome.SignInResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Link Link upgrades an anonymous session to a real account
func (p *Plugin) Link(ctx context.Context, req *authsome.LinkRequest) (*authsome.LinkResponse, error) {
	path := "/anonymous/link"
	var result authsome.LinkResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

