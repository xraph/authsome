package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// InvitationRepository handles invitation data access using schema models.
type InvitationRepository struct {
	db *bun.DB
}

// NewInvitationRepository creates a new invitation repository.
func NewInvitationRepository(db *bun.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// Create creates a new invitation.
func (r *InvitationRepository) Create(ctx context.Context, invitation *schema.Invitation) error {
	_, err := r.db.NewInsert().Model(invitation).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	return nil
}

// FindByID finds an invitation by ID.
func (r *InvitationRepository) FindByID(ctx context.Context, id xid.ID) (*schema.Invitation, error) {
	invitation := &schema.Invitation{}

	err := r.db.NewSelect().Model(invitation).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.NotFound("invitation not found")
		}

		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}

	return invitation, nil
}

// FindByToken finds an invitation by token.
func (r *InvitationRepository) FindByToken(ctx context.Context, token string) (*schema.Invitation, error) {
	invitation := &schema.Invitation{}

	err := r.db.NewSelect().Model(invitation).Where("token = ?", token).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.NotFound("invitation not found")
		}

		return nil, fmt.Errorf("failed to find invitation by token: %w", err)
	}

	return invitation, nil
}

// ListByApp lists invitations by app with pagination.
func (r *InvitationRepository) ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*schema.Invitation, error) {
	var invitations []*schema.Invitation

	// Get paginated results
	err := r.db.NewSelect().
		Model(&invitations).
		Where("app_id = ?", appID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	return invitations, nil
}

// Update updates an invitation.
func (r *InvitationRepository) Update(ctx context.Context, invitation *schema.Invitation) error {
	_, err := r.db.NewUpdate().Model(invitation).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	return nil
}

// Delete deletes an invitation.
func (r *InvitationRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.Invitation)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}

	return nil
}

// DeleteExpired deletes expired invitations.
func (r *InvitationRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.Invitation)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired invitations: %w", err)
	}

	return nil
}
