package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// organizationTeamRepository implements OrganizationTeamRepository using Bun
type organizationTeamRepository struct {
	db *bun.DB
}

// NewOrganizationTeamRepository creates a new organization team repository
func NewOrganizationTeamRepository(db *bun.DB) *organizationTeamRepository {
	return &organizationTeamRepository{db: db}
}

// Create creates a new team
func (r *organizationTeamRepository) Create(ctx context.Context, team *schema.OrganizationTeam) error {
	_, err := r.db.NewInsert().
		Model(team).
		Exec(ctx)
	return err
}

// FindByID retrieves a team by ID
func (r *organizationTeamRepository) FindByID(ctx context.Context, id xid.ID) (*schema.OrganizationTeam, error) {
	team := new(schema.OrganizationTeam)
	err := r.db.NewSelect().
		Model(team).
		Relation("Organization").
		Where("uot.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	return team, err
}

// FindByName retrieves a team by name within an organization
func (r *organizationTeamRepository) FindByName(ctx context.Context, orgID xid.ID, name string) (*schema.OrganizationTeam, error) {
	team := new(schema.OrganizationTeam)
	err := r.db.NewSelect().
		Model(team).
		Where("organization_id = ? AND name = ?", orgID, name).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found")
	}
	return team, err
}

// ListByOrganization lists all teams in an organization
func (r *organizationTeamRepository) ListByOrganization(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationTeam, error) {
	var teams []*schema.OrganizationTeam

	query := r.db.NewSelect().
		Model(&teams).
		Where("organization_id = ?", orgID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return teams, err
}

// Update updates a team
func (r *organizationTeamRepository) Update(ctx context.Context, team *schema.OrganizationTeam) error {
	_, err := r.db.NewUpdate().
		Model(team).
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
func (r *organizationTeamRepository) AddMember(ctx context.Context, teamMember *schema.OrganizationTeamMember) error {
	_, err := r.db.NewInsert().
		Model(teamMember).
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

// ListMembers lists all members of a team
func (r *organizationTeamRepository) ListMembers(ctx context.Context, teamID xid.ID, limit, offset int) ([]*schema.OrganizationTeamMember, error) {
	var members []*schema.OrganizationTeamMember

	query := r.db.NewSelect().
		Model(&members).
		Relation("Member").
		Relation("Member.User").
		Where("team_id = ?", teamID).
		Order("joined_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return members, err
}

// CountMembers counts members in a team
func (r *organizationTeamRepository) CountMembers(ctx context.Context, teamID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.OrganizationTeamMember)(nil)).
		Where("team_id = ?", teamID).
		Count(ctx)
	return count, err
}
