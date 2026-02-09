package anonymous

import (
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

type Plugin struct {
	service       *Service
	db            *bun.DB
	logger        forge.Logger
	config        Config
	defaultConfig Config
	authInst      core.Authsome
}

// Config holds the anonymous plugin configuration.
type Config struct {
	// EnableAnonymous enables anonymous user creation
	EnableAnonymous bool `json:"enableAnonymous"`
	// SessionExpiryHours is the anonymous session expiry time in hours
	SessionExpiryHours int `json:"sessionExpiryHours"`
	// CleanupIntervalHours is how often to clean up expired anonymous users
	CleanupIntervalHours int `json:"cleanupIntervalHours"`
	// AutoConvert allows converting anonymous users to registered users
	AutoConvert bool `json:"autoConvert"`
}

// DefaultConfig returns the default anonymous plugin configuration.
func DefaultConfig() Config {
	return Config{
		EnableAnonymous:      true,
		SessionExpiryHours:   72, // 3 days
		CleanupIntervalHours: 24,
		AutoConvert:          true,
	}
}

// PluginOption is a functional option for configuring the anonymous plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithEnableAnonymous sets whether anonymous users are enabled.
func WithEnableAnonymous(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableAnonymous = enable
	}
}

// WithSessionExpiryHours sets the session expiry time.
func WithSessionExpiryHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SessionExpiryHours = hours
	}
}

// WithCleanupIntervalHours sets the cleanup interval.
func WithCleanupIntervalHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.CleanupIntervalHours = hours
	}
}

// WithAutoConvert sets whether auto-conversion is enabled.
func WithAutoConvert(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoConvert = enable
	}
}

// NewPlugin creates a new anonymous plugin instance with optional configuration.
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

func (p *Plugin) ID() string { return "anonymous" }

// Init accepts auth instance with GetDB method.
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.InternalServerErrorWithMessage("anonymous plugin requires auth instance")
	}

	// Store auth instance for cookie support
	p.authInst = authInst

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return errs.InternalServerErrorWithMessage("database not available for anonymous plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.InternalServerErrorWithMessage("forge app not available for anonymous plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "anonymous"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.anonymous", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind anonymous config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// No specific Bun models for anonymous (uses core User and Session models)

	users := repo.NewUserRepository(p.db)
	sessionSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil, nil)
	p.service = NewService(users, sessionSvc, p.config)

	p.logger.Info("anonymous plugin initialized",
		forge.F("enable_anonymous", p.config.EnableAnonymous),
		forge.F("session_expiry_hours", p.config.SessionExpiryHours),
		forge.F("auto_convert", p.config.AutoConvert))

	return nil
}

// RegisterRoutes registers Anonymous plugin routes.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}

	h := NewHandler(p.service, p.authInst)

	// Sign in as anonymous
	router.POST("/anonymous/signin", h.SignIn,
		forge.WithName("anonymous.signin"),
		forge.WithSummary("Sign in as anonymous user"),
		forge.WithDescription("Creates a guest user and session for anonymous access"),
		forge.WithRequestSchema(SignInRequest{}),
		forge.WithResponseSchema(200, "Anonymous session created", SignInResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(500, "Server error", ErrorResponse{}),
		forge.WithTags("Anonymous", "Authentication"),
		forge.WithValidation(true),
	)

	// Link anonymous account to real account
	router.POST("/anonymous/link", h.Link,
		forge.WithName("anonymous.link"),
		forge.WithSummary("Link anonymous account"),
		forge.WithDescription("Upgrades an anonymous session to a registered account with email/password"),
		forge.WithRequestSchema(LinkRequest{}),
		forge.WithResponseSchema(200, "Account linked successfully", LinkResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("Anonymous", "Authentication"),
		forge.WithValidation(true),
	)

	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error { return nil }
