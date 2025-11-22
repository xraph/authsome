package phone

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
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
	authInst      core.Authsome
}

// Config holds the phone plugin configuration
type Config struct {
	// CodeLength is the length of the verification code
	CodeLength int `json:"codeLength"`
	// ExpiryMinutes is the code expiry time in minutes
	ExpiryMinutes int `json:"expiryMinutes"`
	// MaxAttempts is the maximum verification attempts
	MaxAttempts int `json:"maxAttempts"`
	// AllowImplicitSignup allows creating users if they don't exist
	AllowImplicitSignup bool `json:"allowImplicitSignup"`
	// SMSProvider is the SMS provider to use (twilio, etc.)
	SMSProvider string `json:"smsProvider"`
	// DevExposeCode exposes the code in dev mode (for testing)
	DevExposeCode bool `json:"devExposeCode"`
	
	// Rate limiting configuration
	RateLimit RateLimitConfig `json:"rateLimit"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Enabled enables rate limiting
	Enabled bool `json:"enabled"`
	// UseRedis uses Redis for distributed rate limiting (recommended for production)
	UseRedis bool `json:"useRedis"`
	// RedisAddr is the Redis server address (e.g., "localhost:6379")
	RedisAddr string `json:"redisAddr"`
	// RedisPassword is the Redis password (optional)
	RedisPassword string `json:"redisPassword"`
	// RedisDB is the Redis database number
	RedisDB int `json:"redisDb"`
	
	// SendCodePerPhone limits send code requests per phone number
	SendCodePerPhone RateLimitRule `json:"sendCodePerPhone"`
	// SendCodePerIP limits send code requests per IP address
	SendCodePerIP RateLimitRule `json:"sendCodePerIp"`
	// VerifyPerPhone limits verify requests per phone number
	VerifyPerPhone RateLimitRule `json:"verifyPerPhone"`
	// VerifyPerIP limits verify requests per IP address
	VerifyPerIP RateLimitRule `json:"verifyPerIp"`
}

// RateLimitRule defines a rate limit rule
type RateLimitRule struct {
	// Window is the time window for the rate limit (e.g., "1m", "1h")
	Window time.Duration `json:"window"`
	// Max is the maximum number of requests in the window
	Max int `json:"max"`
}

// DefaultConfig returns the default phone plugin configuration
func DefaultConfig() Config {
	return Config{
		CodeLength:          6,
		ExpiryMinutes:       10,
		MaxAttempts:         5,
		AllowImplicitSignup: true,
		SMSProvider:         "twilio",
		DevExposeCode:       false,
		RateLimit: RateLimitConfig{
			Enabled:   true,
			UseRedis:  false, // Use memory by default, set to true for production
			RedisAddr: "localhost:6379",
			RedisDB:   0,
			SendCodePerPhone: RateLimitRule{
				Window: 1 * time.Minute,
				Max:    3, // 3 requests per minute per phone
			},
			SendCodePerIP: RateLimitRule{
				Window: 1 * time.Hour,
				Max:    20, // 20 requests per hour per IP
			},
			VerifyPerPhone: RateLimitRule{
				Window: 5 * time.Minute,
				Max:    10, // 10 verify attempts per 5 minutes per phone
			},
			VerifyPerIP: RateLimitRule{
				Window: 1 * time.Hour,
				Max:    50, // 50 verify attempts per hour per IP
			},
		},
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

// WithRateLimitSendCodePerPhone sets the send code rate limit per phone
func WithRateLimitSendCodePerPhone(window time.Duration, max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RateLimit.SendCodePerPhone = RateLimitRule{Window: window, Max: max}
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

	// Store auth instance for middleware access
	p.authInst = authInst

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

	// Get notification adapter from service registry (graceful degradation if not available)
	svcRegistry := authInst.GetServiceRegistry()
	if svcRegistry != nil {
		notifSvc, ok := svcRegistry.Get("notification")
		if ok && notifSvc != nil {
			if adapter, ok := notifSvc.(*notificationPlugin.Adapter); ok {
				p.notifAdapter = adapter
				p.logger.Info("notification adapter loaded for phone plugin")
			}
		}
	}
	
	if p.notifAdapter == nil {
		p.logger.Warn("notification adapter not available, SMS sending will be skipped")
	}

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
	
	// Setup rate limiting storage
	var rateLimitStorage rl.Storage
	if p.config.RateLimit.Enabled {
		if p.config.RateLimit.UseRedis {
			// Create Redis client for distributed rate limiting
			redisClient := redis.NewClient(&redis.Options{
				Addr:     p.config.RateLimit.RedisAddr,
				Password: p.config.RateLimit.RedisPassword,
				DB:       p.config.RateLimit.RedisDB,
			})
			
			// Test Redis connection
			ctx := context.Background()
			if err := redisClient.Ping(ctx).Err(); err != nil {
				p.logger.Error("failed to connect to Redis, falling back to memory storage",
					forge.F("error", err.Error()))
				rateLimitStorage = storage.NewMemoryStorage()
			} else {
				rateLimitStorage = storage.NewRedisStorage(redisClient)
				p.logger.Info("using Redis for rate limiting",
					forge.F("addr", p.config.RateLimit.RedisAddr))
			}
		} else {
			rateLimitStorage = storage.NewMemoryStorage()
			p.logger.Info("using in-memory storage for rate limiting")
		}
	} else {
		rateLimitStorage = storage.NewMemoryStorage()
	}
	
	// Configure rate limiting rules
	rules := map[string]rl.Rule{
		"/phone/send-code": {
			Window: p.config.RateLimit.SendCodePerPhone.Window,
			Max:    p.config.RateLimit.SendCodePerPhone.Max,
		},
		"/phone/verify": {
			Window: p.config.RateLimit.VerifyPerPhone.Window,
			Max:    p.config.RateLimit.VerifyPerPhone.Max,
		},
		"/phone/signin": {
			Window: p.config.RateLimit.VerifyPerPhone.Window,
			Max:    p.config.RateLimit.VerifyPerPhone.Max,
		},
	}
	
	rls := rl.NewService(rateLimitStorage, rl.Config{
		Enabled: p.config.RateLimit.Enabled,
		Rules:   rules,
	})
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
	
	router.POST("/phone/send-code", wrapHandler(h.SendCode),
		forge.WithName("phone.sendcode"),
		forge.WithSummary("Send phone verification code"),
		forge.WithDescription("Sends a verification code via SMS to the specified phone number. Rate limited to 5 requests per minute per phone"),
		forge.WithRequestSchema(SendCodeRequest{}),
		forge.WithResponseSchema(200, "Code sent", SendCodeResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", ErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/phone/verify", wrapHandler(h.Verify),
		forge.WithName("phone.verify"),
		forge.WithSummary("Verify phone code"),
		forge.WithDescription("Verifies the phone verification code and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithRequestSchema(VerifyRequest{}),
		forge.WithResponseSchema(200, "Code verified", PhoneVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", ErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/phone/signin", wrapHandler(h.SignIn),
		forge.WithName("phone.signin"),
		forge.WithSummary("Sign in with phone"),
		forge.WithDescription("Alias for phone verification. Verifies the phone code and creates a user session"),
		forge.WithRequestSchema(VerifyRequest{}),
		forge.WithResponseSchema(200, "Sign in successful", PhoneVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", ErrorResponse{}),
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

// Type alias for route registration
type ErrorResponse = errs.AuthsomeError
