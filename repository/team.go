package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// TeamRepository handles team data access using schema models
type TeamRepository struct {
	db *bun.DB
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(db *bun.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// Create creates a new team
func (r *TeamRepository) Create(ctx context.Context, team *schema.Team) error {
	_, err := r.db.NewInsert().Model(team).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

// FindByID finds a team by ID
func (r *TeamRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Team, error) {
	team := &schema.Team{}
	err := r.db.NewSelect().Model(team).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team not found")
		}
		return nil, fmt.Errorf("failed to find team: %w", err)
	}
	return team, nil
}

// FindByExternalID finds a team by external ID (for SCIM lookups)
func (r *TeamRepository) FindByExternalID(ctx context.Context, appID xid.ID, externalID string) (*schema.Team, error) {
	team := &schema.Team{}
	err := r.db.NewSelect().
		Model(team).
		Where("app_id = ? AND external_id = ?", appID, externalID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("team not found")
		}
		return nil, fmt.Errorf("failed to find team by external_id: %w", err)
	}
	return team, nil
}

// ListByApp lists teams by app with pagination
func (r *TeamRepository) ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.Team, int64, error) {
	var teams []*schema.Team

	// Get total count
	total, err := r.db.NewSelect().
		Model((*schema.Team)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count teams: %w", err)
	}

	// Get paginated results
	err = r.db.NewSelect().
		Model(&teams).
		Where("app_id = ?", appID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list teams: %w", err)
	}

	return teams, int64(total), nil
}

// Update updates a team
func (r *TeamRepository) Update(ctx context.Context, team *schema.Team) error {
	_, err := r.db.NewUpdate().Model(team).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}
	return nil
}

// Delete deletes a team
func (r *TeamRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Team)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// CountByApp returns the total number of teams in an app
func (r *TeamRepository) CountByApp(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Team)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count teams by app: %w", err)
	}
	return count, nil
}

// AddMember adds a member to a team
func (r *TeamRepository) AddMember(ctx context.Context, teamID, memberID xid.ID, role string) error {
	teamMember := &schema.TeamMember{
		ID:       xid.New(),
		TeamID:   teamID,
		MemberID: memberID,
	}
	_, err := r.db.NewInsert().Model(teamMember).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}
	return nil
}

// RemoveMember removes a member from a team
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, memberID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.TeamMember)(nil)).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	return nil
}

// ListMembers lists members of a team
func (r *TeamRepository) ListMembers(ctx context.Context, teamID xid.ID) ([]*schema.Member, error) {
	var members []*schema.Member

	// Join team_members with members to get full member details
	err := r.db.NewSelect().
		Model(&members).
		Join("JOIN team_members tm ON tm.member_id = members.id").
		Where("tm.team_id = ?", teamID).
		Order("members.joined_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list team members: %w", err)
	}

	return members, nil
}

// CountTeamMembers returns the total number of members in a team
func (r *TeamRepository) CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.TeamMember)(nil)).
		Where("team_id = ?", teamID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count team members: %w", err)
	}
	return count, nil
}
