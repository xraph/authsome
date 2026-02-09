package mfa

import (
	"fmt"
	"time"

	"github.com/xraph/forge"
)

// RequireMFA ensures the user has completed MFA verification.
func RequireMFA(service *Service) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Get user ID from context
			userID, err := getUserIDFromContext(c)
			if err != nil {
				return c.JSON(401, map[string]string{"error": "unauthorized"})
			}

			// Check if user has enrolled factors
			factors, err := service.ListFactors(c.Request().Context(), userID, true)
			if err != nil {
				return c.JSON(500, map[string]string{"error": "failed to check MFA status"})
			}

			if len(factors) == 0 {
				// No factors enrolled - require enrollment
				return c.JSON(403, map[string]any{
					"error":   "mfa_required",
					"message": "Multi-factor authentication enrollment required",
					"action":  "enroll",
				})
			}

			// Check for valid MFA session token
			mfaToken := c.Request().Header.Get("X-Mfa-Token")
			if mfaToken == "" {
				// Check cookie
				cookie, err := c.Request().Cookie("mfa_token")
				if err == nil {
					mfaToken = cookie.Value
				}
			}

			if mfaToken == "" {
				// No MFA token - require verification
				return c.JSON(403, map[string]any{
					"error":   "mfa_verification_required",
					"message": "Multi-factor authentication verification required",
					"action":  "verify",
				})
			}

			// Validate MFA token
			session, err := service.repo.GetSessionByToken(c.Request().Context(), mfaToken)
			if err != nil || session == nil {
				return c.JSON(403, map[string]any{
					"error":   "invalid_mfa_token",
					"message": "Invalid or expired MFA token",
					"action":  "verify",
				})
			}

			// Check if session is completed
			if session.CompletedAt == nil {
				return c.JSON(403, map[string]any{
					"error":   "mfa_incomplete",
					"message": "MFA verification incomplete",
					"action":  "verify",
				})
			}

			// Check if session expired
			if time.Now().After(session.ExpiresAt) {
				return c.JSON(403, map[string]any{
					"error":   "mfa_expired",
					"message": "MFA session expired",
					"action":  "verify",
				})
			}

			// Store MFA session in context for use by handlers
			c.Set("mfa_session", session)

			return next(c)
		}
	}
}

// RequireFactorType ensures the user has a specific factor type enrolled.
func RequireFactorType(service *Service, factorType FactorType) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			userID, err := getUserIDFromContext(c)
			if err != nil {
				return c.JSON(401, map[string]string{"error": "unauthorized"})
			}

			factors, err := service.ListFactors(c.Request().Context(), userID, true)
			if err != nil {
				return c.JSON(500, map[string]string{"error": "failed to check factors"})
			}

			// Check if user has the required factor type
			hasFactorType := false

			for _, factor := range factors {
				if factor.Type == factorType {
					hasFactorType = true

					break
				}
			}

			if !hasFactorType {
				return c.JSON(403, map[string]any{
					"error":         "factor_required",
					"message":       fmt.Sprintf("%s factor required", factorType),
					"required_type": factorType,
					"action":        "enroll",
				})
			}

			return next(c)
		}
	}
}

