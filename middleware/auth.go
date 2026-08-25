package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// authDebugKey carries the most recent strategy/auth rejection
// reason through the request context so RequireAuth can echo it
// back on a 401 when AUTHSOME_DEBUG_AUTH=1. The mechanism is
// debug-only: production deployments leave the env var unset and
// the reason never leaves the server.
type authDebugKey struct{}

// withAuthDebug attaches a human-readable rejection reason to ctx.
func withAuthDebug(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, authDebugKey{}, reason)
}

// authDebugFrom reads the most recent rejection reason from ctx.
func authDebugFrom(ctx context.Context) string {
	v, _ := ctx.Value(authDebugKey{}).(string) //nolint:errcheck // type-safe via key

	return v
}

// authDebugEnabled reports whether the operator has opted into
// surfacing strategy rejection reasons on 401 responses. Off by
// default — the reason carries credential-shape details (which
// header was missing, which app the key was minted under) that we
// don't want to expose to unauthenticated callers in production.
func authDebugEnabled() bool {
	return os.Getenv("AUTHSOME_DEBUG_AUTH") == "1"
}

// summarizeAuthHeaders renders a one-line description of the
// auth-relevant inbound headers — never the secret values, only
// presence + length so operators can tell "client never sent it"
// apart from "client sent it but it's malformed". Used by the
// debug_reason surface on 401 responses.
func summarizeAuthHeaders(r *http.Request) string {
	parts := []string{}
	if v := r.Header.Get("Authorization"); v != "" {
		parts = append(parts, "Authorization=present(len="+itoaInt(len(v))+",scheme="+authScheme(v)+")")
	} else {
		parts = append(parts, "Authorization=absent")
	}
	if v := r.Header.Get("X-API-Key"); v != "" {
		parts = append(parts, "X-API-Key=present(len="+itoaInt(len(v))+")")
	} else {
		parts = append(parts, "X-API-Key=absent")
	}
	if v := r.Header.Get("X-App-ID"); v != "" {
		parts = append(parts, "X-App-ID=present(value="+v+")")
	} else {
		parts = append(parts, "X-App-ID=absent")
	}
	if v := r.Header.Get("Cookie"); v != "" {
		parts = append(parts, "Cookie=present(len="+itoaInt(len(v))+")")
	} else {
		parts = append(parts, "Cookie=absent")
	}
	return strings.Join(parts, ", ")
}

func authScheme(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return "unknown"
}

func itoaInt(n int) string {
	// Tiny inline helper to avoid importing strconv just for this
	// debug path. n is always small (header lengths).
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// StrategyAuthenticator authenticates requests using registered strategies.
// The strategy.Registry implements this interface.
type StrategyAuthenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (*strategy.Result, error)
}

// SessionResolver loads a session from a token.
type SessionResolver func(token string) (*session.Session, error)

// UserResolver loads a user by ID string.
type UserResolver func(userID string) (*user.User, error)

// JWTValidator validates JWT access tokens and returns claims.
// The engine implements this via its TokenFormatForApp method.
type JWTValidator interface {
	ValidateJWT(token string) (*tokenformat.TokenClaims, error)
}

// SessionExistsChecker checks whether a session ID still exists in the store.
// Used for JWT revocation support — when enabled, JWT tokens are cross-checked
// against the session store so revoked sessions are rejected immediately.
type SessionExistsChecker func(sessionID string) (*session.Session, error)

// CookieNameResolver returns the session cookie name for the current context.
// When nil, the default "authsome_session_token" is used.
type CookieNameResolver func(ctx context.Context) string

// SessionBindingConfig controls session binding validation.
type SessionBindingConfig struct {
	// BindToIP rejects requests when the client IP differs from the
	// IP recorded at session creation.
	BindToIP bool

	// BindToDevice rejects requests when the User-Agent differs from
	// the one recorded at session creation.
	BindToDevice bool

	// CookieNameResolver returns the session cookie name for the current
	// request context. Falls back to "authsome_session_token" if nil or empty.
	CookieNameResolver CookieNameResolver

	// JWTSessionChecker, when set, cross-checks JWT tokens against the session
	// store. This enables JWT revocation — revoked sessions are rejected even
	// if the JWT signature is valid. Also enables IP/device binding for JWTs.
	JWTSessionChecker SessionExistsChecker

	// DPoPValidator enforces RFC 9449 proof-of-possession. When nil, DPoP is
	// not enforced at all, which is only correct in tests: the engine always
	// supplies one.
	DPoPValidator *dpop.Validator

	// DPoPNonceSigner mints the nonces served with a use_dpop_nonce challenge.
	// Nil disables nonce challenges.
	DPoPNonceSigner *dpop.NonceSigner

	// DPoPNonceRequired reports whether the app owning this session demands a
	// nonce. Nil means never.
	DPoPNonceRequired func(ctx context.Context, appID string) bool

	// DPoPAudit records a DPoP failure. Nil disables recording.
	DPoPAudit func(ctx context.Context, action string, md map[string]string)
}

