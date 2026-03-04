package sso

import (
	"context"
	"database/sql"
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

func (s *SqliteStore) CreateSSOConnection(ctx context.Context, c *SSOConnection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromSSOConnection(c)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return ssoSqliteError(err)
}

func (s *SqliteStore) GetSSOConnection(ctx context.Context, connID id.SSOConnectionID) (*SSOConnection, error) {
	m := new(ssoConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", connID.String()).
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	return toSSOConnection(m)
}

func (s *SqliteStore) GetSSOConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*SSOConnection, error) {
	m := new(ssoConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("domain = ?", domain).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	return toSSOConnection(m)
}

func (s *SqliteStore) GetSSOConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*SSOConnection, error) {
	m := new(ssoConnectionModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("provider = ?", provider).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	return toSSOConnection(m)
}

func (s *SqliteStore) ListSSOConnections(ctx context.Context, appID id.AppID) ([]*SSOConnection, error) {
	var models []ssoConnectionModel
	err := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, ssoSqliteError(err)
	}
	result := make([]*SSOConnection, 0, len(models))
	for i := range models {
		c, err := toSSOConnection(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *SqliteStore) UpdateSSOConnection(ctx context.Context, c *SSOConnection) error {
	c.UpdatedAt = time.Now()
	m := fromSSOConnection(c)
	_, err := s.sdb.NewUpdate(m).
		WherePK().
		Exec(ctx)
	return ssoSqliteError(err)
}

func (s *SqliteStore) DeleteSSOConnection(ctx context.Context, connID id.SSOConnectionID) error {
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
	if err == sql.ErrNoRows {
		return ErrConnectionNotFound
	}
	return err
}
