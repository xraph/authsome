package secrets

import "time"

// Config holds the secrets plugin configuration
type Config struct {
	// Encryption settings
	Encryption EncryptionConfig `json:"encryption" yaml:"encryption"`

	// ConfigSource settings for Forge integration
	ConfigSource ConfigSourceConfig `json:"configSource" yaml:"configSource"`

	// Access control settings
	Access AccessConfig `json:"access" yaml:"access"`

	// Versioning settings
	Versioning VersioningConfig `json:"versioning" yaml:"versioning"`

	// Audit settings
	Audit AuditConfig `json:"audit" yaml:"audit"`

	// Dashboard settings
	Dashboard DashboardConfig `json:"dashboard" yaml:"dashboard"`
}

// EncryptionConfig holds encryption settings
type EncryptionConfig struct {
	// MasterKey is the base64-encoded 32-byte master key for encryption
	// This should be set via AUTHSOME_SECRETS_MASTER_KEY environment variable
	MasterKey string `json:"masterKey" yaml:"masterKey"`

	// RotateKeyAfter specifies the duration after which to warn about key rotation
	// Default: 365 days
	RotateKeyAfter time.Duration `json:"rotateKeyAfter" yaml:"rotateKeyAfter"`

	// TestOnStartup tests encryption on startup to verify configuration
	// Default: true
	TestOnStartup bool `json:"testOnStartup" yaml:"testOnStartup"`
}

// ConfigSourceConfig holds Forge ConfigSource integration settings
type ConfigSourceConfig struct {
	// Enabled enables the Forge ConfigSource integration
	// When enabled, secrets can be accessed via Forge's ConfigManager
	// Default: false
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Prefix is the path prefix for secrets to include in config
	// Only secrets with paths starting with this prefix will be exposed
	// Empty string means all secrets are exposed
	Prefix string `json:"prefix" yaml:"prefix"`

	// RefreshInterval is how often to refresh the config cache from database
	// Default: 5 minutes
	RefreshInterval time.Duration `json:"refreshInterval" yaml:"refreshInterval"`

	// AutoRefresh enables automatic refresh on secret changes via hooks
	// Default: true
	AutoRefresh bool `json:"autoRefresh" yaml:"autoRefresh"`

	// Priority is the config source priority (higher = more important)
	// Default: 100
	Priority int `json:"priority" yaml:"priority"`
}

// AccessConfig holds access control settings
type AccessConfig struct {
	// RequireAuthentication requires authentication for all secret access
	// Default: true
	RequireAuthentication bool `json:"requireAuthentication" yaml:"requireAuthentication"`

	// RequireRBAC enables RBAC checks for secret access
	// Default: true
	RequireRBAC bool `json:"requireRbac" yaml:"requireRbac"`

	// AllowAPIAccess allows API access to secrets
	// Default: true
	AllowAPIAccess bool `json:"allowApiAccess" yaml:"allowApiAccess"`

	// AllowDashboardAccess allows dashboard access to secrets
	// Default: true
	AllowDashboardAccess bool `json:"allowDashboardAccess" yaml:"allowDashboardAccess"`

	// RateLimitPerMinute is the rate limit for secret access per minute
	// 0 means no rate limiting
	// Default: 0
	RateLimitPerMinute int `json:"rateLimitPerMinute" yaml:"rateLimitPerMinute"`
}

// VersioningConfig holds versioning settings
type VersioningConfig struct {
	// MaxVersions is the maximum number of versions to keep per secret
	// When exceeded, old versions are automatically deleted
	// Default: 50
	MaxVersions int `json:"maxVersions" yaml:"maxVersions"`

	// RetentionDays is how long to keep old versions in days
	// Versions older than this are eligible for cleanup
	// Default: 90
	RetentionDays int `json:"retentionDays" yaml:"retentionDays"`

	// AutoCleanup enables automatic cleanup of old versions
	// Default: true
	AutoCleanup bool `json:"autoCleanup" yaml:"autoCleanup"`

	// CleanupInterval is how often to run version cleanup
	// Default: 24 hours
	CleanupInterval time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
}

// AuditConfig holds audit settings
type AuditConfig struct {
	// EnableAccessLog enables access logging for secrets
	// Default: true
	EnableAccessLog bool `json:"enableAccessLog" yaml:"enableAccessLog"`

	// LogReads logs read access (can be verbose)
	// Default: false
	LogReads bool `json:"logReads" yaml:"logReads"`

	// LogWrites logs write access (create, update, delete)
	// Default: true
	LogWrites bool `json:"logWrites" yaml:"logWrites"`

	// RetentionDays is how long to keep audit logs in days
	// Default: 365
	RetentionDays int `json:"retentionDays" yaml:"retentionDays"`

	// AutoCleanup enables automatic cleanup of old audit logs
	// Default: true
	AutoCleanup bool `json:"autoCleanup" yaml:"autoCleanup"`
}

