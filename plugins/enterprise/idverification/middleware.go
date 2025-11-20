package idverification

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Context keys for verification
type verificationContextKey string

const (
	VerificationStatusContextKey verificationContextKey = "verification_status"
	VerificationLevelContextKey  verificationContextKey = "verification_level"
)

// Middleware handles identity verification checks
type Middleware struct {
	service *Service
}

// NewMiddleware creates a new identity verification middleware
func NewMiddleware(service *Service) *Middleware {
	return &Middleware{
		service: service,
	}
}

// LoadVerificationStatus loads the user's verification status into context
// This middleware is non-blocking - it will set context values if found,
// but will not reject requests (use RequireVerified for that)
func (m *Middleware) LoadVerificationStatus(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		ctx := c.Request().Context()

		// Get user ID from context (set by auth middleware)
		userID, ok := contexts.GetUserID(ctx)
		if !ok || userID.IsNil() {
			// No user in context, continue without verification status
			return next(c)
		}

		// Get app and org IDs
		appID, ok := contexts.GetAppID(ctx)
		if !ok || appID.IsNil() {
			return next(c)
		}

		orgID, ok := contexts.GetOrganizationID(ctx)
		if !ok || orgID.IsNil() {
			return next(c)
		}

		// Load verification status
		status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
		if err != nil {
			// Status not found, continue without it
			return next(c)
		}

		// Inject verification status into context
		ctx = context.WithValue(ctx, VerificationStatusContextKey, status)
		ctx = context.WithValue(ctx, VerificationLevelContextKey, status.VerificationLevel)

		// Update request with new context
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// RequireVerified enforces that the user must be verified
func (m *Middleware) RequireVerified() func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "app context required",
					"code":  "APP_CONTEXT_REQUIRED",
				})
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "organization context required",
					"code":  "ORGANIZATION_CONTEXT_REQUIRED",
				})
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "verification status not found",
					"code":  "VERIFICATION_NOT_FOUND",
				})
			}

			if !status.IsVerified {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":              "identity verification required",
					"code":               "VERIFICATION_REQUIRED",
					"verification_level": status.VerificationLevel,
					"required_level":     "full",
				})
			}

			// Check if blocked
			if status.IsBlocked {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":  "user blocked from verification",
					"code":   "USER_BLOCKED",
					"reason": status.BlockReason,
				})
			}

			// Check if requires re-verification
			if status.RequiresReverification {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "re-verification required",
					"code":  "REVERIFICATION_REQUIRED",
				})
			}

			return next(c)
		}
	}
}

// RequireVerificationLevel enforces a specific verification level
// Levels: none, basic, enhanced, full
func (m *Middleware) RequireVerificationLevel(level string) func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "app context required",
					"code":  "APP_CONTEXT_REQUIRED",
				})
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "organization context required",
					"code":  "ORGANIZATION_CONTEXT_REQUIRED",
				})
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "verification status not found",
					"code":  "VERIFICATION_NOT_FOUND",
				})
			}

			// Check if user's level meets the requirement
			if !meetsVerificationLevel(status.VerificationLevel, level) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":          fmt.Sprintf("verification level '%s' required", level),
					"code":           "INSUFFICIENT_VERIFICATION_LEVEL",
					"current_level":  status.VerificationLevel,
					"required_level": level,
				})
			}

			// Check if blocked
			if status.IsBlocked {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":  "user blocked from verification",
					"code":   "USER_BLOCKED",
					"reason": status.BlockReason,
				})
			}

			return next(c)
		}
	}
}

// RequireDocumentVerified enforces that document verification is complete
func (m *Middleware) RequireDocumentVerified() func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "app context required",
					"code":  "APP_CONTEXT_REQUIRED",
				})
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "organization context required",
					"code":  "ORGANIZATION_CONTEXT_REQUIRED",
				})
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "verification status not found",
					"code":  "VERIFICATION_NOT_FOUND",
				})
			}

			if !status.DocumentVerified {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "document verification required",
					"code":  "DOCUMENT_VERIFICATION_REQUIRED",
				})
			}

			return next(c)
		}
	}
}

