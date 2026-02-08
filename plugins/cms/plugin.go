// Package cms provides a content management system plugin for AuthSome.
// It allows defining custom content types with configurable fields and
// managing content entries through the dashboard UI and REST API.
package cms

import (
	"context"
	"fmt"
	"sync"

	"github.com/uptrace/bun"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/plugins/cms/handlers"
	"github.com/xraph/authsome/plugins/cms/repository"
	"github.com/xraph/authsome/plugins/cms/schema"
	"github.com/xraph/authsome/plugins/cms/service"
)

const (
	// PluginID is the unique identifier for the CMS plugin
	PluginID = "cms"
	// PluginName is the human-readable name
	PluginName = "Content Management System"
	// PluginVersion is the current version
	PluginVersion = "1.0.0"
	// PluginDescription describes the plugin
	PluginDescription = "Headless CMS with custom content types, dynamic forms, and full query language support"
)

// Plugin implements the CMS plugin for AuthSome
type Plugin struct {
	config        *Config
	defaultConfig *Config
	db            *bun.DB
	logger        forge.Logger
	authInst      core.Authsome

	// Services
	contentTypeSvc     *service.ContentTypeService
	fieldSvc           *service.ContentFieldService
	entrySvc           *service.ContentEntryService
	revisionSvc        *service.RevisionService
	componentSchemaSvc *service.ComponentSchemaService

	// Dashboard extension (lazy initialized)
	dashboardExt     *DashboardExtension
	dashboardExtOnce sync.Once
}

// PluginOption is a functional option for configuring the plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration
func WithDefaultConfig(cfg *Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithEnableRevisions enables/disables revision tracking
func WithEnableRevisions(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Features.EnableRevisions = enabled
	}
}

// WithEnableDrafts enables/disables draft workflow
func WithEnableDrafts(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Features.EnableDrafts = enabled
	}
}

// WithEnableScheduling enables/disables scheduled publishing
func WithEnableScheduling(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Features.EnableScheduling = enabled
	}
}

// WithEnableSearch enables/disables full-text search
func WithEnableSearch(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Features.EnableSearch = enabled
	}
}

// WithEnableRelations enables/disables content relations
func WithEnableRelations(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Features.EnableRelations = enabled
	}
}

// WithMaxContentTypes sets the maximum number of content types
func WithMaxContentTypes(max int) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Limits.MaxContentTypes = max
	}
}

// WithMaxFieldsPerType sets the maximum fields per content type
func WithMaxFieldsPerType(max int) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Limits.MaxFieldsPerType = max
	}
}

// WithMaxRevisionsPerEntry sets the maximum revisions per entry
func WithMaxRevisionsPerEntry(max int) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.Revisions.MaxRevisionsPerEntry = max
	}
}

// WithPublicAPI enables/disables public API access
func WithPublicAPI(enabled bool) PluginOption {
	return func(p *Plugin) {
		if p.defaultConfig == nil {
			p.defaultConfig = DefaultConfig()
		}
		p.defaultConfig.API.EnablePublicAPI = enabled
	}
}

// NewPlugin creates a new CMS plugin instance
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		defaultConfig: DefaultConfig(),
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
func (p *Plugin) Priority() int {
	return 0 // Normal priority
}

// Dependencies returns the plugin dependencies
func (p *Plugin) Dependencies() []string {
	return []string{"multiapp"} // Requires multiapp for app/environment context
}

