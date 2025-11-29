package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/plugins/secrets/core"
)

// SecretsConfigSource implements forge.ConfigSource to provide secrets as configuration
type SecretsConfigSource struct {
	service  *Service
	appID    string
	envID    string
	prefix   string
	priority int
	logger   forge.Logger

	// Cache
	cache     map[string]interface{}
	cacheMu   sync.RWMutex
	loaded    bool
	lastLoad  time.Time

	// Watch
	watching  bool
	watchCtx  context.Context
	watchStop context.CancelFunc
	callback  func(map[string]interface{})
}

// NewSecretsConfigSource creates a new config source for an app/environment
func NewSecretsConfigSource(
	service *Service,
	appID, envID string,
	prefix string,
	priority int,
	logger forge.Logger,
) *SecretsConfigSource {
	return &SecretsConfigSource{
		service:  service,
		appID:    appID,
		envID:    envID,
		prefix:   prefix,
		priority: priority,
		logger:   logger,
		cache:    make(map[string]interface{}),
	}
}

// =============================================================================
// forge.ConfigSource Interface Implementation
// =============================================================================

// Name returns the unique name of the configuration source
func (s *SecretsConfigSource) Name() string {
	return fmt.Sprintf("secrets:%s:%s", s.appID, s.envID)
}

// GetName returns the name (alias for Name)
func (s *SecretsConfigSource) GetName() string {
	return s.Name()
}

// GetType returns the source type
func (s *SecretsConfigSource) GetType() string {
	return "secrets"
}

// Priority returns the priority of this source (higher = more important)
func (s *SecretsConfigSource) Priority() int {
	return s.priority
}

// Load loads configuration data from secrets
func (s *SecretsConfigSource) Load(ctx context.Context) (map[string]interface{}, error) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Create context with app/env IDs
	appID, err := xid.FromString(s.appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}
	envID, err := xid.FromString(s.envID)
	if err != nil {
		return nil, fmt.Errorf("invalid environment ID: %w", err)
	}

	ctx = contexts.SetAppID(ctx, appID)
	ctx = contexts.SetEnvironmentID(ctx, envID)

	// List all secrets with optional prefix
	query := &core.ListSecretsQuery{
		Prefix:   s.prefix,
		PageSize: 1000, // Load all secrets
	}

	secrets, _, err := s.service.List(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	// Clear and rebuild cache
	s.cache = make(map[string]interface{})
	result := make(map[string]interface{})

	for _, secret := range secrets {
		// Get decrypted value
		secretID, err := xid.FromString(secret.ID)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("invalid secret ID", forge.F("id", secret.ID))
			}
			continue
		}

		value, err := s.service.GetValue(ctx, secretID)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("failed to get secret value",
					forge.F("path", secret.Path),
					forge.F("error", err.Error()))
			}
			continue
		}

		// Convert path to config key: "database/postgres/password" -> "database.postgres.password"
		configKey := core.PathToConfigKey(secret.Path)

		// Store in cache and result
		s.cache[configKey] = value
		setNestedValue(result, configKey, value)
	}

	s.loaded = true
	s.lastLoad = time.Now()

	if s.logger != nil {
		s.logger.Debug("loaded secrets into config source",
			forge.F("source", s.Name()),
			forge.F("count", len(secrets)))
	}

	return result, nil
}

// Watch starts watching for configuration changes
func (s *SecretsConfigSource) Watch(ctx context.Context, callback func(map[string]interface{})) error {
	if s.watching {
		return nil // Already watching
	}

	s.watchCtx, s.watchStop = context.WithCancel(ctx)
	s.callback = callback
	s.watching = true

	// The actual watching is done via hooks in the plugin
	// This method just sets up the callback

	if s.logger != nil {
		s.logger.Debug("started watching secrets config source", forge.F("source", s.Name()))
	}

	return nil
}

// StopWatch stops watching for configuration changes
func (s *SecretsConfigSource) StopWatch() error {
	if !s.watching {
		return nil
	}

	if s.watchStop != nil {
		s.watchStop()
	}

	s.watching = false
	s.callback = nil

	if s.logger != nil {
		s.logger.Debug("stopped watching secrets config source", forge.F("source", s.Name()))
	}

	return nil
}

