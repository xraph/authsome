// Package passkey provides WebAuthn/FIDO2 passkey authentication.
//
// ✅ PRODUCTION READY ✅
//
// This plugin provides enterprise-grade WebAuthn/FIDO2 authentication with:
// - Full cryptographic challenge generation and verification
// - Attestation verification during registration
// - Signature verification during authentication
// - Sign count tracking for replay attack prevention
// - Resident key / discoverable credential support
// - App and organization scoping for multi-tenancy
// - Both standalone passwordless and MFA integration modes
//
// See plugins/passkey/README.md for complete documentation and usage examples.
package passkey

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	audit2 "github.com/xraph/authsome/core/audit"
	auth2 "github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

type Plugin struct {
	db            *bun.DB
	service       *Service
	logger        forge.Logger
	config        Config
	defaultConfig Config
	authInst      core.Authsome
}

// PluginOption is a functional option for configuring the passkey plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithRPID sets the Relying Party ID
func WithRPID(rpID string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RPID = rpID
	}
}

// WithRPName sets the Relying Party Name
func WithRPName(rpName string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RPName = rpName
	}
}

// WithTimeout sets the WebAuthn timeout
func WithTimeout(timeout time.Duration) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Timeout = timeout
	}
}

// WithUserVerification sets the user verification requirement
func WithUserVerification(requirement string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.UserVerification = requirement
	}
}

// WithAttestationType sets the attestation conveyance preference
func WithAttestationType(attestation string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AttestationType = attestation
	}
}

// WithRPOrigins sets the allowed origins for WebAuthn
func WithRPOrigins(origins []string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RPOrigins = origins
	}
}

// WithRequireResidentKey sets whether resident keys are required
func WithRequireResidentKey(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireResidentKey = required
	}
}

// WithAuthenticatorAttachment sets the authenticator attachment preference
func WithAuthenticatorAttachment(attachment string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AuthenticatorAttachment = attachment
	}
}

// WithChallengeStorage sets the challenge storage backend
func WithChallengeStorage(storage string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.ChallengeStorage = storage
	}
}

// NewPlugin creates a new passkey plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			RPID:                    "localhost",
			RPName:                  "AuthSome",
			RPOrigins:               []string{"http://localhost", "https://localhost"},
			Timeout:                 60000, // 60 seconds
			UserVerification:        "preferred",
			AttestationType:         "none",
			RequireResidentKey:      false,
			AuthenticatorAttachment: "", // allow both platform and cross-platform
			ChallengeStorage:        "memory",
		},
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Plugin) ID() string { return "passkey" }

