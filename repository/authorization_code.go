package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// AuthorizationCodeRepository handles authorization code persistence
type AuthorizationCodeRepository struct {
	db *bun.DB
}

// NewAuthorizationCodeRepository creates a new authorization code repository
func NewAuthorizationCodeRepository(db *bun.DB) *AuthorizationCodeRepository {
	return &AuthorizationCodeRepository{db: db}
}

// Create stores a new authorization code
func (r *AuthorizationCodeRepository) Create(ctx context.Context, code *schema.AuthorizationCode) error {
	_, err := r.db.NewInsert().Model(code).Exec(ctx)
	return err
}

// FindByCode retrieves an authorization code by its code value
func (r *AuthorizationCodeRepository) FindByCode(ctx context.Context, code string) (*schema.AuthorizationCode, error) {
	authCode := &schema.AuthorizationCode{}
	err := r.db.NewSelect().
		Model(authCode).
		Where("code = ?", code).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return authCode, nil
}

// FindByCodeWithContext retrieves an authorization code with context filtering
func (r *AuthorizationCodeRepository) FindByCodeWithContext(ctx context.Context, code string, appID, envID xid.ID, orgID *xid.ID) (*schema.AuthorizationCode, error) {
	authCode := &schema.AuthorizationCode{}
	query := r.db.NewSelect().Model(authCode).
		Where("code = ?", code).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)
	
	if orgID != nil && !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}
	
	err := query.Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return authCode, nil
}

// MarkAsUsed marks an authorization code as used
func (r *AuthorizationCodeRepository) MarkAsUsed(ctx context.Context, code string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.AuthorizationCode)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Set("updated_at = ?", now).
		Where("code = ?", code).
		Exec(ctx)
	return err
}

// DeleteExpired removes expired authorization codes
func (r *AuthorizationCodeRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.AuthorizationCode)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	return err
}

// FindByUserAndClient retrieves authorization codes for a specific user and client
func (r *AuthorizationCodeRepository) FindByUserAndClient(ctx context.Context, userID xid.ID, clientID string) ([]*schema.AuthorizationCode, error) {
	var codes []*schema.AuthorizationCode
	err := r.db.NewSelect().
		Model(&codes).
		Where("user_id = ? AND client_id = ?", userID, clientID).
		Order("created_at DESC").
		Scan(ctx)
	return codes, err
}

// FindBySession retrieves authorization codes for a specific session
func (r *AuthorizationCodeRepository) FindBySession(ctx context.Context, sessionID xid.ID) ([]*schema.AuthorizationCode, error) {
	var codes []*schema.AuthorizationCode
	err := r.db.NewSelect().
		Model(&codes).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Scan(ctx)
	return codes, err
}

// DeleteBySession removes authorization codes associated with a session
func (r *AuthorizationCodeRepository) DeleteBySession(ctx context.Context, sessionID xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.AuthorizationCode)(nil)).
		Where("session_id = ?", sessionID).
		Exec(ctx)
	return err
}
