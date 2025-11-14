package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// organizationInvitationRepository implements organization.InvitationRepository using Bun
type organizationInvitationRepository struct {
	db *bun.DB
}

// NewOrganizationInvitationRepository creates a new organization invitation repository
func NewOrganizationInvitationRepository(db *bun.DB) organization.InvitationRepository {
	return &organizationInvitationRepository{db: db}
}

// Create creates a new invitation
func (r *organizationInvitationRepository) Create(ctx context.Context, invitation *organization.Invitation) error {
	schemaInvitation := invitation.ToSchema()
	_, err := r.db.NewInsert().
		Model(schemaInvitation).
		Exec(ctx)
	return err
}

// FindByID retrieves an invitation by ID
func (r *organizationInvitationRepository) FindByID(ctx context.Context, id xid.ID) (*organization.Invitation, error) {
	schemaInvitation := new(schema.OrganizationInvitation)
	err := r.db.NewSelect().
		Model(schemaInvitation).
		Where("uoi.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaInvitation(schemaInvitation), nil
}

// FindByToken retrieves an invitation by token
func (r *organizationInvitationRepository) FindByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	schemaInvitation := new(schema.OrganizationInvitation)
	err := r.db.NewSelect().
		Model(schemaInvitation).
		Where("token = ?", token).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, err
	}

	return organization.FromSchemaInvitation(schemaInvitation), nil
}

// ListByOrganization lists invitations for an organization with pagination and filtering
func (r *organizationInvitationRepository) ListByOrganization(ctx context.Context, filter *organization.ListInvitationsFilter) (*pagination.PageResponse[*organization.Invitation], error) {
	var schemaInvitations []*schema.OrganizationInvitation

	query := r.db.NewSelect().
		Model(&schemaInvitations).
		Where("organization_id = ?", filter.OrganizationID)

	// Apply status filter if provided
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	query = query.Order("created_at DESC")

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
	invitations := organization.FromSchemaInvitations(schemaInvitations)

	return pagination.NewPageResponse(invitations, int64(total), &filter.PaginationParams), nil
}

// Update updates an invitation
func (r *organizationInvitationRepository) Update(ctx context.Context, invitation *organization.Invitation) error {
	schemaInvitation := invitation.ToSchema()
	_, err := r.db.NewUpdate().
		Model(schemaInvitation).
		WherePK().
		Exec(ctx)
	return err
}

// Delete deletes an invitation
func (r *organizationInvitationRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.OrganizationInvitation)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// DeleteExpired deletes all expired invitations
func (r *organizationInvitationRepository) DeleteExpired(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.OrganizationInvitation)(nil)).
		Where("expires_at < ? AND status = ?", time.Now(), organization.InvitationStatusPending).
		Exec(ctx)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

// Type assertion to ensure organizationInvitationRepository implements organization.InvitationRepository
var _ organization.InvitationRepository = (*organizationInvitationRepository)(nil)
