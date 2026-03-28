// Package authprovider implements forge auth providers for authsome's
// authentication methods (session, JWT, API key). These providers integrate
// with the forge auth extension's registry so routes can use declarative
// auth options like forge.WithAuth("session").
package authprovider

import (
	"context"
	"net/http"
	"strings"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/auth"

	authmw "github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// SessionData holds the resolved session and user for bridge middleware
// that converts auth.AuthContext into authsome's context values.
type SessionData struct {
	Session *session.Session
	User    *user.User
}

// CookieNameResolver returns the session cookie name for the current request
// context. Implementations should resolve from dynamic settings (per-app)
// and fall back to the default "authsome_session_token".
type CookieNameResolver func(ctx context.Context) string

// SessionProvider implements auth.AuthProvider for authsome session-based
// authentication. It resolves bearer tokens (from Authorization header or
// session cookie) into sessions and users.
type SessionProvider struct {
	resolveSession    authmw.SessionResolver
	resolveUser       authmw.UserResolver
	resolveCookieName CookieNameResolver
	logger            log.Logger
}

// NewSessionProvider creates a new session auth provider.
// cookieNameResolver is optional — if nil, defaults to "authsome_session_token".
func NewSessionProvider(
	resolveSession authmw.SessionResolver,
	resolveUser authmw.UserResolver,
	logger log.Logger,
	cookieNameResolver ...CookieNameResolver,
) *SessionProvider {
	p := &SessionProvider{
		resolveSession: resolveSession,
		resolveUser:    resolveUser,
		logger:         logger,
	}
	if len(cookieNameResolver) > 0 && cookieNameResolver[0] != nil {
		p.resolveCookieName = cookieNameResolver[0]
	}
	return p
}

// Name returns the provider name used in forge.WithAuth("session").
func (p *SessionProvider) Name() string { return "session" }

// Type returns the OpenAPI security scheme type.
func (p *SessionProvider) Type() auth.SecuritySchemeType { return auth.SecurityTypeHTTP }

// Authenticate validates the request and returns the authenticated user.
// It checks the Authorization header first, then falls back to the
// session cookie (resolved from dynamic settings per app).
func (p *SessionProvider) Authenticate(ctx context.Context, r *http.Request) (*auth.AuthContext, error) {
	// 1. Extract token from Authorization: Bearer <token>
	token := extractBearerToken(r)

	// 2. Fall back to session cookie (browser login).
	// Resolve cookie name from dynamic settings (per-app), defaulting
	// to "authsome_session_token".
	if token == "" {
		cookieName := "authsome_session_token"
		if p.resolveCookieName != nil {
			cookieName = p.resolveCookieName(ctx)
		}
		if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
			token = cookie.Value
		}
	}

	if token == "" {
		return nil, auth.ErrMissingCredentials
	}

	// 3. Resolve session from token
	sess, err := p.resolveSession(token)
	if err != nil {
		p.logger.Debug("session auth: invalid token",
			log.String("error", err.Error()),
		)
		return nil, auth.ErrInvalidCredentials
	}

	// 4. Resolve user from session
	u, err := p.resolveUser(sess.UserID.String())
	if err != nil {
		p.logger.Debug("session auth: failed to resolve user",
			log.String("user_id", sess.UserID.String()),
			log.String("error", err.Error()),
		)
		return nil, auth.ErrAuthenticationFailed
	}

	return &auth.AuthContext{
		Subject:      u.ID.String(),
		ProviderName: "session",
		Data: &SessionData{
			Session: sess,
			User:    u,
		},
		Claims: map[string]any{
			"email":      u.Email,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
		},
	}, nil
}

// OpenAPIScheme returns the OpenAPI security scheme definition.
func (p *SessionProvider) OpenAPIScheme() auth.SecurityScheme {
	return auth.SecurityScheme{
		Type:         string(auth.SecurityTypeHTTP),
		Description:  "Session-based authentication via Bearer token or auth_token cookie",
		Scheme:       "bearer",
		BearerFormat: "opaque",
	}
}

// Middleware returns HTTP middleware for this provider. The middleware
// resolves authentication and sets both the forge auth context AND the
// authsome middleware context values (UserID, AppID, Session, etc.) so
// existing handlers continue to work unchanged.
func (p *SessionProvider) Middleware() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			authCtx, err := p.Authenticate(ctx.Context(), ctx.Request())
			if err != nil {
				return ctx.JSON(http.StatusUnauthorized, map[string]any{
					"error": "authentication required",
					"code":  http.StatusUnauthorized,
				})
			}

			// Set forge auth context
			ctx.Set("auth_context", authCtx)

			// Bridge: set authsome middleware context values
			if data, ok := authCtx.Data.(*SessionData); ok {
				BridgeToContext(ctx, data)
			}

			return next(ctx)
		}
	}
}

// BridgeToContext sets authsome middleware context values from a SessionData.
// This bridges the forge auth system to authsome's middleware context so
// existing handlers using middleware.UserIDFrom() etc. continue to work.
func BridgeToContext(ctx forge.Context, data *SessionData) {
	if data == nil || data.Session == nil {
		return
	}

	goCtx := ctx.Context()
	goCtx = authmw.WithSession(goCtx, data.Session)
	goCtx = authmw.WithSessionID(goCtx, data.Session.ID)
	goCtx = authmw.WithAppID(goCtx, data.Session.AppID)
	goCtx = authmw.WithUserID(goCtx, data.Session.UserID)

	if data.Session.EnvID.Prefix() != "" {
		goCtx = authmw.WithEnvID(goCtx, data.Session.EnvID)
	}
	if data.Session.ImpersonatedBy.Prefix() != "" {
		goCtx = authmw.WithImpersonator(goCtx, data.Session.ImpersonatedBy)
	}
	if data.Session.OrgID.Prefix() != "" {
		goCtx = authmw.WithOrgID(goCtx, data.Session.OrgID)
		goCtx = forge.WithScope(goCtx, forge.NewOrgScope(
			data.Session.AppID.String(), data.Session.OrgID.String(),
		))
	} else {
		goCtx = forge.WithScope(goCtx, forge.NewAppScope(data.Session.AppID.String()))
	}

	if data.User != nil {
		goCtx = authmw.WithUser(goCtx, data.User)
		goCtx = authmw.WithAuthMethod(goCtx, "session")
	}

	ctx.WithContext(goCtx)
}

// RegistryMiddleware creates a registry-backed auth middleware that skips
// CORS preflight (OPTIONS) requests. Use this instead of calling
// registry.Middleware() directly to avoid noisy "failed for all providers"
// warnings from preflight requests.
func RegistryMiddleware(registry auth.Registry, providers ...string) forge.Middleware {
	inner := registry.Middleware(providers...)
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if ctx.Request().Method == http.MethodOptions {
				return next(ctx)
			}
			return inner(next)(ctx)
		}
	}
}

// extractBearerToken extracts the bearer token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return h[len(prefix):]
	}
	return ""
}
