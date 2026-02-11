package multisession

import (
	"sync"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/errs"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Plugin wires the multi-session service and registers routes.
type Plugin struct {
	db                     *bun.DB
	service                *Service
	logger                 forge.Logger
	config                 Config
	defaultConfig          Config
	dashboardExtension     *DashboardExtension
	dashboardExtensionOnce sync.Once
}

// Config holds the multisession plugin configuration.
type Config struct {
	// MaxSessionsPerUser is the maximum concurrent sessions per user
	MaxSessionsPerUser int `json:"maxSessionsPerUser"`
	// EnableDeviceTracking enables device fingerprinting
	EnableDeviceTracking bool `json:"enableDeviceTracking"`
	// SessionExpiry is the session expiry time in hours
	SessionExpiryHours int `json:"sessionExpiryHours"`
	// AllowCrossPlatform allows sessions across different platforms
	AllowCrossPlatform bool `json:"allowCrossPlatform"`
}

// DefaultConfig returns the default multisession plugin configuration.
func DefaultConfig() Config {
	return Config{
		MaxSessionsPerUser:   10,
		EnableDeviceTracking: true,
		SessionExpiryHours:   720, // 30 days
		AllowCrossPlatform:   true,
	}
}

// PluginOption is a functional option for configuring the multisession plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithMaxSessionsPerUser sets the maximum concurrent sessions per user.
func WithMaxSessionsPerUser(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxSessionsPerUser = max
	}
}

// WithEnableDeviceTracking sets whether device tracking is enabled.
func WithEnableDeviceTracking(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableDeviceTracking = enable
	}
}

// WithSessionExpiryHours sets the session expiry time.
func WithSessionExpiryHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SessionExpiryHours = hours
	}
}

// WithAllowCrossPlatform sets whether cross-platform sessions are allowed.
func WithAllowCrossPlatform(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowCrossPlatform = allow
	}
}

// NewPlugin creates a new multisession plugin instance with optional configuration.
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

func (p *Plugin) ID() string { return "multisession" }

// Init accepts auth instance with GetDB method.
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.BadRequest("multisession plugin requires auth instance")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return errs.InternalServerErrorWithMessage("database not available for multisession plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.InternalServerErrorWithMessage("forge app not available for multisession plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "multisession"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.multisession", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind multisession config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// No specific Bun models for multisession (uses core Session and Device models)

	// Core services used for auth context
	auditSvc := audit.NewService(repo.NewAuditRepository(p.db))
	webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(p.db), auditSvc)
	userSvc := user.NewService(repo.NewUserRepository(p.db), user.Config{}, webhookSvc, authInst.GetHookRegistry())
	sessSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{AllowMultiple: true}, webhookSvc, authInst.GetHookRegistry())
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{}, authInst.GetHookRegistry())
	devSvc := dev.NewService(repo.NewDeviceRepository(p.db))
	p.service = NewService(
		authInst.Repository().Session(),
		sessSvc,
		authInst.Repository().Device(),
		authSvc,
		devSvc,
	)

	// Dashboard extension is lazy-initialized when first accessed via DashboardExtension()

	p.logger.Info("multisession plugin initialized",
		forge.F("max_sessions_per_user", p.config.MaxSessionsPerUser),
		forge.F("enable_device_tracking", p.config.EnableDeviceTracking),
		forge.F("session_expiry_hours", p.config.SessionExpiryHours))

	return nil
}

