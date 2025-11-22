package middleware

import (
	"strconv"
	"strings"

	"github.com/xraph/forge"
)

// CORSConfig holds CORS middleware configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	MaxAge           int
}

// CORSMiddleware creates a CORS middleware with the given configuration
func CORSMiddleware(config CORSConfig) forge.Middleware {
	// Default methods and headers if not specified
	allowedMethods := config.AllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}

	allowedHeaders := config.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-API-Key",
			"X-App-ID",
			"X-Environment",
			"X-Organization-ID",
		}
	}

	maxAge := config.MaxAge
	if maxAge == 0 {
		maxAge = 86400 // 24 hours
	}

	return func(next forge.Handler) forge.Handler {
		return func(c forge.Context) error {
			origin := c.Request().Header.Get("Origin")

			// Check if origin is allowed
			originAllowed := false
			allowedOrigin := ""

			for _, allowed := range config.AllowedOrigins {
				if allowed == "*" {
					originAllowed = true
					allowedOrigin = "*"
					break
				}
				if allowed == origin {
					originAllowed = true
					allowedOrigin = origin
					break
				}
			}

			if originAllowed {
				// Set CORS headers
				if allowedOrigin != "" {
					c.Response().Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				}

				if config.AllowCredentials {
					c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Handle preflight requests
				if c.Request().Method == "OPTIONS" {
					c.Response().Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
					c.Response().Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
					if len(config.ExposeHeaders) > 0 {
						c.Response().Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
					}
					c.Response().Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
					return c.NoContent(204)
				}

				// For actual requests, set expose headers
				if len(config.ExposeHeaders) > 0 {
					c.Response().Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
				}
			}

			return next(c)
		}
	}
}
