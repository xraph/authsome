package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/uptrace/bun"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/secrets/schema"
)

const (
	// PluginID is the unique identifier for the secrets plugin
	PluginID = "secrets"
	// PluginName is the human-readable name
	PluginName = "Secrets Manager"
	// PluginVersion is the current version
	PluginVersion = "1.0.0"
	// PluginDescription describes the plugin
	PluginDescription = "Secure secrets and configuration management with encryption, versioning, and Forge ConfigSource integration"

	// Environment variable for master key
	EnvMasterKey = "AUTHSOME_SECRETS_MASTER_KEY"
)

// Plugin implements the secrets management plugin for AuthSome
type Plugin struct {
	config        *Config
	defaultConfig *Config
	service       *Service
	handler       *Handler
	encryption    *EncryptionService
	validator     *SchemaValidator
	db            *bun.DB
	logger        forge.Logger
	authInst      core.Authsome

	// ConfigSource integration
	configSources map[string]*SecretsConfigSource // "appID:envID" -> source

	// Dashboard extension
	dashboardExt *DashboardExtension
}

// PluginOption is a functional option for configuring the plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration
func WithDefaultConfig(cfg *Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithMasterKey sets the encryption master key
func WithMasterKey(key string) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Encryption.MasterKey = key
	}
}

// WithConfigSourceEnabled enables the Forge ConfigSource integration
func WithConfigSourceEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.ConfigSource.Enabled = enabled
	}
}

// WithConfigSourcePrefix sets the config source prefix
func WithConfigSourcePrefix(prefix string) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.ConfigSource.Prefix = prefix
	}
}

// WithAuditEnabled enables/disables audit logging
func WithAuditEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Audit.EnableAccessLog = enabled
	}
}

// WithMaxVersions sets the maximum versions to keep
func WithMaxVersions(max int) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Versioning.MaxVersions = max
	}
}

// NewPlugin creates a new secrets plugin instance
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		defaultConfig: DefaultConfig(),
		configSources: make(map[string]*SecretsConfigSource),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// =============================================================================
// Plugin Interface Implementation
// =============================================================================

// ID returns the unique plugin identifier
func (p *Plugin) ID() string {
	return PluginID
}

// Name returns the human-readable plugin name
func (p *Plugin) Name() string {
	return PluginName
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return PluginVersion
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return PluginDescription
}

// Priority returns the plugin initialization priority
// Lower values = higher priority (load first)
// Secrets should load early to provide config values
func (p *Plugin) Priority() int {
	return -100 // High priority
}

