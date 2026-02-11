package sso

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin wires the SSO service and registers routes.
type Plugin struct {
	db            *bun.DB
	service       *Service
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// Config holds the SSO plugin configuration.
type Config struct {
	// Protocol enablement
	AllowSAML bool `json:"allowSAML"`
	AllowOIDC bool `json:"allowOIDC"`

	// JIT (Just-in-Time) user provisioning
	AutoProvision    bool   `json:"autoProvision"`    // Automatically create users on first SSO login
	UpdateAttributes bool   `json:"updateAttributes"` // Update existing user attributes from SSO
	DefaultRole      string `json:"defaultRole"`      // Default role for provisioned users (e.g., "member")

	// Attribute mapping from user fields to SSO attribute names
	// Example: {"email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"}
	AttributeMapping map[string]string `json:"attributeMapping"`

	// SAML configuration
	SAMLMetadataURL   string `json:"samlMetadataURL"`
	SAMLACS           string `json:"samlACS"`           // Assertion Consumer Service URL
	RequireEncryption bool   `json:"requireEncryption"` // Require encrypted SAML assertions

	// OIDC configuration
	OIDCRedirectURL string `json:"oidcRedirectURL"`
}

// DefaultConfig returns the default SSO plugin configuration.
func DefaultConfig() Config {
	return Config{
		AllowSAML:         true,
		AllowOIDC:         true,
		AutoProvision:     true,
		UpdateAttributes:  true,
		DefaultRole:       "member",
		AttributeMapping:  make(map[string]string),
		SAMLMetadataURL:   "",
		SAMLACS:           "",
		RequireEncryption: false,
		OIDCRedirectURL:   "",
	}
}

// PluginOption is a functional option for configuring the SSO plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithAllowSAML sets whether SAML is enabled.
func WithAllowSAML(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowSAML = allow
	}
}

// WithAllowOIDC sets whether OIDC is enabled.
func WithAllowOIDC(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowOIDC = allow
	}
}

// WithSAMLMetadataURL sets the SAML metadata URL.
func WithSAMLMetadataURL(url string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SAMLMetadataURL = url
	}
}

// WithSAMLACS sets the SAML assertion consumer service URL.
func WithSAMLACS(acs string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SAMLACS = acs
	}
}

// WithOIDCRedirectURL sets the OIDC redirect URL.
func WithOIDCRedirectURL(url string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.OIDCRedirectURL = url
	}
}

// WithRequireEncryption sets whether encrypted assertions are required.
func WithRequireEncryption(require bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireEncryption = require
	}
}

// WithAutoProvision sets whether auto-provisioning is enabled.
func WithAutoProvision(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoProvision = enable
	}
}

// NewPlugin creates a new SSO plugin instance with optional configuration.
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

func (p *Plugin) ID() string { return "sso" }

// Init accepts auth instance with GetDB method.
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return errs.InternalServerErrorWithMessage("sso plugin requires auth instance")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return errs.InternalServerErrorWithMessage("database not available for sso plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return errs.InternalServerErrorWithMessage("forge app not available for sso plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "sso"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.sso", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind SSO config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models
	p.db.RegisterModel((*schema.SSOProvider)(nil))

	// Get user and session services from DI container for JIT provisioning
	container := forgeApp.Container()
	if container == nil {
		return errs.InternalServerErrorWithMessage("DI container not available for sso plugin")
	}

	// Resolve user service from container
	userSvcRaw, err := container.Resolve(authsome.ServiceUser)
	if err != nil {
		return fmt.Errorf("failed to resolve user service: %w", err)
	}

	userSvc, ok := userSvcRaw.(user.ServiceInterface)
	if !ok {
		return errs.InternalServerErrorWithMessage("user service has invalid type")
	}

	// Resolve session service from container
	sessionSvcRaw, err := container.Resolve(authsome.ServiceSession)
	if err != nil {
		return fmt.Errorf("failed to resolve session service: %w", err)
	}

	sessionSvc, ok := sessionSvcRaw.(session.ServiceInterface)
	if !ok {
		return errs.InternalServerErrorWithMessage("session service has invalid type")
	}

	// Initialize SSO service with all dependencies
	p.service = NewService(
		authInst.Repository().SSOProvider(),
		p.config,
		userSvc,
		sessionSvc,
	)

	p.logger.Info("SSO plugin initialized",
		forge.F("allow_saml", p.config.AllowSAML),
		forge.F("allow_oidc", p.config.AllowOIDC),
		forge.F("auto_provision", p.config.AutoProvision),
		forge.F("update_attributes", p.config.UpdateAttributes))

	return nil
}

