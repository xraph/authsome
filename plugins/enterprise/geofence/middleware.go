package geofence

import (
	"net/http"
	"strings"

	"github.com/rs/xid"
	"github.com/xraph/forge"
)

// Middleware provides geofence checking middleware
type Middleware struct {
	service *Service
	config  *Config
}

// NewMiddleware creates a new geofence middleware
func NewMiddleware(service *Service, config *Config) *Middleware {
	return &Middleware{
		service: service,
		config:  config,
	}
}

// CheckGeofence middleware checks geofence rules for each request
func (m *Middleware) CheckGeofence(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Skip if not enabled or validation not required
		if !m.config.Enabled || !m.config.Session.ValidateOnRequest {
			return next(c)
		}

		// Get user and org IDs from context (set by auth middleware)
		userID, ok := c.Get("user_id").(xid.ID)
		if !ok || userID.IsNil() {
			// Not authenticated, skip geofence check
			return next(c)
		}

		orgID, ok := c.Get("organization_id").(xid.ID)
		if !ok || orgID.IsNil() {
			// No organization context, skip
			return next(c)
		}

		// Get IP address
		ip := m.getClientIP(c)
		if ip == "" {
			// Cannot determine IP, allow by default if not in strict mode
			if m.config.Restrictions.StrictMode {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "cannot determine client IP address",
				})
			}
			return next(c)
		}

		// Get session ID if available
		var sessionID *xid.ID
		if sid, ok := c.Get("session_id").(xid.ID); ok && !sid.IsNil() {
			sessionID = &sid
		}

		// Perform geofence check
		req := &LocationCheckRequest{
			UserID:         userID,
			OrganizationID: orgID,
			SessionID:      sessionID,
		IPAddress:      ip,
		UserAgent:      c.Request().Header.Get("User-Agent"),
		EventType:      "request",
		}

		result, err := m.service.CheckLocation(c.Context(), req)
		if err != nil {
			// Log error but don't block request unless in strict mode
			if m.config.Restrictions.StrictMode {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "geofence check failed",
				})
			}
			return next(c)
		}

		// Handle result
		if !result.Allowed {
			if m.config.Session.InvalidateOnViolation && sessionID != nil {
				// TODO: Invalidate session
			}

			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":      "access denied by geofence policy",
				"reason":     result.Reason,
				"rule":       result.RuleName,
				"violations": result.Violations,
			})
		}

		// Check if MFA is required
		if result.RequireMFA {
			// Set flag for subsequent middleware to enforce MFA
			c.Set("mfa_required", true)
			c.Set("mfa_reason", "geofence_policy")
		}

		// Proceed with request
		return next(c)
	}
}

// getClientIP extracts the client IP address from the request
func (m *Middleware) getClientIP(c forge.Context) string {
	req := c.Request()
	
	// Check various headers in order of preference
	// X-Forwarded-For
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IP
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// CF-Connecting-IP (Cloudflare)
	if cfip := req.Header.Get("CF-Connecting-IP"); cfip != "" {
		return strings.TrimSpace(cfip)
	}

	// True-Client-IP (Cloudflare/Akamai)
	if tcip := req.Header.Get("True-Client-IP"); tcip != "" {
		return strings.TrimSpace(tcip)
	}

	// Fallback to remote address
	remoteAddr := req.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}

	return remoteAddr
}

// RequireLocation middleware requires geofence check to pass
// This is a stronger enforcement than CheckGeofence
func (m *Middleware) RequireLocation(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Temporarily set strict mode for this check
		originalStrictMode := m.config.Restrictions.StrictMode
		m.config.Restrictions.StrictMode = true
		defer func() {
			m.config.Restrictions.StrictMode = originalStrictMode
		}()

		return m.CheckGeofence(next)(c)
	}
}

// RequireCountry middleware requires the request to come from specific countries
func (m *Middleware) RequireCountry(countries ...string) func(next func(forge.Context) error) func(forge.Context) error {
	return func(next func(forge.Context) error) func(forge.Context) error {
		return func(c forge.Context) error {
			ip := m.getClientIP(c)
			if ip == "" {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "cannot determine client IP address",
				})
			}

			geoData, err := m.service.GetGeolocation(c.Context(), ip)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "geolocation lookup failed",
				})
			}

			allowed := false
			for _, country := range countries {
				if strings.EqualFold(geoData.CountryCode, country) {
					allowed = true
					break
				}
			}

			if !allowed {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error":   "access denied - country not allowed",
					"country": geoData.CountryCode,
				})
			}

			return next(c)
		}
	}
}

// BlockVPN middleware blocks requests from VPNs
func (m *Middleware) BlockVPN(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		ip := m.getClientIP(c)
		if ip == "" {
			return next(c) // Allow if IP cannot be determined
		}

		detection, err := m.service.GetDetection(c.Context(), ip)
		if err != nil {
			return next(c) // Allow on detection error
		}

		if detection != nil && detection.IsVPN {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":    "access denied - VPN detected",
				"provider": detection.VPNProvider,
			})
		}

		return next(c)
	}
}

// BlockProxy middleware blocks requests from proxies
func (m *Middleware) BlockProxy(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		ip := m.getClientIP(c)
		if ip == "" {
			return next(c) // Allow if IP cannot be determined
		}

		detection, err := m.service.GetDetection(c.Context(), ip)
		if err != nil {
			return next(c) // Allow on detection error
		}

		if detection != nil && detection.IsProxy {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "access denied - proxy detected",
			})
		}

		return next(c)
	}
}

// BlockTor middleware blocks requests from Tor exit nodes
func (m *Middleware) BlockTor(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		ip := m.getClientIP(c)
		if ip == "" {
			return next(c) // Allow if IP cannot be determined
		}

		detection, err := m.service.GetDetection(c.Context(), ip)
		if err != nil {
			return next(c) // Allow on detection error
		}

		if detection != nil && detection.IsTor {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "access denied - Tor detected",
			})
		}

		return next(c)
	}
}

