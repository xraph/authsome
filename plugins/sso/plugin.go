package sso

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin wires the SSO service and registers routes
type Plugin struct {
	db      *bun.DB
	service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "sso" }

// Init accepts auth instance with GetDB method
func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	auth, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("sso plugin requires auth instance with GetDB method")
	}

	db := auth.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for sso plugin")
	}

	p.db = db
	p.service = NewService(repo.NewSSOProviderRepository(db))
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
	Status    string `json:"status" example:"registered"`
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
