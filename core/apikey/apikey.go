package apikey

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// APIKey represents an API key with its metadata
// Updated for V2 architecture: App → Environment → Organization
type APIKey struct {
	ID             xid.ID            `json:"id"`
	AppID          xid.ID            `json:"appID"`                    // Platform tenant
	EnvironmentID  *xid.ID           `json:"environmentID,omitempty"`  // Optional: environment-scoped
	OrganizationID *xid.ID           `json:"organizationID,omitempty"` // Optional: org-scoped
	UserID         xid.ID            `json:"userID"`                   // User who created the key
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Prefix         string            `json:"prefix"`
	Scopes         []string          `json:"scopes"`
	Permissions    map[string]string `json:"permissions"`
	RateLimit      int               `json:"rate_limit"`
	AllowedIPs     []string          `json:"allowed_ips,omitempty"`
	Active         bool              `json:"active"`
	ExpiresAt      *time.Time        `json:"expires_at,omitempty"`
	UsageCount     int64             `json:"usage_count"`
	LastUsedAt     *time.Time        `json:"last_used_at,omitempty"`
	LastUsedIP     string            `json:"last_used_ip,omitempty"`
	LastUsedUA     string            `json:"last_used_ua,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`

	// Transient field - only populated during creation
	Key string `json:"key,omitempty"`
}

// CreateAPIKeyRequest represents a request to create an API key
// Updated for V2 architecture
type CreateAPIKeyRequest struct {
	AppID         xid.ID            `json:"appID" validate:"required"`  // Platform tenant
	EnvironmentID *xid.ID           `json:"environmentID,omitempty"`    // Optional: environment-scoped
	OrgID         *xid.ID           `json:"orgID,omitempty"`            // Optional: org-scoped
	UserID        xid.ID            `json:"userID" validate:"required"` // User creating the key
	Name          string            `json:"name" validate:"required,min=1,max=100"`
	Description   string            `json:"description,omitempty" validate:"max=500"`
	Scopes        []string          `json:"scopes" validate:"required,min=1"`
	Permissions   map[string]string `json:"permissions,omitempty"`
	RateLimit     int               `json:"rate_limit,omitempty" validate:"min=0,max=10000"`
	AllowedIPs    []string          `json:"allowed_ips,omitempty"` // IP whitelist (CIDR notation supported)
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// UpdateAPIKeyRequest represents a request to update an API key
type UpdateAPIKeyRequest struct {
	Name        *string           `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string           `json:"description,omitempty" validate:"omitempty,max=500"`
	Scopes      []string          `json:"scopes,omitempty" validate:"omitempty,min=1"`
	Permissions map[string]string `json:"permissions,omitempty"`
	RateLimit   *int              `json:"rate_limit,omitempty" validate:"omitempty,min=0,max=10000"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Active      *bool             `json:"active,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ListAPIKeysRequest represents a request to list API keys
// Updated for V2 architecture
type ListAPIKeysRequest struct {
	AppID          *xid.ID `json:"appId,omitempty"`         // Filter by app
	EnvironmentID  *xid.ID `json:"environmentId,omitempty"` // Filter by environment
	OrganizationID *xid.ID `json:"orgId,omitempty"`         // Filter by organization
	UserID         *xid.ID `json:"userId,omitempty"`        // Filter by user
	Limit          int     `json:"limit,omitempty" validate:"omitempty,min=1,max=100" default:"20"`
	Offset         int     `json:"offset,omitempty" validate:"omitempty,min=0" default:"0"`
}

// ListAPIKeysResponse represents a response containing API keys
type ListAPIKeysResponse struct {
	APIKeys []*APIKey `json:"api_keys"`
	Total   int       `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

// RotateAPIKeyRequest represents a request to rotate an API key
// Updated for V2 architecture
type RotateAPIKeyRequest struct {
	ID             xid.ID     `json:"id" validate:"required"`
	AppID          xid.ID     `json:"appID" validate:"required"`
	EnvironmentID  *xid.ID    `json:"environmentID,omitempty"`
	OrganizationID *xid.ID    `json:"organizationID,omitempty"`
	UserID         xid.ID     `json:"userID" validate:"required"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// VerifyAPIKeyRequest represents a request to verify an API key
type VerifyAPIKeyRequest struct {
	Key                string `json:"key" validate:"required"`
	RequiredScope      string `json:"required_scope,omitempty"`
	RequiredPermission string `json:"required_permission,omitempty"`
	IP                 string `json:"ip,omitempty"`
	UserAgent          string `json:"user_agent,omitempty"`
}

// VerifyAPIKeyResponse represents a response from API key verification
type VerifyAPIKeyResponse struct {
	Valid  bool    `json:"valid"`
	APIKey *APIKey `json:"api_key,omitempty"`
	Error  string  `json:"error,omitempty"`
}

// Repository defines the interface for API key storage
// Updated for V2 architecture
type Repository interface {
	Create(ctx context.Context, apiKey *APIKey) error
	FindByID(ctx context.Context, id xid.ID) (*APIKey, error)
	FindByPrefix(ctx context.Context, prefix string) (*APIKey, error)

	// List with flexible filtering
	FindByApp(ctx context.Context, appID xid.ID, limit, offset int) ([]*APIKey, error)
	FindByUser(ctx context.Context, appID, userID xid.ID, limit, offset int) ([]*APIKey, error)
	FindByOrganization(ctx context.Context, appID xid.ID, orgID xid.ID, limit, offset int) ([]*APIKey, error)
	FindByEnvironment(ctx context.Context, appID, envID xid.ID, limit, offset int) ([]*APIKey, error)

	// Update operations
	Update(ctx context.Context, apiKey *APIKey) error
	UpdateUsage(ctx context.Context, id xid.ID, ip, userAgent string) error
	Delete(ctx context.Context, id xid.ID) error
	Deactivate(ctx context.Context, id xid.ID) error

	// Count operations
	CountByApp(ctx context.Context, appID xid.ID) (int, error)
	CountByUser(ctx context.Context, appID, userID xid.ID) (int, error)
	CountByOrganization(ctx context.Context, appID xid.ID, orgID xid.ID) (int, error)

	// Maintenance
	CleanupExpired(ctx context.Context) (int, error)
}

// IsExpired checks if the API key has expired
func (a *APIKey) IsExpired() bool {
	return a.ExpiresAt != nil && time.Now().After(*a.ExpiresAt)
}

// HasScope checks if the API key has a specific scope
func (a *APIKey) HasScope(scope string) bool {
	for _, s := range a.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasPermission checks if the API key has a specific permission
func (a *APIKey) HasPermission(permission string) bool {
	if a.Permissions == nil {
		return false
	}
	_, exists := a.Permissions[permission]
	return exists
}

// HasScopeWildcard checks if the API key has a scope, supporting wildcards
// Examples: "admin:*" matches "admin:users", "admin:settings", etc.
func (a *APIKey) HasScopeWildcard(scope string) bool {
	for _, s := range a.Scopes {
		if s == scope {
			return true // Exact match
		}
		// Wildcard matching: "admin:*" matches "admin:anything"
		if len(s) > 2 && s[len(s)-2:] == ":*" {
			prefix := s[:len(s)-2]
			if len(scope) > len(prefix) && scope[:len(prefix)] == prefix && scope[len(prefix)] == ':' {
				return true
			}
		}
	}
	return false
}