func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("passkey plugin requires auth instance")
	}

	// Store auth instance for middleware access
	p.authInst = authInst

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for passkey plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for passkey plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "passkey"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.passkey", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind passkey config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models
	p.db.RegisterModel((*schema.Passkey)(nil))

	// Wire services
	userSvc := user.NewService(repo.NewUserRepository(p.db), user.Config{}, nil)
	sessionSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil)
	authSvc := auth2.NewService(userSvc, sessionSvc, auth2.Config{})
	auditSvc := audit2.NewService(repo.NewAuditRepository(p.db))

	// Create passkey service with WebAuthn support
	service, err := NewService(p.db, userSvc, authSvc, auditSvc, p.config)
	if err != nil {
		return fmt.Errorf("failed to create passkey service: %w", err)
	}
	p.service = service

	p.logger.Info("passkey plugin initialized (PRODUCTION READY)",
		forge.F("rp_id", p.config.RPID),
		forge.F("rp_name", p.config.RPName),
		forge.F("timeout", p.config.Timeout),
		forge.F("user_verification", p.config.UserVerification))

	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the auth basePath, create passkey sub-group
	grp := router.Group("/passkey")
	h := NewHandler(p.service)

	// Get authentication middleware for API key validation
	authMw := p.authInst.AuthMiddleware()

	// Wrap handler with middleware if available
	wrapHandler := func(handler func(forge.Context) error) func(forge.Context) error {
		if authMw != nil {
			return authMw(handler)
		}
		return handler
	}

	// Registration endpoints
	grp.POST("/register/begin", wrapHandler(h.BeginRegister),
		forge.WithName("passkey.register.begin"),
		forge.WithSummary("Begin passkey registration"),
		forge.WithDescription("Initiates WebAuthn/FIDO2 passkey registration with cryptographic challenge. Supports platform authenticators (Touch ID, Windows Hello) and cross-platform authenticators (YubiKey, etc.)"),
		forge.WithRequestSchema(BeginRegisterRequest{}),
		forge.WithResponseSchema(200, "Registration options", BeginRegisterResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Registration"),
		forge.WithValidation(true),
	)
	grp.POST("/register/finish", wrapHandler(h.FinishRegister),
		forge.WithName("passkey.register.finish"),
		forge.WithSummary("Finish passkey registration"),
		forge.WithDescription("Completes WebAuthn/FIDO2 passkey registration with attestation verification. Stores credential with cryptographic public key"),
		forge.WithRequestSchema(FinishRegisterRequest{}),
		forge.WithResponseSchema(200, "Passkey registered", FinishRegisterResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Registration"),
		forge.WithValidation(true),
	)

	// Authentication endpoints
	grp.POST("/login/begin", wrapHandler(h.BeginLogin),
		forge.WithName("passkey.login.begin"),
		forge.WithSummary("Begin passkey login"),
		forge.WithDescription("Initiates WebAuthn/FIDO2 passkey authentication. Supports both user-specific and discoverable (usernameless) credentials"),
		forge.WithRequestSchema(BeginLoginRequest{}),
		forge.WithResponseSchema(200, "Login options", BeginLoginResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Authentication"),
		forge.WithValidation(true),
	)
	grp.POST("/login/finish", wrapHandler(h.FinishLogin),
		forge.WithName("passkey.login.finish"),
		forge.WithSummary("Finish passkey login"),
		forge.WithDescription("Completes WebAuthn/FIDO2 passkey authentication with signature verification and creates user session. Validates sign count for replay attack detection"),
		forge.WithRequestSchema(FinishLoginRequest{}),
		forge.WithResponseSchema(200, "Login successful", LoginResponse{}),
		forge.WithResponseSchema(401, "Authentication failed", ErrorResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Authentication"),
		forge.WithValidation(true),
	)

	// Management endpoints
	grp.GET("/list", wrapHandler(h.List),
		forge.WithName("passkey.list"),
		forge.WithSummary("List passkeys"),
		forge.WithDescription("Lists all registered passkeys for a user with metadata including name, type, last used, and sign count"),
		forge.WithRequestSchema(ListPasskeysRequest{}),
		forge.WithResponseSchema(200, "Passkeys retrieved", ListPasskeysResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("Passkey", "Management"),
		forge.WithValidation(true),
	)
	grp.PUT("/:id", wrapHandler(h.Update),
		forge.WithName("passkey.update"),
		forge.WithSummary("Update passkey"),
		forge.WithDescription("Updates a passkey's metadata (currently only name)"),
		forge.WithRequestSchema(UpdatePasskeyRequest{}),
		forge.WithResponseSchema(200, "Passkey updated", UpdatePasskeyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "Passkey not found", ErrorResponse{}),
		forge.WithTags("Passkey", "Management"),
		forge.WithValidation(true),
	)
	grp.DELETE("/:id", wrapHandler(h.Delete),
		forge.WithName("passkey.delete"),
		forge.WithSummary("Delete passkey"),
		forge.WithDescription("Deletes a registered passkey by ID. Scoped to app and organization context"),
		forge.WithRequestSchema(DeletePasskeyRequest{}),
		forge.WithResponseSchema(200, "Passkey deleted", StatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "Passkey not found", ErrorResponse{}),
		forge.WithTags("Passkey", "Management"),
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
	_, err := p.db.NewCreateTable().Model((*schema.Passkey)(nil)).IfNotExists().Exec(ctx)
	return err
}

// Legacy response type aliases for backward compatibility
// Use the proper types defined in response_types.go instead
type PasskeyErrorResponse = ErrorResponse
type PasskeyStatusResponse = StatusResponse
type PasskeyRegistrationOptionsResponse = BeginRegisterResponse
type PasskeyLoginOptionsResponse = BeginLoginResponse
type PasskeyLoginResponse = LoginResponse
type PasskeyListResponse = ListPasskeysResponse
