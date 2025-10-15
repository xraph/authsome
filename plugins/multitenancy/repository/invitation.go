package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/organization"
)

// InvitationRepository implements organization.InvitationRepository
type InvitationRepository struct {
	db *bun.DB
}

// NewInvitationRepository creates a new invitation repository
func NewInvitationRepository(db *bun.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create creates a new invitation
func (r *InvitationRepository) Create(ctx context.Context, invitation *organization.Invitation) error {
	_, err := r.db.NewInsert().Model(invitation).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}
	return nil
}

// FindByID finds an invitation by ID
func (r *InvitationRepository) FindByID(ctx context.Context, id string) (*organization.Invitation, error) {
	invitation := &organization.Invitation{}
	err := r.db.NewSelect().Model(invitation).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}
	return invitation, nil
}

// FindByToken finds an invitation by token
func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*organization.Invitation, error) {
	invitation := &organization.Invitation{}
	err := r.db.NewSelect().Model(invitation).Where("token = ?", token).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, organization.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("failed to find invitation by token: %w", err)
	}
	return invitation, nil
}

// Update updates an invitation
func (r *InvitationRepository) Update(ctx context.Context, invitation *organization.Invitation) error {
	_, err := r.db.NewUpdate().Model(invitation).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}
	return nil
}

// Delete deletes an invitation
func (r *InvitationRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().Model((*organization.Invitation)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}
	return nil
}

// ListByOrganization lists invitations by organization with pagination
func (r *InvitationRepository) ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*organization.Invitation, error) {
	var invitations []*organization.Invitation
	
	// Get paginated results
	err := r.db.NewSelect().
		Model(&invitations).
		Where("organization_id = ?", orgID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}
	
	return invitations, nil
}

// ListByEmail lists invitations by email with pagination
func (r *InvitationRepository) ListByEmail(ctx context.Context, email string, limit, offset int) ([]*organization.Invitation, int, error) {
	var invitations []*organization.Invitation
	
	// Get total count
	count, err := r.db.NewSelect().
		Model((*organization.Invitation)(nil)).
		Where("email = ?", email).
		Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count invitations by email: %w", err)
	}
	
	// Get paginated results
	err = r.db.NewSelect().
		Model(&invitations).
		Where("email = ?", email).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list invitations by email: %w", err)
	}
	
	return invitations, count, nil
}

// DeleteExpired deletes expired invitations
func (r *InvitationRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*organization.Invitation)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired invitations: %w", err)
	}
	return nil
}