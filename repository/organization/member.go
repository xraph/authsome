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

// organizationMemberRepository implements organization.MemberRepository using Bun
type organizationMemberRepository struct {
	db *bun.DB
}

// NewOrganizationMemberRepository creates a new organization member repository
func NewOrganizationMemberRepository(db *bun.DB) organization.MemberRepository {
	return &organizationMemberRepository{db: db}
}

// Create creates a new organization member
func (r *organizationMemberRepository) Create(ctx context.Context, member *organization.Member) error {
	schemaMember := member.ToSchema()
	_, err := r.db.NewInsert().
		Model(schemaMember).
		Exec(ctx)
	return err
}

// FindByID retrieves a member by ID
func (r *organizationMemberRepository) FindByID(ctx context.Context, id xid.ID) (*organization.Member, error) {
	schemaMember := new(schema.OrganizationMember)
	err := r.db.NewSelect().
		Model(schemaMember).
		Where("uom.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaMember(schemaMember), nil
}

// FindByUserAndOrg retrieves a member by user ID and organization ID
func (r *organizationMemberRepository) FindByUserAndOrg(ctx context.Context, userID, orgID xid.ID) (*organization.Member, error) {
	schemaMember := new(schema.OrganizationMember)
	err := r.db.NewSelect().
		Model(schemaMember).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaMember(schemaMember), nil
}

// ListByOrganization lists members of an organization with pagination and filtering
func (r *organizationMemberRepository) ListByOrganization(ctx context.Context, filter *organization.ListMembersFilter) (*pagination.PageResponse[*organization.Member], error) {
	var schemaMembers []*schema.OrganizationMember

	query := r.db.NewSelect().
		Model(&schemaMembers).
		Where("organization_id = ?", filter.OrganizationID)

	// Apply filters
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	query = query.Order("joined_at ASC")

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
	members := organization.FromSchemaMembers(schemaMembers)

	return pagination.NewPageResponse(members, int64(total), &filter.PaginationParams), nil
}

// ListByUser lists all organization memberships for a user with pagination
func (r *organizationMemberRepository) ListByUser(ctx context.Context, userID xid.ID, filter *pagination.PaginationParams) (*pagination.PageResponse[*organization.Member], error) {
	var schemaMembers []*schema.OrganizationMember

	query := r.db.NewSelect().
		Model(&schemaMembers).
		Where("user_id = ?", userID).
		Order("joined_at DESC")

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
	members := organization.FromSchemaMembers(schemaMembers)

	return pagination.NewPageResponse(members, int64(total), filter), nil
}

// Update updates a member
func (r *organizationMemberRepository) Update(ctx context.Context, member *organization.Member) error {
	schemaMember := member.ToSchema()
	_, err := r.db.NewUpdate().
		Model(schemaMember).
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

// Type assertion to ensure organizationMemberRepository implements organization.MemberRepository
var _ organization.MemberRepository = (*organizationMemberRepository)(nil)
