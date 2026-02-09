package providers

import (
	"context"
	"fmt"
	"maps"
	"sync"

	"github.com/rs/xid"
)

// =============================================================================
// RESOURCE PROVIDER REGISTRY
// =============================================================================

// ResourceLoader defines the interface for loading a specific resource type.
type ResourceLoader interface {
	// LoadResource loads a resource by ID and returns its attributes
	LoadResource(ctx context.Context, resourceID string) (map[string]any, error)

	// LoadResources loads multiple resources by IDs
	LoadResources(ctx context.Context, resourceIDs []string) (map[string]map[string]any, error)
}

// ResourceLoaderFunc is a function type that implements ResourceLoader.
type ResourceLoaderFunc func(ctx context.Context, resourceID string) (map[string]any, error)

func (f ResourceLoaderFunc) LoadResource(ctx context.Context, resourceID string) (map[string]any, error) {
	return f(ctx, resourceID)
}

func (f ResourceLoaderFunc) LoadResources(ctx context.Context, resourceIDs []string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	for _, id := range resourceIDs {
		attrs, err := f(ctx, id)
		if err != nil {
			continue
		}

		result[id] = attrs
	}

	return result, nil
}

// ResourceProviderRegistry manages resource loaders for different resource types.
type ResourceProviderRegistry struct {
	loaders map[string]ResourceLoader
	mu      sync.RWMutex
}

// NewResourceProviderRegistry creates a new resource provider registry.
func NewResourceProviderRegistry() *ResourceProviderRegistry {
	return &ResourceProviderRegistry{
		loaders: make(map[string]ResourceLoader),
	}
}

// Register registers a resource loader for a specific resource type.
func (r *ResourceProviderRegistry) Register(resourceType string, loader ResourceLoader) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.loaders[resourceType] = loader
}

// RegisterFunc registers a function as a resource loader.
func (r *ResourceProviderRegistry) RegisterFunc(resourceType string, fn ResourceLoaderFunc) {
	r.Register(resourceType, fn)
}

// Get returns the resource loader for a specific type.
func (r *ResourceProviderRegistry) Get(resourceType string) (ResourceLoader, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	loader, ok := r.loaders[resourceType]

	return loader, ok
}

// List returns all registered resource types.
func (r *ResourceProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.loaders))
	for t := range r.loaders {
		types = append(types, t)
	}

	return types
}

// =============================================================================
// AUTHSOME RESOURCE ATTRIBUTE PROVIDER
// =============================================================================

// AuthsomeResourceAttributeProvider provides resource attributes using the registry.
type AuthsomeResourceAttributeProvider struct {
	registry      *ResourceProviderRegistry
	defaultLoader ResourceLoader
	fallbackAttrs map[string]any // Default attributes for unknown resources
}

// AuthsomeResourceProviderConfig configures the resource provider.
type AuthsomeResourceProviderConfig struct {
	Registry      *ResourceProviderRegistry
	DefaultLoader ResourceLoader
}

// NewAuthsomeResourceAttributeProvider creates a new AuthSome resource attribute provider.
func NewAuthsomeResourceAttributeProvider(cfg AuthsomeResourceProviderConfig) *AuthsomeResourceAttributeProvider {
	registry := cfg.Registry
	if registry == nil {
		registry = NewResourceProviderRegistry()
	}

	return &AuthsomeResourceAttributeProvider{
		registry:      registry,
		defaultLoader: cfg.DefaultLoader,
		fallbackAttrs: map[string]any{
			"type":       "unknown",
			"visibility": "private",
			"status":     "active",
		},
	}
}

// Name returns the provider name.
func (p *AuthsomeResourceAttributeProvider) Name() string {
	return "resource"
}

// GetAttributes fetches resource attributes
// key format: "resourceType:resourceID" (e.g., "document:abc123xyz").
func (p *AuthsomeResourceAttributeProvider) GetAttributes(ctx context.Context, key string) (map[string]any, error) {
	resourceType, resourceID, err := parseResourceKey(key)
	if err != nil {
		return nil, err
	}

	// Try to get a specific loader for this resource type
	if loader, ok := p.registry.Get(resourceType); ok {
		attrs, err := loader.LoadResource(ctx, resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to load resource %s: %w", key, err)
		}

		// Ensure type is set
		if _, ok := attrs["type"]; !ok {
			attrs["type"] = resourceType
		}

		if _, ok := attrs["id"]; !ok {
			attrs["id"] = resourceID
		}

		return attrs, nil
	}

	// Try default loader
	if p.defaultLoader != nil {
		attrs, err := p.defaultLoader.LoadResource(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to load resource %s: %w", key, err)
		}

		// Ensure type is set
		if _, ok := attrs["type"]; !ok {
			attrs["type"] = resourceType
		}

		if _, ok := attrs["id"]; !ok {
			attrs["id"] = resourceID
		}

		return attrs, nil
	}

	// Return fallback attributes with the known info
	attrs := make(map[string]any)
	maps.Copy(attrs, p.fallbackAttrs)

	attrs["type"] = resourceType
	attrs["id"] = resourceID

	return attrs, nil
}

