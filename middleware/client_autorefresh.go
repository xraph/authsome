package middleware

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	authclient "github.com/xraph/authsome/sdk/go"
)

// ClientSessionRefresher rotates the session behind a browser's Cookie header
// and reports the new tokens plus the Set-Cookie headers the identity server
// issued. Client.RefreshTokensWithCookies fulfills it.
type ClientSessionRefresher func(ctx context.Context, cookieHeader string) (*authclient.RefreshedSession, error)

// NewClientSessionRefresher adapts an authclient to ClientSessionRefresher.
func NewClientSessionRefresher(client *authclient.Client) ClientSessionRefresher {
	return func(ctx context.Context, cookieHeader string) (*authclient.RefreshedSession, error) {
		return client.RefreshTokensWithCookies(ctx, cookieHeader)
	}
}

// ClientAutoRefreshMiddleware is the client-mode counterpart to
// AutoRefreshMiddleware: it keeps a browser's cookie session alive by rotating
// it shortly before the access token expires.
//
// Why client mode needs its own:
//
// AutoRefreshMiddleware and SessionActivityMiddleware both reach into the
// session store — one to read sess.ExpiresAt and call Engine.Refresh, the other
// to extend the row — so neither can run in a service that has no engine.
// Extension.Middlewares() therefore returned only ClientAuthMiddleware in
// client mode, and the comment there said session refresh was "Portal's
// responsibility". Nothing took up that responsibility, so a cookie session
// died at the access token's TTL (one hour by default) no matter how actively
// it was being used: the cookie's own Max-Age is that same TTL, and
// ResolveSessionByToken refuses the session the moment it lapses.
//
// How it works:
//
// POST /v1/refresh is cookie-first, rotating from the session's server-side
// refresh token when the request carries a valid session cookie. So this
// middleware never needs the refresh token — which is just as well, since a
// client-mode service is only ever shown the access token. It forwards the
// browser's Cookie header, then replays the identity server's Set-Cookie
// response onto its own response so cookie attributes stay resolved in exactly
// one place.
//
// Why the work happens BEFORE the handler:
//
// Forge writes the status line and headers straight through to the connection —
// it does not buffer the response. A header set after next() returns has
// already missed the wire, however healthy it looks on an httptest recorder
// (whose Header() map keeps accepting writes long after WriteHeader snapshotted
// it). Rotating first is what makes the new cookie actually reach the browser.
//
// The cost is a round-trip in front of the handler, paid only inside the
// refresh window rather than on every request.
//
// Ordering: this MUST run after ClientAuthMiddleware, which is what records the
// expiry it reads. A refresh failure is non-fatal — the handler runs either way
// and the session keeps its original expiry.
func ClientAutoRefreshMiddleware(
	refresher ClientSessionRefresher,
	cfg AutoRefreshConfig,
	logger log.Logger,
) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if refreshed := maybeRefreshClientSession(ctx, refresher, cfg, logger); refreshed != nil {
				applyClientRefresh(ctx, refreshed, cfg)
			}
			return next(ctx)
		}
	}
}

// maybeRefreshClientSession decides whether this request's session is due for
// rotation and performs it. Returns nil when the session is untouched, for any
// reason.
func maybeRefreshClientSession(
	ctx forge.Context,
	refresher ClientSessionRefresher,
	cfg AutoRefreshConfig,
	logger log.Logger,
) *authclient.RefreshedSession {
	if refresher == nil || !cfg.Enabled {
		return nil
	}

	hint, ok := clientSessionHintFrom(ctx.Context())
	if !ok || hint.token == "" {
		return nil // unauthenticated, or introspection never succeeded
	}
	// Only browsers are refreshed on their own behalf. An API client holding
	// its own refresh token rotates when it chooses, and rotating underneath it
	// would strand the token still in its hand.
	if !hint.fromCookie {
		return nil
	}

	// A zero expiry means introspection reported none, so there is no basis to
	// decide; leave the session alone rather than rotating on every request.
	if hint.expiresAt.IsZero() {
		return nil
	}
	threshold := cfg.Threshold
	if threshold == 0 {
		threshold = 5 * time.Minute
	}
	// Already lapsed is equally untouchable: /v1/refresh resolves the session
	// behind the cookie before rotating it, so an expired cookie has nothing
	// left to rotate from.
	timeUntilExpiry := time.Until(hint.expiresAt)
	if timeUntilExpiry > threshold || timeUntilExpiry <= 0 {
		return nil
	}

	cookieHeader := ctx.Request().Header.Get("Cookie")
	if cookieHeader == "" {
		return nil
	}

	refreshed, err := refresher(ctx.Context(), cookieHeader)
	if err != nil {
		// Debug, not Warn: losing a race with another tab's refresh is routine
		// and self-correcting, and the browser still holds a working token
		// until the original expiry.
		logger.Debug("client auto-refresh: refresh failed; session left on its original expiry",
			log.String("error", err.Error()),
			log.String("path", ctx.Request().URL.Path),
		)
		return nil
	}
	logger.Debug("client auto-refresh: rotated session",
		log.String("path", ctx.Request().URL.Path),
	)
	return refreshed
}

// applyClientRefresh puts the rotated session onto the response, and onto the
// context for anything downstream that forwards the caller's token.
func applyClientRefresh(ctx forge.Context, refreshed *authclient.RefreshedSession, cfg AutoRefreshConfig) {
	// Replay the identity server's cookies verbatim. It resolved name, domain,
	// path, Secure, HttpOnly, SameSite and the __Host- prefix from its own
	// settings; this service holds no settings manager and would only be
	// guessing at them.
	header := ctx.Response().Header()
	for _, sc := range refreshed.SetCookies {
		header.Add("Set-Cookie", sc)
	}

	header.Set("X-Auth-Token", refreshed.SessionToken)
	if !refreshed.ExpiresAt.IsZero() {
		header.Set("X-Auth-Token-Expires-At", refreshed.ExpiresAt.Format(time.RFC3339))
	}
	if cfg.ExposeRefreshToken && refreshed.RefreshToken != "" {
		header.Set("X-Auth-Refresh-Token", refreshed.RefreshToken)
	}

	// Rotation is a compare-and-swap on the old access token (Engine.Refresh),
	// so the token this request arrived with is now dead. Anything reading
	// RawTokenFrom to call another service — the reason WithRawToken exists —
	// has to be handed the live one instead.
	goCtx := WithRawToken(ctx.Context(), refreshed.SessionToken)
	if hint, ok := clientSessionHintFrom(goCtx); ok {
		hint.token = refreshed.SessionToken
		hint.expiresAt = refreshed.ExpiresAt
		goCtx = withClientSessionHint(goCtx, hint)
	}
	ctx.WithContext(goCtx)
}
