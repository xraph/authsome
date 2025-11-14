package magiclink

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

type Plugin struct {
	db            *bun.DB
	service       *Service
	notifAdapter  *notificationPlugin.Adapter
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// Config holds the magic link plugin configuration
type Config struct {
	// ExpiryMinutes is the magic link expiry time in minutes
	ExpiryMinutes int `json:"expiryMinutes"`
	// BaseURL is the base URL for magic link generation
	BaseURL string `json:"baseURL"`
	// AllowImplicitSignup allows creating users if they don't exist
	AllowImplicitSignup bool `json:"allowImplicitSignup"`
	// RateLimitPerHour is the max requests per hour per user
	RateLimitPerHour int `json:"rateLimitPerHour"`
}

// DefaultConfig returns the default magic link plugin configuration
func DefaultConfig() Config {
	return Config{
		ExpiryMinutes:       15,
		BaseURL:             "http://localhost:8080",
		AllowImplicitSignup: true,
		RateLimitPerHour:    10,
	}
}

// PluginOption is a functional option for configuring the magic link plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithExpiryMinutes sets the magic link expiry time
func WithExpiryMinutes(minutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ExpiryMinutes = minutes
	}
}

// WithBaseURL sets the base URL for magic links
func WithBaseURL(url string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.BaseURL = url
	}
}

// WithAllowImplicitSignup sets whether implicit signup is allowed
func WithAllowImplicitSignup(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowImplicitSignup = allow
	}
}

// WithRateLimitPerHour sets the rate limit per hour
func WithRateLimitPerHour(limit int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RateLimitPerHour = limit
	}
}

// NewPlugin creates a new magic link plugin instance with optional configuration
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

func (p *Plugin) ID() string { return "magiclink" }

func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("magiclink plugin requires auth instance")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for magiclink plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for magiclink plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "magiclink"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.magiclink", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind magic link config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models
	p.db.RegisterModel((*schema.MagicLink)(nil))

	// TODO: Get notification adapter from service registry when available
	// For now, plugins will work without notification adapter (graceful degradation)
	// The notification plugin should be registered first and will set up its services

	mr := repo.NewMagicLinkRepository(p.db)
	userSvc := user.NewService(repo.NewUserRepository(p.db), user.Config{}, nil)
	// Build full auth service with session
	sessRepo := repo.NewSessionRepository(p.db)
	sessionSvc := session.NewService(sessRepo, session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(p.db))
	p.service = NewService(mr, userSvc, authSvc, auditSvc, p.notifAdapter, p.config)

	p.logger.Info("magic link plugin initialized",
		forge.F("expiry_minutes", p.config.ExpiryMinutes),
		forge.F("base_url", p.config.BaseURL),
		forge.F("allow_implicit_signup", p.config.AllowImplicitSignup))

	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the correct basePath
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/magic-link/send": {Window: time.Minute, Max: 5}}})
	h := NewHandler(p.service, rls)
	router.POST("/magic-link/send", h.Send,
		forge.WithName("magiclink.send"),
		forge.WithSummary("Send magic link"),
		forge.WithDescription("Sends a passwordless authentication link to the specified email address. Rate limited to 5 requests per minute per email"),
		forge.WithResponseSchema(200, "Magic link sent", MagicLinkSendResponse{}),
		forge.WithResponseSchema(400, "Invalid request", MagicLinkErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", MagicLinkErrorResponse{}),
		forge.WithTags("MagicLink", "Authentication"),
		forge.WithValidation(true),
	)
	router.GET("/magic-link/verify", h.Verify,
		forge.WithName("magiclink.verify"),
		forge.WithSummary("Verify magic link"),
		forge.WithDescription("Verifies the magic link token from email and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithResponseSchema(200, "Magic link verified", MagicLinkVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", MagicLinkErrorResponse{}),
		forge.WithTags("MagicLink", "Authentication"),
	)
	return nil
}

// Response types for magic link routes
type MagicLinkErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type MagicLinkSendResponse struct {
	Status string `json:"status" example:"sent"`
	DevURL string `json:"dev_url,omitempty" example:"http://localhost:3000/magic-link/verify?token=abc123"`
}

type MagicLinkVerifyResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.MagicLink)(nil)).IfNotExists().Exec(ctx)
	return err
}
