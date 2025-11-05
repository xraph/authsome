package organization

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// Config represents organization service configuration
type Config struct {
	// ModeSaaS controls multi-tenant behavior when true
	ModeSaaS bool
	// PlatformOrganizationID is the ID of the platform org (super admin)
	PlatformOrganizationID xid.ID
}

// Service provides organization management operations
type Service struct {
	repo   Repository
	config Config
}

// NewService creates a new organization service
func NewService(repo Repository, cfg Config) *Service {
	return &Service{repo: repo, config: cfg}
}

// CreateOrganization creates a new organization
func (s *Service) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*Organization, error) {
	id := xid.New()
	now := time.Now().UTC()
	org := &Organization{
		ID:        id,
		Name:      req.Name,
		Slug:      req.Slug,
		Logo:      req.Logo,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// FindOrganizationByID returns an organization by ID
func (s *Service) FindOrganizationByID(ctx context.Context, id xid.ID) (*Organization, error) {
	return s.repo.FindOrganizationByID(ctx, id)
}

// FindOrganizationBySlug returns an organization by slug
func (s *Service) FindOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.repo.FindOrganizationBySlug(ctx, slug)
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, id xid.ID, req *UpdateOrganizationRequest) (*Organization, error) {
	org, err := s.repo.FindOrganizationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Logo != nil {
		org.Logo = *req.Logo
	}
	if req.Metadata != nil {
		org.Metadata = req.Metadata
	}
	org.UpdatedAt = time.Now()
	if err := s.repo.UpdateOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// DeleteOrganization deletes an organization by ID
func (s *Service) DeleteOrganization(ctx context.Context, id xid.ID) error {
	return s.repo.DeleteOrganization(ctx, id)
}

// ListOrganizations returns a paginated list of organizations
func (s *Service) ListOrganizations(ctx context.Context, limit, offset int) ([]*Organization, error) {
	return s.repo.ListOrganizations(ctx, limit, offset)
}

// CountOrganizations returns total number of organizations
func (s *Service) CountOrganizations(ctx context.Context) (int, error) {
	return s.repo.CountOrganizations(ctx)
}

// CreateMember adds a new member to an organization
func (s *Service) CreateMember(ctx context.Context, member *Member) error {
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()
	return s.repo.CreateMember(ctx, member)
}

// FindMemberByID finds a member by ID
func (s *Service) FindMemberByID(ctx context.Context, id xid.ID) (*Member, error) {
	return s.repo.FindMemberByID(ctx, id)
}

// FindMember finds a member by orgID and userID
func (s *Service) FindMember(ctx context.Context, orgID, userID xid.ID) (*Member, error) {
	return s.repo.FindMember(ctx, orgID, userID)
}

// ListMembers lists members in an organization
func (s *Service) ListMembers(ctx context.Context, orgID xid.ID, limit, offset int) ([]*Member, error) {
	return s.repo.ListMembers(ctx, orgID, limit, offset)
}

// CountMembers returns total number of members in an organization
func (s *Service) CountMembers(ctx context.Context, orgID xid.ID) (int, error) {
	return s.repo.CountMembers(ctx, orgID)
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

// ListTeams lists teams in an organization with pagination
func (s *Service) ListTeams(ctx context.Context, orgID xid.ID, limit, offset int) ([]*Team, error) {
	return s.repo.ListTeams(ctx, orgID, limit, offset)
}

// CountTeams returns total number of teams in an organization
func (s *Service) CountTeams(ctx context.Context, orgID xid.ID) (int, error) {
	return s.repo.CountTeams(ctx, orgID)
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

// CreateInvitation creates an organization invitation
func (s *Service) CreateInvitation(ctx context.Context, inv *Invitation) error {
	inv.CreatedAt = time.Now()
	return s.repo.CreateInvitation(ctx, inv)
}
