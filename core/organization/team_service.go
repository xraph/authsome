package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/rbac"
)

// TeamService handles team aggregate operations
type TeamService struct {
	repo       TeamRepository
	memberRepo MemberRepository // For authorization checks
	config     Config
	rbacSvc    *rbac.Service
}

// NewTeamService creates a new team service
func NewTeamService(repo TeamRepository, memberRepo MemberRepository, cfg Config, rbacSvc *rbac.Service) *TeamService {
	return &TeamService{
		repo:       repo,
		memberRepo: memberRepo,
		config:     cfg,
		rbacSvc:    rbacSvc,
	}
}

// CreateTeam creates a new team in an organization
func (s *TeamService) CreateTeam(ctx context.Context, orgID xid.ID, req *CreateTeamRequest, creatorUserID xid.ID) (*Team, error) {
	// Allow system operations (zero user ID) for SCIM and automated provisioning
	// For regular user operations, verify creator is a member
	if !creatorUserID.IsNil() {
		member, err := s.memberRepo.FindByUserAndOrg(ctx, creatorUserID, orgID)
		if err != nil || member == nil {
			return nil, fmt.Errorf("only organization members can create teams")
		}
	}

	// Check team limit
	count, err := s.repo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to count teams: %w", err)
	}
	if count >= s.config.MaxTeamsPerOrganization {
		return nil, MaxTeamsReached(s.config.MaxTeamsPerOrganization)
	}

	// Handle description pointer
	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	now := time.Now().UTC()
	team := &Team{
		ID:             xid.New(),
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    description,
		Metadata:       req.Metadata,
		ProvisionedBy:  &req.ProvisionedBy,
		ExternalID:     &req.ExternalID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return team, nil
}

// FindTeamByID retrieves a team by ID
func (s *TeamService) FindTeamByID(ctx context.Context, id xid.ID) (*Team, error) {
	team, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, TeamNotFound()
	}
	return team, nil
}

// FindTeamByName retrieves a team by name within an organization
func (s *TeamService) FindTeamByName(ctx context.Context, orgID xid.ID, name string) (*Team, error) {
	team, err := s.repo.FindByName(ctx, orgID, name)
	if err != nil {
		return nil, TeamNotFound()
	}
	return team, nil
}

// ListTeams lists teams in an organization
func (s *TeamService) ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListByOrganization(ctx, filter)
}

// UpdateTeam updates a team
func (s *TeamService) UpdateTeam(ctx context.Context, id xid.ID, req *UpdateTeamRequest, updaterUserID xid.ID) (*Team, error) {
	team, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, TeamNotFound()
	}

	// Allow system operations (zero user ID) for SCIM and automated provisioning
	// For regular user operations, verify updater is admin
	if !updaterUserID.IsNil() {
		member, err := s.memberRepo.FindByUserAndOrg(ctx, updaterUserID, team.OrganizationID)
		if err != nil || member == nil {
			return nil, NotAdmin()
		}
		if member.Role != RoleOwner && member.Role != RoleAdmin {
			return nil, NotAdmin()
		}
	}

	// Update fields
	if req.Name != nil {
		team.Name = *req.Name
	}
	if req.Description != nil {
		team.Description = *req.Description
	}
	if req.Metadata != nil {
		team.Metadata = req.Metadata
	}
	if req.ProvisionedBy != nil {
		team.ProvisionedBy = req.ProvisionedBy
	}
	if req.ExternalID != nil {
		team.ExternalID = req.ExternalID
	}
	team.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, team); err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return team, nil
}

// DeleteTeam deletes a team
func (s *TeamService) DeleteTeam(ctx context.Context, id, deleterUserID xid.ID) error {
	team, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return TeamNotFound()
	}

	// Allow system operations (zero user ID) for SCIM and automated provisioning
	// For regular user operations, verify deleter is admin
	if !deleterUserID.IsNil() {
		member, err := s.memberRepo.FindByUserAndOrg(ctx, deleterUserID, team.OrganizationID)
		if err != nil || member == nil {
			return NotAdmin()
		}
		if member.Role != RoleOwner && member.Role != RoleAdmin {
			return NotAdmin()
		}
	}

	return s.repo.Delete(ctx, id)
}

