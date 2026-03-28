package middleware

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
)

// SessionActivityConfig controls sliding session extension behavior.
type SessionActivityConfig struct {
	// Enabled turns on session extension on activity.
	Enabled bool

	// InactivityTimeout is how long a session lives without activity.
	// On each authenticated request, ExpiresAt is reset to now + InactivityTimeout.
	// Default: 30 minutes.
	InactivityTimeout time.Duration
}

// SessionActivityConfigResolver returns the activity extension configuration
// for the current request context (may vary per app).
type SessionActivityConfigResolver func(ctx context.Context) SessionActivityConfig

// SessionToucher updates a session's last activity time and expiry.
type SessionToucher func(ctx context.Context, sessionID id.SessionID, lastActivityAt, expiresAt time.Time) error

// CookieSetter re-sets the session cookie with the given token and max-age.
// Used by activity and auto-refresh middleware to keep the browser cookie
// in sync with the server-side session lifetime.
type CookieSetter func(ctx forge.Context, token string, maxAge int)

// minTouchInterval is the minimum time between successive database writes
// for activity tracking. Prevents excessive DB writes on rapid requests.
const minTouchInterval = 60 * time.Second

// SessionActivityMiddleware extends session expiry on each authenticated
// request, implementing a sliding session window. It updates LastActivityAt
// and extends ExpiresAt to now + InactivityTimeout.
//
// To avoid a database write on every single request, the session is only
// touched if LastActivityAt is older than minTouchInterval (60 seconds).
//
// When a CookieSetter is provided, the session cookie is re-set with the
// extended MaxAge so the browser cookie stays in sync with the server-side
// session lifetime.
//
// This middleware MUST run after AuthMiddleware so the session is on context.
func SessionActivityMiddleware(
	toucher SessionToucher,
	configResolver SessionActivityConfigResolver,
	logger log.Logger,
	cookieSetter ...CookieSetter,
) forge.Middleware {
	var setter CookieSetter
	if len(cookieSetter) > 0 {
		setter = cookieSetter[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			// Run the actual handler first.
			if err := next(ctx); err != nil {
				return err
			}

			// Only extend for authenticated sessions.
			sess, ok := SessionFrom(ctx.Context())
			if !ok || sess == nil {
				return nil
			}

			// Resolve config (may be per-app).
			cfg := configResolver(ctx.Context())
			if !cfg.Enabled {
				return nil
			}

			timeout := cfg.InactivityTimeout
			if timeout == 0 {
				timeout = 7 * 24 * time.Hour
			}

			// Throttle: only touch the store if enough time has passed since
			// the last activity update to avoid excessive DB writes.
			now := time.Now()
			if !sess.LastActivityAt.IsZero() && now.Sub(sess.LastActivityAt) < minTouchInterval {
				return nil
			}

			newExpiresAt := now.Add(timeout)
			if err := toucher(ctx.Context(), sess.ID, now, newExpiresAt); err != nil {
				logger.Debug("session-activity: failed to touch session",
					log.String("session_id", sess.ID.String()),
					log.String("error", err.Error()),
				)
				return nil // non-fatal: let the original response through
			}

			// Update the in-memory session so downstream middleware sees
			// the extended expiry (e.g. auto-refresh checks).
			sess.LastActivityAt = now
			sess.ExpiresAt = newExpiresAt

			// Re-set the session cookie with the extended expiry so the browser
			// doesn't discard it before the server-side session expires.
			if setter != nil {
				setter(ctx, sess.Token, int(timeout.Seconds()))
			}

			logger.Debug("session-activity: extended session expiry",
				log.String("session_id", sess.ID.String()),
			)

			return nil
		}
	}
}