// RegisterRoutes mounts SSO endpoints under /api/auth/sso.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the auth basePath, create sso sub-group
	grp := router.Group("/sso")
	h := NewHandlerWithLogger(p.service, p.logger)

	// =============================================================================
	// PROVIDER MANAGEMENT
	// =============================================================================

	if err := grp.POST("/provider/register", h.RegisterProvider,
		forge.WithName("sso.provider.register"),
		forge.WithSummary("Register SSO provider"),
		forge.WithDescription("Admin endpoint to register SAML or OIDC SSO provider with multi-tenant scoping"),
		forge.WithRequestSchema(RegisterProviderRequest{}),
		forge.WithResponseSchema(200, "Provider registered", ProviderRegisteredResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(500, "Internal error", ErrorResponse{}),
		forge.WithTags("SSO", "Admin", "Provider Management"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}

	// =============================================================================
	// SAML ENDPOINTS
	// =============================================================================

	if err := grp.GET("/saml2/sp/metadata", h.SAMLSPMetadata,
		forge.WithName("sso.saml.metadata"),
		forge.WithSummary("SAML SP metadata"),
		forge.WithDescription("Returns SAML Service Provider metadata XML for IdP configuration"),
		forge.WithResponseSchema(200, "Metadata XML", MetadataResponse{}),
		forge.WithTags("SSO", "SAML", "Metadata"),
	); err != nil {
		return err
	}

	if err := grp.POST("/saml2/login/:providerId", h.SAMLLogin,
		forge.WithName("sso.saml.login"),
		forge.WithSummary("Initiate SAML login"),
		forge.WithDescription("Initiates SAML authentication flow by generating AuthnRequest and returning redirect URL"),
		forge.WithRequestSchema(SAMLLoginRequest{}),
		forge.WithResponseSchema(200, "Login URL generated", SAMLLoginResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", ErrorResponse{}),
		forge.WithTags("SSO", "SAML", "Authentication"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}

	if err := grp.POST("/saml2/callback/:providerId", h.SAMLCallback,
		forge.WithName("sso.saml.callback"),
		forge.WithSummary("SAML callback"),
		forge.WithDescription("Handles SAML assertion from IdP, validates it, provisions user (JIT), and creates session"),
		forge.WithResponseSchema(200, "Authentication successful", SSOAuthResponse{}),
		forge.WithResponseSchema(400, "Invalid SAML response", ErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", ErrorResponse{}),
		forge.WithResponseSchema(500, "Internal error", ErrorResponse{}),
		forge.WithTags("SSO", "SAML", "Authentication", "Callback"),
	); err != nil {
		return err
	}

	// =============================================================================
	// OIDC ENDPOINTS
	// =============================================================================

	if err := grp.POST("/oidc/login/:providerId", h.OIDCLogin,
		forge.WithName("sso.oidc.login"),
		forge.WithSummary("Initiate OIDC login"),
		forge.WithDescription("Initiates OIDC authentication flow with PKCE, generates authorization URL"),
		forge.WithRequestSchema(OIDCLoginRequest{}),
		forge.WithResponseSchema(200, "Authorization URL generated", OIDCLoginResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", ErrorResponse{}),
		forge.WithTags("SSO", "OIDC", "Authentication"),
		forge.WithValidation(true),
	); err != nil {
		return err
	}

	if err := grp.GET("/oidc/callback/:providerId", h.OIDCCallback,
		forge.WithName("sso.oidc.callback"),
		forge.WithSummary("OIDC callback"),
		forge.WithDescription("Handles OIDC callback, exchanges code for tokens, provisions user (JIT), and creates session"),
		forge.WithResponseSchema(200, "Authentication successful", SSOAuthResponse{}),
		forge.WithResponseSchema(400, "Invalid OIDC response", ErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", ErrorResponse{}),
		forge.WithResponseSchema(500, "Internal error", ErrorResponse{}),
		forge.WithTags("SSO", "OIDC", "Authentication", "Callback"),
	); err != nil {
		return err
	}

	p.logger.Debug("SSO plugin routes registered",
		forge.F("saml_enabled", p.config.AllowSAML),
		forge.F("oidc_enabled", p.config.AllowOIDC),
		forge.F("auto_provision", p.config.AutoProvision))

	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate creates required tables and indexes for SSO providers.
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create SSO providers table
	_, err := p.db.NewCreateTable().
		Model((*schema.SSOProvider)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create sso_providers table: %w", err)
	}

	// Create index for multi-tenant queries (app_id, environment_id, organization_id)
	_, err = p.db.NewCreateIndex().
		Model((*schema.SSOProvider)(nil)).
		Index("idx_sso_providers_tenant").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create tenant index: %w", err)
	}

	// Create unique constraint on provider_id within tenant scope
	_, err = p.db.NewCreateIndex().
		Model((*schema.SSOProvider)(nil)).
		Index("idx_sso_providers_unique").
		Column("app_id", "environment_id", "organization_id", "provider_id").
		Unique().
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create unique provider constraint: %w", err)
	}

	// Create index for domain-based provider discovery
	_, err = p.db.NewCreateIndex().
		Model((*schema.SSOProvider)(nil)).
		Index("idx_sso_providers_domain").
		Column("domain", "app_id", "environment_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create domain index: %w", err)
	}

	p.logger.Info("SSO plugin migrations completed successfully")

	return nil
}
