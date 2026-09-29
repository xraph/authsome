package store

import (
	"context"
	"time"
)

// Retention is the batched removal of rows whose useful life has ended.
// Expired sessions, spent verification codes and reset tokens, and old
// revocation records are evidence of nothing once their window has closed,
// and keeping them is a liability: a copy of the database should hold as
// little as the service still needs.
//
// Every method deletes rows whose expiry (or revocation, for revoked
// refresh tokens) falls before the given instant, removes at most batch rows
// in one call, and reports how many it removed. A batch of zero or less
// removes every matching row. Callers loop until a call comes back short of
// the batch, which keeps each statement small enough not to hold locks or
// swamp the write-ahead log on a large table.
type Retention interface {
	// DeleteExpiredSessions removes sessions whose access token and refresh
	// token both expired before the given instant. A session whose refresh
	// token is still valid can be refreshed and is kept.
	DeleteExpiredSessions(ctx context.Context, before time.Time, batch int) (int64, error)

	// DeleteExpiredVerifications removes verification codes and links,
	// consumed or not, that expired before the given instant.
	DeleteExpiredVerifications(ctx context.Context, before time.Time, batch int) (int64, error)

	// DeleteExpiredPasswordResets removes password reset tokens, consumed or
	// not, that expired before the given instant.
	DeleteExpiredPasswordResets(ctx context.Context, before time.Time, batch int) (int64, error)

	// DeleteExpiredRevokedRefreshTokens removes replay-detection records
	// revoked before the given instant. Once the refresh token they name has
	// passed its own lifetime the record has nothing left to detect.
	DeleteExpiredRevokedRefreshTokens(ctx context.Context, before time.Time, batch int) (int64, error)
}
