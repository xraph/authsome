package organization

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated organization plugin

// Plugin implements the organization plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new organization plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "organization"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateOrganization CreateOrganization handles organization creation
func (p *Plugin) CreateOrganization(ctx context.Context) error {
	path := "/createorganization"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// UpdateOrganization UpdateOrganization handles organization updates
func (p *Plugin) UpdateOrganization(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "PATCH", path, nil, nil, false)
	return err
}

// DeleteOrganization DeleteOrganization handles organization deletion
func (p *Plugin) DeleteOrganization(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// InviteMember InviteMember handles member invitation
func (p *Plugin) InviteMember(ctx context.Context) error {
	path := "/invite"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// RemoveMember RemoveMember handles member removal
func (p *Plugin) RemoveMember(ctx context.Context) error {
	path := "/:memberId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// CreateTeam CreateTeam handles team creation
func (p *Plugin) CreateTeam(ctx context.Context) error {
	path := "/createteam"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// UpdateTeam UpdateTeam handles team updates
func (p *Plugin) UpdateTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "PATCH", path, nil, nil, false)
	return err
}

// DeleteTeam DeleteTeam handles team deletion
func (p *Plugin) DeleteTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// CreateOrganization CreateOrganization handles organization creation requests
func (p *Plugin) CreateOrganization(ctx context.Context) error {
	path := "/createorganization"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetOrganization GetOrganization handles get organization requests
func (p *Plugin) GetOrganization(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListOrganizations ListOrganizations handles list organizations requests (user's organizations)
func (p *Plugin) ListOrganizations(ctx context.Context) error {
	path := "/listorganizations"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateOrganization UpdateOrganization handles organization update requests
func (p *Plugin) UpdateOrganization(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "PATCH", path, nil, nil, false)
	return err
}

// DeleteOrganization DeleteOrganization handles organization deletion requests
func (p *Plugin) DeleteOrganization(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// GetOrganizationBySlug GetOrganizationBySlug handles get organization by slug requests
func (p *Plugin) GetOrganizationBySlug(ctx context.Context) error {
	path := "/slug/:slug"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListMembers ListMembers handles list organization members requests
func (p *Plugin) ListMembers(ctx context.Context) error {
	path := "/listmembers"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// InviteMember InviteMember handles member invitation requests
func (p *Plugin) InviteMember(ctx context.Context) error {
	path := "/invite"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// UpdateMember UpdateMember handles member update requests
func (p *Plugin) UpdateMember(ctx context.Context) error {
	path := "/:memberId"
	err := p.client.Request(ctx, "PATCH", path, nil, nil, false)
	return err
}

// RemoveMember RemoveMember handles member removal requests
func (p *Plugin) RemoveMember(ctx context.Context) error {
	path := "/:memberId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// AcceptInvitation AcceptInvitation handles invitation acceptance requests
func (p *Plugin) AcceptInvitation(ctx context.Context) error {
	path := "/:token/accept"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// DeclineInvitation DeclineInvitation handles invitation decline requests
func (p *Plugin) DeclineInvitation(ctx context.Context) error {
	path := "/:token/decline"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// ListTeams ListTeams handles list teams requests
func (p *Plugin) ListTeams(ctx context.Context) error {
	path := "/listteams"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// CreateTeam CreateTeam handles team creation requests
func (p *Plugin) CreateTeam(ctx context.Context) error {
	path := "/createteam"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// UpdateTeam UpdateTeam handles team update requests
func (p *Plugin) UpdateTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "PATCH", path, nil, nil, false)
	return err
}

// DeleteTeam DeleteTeam handles team deletion requests
func (p *Plugin) DeleteTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

