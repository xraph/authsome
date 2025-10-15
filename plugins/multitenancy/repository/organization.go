package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
)

// OrganizationRepository implements organization.OrganizationRepository
type OrganizationRepository struct {
	db *bun.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *bun.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization
func (r *OrganizationRepository) Create(ctx context.Context, org *organization.Organization) error {
	_, err := r.db.NewInsert().Model(org).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}
	return nil
}

// FindByID finds an organization by ID
func (r *OrganizationRepository) FindByID(ctx context.Context, id string) (*organization.Organization, error) {
	org := &organization.Organization{}
	err := r.db.NewSelect().Model(org).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}
	return org, nil
}

// FindBySlug finds an organization by slug
func (r *OrganizationRepository) FindBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	org := &organization.Organization{}
	err := r.db.NewSelect().Model(org).Where("slug = ?", slug).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to find organization by slug: %w", err)
	}
	return org, nil
}

// Update updates an organization
func (r *OrganizationRepository) Update(ctx context.Context, org *organization.Organization) error {
	_, err := r.db.NewUpdate().Model(org).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}
	return nil
}

// Delete deletes an organization
func (r *OrganizationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*organization.Organization)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

// List lists organizations with pagination
func (r *OrganizationRepository) List(ctx context.Context, limit, offset int) ([]*organization.Organization, error) {
	var orgs []*organization.Organization
	
	// Get paginated results
	err := r.db.NewSelect().
		Model(&orgs).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	
	return orgs, nil
}

// Count returns the total number of organizations
func (r *OrganizationRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*organization.Organization)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count organizations: %w", err)
	}
	return count, nil
}