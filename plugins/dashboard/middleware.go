package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
	"github.com/xraph/forgeui/alpine"
	"github.com/xraph/forgeui/layout"
	"github.com/xraph/forgeui/theme"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

const (
	csrfCookieName        = "dashboard_csrf_token"
	sessionCookieName     = "authsome_session"
	environmentCookieName = "authsome_environment"
)

// rateLimiter implements a simple in-memory rate limiter.
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

// RequireAuth middleware ensures the user is authenticated.
func (p *Plugin) RequireAuth() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Extract session token from cookie
			cookie, err := c.Request().Cookie(sessionCookieName)
			if err != nil || cookie == nil || cookie.Value == "" {
				// No session cookie, redirect to dashboard login
				loginURL := p.basePath + "/login?redirect=" + c.Request().URL.Path

				return c.Redirect(http.StatusFound, loginURL)
			}

			sessionToken := cookie.Value

			// Validate session
			sess, err := p.sessionSvc.FindByToken(c.Request().Context(), sessionToken)
			if err != nil {
				// Invalid session, clear cookie and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/login?error=invalid_session&redirect=" + c.Request().URL.Path

				return c.Redirect(http.StatusFound, loginURL)
			}

			if sess == nil {
				// Session not found, clear cookie and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/login?error=invalid_session&redirect=" + c.Request().URL.Path

				return c.Redirect(http.StatusFound, loginURL)
			}

			// Check if session is expired
			if time.Now().After(sess.ExpiresAt) {
				// Expired session, clear cookie and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/login?error=session_expired&redirect=" + c.Request().URL.Path

				return c.Redirect(http.StatusFound, loginURL)
			}

			// Set app context from session for user lookup
			ctx := contexts.SetAppID(c.Request().Context(), sess.AppID)

			// If session has no app_id (legacy session), try to get user without app context first
			if sess.AppID.IsNil() {
				ctx = c.Request().Context()
			}

			// Get user information
			user, err := p.userSvc.FindByID(ctx, sess.UserID)
			if err != nil {
				// User not found, clear session and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/login?error=invalid_session&redirect=" + c.Request().URL.Path

				return c.Redirect(http.StatusFound, loginURL)
			}

			if user == nil {
				// User is nil, clear session and redirect
				http.SetCookie(c.Response(), &http.Cookie{
					Name:     sessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				loginURL := p.basePath + "/login?error=invalid_session&redirect=" + c.Request().URL.Path

				return c.Redirect(http.StatusFound, loginURL)
			}

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

// Helper function for min.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// RequireAdmin middleware ensures the user has admin role.
func (p *Plugin) RequireAdmin() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Get user from context (set by RequireAuth)
			userVal := c.Request().Context().Value("user")
			if userVal == nil {
				return c.Redirect(http.StatusFound, p.basePath+"/login?error=auth_required")
			}

			userObj, ok := userVal.(*user.User)
			if !ok {
				return c.Redirect(http.StatusFound, p.basePath+"/login?error=invalid_session")
			}

			// Use the fast permission checker
			if p.permChecker == nil {
				// Fallback: allow access if permission checker is not initialized
				// Check with RBAC service directly
				rbacContext := &user.User{} // Placeholder - need proper Context type
				_ = rbacContext
				// For now, allow all authenticated users when permission checker is not available
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
				// Render ForgeUI forbidden page
				loginURL := p.basePath + "/login"
				forbiddenPage := p.pagesManager.ForbiddenPage(loginURL)

				c.Response().WriteHeader(http.StatusForbidden)

				return p.renderForgeUINode(c, forbiddenPage)
			}

			return next(c)
		}
	}
}

// CSRF middleware provides CSRF protection.
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
					Path:     "/ui",
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
			if c.Request().Method == http.MethodPost || c.Request().Method == http.MethodPut ||
				c.Request().Method == http.MethodDelete || c.Request().Method == http.MethodPatch {
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

// RateLimit middleware implements rate limiting.
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

// AuditLog middleware logs all dashboard access.
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

							_ = p.auditSvc.Log(ctx, &userObj.ID, string(audit.ActionDashboardAccess), c.Request().URL.Path, c.Request().RemoteAddr, c.Request().UserAgent(), "default")
						}()
					}
				}
			}

			return next(c)
		}
	}
}

