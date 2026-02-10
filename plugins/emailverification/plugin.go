package emailverification

import (
	"context"
	"net/http"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the email verification plugin.
type Plugin struct {
	service       *Service
	notifAdapter  *notificationPlugin.Adapter
	db            *bun.DB
	logger        forge.Logger
	config        Config
	defaultConfig Config
	authInst      core.Authsome
}

// Config holds the email verification plugin configuration.
type Config struct {
	// TokenLength is the length of the verification token in bytes
	TokenLength int `json:"tokenLength"`
	// ExpiryHours is the token expiry time in hours
	ExpiryHours int `json:"expiryHours"`
	// MaxResendPerHour is the maximum resend requests per hour per user
	MaxResendPerHour int `json:"maxResendPerHour"`
	// AutoSendOnSignup automatically sends verification email after signup
	AutoSendOnSignup bool `json:"autoSendOnSignup"`
	// AutoLoginAfterVerify creates a session after successful verification
	AutoLoginAfterVerify bool `json:"autoLoginAfterVerify"`
	// VerificationURL is the frontend URL template for verification links
	VerificationURL string `json:"verificationURL"`
	// DevExposeToken exposes token in response for development/testing
	DevExposeToken bool `json:"devExposeToken"`
}

// DefaultConfig returns the default email verification plugin configuration.
func DefaultConfig() Config {
	return Config{
		TokenLength:          32,
		ExpiryHours:          24,
		MaxResendPerHour:     3,
		AutoSendOnSignup:     true,
		AutoLoginAfterVerify: true,
		VerificationURL:      "",
		DevExposeToken:       false,
	}
}

// PluginOption is a functional option for configuring the email verification plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithTokenLength sets the verification token length.
func WithTokenLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.TokenLength = length
	}
}

// WithExpiryHours sets the token expiry time in hours.
func WithExpiryHours(hours int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ExpiryHours = hours
	}
}

// WithMaxResendPerHour sets the maximum resend requests per hour.
func WithMaxResendPerHour(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.MaxResendPerHour = max
	}
}

// WithAutoSendOnSignup sets whether to automatically send verification on signup.
func WithAutoSendOnSignup(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoSendOnSignup = enable
	}
}

// WithAutoLoginAfterVerify sets whether to auto-login after verification.
func WithAutoLoginAfterVerify(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoLoginAfterVerify = enable
	}
}

// WithVerificationURL sets the frontend verification URL.
func WithVerificationURL(url string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.VerificationURL = url
	}
}

// NewPlugin creates a new email verification plugin instance.
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Plugin) ID() string { return "emailverification" }

