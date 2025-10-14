package repository

import (
	"context"
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