// AuthMiddleware extracts the session token from the Authorization header,
// resolves the session and user, and stores them in context.
// This middleware is the forge.Scope producer — it resolves the authenticated
// identity and sets AppID/OrgID on context for all downstream extensions.
func AuthMiddleware(resolveSession SessionResolver, resolveUser UserResolver, logger log.Logger, binding ...SessionBindingConfig) forge.Middleware {
	var bindCfg SessionBindingConfig
	if len(binding) > 0 {
		bindCfg = binding[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			scheme, token := extractCredential(ctx.Request(), cookieName)
			if token == "" {
				return next(ctx)
			}

			sess, err := resolveSession(token)
			if err != nil {
				logger.Debug("auth middleware: invalid session token",
					log.String("error", err.Error()),
				)
				return next(ctx)
			}

			// Cross-tenant guard: a session minted under one app must not be
			// honored when the caller presents a different app's publishable key.
			if requestAppIDMismatch(ctx.Context(), sess.AppID.String()) {
				logger.Warn("auth middleware: session app mismatch (publishable key switched)",
					log.String("session_id", sess.ID.String()),
				)
				return next(ctx)
			}

			// Session binding: validate IP and/or device match
			if bindCfg.BindToIP && sess.IPAddress != "" {
				clientIP := clientIPFromRequest(ctx.Request())
				if clientIP != sess.IPAddress {
					logger.Warn("auth middleware: session IP mismatch",
						log.String("session_ip", sess.IPAddress),
						log.String("client_ip", clientIP),
						log.String("session_id", sess.ID.String()),
					)
					return forge.Unauthorized("session bound to different IP address")
				}
			}
			if bindCfg.BindToDevice && sess.UserAgent != "" {
				ua := ctx.Request().UserAgent()
				if ua != sess.UserAgent {
					logger.Warn("auth middleware: session device mismatch",
						log.String("session_ua", sess.UserAgent),
						log.String("client_ua", ua),
						log.String("session_id", sess.ID.String()),
					)
					return forge.Unauthorized("session bound to different device")
				}
			}

			if dpopErr := enforceDPoP(ctx, bindCfg, sess.DPoPJKT, scheme, token, sess.AppID.String(), sess.ID.String(), logger); dpopErr != nil {
				return dpopErr
			}

			goCtx := ctx.Context()
			goCtx = WithSession(goCtx, sess)
			goCtx = WithSessionID(goCtx, sess.ID)
			goCtx = WithAppID(goCtx, sess.AppID)
			goCtx = WithUserID(goCtx, sess.UserID)

			// Set environment ID from session (if present).
			if sess.EnvID.Prefix() != "" {
				goCtx = WithEnvID(goCtx, sess.EnvID)
			}

			// Detect impersonation
			if sess.ImpersonatedBy.Prefix() != "" {
				goCtx = WithImpersonator(goCtx, sess.ImpersonatedBy)
			}

			if sess.OrgID.Prefix() != "" {
				goCtx = WithOrgID(goCtx, sess.OrgID)
				goCtx = forge.WithScope(goCtx, forge.NewOrgScope(sess.AppID.String(), sess.OrgID.String()))
			} else {
				goCtx = forge.WithScope(goCtx, forge.NewAppScope(sess.AppID.String()))
			}

			u, err := resolveUser(sess.UserID.String())
			if err != nil {
				logger.Warn("auth middleware: failed to resolve user",
					log.String("user_id", sess.UserID.String()),
					log.String("error", err.Error()),
				)
				ctx.WithContext(goCtx)
				return next(ctx)
			}
			goCtx = WithUser(goCtx, u)

			ctx.WithContext(goCtx)
			return next(ctx)
		}
	}
}

