package sso

import (
	"context"

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

// Init accepts *bun.DB; current service uses in-memory provider registry
func (p *Plugin) Init(dep interface{}) error {
	if db, ok := dep.(*bun.DB); ok && db != nil {
		p.db = db
		p.service = NewService(repo.NewSSOProviderRepository(db))
	}
	return nil
}

// RegisterRoutes mounts SSO endpoints under /api/auth/sso
func (p *Plugin) RegisterRoutes(router interface{}) error {
	if p.service == nil {
		return nil
	}
	switch v := router.(type) {
	case *forge.App:
		// For direct forge.App usage (not from Mount method)
		grp := v.Group("/api/auth/sso")
		h := NewHandler(p.service)
		grp.POST("/provider/register", h.RegisterProvider)
		grp.GET("/saml2/sp/metadata", h.SAMLSPMetadata)
		grp.GET("/saml2/login/{providerId}", h.SAMLLogin)
		grp.POST("/saml2/callback/{providerId}", h.SAMLCallback)
		grp.GET("/oidc/callback/{providerId}", h.OIDCCallback)
		return nil
	case *forge.Group:
		// Use relative paths - the router is already a group with the correct basePath
		grp := v.Group("/sso")
		h := NewHandler(p.service)
		grp.POST("/provider/register", h.RegisterProvider)
		grp.GET("/saml2/sp/metadata", h.SAMLSPMetadata)
		grp.GET("/saml2/login/{providerId}", h.SAMLLogin)
		grp.POST("/saml2/callback/{providerId}", h.SAMLCallback)
		grp.GET("/oidc/callback/{providerId}", h.OIDCCallback)
		return nil
	default:
		return nil
	}
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
