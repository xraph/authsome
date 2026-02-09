package organization

import (
	"fmt"

	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
	"github.com/xraph/vessel"
)

// Service name constants for DI container registration.
const (
	ServiceNamePlugin            = "organization.plugin"
	ServiceNameService           = "organization.service"
	ServiceNameMemberService     = "organization.member_service"
	ServiceNameTeamService       = "organization.team_service"
	ServiceNameInvitationService = "organization.invitation_service"
)

// ResolveOrganizationPlugin resolves the organization plugin from the container.
func ResolveOrganizationPlugin(container forge.Container) (*Plugin, error) {
	plugin, err := vessel.InjectType[*Plugin](container)
	if plugin != nil {
		return plugin, nil
	}

	resolved, err := container.Resolve(ServiceNamePlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization plugin: %w", err)
	}

	plugin, ok := resolved.(*Plugin)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("invalid organization plugin type")
	}

	return plugin, nil
}

// ResolveOrganizationService resolves the organization service from the container.
func ResolveOrganizationService(container forge.Container) (*organization.Service, error) {
	svc, err := vessel.InjectType[*organization.Service](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve organization service: %w", err)
	}

	svc, ok := resolved.(*organization.Service)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("invalid organization service type")
	}

	return svc, nil
}

// ResolveMemberService resolves the member service from the container.
func ResolveMemberService(container forge.Container) (*organization.MemberService, error) {
	svc, err := vessel.InjectType[*organization.MemberService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameMemberService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve member service: %w", err)
	}

	svc, ok := resolved.(*organization.MemberService)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("invalid member service type")
	}

	return svc, nil
}

// ResolveTeamService resolves the team service from the container.
func ResolveTeamService(container forge.Container) (*organization.TeamService, error) {
	svc, err := vessel.InjectType[*organization.TeamService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameTeamService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve team service: %w", err)
	}

	svc, ok := resolved.(*organization.TeamService)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("invalid team service type")
	}

	return svc, nil
}

// ResolveInvitationService resolves the invitation service from the container.
func ResolveInvitationService(container forge.Container) (*organization.InvitationService, error) {
	svc, err := vessel.InjectType[*organization.InvitationService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameInvitationService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve invitation service: %w", err)
	}

	svc, ok := resolved.(*organization.InvitationService)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("invalid invitation service type")
	}

	return svc, nil
}

// RegisterServices registers all organization services in the DI container
// Uses vessel.ProvideConstructor for type-safe, constructor-based dependency injection
// Note: If services are already registered (e.g., from a previous plugin initialization),
// this will silently skip re-registration to allow for graceful handling of plugin reloads.
func (p *Plugin) RegisterServices(container forge.Container) error {
	// Register plugin itself
	if err := forge.ProvideConstructor(container, func() (*Plugin, error) {
		return p, nil
	}, vessel.WithAliases(ServiceNamePlugin)); err != nil {
		// Service already registered - this is OK, silently continue
		return nil
	}

	// Register organization service (composite service)
	if err := forge.ProvideConstructor(container, func() (*organization.Service, error) {
		return p.orgService, nil
	}, vessel.WithAliases(ServiceNameService)); err != nil {
		// Service already registered - this is OK, silently continue
		return nil
	}

	// Register member service
	if err := forge.ProvideConstructor(container, func() (*organization.MemberService, error) {
		if p.orgService != nil {
			return p.orgService.Member, nil
		}

		return nil, errs.InternalServerErrorWithMessage("organization service not initialized")
	}, vessel.WithAliases(ServiceNameMemberService)); err != nil {
		// Service already registered - this is OK, silently continue
		return nil
	}

	// Register team service
	if err := forge.ProvideConstructor(container, func() (*organization.TeamService, error) {
		if p.orgService != nil {
			return p.orgService.Team, nil
		}

		return nil, errs.InternalServerErrorWithMessage("organization service not initialized")
	}, vessel.WithAliases(ServiceNameTeamService)); err != nil {
		// Service already registered - this is OK, silently continue
		return nil
	}

	// Register invitation service
	if err := forge.ProvideConstructor(container, func() (*organization.InvitationService, error) {
		if p.orgService != nil {
			return p.orgService.Invitation, nil
		}

		return nil, errs.InternalServerErrorWithMessage("organization service not initialized")
	}, vessel.WithAliases(ServiceNameInvitationService)); err != nil {
		// Service already registered - this is OK, silently continue
		return nil
	}

	return nil
}

// GetServices returns a map of all available services for inspection.
func (p *Plugin) GetServices() map[string]any {
	services := map[string]any{
		"organizationService": p.orgService,
	}

	if p.orgService != nil {
		services["memberService"] = p.orgService.Member
		services["teamService"] = p.orgService.Team
		services["invitationService"] = p.orgService.Invitation
	}

	return services
}

// GetOrganizationService returns the organization service directly.
func (p *Plugin) GetOrganizationService() *organization.Service {
	return p.orgService
}

// GetMemberService returns the member service directly.
func (p *Plugin) GetMemberService() *organization.MemberService {
	if p.orgService != nil {
		return p.orgService.Member
	}

	return nil
}

// GetTeamService returns the team service directly.
func (p *Plugin) GetTeamService() *organization.TeamService {
	if p.orgService != nil {
		return p.orgService.Team
	}

	return nil
}

// GetInvitationService returns the invitation service directly.
func (p *Plugin) GetInvitationService() *organization.InvitationService {
	if p.orgService != nil {
		return p.orgService.Invitation
	}

	return nil
}