// RequireLivenessVerified enforces that liveness detection is complete
func (m *Middleware) RequireLivenessVerified() func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "app context required",
					"code":  "APP_CONTEXT_REQUIRED",
				})
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "organization context required",
					"code":  "ORGANIZATION_CONTEXT_REQUIRED",
				})
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "verification status not found",
					"code":  "VERIFICATION_NOT_FOUND",
				})
			}

			if !status.LivenessVerified {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "liveness verification required",
					"code":  "LIVENESS_VERIFICATION_REQUIRED",
				})
			}

			return next(c)
		}
	}
}

// RequireAMLClear enforces that AML screening is complete and clear
func (m *Middleware) RequireAMLClear() func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "app context required",
					"code":  "APP_CONTEXT_REQUIRED",
				})
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "organization context required",
					"code":  "ORGANIZATION_CONTEXT_REQUIRED",
				})
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "verification status not found",
					"code":  "VERIFICATION_NOT_FOUND",
				})
			}

			if !status.AMLScreened {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "AML screening required",
					"code":  "AML_SCREENING_REQUIRED",
				})
			}

			if !status.AMLClear {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "AML screening failed",
					"code":  "AML_SCREENING_FAILED",
				})
			}

			return next(c)
		}
	}
}

// RequireAge enforces minimum age requirement
func (m *Middleware) RequireAge(minimumAge int) func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "app context required",
					"code":  "APP_CONTEXT_REQUIRED",
				})
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "organization context required",
					"code":  "ORGANIZATION_CONTEXT_REQUIRED",
				})
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "verification status not found",
					"code":  "VERIFICATION_NOT_FOUND",
				})
			}

			if !status.AgeVerified {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "age verification required",
					"code":  "AGE_VERIFICATION_REQUIRED",
				})
			}

			return next(c)
		}
	}
}

// RequireNotBlocked ensures the user is not blocked from verification
func (m *Middleware) RequireNotBlocked() func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ctx := c.Request().Context()

			userID, ok := contexts.GetUserID(ctx)
			if !ok || userID.IsNil() {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "authentication required",
					"code":  "AUTHENTICATION_REQUIRED",
				})
			}

			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				// No app context means we can't check, allow request
				return next(c)
			}

			orgID, ok := contexts.GetOrganizationID(ctx)
			if !ok || orgID.IsNil() {
				// No org context means we can't check, allow request
				return next(c)
			}

			status, err := m.service.GetUserVerificationStatus(ctx, appID, orgID, userID)
			if err != nil {
				// No status means not blocked
				return next(c)
			}

			if status.IsBlocked {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":  "user blocked from verification",
					"code":   "USER_BLOCKED",
					"reason": status.BlockReason,
				})
			}

			return next(c)
		}
	}
}

// Helper functions

func meetsVerificationLevel(currentLevel, requiredLevel string) bool {
	// Define level hierarchy: none < basic < enhanced < full
	levels := map[string]int{
		"none":     0,
		"basic":    1,
		"enhanced": 2,
		"full":     3,
	}

	current, ok1 := levels[currentLevel]
	required, ok2 := levels[requiredLevel]

	if !ok1 || !ok2 {
		return false
	}

	return current >= required
}

// GetVerificationStatus retrieves the verification status from context
func GetVerificationStatus(c forge.Context) (*schema.UserVerificationStatus, bool) {
	if status := c.Request().Context().Value(VerificationStatusContextKey); status != nil {
		if s, ok := status.(*schema.UserVerificationStatus); ok {
			return s, true
		}
	}
	return nil, false
}

// GetVerificationLevel retrieves the verification level from context
func GetVerificationLevel(c forge.Context) string {
	if level := c.Request().Context().Value(VerificationLevelContextKey); level != nil {
		if l, ok := level.(string); ok {
			return l
		}
	}
	return "none"
}

// IsVerified checks if the user is verified
func IsVerified(c forge.Context) bool {
	status, ok := GetVerificationStatus(c)
	if !ok {
		return false
	}
	return status.IsVerified
}
