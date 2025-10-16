package main

import (
	"fmt"

	"github.com/spf13/viper"
)

// SimpleConfigManager wraps viper for CLI usage
// Implements a minimal config manager interface for AuthSome
type SimpleConfigManager struct {
	v *viper.Viper
}

// NewConfigManager creates a config manager from viper
func NewConfigManager() *SimpleConfigManager {
	return &SimpleConfigManager{
		v: viper.GetViper(),
	}
}

// Get retrieves a config value
func (c *SimpleConfigManager) Get(key string) interface{} {
	return c.v.Get(key)
}

// GetString retrieves a string config value
func (c *SimpleConfigManager) GetString(key string) string {
	return c.v.GetString(key)
}

// GetInt retrieves an int config value
func (c *SimpleConfigManager) GetInt(key string) int {
	return c.v.GetInt(key)
}

// GetBool retrieves a bool config value
func (c *SimpleConfigManager) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// Bind binds a config section to a struct (no-op for CLI)
func (c *SimpleConfigManager) Bind(key string, target interface{}) error {
	// For CLI, we don't need complex binding
	// Just return nil to satisfy the interface
	if !c.v.IsSet(key) {
		return fmt.Errorf("config key not found: %s", key)
	}

	// Use viper's unmarshal if the key exists
	if c.v.IsSet(key) {
		return c.v.UnmarshalKey(key, target)
	}

	return nil
}

// Set sets a config value
func (c *SimpleConfigManager) Set(key string, value interface{}) {
	c.v.Set(key, value)
}
