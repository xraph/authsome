package core

import (
	"time"

	"github.com/rs/xid"
)

// Policy represents a permission policy
type Policy struct {
	ID           xid.ID    `json:"id" bun:"id,pk"`
	OrgID        xid.ID    `json:"orgId" bun:"org_id,notnull"`
	NamespaceID  xid.ID    `json:"namespaceId" bun:"namespace_id,notnull"`
	Name         string    `json:"name" bun:"name,notnull"`
	Description  string    `json:"description" bun:"description"`
	Expression   string    `json:"expression" bun:"expression,notnull"`
	ResourceType string    `json:"resourceType" bun:"resource_type,notnull"`
	Actions      []string  `json:"actions" bun:"actions,array"`
	Priority     int       `json:"priority" bun:"priority,default:0"`
	Enabled      bool      `json:"enabled" bun:"enabled,default:true"`
	Version      int       `json:"version" bun:"version,default:1"`
	CreatedBy    xid.ID    `json:"createdBy" bun:"created_by"`
	CreatedAt    time.Time `json:"createdAt" bun:"created_at,notnull"`
	UpdatedAt    time.Time `json:"updatedAt" bun:"updated_at,notnull"`
}

// Namespace represents an organization-scoped policy namespace
type Namespace struct {
	ID              xid.ID                `json:"id" bun:"id,pk"`
	OrgID           xid.ID                `json:"orgId" bun:"org_id,notnull"`
	Name            string                `json:"name" bun:"name"`
	Description     string                `json:"description" bun:"description"`
	TemplateID      *xid.ID               `json:"templateId,omitempty" bun:"template_id"`
	InheritPlatform bool                  `json:"inheritPlatform" bun:"inherit_platform,default:false"`
	Resources       []*ResourceDefinition `json:"resources" bun:"-"`
	Actions         []*ActionDefinition   `json:"actions" bun:"-"`
	CreatedAt       time.Time             `json:"createdAt" bun:"created_at,notnull"`
	UpdatedAt       time.Time             `json:"updatedAt" bun:"updated_at,notnull"`
}

// ResourceDefinition defines a custom resource type for an organization
type ResourceDefinition struct {
	ID          xid.ID              `json:"id" bun:"id,pk"`
	NamespaceID xid.ID              `json:"namespaceId" bun:"namespace_id,notnull"`
	Type        string              `json:"type" bun:"type,notnull"`
	Description string              `json:"description" bun:"description"`
	Attributes  []ResourceAttribute `json:"attributes" bun:"attributes,type:jsonb"`
	CreatedAt   time.Time           `json:"createdAt" bun:"created_at,notnull"`
}

// ResourceAttribute defines an attribute that can be used in policy expressions
type ResourceAttribute struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // string, int, bool, array, object
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// ActionDefinition defines a custom action for an organization
type ActionDefinition struct {
	ID          xid.ID    `json:"id" bun:"id,pk"`
	NamespaceID xid.ID    `json:"namespaceId" bun:"namespace_id,notnull"`
	Name        string    `json:"name" bun:"name,notnull"`
	Description string    `json:"description" bun:"description"`
	CreatedAt   time.Time `json:"createdAt" bun:"created_at,notnull"`
}

// PolicyTemplate represents a reusable policy pattern
type PolicyTemplate struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Category    string              `json:"category"`
	Expression  string              `json:"expression"`
	Parameters  []TemplateParameter `json:"parameters"`
	Examples    []string            `json:"examples"`
}

// TemplateParameter defines a parameter that can be substituted in a template
type TemplateParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
}

// MigrationStatus tracks RBAC to Permissions migration progress
type MigrationStatus struct {
	OrgID            xid.ID     `json:"orgId"`
	StartedAt        time.Time  `json:"startedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
	Status           string     `json:"status"` // pending, in_progress, completed, failed
	TotalPolicies    int        `json:"totalPolicies"`
	MigratedCount    int        `json:"migratedCount"`
	FailedCount      int        `json:"failedCount"`
	ValidationPassed bool       `json:"validationPassed"`
	Errors           []string   `json:"errors,omitempty"`
}

// AuditEvent records a permission-related event
type AuditEvent struct {
	ID           xid.ID                 `json:"id" bun:"id,pk"`
	OrgID        xid.ID                 `json:"orgId" bun:"org_id,notnull"`
	ActorID      xid.ID                 `json:"actorId" bun:"actor_id,notnull"`
	Action       string                 `json:"action" bun:"action,notnull"`
	ResourceType string                 `json:"resourceType" bun:"resource_type"`
	ResourceID   xid.ID                 `json:"resourceId" bun:"resource_id"`
	OldValue     map[string]interface{} `json:"oldValue,omitempty" bun:"old_value,type:jsonb"`
	NewValue     map[string]interface{} `json:"newValue,omitempty" bun:"new_value,type:jsonb"`
	IPAddress    string                 `json:"ipAddress" bun:"ip_address"`
	UserAgent    string                 `json:"userAgent" bun:"user_agent"`
	Timestamp    time.Time              `json:"timestamp" bun:"timestamp,notnull"`
}
