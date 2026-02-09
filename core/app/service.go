package app

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
)

// ServiceImpl provides access to all app-related services
// Internally delegates to focused services for better separation of concerns.
type ServiceImpl struct {
	App        *AppService
	Member     *MemberService
	Team       *TeamService
	Invitation *InvitationService
}

// NewService creates a new service with all focused services.
func NewService(
	appRepo AppRepository,
	memberRepo MemberRepository,
	teamRepo TeamRepository,
	invitationRepo InvitationRepository,
	roleRepo rbac.RoleRepository, // From core/rbac package
	userRoleRepo rbac.UserRoleRepository, // From core/rbac package
	cfg Config,
	rbacSvc *rbac.Service,
) *ServiceImpl {
	// Create member service first (needed by invitation service)
	memberService := NewMemberService(memberRepo, appRepo, roleRepo, userRoleRepo, cfg, rbacSvc)

	return &ServiceImpl{
		App:        NewAppService(appRepo, cfg, rbacSvc),
		Member:     memberService,
		Team:       NewTeamService(teamRepo, memberRepo, cfg, rbacSvc),
		Invitation: NewInvitationService(invitationRepo, memberRepo, memberService, appRepo, cfg, rbacSvc),
	}
}

// =============================================================================
// App Operations Delegation
// =============================================================================

func (s *ServiceImpl) UpdateConfig(cfg Config) {
	s.App.UpdateConfig(cfg)
	// Note: You may want to update config for other services too
}

func (s *ServiceImpl) CreateApp(ctx context.Context, req *CreateAppRequest) (*App, error) {
	return s.App.CreateApp(ctx, req)
}

func (s *ServiceImpl) GetPlatformApp(ctx context.Context) (*App, error) {
	return s.App.GetPlatformApp(ctx)
}

func (s *ServiceImpl) FindAppByID(ctx context.Context, id xid.ID) (*App, error) {
	return s.App.FindAppByID(ctx, id)
}

func (s *ServiceImpl) FindAppBySlug(ctx context.Context, slug string) (*App, error) {
	return s.App.FindAppBySlug(ctx, slug)
}

func (s *ServiceImpl) UpdateApp(ctx context.Context, id xid.ID, req *UpdateAppRequest) (*App, error) {
	return s.App.UpdateApp(ctx, id, req)
}

func (s *ServiceImpl) DeleteApp(ctx context.Context, id xid.ID) error {
	return s.App.DeleteApp(ctx, id)
}

func (s *ServiceImpl) ListApps(ctx context.Context, filter *ListAppsFilter) (*pagination.PageResponse[*App], error) {
	return s.App.ListApps(ctx, filter)
}

func (s *ServiceImpl) CountApps(ctx context.Context) (int, error) {
	return s.App.CountApps(ctx)
}

func (s *ServiceImpl) SetPlatformApp(ctx context.Context, newPlatformAppID xid.ID) error {
	return s.App.SetPlatformApp(ctx, newPlatformAppID)
}

func (s *ServiceImpl) IsPlatformApp(ctx context.Context, appID xid.ID) (bool, error) {
	return s.App.IsPlatformApp(ctx, appID)
}

// =============================================================================
// Member Operations Delegation
// =============================================================================

func (s *ServiceImpl) CreateMember(ctx context.Context, member *Member) (*Member, error) {
	return s.Member.CreateMember(ctx, member)
}

func (s *ServiceImpl) FindMemberByID(ctx context.Context, id xid.ID) (*Member, error) {
	return s.Member.FindMemberByID(ctx, id)
}

func (s *ServiceImpl) FindMember(ctx context.Context, appID, userID xid.ID) (*Member, error) {
	return s.Member.FindMember(ctx, appID, userID)
}

func (s *ServiceImpl) ListMembers(ctx context.Context, filter *ListMembersFilter) (*pagination.PageResponse[*Member], error) {
	return s.Member.ListMembers(ctx, filter)
}

func (s *ServiceImpl) GetUserMemberships(ctx context.Context, userID xid.ID) ([]*Member, error) {
	return s.Member.GetUserMemberships(ctx, userID)
}

func (s *ServiceImpl) UpdateMember(ctx context.Context, member *Member) error {
	return s.Member.UpdateMember(ctx, member)
}

func (s *ServiceImpl) DeleteMember(ctx context.Context, id xid.ID) error {
	return s.Member.DeleteMember(ctx, id)
}

func (s *ServiceImpl) CountMembers(ctx context.Context, appID xid.ID) (int, error) {
	return s.Member.CountMembers(ctx, appID)
}

func (s *ServiceImpl) IsUserMember(ctx context.Context, appID, userID xid.ID) (bool, error) {
	return s.Member.IsUserMember(ctx, appID, userID)
}

func (s *ServiceImpl) IsOwner(ctx context.Context, appID, userID xid.ID) (bool, error) {
	return s.Member.IsOwner(ctx, appID, userID)
}

func (s *ServiceImpl) IsAdmin(ctx context.Context, appID, userID xid.ID) (bool, error) {
	return s.Member.IsAdmin(ctx, appID, userID)
}

func (s *ServiceImpl) RequireOwner(ctx context.Context, appID, userID xid.ID) error {
	return s.Member.RequireOwner(ctx, appID, userID)
}