// StepUpAuth requires recent MFA verification for sensitive operations.
func StepUpAuth(service *Service, maxAge time.Duration) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			userID, err := getUserIDFromContext(c)
			if err != nil {
				return c.JSON(401, map[string]string{"error": "unauthorized"})
			}

			// Get MFA token
			mfaToken := c.Request().Header.Get("X-Mfa-Token")
			if mfaToken == "" {
				cookie, err := c.Request().Cookie("mfa_token")
				if err == nil {
					mfaToken = cookie.Value
				}
			}

			if mfaToken == "" {
				return c.JSON(403, map[string]any{
					"error":   "step_up_required",
					"message": "Step-up authentication required for this operation",
					"action":  "step_up",
				})
			}

			// Get session
			session, err := service.repo.GetSessionByToken(c.Request().Context(), mfaToken)
			if err != nil || session == nil || session.CompletedAt == nil {
				return c.JSON(403, map[string]any{
					"error":   "step_up_required",
					"message": "Step-up authentication required",
					"action":  "step_up",
				})
			}

			// Check if verification is recent enough
			age := time.Since(*session.CompletedAt)
			if age > maxAge {
				return c.JSON(403, map[string]any{
					"error":   "step_up_expired",
					"message": fmt.Sprintf("Step-up authentication expired (max age: %v)", maxAge),
					"action":  "step_up",
					"age":     age.String(),
					"max_age": maxAge.String(),
				})
			}

			// Verify session belongs to user
			if session.UserID != userID {
				return c.JSON(403, map[string]string{"error": "forbidden"})
			}

			c.Set("mfa_session", session)
			c.Set("step_up_verified", true)

			return next(c)
		}
	}
}

// AdaptiveMFA applies risk-based MFA requirements.
func AdaptiveMFA(service *Service) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			if !service.config.AdaptiveMFA.Enabled {
				return next(c)
			}

			userID, err := getUserIDFromContext(c)
			if err != nil {
				return c.JSON(401, map[string]string{"error": "unauthorized"})
			}

			// Perform risk assessment
			riskCtx := &RiskContext{
				UserID:    userID,
				IPAddress: c.Request().RemoteAddr,
				UserAgent: c.Request().UserAgent(),
				DeviceID:  c.Request().Header.Get("X-Device-Id"),
				Timestamp: time.Now(),
			}

			assessment, err := service.riskEngine.AssessRisk(c.Request().Context(), riskCtx)
			if err != nil {
				// Log error but don't block request
				// In production, you might want to block on assessment failure
				return next(c)
			}

			// Store assessment in context
			c.Set("risk_assessment", assessment)

			// If risk is high, require MFA
			if assessment.Level == RiskLevelHigh || assessment.Level == RiskLevelCritical {
				// Check for valid MFA session
				mfaToken := c.Request().Header.Get("X-Mfa-Token")
				if mfaToken == "" {
					cookie, err := c.Request().Cookie("mfa_token")
					if err == nil {
						mfaToken = cookie.Value
					}
				}

				if mfaToken == "" {
					return c.JSON(403, map[string]any{
						"error":        "high_risk_mfa_required",
						"message":      "Multi-factor authentication required due to high risk",
						"risk_level":   assessment.Level,
						"risk_score":   assessment.Score,
						"risk_factors": assessment.Factors,
						"action":       "verify",
					})
				}

				// Validate session
				session, err := service.repo.GetSessionByToken(c.Request().Context(), mfaToken)
				if err != nil || session == nil || session.CompletedAt == nil {
					return c.JSON(403, map[string]any{
						"error":      "high_risk_mfa_required",
						"message":    "Valid MFA session required due to high risk",
						"risk_level": assessment.Level,
						"action":     "verify",
					})
				}

				// For critical risk, require step-up even if session exists
				if assessment.Level == RiskLevelCritical {
					// Check if session is recent (within last 5 minutes)
					if session.CompletedAt == nil || time.Since(*session.CompletedAt) > 5*time.Minute {
						return c.JSON(403, map[string]any{
							"error":      "critical_risk_step_up_required",
							"message":    "Recent MFA verification required due to critical risk",
							"risk_level": assessment.Level,
							"action":     "step_up",
						})
					}
				}
			}

			return next(c)
		}
	}
}

// OptionalMFA suggests MFA but doesn't require it.
func OptionalMFA(service *Service) func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			userID, err := getUserIDFromContext(c)
			if err != nil {
				return next(c)
			}

			// Check if user has MFA enrolled
			factors, err := service.ListFactors(c.Request().Context(), userID, true)
			if err == nil && len(factors) > 0 {
				// User has MFA, store flag in context
				c.Set("mfa_available", true)
			}

			return next(c)
		}
	}
}
