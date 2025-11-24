package emailotp

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated emailotp plugin

// Plugin implements the emailotp plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new emailotp plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "emailotp"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// Send Send handles sending of OTP to email
func (p *Plugin) Send(ctx context.Context, req *authsome.SendRequest) error {
	path := "/email-otp/send"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// Verify Verify checks the OTP and creates a session on success
func (p *Plugin) Verify(ctx context.Context, req *authsome.VerifyRequest) error {
	path := "/email-otp/verify"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

