package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/schema"
)

// SessionRepository is a Bun-backed implementation of core session repository
type SessionRepository struct {
	db *bun.DB
}

func NewSessionRepository(db *bun.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// CreateSession inserts a new session
func (r *SessionRepository) CreateSession(ctx context.Context, s *schema.Session) error {
	_, err := r.db.NewInsert().Model(s).Exec(ctx)
	return err
}

// FindSessionByToken retrieves a session by token
func (r *SessionRepository) FindSessionByToken(ctx context.Context, token string) (*schema.Session, error) {
	ss := new(schema.Session)
	err := r.db.NewSelect().
		Model(ss).
		Where("token = ?", token).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

// FindSessionByID retrieves a session by id
func (r *SessionRepository) FindSessionByID(ctx context.Context, id xid.ID) (*schema.Session, error) {
	ss := new(schema.Session)
	err := r.db.NewSelect().
		Model(ss).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

// ListSessions lists sessions with filtering and pagination
func (r *SessionRepository) ListSessions(ctx context.Context, filter *session.ListSessionsFilter) (*pagination.PageResponse[*schema.Session], error) {
	var sessions []*schema.Session

	// Build base query with filters
	query := r.db.NewSelect().
		Model(&sessions).
		Where("deleted_at IS NULL").
		Where("app_id = ?", filter.AppID)

	if filter.EnvironmentID != nil {
		query = query.Where("environment_id = ?", *filter.EnvironmentID)
	}
	if filter.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filter.OrganizationID)
	}
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Active != nil && *filter.Active {
		query = query.Where("expires_at > ?", time.Now())
	} else if filter.Active != nil && !*filter.Active {
		query = query.Where("expires_at <= ?", time.Now())
	}

	// Count total matching records
	countQuery := r.db.NewSelect().
		Model((*schema.Session)(nil)).
		Where("deleted_at IS NULL").
		Where("app_id = ?", filter.AppID)

	if filter.EnvironmentID != nil {
		countQuery = countQuery.Where("environment_id = ?", *filter.EnvironmentID)
	}
	if filter.OrganizationID != nil {
		countQuery = countQuery.Where("organization_id = ?", *filter.OrganizationID)
	}
	if filter.UserID != nil {
		countQuery = countQuery.Where("user_id = ?", *filter.UserID)
	}
	if filter.Active != nil && *filter.Active {
		countQuery = countQuery.Where("expires_at > ?", time.Now())
	} else if filter.Active != nil && !*filter.Active {
		countQuery = countQuery.Where("expires_at <= ?", time.Now())
	}

	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination and ordering
	query = query.Limit(filter.GetLimit()).
		Offset(filter.GetOffset()).
		Order("created_at DESC")

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return pagination.NewPageResponse(sessions, int64(total), &filter.PaginationParams), nil
}

// RevokeSession deletes a session by token
func (r *SessionRepository) RevokeSession(ctx context.Context, token string) error {
	_, err := r.db.NewDelete().
		Model((*schema.Session)(nil)).
		Where("token = ?", token).
		Exec(ctx)
	return err
}

// RevokeSessionByID deletes a session by id
func (r *SessionRepository) RevokeSessionByID(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.Session)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// UpdateSessionExpiry updates the expiry time of a session (for sliding window renewal)
func (r *SessionRepository) UpdateSessionExpiry(ctx context.Context, id xid.ID, expiresAt time.Time) error {
	_, err := r.db.NewUpdate().
		Model((*schema.Session)(nil)).
		Set("expires_at = ?", expiresAt).
		Set("updated_at = ?", time.Now().UTC()).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// FindSessionByRefreshToken retrieves a session by refresh token
func (r *SessionRepository) FindSessionByRefreshToken(ctx context.Context, refreshToken string) (*schema.Session, error) {
	ss := new(schema.Session)
	err := r.db.NewSelect().
		Model(ss).
		Where("refresh_token = ?", refreshToken).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return ss, nil
}

// RefreshSessionTokens updates both access and refresh tokens for a session
func (r *SessionRepository) RefreshSessionTokens(ctx context.Context, id xid.ID, newAccessToken string, accessTokenExpiresAt time.Time, newRefreshToken string, refreshTokenExpiresAt time.Time) error {
	now := time.Now().UTC()
	_, err := r.db.NewUpdate().
		Model((*schema.Session)(nil)).
		Set("token = ?", newAccessToken).
		Set("expires_at = ?", accessTokenExpiresAt).
		Set("refresh_token = ?", newRefreshToken).
		Set("refresh_token_expires_at = ?", refreshTokenExpiresAt).
		Set("last_refreshed_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)
	return err
}

// CountSessions counts sessions for an app and optionally a user
func (r *SessionRepository) CountSessions(ctx context.Context, appID xid.ID, userID *xid.ID) (int, error) {
	query := r.db.NewSelect().
		Model((*schema.Session)(nil)).
		Where("deleted_at IS NULL").
		Where("app_id = ?", appID)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	return query.Count(ctx)
}

// CleanupExpiredSessions removes expired sessions
func (r *SessionRepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	result, err := r.db.NewDelete().
		Model((*schema.Session)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}
