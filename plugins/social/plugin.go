package social

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/authflow"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/social/providers"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the social OAuth plugin.
type Plugin struct {
	db               *bun.DB
	service          *Service
	handler          *Handler
	config           Config
	defaultConfig    Config
	authInst         core.Authsome
	configRepo       repository.SocialProviderConfigRepository
	dashboardExt     *DashboardExtension
	dashboardExtOnce sync.Once
	logger           forge.Logger
}

// PluginOption is a functional option for configuring the social plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithProvider adds a provider configuration.
func WithProvider(name string, clientID, clientSecret, callbackURL string, scopes []string) PluginOption {
	return func(p *Plugin) {
		// ProvidersConfig is a struct, not a map - no nil check needed
		// Just set the provider directly based on name
		switch name {
		case "google":
			p.defaultConfig.Providers.Google = &providers.ProviderConfig{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  callbackURL,
				Scopes:       scopes,
				Enabled:      true,
			}
		case "github":
			p.defaultConfig.Providers.GitHub = &providers.ProviderConfig{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				RedirectURL:  callbackURL,
				Scopes:       scopes,
				Enabled:      true,
			}
			// Add more providers as needed
		}
	}
}

// WithAutoCreateUser sets whether to auto-create users.
func WithAutoCreateUser(auto bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoCreateUser = auto
	}
}

// WithAllowLinking sets whether to allow account linking.
func WithAllowLinking(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowAccountLinking = allow
	}
}

// WithTrustEmailVerified sets whether to trust provider email verification.
func WithTrustEmailVerified(trust bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TrustEmailVerified = trust
	}
}

// NewPlugin creates a new social OAuth plugin with optional configuration.
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	// Set config to default config
	p.config = p.defaultConfig

	return p
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return "social"
}

// Init initializes the plugin with dependencies.
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.BadRequest("social plugin requires auth instance with GetDB and GetForgeApp methods")
	}

	db := authInst.GetDB()
	if db == nil {
		return errs.InternalServerErrorWithMessage("database not available for social plugin")
	}

	p.db = db
	p.authInst = authInst

	// Get Forge app and config manager
	forgeApp := authInst.GetForgeApp()
	if forgeApp != nil {
		configManager := forgeApp.Config()

		// Bind configuration using Forge ConfigManager with provided defaults
		if err := configManager.BindWithDefault("auth.social", &p.config, p.defaultConfig); err != nil {
			// Log but don't fail - use defaults
			p.config = p.defaultConfig
		}
	} else {
		// Fallback to default config if no Forge app
		p.config = p.defaultConfig
	}

	// Create repositories
	socialRepo := authInst.Repository().SocialAccount()
	userRepo := authInst.Repository().User()

	// Create user service (simplified - in production, get from registry)
	userSvc := user.NewService(userRepo, user.Config{}, nil, nil)

	// Create audit service
	auditSvc := authInst.GetServiceRegistry().AuditService()

	// stateStore state store
	var stateStore StateStore

	if p.config.StateStorage.UseRedis {
		// Create Redis client
		redisClient := redis.NewClient(&redis.Options{
			Addr:     p.config.StateStorage.RedisAddr,
			Password: p.config.StateStorage.RedisPassword,
			DB:       p.config.StateStorage.RedisDB,
		})

		// Test Redis connection
		ctx := context.Background()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			stateStore = NewMemoryStateStore()
		} else {
			stateStore = NewRedisStateStore(redisClient)
		}
	} else {
		stateStore = NewMemoryStateStore()
	}

	// Create social service
	p.service = NewService(p.config, socialRepo, userSvc, stateStore, auditSvc)

	// Create config repository for DB-backed provider configs
	p.configRepo = repository.NewSocialProviderConfigRepository(db)

	// Set config repository on service for environment-scoped loading
	p.service.SetConfigRepository(p.configRepo)

	// rateLimiter rate limiter (only if Redis is available)
	var rateLimiter *RateLimiter

	if p.config.StateStorage.UseRedis {
		// Create Redis client for rate limiting
		redisClient := redis.NewClient(&redis.Options{
			Addr:     p.config.StateStorage.RedisAddr,
			Password: p.config.StateStorage.RedisPassword,
			DB:       p.config.StateStorage.RedisDB,
		})
		rateLimiter = NewRateLimiter(redisClient)
	}

	// authCompletion centralized authentication completion service
	var authCompletion *authflow.CompletionService

	serviceRegistry := authInst.GetServiceRegistry()
	if serviceRegistry != nil {
		authService := serviceRegistry.AuthService()
		deviceService := serviceRegistry.DeviceService()
		auditService := serviceRegistry.AuditService()
		appServiceImpl := serviceRegistry.AppService()

		// Create completion service - use App sub-service for cookie config
		// The appService.GetCookieConfig() method already handles getting the global
		// appService cookie config (set via SetGlobalCookieConfig in authsome.go) as a fallback
		var appService authflow.AppServiceInterface
		if appServiceImpl != nil {
			appService = appServiceImpl.App // Access the AppService from ServiceImpl
		}

		// authCompletion nil for cookieConfig - appService.GetCookieConfig() handles global config
		authCompletion = authflow.NewCompletionService(
			authService,
			deviceService,
			auditService,
			appService,
			nil, // Cookie config comes from appService.GetCookieConfig()
		)
	}

	// Create handler with centralized completion service
	p.handler = NewHandler(p.service, rateLimiter, authCompletion)

	// Dashboard extension is lazy-initialized when first accessed via DashboardExtension()

	return nil
}

