package sso

import (
	"context"
	"database/sql"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements sso.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed SSO connection store.
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

func (s *PostgresStore) CreateSSOConnection(ctx context.Context, c *SSOConnection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromSSOConnection(c)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return ssoPgError(err)
}

func (s *PostgresStore) GetSSOConnection(ctx context.Context, connID id.SSOConnectionID) (*SSOConnection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", connID.String()).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toSSOConnection(m)
}

func (s *PostgresStore) GetSSOConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*SSOConnection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("domain = ?", domain).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toSSOConnection(m)
}

func (s *PostgresStore) GetSSOConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*SSOConnection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("provider = ?", provider).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toSSOConnection(m)
}

func (s *PostgresStore) ListSSOConnections(ctx context.Context, appID id.AppID) ([]*SSOConnection, error) {
	var models []ssoConnectionModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
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

func (s *PostgresStore) UpdateSSOConnection(ctx context.Context, c *SSOConnection) error {
	c.UpdatedAt = time.Now()
	m := fromSSOConnection(c)
	_, err := s.pg.NewUpdate(m).
		WherePK().
		Exec(ctx)
	return ssoPgError(err)
}

func (s *PostgresStore) DeleteSSOConnection(ctx context.Context, connID id.SSOConnectionID) error {
	_, err := s.pg.NewDelete((*ssoConnectionModel)(nil)).
		Where("id = ?", connID.String()).
		Exec(ctx)
	return ssoPgError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func ssoPgError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrConnectionNotFound
	}
	return err
}
