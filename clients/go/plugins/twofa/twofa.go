package twofa

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated twofa plugin

// Plugin implements the twofa plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new twofa plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "twofa"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

func (p *Plugin) Enable(ctx context.Context) error {
	path := "/2fa/enable"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

func (p *Plugin) Verify(ctx context.Context) error {
	path := "/2fa/verify"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

func (p *Plugin) Disable(ctx context.Context) error {
	path := "/2fa/disable"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

func (p *Plugin) GenerateBackupCodes(ctx context.Context) (*authsome.GenerateBackupCodesResponse, error) {
	path := "/2fa/generate-backup-codes"
	var result authsome.GenerateBackupCodesResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SendOTP SendOTP triggers generation of an OTP code for a user (returns code in response for dev/testing)
func (p *Plugin) SendOTP(ctx context.Context) (*authsome.SendOTPResponse, error) {
	path := "/2fa/send-otp"
	var result authsome.SendOTPResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Status Status returns whether 2FA is enabled and whether the device is trusted
func (p *Plugin) Status(ctx context.Context) (*authsome.StatusResponse, error) {
	path := "/2fa/status"
	var result authsome.StatusResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

