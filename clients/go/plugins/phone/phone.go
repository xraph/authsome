package phone

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated phone plugin

// Plugin implements the phone plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new phone plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "phone"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// SendCode SendCode handles sending of verification code via SMS
func (p *Plugin) SendCode(ctx context.Context, req *authsome.SendCodeRequest) error {
	path := "/phone/send-code"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// Verify Verify checks the code and creates a session on success
func (p *Plugin) Verify(ctx context.Context, req *authsome.VerifyRequest) (*authsome.VerifyResponse, error) {
	path := "/phone/verify"
	var result authsome.VerifyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SignIn SignIn aliases to Verify for convenience
func (p *Plugin) SignIn(ctx context.Context, req *authsome.SignInRequest) (*authsome.SignInResponse, error) {
	path := "/phone/signin"
	var result authsome.SignInResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