// Reload forces a reload of the configuration source
func (s *SecretsConfigSource) Reload(ctx context.Context) error {
	data, err := s.Load(ctx)
	if err != nil {
		return err
	}

	// Notify callback if watching
	if s.watching && s.callback != nil {
		s.callback(data)
	}

	return nil
}

// IsWatchable returns true if the source supports watching for changes
func (s *SecretsConfigSource) IsWatchable() bool {
	return true
}

// SupportsSecrets returns true if the source supports secret management
func (s *SecretsConfigSource) SupportsSecrets() bool {
	return true
}

// GetSecret retrieves a secret value from the source
func (s *SecretsConfigSource) GetSecret(ctx context.Context, key string) (string, error) {
	// Convert config key to path
	path := core.ConfigKeyToPath(key)

	// Create context with app/env IDs
	appID, err := xid.FromString(s.appID)
	if err != nil {
		return "", fmt.Errorf("invalid app ID: %w", err)
	}
	envID, err := xid.FromString(s.envID)
	if err != nil {
		return "", fmt.Errorf("invalid environment ID: %w", err)
	}

	ctx = contexts.SetAppID(ctx, appID)
	ctx = contexts.SetEnvironmentID(ctx, envID)

	value, err := s.service.GetValueByPath(ctx, path)
	if err != nil {
		return "", err
	}

	// Convert to string
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		// Marshal complex types to JSON
		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal secret value: %w", err)
		}
		return string(data), nil
	}
}

// IsAvailable checks if the source is available
func (s *SecretsConfigSource) IsAvailable(ctx context.Context) bool {
	// Check if we can access the service
	return s.service != nil
}

// =============================================================================
// Additional Methods
// =============================================================================

// Get retrieves a configuration value by key from cache
func (s *SecretsConfigSource) Get(key string) (interface{}, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	value, ok := s.cache[key]
	return value, ok
}

// GetString retrieves a string configuration value
func (s *SecretsConfigSource) GetString(key string) (string, bool) {
	value, ok := s.Get(key)
	if !ok {
		return "", false
	}

	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		// Try to marshal to JSON string
		data, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
}

// Keys returns all available keys in the cache
func (s *SecretsConfigSource) Keys() []string {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	keys := make([]string, 0, len(s.cache))
	for k := range s.cache {
		keys = append(keys, k)
	}
	return keys
}

// IsLoaded returns whether the source has been loaded
func (s *SecretsConfigSource) IsLoaded() bool {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.loaded
}

// LastLoadTime returns when the source was last loaded
func (s *SecretsConfigSource) LastLoadTime() time.Time {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.lastLoad
}

// CacheSize returns the number of items in the cache
func (s *SecretsConfigSource) CacheSize() int {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return len(s.cache)
}

// ClearCache clears the configuration cache
func (s *SecretsConfigSource) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = make(map[string]interface{})
	s.loaded = false
}

// =============================================================================
// Helper Functions
// =============================================================================

// setNestedValue sets a value in a nested map using dot notation
// Example: setNestedValue(m, "a.b.c", value) sets m["a"]["b"]["c"] = value
func setNestedValue(m map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - set the value
			current[part] = value
		} else {
			// Intermediate part - create nested map if needed
			if _, ok := current[part]; !ok {
				current[part] = make(map[string]interface{})
			}
			if nested, ok := current[part].(map[string]interface{}); ok {
				current = nested
			} else {
				// Can't set nested value, path conflicts with existing value
				return
			}
		}
	}
}

// getNestedValue gets a value from a nested map using dot notation
func getNestedValue(m map[string]interface{}, key string) (interface{}, bool) {
	parts := strings.Split(key, ".")
	current := m

	for i, part := range parts {
		value, ok := current[part]
		if !ok {
			return nil, false
		}

		if i == len(parts)-1 {
			return value, true
		}

		if nested, ok := value.(map[string]interface{}); ok {
			current = nested
		} else {
			return nil, false
		}
	}

	return nil, false
}

// flattenMap flattens a nested map to dot-notation keys
func flattenMap(m map[string]interface{}, prefix string, result map[string]interface{}) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if nested, ok := v.(map[string]interface{}); ok {
			flattenMap(nested, key, result)
		} else {
			result[key] = v
		}
	}
}