// RegisterRoutes mounts endpoints under /api/auth/multi-session.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the auth basePath, create multi-session sub-group
	grp := router.Group("/multi-session")
	h := NewHandler(p.service)
	if err := grp.GET("/list", h.List,
		forge.WithName("multisession.list"),
		forge.WithSummary("List user sessions"),
		forge.WithDescription("Returns all active sessions for the current authenticated user with optional filtering, sorting, and pagination"),
		forge.WithRequestSchema(ListSessionsRequest{}),
		forge.WithResponseSchema(200, "Sessions retrieved", session.ListSessionsResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}
	if err := grp.POST("/set-active", h.SetActive,
		forge.WithName("multisession.setactive"),
		forge.WithSummary("Set active session"),
		forge.WithDescription("Switches the current session cookie to the specified session ID"),
		forge.WithRequestSchema(SetActiveRequest{}),
		forge.WithResponseSchema(200, "Session activated", SessionTokenResponse{}),
		forge.WithResponseSchema(400, "Invalid request", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}
	if err := grp.POST("/delete/{id}", h.Delete,
		forge.WithName("multisession.delete"),
		forge.WithSummary("Delete session"),
		forge.WithDescription("Revokes and deletes a specific session by ID for the current user"),
		forge.WithResponseSchema(200, "Session deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}
	if err := grp.GET("/current", h.GetCurrent,
		forge.WithName("multisession.current"),
		forge.WithSummary("Get current session"),
		forge.WithDescription("Returns detailed information about the currently active session"),
		forge.WithResponseSchema(200, "Current session retrieved", SessionTokenResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(404, "Session not found", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}
	if err := grp.GET("/{id}", h.GetByID,
		forge.WithName("multisession.get"),
		forge.WithSummary("Get session by ID"),
		forge.WithDescription("Returns details about a specific session by ID with ownership verification"),
		forge.WithResponseSchema(200, "Session retrieved", SessionTokenResponse{}),
		forge.WithResponseSchema(400, "Invalid session ID", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(404, "Session not found", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}
	if err := grp.POST("/revoke-all", h.RevokeAll,
		forge.WithName("multisession.revokeall"),
		forge.WithSummary("Revoke all sessions"),
		forge.WithDescription("Revokes all sessions for the current user. Optionally include current session with includeCurrentSession flag."),
		forge.WithRequestSchema(RevokeAllRequest{}),
		forge.WithResponseSchema(200, "Sessions revoked", RevokeResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(500, "Failed to revoke sessions", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}
	if err := grp.POST("/revoke-others", h.RevokeOthers,
		forge.WithName("multisession.revokeothers"),
		forge.WithSummary("Revoke all other sessions"),
		forge.WithDescription("Revokes all sessions except the current one. Useful after password change or suspicious activity."),
		forge.WithResponseSchema(200, "Other sessions revoked", RevokeResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(500, "Failed to revoke sessions", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}
	if err := grp.POST("/refresh", h.Refresh,
		forge.WithName("multisession.refresh"),
		forge.WithSummary("Refresh current session"),
		forge.WithDescription("Extends the expiry time of the current session"),
		forge.WithResponseSchema(200, "Session refreshed", SessionTokenResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(500, "Failed to refresh session", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}
	if err := grp.GET("/stats", h.GetStats,
		forge.WithName("multisession.stats"),
		forge.WithSummary("Get session statistics"),
		forge.WithDescription("Returns aggregated statistics about user sessions including active count, device count, and location count"),
		forge.WithResponseSchema(200, "Statistics retrieved", SessionStatsResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", MultiSessionErrorResponse{}),
		forge.WithResponseSchema(500, "Failed to retrieve statistics", MultiSessionErrorResponse{}),
		forge.WithTags("MultiSession", "Sessions"),
	); err != nil {
		return err
	}

	return nil
}

// MultiSessionErrorResponse types for multi-session routes.
type MultiSessionErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

// Note: SessionTokenResponse, StatusResponse, RevokeResponse, and SessionStatsResponse
// are defined in handlers.go and reused here since we're in the same package

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }
func (p *Plugin) Migrate() error                                              { return nil }

// GetAuthService returns the auth service for testing.
func (p *Plugin) GetAuthService() *auth.Service {
	if p.service == nil {
		return nil
	}

	return p.service.auth
}

// DashboardExtension implements the PluginWithDashboardExtension interface
// This allows the multisession plugin to extend the dashboard with custom screens
// DashboardExtension lazy initialization to ensure plugin is fully initialized before creating extension.
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	p.dashboardExtensionOnce.Do(func() {
		p.dashboardExtension = NewDashboardExtension(p)
	})

	return p.dashboardExtension
}
