package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
)

// TeamRepository implements organization.TeamRepository
type TeamRepository struct {
	db *bun.DB
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(db *bun.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// Create creates a new team
func (r *TeamRepository) Create(ctx context.Context, team *organization.Team) error {
	_, err := r.db.NewInsert().Model(team).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

// FindByID finds a team by ID
func (r *TeamRepository) FindByID(ctx context.Context, id string) (*organization.Team, error) {
	team := &organization.Team{}
	err := r.db.NewSelect().Model(team).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrTeamNotFound
		}
		return nil, fmt.Errorf("failed to find team: %w", err)
	}
	return team, nil
}

// Update updates a team
func (r *TeamRepository) Update(ctx context.Context, team *organization.Team) error {
	_, err := r.db.NewUpdate().Model(team).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}
	return nil
}

// Delete deletes a team
func (r *TeamRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*organization.Team)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// ListByOrganization lists teams by organization with pagination
func (r *TeamRepository) ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*organization.Team, error) {
	var teams []*organization.Team
	
	// Get paginated results
	err := r.db.NewSelect().
		Model(&teams).
		Where("organization_id = ?", orgID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	
	return teams, nil
}

// AddMember adds a member to a team
func (r *TeamRepository) AddMember(ctx context.Context, teamID, memberID, role string) error {
	teamMember := &organization.TeamMember{
		TeamID:   teamID,
		MemberID: memberID,
		Role:     role,
	}
	_, err := r.db.NewInsert().Model(teamMember).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}
	return nil
}

// RemoveMember removes a member from a team
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, memberID string) error {
	_, err := r.db.NewDelete().
		Model((*organization.TeamMember)(nil)).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	return nil
}

// ListMembers lists members of a team
func (r *TeamRepository) ListMembers(ctx context.Context, teamID string) ([]*organization.Member, error) {
	var members []*organization.Member
	
	// Join team_members with members to get full member details
	err := r.db.NewSelect().
		Model(&members).
		Join("JOIN team_members tm ON tm.member_id = members.id").
		Where("tm.team_id = ?", teamID).
		Order("members.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}
	
	return members, nil
}

// CountByOrganization returns the total number of teams in an organization
func (r *TeamRepository) CountByOrganization(ctx context.Context, orgID string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*organization.Team)(nil)).
		Where("organization_id = ?", orgID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count teams by organization: %w", err)
	}
	return count, nil
}