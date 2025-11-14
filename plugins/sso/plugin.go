package sso

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin wires the SSO service and registers routes
type Plugin struct {
	db            *bun.DB
	service       *Service
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// Config holds the SSO plugin configuration
type Config struct {
	// AllowSAML enables SAML authentication
	AllowSAML bool `json:"allowSAML"`
	// AllowOIDC enables OIDC authentication
	AllowOIDC bool `json:"allowOIDC"`
	// SAMLMetadataURL is the SP metadata URL
	SAMLMetadataURL string `json:"samlMetadataURL"`
	// SAMLACS is the assertion consumer service URL
	SAMLACS string `json:"samlACS"`
	// OIDCRedirectURL is the OIDC redirect URL
	OIDCRedirectURL string `json:"oidcRedirectURL"`
	// RequireEncryption requires encrypted SAML assertions
	RequireEncryption bool `json:"requireEncryption"`
	// AutoProvision automatically provisions users from SSO
	AutoProvision bool `json:"autoProvision"`
}

// DefaultConfig returns the default SSO plugin configuration
func DefaultConfig() Config {
	return Config{
		AllowSAML:         true,
		AllowOIDC:         true,
		SAMLMetadataURL:   "",
		SAMLACS:           "",
		OIDCRedirectURL:   "",
		RequireEncryption: false,
		AutoProvision:     true,
	}
}

// PluginOption is a functional option for configuring the SSO plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithAllowSAML sets whether SAML is enabled
func WithAllowSAML(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowSAML = allow
	}
}

// WithAllowOIDC sets whether OIDC is enabled
func WithAllowOIDC(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowOIDC = allow
	}
}

// WithSAMLMetadataURL sets the SAML metadata URL
func WithSAMLMetadataURL(url string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SAMLMetadataURL = url
	}
}

// WithSAMLACS sets the SAML assertion consumer service URL
func WithSAMLACS(acs string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SAMLACS = acs
	}
}

// WithOIDCRedirectURL sets the OIDC redirect URL
func WithOIDCRedirectURL(url string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.OIDCRedirectURL = url
	}
}

// WithRequireEncryption sets whether encrypted assertions are required
func WithRequireEncryption(require bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireEncryption = require
	}
}

// WithAutoProvision sets whether auto-provisioning is enabled
func WithAutoProvision(enable bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoProvision = enable
	}
}

// NewPlugin creates a new SSO plugin instance with optional configuration
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

// Init accepts auth instance with GetDB method
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("sso plugin requires auth instance")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for sso plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for sso plugin")
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

	p.service = NewService(authInst.Repository().SSOProvider(), p.config)

	p.logger.Info("SSO plugin initialized",
		forge.F("allow_saml", p.config.AllowSAML),
		forge.F("allow_oidc", p.config.AllowOIDC),
		forge.F("auto_provision", p.config.AutoProvision))

	return nil
}

// RegisterRoutes mounts SSO endpoints under /api/auth/sso
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the auth basePath, create sso sub-group
	grp := router.Group("/sso")
	h := NewHandler(p.service)
	grp.POST("/provider/register", h.RegisterProvider,
		forge.WithName("sso.provider.register"),
		forge.WithSummary("Register SSO provider"),
		forge.WithDescription("Registers a new SSO provider (SAML or OIDC) with configuration for authentication"),
		forge.WithResponseSchema(200, "Provider registered", SSOProviderResponse{}),
		forge.WithResponseSchema(400, "Invalid request", SSOErrorResponse{}),
		forge.WithTags("SSO", "Providers"),
		forge.WithValidation(true),
	)
	grp.GET("/saml2/sp/metadata", h.SAMLSPMetadata,
		forge.WithName("sso.saml2.sp.metadata"),
		forge.WithSummary("SAML2 Service Provider metadata"),
		forge.WithDescription("Returns SAML2 Service Provider metadata XML for IdP configuration"),
		forge.WithResponseSchema(200, "SAML metadata", SSOSAMLMetadataResponse{}),
		forge.WithTags("SSO", "SAML2"),
	)
	grp.GET("/saml2/login/{providerId}", h.SAMLLogin,
		forge.WithName("sso.saml2.login"),
		forge.WithSummary("Initiate SAML2 login"),
		forge.WithDescription("Initiates SAML2 authentication flow by redirecting to Identity Provider"),
		forge.WithResponseSchema(302, "Redirect to IdP", nil),
		forge.WithResponseSchema(400, "Invalid request", SSOErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", SSOErrorResponse{}),
		forge.WithTags("SSO", "SAML2", "Authentication"),
	)
	grp.POST("/saml2/callback/{providerId}", h.SAMLCallback,
		forge.WithName("sso.saml2.callback"),
		forge.WithSummary("SAML2 callback"),
		forge.WithDescription("Handles SAML2 authentication response from Identity Provider and creates user session"),
		forge.WithResponseSchema(200, "SAML callback processed", SSOSAMLCallbackResponse{}),
		forge.WithResponseSchema(400, "Invalid SAML response", SSOErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", SSOErrorResponse{}),
		forge.WithTags("SSO", "SAML2", "Callback"),
	)
	grp.GET("/oidc/callback/{providerId}", h.OIDCCallback,
		forge.WithName("sso.oidc.callback"),
		forge.WithSummary("OIDC callback"),
		forge.WithDescription("Handles OIDC authentication callback from Identity Provider and creates user session"),
		forge.WithResponseSchema(302, "Redirect after authentication", nil),
		forge.WithResponseSchema(400, "Invalid OIDC response", SSOErrorResponse{}),
		forge.WithResponseSchema(404, "Provider not found", SSOErrorResponse{}),
		forge.WithTags("SSO", "OIDC", "Callback"),
	)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate creates required tables for SSO providers
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.SSOProvider)(nil)).IfNotExists().Exec(ctx)
	return err
}

// Response types for SSO routes
type SSOErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type SSOProviderResponse struct {
	Status     string `json:"status" example:"registered"`
	ProviderID string `json:"providerId" example:"provider_123"`
}

type SSOSAMLMetadataResponse struct {
	Metadata string `json:"metadata" example:"<?xml version=\"1.0\"?>..."`
}

type SSOSAMLCallbackResponse struct {
	Status     string                 `json:"status" example:"saml_callback_ok"`
	Subject    string                 `json:"subject" example:"user@example.com"`
	Issuer     string                 `json:"issuer" example:"https://idp.example.com"`
	Attributes map[string]interface{} `json:"attributes"`
	ProviderID string                 `json:"providerId" example:"provider_123"`
}
