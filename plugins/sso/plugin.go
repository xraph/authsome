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
	grp.POST("/provider/register", h.RegisterProvider)
	grp.GET("/saml2/sp/metadata", h.SAMLSPMetadata)
	grp.GET("/saml2/login/{providerId}", h.SAMLLogin)
	grp.POST("/saml2/callback/{providerId}", h.SAMLCallback)
	grp.GET("/oidc/callback/{providerId}", h.OIDCCallback)
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
