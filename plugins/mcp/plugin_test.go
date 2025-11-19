package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// mockAuth implements the required interfaces for testing
type mockAuth struct {
	db              *bun.DB
	serviceRegistry *registry.ServiceRegistry
	forgeApp        forge.App
}

func (m *mockAuth) GetDB() *bun.DB {
	return m.db
}

func (m *mockAuth) GetServiceRegistry() *registry.ServiceRegistry {
	return m.serviceRegistry
}

func (m *mockAuth) GetForgeApp() forge.App {
	return m.forgeApp
}

func (m *mockAuth) Initialize(ctx context.Context) error             { return nil }
func (m *mockAuth) Mount(router forge.Router, basePath string) error { return nil }
func (m *mockAuth) RegisterPlugin(plugin core.Plugin) error          { return nil }
func (m *mockAuth) GetConfig() core.Config                           { return core.Config{} }
func (m *mockAuth) GetHookRegistry() *hooks.HookRegistry             { return nil }
func (m *mockAuth) GetBasePath() string                              { return "" }
func (m *mockAuth) GetPluginRegistry() core.PluginRegistry           { return nil }
func (m *mockAuth) IsPluginEnabled(pluginID string) bool             { return false }
func (m *mockAuth) Repository() repository.Repository                { return nil }

func TestPluginID(t *testing.T) {
	plugin := NewPlugin(WithDefaultConfig(DefaultConfig()))
	assert.Equal(t, "mcp", plugin.ID())
}

func TestPluginInit(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.Transport = TransportStdio

	plugin := NewPlugin(WithDefaultConfig(config))

	// Mock auth with nil DB (will fail)
	auth := &mockAuth{
		db:              nil,
		serviceRegistry: registry.NewServiceRegistry(),
	}

	err := plugin.Init(auth)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not available")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.False(t, config.Enabled) // Opt-in
	assert.Equal(t, ModeReadOnly, config.Mode)
	assert.Equal(t, TransportStdio, config.Transport)
	assert.Equal(t, 9090, config.Port)
	assert.True(t, config.Authorization.RequireAPIKey)
	assert.Greater(t, len(config.Authorization.AllowedOperations), 0)
	assert.Equal(t, 60, config.RateLimit.RequestsPerMinute)
}

func TestConfigModes(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
	}{
		{"readonly", ModeReadOnly},
		{"admin", ModeAdmin},
		{"development", ModeDevelopment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Mode = tt.mode
			assert.Equal(t, tt.mode, config.Mode)
		})
	}
}

func TestConfigTransports(t *testing.T) {
	tests := []struct {
		name      string
		transport Transport
	}{
		{"stdio", TransportStdio},
		{"http", TransportHTTP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Transport = tt.transport
			assert.Equal(t, tt.transport, config.Transport)
		})
	}
}

func TestPluginHooksAndDecorators(t *testing.T) {
	plugin := NewPlugin(WithDefaultConfig(DefaultConfig()))

	// MCP plugin doesn't use hooks or decorators
	err := plugin.RegisterHooks(nil)
	assert.NoError(t, err)

	err = plugin.RegisterServiceDecorators(nil)
	assert.NoError(t, err)

	err = plugin.Migrate()
	assert.NoError(t, err)
}

func TestSecurityLayer(t *testing.T) {
	config := DefaultConfig()
	security := NewSecurityLayer(config, nil)

	require.NotNil(t, security)

	// Test authorization
	err := security.CheckOperationAllowed("query_user")
	assert.NoError(t, err) // query_user is in allowed list

	err = security.CheckOperationAllowed("unknown_operation")
	assert.Error(t, err)
}

func TestSecurityLayerAdminOperations(t *testing.T) {
	// Test readonly mode - admin operations not allowed
	config := Config{
		Mode: ModeReadOnly,
		Authorization: AuthorizationConfig{
			AdminOperations: []string{"revoke_session"},
		},
	}
	security := NewSecurityLayer(config, nil)

	err := security.CheckOperationAllowed("revoke_session")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires admin mode")

	// Test admin mode - admin operations allowed
	config.Mode = ModeAdmin
	security = NewSecurityLayer(config, nil)

	err = security.CheckOperationAllowed("revoke_session")
	assert.NoError(t, err)
}

func TestSanitizeUser(t *testing.T) {
	config := DefaultConfig()
	security := NewSecurityLayer(config, nil)

	userData := map[string]interface{}{
		"id":            "user-123",
		"email":         "test@example.com",
		"password_hash": "secret-hash",
		"twofa_secret":  "secret-totp",
		"name":          "Test User",
	}

	sanitized := security.sanitizeUser(userData)
	sanitizedMap, ok := sanitized.(map[string]interface{})
	require.True(t, ok)

	// Sensitive fields removed
	assert.NotContains(t, sanitizedMap, "password_hash")
	assert.NotContains(t, sanitizedMap, "twofa_secret")

	// Safe fields preserved
	assert.Equal(t, "user-123", sanitizedMap["id"])
	assert.Equal(t, "Test User", sanitizedMap["name"])
}

func TestSanitizeConfig(t *testing.T) {
	config := DefaultConfig()
	security := NewSecurityLayer(config, nil)

	configData := map[string]interface{}{
		"database_url": "postgres://localhost/db",
		"secret_key":   "very-secret",
		"api_token":    "token-123",
		"public_key":   "public-data",
		"app_name":     "AuthSome",
	}

	sanitized := security.sanitizeConfig(configData)
	sanitizedMap, ok := sanitized.(map[string]interface{})
	require.True(t, ok)

	// Secrets redacted
	assert.Equal(t, "[REDACTED]", sanitizedMap["secret_key"])
	assert.Equal(t, "[REDACTED]", sanitizedMap["api_token"])

	// Public key preserved
	assert.Equal(t, "public-data", sanitizedMap["public_key"])
	assert.Equal(t, "AuthSome", sanitizedMap["app_name"])
}

