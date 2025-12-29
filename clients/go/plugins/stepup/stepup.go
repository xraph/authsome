package stepup

import (
	"context"
	"net/url"

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
func (p *Plugin) Evaluate(ctx context.Context, req *authsome.EvaluateRequest) (*authsome.EvaluateResponse, error) {
	path := "/stepup/evaluate"
	var result authsome.EvaluateResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Verify Verify handles POST /stepup/verify
func (p *Plugin) Verify(ctx context.Context, req *authsome.VerifyRequest) (*authsome.VerifyResponse, error) {
	path := "/stepup/verify"
	var result authsome.VerifyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRequirement GetRequirement handles GET /stepup/requirements/:id
func (p *Plugin) GetRequirement(ctx context.Context, id xid.ID) (*authsome.GetRequirementResponse, error) {
	path := "/stepup/requirements/:id"
	var result authsome.GetRequirementResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPendingRequirements ListPendingRequirements handles GET /stepup/requirements/pending
func (p *Plugin) ListPendingRequirements(ctx context.Context) (*authsome.ListPendingRequirementsResponse, error) {
	path := "/stepup/requirements/pending"
	var result authsome.ListPendingRequirementsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListVerifications ListVerifications handles GET /stepup/verifications
func (p *Plugin) ListVerifications(ctx context.Context) (*authsome.ListVerificationsResponse, error) {
	path := "/stepup/verifications"
	var result authsome.ListVerificationsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRememberedDevices ListRememberedDevices handles GET /stepup/devices
func (p *Plugin) ListRememberedDevices(ctx context.Context) (*authsome.ListRememberedDevicesResponse, error) {
	path := "/stepup/devices"
	var result authsome.ListRememberedDevicesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ForgetDevice ForgetDevice handles DELETE /stepup/devices/:id
func (p *Plugin) ForgetDevice(ctx context.Context, id xid.ID) (*authsome.ForgetDeviceResponse, error) {
	path := "/stepup/devices/:id"
	var result authsome.ForgetDeviceResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreatePolicy CreatePolicy handles POST /stepup/policies
func (p *Plugin) CreatePolicy(ctx context.Context, req *authsome.CreatePolicyRequest) (*authsome.CreatePolicyResponse, error) {
	path := "/stepup/policies"
	var result authsome.CreatePolicyResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListPolicies ListPolicies handles GET /stepup/policies
func (p *Plugin) ListPolicies(ctx context.Context) (*authsome.ListPoliciesResponse, error) {
	path := "/stepup/policies"
	var result authsome.ListPoliciesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPolicy GetPolicy handles GET /stepup/policies/:id
func (p *Plugin) GetPolicy(ctx context.Context, id xid.ID) (*authsome.GetPolicyResponse, error) {
	path := "/stepup/policies/:id"
	var result authsome.GetPolicyResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdatePolicy UpdatePolicy handles PUT /stepup/policies/:id
func (p *Plugin) UpdatePolicy(ctx context.Context, req *authsome.UpdatePolicyRequest, id xid.ID) (*authsome.UpdatePolicyResponse, error) {
	path := "/stepup/policies/:id"
	var result authsome.UpdatePolicyResponse
	err := p.client.Request(ctx, "PUT", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeletePolicy DeletePolicy handles DELETE /stepup/policies/:id
func (p *Plugin) DeletePolicy(ctx context.Context, id xid.ID) (*authsome.DeletePolicyResponse, error) {
	path := "/stepup/policies/:id"
	var result authsome.DeletePolicyResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAuditLogs GetAuditLogs handles GET /stepup/audit
func (p *Plugin) GetAuditLogs(ctx context.Context) (*authsome.GetAuditLogsResponse, error) {
	path := "/stepup/audit"
	var result authsome.GetAuditLogsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Status Status handles GET /stepup/status
func (p *Plugin) Status(ctx context.Context) (*authsome.StatusResponse, error) {
	path := "/stepup/status"
	var result authsome.StatusResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

