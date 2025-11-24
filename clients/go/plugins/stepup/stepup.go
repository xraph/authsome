package stepup

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated stepup plugin

// Plugin implements the stepup plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new stepup plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "stepup"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// Evaluate Evaluate handles POST /stepup/evaluate
func (p *Plugin) Evaluate(ctx context.Context, req *authsome.EvaluateRequest) error {
	path := "/evaluate"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// Verify Verify handles POST /stepup/verify
func (p *Plugin) Verify(ctx context.Context, req *authsome.VerifyRequest) error {
	path := "/verify"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetRequirement GetRequirement handles GET /stepup/requirements/:id
func (p *Plugin) GetRequirement(ctx context.Context) error {
	path := "/requirements/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListPendingRequirements ListPendingRequirements handles GET /stepup/requirements/pending
func (p *Plugin) ListPendingRequirements(ctx context.Context) (*authsome.ListPendingRequirementsResponse, error) {
	path := "/requirements/pending"
	var result authsome.ListPendingRequirementsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListVerifications ListVerifications handles GET /stepup/verifications
func (p *Plugin) ListVerifications(ctx context.Context) error {
	path := "/verifications"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListRememberedDevices ListRememberedDevices handles GET /stepup/devices
func (p *Plugin) ListRememberedDevices(ctx context.Context) (*authsome.ListRememberedDevicesResponse, error) {
	path := "/devices"
	var result authsome.ListRememberedDevicesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ForgetDevice ForgetDevice handles DELETE /stepup/devices/:id
func (p *Plugin) ForgetDevice(ctx context.Context) (*authsome.ForgetDeviceResponse, error) {
	path := "/devices/:id"
	var result authsome.ForgetDeviceResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePolicy CreatePolicy handles POST /stepup/policies
func (p *Plugin) CreatePolicy(ctx context.Context, req *authsome.CreatePolicyRequest) (*authsome.CreatePolicyResponse, error) {
	path := "/policies"
	var result authsome.CreatePolicyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPolicies ListPolicies handles GET /stepup/policies
func (p *Plugin) ListPolicies(ctx context.Context) error {
	path := "/policies"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetPolicy GetPolicy handles GET /stepup/policies/:id
func (p *Plugin) GetPolicy(ctx context.Context) error {
	path := "/policies/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdatePolicy UpdatePolicy handles PUT /stepup/policies/:id
func (p *Plugin) UpdatePolicy(ctx context.Context, req *authsome.UpdatePolicyRequest) (*authsome.UpdatePolicyResponse, error) {
	path := "/policies/:id"
	var result authsome.UpdatePolicyResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePolicy DeletePolicy handles DELETE /stepup/policies/:id
func (p *Plugin) DeletePolicy(ctx context.Context) error {
	path := "/policies/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// GetAuditLogs GetAuditLogs handles GET /stepup/audit
func (p *Plugin) GetAuditLogs(ctx context.Context) error {
	path := "/audit"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// Status Status handles GET /stepup/status
func (p *Plugin) Status(ctx context.Context) error {
	path := "/status"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

