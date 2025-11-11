package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	aud "github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/organization"
	rl "github.com/xraph/authsome/core/ratelimit"
	rbac "github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	sec "github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/authsome/internal/dbschema"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/plugins"
	jwtplugin "github.com/xraph/authsome/plugins/jwt"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/routes"
	"github.com/xraph/authsome/schema"
	memstore "github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/database"
)

// Service name constants for DI container
const (
	ServiceDatabase       = "authsome.database"
	ServiceUser           = "authsome.user"
	ServiceSession        = "authsome.session"
	ServiceAuth           = "authsome.auth"
	ServiceOrganization   = "authsome.organization"
	ServiceRateLimit      = "authsome.ratelimit"
	ServiceDevice         = "authsome.device"
	ServiceSecurity       = "authsome.security"
	ServiceAudit          = "authsome.audit"
	ServiceRBAC           = "authsome.rbac"
	ServiceWebhook        = "authsome.webhook"
	ServiceNotification   = "authsome.notification"
	ServiceJWT            = "authsome.jwt"
	ServiceAPIKey         = "authsome.apikey"
	ServiceHookRegistry   = "authsome.hooks"
	ServicePluginRegistry = "authsome.plugins"
)

// Auth is the main authentication instance
type Auth struct {
	config   Config
	forgeApp forge.App
	db       interface{} // Will be *bun.DB

	// Core services (using interfaces to allow plugin decoration)
	userService         user.ServiceInterface
	sessionService      session.ServiceInterface
	authService         auth.ServiceInterface
	organizationService *organization.Service
	rateLimitService    *rl.Service
	rateLimitStorage    rl.Storage
	rateLimitConfig     rl.Config
	deviceService       *dev.Service
	securityService     *sec.Service
	securityConfig      sec.Config
	geoipProvider       sec.GeoIPProvider
	auditService        *aud.Service
	rbacService         *rbac.Service
	userRoleRepo        *repo.UserRoleRepository
	roleRepo            *repo.RoleRepository
	policyRepo          *repo.PolicyRepository
	twofaRepo           *repo.TwoFARepository

	// Phase 10 services
	webhookService      *webhook.Service
	notificationService *notification.Service
	jwtService          *jwt.Service
	apikeyService       *apikey.Service

	// Plugin registry
	pluginRegistry *plugins.Registry

	// Service registry for plugin decorator pattern
	serviceRegistry *registry.ServiceRegistry
	hookRegistry    *hooks.HookRegistry
}

