package phone

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

// Config holds the phone plugin configuration
type Config struct {
	// CodeLength is the length of the verification code
	CodeLength int `json:"codeLength"`
	// ExpiryMinutes is the code expiry time in minutes
	ExpiryMinutes int `json:"expiryMinutes"`
	// MaxAttempts is the maximum verification attempts
	MaxAttempts int `json:"maxAttempts"`
	// RateLimitPerHour is the max SMS requests per hour
	RateLimitPerHour int `json:"rateLimitPerHour"`
	// AllowImplicitSignup allows creating users if they don't exist
	AllowImplicitSignup bool `json:"allowImplicitSignup"`
	// SMSProvider is the SMS provider to use (twilio, etc.)
	SMSProvider string `json:"smsProvider"`
	// DevExposeCode exposes the code in dev mode (for testing)
	DevExposeCode bool `json:"devExposeCode"`
}

// DefaultConfig returns the default phone plugin configuration
func DefaultConfig() Config {
	return Config{
		CodeLength:          6,
		ExpiryMinutes:       10,
		MaxAttempts:         5,
		RateLimitPerHour:    10,
		AllowImplicitSignup: true,
		SMSProvider:         "twilio",
		DevExposeCode:       false,
	}
}

// PluginOption is a functional option for configuring the phone plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithCodeLength sets the verification code length
func WithCodeLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.CodeLength = length
	}
}

// WithExpiryMinutes sets the code expiry time
func WithExpiryMinutes(minutes int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ExpiryMinutes = minutes
	}
}

// WithMaxAttempts sets the maximum verification attempts
func WithMaxAttempts(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxAttempts = max
	}
}

// WithRateLimitPerHour sets the rate limit per hour
func WithRateLimitPerHour(limit int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RateLimitPerHour = limit
	}
}

// WithSMSProvider sets the SMS provider
func WithSMSProvider(provider string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SMSProvider = provider
	}
}

// WithAllowImplicitSignup sets whether implicit signup is allowed
func WithAllowImplicitSignup(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowImplicitSignup = allow
	}
}

// WithDevExposeCode sets whether to expose codes in dev mode
func WithDevExposeCode(expose bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DevExposeCode = expose
	}
}

// NewPlugin creates a new phone plugin instance with optional configuration
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

func (p *Plugin) ID() string { return "phone" }

func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("phone plugin requires auth instance")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for phone plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for phone plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "phone"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.phone", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind phone config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models
	p.db.RegisterModel((*schema.PhoneVerification)(nil))

	// TODO: Get notification adapter from service registry when available
	// For now, plugins will work without notification adapter (graceful degradation)
	// The notification plugin should be registered first and will set up its services
	p.notifAdapter = authInst.GetServiceRegistry().Get("notification").(*notificationPlugin.Adapter)

	pr := authInst.Repository().Phone()
	userSvc := user.NewService(authInst.Repository().User(), user.Config{}, nil)
	sessSvc := session.NewService(authInst.Repository().Session(), session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
	auditSvc := audit.NewService(authInst.Repository().Audit())
	p.service = NewService(pr, userSvc, authSvc, auditSvc, p.notifAdapter, p.config)

	p.logger.Info("phone plugin initialized",
		forge.F("code_length", p.config.CodeLength),
		forge.F("expiry_minutes", p.config.ExpiryMinutes),
		forge.F("sms_provider", p.config.SMSProvider))

	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the correct basePath
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/phone/send-code": {Window: time.Minute, Max: 5}}})
	h := NewHandler(p.service, rls)
	router.POST("/phone/send-code", h.SendCode,
		forge.WithName("phone.sendcode"),
		forge.WithSummary("Send phone verification code"),
		forge.WithDescription("Sends a verification code via SMS to the specified phone number. Rate limited to 5 requests per minute per phone"),
		forge.WithResponseSchema(200, "Code sent", PhoneSendCodeResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PhoneErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", PhoneErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/phone/verify", h.Verify,
		forge.WithName("phone.verify"),
		forge.WithSummary("Verify phone code"),
		forge.WithDescription("Verifies the phone verification code and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithResponseSchema(200, "Code verified", PhoneVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PhoneErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", PhoneErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/phone/signin", h.SignIn,
		forge.WithName("phone.signin"),
		forge.WithSummary("Sign in with phone"),
		forge.WithDescription("Alias for phone verification. Verifies the phone code and creates a user session"),
		forge.WithResponseSchema(200, "Sign in successful", PhoneVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PhoneErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", PhoneErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.PhoneVerification)(nil)).IfNotExists().Exec(ctx)
	return err
}

// Response types for phone routes
type PhoneErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type PhoneSendCodeResponse struct {
	Status  string `json:"status" example:"sent"`
	DevCode string `json:"dev_code,omitempty" example:"123456"`
}

type PhoneVerifyResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
}
