package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// OAuthTokenRepository handles OAuth token persistence
type OAuthTokenRepository struct {
	db *bun.DB
}

// NewOAuthTokenRepository creates a new OAuth token repository
func NewOAuthTokenRepository(db *bun.DB) *OAuthTokenRepository {
	return &OAuthTokenRepository{db: db}
}

// Create stores a new OAuth token
func (r *OAuthTokenRepository) Create(ctx context.Context, token *schema.OAuthToken) error {
	_, err := r.db.NewInsert().Model(token).Exec(ctx)
	return err
}

// FindByAccessToken retrieves a token by its access token value
func (r *OAuthTokenRepository) FindByAccessToken(ctx context.Context, accessToken string) (*schema.OAuthToken, error) {
	token := &schema.OAuthToken{}
	err := r.db.NewSelect().
		Model(token).
		Where("access_token = ?", accessToken).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

// FindByRefreshToken retrieves a token by its refresh token value
func (r *OAuthTokenRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*schema.OAuthToken, error) {
	token := &schema.OAuthToken{}
	err := r.db.NewSelect().
		Model(token).
		Where("refresh_token = ?", refreshToken).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

// FindByJTI retrieves a token by its JWT ID
func (r *OAuthTokenRepository) FindByJTI(ctx context.Context, jti string) (*schema.OAuthToken, error) {
	token := &schema.OAuthToken{}
	err := r.db.NewSelect().
		Model(token).
		Where("jti = ?", jti).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return token, nil
}

// RevokeToken marks a token as revoked
func (r *OAuthTokenRepository) RevokeToken(ctx context.Context, accessToken string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("access_token = ?", accessToken).
		Exec(ctx)
	return err
}

// RevokeByRefreshToken marks a token as revoked by refresh token
func (r *OAuthTokenRepository) RevokeByRefreshToken(ctx context.Context, refreshToken string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("refresh_token = ?", refreshToken).
		Exec(ctx)
	return err
}

// RevokeByJTI marks a token as revoked by JWT ID
func (r *OAuthTokenRepository) RevokeByJTI(ctx context.Context, jti string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("jti = ?", jti).
		Exec(ctx)
	return err
}

// RevokeBySession revokes all tokens associated with a session (cascade revocation)
func (r *OAuthTokenRepository) RevokeBySession(ctx context.Context, sessionID xid.ID) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("session_id = ?", sessionID).
		Where("revoked = ?", false). // Only revoke non-revoked tokens
		Exec(ctx)
	return err
}

// RevokeAllForUser revokes all tokens for a user in an org
func (r *OAuthTokenRepository) RevokeAllForUser(ctx context.Context, userID xid.ID, appID, envID xid.ID, orgID *xid.ID) error {
	now := time.Now()
	query := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("revoked = ?", false)
	
	if orgID != nil && !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}
	
	_, err := query.Exec(ctx)
	return err
}

// RevokeAllForClient revokes all tokens for a client
func (r *OAuthTokenRepository) RevokeAllForClient(ctx context.Context, clientID string) error {
	now := time.Now()
	_, err := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("revoked = ?", true).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("client_id = ?", clientID).
		Where("revoked = ?", false).
		Exec(ctx)
	return err
}

// FindByUserAndClient retrieves tokens for a specific user and client
func (r *OAuthTokenRepository) FindByUserAndClient(ctx context.Context, userID xid.ID, clientID string) ([]*schema.OAuthToken, error) {
	var tokens []*schema.OAuthToken
	err := r.db.NewSelect().
		Model(&tokens).
		Where("user_id = ? AND client_id = ?", userID, clientID).
		Where("revoked = ?", false).
		Order("created_at DESC").
		Scan(ctx)
	return tokens, err
}

// FindByUserInOrg retrieves all active tokens for a user in an organization
func (r *OAuthTokenRepository) FindByUserInOrg(ctx context.Context, userID xid.ID, appID, envID xid.ID, orgID *xid.ID) ([]*schema.OAuthToken, error) {
	var tokens []*schema.OAuthToken
	query := r.db.NewSelect().
		Model(&tokens).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("revoked = ?", false).
		Order("created_at DESC")
	
	if orgID != nil && !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}
	
	err := query.Scan(ctx)
	return tokens, err
}

// DeleteExpired removes expired tokens
func (r *OAuthTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.OAuthToken)(nil)).
		Where("expires_at < ? AND revoked = ?", time.Now(), true).
		Exec(ctx)
	return err
}

// UpdateRefreshToken updates the refresh token for an existing token
func (r *OAuthTokenRepository) UpdateRefreshToken(ctx context.Context, accessToken, newRefreshToken string, refreshExpiresAt *time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*schema.OAuthToken)(nil)).
		Set("refresh_token = ?", newRefreshToken).
		Set("refresh_expires_at = ?", refreshExpiresAt).
		Set("updated_at = ?", time.Now()).
		Where("access_token = ?", accessToken).
		Exec(ctx)
	return err
}

// Update updates an existing OAuth token
func (r *OAuthTokenRepository) Update(ctx context.Context, token *schema.OAuthToken) error {
	_, err := r.db.NewUpdate().Model(token).WherePK().Exec(ctx)
	return err
}
