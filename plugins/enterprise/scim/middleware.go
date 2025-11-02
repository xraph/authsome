package scim

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/xraph/forge"
	"golang.org/x/time/rate"
)

// AuthMiddleware validates SCIM bearer tokens
func (p *Plugin) AuthMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Extract bearer token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusUnauthorized,
					Detail:   "Authorization header required",
				})
			}
			
			// Parse bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusUnauthorized,
					Detail:   "Invalid authorization header format. Expected: Bearer <token>",
				})
			}
			
			token := parts[1]
			
			// Validate token
			provToken, err := p.service.ValidateProvisioningToken(c.Request().Context(), token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusUnauthorized,
					Detail:   "Invalid or expired token",
				})
			}
			
			// Store org ID and token info in forge context values
			c.Set("org_id", provToken.OrgID.String())
			c.Set("scim_token", provToken)
			c.Set("token_scopes", provToken.Scopes)
			
			return next(c)
		}
	}
}

// OrgResolutionMiddleware ensures organization context is set
func (p *Plugin) OrgResolutionMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Organization ID should already be set by AuthMiddleware
			orgID := c.Get("org_id")
			if orgID == nil || orgID.(string) == "" {
				return c.JSON(http.StatusForbidden, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusForbidden,
					Detail:   "Organization context not found",
				})
			}
			
			// Validate organization exists (basic check)
			// In production, you might want to verify the organization exists in the database
			if orgID.(string) == "" {
				return c.JSON(http.StatusForbidden, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusForbidden,
					Detail:   "Invalid organization ID",
				})
			}
			
			return next(c)
		}
	}
}

// RateLimitMiddleware implements rate limiting for SCIM endpoints
func (p *Plugin) RateLimitMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	if !p.config.RateLimit.Enabled {
		// Rate limiting disabled, pass through
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}
	
	// Create rate limiter per organization
	limiters := &sync.Map{}
	
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			orgID := c.Get("org_id")
			if orgID == nil {
				// No org ID, skip rate limiting
				return next(c)
			}
			
			orgIDStr := orgID.(string)
			
			// Get or create rate limiter for this organization
			limiterInterface, _ := limiters.LoadOrStore(orgIDStr, rate.NewLimiter(
				rate.Limit(float64(p.config.RateLimit.RequestsPerMin)/60.0), // Per second rate
				p.config.RateLimit.BurstSize,
			))
			
			limiter := limiterInterface.(*rate.Limiter)
			
			// Check rate limit
			if !limiter.Allow() {
				// Record rate limit hit
				GetMetrics().RecordRateLimitHit()
				
				return c.JSON(http.StatusTooManyRequests, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusTooManyRequests,
					ScimType: "tooMany",
					Detail:   fmt.Sprintf("Rate limit exceeded. Maximum %d requests per minute allowed.", p.config.RateLimit.RequestsPerMin),
				})
			}
			
			return next(c)
		}
	}
}

// RequireAdminMiddleware ensures the request is from an admin
func (p *Plugin) RequireAdminMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Check if user has admin privileges
			// For SCIM admin endpoints, check token scopes
			scopes := c.Get("token_scopes")
			if scopes == nil {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Admin access required",
				})
			}
			
			scopeList := scopes.([]string)
			hasAdminScope := false
			for _, scope := range scopeList {
				if scope == "admin" || scope == "scim:admin" {
					hasAdminScope = true
					break
				}
			}
			
			if !hasAdminScope {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Admin scope required for this endpoint",
				})
			}
			
			return next(c)
		}
	}
}

// LoggingMiddleware logs SCIM operations for audit
func (p *Plugin) LoggingMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	metrics := GetMetrics()
	
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			start := time.Now()
			
			// Increment active requests
			metrics.IncrementActiveRequests()
			defer metrics.DecrementActiveRequests()
			
			// Execute handler
			err := next(c)
			
			duration := time.Since(start)
			
			// Log the operation
			orgID := c.Get("org_id")
			if orgID != nil {
				// TODO: Create provisioning log entry
				_ = duration // Use duration for logging
			}
			
			return err
		}
	}
}

// SecurityHeadersMiddleware adds security headers to SCIM responses
func (p *Plugin) SecurityHeadersMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			// Add security headers
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Response().Header().Set("Content-Type", "application/scim+json; charset=utf-8")
			
			return next(c)
		}
	}
}

// IPWhitelistMiddleware enforces IP whitelisting if configured
func (p *Plugin) IPWhitelistMiddleware() func(func(forge.Context) error) func(forge.Context) error {
	if len(p.config.Security.IPWhitelist) == 0 {
		// No whitelist, pass through
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}
	
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			clientIP := getClientIP(c)
			
			// Check if IP is in whitelist
			allowed := false
			for _, ip := range p.config.Security.IPWhitelist {
				if ip == clientIP || ip == "*" {
					allowed = true
					break
				}
				// TODO: Add CIDR range matching
			}
			
			if !allowed {
				return c.JSON(http.StatusForbidden, &ErrorResponse{
					Schemas:  []string{SchemaError},
					Status:   http.StatusForbidden,
					Detail:   "Access denied: IP not whitelisted",
				})
			}
			
			return next(c)
		}
	}
}

// Helper functions

func getClientIP(c forge.Context) string {
	// Check X-Forwarded-For header first
	forwarded := c.Request().Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the list
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Fall back to X-Real-IP
	realIP := c.Request().Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fall back to RemoteAddr
	return c.Request().RemoteAddr
}

