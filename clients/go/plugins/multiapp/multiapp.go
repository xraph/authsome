package multiapp

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated multiapp plugin

// Plugin implements the multiapp plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new multiapp plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "multiapp"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// CreateApp CreateApp handles app creation requests
func (p *Plugin) CreateApp(ctx context.Context) error {
	path := "/createapp"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetApp GetApp handles get app requests
func (p *Plugin) GetApp(ctx context.Context) error {
	path := "/:appId"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateApp UpdateApp handles app update requests
func (p *Plugin) UpdateApp(ctx context.Context) error {
	path := "/:appId"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteApp DeleteApp handles app deletion requests
func (p *Plugin) DeleteApp(ctx context.Context) error {
	path := "/:appId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ListApps ListApps handles list apps requests
func (p *Plugin) ListApps(ctx context.Context) error {
	path := "/listapps"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// RemoveMember RemoveMember handles removing a member from an organization
func (p *Plugin) RemoveMember(ctx context.Context) error {
	path := "/:memberId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ListMembers ListMembers handles listing app members
func (p *Plugin) ListMembers(ctx context.Context) error {
	path := "/listmembers"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// InviteMember InviteMember handles inviting a member to an organization
func (p *Plugin) InviteMember(ctx context.Context) error {
	path := "/invite"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// UpdateMember UpdateMember handles updating a member in an organization
func (p *Plugin) UpdateMember(ctx context.Context) error {
	path := "/:memberId"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// GetInvitation GetInvitation handles getting an invitation by token
func (p *Plugin) GetInvitation(ctx context.Context) error {
	path := "/:token"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// AcceptInvitation AcceptInvitation handles accepting an invitation
func (p *Plugin) AcceptInvitation(ctx context.Context) error {
	path := "/:token/accept"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// DeclineInvitation DeclineInvitation handles declining an invitation
func (p *Plugin) DeclineInvitation(ctx context.Context) error {
	path := "/:token/decline"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// CreateTeam CreateTeam handles team creation requests
func (p *Plugin) CreateTeam(ctx context.Context) error {
	path := "/createteam"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetTeam GetTeam handles team retrieval requests
func (p *Plugin) GetTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateTeam UpdateTeam handles team update requests
func (p *Plugin) UpdateTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteTeam DeleteTeam handles team deletion requests
func (p *Plugin) DeleteTeam(ctx context.Context) error {
	path := "/:teamId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ListTeams ListTeams handles team listing requests
func (p *Plugin) ListTeams(ctx context.Context) error {
	path := "/listteams"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// AddTeamMember AddTeamMember handles adding a member to a team
func (p *Plugin) AddTeamMember(ctx context.Context, req *authsome.AddTeamMemberRequest) error {
	path := "/:teamId/members"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// RemoveTeamMember RemoveTeamMember handles removing a member from a team
func (p *Plugin) RemoveTeamMember(ctx context.Context) error {
	path := "/:teamId/members/:memberId"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

