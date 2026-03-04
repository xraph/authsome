package middleware

import (
	"context"
	log "github.com/xraph/go-utils/log"
	"net/http"
	"strings"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

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

// SessionBindingConfig controls session binding validation.
type SessionBindingConfig struct {
	// BindToIP rejects requests when the client IP differs from the
	// IP recorded at session creation.
	BindToIP bool

	// BindToDevice rejects requests when the User-Agent differs from
	// the one recorded at session creation.
	BindToDevice bool
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
			token := extractBearerToken(ctx.Request())
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

			// Session binding: validate IP and/or device match
			if bindCfg.BindToIP && sess.IPAddress != "" {
				clientIP := clientIPFromRequest(ctx.Request())
				if clientIP != sess.IPAddress {
					logger.Warn("auth middleware: session IP mismatch",
						log.String("session_ip", sess.IPAddress),
						log.String("client_ip", clientIP),
						log.String("session_id", sess.ID.String()),
					)
					return ctx.JSON(http.StatusUnauthorized, map[string]any{
						"error": "session bound to different IP address",
						"code":  http.StatusUnauthorized,
					})
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
					return ctx.JSON(http.StatusUnauthorized, map[string]any{
						"error": "session bound to different device",
						"code":  http.StatusUnauthorized,
					})
				}
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
			token := extractBearerToken(ctx.Request())

			// Try bearer session resolution first (skip if token looks like an API key).
			if token != "" && !strings.HasPrefix(token, "ask_") {
				if resolved := trySessionAuth(ctx, token, resolveSession, resolveUser, logger, bindCfg); resolved {
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
			token := extractBearerToken(ctx.Request())

			if token != "" {
				// JWT detection: tokens with two dots are JWTs.
				if tokenformat.IsJWT(token) && jwtValidator != nil {
					if resolved := tryJWTAuth(ctx, token, jwtValidator, resolveUser, logger); resolved {
						return next(ctx)
					}
				}

				// Try opaque session resolution (skip API key prefixed tokens).
				if !strings.HasPrefix(token, "ask_") {
					if resolved := trySessionAuth(ctx, token, resolveSession, resolveUser, logger, bindCfg); resolved {
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
func tryJWTAuth(
	ctx forge.Context,
	token string,
	validator JWTValidator,
	resolveUser UserResolver,
	logger log.Logger,
) bool {
	claims, err := validator.ValidateJWT(token)
	if err != nil {
		logger.Debug("auth middleware: JWT validation failed",
			log.String("error", err.Error()),
		)
		return false
	}

	goCtx := ctx.Context()

	// Build a virtual session from JWT claims (no DB record needed).
	appID := id.ID(id.MustParse(claims.AppID))
	userID := id.ID(id.MustParse(claims.UserID))

	goCtx = WithAppID(goCtx, appID)
	goCtx = WithUserID(goCtx, userID)
	goCtx = WithAuthMethod(goCtx, "jwt")

	if claims.SessionID != "" {
		sessionID := id.ID(id.MustParse(claims.SessionID))
		goCtx = WithSessionID(goCtx, sessionID)
	}

	if claims.EnvID != "" {
		envID := id.ID(id.MustParse(claims.EnvID))
		goCtx = WithEnvID(goCtx, envID)
	}

	if claims.OrgID != "" {
		orgID := id.ID(id.MustParse(claims.OrgID))
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
		return true // Authenticated via JWT even if user lookup fails
	}
	goCtx = WithUser(goCtx, u)

	ctx.WithContext(goCtx)
	return true
}

// trySessionAuth attempts to authenticate via session token resolution.
// Returns true if authentication succeeded and context was updated.
func trySessionAuth(
	ctx forge.Context,
	token string,
	resolveSession SessionResolver,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) bool {
	sess, err := resolveSession(token)
	if err != nil {
		logger.Debug("auth middleware: invalid session token",
			log.String("error", err.Error()),
		)
		return false
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
			return false
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
			return false
		}
	}

	setSessionContext(ctx, sess, resolveUser, logger)
	return true
}

// tryStrategyAuth attempts to authenticate via the strategy registry.
// Returns true if a strategy successfully authenticated the request.
func tryStrategyAuth(
	ctx forge.Context,
	strategies StrategyAuthenticator,
	resolveUser UserResolver,
	logger log.Logger,
) bool {
	result, err := strategies.Authenticate(ctx.Context(), ctx.Request())
	if err != nil {
		logger.Debug("auth middleware: strategy auth failed",
			log.String("error", err.Error()),
		)
		return false
	}
	if result == nil || result.User == nil {
		return false
	}

	goCtx := ctx.Context()

	// Set session context from strategy result.
	if result.Session != nil {
		goCtx = WithSession(goCtx, result.Session)
		goCtx = WithSessionID(goCtx, result.Session.ID)
		goCtx = WithAppID(goCtx, result.Session.AppID)
		goCtx = WithUserID(goCtx, result.Session.UserID)

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

	goCtx = WithUser(goCtx, result.User)
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
				return ctx.JSON(http.StatusUnauthorized, map[string]any{
					"error": "authentication required",
					"code":  http.StatusUnauthorized,
				})
			}
			return next(ctx)
		}
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

// clientIPFromRequest extracts the client IP from the request, checking
// X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (client IP).
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// RemoteAddr is "host:port"; strip the port.
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}
