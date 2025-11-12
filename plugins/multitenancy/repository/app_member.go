package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/app"
)

// AppMemberRepository implements app.MemberRepository
type AppMemberRepository struct {
	db *bun.DB
}

// NewAppMemberRepository creates a new app member repository
func NewAppMemberRepository(db *bun.DB) *AppMemberRepository {
	return &AppMemberRepository{db: db}
}

// Create creates a new member
func (r *AppMemberRepository) Create(ctx context.Context, member *app.Member) error {
	_, err := r.db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}
	return nil
}

// FindByID finds a member by ID
func (r *AppMemberRepository) FindByID(ctx context.Context, id xid.ID) (*app.Member, error) {
	member := &app.Member{}
	err := r.db.NewSelect().Model(member).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return member, nil
}

// FindByUserAndApp finds a member by user and app ID
func (r *AppMemberRepository) FindByUserAndApp(ctx context.Context, userID, appID xid.ID) (*app.Member, error) {
	var member app.Member
	err := r.db.NewSelect().
		Model(&member).
		Where("user_id = ? AND organization_id = ?", userID, appID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return &member, nil
}

// ListByApp lists members by app with pagination
func (r *AppMemberRepository) ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*app.Member, error) {
	var members []*app.Member

	// Get paginated results
	err := r.db.NewSelect().
		Model(&members).
		Where("organization_id = ?", appID).
		Limit(limit).
		Offset(offset).
		Order("joined_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return members, nil
}

// ListByUser lists apps a user is a member of
func (r *AppMemberRepository) ListByUser(ctx context.Context, userID xid.ID) ([]*app.Member, error) {
	var members []*app.Member

	// Get all memberships for the user
	err := r.db.NewSelect().
		Model(&members).
		Where("user_id = ?", userID).
		Order("joined_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list user memberships: %w", err)
	}

	return members, nil
}

// Update updates a member
func (r *AppMemberRepository) Update(ctx context.Context, member *app.Member) error {
	_, err := r.db.NewUpdate().Model(member).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}
	return nil
}

// Delete deletes a member
func (r *AppMemberRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*app.Member)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all memberships for a user
func (r *AppMemberRepository) DeleteByUserID(ctx context.Context, userID xid.ID) error {
	_, err := r.db.NewDelete().Model((*app.Member)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user memberships: %w", err)
	}
	return nil
}

// CountByApp returns the total number of members in an app
func (r *AppMemberRepository) CountByApp(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*app.Member)(nil)).
		Where("organization_id = ?", appID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count members by app: %w", err)
	}
	return count, nil
}