// New creates a new Auth instance with the given options
func New(opts ...Option) *Auth {
	a := &Auth{
		config: Config{
			Mode:     ModeStandalone,
			BasePath: "/api/auth",
		},
		pluginRegistry:  plugins.NewRegistry(),
		serviceRegistry: registry.NewServiceRegistry(),
		hookRegistry:    hooks.NewHookRegistry(),
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Initialize initializes all core services
func (a *Auth) Initialize(ctx context.Context) error {
	if a.forgeApp == nil {
		return fmt.Errorf("forge app not set")
	}

	// Resolve database from various sources
	if err := a.resolveDatabase(); err != nil {
		return fmt.Errorf("failed to resolve database: %w", err)
	}

	fmt.Println("[AuthSome] Resolved database", a.db)
	// Cast database
	db, ok := a.db.(*bun.DB)
	if !ok || db == nil {
		return errors.New("invalid or missing bun.DB instance")
	}

	// Apply custom database schema if configured
	// This creates the schema and sets the search_path for all subsequent operations
	if a.config.DatabaseSchema != "" {
		if err := dbschema.ApplySchema(ctx, db, a.config.DatabaseSchema); err != nil {
			return fmt.Errorf("failed to apply database schema '%s': %w", a.config.DatabaseSchema, err)
		}
		fmt.Printf("[AuthSome] ✅ Applied custom database schema: %s\n", a.config.DatabaseSchema)
	}

	// Initialize repositories
	userRepo := repo.NewUserRepository(db)
	sessionRepo := repo.NewSessionRepository(db)
	a.twofaRepo = repo.NewTwoFARepository(db)
	orgRepo := repo.NewOrganizationRepository(db)
	// Rate limit storage and service
	if a.rateLimitStorage == nil {
		a.rateLimitStorage = memstore.NewMemoryStorage()
	}
	a.rateLimitService = rl.NewService(a.rateLimitStorage, a.rateLimitConfig)
	// Device service
	deviceRepo := repo.NewDeviceRepository(db)
	a.deviceService = dev.NewService(deviceRepo)
	// Security service
	secRepo := repo.NewSecurityRepository(db)
	// Default enable if not explicitly set
	if !a.securityConfig.Enabled && len(a.securityConfig.IPWhitelist) == 0 && len(a.securityConfig.IPBlacklist) == 0 && len(a.securityConfig.AllowedCountries) == 0 && len(a.securityConfig.BlockedCountries) == 0 {
		a.securityConfig.Enabled = true
	}
	a.securityService = sec.NewService(secRepo, a.securityConfig)
	if a.geoipProvider != nil {
		a.securityService.SetGeoIPProvider(a.geoipProvider)
	}
	// Audit service
	auditRepo := repo.NewAuditRepository(db)
	a.auditService = aud.NewService(auditRepo)

	// RBAC service: load policies from storage
	a.rbacService = rbac.NewService()
	a.policyRepo = repo.NewPolicyRepository(db)
	_ = a.rbacService.LoadPolicies(ctx, a.policyRepo)
	// Seed default policies if storage is empty
	if exprs, err := a.policyRepo.ListAll(ctx); err == nil && len(exprs) == 0 {
		defaults := []string{
			// Organization permissions
			"role:owner:create,read,update,delete on organization:*",
			"role:admin:read,update on organization:*",
			// Member permissions
			"role:owner:create,read,update,delete on member:*",
			"role:admin:create,read,update,delete on member:*",
			// Team permissions
			"role:owner:create,read,update,delete on team:*",
			"role:admin:create,read,update,delete on team:*",
			// Invitations
			"role:owner:create,read,update,delete on invitation:*",
			"role:admin:create,read,update on invitation:*",
			// Roles and assignments
			"role:owner:create,read,update,delete on role:*",
			"role:admin:read,update on role:*",
			// Policy management
			"role:owner:create,read,update,delete on policy:*",
			"role:admin:read on policy:*",
		}
		for _, ex := range defaults {
			_ = a.policyRepo.Create(ctx, ex)
		}
		_ = a.rbacService.LoadPolicies(ctx, a.policyRepo)
	}
	// User role repository for RBAC role assignments
	a.userRoleRepo = repo.NewUserRoleRepository(db)
	// Role repository for role management
	a.roleRepo = repo.NewRoleRepository(db)

	// Initialize Phase 10 services first (webhook is a dependency for user/session services)
	webhookRepo := repo.NewWebhookRepository(db)
	a.webhookService = webhook.NewService(webhook.Config{}, webhookRepo, a.auditService)

	notificationRepo := repo.NewNotificationRepository(db)
	// TODO: Add template engine when available
	a.notificationService = notification.NewService(notificationRepo, nil, a.auditService, notification.Config{})

	// JWT service uses a wrapper around the JWT key repository
	jwtKeyRepo := repo.NewJWTKeyRepository(db)
	jwtRepo := jwt.NewRepositoryWrapper(jwtKeyRepo)
	a.jwtService = jwt.NewService(jwt.Config{}, jwtRepo, a.auditService)

	apikeyRepo := repo.NewAPIKeyRepository(db)
	a.apikeyService = apikey.NewService(apikeyRepo, a.auditService, apikey.Config{})

	// Initialize services (now with webhook service dependency)
	a.userService = user.NewService(userRepo, user.Config{
		PasswordRequirements: validator.DefaultPasswordRequirements(),
	}, a.webhookService)
	a.sessionService = session.NewService(sessionRepo, session.Config{}, a.webhookService)
	a.authService = auth.NewService(a.userService, a.sessionService, auth.Config{})
	a.organizationService = organization.NewService(orgRepo, organization.Config{ModeSaaS: a.config.Mode == ModeSaaS})

	// Populate service registry BEFORE plugin initialization
	// This allows plugins to access and decorate services
	a.serviceRegistry.SetUserService(a.userService)
	a.serviceRegistry.SetSessionService(a.sessionService)
	a.serviceRegistry.SetAuthService(a.authService)
	a.serviceRegistry.SetJWTService(a.jwtService)
	a.serviceRegistry.SetAPIKeyService(a.apikeyService)
	a.serviceRegistry.SetAuditService(a.auditService)
	a.serviceRegistry.SetWebhookService(a.webhookService)
	a.serviceRegistry.SetNotificationService(a.notificationService)
	a.serviceRegistry.SetDeviceService(a.deviceService)
	a.serviceRegistry.SetRBACService(a.rbacService)
	a.serviceRegistry.SetRateLimitService(a.rateLimitService)
	a.serviceRegistry.SetOrganizationService(a.organizationService)

	// Register services into Forge DI container
	if err := a.registerServicesIntoContainer(db); err != nil {
		return fmt.Errorf("failed to register services into DI container: %w", err)
	}

	// Ensure platform organization exists before plugins initialize
	// This is needed for role bootstrap later
	platformOrg, err := a.ensurePlatformOrganization(ctx)
	if err != nil {
		return fmt.Errorf("failed to ensure platform organization: %w", err)
	}

	// Register default platform roles before plugins initialize
	// Plugins can then extend or override these roles
	if err := rbac.RegisterDefaultPlatformRoles(a.serviceRegistry.RoleRegistry()); err != nil {
		return fmt.Errorf("failed to register default platform roles: %w", err)
	}

	// Initialize plugins with full Auth instance and run complete lifecycle
	if a.pluginRegistry != nil {
		for _, p := range a.pluginRegistry.List() {
			// 1. Initialize plugin with Auth instance (not just DB)
			if err := p.Init(a); err != nil {
				return fmt.Errorf("plugin %s init failed: %w", p.ID(), err)
			}

			// 2. Register roles (optional interface)
			// If plugin implements PluginWithRoles, it can register its roles
			if rolePlugin, ok := p.(interface {
				RegisterRoles(registry interface{}) error
			}); ok {
				fmt.Printf("[AuthSome] Plugin %s registering roles...\n", p.ID())
				if err := rolePlugin.RegisterRoles(a.serviceRegistry.RoleRegistry()); err != nil {
					return fmt.Errorf("plugin %s register roles failed: %w", p.ID(), err)
				}
			}

			// 3. Register hooks
			if err := p.RegisterHooks(a.hookRegistry); err != nil {
				return fmt.Errorf("plugin %s register hooks failed: %w", p.ID(), err)
			}

			// 4. Register service decorators (plugins can replace core services)
			if err := p.RegisterServiceDecorators(a.serviceRegistry); err != nil {
				return fmt.Errorf("plugin %s register decorators failed: %w", p.ID(), err)
			}

			// 5. Run migrations
			if err := p.Migrate(); err != nil {
				return fmt.Errorf("plugin %s migrate failed: %w", p.ID(), err)
			}
		}

		// After plugins have potentially replaced services, update Auth's references
		// to use the decorated versions from the registry
		if a.serviceRegistry.UserService() != nil {
			a.userService = a.serviceRegistry.UserService()
		}
		if a.serviceRegistry.SessionService() != nil {
			a.sessionService = a.serviceRegistry.SessionService()
		}
		if a.serviceRegistry.AuthService() != nil {
			a.authService = a.serviceRegistry.AuthService()
		}
	}

	// Bootstrap roles and permissions to platform organization
	// This happens AFTER plugins have registered their roles
	if err := a.bootstrapRoles(ctx, db, platformOrg.ID); err != nil {
		return fmt.Errorf("failed to bootstrap roles: %w", err)
	}

	return nil
}

// ensurePlatformOrganization ensures the platform organization exists
// This is the single foundational organization for the entire system
// Returns the platform organization (existing or newly created)
func (a *Auth) ensurePlatformOrganization(ctx context.Context) (*schema.Organization, error) {
	db, ok := a.db.(*bun.DB)
	if !ok {
		return nil, fmt.Errorf("invalid database instance")
	}

	// Check if platform org exists
	var platformOrg schema.Organization
	err := db.NewSelect().
		Model(&platformOrg).
		Where("is_platform = ?", true).
		Scan(ctx)

	if err == nil {
		// Platform org exists
		fmt.Printf("[AuthSome] ✅ Platform organization found: %s (ID: %s)\n",
			platformOrg.Name, platformOrg.ID.String())
		return &platformOrg, nil
	}

	// Platform org doesn't exist - create it
	fmt.Println("[AuthSome] Platform organization not found, creating...")

	platformOrg = schema.Organization{
		ID:         xid.New(),
		Name:       "Platform Organization",
		Slug:       "platform",
		IsPlatform: true,
		Metadata:   map[string]interface{}{},
	}
	platformOrg.CreatedAt = time.Now()
	platformOrg.UpdatedAt = time.Now()
	platformOrg.CreatedBy = platformOrg.ID // Self-created
	platformOrg.UpdatedBy = platformOrg.ID
	platformOrg.Version = 1

	_, err = db.NewInsert().Model(&platformOrg).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform organization: %w", err)
	}

	fmt.Printf("[AuthSome] ✅ Created platform organization: %s (ID: %s)\n",
		platformOrg.Name, platformOrg.ID.String())

	return &platformOrg, nil
}

// bootstrapRoles applies all registered roles to the platform organization
// This is called after plugins have initialized and registered their roles
func (a *Auth) bootstrapRoles(ctx context.Context, db *bun.DB, platformOrgID xid.ID) error {
	roleRegistry := a.serviceRegistry.RoleRegistry()
	if roleRegistry == nil {
		return fmt.Errorf("role registry not initialized")
	}

	fmt.Println("[AuthSome] Starting role bootstrap...")
	if err := roleRegistry.Bootstrap(ctx, db, a.rbacService, platformOrgID); err != nil {
		return fmt.Errorf("role bootstrap failed: %w", err)
	}

	fmt.Println("[AuthSome] ✅ Role bootstrap complete")
	return nil
}

// Mount mounts the auth routes to the Forge router
func (a *Auth) Mount(router forge.Router, basePath string) error {
	if a.authService == nil {
		return fmt.Errorf("auth service not initialized; call Initialize first")
	}
	if basePath == "" {
		basePath = a.config.BasePath
	}
	h := handlers.NewAuthHandler(a.authService, a.rateLimitService, a.deviceService, a.securityService, a.auditService, a.twofaRepo)
	audH := handlers.NewAuditHandler(a.auditService)
	orgH := handlers.NewOrganizationHandler(a.organizationService, a.rateLimitService, a.sessionService, a.rbacService, a.userRoleRepo, a.roleRepo, a.policyRepo, a.config.RBACEnforce)

	// Phase 10 handlers
	webhookH := handlers.NewWebhookHandler(a.webhookService)
	notificationH := handlers.NewNotificationHandler(a.notificationService)
	jwtH := jwtplugin.NewHandler(a.jwtService)
	// Note: API key routes are now handled by the apikey plugin
	// The core apikey.Service is still available for internal use

	// Register core auth routes
	routes.Register(router, basePath, h)
	routes.RegisterAudit(router, basePath, audH)

	// Check if multitenancy plugin is enabled
	hasMultitenancyPlugin := false
	if a.pluginRegistry != nil {
		for _, p := range a.pluginRegistry.List() {
			if p.ID() == "multitenancy" {
				hasMultitenancyPlugin = true
				break
			}
		}
	}

	// Only register built-in organization routes if multitenancy plugin is NOT enabled
	// This prevents route duplication and allows the plugin to fully control org routes
	if !hasMultitenancyPlugin {
		// Mount organization routes under basePath (not hardcoded)
		routes.RegisterOrganization(router, basePath+"/organizations", orgH)
		fmt.Println("[AuthSome] Registered built-in organization routes (multitenancy plugin not detected)")
	} else {
		fmt.Println("[AuthSome] Skipping built-in organization routes (multitenancy plugin detected)")

		// Register RBAC-related routes that the multitenancy plugin doesn't handle
		// These are still needed even with the multitenancy plugin
		rbacGroup := router.Group(basePath + "/organizations")
		routes.RegisterOrganizationRBAC(rbacGroup, orgH)
		fmt.Println("[AuthSome] Registered organization RBAC routes")
	}

	// Phase 10 routes - create a scoped group for these routes
	authGroup := router.Group(basePath)
	routes.RegisterWebhookRoutes(authGroup, webhookH)
	routes.RegisterNotificationRoutes(authGroup, notificationH)
	routes.RegisterJWTRoutes(authGroup, jwtH)
	// API key routes removed - handled by apikey plugin with middleware support

	// Register plugin routes (scoped to basePath)
	if a.pluginRegistry != nil {
		// Pass a group with the basePath so plugins are scoped under the auth mount point
		pluginGroup := router.Group(basePath)
		for _, p := range a.pluginRegistry.List() {
			fmt.Printf("[AuthSome] Registering routes for plugin: %s\n", p.ID())
			if err := p.RegisterRoutes(pluginGroup); err != nil {
				fmt.Printf("[AuthSome] Error registering routes for plugin %s: %v\n", p.ID(), err)
			}
		}
	}

	return nil
}

// RegisterPlugin registers a plugin
func (a *Auth) RegisterPlugin(plugin plugins.Plugin) error {
	return a.pluginRegistry.Register(plugin)
}

// GetMode returns the current mode
func (a *Auth) GetMode() Mode {
	return a.config.Mode
}

// GetConfig returns the auth config
func (a *Auth) GetConfig() Config {
	return a.config
}

// GetDB returns the database instance
func (a *Auth) GetDB() *bun.DB {
	if db, ok := a.db.(*bun.DB); ok {
		return db
	}
	return nil
}

// GetForgeApp returns the forge application instance
func (a *Auth) GetForgeApp() forge.App {
	return a.forgeApp
}

// GetServiceRegistry returns the service registry for plugins
func (a *Auth) GetServiceRegistry() *registry.ServiceRegistry {
	return a.serviceRegistry
}

// GetHookRegistry returns the hook registry for plugins
func (a *Auth) GetHookRegistry() *hooks.HookRegistry {
	return a.hookRegistry
}

// GetBasePath returns the base path for AuthSome routes
func (a *Auth) GetBasePath() string {
	return a.config.BasePath
}

// GetPluginRegistry returns the plugin registry
func (a *Auth) GetPluginRegistry() *plugins.Registry {
	return a.pluginRegistry
}

// IsPluginEnabled checks if a plugin is registered and enabled
func (a *Auth) IsPluginEnabled(pluginID string) bool {
	if a.pluginRegistry == nil {
		return false
	}
	_, exists := a.pluginRegistry.Get(pluginID)
	return exists
}

// registerServicesIntoContainer registers all AuthSome services into the Forge DI container
// This enables dependency injection for handlers, plugins, and middleware
func (a *Auth) registerServicesIntoContainer(db *bun.DB) error {
	container := a.forgeApp.Container()
	if container == nil {
		// Container is optional - if not available, skip registration
		return nil
	}

	// Register database as singleton
	if err := container.Register(ServiceDatabase, func(c forge.Container) (interface{}, error) {
		return db, nil
	}); err != nil {
		return fmt.Errorf("failed to register database: %w", err)
	}

	// Register core services as singletons
	if err := container.Register(ServiceUser, func(c forge.Container) (interface{}, error) {
		return a.userService, nil
	}); err != nil {
		return fmt.Errorf("failed to register user service: %w", err)
	}

	if err := container.Register(ServiceSession, func(c forge.Container) (interface{}, error) {
		return a.sessionService, nil
	}); err != nil {
		return fmt.Errorf("failed to register session service: %w", err)
	}

	if err := container.Register(ServiceAuth, func(c forge.Container) (interface{}, error) {
		return a.authService, nil
	}); err != nil {
		return fmt.Errorf("failed to register auth service: %w", err)
	}

	if err := container.Register(ServiceOrganization, func(c forge.Container) (interface{}, error) {
		return a.organizationService, nil
	}); err != nil {
		return fmt.Errorf("failed to register organization service: %w", err)
	}

	if err := container.Register(ServiceRateLimit, func(c forge.Container) (interface{}, error) {
		return a.rateLimitService, nil
	}); err != nil {
		return fmt.Errorf("failed to register rate limit service: %w", err)
	}

	if err := container.Register(ServiceDevice, func(c forge.Container) (interface{}, error) {
		return a.deviceService, nil
	}); err != nil {
		return fmt.Errorf("failed to register device service: %w", err)
	}

	if err := container.Register(ServiceSecurity, func(c forge.Container) (interface{}, error) {
		return a.securityService, nil
	}); err != nil {
		return fmt.Errorf("failed to register security service: %w", err)
	}

	if err := container.Register(ServiceAudit, func(c forge.Container) (interface{}, error) {
		return a.auditService, nil
	}); err != nil {
		return fmt.Errorf("failed to register audit service: %w", err)
	}

	if err := container.Register(ServiceRBAC, func(c forge.Container) (interface{}, error) {
		return a.rbacService, nil
	}); err != nil {
		return fmt.Errorf("failed to register RBAC service: %w", err)
	}

	if err := container.Register(ServiceWebhook, func(c forge.Container) (interface{}, error) {
		return a.webhookService, nil
	}); err != nil {
		return fmt.Errorf("failed to register webhook service: %w", err)
	}

	if err := container.Register(ServiceNotification, func(c forge.Container) (interface{}, error) {
		return a.notificationService, nil
	}); err != nil {
		return fmt.Errorf("failed to register notification service: %w", err)
	}

	if err := container.Register(ServiceJWT, func(c forge.Container) (interface{}, error) {
		return a.jwtService, nil
	}); err != nil {
		return fmt.Errorf("failed to register JWT service: %w", err)
	}

	if err := container.Register(ServiceAPIKey, func(c forge.Container) (interface{}, error) {
		return a.apikeyService, nil
	}); err != nil {
		return fmt.Errorf("failed to register API key service: %w", err)
	}

	// Register registries
	if err := container.Register(ServiceHookRegistry, func(c forge.Container) (interface{}, error) {
		return a.hookRegistry, nil
	}); err != nil {
		return fmt.Errorf("failed to register hook registry: %w", err)
	}

	if err := container.Register(ServicePluginRegistry, func(c forge.Container) (interface{}, error) {
		return a.pluginRegistry, nil
	}); err != nil {
		return fmt.Errorf("failed to register plugin registry: %w", err)
	}

	fmt.Println("[AuthSome] Successfully registered all services into Forge DI container")
	return nil
}

// resolveDatabase resolves the database from various sources
// Priority: Direct db > DatabaseManager > Forge DI
func (a *Auth) resolveDatabase() error {
	// If database already set directly, use it (backwards compatibility)
	if a.db != nil {
		return nil
	}

	// Try DatabaseManager if configured
	if a.config.DatabaseManager != nil {
		dbName := a.config.DatabaseManagerName
		if dbName == "" {
			dbName = "default"
		}

		db, err := a.config.DatabaseManager.SQL(dbName)
		if err != nil {
			return fmt.Errorf("failed to get database %s from DatabaseManager: %w", dbName, err)
		}

		a.db = db
		fmt.Printf("[AuthSome] Resolved database from Forge DatabaseManager: %s\n", dbName)
		return nil
	}

	// Try Forge DI container if configured
	if a.config.UseForgeDI && a.forgeApp != nil {
		container := a.forgeApp.Container()
		if container == nil {
			return fmt.Errorf("forge DI container not available")
		}

		// Try to resolve from Forge database extension
		dbInterface, err := database.GetDatabase(container)
		if err != nil {
			return fmt.Errorf("failed to resolve database from Forge DI: %w", err)
		}

		sdb, ok := dbInterface.(*database.SQLDatabase)
		if !ok {
			return fmt.Errorf("resolved database is not *database.SQLDatabase, got %T", dbInterface)
		}

		db := sdb.Bun()
		if db == nil {
			return fmt.Errorf("forge database extension returned nil *bun.DB - ensure database extension is properly initialized before authsome")
		}

		a.db = db
		fmt.Println("[AuthSome] ✅ Successfully resolved database from Forge DI container")
		return nil
	}

	return fmt.Errorf("database not configured: use WithDatabase(), WithDatabaseManager(), or WithDatabaseFromForge()")
}