// AuthMiddlewareWithStrategies creates middleware that tries bearer-session
// resolution first, then falls back to the strategy registry for alternative
// auth methods (API keys, etc.). Bearer tokens with the "ask_" prefix skip
// session resolution and go directly to the strategy chain.
//
// When a JWTValidator is provided, JWT tokens (detected by containing two
// dots) are validated stateless — no DB lookup needed for the access token.
//
// Flow:
//  1. Extract bearer token from Authorization header
//  2. If token is JWT and jwtValidator is set → validate stateless
//  3. If token exists and is NOT an API key prefix → resolve as session
//  4. If no session resolved → try strategies.Authenticate(ctx, r)
//  5. If strategy succeeds → set context with User, Session, AppID, Scope
//  6. If all fail → continue unauthenticated (RequireAuth enforces)
func AuthMiddlewareWithStrategies(
	resolveSession SessionResolver,
	resolveUser UserResolver,
	strategies StrategyAuthenticator,
	logger log.Logger,
	binding ...SessionBindingConfig,
) forge.Middleware {
	var bindCfg SessionBindingConfig
	if len(binding) > 0 {
		bindCfg = binding[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			scheme, token := extractCredential(ctx.Request(), cookieName)

			// Try bearer session resolution first (skip if token looks like an API key).
			if token != "" && !isAPIKeyToken(token) {
				resolved, err := trySessionAuth(ctx, scheme, token, resolveSession, resolveUser, logger, bindCfg)
				if err != nil {
					return err
				}
				if resolved {
					return next(ctx)
				}
			}

			// Fall back to strategy registry.
			if strategies != nil {
				if resolved := tryStrategyAuth(ctx, strategies, resolveUser, logger); resolved {
					return next(ctx)
				}
			}

			// No auth resolved — continue unauthenticated.
			return next(ctx)
		}
	}
}

// AuthMiddlewareWithJWT creates middleware that supports JWT token validation
// in addition to opaque session tokens and strategy auth. JWT tokens are
// validated stateless (no DB lookup for the access token itself).
func AuthMiddlewareWithJWT(
	resolveSession SessionResolver,
	resolveUser UserResolver,
	strategies StrategyAuthenticator,
	jwtValidator JWTValidator,
	logger log.Logger,
	binding ...SessionBindingConfig,
) forge.Middleware {
	var bindCfg SessionBindingConfig
	if len(binding) > 0 {
		bindCfg = binding[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			scheme, token := extractCredential(ctx.Request(), cookieName)

			if token != "" {
				// JWT detection: tokens with two dots are JWTs.
				if tokenformat.IsJWT(token) && jwtValidator != nil {
					resolved, err := tryJWTAuth(ctx, scheme, token, jwtValidator, resolveUser, logger, bindCfg)
					if err != nil {
						return err
					}
					if resolved {
						return next(ctx)
					}
				}

				// Try opaque session resolution (skip API key prefixed tokens).
				if !isAPIKeyToken(token) {
					resolved, err := trySessionAuth(ctx, scheme, token, resolveSession, resolveUser, logger, bindCfg)
					if err != nil {
						return err
					}
					if resolved {
						return next(ctx)
					}
				}
			}

			// Fall back to strategy registry.
			if strategies != nil {
				if resolved := tryStrategyAuth(ctx, strategies, resolveUser, logger); resolved {
					return next(ctx)
				}
			}

			return next(ctx)
		}
	}
}

