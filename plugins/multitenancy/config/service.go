package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// Service provides organization-scoped configuration management
type Service struct {
	globalViper *viper.Viper
	orgVipers   map[string]*viper.Viper
}

// NewService creates a new configuration service
func NewService(globalViper *viper.Viper) *Service {
	return &Service{
		globalViper: globalViper,
		orgVipers:   make(map[string]*viper.Viper),
	}
}

// Bind binds configuration for a specific organization
// If orgID is empty, uses global configuration
func (s *Service) Bind(orgID, key string, target interface{}) error {
	var v *viper.Viper

	if orgID == "" {
		// Use global configuration
		v = s.globalViper
	} else {
		// Try organization-specific configuration first
		orgViper := s.getOrganizationViper(orgID)
		if orgViper != nil && orgViper.IsSet(key) {
			v = orgViper
		} else {
			// Fall back to global configuration
			v = s.globalViper
		}
	}

	if !v.IsSet(key) {
		return fmt.Errorf("configuration key '%s' not found", key)
	}

	return v.UnmarshalKey(key, target)
}

// Set sets a configuration value for a specific organization
func (s *Service) Set(orgID, key string, value interface{}) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required for setting org-specific config")
	}

	orgViper := s.getOrCreateOrganizationViper(orgID)
	orgViper.Set(key, value)

	return nil
}

// Get gets a configuration value for a specific organization
func (s *Service) Get(orgID, key string) interface{} {
	if orgID == "" {
		return s.globalViper.Get(key)
	}

	orgViper := s.getOrganizationViper(orgID)
	if orgViper != nil && orgViper.IsSet(key) {
		return orgViper.Get(key)
	}

	// Fall back to global configuration
	return s.globalViper.Get(key)
}

// IsSet checks if a configuration key is set for a specific organization
func (s *Service) IsSet(orgID, key string) bool {
	if orgID == "" {
		return s.globalViper.IsSet(key)
	}

	orgViper := s.getOrganizationViper(orgID)
	if orgViper != nil && orgViper.IsSet(key) {
		return true
	}

	// Check global configuration
	return s.globalViper.IsSet(key)
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
	if orgID == "" {
		return s.globalViper.GetInt(key)
	}

	orgViper := s.getOrganizationViper(orgID)
	if orgViper != nil && orgViper.IsSet(key) {
		return orgViper.GetInt(key)
	}

	return s.globalViper.GetInt(key)
}

// GetBool gets a boolean configuration value
func (s *Service) GetBool(orgID, key string) bool {
	if orgID == "" {
		return s.globalViper.GetBool(key)
	}

	orgViper := s.getOrganizationViper(orgID)
	if orgViper != nil && orgViper.IsSet(key) {
		return orgViper.GetBool(key)
	}

	return s.globalViper.GetBool(key)
}

// LoadOrganizationConfig loads configuration for a specific organization
func (s *Service) LoadOrganizationConfig(orgID string, config map[string]interface{}) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}

	orgViper := s.getOrCreateOrganizationViper(orgID)

	// Set all configuration values
	for key, value := range config {
		orgViper.Set(key, value)
	}

	return nil
}

// RemoveOrganizationConfig removes all configuration for a specific organization
func (s *Service) RemoveOrganizationConfig(orgID string) {
	delete(s.orgVipers, orgID)
}

// GetOrganizationConfig gets all configuration for a specific organization
func (s *Service) GetOrganizationConfig(orgID string) map[string]interface{} {
	orgViper := s.getOrganizationViper(orgID)
	if orgViper == nil {
		return make(map[string]interface{})
	}

	return orgViper.AllSettings()
}

// MergeConfig merges organization-specific config with global config
func (s *Service) MergeConfig(orgID string, target interface{}) error {
	// First bind global config
	if err := s.globalViper.Unmarshal(target); err != nil {
		return fmt.Errorf("failed to unmarshal global config: %w", err)
	}

	// Then overlay organization-specific config if it exists
	if orgID != "" {
		orgViper := s.getOrganizationViper(orgID)
		if orgViper != nil {
			if err := s.mergeViperConfig(orgViper, target); err != nil {
				return fmt.Errorf("failed to merge organization config: %w", err)
			}
		}
	}

	return nil
}

// getOrganizationViper gets the viper instance for an organization
func (s *Service) getOrganizationViper(orgID string) *viper.Viper {
	return s.orgVipers[orgID]
}

// getOrCreateOrganizationViper gets or creates the viper instance for an organization
func (s *Service) getOrCreateOrganizationViper(orgID string) *viper.Viper {
	if v, exists := s.orgVipers[orgID]; exists {
		return v
	}

	v := viper.New()
	s.orgVipers[orgID] = v
	return v
}

// mergeViperConfig merges viper configuration into a target struct
func (s *Service) mergeViperConfig(v *viper.Viper, target interface{}) error {
	// Get all settings from viper
	settings := v.AllSettings()

	// Use reflection to merge settings into target
	return s.mergeMap(settings, target)
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
