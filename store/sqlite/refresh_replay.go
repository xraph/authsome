package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// Refresh-token replay detection — native SQLite implementation.
//
// Backed by `authsome_revoked_refresh_tokens` (migration 20) and
// `authsome_sessions.family_id` (migration 19). Idempotent inserts via
// `INSERT ... ON CONFLICT DO NOTHING`; family revocation runs in a single
// SqliteTx so a partial failure rolls back.

// IsRefreshTokenRevoked reports whether tokenHash has been revoked.
func (s *Store) IsRefreshTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	if tokenHash == "" {
		return false, nil
	}
	m := new(RevokedRefreshTokenModel)
	err := s.sdb.NewSelect(m).Where("token_hash = ?", tokenHash).Scan(ctx)
	if err != nil {
		mapped := sqliteError(err)
		if errors.Is(mapped, store.ErrNotFound) {
			return false, nil
		}
		return false, mapped
	}
	return true, nil
}

// MarkRefreshTokenRevoked records tokenHash as revoked. Idempotent.
func (s *Store) MarkRefreshTokenRevoked(ctx context.Context, tokenHash string, familyID id.SessionFamilyID, reason string) error {
	if tokenHash == "" {
		return nil
	}
	m := &RevokedRefreshTokenModel{
		TokenHash: tokenHash,
		FamilyID:  familyID.String(),
		RevokedAt: time.Now().UTC(),
		Reason:    reason,
	}
	if _, err := s.sdb.NewInsert(m).
		OnConflict("(token_hash) DO NOTHING").
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/sqlite: insert revoked refresh token: %w", sqliteError(err))
	}
	return nil
}

// GetRevokedRefreshTokenFamily returns the family bound to a revoked token,
// or store.ErrNotFound if the hash is unknown.
func (s *Store) GetRevokedRefreshTokenFamily(ctx context.Context, tokenHash string) (id.SessionFamilyID, error) {
	if tokenHash == "" {
		return id.Nil, store.ErrNotFound
	}
	m := new(RevokedRefreshTokenModel)
	err := s.sdb.NewSelect(m).Where("token_hash = ?", tokenHash).Scan(ctx)
	if err != nil {
		return id.Nil, sqliteError(err)
	}
	if m.FamilyID == "" {
		return id.Nil, nil
	}
	famID, err := id.ParseSessionFamilyID(m.FamilyID)
	if err != nil {
		return id.Nil, err
	}
	return famID, nil
}

// RevokeRefreshTokenFamily revokes every active session sharing familyID,
// recording each refresh-token hash as revoked. Runs inside a SqliteTx.
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, familyID id.SessionFamilyID, reason string) error {
	if familyID.IsNil() {
		return nil
	}
	famStr := familyID.String()
	now := time.Now().UTC()

	stx, err := s.sdb.BeginTxQuery(ctx, nil)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: begin tx for refresh-family revoke: %w", err)
	}
	defer func() { _ = stx.Rollback() }() //nolint:errcheck // best-effort rollback after commit

	var sessions []SessionModel
	if err := stx.NewSelect(&sessions).Where("family_id = ?", famStr).Scan(ctx); err != nil {
		return fmt.Errorf("authsome/sqlite: list family sessions: %w", sqliteError(err))
	}

	for _, sess := range sessions {
		if sess.RefreshToken == "" {
			continue
		}
		h := hashRefreshTokenSqlite(sess.RefreshToken)
		rec := &RevokedRefreshTokenModel{
			TokenHash: h,
			FamilyID:  famStr,
			RevokedAt: now,
			Reason:    reason,
		}
		if _, err := stx.NewInsert(rec).
			OnConflict("(token_hash) DO NOTHING").
			Exec(ctx); err != nil {
			return fmt.Errorf("authsome/sqlite: insert revoked sibling: %w", sqliteError(err))
		}
	}

	if _, err := stx.NewDelete((*SessionModel)(nil)).
		Where("family_id = ?", famStr).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/sqlite: delete family sessions: %w", sqliteError(err))
	}

	if err := stx.Commit(); err != nil {
		return fmt.Errorf("authsome/sqlite: commit refresh-family revoke: %w", err)
	}
	return nil
}

// hashRefreshTokenSqlite returns hex(SHA-256(tok)).
func hashRefreshTokenSqlite(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}