// Init initializes the plugin with dependencies from the Auth instance
func (p *Plugin) Init(auth core.Authsome) error {
	p.authInst = auth
	p.db = auth.GetDB()
	forgeApp := auth.GetForgeApp()
	p.logger = forgeApp.Logger().With(forge.F("plugin", PluginID))

	p.logger.Debug("initializing CMS plugin")

	// Load configuration
	configManager := forgeApp.Config()
	p.config = p.defaultConfig
	if p.config == nil {
		p.config = DefaultConfig()
	}

	// Try to load config from file
	var fileConfig Config
	if err := configManager.Bind("auth.cms", &fileConfig); err == nil {
		p.config.Merge(&fileConfig)
	}

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid CMS configuration: %w", err)
	}

	// Initialize repositories
	contentTypeRepo := repository.NewContentTypeRepository(p.db)
	fieldRepo := repository.NewContentFieldRepository(p.db)
	entryRepo := repository.NewContentEntryRepository(p.db)
	revisionRepo := repository.NewRevisionRepository(p.db)
	componentSchemaRepo := repository.NewComponentSchemaRepository(p.db)

	// Initialize services
	p.contentTypeSvc = service.NewContentTypeService(
		contentTypeRepo,
		fieldRepo,
		service.ContentTypeServiceConfig{
			MaxContentTypes: p.config.Limits.MaxContentTypes,
			Logger:          p.logger,
		},
	)

	p.fieldSvc = service.NewContentFieldService(
		fieldRepo,
		contentTypeRepo,
		service.ContentFieldServiceConfig{
			MaxFieldsPerType: p.config.Limits.MaxFieldsPerType,
			Logger:           p.logger,
		},
	)

	p.entrySvc = service.NewContentEntryService(
		entryRepo,
		contentTypeRepo,
		revisionRepo,
		service.ContentEntryServiceConfig{
			EnableRevisions:      p.config.Features.EnableRevisions,
			MaxRevisionsPerEntry: p.config.Revisions.MaxRevisionsPerEntry,
			Logger:               p.logger,
		},
	)

	p.revisionSvc = service.NewRevisionService(revisionRepo, p.logger)

	p.componentSchemaSvc = service.NewComponentSchemaService(
		componentSchemaRepo,
		service.ComponentSchemaServiceConfig{
			MaxComponentSchemas: p.config.Limits.MaxComponentSchemas,
			Logger:              p.logger,
		},
	)

	// Dashboard extension is lazy-initialized when first accessed via DashboardExtension()
	// This ensures all plugin dependencies are ready before extension creation

	// Register services in DI container if available
	if container := forgeApp.Container(); container != nil {
		if err := p.RegisterServices(container); err != nil {
			p.logger.Warn("failed to register CMS services in DI container", forge.F("error", err.Error()))
		}
	}

	p.logger.Info("CMS plugin initialized",
		forge.F("enableRevisions", p.config.Features.EnableRevisions),
		forge.F("enableDrafts", p.config.Features.EnableDrafts),
		forge.F("enableSearch", p.config.Features.EnableSearch),
		forge.F("maxContentTypes", p.config.Limits.MaxContentTypes))

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	p.logger.Debug("registering CMS routes")

	// Create handlers
	contentTypeHandler := handlers.NewContentTypeHandler(p.contentTypeSvc, p.fieldSvc)
	contentEntryHandler := handlers.NewContentEntryHandler(p.entrySvc, p.contentTypeSvc)
	revisionHandler := handlers.NewRevisionHandler(p.revisionSvc, p.entrySvc, p.contentTypeSvc)

	// API routes under /cms
	cms := router.Group("/cms")

	// Health check
	cms.GET("/ping", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"status":  "ok",
			"plugin":  PluginID,
			"version": PluginVersion,
		})
	},
		forge.WithName("cms.ping"),
		forge.WithSummary("CMS health check"),
		forge.WithDescription("Verify that the CMS plugin is loaded and working"),
		forge.WithTags("CMS", "Health"),
	)

	// Field Types endpoint
	cms.GET("/field-types", contentTypeHandler.GetFieldTypes,
		forge.WithName("cms.field_types"),
		forge.WithSummary("List available field types"),
		forge.WithDescription("Returns all available field types that can be used in content type definitions"),
		forge.WithTags("CMS", "Content Types"),
	)

	// ==========================================================================
	// Content Type Routes
	// ==========================================================================
	types := cms.Group("/types")

	// List content types
	types.GET("", contentTypeHandler.ListContentTypes,
		forge.WithName("cms.content_types.list"),
		forge.WithSummary("List content types"),
		forge.WithDescription("Returns all content types in the current app/environment"),
		forge.WithTags("CMS", "Content Types"),
	)

	// Create content type
	types.POST("", contentTypeHandler.CreateContentType,
		forge.WithName("cms.content_types.create"),
		forge.WithSummary("Create content type"),
		forge.WithDescription("Creates a new content type definition"),
		forge.WithTags("CMS", "Content Types"),
	)

	// Get content type by slug
	types.GET("/:slug", contentTypeHandler.GetContentType,
		forge.WithName("cms.content_types.get"),
		forge.WithSummary("Get content type"),
		forge.WithDescription("Returns a specific content type by its name"),
		forge.WithTags("CMS", "Content Types"),
	)

	// Update content type
	types.PUT("/:slug", contentTypeHandler.UpdateContentType,
		forge.WithName("cms.content_types.update"),
		forge.WithSummary("Update content type"),
		forge.WithDescription("Updates an existing content type definition"),
		forge.WithTags("CMS", "Content Types"),
	)

	// Delete content type
	types.DELETE("/:slug", contentTypeHandler.DeleteContentType,
		forge.WithName("cms.content_types.delete"),
		forge.WithSummary("Delete content type"),
		forge.WithDescription("Deletes a content type and all its fields (entries must be deleted first)"),
		forge.WithTags("CMS", "Content Types"),
	)

	// ==========================================================================
	// Content Field Routes
	// ==========================================================================
	fields := types.Group("/:slug/fields")

	// List fields
	fields.GET("", contentTypeHandler.ListFields,
		forge.WithName("cms.content_fields.list"),
		forge.WithSummary("List fields"),
		forge.WithDescription("Returns all fields for a content type"),
		forge.WithTags("CMS", "Content Fields"),
	)

	// Add field
	fields.POST("", contentTypeHandler.AddField,
		forge.WithName("cms.content_fields.create"),
		forge.WithSummary("Add field"),
		forge.WithDescription("Adds a new field to a content type"),
		forge.WithTags("CMS", "Content Fields"),
	)

	// Reorder fields
	fields.POST("/reorder", contentTypeHandler.ReorderFields,
		forge.WithName("cms.content_fields.reorder"),
		forge.WithSummary("Reorder fields"),
		forge.WithDescription("Changes the display order of fields"),
		forge.WithTags("CMS", "Content Fields"),
	)

	// Get field
	fields.GET("/:fieldSlug", contentTypeHandler.GetField,
		forge.WithName("cms.content_fields.get"),
		forge.WithSummary("Get field"),
		forge.WithDescription("Returns a specific field by slug"),
		forge.WithTags("CMS", "Content Fields"),
	)

	// Update field
	fields.PUT("/:fieldSlug", contentTypeHandler.UpdateField,
		forge.WithName("cms.content_fields.update"),
		forge.WithSummary("Update field"),
		forge.WithDescription("Updates an existing field definition"),
		forge.WithTags("CMS", "Content Fields"),
	)

	// Delete field
	fields.DELETE("/:fieldSlug", contentTypeHandler.DeleteField,
		forge.WithName("cms.content_fields.delete"),
		forge.WithSummary("Delete field"),
		forge.WithDescription("Removes a field from a content type"),
		forge.WithTags("CMS", "Content Fields"),
	)

	// ==========================================================================
	// Content Entry Routes
	// ==========================================================================
	entries := cms.Group("/:typeSlug")

	// List entries
	entries.GET("", contentEntryHandler.ListEntries,
		forge.WithName("cms.entries.list"),
		forge.WithSummary("List entries"),
		forge.WithDescription("Returns entries for a content type with filtering and pagination"),
		forge.WithTags("CMS", "Content Entries"),
	)

	// Create entry
	entries.POST("", contentEntryHandler.CreateEntry,
		forge.WithName("cms.entries.create"),
		forge.WithSummary("Create entry"),
		forge.WithDescription("Creates a new content entry"),
		forge.WithTags("CMS", "Content Entries"),
	)

	// Advanced query
	entries.POST("/query", contentEntryHandler.QueryEntries,
		forge.WithName("cms.entries.query"),
		forge.WithSummary("Query entries"),
		forge.WithDescription("Performs an advanced query on entries using the CMS query language"),
		forge.WithTags("CMS", "Content Entries"),
	)

	// Stats
	entries.GET("/stats", contentEntryHandler.GetEntryStats,
		forge.WithName("cms.entries.stats"),
		forge.WithSummary("Get entry stats"),
		forge.WithDescription("Returns statistics for entries of a content type"),
		forge.WithTags("CMS", "Content Entries"),
	)

	// Bulk operations
	bulk := entries.Group("/bulk")

	bulk.POST("/publish", contentEntryHandler.BulkPublish,
		forge.WithName("cms.entries.bulk_publish"),
		forge.WithSummary("Bulk publish"),
		forge.WithDescription("Publishes multiple entries at once"),
		forge.WithTags("CMS", "Content Entries", "Bulk"),
	)

	bulk.POST("/unpublish", contentEntryHandler.BulkUnpublish,
		forge.WithName("cms.entries.bulk_unpublish"),
		forge.WithSummary("Bulk unpublish"),
		forge.WithDescription("Unpublishes multiple entries at once"),
		forge.WithTags("CMS", "Content Entries", "Bulk"),
	)

	bulk.POST("/delete", contentEntryHandler.BulkDelete,
		forge.WithName("cms.entries.bulk_delete"),
		forge.WithSummary("Bulk delete"),
		forge.WithDescription("Deletes multiple entries at once"),
		forge.WithTags("CMS", "Content Entries", "Bulk"),
	)

	// Individual entry routes
	entry := entries.Group("/:entryId")

	entry.GET("", contentEntryHandler.GetEntry,
		forge.WithName("cms.entries.get"),
		forge.WithSummary("Get entry"),
		forge.WithDescription("Returns a specific entry by ID"),
		forge.WithTags("CMS", "Content Entries"),
	)

	entry.PUT("", contentEntryHandler.UpdateEntry,
		forge.WithName("cms.entries.update"),
		forge.WithSummary("Update entry"),
		forge.WithDescription("Updates an existing entry"),
		forge.WithTags("CMS", "Content Entries"),
	)

	entry.DELETE("", contentEntryHandler.DeleteEntry,
		forge.WithName("cms.entries.delete"),
		forge.WithSummary("Delete entry"),
		forge.WithDescription("Deletes an entry"),
		forge.WithTags("CMS", "Content Entries"),
	)

	entry.POST("/publish", contentEntryHandler.PublishEntry,
		forge.WithName("cms.entries.publish"),
		forge.WithSummary("Publish entry"),
		forge.WithDescription("Publishes a draft entry or schedules it for publication"),
		forge.WithTags("CMS", "Content Entries"),
	)

	entry.POST("/unpublish", contentEntryHandler.UnpublishEntry,
		forge.WithName("cms.entries.unpublish"),
		forge.WithSummary("Unpublish entry"),
		forge.WithDescription("Moves a published entry back to draft status"),
		forge.WithTags("CMS", "Content Entries"),
	)

	entry.POST("/archive", contentEntryHandler.ArchiveEntry,
		forge.WithName("cms.entries.archive"),
		forge.WithSummary("Archive entry"),
		forge.WithDescription("Archives an entry"),
		forge.WithTags("CMS", "Content Entries"),
	)

	// ==========================================================================
	// Revision Routes
	// ==========================================================================
	revisions := entry.Group("/revisions")

	revisions.GET("", revisionHandler.ListRevisions,
		forge.WithName("cms.revisions.list"),
		forge.WithSummary("List revisions"),
		forge.WithDescription("Returns the revision history for an entry"),
		forge.WithTags("CMS", "Revisions"),
	)

	revisions.GET("/compare", revisionHandler.CompareRevisions,
		forge.WithName("cms.revisions.compare"),
		forge.WithSummary("Compare revisions"),
		forge.WithDescription("Compares two revisions and returns the differences"),
		forge.WithTags("CMS", "Revisions"),
	)

	revisions.GET("/:version", revisionHandler.GetRevision,
		forge.WithName("cms.revisions.get"),
		forge.WithSummary("Get revision"),
		forge.WithDescription("Returns a specific revision by version number"),
		forge.WithTags("CMS", "Revisions"),
	)

	revisions.POST("/:version/restore", revisionHandler.RestoreRevision,
		forge.WithName("cms.revisions.restore"),
		forge.WithSummary("Restore revision"),
		forge.WithDescription("Restores an entry to a specific revision"),
		forge.WithTags("CMS", "Revisions"),
	)

	p.logger.Debug("registered CMS routes")
	return nil
}

