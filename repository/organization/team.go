package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// organizationTeamRepository implements organization.TeamRepository using Bun
type organizationTeamRepository struct {
	db *bun.DB
}

// NewOrganizationTeamRepository creates a new organization team repository
func NewOrganizationTeamRepository(db *bun.DB) organization.TeamRepository {
	return &organizationTeamRepository{db: db}
}

// Create creates a new team
func (r *organizationTeamRepository) Create(ctx context.Context, team *organization.Team) error {
	schemaTeam := team.ToSchema()
	_, err := r.db.NewInsert().
		Model(schemaTeam).
		Exec(ctx)
	return err
}

// FindByID retrieves a team by ID
func (r *organizationTeamRepository) FindByID(ctx context.Context, id xid.ID) (*organization.Team, error) {
	schemaTeam := new(schema.OrganizationTeam)
	err := r.db.NewSelect().
		Model(schemaTeam).
		Where("uot.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaTeam(schemaTeam), nil
}

// FindByName retrieves a team by name within an organization
func (r *organizationTeamRepository) FindByName(ctx context.Context, orgID xid.ID, name string) (*organization.Team, error) {
	schemaTeam := new(schema.OrganizationTeam)
	err := r.db.NewSelect().
		Model(schemaTeam).
		Where("organization_id = ? AND name = ?", orgID, name).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaTeam(schemaTeam), nil
}

// FindByExternalID retrieves a team by external ID (for SCIM lookups)
func (r *organizationTeamRepository) FindByExternalID(ctx context.Context, orgID xid.ID, externalID string) (*organization.Team, error) {
	schemaTeam := new(schema.OrganizationTeam)
	err := r.db.NewSelect().
		Model(schemaTeam).
		Where("organization_id = ? AND external_id = ?", orgID, externalID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaTeam(schemaTeam), nil
}

// ListByOrganization lists all teams in an organization with pagination
func (r *organizationTeamRepository) ListByOrganization(ctx context.Context, filter *organization.ListTeamsFilter) (*pagination.PageResponse[*organization.Team], error) {
	var schemaTeams []*schema.OrganizationTeam

	query := r.db.NewSelect().
		Model(&schemaTeams).
		Where("organization_id = ?", filter.OrganizationID).
		Order("created_at ASC")

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	// Convert to DTOs
	teams := organization.FromSchemaTeams(schemaTeams)

	return pagination.NewPageResponse(teams, int64(total), &filter.PaginationParams), nil
}

// Update updates a team
func (r *organizationTeamRepository) Update(ctx context.Context, team *organization.Team) error {
	schemaTeam := team.ToSchema()
	_, err := r.db.NewUpdate().
		Model(schemaTeam).
		WherePK().
		Exec(ctx)
	return err
}

// Delete deletes a team
func (r *organizationTeamRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.OrganizationTeam)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CountByOrganization counts teams in an organization
func (r *organizationTeamRepository) CountByOrganization(ctx context.Context, orgID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.OrganizationTeam)(nil)).
		Where("organization_id = ?", orgID).
		Count(ctx)
	return count, err
}

// AddMember adds a member to a team
func (r *organizationTeamRepository) AddMember(ctx context.Context, tm *organization.TeamMember) error {
	schemaTeamMember := tm.ToSchema()
	_, err := r.db.NewInsert().
		Model(schemaTeamMember).
		Exec(ctx)
	return err
}

// RemoveMember removes a member from a team
func (r *organizationTeamRepository) RemoveMember(ctx context.Context, teamID, memberID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.OrganizationTeamMember)(nil)).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Exec(ctx)
	return err
}

// ListMembers lists all members of a team with pagination
func (r *organizationTeamRepository) ListMembers(ctx context.Context, filter *organization.ListTeamMembersFilter) (*pagination.PageResponse[*organization.TeamMember], error) {
	var schemaTeamMembers []*schema.OrganizationTeamMember

	query := r.db.NewSelect().
		Model(&schemaTeamMembers).
		Where("team_id = ?", filter.TeamID).
		Order("joined_at ASC")

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	// Convert to DTOs
	teamMembers := organization.FromSchemaTeamMembers(schemaTeamMembers)

	return pagination.NewPageResponse(teamMembers, int64(total), &filter.PaginationParams), nil
}

// CountMembers counts members in a team
func (r *organizationTeamRepository) CountMembers(ctx context.Context, teamID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.OrganizationTeamMember)(nil)).
		Where("team_id = ?", teamID).
		Count(ctx)
	return count, err
}

// IsTeamMember checks if a member belongs to a team
func (r *organizationTeamRepository) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*schema.OrganizationTeamMember)(nil)).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Type assertion to ensure organizationTeamRepository implements organization.TeamRepository
var _ organization.TeamRepository = (*organizationTeamRepository)(nil)