// tryJWTAuth validates a JWT token stateless and sets context from claims.
// When a session checker is provided, the session is cross-checked against
// the store to support JWT revocation and IP/device binding.
//
// Returns (true, nil) on success, (false, nil) when the JWT simply does not
// apply here (so the caller can fall back to opaque-session or strategy
// auth), and (false, err) when a DPoP-bound token failed proof-of-possession
// enforcement. That failure must reach the caller as an actual 401, never as
// a silent "try something else" signal, or a bound token could be laundered
// through a fallback path that skips enforcement entirely.
func tryJWTAuth(
	ctx forge.Context,
	scheme, token string,
	validator JWTValidator,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) (bool, error) {
	claims, err := validator.ValidateJWT(token)
	if err != nil {
		logger.Debug("auth middleware: JWT validation failed",
			log.String("error", err.Error()),
		)
		return false, nil
	}

	// Cross-tenant guard: a JWT minted under one app must not be honored when
	// the caller presents a different app's publishable key.
	if requestAppIDMismatch(ctx.Context(), claims.AppID) {
		logger.Warn("auth middleware: JWT app mismatch (publishable key switched)",
			log.String("session_id", claims.SessionID),
		)
		return false, nil
	}

	// When a session checker is configured, cross-check the JWT's session ID
	// against the store. This enables revocation and IP/device binding for JWTs.
	// The checker returns (nil, nil) when the feature is disabled via settings.
	if bindCfg.JWTSessionChecker != nil && claims.SessionID != "" {
		sess, sessErr := bindCfg.JWTSessionChecker(claims.SessionID)
		if sessErr != nil {
			logger.Debug("auth middleware: JWT session not found in store (revoked?)",
				log.String("session_id", claims.SessionID),
			)
			return false, nil
		}

		// sess == nil && sessErr == nil means "feature disabled, skip checks".
		if sess != nil {
			// Validate IP binding for JWT.
			if bindCfg.BindToIP && sess.IPAddress != "" {
				clientIP := clientIPFromRequest(ctx.Request())
				if clientIP != sess.IPAddress {
					logger.Warn("auth middleware: JWT session IP mismatch",
						log.String("session_ip", sess.IPAddress),
						log.String("client_ip", clientIP),
						log.String("session_id", sess.ID.String()),
					)
					return false, nil
				}
			}

			// Validate device binding for JWT.
			if bindCfg.BindToDevice && sess.UserAgent != "" {
				ua := ctx.Request().UserAgent()
				if ua != sess.UserAgent {
					logger.Warn("auth middleware: JWT session device mismatch",
						log.String("session_ua", sess.UserAgent),
						log.String("client_ua", ua),
						log.String("session_id", sess.ID.String()),
					)
					return false, nil
				}
			}
		}
	}

	if dpopErr := enforceDPoP(ctx, bindCfg, claims.DPoPJKT, scheme, token, claims.AppID, claims.SessionID, logger); dpopErr != nil {
		return false, dpopErr
	}

	goCtx := ctx.Context()

	// Build a virtual session from JWT claims (no DB record needed).
	appID := id.MustParse(claims.AppID)
	userID := id.MustParse(claims.UserID)

	goCtx = WithAppID(goCtx, appID)
	goCtx = WithUserID(goCtx, userID)
	goCtx = WithAuthMethod(goCtx, "jwt")

	if claims.SessionID != "" {
		sessionID := id.MustParse(claims.SessionID)
		goCtx = WithSessionID(goCtx, sessionID)
	}

	if claims.EnvID != "" {
		envID := id.MustParse(claims.EnvID)
		goCtx = WithEnvID(goCtx, envID)
	}

	if claims.OrgID != "" {
		orgID := id.MustParse(claims.OrgID)
		goCtx = WithOrgID(goCtx, orgID)
		goCtx = forge.WithScope(goCtx, forge.NewOrgScope(claims.AppID, claims.OrgID))
	} else {
		goCtx = forge.WithScope(goCtx, forge.NewAppScope(claims.AppID))
	}

	// Resolve user from claims.
	u, err := resolveUser(claims.UserID)
	if err != nil {
		logger.Debug("auth middleware: JWT user resolution failed",
			log.String("user_id", claims.UserID),
			log.String("error", err.Error()),
		)
		ctx.WithContext(goCtx)
		return true, nil // Authenticated via JWT even if user lookup fails
	}
	goCtx = WithUser(goCtx, u)

	ctx.WithContext(goCtx)
	return true, nil
}

// trySessionAuth attempts to authenticate via session token resolution.
// Returns (true, nil) if authentication succeeded and context was updated,
// (false, nil) when the token simply did not resolve to a session (so the
// caller can fall back to strategy auth), and (false, err) when a
// DPoP-bound session failed proof-of-possession enforcement. That last case
// must surface as an actual 401 rather than "not resolved": otherwise a
// bound token with a missing or invalid proof would just fall through to
// the strategy chain and, finding nothing there either, be treated as an
// ordinary unauthenticated request instead of a rejected one.
func trySessionAuth(
	ctx forge.Context,
	scheme, token string,
	resolveSession SessionResolver,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) (bool, error) {
	sess, err := resolveSession(token)
	if err != nil {
		logger.Debug("auth middleware: invalid session token",
			log.String("error", err.Error()),
		)
		return false, nil
	}

	// Cross-tenant guard: a session minted under one app must not be honored
	// when the caller presents a different app's publishable key.
	if requestAppIDMismatch(ctx.Context(), sess.AppID.String()) {
		logger.Warn("auth middleware: session app mismatch (publishable key switched)",
			log.String("session_id", sess.ID.String()),
		)
		return false, nil
	}

	// Session binding: validate IP and/or device match.
	if bindCfg.BindToIP && sess.IPAddress != "" {
		clientIP := clientIPFromRequest(ctx.Request())
		if clientIP != sess.IPAddress {
			logger.Warn("auth middleware: session IP mismatch",
				log.String("session_ip", sess.IPAddress),
				log.String("client_ip", clientIP),
				log.String("session_id", sess.ID.String()),
			)
			return false, nil
		}
	}
	if bindCfg.BindToDevice && sess.UserAgent != "" {
		ua := ctx.Request().UserAgent()
		if ua != sess.UserAgent {
			logger.Warn("auth middleware: session device mismatch",
				log.String("session_ua", sess.UserAgent),
				log.String("client_ua", ua),
				log.String("session_id", sess.ID.String()),
			)
			return false, nil
		}
	}

	if dpopErr := enforceDPoP(ctx, bindCfg, sess.DPoPJKT, scheme, token, sess.AppID.String(), sess.ID.String(), logger); dpopErr != nil {
		return false, dpopErr
	}

	setSessionContext(ctx, sess, resolveUser, logger)
	return true, nil
}

