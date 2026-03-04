package social

import (
	"context"
	"database/sql"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements social.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed social/OAuth store.
func NewSqliteStore(db *grove.DB) *SqliteStore {
	return &SqliteStore{
		db:  db,
		sdb: sqlitedriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*SqliteStore)(nil)

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateOAuthConnection(ctx context.Context, c *OAuthConnection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromOAuthConnection(c)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return socialSqliteError(err)
}

func (s *SqliteStore) GetOAuthConnection(ctx context.Context, provider, providerUserID string) (*OAuthConnection, error) {
	m := new(oauthConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("provider = ?", provider).
		Where("provider_user_id = ?", providerUserID).
		Scan(ctx)
	if err != nil {
		return nil, socialSqliteError(err)
	}
	return toOAuthConnection(m)
}

func (s *SqliteStore) GetOAuthConnectionsByUserID(ctx context.Context, userID id.UserID) ([]*OAuthConnection, error) {
	var models []oauthConnectionModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, socialSqliteError(err)
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

func (s *SqliteStore) DeleteOAuthConnection(ctx context.Context, connID id.OAuthConnectionID) error {
	_, err := s.sdb.NewDelete((*oauthConnectionModel)(nil)).
		Where("id = ?", connID.String()).
		Exec(ctx)
	return socialSqliteError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func socialSqliteError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrConnectionNotFound
	}
	return err
}