// RegisterHooks registers the plugin's hooks
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	// CMS hooks will be implemented in later phases
	// - Before/after entry create
	// - Before/after entry update
	// - Before/after entry delete
	// - Before/after publish
	// - Scheduled publishing job
	return nil
}

// RegisterServiceDecorators allows the plugin to decorate core services
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// No service decorators needed for CMS plugin
	return nil
}

// RegisterRoles registers RBAC roles for the plugin
func (p *Plugin) RegisterRoles(roleRegistry rbac.RoleRegistryInterface) error {
	p.logger.Debug("registering CMS RBAC roles")

	// CMS Admin - full access
	err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "cms_admin",
		DisplayName: "CMS Administrator",
		Description: "Full CMS administration access",
		Permissions: []string{
			"create on cms_content_types",
			"read on cms_content_types",
			"update on cms_content_types",
			"delete on cms_content_types",
			"create on cms_content_entries",
			"read on cms_content_entries",
			"update on cms_content_entries",
			"delete on cms_content_entries",
			"publish on cms_content_entries",
			"read on cms_content_revisions",
			"rollback on cms_content_revisions",
		},
	})
	if err != nil {
		p.logger.Warn("failed to register cms_admin role", forge.F("error", err.Error()))
	}

	// CMS Editor - manage entries only
	err = roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "cms_editor",
		DisplayName: "CMS Editor",
		Description: "Create and manage content entries",
		Permissions: []string{
			"read on cms_content_types",
			"create on cms_content_entries",
			"read on cms_content_entries",
			"update on cms_content_entries",
			"publish on cms_content_entries",
			"read on cms_content_revisions",
		},
	})
	if err != nil {
		p.logger.Warn("failed to register cms_editor role", forge.F("error", err.Error()))
	}

	// CMS Author - create and manage own entries
	err = roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "cms_author",
		DisplayName: "CMS Author",
		Description: "Create and manage own content entries",
		Permissions: []string{
			"read on cms_content_types",
			"create on cms_content_entries",
			"read on cms_content_entries where createdBy = @user.id",
			"update on cms_content_entries where createdBy = @user.id",
		},
	})
	if err != nil {
		p.logger.Warn("failed to register cms_author role", forge.F("error", err.Error()))
	}

	// CMS Viewer - read only
	err = roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "cms_viewer",
		DisplayName: "CMS Viewer",
		Description: "View content entries",
		Permissions: []string{
			"read on cms_content_types",
			"read on cms_content_entries",
		},
	})
	if err != nil {
		p.logger.Warn("failed to register cms_viewer role", forge.F("error", err.Error()))
	}

	p.logger.Debug("registered CMS RBAC roles")
	return nil
}

