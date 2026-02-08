package pages

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/router"
)

const (
	csrfCookieName        = "dashboard_csrf_token"
	sessionCookieName     = "authsome_session"
	environmentCookieName = "authsome_environment"
)

func (p *PagesManager) checkExistingPageSession(ctx *router.PageContext) *user.User {
	return nil
}

// checkExistingSession checks if there's a valid session without middleware
// Returns user if authenticated, nil otherwise
func (p *PagesManager) checkExistingSession(c forge.Context) *user.User {
	// Extract session token from cookie
	cookie, err := c.Request().Cookie(sessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return nil
	}

	sessionToken := cookie.Value

	// Validate session
	sess, err := p.services.SessionService().FindByToken(c.Request().Context(), sessionToken)
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
	user, err := p.services.UserService().FindByID(ctx, sess.UserID)
	if err != nil || user == nil {
		return nil
	}

	return user
}

// isFirstUser checks if there are any users in the system
// This is a global check that bypasses organization context for the first system user
func (p *PagesManager) isFirstUser(ctx context.Context) (bool, error) {
	// Check if platform app exists and has any members
	platformApp, err := p.services.AppService().GetPlatformApp(ctx)
	if err != nil {
		// No platform app exists - this is definitely the first user
		return true, nil
	}

	// Count members in the platform app
	count, err := p.services.AppService().CountMembers(ctx, platformApp.ID)
	if err != nil {
		return false, fmt.Errorf("failed to count members: %w", err)
	}

	// If no members exist, this is the first user
	return count == 0, nil
}

// generateCSRFToken generates a simple CSRF token
func (p *PagesManager) generateCSRFToken() string {
	return xid.New().String()
}
