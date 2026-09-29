package store

import (
	"context"
	"time"
)

// Retention kinds: the names the sweeper and plugins use for each class of
// row when asking for its cutoff.
const (
	RetentionSessions             = "sessions"
	RetentionVerifications        = "verifications"
	RetentionPasswordResets       = "password_resets"
	RetentionRevokedRefreshTokens = "revoked_refresh_tokens"
	RetentionDeviceCodes          = "device_codes"
	RetentionAuthCodes            = "auth_codes"
)

// TTLIndexer is offered by a store whose database can expire rows on its
// own. The sweeper calls it once at start with the retention window for each
// kind, so the database drops what the sweeper would otherwise have to.
type TTLIndexer interface {
	// EnsureTTLIndexes creates or updates the expiry index for each kind in
	// ttl. A kind missing from the map, or mapped to zero or less, gets no
	// index (and an existing one is left alone).
	EnsureTTLIndexes(ctx context.Context, ttl map[string]time.Duration) error
}

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

// DrainExpired calls del with the batch size until a call removes fewer
// rows than the batch, or errors, and returns the total removed. A batch of
// zero or less means one unbounded call. This is the loop every sweeper
// runs, so the shape of "keep going until short" lives in one place.
func DrainExpired(ctx context.Context, batch int, del func(ctx context.Context, batch int) (int64, error)) (int64, error) {
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := del(ctx, batch)
		total += n
		if err != nil {
			return total, err
		}
		if batch <= 0 || n < int64(batch) {
			return total, nil
		}
	}
}
