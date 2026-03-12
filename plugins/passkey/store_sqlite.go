package passkey

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements passkey.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed passkey store.
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

func (s *SqliteStore) CreateCredential(ctx context.Context, c *Credential) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	m := credentialToModel(c)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return passkeySqliteError(err)
}

func (s *SqliteStore) GetCredential(ctx context.Context, credentialID []byte) (*Credential, error) {
	m := new(credentialModel)
	err := s.sdb.NewSelect(m).
		Where("credential_id = ?", credentialID).
		Scan(ctx)
	if err != nil {
		return nil, passkeySqliteError(err)
	}
	return credentialFromModel(m)
}

func (s *SqliteStore) ListUserCredentials(ctx context.Context, userID id.UserID) ([]*Credential, error) {
	var models []credentialModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, passkeySqliteError(err)
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

func (s *SqliteStore) DeleteCredential(ctx context.Context, credentialID []byte) error {
	_, err := s.sdb.NewDelete((*credentialModel)(nil)).
		Where("credential_id = ?", credentialID).
		Exec(ctx)
	return passkeySqliteError(err)
}

func (s *SqliteStore) UpdateSignCount(ctx context.Context, credentialID []byte, count uint32) error {
	now := time.Now()
	_, err := s.sdb.NewUpdate((*credentialModel)(nil)).
		Set("sign_count = ?", int(count)).
		Set("updated_at = ?", now).
		Where("credential_id = ?", credentialID).
		Exec(ctx)
	return passkeySqliteError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func passkeySqliteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCredentialNotFound
	}
	return err
}
