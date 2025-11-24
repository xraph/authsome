package impersonation

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated impersonation plugin

// Plugin implements the impersonation plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new impersonation plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "impersonation"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// StartImpersonation StartImpersonation handles POST /impersonation/start
func (p *Plugin) StartImpersonation(ctx context.Context, req *authsome.StartImpersonationRequest) error {
	path := "/start"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// EndImpersonation EndImpersonation handles POST /impersonation/end
func (p *Plugin) EndImpersonation(ctx context.Context, req *authsome.EndImpersonationRequest) error {
	path := "/end"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// GetImpersonation GetImpersonation handles GET /impersonation/:id
func (p *Plugin) GetImpersonation(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListImpersonations ListImpersonations handles GET /impersonation
func (p *Plugin) ListImpersonations(ctx context.Context) error {
	path := "/"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListAuditEvents ListAuditEvents handles GET /impersonation/audit
func (p *Plugin) ListAuditEvents(ctx context.Context) error {
	path := "/audit"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// VerifyImpersonation VerifyImpersonation handles GET /impersonation/verify/:sessionId
func (p *Plugin) VerifyImpersonation(ctx context.Context) error {
	path := "/verify"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

