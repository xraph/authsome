package bearer

import (
	"context"
	"strings"
	"time"

	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// Plugin implements bearer token authentication middleware
type Plugin struct {
	sessionSvc *session.Service
	userSvc    *user.Service
}

// NewPlugin creates a new bearer token plugin
func NewPlugin(sessionSvc *session.Service, userSvc *user.Service) *Plugin {
	return &Plugin{
		sessionSvc: sessionSvc,
		userSvc:    userSvc,
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "bearer"
}

// AuthenticateHandler returns a handler function that can be used as middleware
func (p *Plugin) AuthenticateHandler(next func(*forge.Context) error) func(*forge.Context) error {
	return func(c *forge.Context) error {
		// Extract bearer token from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return next(c)
		}

		// Check if it's a bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return next(c)
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return next(c)
		}

		// Validate the token using session service
		sess, err := p.sessionSvc.FindByToken(c.Request().Context(), token)
		if err != nil {
			return next(c)
		}

		// Check if session is valid and not expired
		if sess == nil || time.Now().After(sess.ExpiresAt) {
			return next(c)
		}

		// Get user information
		user, err := p.userSvc.FindByID(c.Request().Context(), sess.UserID)
		if err != nil {
			return next(c)
		}

		// Store user and session in request context
		ctx := context.WithValue(c.Request().Context(), "user", user)
		ctx = context.WithValue(ctx, "session", sess)
		ctx = context.WithValue(ctx, "authenticated", true)
		
		// Update request with new context
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// RequireAuthHandler returns a handler that requires authentication
func (p *Plugin) RequireAuthHandler(next func(*forge.Context) error) func(*forge.Context) error {
	return func(c *forge.Context) error {
		// Check if user is authenticated
		if c.Request().Context().Value("authenticated") != true {
			return c.JSON(401, map[string]string{
				"error": "Authentication required",
			})
		}
		return next(c)
	}
}

// GetUser extracts the authenticated user from context
func GetUser(c *forge.Context) *user.User {
	if u := c.Request().Context().Value("user"); u != nil {
		if user, ok := u.(*user.User); ok {
			return user
		}
	}
	return nil
}

// GetSession extracts the session from context
func GetSession(c *forge.Context) *session.Session {
	if s := c.Request().Context().Value("session"); s != nil {
		if sess, ok := s.(*session.Session); ok {
			return sess
		}
	}
	return nil
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c *forge.Context) bool {
	return c.Request().Context().Value("authenticated") == true
}