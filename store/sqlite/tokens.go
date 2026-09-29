package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/xraph/grove/migrate"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/organization"
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

// Verifications, password resets and invitations follow the session
// contract: token_hash carries store.HashToken of the plaintext, the token
// column stays empty on new rows and holds plaintext only on rows older than
// hash_credential_tokens, which the first lookup rewrites.

// selectByToken loads one row by the hash of the presented plaintext, falling
// back to the legacy plaintext column. It reports whether the plaintext path
// matched so the caller can upgrade the row.
func (s *Store) selectByToken(ctx context.Context, m any, plaintext string) (legacy bool, err error) {
	if plaintext == "" {
		return false, store.ErrNotFound
	}
	err = s.sdb.NewSelect(m).Where("token_hash = ?", store.HashToken(plaintext)).Scan(ctx)
	if err == nil {
		return false, nil
	}
	if !errors.Is(sqliteError(err), store.ErrNotFound) {
		return false, sqliteError(err)
	}
	if plainErr := s.sdb.NewSelect(m).Where("token = ?", plaintext).Scan(ctx); plainErr != nil {
		return false, sqliteError(plainErr)
	}
	return true, nil
}

// upgradeLegacyToken rewrites one plaintext row of model's table as a hash.
// A row another request upgraded first is left alone by the NULL filter.
func (s *Store) upgradeLegacyToken(ctx context.Context, model any, rowID, plaintext string) error {
	_, err := s.sdb.NewUpdate(model).
		Set("token_hash = ?", store.HashToken(plaintext)).
		Set("token = ?", "").
		Where("id = ?", rowID).
		Where("token_hash IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: hash legacy token: %w", sqliteError(err))
	}
	return nil
}

// consumeByToken flips consumed on the row behind plaintext, hashed or
// legacy, and reports whether a row was still unconsumed.
func (s *Store) consumeByToken(ctx context.Context, model any, plaintext string) (bool, error) {
	if plaintext == "" {
		return false, nil
	}
	res, err := s.sdb.NewUpdate(model).
		Set("consumed = ?", true).
		Where("(token_hash = ? OR token = ?)", store.HashToken(plaintext), plaintext).
		Where("consumed = ?", false).
		Exec(ctx)
	if err != nil {
		return false, sqliteError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, sqliteError(err)
	}
	return n > 0, nil
}

// SeedLegacyVerification, SeedLegacyPasswordReset and SeedLegacyInvitation
// write rows the way releases before token hashing did: plaintext and a NULL
// hash. They exist for the conformance suite and are refused outside
// `go test`.
func (s *Store) SeedLegacyVerification(ctx context.Context, v *account.Verification) error {
	if !testing.Testing() {
		return errors.New("authsome/sqlite: SeedLegacyVerification is a test seam")
	}
	m := fromVerification(v)
	m.Token, m.TokenHash = v.Token, sql.NullString{}
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) SeedLegacyPasswordReset(ctx context.Context, pr *account.PasswordReset) error {
	if !testing.Testing() {
		return errors.New("authsome/sqlite: SeedLegacyPasswordReset is a test seam")
	}
	m := fromPasswordReset(pr)
	m.Token, m.TokenHash = pr.Token, sql.NullString{}
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) SeedLegacyInvitation(ctx context.Context, inv *organization.Invitation) error {
	if !testing.Testing() {
		return errors.New("authsome/sqlite: SeedLegacyInvitation is a test seam")
	}
	m := fromInvitation(inv)
	m.Token, m.TokenHash = inv.Token, sql.NullString{}
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return sqliteError(err)
}