// SetConfig allows setting configuration after plugin creation.
func (p *Plugin) SetConfig(config Config) {
	p.config = config
	if p.service != nil && p.authInst != nil {
		// Reinitialize service with new config
		userSvc := user.NewService(p.authInst.Repository().User(), user.Config{}, nil, p.authInst.GetHookRegistry())
		auditSvc := p.authInst.GetServiceRegistry().AuditService()

		// stateStore state store
		var stateStore StateStore

		if config.StateStorage.UseRedis {
			redisClient := redis.NewClient(&redis.Options{
				Addr:     config.StateStorage.RedisAddr,
				Password: config.StateStorage.RedisPassword,
				DB:       config.StateStorage.RedisDB,
			})

			ctx := context.Background()
			if err := redisClient.Ping(ctx).Err(); err == nil {
				stateStore = NewRedisStateStore(redisClient)
			} else {
				stateStore = NewMemoryStateStore()
			}
		} else {
			stateStore = NewMemoryStateStore()
		}

		p.service = NewService(config, p.authInst.Repository().SocialAccount(), userSvc, stateStore, auditSvc)
	}
}

// RegisterRoutes registers the plugin's HTTP routes.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil || p.handler == nil {
		return errs.InternalServerErrorWithMessage("social plugin not initialized")
	}

	// Use the handler created during Init()
	handler := p.handler

	// Get authentication middleware for API key validation
	authMw := p.authInst.AuthMiddleware()

	// Wrap handler with middleware if available
	wrapHandler := func(h func(forge.Context) error) func(forge.Context) error {
		if authMw != nil {
			return authMw(h)
		}

		return h
	}

	// Router is already scoped to the correct basePath
	if err := router.POST("/signin/social", wrapHandler(handler.SignIn),
		forge.WithName("social.signin"),
		forge.WithSummary("Sign in with social provider"),
		forge.WithDescription("Initiate OAuth sign-in flow with a social provider (Google, GitHub, Facebook, etc.)"),
		forge.WithRequestSchema(SignInRequest{}),
		forge.WithResponseSchema(200, "OAuth redirect URL", AuthURLResponse{}),
		forge.WithResponseSchema(400, "Invalid provider", ErrorResponse{}),
		forge.WithTags("Social", "Authentication"),
		forge.WithValidation(true),
	
	); err != nil {
		return err
	}
	if err := router.GET("/callback/:provider", wrapHandler(handler.Callback),
		forge.WithName("social.callback"),
		forge.WithSummary("OAuth callback"),
		forge.WithDescription("Handle OAuth provider callback and complete authentication. This endpoint receives the authorization code and state from the OAuth provider after user grants permission."),
		forge.WithRequestSchema(CallbackRequest{}),
		forge.WithResponseSchema(200, "Authentication successful", CallbackDataResponse{}),
		forge.WithResponseSchema(400, "OAuth error or missing parameters", ErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid state or callback failed", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("Social", "Authentication"),
		forge.WithValidation(true),
	
	); err != nil {
		return err
	}
	if err := router.POST("/account/link", wrapHandler(handler.LinkAccount),
		forge.WithName("social.link"),
		forge.WithSummary("Link social account"),
		forge.WithDescription("Link a social provider account to existing user account"),
		forge.WithRequestSchema(LinkAccountRequest{}),
		forge.WithResponseSchema(200, "Link account URL", AuthURLResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("Social", "Account Management"),
		forge.WithValidation(true),
	
	); err != nil {
		return err
	}
	if err := router.DELETE("/account/unlink/:provider", wrapHandler(handler.UnlinkAccount),
		forge.WithName("social.unlink"),
		forge.WithSummary("Unlink social account"),
		forge.WithDescription("Unlink a social provider account from user account"),
		forge.WithResponseSchema(200, "Account unlinked", MessageResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not linked", ErrorResponse{}),
		forge.WithTags("Social", "Account Management"),
	
	); err != nil {
		return err
	}
	if err := router.GET("/providers", wrapHandler(handler.ListProviders),
		forge.WithName("social.providers.list"),
		forge.WithSummary("List available providers"),
		forge.WithDescription("List all configured social authentication providers"),
		forge.WithResponseSchema(200, "Providers list", ProvidersResponse{}),
		forge.WithTags("Social", "Configuration"),
	
	); err != nil {
		return err
	}

	return nil
}

// RegisterHooks registers plugin hooks.
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error {
	// TODO: Add hooks for OAuth flow events
	// - OnSocialSignIn
	// - OnSocialSignUp
	// - OnAccountLinked
	// - OnAccountUnlinked
	return nil
}

// RegisterServiceDecorators registers service decorators.
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	// No service decorators needed for social plugin
	return nil
}

// Migrate creates database tables.
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create social_accounts table
	_, err := p.db.NewCreateTable().
		Model((*schema.SocialAccount)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create social_accounts table: %w", err)
	}

	// Create social_provider_configs table for dashboard-managed provider configs
	_, err = p.db.NewCreateTable().
		Model((*schema.SocialProviderConfig)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create social_provider_configs table: %w", err)
	}

	// Create indexes for social_accounts
	accountIndexes := []struct {
		name    string
		columns []string
	}{
		{"idx_social_accounts_user_id", []string{"user_id"}},
		{"idx_social_accounts_provider", []string{"provider"}},
		{"idx_social_accounts_provider_id", []string{"provider_id"}},
		{"idx_social_accounts_app_id", []string{"app_id"}},
		{"idx_social_accounts_user_org_id", []string{"user_organization_id"}},
		{"idx_social_accounts_email", []string{"email"}},
		{"idx_social_accounts_provider_provider_id_app", []string{"provider", "provider_id", "app_id"}},
	}

	for _, idx := range accountIndexes {
		_, err := p.db.NewCreateIndex().
			Model((*schema.SocialAccount)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	// Create indexes for social_provider_configs
	configIndexes := []struct {
		name    string
		columns []string
	}{
		{"idx_spc_app_id", []string{"app_id"}},
		{"idx_spc_environment_id", []string{"environment_id"}},
		{"idx_spc_provider_name", []string{"provider_name"}},
		{"idx_spc_app_env", []string{"app_id", "environment_id"}},
		{"idx_spc_app_env_provider", []string{"app_id", "environment_id", "provider_name"}},
	}

	for _, idx := range configIndexes {
		_, err := p.db.NewCreateIndex().
			Model((*schema.SocialProviderConfig)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	return nil
}

// GetService returns the social service (for testing/internal use).
func (p *Plugin) GetService() *Service {
	return p.service
}

// DashboardExtension returns the dashboard extension for the social plugin.
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	p.dashboardExtOnce.Do(func() {
		p.dashboardExt = NewDashboardExtension(p, p.configRepo)
	})

	return p.dashboardExt
}

// GetConfigRepository returns the config repository (for testing/internal use).
func (p *Plugin) GetConfigRepository() repository.SocialProviderConfigRepository {
	return p.configRepo
}

// ErrorResponse alias for error responses.
type ErrorResponse = errs.AuthsomeError
