package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// sweep runs one batched delete. query names the table and its condition,
// binding the cutoff once per comparison and the row limit last; a negative
// LIMIT is how sqlite spells "no limit", which is what a batch of zero or
// less asks for. The cutoff is bound in UTC because timestamps are TEXT in
// this schema and compare as strings, and it goes on the left of each
// comparison so the column is never coerced.
func (s *Store) sweep(ctx context.Context, what, query string, before time.Time, batch int) (int64, error) {
	limit := -1
	if batch > 0 {
		limit = batch
	}
	args := make([]any, 0, 3)
	for range strings.Count(query, "?") - 1 {
		args = append(args, before.UTC())
	}
	args = append(args, limit)
	res, err := s.sdb.NewRaw(query, args...).Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("authsome/sqlite: delete expired %s: %w", what, sqliteError(err))
	}
	n, _ := res.RowsAffected() //nolint:errcheck // sqlite always reports rows affected
	return n, nil
}

// DeleteExpiredSessions implements store.Retention.
func (s *Store) DeleteExpiredSessions(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "sessions", `
DELETE FROM authsome_sessions WHERE id IN (
    SELECT id FROM authsome_sessions
    WHERE ? > expires_at AND ? > refresh_token_expires_at
    LIMIT ?)`, before, batch)
}

// DeleteExpiredVerifications implements store.Retention.
func (s *Store) DeleteExpiredVerifications(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "verifications", `
DELETE FROM authsome_verifications WHERE id IN (
    SELECT id FROM authsome_verifications WHERE ? > expires_at LIMIT ?)`, before, batch)
}

// DeleteExpiredPasswordResets implements store.Retention.
func (s *Store) DeleteExpiredPasswordResets(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "password resets", `
DELETE FROM authsome_password_resets WHERE id IN (
    SELECT id FROM authsome_password_resets WHERE ? > expires_at LIMIT ?)`, before, batch)
}

// DeleteExpiredRevokedRefreshTokens implements store.Retention.
func (s *Store) DeleteExpiredRevokedRefreshTokens(ctx context.Context, before time.Time, batch int) (int64, error) {
	return s.sweep(ctx, "revoked refresh tokens", `
DELETE FROM authsome_revoked_refresh_tokens WHERE token_hash IN (
    SELECT token_hash FROM authsome_revoked_refresh_tokens WHERE ? > revoked_at LIMIT ?)`, before, batch)
}
