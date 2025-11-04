package config

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/xraph/forge"
)

// Service provides organization-scoped configuration management
// It wraps Forge's ConfigManager to provide multi-tenant configuration support
type Service struct {
	globalConfig forge.ConfigManager
	orgConfigs   map[string]map[string]interface{}
	mu           sync.RWMutex
}

// NewService creates a new configuration service
func NewService(forgeConfig forge.ConfigManager) *Service {
	return &Service{
		globalConfig: forgeConfig,
		orgConfigs:   make(map[string]map[string]interface{}),
	}
}

// Bind binds configuration for a specific organization
// If orgID is empty, uses global configuration from Forge ConfigManager
// If orgID is provided, merges organization-specific overrides with global config
func (s *Service) Bind(orgID, key string, target interface{}) error {
	// First, bind global configuration using Forge ConfigManager
	if err := s.globalConfig.Bind(key, target); err != nil {
		return fmt.Errorf("failed to bind global config for key '%s': %w", key, err)
	}

	// If no organization ID, just use global config
	if orgID == "" {
		return nil
	}

	// Apply organization-specific overrides if they exist
	s.mu.RLock()
	orgConfig, exists := s.orgConfigs[orgID]
	s.mu.RUnlock()

	if !exists {
		return nil // No org-specific overrides, use global config
	}

	// Check if there's an override for this key in the org config
	if orgValue, hasOverride := s.getNestedValue(orgConfig, key); hasOverride {
		// Merge the override into the target
		return s.mergeValue(target, orgValue)
	}

	return nil
}

// Set sets a configuration value for a specific organization
func (s *Service) Set(orgID, key string, value interface{}) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required for setting org-specific config")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create org config
	if _, exists := s.orgConfigs[orgID]; !exists {
		s.orgConfigs[orgID] = make(map[string]interface{})
	}

	// Set the nested key value
	s.setNestedValue(s.orgConfigs[orgID], key, value)

	return nil
}

// Get gets a configuration value for a specific organization
func (s *Service) Get(orgID, key string) interface{} {
	// Check for org-specific override first
	if orgID != "" {
		s.mu.RLock()
		orgConfig, exists := s.orgConfigs[orgID]
		s.mu.RUnlock()

		if exists {
			if value, found := s.getNestedValue(orgConfig, key); found {
				return value
			}
		}
	}

	// Fall back to global configuration
	return s.globalConfig.Get(key)
}

// IsSet checks if a configuration key is set for a specific organization
func (s *Service) IsSet(orgID, key string) bool {
	// Check org-specific config first
	if orgID != "" {
		s.mu.RLock()
		orgConfig, exists := s.orgConfigs[orgID]
		s.mu.RUnlock()

		if exists {
			if _, found := s.getNestedValue(orgConfig, key); found {
				return true
			}
		}
	}

	// Check global configuration
	return s.globalConfig.IsSet(key)
}

// GetString gets a string configuration value
func (s *Service) GetString(orgID, key string) string {
	value := s.Get(orgID, key)
	if value == nil {
		return ""
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprintf("%v", value)
}

// GetInt gets an integer configuration value
func (s *Service) GetInt(orgID, key string) int {
	value := s.Get(orgID, key)
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetBool gets a boolean configuration value
func (s *Service) GetBool(orgID, key string) bool {
	value := s.Get(orgID, key)
	if value == nil {
		return false
	}

	if b, ok := value.(bool); ok {
		return b
	}

	return false
}

// LoadOrganizationConfig loads configuration for a specific organization
func (s *Service) LoadOrganizationConfig(orgID string, config map[string]interface{}) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the entire config for this organization
	s.orgConfigs[orgID] = config

	return nil
}

// RemoveOrganizationConfig removes all configuration for a specific organization
func (s *Service) RemoveOrganizationConfig(orgID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.orgConfigs, orgID)
}

