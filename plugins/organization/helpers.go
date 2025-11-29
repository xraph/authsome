package organization

import (
	"fmt"

	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/forge"
)

// Service name constants for DI container registration
const (
	ServiceNamePlugin            = "organization.plugin"
	ServiceNameService           = "organization.service"
	ServiceNameMemberService     = "organization.member_service"
	ServiceNameTeamService       = "organization.team_service"
	ServiceNameInvitationService = "organization.invitation_service"
)

// ResolveOrganizationPlugin resolves the organization plugin from the container
func ResolveOrganizationPlugin(container forge.Container) (*Plugin, error) {
	resolved, err := container.Resolve(ServiceNamePlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization plugin: %w", err)
	}
	plugin, ok := resolved.(*Plugin)
	if !ok {
		return nil, fmt.Errorf("invalid organization plugin type")
	}
	return plugin, nil
}

// ResolveOrganizationService resolves the organization service from the container
func ResolveOrganizationService(container forge.Container) (*organization.Service, error) {
	resolved, err := container.Resolve(ServiceNameService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization service: %w", err)
	}
	svc, ok := resolved.(*organization.Service)
	if !ok {
		return nil, fmt.Errorf("invalid organization service type")
	}
	return svc, nil
}

// ResolveMemberService resolves the member service from the container
func ResolveMemberService(container forge.Container) (*organization.MemberService, error) {
	resolved, err := container.Resolve(ServiceNameMemberService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve member service: %w", err)
	}
	svc, ok := resolved.(*organization.MemberService)
	if !ok {
		return nil, fmt.Errorf("invalid member service type")
	}
	return svc, nil
}

// ResolveTeamService resolves the team service from the container
func ResolveTeamService(container forge.Container) (*organization.TeamService, error) {
	resolved, err := container.Resolve(ServiceNameTeamService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve team service: %w", err)
	}
	svc, ok := resolved.(*organization.TeamService)
	if !ok {
		return nil, fmt.Errorf("invalid team service type")
	}
	return svc, nil
}

// ResolveInvitationService resolves the invitation service from the container
func ResolveInvitationService(container forge.Container) (*organization.InvitationService, error) {
	resolved, err := container.Resolve(ServiceNameInvitationService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve invitation service: %w", err)
	}
	svc, ok := resolved.(*organization.InvitationService)
	if !ok {
		return nil, fmt.Errorf("invalid invitation service type")
	}
	return svc, nil
}

// RegisterServices registers all organization services in the DI container
func (p *Plugin) RegisterServices(container forge.Container) error {
	// Register plugin itself
	if err := container.Register(ServiceNamePlugin, func(_ forge.Container) (any, error) {
		return p, nil
	}); err != nil {
		return fmt.Errorf("failed to register organization plugin: %w", err)
	}

	// Register organization service (composite service)
	if err := container.Register(ServiceNameService, func(_ forge.Container) (any, error) {
		return p.orgService, nil
	}); err != nil {
		return fmt.Errorf("failed to register organization service: %w", err)
	}

	// Register member service
	if err := container.Register(ServiceNameMemberService, func(_ forge.Container) (any, error) {
		if p.orgService != nil {
			return p.orgService.Member, nil
		}
		return nil, fmt.Errorf("organization service not initialized")
	}); err != nil {
		return fmt.Errorf("failed to register member service: %w", err)
	}

	// Register team service
	if err := container.Register(ServiceNameTeamService, func(_ forge.Container) (any, error) {
		if p.orgService != nil {
			return p.orgService.Team, nil
		}
		return nil, fmt.Errorf("organization service not initialized")
	}); err != nil {
		return fmt.Errorf("failed to register team service: %w", err)
	}

	// Register invitation service
	if err := container.Register(ServiceNameInvitationService, func(_ forge.Container) (any, error) {
		if p.orgService != nil {
			return p.orgService.Invitation, nil
		}
		return nil, fmt.Errorf("organization service not initialized")
	}); err != nil {
		return fmt.Errorf("failed to register invitation service: %w", err)
	}

	return nil
}

// GetServices returns a map of all available services for inspection
func (p *Plugin) GetServices() map[string]interface{} {
	services := map[string]interface{}{
		"organizationService": p.orgService,
	}

	if p.orgService != nil {
		services["memberService"] = p.orgService.Member
		services["teamService"] = p.orgService.Team
		services["invitationService"] = p.orgService.Invitation
	}

	return services
}

// GetOrganizationService returns the organization service directly
func (p *Plugin) GetOrganizationService() *organization.Service {
	return p.orgService
}

// GetMemberService returns the member service directly
func (p *Plugin) GetMemberService() *organization.MemberService {
	if p.orgService != nil {
		return p.orgService.Member
	}
	return nil
}

// GetTeamService returns the team service directly
func (p *Plugin) GetTeamService() *organization.TeamService {
	if p.orgService != nil {
		return p.orgService.Team
	}
	return nil
}

// GetInvitationService returns the invitation service directly
func (p *Plugin) GetInvitationService() *organization.InvitationService {
	if p.orgService != nil {
		return p.orgService.Invitation
	}
	return nil
}

