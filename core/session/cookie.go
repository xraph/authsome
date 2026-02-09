package session

import (
	"net/http"
	"time"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// SetCookie sets a session cookie based on the provided configuration
// It handles auto-detection of the Secure flag, SameSite parsing, and MaxAge calculation.
func SetCookie(
	c forge.Context,
	token string,
	expiresAt time.Time,
	config *CookieConfig,
) error {
	if config == nil {
		return errs.RequiredField("config")
	}

	if !config.Enabled {
		// Cookie setting is disabled, nothing to do
		return nil
	}

	if token == "" {
		return errs.RequiredField("token")
	}

	// Determine cookie name (use default if not specified)
	cookieName := config.Name
	if cookieName == "" {
		cookieName = "authsome_session"
	}

	// Determine path (use default if not specified)
	path := config.Path
	if path == "" {
		path = "/"
	}

	// Determine Secure flag
	// If explicitly set, use that value
	// If nil, auto-detect based on whether the request is over TLS
	secure := false
	if config.Secure != nil {
		secure = *config.Secure
	} else {
		// Auto-detect: use secure if request is over TLS
		secure = c.Request().TLS != nil
	}

	// Parse SameSite mode
	sameSite := ParseSameSite(config.SameSite)

	// Calculate MaxAge
	// If explicitly set, use that value
	// Otherwise, calculate from expiresAt
	maxAge := 0
	if config.MaxAge != nil {
		maxAge = *config.MaxAge
	} else {
		// Calculate from expires time
		duration := time.Until(expiresAt)
		if duration > 0 {
			maxAge = int(duration.Seconds())
		}
	}

	// Build the cookie
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     path,
		Domain:   config.Domain,
		Expires:  expiresAt,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: config.HttpOnly,
		SameSite: sameSite,
	}

	// Set the cookie on the response
	http.SetCookie(c.Response(), cookie)

	return nil
}

// ClearCookie clears a session cookie by setting it to expire immediately.
func ClearCookie(c forge.Context, config *CookieConfig) error {
	if config == nil {
		return errs.RequiredField("config")
	}

	// Determine cookie name (use default if not specified)
	cookieName := config.Name
	if cookieName == "" {
		cookieName = "authsome_session"
	}

	// Determine path (use default if not specified)
	path := config.Path
	if path == "" {
		path = "/"
	}

	// Create an expired cookie to clear it
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     path,
		Domain:   config.Domain,
		Expires:  time.Unix(0, 0), // Set to epoch to expire immediately
		MaxAge:   -1,              // MaxAge < 0 means delete cookie now
		Secure:   config.Secure != nil && *config.Secure,
		HttpOnly: config.HttpOnly,
		SameSite: ParseSameSite(config.SameSite),
	}

	// Set the expired cookie to clear it
	http.SetCookie(c.Response(), cookie)

	return nil
}
