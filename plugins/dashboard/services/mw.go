package services

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/router"
)

const (
	csrfCookieName        = "dashboard_csrf_token"
	sessionCookieName     = "authsome_session"
	environmentCookieName = "authsome_environment"

	UserKey          = "user"
	SessionKey       = "session"
	AuthenticatedKey = "authenticated"
)

func (s *Services) CheckExistingPageSession(c *router.PageContext) (*user.User, *session.Session, error) {
	// Extract session token from cookie
	cookie, err := c.Request.Cookie(sessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil, nil, errs.SessionNotFound()
	}

	sessionToken := cookie.Value

	// Validate session
	sess, err := s.SessionService().FindByToken(c.Request.Context(), sessionToken)
	if err != nil || sess == nil {
		return nil, nil, errs.SessionInvalid()
	}

	// Check if session is expired
	if time.Now().After(sess.ExpiresAt) {
		return nil, nil, errs.SessionExpired()
	}

	// Set app context from session for user lookup (required for multi-tenancy)
	ctx := c.Request.Context()
	if !sess.AppID.IsNil() {
		ctx = contexts.SetAppID(ctx, sess.AppID)
	} else {
	}

	// Get user information
	user, err := s.UserService().FindByID(ctx, sess.UserID)
	if err != nil || user == nil {
		return nil, nil, errs.UserNotFound()
	}

	return user, sess, nil
}

// checkExistingSession checks if there's a valid session without middleware
// checkExistingSession user if authenticated, nil otherwise.
func (s *Services) checkExistingSession(c forge.Context) *user.User {
	// Extract session token from cookie
	cookie, err := c.Request().Cookie(sessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil
	}

	sessionToken := cookie.Value

	// Validate session
	sess, err := s.SessionService().FindByToken(c.Request().Context(), sessionToken)
	if err != nil || sess == nil {
		return nil
	}

	// Check if session is expired
	if time.Now().After(sess.ExpiresAt) {
		return nil
	}

	// Set app context from session for user lookup (required for multi-tenancy)
	ctx := c.Request().Context()
	if !sess.AppID.IsNil() {
		ctx = contexts.SetAppID(ctx, sess.AppID)
	} else {
	}

	// Get user information
	user, err := s.UserService().FindByID(ctx, sess.UserID)
	if err != nil || user == nil {
		return nil
	}

	return user
}

// isFirstUser checks if there are any users in the system
// IsFirstUser is a global check that bypasses organization context for the first system user.
func (s *Services) IsFirstUser(ctx context.Context) (bool, error) {
	// Check if platform app exists and has any members
	platformApp, err := s.AppService().GetPlatformApp(ctx)
	if err != nil {
		// No platform app exists - this is definitely the first user
		return true, nil
	}

	// Count members in the platform app
	count, err := s.AppService().CountMembers(ctx, platformApp.ID)
	if err != nil {
		return false, fmt.Errorf("failed to count members: %w", err)
	}

	// If no members exist, this is the first user
	return count == 0, nil
}

// GenerateCSRFToken generateCSRFToken generates a simple CSRF token.
func (s *Services) GenerateCSRFToken() string {
	return xid.New().String()
}
