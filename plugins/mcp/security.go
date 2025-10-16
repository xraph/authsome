package mcp

import (
	"context"
	"fmt"
	"strings"
)

// SecurityLayer handles authorization and data sanitization for MCP
type SecurityLayer struct {
	config Config
}

// NewSecurityLayer creates a new security layer
func NewSecurityLayer(config Config) *SecurityLayer {
	return &SecurityLayer{
		config: config,
	}
}

// AuthorizeRequest checks if a request is authorized
func (s *SecurityLayer) AuthorizeRequest(ctx context.Context, operation string, apiKey string) error {
	// If API key not required (e.g., stdio in dev mode), allow
	if !s.config.Authorization.RequireAPIKey {
		return nil
	}

	// Check if API key provided
	if apiKey == "" {
		return fmt.Errorf("API key required but not provided")
	}

	// TODO: Validate API key against database
	// For now, accept any non-empty key in development mode
	if s.config.Mode == ModeDevelopment {
		return nil
	}

	return fmt.Errorf("API key validation not yet implemented")
}

// CheckOperationAllowed checks if operation is allowed in current mode
func (s *SecurityLayer) CheckOperationAllowed(operation string) error {
	// Check if operation is in allowed list
	for _, allowed := range s.config.Authorization.AllowedOperations {
		if allowed == operation {
			return nil
		}
	}

	// Check if it's an admin operation
	for _, admin := range s.config.Authorization.AdminOperations {
		if admin == operation {
			// Only allowed in admin or development mode
			if s.config.Mode == ModeAdmin || s.config.Mode == ModeDevelopment {
				return nil
			}
			return fmt.Errorf("operation %s requires admin mode", operation)
		}
	}

	return fmt.Errorf("operation %s not allowed", operation)
}

// SanitizeData removes sensitive information from data
func (s *SecurityLayer) SanitizeData(data interface{}, dataType string) interface{} {
	// Type-specific sanitization
	switch dataType {
	case "user":
		return s.sanitizeUser(data)
	case "session":
		return s.sanitizeSession(data)
	case "config":
		return s.sanitizeConfig(data)
	default:
		return data
	}
}

// sanitizeUser removes sensitive user fields
func (s *SecurityLayer) sanitizeUser(data interface{}) interface{} {
	userMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	// Always remove these fields
	sensitiveFields := []string{
		"password",
		"password_hash",
		"twofa_secret",
		"backup_codes",
		"oauth_tokens",
		"api_keys",
	}

	for _, field := range sensitiveFields {
		delete(userMap, field)
	}

	// Conditionally mask these based on mode
	if s.config.Mode == ModeReadOnly {
		maskFields := []string{
			"phone",
			"last_ip",
		}

		for _, field := range maskFields {
			if val, exists := userMap[field]; exists {
				if str, ok := val.(string); ok {
					userMap[field] = maskString(str)
				}
			}
		}
	}

	return userMap
}

// sanitizeSession removes sensitive session fields
func (s *SecurityLayer) sanitizeSession(data interface{}) interface{} {
	sessionMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	// Remove token/secret fields
	delete(sessionMap, "token")
	delete(sessionMap, "refresh_token")
	delete(sessionMap, "session_token")

	return sessionMap
}

// sanitizeConfig removes secrets from configuration
func (s *SecurityLayer) sanitizeConfig(data interface{}) interface{} {
	configMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	// Recursively remove fields containing "secret", "key", "password"
	for key, val := range configMap {
		keyLower := strings.ToLower(key)

		// Remove sensitive keys
		if strings.Contains(keyLower, "secret") ||
			strings.Contains(keyLower, "password") ||
			strings.Contains(keyLower, "token") ||
			(strings.Contains(keyLower, "key") && !strings.Contains(keyLower, "public")) {
			configMap[key] = "[REDACTED]"
			continue
		}

		// Recursively sanitize nested objects
		if nestedMap, ok := val.(map[string]interface{}); ok {
			configMap[key] = s.sanitizeConfig(nestedMap)
		}
	}

	return configMap
}

// maskString masks part of a string (e.g., email, phone)
func maskString(s string) string {
	if len(s) <= 4 {
		return "***"
	}

	// Show first 2 and last 2 characters
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

// maskIP partially masks an IP address
func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		// IPv4: show first two octets
		return parts[0] + "." + parts[1] + ".*.*"
	}

	// IPv6 or other: mask most of it
	if len(ip) > 8 {
		return ip[:4] + "..." + ip[len(ip)-4:]
	}

	return "***"
}
