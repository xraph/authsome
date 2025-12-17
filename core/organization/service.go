package organization

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
)

// Service provides access to all organization-related services
// Internally delegates to focused services for better separation of concerns
type Service struct {
	Organization *OrganizationService
	Member       *MemberService
	Team         *TeamService
	Invitation   *InvitationService
	hookRegistry interface{} // Hook registry for lifecycle events (interface{} to avoid import cycle)
}

// NewService creates a new service with all focused services
func NewService(
	orgRepo OrganizationRepository,
	memberRepo MemberRepository,
	teamRepo TeamRepository,
	invitationRepo InvitationRepository,
	cfg Config,
	rbacSvc *rbac.Service,
	roleRepo rbac.RoleRepository,
) *Service {
	return &Service{
		Organization: NewOrganizationService(orgRepo, cfg, rbacSvc),
		Member:       NewMemberService(memberRepo, orgRepo, cfg, rbacSvc, roleRepo),
		Team:         NewTeamService(teamRepo, memberRepo, cfg, rbacSvc),
		Invitation:   NewInvitationService(invitationRepo, memberRepo, orgRepo, cfg, rbacSvc, roleRepo),
	}
}

// SetHookRegistry sets the hook registry for executing lifecycle hooks
func (s *Service) SetHookRegistry(registry interface{}) {
	s.hookRegistry = registry
}

// =============================================================================
// Organization Operations Delegation
// =============================================================================

func (s *Service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest, creatorUserID, appID, environmentID xid.ID) (*Organization, error) {
	return s.Organization.CreateOrganization(ctx, req, creatorUserID, appID, environmentID)
}

func (s *Service) FindOrganizationByID(ctx context.Context, id xid.ID) (*Organization, error) {
	return s.Organization.FindOrganizationByID(ctx, id)
}

func (s *Service) FindOrganizationBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*Organization, error) {
	return s.Organization.FindOrganizationBySlug(ctx, appID, environmentID, slug)
}

func (s *Service) ListOrganizations(ctx context.Context, filter *ListOrganizationsFilter) (*pagination.PageResponse[*Organization], error) {
	return s.Organization.ListOrganizations(ctx, filter)
}

func (s *Service) ListUserOrganizations(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Organization], error) {
	return s.Organization.ListUserOrganizations(ctx, userID, filter)
}

func (s *Service) UpdateOrganization(ctx context.Context, id xid.ID, req *UpdateOrganizationRequest) (*Organization, error) {
	return s.Organization.UpdateOrganization(ctx, id, req)
}

func (s *Service) DeleteOrganization(ctx context.Context, id, userID xid.ID) error {
	// Check authorization before deletion
	if err := s.Member.RequireOwner(ctx, id, userID); err != nil {
		return err
	}

	// Get organization details before deletion
	org, err := s.Organization.FindOrganizationByID(ctx, id)
	if err != nil {
		return err
	}
	orgName := org.Name

	// Delete organization
	err = s.Organization.DeleteOrganization(ctx, id, userID)
	if err != nil {
		return err
	}

	// Execute after organization delete hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteAfterOrganizationDelete(context.Context, xid.ID, string) error
		}); ok {
			_ = registry.ExecuteAfterOrganizationDelete(ctx, id, orgName)
		}
	}

	return nil
}

// ForceDeleteOrganization deletes an organization without permission checks
// This should only be called by admin users or in administrative contexts
func (s *Service) ForceDeleteOrganization(ctx context.Context, id xid.ID) error {
	return s.Organization.ForceDeleteOrganization(ctx, id)
}

// =============================================================================
// Member Operations Delegation
// =============================================================================

func (s *Service) AddMember(ctx context.Context, orgID, userID xid.ID, role string) (*Member, error) {
	member, err := s.Member.AddMember(ctx, orgID, userID, role)
	if err != nil {
		return nil, err
	}

	// Execute after member add hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteAfterMemberAdd(context.Context, interface{}) error
		}); ok {
			_ = registry.ExecuteAfterMemberAdd(ctx, member)
		}
	}

	return member, nil
}

