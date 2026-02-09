package permissions

import (
	"fmt"
	"time"

	"github.com/xraph/forge"
)

// Config represents the permissions plugin configuration.
type Config struct {
	// Enabled controls whether the permissions system is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Mode determines the evaluation mode
	// - "strict": Only use permissions system (RBAC disabled)
	// - "hybrid": Try permissions first, fallback to RBAC
	// - "rbac-primary": Try RBAC first, fallback to permissions
	Mode string `json:"mode" yaml:"mode"`

	// Engine configuration
	Engine EngineConfig `json:"engine" yaml:"engine"`

	// Cache configuration
	Cache CacheConfig `json:"cache" yaml:"cache"`

	// Performance tuning
	Performance PerformanceConfig `json:"performance" yaml:"performance"`

	// Migration settings
	Migration MigrationConfig `json:"migration" yaml:"migration"`

	// Organization-specific overrides
	Organizations map[string]*OrgConfig `json:"organizations" yaml:"organizations"`
}

// EngineConfig controls the policy evaluation engine.
type EngineConfig struct {
	// MaxPolicyComplexity limits the number of operations in a policy
	// Default: 100
	MaxPolicyComplexity int `json:"maxPolicyComplexity" yaml:"maxPolicyComplexity"`

	// EvaluationTimeout is the maximum time for policy evaluation
	// Default: 10ms
	EvaluationTimeout time.Duration `json:"evaluationTimeout" yaml:"evaluationTimeout"`

	// MaxPoliciesPerOrg limits policies per organization
	// Default: 10000
	MaxPoliciesPerOrg int `json:"maxPoliciesPerOrg" yaml:"maxPoliciesPerOrg"`

	// ParallelEvaluation enables concurrent policy evaluation
	// Default: true
	ParallelEvaluation bool `json:"parallelEvaluation" yaml:"parallelEvaluation"`

	// MaxParallelEvaluations controls concurrency level
	// Default: 4
	MaxParallelEvaluations int `json:"maxParallelEvaluations" yaml:"maxParallelEvaluations"`

	// EnableAttributeCaching caches attribute lookups
	// Default: true
	EnableAttributeCaching bool `json:"enableAttributeCaching" yaml:"enableAttributeCaching"`

	// AttributeCacheTTL is the TTL for attribute cache
	// Default: 5 minutes
	AttributeCacheTTL time.Duration `json:"attributeCacheTTL" yaml:"attributeCacheTTL"`
}

// CacheConfig controls caching behavior.
type CacheConfig struct {
	// Enabled controls whether caching is active
	// Default: true
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Backend specifies the cache backend
	// Options: "memory", "redis", "hybrid"
	// Default: "hybrid"
	Backend string `json:"backend" yaml:"backend"`

	// LocalCacheSize is the size of the in-memory LRU cache
	// Default: 10000
	LocalCacheSize int `json:"localCacheSize" yaml:"localCacheSize"`

	// LocalCacheTTL is the TTL for local cache entries
	// Default: 5 minutes
	LocalCacheTTL time.Duration `json:"localCacheTTL" yaml:"localCacheTTL"`

	// RedisTTL is the TTL for Redis cache entries
	// Default: 15 minutes
	RedisTTL time.Duration `json:"redisTTL" yaml:"redisTTL"`

	// WarmupOnStart pre-loads policies on startup
	// Default: true
	WarmupOnStart bool `json:"warmupOnStart" yaml:"warmupOnStart"`

	// InvalidateOnChange immediately invalidates cache on policy changes
	// Default: true
	InvalidateOnChange bool `json:"invalidateOnChange" yaml:"invalidateOnChange"`
}

// PerformanceConfig controls performance tuning.
type PerformanceConfig struct {
	// EnableMetrics enables Prometheus metrics
	// Default: true
	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`

	// EnableTracing enables OpenTelemetry tracing
	// Default: false (enable in production)
	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`

	// TraceSamplingRate is the percentage of requests to trace
	// Default: 0.01 (1%)
	TraceSamplingRate float64 `json:"traceSamplingRate" yaml:"traceSamplingRate"`

	// SlowQueryThreshold logs queries slower than this
	// Default: 5ms
	SlowQueryThreshold time.Duration `json:"slowQueryThreshold" yaml:"slowQueryThreshold"`

	// EnableProfiling enables pprof endpoints
	// Default: false (enable for debugging)
	EnableProfiling bool `json:"enableProfiling" yaml:"enableProfiling"`
}

