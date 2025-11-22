package apikey

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Plugin implements API key authentication for external clients
type Plugin struct {
	service            *apikey.Service
	userSvc            *user.Service
	handler            *Handler
	middleware         *Middleware
	config             Config
	defaultConfig      Config
	cleanupTicker      *time.Ticker
	cleanupDone        chan bool
	dashboardExtension *DashboardExtension
}

// PluginOption is a functional option for configuring the API key plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithDefaultRateLimit sets the default rate limit
func WithDefaultRateLimit(limit int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultRateLimit = limit
	}
}

// WithMaxRateLimit sets the maximum rate limit
func WithMaxRateLimit(limit int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxRateLimit = limit
	}
}

// WithDefaultExpiry sets the default key expiry
func WithDefaultExpiry(expiry time.Duration) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultExpiry = expiry
	}
}

// WithMaxKeysPerUser sets the maximum keys per user
func WithMaxKeysPerUser(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxKeysPerUser = max
	}
}

// WithMaxKeysPerOrg sets the maximum keys per organization
func WithMaxKeysPerOrg(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxKeysPerOrg = max
	}
}

// WithKeyLength sets the API key length
func WithKeyLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.KeyLength = length
	}
}

// WithAllowQueryParam sets whether to allow API keys in query params
func WithAllowQueryParam(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowQueryParam = allow
	}
}

// WithAutoCleanup sets the auto cleanup configuration
func WithAutoCleanup(enabled bool, interval time.Duration) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoCleanup.Enabled = enabled
		p.defaultConfig.AutoCleanup.Interval = interval
	}
}

// NewPlugin creates a new API key plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "apikey"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return fmt.Errorf("auth instance cannot be nil")
	}

	db := authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// Load configuration from Forge config manager with provided defaults
	if err := configManager.BindWithDefault("auth.apikey", &p.config, p.defaultConfig); err != nil {
		// Use defaults if binding fails
		fmt.Printf("[APIKey] Warning: failed to bind config: %v\n", err)
		p.config = p.defaultConfig
	}
	p.config.Validate() // Ensure defaults are set

	// Get services from registry
	auditSvc := serviceRegistry.AuditService()
	rateLimitSvc := serviceRegistry.RateLimitService()

	userSvcInterface := serviceRegistry.UserService()
	var userSvc *user.Service
	if userSvcInterface != nil {
		userSvc, _ = userSvcInterface.(*user.Service)
	}
	p.userSvc = userSvc

	// Initialize repository
	apikeyRepo := repository.NewAPIKeyRepository(db)

	// Initialize service with rate limiting support
	serviceCfg := apikey.Config{
		DefaultRateLimit: p.config.DefaultRateLimit,
		MaxRateLimit:     p.config.MaxRateLimit,
		DefaultExpiry:    p.config.DefaultExpiry,
		MaxKeysPerUser:   p.config.MaxKeysPerUser,
		MaxKeysPerOrg:    p.config.MaxKeysPerOrg,
		KeyLength:        p.config.KeyLength,
	}
	p.service = apikey.NewService(apikeyRepo, auditSvc, serviceCfg)

	// Initialize middleware with rate limiting
	p.middleware = NewMiddleware(p.service, userSvc, rateLimitSvc, p.config)

	// Initialize handler
	p.handler = NewHandler(p.service, p.config)

	// Start cleanup scheduler if enabled
	if p.config.AutoCleanup.Enabled {
		p.startCleanupScheduler()
	}

	// Initialize dashboard extension
	p.dashboardExtension = NewDashboardExtension(p)

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return nil
	}

	// API key management routes (protected by session auth)
	apikeys := router.Group("/api-keys")
	{
		apikeys.POST("", p.handler.CreateAPIKey)
		apikeys.GET("", p.handler.ListAPIKeys)
		apikeys.GET("/:id", p.handler.GetAPIKey)
		apikeys.PUT("/:id", p.handler.UpdateAPIKey)
		apikeys.DELETE("/:id", p.handler.DeleteAPIKey)
		apikeys.POST("/:id/rotate", p.handler.RotateAPIKey)

		// Public verification endpoint for testing
		apikeys.POST("/verify", p.handler.VerifyAPIKey)
	}

	return nil
}

// Middleware returns the authentication middleware
func (p *Plugin) Middleware() func(next func(forge.Context) error) func(forge.Context) error {
	if p.middleware == nil {
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}
	return p.middleware.Authenticate
}

// RequireAPIKey returns middleware that requires a valid API key
func (p *Plugin) RequireAPIKey(scopes ...string) func(next func(forge.Context) error) func(forge.Context) error {
	if p.middleware == nil {
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}
	return p.middleware.RequireAPIKey(scopes...)
}

// RequirePermission returns middleware that requires specific permissions
func (p *Plugin) RequirePermission(permissions ...string) func(next func(forge.Context) error) func(forge.Context) error {
	if p.middleware == nil {
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}
	return p.middleware.RequirePermission(permissions...)
}

// Service returns the API key service for direct access
func (p *Plugin) Service() *apikey.Service {
	return p.service
}

// RegisterHooks registers plugin hooks with the hook registry
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	// API Key plugin can register hooks for:
	// - Before/after key creation
	// - Before/after key verification
	// - Rate limit exceeded events
	// Currently no hooks implemented, but the structure is ready
	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// API Key plugin doesn't decorate core services
	// It provides its own service that's already registered
	return nil
}

// Migrate runs plugin migrations
func (p *Plugin) Migrate() error {
	// Migrations are handled by the main migration system
	// The api_keys table is already created in core migrations
	return nil
}

// startCleanupScheduler starts a background goroutine to cleanup expired API keys
func (p *Plugin) startCleanupScheduler() {
	if p.service == nil {
		return
	}

	p.cleanupTicker = time.NewTicker(p.config.AutoCleanup.Interval)
	p.cleanupDone = make(chan bool)

	go func() {
		log.Printf("[APIKey Plugin] Cleanup scheduler started (interval: %v)", p.config.AutoCleanup.Interval)

		// Run cleanup immediately on start
		p.runCleanup()

		for {
			select {
			case <-p.cleanupTicker.C:
				p.runCleanup()
			case <-p.cleanupDone:
				return
			}
		}
	}()
}

// runCleanup executes the cleanup of expired API keys
func (p *Plugin) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count, err := p.service.CleanupExpired(ctx)
	if err != nil {
		log.Printf("[APIKey Plugin] Cleanup failed: %v", err)
		return
	}

	if count > 0 {
		log.Printf("[APIKey Plugin] Cleaned up %d expired API key(s)", count)
	}
}

// StopCleanupScheduler stops the cleanup scheduler (for graceful shutdown)
func (p *Plugin) StopCleanupScheduler() {
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
		close(p.cleanupDone)
		log.Println("[APIKey Plugin] Cleanup scheduler stopped")
	}
}

// DashboardExtension returns the dashboard extension for this plugin
// This implements the PluginWithDashboardExtension interface
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	if p.dashboardExtension == nil {
		p.dashboardExtension = NewDashboardExtension(p)
	}
	return p.dashboardExtension
}
