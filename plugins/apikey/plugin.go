package apikey

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Plugin implements API key authentication for external clients
type Plugin struct {
	service       *apikey.Service
	userSvc       *user.Service
	handler       *Handler
	middleware    *Middleware
	config        Config
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
}

// NewPlugin creates a new API key plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "apikey"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(auth interface{}) error {
	// Type assert to get the auth instance with required methods
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	db := authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// Load configuration from Forge config manager
	var config Config
	if err := configManager.Bind("auth.apikey", &config); err != nil {
		// Use defaults if binding fails
		config = DefaultConfig()
	}
	config.Validate() // Ensure defaults are set
	p.config = config

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
		DefaultRateLimit: config.DefaultRateLimit,
		MaxRateLimit:     config.MaxRateLimit,
		DefaultExpiry:    config.DefaultExpiry,
		MaxKeysPerUser:   config.MaxKeysPerUser,
		MaxKeysPerOrg:    config.MaxKeysPerOrg,
		KeyLength:        config.KeyLength,
	}
	p.service = apikey.NewService(apikeyRepo, auditSvc, serviceCfg)

	// Initialize middleware with rate limiting
	p.middleware = NewMiddleware(p.service, userSvc, rateLimitSvc, config)

	// Initialize handler
	p.handler = NewHandler(p.service, config)

	// Start cleanup scheduler if enabled
	if config.AutoCleanup.Enabled {
		p.startCleanupScheduler()
	}

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

