package passkey

import (
	"context"
	"database/sql"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements passkey.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed passkey store.
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

func (s *PostgresStore) CreateCredential(ctx context.Context, c *Credential) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := credentialToModel(c)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return passkeyPgError(err)
}

func (s *PostgresStore) GetCredential(ctx context.Context, credentialID []byte) (*Credential, error) {
	m := new(credentialModel)
	err := s.pg.NewSelect(m).
		Where("credential_id = ?", credentialID).
		Scan(ctx)
	if err != nil {
		return nil, passkeyPgError(err)
	}
	return credentialFromModel(m)
}

func (s *PostgresStore) ListUserCredentials(ctx context.Context, userID id.UserID) ([]*Credential, error) {
	var models []credentialModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, passkeyPgError(err)
	}
	result := make([]*Credential, 0, len(models))
	for i := range models {
		c, err := credentialFromModel(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (s *PostgresStore) DeleteCredential(ctx context.Context, credentialID []byte) error {
	_, err := s.pg.NewDelete((*credentialModel)(nil)).
		Where("credential_id = ?", credentialID).
		Exec(ctx)
	return passkeyPgError(err)
}

func (s *PostgresStore) UpdateSignCount(ctx context.Context, credentialID []byte, count uint32) error {
	now := time.Now()
	_, err := s.pg.NewUpdate((*credentialModel)(nil)).
		Set("sign_count = ?", int(count)).
		Set("updated_at = ?", now).
		Where("credential_id = ?", credentialID).
		Exec(ctx)
	return passkeyPgError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func passkeyPgError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrCredentialNotFound
	}
	return err
}
