package repository

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// AppRepository is a Bun-backed implementation of app repository
type AppRepository struct {
	db *bun.DB
}

func NewAppRepository(db *bun.DB) *AppRepository {
	return &AppRepository{db: db}
}

// ===== App Operations =====

func (r *AppRepository) CreateApp(ctx context.Context, app *schema.App) error {
	_, err := r.db.NewInsert().Model(app).Exec(ctx)
	return err
}

func (r *AppRepository) GetPlatformApp(ctx context.Context) (*schema.App, error) {
	app := new(schema.App)
	err := r.db.NewSelect().Model(app).
		Where("is_platform = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) FindAppByID(ctx context.Context, id xid.ID) (*schema.App, error) {
	app := new(schema.App)
	err := r.db.NewSelect().Model(app).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) FindAppBySlug(ctx context.Context, slug string) (*schema.App, error) {
	app := new(schema.App)
	err := r.db.NewSelect().Model(app).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) UpdateApp(ctx context.Context, app *schema.App) error {
	_, err := r.db.NewUpdate().Model(app).WherePK().Exec(ctx)
	return err
}

func (r *AppRepository) DeleteApp(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.App)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

func (r *AppRepository) ListApps(ctx context.Context, filter *app.ListAppsFilter) (*pagination.PageResponse[*schema.App], error) {
	var apps []*schema.App

	// Build base query with filters
	query := r.db.NewSelect().Model(&apps)
	if filter.IsPlatform != nil {
		query = query.Where("is_platform = ?", *filter.IsPlatform)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().Model((*schema.App)(nil))
	if filter.IsPlatform != nil {
		countQuery = countQuery.Where("is_platform = ?", *filter.IsPlatform)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination using helper
	qb := pagination.NewQueryBuilder(&filter.PaginationParams)
	query = qb.ApplyToQuery(query)

	// Execute query
	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(apps, int64(total), &filter.PaginationParams), nil
}

// CountApps returns total number of apps
func (r *AppRepository) CountApps(ctx context.Context) (int, error) {
	q := r.db.NewSelect().Model((*schema.App)(nil))
	return q.Count(ctx)
}

// ===== Member Operations =====
// Note: Most member operations have been moved to repository/member.go
// These wrapper methods exist for backward compatibility with core/app.Repository interface

// CreateMember creates a new member
func (r *AppRepository) CreateMember(ctx context.Context, member *schema.Member) error {
	_, err := r.db.NewInsert().Model(member).Exec(ctx)
	return err
}

// FindMemberByID finds a member by ID
func (r *AppRepository) FindMemberByID(ctx context.Context, id xid.ID) (*schema.Member, error) {
	member := new(schema.Member)
	err := r.db.NewSelect().Model(member).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return member, nil
}

// FindMember finds a member by app ID and user ID (for IsUserMember)
func (r *AppRepository) FindMember(ctx context.Context, appID, userID xid.ID) (*schema.Member, error) {
	member := new(schema.Member)
	err := r.db.NewSelect().Model(member).
		Where("organization_id = ? AND user_id = ?", appID, userID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return member, nil
}

// ListMembers lists members by app with optional filters and pagination
func (r *AppRepository) ListMembers(ctx context.Context, filter *app.ListMembersFilter) (*pagination.PageResponse[*schema.Member], error) {
	var members []*schema.Member

	// Build base query with filters
	query := r.db.NewSelect().Model(&members).Where("organization_id = ?", filter.AppID)
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().Model((*schema.Member)(nil)).Where("organization_id = ?", filter.AppID)
	if filter.Role != nil {
		countQuery = countQuery.Where("role = ?", *filter.Role)
	}
	if filter.Status != nil {
		countQuery = countQuery.Where("status = ?", *filter.Status)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination using helper
	qb := pagination.NewQueryBuilder(&filter.PaginationParams)
	query = qb.ApplyToQuery(query)

	// Execute query
	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(members, int64(total), &filter.PaginationParams), nil
}

// UpdateMember updates a member
func (r *AppRepository) UpdateMember(ctx context.Context, member *schema.Member) error {
	_, err := r.db.NewUpdate().Model(member).WherePK().Exec(ctx)
	return err
}

// DeleteMember deletes a member
func (r *AppRepository) DeleteMember(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Member)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// CountMembers returns the total number of members in an app
func (r *AppRepository) CountMembers(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Member)(nil)).
		Where("organization_id = ?", appID).
		Count(ctx)
	return count, err
}

// ListMembersByUser lists all memberships for a user across all apps
func (r *AppRepository) ListMembersByUser(ctx context.Context, userID xid.ID) ([]*schema.Member, error) {
	var members []*schema.Member
	err := r.db.NewSelect().Model(&members).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Relation("App").
		Scan(ctx)
	return members, err
}

// ===== Team Operations =====
// Note: Most team operations have been moved to repository/team.go

// CreateTeam creates a new team
func (r *AppRepository) CreateTeam(ctx context.Context, team *schema.Team) error {
	_, err := r.db.NewInsert().Model(team).Exec(ctx)
	return err
}

// FindTeamByID finds a team by ID
func (r *AppRepository) FindTeamByID(ctx context.Context, id xid.ID) (*schema.Team, error) {
	team := new(schema.Team)
	err := r.db.NewSelect().Model(team).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return team, nil
}

// ListTeams lists teams by app with pagination
func (r *AppRepository) ListTeams(ctx context.Context, filter *app.ListTeamsFilter) (*pagination.PageResponse[*schema.Team], error) {
	var teams []*schema.Team

	// Build base query
	query := r.db.NewSelect().Model(&teams).Where("organization_id = ?", filter.AppID)

	// Get total count before pagination
	total, err := r.db.NewSelect().
		Model((*schema.Team)(nil)).
		Where("organization_id = ?", filter.AppID).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination using helper
	qb := pagination.NewQueryBuilder(&filter.PaginationParams)
	query = qb.ApplyToQuery(query)

	// Execute query
	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(teams, int64(total), &filter.PaginationParams), nil
}

// UpdateTeam updates a team
func (r *AppRepository) UpdateTeam(ctx context.Context, team *schema.Team) error {
	_, err := r.db.NewUpdate().Model(team).WherePK().Exec(ctx)
	return err
}

// DeleteTeam deletes a team
func (r *AppRepository) DeleteTeam(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Team)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// CountTeams returns the total number of teams in an app
func (r *AppRepository) CountTeams(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Team)(nil)).
		Where("organization_id = ?", appID).
		Count(ctx)
	return count, err
}

// ===== Team Member Operations =====
// Note: Most team member operations have been moved to repository/team.go
// These wrapper methods exist for backward compatibility with core/app.Repository interface

// AddTeamMember adds a member to a team
func (r *AppRepository) AddTeamMember(ctx context.Context, tm *schema.TeamMember) error {
	_, err := r.db.NewInsert().Model(tm).Exec(ctx)
	return err
}

// RemoveTeamMember removes a member from a team
func (r *AppRepository) RemoveTeamMember(ctx context.Context, teamID, memberID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.TeamMember)(nil)).
		Where("team_id = ? AND member_id = ?", teamID, memberID).
		Exec(ctx)
	return err
}

