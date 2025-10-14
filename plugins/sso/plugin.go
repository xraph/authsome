package sso

import (
    "net/http"
    "github.com/uptrace/bun"
    repo "github.com/xraph/authsome/repository"
    "context"
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
    if p.service == nil { return nil }
    switch v := router.(type) {
    case *forge.App:
        grp := v.Group("/api/auth/sso")
        h := NewHandler(p.service)
        grp.POST("/provider/register", h.RegisterProvider)
        grp.GET("/saml2/sp/metadata", h.SAMLSPMetadata)
        grp.GET("/saml2/login/{providerId}", h.SAMLLogin)
        grp.POST("/saml2/callback/{providerId}", h.SAMLCallback)
        grp.GET("/oidc/callback/{providerId}", h.OIDCCallback)
        return nil
    case *http.ServeMux:
        app := forge.NewApp(v)
        grp := app.Group("/api/auth/sso")
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

func (p *Plugin) RegisterHooks(_ interface{}) error { return nil }

// Migrate creates required tables for SSO providers
func (p *Plugin) Migrate() error {
    if p.db == nil { return nil }
    ctx := context.Background()
    _, err := p.db.NewCreateTable().Model((*schema.SSOProvider)(nil)).IfNotExists().Exec(ctx)
    return err
}