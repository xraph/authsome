package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/xraph/authsome/schema"
)

// Resource defines the interface for MCP resources
type Resource interface {
	Read(ctx context.Context, uri string, plugin *Plugin) (string, error)
	Describe() ResourceDescription
}

// ResourceDescription describes a resource for MCP clients
type ResourceDescription struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// ResourceRegistry manages available resources
type ResourceRegistry struct {
	resources map[string]Resource
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resources: make(map[string]Resource),
	}
}

// Register registers a resource handler
func (r *ResourceRegistry) Register(uri string, resource Resource) {
	r.resources[uri] = resource
}

// List returns descriptions of all resources
func (r *ResourceRegistry) List() []ResourceDescription {
	var descriptions []ResourceDescription
	for _, resource := range r.resources {
		descriptions = append(descriptions, resource.Describe())
	}
	return descriptions
}

// Read reads a resource by URI
func (r *ResourceRegistry) Read(ctx context.Context, uri string, plugin *Plugin) (string, error) {
	// Find matching resource (support wildcards)
	for pattern, resource := range r.resources {
		if matchesPattern(pattern, uri) {
			return resource.Read(ctx, uri, plugin)
		}
	}
	return "", fmt.Errorf("resource not found: %s", uri)
}

// matchesPattern matches URI patterns (basic implementation)
func matchesPattern(pattern, uri string) bool {
	// Exact match
	if pattern == uri {
		return true
	}
	// TODO: Add wildcard support
	return false
}

// ConfigResource exposes sanitized configuration
type ConfigResource struct{}

func (r *ConfigResource) Describe() ResourceDescription {
	return ResourceDescription{
		URI:         "authsome://config",
		Name:        "Configuration",
		Description: "AuthSome configuration (sanitized, no secrets)",
		MimeType:    "application/json",
	}
}

func (r *ConfigResource) Read(ctx context.Context, uri string, plugin *Plugin) (string, error) {
	// Get auth instance config
	authConfig := plugin.auth.GetConfig()

	config := map[string]interface{}{
		"authsome": map[string]interface{}{
			"base_path":       authConfig.BasePath,
			"rbac_enforce":    authConfig.RBACEnforce,
			"database_schema": authConfig.DatabaseSchema,
			"trusted_origins": authConfig.TrustedOrigins,
		},
		"mcp_plugin": plugin.config,
		// Add more config sections as needed
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(data), nil
}

// SchemaResource exposes database schema information
type SchemaResource struct{}

func (r *SchemaResource) Describe() ResourceDescription {
	return ResourceDescription{
		URI:         "authsome://schema",
		Name:        "Database Schema",
		Description: "AuthSome database schema documentation",
		MimeType:    "application/json",
	}
}

func (r *SchemaResource) Read(ctx context.Context, uri string, plugin *Plugin) (string, error) {
	// Extract entity type from URI (e.g., authsome://schema/user)
	// For now, return all schemas

	schemas := map[string]interface{}{
		"user":         describeSchema(&schema.User{}),
		"session":      describeSchema(&schema.Session{}),
		"organization": describeSchema(&schema.Organization{}),
		"member":       describeSchema(&schema.Member{}),
		"team":         describeSchema(&schema.Team{}),
		"role":         describeSchema(&schema.Role{}),
		"permission":   describeSchema(&schema.Permission{}),
		"audit":        describeSchema(&schema.AuditEvent{}),
		"webhook":      describeSchema(&schema.Webhook{}),
		"api_key":      describeSchema(&schema.APIKey{}),
	}

	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schemas: %w", err)
	}

	return string(data), nil
}

// describeSchema generates schema description from struct
func describeSchema(v interface{}) map[string]interface{} {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	fields := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse tag (remove omitempty etc)
		fieldName := jsonTag
		if idx := len(jsonTag); idx > 0 {
			for j, c := range jsonTag {
				if c == ',' {
					fieldName = jsonTag[:j]
					break
				}
			}
		}

		fields[fieldName] = map[string]interface{}{
			"type":        field.Type.String(),
			"description": field.Tag.Get("description"), // If we add these
		}
	}

	return map[string]interface{}{
		"name":   t.Name(),
		"fields": fields,
	}
}

// RoutesResource exposes registered API routes
type RoutesResource struct{}

func (r *RoutesResource) Describe() ResourceDescription {
	return ResourceDescription{
		URI:         "authsome://routes",
		Name:        "API Routes",
		Description: "Registered HTTP routes and endpoints",
		MimeType:    "application/json",
	}
}

func (r *RoutesResource) Read(ctx context.Context, uri string, plugin *Plugin) (string, error) {
	// TODO: Implement route introspection
	// This requires enhancing AuthSome to track registered routes

	routes := map[string]interface{}{
		"routes": []map[string]interface{}{
			{
				"method":      "POST",
				"path":        "/api/auth/signup",
				"description": "User registration",
				"auth":        "none",
			},
			{
				"method":      "POST",
				"path":        "/api/auth/signin",
				"description": "User authentication",
				"auth":        "none",
			},
			{
				"method":      "POST",
				"path":        "/api/auth/signout",
				"description": "User logout",
				"auth":        "required",
			},
			{
				"method":      "GET",
				"path":        "/api/auth/session",
				"description": "Get current session",
				"auth":        "required",
			},
			// TODO: Auto-discover routes from actual registrations
		},
	}

	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal routes: %w", err)
	}

	return string(data), nil
}
