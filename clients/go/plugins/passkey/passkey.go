package passkey

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated passkey plugin

// Plugin implements the passkey plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new passkey plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "passkey"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// BeginRegister BeginRegister initiates passkey registration with WebAuthn challenge
func (p *Plugin) BeginRegister(ctx context.Context) error {
	path := "/register/begin"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// FinishRegister FinishRegister completes passkey registration with attestation verification
func (p *Plugin) FinishRegister(ctx context.Context) error {
	path := "/register/finish"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// BeginLogin BeginLogin initiates passkey authentication with WebAuthn challenge
func (p *Plugin) BeginLogin(ctx context.Context) error {
	path := "/login/begin"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// FinishLogin FinishLogin completes passkey authentication with signature verification
func (p *Plugin) FinishLogin(ctx context.Context) error {
	path := "/login/finish"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// List List retrieves all passkeys for a user
func (p *Plugin) List(ctx context.Context) error {
	path := "/list"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// Update Update updates a passkey's metadata (name)
func (p *Plugin) Update(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// Delete Delete removes a passkey
func (p *Plugin) Delete(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

