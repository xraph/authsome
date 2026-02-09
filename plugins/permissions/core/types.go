package core

import (
	"time"

	"github.com/rs/xid"
)

// Policy represents a permission policy
// V2 Architecture: App → Environment → Organization.
type Policy struct {
	ID                 xid.ID    `bun:"id,pk"                  json:"id"`
	AppID              xid.ID    `bun:"app_id,notnull"         json:"appId"`                        // Platform app (required)
	EnvironmentID      xid.ID    `bun:"environment_id,notnull" json:"environmentId"`                // Environment (required)
	UserOrganizationID *xid.ID   `bun:"user_organization_id"   json:"userOrganizationId,omitempty"` // User-created org (optional)
	NamespaceID        xid.ID    `bun:"namespace_id,notnull"   json:"namespaceId"`
	Name               string    `bun:"name,notnull"           json:"name"`
	Description        string    `bun:"description"            json:"description"`
	Expression         string    `bun:"expression,notnull"     json:"expression"`
	ResourceType       string    `bun:"resource_type,notnull"  json:"resourceType"`
	Actions            []string  `bun:"actions,array"          json:"actions"`
	Priority           int       `bun:"priority,default:0"     json:"priority"`
	Enabled            bool      `bun:"enabled,default:true"   json:"enabled"`
	Version            int       `bun:"version,default:1"      json:"version"`
	CreatedBy          xid.ID    `bun:"created_by"             json:"createdBy"`
	CreatedAt          time.Time `bun:"created_at,notnull"     json:"createdAt"`
	UpdatedAt          time.Time `bun:"updated_at,notnull"     json:"updatedAt"`
}

// Namespace represents an organization-scoped policy namespace
// V2 Architecture: App → Environment → Organization.
type Namespace struct {
	ID                 xid.ID                `bun:"id,pk"                          json:"id"`
	AppID              xid.ID                `bun:"app_id,notnull"                 json:"appId"`                        // Platform app (required)
	EnvironmentID      xid.ID                `bun:"environment_id,notnull"         json:"environmentId"`                // Environment (required)
	UserOrganizationID *xid.ID               `bun:"user_organization_id"           json:"userOrganizationId,omitempty"` // User-created org (optional)
	Name               string                `bun:"name"                           json:"name"`
	Description        string                `bun:"description"                    json:"description"`
	TemplateID         *xid.ID               `bun:"template_id"                    json:"templateId,omitempty"`
	InheritPlatform    bool                  `bun:"inherit_platform,default:false" json:"inheritPlatform"`
	Resources          []*ResourceDefinition `bun:"-"                              json:"resources"`
	Actions            []*ActionDefinition   `bun:"-"                              json:"actions"`
	CreatedAt          time.Time             `bun:"created_at,notnull"             json:"createdAt"`
	UpdatedAt          time.Time             `bun:"updated_at,notnull"             json:"updatedAt"`
}

// ResourceDefinition defines a custom resource type for an organization.
type ResourceDefinition struct {
	ID          xid.ID              `bun:"id,pk"                 json:"id"`
	NamespaceID xid.ID              `bun:"namespace_id,notnull"  json:"namespaceId"`
	Type        string              `bun:"type,notnull"          json:"type"`
	Description string              `bun:"description"           json:"description"`
	Attributes  []ResourceAttribute `bun:"attributes,type:jsonb" json:"attributes"`
	CreatedAt   time.Time           `bun:"created_at,notnull"    json:"createdAt"`
}

// ResourceAttribute defines an attribute that can be used in policy expressions.
type ResourceAttribute struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, int, bool, array, object
	Required    bool   `json:"required"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// ActionDefinition defines a custom action for an organization.
type ActionDefinition struct {
	ID          xid.ID    `bun:"id,pk"                json:"id"`
	NamespaceID xid.ID    `bun:"namespace_id,notnull" json:"namespaceId"`
	Name        string    `bun:"name,notnull"         json:"name"`
	Description string    `bun:"description"          json:"description"`
	CreatedAt   time.Time `bun:"created_at,notnull"   json:"createdAt"`
}

// PolicyTemplate represents a reusable policy pattern.
type PolicyTemplate struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Category    string              `json:"category"`
	Expression  string              `json:"expression"`
	Parameters  []TemplateParameter `json:"parameters"`
	Examples    []string            `json:"examples"`
}

// TemplateParameter defines a parameter that can be substituted in a template.
type TemplateParameter struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	DefaultValue any    `json:"defaultValue,omitempty"`
}

// MigrationStatus tracks RBAC to Permissions migration progress
// V2 Architecture: App → Environment → Organization.
type MigrationStatus struct {
	AppID              xid.ID     `json:"appId"`                        // Platform app
	EnvironmentID      xid.ID     `json:"environmentId"`                // Environment
	UserOrganizationID *xid.ID    `json:"userOrganizationId,omitempty"` // User-created org (optional)
	StartedAt          time.Time  `json:"startedAt"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	Status             string     `json:"status"` // pending, in_progress, completed, failed
	TotalPolicies      int        `json:"totalPolicies"`
	MigratedCount      int        `json:"migratedCount"`
	FailedCount        int        `json:"failedCount"`
	ValidationPassed   bool       `json:"validationPassed"`
	Errors             []string   `json:"errors,omitempty"`
}

// AuditEvent records a permission-related event
// V2 Architecture: App → Environment → Organization.
type AuditEvent struct {
	ID                 xid.ID         `bun:"id,pk"                  json:"id"`
	AppID              xid.ID         `bun:"app_id,notnull"         json:"appId"`                        // Platform app (required)
	EnvironmentID      xid.ID         `bun:"environment_id,notnull" json:"environmentId"`                // Environment (required)
	UserOrganizationID *xid.ID        `bun:"user_organization_id"   json:"userOrganizationId,omitempty"` // User-created org (optional)
	ActorID            xid.ID         `bun:"actor_id,notnull"       json:"actorId"`
	Action             string         `bun:"action,notnull"         json:"action"`
	ResourceType       string         `bun:"resource_type"          json:"resourceType"`
	ResourceID         xid.ID         `bun:"resource_id"            json:"resourceId"`
	OldValue           map[string]any `bun:"old_value,type:jsonb"   json:"oldValue,omitempty"`
	NewValue           map[string]any `bun:"new_value,type:jsonb"   json:"newValue,omitempty"`
	IPAddress          string         `bun:"ip_address"             json:"ipAddress"`
	UserAgent          string         `bun:"user_agent"             json:"userAgent"`
	Timestamp          time.Time      `bun:"timestamp,notnull"      json:"timestamp"`
}