// GetBatchAttributes fetches attributes for multiple resources.
func (p *AuthsomeResourceAttributeProvider) GetBatchAttributes(ctx context.Context, keys []string) (map[string]map[string]any, error) {
	// Group by resource type for batch loading
	grouped := make(map[string][]string)  // resourceType -> []resourceIDs
	keyMapping := make(map[string]string) // resourceID -> original key

	for _, key := range keys {
		resourceType, resourceID, err := parseResourceKey(key)
		if err != nil {
			continue
		}

		grouped[resourceType] = append(grouped[resourceType], resourceID)
		keyMapping[resourceType+":"+resourceID] = key
	}

	result := make(map[string]map[string]any)

	// Load each type batch
	for resourceType, resourceIDs := range grouped {
		if loader, ok := p.registry.Get(resourceType); ok {
			batchResult, err := loader.LoadResources(ctx, resourceIDs)
			if err != nil {
				continue
			}

			for resourceID, attrs := range batchResult {
				originalKey := keyMapping[resourceType+":"+resourceID]
				if originalKey == "" {
					originalKey = resourceType + ":" + resourceID
				}

				// Ensure type/id are set
				if _, ok := attrs["type"]; !ok {
					attrs["type"] = resourceType
				}

				if _, ok := attrs["id"]; !ok {
					attrs["id"] = resourceID
				}

				result[originalKey] = attrs
			}
		} else {
			// Use individual fallback
			for _, resourceID := range resourceIDs {
				originalKey := keyMapping[resourceType+":"+resourceID]
				if originalKey == "" {
					originalKey = resourceType + ":" + resourceID
				}

				attrs := make(map[string]any)
				maps.Copy(attrs, p.fallbackAttrs)

				attrs["type"] = resourceType
				attrs["id"] = resourceID

				result[originalKey] = attrs
			}
		}
	}

	return result, nil
}

// GetRegistry returns the resource registry for external registration.
func (p *AuthsomeResourceAttributeProvider) GetRegistry() *ResourceProviderRegistry {
	return p.registry
}

// =============================================================================
// BUILT-IN RESOURCE LOADERS
// =============================================================================

// OrganizationResourceLoader loads organization resources.
type OrganizationResourceLoader struct {
	getOrgFunc func(ctx context.Context, orgID xid.ID) (map[string]any, error)
}

// NewOrganizationResourceLoader creates an organization resource loader.
func NewOrganizationResourceLoader(getOrg func(ctx context.Context, orgID xid.ID) (map[string]any, error)) *OrganizationResourceLoader {
	return &OrganizationResourceLoader{getOrgFunc: getOrg}
}

func (l *OrganizationResourceLoader) LoadResource(ctx context.Context, resourceID string) (map[string]any, error) {
	orgID, err := xid.FromString(resourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	if l.getOrgFunc == nil {
		return map[string]any{
			"id":   resourceID,
			"type": "organization",
		}, nil
	}

	return l.getOrgFunc(ctx, orgID)
}

func (l *OrganizationResourceLoader) LoadResources(ctx context.Context, resourceIDs []string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	for _, id := range resourceIDs {
		attrs, err := l.LoadResource(ctx, id)
		if err != nil {
			continue
		}

		result[id] = attrs
	}

	return result, nil
}

// UserResourceLoader loads user resources (for user-as-resource scenarios).
type UserResourceLoader struct {
	getUserFunc func(ctx context.Context, userID xid.ID) (map[string]any, error)
}

// NewUserResourceLoader creates a user resource loader.
func NewUserResourceLoader(getUser func(ctx context.Context, userID xid.ID) (map[string]any, error)) *UserResourceLoader {
	return &UserResourceLoader{getUserFunc: getUser}
}

func (l *UserResourceLoader) LoadResource(ctx context.Context, resourceID string) (map[string]any, error) {
	userID, err := xid.FromString(resourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	if l.getUserFunc == nil {
		return map[string]any{
			"id":   resourceID,
			"type": "user",
		}, nil
	}

	return l.getUserFunc(ctx, userID)
}

func (l *UserResourceLoader) LoadResources(ctx context.Context, resourceIDs []string) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)

	for _, id := range resourceIDs {
		attrs, err := l.LoadResource(ctx, id)
		if err != nil {
			continue
		}

		result[id] = attrs
	}

	return result, nil
}

// =============================================================================
// GENERIC RESOURCE HELPERS
// =============================================================================

// GenericResourceAttrs creates a basic resource attributes map.
func GenericResourceAttrs(resourceType, resourceID, owner, orgID string) map[string]any {
	return map[string]any{
		"id":         resourceID,
		"type":       resourceType,
		"owner":      owner,
		"org_id":     orgID,
		"visibility": "private",
		"status":     "active",
	}
}

// ResourceWithOwnership creates resource attributes with ownership info.
func ResourceWithOwnership(resourceType, resourceID, ownerID, orgID, teamID string, isPublic bool) map[string]any {
	visibility := "private"
	if isPublic {
		visibility = "public"
	}

	return map[string]any{
		"id":         resourceID,
		"type":       resourceType,
		"owner":      ownerID,
		"org_id":     orgID,
		"team_id":    teamID,
		"visibility": visibility,
		"status":     "active",
	}
}
