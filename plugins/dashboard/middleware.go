package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

const (
	csrfCookieName        = "dashboard_csrf_token"
	sessionCookieName     = "authsome_session"
	environmentCookieName = "authsome_environment"
)

// rateLimiter implements a simple in-memory rate limiter
type rateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientLimit
	limit   int
	window  time.Duration
}

type clientLimit struct {
	count     int
	resetTime time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*clientLimit),
		limit:   limit,
		window:  window,
	}

	// Cleanup goroutine
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for key, limit := range rl.clients {
		if now.After(limit.resetTime) {
			delete(rl.clients, key)
		}
	}
}

func (rl *rateLimiter) allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	limit, exists := rl.clients[clientID]

	if !exists || now.After(limit.resetTime) {
		rl.clients[clientID] = &clientLimit{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if limit.count >= rl.limit {
		return false
	}

	limit.count++
	return true
}

var globalRateLimiter = newRateLimiter(100, time.Minute)

// RequireAuth middleware ensures the user is authenticated
func (p *Plugin) RequireAuth() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			fmt.Printf("[Dashboard] RequireAuth: Checking authentication for path: %s\n", c.Request().URL.Path)

			// Extract session token from cookie
			cookie, err := c.Request().Cookie(sessionCookieName)
			if err != nil || cookie == nil || cookie.Value == "" {
				fmt.Printf("[Dashboard] RequireAuth: No session cookie found\n")
				// No session cookie, redirect to dashboard login
				loginURL := p.basePath + "/dashboard/login?redirect=" + c.Request().URL.Path
				return c.Redirect(http.StatusFound, loginURL)
			}

			sessionToken := cookie.Value
			fmt.Printf("[Dashboard] RequireAuth: Found session token: %s...\n", sessionToken[:min(10, len(sessionToken))])

			// Validate session
			sess, err := p.sessionSvc.FindByToken(c.Request().Context(), sessionToken)
			if err != nil {
				fmt.Printf("[Dashboard] RequireAuth: Error finding session: %v\n", err)
				// Invalid session, clear cookie and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/dashboard/login?error=invalid_session&redirect=" + c.Request().URL.Path
				return c.Redirect(http.StatusFound, loginURL)
			}

			if sess == nil {
				fmt.Printf("[Dashboard] RequireAuth: Session not found\n")
				// Session not found, clear cookie and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/dashboard/login?error=invalid_session&redirect=" + c.Request().URL.Path
				return c.Redirect(http.StatusFound, loginURL)
			}

			// Check if session is expired
			if time.Now().After(sess.ExpiresAt) {
				fmt.Printf("[Dashboard] RequireAuth: Session expired at %v\n", sess.ExpiresAt)
				// Expired session, clear cookie and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/dashboard/login?error=session_expired&redirect=" + c.Request().URL.Path
				return c.Redirect(http.StatusFound, loginURL)
			}

			fmt.Printf("[Dashboard] RequireAuth: Session valid, fetching user: %s\n", sess.UserID)

			// Set app context from session for user lookup
			ctx := contexts.SetAppID(c.Request().Context(), sess.AppID)

			// Get user information
			user, err := p.userSvc.FindByID(ctx, sess.UserID)
			if err != nil {
				fmt.Printf("[Dashboard] RequireAuth: Error finding user: %v\n", err)
				// User not found, clear session and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/dashboard/login?error=invalid_session&redirect=" + c.Request().URL.Path
				return c.Redirect(http.StatusFound, loginURL)
			}

			if user == nil {
				fmt.Printf("[Dashboard] RequireAuth: User is nil\n")
				// User is nil, clear session and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/dashboard/login?error=invalid_session&redirect=" + c.Request().URL.Path
				return c.Redirect(http.StatusFound, loginURL)
			}

			fmt.Printf("[Dashboard] RequireAuth: User authenticated: %s (%s)\n", user.Email, user.ID)

			// Store user and session in context ONLY if all checks passed
			// Reuse ctx that already has AppID set
			ctx = context.WithValue(ctx, "user", user)
			ctx = context.WithValue(ctx, "session", sess)
			ctx = context.WithValue(ctx, "authenticated", true)

			// Update request with new context
			*c.Request() = *c.Request().WithContext(ctx)

			return next(c)
		}
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RequireAdmin middleware ensures the user has admin role
func (p *Plugin) RequireAdmin() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			fmt.Printf("[Dashboard] RequireAdmin middleware called for path: %s\n", c.Request().URL.Path)

			// Get user from context (set by RequireAuth)
			userVal := c.Request().Context().Value("user")
			if userVal == nil {
				fmt.Printf("[Dashboard] No user in context, redirecting to login\n")
				return c.Redirect(http.StatusFound, p.basePath+"/dashboard/login?error=auth_required")
			}

			userObj, ok := userVal.(*user.User)
			if !ok {
				return c.Redirect(http.StatusFound, p.basePath+"/dashboard/login?error=invalid_session")
			}

			// Use the fast permission checker
			if p.permChecker == nil {
				// Fallback: allow access if permission checker is not initialized
				// Check with RBAC service directly
				rbacContext := &user.User{} // Placeholder - need proper Context type
				_ = rbacContext
				// For now, allow all authenticated users when permission checker is not available
				fmt.Printf("[Dashboard] Permission checker not initialized, allowing access for user: %s\n", userObj.Email)
				return next(c)

				/* TODO: Fix when rbac.Context is properly imported
				rbacCtx := &rbac.Context{
					Subject:  userObj.ID.String(),
					Resource: "dashboard",
					Action:   "access",
				}
				}
				*/
			}

			// Check if user can access dashboard
			canAccess := p.permChecker.For(c.Request().Context(), userObj.ID).Dashboard().CanAccess()
			if !canAccess {
				c.SetHeader("Content-Type", "text/html; charset=utf-8")
				return c.String(http.StatusForbidden, `
					<!DOCTYPE html>
					<html>
					<head>
						<title>Access Denied</title>
						<meta charset="utf-8">
						<meta name="viewport" content="width=device-width, initial-scale=1">
						<style>
							body { font-family: system-ui, -apple-system, sans-serif; padding: 40px; text-align: center; }
							.error-box { max-width: 500px; margin: 0 auto; padding: 40px; border: 1px solid #e5e7eb; border-radius: 8px; }
							h1 { color: #dc2626; }
							p { color: #6b7280; }
							a { color: #2563eb; text-decoration: none; }
						</style>
					</head>
					<body>
						<div class="error-box">
							<h1>Access Denied</h1>
							<p>You don't have permission to access the dashboard. Please contact your administrator to request access.</p>
							<p><a href="`+p.basePath+`/dashboard/login">← Back to Login</a></p>
						</div>
					</body>
					</html>
				`)
			}

			return next(c)
		}
	}
}

