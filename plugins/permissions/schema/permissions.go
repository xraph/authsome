package schema

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// =============================================================================
// PERMISSION POLICY
// =============================================================================

// PermissionPolicy represents a permission policy in the database
// V2 Architecture: App → Environment → Organization
type PermissionPolicy struct {
	bun.BaseModel `bun:"table:permission_policies,alias:pp"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// V2 Multi-tenant context
	AppID              xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID      xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	UserOrganizationID *xid.ID `bun:"user_organization_id,type:varchar(20)" json:"userOrganizationId,omitempty"`

	// Policy details
	NamespaceID  xid.ID   `bun:"namespace_id,notnull,type:varchar(20)" json:"namespaceId"`
	Name         string   `bun:"name,notnull" json:"name"`
	Description  string   `bun:"description" json:"description"`
	Expression   string   `bun:"expression,notnull" json:"expression"`
	ResourceType string   `bun:"resource_type,notnull" json:"resourceType"`
	Actions      []string `bun:"actions,array" json:"actions"`
	Priority     int      `bun:"priority,default:0" json:"priority"`
	Enabled      bool     `bun:"enabled,default:true" json:"enabled"`
	Version      int      `bun:"version,default:1" json:"version"`

	// Audit fields
	CreatedBy xid.ID    `bun:"created_by,type:varchar(20)" json:"createdBy"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// TableName returns the table name for PermissionPolicy
func (PermissionPolicy) TableName() string {
	return "permission_policies"
}

// =============================================================================
// PERMISSION NAMESPACE
// =============================================================================

// PermissionNamespace represents a permission namespace in the database
// V2 Architecture: App → Environment → Organization
type PermissionNamespace struct {
	bun.BaseModel `bun:"table:permission_namespaces,alias:pn"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// V2 Multi-tenant context
	AppID              xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID      xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	UserOrganizationID *xid.ID `bun:"user_organization_id,type:varchar(20)" json:"userOrganizationId,omitempty"`

	// Namespace details
	Name            string  `bun:"name,notnull" json:"name"`
	Description     string  `bun:"description" json:"description"`
	TemplateID      *xid.ID `bun:"template_id,type:varchar(20)" json:"templateId,omitempty"`
	InheritPlatform bool    `bun:"inherit_platform,default:false" json:"inheritPlatform"`

	// Audit fields
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// TableName returns the table name for PermissionNamespace
func (PermissionNamespace) TableName() string {
	return "permission_namespaces"
}

// =============================================================================
// PERMISSION RESOURCE
// =============================================================================

// ResourceAttribute represents an attribute definition for a resource type
type ResourceAttribute struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// PermissionResource represents a resource type definition in the database
type PermissionResource struct {
	bun.BaseModel `bun:"table:permission_resources,alias:pr"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// Namespace relationship
	NamespaceID xid.ID `bun:"namespace_id,notnull,type:varchar(20)" json:"namespaceId"`

	// Resource details
	Type        string              `bun:"type,notnull" json:"type"`
	Description string              `bun:"description" json:"description"`
	Attributes  []ResourceAttribute `bun:"attributes,type:jsonb" json:"attributes"`

	// Audit fields
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}

// TableName returns the table name for PermissionResource
func (PermissionResource) TableName() string {
	return "permission_resources"
}

// =============================================================================
// PERMISSION ACTION
// =============================================================================

// PermissionAction represents an action definition in the database
type PermissionAction struct {
	bun.BaseModel `bun:"table:permission_actions,alias:pa"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// Namespace relationship
	NamespaceID xid.ID `bun:"namespace_id,notnull,type:varchar(20)" json:"namespaceId"`

	// Action details
	Name        string `bun:"name,notnull" json:"name"`
	Description string `bun:"description" json:"description"`

	// Audit fields
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
}

// TableName returns the table name for PermissionAction
func (PermissionAction) TableName() string {
	return "permission_actions"
}

// =============================================================================
// PERMISSION AUDIT LOG
// =============================================================================

// PermissionAuditLog represents an audit log entry in the database
// V2 Architecture: App → Environment → Organization
type PermissionAuditLog struct {
	bun.BaseModel `bun:"table:permission_audit_logs,alias:pal"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// V2 Multi-tenant context
	AppID              xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID      xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	UserOrganizationID *xid.ID `bun:"user_organization_id,type:varchar(20)" json:"userOrganizationId,omitempty"`

	// Audit details
	ActorID      xid.ID                 `bun:"actor_id,notnull,type:varchar(20)" json:"actorId"`
	Action       string                 `bun:"action,notnull" json:"action"`
	ResourceType string                 `bun:"resource_type" json:"resourceType"`
	ResourceID   xid.ID                 `bun:"resource_id,type:varchar(20)" json:"resourceId"`
	OldValue     map[string]interface{} `bun:"old_value,type:jsonb" json:"oldValue,omitempty"`
	NewValue     map[string]interface{} `bun:"new_value,type:jsonb" json:"newValue,omitempty"`

	// Request metadata
	IPAddress string `bun:"ip_address" json:"ipAddress"`
	UserAgent string `bun:"user_agent" json:"userAgent"`

	// Timestamp
	Timestamp time.Time `bun:"timestamp,notnull,default:current_timestamp" json:"timestamp"`
}

// TableName returns the table name for PermissionAuditLog
func (PermissionAuditLog) TableName() string {
	return "permission_audit_logs"
}

// =============================================================================
// PERMISSION EVALUATION STATS (for analytics)
// =============================================================================

// PermissionEvaluationStats tracks policy evaluation statistics
// V2 Architecture: App → Environment → Organization
type PermissionEvaluationStats struct {
	bun.BaseModel `bun:"table:permission_evaluation_stats,alias:pes"`

	// Primary key
	ID xid.ID `bun:"id,pk,type:varchar(20)" json:"id"`

	// V2 Multi-tenant context
	AppID              xid.ID  `bun:"app_id,notnull,type:varchar(20)" json:"appId"`
	EnvironmentID      xid.ID  `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"`
	UserOrganizationID *xid.ID `bun:"user_organization_id,type:varchar(20)" json:"userOrganizationId,omitempty"`

	// Policy reference
	PolicyID xid.ID `bun:"policy_id,notnull,type:varchar(20)" json:"policyId"`

	// Statistics
	EvaluationCount int64     `bun:"evaluation_count,default:0" json:"evaluationCount"`
	AllowCount      int64     `bun:"allow_count,default:0" json:"allowCount"`
	DenyCount       int64     `bun:"deny_count,default:0" json:"denyCount"`
	ErrorCount      int64     `bun:"error_count,default:0" json:"errorCount"`
	TotalLatencyMs  float64   `bun:"total_latency_ms,default:0" json:"totalLatencyMs"`
	AvgLatencyMs    float64   `bun:"avg_latency_ms,default:0" json:"avgLatencyMs"`
	P50LatencyMs    float64   `bun:"p50_latency_ms,default:0" json:"p50LatencyMs"`
	P99LatencyMs    float64   `bun:"p99_latency_ms,default:0" json:"p99LatencyMs"`
	LastEvaluated   time.Time `bun:"last_evaluated" json:"lastEvaluated"`

	// Audit fields
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
}

// TableName returns the table name for PermissionEvaluationStats
func (PermissionEvaluationStats) TableName() string {
	return "permission_evaluation_stats"
}

// =============================================================================
// INDEXES AND CONSTRAINTS
// =============================================================================

// The following indexes should be created for optimal query performance:
//
// PermissionPolicy indexes:
// - idx_permission_policies_app_id ON permission_policies(app_id)
// - idx_permission_policies_env_id ON permission_policies(environment_id)
// - idx_permission_policies_user_org_id ON permission_policies(user_organization_id) WHERE user_organization_id IS NOT NULL
// - idx_permission_policies_namespace_id ON permission_policies(namespace_id)
// - idx_permission_policies_resource_type ON permission_policies(resource_type)
// - idx_permission_policies_enabled ON permission_policies(enabled) WHERE enabled = true
// - idx_permission_policies_lookup ON permission_policies(app_id, environment_id, user_organization_id, resource_type, enabled)
//
// PermissionNamespace indexes:
// - idx_permission_namespaces_app_id ON permission_namespaces(app_id)
// - idx_permission_namespaces_env_id ON permission_namespaces(environment_id)
// - idx_permission_namespaces_user_org_id ON permission_namespaces(user_organization_id) WHERE user_organization_id IS NOT NULL
// - idx_permission_namespaces_name ON permission_namespaces(app_id, environment_id, user_organization_id, name) UNIQUE
//
// PermissionResource indexes:
// - idx_permission_resources_namespace_id ON permission_resources(namespace_id)
// - idx_permission_resources_type ON permission_resources(namespace_id, type) UNIQUE
//
// PermissionAction indexes:
// - idx_permission_actions_namespace_id ON permission_actions(namespace_id)
// - idx_permission_actions_name ON permission_actions(namespace_id, name) UNIQUE
//
// PermissionAuditLog indexes:
// - idx_permission_audit_logs_app_id ON permission_audit_logs(app_id)
// - idx_permission_audit_logs_env_id ON permission_audit_logs(environment_id)
// - idx_permission_audit_logs_user_org_id ON permission_audit_logs(user_organization_id) WHERE user_organization_id IS NOT NULL
// - idx_permission_audit_logs_actor_id ON permission_audit_logs(actor_id)
// - idx_permission_audit_logs_timestamp ON permission_audit_logs(timestamp DESC)
// - idx_permission_audit_logs_action ON permission_audit_logs(action)
//
// PermissionEvaluationStats indexes:
// - idx_permission_eval_stats_app_id ON permission_evaluation_stats(app_id)
// - idx_permission_eval_stats_env_id ON permission_evaluation_stats(environment_id)
// - idx_permission_eval_stats_user_org_id ON permission_evaluation_stats(user_organization_id) WHERE user_organization_id IS NOT NULL
// - idx_permission_eval_stats_policy_id ON permission_evaluation_stats(policy_id)
// - idx_permission_eval_stats_lookup ON permission_evaluation_stats(app_id, environment_id, user_organization_id, policy_id) UNIQUE

