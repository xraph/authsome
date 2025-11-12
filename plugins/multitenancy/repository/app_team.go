package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/app"
)

// AppTeamRepository implements app.TeamRepository
type AppTeamRepository struct {
	db *bun.DB
}

// NewAppTeamRepository creates a new app team repository
func NewAppTeamRepository(db *bun.DB) *AppTeamRepository {
	return &AppTeamRepository{db: db}
}

// Create creates a new team
func (r *AppTeamRepository) Create(ctx context.Context, team *app.Team) error {
	_, err := r.db.NewInsert().Model(team).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	return nil
}

// FindByID finds a team by ID
func (r *AppTeamRepository) FindByID(ctx context.Context, id xid.ID) (*app.Team, error) {
	team := &app.Team{}
	err := r.db.NewSelect().Model(team).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrTeamNotFound
		}
		return nil, fmt.Errorf("failed to find team: %w", err)
	}
	return team, nil
}

// ListByApp lists teams by app with pagination
func (r *AppTeamRepository) ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*app.Team, error) {
	var teams []*app.Team

	// Get paginated results
	err := r.db.NewSelect().
		Model(&teams).
		Where("organization_id = ?", appID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}

	return teams, nil
}

// Update updates a team
func (r *AppTeamRepository) Update(ctx context.Context, team *app.Team) error {
	_, err := r.db.NewUpdate().Model(team).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}
	return nil
}

// Delete deletes a team
func (r *AppTeamRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*app.Team)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// CountByApp returns the total number of teams in an app
func (r *AppTeamRepository) CountByApp(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*app.Team)(nil)).
		Where("organization_id = ?", appID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count teams by app: %w", err)
	}
	return count, nil
}

// AddMember adds a member to a team
func (r *AppTeamRepository) AddMember(ctx context.Context, teamID, memberID xid.ID, role string) error {
	teamMember := &app.TeamMember{
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
func (r *AppTeamRepository) RemoveMember(ctx context.Context, teamID, memberID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*app.TeamMember)(nil)).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	return nil
}

// ListMembers lists members of a team
func (r *AppTeamRepository) ListMembers(ctx context.Context, teamID xid.ID) ([]*app.Member, error) {
	var members []*app.Member

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
