package emailverification

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated emailverification plugin

// Plugin implements the emailverification plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new emailverification plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "emailverification"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// Send Send handles manual verification email sending
POST /email-verification/send
func (p *Plugin) Send(ctx context.Context, req *authsome.SendRequest) (*authsome.SendResponse, error) {
	path := "/email-verification/send"
	var result authsome.SendResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Verify Verify handles email verification via token
GET /email-verification/verify?token=xyz
func (p *Plugin) Verify(ctx context.Context, req *authsome.VerifyRequest) (*authsome.VerifyResponse, error) {
	path := "/email-verification/verify"
	var result authsome.VerifyResponse
	err := p.client.Request(ctx, "GET", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Resend Resend handles resending verification email
POST /email-verification/resend
func (p *Plugin) Resend(ctx context.Context, req *authsome.ResendRequest) (*authsome.ResendResponse, error) {
	path := "/email-verification/resend"
	var result authsome.ResendResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

