package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// organizationRepository implements organization.OrganizationRepository using Bun
type organizationRepository struct {
	db *bun.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *bun.DB) organization.OrganizationRepository {
	return &organizationRepository{db: db}
}

// Create creates a new organization
func (r *organizationRepository) Create(ctx context.Context, org *organization.Organization) error {
	schemaOrg := org.ToSchema()
	_, err := r.db.NewInsert().
		Model(schemaOrg).
		Exec(ctx)
	return err
}

// FindByID retrieves an organization by ID
func (r *organizationRepository) FindByID(ctx context.Context, id xid.ID) (*organization.Organization, error) {
	schemaOrg := new(schema.Organization)
	err := r.db.NewSelect().
		Model(schemaOrg).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaOrganization(schemaOrg), nil
}

// FindBySlug retrieves an organization by slug within an app and environment
func (r *organizationRepository) FindBySlug(ctx context.Context, appID, envID xid.ID, slug string) (*organization.Organization, error) {
	schemaOrg := new(schema.Organization)
	err := r.db.NewSelect().
		Model(schemaOrg).
		Where("app_id = ? AND environment_id = ? AND slug = ?", appID, envID, slug).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaOrganization(schemaOrg), nil
}

// ListByApp retrieves a paginated list of organizations within an app and environment
func (r *organizationRepository) ListByApp(ctx context.Context, filter *organization.ListOrganizationsFilter) (*pagination.PageResponse[*organization.Organization], error) {
	var schemaOrgs []*schema.Organization

	query := r.db.NewSelect().
		Model(&schemaOrgs).
		Where("app_id = ? AND environment_id = ?", filter.AppID, filter.EnvironmentID).
		Order("created_at DESC")

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	// Convert to DTOs
	orgs := organization.FromSchemaOrganizations(schemaOrgs)

	return pagination.NewPageResponse(orgs, int64(total), &filter.PaginationParams), nil
}

// ListByUser retrieves a paginated list of organizations a user is a member of
func (r *organizationRepository) ListByUser(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*organization.Organization], error) {
	var schemaOrgs []*schema.Organization

	query := r.db.NewSelect().
		Model(&schemaOrgs).
		Join("INNER JOIN organization_members AS m ON m.organization_id = uo.id").
		Where("m.user_id = ?", userID).
		Order("uo.created_at DESC")

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	// Convert to DTOs
	orgs := organization.FromSchemaOrganizations(schemaOrgs)

	return pagination.NewPageResponse(orgs, int64(total), filter), nil
}

// Update updates an organization
func (r *organizationRepository) Update(ctx context.Context, org *organization.Organization) error {
	schemaOrg := org.ToSchema()
	_, err := r.db.NewUpdate().
		Model(schemaOrg).
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

// CountByUser counts organizations a user is a member of or has created
func (r *organizationRepository) CountByUser(ctx context.Context, userID xid.ID) (int, error) {
	// Count organizations where user is either the creator or a member
	count, err := r.db.NewSelect().
		Model((*schema.Organization)(nil)).
		Join("LEFT JOIN organization_members AS m ON m.organization_id = uo.id").
		Where("uo.created_by = ? OR m.user_id = ?", userID, userID).
		Count(ctx)

	return count, err
}

// Type assertion to ensure organizationRepository implements organization.OrganizationRepository
var _ organization.OrganizationRepository = (*organizationRepository)(nil)