// tryStrategyAuth attempts to authenticate via the strategy registry.
// Returns true if a strategy successfully authenticated the request.
func tryStrategyAuth(
	ctx forge.Context,
	strategies StrategyAuthenticator,
	_ UserResolver,
	logger log.Logger,
) bool {
	result, err := strategies.Authenticate(ctx.Context(), ctx.Request())
	if err != nil {
		// NotApplicableError means "no credentials of my flavor in
		// this request" — that's silent (a JWT request hitting an
		// API-key strategy is normal). Anything else is the
		// strategy explicitly rejecting credentials it found
		// (wrong key, expired, missing X-App-ID header, etc.) and
		// the operator needs to see the reason in their logs;
		// otherwise the 401 chain looks like an unexplained
		// "authentication required" with no diagnostic.
		var notApplicable strategy.NotApplicableError
		if errors.As(err, &notApplicable) {
			logger.Debug("auth middleware: strategy not applicable",
				log.String("reason", err.Error()),
			)
			// Even NotApplicable surfaces something — record it so
			// the operator can see *which* strategy bowed out and
			// which inbound headers were present. Without this, the
			// 401 chain blames "no strategy ran" with zero
			// information about what arrived on the wire.
			r := ctx.Request()
			present := summarizeAuthHeaders(r)
			ctx.WithContext(withAuthDebug(ctx.Context(),
				"strategy NotApplicable: "+err.Error()+"; inbound auth headers: "+present))
		} else {
			logger.Warn("auth middleware: strategy rejected credentials",
				log.String("error", err.Error()),
				log.String("path", ctx.Request().URL.Path),
				log.String("method", ctx.Request().Method),
			)
			// Stash the rejection reason on the context so RequireAuth
			// can echo it on 401 — gives operators a diagnostic
			// without needing access to the server logs.
			ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: "+err.Error()))
		}
		return false
	}
	if result == nil {
		// Strategy returned no error but produced no result at all.
		ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: returned nil result"))
		return false
	}
	isServiceAccount := result.Session != nil && result.Session.PrincipalKind == "service_account"
	if result.User == nil && !isServiceAccount {
		// no user and not a service account — treat as unauthenticated
		ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: returned no user (resolveUser nil or Result empty)"))
		return false
	}

	// Cross-tenant guard: credentials that resolve a session bound to one app
	// must not be honored when the caller presents a different app's
	// publishable key (e.g. an API key for app A sent with app B's pk header).
	if result.Session != nil && requestAppIDMismatch(ctx.Context(), result.Session.AppID.String()) {
		logger.Warn("auth middleware: strategy app mismatch (publishable key switched)",
			log.String("session_id", result.Session.ID.String()),
		)
		ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: session app mismatch with request publishable key"))
		return false
	}

	goCtx := ctx.Context()

	// Set session context from strategy result.
	if result.Session != nil {
		goCtx = WithSession(goCtx, result.Session)
		goCtx = WithSessionID(goCtx, result.Session.ID)
		goCtx = WithAppID(goCtx, result.Session.AppID)
		if !result.Session.UserID.IsNil() {
			goCtx = WithUserID(goCtx, result.Session.UserID)
		}

		if result.Session.EnvID.Prefix() != "" {
			goCtx = WithEnvID(goCtx, result.Session.EnvID)
		}
		if result.Session.OrgID.Prefix() != "" {
			goCtx = WithOrgID(goCtx, result.Session.OrgID)
			goCtx = forge.WithScope(goCtx, forge.NewOrgScope(result.Session.AppID.String(), result.Session.OrgID.String()))
		} else {
			goCtx = forge.WithScope(goCtx, forge.NewAppScope(result.Session.AppID.String()))
		}
	}

	if result.User != nil {
		goCtx = WithUser(goCtx, result.User)
	}
	goCtx = WithAuthMethod(goCtx, "strategy")

	ctx.WithContext(goCtx)
	return true
}