// MigrationConfig controls RBAC → Permissions migration.
type MigrationConfig struct {
	// AutoMigrate automatically converts RBAC policies
	// Default: false (requires manual migration)
	AutoMigrate bool `json:"autoMigrate" yaml:"autoMigrate"`

	// ValidateEquivalence checks that migrated policies match RBAC
	// Default: true
	ValidateEquivalence bool `json:"validateEquivalence" yaml:"validateEquivalence"`

	// KeepRBACPolicies retains RBAC policies after migration
	// Default: true (safe to delete after validation)
	KeepRBACPolicies bool `json:"keepRBACPolicies" yaml:"keepRBACPolicies"`

	// DryRun simulates migration without making changes
	// Default: false
	DryRun bool `json:"dryRun" yaml:"dryRun"`
}

// OrgConfig allows organization-specific overrides.
type OrgConfig struct {
	// Enabled controls if permissions are enabled for this org
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// MaxPolicies overrides the global limit for this org
	MaxPolicies *int `json:"maxPolicies,omitempty" yaml:"maxPolicies,omitempty"`

	// CustomResources defines org-specific resource types
	CustomResources []string `json:"customResources,omitempty" yaml:"customResources,omitempty"`

	// CustomActions defines org-specific actions
	CustomActions []string `json:"customActions,omitempty" yaml:"customActions,omitempty"`

	// TemplateID specifies which platform template to inherit
	TemplateID *string `json:"templateId,omitempty" yaml:"templateId,omitempty"`

	// InheritPlatform controls platform policy inheritance
	InheritPlatform *bool `json:"inheritPlatform,omitempty" yaml:"inheritPlatform,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Mode:    "hybrid",
		Engine: EngineConfig{
			MaxPolicyComplexity:    100,
			EvaluationTimeout:      10 * time.Millisecond,
			MaxPoliciesPerOrg:      10000,
			ParallelEvaluation:     true,
			MaxParallelEvaluations: 4,
			EnableAttributeCaching: true,
			AttributeCacheTTL:      5 * time.Minute,
		},
		Cache: CacheConfig{
			Enabled:            true,
			Backend:            "hybrid",
			LocalCacheSize:     10000,
			LocalCacheTTL:      5 * time.Minute,
			RedisTTL:           15 * time.Minute,
			WarmupOnStart:      true,
			InvalidateOnChange: true,
		},
		Performance: PerformanceConfig{
			EnableMetrics:      true,
			EnableTracing:      false,
			TraceSamplingRate:  0.01,
			SlowQueryThreshold: 5 * time.Millisecond,
			EnableProfiling:    false,
		},
		Migration: MigrationConfig{
			AutoMigrate:         false,
			ValidateEquivalence: true,
			KeepRBACPolicies:    true,
			DryRun:              false,
		},
		Organizations: make(map[string]*OrgConfig),
	}
}

// LoadConfig loads configuration from Forge config manager.
func LoadConfig(configManager forge.ConfigManager) (*Config, error) {
	config := DefaultConfig()

	// Bind main config
	if err := configManager.Bind("auth.permissions", config); err != nil {
		return nil, fmt.Errorf("failed to bind permissions config: %w", err)
	}

	// Load organization-specific overrides
	if err := configManager.Bind("auth.permissions.organizations", &config.Organizations); err != nil {
		// Organization configs are optional
		config.Organizations = make(map[string]*OrgConfig)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid permissions config: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// Validate mode
	validModes := map[string]bool{
		"strict":       true,
		"hybrid":       true,
		"rbac-primary": true,
	}
	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode: %s (must be: strict, hybrid, rbac-primary)", c.Mode)
	}

	// Validate engine config
	if c.Engine.MaxPolicyComplexity < 10 {
		return fmt.Errorf("maxPolicyComplexity too low: %d (minimum: 10)", c.Engine.MaxPolicyComplexity)
	}

	if c.Engine.MaxPolicyComplexity > 10000 {
		return fmt.Errorf("maxPolicyComplexity too high: %d (maximum: 10000)", c.Engine.MaxPolicyComplexity)
	}

	if c.Engine.EvaluationTimeout < time.Millisecond {
		return fmt.Errorf("evaluationTimeout too low: %s (minimum: 1ms)", c.Engine.EvaluationTimeout)
	}

	if c.Engine.EvaluationTimeout > time.Second {
		return fmt.Errorf("evaluationTimeout too high: %s (maximum: 1s)", c.Engine.EvaluationTimeout)
	}

	if c.Engine.MaxPoliciesPerOrg < 1 {
		return fmt.Errorf("maxPoliciesPerOrg too low: %d (minimum: 1)", c.Engine.MaxPoliciesPerOrg)
	}

	if c.Engine.MaxPoliciesPerOrg > 100000 {
		return fmt.Errorf("maxPoliciesPerOrg too high: %d (maximum: 100000)", c.Engine.MaxPoliciesPerOrg)
	}

	if c.Engine.MaxParallelEvaluations < 1 {
		return fmt.Errorf("maxParallelEvaluations too low: %d (minimum: 1)", c.Engine.MaxParallelEvaluations)
	}

	if c.Engine.MaxParallelEvaluations > 32 {
		return fmt.Errorf("maxParallelEvaluations too high: %d (maximum: 32)", c.Engine.MaxParallelEvaluations)
	}

	// Validate cache config
	validBackends := map[string]bool{
		"memory": true,
		"redis":  true,
		"hybrid": true,
	}
	if !validBackends[c.Cache.Backend] {
		return fmt.Errorf("invalid cache backend: %s (must be: memory, redis, hybrid)", c.Cache.Backend)
	}

	if c.Cache.LocalCacheSize < 100 {
		return fmt.Errorf("localCacheSize too low: %d (minimum: 100)", c.Cache.LocalCacheSize)
	}

	if c.Cache.LocalCacheSize > 1000000 {
		return fmt.Errorf("localCacheSize too high: %d (maximum: 1000000)", c.Cache.LocalCacheSize)
	}

	// Validate performance config
	if c.Performance.TraceSamplingRate < 0 || c.Performance.TraceSamplingRate > 1 {
		return fmt.Errorf("traceSamplingRate must be between 0 and 1: %f", c.Performance.TraceSamplingRate)
	}

	return nil
}

// GetOrgConfig returns the effective configuration for an organization.
func (c *Config) GetOrgConfig(orgID string) *Config {
	// Start with global config
	orgConfig := *c

	// Apply organization-specific overrides
	if override, exists := c.Organizations[orgID]; exists {
		if override.Enabled != nil {
			orgConfig.Enabled = *override.Enabled
		}

		if override.MaxPolicies != nil {
			orgConfig.Engine.MaxPoliciesPerOrg = *override.MaxPolicies
		}
	}

	return &orgConfig
}

// MergeOrgConfig merges organization-specific settings.
func (c *Config) MergeOrgConfig(orgID string, override *OrgConfig) {
	if c.Organizations == nil {
		c.Organizations = make(map[string]*OrgConfig)
	}

	c.Organizations[orgID] = override
}

// Example configuration in YAML format:
/*
auth:
  permissions:
    enabled: true
    mode: hybrid  # strict, hybrid, rbac-primary

    engine:
      maxPolicyComplexity: 100
      evaluationTimeout: 10ms
      maxPoliciesPerOrg: 10000
      parallelEvaluation: true
      maxParallelEvaluations: 4
      enableAttributeCaching: true
      attributeCacheTTL: 5m

    cache:
      enabled: true
      backend: hybrid  # memory, redis, hybrid
      localCacheSize: 10000
      localCacheTTL: 5m
      redisTTL: 15m
      warmupOnStart: true
      invalidateOnChange: true

    performance:
      enableMetrics: true
      enableTracing: false
      traceSamplingRate: 0.01
      slowQueryThreshold: 5ms
      enableProfiling: false

    migration:
      autoMigrate: false
      validateEquivalence: true
      keepRBACPolicies: true
      dryRun: false

    # Organization-specific overrides
    organizations:
      org_abc123:
        enabled: true
        maxPolicies: 20000
        customResources:
          - "document"
          - "workspace"
          - "project"
        customActions:
          - "view"
          - "edit"
          - "share"
          - "export"
        templateId: "enterprise-base"
        inheritPlatform: true

      org_xyz789:
        enabled: true
        maxPolicies: 5000
        inheritPlatform: false
*/
