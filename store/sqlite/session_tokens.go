package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// Sessions are stored as hashes. token_hash and refresh_token_hash carry
// store.HashToken of the plaintext; the legacy token and refresh_token
// columns stay in the schema, empty on every row written after this release,
// so a lookup can fall back to them for rows written before it and rewrite
// those rows on first use.

// hashOrDerive returns the stored hash for a credential: the hash the caller
// already carries, or the digest of the plaintext when only that is known.
// An empty result becomes NULL so the unique index ignores it.
func hashOrDerive(hash, plaintext string) sql.NullString {
	if hash == "" && plaintext != "" {
		hash = store.HashToken(plaintext)
	}
	return sql.NullString{String: hash, Valid: hash != ""}
}

// findSessionByCredential resolves a session by the hash of the presented
// plaintext, falling back to the legacy plaintext column and upgrading the
// row when that is where the match came from.
func (s *Store) findSessionByCredential(ctx context.Context, hashCol, plainCol, plaintext string) (*SessionModel, error) {
	if plaintext == "" {
		return nil, store.ErrNotFound
	}
	m := new(SessionModel)
	err := s.sdb.NewSelect(m).Where(hashCol+" = ?", store.HashToken(plaintext)).Scan(ctx)
	if err == nil {
		return m, nil
	}
	if !errors.Is(sqliteError(err), store.ErrNotFound) {
		return nil, sqliteError(err)
	}
	m = new(SessionModel)
	if err := s.sdb.NewSelect(m).Where(plainCol+" = ?", plaintext).Scan(ctx); err != nil {
		return nil, sqliteError(err)
	}
	if err := s.upgradeLegacySession(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// upgradeLegacySession rewrites a plaintext row as hashes. A row another
// request upgraded first is left alone: the filter on an empty hash makes the
// write a no-op, and m is updated the same way either way.
func (s *Store) upgradeLegacySession(ctx context.Context, m *SessionModel) error {
	tokenHash := hashOrDerive("", m.Token)
	refreshHash := hashOrDerive("", m.RefreshToken)
	_, err := s.sdb.NewUpdate((*SessionModel)(nil)).
		Set("token_hash = ?", tokenHash).
		Set("refresh_token_hash = ?", refreshHash).
		Set("token = ?", "").
		Set("refresh_token = ?", "").
		Where("id = ?", m.ID).
		Where("token_hash IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: hash legacy session: %w", sqliteError(err))
	}
	m.TokenHash = tokenHash
	m.RefreshTokenHash = refreshHash
	m.Token = ""
	m.RefreshToken = ""
	return nil
}

// SeedLegacySession writes sess exactly as a release before token hashing
// did: plaintext tokens and NULL hashes. It exists so the conformance suite
// can prove the upgrade path and is refused outside `go test`.
func (s *Store) SeedLegacySession(ctx context.Context, sess *session.Session) error {
	if !testing.Testing() {
		return errors.New("authsome/sqlite: SeedLegacySession is a test seam")
	}
	m := fromSession(sess)
	m.Token = sess.Token
	m.RefreshToken = sess.RefreshToken
	m.TokenHash = sql.NullString{}
	m.RefreshTokenHash = sql.NullString{}
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "hash_session_tokens",
		Version: "20260922000002",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			// The plaintext columns lose their uniqueness because every new
			// row leaves them empty; a plain index keeps the legacy fallback
			// lookup off a table scan until the rows are converted. The hash
			// columns are NULL on legacy rows, which the unique index ignores.
			for _, stmt := range []string{
				`ALTER TABLE authsome_sessions ADD COLUMN token_hash TEXT`,
				`ALTER TABLE authsome_sessions ADD COLUMN refresh_token_hash TEXT`,
				`DROP INDEX IF EXISTS idx_authsome_sessions_token`,
				`DROP INDEX IF EXISTS idx_authsome_sessions_refresh_token`,
				`CREATE INDEX IF NOT EXISTS idx_authsome_sessions_token ON authsome_sessions (token)`,
				`CREATE INDEX IF NOT EXISTS idx_authsome_sessions_refresh_token ON authsome_sessions (refresh_token)`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sessions_token_hash ON authsome_sessions (token_hash)`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_sessions_refresh_token_hash ON authsome_sessions (refresh_token_hash)`,
			} {
				if _, err := exec.Exec(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			for _, stmt := range []string{
				`DROP INDEX IF EXISTS idx_authsome_sessions_token_hash`,
				`DROP INDEX IF EXISTS idx_authsome_sessions_refresh_token_hash`,
				`ALTER TABLE authsome_sessions DROP COLUMN token_hash`,
				`ALTER TABLE authsome_sessions DROP COLUMN refresh_token_hash`,
			} {
				if _, err := exec.Exec(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		},
	})
}
