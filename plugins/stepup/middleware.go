package stepup

import (
	"context"
	"fmt"

	"github.com/xraph/forge"
)

// Middleware provides step-up authentication middleware
type Middleware struct {
	service *Service
	config  *Config
}

// NewMiddleware creates a new step-up middleware
func NewMiddleware(service *Service, config *Config) *Middleware {
	return &Middleware{
		service: service,
		config:  config,
	}
}

// RequireLevel returns middleware that enforces a specific security level
func (m *Middleware) RequireLevel(level SecurityLevel) forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(c forge.Context) error {
			// Extract user context from request
			userID, orgID, sessionID := m.extractUserContext(c)
			if userID == "" {
				return c.JSON(401, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			// Build evaluation context
			evalCtx := &EvaluationContext{
				UserID:    userID,
				OrgID:     orgID,
				SessionID: sessionID,
				Route:     c.Request().URL.Path,
				Method:    c.Request().Method,
				IP:        c.Request().RemoteAddr,
				UserAgent: c.Request().Header.Get("User-Agent"),
				DeviceID:  m.extractDeviceID(c),
			}

			// Evaluate if step-up is required
			result, err := m.service.EvaluateRequirement(c.Request().Context(), evalCtx)
			if err != nil {
				return c.JSON(500, map[string]interface{}{
					"error": "Failed to evaluate step-up requirement",
				})
			}

			// Check if the current level meets the required level
			if !m.meetsLevel(result.CurrentLevel, level) {
				return c.JSON(403, map[string]interface{}{
					"error":           "Step-up authentication required",
					"required_level":  level,
					"current_level":   result.CurrentLevel,
					"requirement_id":  result.RequirementID,
					"challenge_token": result.ChallengeToken,
					"allowed_methods": result.AllowedMethods,
					"expires_at":      result.ExpiresAt,
					"reason":          result.Reason,
				})
			}

			// Continue to next handler
			return next(c)
		}
	}
}

