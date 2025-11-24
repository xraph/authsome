package admin

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated admin plugin

// Plugin implements the admin plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new admin plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "admin"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateUser CreateUser handles POST /admin/users
func (p *Plugin) CreateUser(ctx context.Context, req *authsome.CreateUserRequest) error {
	path := "/users"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListUsers ListUsers handles GET /admin/users
func (p *Plugin) ListUsers(ctx context.Context) error {
	path := "/users"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// DeleteUser DeleteUser handles DELETE /admin/users/:id
func (p *Plugin) DeleteUser(ctx context.Context) error {
	path := "/users/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// BanUser BanUser handles POST /admin/users/:id/ban
func (p *Plugin) BanUser(ctx context.Context, req *authsome.BanUserRequest) error {
	path := "/users/:id/ban"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// UnbanUser UnbanUser handles POST /admin/users/:id/unban
func (p *Plugin) UnbanUser(ctx context.Context, req *authsome.UnbanUserRequest) error {
	path := "/users/:id/unban"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ImpersonateUser ImpersonateUser handles POST /admin/users/:id/impersonate
func (p *Plugin) ImpersonateUser(ctx context.Context, req *authsome.ImpersonateUserRequest) error {
	path := "/users/:id/impersonate"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// SetUserRole SetUserRole handles POST /admin/users/:id/role
func (p *Plugin) SetUserRole(ctx context.Context, req *authsome.SetUserRoleRequest) error {
	path := "/users/:id/role"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// ListSessions ListSessions handles GET /admin/sessions
func (p *Plugin) ListSessions(ctx context.Context) error {
	path := "/sessions"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RevokeSession RevokeSession handles DELETE /admin/sessions/:id
func (p *Plugin) RevokeSession(ctx context.Context) error {
	path := "/sessions/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// GetStats GetStats handles GET /admin/stats
func (p *Plugin) GetStats(ctx context.Context) error {
	path := "/stats"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// GetAuditLogs GetAuditLogs handles GET /admin/audit
func (p *Plugin) GetAuditLogs(ctx context.Context) error {
	path := "/audit-logs"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

