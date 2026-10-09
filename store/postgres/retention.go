package postgres

import (
	"context"
	"fmt"
	"time"
)

// sweep runs one batched delete. query names the table and its condition
// with $1 as the cutoff and $2 as the row limit; LIMIT NULL is how postgres
// spells "no limit", which is what a batch of zero or less asks for.
func (s *Store) sweep(ctx context.Context, what, query string, before time.Time, batch int) (int64, error) {
	var limit any
	if batch > 0 {
		limit = batch
	}
	res, err := s.pg.NewRaw(query, before, limit).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("authsome/postgres: delete expired %s: %w", what, pgError(err))
	}
	n, _ := res.RowsAffected() //nolint:errcheck // pgx always reports rows affected
	return n, nil
}

// DeleteExpiredSessions implements store.Retention.
func (s *Store) DeleteExpiredSessions(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "sessions", `
DELETE FROM authsome_sessions WHERE id IN (
    SELECT id FROM authsome_sessions
    WHERE expires_at < $1 AND refresh_token_expires_at < $1
    LIMIT $2)`, before, batch)
}

// DeleteExpiredVerifications implements store.Retention.
func (s *Store) DeleteExpiredVerifications(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "verifications", `
DELETE FROM authsome_verifications WHERE id IN (
    SELECT id FROM authsome_verifications WHERE expires_at < $1 LIMIT $2)`, before, batch)
}

// DeleteExpiredPasswordResets implements store.Retention.
func (s *Store) DeleteExpiredPasswordResets(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "password resets", `
DELETE FROM authsome_password_resets WHERE id IN (
    SELECT id FROM authsome_password_resets WHERE expires_at < $1 LIMIT $2)`, before, batch)
}

// DeleteExpiredRevokedRefreshTokens implements store.Retention.
func (s *Store) DeleteExpiredRevokedRefreshTokens(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "revoked refresh tokens", `
DELETE FROM authsome_revoked_refresh_tokens WHERE token_hash IN (
    SELECT token_hash FROM authsome_revoked_refresh_tokens WHERE revoked_at < $1 LIMIT $2)`, before, batch)
}