// ListTeamMembers lists members of a team
func (r *AppRepository) ListTeamMembers(ctx context.Context, filter *app.ListTeamMembersFilter) (*pagination.PageResponse[*schema.TeamMember], error) {
	var teamMembers []*schema.TeamMember

	// Build base query
	query := r.db.NewSelect().Model(&teamMembers).Where("team_id = ?", filter.TeamID)

	// Get total count before pagination
	total, err := r.db.NewSelect().
		Model((*schema.TeamMember)(nil)).
		Where("team_id = ?", filter.TeamID).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination using helper
	qb := pagination.NewQueryBuilder(&filter.PaginationParams)
	query = qb.ApplyToQuery(query)

	// Execute query
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(teamMembers, int64(total), &filter.PaginationParams), nil
}

// CountTeamMembers returns the total number of members in a team
func (r *AppRepository) CountTeamMembers(ctx context.Context, teamID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.TeamMember)(nil)).
		Where("team_id = ?", teamID).
		Count(ctx)
	return count, err
}

// ===== Invitation Operations =====
// Note: Most invitation operations have been moved to repository/invitation.go
// These wrapper methods exist for backward compatibility with core/app.Repository interface

// CreateInvitation creates an invitation
func (r *AppRepository) CreateInvitation(ctx context.Context, inv *schema.Invitation) error {
	_, err := r.db.NewInsert().Model(inv).Exec(ctx)
	return err
}

