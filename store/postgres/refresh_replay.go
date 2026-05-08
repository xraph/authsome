package postgres

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

// Refresh-token replay detection — native PostgreSQL implementation.
//
// Backed by the `authsome_revoked_refresh_tokens` table (migration 20) and
// the `authsome_sessions.family_id` column (migration 19). Insert idempotency
// is enforced via ON CONFLICT DO NOTHING; family revocation runs in a single
// PgTx so a partial failure doesn't leave the family half-revoked.

// IsRefreshTokenRevoked reports whether tokenHash has been recorded as
// revoked. A miss returns (false, nil); other DB errors propagate.
func (s *Store) IsRefreshTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	if tokenHash == "" {
		return false, nil
	}
	m := new(RevokedRefreshTokenModel)
	err := s.pg.NewSelect(m).Where("token_hash = ?", tokenHash).Scan(ctx)
	if err != nil {
		mapped := pgError(err)
		if errors.Is(mapped, store.ErrNotFound) {
			return false, nil
		}
		return false, mapped
	}
	return true, nil
}

// MarkRefreshTokenRevoked records tokenHash as revoked. Idempotent: a
// duplicate insert silently succeeds (the original record wins).
func (s *Store) MarkRefreshTokenRevoked(ctx context.Context, tokenHash string, familyID id.SessionFamilyID, reason string) error {
	if tokenHash == "" {
		return nil
	}
	revokedAt := time.Now().UTC()
	m := &RevokedRefreshTokenModel{
		TokenHash: tokenHash,
		FamilyID:  familyID.String(),
		RevokedAt: revokedAt,
		Reason:    reason,
	}
	_, err := s.pg.NewInsert(m).
		OnConflict("(token_hash) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/postgres: insert revoked refresh token: %w", pgError(err))
	}
	return nil
}

// GetRevokedRefreshTokenFamily returns the family bound to a previously-
// revoked token, or store.ErrNotFound if the hash is unknown.
func (s *Store) GetRevokedRefreshTokenFamily(ctx context.Context, tokenHash string) (id.SessionFamilyID, error) {
	if tokenHash == "" {
		return id.Nil, store.ErrNotFound
	}
	m := new(RevokedRefreshTokenModel)
	err := s.pg.NewSelect(m).Where("token_hash = ?", tokenHash).Scan(ctx)
	if err != nil {
		return id.Nil, pgError(err)
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

// RevokeRefreshTokenFamily revokes every active session sharing familyID.
// Each surviving refresh-token hash is also recorded as revoked with reason
// so later replays of any sibling token are detected. The whole cascade
// runs inside a single PgTx — on any error it rolls back.
func (s *Store) RevokeRefreshTokenFamily(ctx context.Context, familyID id.SessionFamilyID, reason string) error {
	if familyID.IsNil() {
		return nil
	}
	famStr := familyID.String()
	now := time.Now().UTC()

	ptx, err := s.pg.BeginTxQuery(ctx, nil)
	if err != nil {
		return fmt.Errorf("authsome/postgres: begin tx for refresh-family revoke: %w", err)
	}
	defer func() { _ = ptx.Rollback() }() //nolint:errcheck // best-effort rollback after commit

	var sessions []SessionModel
	if err := ptx.NewSelect(&sessions).Where("family_id = ?", famStr).Scan(ctx); err != nil {
		return fmt.Errorf("authsome/postgres: list family sessions: %w", pgError(err))
	}

	for _, sess := range sessions {
		if sess.RefreshToken == "" {
			continue
		}
		h := hashRefreshTokenPg(sess.RefreshToken)
		rec := &RevokedRefreshTokenModel{
			TokenHash: h,
			FamilyID:  famStr,
			RevokedAt: now,
			Reason:    reason,
		}
		if _, err := ptx.NewInsert(rec).
			OnConflict("(token_hash) DO NOTHING").
			Exec(ctx); err != nil {
			return fmt.Errorf("authsome/postgres: insert revoked sibling: %w", pgError(err))
		}
	}

	if _, err := ptx.NewDelete((*SessionModel)(nil)).
		Where("family_id = ?", famStr).
		Exec(ctx); err != nil {
		return fmt.Errorf("authsome/postgres: delete family sessions: %w", pgError(err))
	}

	if err := ptx.Commit(); err != nil {
		return fmt.Errorf("authsome/postgres: commit refresh-family revoke: %w", err)
	}
	return nil
}

// hashRefreshTokenPg returns the hex-encoded SHA-256 of a refresh token.
// Mirrors the canonicalisation used by the in-memory store and engine.
func hashRefreshTokenPg(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

// Compile-time assertion that the Store implements the session.Store
// replay-detection contract.
var _ interface {
	IsRefreshTokenRevoked(context.Context, string) (bool, error)
	MarkRefreshTokenRevoked(context.Context, string, id.SessionFamilyID, string) error
	GetRevokedRefreshTokenFamily(context.Context, string) (id.SessionFamilyID, error)
	RevokeRefreshTokenFamily(context.Context, id.SessionFamilyID, string) error
} = (*Store)(nil)
