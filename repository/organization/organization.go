package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// organizationRepository implements OrganizationRepository using Bun
type organizationRepository struct {
	db *bun.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *bun.DB) *organizationRepository {
	return &organizationRepository{db: db}
}

// Create creates a new organization
func (r *organizationRepository) Create(ctx context.Context, org *schema.Organization) error {
	_, err := r.db.NewInsert().
		Model(org).
		Exec(ctx)
	return err
}

// FindByID retrieves an organization by ID
func (r *organizationRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Organization, error) {
	org := new(schema.Organization)
	err := r.db.NewSelect().
		Model(org).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	return org, err
}

// FindBySlug retrieves an organization by slug within an app and environment
func (r *organizationRepository) FindBySlug(ctx context.Context, appID, environmentID xid.ID, slug string) (*schema.Organization, error) {
	org := new(schema.Organization)
	err := r.db.NewSelect().
		Model(org).
		Where("app_id = ? AND environment_id = ? AND slug = ?", appID, environmentID, slug).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	return org, err
}

// ListByUser lists all organizations a user is a member of
func (r *organizationRepository) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*schema.Organization, error) {
	var orgs []*schema.Organization

	query := r.db.NewSelect().
		Model(&orgs).
		Join("INNER JOIN organization_members AS m ON m.organization_id = uo.id").
		Where("m.user_id = ?", userID).
		Order("uo.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return orgs, err
}

// ListByApp lists all organizations within an app and environment
func (r *organizationRepository) ListByApp(ctx context.Context, appID, environmentID xid.ID, limit, offset int) ([]*schema.Organization, error) {
	var orgs []*schema.Organization

	query := r.db.NewSelect().
		Model(&orgs).
		Where("app_id = ? AND environment_id = ?", appID, environmentID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return orgs, err
}

// Update updates an organization
func (r *organizationRepository) Update(ctx context.Context, org *schema.Organization) error {
	_, err := r.db.NewUpdate().
		Model(org).
		WherePK().
		Exec(ctx)
	return err
}

// Delete deletes an organization
func (r *organizationRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Organization)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// CountByUser counts organizations a user is a member of
func (r *organizationRepository) CountByUser(ctx context.Context, userID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Organization)(nil)).
		Join("INNER JOIN organization_members AS m ON m.organization_id = organizations.id").
		Where("m.user_id = ?", userID).
		Count(ctx)
	return count, err
}

// CountByApp counts organizations within an app and environment
func (r *organizationRepository) CountByApp(ctx context.Context, appID, environmentID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.Organization)(nil)).
		Where("app_id = ? AND environment_id = ?", appID, environmentID).
		Count(ctx)
	return count, err
}