// Init initializes the plugin with dependencies from the Auth instance
func (p *Plugin) Init(auth core.Authsome) error {
	p.authInst = auth
	p.db = auth.GetDB()
	forgeApp := auth.GetForgeApp()
	p.logger = forgeApp.Logger().With(forge.F("plugin", PluginID))

	p.logger.Info("initializing secrets plugin")

	// Load configuration
	configManager := forgeApp.Config()
	p.config = p.defaultConfig
	if p.config == nil {
		p.config = DefaultConfig()
	}

	// Try to load config from file
	var fileConfig Config
	if err := configManager.Bind("auth.secrets", &fileConfig); err == nil {
		p.config.Merge(&fileConfig)
	}

	// Check for master key in environment variable
	if p.config.Encryption.MasterKey == "" {
		p.config.Encryption.MasterKey = os.Getenv(EnvMasterKey)
	}

	// Validate master key
	if p.config.Encryption.MasterKey == "" {
		return errs.InternalServerError(
			fmt.Sprintf("secrets plugin requires encryption master key; set %s environment variable", EnvMasterKey),
			nil,
		)
	}

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return errs.InternalServerError("invalid secrets configuration", err)
	}

	// Initialize encryption service
	var err error
	p.encryption, err = NewEncryptionService(p.config.Encryption.MasterKey)
	if err != nil {
		return errs.InternalServerError("failed to initialize encryption", err)
	}

	// Test encryption if configured
	if p.config.Encryption.TestOnStartup {
		if err := p.encryption.TestEncryption(); err != nil {
			return errs.InternalServerError("encryption test failed", err)
		}
		p.logger.Debug("encryption service verified")
	}

	// Initialize validator
	p.validator = NewSchemaValidator()

	// Initialize repository
	repo := NewRepository(p.db)

	// Get audit service from registry
	serviceRegistry := auth.GetServiceRegistry()
	auditSvc := serviceRegistry.AuditService()

	// Initialize service
	p.service = NewService(repo, p.encryption, p.validator, auditSvc, p.config, p.logger)

	// Initialize handler
	p.handler = NewHandler(p.service, p.logger)

	// Initialize dashboard extension
	p.dashboardExt = NewDashboardExtension(p)

	p.logger.Info("secrets plugin initialized",
		forge.F("configSourceEnabled", p.config.ConfigSource.Enabled),
		forge.F("auditEnabled", p.config.Audit.EnableAccessLog),
		forge.F("maxVersions", p.config.Versioning.MaxVersions))

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("secrets handler not initialized; call Init first")
	}

	// API routes under /secrets
	secrets := router.Group("/secrets")

	// List and create
	secrets.GET("", p.handler.List,
		forge.WithName("secrets.list"),
		forge.WithSummary("List secrets"),
		forge.WithDescription("List secrets with optional filtering and pagination"),
		forge.WithTags("Secrets"))

	secrets.POST("", p.handler.Create,
		forge.WithName("secrets.create"),
		forge.WithSummary("Create a secret"),
		forge.WithDescription("Create a new encrypted secret"),
		forge.WithTags("Secrets"))

	// Stats and tree (before :id routes)
	secrets.GET("/stats", p.handler.GetStats,
		forge.WithName("secrets.stats"),
		forge.WithSummary("Get secrets statistics"),
		forge.WithDescription("Get statistics about secrets"),
		forge.WithTags("Secrets"))

	secrets.GET("/tree", p.handler.GetTree,
		forge.WithName("secrets.tree"),
		forge.WithSummary("Get secrets tree"),
		forge.WithDescription("Get secrets organized in a tree structure"),
		forge.WithTags("Secrets"))

	// Path-based access
	secrets.GET("/path/*path", p.handler.GetByPath,
		forge.WithName("secrets.getByPath"),
		forge.WithSummary("Get secret by path"),
		forge.WithDescription("Retrieve a secret by its hierarchical path"),
		forge.WithTags("Secrets"))

	// Single secret operations
	secrets.GET("/:id", p.handler.Get,
		forge.WithName("secrets.get"),
		forge.WithSummary("Get a secret"),
		forge.WithDescription("Retrieve secret metadata by ID"),
		forge.WithTags("Secrets"))

	secrets.GET("/:id/value", p.handler.GetValue,
		forge.WithName("secrets.getValue"),
		forge.WithSummary("Get secret value"),
		forge.WithDescription("Retrieve the decrypted secret value"),
		forge.WithTags("Secrets"))

	secrets.PUT("/:id", p.handler.Update,
		forge.WithName("secrets.update"),
		forge.WithSummary("Update a secret"),
		forge.WithDescription("Update an existing secret"),
		forge.WithTags("Secrets"))

	secrets.DELETE("/:id", p.handler.Delete,
		forge.WithName("secrets.delete"),
		forge.WithSummary("Delete a secret"),
		forge.WithDescription("Soft-delete a secret"),
		forge.WithTags("Secrets"))

	// Version operations
	secrets.GET("/:id/versions", p.handler.GetVersions,
		forge.WithName("secrets.versions"),
		forge.WithSummary("Get secret versions"),
		forge.WithDescription("Get version history for a secret"),
		forge.WithTags("Secrets"))

	secrets.POST("/:id/rollback/:version", p.handler.Rollback,
		forge.WithName("secrets.rollback"),
		forge.WithSummary("Rollback secret"),
		forge.WithDescription("Rollback a secret to a previous version"),
		forge.WithTags("Secrets"))

	p.logger.Debug("registered secrets routes")
	return nil
}

// RegisterHooks registers the plugin's hooks
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	// Note: The hook registry doesn't support generic custom event hooks.
	// Config source refresh is handled directly in the service methods.
	// If auto-refresh is needed, the service will call refreshConfigSources after
	// create/update/delete operations.

	if p.config.ConfigSource.Enabled && p.config.ConfigSource.AutoRefresh {
		p.logger.Debug("config source auto-refresh enabled; refresh will be triggered by service operations")
	}

	return nil
}

