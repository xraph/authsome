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

// teamMemberWithUser holds a team member joined with user data (via organization_members)
type teamMemberWithUser struct {
	schema.OrganizationTeamMember
	UserID          xid.ID `bun:"user_id"`
	UserName        string `bun:"user_name"`
	UserEmail       string `bun:"user_email"`
	UserImage       string `bun:"user_image"`
	UserUsername    string `bun:"user_username"`
	UserDisplayName string `bun:"user_display_username"`
}

// toTeamMemberWithUserInfo converts teamMemberWithUser to organization.TeamMember with UserInfo populated
func (tm *teamMemberWithUser) toTeamMemberWithUserInfo() *organization.TeamMember {
	teamMember := organization.FromSchemaTeamMember(&tm.OrganizationTeamMember)
	if teamMember != nil {
		teamMember.User = &organization.UserInfo{
			ID:              tm.UserID,
			Name:            tm.UserName,
			Email:           tm.UserEmail,
			Image:           tm.UserImage,
			Username:        tm.UserUsername,
			DisplayUsername: tm.UserDisplayName,
		}
	}
	return teamMember
}

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
	var teamMembersWithUsers []*teamMemberWithUser

	// Join team_members -> organization_members -> users to get user info
	query := r.db.NewSelect().
		Model((*schema.OrganizationTeamMember)(nil)).
		ColumnExpr("uotm.*").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.name AS user_name").
		ColumnExpr("u.email AS user_email").
		ColumnExpr("u.image AS user_image").
		ColumnExpr("u.username AS user_username").
		ColumnExpr("u.display_username AS user_display_username").
		Join("LEFT JOIN organization_members AS om ON om.id = uotm.member_id").
		Join("LEFT JOIN users AS u ON u.id = om.user_id").
		Where("uotm.team_id = ?", filter.TeamID).
		Order("uotm.joined_at ASC")

	// Get total count (separate query for accuracy)
	countQuery := r.db.NewSelect().
		Model((*schema.OrganizationTeamMember)(nil)).
		Where("team_id = ?", filter.TeamID)
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	if err := query.Scan(ctx, &teamMembersWithUsers); err != nil {
		return nil, err
	}

	// Convert to DTOs with user info
	teamMembers := make([]*organization.TeamMember, len(teamMembersWithUsers))
	for i, tm := range teamMembersWithUsers {
		teamMembers[i] = tm.toTeamMemberWithUserInfo()
	}

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

// FindTeamMemberByID retrieves a team member by its ID
func (r *organizationTeamRepository) FindTeamMemberByID(ctx context.Context, id xid.ID) (*organization.TeamMember, error) {
	schemaTeamMember := new(schema.OrganizationTeamMember)
	err := r.db.NewSelect().
		Model(schemaTeamMember).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team member not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaTeamMember(schemaTeamMember), nil
}

// FindTeamMember retrieves a team member by team ID and member ID
func (r *organizationTeamRepository) FindTeamMember(ctx context.Context, teamID, memberID xid.ID) (*organization.TeamMember, error) {
	schemaTeamMember := new(schema.OrganizationTeamMember)
	err := r.db.NewSelect().
		Model(schemaTeamMember).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team member not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaTeamMember(schemaTeamMember), nil
}

// ListMemberTeams retrieves all teams that a member belongs to
func (r *organizationTeamRepository) ListMemberTeams(ctx context.Context, memberID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*organization.Team], error) {
	var schemaTeams []*schema.OrganizationTeam

	// Query teams through the team_members join table
	query := r.db.NewSelect().
		Model(&schemaTeams).
		Join("INNER JOIN organization_team_members AS otm ON otm.team_id = uot.id").
		Where("otm.member_id = ?", memberID).
		Order("uot.created_at ASC")

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

	return pagination.NewPageResponse(teams, int64(total), filter), nil
}

// Type assertion to ensure organizationTeamRepository implements organization.TeamRepository
var _ organization.TeamRepository = (*organizationTeamRepository)(nil)
