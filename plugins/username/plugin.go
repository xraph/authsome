package username

import (
    "github.com/uptrace/bun"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/auth"
    "github.com/xraph/authsome/core/hooks"
    "github.com/xraph/authsome/core/registry"
    "net/http"
    "github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Username auth
type Plugin struct {
    service *Service
    db *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "username" }

// Init allows wiring dependencies later; noop for initial scaffold
func (p *Plugin) Init(dep interface{}) error {
    if db, ok := dep.(*bun.DB); ok && db != nil {
        p.db = db
        // Construct local core services
        userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
        sessionSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
        authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
        p.service = NewService(userSvc, authSvc)
    }
    return nil
}

// RegisterRoutes registers Username plugin routes
func (p *Plugin) RegisterRoutes(router interface{}) error {
    switch v := router.(type) {
    case *forge.App:
        // For direct forge.App usage (not from Mount method)
        grp := v.Group("/api/auth")
        h := NewHandler(p.service, repo.NewTwoFARepository(p.db))
        grp.POST("/username/signup", h.SignUp)
        grp.POST("/username/signin", h.SignIn)
        return nil
    case *forge.Group:
        // Use relative paths - the router is already a group with the correct basePath
        h := NewHandler(p.service, repo.NewTwoFARepository(p.db))
        v.POST("/username/signup", h.SignUp)
        v.POST("/username/signin", h.SignIn)
        return nil
    case *http.ServeMux:
        // For direct http.ServeMux usage (not from Mount method)
        app := forge.NewApp(v)
        grp := app.Group("/api/auth")
        h := NewHandler(p.service, repo.NewTwoFARepository(p.db))
        grp.POST("/username/signup", h.SignUp)
        grp.POST("/username/signin", h.SignIn)
        return nil
    default:
        return nil
    }
}

// RegisterHooks placeholder
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate placeholder for DB migrations
func (p *Plugin) Migrate() error { return nil }