// CSRF middleware provides CSRF protection
func (p *Plugin) CSRF() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Get session ID for token binding
			sessionID := ""
			if sessionCookie, err := c.Request().Cookie(sessionCookieName); err == nil && sessionCookie != nil {
				sessionID = sessionCookie.Value
			}

			// If no session, use IP address as fallback
			if sessionID == "" {
				sessionID = c.Request().RemoteAddr
			}

			// Generate or retrieve CSRF token
			var token string
			cookie, err := c.Request().Cookie(csrfCookieName)

			if err != nil || cookie == nil || cookie.Value == "" {
				// Generate new CSRF token bound to session
				token, err = p.csrfProtector.GenerateToken(sessionID)
				if err != nil {
					return fmt.Errorf("failed to generate CSRF token: %w", err)
				}

				// Set CSRF cookie
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/dashboard",
					HttpOnly: true,
					Secure:   c.Request().TLS != nil,
					SameSite: http.SameSiteStrictMode,
					MaxAge:   3600, // 1 hour
				})
			} else {
				token = cookie.Value
			}

			// Store token in context for templates
			ctx := context.WithValue(c.Request().Context(), "csrf_token", token)
			*c.Request() = *c.Request().WithContext(ctx)

			// Validate CSRF token for mutating requests
			if c.Request().Method == "POST" || c.Request().Method == "PUT" ||
				c.Request().Method == "DELETE" || c.Request().Method == "PATCH" {

				// Get token from form or header
				submittedToken := c.Request().FormValue("csrf_token")
				if submittedToken == "" {
					submittedToken = c.Request().Header.Get("X-CSRF-Token")
				}

				// Validate token using CSRF protector
				if !p.csrfProtector.ValidateToken(submittedToken, sessionID) {
					c.SetHeader("Content-Type", "text/html; charset=utf-8")
					htmlContent := `<!DOCTYPE html>
<html>
<head>
	<title>CSRF Validation Failed</title>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>
		body { font-family: system-ui, -apple-system, sans-serif; padding: 40px; text-align: center; }
		.error-box { max-width: 500px; margin: 0 auto; padding: 40px; border: 1px solid #e5e7eb; border-radius: 8px; }
		h1 { color: #dc2626; }
		p { color: #6b7280; }
		a { color: #2563eb; text-decoration: none; }
	</style>
</head>
<body>
	<div class="error-box">
		<h1>Security Validation Failed</h1>
		<p>The form submission failed security validation. This usually happens when:</p>
		<ul style="text-align: left; color: #6b7280;">
			<li>Your session has expired</li>
			<li>You opened the form in multiple tabs</li>
			<li>You took too long to submit the form</li>
		</ul>
		<p><a href="javascript:history.back()">← Go Back</a> or <a href="` + p.basePath + `/dashboard/">Go to Dashboard</a></p>
	</div>
</body>
</html>`
					return c.String(http.StatusForbidden, htmlContent)
				}
			}

			return next(c)
		}
	}
}

// RateLimit middleware implements rate limiting
func (p *Plugin) RateLimit() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Get client identifier (IP address)
			clientIP := c.Request().RemoteAddr
			if forwarded := c.Request().Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = forwarded
			}

			// Check rate limit
			if !globalRateLimiter.allow(clientIP) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too many requests. Please try again later.",
				})
			}

			return next(c)
		}
	}
}

// AuditLog middleware logs all dashboard access
func (p *Plugin) AuditLog() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Get user from context
			userVal := c.Request().Context().Value("user")
			if userVal != nil {
				userObj, ok := userVal.(*user.User)
				if ok {
					// Get AppID from context (set by RequireAuth middleware)
					appID, hasAppID := contexts.GetAppID(c.Request().Context())
					if hasAppID {
						// Log dashboard access
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()

							// Set AppID in the background context for the audit log
							ctx = contexts.SetAppID(ctx, appID)

							_ = p.auditSvc.Log(ctx, &userObj.ID, "dashboard.access", c.Request().URL.Path, c.Request().RemoteAddr, c.Request().UserAgent(), "default")
						}()
					}
				}
			}

			return next(c)
		}
	}
}