// setSessionContext populates the forge context with session and user data.
func setSessionContext(ctx forge.Context, sess *session.Session, resolveUser UserResolver, logger log.Logger) {
	goCtx := ctx.Context()
	goCtx = WithSession(goCtx, sess)
	goCtx = WithSessionID(goCtx, sess.ID)
	goCtx = WithAppID(goCtx, sess.AppID)
	goCtx = WithUserID(goCtx, sess.UserID)

	if sess.EnvID.Prefix() != "" {
		goCtx = WithEnvID(goCtx, sess.EnvID)
	}
	if sess.ImpersonatedBy.Prefix() != "" {
		goCtx = WithImpersonator(goCtx, sess.ImpersonatedBy)
	}

	if sess.OrgID.Prefix() != "" {
		goCtx = WithOrgID(goCtx, sess.OrgID)
		goCtx = forge.WithScope(goCtx, forge.NewOrgScope(sess.AppID.String(), sess.OrgID.String()))
	} else {
		goCtx = forge.WithScope(goCtx, forge.NewAppScope(sess.AppID.String()))
	}

	u, err := resolveUser(sess.UserID.String())
	if err != nil {
		logger.Warn("auth middleware: failed to resolve user",
			log.String("user_id", sess.UserID.String()),
			log.String("error", err.Error()),
		)
		goCtx = WithAuthMethod(goCtx, "session")
		ctx.WithContext(goCtx)
		return
	}
	goCtx = WithUser(goCtx, u)
	goCtx = WithAuthMethod(goCtx, "session")

	ctx.WithContext(goCtx)
}

// RequireAuth returns a forge middleware that rejects unauthenticated requests.
func RequireAuth() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if _, ok := UserFrom(ctx.Context()); !ok {
				body := map[string]any{
					"error": "authentication required",
					"code":  http.StatusUnauthorized,
				}
				// Surface the strategy rejection reason only when the
				// operator has opted in via AUTHSOME_DEBUG_AUTH=1. The
				// reason carries credential-shape reconnaissance
				// (which header was missing, scheme used, presence of
				// app-id) that an unauthenticated caller in production
				// must never see.
				if authDebugEnabled() {
					if reason := authDebugFrom(ctx.Context()); reason != "" {
						body["debug_reason"] = reason
					} else {
						body["debug_reason"] = "no strategy ran (no credentials presented or all strategies returned NotApplicable)"
					}
				}

				// Written by hand: this envelope carries a conditional
				// debug_reason field, which the typed constructors cannot
				// express — they take a message only.
				return ctx.JSON(http.StatusUnauthorized, body)
			}
			return next(ctx)
		}
	}
}

// isAPIKeyToken returns true if the token has an API key prefix (ask_, sk_*, pk_*).
func isAPIKeyToken(token string) bool {
	return apikey.IsAPIKey(token)
}

// requestAppIDMismatch reports whether the request carries a resolved
// publishable-key AppID that differs from the credential's bound app. A
// session/JWT minted under one app must not be honored when the caller
// presents a different app's publishable key. Returns false when the request
// has no publishable-key app (nothing to compare) so bearer-only flows are
// unaffected.
func requestAppIDMismatch(ctx context.Context, boundAppID string) bool {
	reqAppID, ok := AppIDFrom(ctx)
	if !ok || reqAppID.IsNil() || boundAppID == "" {
		return false
	}
	return reqAppID.String() != boundAppID
}

// Credential schemes returned by extractCredential.
const (
	schemeBearer = "bearer"
	schemeDPoP   = "dpop"
	schemeCookie = "cookie"
)

// extractCredential pulls the token out of the request and reports which
// scheme carried it.
//
// The scheme matters because a DPoP-bound token must be presented under the
// DPoP scheme (RFC 9449 section 7.1). Returning it here rather than discarding
// it lets the enforcement path refuse a bound token sent as Bearer.
func extractCredential(r *http.Request, cookieName string) (scheme, token string) {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 {
			switch {
			case strings.EqualFold(parts[0], "bearer"):
				return schemeBearer, parts[1]
			case strings.EqualFold(parts[0], "dpop"):
				return schemeDPoP, parts[1]
			}
		}
	}
	if cookieName == "" {
		cookieName = "authsome_session_token"
	}
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return schemeCookie, cookie.Value
	}
	return "", ""
}