func init() {
	Migrations.MustRegister(&migrate.Migration{
		Name:    "hash_credential_tokens",
		Version: "20260922000003",
		Up: func(ctx context.Context, exec migrate.Executor) error {
			// Verification codes may repeat across users (6-digit OTPs), so
			// their hash index is plain; reset and invitation tokens are
			// long and random, so theirs are unique. The plaintext indexes
			// on resets and invitations lose their uniqueness because every
			// new row leaves the column empty.
			for _, stmt := range []string{
				`ALTER TABLE authsome_verifications ADD COLUMN token_hash TEXT`,
				`CREATE INDEX IF NOT EXISTS idx_authsome_verifications_token_hash ON authsome_verifications (token_hash)`,
				`ALTER TABLE authsome_password_resets ADD COLUMN token_hash TEXT`,
				`DROP INDEX IF EXISTS idx_authsome_password_resets_token`,
				`CREATE INDEX IF NOT EXISTS idx_authsome_password_resets_token ON authsome_password_resets (token)`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_password_resets_token_hash ON authsome_password_resets (token_hash)`,
				`ALTER TABLE authsome_invitations ADD COLUMN token_hash TEXT`,
				`DROP INDEX IF EXISTS idx_authsome_invitations_token`,
				`CREATE INDEX IF NOT EXISTS idx_authsome_invitations_token ON authsome_invitations (token)`,
				`CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_invitations_token_hash ON authsome_invitations (token_hash)`,
			} {
				if _, err := exec.Exec(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		},
		Down: func(ctx context.Context, exec migrate.Executor) error {
			for _, stmt := range []string{
				`DROP INDEX IF EXISTS idx_authsome_verifications_token_hash`,
				`DROP INDEX IF EXISTS idx_authsome_password_resets_token_hash`,
				`DROP INDEX IF EXISTS idx_authsome_invitations_token_hash`,
				`ALTER TABLE authsome_verifications DROP COLUMN token_hash`,
				`ALTER TABLE authsome_password_resets DROP COLUMN token_hash`,
				`ALTER TABLE authsome_invitations DROP COLUMN token_hash`,
			} {
				if _, err := exec.Exec(ctx, stmt); err != nil {
					return err
				}
			}
			return nil
		},
	})
}

// HashLegacyTokens implements store.LegacyTokenHasher. Each table is swept
// with a bounded select of rows whose hash is NULL and whose plaintext is
// present, then rewritten row by row: the digest is computed here, not in
// SQL, so the two SQL backends and mongo convert identically. A row with
// neither plaintext nor hash is unusable and left alone.
func (s *Store) HashLegacyTokens(ctx context.Context, batch int) (int64, error) {
	if batch <= 0 {
		batch = 500
	}
	var n int64

	var sessions []SessionModel
	if err := s.sdb.NewSelect(&sessions).
		Where("(token_hash IS NULL AND token <> '') OR (refresh_token_hash IS NULL AND refresh_token <> '')").
		Limit(batch).Scan(ctx); err != nil && !errors.Is(sqliteError(err), store.ErrNotFound) {
		return n, fmt.Errorf("authsome/sqlite: list legacy sessions: %w", sqliteError(err))
	}
	for i := range sessions {
		m := &sessions[i]
		_, err := s.sdb.NewUpdate((*SessionModel)(nil)).
			Set("token_hash = ?", hashOrDerive(m.TokenHash.String, m.Token)).
			Set("refresh_token_hash = ?", hashOrDerive(m.RefreshTokenHash.String, m.RefreshToken)).
			Set("token = ?", "").
			Set("refresh_token = ?", "").
			Where("id = ?", m.ID).
			Exec(ctx)
		if err != nil {
			return n, fmt.Errorf("authsome/sqlite: hash legacy session: %w", sqliteError(err))
		}
		n++
	}

	for _, t := range []struct {
		name  string
		rows  func() ([]legacyRow, error)
		model any
	}{
		{"verification", func() ([]legacyRow, error) {
			var ms []VerificationModel
			err := s.sdb.NewSelect(&ms).Where("token_hash IS NULL").Where("token <> ''").Limit(batch - int(n)).Scan(ctx)
			rows := make([]legacyRow, 0, len(ms))
			for i := range ms {
				rows = append(rows, legacyRow{ms[i].ID, ms[i].Token})
			}
			return rows, err
		}, (*VerificationModel)(nil)},
		{"password reset", func() ([]legacyRow, error) {
			var ms []PasswordResetModel
			err := s.sdb.NewSelect(&ms).Where("token_hash IS NULL").Where("token <> ''").Limit(batch - int(n)).Scan(ctx)
			rows := make([]legacyRow, 0, len(ms))
			for i := range ms {
				rows = append(rows, legacyRow{ms[i].ID, ms[i].Token})
			}
			return rows, err
		}, (*PasswordResetModel)(nil)},
		{"invitation", func() ([]legacyRow, error) {
			var ms []InvitationModel
			err := s.sdb.NewSelect(&ms).Where("token_hash IS NULL").Where("token <> ''").Limit(batch - int(n)).Scan(ctx)
			rows := make([]legacyRow, 0, len(ms))
			for i := range ms {
				rows = append(rows, legacyRow{ms[i].ID, ms[i].Token})
			}
			return rows, err
		}, (*InvitationModel)(nil)},
	} {
		if int(n) >= batch {
			return n, nil
		}
		rows, err := t.rows()
		if err != nil && !errors.Is(sqliteError(err), store.ErrNotFound) {
			return n, fmt.Errorf("authsome/sqlite: list legacy %ss: %w", t.name, sqliteError(err))
		}
		for _, r := range rows {
			if err := s.upgradeLegacyToken(ctx, t.model, r.id, r.token); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// legacyRow is one plaintext credential row awaiting conversion.
type legacyRow struct {
	id    string
	token string
}
