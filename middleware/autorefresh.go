package middleware

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/session"
)

// AutoRefreshConfig controls automatic token refresh behavior.
type AutoRefreshConfig struct {
	// Enabled turns on auto-refresh.
	Enabled bool

	// Threshold is the time before access token expiry to trigger refresh.
	// Default: 5 minutes.
	Threshold time.Duration
}

// SessionRefresher refreshes a session using its refresh token and returns the
// updated session with new tokens. The engine's Refresh method fulfills this.
type SessionRefresher func(ctx context.Context, refreshToken string) (*session.Session, error)

// AutoRefreshConfigResolver returns the auto-refresh configuration for the
// current request context (may vary per app).
type AutoRefreshConfigResolver func(ctx context.Context) AutoRefreshConfig

// AutoRefreshMiddleware checks if the authenticated session's access token is
// near expiry and, if so, transparently refreshes it. New tokens are returned
// in response headers:
//   - X-Auth-Token: the new access token
//   - X-Auth-Refresh-Token: the new refresh token
//   - X-Auth-Token-Expires-At: RFC 3339 expiration timestamp
//
// This middleware MUST run after AuthMiddleware so the session is on context.
// On refresh failure, the original response is returned unchanged.
func AutoRefreshMiddleware(
	refresher SessionRefresher,
	configResolver AutoRefreshConfigResolver,
	logger log.Logger,
) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			// Run the actual handler first.
			if err := next(ctx); err != nil {
				return err
			}

			// Only attempt auto-refresh for authenticated sessions.
			sess, ok := SessionFrom(ctx.Context())
			if !ok || sess == nil {
				return nil
			}

			// Resolve config (may be per-app).
			cfg := configResolver(ctx.Context())
			if !cfg.Enabled {
				return nil
			}

			threshold := cfg.Threshold
			if threshold == 0 {
				threshold = 5 * time.Minute
			}

			// Check if the access token is within the refresh threshold.
			timeUntilExpiry := time.Until(sess.ExpiresAt)
			if timeUntilExpiry > threshold || timeUntilExpiry <= 0 {
				return nil // not near expiry or already expired
			}

			// Perform the refresh.
			refreshed, err := refresher(ctx.Context(), sess.RefreshToken)
			if err != nil {
				logger.Debug("auto-refresh: failed to refresh session",
					log.String("session_id", sess.ID.String()),
					log.String("error", err.Error()),
				)
				return nil // non-fatal: let the original response through
			}

			// Set new tokens in response headers.
			ctx.Response().Header().Set("X-Auth-Token", refreshed.Token)
			ctx.Response().Header().Set("X-Auth-Refresh-Token", refreshed.RefreshToken)
			ctx.Response().Header().Set("X-Auth-Token-Expires-At", refreshed.ExpiresAt.Format(time.RFC3339))

			logger.Debug("auto-refresh: refreshed near-expiry access token",
				log.String("session_id", sess.ID.String()),
			)

			return nil
		}
	}
}
