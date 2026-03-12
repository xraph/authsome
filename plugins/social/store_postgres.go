package social

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements social.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed social/OAuth store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
		pg: pgdriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*PostgresStore)(nil)

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *PostgresStore) CreateOAuthConnection(ctx context.Context, c *OAuthConnection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromOAuthConnection(c)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return socialPgError(err)
}

func (s *PostgresStore) GetOAuthConnection(ctx context.Context, provider, providerUserID string) (*OAuthConnection, error) {
	m := new(oauthConnectionModel)
	err := s.pg.NewSelect(m).
		Where("provider = ?", provider).
		Where("provider_user_id = ?", providerUserID).
		Scan(ctx)
	if err != nil {
		return nil, socialPgError(err)
	}
	return toOAuthConnection(m)
}

func (s *PostgresStore) GetOAuthConnectionsByUserID(ctx context.Context, userID id.UserID) ([]*OAuthConnection, error) {
	var models []oauthConnectionModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, socialPgError(err)
	}
	result := make([]*OAuthConnection, 0, len(models))
	for i := range models {
		c, err := toOAuthConnection(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *PostgresStore) DeleteOAuthConnection(ctx context.Context, connID id.OAuthConnectionID) error {
	_, err := s.pg.NewDelete((*oauthConnectionModel)(nil)).
		Where("id = ?", connID.String()).
		Exec(ctx)
	return socialPgError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func socialPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConnectionNotFound
	}
	return err
}
