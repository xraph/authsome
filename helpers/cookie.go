package helpers

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/forge"
)

// SetSessionCookieFromAuth is a helper that retrieves cookie configuration from the auth instance
// and sets the session cookie if enabled. This is a convenience method for handlers and plugins.
func SetSessionCookieFromAuth(
	c forge.Context,
	authInst core.Authsome,
	token string,
	expiresAt time.Time,
) error {
	// Get global cookie config
	config := authInst.GetConfig()
	if !config.SessionCookie.Enabled {
		// Cookies disabled globally
		return nil
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID == xid.NilID() {
		// No app context, use global config
		return session.SetCookie(c, token, expiresAt, &config.SessionCookie)
	}

	// Get app service from registry
	serviceRegistry := authInst.GetServiceRegistry()
	if serviceRegistry == nil {
		// Fall back to global config
		return session.SetCookie(c, token, expiresAt, &config.SessionCookie)
	}

	appService := serviceRegistry.AppService()
	if appService == nil {
		// Fall back to global config
		return session.SetCookie(c, token, expiresAt, &config.SessionCookie)
	}

	// Get app-specific cookie config (merged with global)
	appCookieConfig, err := appService.App.GetCookieConfig(c.Request().Context(), appID)
	if err != nil {
		// Fall back to global config on error
		return session.SetCookie(c, token, expiresAt, &config.SessionCookie)
	}

	// Set cookie with app-specific config
	return session.SetCookie(c, token, expiresAt, appCookieConfig)
}

