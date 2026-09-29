package sso

import (
	"context"
	"database/sql"
	"errors"
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

func (s *PostgresStore) CreateConnection(ctx context.Context, c *Connection) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := fromConnection(c)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return ssoPgError(err)
}

func (s *PostgresStore) GetConnection(ctx context.Context, connID id.SSOConnectionID) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", connID.String()).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toConnection(m)
}

func (s *PostgresStore) GetConnectionByDomain(ctx context.Context, appID id.AppID, domain string) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("domain = ?", domain).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toConnection(m)
}

func (s *PostgresStore) GetConnectionByDomainAndOrg(ctx context.Context, appID id.AppID, orgID id.OrgID, domain string) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("org_id = ?", orgID.String()).
		Where("domain = ?", domain).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toConnection(m)
}

func (s *PostgresStore) GetConnectionByProvider(ctx context.Context, appID id.AppID, provider string) (*Connection, error) {
	m := new(ssoConnectionModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("provider = ?", provider).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
	}
	return toConnection(m)
}

func (s *PostgresStore) ListConnections(ctx context.Context, appID id.AppID) ([]*Connection, error) {
	var models []ssoConnectionModel
	err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, ssoPgError(err)
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

func (s *PostgresStore) UpdateConnection(ctx context.Context, c *Connection) error {
	c.UpdatedAt = time.Now()
	m := fromConnection(c)
	_, err := s.pg.NewUpdate(m).
		WherePK().
		Exec(ctx)
	return ssoPgError(err)
}

func (s *PostgresStore) DeleteConnection(ctx context.Context, connID id.SSOConnectionID) error {
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
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConnectionNotFound
	}
	return err
}

func (s *PostgresStore) GetIdentity(ctx context.Context, connID id.SSOConnectionID, subject string) (*Identity, error) {
	if subject == "" {
		return nil, ErrIdentityNotFound
	}
	m := new(ssoIdentityModel)
	err := s.pg.NewSelect(m).
		Where("connection_id = ?", connID.String()).
		Where("subject = ?", subject).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrIdentityNotFound
		}
		return nil, ssoPgError(err)
	}
	return toIdentity(m)
}

func (s *PostgresStore) CreateIdentity(ctx context.Context, ident *Identity) error {
	if ident == nil || ident.Subject == "" {
		return errors.New("sso: identity needs a subject")
	}
	// Round(0) drops the monotonic reading, which the sqlite driver would
	// otherwise write into the column.
	if ident.CreatedAt.IsZero() {
		ident.CreatedAt = time.Now()
	}
	ident.CreatedAt = ident.CreatedAt.UTC().Round(0)
	// Check before insert: the unique index still guards the race, and a
	// duplicate then surfaces as the driver's own error rather than as a
	// second binding.
	if _, err := s.GetIdentity(ctx, ident.ConnectionID, ident.Subject); err == nil {
		return ErrIdentityExists
	} else if !errors.Is(err, ErrIdentityNotFound) {
		return err
	}
	if _, err := s.pg.NewInsert(fromIdentity(ident)).Exec(ctx); err != nil {
		return ssoPgError(err)
	}
	return nil
}
