package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/plugins/multitenancy/app"
)

// AppInvitationRepository implements app.InvitationRepository
type AppInvitationRepository struct {
	db *bun.DB
}

// NewAppInvitationRepository creates a new app invitation repository
func NewAppInvitationRepository(db *bun.DB) *AppInvitationRepository {
	return &AppInvitationRepository{db: db}
}

// Create creates a new invitation
func (r *AppInvitationRepository) Create(ctx context.Context, invitation *app.Invitation) error {
	_, err := r.db.NewInsert().Model(invitation).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}
	return nil
}

// FindByID finds an invitation by ID
func (r *AppInvitationRepository) FindByID(ctx context.Context, id xid.ID) (*app.Invitation, error) {
	invitation := &app.Invitation{}
	err := r.db.NewSelect().Model(invitation).Where("id = ?", id).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}
	return invitation, nil
}

// FindByToken finds an invitation by token
func (r *AppInvitationRepository) FindByToken(ctx context.Context, token string) (*app.Invitation, error) {
	invitation := &app.Invitation{}
	err := r.db.NewSelect().Model(invitation).Where("token = ?", token).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("failed to find invitation by token: %w", err)
	}
	return invitation, nil
}

// ListByApp lists invitations by app with pagination
func (r *AppInvitationRepository) ListByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*app.Invitation, error) {
	var invitations []*app.Invitation

	// Get paginated results
	err := r.db.NewSelect().
		Model(&invitations).
		Where("organization_id = ?", appID).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	return invitations, nil
}

// Update updates an invitation
func (r *AppInvitationRepository) Update(ctx context.Context, invitation *app.Invitation) error {
	_, err := r.db.NewUpdate().Model(invitation).WherePK().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}
	return nil
}

// Delete deletes an invitation
func (r *AppInvitationRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*app.Invitation)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}
	return nil
}

// DeleteExpired deletes expired invitations
func (r *AppInvitationRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*app.Invitation)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete expired invitations: %w", err)
	}
	return nil
}