// FindInvitationByID finds an invitation by ID
func (r *AppRepository) FindInvitationByID(ctx context.Context, id xid.ID) (*schema.Invitation, error) {
	invitation := new(schema.Invitation)
	err := r.db.NewSelect().Model(invitation).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return invitation, nil
}

// FindInvitationByToken finds an invitation by token
func (r *AppRepository) FindInvitationByToken(ctx context.Context, token string) (*schema.Invitation, error) {
	invitation := new(schema.Invitation)
	err := r.db.NewSelect().Model(invitation).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return invitation, nil
}

// ListInvitations lists invitations by app with optional status filter and pagination
func (r *AppRepository) ListInvitations(ctx context.Context, filter *app.ListInvitationsFilter) (*pagination.PageResponse[*schema.Invitation], error) {
	var invitations []*schema.Invitation

	// Build base query with filters
	query := r.db.NewSelect().Model(&invitations).Where("organization_id = ?", filter.AppID)
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Email != nil {
		query = query.Where("email = ?", *filter.Email)
	}

	// Get total count before pagination
	countQuery := r.db.NewSelect().Model((*schema.Invitation)(nil)).Where("organization_id = ?", filter.AppID)
	if filter.Status != nil {
		countQuery = countQuery.Where("status = ?", *filter.Status)
	}
	if filter.Email != nil {
		countQuery = countQuery.Where("email = ?", *filter.Email)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination using helper
	qb := pagination.NewQueryBuilder(&filter.PaginationParams)
	query = qb.ApplyToQuery(query)

	// Execute query
	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(invitations, int64(total), &filter.PaginationParams), nil
}

// UpdateInvitation updates an invitation
func (r *AppRepository) UpdateInvitation(ctx context.Context, inv *schema.Invitation) error {
	_, err := r.db.NewUpdate().Model(inv).WherePK().Exec(ctx)
	return err
}

// DeleteInvitation deletes an invitation
func (r *AppRepository) DeleteInvitation(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Invitation)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// DeleteExpiredInvitations deletes expired invitations
func (r *AppRepository) DeleteExpiredInvitations(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.Invitation)(nil)).
		Where("expires_at < CURRENT_TIMESTAMP").
		Where("status = ?", schema.InvitationStatusPending).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// ===== Team Query Operations =====

// FindTeamByName finds a team by name within an app
func (r *AppRepository) FindTeamByName(ctx context.Context, appID xid.ID, name string) (*schema.Team, error) {
	team := new(schema.Team)
	err := r.db.NewSelect().Model(team).
		Where("organization_id = ?", appID).
		Where("name = ?", name).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return team, nil
}

// IsTeamMember checks if a member is part of a team
func (r *AppRepository) IsTeamMember(ctx context.Context, teamID, memberID xid.ID) (bool, error) {
	count, err := r.db.NewSelect().
		Model((*schema.TeamMember)(nil)).
		Where("team_id = ?", teamID).
		Where("member_id = ?", memberID).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListMemberTeams lists all teams a member belongs to with pagination
func (r *AppRepository) ListMemberTeams(ctx context.Context, filter *app.ListMemberTeamsFilter) (*pagination.PageResponse[*schema.Team], error) {
	var teams []*schema.Team

	// Build base query
	query := r.db.NewSelect().
		Model(&teams).
		Join("JOIN team_members tm ON tm.team_id = team.id").
		Where("tm.member_id = ?", filter.MemberID)

	// Get total count before pagination
	total, err := r.db.NewSelect().
		Model((*schema.Team)(nil)).
		Join("JOIN team_members tm ON tm.team_id = team.id").
		Where("tm.member_id = ?", filter.MemberID).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination using helper
	qb := pagination.NewQueryBuilder(&filter.PaginationParams)
	query = qb.ApplyToQuery(query)

	// Execute query
	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(teams, int64(total), &filter.PaginationParams), nil
}

// Type assertion to ensure AppRepository implements all focused repository interfaces
var _ app.AppRepository = (*AppRepository)(nil)
var _ app.MemberRepository = (*AppRepository)(nil)
var _ app.TeamRepository = (*AppRepository)(nil)
var _ app.InvitationRepository = (*AppRepository)(nil)
