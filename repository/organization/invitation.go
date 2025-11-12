package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// organizationInvitationRepository implements OrganizationInvitationRepository using Bun
type organizationInvitationRepository struct {
	db *bun.DB
}

// NewOrganizationInvitationRepository creates a new organization invitation repository
func NewOrganizationInvitationRepository(db *bun.DB) *organizationInvitationRepository {
	return &organizationInvitationRepository{db: db}
}

// Create creates a new invitation
func (r *organizationInvitationRepository) Create(ctx context.Context, invitation *schema.OrganizationInvitation) error {
	_, err := r.db.NewInsert().
		Model(invitation).
		Exec(ctx)
	return err
}

// FindByID retrieves an invitation by ID
func (r *organizationInvitationRepository) FindByID(ctx context.Context, id xid.ID) (*schema.OrganizationInvitation, error) {
	invitation := new(schema.OrganizationInvitation)
	err := r.db.NewSelect().
		Model(invitation).
		Relation("Organization").
		Where("uoi.id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	return invitation, err
}

// FindByToken retrieves an invitation by token
func (r *organizationInvitationRepository) FindByToken(ctx context.Context, token string) (*schema.OrganizationInvitation, error) {
	invitation := new(schema.OrganizationInvitation)
	err := r.db.NewSelect().
		Model(invitation).
		Relation("Organization").
		Where("token = ?", token).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	return invitation, err
}

// FindByEmail retrieves an invitation by email and organization
func (r *organizationInvitationRepository) FindByEmail(ctx context.Context, orgID xid.ID, email string) (*schema.OrganizationInvitation, error) {
	invitation := new(schema.OrganizationInvitation)
	err := r.db.NewSelect().
		Model(invitation).
		Where("organization_id = ? AND email = ? AND status = ?", orgID, email, "pending").
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	return invitation, err
}

// ListByOrganization lists all invitations for an organization
func (r *organizationInvitationRepository) ListByOrganization(ctx context.Context, orgID xid.ID, status string, limit, offset int) ([]*schema.OrganizationInvitation, error) {
	var invitations []*schema.OrganizationInvitation
	
	query := r.db.NewSelect().
		Model(&invitations).
		Where("organization_id = ?", orgID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query = query.Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return invitations, err
}

// ListByEmail lists all invitations for an email address
func (r *organizationInvitationRepository) ListByEmail(ctx context.Context, email string, status string, limit, offset int) ([]*schema.OrganizationInvitation, error) {
	var invitations []*schema.OrganizationInvitation
	
	query := r.db.NewSelect().
		Model(&invitations).
		Relation("Organization").
		Where("email = ?", email)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query = query.Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Scan(ctx)
	return invitations, err
}

// Update updates an invitation
func (r *organizationInvitationRepository) Update(ctx context.Context, invitation *schema.OrganizationInvitation) error {
	_, err := r.db.NewUpdate().
		Model(invitation).
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
		Where("expires_at < ? AND status = ?", time.Now(), "pending").
		Exec(ctx)
	
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}

// CountByOrganization counts invitations for an organization
func (r *organizationInvitationRepository) CountByOrganization(ctx context.Context, orgID xid.ID, status string) (int, error) {
	query := r.db.NewSelect().
		Model((*schema.OrganizationInvitation)(nil)).
		Where("organization_id = ?", orgID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	count, err := query.Count(ctx)
	return count, err
}

// CountByEmail counts invitations for an email address
func (r *organizationInvitationRepository) CountByEmail(ctx context.Context, email string, status string) (int, error) {
	query := r.db.NewSelect().
		Model((*schema.OrganizationInvitation)(nil)).
		Where("email = ?", email)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	count, err := query.Count(ctx)
	return count, err
}

