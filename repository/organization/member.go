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

// memberWithUser holds a member joined with user data
type memberWithUser struct {
	schema.OrganizationMember
	UserID          xid.ID `bun:"user_id"`
	UserName        string `bun:"user_name"`
	UserEmail       string `bun:"user_email"`
	UserImage       string `bun:"user_image"`
	UserUsername    string `bun:"user_username"`
	UserDisplayName string `bun:"user_display_username"`
}

// toMemberWithUserInfo converts memberWithUser to organization.Member with UserInfo populated
func (m *memberWithUser) toMemberWithUserInfo() *organization.Member {
	member := organization.FromSchemaMember(&m.OrganizationMember)
	if member != nil {
		member.UserID = m.UserID
		member.OrganizationID = m.OrganizationID
		member.Role = m.Role
		member.Status = m.Status
		member.JoinedAt = m.JoinedAt
		member.CreatedAt = m.CreatedAt
		member.UpdatedAt = m.UpdatedAt
		member.DeletedAt = m.DeletedAt
		

		member.User = &organization.UserInfo{
			ID:              m.UserID,
			Name:            m.UserName,
			Email:           m.UserEmail,
			Image:           m.UserImage,
			Username:        m.UserUsername,
			DisplayUsername: m.UserDisplayName,
		}
	}
	return member
}

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
	var membersWithUsers []*memberWithUser

	query := r.db.NewSelect().
		Model((*schema.OrganizationMember)(nil)).
		ColumnExpr("uom.*").
		ColumnExpr("u.id AS user_id").
		ColumnExpr("u.name AS user_name").
		ColumnExpr("u.email AS user_email").
		ColumnExpr("u.image AS user_image").
		ColumnExpr("u.username AS user_username").
		ColumnExpr("u.display_username AS user_display_username").
		Join("LEFT JOIN users AS u ON u.id = uom.user_id").
		Where("uom.organization_id = ?", filter.OrganizationID)

	// Apply filters
	if filter.Role != nil {
		query = query.Where("uom.role = ?", *filter.Role)
	}
	if filter.Status != nil {
		query = query.Where("uom.status = ?", *filter.Status)
	}

	query = query.Order("uom.joined_at ASC")

	// Get total count (need a separate query without joins for accurate count)
	countQuery := r.db.NewSelect().
		Model((*schema.OrganizationMember)(nil)).
		Where("organization_id = ?", filter.OrganizationID)
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

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	if err := query.Scan(ctx, &membersWithUsers); err != nil {
		return nil, err
	}

	// Convert to DTOs with user info
	members := make([]*organization.Member, len(membersWithUsers))
	for i, m := range membersWithUsers {
		members[i] = m.toMemberWithUserInfo()
	}

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
