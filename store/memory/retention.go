package memory

import (
	"context"
	"time"
)

// DeleteExpiredSessions implements store.Retention.
func (s *Store) DeleteExpiredSessions(_ context.Context, before time.Time, batch int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, sess := range s.sessions {
		if batch > 0 && n >= int64(batch) {
			break
		}
		if sess.ExpiresAt.Before(before) && sess.RefreshTokenExpiresAt.Before(before) {
			delete(s.sessions, k)
			n++
		}
	}
	return n, nil
}

// DeleteExpiredVerifications implements store.Retention.
func (s *Store) DeleteExpiredVerifications(_ context.Context, before time.Time, batch int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, v := range s.verifications {
		if batch > 0 && n >= int64(batch) {
			break
		}
		if v.ExpiresAt.Before(before) {
			delete(s.verifications, k)
			n++
		}
	}
	return n, nil
}

// DeleteExpiredPasswordResets implements store.Retention.
func (s *Store) DeleteExpiredPasswordResets(_ context.Context, before time.Time, batch int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, pr := range s.passwordResets {
		if batch > 0 && n >= int64(batch) {
			break
		}
		if pr.ExpiresAt.Before(before) {
			delete(s.passwordResets, k)
			n++
		}
	}
	return n, nil
}

// DeleteExpiredRevokedRefreshTokens implements store.Retention.
func (s *Store) DeleteExpiredRevokedRefreshTokens(_ context.Context, before time.Time, batch int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for k, r := range s.revokedRefreshTokens {
		if batch > 0 && n >= int64(batch) {
			break
		}
		if r.RevokedAt.Before(before) {
			delete(s.revokedRefreshTokens, k)
			n++
		}
	}
	return n, nil
}
