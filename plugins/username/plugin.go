package username

import (
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
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
}

// Config holds the username plugin configuration
type Config struct {
	// MinPasswordLength is the minimum password length
	MinPasswordLength int `json:"minPasswordLength"`
	// MaxPasswordLength is the maximum password length
	MaxPasswordLength int `json:"maxPasswordLength"`
	// RequireUppercase requires at least one uppercase letter
	RequireUppercase bool `json:"requireUppercase"`
	// RequireLowercase requires at least one lowercase letter
	RequireLowercase bool `json:"requireLowercase"`
	// RequireNumber requires at least one number
	RequireNumber bool `json:"requireNumber"`
	// RequireSpecialChar requires at least one special character
	RequireSpecialChar bool `json:"requireSpecialChar"`
	// AllowUsernameLogin allows login with username instead of email
	AllowUsernameLogin bool `json:"allowUsernameLogin"`
}

// DefaultConfig returns the default username plugin configuration
func DefaultConfig() Config {
	return Config{
		MinPasswordLength:  8,
		MaxPasswordLength:  128,
		RequireUppercase:   false,
		RequireLowercase:   false,
		RequireNumber:      false,
		RequireSpecialChar: false,
		AllowUsernameLogin: true,
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

	// Register Bun models (no specific models for username plugin currently)
	// Username auth uses core User schema

	// Construct local core services
	userSvc := user.NewService(authInst.Repository().User(), user.Config{}, nil)
	sessionSvc := session.NewService(authInst.Repository().Session(), session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	p.service = NewService(userSvc, authSvc, p.config)

	p.logger.Info("username plugin initialized",
		forge.F("min_password_length", p.config.MinPasswordLength),
		forge.F("allow_username_login", p.config.AllowUsernameLogin))

	return nil
}

// RegisterRoutes registers Username plugin routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Router is already scoped to the correct basePath
	h := NewHandler(p.service, p.authInst.Repository().TwoFA())

	router.POST("/username/signup", h.SignUp,
		forge.WithName("username.signup"),
		forge.WithSummary("Sign up with username"),
		forge.WithDescription("Creates a new user account with username and password"),
		forge.WithResponseSchema(201, "User created", UsernameStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", UsernameErrorResponse{}),
		forge.WithTags("Username", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/username/signin", h.SignIn,
		forge.WithName("username.signin"),
		forge.WithSummary("Sign in with username"),
		forge.WithDescription("Authenticates user with username and password. Returns 2FA requirement if enabled and device is not trusted"),
		forge.WithResponseSchema(200, "Sign in successful", UsernameSignInResponse{}),
		forge.WithResponseSchema(200, "2FA required", Username2FARequiredResponse{}),
		forge.WithResponseSchema(400, "Invalid request", UsernameErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid credentials", UsernameErrorResponse{}),
		forge.WithTags("Username", "Authentication"),
		forge.WithValidation(true),
	)
	return nil
}

// Response types for username routes
type UsernameErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type UsernameStatusResponse struct {
	Status string `json:"status" example:"created"`
}

type UsernameSignInResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
}

type Username2FARequiredResponse struct {
	User         interface{} `json:"user"`
	RequireTwoFA bool        `json:"require_twofa" example:"true"`
	DeviceID     string      `json:"device_id" example:"device_fingerprint"`
}

// RegisterHooks placeholder
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate placeholder for DB migrations
func (p *Plugin) Migrate() error { return nil }
