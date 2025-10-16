package authsome

import (
    "context"
    "errors"
    "fmt"
    "net/http"

    "github.com/uptrace/bun"
    "github.com/xraph/authsome/core/auth"
    aud "github.com/xraph/authsome/core/audit"
    rbac "github.com/xraph/authsome/core/rbac"
    rl "github.com/xraph/authsome/core/ratelimit"
    dev "github.com/xraph/authsome/core/device"
    sec "github.com/xraph/authsome/core/security"
    "github.com/xraph/authsome/core/organization"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/registry"
	jwtplugin "github.com/xraph/authsome/plugins/jwt"
    "github.com/xraph/authsome/handlers"
    "github.com/xraph/authsome/internal/validator"
    "github.com/xraph/authsome/plugins"
    repo "github.com/xraph/authsome/repository"
    memstore "github.com/xraph/authsome/storage"
    "github.com/xraph/authsome/routes"
    "github.com/xraph/forge"
)

// Auth is the main authentication instance
type Auth struct {
    config      Config
    forgeConfig interface{}
    db          interface{} // Will be *bun.DB

    // Core services
    userService    *user.Service
    sessionService *session.Service
    authService    *auth.Service
    organizationService *organization.Service
    rateLimitService *rl.Service
    rateLimitStorage rl.Storage
    rateLimitConfig  rl.Config
    deviceService    *dev.Service
    securityService  *sec.Service
    securityConfig   sec.Config
    geoipProvider    sec.GeoIPProvider
    auditService     *aud.Service
    rbacService      *rbac.Service
    userRoleRepo     *repo.UserRoleRepository
    roleRepo         *repo.RoleRepository
    policyRepo       *repo.PolicyRepository
    twofaRepo        *repo.TwoFARepository

    // Phase 10 services
    webhookService      *webhook.Service
    notificationService *notification.Service
    jwtService          *jwt.Service
    apikeyService       *apikey.Service

	// Plugin registry
	pluginRegistry *plugins.Registry
	
	// Service registry
	serviceRegistry *registry.ServiceRegistry
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
	}

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Initialize initializes all core services
func (a *Auth) Initialize(ctx context.Context) error {
	if a.forgeConfig == nil {
		return fmt.Errorf("forge config manager not set")
	}

	if a.db == nil {
		return fmt.Errorf("database not set")
	}

	// Cast database
	db, ok := a.db.(*bun.DB)
	if !ok || db == nil {
		return errors.New("invalid or missing bun.DB instance")
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
        for _, ex := range defaults { _ = a.policyRepo.Create(ctx, ex) }
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

    // Initialize plugins with database and run migrations
    if a.pluginRegistry != nil {
        for _, p := range a.pluginRegistry.List() {
            // Pass the Auth instance to plugin Init so it can access all dependencies
            if err := p.Init(a); err != nil {
                return fmt.Errorf("plugin %s init failed: %w", p.ID(), err)
            }
            if err := p.Migrate(); err != nil {
                return fmt.Errorf("plugin %s migrate failed: %w", p.ID(), err)
            }
        }
    }

    return nil
}

// Mount mounts the auth routes to the Forge app
func (a *Auth) Mount(app interface{}, basePath string) error {
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
    apikeyH := handlers.NewAPIKeyHandler(a.apikeyService)
    
    switch v := app.(type) {
    case *forge.App:
        routes.Register(v, basePath, h)
        routes.RegisterAudit(v, basePath, audH)
        // Mount organization routes under a fixed base for Phase 3
        routes.RegisterOrganization(v, "/api/orgs", orgH)
        
        // Phase 10 routes
        routes.RegisterWebhookRoutes(v.Group(basePath), webhookH)
        routes.RegisterNotificationRoutes(v.Group(basePath), notificationH)
        routes.RegisterJWTRoutes(v.Group(basePath), jwtH)
        routes.RegisterAPIKeyRoutes(v.Group(basePath), apikeyH)
        
        // Register plugin routes with basePath group
        if a.pluginRegistry != nil {
            // Create a group with the basePath so plugins can use relative paths
            pluginGroup := v.Group(basePath)
            for _, p := range a.pluginRegistry.List() {
                _ = p.RegisterRoutes(pluginGroup)
            }
        }
        return nil
    case *http.ServeMux:
        // Wrap ServeMux with local forge shim for identical behavior
        f := forge.NewApp(v)
        routes.Register(f, basePath, h)
        routes.RegisterAudit(f, basePath, audH)
        routes.RegisterOrganization(f, "/api/orgs", orgH)
        
        // Phase 10 routes
        routes.RegisterWebhookRoutes(f.Group(basePath), webhookH)
        routes.RegisterNotificationRoutes(f.Group(basePath), notificationH)
        routes.RegisterJWTRoutes(f.Group(basePath), jwtH)
        routes.RegisterAPIKeyRoutes(f.Group(basePath), apikeyH)
        
        if a.pluginRegistry != nil {
            // Create a group with the basePath so plugins can use relative paths
            pluginGroup := f.Group(basePath)
            for _, p := range a.pluginRegistry.List() {
                _ = p.RegisterRoutes(pluginGroup)
            }
        }
        return nil
    default:
        return errors.New("unsupported app type for Mount; expected *forge.App or *http.ServeMux")
    }
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

// GetConfigManager returns the forge config manager
func (a *Auth) GetConfigManager() interface{} {
	return a.forgeConfig
}

func (a *Auth) GetServiceRegistry() *registry.ServiceRegistry {
	return a.serviceRegistry
}