// RegisterServiceDecorators allows the plugin to decorate core services
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// No service decorators needed for secrets plugin
	return nil
}

// RegisterRoles registers RBAC roles for the plugin
func (p *Plugin) RegisterRoles(roleRegistry rbac.RoleRegistryInterface) error {
	// Register secrets admin role
	err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "secrets_admin",
		Description: "Full access to manage secrets",
		Permissions: []string{
			"create on secrets",
			"read on secrets",
			"update on secrets",
			"delete on secrets",
			"view on secrets",
			"rollback on secrets",
		},
	})
	if err != nil {
		p.logger.Warn("failed to register secrets_admin role", forge.F("error", err.Error()))
	}

	// Register secrets viewer role
	err = roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "secrets_viewer",
		Description: "Read-only access to secrets (metadata only)",
		Permissions: []string{
			"view on secrets",
		},
	})
	if err != nil {
		p.logger.Warn("failed to register secrets_viewer role", forge.F("error", err.Error()))
	}

	p.logger.Debug("registered secrets RBAC roles")
	return nil
}

// DashboardExtension returns the dashboard extension for the plugin
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	return p.dashboardExt
}

// Migrate runs database migrations for the plugin
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	p.logger.Debug("running secrets migrations")

	// Create secrets table
	if _, err := p.db.NewCreateTable().
		Model((*schema.Secret)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create secrets table: %w", err)
	}

	// Create secret_versions table
	if _, err := p.db.NewCreateTable().
		Model((*schema.SecretVersion)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create secret_versions table: %w", err)
	}

	// Create secret_access_logs table
	if _, err := p.db.NewCreateTable().
		Model((*schema.SecretAccessLog)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create secret_access_logs table: %w", err)
	}

	// Create indexes
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_secrets_app_env_path ON secrets(app_id, environment_id, path) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_secrets_app_env ON secrets(app_id, environment_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_secrets_tags ON secrets USING GIN(tags) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_secret_versions_secret ON secret_versions(secret_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_secret_versions_unique ON secret_versions(secret_id, version)`,
		`CREATE INDEX IF NOT EXISTS idx_secret_access_logs_secret ON secret_access_logs(secret_id)`,
		`CREATE INDEX IF NOT EXISTS idx_secret_access_logs_app_env ON secret_access_logs(app_id, environment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_secret_access_logs_created ON secret_access_logs(created_at)`,
	}

	for _, idx := range indexes {
		if _, err := p.db.ExecContext(ctx, idx); err != nil {
			// Log warning but don't fail - index might already exist or use different syntax
			p.logger.Warn("failed to create index", forge.F("error", err.Error()))
		}
	}

	p.logger.Info("secrets migrations completed")
	return nil
}

// =============================================================================
// ConfigSource Integration
// =============================================================================

// GetConfigSource returns a config source for the given app/environment
func (p *Plugin) GetConfigSource(appID, envID string) *SecretsConfigSource {
	key := appID + ":" + envID
	return p.configSources[key]
}

// CreateConfigSource creates a new config source for an app/environment
func (p *Plugin) CreateConfigSource(appID, envID string) (*SecretsConfigSource, error) {
	if !p.config.ConfigSource.Enabled {
		return nil, fmt.Errorf("config source integration is not enabled")
	}

	key := appID + ":" + envID

	// Return existing if available
	if source, ok := p.configSources[key]; ok {
		return source, nil
	}

	// Create new source
	source := NewSecretsConfigSource(
		p.service,
		appID,
		envID,
		p.config.ConfigSource.Prefix,
		p.config.ConfigSource.Priority,
		p.logger,
	)

	p.configSources[key] = source
	return source, nil
}

// refreshConfigSources refreshes all config sources
func (p *Plugin) refreshConfigSources(ctx context.Context) {
	for _, source := range p.configSources {
		if err := source.Reload(ctx); err != nil {
			p.logger.Warn("failed to refresh config source",
				forge.F("source", source.Name()),
				forge.F("error", err.Error()))
		}
	}
}

// =============================================================================
// Public Accessors
// =============================================================================

// Service returns the secrets service
func (p *Plugin) Service() *Service {
	return p.service
}

// Config returns the plugin configuration
func (p *Plugin) Config() *Config {
	return p.config
}

// Logger returns the plugin logger
func (p *Plugin) Logger() forge.Logger {
	return p.logger
}

