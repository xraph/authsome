package config

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"
)

// MockConfigManager implements forge.ConfigManager for testing
type MockConfigManager struct {
	data map[string]interface{}
}

func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		data: make(map[string]interface{}),
	}
}

// Required core methods that we actually use
func (m *MockConfigManager) Bind(key string, target interface{}) error {
	return nil
}

func (m *MockConfigManager) Get(key string) interface{} {
	return m.data[key]
}

func (m *MockConfigManager) GetString(key string, defaultValue ...string) string {
	if v, ok := m.data[key].(string); ok {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (m *MockConfigManager) GetInt(key string, defaultValue ...int) int {
	if v, ok := m.data[key].(int); ok {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (m *MockConfigManager) GetBool(key string, defaultValue ...bool) bool {
	if v, ok := m.data[key].(bool); ok {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

func (m *MockConfigManager) IsSet(key string) bool {
	_, exists := m.data[key]
	return exists
}

func (m *MockConfigManager) Set(key string, value interface{}) {
	m.data[key] = value
}

func (m *MockConfigManager) AllKeys() []string {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

// Stub methods to satisfy interface (not used in our tests)
func (m *MockConfigManager) Name() string                                                                     { return "mock" }
func (m *MockConfigManager) SecretsManager() forge.SecretsManager                                             { return nil }
func (m *MockConfigManager) LoadFrom(sources ...forge.ConfigSource) error                                     { return nil }
func (m *MockConfigManager) Watch(ctx context.Context) error                                                  { return nil }
func (m *MockConfigManager) Reload() error                                                                    { return nil }
func (m *MockConfigManager) ReloadContext(ctx context.Context) error                                          { return nil }
func (m *MockConfigManager) Validate() error                                                                  { return nil }
func (m *MockConfigManager) Stop() error                                                                      { return nil }
func (m *MockConfigManager) GetInt8(key string, defaultValue ...int8) int8                                    { return 0 }
func (m *MockConfigManager) GetInt16(key string, defaultValue ...int16) int16                                 { return 0 }
func (m *MockConfigManager) GetInt32(key string, defaultValue ...int32) int32                                 { return 0 }
func (m *MockConfigManager) GetInt64(key string, defaultValue ...int64) int64                                 { return 0 }
func (m *MockConfigManager) GetUint(key string, defaultValue ...uint) uint                                    { return 0 }
func (m *MockConfigManager) GetUint8(key string, defaultValue ...uint8) uint8                                 { return 0 }
func (m *MockConfigManager) GetUint16(key string, defaultValue ...uint16) uint16                              { return 0 }
func (m *MockConfigManager) GetUint32(key string, defaultValue ...uint32) uint32                              { return 0 }
func (m *MockConfigManager) GetUint64(key string, defaultValue ...uint64) uint64                              { return 0 }
func (m *MockConfigManager) GetFloat32(key string, defaultValue ...float32) float32                           { return 0 }
func (m *MockConfigManager) GetFloat64(key string, defaultValue ...float64) float64                           { return 0 }
func (m *MockConfigManager) GetDuration(key string, defaultValue ...time.Duration) time.Duration              { return 0 }
func (m *MockConfigManager) GetTime(key string, defaultValue ...time.Time) time.Time                          { return time.Time{} }
func (m *MockConfigManager) GetSizeInBytes(key string, defaultValue ...uint64) uint64                         { return 0 }
func (m *MockConfigManager) GetStringSlice(key string, defaultValue ...[]string) []string                     { return nil }
func (m *MockConfigManager) GetIntSlice(key string, defaultValue ...[]int) []int                              { return nil }
func (m *MockConfigManager) GetInt64Slice(key string, defaultValue ...[]int64) []int64                        { return nil }
func (m *MockConfigManager) GetFloat64Slice(key string, defaultValue ...[]float64) []float64                  { return nil }
func (m *MockConfigManager) GetBoolSlice(key string, defaultValue ...[]bool) []bool                           { return nil }
func (m *MockConfigManager) GetStringMap(key string, defaultValue ...map[string]string) map[string]string     { return nil }
func (m *MockConfigManager) GetStringMapString(key string, defaultValue ...map[string]string) map[string]string { return nil }
func (m *MockConfigManager) GetStringMapStringSlice(key string, defaultValue ...map[string][]string) map[string][]string { return nil }
func (m *MockConfigManager) GetWithOptions(key string, opts ...forge.GetOption) (interface{}, error)          { return nil, nil }
func (m *MockConfigManager) GetStringWithOptions(key string, opts ...forge.GetOption) (string, error)         { return "", nil }
func (m *MockConfigManager) GetIntWithOptions(key string, opts ...forge.GetOption) (int, error)               { return 0, nil }
func (m *MockConfigManager) GetBoolWithOptions(key string, opts ...forge.GetOption) (bool, error)             { return false, nil }
func (m *MockConfigManager) GetDurationWithOptions(key string, opts ...forge.GetOption) (time.Duration, error) { return 0, nil }
func (m *MockConfigManager) BindWithDefault(key string, target interface{}, defaultValue interface{}) error  { return nil }
func (m *MockConfigManager) BindWithOptions(key string, target interface{}, options forge.BindOptions) error { return nil }
func (m *MockConfigManager) WatchWithCallback(key string, callback func(string, interface{}))                {}
func (m *MockConfigManager) WatchChanges(callback func(forge.ConfigChange))                                  {}
func (m *MockConfigManager) GetSourceMetadata() map[string]*forge.SourceMetadata                             { return nil }
func (m *MockConfigManager) GetKeys() []string                                                                { return m.AllKeys() }
func (m *MockConfigManager) GetSection(key string) map[string]interface{}                                     { return nil }
func (m *MockConfigManager) HasKey(key string) bool                                                           { return m.IsSet(key) }
func (m *MockConfigManager) Size() int                                                                        { return len(m.data) }
func (m *MockConfigManager) Sub(key string) forge.ConfigManager                                               { return nil }
func (m *MockConfigManager) MergeWith(other forge.ConfigManager) error                                        { return nil }
func (m *MockConfigManager) Clone() forge.ConfigManager                                                       { return m }
func (m *MockConfigManager) GetAllSettings() map[string]interface{}                                           { return m.data }
func (m *MockConfigManager) Reset()                                                                           {}
func (m *MockConfigManager) ExpandEnvVars() error                                                             { return nil }
func (m *MockConfigManager) SafeGet(key string, expectedType reflect.Type) (interface{}, error)              { return nil, nil }
func (m *MockConfigManager) GetBytesSize(key string, defaultValue ...uint64) uint64                           { return 0 }
func (m *MockConfigManager) InConfig(key string) bool                                                         { return m.IsSet(key) }
func (m *MockConfigManager) UnmarshalKey(key string, rawVal interface{}) error                                { return nil }
func (m *MockConfigManager) Unmarshal(rawVal interface{}) error                                               { return nil }
func (m *MockConfigManager) AllSettings() map[string]interface{}                                              { return m.data }
func (m *MockConfigManager) ReadInConfig() error                                                              { return nil }
func (m *MockConfigManager) SetConfigType(configType string)                                                  {}
func (m *MockConfigManager) SetConfigFile(filePath string) error                                              { return nil }
func (m *MockConfigManager) ConfigFileUsed() string                                                           { return "" }
func (m *MockConfigManager) WatchConfig() error                                                               { return nil }
func (m *MockConfigManager) OnConfigChange(callback func(forge.ConfigChange))                                {}

// Ensure MockConfigManager implements forge.ConfigManager
var _ forge.ConfigManager = (*MockConfigManager)(nil)

func TestNewService(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	assert.NotNil(t, service)
	assert.NotNil(t, service.globalConfig)
	assert.NotNil(t, service.orgConfigs)
}

func TestService_SetAndGet(t *testing.T) {
	mockConfig := NewMockConfigManager()
	mockConfig.Set("auth.test.key", "global-value")

	service := NewService(mockConfig)

	// Set org-specific override
	err := service.Set("org-123", "auth.test.key", "org-value")
	require.NoError(t, err)

	// Get org-specific value
	value := service.Get("org-123", "auth.test.key")
	assert.Equal(t, "org-value", value)

	// Get global value for different org
	value = service.Get("org-456", "auth.test.key")
	assert.Equal(t, "global-value", value)

	// Get global value with no org
	value = service.Get("", "auth.test.key")
	assert.Equal(t, "global-value", value)
}

func TestService_GetString(t *testing.T) {
	mockConfig := NewMockConfigManager()
	mockConfig.Set("auth.client.id", "default-client")

	service := NewService(mockConfig)

	// Set org-specific override
	err := service.Set("org-123", "auth.client.id", "custom-client")
	require.NoError(t, err)

	// Test org-specific value
	clientId := service.GetString("org-123", "auth.client.id")
	assert.Equal(t, "custom-client", clientId)

	// Test fallback to global
	clientId = service.GetString("org-456", "auth.client.id")
	assert.Equal(t, "default-client", clientId)
}

func TestService_GetInt(t *testing.T) {
	mockConfig := NewMockConfigManager()
	mockConfig.Set("auth.session.timeout", 60)

	service := NewService(mockConfig)

	// Set org-specific override
	err := service.Set("org-123", "auth.session.timeout", 120)
	require.NoError(t, err)

	// Test org-specific value
	timeout := service.GetInt("org-123", "auth.session.timeout")
	assert.Equal(t, 120, timeout)

	// Test fallback to global
	timeout = service.GetInt("org-456", "auth.session.timeout")
	assert.Equal(t, 60, timeout)
}

func TestService_GetBool(t *testing.T) {
	mockConfig := NewMockConfigManager()
	mockConfig.Set("auth.features.enabled", true)

	service := NewService(mockConfig)

	// Set org-specific override
	err := service.Set("org-123", "auth.features.enabled", false)
	require.NoError(t, err)

	// Test org-specific value
	enabled := service.GetBool("org-123", "auth.features.enabled")
	assert.False(t, enabled)

	// Test fallback to global
	enabled = service.GetBool("org-456", "auth.features.enabled")
	assert.True(t, enabled)
}

func TestService_IsSet(t *testing.T) {
	mockConfig := NewMockConfigManager()
	mockConfig.Set("auth.key1", "value1")

	service := NewService(mockConfig)

	// Test global key
	assert.True(t, service.IsSet("", "auth.key1"))
	assert.False(t, service.IsSet("", "auth.key2"))

	// Set org-specific key
	err := service.Set("org-123", "auth.key2", "value2")
	require.NoError(t, err)

	// Test org-specific key
	assert.True(t, service.IsSet("org-123", "auth.key2"))
	assert.False(t, service.IsSet("org-456", "auth.key2"))
}

func TestService_LoadOrganizationConfig(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	orgConfig := map[string]interface{}{
		"auth": map[string]interface{}{
			"oauth": map[string]interface{}{
				"google": map[string]interface{}{
					"clientId": "org-client-id",
					"enabled":  true,
				},
			},
		},
	}

	err := service.LoadOrganizationConfig("org-123", orgConfig)
	require.NoError(t, err)

	// Test nested key access
	clientId := service.GetString("org-123", "auth.oauth.google.clientId")
	assert.Equal(t, "org-client-id", clientId)

	enabled := service.GetBool("org-123", "auth.oauth.google.enabled")
	assert.True(t, enabled)
}

func TestService_RemoveOrganizationConfig(t *testing.T) {
	mockConfig := NewMockConfigManager()
	mockConfig.Set("auth.test.key", "global-value")

	service := NewService(mockConfig)

	// Set org-specific config
	err := service.Set("org-123", "auth.test.key", "org-value")
	require.NoError(t, err)

	// Verify org-specific value
	value := service.GetString("org-123", "auth.test.key")
	assert.Equal(t, "org-value", value)

	// Remove org config
	service.RemoveOrganizationConfig("org-123")

	// Should now fall back to global
	value = service.GetString("org-123", "auth.test.key")
	assert.Equal(t, "global-value", value)
}

func TestService_GetOrganizationConfig(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	orgConfig := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err := service.LoadOrganizationConfig("org-123", orgConfig)
	require.NoError(t, err)

	// Get org config
	retrieved := service.GetOrganizationConfig("org-123")
	assert.Equal(t, "value1", retrieved["key1"])
	assert.Equal(t, 123, retrieved["key2"])
	assert.Equal(t, true, retrieved["key3"])

	// Test non-existent org
	emptyConfig := service.GetOrganizationConfig("org-456")
	assert.Empty(t, emptyConfig)
}

func TestService_NestedKeyOperations(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	// Test setting nested keys
	err := service.Set("org-123", "level1.level2.level3.key", "deep-value")
	require.NoError(t, err)

	// Verify the value was set
	value := service.GetString("org-123", "level1.level2.level3.key")
	assert.Equal(t, "deep-value", value)

	// Verify nested structure was created
	orgConfig := service.GetOrganizationConfig("org-123")
	assert.NotNil(t, orgConfig["level1"])
}

func TestService_ConcurrentAccess(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	// Test concurrent reads and writes
	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 10; i++ {
		go func(id int) {
			orgID := "org-" + string(rune('A'+id))
			for j := 0; j < 100; j++ {
				_ = service.Set(orgID, "auth.test.key", j)
			}
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 10; i++ {
		go func(id int) {
			orgID := "org-" + string(rune('A'+id))
			for j := 0; j < 100; j++ {
				_ = service.GetString(orgID, "auth.test.key")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// If we get here without race conditions, the test passes
	assert.True(t, true)
}

func TestService_SetRequiresOrgID(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	// Setting without org ID should fail
	err := service.Set("", "auth.test.key", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "organization ID is required")
}

func TestService_LoadOrganizationConfigRequiresOrgID(t *testing.T) {
	mockConfig := NewMockConfigManager()
	service := NewService(mockConfig)

	// Loading config without org ID should fail
	err := service.LoadOrganizationConfig("", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "organization ID is required")
}

