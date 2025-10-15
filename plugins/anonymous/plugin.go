package anonymous

import (
    "net/http"
    "github.com/uptrace/bun"
    "github.com/xraph/forge"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/hooks"
    "github.com/xraph/authsome/core/registry"
    repo "github.com/xraph/authsome/repository"
)

type Plugin struct{
    service *Service
    db *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "anonymous" }

// Init wires up repository and services for anonymous signin
func (p *Plugin) Init(dep interface{}) error {
    if db, ok := dep.(*bun.DB); ok && db != nil {
        p.db = db
        users := repo.NewUserRepository(db)
        sessionSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
        p.service = NewService(users, sessionSvc)
    }
    return nil
}

// RegisterRoutes registers Anonymous plugin routes
func (p *Plugin) RegisterRoutes(router interface{}) error {
    switch v := router.(type) {
    case *forge.App:
        grp := v.Group("/api/auth")
        h := NewHandler(p.service)
        grp.POST("/anonymous/signin", h.SignIn)
        grp.POST("/anonymous/link", h.Link)
        return nil
    case *http.ServeMux:
        app := forge.NewApp(v)
        grp := app.Group("/api/auth")
        h := NewHandler(p.service)
        grp.POST("/anonymous/signin", h.SignIn)
        grp.POST("/anonymous/link", h.Link)
        return nil
    default:
        return nil
    }
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error { return nil }