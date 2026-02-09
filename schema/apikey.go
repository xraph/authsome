package schema

import (
	"context"
	"slices"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// APIKey represents an API key for programmatic access
// Updated for V2 architecture: App → Environment → Organization.
type APIKey struct {
	AuditableModel
	bun.BaseModel `bun:"table:api_keys"`

	// 3-tier context (V2 architecture)
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)"         json:"appID"`                    // Platform tenant
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`            // Required: environment-scoped key
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)"        json:"organizationID,omitempty"` // Optional: org-scoped key
	UserID         xid.ID  `bun:"user_id,notnull,type:varchar(20)"        json:"userID"`                   // User who created the key

	// Key identification
	Name        string `bun:"name,notnull"                  json:"name"`
	Description string `bun:"description"                   json:"description,omitempty"`
	Prefix      string `bun:"prefix,notnull,unique"         json:"prefix"`  // pk_test_xxx, sk_prod_xxx, rk_dev_xxx
	KeyType     string `bun:"key_type,notnull,default:'rk'" json:"keyType"` // pk/sk/rk
	KeyHash     string `bun:"key_hash,notnull"              json:"-"`       // Hashed key for verification

	// Permissions and scopes
	Scopes      []string          `bun:"scopes,type:jsonb"       json:"scopes"`                // ["read", "write", "admin"]
	Permissions map[string]string `bun:"permissions,type:jsonb"  json:"permissions"`           // Custom permissions
	RateLimit   int               `bun:"rate_limit,default:1000" json:"rate_limit"`            // Requests per hour
	AllowedIPs  []string          `bun:"allowed_ips,type:jsonb"  json:"allowed_ips,omitempty"` // IP whitelist (CIDR notation supported)

	// Status and expiration
	Active    bool       `bun:"active,notnull,default:true" json:"active"`
	ExpiresAt *time.Time `bun:"expires_at"                  json:"expires_at,omitempty"`

	// Usage tracking
	UsageCount int64      `bun:"usage_count,notnull,default:0" json:"usage_count"`
	LastUsedAt *time.Time `bun:"last_used_at"                  json:"last_used_at,omitempty"`
	LastUsedIP string     `bun:"last_used_ip"                  json:"last_used_ip,omitempty"`
	LastUsedUA string     `bun:"last_used_ua"                  json:"last_used_ua,omitempty"`

	// Metadata
	Metadata map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// RBAC Integration (Hybrid Approach)
	DelegateUserPermissions bool    `bun:"delegate_user_permissions,notnull,default:false" json:"delegateUserPermissions"`     // Inherit creator's permissions
	ImpersonateUserID       *xid.ID `bun:"impersonate_user_id,type:varchar(20)"            json:"impersonateUserID,omitempty"` // Act as specific user

	// RBAC Relationships
	Roles []*Role `bun:"m2m:apikey_roles,join:APIKey=Role" json:"-"` // Many-to-many with roles

	// Transient field - only populated during creation
	Key string `bun:"-" json:"key,omitempty"`
}

// BeforeAppendModel implements bun.BeforeAppendModelHook.
func (a *APIKey) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = xid.New()
		}

		a.CreatedAt = time.Now()
		a.UpdatedAt = time.Now()
	case *bun.UpdateQuery:
		a.UpdatedAt = time.Now()
	}

	return nil
}

// IsExpired checks if the API key has expired.
func (a *APIKey) IsExpired() bool {
	return a.ExpiresAt != nil && time.Now().After(*a.ExpiresAt)
}

// HasScope checks if the API key has a specific scope.
func (a *APIKey) HasScope(scope string) bool {

	return slices.Contains(a.Scopes, scope)
}

// HasPermission checks if the API key has a specific permission.
func (a *APIKey) HasPermission(permission string) bool {
	if a.Permissions == nil {
		return false
	}

	_, exists := a.Permissions[permission]

	return exists
}

// IsIPAllowed checks if an IP address is in the allowed list
// Supports exact IP matching and CIDR notation.
func (a *APIKey) IsIPAllowed(ip string) bool {
	if len(a.AllowedIPs) == 0 {
		return true // No whitelist = allow all
	}

	return slices.Contains(a.AllowedIPs, ip)
}