// DashboardConfig holds dashboard-specific settings
type DashboardConfig struct {
	// EnableTreeView enables the tree view in the dashboard
	// Default: true
	EnableTreeView bool `json:"enableTreeView" yaml:"enableTreeView"`

	// EnableReveal enables the reveal value feature in the dashboard
	// Default: true
	EnableReveal bool `json:"enableReveal" yaml:"enableReveal"`

	// RevealTimeout is how long to show the revealed value before auto-hiding
	// Default: 30 seconds
	RevealTimeout time.Duration `json:"revealTimeout" yaml:"revealTimeout"`

	// EnableExport enables exporting secrets (requires admin)
	// Default: false
	EnableExport bool `json:"enableExport" yaml:"enableExport"`

	// EnableImport enables importing secrets (requires admin)
	// Default: false
	EnableImport bool `json:"enableImport" yaml:"enableImport"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Encryption: EncryptionConfig{
			RotateKeyAfter: 365 * 24 * time.Hour, // 1 year
			TestOnStartup:  true,
		},
		ConfigSource: ConfigSourceConfig{
			Enabled:         false,
			Prefix:          "",
			RefreshInterval: 5 * time.Minute,
			AutoRefresh:     true,
			Priority:        100,
		},
		Access: AccessConfig{
			RequireAuthentication: true,
			RequireRBAC:           true,
			AllowAPIAccess:        true,
			AllowDashboardAccess:  true,
			RateLimitPerMinute:    0,
		},
		Versioning: VersioningConfig{
			MaxVersions:     50,
			RetentionDays:   90,
			AutoCleanup:     true,
			CleanupInterval: 24 * time.Hour,
		},
		Audit: AuditConfig{
			EnableAccessLog: true,
			LogReads:        false,
			LogWrites:       true,
			RetentionDays:   365,
			AutoCleanup:     true,
		},
		Dashboard: DashboardConfig{
			EnableTreeView: true,
			EnableReveal:   true,
			RevealTimeout:  30 * time.Second,
			EnableExport:   false,
			EnableImport:   false,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Master key validation is done during encryption service initialization
	// Additional validation can be added here as needed

	if c.Versioning.MaxVersions < 1 {
		c.Versioning.MaxVersions = 1
	}

	if c.Versioning.RetentionDays < 1 {
		c.Versioning.RetentionDays = 1
	}

	if c.ConfigSource.RefreshInterval < time.Second {
		c.ConfigSource.RefreshInterval = time.Second
	}

	if c.ConfigSource.Priority < 0 {
		c.ConfigSource.Priority = 0
	}

	return nil
}

// Merge merges another config into this one (non-zero values override)
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Encryption
	if other.Encryption.MasterKey != "" {
		c.Encryption.MasterKey = other.Encryption.MasterKey
	}
	if other.Encryption.RotateKeyAfter > 0 {
		c.Encryption.RotateKeyAfter = other.Encryption.RotateKeyAfter
	}

	// ConfigSource
	if other.ConfigSource.Enabled {
		c.ConfigSource.Enabled = other.ConfigSource.Enabled
	}
	if other.ConfigSource.Prefix != "" {
		c.ConfigSource.Prefix = other.ConfigSource.Prefix
	}
	if other.ConfigSource.RefreshInterval > 0 {
		c.ConfigSource.RefreshInterval = other.ConfigSource.RefreshInterval
	}
	if other.ConfigSource.Priority > 0 {
		c.ConfigSource.Priority = other.ConfigSource.Priority
	}

	// Access
	c.Access.RequireAuthentication = other.Access.RequireAuthentication
	c.Access.RequireRBAC = other.Access.RequireRBAC
	c.Access.AllowAPIAccess = other.Access.AllowAPIAccess
	c.Access.AllowDashboardAccess = other.Access.AllowDashboardAccess
	if other.Access.RateLimitPerMinute > 0 {
		c.Access.RateLimitPerMinute = other.Access.RateLimitPerMinute
	}

	// Versioning
	if other.Versioning.MaxVersions > 0 {
		c.Versioning.MaxVersions = other.Versioning.MaxVersions
	}
	if other.Versioning.RetentionDays > 0 {
		c.Versioning.RetentionDays = other.Versioning.RetentionDays
	}
	c.Versioning.AutoCleanup = other.Versioning.AutoCleanup
	if other.Versioning.CleanupInterval > 0 {
		c.Versioning.CleanupInterval = other.Versioning.CleanupInterval
	}

	// Audit
	c.Audit.EnableAccessLog = other.Audit.EnableAccessLog
	c.Audit.LogReads = other.Audit.LogReads
	c.Audit.LogWrites = other.Audit.LogWrites
	if other.Audit.RetentionDays > 0 {
		c.Audit.RetentionDays = other.Audit.RetentionDays
	}
	c.Audit.AutoCleanup = other.Audit.AutoCleanup

	// Dashboard
	c.Dashboard.EnableTreeView = other.Dashboard.EnableTreeView
	c.Dashboard.EnableReveal = other.Dashboard.EnableReveal
	if other.Dashboard.RevealTimeout > 0 {
		c.Dashboard.RevealTimeout = other.Dashboard.RevealTimeout
	}
	c.Dashboard.EnableExport = other.Dashboard.EnableExport
	c.Dashboard.EnableImport = other.Dashboard.EnableImport
}