// GetOrganizationConfig gets all configuration for a specific organization
func (s *Service) GetOrganizationConfig(orgID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orgConfig, exists := s.orgConfigs[orgID]
	if !exists {
		return make(map[string]interface{})
	}

	// Return a copy to prevent external modifications
	result := make(map[string]interface{})
	for k, v := range orgConfig {
		result[k] = v
	}

	return result
}

// MergeConfig merges organization-specific config with global config
// This is similar to Bind but unmarshals the entire config instead of a specific key
func (s *Service) MergeConfig(orgID string, target interface{}) error {
	// Get all global config (we'll need to unmarshal from Forge's ConfigManager)
	// Note: Forge's ConfigManager doesn't have an Unmarshal method, so we'll work with Bind
	// For now, this method assumes the target has already been bound with global config

	// If no org ID, nothing to merge
	if orgID == "" {
		return nil
	}

	// Apply organization-specific overrides
	s.mu.RLock()
	orgConfig, exists := s.orgConfigs[orgID]
	s.mu.RUnlock()

	if !exists || len(orgConfig) == 0 {
		return nil // No org-specific overrides
	}

	// Merge the org config into the target
	return s.mergeMap(orgConfig, target)
}

// getNestedValue retrieves a value from a nested map using dot notation
// e.g., "auth.oauth.google.clientId" -> map[auth][oauth][google][clientId]
func (s *Service) getNestedValue(config map[string]interface{}, key string) (interface{}, bool) {
	parts := strings.Split(key, ".")
	current := config

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, false
		}

		// If this is the last part, return the value
		if i == len(parts)-1 {
			return value, true
		}

		// Otherwise, navigate deeper
		nextMap, ok := value.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current = nextMap
	}

	return nil, false
}

// setNestedValue sets a value in a nested map using dot notation
// e.g., "auth.oauth.google.clientId" -> map[auth][oauth][google][clientId] = value
func (s *Service) setNestedValue(config map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := config

	for i, part := range parts {
		// If this is the last part, set the value
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		// Otherwise, navigate or create nested map
		if nextMap, ok := current[part].(map[string]interface{}); ok {
			current = nextMap
		} else {
			// Create new nested map
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		}
	}
}

// mergeValue merges a single configuration value into the target struct
func (s *Service) mergeValue(target interface{}, value interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetValue = targetValue.Elem()

	// Handle different value types
	switch v := value.(type) {
	case map[string]interface{}:
		// If the value is a map, merge it into the struct
		return s.mergeMap(v, target)
	default:
		// If the value is a simple type, set it directly
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type().ConvertibleTo(targetValue.Type()) {
			targetValue.Set(valueReflect.Convert(targetValue.Type()))
			return nil
		}
		return fmt.Errorf("cannot convert %T to %s", value, targetValue.Type())
	}
}

// mergeMap merges a map into a target struct using reflection
func (s *Service) mergeMap(source map[string]interface{}, target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetType := targetValue.Type()

	for i := 0; i < targetValue.NumField(); i++ {
		field := targetValue.Field(i)
		fieldType := targetType.Field(i)

		if !field.CanSet() {
			continue
		}

		// Get the field name (use json tag if available)
		fieldName := fieldType.Name
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Convert to lowercase for case-insensitive matching
		fieldName = strings.ToLower(fieldName)

		// Look for the value in source map
		if value, exists := source[fieldName]; exists {
			if err := s.setFieldValue(field, value); err != nil {
				return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

// setFieldValue sets a field value using reflection
func (s *Service) setFieldValue(field reflect.Value, value interface{}) error {
	valueReflect := reflect.ValueOf(value)

	if !valueReflect.IsValid() {
		return nil
	}

	// Handle type conversion
	if valueReflect.Type().ConvertibleTo(field.Type()) {
		field.Set(valueReflect.Convert(field.Type()))
		return nil
	}

	// Handle nested structs
	if field.Kind() == reflect.Struct && valueReflect.Kind() == reflect.Map {
		if mapValue, ok := value.(map[string]interface{}); ok {
			return s.mergeMap(mapValue, field.Addr().Interface())
		}
	}

	return fmt.Errorf("cannot convert %T to %s", value, field.Type())
}
