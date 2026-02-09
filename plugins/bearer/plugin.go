package bearer

import (
	"strings"
	"time"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Plugin implements bearer token authentication middleware.
type Plugin struct {
	sessionSvc    session.ServiceInterface
	userSvc       user.ServiceInterface
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// Config holds the bearer plugin configuration.
type Config struct {
	// TokenPrefix is the expected prefix for bearer tokens
	TokenPrefix string `json:"tokenPrefix"`
	// ValidateIssuer checks the token issuer
	ValidateIssuer bool `json:"validateIssuer"`
	// RequireScopes enforces scope validation
	RequireScopes []string `json:"requireScopes"`
	// CaseSensitive makes token comparison case-sensitive
	CaseSensitive bool `json:"caseSensitive"`
}

// DefaultConfig returns the default bearer plugin configuration.
func DefaultConfig() Config {
	return Config{
		TokenPrefix:    "Bearer",
		ValidateIssuer: false,
		RequireScopes:  []string{},
		CaseSensitive:  false,
	}
}

// PluginOption is a functional option for configuring the bearer plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithTokenPrefix sets the token prefix.
func WithTokenPrefix(prefix string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TokenPrefix = prefix
	}
}

// WithValidateIssuer sets whether to validate the issuer.
func WithValidateIssuer(validate bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ValidateIssuer = validate
	}
}

// WithRequireScopes sets the required scopes.
func WithRequireScopes(scopes []string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireScopes = scopes
	}
}

// NewPlugin creates a new bearer token plugin with optional configuration.
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

// Init initializes the bearer plugin.
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.InternalServerErrorWithMessage("bearer plugin requires auth instance")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.InternalServerErrorWithMessage("forge app not available for bearer plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "bearer"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.bearer", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind bearer config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Get services from registry
	serviceRegistry := authInst.GetServiceRegistry()
	p.sessionSvc = serviceRegistry.SessionService()
	p.userSvc = serviceRegistry.UserService()

	if p.sessionSvc == nil || p.userSvc == nil {
		return errs.InternalServerErrorWithMessage("bearer plugin requires session and user services")
	}

	// Register bearer authentication strategy
	bearerStrategy := NewBearerStrategy(p.sessionSvc, p.userSvc, p.config, p.logger)
	if err := authInst.RegisterAuthStrategy(bearerStrategy); err != nil {
		p.logger.Warn("failed to register bearer strategy",
			forge.F("error", err.Error()))
		// Don't fail initialization if strategy registration fails
		// The middleware handlers can still work independently
	}

	p.logger.Info("bearer plugin initialized",
		forge.F("token_prefix", p.config.TokenPrefix),
		forge.F("validate_issuer", p.config.ValidateIssuer),
		forge.F("strategy_registered", true))

	return nil
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return "bearer"
}

// RegisterRoutes registers plugin routes (no-op for bearer - it's middleware-only).
func (p *Plugin) RegisterRoutes(_ forge.Router) error {
	return nil
}

// RegisterHooks registers plugin hooks (no-op for bearer).
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error {
	return nil
}

// RegisterServiceDecorators registers service decorators (no-op for bearer).
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	return nil
}

// Migrate runs plugin migrations (no-op for bearer - no database tables).
func (p *Plugin) Migrate() error {
	return nil
}

// AuthenticateHandler returns a handler function that can be used as middleware.
func (p *Plugin) AuthenticateHandler(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Extract bearer token from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return next(c)
		}

		// Check if it's a bearer token with configured prefix
		prefix := p.config.TokenPrefix + " "
		if !p.config.CaseSensitive {
			if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
				return next(c)
			}
		} else {
			if !strings.HasPrefix(authHeader, prefix) {
				return next(c)
			}
		}

		token := strings.TrimPrefix(authHeader, prefix)

		token = strings.TrimSpace(token)
		if token == "" {
			return next(c)
		}

		// Validate the token using session service
		sess, err := p.sessionSvc.FindByToken(c.Request().Context(), token)
		if err != nil {
			p.logger.Debug("failed to find session by token",
				forge.F("error", err.Error()))

			return next(c)
		}

		// Check if session is valid and not expired
		if sess == nil {
			return next(c)
		}

		if time.Now().After(sess.ExpiresAt) {
			p.logger.Debug("session expired",
				forge.F("session_id", sess.ID.String()),
				forge.F("expires_at", sess.ExpiresAt))

			return next(c)
		}

		// Get user information
		usr, err := p.userSvc.FindByID(c.Request().Context(), sess.UserID)
		if err != nil {
			p.logger.Warn("failed to find user for session",
				forge.F("user_id", sess.UserID.String()),
				forge.F("error", err.Error()))

			return next(c)
		}

		if usr == nil {
			p.logger.Warn("user not found for valid session",
				forge.F("user_id", sess.UserID.String()))

			return next(c)
		}

		// Build AuthContext
		authCtx := &contexts.AuthContext{
			Session:         sess,
			User:            usr,
			AppID:           sess.AppID,
			EnvironmentID:   *sess.EnvironmentID,
			OrganizationID:  sess.OrganizationID,
			Method:          contexts.AuthMethodSession,
			IsAuthenticated: true,
			IsUserAuth:      true,
			IPAddress:       c.Request().RemoteAddr,
			UserAgent:       c.Request().Header.Get("User-Agent"),
		}

		// Store auth context in request context
		ctx := contexts.SetAuthContext(c.Request().Context(), authCtx)

		// Also set the individual context values for backward compatibility
		ctx = contexts.SetAppID(ctx, sess.AppID)
		if sess.EnvironmentID != nil {
			ctx = contexts.SetEnvironmentID(ctx, *sess.EnvironmentID)
		}

		if sess.OrganizationID != nil {
			ctx = contexts.SetOrganizationID(ctx, *sess.OrganizationID)
		}

		ctx = contexts.SetUserID(ctx, usr.ID)

		// Update request with new context
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// RequireAuthHandler returns a handler that requires authentication.
func (p *Plugin) RequireAuthHandler(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Check if user is authenticated
		if _, err := contexts.RequireUser(c.Request().Context()); err != nil {
			authErr := errs.Unauthorized().
				WithContext("plugin", "bearer").
				WithContext("reason", "authentication_required")

			return c.JSON(authErr.HTTPStatus, map[string]any{
				"error":   authErr.Code,
				"message": authErr.Message,
			})
		}

		return next(c)
	}
}
