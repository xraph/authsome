// Package schema defines the database schema for the secrets plugin.
package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainSchema "github.com/xraph/authsome/schema"
)

// SecretValueType defines the type of secret value
type SecretValueType string

const (
	// SecretValueTypePlain is a plain string value
	SecretValueTypePlain SecretValueType = "plain"
	// SecretValueTypeJSON is a JSON object/array value
	SecretValueTypeJSON SecretValueType = "json"
	// SecretValueTypeYAML is a YAML document value
	SecretValueTypeYAML SecretValueType = "yaml"
	// SecretValueTypeBinary is a base64-encoded binary value
	SecretValueTypeBinary SecretValueType = "binary"
)

// Secret represents a secret entry in the database
type Secret struct {
	bun.BaseModel `bun:"table:secrets,alias:s"`

	ID             xid.ID                 `bun:"id,pk,type:varchar(20)" json:"id"`
	AppID          xid.ID                 `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID  xid.ID                 `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	Path           string                 `bun:"path,notnull" json:"path"`            // Hierarchical path e.g., "database/postgres/password"
	Key            string                 `bun:"key,notnull" json:"key"`              // Leaf key name extracted from path
	ValueType      SecretValueType        `bun:"value_type,notnull" json:"valueType"` // plain, json, yaml, binary
	EncryptedValue []byte                 `bun:"encrypted_value,notnull" json:"-"`    // AES-256-GCM encrypted value
	Nonce          []byte                 `bun:"nonce,notnull" json:"-"`              // Encryption nonce (12 bytes for GCM)
	SchemaJSON     string                 `bun:"schema_json,nullzero" json:"schema"`  // Optional JSON Schema for validation
	Description    string                 `bun:"description,nullzero" json:"description"`
	Tags           []string               `bun:"tags,array" json:"tags"`
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata"`
	Version        int                    `bun:"version,notnull,default:1" json:"version"`
	IsActive       bool                   `bun:"is_active,notnull,default:true" json:"isActive"`
	ExpiresAt      *time.Time             `bun:"expires_at,nullzero" json:"expiresAt"`
	CreatedBy      xid.ID                 `bun:"created_by,type:varchar(20)" json:"createdBy"`
	UpdatedBy      xid.ID                 `bun:"updated_by,type:varchar(20)" json:"updatedBy"`
	CreatedAt      time.Time              `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time              `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt      *time.Time             `bun:"deleted_at,soft_delete,nullzero" json:"-"`

	// Relations
	App         *mainSchema.App         `bun:"rel:belongs-to,join:app_id=id" json:"app,omitempty"`
	Environment *mainSchema.Environment `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`
}

// SecretVersion stores historical versions of secrets for audit and rollback
type SecretVersion struct {
	bun.BaseModel `bun:"table:secret_versions,alias:sv"`

	ID             xid.ID          `bun:"id,pk,type:varchar(20)" json:"id"`
	SecretID       xid.ID          `bun:"secret_id,notnull,type:varchar(20)" json:"secretId"`
	Version        int             `bun:"version,notnull" json:"version"`
	EncryptedValue []byte          `bun:"encrypted_value,notnull" json:"-"`
	Nonce          []byte          `bun:"nonce,notnull" json:"-"`
	ValueType      SecretValueType `bun:"value_type,notnull" json:"valueType"`
	SchemaJSON     string          `bun:"schema_json,nullzero" json:"schema"`
	ChangedBy      xid.ID          `bun:"changed_by,type:varchar(20)" json:"changedBy"`
	ChangeReason   string          `bun:"change_reason,nullzero" json:"changeReason"`
	CreatedAt      time.Time       `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`

	// Relations
	Secret *Secret `bun:"rel:belongs-to,join:secret_id=id" json:"secret,omitempty"`
}

// SecretAccessLog tracks access to secrets for audit purposes
type SecretAccessLog struct {
	bun.BaseModel `bun:"table:secret_access_logs,alias:sal"`

	ID            xid.ID    `bun:"id,pk,type:varchar(20)" json:"id"`
	SecretID      xid.ID    `bun:"secret_id,notnull,type:varchar(20)" json:"secretId"`
	AppID         xid.ID    `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID xid.ID    `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	Path          string    `bun:"path,notnull" json:"path"`
	Action        string    `bun:"action,notnull" json:"action"`        // read, create, update, delete, rollback, reveal
	AccessedBy    xid.ID    `bun:"accessed_by,type:varchar(20)" json:"accessedBy"`
	AccessMethod  string    `bun:"access_method,notnull" json:"accessMethod"` // api, dashboard, configsource
	IPAddress     string    `bun:"ip_address,nullzero" json:"ipAddress"`
	UserAgent     string    `bun:"user_agent,nullzero" json:"userAgent"`
	Success       bool      `bun:"success,notnull" json:"success"`
	ErrorMessage  string    `bun:"error_message,nullzero" json:"errorMessage"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}

// TableName returns the table name for Secret
func (s *Secret) TableName() string {
	return "secrets"
}

// TableName returns the table name for SecretVersion
func (sv *SecretVersion) TableName() string {
	return "secret_versions"
}

// TableName returns the table name for SecretAccessLog
func (sal *SecretAccessLog) TableName() string {
	return "secret_access_logs"
}

// IsExpired checks if the secret has expired
func (s *Secret) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// GetFullPath returns the full hierarchical path
func (s *Secret) GetFullPath() string {
	return s.Path
}

