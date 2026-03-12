package sso

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements sso.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed SSO connection store.
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

func (s *SqliteStore) CreateConnection(ctx context.Context, c *Connection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromConnection(c)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return ssoSqliteError(err)
}

func (s *SqliteStore) GetConnection(ctx context.Context, connID id.SSOConnectionID) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", connID.String()).
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	return toConnection(m)
}

func (s *SqliteStore) GetConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("domain = ?", domain).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	return toConnection(m)
}

func (s *SqliteStore) GetConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("provider = ?", provider).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	return toConnection(m)
}

func (s *SqliteStore) ListConnections(ctx context.Context, appID id.AppID) ([]*Connection, error) {
	var models []ssoConnectionModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	result := make([]*Connection, 0, len(models))
	for i := range models {
		c, err := toConnection(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *SqliteStore) UpdateConnection(ctx context.Context, c *Connection) error {
	c.UpdatedAt = time.Now()
	m := fromConnection(c)
	_, err := s.sdb.NewUpdate(m).
		WherePK().
		Exec(ctx)
	return ssoSqliteError(err)
}

func (s *SqliteStore) DeleteConnection(ctx context.Context, connID id.SSOConnectionID) error {
	_, err := s.sdb.NewDelete((*ssoConnectionModel)(nil)).
		Where("id = ?", connID.String()).
		Exec(ctx)
	return ssoSqliteError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func ssoSqliteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConnectionNotFound
	}
	return err
}