// RequireForRoute returns middleware that checks route-based rules
func (m *Middleware) RequireForRoute() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(c forge.Context) error {
			// Skip if not enabled
			if !m.config.Enabled {
				return next(c)
			}

			// Extract user context
			userID, orgID, sessionID := m.extractUserContext(c)
			if userID == "" {
				// Not authenticated, skip step-up check
				return next(c)
			}

			// Build evaluation context
			evalCtx := &EvaluationContext{
				UserID:    userID,
				OrgID:     orgID,
				SessionID: sessionID,
				Route:     c.Request().URL.Path,
				Method:    c.Request().Method,
				IP:        c.Request().RemoteAddr,
				UserAgent: c.Request().Header.Get("User-Agent"),
				DeviceID:  m.extractDeviceID(c),
			}

			// Evaluate requirement
			result, err := m.service.EvaluateRequirement(c.Request().Context(), evalCtx)
			if err != nil {
				// Log error but don't block request
				return next(c)
			}

			if result.Required {
				return c.JSON(403, map[string]interface{}{
					"error":           "Step-up authentication required",
					"security_level":  result.SecurityLevel,
					"current_level":   result.CurrentLevel,
					"requirement_id":  result.RequirementID,
					"challenge_token": result.ChallengeToken,
					"allowed_methods": result.AllowedMethods,
					"expires_at":      result.ExpiresAt,
					"reason":          result.Reason,
					"matched_rules":   result.MatchedRules,
					"can_remember":    result.CanRemember,
				})
			}

			// Store result in context for handlers
			ctx := context.WithValue(c.Request().Context(), "stepup_result", result)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// RequireForAmount returns middleware that checks amount-based rules
func (m *Middleware) RequireForAmount(amount float64, currency string) forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(c forge.Context) error {
			userID, orgID, sessionID := m.extractUserContext(c)
			if userID == "" {
				return c.JSON(401, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			evalCtx := &EvaluationContext{
				UserID:    userID,
				OrgID:     orgID,
				SessionID: sessionID,
				Route:     c.Request().URL.Path,
				Method:    c.Request().Method,
				Amount:    amount,
				Currency:  currency,
				IP:        c.Request().RemoteAddr,
				UserAgent: c.Request().Header.Get("User-Agent"),
				DeviceID:  m.extractDeviceID(c),
			}

			result, err := m.service.EvaluateRequirement(c.Request().Context(), evalCtx)
			if err != nil {
				return c.JSON(500, map[string]interface{}{
					"error": "Failed to evaluate step-up requirement",
				})
			}

			if result.Required {
				return c.JSON(403, map[string]interface{}{
					"error":           "Step-up authentication required",
					"security_level":  result.SecurityLevel,
					"current_level":   result.CurrentLevel,
					"requirement_id":  result.RequirementID,
					"challenge_token": result.ChallengeToken,
					"allowed_methods": result.AllowedMethods,
					"expires_at":      result.ExpiresAt,
					"reason":          result.Reason,
					"amount":          amount,
					"currency":        currency,
				})
			}

			return next(c)
		}
	}
}

// RequireForResource returns middleware that checks resource-based rules
func (m *Middleware) RequireForResource(resourceType, action string) forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(c forge.Context) error {
			userID, orgID, sessionID := m.extractUserContext(c)
			if userID == "" {
				return c.JSON(401, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			evalCtx := &EvaluationContext{
				UserID:       userID,
				OrgID:        orgID,
				SessionID:    sessionID,
				Route:        c.Request().URL.Path,
				Method:       c.Request().Method,
				ResourceType: resourceType,
				Action:       action,
				IP:           c.Request().RemoteAddr,
				UserAgent:    c.Request().Header.Get("User-Agent"),
				DeviceID:     m.extractDeviceID(c),
			}

			result, err := m.service.EvaluateRequirement(c.Request().Context(), evalCtx)
			if err != nil {
				return c.JSON(500, map[string]interface{}{
					"error": "Failed to evaluate step-up requirement",
				})
			}

			if result.Required {
				return c.JSON(403, map[string]interface{}{
					"error":           "Step-up authentication required",
					"security_level":  result.SecurityLevel,
					"current_level":   result.CurrentLevel,
					"requirement_id":  result.RequirementID,
					"challenge_token": result.ChallengeToken,
					"allowed_methods": result.AllowedMethods,
					"expires_at":      result.ExpiresAt,
					"reason":          result.Reason,
					"resource_type":   resourceType,
					"action":          action,
				})
			}

			return next(c)
		}
	}
}

// Helper methods

func (m *Middleware) extractUserContext(c forge.Context) (userID, orgID, sessionID string) {
	// Try to extract from session
	if session := c.Get("session"); session != nil {
		if sessionMap, ok := session.(map[string]interface{}); ok {
			if uid, ok := sessionMap["user_id"].(string); ok {
				userID = uid
			}
			if oid, ok := sessionMap["org_id"].(string); ok {
				orgID = oid
			}
			if sid, ok := sessionMap["id"].(string); ok {
				sessionID = sid
			}
		}
	}

	// Try to extract from context
	if userID == "" {
		if uid := c.Get("user_id"); uid != nil {
			if s, ok := uid.(string); ok {
				userID = s
			}
		}
	}

	if orgID == "" {
		if oid := c.Get("org_id"); oid != nil {
			if s, ok := oid.(string); ok {
				orgID = s
			}
		}
	}

	// Try to extract from headers (for API keys)
	if userID == "" {
		userID = c.Request().Header.Get("X-User-ID")
	}
	if orgID == "" {
		orgID = c.Request().Header.Get("X-Org-ID")
	}

	return userID, orgID, sessionID
}

func (m *Middleware) extractDeviceID(c forge.Context) string {
	// Check cookie first
	if cookie, err := c.Request().Cookie("device_id"); err == nil {
		return cookie.Value
	}

	// Check header
	if deviceID := c.Request().Header.Get("X-Device-ID"); deviceID != "" {
		return deviceID
	}

	// Check context
	if deviceID := c.Get("device_id"); deviceID != nil {
		if s, ok := deviceID.(string); ok {
			return s
		}
	}

	return ""
}

func (m *Middleware) meetsLevel(current, required SecurityLevel) bool {
	levels := map[SecurityLevel]int{
		SecurityLevelNone:     0,
		SecurityLevelLow:      1,
		SecurityLevelMedium:   2,
		SecurityLevelHigh:     3,
		SecurityLevelCritical: 4,
	}
	return levels[current] >= levels[required]
}

// EvaluateMiddleware evaluates but doesn't block - stores result in context
func (m *Middleware) EvaluateMiddleware() forge.MiddlewareFunc {
	return func(next forge.HandlerFunc) forge.HandlerFunc {
		return func(c forge.Context) error {
			userID, orgID, sessionID := m.extractUserContext(c)
			
			// Only evaluate if user is authenticated
			if userID != "" {
				evalCtx := &EvaluationContext{
					UserID:    userID,
					OrgID:     orgID,
					SessionID: sessionID,
					Route:     c.Request().URL.Path,
					Method:    c.Request().Method,
					IP:        c.Request().RemoteAddr,
					UserAgent: c.Request().Header.Get("User-Agent"),
					DeviceID:  m.extractDeviceID(c),
				}

				if result, err := m.service.EvaluateRequirement(c.Request().Context(), evalCtx); err == nil {
					// Store in context for handlers
					ctx := context.WithValue(c.Request().Context(), "stepup_evaluation", result)
					c.SetRequest(c.Request().WithContext(ctx))
					
					// Also set as context value for easy access
					c.Set("stepup_evaluation", result)
				}
			}

			return next(c)
		}
	}
}

// GetEvaluationFromContext extracts the step-up evaluation result from context
func GetEvaluationFromContext(c forge.Context) (*EvaluationResult, error) {
	if result := c.Get("stepup_evaluation"); result != nil {
		if evalResult, ok := result.(*EvaluationResult); ok {
			return evalResult, nil
		}
	}
	return nil, fmt.Errorf("no step-up evaluation found in context")
}

