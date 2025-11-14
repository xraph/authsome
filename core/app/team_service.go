package app

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
	memberRepo MemberRepository // For validation (members exist)
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

// CreateTeam creates a new team
func (s *TeamService) CreateTeam(ctx context.Context, team *Team) error {
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()
	return s.repo.CreateTeam(ctx, team.ToSchema())
}

// FindTeamByID finds a team by ID
func (s *TeamService) FindTeamByID(ctx context.Context, id xid.ID) (*Team, error) {
	teamSchema, err := s.repo.FindTeamByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return FromSchemaTeam(teamSchema), nil
}

// FindTeamByName finds a team by name within an app
func (s *TeamService) FindTeamByName(ctx context.Context, appID xid.ID, name string) (*Team, error) {
	schemaTeam, err := s.repo.FindTeamByName(ctx, appID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to find team by name: %w", err)
	}
	return FromSchemaTeam(schemaTeam), nil
}

// ListTeams lists teams in an app with pagination
func (s *TeamService) ListTeams(ctx context.Context, filter *ListTeamsFilter) (*pagination.PageResponse[*Team], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	response, err := s.repo.ListTeams(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	// Convert schema teams to DTOs
	teams := FromSchemaTeams(response.Data)
	return &pagination.PageResponse[*Team]{
		Data:       teams,
		Pagination: response.Pagination,
	}, nil
}

// UpdateTeam updates a team
func (s *TeamService) UpdateTeam(ctx context.Context, team *Team) error {
	team.UpdatedAt = time.Now()
	return s.repo.UpdateTeam(ctx, team.ToSchema())
}

// DeleteTeam deletes a team by ID
func (s *TeamService) DeleteTeam(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteTeam(ctx, id)
}

// CountTeams returns total number of teams in an app
func (s *TeamService) CountTeams(ctx context.Context, appID xid.ID) (int, error) {
	return s.repo.CountTeams(ctx, appID)
}

// AddTeamMember adds a member to a team
func (s *TeamService) AddTeamMember(ctx context.Context, tm *TeamMember) (*TeamMember, error) {
	if err := s.repo.AddTeamMember(ctx, tm.ToSchema()); err != nil {
		return nil, err
	}
	return tm, nil
}

// RemoveTeamMember removes a member from a team
func (s *TeamService) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	return s.repo.RemoveTeamMember(ctx, teamID, memberID)
}

// ListTeamMembers lists members of a team with pagination
func (s *TeamService) ListTeamMembers(ctx context.Context, filter *ListTeamMembersFilter) (*pagination.PageResponse[*TeamMember], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	response, err := s.repo.ListTeamMembers(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}

	// Convert schema team members to DTOs
	teamMembers := FromSchemaTeamMembers(response.Data)
	return &pagination.PageResponse[*TeamMember]{
		Data:       teamMembers,
		Pagination: response.Pagination,
	}, nil
}

// CountTeamMembers returns total number of members in a team
func (s *TeamService) CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error) {
	return s.repo.CountTeamMembers(ctx, teamID)
}

// IsTeamMember checks if a member is part of a team
func (s *TeamService) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	return s.repo.IsTeamMember(ctx, teamID, memberID)
}

// ListMemberTeams lists all teams a member belongs to with pagination
func (s *TeamService) ListMemberTeams(ctx context.Context, filter *ListMemberTeamsFilter) (*pagination.PageResponse[*Team], error) {
	// Validate pagination params
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pagination params: %w", err)
	}

	response, err := s.repo.ListMemberTeams(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list member teams: %w", err)
	}

	// Convert schema teams to DTOs
	teams := FromSchemaTeams(response.Data)
	return &pagination.PageResponse[*Team]{
		Data:       teams,
		Pagination: response.Pagination,
	}, nil
}

// Type assertion to ensure TeamService implements TeamOperations
var _ TeamOperations = (*TeamService)(nil)
