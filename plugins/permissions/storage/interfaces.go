package storage

import (
	"context"
	"time"

	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/engine"
)

// Repository defines the data access interface for permissions
type Repository interface {
	// Policy operations
	CreatePolicy(ctx context.Context, policy *core.Policy) error
	GetPolicy(ctx context.Context, id string) (*core.Policy, error)
	ListPolicies(ctx context.Context, orgID string, filters PolicyFilters) ([]*core.Policy, error)
	UpdatePolicy(ctx context.Context, policy *core.Policy) error
	DeletePolicy(ctx context.Context, id string) error
	GetPoliciesByResourceType(ctx context.Context, orgID, resourceType string) ([]*core.Policy, error)
	GetActivePolicies(ctx context.Context, orgID string) ([]*core.Policy, error)
	
	// Namespace operations
	CreateNamespace(ctx context.Context, ns *core.Namespace) error
	GetNamespace(ctx context.Context, id string) (*core.Namespace, error)
	GetNamespaceByOrg(ctx context.Context, orgID string) (*core.Namespace, error)
	UpdateNamespace(ctx context.Context, ns *core.Namespace) error
	DeleteNamespace(ctx context.Context, id string) error
	
	// Resource definition operations
	CreateResourceDefinition(ctx context.Context, res *core.ResourceDefinition) error
	ListResourceDefinitions(ctx context.Context, namespaceID string) ([]*core.ResourceDefinition, error)
	DeleteResourceDefinition(ctx context.Context, id string) error
	
	// Action definition operations
	CreateActionDefinition(ctx context.Context, action *core.ActionDefinition) error
	ListActionDefinitions(ctx context.Context, namespaceID string) ([]*core.ActionDefinition, error)
	DeleteActionDefinition(ctx context.Context, id string) error
	
	// Audit operations
	CreateAuditEvent(ctx context.Context, event *core.AuditEvent) error
	ListAuditEvents(ctx context.Context, orgID string, filters AuditFilters) ([]*core.AuditEvent, error)
}

// PolicyFilters defines filtering options for policy queries
type PolicyFilters struct {
	ResourceType *string
	Actions      []string
	Enabled      *bool
	NamespaceID  *string
	Limit        int
	Offset       int
}

// AuditFilters defines filtering options for audit queries
type AuditFilters struct {
	ActorID      *string
	Action       *string
	ResourceType *string
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// Cache defines the caching interface for compiled policies
type Cache interface {
	// Get retrieves a compiled policy from cache
	Get(ctx context.Context, key string) (*engine.CompiledPolicy, error)
	
	// Set stores a compiled policy in cache
	Set(ctx context.Context, key string, policy *engine.CompiledPolicy, ttl time.Duration) error
	
	// Delete removes a policy from cache
	Delete(ctx context.Context, key string) error
	
	// DeleteByOrg removes all policies for an organization
	DeleteByOrg(ctx context.Context, orgID string) error
	
	// GetMulti retrieves multiple policies
	GetMulti(ctx context.Context, keys []string) (map[string]*engine.CompiledPolicy, error)
	
	// SetMulti stores multiple policies
	SetMulti(ctx context.Context, policies map[string]*engine.CompiledPolicy, ttl time.Duration) error
	
	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	Size        int64
	HitRate     float64
	LastUpdated time.Time
}

// AttributeProvider fetches attributes for ABAC evaluation
type AttributeProvider interface {
	// GetUserAttributes fetches user attributes (roles, department, metadata)
	GetUserAttributes(ctx context.Context, userID string) (map[string]interface{}, error)
	
	// GetResourceAttributes fetches resource attributes (owner, tags, metadata)
	GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (map[string]interface{}, error)
	
	// GetRequestAttributes fetches request context (IP, time, geo)
	GetRequestAttributes(ctx context.Context) (map[string]interface{}, error)
}

