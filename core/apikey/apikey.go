package apikey

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// APIKey represents an API key with its metadata
type APIKey struct {
	ID          xid.ID            `json:"id"`
	OrgID       string            `json:"org_id"`
	UserID      string            `json:"user_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Prefix      string            `json:"prefix"`
	Scopes      []string          `json:"scopes"`
	Permissions map[string]string `json:"permissions"`
	RateLimit   int               `json:"rate_limit"`
	Active      bool              `json:"active"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	UsageCount  int64             `json:"usage_count"`
	LastUsedAt  *time.Time        `json:"last_used_at,omitempty"`
	LastUsedIP  string            `json:"last_used_ip,omitempty"`
	LastUsedUA  string            `json:"last_used_ua,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`

	// Transient field - only populated during creation
	Key string `json:"key,omitempty"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	OrgID       string            `json:"org_id" validate:"required"`
	UserID      string            `json:"user_id" validate:"required"`
	Name        string            `json:"name" validate:"required,min=1,max=100"`
	Description string            `json:"description,omitempty" validate:"max=500"`
	Scopes      []string          `json:"scopes" validate:"required,min=1"`
	Permissions map[string]string `json:"permissions,omitempty"`
	RateLimit   int               `json:"rate_limit,omitempty" validate:"min=0,max=10000"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
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
type ListAPIKeysRequest struct {
	OrgID  string `json:"org_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
	Limit  int    `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset int    `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// ListAPIKeysResponse represents a response containing API keys
type ListAPIKeysResponse struct {
	APIKeys []*APIKey `json:"api_keys"`
	Total   int       `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

// RotateAPIKeyRequest represents a request to rotate an API key
type RotateAPIKeyRequest struct {
	ID        string     `json:"id" validate:"required"`
	OrgID     string     `json:"org_id" validate:"required"`
	UserID    string     `json:"user_id" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
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
type Repository interface {
	Create(ctx context.Context, apiKey *APIKey) error
	FindByID(ctx context.Context, id string) (*APIKey, error)
	FindByPrefix(ctx context.Context, prefix string) (*APIKey, error)
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*APIKey, error)
	FindByOrgID(ctx context.Context, orgID string, limit, offset int) ([]*APIKey, error)
	Update(ctx context.Context, apiKey *APIKey) error
	UpdateUsage(ctx context.Context, id string, ip, userAgent string) error
	Delete(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
	CountByUserID(ctx context.Context, userID string) (int, error)
	CountByOrgID(ctx context.Context, orgID string) (int, error)
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
