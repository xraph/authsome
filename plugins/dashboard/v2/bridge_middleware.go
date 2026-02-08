package dashboard

import (
	"context"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

// BridgeContextMiddleware enriches the HTTP request context with user, app, and environment IDs
// before the bridge handler processes it. This eliminates the need for each bridge function
// to manually extract and build context, providing a single source of truth for context enrichment.
//
// The middleware extracts:
// 1. User ID from session cookie
// 2. App ID from session
// 3. Environment ID from cookie
//
// This middleware runs BEFORE bridge function execution, so all bridge functions
// automatically receive an enriched context via bridgeCtx.Context()
func (p *Plugin) BridgeContextMiddleware() http.Handler {
	// Get the original bridge HTTP handler
	originalHandler := p.fuiBridge.Handler()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Enrich context before bridge processes the request
		enrichedCtx := p.enrichBridgeContext(r)
		r = r.WithContext(enrichedCtx)

		// Call the original bridge handler with enriched context
		originalHandler.ServeHTTP(w, r)
	})
}

// enrichBridgeContext extracts authentication and context information from the HTTP request
// and enriches the Go context with user ID, app ID, and environment ID.
//
// This follows the same authentication flow as the RequireAuth middleware but is designed
// for bridge calls which use a different request flow.
func (p *Plugin) enrichBridgeContext(r *http.Request) context.Context {
	ctx := r.Context()

	// Log all cookies for debugging
	cookieNames := []string{}
	for _, c := range r.Cookies() {
		cookieNames = append(cookieNames, c.Name)
	}
	
	p.log.Debug("[BridgeMiddleware] Starting context enrichment", 
		forge.F("path", r.URL.Path),
		forge.F("method", r.Method),
		forge.F("cookies", cookieNames),
		forge.F("expectedSessionCookie", sessionCookieName))

	// Step 1: Extract and validate session from cookie
	sessionCookie, err := r.Cookie(sessionCookieName)
	if err != nil || sessionCookie == nil || sessionCookie.Value == "" {
		// No session - return context as-is (bridge function will handle auth check)
		p.log.Error("[BridgeMiddleware] No session cookie found - AUTHENTICATION WILL FAIL",
			forge.F("error", err),
			forge.F("availableCookies", cookieNames),
			forge.F("lookingFor", sessionCookieName))
		return ctx
	}

	sessionToken := sessionCookie.Value
	p.log.Debug("[BridgeMiddleware] Found session cookie", 
		forge.F("tokenLength", len(sessionToken)))

	// Validate session using session service
	sess, err := p.sessionSvc.FindByToken(ctx, sessionToken)
	if err != nil || sess == nil {
		// Invalid session - return context as-is
		p.log.Warn("[BridgeMiddleware] Invalid session token in bridge request", 
			forge.F("error", err),
			forge.F("hasSession", sess != nil))
		return ctx
	}

	p.log.Debug("[BridgeMiddleware] Session validated", 
		forge.F("userID", sess.UserID.String()),
		forge.F("appID", sess.AppID.String()))

	// Step 2: Set User ID in context
	if !sess.UserID.IsNil() {
		ctx = contexts.SetUserID(ctx, sess.UserID)
		p.log.Debug("[BridgeMiddleware] Enriched context with user ID", 
			forge.F("userId", sess.UserID.String()))
	} else {
		p.log.Warn("[BridgeMiddleware] Session has nil user ID")
	}

	// Step 3: Set App ID in context from session
	if !sess.AppID.IsNil() {
		ctx = contexts.SetAppID(ctx, sess.AppID)
		p.log.Debug("[BridgeMiddleware] Enriched context with app ID", 
			forge.F("appId", sess.AppID.String()))
	} else {
		p.log.Warn("[BridgeMiddleware] Session has nil app ID")
	}

	// Step 4: Extract Environment ID from cookie OR fall back to default
	envSet := false
	if envCookie, err := r.Cookie(environmentCookieName); err == nil && envCookie != nil && envCookie.Value != "" {
		if envID, err := xid.FromString(envCookie.Value); err == nil && !envID.IsNil() {
			ctx = contexts.SetEnvironmentID(ctx, envID)
			p.log.Debug("[BridgeMiddleware] Enriched context with environment ID from cookie", 
				forge.F("envId", envID.String()))
			envSet = true
		}
	}
	
	// If no env cookie, fall back to default environment for the app
	if !envSet && !sess.AppID.IsNil() && p.envSvc != nil {
		defaultEnv, err := p.envSvc.GetDefaultEnvironment(ctx, sess.AppID)
		if err == nil && defaultEnv != nil {
			ctx = contexts.SetEnvironmentID(ctx, defaultEnv.ID)
			p.log.Debug("[BridgeMiddleware] Using default environment for app", 
				forge.F("envId", defaultEnv.ID.String()),
				forge.F("appId", sess.AppID.String()))
		} else {
			p.log.Warn("[BridgeMiddleware] Failed to get default environment",
				forge.F("error", err),
				forge.F("appId", sess.AppID.String()))
		}
	}

	// Step 5: Store session in context for potential use by bridge functions
	ctx = context.WithValue(ctx, "session", sess)

	p.log.Debug("[BridgeMiddleware] Context enrichment complete")
	return ctx
}
