package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
)

// MemberRepository implements organization.MemberRepository
type MemberRepository struct {
	db *bun.DB
}

// NewMemberRepository creates a new member repository
func NewMemberRepository(db *bun.DB) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create creates a new member
func (r *MemberRepository) Create(ctx context.Context, member *organization.Member) error {
	_, err := r.db.NewInsert().Model(member).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create member: %w", err)
	}
	return nil
}

// FindByID finds a member by ID
func (r *MemberRepository) FindByID(ctx context.Context, id xid.ID) (*organization.Member, error) {
	member := &organization.Member{}
	err := r.db.NewSelect().Model(member).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return member, nil
}

// FindByOrgAndUser finds a member by organization and user ID
func (r *MemberRepository) FindByOrgAndUser(ctx context.Context, orgID, userID xid.ID) (*organization.Member, error) {
	member := &organization.Member{}
	err := r.db.NewSelect().Model(member).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return member, nil
}

// Update updates a member
func (r *MemberRepository) Update(ctx context.Context, member *organization.Member) error {
	_, err := r.db.NewUpdate().Model(member).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}
	return nil
}

// Delete deletes a member
func (r *MemberRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*organization.Member)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete member: %w", err)
	}
	return nil
}

// ListByOrganization lists members by organization with pagination
func (r *MemberRepository) ListByOrganization(ctx context.Context, orgID xid.ID, limit, offset int) ([]*organization.Member, error) {
	var members []*organization.Member

	// Get paginated results
	err := r.db.NewSelect().
		Model(&members).
		Where("organization_id = ?", orgID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}

	return members, nil
}

// FindByUserAndOrg finds a member by user ID and organization ID
func (r *MemberRepository) FindByUserAndOrg(ctx context.Context, userID, orgID xid.ID) (*organization.Member, error) {
	var member organization.Member
	err := r.db.NewSelect().
		Model(&member).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find member: %w", err)
	}
	return &member, nil
}

// DeleteByUserID deletes all memberships for a user
func (r *MemberRepository) DeleteByUserID(ctx context.Context, userID xid.ID) error {
	_, err := r.db.NewDelete().Model((*organization.Member)(nil)).Where("user_id = ?", userID).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user memberships: %w", err)
	}
	return nil
}

// CountByOrganization returns the total number of members in an organization
func (r *MemberRepository) CountByOrganization(ctx context.Context, orgID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*organization.Member)(nil)).
		Where("organization_id = ?", orgID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count members by organization: %w", err)
	}
	return count, nil
}

// ListByUser lists organizations a user is a member of
func (r *MemberRepository) ListByUser(ctx context.Context, userID xid.ID) ([]*organization.Member, error) {
	var members []*organization.Member

	// Get all memberships for the user
	err := r.db.NewSelect().
		Model(&members).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list user memberships: %w", err)
	}

	return members, nil
}
