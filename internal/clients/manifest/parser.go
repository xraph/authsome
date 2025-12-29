package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser handles loading and parsing manifest files
type Parser struct {
	manifests map[string]*Manifest
}

// NewParser creates a new manifest parser
func NewParser() *Parser {
	return &Parser{
		manifests: make(map[string]*Manifest),
	}
}

// LoadFile loads a single manifest file
func (p *Parser) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read manifest file %s: %w", path, err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest file %s: %w", path, err)
	}

	// Enrich manifest with extracted path parameters
	enrichManifestWithPathParams(&manifest)

	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("invalid manifest %s: %w", path, err)
	}

	p.manifests[manifest.PluginID] = &manifest
	return nil
}

// LoadDirectory loads all manifest files from a directory
func (p *Parser) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml and .yml files
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := p.LoadFile(path); err != nil {
			return err
		}
	}

	return nil
}

// Get returns a manifest by plugin ID
func (p *Parser) Get(pluginID string) (*Manifest, bool) {
	m, ok := p.manifests[pluginID]
	return m, ok
}

// List returns all loaded manifests
func (p *Parser) List() []*Manifest {
	manifests := make([]*Manifest, 0, len(p.manifests))
	for _, m := range p.manifests {
		manifests = append(manifests, m)
	}
	return manifests
}

// Filter returns manifests matching the given plugin IDs
func (p *Parser) Filter(pluginIDs []string) []*Manifest {
	if len(pluginIDs) == 0 {
		return p.List()
	}

	manifests := make([]*Manifest, 0, len(pluginIDs))
	for _, id := range pluginIDs {
		if m, ok := p.Get(id); ok {
			manifests = append(manifests, m)
		}
	}
	return manifests
}

// GetCore returns the core manifest (plugin_id: "core")
func (p *Parser) GetCore() (*Manifest, error) {
	m, ok := p.Get("core")
	if !ok {
		return nil, fmt.Errorf("core manifest not found")
	}
	return m, nil
}

// GetPluginManifests returns all non-core manifests
func (p *Parser) GetPluginManifests() []*Manifest {
	var manifests []*Manifest
	for _, m := range p.manifests {
		if m.PluginID != "core" {
			manifests = append(manifests, m)
		}
	}
	return manifests
}

// enrichManifestWithPathParams extracts path parameters from route paths
// and populates the Params field if it's empty
func enrichManifestWithPathParams(manifest *Manifest) {
	for i := range manifest.Routes {
		route := &manifest.Routes[i]
		
		// Only extract if Params is not already populated
		if len(route.Params) == 0 {
			route.Params = extractPathParams(route.Path)
		}
	}
}

// extractPathParams extracts path parameters from a route path string
// Supports both :param (Forge/Express style) and {param} (OpenAPI style)
// Returns a map of parameter names to their inferred types
func extractPathParams(path string) map[string]string {
	params := make(map[string]string)
	
	// Split path by '/'
	segments := strings.Split(path, "/")
	
	for _, segment := range segments {
		var paramName string
		
		// Check for :param style
		if strings.HasPrefix(segment, ":") {
			paramName = strings.TrimPrefix(segment, ":")
		} else if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			// Check for {param} style
			paramName = strings.TrimPrefix(strings.TrimSuffix(segment, "}"), "{")
		}
		
		if paramName != "" {
			// Infer type based on parameter name patterns
			paramType := inferParamType(paramName)
			params[paramName] = paramType
		}
	}
	
	return params
}

// inferParamType infers the type of a path parameter based on its name
func inferParamType(paramName string) string {
	// Lowercase for case-insensitive matching
	lowerName := strings.ToLower(paramName)
	
	// ID parameters (ending with 'id' or exactly 'id')
	if lowerName == "id" || strings.HasSuffix(lowerName, "id") {
		// Special cases that should remain strings
		if lowerName == "clientid" || lowerName == "providerid" {
			return "string!"
		}
		return "xid.ID!"
	}
	
	// Version numbers
	if lowerName == "version" {
		return "int!"
	}
	
	// Slug parameters (SEO-friendly identifiers)
	if lowerName == "slug" || strings.HasSuffix(lowerName, "slug") {
		return "string!"
	}
	
	// Provider names (e.g., OAuth providers)
	if lowerName == "provider" {
		return "string!"
	}
	
	// Token parameters
	if lowerName == "token" || strings.HasSuffix(lowerName, "token") {
		return "string!"
	}
	
	// Standard parameters
	if lowerName == "standard" {
		return "string!"
	}
	
	// Default to string for all other parameters
	return "string!"
}
