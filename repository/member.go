package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// MemberRepository handles member data access using schema models
type MemberRepository struct {
	db *bun.DB
}

// NewMemberRepository creates a new member repository
func NewMemberRepository(db *bun.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create creates a new member
func (r *MemberRepository) Create(ctx context.Context, member *schema.Member) error {
	_, err := r.db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}
	return nil
}

// FindByID finds a member by ID
func (r *MemberRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Member, error) {
	member := &schema.Member{}
	err := r.db.NewSelect().Model(member).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return member, nil
}

// FindByUserAndApp finds a member by user and app ID
func (r *MemberRepository) FindByUserAndApp(ctx context.Context, userID, appID xid.ID) (*schema.Member, error) {
	var member schema.Member
	err := r.db.NewSelect().
		Model(&member).
		Where("user_id = ? AND app_id = ?", userID, appID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return &member, nil
}

// ListByApp lists members by app with pagination and optional filters
func (r *MemberRepository) ListByApp(ctx context.Context, appID xid.ID, role *schema.MemberRole, status *schema.MemberStatus, limit, offset int) ([]*schema.Member, int64, error) {
	var members []*schema.Member

	// Build query
	query := r.db.NewSelect().Model(&members).Where("app_id = ?", appID)
	if role != nil {
		query = query.Where("role = ?", *role)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// Get total count
	countQuery := r.db.NewSelect().Model((*schema.Member)(nil)).Where("app_id = ?", appID)
	if role != nil {
		countQuery = countQuery.Where("role = ?", *role)
	}
	if status != nil {
		countQuery = countQuery.Where("status = ?", *status)
	}
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count members: %w", err)
	}

	// Get paginated results
	err = query.Limit(limit).Offset(offset).Order("joined_at DESC").Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list members: %w", err)
	}

	return members, int64(total), nil
}

// ListByUser lists apps a user is a member of
func (r *MemberRepository) ListByUser(ctx context.Context, userID xid.ID) ([]*schema.Member, error) {
	var members []*schema.Member

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
func (r *MemberRepository) Update(ctx context.Context, member *schema.Member) error {
	_, err := r.db.NewUpdate().Model(member).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}
	return nil
}

// Delete deletes a member
func (r *MemberRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Member)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}
	return nil
}

// DeleteByUserID deletes all memberships for a user
func (r *MemberRepository) DeleteByUserID(ctx context.Context, userID xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Member)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user memberships: %w", err)
	}
	return nil
}

// CountByApp returns the total number of members in an app
func (r *MemberRepository) CountByApp(ctx context.Context, appID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Member)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count members by app: %w", err)
	}
	return count, nil
}
