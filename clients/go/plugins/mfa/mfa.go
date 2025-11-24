package mfa

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated mfa plugin

// Plugin implements the mfa plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new mfa plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "mfa"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// EnrollFactor EnrollFactor handles POST /mfa/factors/enroll
func (p *Plugin) EnrollFactor(ctx context.Context, req *authsome.EnrollFactorRequest) error {
	path := "/mfa/factors/enroll"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListFactors ListFactors handles GET /mfa/factors
func (p *Plugin) ListFactors(ctx context.Context) (*authsome.ListFactorsResponse, error) {
	path := "/mfa/factors"
	var result authsome.ListFactorsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetFactor GetFactor handles GET /mfa/factors/:id
func (p *Plugin) GetFactor(ctx context.Context) error {
	path := "/mfa/factors/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateFactor UpdateFactor handles PUT /mfa/factors/:id
func (p *Plugin) UpdateFactor(ctx context.Context) error {
	path := "/mfa/factors/:id"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteFactor DeleteFactor handles DELETE /mfa/factors/:id
func (p *Plugin) DeleteFactor(ctx context.Context) error {
	path := "/mfa/factors/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// VerifyFactor VerifyFactor handles POST /mfa/factors/:id/verify
func (p *Plugin) VerifyFactor(ctx context.Context, req *authsome.VerifyFactorRequest) error {
	path := "/mfa/factors/:id/verify"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// InitiateChallenge InitiateChallenge handles POST /mfa/challenge
func (p *Plugin) InitiateChallenge(ctx context.Context, req *authsome.InitiateChallengeRequest) error {
	path := "/mfa/challenge"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// VerifyChallenge VerifyChallenge handles POST /mfa/verify
func (p *Plugin) VerifyChallenge(ctx context.Context, req *authsome.VerifyChallengeRequest) error {
	path := "/mfa/verify"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetChallengeStatus GetChallengeStatus handles GET /mfa/challenge/:id
func (p *Plugin) GetChallengeStatus(ctx context.Context) error {
	path := "/mfa/challenge/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// TrustDevice TrustDevice handles POST /mfa/devices/trust
func (p *Plugin) TrustDevice(ctx context.Context, req *authsome.TrustDeviceRequest) error {
	path := "/mfa/devices/trust"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListTrustedDevices ListTrustedDevices handles GET /mfa/devices
func (p *Plugin) ListTrustedDevices(ctx context.Context) (*authsome.ListTrustedDevicesResponse, error) {
	path := "/mfa/devices"
	var result authsome.ListTrustedDevicesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeTrustedDevice RevokeTrustedDevice handles DELETE /mfa/devices/:id
func (p *Plugin) RevokeTrustedDevice(ctx context.Context) error {
	path := "/mfa/devices/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// GetStatus GetStatus handles GET /mfa/status
func (p *Plugin) GetStatus(ctx context.Context) error {
	path := "/mfa/status"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetPolicy GetPolicy handles GET /mfa/policy
func (p *Plugin) GetPolicy(ctx context.Context) (*authsome.GetPolicyResponse, error) {
	path := "/mfa/policy"
	var result authsome.GetPolicyResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// AdminUpdatePolicy AdminUpdatePolicy handles PUT /mfa/admin/policy
Updates the MFA policy for an app (admin only)
func (p *Plugin) AdminUpdatePolicy(ctx context.Context, req *authsome.AdminUpdatePolicyRequest) error {
	path := "/mfa/policy"
	err := p.client.Request(ctx, "PUT", path, req, nil, false)
	return err
}

// AdminResetUserMFA AdminResetUserMFA handles POST /mfa/admin/users/:id/reset
Resets all MFA factors for a user (admin only)
func (p *Plugin) AdminResetUserMFA(ctx context.Context) error {
	path := "/mfa/users/:id/reset"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