func (s *Service) FindMemberByID(ctx context.Context, id xid.ID) (*Member, error) {
	return s.Member.FindMemberByID(ctx, id)
}

func (s *Service) FindMember(ctx context.Context, orgID, userID xid.ID) (*Member, error) {
	return s.Member.FindMember(ctx, orgID, userID)
}

func (s *Service) ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error) {
	return s.Member.ListMembers(ctx, filter)
}

func (s *Service) UpdateMember(ctx context.Context, id xid.ID, req *UpdateMemberRequest, updaterUserID xid.ID) (*Member, error) {
	return s.Member.UpdateMember(ctx, id, req, updaterUserID)
}

func (s *Service) UpdateMemberRole(ctx context.Context, orgID, memberID xid.ID, newRole string, updaterUserID xid.ID) (*Member, error) {
	// Get member details before updating to get old role
	member, err := s.Member.FindMemberByID(ctx, memberID)
	if err != nil {
		return nil, err
	}
	oldRole := member.Role

	// Update role
	member, err = s.Member.UpdateMemberRole(ctx, orgID, memberID, newRole, updaterUserID)
	if err != nil {
		return nil, err
	}

	// Execute after member role change hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteAfterMemberRoleChange(context.Context, xid.ID, xid.ID, string, string) error
		}); ok {
			_ = registry.ExecuteAfterMemberRoleChange(ctx, orgID, member.UserID, oldRole, newRole)
		}
	}

	return member, nil
}

func (s *Service) RemoveMember(ctx context.Context, id, removerUserID xid.ID) error {
	// Get member details before removing
	member, err := s.Member.FindMemberByID(ctx, id)
	if err != nil {
		return err
	}

	orgID := member.OrganizationID
	userID := member.UserID
	memberName := ""
	if member.User != nil {
		memberName = member.User.Name
		if memberName == "" {
			memberName = member.User.Email
		}
	}

	// Remove member
	err = s.Member.RemoveMember(ctx, id, removerUserID)
	if err != nil {
		return err
	}

	// Execute after member remove hook
	if s.hookRegistry != nil {
		if registry, ok := s.hookRegistry.(interface {
			ExecuteAfterMemberRemove(context.Context, xid.ID, xid.ID, string) error
		}); ok {
			_ = registry.ExecuteAfterMemberRemove(ctx, orgID, userID, memberName)
		}
	}

	return nil
}

func (s *Service) GetUserMemberships(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Member], error) {
	return s.Member.GetUserMemberships(ctx, userID, filter)
}

func (s *Service) RemoveUserFromAllOrganizations(ctx context.Context, userID xid.ID) error {
	return s.Member.RemoveUserFromAllOrganizations(ctx, userID)
}

func (s *Service) IsMember(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	return s.Member.IsMember(ctx, orgID, userID)
}

func (s *Service) IsOwner(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	return s.Member.IsOwner(ctx, orgID, userID)
}

func (s *Service) IsAdmin(ctx context.Context, orgID, userID xid.ID) (bool, error) {
	return s.Member.IsAdmin(ctx, orgID, userID)
}

func (s *Service) RequireOwner(ctx context.Context, orgID, userID xid.ID) error {
	return s.Member.RequireOwner(ctx, orgID, userID)
}

func (s *Service) RequireAdmin(ctx context.Context, orgID, userID xid.ID) error {
	return s.Member.RequireAdmin(ctx, orgID, userID)
}

// RBAC Permission methods delegation
func (s *Service) CheckPermission(ctx context.Context, orgID, userID xid.ID, action, resource string) (bool, error) {
	return s.Member.CheckPermission(ctx, orgID, userID, action, resource)
}

func (s *Service) CheckPermissionWithContext(ctx context.Context, orgID, userID xid.ID, action, resource string, contextVars map[string]string) (bool, error) {
	return s.Member.CheckPermissionWithContext(ctx, orgID, userID, action, resource, contextVars)
}

