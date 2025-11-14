// Package passkey provides WebAuthn/FIDO2 passkey authentication.
//
// ⚠️ EXPERIMENTAL / BETA STATUS ⚠️
//
// This plugin is currently in experimental/beta status. The WebAuthn implementation
// is a basic stub and NOT production-ready. Critical cryptographic operations including
// challenge generation, attestation verification, and signature validation are not
// properly implemented.
//
// DO NOT USE IN PRODUCTION without completing the WebAuthn implementation.
// See plugins/passkey/README.md for details and roadmap.
package passkey

import (
	"context"
	"fmt"

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
func WithTimeout(timeout int) PluginOption {
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

// NewPlugin creates a new passkey plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: Config{
			RPID:             "localhost",
			RPName:           "AuthSome",
			Timeout:          60000,
			UserVerification: "preferred",
			AttestationType:  "none",
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
	sessSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil)
	authSvc := auth2.NewService(userSvc, sessSvc, auth2.Config{})
	auditSvc := audit2.NewService(repo.NewAuditRepository(p.db))
	p.service = NewService(p.db, userSvc, authSvc, auditSvc, p.config)

	p.logger.Info("passkey plugin initialized (BETA)",
		forge.F("rp_id", p.config.RPID),
		forge.F("rp_name", p.config.RPName),
		forge.F("timeout", p.config.Timeout))

	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the auth basePath, create passkey sub-group
	grp := router.Group("/passkey")
	h := NewHandler(p.service)
	grp.POST("/register/begin", h.BeginRegister,
		forge.WithName("passkey.register.begin"),
		forge.WithSummary("Begin passkey registration"),
		forge.WithDescription("Initiates WebAuthn/FIDO2 passkey registration. Returns challenge and credential creation options"),
		forge.WithResponseSchema(200, "Registration options", PasskeyRegistrationOptionsResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PasskeyErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Registration"),
		forge.WithValidation(true),
	)
	grp.POST("/register/finish", h.FinishRegister,
		forge.WithName("passkey.register.finish"),
		forge.WithSummary("Finish passkey registration"),
		forge.WithDescription("Completes WebAuthn/FIDO2 passkey registration with credential attestation"),
		forge.WithResponseSchema(200, "Passkey registered", PasskeyStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PasskeyErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Registration"),
		forge.WithValidation(true),
	)
	grp.POST("/login/begin", h.BeginLogin,
		forge.WithName("passkey.login.begin"),
		forge.WithSummary("Begin passkey login"),
		forge.WithDescription("Initiates WebAuthn/FIDO2 passkey authentication. Returns challenge and credential request options"),
		forge.WithResponseSchema(200, "Login options", PasskeyLoginOptionsResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PasskeyErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Authentication"),
		forge.WithValidation(true),
	)
	grp.POST("/login/finish", h.FinishLogin,
		forge.WithName("passkey.login.finish"),
		forge.WithSummary("Finish passkey login"),
		forge.WithDescription("Completes WebAuthn/FIDO2 passkey authentication with credential assertion and creates user session"),
		forge.WithResponseSchema(200, "Login successful", PasskeyLoginResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PasskeyErrorResponse{}),
		forge.WithTags("Passkey", "WebAuthn", "Authentication"),
		forge.WithValidation(true),
	)
	grp.GET("/list", h.List,
		forge.WithName("passkey.list"),
		forge.WithSummary("List passkeys"),
		forge.WithDescription("Lists all registered passkeys for a user"),
		forge.WithResponseSchema(200, "Passkeys retrieved", PasskeyListResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PasskeyErrorResponse{}),
		forge.WithTags("Passkey", "Management"),
	)
	grp.DELETE("/:id", h.Delete,
		forge.WithName("passkey.delete"),
		forge.WithSummary("Delete passkey"),
		forge.WithDescription("Deletes a registered passkey by ID"),
		forge.WithResponseSchema(200, "Passkey deleted", PasskeyStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PasskeyErrorResponse{}),
		forge.WithResponseSchema(404, "Passkey not found", PasskeyErrorResponse{}),
		forge.WithTags("Passkey", "Management"),
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

// Response types for passkey routes
type PasskeyErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type PasskeyStatusResponse struct {
	Status string `json:"status" example:"registered"`
}

type PasskeyRegistrationOptionsResponse struct {
	Options interface{} `json:"options"`
}

type PasskeyLoginOptionsResponse struct {
	Options interface{} `json:"options"`
}

type PasskeyLoginResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
}

type PasskeyListResponse struct {
	Passkeys []interface{} `json:"passkeys"`
}
