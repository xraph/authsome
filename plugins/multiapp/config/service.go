package config

import (
	"fmt"
	"maps"
	"reflect"
	"strings"
	"sync"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Service provides app-scoped configuration management
// It wraps Forge's ConfigManager to provide multi-tenant configuration support.
type Service struct {
	globalConfig forge.ConfigManager
	appConfigs   map[string]map[string]any
	mu           sync.RWMutex
}

// NewService creates a new configuration service.
func NewService(forgeConfig forge.ConfigManager) *Service {
	return &Service{
		globalConfig: forgeConfig,
		appConfigs:   make(map[string]map[string]any),
	}
}

// Bind binds configuration for a specific app
// If appID is empty, uses global configuration from Forge ConfigManager
// If appID is provided, merges app-specific overrides with global config.
func (s *Service) Bind(appID, key string, target any) error {
	// First, bind global configuration using Forge ConfigManager
	if err := s.globalConfig.Bind(key, target); err != nil {
		return fmt.Errorf("failed to bind global config for key '%s': %w", key, err)
	}

	// If no app ID, just use global config
	if appID == "" {
		return nil
	}

	// Apply app-specific overrides if they exist
	s.mu.RLock()
	appConfig, exists := s.appConfigs[appID]
	s.mu.RUnlock()

	if !exists {
		return nil // No app-specific overrides, use global config
	}

	// Check if there's an override for this key in the app config
	if appValue, hasOverride := s.getNestedValue(appConfig, key); hasOverride {
		// Merge the override into the target
		return s.mergeValue(target, appValue)
	}

	return nil
}

// Set sets a configuration value for a specific app.
func (s *Service) Set(appID, key string, value any) error {
	if appID == "" {
		return errs.RequiredField("appID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create app config
	if _, exists := s.appConfigs[appID]; !exists {
		s.appConfigs[appID] = make(map[string]any)
	}

	// Set the nested key value
	s.setNestedValue(s.appConfigs[appID], key, value)

	return nil
}

// Get gets a configuration value for a specific app.
func (s *Service) Get(appID, key string) any {
	// Check for app-specific override first
	if appID != "" {
		s.mu.RLock()

		appConfig, exists := s.appConfigs[appID]
		if exists {
			if value, found := s.getNestedValue(appConfig, key); found {
				s.mu.RUnlock()

				return value
			}
		}

		s.mu.RUnlock()
	}

	// Fall back to global configuration
	return s.globalConfig.Get(key)
}

// IsSet checks if a configuration key is set for a specific app.
func (s *Service) IsSet(appID, key string) bool {
	// Check app-specific config first
	if appID != "" {
		s.mu.RLock()

		appConfig, exists := s.appConfigs[appID]
		if exists {
			if _, found := s.getNestedValue(appConfig, key); found {
				s.mu.RUnlock()

				return true
			}
		}

		s.mu.RUnlock()
	}

	// Check global configuration
	return s.globalConfig.IsSet(key)
}

// GetString gets a string configuration value.
func (s *Service) GetString(appID, key string) string {
	value := s.Get(appID, key)
	if value == nil {
		return ""
	}

	if str, ok := value.(string); ok {
		return str
	}

	return fmt.Sprintf("%v", value)
}

// GetInt gets an integer configuration value.
func (s *Service) GetInt(appID, key string) int {
	value := s.Get(appID, key)
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

// GetBool gets a boolean configuration value.
func (s *Service) GetBool(appID, key string) bool {
	value := s.Get(appID, key)
	if value == nil {
		return false
	}

	if b, ok := value.(bool); ok {
		return b
	}

	return false
}

// LoadAppConfig loads configuration for a specific app.
func (s *Service) LoadAppConfig(appID string, config map[string]any) error {
	if appID == "" {
		return errs.RequiredField("appID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store the entire config for this app
	s.appConfigs[appID] = config

	return nil
}

// RemoveAppConfig removes all configuration for a specific app.
func (s *Service) RemoveAppConfig(appID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.appConfigs, appID)
}

// GetAppConfig gets all configuration for a specific app.
func (s *Service) GetAppConfig(appID string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	appConfig, exists := s.appConfigs[appID]
	if !exists {
		return make(map[string]any)
	}

	// Return a copy to prevent external modifications
	result := make(map[string]any)
	maps.Copy(result, appConfig)

	return result
}

// MergeConfig merges app-specific config with global config
// This is similar to Bind but unmarshals the entire config instead of a specific key.
func (s *Service) MergeConfig(appID string, target any) error {
	// Get all global config (we'll need to unmarshal from Forge's ConfigManager)
	// Note: Forge's ConfigManager doesn't have an Unmarshal method, so we'll work with Bind
	// For now, this method assumes the target has already been bound with global config

	// If no app ID, nothing to merge
	if appID == "" {
		return nil
	}

	// Apply app-specific overrides
	s.mu.RLock()
	appConfig, exists := s.appConfigs[appID]
	s.mu.RUnlock()

	if !exists || len(appConfig) == 0 {
		return nil // No app-specific overrides
	}

	// Merge the app config into the target
	return s.mergeMap(appConfig, target)
}

// getNestedValue retrieves a value from a nested map using dot notation
// e.g., "auth.oauth.google.clientId" -> map[auth][oauth][google][clientId].
func (s *Service) getNestedValue(config map[string]any, key string) (any, bool) {
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
		nextMap, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}

		current = nextMap
	}

	return nil, false
}

// setNestedValue sets a value in a nested map using dot notation
// e.g., "auth.oauth.google.clientId" -> map[auth][oauth][google][clientId] = value.
func (s *Service) setNestedValue(config map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	current := config

	for i, part := range parts {
		// If this is the last part, set the value
		if i == len(parts)-1 {
			current[part] = value

			return
		}

		// Otherwise, navigate or create nested map
		if nextMap, ok := current[part].(map[string]any); ok {
			current = nextMap
		} else {
			// Create new nested map
			newMap := make(map[string]any)
			current[part] = newMap
			current = newMap
		}
	}
}

// mergeValue merges a single configuration value into the target struct.
func (s *Service) mergeValue(target any, value any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return errs.BadRequest("target must be a pointer")
	}

	targetValue = targetValue.Elem()

	// Handle different value types
	switch v := value.(type) {
	case map[string]any:
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

// mergeMap merges a map into a target struct using reflection.
func (s *Service) mergeMap(source map[string]any, target any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return errs.BadRequest("target must be a pointer")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Struct {
		return errs.BadRequest("target must be a pointer to struct")
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

// setFieldValue sets a field value using reflection.
func (s *Service) setFieldValue(field reflect.Value, value any) error {
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
		if mapValue, ok := value.(map[string]any); ok {
			return s.mergeMap(mapValue, field.Addr().Interface())
		}
	}

	return fmt.Errorf("cannot convert %T to %s", value, field.Type())
}