func (s *Service) RequirePermission(ctx context.Context, orgID, userID xid.ID, action, resource string) error {
	return s.Member.RequirePermission(ctx, orgID, userID, action, resource)
}

// =============================================================================
// Team Operations Delegation
// =============================================================================

func (s *Service) CreateTeam(ctx context.Context, orgID xid.ID, req *CreateTeamRequest, creatorUserID xid.ID) (*Team, error) {
	return s.Team.CreateTeam(ctx, orgID, req, creatorUserID)
}

func (s *Service) FindTeamByID(ctx context.Context, id xid.ID) (*Team, error) {
	return s.Team.FindTeamByID(ctx, id)
}

func (s *Service) FindTeamByName(ctx context.Context, orgID xid.ID, name string) (*Team, error) {
	return s.Team.FindTeamByName(ctx, orgID, name)
}

func (s *Service) ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error) {
	return s.Team.ListTeams(ctx, filter)
}

func (s *Service) UpdateTeam(ctx context.Context, id xid.ID, req *UpdateTeamRequest, updaterUserID xid.ID) (*Team, error) {
	return s.Team.UpdateTeam(ctx, id, req, updaterUserID)
}

func (s *Service) DeleteTeam(ctx context.Context, id, deleterUserID xid.ID) error {
	return s.Team.DeleteTeam(ctx, id, deleterUserID)
}

func (s *Service) AddTeamMember(ctx context.Context, teamID, memberID, adderUserID xid.ID) error {
	return s.Team.AddTeamMember(ctx, teamID, memberID, adderUserID)
}

func (s *Service) RemoveTeamMember(ctx context.Context, teamID, memberID, removerUserID xid.ID) error {
	return s.Team.RemoveTeamMember(ctx, teamID, memberID, removerUserID)
}

func (s *Service) ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error) {
	return s.Team.ListTeamMembers(ctx, filter)
}

func (s *Service) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	return s.Team.IsTeamMember(ctx, teamID, memberID)
}

func (s *Service) FindTeamMemberByID(ctx context.Context, id xid.ID) (*TeamMember, error) {
	return s.Team.FindTeamMemberByID(ctx, id)
}

func (s *Service) FindTeamMember(ctx context.Context, teamID, memberID xid.ID) (*TeamMember, error) {
	return s.Team.FindTeamMember(ctx, teamID, memberID)
}

func (s *Service) ListMemberTeams(ctx context.Context, memberID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Team], error) {
	return s.Team.ListMemberTeams(ctx, memberID, filter)
}

// =============================================================================
// Invitation Operations Delegation
// =============================================================================

func (s *Service) InviteMember(ctx context.Context, orgID xid.ID, req *InviteMemberRequest, inviterUserID xid.ID) (*Invitation, error) {
	return s.Invitation.InviteMember(ctx, orgID, req, inviterUserID)
}

func (s *Service) FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error) {
	return s.Invitation.FindInvitationByID(ctx, id)
}

func (s *Service) FindInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	return s.Invitation.FindInvitationByToken(ctx, token)
}

func (s *Service) ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error) {
	return s.Invitation.ListInvitations(ctx, filter)
}

func (s *Service) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error) {
	return s.Invitation.AcceptInvitation(ctx, token, userID)
}

func (s *Service) DeclineInvitation(ctx context.Context, token string) error {
	return s.Invitation.DeclineInvitation(ctx, token)
}

func (s *Service) CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error {
	return s.Invitation.CancelInvitation(ctx, id, cancellerUserID)
}

func (s *Service) ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error) {
	return s.Invitation.ResendInvitation(ctx, id, resenderUserID)
}

func (s *Service) CleanupExpiredInvitations(ctx context.Context) (int, error) {
	return s.Invitation.CleanupExpiredInvitations(ctx)
}

// Type assertion to ensure Service implements CompositeOrganizationService
var _ CompositeOrganizationService = (*Service)(nil)
