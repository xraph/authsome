package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	aud "github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	dev "github.com/xraph/authsome/core/device"
	env "github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/middleware"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/organization"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	sec "github.com/xraph/authsome/core/security"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/handlers"
	"github.com/xraph/authsome/internal/dbschema"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/internal/validator"
	"github.com/xraph/authsome/plugins"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/routes"
	"github.com/xraph/authsome/schema"
	memstore "github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
	"github.com/xraph/forge/extensions/database"
)

// ServiceImpl name constants for DI container
const (
	ServiceDatabase       = "authsome.database"
	ServiceUser           = "authsome.user"
	ServiceSession        = "authsome.session"
	ServiceAuth           = "authsome.auth"
	ServiceApp            = "authsome.app"
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
	logger   forge.Logger

	// Core services (using interfaces to allow plugin decoration)
	userService        user.ServiceInterface
	sessionService     session.ServiceInterface
	authService        auth.ServiceInterface
	appService         *app.ServiceImpl
	orgService         *organization.Service
	environmentService *env.Service
	rateLimitService   *rl.Service
	rateLimitStorage   rl.Storage
	rateLimitConfig    rl.Config
	deviceService      *dev.Service
	securityService    *sec.Service
	securityConfig     sec.Config
	geoipProvider      sec.GeoIPProvider
	auditService       *aud.Service
	rbacService        *rbac.Service
	repo               repo.Repository

	// Phase 10 services
	webhookService      *webhook.Service
	notificationService *notification.Service
	jwtService          *jwt.Service
	apikeyService       *apikey.Service

	// Global authentication middleware
	authMiddleware       *middleware.AuthMiddleware
	authMiddlewareConfig middleware.AuthMiddlewareConfig
	authStrategyRegistry *middleware.AuthStrategyRegistry

	// Plugin registry
	pluginRegistry plugins.PluginRegistry

	// ServiceImpl registry for plugin decorator pattern
	serviceRegistry *registry.ServiceRegistry
	hookRegistry    *hooks.HookRegistry

	// Global routes options
	globalRoutesOptions      []forge.RouteOption
	globalGroupRoutesOptions []forge.GroupOption
}

