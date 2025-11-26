package username

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
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Username auth
type Plugin struct {
	service       *Service
	db            *bun.DB
	logger        forge.Logger
	config        Config
	defaultConfig Config
	authInst      core.Authsome
	usernameRepo  *repo.UsernameRepository
}

// Config holds the username plugin configuration
type Config struct {
	// Password requirements (existing)
	MinPasswordLength  int  `json:"minPasswordLength"`
	MaxPasswordLength  int  `json:"maxPasswordLength"`
	RequireUppercase   bool `json:"requireUppercase"`
	RequireLowercase   bool `json:"requireLowercase"`
	RequireNumber      bool `json:"requireNumber"`
	RequireSpecialChar bool `json:"requireSpecialChar"`
	AllowUsernameLogin bool `json:"allowUsernameLogin"`

	// Account lockout configuration
	LockoutEnabled      bool          `json:"lockoutEnabled"`
	MaxFailedAttempts   int           `json:"maxFailedAttempts"`
	LockoutDuration     time.Duration `json:"lockoutDuration"`
	FailedAttemptWindow time.Duration `json:"failedAttemptWindow"`

	// Password history configuration
	PasswordHistorySize  int  `json:"passwordHistorySize"`
	PreventPasswordReuse bool `json:"preventPasswordReuse"`

	// Password expiry configuration
	PasswordExpiryEnabled bool `json:"passwordExpiryEnabled"`
	PasswordExpiryDays    int  `json:"passwordExpiryDays"`
	PasswordExpiryWarning int  `json:"passwordExpiryWarningDays"`

	// Rate limiting configuration
	RateLimit RateLimitConfig `json:"rateLimit"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled       bool   `json:"enabled"`
	UseRedis      bool   `json:"useRedis"`
	RedisAddr     string `json:"redisAddr"`
	RedisPassword string `json:"redisPassword"`
	RedisDB       int    `json:"redisDb"`

	SignUpPerIP   RateLimitRule `json:"signupPerIp"`
	SignInPerIP   RateLimitRule `json:"signinPerIp"`
	SignInPerUser RateLimitRule `json:"signinPerUser"`
}

// RateLimitRule defines a rate limit rule
type RateLimitRule struct {
	Window time.Duration `json:"window"`
	Max    int           `json:"max"`
}

// DefaultConfig returns the default username plugin configuration
func DefaultConfig() Config {
	return Config{
		// Password requirements
		MinPasswordLength:  8,
		MaxPasswordLength:  128,
		RequireUppercase:   false,
		RequireLowercase:   false,
		RequireNumber:      false,
		RequireSpecialChar: false,
		AllowUsernameLogin: true,

		// Account lockout
		LockoutEnabled:      true,
		MaxFailedAttempts:   5,
		LockoutDuration:     15 * time.Minute,
		FailedAttemptWindow: 10 * time.Minute,

		// Password history
		PasswordHistorySize:  5,
		PreventPasswordReuse: true,

		// Password expiry
		PasswordExpiryEnabled: false,
		PasswordExpiryDays:    90,
		PasswordExpiryWarning: 7,

		// Rate limiting
		RateLimit: RateLimitConfig{
			Enabled:   true,
			UseRedis:  false,
			RedisAddr: "localhost:6379",
			RedisDB:   0,
			SignUpPerIP:   RateLimitRule{Window: 1 * time.Hour, Max: 10},
			SignInPerIP:   RateLimitRule{Window: 15 * time.Minute, Max: 20},
			SignInPerUser: RateLimitRule{Window: 5 * time.Minute, Max: 5},
		},
	}
}

// PluginOption is a functional option for configuring the username plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithMinPasswordLength sets the minimum password length
func WithMinPasswordLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MinPasswordLength = length
	}
}

// WithMaxPasswordLength sets the maximum password length
func WithMaxPasswordLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxPasswordLength = length
	}
}

// WithRequireUppercase sets whether uppercase letters are required
func WithRequireUppercase(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireUppercase = required
	}
}

// WithRequireLowercase sets whether lowercase letters are required
func WithRequireLowercase(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireLowercase = required
	}
}

// WithRequireNumber sets whether numbers are required
func WithRequireNumber(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireNumber = required
	}
}

// WithRequireSpecialChar sets whether special characters are required
func WithRequireSpecialChar(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireSpecialChar = required
	}
}

// WithAllowUsernameLogin sets whether username login is allowed
func WithAllowUsernameLogin(allowed bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowUsernameLogin = allowed
	}
}

// WithLockoutEnabled sets whether account lockout is enabled
func WithLockoutEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.LockoutEnabled = enabled
	}
}

// WithMaxFailedAttempts sets the maximum failed attempts before lockout
func WithMaxFailedAttempts(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxFailedAttempts = max
	}
}

// WithLockoutDuration sets the account lockout duration
func WithLockoutDuration(duration time.Duration) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.LockoutDuration = duration
	}
}

// WithPasswordHistorySize sets the password history size
func WithPasswordHistorySize(size int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.PasswordHistorySize = size
	}
}

// WithPreventPasswordReuse sets whether password reuse is prevented
func WithPreventPasswordReuse(prevent bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.PreventPasswordReuse = prevent
	}
}

// WithPasswordExpiryEnabled sets whether password expiry is enabled
func WithPasswordExpiryEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.PasswordExpiryEnabled = enabled
	}
}

// WithPasswordExpiryDays sets the password expiry days
func WithPasswordExpiryDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.PasswordExpiryDays = days
	}
}

// NewPlugin creates a new username plugin instance with optional configuration
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

func (p *Plugin) ID() string { return "username" }

// Init accepts auth instance with GetDB method
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("username plugin requires auth instance")
	}

	p.authInst = authInst

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for username plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for username plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "username"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.username", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind username config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models for username plugin features
	p.db.RegisterModel((*schema.FailedLoginAttempt)(nil))
	p.db.RegisterModel((*schema.PasswordHistory)(nil))
	p.db.RegisterModel((*schema.AccountLockout)(nil))

	// Create username repository for lockout and password history
	p.usernameRepo = repo.NewUsernameRepository(p.db)

	// Construct core services
	userSvc := user.NewService(authInst.Repository().User(), user.Config{}, nil, authInst.GetHookRegistry())
	sessionSvc := session.NewService(authInst.Repository().Session(), session.Config{}, nil, authInst.GetHookRegistry())
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{}, authInst.GetHookRegistry())
	auditSvc := audit.NewService(authInst.Repository().Audit())
	
	// Create username service with all dependencies
	p.service = NewService(userSvc, authSvc, auditSvc, p.usernameRepo, p.config)

	p.logger.Info("username plugin initialized",
		forge.F("min_password_length", p.config.MinPasswordLength),
		forge.F("allow_username_login", p.config.AllowUsernameLogin),
		forge.F("lockout_enabled", p.config.LockoutEnabled),
		forge.F("password_history_size", p.config.PasswordHistorySize),
		forge.F("password_expiry_enabled", p.config.PasswordExpiryEnabled))

	return nil
}

// RegisterRoutes registers Username plugin routes
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
		"/username/signup": {
			Window: p.config.RateLimit.SignUpPerIP.Window,
			Max:    p.config.RateLimit.SignUpPerIP.Max,
		},
		"/username/signin": {
			Window: p.config.RateLimit.SignInPerIP.Window,
			Max:    p.config.RateLimit.SignInPerIP.Max,
		},
	}

	rls := rl.NewService(rateLimitStorage, rl.Config{
		Enabled: p.config.RateLimit.Enabled,
		Rules:   rules,
	})
	
	h := NewHandler(p.service, rls, p.authInst.Repository().TwoFA())

	router.POST("/username/signup", h.SignUp,
		forge.WithName("username.signup"),
		forge.WithSummary("Sign up with username"),
		forge.WithDescription("Creates a new user account with username and password. Validates password strength and checks for username availability"),
		forge.WithRequestSchema(SignUpRequest{}),
		forge.WithResponseSchema(201, "User created", SignUpResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(409, "Username exists", errs.AuthsomeError{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", errs.AuthsomeError{}),
		forge.WithTags("Username", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/username/signin", h.SignIn,
		forge.WithName("username.signin"),
		forge.WithSummary("Sign in with username"),
		forge.WithDescription("Authenticates user with username and password. Returns 2FA requirement if enabled. Implements account lockout after failed attempts"),
		forge.WithRequestSchema(SignInRequest{}),
		forge.WithResponseSchema(200, "Sign in successful", SignInResponse{}),
		forge.WithResponseSchema(200, "2FA required", TwoFARequiredResponse{}),
		forge.WithResponseSchema(400, "Invalid request", errs.AuthsomeError{}),
		forge.WithResponseSchema(401, "Invalid credentials", errs.AuthsomeError{}),
		forge.WithResponseSchema(403, "Account locked", AccountLockedResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", errs.AuthsomeError{}),
		forge.WithTags("Username", "Authentication"),
		forge.WithValidation(true),
	)
	return nil
}


// RegisterHooks placeholder
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate creates database tables for username plugin features
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()

	// Create failed_login_attempts table
	if _, err := p.db.NewCreateTable().Model((*schema.FailedLoginAttempt)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("failed to create failed_login_attempts table: %w", err)
	}

	// Create password_histories table
	if _, err := p.db.NewCreateTable().Model((*schema.PasswordHistory)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("failed to create password_histories table: %w", err)
	}

	// Create account_lockouts table
	if _, err := p.db.NewCreateTable().Model((*schema.AccountLockout)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("failed to create account_lockouts table: %w", err)
	}

	return nil
}
