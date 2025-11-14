package bearer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forge"
)

// Plugin implements bearer token authentication middleware
type Plugin struct {
	sessionSvc    *session.Service
	userSvc       *user.Service
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// Config holds the bearer plugin configuration
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

// DefaultConfig returns the default bearer plugin configuration
func DefaultConfig() Config {
	return Config{
		TokenPrefix:    "Bearer",
		ValidateIssuer: false,
		RequireScopes:  []string{},
		CaseSensitive:  false,
	}
}

// PluginOption is a functional option for configuring the bearer plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithTokenPrefix sets the token prefix
func WithTokenPrefix(prefix string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TokenPrefix = prefix
	}
}

// WithValidateIssuer sets whether to validate the issuer
func WithValidateIssuer(validate bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ValidateIssuer = validate
	}
}

// WithRequireScopes sets the required scopes
func WithRequireScopes(scopes []string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireScopes = scopes
	}
}

// NewPlugin creates a new bearer token plugin with optional configuration
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

// Init initializes the bearer plugin
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("bearer plugin requires auth instance")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for bearer plugin")
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
	if sessSvc := serviceRegistry.Get("session"); sessSvc != nil {
		if svc, ok := sessSvc.(*session.Service); ok {
			p.sessionSvc = svc
		}
	}
	if userSvc := serviceRegistry.Get("user"); userSvc != nil {
		if svc, ok := userSvc.(*user.Service); ok {
			p.userSvc = svc
		}
	}

	if p.sessionSvc == nil || p.userSvc == nil {
		return fmt.Errorf("bearer plugin requires session and user services")
	}

	p.logger.Info("bearer plugin initialized",
		forge.F("token_prefix", p.config.TokenPrefix),
		forge.F("validate_issuer", p.config.ValidateIssuer))

	return nil
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "bearer"
}

// RegisterServiceDecorators registers service decorators (no-op for bearer)
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	return nil
}

// AuthenticateHandler returns a handler function that can be used as middleware
func (p *Plugin) AuthenticateHandler(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Extract bearer token from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return next(c)
		}

		// Check if it's a bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return next(c)
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return next(c)
		}

		// Validate the token using session service
		sess, err := p.sessionSvc.FindByToken(c.Request().Context(), token)
		if err != nil {
			return next(c)
		}

		// Check if session is valid and not expired
		if sess == nil || time.Now().After(sess.ExpiresAt) {
			return next(c)
		}

		// Get user information
		user, err := p.userSvc.FindByID(c.Request().Context(), sess.UserID)
		if err != nil {
			return next(c)
		}

		// Store user and session in request context
		ctx := context.WithValue(c.Request().Context(), "user", user)
		ctx = context.WithValue(ctx, "session", sess)
		ctx = context.WithValue(ctx, "authenticated", true)

		// Update request with new context
		*c.Request() = *c.Request().WithContext(ctx)

		return next(c)
	}
}

// RequireAuthHandler returns a handler that requires authentication
func (p *Plugin) RequireAuthHandler(next func(forge.Context) error) func(forge.Context) error {
	return func(c forge.Context) error {
		// Check if user is authenticated
		if c.Request().Context().Value("authenticated") != true {
			return c.JSON(401, map[string]string{
				"error": "Authentication required",
			})
		}
		return next(c)
	}
}

// GetUser extracts the authenticated user from context
func GetUser(c forge.Context) *user.User {
	if u := c.Request().Context().Value("user"); u != nil {
		if user, ok := u.(*user.User); ok {
			return user
		}
	}
	return nil
}

// GetSession extracts the session from context
func GetSession(c forge.Context) *session.Session {
	if s := c.Request().Context().Value("session"); s != nil {
		if sess, ok := s.(*session.Session); ok {
			return sess
		}
	}
	return nil
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c forge.Context) bool {
	return c.Request().Context().Value("authenticated") == true
}
