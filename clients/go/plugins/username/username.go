package username

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated username plugin

// Plugin implements the username plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new username plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "username"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// SignUp SignUp handles user registration with username and password
func (p *Plugin) SignUp(ctx context.Context, req *authsome.SignUpRequest) (*authsome.SignUpResponse, error) {
	path := "/username/signup"
	var result authsome.SignUpResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SignIn SignIn handles user authentication with username and password
func (p *Plugin) SignIn(ctx context.Context, req *authsome.SignInRequest) (*authsome.SignInResponse, error) {
	path := "/username/signin"
	var result authsome.SignInResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