// DashboardExtension returns the dashboard extension for the plugin
// Uses lazy initialization to ensure plugin is fully initialized before creating extension
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	p.dashboardExtOnce.Do(func() {
		p.dashboardExt = NewDashboardExtension(p)
	})
	return p.dashboardExt
}

// Migrate runs database migrations for the plugin
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	p.logger.Debug("running CMS migrations")

	// Create cms_content_types table
	if _, err := p.db.NewCreateTable().
		Model((*schema.ContentType)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_content_types table: %w", err)
	}

	// Create cms_content_fields table
	if _, err := p.db.NewCreateTable().
		Model((*schema.ContentField)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_content_fields table: %w", err)
	}

	// Create cms_content_entries table
	if _, err := p.db.NewCreateTable().
		Model((*schema.ContentEntry)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_content_entries table: %w", err)
	}

	// Create cms_content_revisions table
	if _, err := p.db.NewCreateTable().
		Model((*schema.ContentRevision)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_content_revisions table: %w", err)
	}

	// Create cms_content_relations table (for many-to-many)
	if _, err := p.db.NewCreateTable().
		Model((*schema.ContentRelation)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_content_relations table: %w", err)
	}

	// Create cms_content_type_relations table
	if _, err := p.db.NewCreateTable().
		Model((*schema.ContentTypeRelation)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_content_type_relations table: %w", err)
	}

	// Create cms_component_schemas table
	if _, err := p.db.NewCreateTable().
		Model((*schema.ComponentSchema)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to create cms_component_schemas table: %w", err)
	}

	// Drop old indexes with old column names
	dropIndexes := []string{
		`DROP INDEX IF EXISTS idx_cms_content_types_slug`,
		`DROP INDEX IF EXISTS idx_cms_content_fields_slug`,
		`DROP INDEX IF EXISTS idx_cms_component_schemas_slug`,
		`DROP INDEX IF EXISTS idx_cms_content_relations_unique`,
		`DROP INDEX IF EXISTS idx_cms_content_types_name`,
		`DROP INDEX IF EXISTS idx_cms_content_fields_name`,
		`DROP INDEX IF EXISTS idx_cms_component_schemas_name`,
	}

	for _, idx := range dropIndexes {
		if _, err := p.db.ExecContext(ctx, idx); err != nil {
			p.logger.Warn("failed to drop old index", forge.F("error", err.Error()))
		}
	}

	// Create indexes (case-insensitive for name fields)
	indexes := []string{
		// Content Types - CASE INSENSITIVE
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cms_content_types_name ON cms_content_types(app_id, environment_id, LOWER(name)) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_types_app_env ON cms_content_types(app_id, environment_id) WHERE deleted_at IS NULL`,

		// Content Fields - CASE INSENSITIVE
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cms_content_fields_name ON cms_content_fields(content_type_id, LOWER(name))`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_fields_type ON cms_content_fields(content_type_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_fields_order ON cms_content_fields(content_type_id, "order")`,

		// Content Entries
		`CREATE INDEX IF NOT EXISTS idx_cms_content_entries_type ON cms_content_entries(content_type_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_entries_status ON cms_content_entries(status) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_entries_app_env ON cms_content_entries(app_id, environment_id) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_entries_created ON cms_content_entries(created_at) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_entries_updated ON cms_content_entries(updated_at) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_entries_scheduled ON cms_content_entries(scheduled_at) WHERE status = 'scheduled' AND deleted_at IS NULL`,

		// Content Revisions
		`CREATE INDEX IF NOT EXISTS idx_cms_content_revisions_entry ON cms_content_revisions(entry_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cms_content_revisions_version ON cms_content_revisions(entry_id, version)`,

		// Content Relations (many-to-many) - CASE INSENSITIVE for field_name
		`CREATE INDEX IF NOT EXISTS idx_cms_content_relations_source ON cms_content_relations(source_entry_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_relations_target ON cms_content_relations(target_entry_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cms_content_relations_unique ON cms_content_relations(source_entry_id, target_entry_id, LOWER(field_name))`,

		// Content Type Relations
		`CREATE INDEX IF NOT EXISTS idx_cms_content_type_relations_source ON cms_content_type_relations(source_content_type_id)`,
		`CREATE INDEX IF NOT EXISTS idx_cms_content_type_relations_target ON cms_content_type_relations(target_content_type_id)`,

		// Component Schemas - CASE INSENSITIVE
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cms_component_schemas_name ON cms_component_schemas(app_id, environment_id, LOWER(name)) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cms_component_schemas_app_env ON cms_component_schemas(app_id, environment_id) WHERE deleted_at IS NULL`,
	}

	for _, idx := range indexes {
		if _, err := p.db.ExecContext(ctx, idx); err != nil {
			// Log warning but don't fail - index might already exist or use different syntax
			p.logger.Warn("failed to create index", forge.F("error", err.Error()))
		}
	}

	p.logger.Info("CMS migrations completed")
	return nil
}

// =============================================================================
// Public Accessors
// =============================================================================

// Config returns the plugin configuration
func (p *Plugin) Config() *Config {
	return p.config
}

// Logger returns the plugin logger
func (p *Plugin) Logger() forge.Logger {
	return p.logger
}

// DB returns the database connection
func (p *Plugin) DB() *bun.DB {
	return p.db
}
