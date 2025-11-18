package base

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// APIKey represents an API key with its metadata (DTO)
// Updated for V2 architecture: App → Environment → Organization
type APIKey struct {
	ID             xid.ID            `json:"id"`
	AppID          xid.ID            `json:"appID"`                    // Platform tenant
	EnvironmentID  xid.ID            `json:"environmentID"`            // Required: environment-scoped
	OrganizationID *xid.ID           `json:"organizationID,omitempty"` // Optional: org-scoped
	UserID         xid.ID            `json:"userID"`                   // User who created the key
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Prefix         string            `json:"prefix"`
	KeyType        KeyType           `json:"keyType"` // pk/sk/rk
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

	// RBAC Integration (Hybrid Approach)
	DelegateUserPermissions bool     `json:"delegateUserPermissions"`     // Inherit creator's permissions
	ImpersonateUserID       *xid.ID  `json:"impersonateUserID,omitempty"` // Act as specific user
	Roles                   []string `json:"roles,omitempty"`             // Role IDs or names
	RBACPermissions         []string `json:"rbacPermissions,omitempty"`   // Computed RBAC permissions

	// Transient field - only populated during creation
	Key string `json:"key,omitempty"`
}

// ToSchema converts the APIKey DTO to schema.APIKey
func (a *APIKey) ToSchema() *schema.APIKey {
	return &schema.APIKey{
		AppID:                   a.AppID,
		EnvironmentID:           a.EnvironmentID,
		OrganizationID:          a.OrganizationID,
		UserID:                  a.UserID,
		Name:                    a.Name,
		Description:             a.Description,
		Prefix:                  a.Prefix,
		KeyType:                 string(a.KeyType),
		KeyHash:                 "", // Hash is never sent back
		Scopes:                  a.Scopes,
		Permissions:             a.Permissions,
		RateLimit:               a.RateLimit,
		AllowedIPs:              a.AllowedIPs,
		Active:                  a.Active,
		ExpiresAt:               a.ExpiresAt,
		UsageCount:              a.UsageCount,
		LastUsedAt:              a.LastUsedAt,
		LastUsedIP:              a.LastUsedIP,
		LastUsedUA:              a.LastUsedUA,
		Metadata:                a.Metadata,
		DelegateUserPermissions: a.DelegateUserPermissions,
		ImpersonateUserID:       a.ImpersonateUserID,
		Key:                     a.Key,
		AuditableModel: schema.AuditableModel{
			ID:        a.ID,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: nil,
		},
	}
}

// FromSchemaAPIKey converts a schema.APIKey to APIKey DTO
func FromSchemaAPIKey(s *schema.APIKey) *APIKey {
	return &APIKey{
		ID:                      s.ID,
		AppID:                   s.AppID,
		EnvironmentID:           s.EnvironmentID,
		OrganizationID:          s.OrganizationID,
		UserID:                  s.UserID,
		Name:                    s.Name,
		Description:             s.Description,
		Prefix:                  s.Prefix,
		KeyType:                 KeyType(s.KeyType),
		Scopes:                  s.Scopes,
		Permissions:             s.Permissions,
		RateLimit:               s.RateLimit,
		AllowedIPs:              s.AllowedIPs,
		Active:                  s.Active,
		ExpiresAt:               s.ExpiresAt,
		UsageCount:              s.UsageCount,
		LastUsedAt:              s.LastUsedAt,
		LastUsedIP:              s.LastUsedIP,
		LastUsedUA:              s.LastUsedUA,
		CreatedAt:               s.CreatedAt,
		UpdatedAt:               s.UpdatedAt,
		Metadata:                s.Metadata,
		DelegateUserPermissions: s.DelegateUserPermissions,
		ImpersonateUserID:       s.ImpersonateUserID,
		Key:                     s.Key,
	}
}

// FromSchemaAPIKeys converts multiple schema.APIKey to APIKey DTOs
func FromSchemaAPIKeys(keys []*schema.APIKey) []*APIKey {
	result := make([]*APIKey, len(keys))
	for i, key := range keys {
		result[i] = FromSchemaAPIKey(key)
	}
	return result
}

// CreateAPIKeyRequest represents a request to create an API key
// Updated for V2 architecture
type CreateAPIKeyRequest struct {
	AppID         xid.ID            `json:"appID" validate:"required"`         // Platform tenant
	EnvironmentID xid.ID            `json:"environmentID" validate:"required"` // Required: environment-scoped
	OrgID         *xid.ID           `json:"orgID,omitempty"`                   // Optional: org-scoped
	UserID        xid.ID            `json:"userID" validate:"required"`        // User creating the key
	Name          string            `json:"name" validate:"required,min=1,max=100"`
	Description   string            `json:"description,omitempty" validate:"max=500"`
	KeyType       KeyType           `json:"keyType" validate:"required"` // pk/sk/rk
	Scopes        []string          `json:"scopes" validate:"required,min=1"`
	Permissions   map[string]string `json:"permissions,omitempty"`
	RateLimit     int               `json:"rate_limit,omitempty" validate:"min=0,max=10000"`
	AllowedIPs    []string          `json:"allowed_ips,omitempty"` // IP whitelist (CIDR notation supported)
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`

	// RBAC Integration
	DelegateUserPermissions bool     `json:"delegateUserPermissions,omitempty"` // Inherit creator's permissions
	ImpersonateUserID       *xid.ID  `json:"impersonateUserID,omitempty"`       // Act as specific user
	RoleIDs                 []xid.ID `json:"roleIDs,omitempty"`                 // Assign roles on creation
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

// ListAPIKeysResponse is a type alias for the paginated response
type ListAPIKeysResponse = pagination.PageResponse[*APIKey]

// RotateAPIKeyRequest represents a request to rotate an API key
// Updated for V2 architecture
type RotateAPIKeyRequest struct {
	ID             xid.ID     `json:"id" validate:"required"`
	AppID          xid.ID     `json:"appID" validate:"required"`
	EnvironmentID  xid.ID     `json:"environmentID" validate:"required"`
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

// IsPublishable returns true if this is a publishable (frontend-safe) key
func (a *APIKey) IsPublishable() bool {
	return a.KeyType == KeyTypePublishable
}

// IsSecret returns true if this is a secret (backend-only, admin) key
func (a *APIKey) IsSecret() bool {
	return a.KeyType == KeyTypeSecret
}

// IsRestricted returns true if this is a restricted (backend-only, scoped) key
func (a *APIKey) IsRestricted() bool {
	return a.KeyType == KeyTypeRestricted
}

// CanPerformAdminOperation returns true if the key has admin privileges
func (a *APIKey) CanPerformAdminOperation() bool {
	return a.HasScope("admin:full")
}

// GetAllScopes returns all scopes including default key type scopes
func (a *APIKey) GetAllScopes() []string {
	allScopes := make(map[string]bool)

	// Add key type default scopes
	for _, s := range a.KeyType.GetDefaultScopes() {
		allScopes[s] = true
	}

	// Add custom scopes
	for _, s := range a.Scopes {
		allScopes[s] = true
	}

	// Convert to slice
	result := make([]string, 0, len(allScopes))
	for s := range allScopes {
		result = append(result, s)
	}
	return result
}
