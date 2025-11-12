package app

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// Config represents app service configuration
type Config struct {
	// PlatformAppID is the ID of the platform app (super admin)
	PlatformAppID xid.ID
}

// Service provides app management operations
type Service struct {
	repo   Repository
	config Config
}

// NewService creates a new app service
func NewService(repo Repository, cfg Config) *Service {
	return &Service{repo: repo, config: cfg}
}

// CreateApp creates a new app
func (s *Service) CreateApp(ctx context.Context, req *CreateAppRequest) (*App, error) {
	id := xid.New()
	now := time.Now().UTC()
	app := &App{
		ID:        id,
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateApp(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

// FindAppByID returns an app by ID
func (s *Service) FindAppByID(ctx context.Context, id xid.ID) (*App, error) {
	return s.repo.FindAppByID(ctx, id)
}

// FindAppBySlug returns an app by slug
func (s *Service) FindAppBySlug(ctx context.Context, slug string) (*App, error) {
	return s.repo.FindAppBySlug(ctx, slug)
}

// UpdateApp updates an app
func (s *Service) UpdateApp(ctx context.Context, id xid.ID, req *UpdateAppRequest) (*App, error) {
	app, err := s.repo.FindAppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.Logo != nil {
		app.Logo = *req.Logo
	}
	if req.Metadata != nil {
		app.Metadata = req.Metadata
	}
	app.UpdatedAt = time.Now()
	if err := s.repo.UpdateApp(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

// DeleteApp deletes an app by ID
func (s *Service) DeleteApp(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteApp(ctx, id)
}

// ListApps returns a paginated list of apps
func (s *Service) ListApps(ctx context.Context, limit, offset int) ([]*App, error) {
	return s.repo.ListApps(ctx, limit, offset)
}

// CountApps returns total number of apps
func (s *Service) CountApps(ctx context.Context) (int, error) {
	return s.repo.CountApps(ctx)
}

// CreateMember adds a new member to an app
func (s *Service) CreateMember(ctx context.Context, member *Member) error {
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()
	return s.repo.CreateMember(ctx, member)
}

// FindMemberByID finds a member by ID
func (s *Service) FindMemberByID(ctx context.Context, id xid.ID) (*Member, error) {
	return s.repo.FindMemberByID(ctx, id)
}

// FindMember finds a member by appID and userID
func (s *Service) FindMember(ctx context.Context, appID, userID xid.ID) (*Member, error) {
	return s.repo.FindMember(ctx, appID, userID)
}

// ListMembers lists members in an app
func (s *Service) ListMembers(ctx context.Context, appID xid.ID, limit, offset int) ([]*Member, error) {
	return s.repo.ListMembers(ctx, appID, limit, offset)
}

// CountMembers returns total number of members in an app
func (s *Service) CountMembers(ctx context.Context, appID xid.ID) (int, error) {
	return s.repo.CountMembers(ctx, appID)
}

// UpdateMember updates a member
func (s *Service) UpdateMember(ctx context.Context, member *Member) error {
	member.UpdatedAt = time.Now()
	return s.repo.UpdateMember(ctx, member)
}

// DeleteMember deletes a member by ID
func (s *Service) DeleteMember(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteMember(ctx, id)
}

// CreateTeam creates a new team
func (s *Service) CreateTeam(ctx context.Context, team *Team) error {
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()
	return s.repo.CreateTeam(ctx, team)
}

// FindTeamByID finds a team by ID
func (s *Service) FindTeamByID(ctx context.Context, id xid.ID) (*Team, error) {
	return s.repo.FindTeamByID(ctx, id)
}

// ListTeams lists teams in an app with pagination
func (s *Service) ListTeams(ctx context.Context, appID xid.ID, limit, offset int) ([]*Team, error) {
	return s.repo.ListTeams(ctx, appID, limit, offset)
}

// CountTeams returns total number of teams in an app
func (s *Service) CountTeams(ctx context.Context, appID xid.ID) (int, error) {
	return s.repo.CountTeams(ctx, appID)
}

// UpdateTeam updates a team
func (s *Service) UpdateTeam(ctx context.Context, team *Team) error {
	team.UpdatedAt = time.Now()
	return s.repo.UpdateTeam(ctx, team)
}

// DeleteTeam deletes a team by ID
func (s *Service) DeleteTeam(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteTeam(ctx, id)
}

// AddTeamMember adds a member to a team
func (s *Service) AddTeamMember(ctx context.Context, tm *TeamMember) error {
	return s.repo.AddTeamMember(ctx, tm)
}

// RemoveTeamMember removes a member from a team
func (s *Service) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	return s.repo.RemoveTeamMember(ctx, teamID, memberID)
}

// ListTeamMembers lists members of a team with pagination
func (s *Service) ListTeamMembers(ctx context.Context, teamID xid.ID, limit, offset int) ([]*TeamMember, error) {
	return s.repo.ListTeamMembers(ctx, teamID, limit, offset)
}

// CountTeamMembers returns total number of members in a team
func (s *Service) CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error) {
	return s.repo.CountTeamMembers(ctx, teamID)
}

// CreateInvitation creates an app invitation
func (s *Service) CreateInvitation(ctx context.Context, inv *Invitation) error {
	inv.CreatedAt = time.Now()
	return s.repo.CreateInvitation(ctx, inv)
}
