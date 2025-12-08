// Package core provides core types and utilities for the secrets plugin.
package core

import (
	"time"
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

// String returns the string representation of the value type
func (t SecretValueType) String() string {
	return string(t)
}

// IsValid checks if the value type is valid
func (t SecretValueType) IsValid() bool {
	switch t {
	case SecretValueTypePlain, SecretValueTypeJSON, SecretValueTypeYAML, SecretValueTypeBinary:
		return true
	default:
		return false
	}
}

// ParseSecretValueType parses a string into a SecretValueType
func ParseSecretValueType(s string) (SecretValueType, bool) {
	t := SecretValueType(s)
	if t.IsValid() {
		return t, true
	}
	return SecretValueTypePlain, false
}

// =============================================================================
// DTOs - Data Transfer Objects
// =============================================================================

// SecretDTO is the API response for a secret (value excluded for security)
type SecretDTO struct {
	ID          string                 `json:"id"`
	Path        string                 `json:"path"`
	Key         string                 `json:"key"`
	ValueType   string                 `json:"valueType"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Version     int                    `json:"version"`
	IsActive    bool                   `json:"isActive"`
	HasSchema   bool                   `json:"hasSchema"`
	HasExpiry   bool                   `json:"hasExpiry"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
	CreatedBy   string                 `json:"createdBy,omitempty"`
	UpdatedBy   string                 `json:"updatedBy,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// SecretWithValueDTO includes the decrypted value (for authorized access)
type SecretWithValueDTO struct {
	SecretDTO
	Value interface{} `json:"value"` // string, map, or slice depending on type
}

// SecretVersionDTO represents a historical version of a secret
type SecretVersionDTO struct {
	ID           string    `json:"id"`
	Version      int       `json:"version"`
	ValueType    string    `json:"valueType"`
	HasSchema    bool      `json:"hasSchema"`
	ChangedBy    string    `json:"changedBy,omitempty"`
	ChangeReason string    `json:"changeReason,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SecretAccessLogDTO represents an access log entry
type SecretAccessLogDTO struct {
	ID           string    `json:"id"`
	SecretID     string    `json:"secretId"`
	Path         string    `json:"path"`
	Action       string    `json:"action"`
	AccessedBy   string    `json:"accessedBy,omitempty"`
	AccessMethod string    `json:"accessMethod"`
	IPAddress    string    `json:"ipAddress,omitempty"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// =============================================================================
// Request DTOs
// =============================================================================

// CreateSecretRequest is the request to create a new secret
type CreateSecretRequest struct {
	Path        string                 `json:"path" validate:"required"`
	Value       interface{}            `json:"value" validate:"required"`
	ValueType   string                 `json:"valueType,omitempty"` // Defaults to "plain" if not specified
	Schema      string                 `json:"schema,omitempty"`    // Optional JSON Schema for validation
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt   *time.Time             `json:"expiresAt,omitempty"`
}

// UpdateSecretRequest is the request to update an existing secret
type UpdateSecretRequest struct {
	Value        interface{}            `json:"value,omitempty"`
	ValueType    string                 `json:"valueType,omitempty"`
	Schema       string                 `json:"schema,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
	ClearExpiry  bool                   `json:"clearExpiry,omitempty"` // Set to true to remove expiry
	ChangeReason string                 `json:"changeReason,omitempty"`
}

// RollbackSecretRequest is the request to rollback a secret to a previous version
type RollbackSecretRequest struct {
	TargetVersion int    `json:"targetVersion" validate:"required,min=1"`
	Reason        string `json:"reason,omitempty"`
}

// =============================================================================
// Query DTOs
// =============================================================================

// ListSecretsQuery defines query parameters for listing secrets
type ListSecretsQuery struct {
	Prefix    string   `json:"prefix,omitempty"`    // Path prefix filter (e.g., "database/")
	Tags      []string `json:"tags,omitempty"`      // Tags filter (AND condition)
	ValueType string   `json:"valueType,omitempty"` // Filter by value type
	Recursive bool     `json:"recursive,omitempty"` // Include nested paths (default: true)
	Search    string   `json:"search,omitempty"`    // Search in path, description
	Page      int      `json:"page,omitempty"`      // Page number (1-based)
	PageSize  int      `json:"pageSize,omitempty"`  // Items per page
	SortBy    string   `json:"sortBy,omitempty"`    // Sort field: path, created_at, updated_at
	SortOrder string   `json:"sortOrder,omitempty"` // Sort order: asc, desc
}

// GetVersionsQuery defines query parameters for listing secret versions
type GetVersionsQuery struct {
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
}

// GetAccessLogsQuery defines query parameters for listing access logs
type GetAccessLogsQuery struct {
	Action   string     `json:"action,omitempty"`   // Filter by action type
	FromDate *time.Time `json:"fromDate,omitempty"` // Filter from date
	ToDate   *time.Time `json:"toDate,omitempty"`   // Filter to date
	Page     int        `json:"page,omitempty"`
	PageSize int        `json:"pageSize,omitempty"`
}

// =============================================================================
// Response DTOs
// =============================================================================

// ListSecretsResponse is the response for listing secrets
type ListSecretsResponse struct {
	Secrets    []*SecretDTO `json:"secrets"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalItems int          `json:"totalItems"`
	TotalPages int          `json:"totalPages"`
}

// ListVersionsResponse is the response for listing secret versions
type ListVersionsResponse struct {
	Versions   []*SecretVersionDTO `json:"versions"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	TotalItems int                 `json:"totalItems"`
	TotalPages int                 `json:"totalPages"`
}

// ListAccessLogsResponse is the response for listing access logs
type ListAccessLogsResponse struct {
	Logs       []*SecretAccessLogDTO `json:"logs"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
	TotalItems int                   `json:"totalItems"`
	TotalPages int                   `json:"totalPages"`
}

// RevealValueResponse is the response for revealing a secret value
type RevealValueResponse struct {
	Value     interface{} `json:"value"`
	ValueType string      `json:"valueType"`
}

// =============================================================================
// Tree Structure DTOs (for dashboard tree view)
// =============================================================================

// SecretTreeNode represents a node in the secrets tree view
type SecretTreeNode struct {
	Name     string            `json:"name"`               // Node name (folder name or secret key)
	Path     string            `json:"path"`               // Full path to this node
	IsSecret bool              `json:"isSecret"`           // True if this is a secret, false if folder
	Secret   *SecretDTO        `json:"secret,omitempty"`   // Secret data if isSecret is true
	Children []*SecretTreeNode `json:"children,omitempty"` // Child nodes if folder
}

// StatsDTO contains statistics about secrets
type StatsDTO struct {
	TotalSecrets    int            `json:"totalSecrets"`
	TotalVersions   int            `json:"totalVersions"`
	SecretsByType   map[string]int `json:"secretsByType"`
	ExpiringSecrets int            `json:"expiringSecrets"` // Secrets expiring in next 30 days
	ExpiredSecrets  int            `json:"expiredSecrets"`
	RecentlyUpdated int            `json:"recentlyUpdated"` // Updated in last 7 days
}
