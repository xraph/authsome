package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// SecurityLayer handles authorization and data sanitization for MCP.
type SecurityLayer struct {
	config Config
	db     *bun.DB
}

// NewSecurityLayer creates a new security layer.
func NewSecurityLayer(config Config, db *bun.DB) *SecurityLayer {
	return &SecurityLayer{
		config: config,
		db:     db,
	}
}

// AuthorizeRequest checks if a request is authorized.
func (s *SecurityLayer) AuthorizeRequest(ctx context.Context, operation string, apiKey string) error {
	// If API key not required (e.g., stdio in dev mode), allow
	if !s.config.Authorization.RequireAPIKey {
		return nil
	}

	// Check if API key provided
	if apiKey == "" {
		return errs.BadRequest("API key required but not provided")
	}

	// Development mode: accept any non-empty key
	if s.config.Mode == ModeDevelopment {
		return nil
	}

	// Validate API key against database
	if s.db != nil {
		var key schema.APIKey

		err := s.db.NewSelect().
			Model(&key).
			Where("key = ?", apiKey).
			Scan(ctx)
		if err != nil {
			return errs.BadRequest("invalid API key")
		}

		// Check if key is active
		if !key.Active {
			return errs.BadRequest("API key is inactive")
		}

		// Check if key is expired
		if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
			return errs.BadRequest("API key has expired")
		}

		// Check if key has MCP scope/permission
		// API keys should have "mcp:read" or "mcp:admin" permission
		if !s.hasPermission(&key, operation) {
			return fmt.Errorf("API key does not have required permission for operation: %s", operation)
		}

		// Update last used timestamp
		now := time.Now()
		key.LastUsedAt = &now
		_, _ = s.db.NewUpdate().Model(&key).Column("last_used_at").Where("id = ?", key.ID).Exec(ctx)

		return nil
	}

	return errs.InternalServerErrorWithMessage("API key validation requires database connection")
}

// hasPermission checks if API key has permission for operation.
func (s *SecurityLayer) hasPermission(key *schema.APIKey, operation string) bool {
	// API keys have a Permissions map[string]string field
	// Check for admin permissions
	if _, hasAdmin := key.Permissions["mcp:admin"]; hasAdmin {
		return true
	}

	if _, hasAdmin := key.Permissions["admin"]; hasAdmin {
		return true
	}

	// Check if operation requires admin
	if slices.Contains(s.config.Authorization.AdminOperations, operation) {
		// Operation requires admin but key doesn't have it
		_, hasAdmin := key.Permissions["mcp:admin"]

		return hasAdmin
	}

	// For read operations, mcp:read is sufficient
	if strings.HasPrefix(operation, "query_") || strings.HasPrefix(operation, "get_") || strings.HasPrefix(operation, "list_") {
		_, hasRead := key.Permissions["mcp:read"]
		_, hasAdmin := key.Permissions["mcp:admin"]

		return hasRead || hasAdmin
	}

	// For write operations, need mcp:write or mcp:admin
	_, hasWrite := key.Permissions["mcp:write"]
	_, hasAdmin := key.Permissions["mcp:admin"]

	return hasWrite || hasAdmin
}

// CheckOperationAllowed checks if operation is allowed in current mode.
func (s *SecurityLayer) CheckOperationAllowed(operation string) error {
	// Check if operation is in allowed list
	if slices.Contains(s.config.Authorization.AllowedOperations, operation) {
		return nil
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

// SanitizeData removes sensitive information from data.
func (s *SecurityLayer) SanitizeData(data any, dataType string) any {
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

// sanitizeUser removes sensitive user fields.
func (s *SecurityLayer) sanitizeUser(data any) any {
	userMap, ok := data.(map[string]any)
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
		if val, exists := userMap["phone"]; exists {
			if str, ok := val.(string); ok {
				userMap["phone"] = maskString(str)
			}
		}

		if val, exists := userMap["last_ip"]; exists {
			if str, ok := val.(string); ok {
				userMap["last_ip"] = maskIP(str)
			}
		}
	}

	return userMap
}

// sanitizeSession removes sensitive session fields.
func (s *SecurityLayer) sanitizeSession(data any) any {
	sessionMap, ok := data.(map[string]any)
	if !ok {
		return data
	}

	// Remove token/secret fields
	delete(sessionMap, "token")
	delete(sessionMap, "refresh_token")
	delete(sessionMap, "session_token")

	return sessionMap
}

// sanitizeConfig removes secrets from configuration.
func (s *SecurityLayer) sanitizeConfig(data any) any {
	configMap, ok := data.(map[string]any)
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
		if nestedMap, ok := val.(map[string]any); ok {
			configMap[key] = s.sanitizeConfig(nestedMap)
		}
	}

	return configMap
}

// maskString masks part of a string (e.g., email, phone).
func maskString(s string) string {
	if len(s) <= 4 {
		return "***"
	}

	// Show first 2 and last 2 characters
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

// maskIP partially masks an IP address.
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
