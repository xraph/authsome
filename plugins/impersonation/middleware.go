package impersonation

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/impersonation"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// ImpersonationContextKey is the context key for impersonation data.
	ImpersonationContextKey ContextKey = "impersonation"
	// ImpersonationActionContextKey is the context key for impersonation action.
	ImpersonationActionContextKey ContextKey = "impersonation_action"
)

// ImpersonationContext holds impersonation data in the request context.
type ImpersonationContext struct {
	IsImpersonating bool    `json:"is_impersonating"`
	ImpersonationID *xid.ID `json:"impersonation_id,omitempty"`
	ImpersonatorID  *xid.ID `json:"impersonator_id,omitempty"`
	TargetUserID    *xid.ID `json:"target_user_id,omitempty"`
	IndicatorMsg    string  `json:"indicator_message,omitempty"`
}

// ImpersonationMiddleware adds impersonation context to requests.
type ImpersonationMiddleware struct {
	service *impersonation.Service
	config  Config
}

// NewMiddleware creates a new impersonation middleware.
func NewMiddleware(service *impersonation.Service, config Config) *ImpersonationMiddleware {
	return &ImpersonationMiddleware{
		service: service,
		config:  config,
	}
}

// Handle checks if the current session is an impersonation session and adds context.
func (m *ImpersonationMiddleware) Handle() func(forge.Context) error {
	return func(c forge.Context) error {
		// Try to get session ID from context or cookie
		sessionID := m.getSessionID(c)
		if sessionID == nil {
			// No session, continue without impersonation context
			return nil
		}

		// Check if this session is an impersonation session
		verifyReq := &impersonation.VerifyRequest{
			SessionID: *sessionID,
		}

		verifyResp, err := m.service.Verify(c.Request().Context(), verifyReq)
		if err != nil {
			// Error checking, continue without impersonation context
			return nil
		}

		// Add impersonation context to request
		impCtx := &ImpersonationContext{
			IsImpersonating: verifyResp.IsImpersonating,
			ImpersonationID: verifyResp.ImpersonationID,
			ImpersonatorID:  verifyResp.ImpersonatorID,
			TargetUserID:    verifyResp.TargetUserID,
		}

		if verifyResp.IsImpersonating && m.config.ShowIndicator {
			impCtx.IndicatorMsg = m.config.IndicatorMessage
		}

		// Store in context
		ctx := context.WithValue(c.Request().Context(), ImpersonationContextKey, impCtx)
		*c.Request() = *c.Request().WithContext(ctx)

		// Add response header if impersonating (for UI to show banner)
		if verifyResp.IsImpersonating && m.config.ShowIndicator {
			c.Response().Header().Set("X-Impersonating", "true")
			c.Response().Header().Set("X-Impersonator-Id", verifyResp.ImpersonatorID.String())
			c.Response().Header().Set("X-Target-User-Id", verifyResp.TargetUserID.String())
		}

		return nil
	}
}

// RequireNoImpersonation ensures the request is NOT from an impersonation session
// Useful for sensitive operations that should not be allowed during impersonation.
func (m *ImpersonationMiddleware) RequireNoImpersonation() func(forge.Context) error {
	return func(c forge.Context) error {
		impCtx := GetImpersonationContext(c)
		if impCtx != nil && impCtx.IsImpersonating {
			return c.JSON(403, errs.New("IMPERSONATION_NOT_ALLOWED",
				"This action is not allowed during impersonation",
				403))
		}

		return nil
	}
}

// RequireImpersonation ensures the request IS from an impersonation session
// Useful for impersonation-specific endpoints.
func (m *ImpersonationMiddleware) RequireImpersonation() func(forge.Context) error {
	return func(c forge.Context) error {
		impCtx := GetImpersonationContext(c)
		if impCtx == nil || !impCtx.IsImpersonating {
			return c.JSON(403, errs.New("IMPERSONATION_REQUIRED",
				"This action requires an active impersonation session",
				403))
		}

		return nil
	}
}

// AuditImpersonationAction logs all actions during impersonation if enabled.
func (m *ImpersonationMiddleware) AuditImpersonationAction() func(forge.Context) error {
	return func(c forge.Context) error {
		if !m.config.AuditAllActions {
			return nil
		}

		impCtx := GetImpersonationContext(c)
		if impCtx == nil || !impCtx.IsImpersonating {
			return nil
		}

		// Log the action to impersonation audit trail
		// This would typically be done after the handler completes
		// For now, we'll just add it to the context for the handler to use
		action := fmt.Sprintf("%s %s", c.Request().Method, c.Request().URL.Path)

		// Store action for later auditing
		ctx := context.WithValue(c.Request().Context(), ImpersonationActionContextKey, action)
		*c.Request() = *c.Request().WithContext(ctx)

		return nil
	}
}

// Helper functions

// getSessionID extracts session ID from the request.
func (m *ImpersonationMiddleware) getSessionID(c forge.Context) *xid.ID {
	// Try to get from Authorization header (Bearer token)
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" {
		// Parse Bearer token
		// This is a simplified version - you'd need to parse the actual token
		// and extract the session ID
		// For now, we'll skip this and rely on cookies
	}

	// Try to get from cookie
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return nil
	}

	// Parse session ID from cookie value
	// The actual implementation depends on your session token format
	// For now, we'll try to parse it as an xid
	sessionID, err := xid.FromString(cookie.Value)
	if err != nil {
		return nil
	}

	return &sessionID
}

// GetImpersonationContext retrieves impersonation context from request context.
func GetImpersonationContext(c forge.Context) *ImpersonationContext {
	val := c.Request().Context().Value(ImpersonationContextKey)
	if val == nil {
		return nil
	}

	impCtx, ok := val.(*ImpersonationContext)
	if !ok {
		return nil
	}

	return impCtx
}

// IsImpersonating checks if the current request is from an impersonation session.
func IsImpersonating(c forge.Context) bool {
	impCtx := GetImpersonationContext(c)

	return impCtx != nil && impCtx.IsImpersonating
}

// GetImpersonatorID returns the impersonator's user ID if impersonating.
func GetImpersonatorID(c forge.Context) *xid.ID {
	impCtx := GetImpersonationContext(c)
	if impCtx == nil || !impCtx.IsImpersonating {
		return nil
	}

	return impCtx.ImpersonatorID
}

// GetTargetUserID returns the target user's ID if impersonating.
func GetTargetUserID(c forge.Context) *xid.ID {
	impCtx := GetImpersonationContext(c)
	if impCtx == nil || !impCtx.IsImpersonating {
		return nil
	}

	return impCtx.TargetUserID
}