func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.New("EMAILVERIFICATION_PLUGIN_REQUIRES_AUTH_INSTANCE", "Email verification plugin requires auth instance", http.StatusInternalServerError)
	}

	// Store auth instance
	p.authInst = authInst

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return errs.New("DATABASE_NOT_AVAILABLE", "Database not available for email verification plugin", http.StatusInternalServerError)
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.New("FORGE_APP_NOT_AVAILABLE", "Forge app not available for email verification plugin", http.StatusInternalServerError)
	}

	// Initialize logger
	p.logger = authInst.Logger().With(forge.F("plugin", "emailverification"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.emailverification", &p.config, p.defaultConfig); err != nil {
		p.logger.Warn("failed to bind email verification config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models (Verification table already exists)
	p.db.RegisterModel((*schema.Verification)(nil))

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

	// Wire repository and services
	verifRepo := NewVerificationRepository(p.db)
	userSvc := user.NewService(repo.NewUserRepository(p.db), user.Config{}, nil, nil)
	sessionSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil, nil)

	p.service = NewService(verifRepo, userSvc, sessionSvc, p.notifAdapter, p.config, p.logger)

	p.logger.Info("email verification plugin initialized",
		forge.F("token_length", p.config.TokenLength),
		forge.F("expiry_hours", p.config.ExpiryHours),
		forge.F("auto_send_on_signup", p.config.AutoSendOnSignup))

	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}

	h := NewHandler(p.service, p.authInst)

	// Get authentication middleware for protected routes
	authMw := p.authInst.AuthMiddleware()

	// Wrap handler with middleware if available
	wrapHandler := func(handler func(forge.Context) error) func(forge.Context) error {
		if authMw != nil {
			return authMw(handler)
		}

		return handler
	}

	// Public routes (no auth required)
	if err := router.GET("/email-verification/verify", h.Verify,
		forge.WithName("emailverification.verify"),
		forge.WithSummary("Verify email token"),
		forge.WithDescription("Verifies email address using token from verification link. Optionally creates a session for auto-login."),
		forge.WithRequestSchema(VerifyRequest{}),
		forge.WithResponseSchema(200, "Email verified", VerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request or already verified", ErrorResponse{}),
		forge.WithResponseSchema(404, "Token not found", ErrorResponse{}),
		forge.WithResponseSchema(410, "Token expired or used", ErrorResponse{}),
		forge.WithTags("EmailVerification", "Authentication"),
	
	); err != nil {
		return err
	}

	if err := router.POST("/email-verification/resend", h.Resend,
		forge.WithName("emailverification.resend"),
		forge.WithSummary("Resend verification email"),
		forge.WithDescription("Requests a new verification email to be sent. Rate limited to 3 per hour per user."),
		forge.WithRequestSchema(ResendRequest{}),
		forge.WithResponseSchema(200, "Verification email sent", ResendResponse{}),
		forge.WithResponseSchema(400, "Invalid request or already verified", ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", ErrorResponse{}),
		forge.WithResponseSchema(429, "Rate limit exceeded", ErrorResponse{}),
		forge.WithTags("EmailVerification", "Authentication"),
		forge.WithValidation(true),
	
	); err != nil {
		return err
	}

	if err := router.POST("/email-verification/send", h.Send,
		forge.WithName("emailverification.send"),
		forge.WithSummary("Send verification email"),
		forge.WithDescription("Manually sends a verification email to a user"),
		forge.WithRequestSchema(SendRequest{}),
		forge.WithResponseSchema(200, "Verification email sent", SendResponse{}),
		forge.WithResponseSchema(400, "Invalid request or already verified", ErrorResponse{}),
		forge.WithResponseSchema(404, "User not found", ErrorResponse{}),
		forge.WithTags("EmailVerification", "Authentication"),
		forge.WithValidation(true),
	
	); err != nil {
		return err
	}

	// Authenticated routes
	if err := router.GET("/email-verification/status", wrapHandler(h.Status),
		forge.WithName("emailverification.status"),
		forge.WithSummary("Check verification status"),
		forge.WithDescription("Returns the email verification status for the current authenticated user"),
		forge.WithResponseSchema(200, "Verification status retrieved", StatusResponse{}),
		forge.WithResponseSchema(401, "Not authenticated", ErrorResponse{}),
		forge.WithTags("EmailVerification", "Authentication"),
	
	); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if p.service == nil || hookRegistry == nil {
		return nil
	}

	// Register after sign-up hook to automatically send verification email
	if p.config.AutoSendOnSignup {
		hookRegistry.RegisterAfterSignUp(func(ctx context.Context, resp *responses.AuthResponse) error {
			// Only send if user is not verified
			if resp.User != nil && !resp.User.EmailVerified {
				// Get AuthContext with complete authentication state
				authCtx, ok := contexts.GetAuthContext(ctx)
				if !ok || authCtx == nil {
					p.logger.Warn("auth context not available in after sign up hook")

					return nil // Don't fail the sign-up
				}

				// Use AppID from AuthContext
				appID := authCtx.AppID
				if appID.IsNil() {
					p.logger.Warn("app ID not available in auth context")

					return nil // Don't fail the sign-up
				}

				// Send verification email
				_, err := p.service.SendVerification(ctx, appID, resp.User.ID, resp.User.Email)
				if err != nil {
					p.logger.Error("failed to send verification email in after sign up hook",
						forge.F("error", err.Error()),
						forge.F("user_id", resp.User.ID.String()))
					// Don't fail the sign-up - user can request resend
				}
			}

			return nil
		})

		p.logger.Debug("registered after sign up hook for automatic email verification")
	}

	return nil
}

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	return nil
}

func (p *Plugin) Migrate() error {
	// Verification table already exists in schema
	// No additional migrations needed
	return nil
}
