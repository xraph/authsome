package mcp

// Mode defines the MCP plugin operation mode
type Mode string

const (
	// ModeReadOnly only allows read operations and read-only tools
	ModeReadOnly Mode = "readonly"
	// ModeAdmin allows read and administrative write operations
	ModeAdmin Mode = "admin"
	// ModeDevelopment allows all operations including test data creation
	ModeDevelopment Mode = "development"
)

// Transport defines how MCP communicates
type Transport string

const (
	// TransportStdio uses stdin/stdout (for local CLI)
	TransportStdio Transport = "stdio"
	// TransportHTTP uses HTTP endpoints (for remote access)
	TransportHTTP Transport = "http"
)

// Config defines the MCP plugin configuration
type Config struct {
	// Enabled determines if MCP plugin is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Mode controls what operations are allowed
	Mode Mode `json:"mode" yaml:"mode"`

	// Transport specifies communication method
	Transport Transport `json:"transport" yaml:"transport"`

	// Port for HTTP transport (default: 9090)
	Port int `json:"port" yaml:"port"`

	// ExposeSecrets determines if secrets are exposed (dev only)
	ExposeSecrets bool `json:"expose_secrets" yaml:"expose_secrets"`

	// Authorization settings
	Authorization AuthorizationConfig `json:"authorization" yaml:"authorization"`

	// RateLimit settings
	RateLimit RateLimitConfig `json:"rate_limit" yaml:"rate_limit"`
}

// AuthorizationConfig defines authorization requirements
type AuthorizationConfig struct {
	// RequireAPIKey enforces API key authentication
	RequireAPIKey bool `json:"require_api_key" yaml:"require_api_key"`

	// AllowedOperations lists permitted read-only operations
	AllowedOperations []string `json:"allowed_operations" yaml:"allowed_operations"`

	// AdminOperations require admin role (only in admin/development mode)
	AdminOperations []string `json:"admin_operations" yaml:"admin_operations"`
}

// RateLimitConfig defines rate limiting
type RateLimitConfig struct {
	// RequestsPerMinute limits MCP requests
	RequestsPerMinute int `json:"requests_per_minute" yaml:"requests_per_minute"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Enabled:       false, // Opt-in
		Mode:          ModeReadOnly,
		Transport:     TransportStdio,
		Port:          9090,
		ExposeSecrets: false,
		Authorization: AuthorizationConfig{
			RequireAPIKey: true,
			AllowedOperations: []string{
				"query_user",
				"list_sessions",
				"check_permission",
				"search_audit_logs",
				"explain_route",
				"validate_policy",
			},
			AdminOperations: []string{
				"revoke_session",
				"rotate_api_key",
			},
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
		},
	}
}