func TestResourceRegistry(t *testing.T) {
	registry := NewResourceRegistry()
	require.NotNil(t, registry)

	// Register a resource
	resource := &ConfigResource{}
	registry.Register("authsome://config", resource)

	// List resources
	resources := registry.List()
	assert.Equal(t, 1, len(resources))
	assert.Equal(t, "authsome://config", resources[0].URI)
}

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()
	require.NotNil(t, registry)

	// Register a tool
	tool := &QueryUserTool{}
	registry.Register("query_user", tool)

	// List tools in readonly mode
	tools := registry.List(ModeReadOnly)
	assert.Greater(t, len(tools), 0)

	found := false
	for _, t := range tools {
		if t.Name == "query_user" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "te************om"}, // 16 chars: first 2 + last 2 + (16-4) asterisks
		{"short", "sh*rt"},                       // 5 chars: first 2 + last 2 + (5-4) asterisks
		{"ab", "***"},                            // ≤4 chars: all asterisks
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.100", "192.168.*.*"},
		{"10.0.0.1", "10.0.*.*"},
		{"invalid", "***"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskIP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigResourceDescribe(t *testing.T) {
	resource := &ConfigResource{}
	desc := resource.Describe()

	assert.Equal(t, "authsome://config", desc.URI)
	assert.NotEmpty(t, desc.Name)
	assert.NotEmpty(t, desc.Description)
	assert.Equal(t, "application/json", desc.MimeType)
}

func TestSchemaResourceDescribe(t *testing.T) {
	resource := &SchemaResource{}
	desc := resource.Describe()

	assert.Equal(t, "authsome://schema", desc.URI)
	assert.NotEmpty(t, desc.Name)
	assert.Equal(t, "application/json", desc.MimeType)
}

func TestQueryUserToolDescribe(t *testing.T) {
	tool := &QueryUserTool{}
	desc := tool.Describe()

	assert.Equal(t, "query_user", desc.Name)
	assert.NotEmpty(t, desc.Description)
	assert.NotNil(t, desc.InputSchema)
	assert.False(t, tool.RequiresAuth())
	assert.False(t, tool.RequiresAdmin())
}

func TestCheckPermissionToolDescribe(t *testing.T) {
	tool := &CheckPermissionTool{}
	desc := tool.Describe()

	assert.Equal(t, "check_permission", desc.Name)
	assert.NotEmpty(t, desc.Description)
	assert.NotNil(t, desc.InputSchema)
	assert.False(t, tool.RequiresAuth())
	assert.False(t, tool.RequiresAdmin())
}

func TestMCPRequest(t *testing.T) {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/list",
		Params:  nil,
	}

	assert.Equal(t, "2.0", req.JSONRPC)
	assert.Equal(t, 1, req.ID)
	assert.Equal(t, "resources/list", req.Method)
}

func TestMCPResponse(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]string{"status": "ok"},
	}

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	assert.NotNil(t, resp.Result)
	assert.Nil(t, resp.Error)
}

func TestMCPError(t *testing.T) {
	err := MCPError{
		Code:    -32601,
		Message: "Method not found",
	}

	assert.Equal(t, -32601, err.Code)
	assert.Equal(t, "Method not found", err.Message)
}

func TestServerCreation(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true

	plugin := NewPlugin(WithDefaultConfig(config))
	server, err := NewServer(config, plugin)

	require.NoError(t, err)
	require.NotNil(t, server)
	assert.NotNil(t, server.resources)
	assert.NotNil(t, server.tools)
	assert.NotNil(t, server.security)
}

func TestServerHandleInitialize(t *testing.T) {
	config := DefaultConfig()
	plugin := NewPlugin(WithDefaultConfig(config))
	server, err := NewServer(config, plugin)
	require.NoError(t, err)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := server.handleInitialize(context.Background(), req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestServerHandlePing(t *testing.T) {
	config := DefaultConfig()
	plugin := NewPlugin(WithDefaultConfig(config))
	server, err := NewServer(config, plugin)
	require.NoError(t, err)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "ping",
	}

	resp := server.handlePing(context.Background(), req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 2, resp.ID)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "ok", result["status"])
}

func TestServerHandleResourcesList(t *testing.T) {
	config := DefaultConfig()
	plugin := NewPlugin(WithDefaultConfig(config))
	server, err := NewServer(config, plugin)
	require.NoError(t, err)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "resources/list",
	}

	resp := server.handleResourcesList(context.Background(), req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	resources, ok := result["resources"].([]ResourceDescription)
	require.True(t, ok)
	assert.Greater(t, len(resources), 0) // Should have registered resources
}

func TestServerHandleToolsList(t *testing.T) {
	config := DefaultConfig()
	plugin := NewPlugin(WithDefaultConfig(config))
	server, err := NewServer(config, plugin)
	require.NoError(t, err)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/list",
	}

	resp := server.handleToolsList(context.Background(), req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)

	tools, ok := result["tools"].([]ToolDescription)
	require.True(t, ok)
	assert.Greater(t, len(tools), 0) // Should have registered tools
}

func TestServerHandleUnknownMethod(t *testing.T) {
	config := DefaultConfig()
	plugin := NewPlugin(WithDefaultConfig(config))
	server, err := NewServer(config, plugin)
	require.NoError(t, err)

	req := &MCPRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "unknown/method",
	}

	resp := server.handleRequest(context.Background(), req)

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "method not found")
}