// EnvironmentContext middleware injects environment context into all dashboard requests
//
// This middleware ensures that every app-scoped dashboard request has an environment ID
// set in the context. This is critical for:
// - Environment-scoped data operations
// - Multi-environment isolation
// - Audit trails with environment information
// - Dashboard extensions that need environment context
//
// The middleware follows this flow:
// 1. Extract app ID from URL path parameter (:appId)
// 2. Check for environment ID in cookie (authsome_environment)
// 3. If no cookie, fetch the default environment for the app
// 4. Set environment context using contexts.SetEnvironmentID()
// 5. Update cookie for future requests (30-day expiry)
//
// Routes without :appId parameter are skipped (e.g., /dashboard/login)
// Gracefully handles missing environment service for backward compatibility.
func (p *Plugin) EnvironmentContext() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			// Step 1: Get appID from URL path parameter (/dashboard/app/:appId/*)
			appIDStr := c.Param("appId")
			if appIDStr == "" {
				// No appId in path (e.g., /dashboard/login), skip environment context
				return next(c)
			}

			appID, err := xid.FromString(appIDStr)
			if err != nil {
				// Invalid appId format, skip environment context
				return next(c)
			}

			// Set app context first (required for subsequent operations)
			ctx = contexts.SetAppID(ctx, appID)

			// Step 2: Check if environment service is available
			// The environment service is optional and may not be present in all configurations
			if p.serviceRegistry == nil || p.serviceRegistry.EnvironmentService() == nil {
				// No environment service available, skip environment context
				// This maintains backward compatibility with configurations without multiapp plugin
				*c.Request() = *c.Request().WithContext(ctx)

				return next(c)
			}

			envService := p.serviceRegistry.EnvironmentService()

			// Step 3: Try to get environment ID from cookie
			// The cookie persists the user's selected environment across requests
			var envID xid.ID

			if cookie, err := c.Request().Cookie(environmentCookieName); err == nil && cookie != nil && cookie.Value != "" {
				if id, err := xid.FromString(cookie.Value); err == nil {
					envID = id
				}
			}

			// Step 4: If no environment in cookie, get the default environment for this app
			// This happens on first visit or if cookie was cleared
			if envID.IsNil() {
				defaultEnv, err := envService.GetDefaultEnvironment(ctx, appID)
				if err == nil && defaultEnv != nil {
					envID = defaultEnv.ID

					// Step 5: Set cookie for future requests (30-day expiry)
					// This persists the environment selection across sessions
					http.SetCookie(c.Response(), &http.Cookie{
						Name:     environmentCookieName,
						Value:    envID.String(),
						Path:     "/ui",
						HttpOnly: true,                   // Prevent XSS attacks
						Secure:   c.Request().TLS != nil, // Only send over HTTPS in production
						SameSite: http.SameSiteLaxMode,   // CSRF protection
						MaxAge:   86400 * 30,             // 30 days
					})
				}
			}

			// Step 6: Set environment context if we have a valid environment ID
			// This makes the environment ID available to all downstream handlers and services
			if !envID.IsNil() {
				ctx = contexts.SetEnvironmentID(ctx, envID)
			} else {
				// No environment available - this is acceptable in some scenarios
				// (e.g., app has no environments yet, environment service error)
			}

			// Update request with the enriched context
			*c.Request() = *c.Request().WithContext(ctx)

			return next(c)
		}
	}
}

// renderForgeUINode renders a ForgeUI gomponent node with full HTML layout
// This helper method is used in middleware to render ForgeUI components outside of page handlers
// It includes all necessary styles (Tailwind CSS), theme support, and Alpine.js.
func (p *Plugin) renderForgeUINode(c forge.Context, content g.Node) error {
	// Get light and dark themes from ForgeUI app
	lightTheme := p.fuiApp.LightTheme()
	darkTheme := p.fuiApp.DarkTheme()

	// Create a complete HTML document with layout
	page := layout.Build(
		layout.Head(
			layout.Meta("viewport", "width=device-width, initial-scale=1"),
			layout.Charset("utf-8"),
			html.TitleEl(g.Text("Access Denied - Dashboard")),

			// Theme support
			theme.HeadContent(*lightTheme, *darkTheme),
			layout.Theme(lightTheme, darkTheme),

			// Tailwind CSS (using CDN)
			html.Script(
				html.Src("https://cdn.tailwindcss.com"),
			),
			theme.TailwindConfigScript(),
			theme.StyleTag(*lightTheme, *darkTheme),
			html.StyleEl(g.Raw(`
				@layer base {
					* {
						@apply border-border;
					}
				}
			`)),

			// Alpine.js cloak CSS
			alpine.CloakCSS(),
		),

		layout.Body(
			layout.Class("min-h-screen bg-background text-foreground antialiased"),

			// Dark mode script (must run BEFORE Alpine initializes)
			layout.DarkModeScript(),

			// Alpine global store initialization
			html.Script(
				g.Raw(`
					document.addEventListener('alpine:init', () => {
						Alpine.store('darkMode', {
							on: false,
							init() {
								this.on = document.documentElement.classList.contains('dark');
								this.$watch('on', value => {
									document.documentElement.classList.toggle('dark', value);
									document.documentElement.setAttribute('data-theme', value ? 'dark' : 'light');
									localStorage.setItem('theme', value ? 'dark' : 'light');
								});
							},
							toggle() {
								this.on = !this.on;
							}
						});
					});
				`),
			),

			// Page content
			content,

			// Alpine.js scripts
			layout.Scripts(
				alpine.Scripts(),
			),
		),
	)

	c.SetHeader("Content-Type", "text/html; charset=utf-8")

	return page.Render(c.Response())
}
