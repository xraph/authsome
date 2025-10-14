package username

import (
    "github.com/uptrace/bun"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/auth"
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
        grp := v.Group("/api/auth")
        h := NewHandler(p.service, repo.NewTwoFARepository(p.db))
        grp.POST("/username/signup", h.SignUp)
        grp.POST("/username/signin", h.SignIn)
        return nil
    case *http.ServeMux:
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
func (p *Plugin) RegisterHooks(_ interface{}) error { return nil }

// Migrate placeholder for DB migrations
func (p *Plugin) Migrate() error { return nil }