// New creates a new Auth instance with the given options
func New(opts ...Option) *Auth {
	a := &Auth{
		config: Config{
			BasePath:          "/api/auth",
			SessionCookieName: "authsome_session",
		},
		pluginRegistry:           plugins.NewRegistry(),
		serviceRegistry:          registry.NewServiceRegistry(),
		hookRegistry:             hooks.NewHookRegistry(),
		authStrategyRegistry:     middleware.NewAuthStrategyRegistry(),
		globalRoutesOptions:      []forge.RouteOption{},
		globalGroupRoutesOptions: []forge.GroupOption{},
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Initialize initializes all core services
func (a *Auth) Initialize(ctx context.Context) error {
	a.logger = a.forgeApp.Logger()
	a.logger.Info("initializing authsome")
	if a.forgeApp == nil {
		return fmt.Errorf("forge app not set")
	}

	// Resolve database from various sources
	if err := a.resolveDatabase(); err != nil {
		return errs.InternalServerError("failed to resolve database", err)
	}

	a.logger.Info("resolved database", forge.F("database", a.db))

	// Cast database
	db, ok := a.db.(*bun.DB)
	if !ok || db == nil {
		return errors.New("invalid or missing bun.DB instance")
	}

	// Apply custom database schema if configured
	// This creates the schema and sets the search_path for all subsequent operations
	if a.config.DatabaseSchema != "" {
		if err := dbschema.ApplySchema(ctx, db, a.config.DatabaseSchema); err != nil {
			return errs.InternalServerError("failed to apply database schema", err)
		}
		a.logger.Info("applied custom database schema", forge.F("schema", a.config.DatabaseSchema))
	}

	// Initialize repositories
	a.repo = repo.NewRepo(db)

	// Register m2m models for Bun ORM
	// These models are used as join tables for many-to-many relationships
	// and must be explicitly registered with Bun
	db.RegisterModel((*schema.TeamMember)(nil))
	db.RegisterModel((*schema.OrganizationTeamMember)(nil))
	db.RegisterModel((*schema.RolePermission)(nil))
	db.RegisterModel((*schema.APIKeyRole)(nil))

	// Rate limit storage and service
	if a.rateLimitStorage == nil {
		a.rateLimitStorage = memstore.NewMemoryStorage()
	}

	a.rateLimitService = rl.NewService(a.rateLimitStorage, a.rateLimitConfig)

	// Device service
	a.deviceService = dev.NewService(a.repo.Device())

	// Security service
	// Default enable if not explicitly set
	if !a.securityConfig.Enabled && len(a.securityConfig.IPWhitelist) == 0 && len(a.securityConfig.IPBlacklist) == 0 && len(a.securityConfig.AllowedCountries) == 0 && len(a.securityConfig.BlockedCountries) == 0 {
		a.securityConfig.Enabled = true
	}
	a.securityService = sec.NewService(a.repo.Security(), a.securityConfig)
	if a.geoipProvider != nil {
		a.securityService.SetGeoIPProvider(a.geoipProvider)
	}

	// Audit service
	a.auditService = aud.NewService(a.repo.Audit())

	// RBAC service: load policies from storage and set repositories
	a.rbacService = rbac.NewService()
	a.rbacService.SetRepositories(
		a.repo.Role(),
		a.repo.Permission(),
		a.repo.RolePermission(),
		a.repo.UserRole(),
	)
	_ = a.rbacService.LoadPolicies(ctx, a.repo.Policy())
	// Seed default policies if storage is empty
	if exprs, err := a.repo.Policy().ListAll(ctx); err == nil && len(exprs) == 0 {
		defaults := []string{
			// App permissions (platform tenant)
			"role:owner:create,read,update,delete on app:*",
			"role:admin:read,update on app:*",
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
			_ = a.repo.Policy().Create(ctx, ex)
		}
		_ = a.rbacService.LoadPolicies(ctx, a.repo.Policy())
	}

	// Initialize Phase 10 services first (webhook is a dependency for user/session services)
	a.webhookService = webhook.NewService(webhook.Config{}, a.repo.Webhook(), a.auditService)

	// TODO: Add template engine when available
	a.notificationService = notification.NewService(a.repo.Notification(), nil, a.auditService, notification.Config{})

	// JWT service uses a wrapper around the JWT key repository
	a.jwtService = jwt.NewService(jwt.Config{}, a.repo.JWTKey(), a.auditService)

	a.apikeyService = apikey.NewService(a.repo.APIKey(), a.auditService, apikey.Config{})

	// Initialize global authentication middleware
	// Note: This will be initialized fully after session and user services are created
	// For now, we'll create a placeholder that will be updated later

	// Initialize services (now with webhook service dependency and hook registry)
	// Initialize user config with defaults if not set
	userConfig := a.config.UserConfig
	if userConfig.PasswordRequirements.MinLength == 0 {
		userConfig.PasswordRequirements = validator.DefaultPasswordRequirements()
	}

	a.userService = user.NewService(a.repo.User(), userConfig, a.webhookService, a.hookRegistry)
	a.sessionService = session.NewService(a.repo.Session(), a.config.SessionConfig, a.webhookService, a.hookRegistry)
	a.authService = auth.NewService(a.userService, a.sessionService, auth.Config{
		RequireEmailVerification: a.config.RequireEmailVerification,
	}, a.hookRegistry)

	// Initialize global authentication middleware now that all required services are ready
	// Use provided config or sensible defaults
	middlewareConfig := a.authMiddlewareConfig

	// If no config was provided via options, set secure defaults
	if middlewareConfig.SessionCookieName == "" && len(middlewareConfig.APIKeyHeaders) == 0 {
		// No explicit config provided, use sensible defaults
		middlewareConfig = middleware.AuthMiddlewareConfig{
			SessionCookieName:   a.config.SessionCookieName,
			Optional:            true,  // Don't block unauthenticated requests by default
			AllowAPIKeyInQuery:  false, // Security best practice
			AllowSessionInQuery: false, // Security best practice
		}
		if middlewareConfig.SessionCookieName == "" {
			middlewareConfig.SessionCookieName = "authsome_session"
		}
	} else {
		// Config was partially or fully provided, only fill in missing session cookie name
		if middlewareConfig.SessionCookieName == "" {
			middlewareConfig.SessionCookieName = a.config.SessionCookieName
			if middlewareConfig.SessionCookieName == "" {
				middlewareConfig.SessionCookieName = "authsome_session"
			}
		}
	}

	a.authMiddleware = middleware.NewAuthMiddleware(
		a.apikeyService,
		a.sessionService,
		a.userService,
		middlewareConfig,
		&a.config.SessionCookie, // Pass cookie config for session renewal
		a.authStrategyRegistry,  // Pass strategy registry for pluggable auth
	)

	// App service (platform tenant management)
	a.appService = app.NewService(
		a.repo.App(),
		a.repo.App(),
		a.repo.App(),
		a.repo.App(),
		a.repo.Role(),     // NEW: Role repository for RBAC
		a.repo.UserRole(), // NEW: UserRole repository for RBAC
		app.Config{},
		a.rbacService,
	)

	// Set global cookie config on app service
	a.appService.App.SetGlobalCookieConfig(&a.config.SessionCookie)

	if err := app.RegisterAppPermissions(a.serviceRegistry.RoleRegistry()); err != nil {
		return errs.InternalServerError("failed to register app permissions", err)
	}

	// Set hook registry on app service for app creation hooks
	a.appService.App.SetHookRegistry(a.hookRegistry)

	// Organization service (end-user workspace management)
	a.orgService = organization.NewService(
		a.repo.Organization(),
		a.repo.OrganizationMember(),
		a.repo.OrganizationTeam(),
		a.repo.OrganizationInvitation(),
		organization.Config{},
		a.rbacService,
		a.repo.Role(),
	)

	// Environment service (app environment management)
	// Initialize with default config - plugins can override via decorator pattern
	a.environmentService = env.NewService(
		a.repo.Environment(),
		env.Config{
			AutoCreateDev:                  true,
			DefaultDevName:                 "Development",
			AllowPromotion:                 true,
			RequireConfirmationForDataCopy: true,
			MaxEnvironmentsPerApp:          10,
		},
	)

	// Populate service registry BEFORE plugin initialization
	// This allows plugins to access and decorate services
	a.serviceRegistry.SetAppService(a.appService)
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
	a.serviceRegistry.SetOrganizationService(a.orgService)
	a.serviceRegistry.SetEnvironmentService(a.environmentService)

	// Set hook registry on device service for security notifications
	a.deviceService.SetHookRegistry(a.hookRegistry)

	// Set hook registry on user service for account lifecycle notifications
	a.userService.SetHookRegistry(a.hookRegistry)

	// Set verification repository on user service for password resets
	a.userService.SetVerificationRepo(a.repo.Verification())

	// Register services into Forge DI container
	if err := a.registerServicesIntoContainer(db); err != nil {
		return errs.InternalServerError("failed to register services into DI container", err)
	}

	// Ensure platform organization exists before plugins initialize
	// This is needed for role bootstrap later
	platformOrg, err := a.ensurePlatformApp(ctx)
	if err != nil {
		return errs.InternalServerError("failed to ensure platform app", err)
	}

	// Register default platform roles before plugins initialize
	// Plugins can then extend or override these roles
	if err := rbac.RegisterDefaultPlatformRoles(a.serviceRegistry.RoleRegistry()); err != nil {
		return errs.InternalServerError("failed to register default platform roles", err)
	}

	// Initialize plugins with full Auth instance and run complete lifecycle
	// Plugins are automatically sorted by dependencies using topological sort
	if a.pluginRegistry != nil {
		// Get plugins sorted by dependencies
		sortedPlugins, err := a.pluginRegistry.(*plugins.Registry).ListSorted()
		if err != nil {
			return errs.InternalServerError("plugin dependency validation failed", err)
		}

		a.logger.Info("initializing plugins in dependency order",
			forge.F("count", len(sortedPlugins)))

		for _, p := range sortedPlugins {
			// 1. Initialize plugin with Auth instance (not just DB)
			if err := p.Init(a); err != nil {
				return errs.InternalServerError("plugin init failed", err)
			}

			// 2. Register roles (optional interface)
			// If plugin implements PluginWithRoles, it can register its roles
			if rolePlugin, ok := p.(interface {
				RegisterRoles(registry interface{}) error
			}); ok {
				a.logger.Info("plugin registering roles", forge.F("plugin", p.ID()))
				if err := rolePlugin.RegisterRoles(a.serviceRegistry.RoleRegistry()); err != nil {
					return errs.InternalServerError("plugin register roles failed", err)
				}
			}

			// 3. Register hooks
			if err := p.RegisterHooks(a.hookRegistry); err != nil {
				return errs.InternalServerError("plugin register hooks failed", err)
			}

			// 4. Register service decorators (plugins can replace core services)
			if err := p.RegisterServiceDecorators(a.serviceRegistry); err != nil {
				return errs.InternalServerError("plugin register decorators failed", err)
			}

			// 5. Run migrations
			if err := p.Migrate(); err != nil {
				return errs.InternalServerError("plugin migrate failed", err)
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
		return errs.InternalServerError("failed to bootstrap roles", err)
	}

	return nil
}

// ensurePlatformApp ensures the platform organization exists
// This is the single foundational organization for the entire system
// Returns the platform organization (existing or newly created)
func (a *Auth) ensurePlatformApp(ctx context.Context) (*schema.App, error) {
	db, ok := a.db.(*bun.DB)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("invalid database instance")
	}

	// Check if platform org exists
	var platformApp schema.App
	err := db.NewSelect().
		Model(&platformApp).
		Where("is_platform = ?", true).
		Scan(ctx)

	if err == nil {
		// Platform org exists
		a.logger.Info("platform app found", forge.F("app", platformApp.Name), forge.F("id", platformApp.ID.String()))
		return &platformApp, nil
	}

	// Platform org doesn't exist - create it
	a.logger.Info("platform app not found, creating...")

	platformApp = schema.App{
		ID:       xid.New(),
		Name:     "Platform App",
		Slug:     "platform",
		Metadata: map[string]interface{}{},
	}
	platformApp.CreatedAt = time.Now()
	platformApp.UpdatedAt = time.Now()
	platformApp.CreatedBy = platformApp.ID // Self-created
	platformApp.UpdatedBy = platformApp.ID
	platformApp.Version = 1

	_, err = db.NewInsert().Model(&platformApp).Exec(ctx)
	if err != nil {
		return nil, errs.InternalServerError("failed to create platform app", err)
	}

	a.logger.Info("created platform app", forge.F("app", platformApp.Name), forge.F("id", platformApp.ID.String()))

	return &platformApp, nil
}

// bootstrapRoles applies all registered roles to the platform organization
// This is called after plugins have initialized and registered their roles
func (a *Auth) bootstrapRoles(ctx context.Context, db *bun.DB, platformOrgID xid.ID) error {
	roleRegistry := a.serviceRegistry.RoleRegistry()
	if roleRegistry == nil {
		return errs.InternalServerErrorWithMessage("role registry not initialized")
	}

	a.logger.Info("starting role bootstrap...")
	if err := roleRegistry.Bootstrap(ctx, db, a.rbacService, platformOrgID); err != nil {
		return errs.InternalServerError("role bootstrap failed", err)
	}

	a.logger.Info("role bootstrap complete")
	return nil
}

// Mount mounts the auth routes to the Forge router
func (a *Auth) Mount(router forge.Router, basePath string) error {
	if a.authService == nil {
		return errs.InternalServerErrorWithMessage("auth service not initialized; call Initialize first")
	}
	if basePath == "" {
		basePath = a.config.BasePath
	}

	// Apply CORS middleware if enabled
	if a.config.CORSEnabled && len(a.config.TrustedOrigins) > 0 {
		corsMiddleware := middleware.CORSMiddleware(middleware.CORSConfig{
			AllowedOrigins:   a.config.TrustedOrigins,
			AllowCredentials: true, // Required for cookies
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{
				"Accept",
				"Authorization",
				"Content-Type",
				"X-API-Key",
				"X-App-ID",
				"X-Environment",
				"X-Organization-ID",
			},
		})

		// Create a router group with CORS middleware
		corsGroup := router.Group(basePath)
		corsGroup.Use(corsMiddleware)
		router = corsGroup
		basePath = "" // Reset basePath since we're now in a group

		a.logger.Info("CORS enabled", forge.F("origins", a.config.TrustedOrigins))
	}

	// Backward compatibility: If SessionCookie.Name is empty, use SessionCookieName
	if a.config.SessionCookie.Name == "" && a.config.SessionCookieName != "" {
		a.config.SessionCookie.Name = a.config.SessionCookieName
	}

	h := handlers.NewAuthHandler(a.authService, a.rateLimitService, a.deviceService, a.securityService, a.auditService, a.repo.TwoFA(), a.appService, &a.config.SessionCookie, a.serviceRegistry, basePath)
	audH := handlers.NewAuditHandler(a.auditService)
	appH := handlers.NewAppHandler(a.appService, a.rateLimitService, a.sessionService, a.rbacService, a.repo.UserRole(), a.repo.Role(), a.repo.Policy(), a.config.RBACEnforce)

	webhookH := handlers.NewWebhookHandler(a.webhookService)

	excludeableGroupp := router.Group("", a.GetGlobalGroupRoutesOptions()...)

	// Register core auth routes with authentication middleware
	// Middleware extracts and validates API keys for app identification
	routes.Register(excludeableGroupp, basePath, h, a.AuthMiddleware())
	routes.RegisterAudit(excludeableGroupp, basePath, audH, a.AuthMiddleware())

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

	// Only register built-in app routes if multitenancy plugin is NOT enabled
	// This prevents route duplication and allows the plugin to fully control app routes
	if !hasMultitenancyPlugin {
		// Mount app routes under basePath (not hardcoded) with authentication middleware
		routes.RegisterApp(excludeableGroupp, basePath+"/apps", appH, a.AuthMiddleware())
		a.logger.Info("registered built-in app routes (multitenancy plugin not detected)")
	} else {
		a.logger.Info("skipping built-in app routes (multitenancy plugin detected)")

		// Register RBAC-related routes that the multitenancy plugin doesn't handle
		// These are still needed even with the multitenancy plugin
		rbacGroup := excludeableGroupp.Group(basePath + "/apps")
		routes.RegisterAppRBAC(rbacGroup, appH)
		a.logger.Info("registered app RBAC routes")
	}

	authGroup := excludeableGroupp.Group(basePath)
	routes.RegisterWebhookRoutes(authGroup, webhookH)

	// Register plugin routes (scoped to basePath)
	if a.pluginRegistry != nil {
		// Pass a group with the basePath so plugins are scoped under the auth mount point
		pluginGroup := excludeableGroupp.Group(basePath)
		for _, p := range a.pluginRegistry.List() {
			a.logger.Info("registering routes for plugin", forge.F("plugin", p.ID()))
			if err := p.RegisterRoutes(pluginGroup); err != nil {
				a.logger.Error("error registering routes for plugin", forge.F("plugin", p.ID()), forge.F("error", err))
			}
		}
	}

	return nil
}

// RegisterPlugin registers a plugin
func (a *Auth) RegisterPlugin(plugin plugins.Plugin) error {
	return a.pluginRegistry.Register(plugin)
}

// RegisterAuthStrategy registers an authentication strategy
// This allows plugins to add custom authentication methods
// Strategies are tried in priority order during authentication
func (a *Auth) RegisterAuthStrategy(strategy middleware.AuthStrategy) error {
	if a.authStrategyRegistry == nil {
		a.authStrategyRegistry = middleware.NewAuthStrategyRegistry()
	}

	a.logger.Info("registering authentication strategy",
		forge.F("strategy_id", strategy.ID()),
		forge.F("priority", strategy.Priority()))

	return a.authStrategyRegistry.Register(strategy)
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

// Repository implements core.Authsome.
func (a *Auth) Repository() repo.Repository {
	return a.repo
}

// ServiceRegistry returns the service registry for plugins
func (a *Auth) ServiceRegistry() *registry.ServiceRegistry {
	return a.serviceRegistry
}

// Hooks returns the hook registry for plugins
func (a *Auth) Hooks() *hooks.HookRegistry {
	return a.hookRegistry
}

// GetHookRegistry returns the hook registry for plugins
func (a *Auth) GetHookRegistry() *hooks.HookRegistry {
	return a.hookRegistry
}

// GetBasePath returns the base path for AuthSome routes
func (a *Auth) GetBasePath() string {
	return a.config.BasePath
}

// Logger returns the logger for AuthSome
func (a *Auth) Logger() forge.Logger {
	return a.logger
}

// GetPluginRegistry returns the plugin registry
func (a *Auth) GetPluginRegistry() plugins.PluginRegistry {
	return a.pluginRegistry
}

// GetGlobalRoutesOptions returns the global routes options
func (a *Auth) GetGlobalRoutesOptions() []forge.RouteOption {
	return a.globalRoutesOptions
}

// GetGlobalGroupRoutesOptions returns the global group routes options
func (a *Auth) GetGlobalGroupRoutesOptions() []forge.GroupOption {
	return a.globalGroupRoutesOptions
}

// GetDefaultApp returns the default app when in standalone mode
// This is useful for middleware context auto-detection
// Returns nil if not in standalone mode or app not found
func (a *Auth) GetDefaultApp(ctx context.Context) (*app.App, error) {
	if a.appService == nil {
		return nil, fmt.Errorf("app service not initialized")
	}

	// Query for apps - in standalone mode there should be one default app
	filter := &app.ListAppsFilter{}
	result, err := a.appService.ListApps(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	if result == nil || len(result.Data) == 0 {
		return nil, fmt.Errorf("no default app found")
	}

	// Return the first app (in standalone mode there's typically only one)
	return result.Data[0], nil
}

// GetDefaultEnvironment returns the default environment for an app
// This is useful for middleware context auto-detection
// Returns nil if environment not found
func (a *Auth) GetDefaultEnvironment(ctx context.Context, appID xid.ID) (*env.Environment, error) {
	if a.environmentService == nil {
		return nil, fmt.Errorf("environment service not initialized")
	}

	// Query for environments for the given app
	filter := &env.ListEnvironmentsFilter{
		AppID: appID,
	}
	result, err := a.environmentService.ListEnvironments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	if result == nil || len(result.Data) == 0 {
		return nil, fmt.Errorf("no default environment found for app %s", appID.String())
	}

	// Look for an environment named "production" or "default" first
	for _, e := range result.Data {
		if e.Name == "production" || e.Name == "default" {
			return e, nil
		}
	}

	// Otherwise return the first environment
	return result.Data[0], nil
}

// IsPluginEnabled checks if a plugin is registered and enabled
func (a *Auth) IsPluginEnabled(pluginID string) bool {
	if a.pluginRegistry == nil {
		return false
	}
	_, exists := a.pluginRegistry.Get(pluginID)
	return exists
}

// =============================================================================
// AUTHENTICATION MIDDLEWARE
// =============================================================================

// AuthMiddleware returns the optional authentication middleware
// This middleware populates the auth context with API key and/or session data
// but does not block unauthenticated requests
func (a *Auth) AuthMiddleware() forge.Middleware {
	return a.authMiddleware.Authenticate
}

// RequireAuth returns middleware that requires authentication
// Blocks requests that are not authenticated via API key or session
func (a *Auth) RequireAuth() forge.Middleware {
	return a.authMiddleware.RequireAuth
}

// Authenticate returns the authentication middleware
func (a *Auth) Authenticate() forge.Middleware {
	return a.authMiddleware.Authenticate
}

// RequireUser returns middleware that requires user authentication (session)
// Blocks requests that don't have a valid user session
func (a *Auth) RequireUser() forge.Middleware {
	return a.authMiddleware.RequireUser
}

// RequireAPIKey returns middleware that requires API key authentication
// Blocks requests that don't have a valid API key
func (a *Auth) RequireAPIKey() forge.Middleware {
	return a.authMiddleware.RequireAPIKey
}

// RequireScope returns middleware that requires a specific API key scope
// Blocks requests where the API key lacks the specified scope
func (a *Auth) RequireScope(scope string) forge.Middleware {
	return a.authMiddleware.RequireScope(scope)
}

// RequireAnyScope returns middleware that requires any of the specified scopes
func (a *Auth) RequireAnyScope(scopes ...string) forge.Middleware {
	return a.authMiddleware.RequireAnyScope(scopes...)
}

// RequireAllScopes returns middleware that requires all of the specified scopes
func (a *Auth) RequireAllScopes(scopes ...string) forge.Middleware {
	return a.authMiddleware.RequireAllScopes(scopes...)
}

// RequireSecretKey returns middleware that requires a secret (sk_) API key
func (a *Auth) RequireSecretKey() forge.Middleware {
	return a.authMiddleware.RequireSecretKey
}

// RequirePublishableKey returns middleware that requires a publishable (pk_) API key
func (a *Auth) RequirePublishableKey() forge.Middleware {
	return a.authMiddleware.RequirePublishableKey
}

// RequireAdmin returns middleware that requires admin privileges
// Blocks requests that don't have admin:full scope via secret API key
func (a *Auth) RequireAdmin() forge.Middleware {
	return a.authMiddleware.RequireAdmin
}

// =============================================================================
// RBAC-AWARE MIDDLEWARE (Hybrid Approach)
// =============================================================================

// RequireRBACPermission returns middleware that requires a specific RBAC permission
// Checks only RBAC permissions (not legacy scopes)
func (a *Auth) RequireRBACPermission(action, resource string) forge.Middleware {
	return a.authMiddleware.RequireRBACPermission(action, resource)
}

// RequireCanAccess returns middleware that checks if auth context can access a resource
// This is flexible - accepts EITHER legacy scopes OR RBAC permissions
// Recommended for backward compatibility
func (a *Auth) RequireCanAccess(action, resource string) forge.Middleware {
	return a.authMiddleware.RequireCanAccess(action, resource)
}

// RequireAnyPermission returns middleware that requires any of the specified permissions
func (a *Auth) RequireAnyPermission(permissions ...string) forge.Middleware {
	return a.authMiddleware.RequireAnyPermission(permissions...)
}

// RequireAllPermissions returns middleware that requires all of the specified permissions
func (a *Auth) RequireAllPermissions(permissions ...string) forge.Middleware {
	return a.authMiddleware.RequireAllPermissions(permissions...)
}

// =============================================================================
// SERVICE REGISTRATION
// =============================================================================

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
		return errs.InternalServerError("failed to register database", err)
	}

	// Register core services as singletons
	if err := container.Register(ServiceUser, func(c forge.Container) (interface{}, error) {
		return a.userService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register user service", err)
	}

	if err := container.Register(ServiceSession, func(c forge.Container) (interface{}, error) {
		return a.sessionService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register session service", err)
	}

	if err := container.Register(ServiceAuth, func(c forge.Container) (interface{}, error) {
		return a.authService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register auth service", err)
	}

	if err := container.Register(ServiceApp, func(c forge.Container) (interface{}, error) {
		return a.appService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register app service", err)
	}

	if err := container.Register(ServiceOrganization, func(c forge.Container) (interface{}, error) {
		return a.orgService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register organization service", err)
	}

	if err := container.Register(ServiceRateLimit, func(c forge.Container) (interface{}, error) {
		return a.rateLimitService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register rate limit service", err)
	}

	if err := container.Register(ServiceDevice, func(c forge.Container) (interface{}, error) {
		return a.deviceService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register device service", err)
	}

	if err := container.Register(ServiceSecurity, func(c forge.Container) (interface{}, error) {
		return a.securityService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register security service", err)
	}

	if err := container.Register(ServiceAudit, func(c forge.Container) (interface{}, error) {
		return a.auditService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register audit service", err)
	}

	if err := container.Register(ServiceRBAC, func(c forge.Container) (interface{}, error) {
		return a.rbacService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register RBAC service", err)
	}

	if err := container.Register(ServiceWebhook, func(c forge.Container) (interface{}, error) {
		return a.webhookService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register webhook service", err)
	}

	if err := container.Register(ServiceNotification, func(c forge.Container) (interface{}, error) {
		return a.notificationService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register notification service", err)
	}

	if err := container.Register(ServiceJWT, func(c forge.Container) (interface{}, error) {
		return a.jwtService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register JWT service", err)
	}

	if err := container.Register(ServiceAPIKey, func(c forge.Container) (interface{}, error) {
		return a.apikeyService, nil
	}); err != nil {
		return errs.InternalServerError("failed to register API key service", err)
	}

	// Register registries
	if err := container.Register(ServiceHookRegistry, func(c forge.Container) (interface{}, error) {
		return a.hookRegistry, nil
	}); err != nil {
		return errs.InternalServerError("failed to register hook registry", err)
	}

	if err := container.Register(ServicePluginRegistry, func(c forge.Container) (interface{}, error) {
		return a.pluginRegistry, nil
	}); err != nil {
		return errs.InternalServerError("failed to register plugin registry", err)
	}

	a.logger.Info("successfully registered all services into Forge DI container")
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
		return nil
	}

	return fmt.Errorf("database not configured: use WithDatabase(), WithDatabaseManager(), or WithDatabaseFromForge()")
}