// resolveCookieName returns the session cookie name using the resolver, or the default.
func resolveCookieName(resolver CookieNameResolver, ctx context.Context) string { //nolint:revive // ctx not first param: resolver is the primary input
	if resolver != nil {
		if name := resolver(ctx); name != "" {
			return name
		}
	}
	return "authsome_session_token"
}

// clientIPFromRequest extracts the client IP from the request. It delegates to
// the trusted-proxy-aware resolver so X-Forwarded-For / X-Real-IP are honored
// only when the immediate peer is a trusted proxy (a direct client cannot spoof
// its IP). Kept as a thin alias so existing call sites read naturally.
func clientIPFromRequest(r *http.Request) string {
	return ClientIP(r)
}

// dpopCheck is the outcome of applying RFC 9449 to a presented credential. It
// makes no HTTP response decisions, so the middleware (which can answer with a
// challenge) and the forge auth provider (which only gets to say yes or no)
// run one rule instead of two that drift apart.
type dpopCheck struct {
	ok    bool
	proof *dpop.Proof

	// code is the RFC 9449 error code for the challenge. Empty means refuse
	// without a challenge: the server could not check the binding at all, so
	// there is no proof the client could retry with.
	code string

	// action and reason describe the audit record. action is empty when
	// there is nothing worth auditing.
	action string
	reason string

	// nonceRequired carries what the app's nonce policy said, so a caller
	// that rotates the nonce does not resolve the setting a second time.
	nonceRequired bool
}

// checkDPoP runs the RFC 9449 checks for a credential already known to be
// bound. Callers deal with the unbound case before getting here.
func checkDPoP(
	ctx context.Context,
	cfg SessionBindingConfig,
	r *http.Request,
	boundJKT, scheme, token, appID string,
	logger log.Logger,
) dpopCheck {
	if cfg.DPoPValidator == nil {
		// A bound token with no validator configured cannot be checked. Refuse
		// rather than admit it: failing closed here turns a misconfiguration
		// into an outage, and failing open turns it into a silent bypass.
		logger.Error("auth middleware: DPoP-bound token but no validator configured")
		return dpopCheck{}
	}

	if scheme != schemeDPoP {
		logger.Warn("auth middleware: DPoP-bound token presented under the wrong scheme",
			log.String("scheme", scheme),
		)
		return dpopCheck{code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "wrong_scheme"}
	}

	raw := r.Header.Get("DPoP")
	if raw == "" {
		return dpopCheck{code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "missing_proof"}
	}

	proof, err := dpop.Parse(raw)
	if err != nil {
		logger.Warn("auth middleware: DPoP proof rejected",
			log.String("error", err.Error()),
		)
		return dpopCheck{code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "parse_failed"}
	}

	nonceRequired := cfg.DPoPNonceRequired != nil && cfg.DPoPNonceRequired(ctx, appID)

	err = cfg.DPoPValidator.Validate(ctx, proof, dpop.Expectation{
		Method:        r.Method,
		URL:           RequestURL(r),
		AccessToken:   token,
		ExpectedJKT:   boundJKT,
		NonceRequired: nonceRequired,
	})
	switch {
	case err == nil:
		return dpopCheck{ok: true, proof: proof, nonceRequired: nonceRequired}

	case errors.Is(err, dpop.ErrNonceRequired), errors.Is(err, dpop.ErrNonceMismatch):
		return dpopCheck{proof: proof, code: dpopErrUseNonce}

	case errors.Is(err, dpop.ErrKeyMismatch):
		// A structurally valid proof for the wrong key has no innocent
		// explanation the way a changed IP does, so it is logged apart from
		// ordinary client breakage.
		logger.Warn("auth middleware: DPoP key mismatch",
			log.String("bound_jkt", boundJKT),
			log.String("proof_jkt", proof.JKT),
		)
		return dpopCheck{proof: proof, code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "key_mismatch"}

	case errors.Is(err, dpop.ErrReplayed):
		// Rare, and interesting when it happens: it means somebody captured a
		// proof and the token it was minted for.
		logger.Warn("auth middleware: DPoP proof replayed",
			log.String("proof_jkt", proof.JKT),
		)
		return dpopCheck{proof: proof, code: dpopErrInvalidToken, action: hook.ActionDPoPProofReplayed}

	default:
		logger.Warn("auth middleware: DPoP validation failed",
			log.String("error", err.Error()),
		)
		return dpopCheck{proof: proof, code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "validation_failed"}
	}
}

// RFC 9449 error codes carried in a WWW-Authenticate challenge.
const (
	dpopErrInvalidToken = "invalid_token"
	dpopErrUseNonce     = "use_dpop_nonce"
)

