package authsome

import (
	"context"
	"net/http"

	"github.com/xraph/forge"
)

// Auto-generated Forge middleware

// ForgeMiddleware returns a Forge middleware that injects auth into context
// This middleware verifies the session with the AuthSome backend and populates
// the request context with user and session information
func (c *Client) ForgeMiddleware() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			// Try to verify session with AuthSome backend
			session, err := c.GetCurrentSession(ctx.Request().Context())
			if err == nil && session != nil {
				// Inject user/session into request context
				newCtx := withAuthContext(ctx.Request().Context(), session)
				*ctx.Request() = *ctx.Request().WithContext(newCtx)
			}
			return next(ctx)
		}
	}
}

// RequireAuth returns Forge middleware that requires authentication
// Requests without valid authentication will receive a 401 response
func (c *Client) RequireAuth() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			authCtx := getAuthContext(ctx.Request().Context())
			if authCtx == nil || authCtx.Session == nil {
				return ctx.JSON(http.StatusUnauthorized, map[string]string{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}
			return next(ctx)
		}
	}
}

// OptionalAuth returns Forge middleware that optionally loads auth if present
// Unlike RequireAuth, this does not block unauthenticated requests
func (c *Client) OptionalAuth() forge.Middleware {
	return c.ForgeMiddleware()
}

// Context management for Forge middleware
type contextKey string

const (
	sessionContextKey contextKey = "authsome_session"
	userContextKey    contextKey = "authsome_user"
)

type authContext struct {
	Session *Session
	User    *User
}

func withAuthContext(ctx context.Context, session *Session) context.Context {
	ctx = context.WithValue(ctx, sessionContextKey, session)
	return ctx
}

func getAuthContext(ctx context.Context) *authContext {
	session, _ := ctx.Value(sessionContextKey).(*Session)
	if session == nil {
		return nil
	}
	return &authContext{
		Session: session,
	}
}

// GetUserFromContext retrieves user ID from Forge context
func GetUserFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}

// GetSessionFromContext retrieves session from Forge context
func GetSessionFromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}
