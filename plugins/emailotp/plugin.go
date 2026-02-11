package emailotp

import (
	"context"
	"net/http"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Email OTP.
type Plugin struct {
	service       *Service
	notifAdapter  *notificationPlugin.Adapter
	db            *bun.DB
	logger        forge.Logger
	config        Config
	defaultConfig Config
	authInst      core.Authsome
}

// Config holds the email OTP plugin configuration.
type Config struct {
	// OTPLength is the length of the OTP code
	OTPLength int `json:"otpLength"`
	// ExpiryMinutes is the OTP expiry time in minutes
	ExpiryMinutes int `json:"expiryMinutes"`
	// MaxAttempts is the maximum verification attempts
	MaxAttempts int `json:"maxAttempts"`
	// RateLimitPerHour is the max OTP requests per hour
	RateLimitPerHour int `json:"rateLimitPerHour"`
	// AllowImplicitSignup allows creating users if they don't exist
	AllowImplicitSignup bool `json:"allowImplicitSignup"`
	// DevExposeOTP exposes the OTP in dev mode (for testing)
	DevExposeOTP bool `json:"devExposeOTP"`
}

// DefaultConfig returns the default email OTP plugin configuration.
func DefaultConfig() Config {
	return Config{
		OTPLength:           6,
		ExpiryMinutes:       10,
		MaxAttempts:         5,
		RateLimitPerHour:    10,
		AllowImplicitSignup: true,
	}
}

// PluginOption is a functional option for configuring the email OTP plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithOTPLength sets the OTP code length.
func WithOTPLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.OTPLength = length
	}
}

// WithExpiryMinutes sets the OTP expiry time.
func WithExpiryMinutes(minutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ExpiryMinutes = minutes
	}
}

// WithMaxAttempts sets the maximum verification attempts.
func WithMaxAttempts(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxAttempts = max
	}
}

// WithRateLimitPerHour sets the rate limit per hour.
func WithRateLimitPerHour(limit int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RateLimitPerHour = limit
	}
}

// WithAllowImplicitSignup sets whether implicit signup is allowed.
func WithAllowImplicitSignup(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowImplicitSignup = allow
	}
}

// NewPlugin creates a new email OTP plugin instance with optional configuration.
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

func (p *Plugin) ID() string { return "emailotp" }

func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.New("EMAILOTP_PLUGIN_REQUIRES_AUTH_INSTANCE", "Email OTP plugin requires auth instance", http.StatusInternalServerError)
	}

	// Store auth instance for middleware access
	p.authInst = authInst

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return errs.New("DATABASE_NOT_AVAILABLE", "Database not available for email OTP plugin", http.StatusInternalServerError)
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.New("FORGE_APP_NOT_AVAILABLE", "Forge app not available for email OTP plugin", http.StatusInternalServerError)
	}

	// Initialize logger
	p.logger = authInst.Logger().With(forge.F("plugin", "emailotp"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.emailotp", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind email OTP config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models
	p.db.RegisterModel((*schema.EmailOTP)(nil))

	// Get notification adapter from service registry
	serviceRegistry := authInst.GetServiceRegistry()
	if serviceRegistry != nil {
		if adapter, exists := serviceRegistry.Get("notification.adapter"); exists {
			if typedAdapter, ok := adapter.(*notificationPlugin.Adapter); ok {
				p.notifAdapter = typedAdapter
				p.logger.Debug("retrieved notification adapter from service registry")
			} else {
				p.logger.Warn("notification adapter type assertion failed")
			}
		} else {
			p.logger.Debug("notification adapter not available in service registry (graceful degradation)")
		}
	}

	// wire repo and services
	eotpr := repo.NewEmailOTPRepository(p.db)
	userSvc := user.NewService(repo.NewUserRepository(p.db), user.Config{}, nil, nil)
	sessionSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil, nil)
	auditSvc := audit.NewService(repo.NewAuditRepository(p.db))
	p.service = NewService(eotpr, userSvc, sessionSvc, auditSvc, p.notifAdapter, p.config, p.logger)

	p.logger.Info("email OTP plugin initialized",
		forge.F("otp_length", p.config.OTPLength),
		forge.F("expiry_minutes", p.config.ExpiryMinutes),
		forge.F("max_attempts", p.config.MaxAttempts))

	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}

	// Router is already scoped to the correct basePath
	// Set up a simple in-memory rate limit: 5 sends per minute per email
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/email-otp/send": {Window: time.Minute, Max: 5}}})
	h := NewHandler(p.service, rls, p.authInst)

	// Get authentication middleware for API key validation
	authMw := p.authInst.AuthMiddleware()

	// Wrap handler with middleware if available
	wrapHandler := func(handler func(forge.Context) error) func(forge.Context) error {
		if authMw != nil {
			return authMw(handler)
		}

		return handler
	}

	if err := router.POST("/email-otp/send", wrapHandler(h.Send),
		forge.WithName("emailotp.send"),
		forge.WithSummary("Send email OTP"),
		forge.WithDescription("Sends a one-time password (OTP) to the specified email address. Rate limited to 5 requests per minute per email"),
		forge.WithRequestSchema(SendRequest{}),
		forge.WithResponseSchema(200, "OTP sent", SendResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", ErrorResponse{}),
		forge.WithResponseSchema(500, "Server error", ErrorResponse{}),
		forge.WithTags("EmailOTP", "Authentication"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}

	if err := router.POST("/email-otp/verify", wrapHandler(h.Verify),
		forge.WithName("emailotp.verify"),
		forge.WithSummary("Verify email OTP"),
		forge.WithDescription("Verifies the OTP code and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithRequestSchema(VerifyRequest{}),
		forge.WithResponseSchema(200, "OTP verified", VerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid OTP", ErrorResponse{}),
		forge.WithTags("EmailOTP", "Authentication"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.EmailOTP)(nil)).IfNotExists().Exec(ctx)

	return err
}
