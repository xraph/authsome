package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// OAuthTokenRepository handles OAuth token persistence.
type OAuthTokenRepository struct {
	db *bun.DB
}

// NewOAuthTokenRepository creates a new OAuth token repository.
func NewOAuthTokenRepository(db *bun.DB) *OAuthTokenRepository {
	return &OAuthTokenRepository{db: db}
}

// Create stores a new OAuth token.
func (r *OAuthTokenRepository) Create(ctx context.Context, token *schema.OAuthToken) error {
	_, err := r.db.NewInsert().Model(token).Exec(ctx)

	return err
}

// FindByAccessToken retrieves a token by its access token value.
func (r *OAuthTokenRepository) FindByAccessToken(ctx context.Context, accessToken string) (*schema.OAuthToken, error) {
	token := &schema.OAuthToken{}

	err := r.db.NewSelect().
		Model(token).
		Where("access_token = ?", accessToken).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return token, nil
}

// FindByRefreshToken retrieves a token by its refresh token value.
func (r *OAuthTokenRepository) FindByRefreshToken(ctx context.Context, refreshToken string) (*schema.OAuthToken, error) {
	token := &schema.OAuthToken{}

	err := r.db.NewSelect().
		Model(token).
		Where("refresh_token = ?", refreshToken).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return token, nil
}

// FindByJTI retrieves a token by its JWT ID.
func (r *OAuthTokenRepository) FindByJTI(ctx context.Context, jti string) (*schema.OAuthToken, error) {
	token := &schema.OAuthToken{}

	err := r.db.NewSelect().
		Model(token).
		Where("jti = ?", jti).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return token, nil
}

// RevokeToken marks a token as revoked.
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

// RevokeByRefreshToken marks a token as revoked by refresh token.
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

// RevokeByJTI marks a token as revoked by JWT ID.
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

// RevokeBySession revokes all tokens associated with a session (cascade revocation).
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

// RevokeAllForUser revokes all tokens for a user in an org.
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

// RevokeAllForClient revokes all tokens for a client.
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

// FindByUserAndClient retrieves tokens for a specific user and client.
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

// FindByUserInOrg retrieves all active tokens for a user in an organization.
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

// DeleteExpired removes expired tokens.
func (r *OAuthTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.OAuthToken)(nil)).
		Where("expires_at < ? AND revoked = ?", time.Now(), true).
		Exec(ctx)

	return err
}

// UpdateRefreshToken updates the refresh token for an existing token.
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

// Update updates an existing OAuth token.
func (r *OAuthTokenRepository) Update(ctx context.Context, token *schema.OAuthToken) error {
	_, err := r.db.NewUpdate().Model(token).WherePK().Exec(ctx)

	return err
}

// RevokeByClientID revokes all tokens for a specific client.
func (r *OAuthTokenRepository) RevokeByClientID(ctx context.Context, clientID string) error {
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

// CountByClientID returns total token count for a client.
func (r *OAuthTokenRepository) CountByClientID(ctx context.Context, clientID string) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("client_id = ?", clientID).
		Count(ctx)

	return int64(count), err
}

// CountActiveByClientID returns active token count for a client.
func (r *OAuthTokenRepository) CountActiveByClientID(ctx context.Context, clientID string) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("client_id = ?", clientID).
		Where("revoked = ?", false).
		Where("expires_at > ?", time.Now()).
		Count(ctx)

	return int64(count), err
}

// CountUniqueUsersByClientID returns count of unique users who have tokens for a client.
func (r *OAuthTokenRepository) CountUniqueUsersByClientID(ctx context.Context, clientID string) (int64, error) {
	var count int64

	err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		ColumnExpr("COUNT(DISTINCT user_id)").
		Where("client_id = ?", clientID).
		Scan(ctx, &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountByClientIDSince returns token count for a client since a specific time.
func (r *OAuthTokenRepository) CountByClientIDSince(ctx context.Context, clientID string, since time.Time) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("client_id = ?", clientID).
		Where("created_at >= ?", since).
		Count(ctx)

	return int64(count), err
}

// CountActiveByApp returns active token count for an app.
func (r *OAuthTokenRepository) CountActiveByApp(ctx context.Context, appID xid.ID) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("app_id = ?", appID).
		Where("revoked = ?", false).
		Where("expires_at > ?", time.Now()).
		Count(ctx)

	return int64(count), err
}

// CountByApp returns total token count for an app.
func (r *OAuthTokenRepository) CountByApp(ctx context.Context, appID xid.ID) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("app_id = ?", appID).
		Count(ctx)

	return int64(count), err
}

// CountByAppSince returns token count for an app since a specific time.
func (r *OAuthTokenRepository) CountByAppSince(ctx context.Context, appID xid.ID, since time.Time) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("app_id = ?", appID).
		Where("created_at >= ?", since).
		Count(ctx)

	return int64(count), err
}

// CountUniqueUsersByApp returns count of unique users who have tokens in an app.
func (r *OAuthTokenRepository) CountUniqueUsersByApp(ctx context.Context, appID xid.ID) (int64, error) {
	var count int64

	err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		ColumnExpr("COUNT(DISTINCT user_id)").
		Where("app_id = ?", appID).
		Scan(ctx, &count)

	return count, err
}

// CountByAppAndType returns token count for an app by token class.
func (r *OAuthTokenRepository) CountByAppAndType(ctx context.Context, appID xid.ID, tokenClass string) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("app_id = ?", appID).
		Where("token_class = ?", tokenClass).
		Count(ctx)

	return int64(count), err
}

// CountByAppBetween returns token count for an app between two times.
func (r *OAuthTokenRepository) CountByAppBetween(ctx context.Context, appID xid.ID, start, end time.Time) (int64, error) {
	count, err := r.db.NewSelect().Model((*schema.OAuthToken)(nil)).
		Where("app_id = ?", appID).
		Where("created_at >= ?", start).
		Where("created_at < ?", end).
		Count(ctx)

	return int64(count), err
}
