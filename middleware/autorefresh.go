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

	// ExposeRefreshToken controls whether the refresh token is included in
	// the X-Auth-Refresh-Token response header. When false (recommended),
	// only the access token is returned in headers. The refresh token should
	// only be obtained via the /v1/refresh endpoint.
	ExposeRefreshToken bool
}

// RefreshRequest carries everything the engine needs to re-check a session's
// bindings while rotating it. Auto-refresh happens on an ordinary request
// rather than a call to /v1/refresh, so the request's own details are what the
// checks run against: the DPoP proof is bound to this method and URL, and the
// IP and User-Agent are this request's.
type RefreshRequest struct {
	// RefreshToken is the session's current refresh token.
	RefreshToken string

	// IPAddress and UserAgent are validated against the session's stored
	// values when session binding is enabled.
	IPAddress string
	UserAgent string

	// DPoPProof is the raw DPoP header from this request. It is the same proof
	// the auth middleware already accepted, and re-checking it is safe: the
	// request-scoped record of accepted proofs keeps the replay cache from
	// reading it as a replay of itself.
	DPoPProof string
	// Method and RequestURL are this request's, for the proof's htm and htu.
	Method     string
	RequestURL string
}

// SessionRefresher refreshes a session using its refresh token and returns the
// updated session with new tokens. The engine's Refresh method fulfills this.
type SessionRefresher func(ctx context.Context, req RefreshRequest) (*session.Session, error)

// AutoRefreshConfigResolver returns the auto-refresh configuration for the
// current request context (may vary per app).
type AutoRefreshConfigResolver func(ctx context.Context) AutoRefreshConfig

// AutoRefreshMiddleware checks if the authenticated session's access token is
// near expiry and, if so, transparently refreshes it. New tokens are returned
// in response headers:
//   - X-Auth-Token: the new access token
//   - X-Auth-Refresh-Token: the new refresh token (only if ExposeRefreshToken is true)
//   - X-Auth-Token-Expires-At: RFC 3339 expiration timestamp
//
// When a CookieSetter is provided, the session cookie is also updated with the
// new access token so the browser cookie stays in sync.
//
// This middleware MUST run after AuthMiddleware so the session is on context.
// On refresh failure, the original response is returned unchanged.
func AutoRefreshMiddleware(
	refresher SessionRefresher,
	configResolver AutoRefreshConfigResolver,
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
			// Auto-refresh rotates the session on an ordinary request, so the
			// bindings are re-checked against that request rather than against
			// a call to /v1/refresh that never happened.
			httpReq := ctx.Request()
			refreshed, err := refresher(ctx.Context(), RefreshRequest{
				RefreshToken: sess.RefreshToken,
				IPAddress:    ClientIP(httpReq),
				UserAgent:    httpReq.UserAgent(),
				DPoPProof:    httpReq.Header.Get("DPoP"),
				Method:       httpReq.Method,
				RequestURL:   RequestURL(httpReq),
			})
			if err != nil {
				logRefreshFailure(logger, sess, err)
				return nil // non-fatal: let the original response through
			}

			// Set new tokens in response headers.
			ctx.Response().Header().Set("X-Auth-Token", refreshed.Token)
			ctx.Response().Header().Set("X-Auth-Token-Expires-At", refreshed.ExpiresAt.Format(time.RFC3339))

			// Only expose refresh token in headers when explicitly enabled.
			if cfg.ExposeRefreshToken {
				ctx.Response().Header().Set("X-Auth-Refresh-Token", refreshed.RefreshToken)
			}

			// Re-set the session cookie with the new access token so the
			// browser cookie stays in sync after auto-refresh.
			if setter != nil {
				maxAge := int(time.Until(refreshed.ExpiresAt).Seconds())
				if maxAge <= 0 {
					maxAge = 3600
				}
				setter(ctx, refreshed.Token, maxAge)
			}

			logger.Debug("auto-refresh: refreshed near-expiry access token",
				log.String("session_id", sess.ID.String()),
			)

			return nil
		}
	}
}

// logRefreshFailure reports an auto-refresh that did not happen.
//
// The level depends on how surprising the failure is. An unbound session
// failing to refresh is ordinary: the refresh token has expired, or a
// concurrent request already rotated it, and neither is worth more than a
// Debug line on a hot path.
//
// A DPoP-bound session failing is not ordinary. The auth middleware already
// accepted a valid proof for this very request, so a refusal here means
// something is genuinely off: clock skew, a nonce rotating mid-request, or a
// validator that is not configured the way the issuer is. Logging that at
// Debug is what let bound clients lose transparent refresh silently, with the
// only trace at a level nobody enables in production.
//
// Auto-refresh runs on every request inside the threshold window, so a
// persistent fault will repeat this line. That is the intended trade: a
// security control that has quietly stopped working should be noisy.
func logRefreshFailure(logger log.Logger, sess *session.Session, err error) {
	if sess.DPoPJKT == "" {
		logger.Debug("auto-refresh: failed to refresh session",
			log.String("session_id", sess.ID.String()),
			log.String("error", err.Error()),
		)
		return
	}

	logger.Warn("auto-refresh: DPoP-bound session failed to refresh",
		log.String("session_id", sess.ID.String()),
		log.String("error", err.Error()),
	)
}