// AddTeamMember adds a member to a team
func (s *TeamService) AddTeamMember(ctx context.Context, teamID, memberID, adderUserID xid.ID) error {
	team, err := s.repo.FindByID(ctx, teamID)
	if err != nil {
		return TeamNotFound()
	}

	// Allow system operations (zero user ID) for SCIM and automated provisioning
	// For regular user operations, verify adder is admin
	if !adderUserID.IsNil() {
		member, err := s.memberRepo.FindByUserAndOrg(ctx, adderUserID, team.OrganizationID)
		if err != nil || member == nil {
			return NotAdmin()
		}
		if member.Role != RoleOwner && member.Role != RoleAdmin {
			return NotAdmin()
		}
	}

	now := time.Now().UTC()
	teamMember := &TeamMember{
		ID:        xid.New(),
		TeamID:    teamID,
		MemberID:  memberID,
		JoinedAt:  now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.AddMember(ctx, teamMember)
}

// RemoveTeamMember removes a member from a team
func (s *TeamService) RemoveTeamMember(ctx context.Context, teamID, memberID, removerUserID xid.ID) error {
	team, err := s.repo.FindByID(ctx, teamID)
	if err != nil {
		return TeamNotFound()
	}

	// Allow system operations (zero user ID) for SCIM and automated provisioning
	// For regular user operations, verify remover is admin
	if !removerUserID.IsNil() {
		member, err := s.memberRepo.FindByUserAndOrg(ctx, removerUserID, team.OrganizationID)
		if err != nil || member == nil {
			return NotAdmin()
		}
		if member.Role != RoleOwner && member.Role != RoleAdmin {
			return NotAdmin()
		}
	}

	return s.repo.RemoveMember(ctx, teamID, memberID)
}

// ListTeamMembers lists members of a team
func (s *TeamService) ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	return s.repo.ListMembers(ctx, filter)
}

// IsTeamMember checks if a member belongs to a team
func (s *TeamService) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	return s.repo.IsTeamMember(ctx, teamID, memberID)
}

// FindTeamMemberByID retrieves a team member by its ID
func (s *TeamService) FindTeamMemberByID(ctx context.Context, id xid.ID) (*TeamMember, error) {
	teamMember, err := s.repo.FindTeamMemberByID(ctx, id)
	if err != nil {
		return nil, TeamMemberNotFound()
	}
	return teamMember, nil
}

// FindTeamMember retrieves a team member by team ID and member ID
func (s *TeamService) FindTeamMember(ctx context.Context, teamID, memberID xid.ID) (*TeamMember, error) {
	teamMember, err := s.repo.FindTeamMember(ctx, teamID, memberID)
	if err != nil {
		return nil, TeamMemberNotFound()
	}
	return teamMember, nil
}

// ListMemberTeams retrieves all teams that a member belongs to
func (s *TeamService) ListMemberTeams(ctx context.Context, memberID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*Team], error) {
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}
	return s.repo.ListMemberTeams(ctx, memberID, filter)
}

// IsSCIMManaged checks if a team is managed via SCIM provisioning
func (s *TeamService) IsSCIMManaged(team *Team) bool {
	return team.ProvisionedBy != nil && *team.ProvisionedBy == "scim"
}

// IsTeamMemberSCIMManaged checks if a team membership is managed via SCIM provisioning
func (s *TeamService) IsTeamMemberSCIMManaged(teamMember *TeamMember) bool {
	return teamMember.ProvisionedBy != nil && *teamMember.ProvisionedBy == "scim"
}

// Type assertion to ensure TeamService implements TeamOperations
var _ TeamOperations = (*TeamService)(nil)