func (s *ServiceImpl) RequireAdmin(ctx context.Context, appID, userID xid.ID) error {
	return s.Member.RequireAdmin(ctx, appID, userID)
}

// =============================================================================
// Team Operations Delegation
// =============================================================================

func (s *ServiceImpl) CreateTeam(ctx context.Context, team *Team) error {
	return s.Team.CreateTeam(ctx, team)
}

func (s *ServiceImpl) FindTeamByID(ctx context.Context, id xid.ID) (*Team, error) {
	return s.Team.FindTeamByID(ctx, id)
}

func (s *ServiceImpl) FindTeamByName(ctx context.Context, appID xid.ID, name string) (*Team, error) {
	return s.Team.FindTeamByName(ctx, appID, name)
}

func (s *ServiceImpl) ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error) {
	return s.Team.ListTeams(ctx, filter)
}

func (s *ServiceImpl) UpdateTeam(ctx context.Context, team *Team) error {
	return s.Team.UpdateTeam(ctx, team)
}

func (s *ServiceImpl) DeleteTeam(ctx context.Context, id xid.ID) error {
	return s.Team.DeleteTeam(ctx, id)
}

func (s *ServiceImpl) CountTeams(ctx context.Context, appID xid.ID) (int, error) {
	return s.Team.CountTeams(ctx, appID)
}

func (s *ServiceImpl) AddTeamMember(ctx context.Context, tm *TeamMember) (*TeamMember, error) {
	return s.Team.AddTeamMember(ctx, tm)
}

func (s *ServiceImpl) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	return s.Team.RemoveTeamMember(ctx, teamID, memberID)
}

func (s *ServiceImpl) ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error) {
	return s.Team.ListTeamMembers(ctx, filter)
}

func (s *ServiceImpl) CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error) {
	return s.Team.CountTeamMembers(ctx, teamID)
}

func (s *ServiceImpl) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	return s.Team.IsTeamMember(ctx, teamID, memberID)
}

func (s *ServiceImpl) ListMemberTeams(ctx context.Context, filter *ListMemberTeamsFilter) (*pagination.PageResponse[*Team], error) {
	return s.Team.ListMemberTeams(ctx, filter)
}

// =============================================================================
// Invitation Operations Delegation
// =============================================================================

func (s *ServiceImpl) CreateInvitation(ctx context.Context, inv *Invitation) error {
	return s.Invitation.CreateInvitation(ctx, inv)
}

func (s *ServiceImpl) FindInvitationByID(ctx context.Context, id xid.ID) (*Invitation, error) {
	return s.Invitation.FindInvitationByID(ctx, id)
}

func (s *ServiceImpl) FindInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	return s.Invitation.FindInvitationByToken(ctx, token)
}

func (s *ServiceImpl) ListInvitations(ctx context.Context, filter *ListInvitationsFilter) (*pagination.PageResponse[*Invitation], error) {
	return s.Invitation.ListInvitations(ctx, filter)
}

func (s *ServiceImpl) AcceptInvitation(ctx context.Context, token string, userID xid.ID) (*Member, error) {
	return s.Invitation.AcceptInvitation(ctx, token, userID)
}

func (s *ServiceImpl) DeclineInvitation(ctx context.Context, token string) error {
	return s.Invitation.DeclineInvitation(ctx, token)
}

func (s *ServiceImpl) CancelInvitation(ctx context.Context, id, cancellerUserID xid.ID) error {
	return s.Invitation.CancelInvitation(ctx, id, cancellerUserID)
}

func (s *ServiceImpl) ResendInvitation(ctx context.Context, id, resenderUserID xid.ID) (*Invitation, error) {
	return s.Invitation.ResendInvitation(ctx, id, resenderUserID)
}

func (s *ServiceImpl) CleanupExpiredInvitations(ctx context.Context) (int, error) {
	return s.Invitation.CleanupExpiredInvitations(ctx)
}

// =============================================================================
// RBAC Operations (from rbac.go - these methods will need to be added)
// =============================================================================

// CheckPermission checks if a user has permission to perform an action on a resource.
// This would typically be implemented in a shared RBAC helper or in the MemberService.
func (s *ServiceImpl) CheckPermission(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string) (bool, error) {
	// TODO: Implement RBAC check - likely needs to be in a shared location
	// For now, delegate to member service if it has RBAC methods
	return false, nil
}

// CheckPermissionWithContext checks permission with additional context variables for conditional permissions.
func (s *ServiceImpl) CheckPermissionWithContext(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string, contextVars map[string]string) (bool, error) {
	// TODO: Implement RBAC check with context
	return false, nil
}

// RequirePermission checks if a user has permission and returns an error if denied.
func (s *ServiceImpl) RequirePermission(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string) error {
	// TODO: Implement RBAC requirement check
	return nil
}

// RequirePermissionWithContext checks permission with context variables and returns error if denied.
func (s *ServiceImpl) RequirePermissionWithContext(ctx context.Context, userID, appID xid.ID, action, resourceType, resourceID string, contextVars map[string]string) error {
	// TODO: Implement RBAC requirement check with context
	return nil
}

// Type assertion to ensure ServiceImpl implements Service.
var _ Service = (*ServiceImpl)(nil)