// auditDPoP records a DPoP failure. session_id and app_id identify who tripped
// the rule; the raw proof and the token itself never go in, since metadata
// routinely ends up in longer-lived audit storage than the request that
// produced it.
func auditDPoP(ctx context.Context, cfg SessionBindingConfig, result dpopCheck, appID, sessionID string) {
	if cfg.DPoPAudit == nil || result.action == "" {
		return
	}
	md := map[string]string{"session_id": sessionID, "app_id": appID}
	if result.reason != "" {
		md["reason"] = result.reason
	}
	cfg.DPoPAudit(ctx, result.action, md)
}

// enforceDPoP applies RFC 9449 to a resolved credential.
//
// Enforcement follows the token, not the route and not the configuration. A
// token with no binding takes one string comparison and the identical path it
// took before this function existed, which is what lets DPoP roll out without
// a flag day. A token that is bound cannot be used without a matching proof,
// whatever the app or client mode says at the time of the request.
func enforceDPoP(
	ctx forge.Context,
	cfg SessionBindingConfig,
	boundJKT, scheme, token, appID, sessionID string,
	logger log.Logger,
) error {
	if boundJKT == "" {
		return nil // Unbound. Nothing to enforce.
	}

	result := checkDPoP(ctx.Context(), cfg, ctx.Request(), boundJKT, scheme, token, appID, logger)
	auditDPoP(ctx.Context(), cfg, result, appID, sessionID)

	switch {
	case result.ok:
		// Rotate the nonce before the client's current one expires, so a
		// long-lived client never has to eat a challenge round trip.
		if result.nonceRequired && cfg.DPoPNonceSigner != nil && cfg.DPoPNonceSigner.NeedsRefresh(result.proof.Nonce) {
			ctx.Response().Header().Set("DPoP-Nonce", cfg.DPoPNonceSigner.Issue(result.proof.JKT))
		}
		return nil

	case result.code == "":
		// The binding could not be checked at all, so there is no retry to
		// point the client at and no challenge to write.
		return forge.Unauthorized("token requires proof of possession")

	case result.code == dpopErrUseNonce:
		return dpopChallenge(ctx, cfg, result.proof.JKT, result.code)

	default:
		return dpopChallenge(ctx, cfg, "", result.code)
	}
}

// ErrDPoPBindingFailed reports a bound token presented without an acceptable
// proof, for callers outside the middleware chain that only get to answer yes
// or no and cannot write an RFC 9449 challenge.
var ErrDPoPBindingFailed = errors.New("authsome: token requires proof of possession")

// EnforceDPoPForRequest applies the rule enforceDPoP applies, for a credential
// resolved outside the HTTP middleware chain. The forge auth providers are the
// case that needs it: they return an auth context rather than a response, so
// they cannot answer with a challenge, but the binding still has to hold. A
// caller that can write a response should go through the middleware instead,
// so the client learns which retry would work.
func EnforceDPoPForRequest(
	ctx context.Context,
	cfg SessionBindingConfig,
	r *http.Request,
	boundJKT, scheme, token, appID, sessionID string,
	logger log.Logger,
) error {
	if boundJKT == "" {
		return nil // Unbound. Nothing to enforce.
	}

	result := checkDPoP(ctx, cfg, r, boundJKT, scheme, token, appID, logger)
	auditDPoP(ctx, cfg, result, appID, sessionID)
	if result.ok {
		return nil
	}
	return ErrDPoPBindingFailed
}

// ExtractCredential pulls the token out of a request and reports which scheme
// carried it, for auth paths outside this package that have to read a
// credential the way the middleware reads it. cookieName may be empty for the
// default session cookie.
func ExtractCredential(r *http.Request, cookieName string) (scheme, token string) {
	return extractCredential(r, cookieName)
}

// dpopChallenge writes an RFC 9449 challenge and returns the 401 to propagate.
func dpopChallenge(ctx forge.Context, cfg SessionBindingConfig, jkt, code string) error {
	header := `DPoP error="` + code + `", algs="` + strings.Join(dpop.SupportedAlgs(), " ") + `"`
	ctx.Response().Header().Set("WWW-Authenticate", header)

	if code == "use_dpop_nonce" && cfg.DPoPNonceSigner != nil && jkt != "" {
		ctx.Response().Header().Set("DPoP-Nonce", cfg.DPoPNonceSigner.Issue(jkt))
	}
	return forge.Unauthorized("invalid proof of possession")
}
