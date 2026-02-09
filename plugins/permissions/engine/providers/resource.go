package providers

import (
	"context"
	"fmt"
)

// ResourceService defines the interface for fetching resource data
// This should be implemented by your application's resource services.
type ResourceService interface {
	// GetResource fetches a resource by type and ID
	GetResource(ctx context.Context, resourceType, resourceID string) (*Resource, error)

	// GetResources fetches multiple resources
	GetResources(ctx context.Context, requests []ResourceRequest) ([]*Resource, error)
}

// ResourceRequest represents a request for a specific resource.
type ResourceRequest struct {
	Type string
	ID   string
}

// Resource represents resource data for attribute resolution.
type Resource struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Owner        string         `json:"owner"`
	OrgID        string         `json:"org_id"`
	TeamID       string         `json:"team_id"`
	ProjectID    string         `json:"project_id"`
	Visibility   string         `json:"visibility"` // public, private, team, org
	Status       string         `json:"status"`     // active, archived, deleted
	Tags         []string       `json:"tags"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	CreatedBy    string         `json:"created_by"`
	Confidential string         `json:"confidential"` // public, internal, confidential, secret
}

// ResourceAttributeProvider fetches resource attributes from resource services.
type ResourceAttributeProvider struct {
	resourceService ResourceService
}

// NewResourceAttributeProvider creates a new resource attribute provider.
func NewResourceAttributeProvider(resourceService ResourceService) *ResourceAttributeProvider {
	return &ResourceAttributeProvider{
		resourceService: resourceService,
	}
}

// Name returns the provider name.
func (p *ResourceAttributeProvider) Name() string {
	return "resource"
}

// GetAttributes fetches resource attributes
// key is expected to be in format "type:id" (e.g., "document:123").
func (p *ResourceAttributeProvider) GetAttributes(ctx context.Context, key string) (map[string]any, error) {
	// Parse the key to extract type and id
	resourceType, resourceID, err := parseResourceKey(key)
	if err != nil {
		return nil, err
	}

	resource, err := p.resourceService.GetResource(ctx, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resource: %w", err)
	}

	return resourceToAttributes(resource), nil
}

// GetBatchAttributes fetches attributes for multiple resources.
func (p *ResourceAttributeProvider) GetBatchAttributes(ctx context.Context, keys []string) (map[string]map[string]any, error) {
	// Parse all keys
	requests := make([]ResourceRequest, 0, len(keys))
	for _, key := range keys {
		resourceType, resourceID, err := parseResourceKey(key)
		if err != nil {
			return nil, err
		}

		requests = append(requests, ResourceRequest{Type: resourceType, ID: resourceID})
	}

	// Fetch resources
	resources, err := p.resourceService.GetResources(ctx, requests)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch resources: %w", err)
	}

	// Build result map
	result := make(map[string]map[string]any)

	for _, resource := range resources {
		key := fmt.Sprintf("%s:%s", resource.Type, resource.ID)
		result[key] = resourceToAttributes(resource)
	}

	return result, nil
}

// parseResourceKey parses a resource key in format "type:id".
func parseResourceKey(key string) (string, string, error) {
	// Simple split by ":"
	// In production, you might want more sophisticated parsing
	var resourceType, resourceID string

	for i, ch := range key {
		if ch == ':' {
			resourceType = key[:i]
			resourceID = key[i+1:]

			break
		}
	}

	if resourceType == "" || resourceID == "" {
		return "", "", fmt.Errorf("invalid resource key format: %s (expected 'type:id')", key)
	}

	return resourceType, resourceID, nil
}

// resourceToAttributes converts a Resource to an attributes map.
func resourceToAttributes(resource *Resource) map[string]any {
	if resource == nil {
		return make(map[string]any)
	}

	attrs := map[string]any{
		"id":           resource.ID,
		"type":         resource.Type,
		"name":         resource.Name,
		"owner":        resource.Owner,
		"org_id":       resource.OrgID,
		"team_id":      resource.TeamID,
		"project_id":   resource.ProjectID,
		"visibility":   resource.Visibility,
		"status":       resource.Status,
		"tags":         resource.Tags,
		"created_at":   resource.CreatedAt,
		"updated_at":   resource.UpdatedAt,
		"created_by":   resource.CreatedBy,
		"confidential": resource.Confidential,
	}

	// Merge metadata
	if resource.Metadata != nil {
		for k, v := range resource.Metadata {
			attrs["meta_"+k] = v
		}
	}

	return attrs
}

// MockResourceService provides a mock implementation for testing.
type MockResourceService struct {
	resources map[string]*Resource // key is "type:id"
}

// NewMockResourceService creates a new mock resource service.
func NewMockResourceService() *MockResourceService {
	return &MockResourceService{
		resources: make(map[string]*Resource),
	}
}

// AddResource adds a resource to the mock service.
func (m *MockResourceService) AddResource(resource *Resource) {
	key := fmt.Sprintf("%s:%s", resource.Type, resource.ID)
	m.resources[key] = resource
}

// GetResource fetches a resource by type and ID.
func (m *MockResourceService) GetResource(ctx context.Context, resourceType, resourceID string) (*Resource, error) {
	key := fmt.Sprintf("%s:%s", resourceType, resourceID)

	resource, exists := m.resources[key]
	if !exists {
		return nil, fmt.Errorf("resource not found: %s", key)
	}

	return resource, nil
}

// GetResources fetches multiple resources.
func (m *MockResourceService) GetResources(ctx context.Context, requests []ResourceRequest) ([]*Resource, error) {
	result := make([]*Resource, 0, len(requests))
	for _, req := range requests {
		key := fmt.Sprintf("%s:%s", req.Type, req.ID)
		if resource, exists := m.resources[key]; exists {
			result = append(result, resource)
		}
	}

	return result, nil
}
