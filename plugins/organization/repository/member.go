package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// organizationMemberRepository implements OrganizationMemberRepository using Bun
type organizationMemberRepository struct {
	db *bun.DB
}

// NewOrganizationMemberRepository creates a new organization member repository
func NewOrganizationMemberRepository(db *bun.DB) *organizationMemberRepository {
	return &organizationMemberRepository{db: db}
}

// Create creates a new organization member
func (r *organizationMemberRepository) Create(ctx context.Context, member *schema.OrganizationMember) error {
	_, err := r.db.NewInsert().
		Model(member).
		Exec(ctx)
	return err
}

// FindByID retrieves a member by ID
func (r *organizationMemberRepository) FindByID(ctx context.Context, id xid.ID) (*schema.OrganizationMember, error) {
	member := new(schema.OrganizationMember)
	err := r.db.NewSelect().
		Model(member).
		Relation("Organization").
		Relation("User").
		Where("uom.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	return member, err
}

// FindByUserAndOrg retrieves a member by user ID and organization ID
func (r *organizationMemberRepository) FindByUserAndOrg(ctx context.Context, userID, orgID xid.ID) (*schema.OrganizationMember, error) {
	member := new(schema.OrganizationMember)
	err := r.db.NewSelect().
		Model(member).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	return member, err
}

// ListByOrganization lists all members of an organization
func (r *organizationMemberRepository) ListByOrganization(ctx context.Context, orgID xid.ID, limit, offset int) ([]*schema.OrganizationMember, error) {
	var members []*schema.OrganizationMember

	query := r.db.NewSelect().
		Model(&members).
		Relation("User").
		Where("organization_id = ?", orgID).
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

// ListByUser lists all organization memberships for a user
func (r *organizationMemberRepository) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*schema.OrganizationMember, error) {
	var members []*schema.OrganizationMember

	query := r.db.NewSelect().
		Model(&members).
		Relation("Organization").
		Where("user_id = ?", userID).
		Order("joined_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return members, err
}

// Update updates a member
func (r *organizationMemberRepository) Update(ctx context.Context, member *schema.OrganizationMember) error {
	_, err := r.db.NewUpdate().
		Model(member).
		WherePK().
		Exec(ctx)
	return err
}

// Delete deletes a member
func (r *organizationMemberRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.OrganizationMember)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// DeleteByUserAndOrg deletes a member by user ID and organization ID
func (r *organizationMemberRepository) DeleteByUserAndOrg(ctx context.Context, userID, orgID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.OrganizationMember)(nil)).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Exec(ctx)
	return err
}

// CountByOrganization counts members in an organization
func (r *organizationMemberRepository) CountByOrganization(ctx context.Context, orgID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.OrganizationMember)(nil)).
		Where("organization_id = ?", orgID).
		Count(ctx)
	return count, err
}

// CountByUser counts organization memberships for a user
func (r *organizationMemberRepository) CountByUser(ctx context.Context, userID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.OrganizationMember)(nil)).
		Where("user_id = ?", userID).
		Count(ctx)
	return count, err
}
