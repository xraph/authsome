package social

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/social/providers"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the social OAuth plugin
type Plugin struct {
	db            *bun.DB
	service       *Service
	handler       *Handler
	config        Config
	defaultConfig Config
	authInst      core.Authsome
}

// PluginOption is a functional option for configuring the social plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithProvider adds a provider configuration
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

// WithAutoCreateUser sets whether to auto-create users
func WithAutoCreateUser(auto bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoCreateUser = auto
	}
}

// WithAllowLinking sets whether to allow account linking
func WithAllowLinking(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowAccountLinking = allow
	}
}

// WithTrustEmailVerified sets whether to trust provider email verification
func WithTrustEmailVerified(trust bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TrustEmailVerified = trust
	}
}

// NewPlugin creates a new social OAuth plugin with optional configuration
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

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "social"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("social plugin requires auth instance with GetDB and GetForgeApp methods")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for social plugin")
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
			fmt.Printf("[Social] Warning: failed to bind config: %v\n", err)
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
	userSvc := user.NewService(userRepo, user.Config{}, nil)

	// Create audit service
	auditSvc := authInst.GetServiceRegistry().AuditService()

	// Create state store
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
			fmt.Printf("[Social] Warning: failed to connect to Redis, falling back to memory storage: %v\n", err)
			stateStore = NewMemoryStateStore()
		} else {
			stateStore = NewRedisStateStore(redisClient)
			fmt.Printf("[Social] Using Redis for OAuth state storage\n")
		}
	} else {
		stateStore = NewMemoryStateStore()
		fmt.Printf("[Social] Using in-memory OAuth state storage\n")
	}

	// Create social service
	p.service = NewService(p.config, socialRepo, userSvc, stateStore, auditSvc)

	// Create rate limiter (only if Redis is available)
	var rateLimiter *RateLimiter
	if p.config.StateStorage.UseRedis {
		// Create Redis client for rate limiting
		redisClient := redis.NewClient(&redis.Options{
			Addr:     p.config.StateStorage.RedisAddr,
			Password: p.config.StateStorage.RedisPassword,
			DB:       p.config.StateStorage.RedisDB,
		})
		rateLimiter = NewRateLimiter(redisClient)
		fmt.Printf("[Social] Rate limiting enabled with Redis\n")
	}

	// Create handler
	p.handler = NewHandler(p.service, rateLimiter)

	return nil
}

// SetConfig allows setting configuration after plugin creation
func (p *Plugin) SetConfig(config Config) {
	p.config = config
	if p.service != nil && p.authInst != nil {
		// Reinitialize service with new config
		userSvc := user.NewService(p.authInst.Repository().User(), user.Config{}, nil)
		auditSvc := p.authInst.GetServiceRegistry().AuditService()

		// Recreate state store
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

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil || p.handler == nil {
		return fmt.Errorf("social plugin not initialized")
	}

	// Use the handler created during Init()
	handler := p.handler

	// Router is already scoped to the correct basePath
	router.POST("/signin/social", handler.SignIn,
		forge.WithName("social.signin"),
		forge.WithSummary("Sign in with social provider"),
		forge.WithDescription("Initiate OAuth sign-in flow with a social provider (Google, GitHub, Facebook, etc.)"),
		forge.WithRequestSchema(SignInRequest{}),
		forge.WithResponseSchema(200, "OAuth redirect URL", AuthURLResponse{}),
		forge.WithResponseSchema(400, "Invalid provider", ErrorResponse{}),
		forge.WithTags("Social", "Authentication"),
		forge.WithValidation(true),
	)
	router.GET("/callback/:provider", handler.Callback,
		forge.WithName("social.callback"),
		forge.WithSummary("OAuth callback"),
		forge.WithDescription("Handle OAuth provider callback and complete authentication"),
		forge.WithResponseSchema(200, "Authentication successful", CallbackDataResponse{}),
		forge.WithResponseSchema(400, "OAuth error", ErrorResponse{}),
		forge.WithTags("Social", "Authentication"),
	)
	router.POST("/account/link", handler.LinkAccount,
		forge.WithName("social.link"),
		forge.WithSummary("Link social account"),
		forge.WithDescription("Link a social provider account to existing user account"),
		forge.WithRequestSchema(LinkAccountRequest{}),
		forge.WithResponseSchema(200, "Link account URL", AuthURLResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("Social", "Account Management"),
		forge.WithValidation(true),
	)
	router.DELETE("/account/unlink/:provider", handler.UnlinkAccount,
		forge.WithName("social.unlink"),
		forge.WithSummary("Unlink social account"),
		forge.WithDescription("Unlink a social provider account from user account"),
		forge.WithResponseSchema(200, "Account unlinked", MessageResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not linked", ErrorResponse{}),
		forge.WithTags("Social", "Account Management"),
	)
	router.GET("/providers", handler.ListProviders,
		forge.WithName("social.providers.list"),
		forge.WithSummary("List available providers"),
		forge.WithDescription("List all configured social authentication providers"),
		forge.WithResponseSchema(200, "Providers list", ProvidersResponse{}),
		forge.WithTags("Social", "Configuration"),
	)
	return nil
}

// RegisterHooks registers plugin hooks
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error {
	// TODO: Add hooks for OAuth flow events
	// - OnSocialSignIn
	// - OnSocialSignUp
	// - OnAccountLinked
	// - OnAccountUnlinked
	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	// No service decorators needed for social plugin
	return nil
}

// Migrate creates database tables
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

	// Create indexes for performance
	indexes := []struct {
		name    string
		columns []string
	}{
		{"idx_social_accounts_user_id", []string{"user_id"}},
		{"idx_social_accounts_provider", []string{"provider"}},
		{"idx_social_accounts_provider_id", []string{"provider_id"}},
		{"idx_social_accounts_org_id", []string{"organization_id"}},
		{"idx_social_accounts_email", []string{"email"}},
		{"idx_social_accounts_provider_provider_id_org", []string{"provider", "provider_id", "organization_id"}},
	}

	for _, idx := range indexes {
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

	return nil
}

// GetService returns the social service (for testing/internal use)
func (p *Plugin) GetService() *Service {
	return p.service
}

// Type alias for error responses
type ErrorResponse = errs.AuthsomeError
