package schema

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// APIKey represents an API key for programmatic access
// Updated for V2 architecture: App → Environment → Organization
type APIKey struct {
	bun.BaseModel `bun:"table:api_keys"`

	// Primary key
	ID xid.ID `bun:"id,pk" json:"id"`

	// 3-tier context (V2 architecture)
	AppID          xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appID"`                     // Platform tenant
	EnvironmentID  xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentID"`     // Required: environment-scoped key
	OrganizationID *xid.ID `bun:"organization_id,type:varchar(20)" json:"organizationID,omitempty"` // Optional: org-scoped key
	UserID         xid.ID  `bun:"user_id,notnull,type:varchar(20)" json:"userID"`                   // User who created the key

	// Key identification
	Name        string `bun:"name,notnull" json:"name"`
	Description string `bun:"description" json:"description,omitempty"`
	Prefix      string `bun:"prefix,notnull,unique" json:"prefix"` // ak_prod_abc123
	KeyHash     string `bun:"key_hash,notnull" json:"-"`           // Hashed key for verification

	// Permissions and scopes
	Scopes      []string          `bun:"scopes,type:jsonb" json:"scopes"`                     // ["read", "write", "admin"]
	Permissions map[string]string `bun:"permissions,type:jsonb" json:"permissions"`           // Custom permissions
	RateLimit   int               `bun:"rate_limit,default:1000" json:"rate_limit"`           // Requests per hour
	AllowedIPs  []string          `bun:"allowed_ips,type:jsonb" json:"allowed_ips,omitempty"` // IP whitelist (CIDR notation supported)

	// Status and expiration
	Active    bool       `bun:"active,notnull,default:true" json:"active"`
	ExpiresAt *time.Time `bun:"expires_at" json:"expires_at,omitempty"`

	// Usage tracking
	UsageCount int64      `bun:"usage_count,notnull,default:0" json:"usage_count"`
	LastUsedAt *time.Time `bun:"last_used_at" json:"last_used_at,omitempty"`
	LastUsedIP string     `bun:"last_used_ip" json:"last_used_ip,omitempty"`
	LastUsedUA string     `bun:"last_used_ua" json:"last_used_ua,omitempty"`

	// Audit fields
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete" json:"deleted_at,omitempty"`

	// Metadata
	Metadata map[string]string `bun:"metadata,type:jsonb" json:"metadata,omitempty"`

	// Transient field - only populated during creation
	Key string `bun:"-" json:"key,omitempty"`
}

// BeforeAppendModel implements bun.BeforeAppendModelHook
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

// IsIPAllowed checks if an IP address is in the allowed list
// Supports exact IP matching and CIDR notation
func (a *APIKey) IsIPAllowed(ip string) bool {
	if len(a.AllowedIPs) == 0 {
		return true // No whitelist = allow all
	}

	for _, allowed := range a.AllowedIPs {
		if allowed == ip {
			return true // Exact match
		}
		// CIDR matching handled in middleware
	}
	return false
}
