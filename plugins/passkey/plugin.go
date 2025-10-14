package passkey

import (
    "context"
    "net/http"
    "github.com/uptrace/bun"
    "github.com/xraph/authsome/core/audit"
    "github.com/xraph/authsome/core/auth"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/user"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/schema"
    "github.com/xraph/forge"
)

type Plugin struct{
    db      *bun.DB
    service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "passkey" }

func (p *Plugin) Init(dep interface{}) error {
    db, ok := dep.(*bun.DB)
    if !ok { return nil }
    p.db = db
    userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
        sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
    authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
    auditSvc := audit.NewService(repo.NewAuditRepository(db))
    p.service = NewService(db, userSvc, authSvc, auditSvc, Config{RPID: "localhost", RPName: "Authsome"})
    return nil
}

func (p *Plugin) RegisterRoutes(router interface{}) error {
    if p.service == nil { return nil }
    switch v := router.(type) {
    case *forge.App:
        grp := v.Group("/api/auth/passkey")
        h := NewHandler(p.service)
        grp.POST("/register/begin", h.BeginRegister)
        grp.POST("/register/finish", h.FinishRegister)
        grp.POST("/login/begin", h.BeginLogin)
        grp.POST("/login/finish", h.FinishLogin)
        grp.GET("/list", h.List)
        grp.POST("/delete/{id}", h.Delete)
        return nil
    case *http.ServeMux:
        app := forge.NewApp(v)
        grp := app.Group("/api/auth/passkey")
        h := NewHandler(p.service)
        grp.POST("/register/begin", h.BeginRegister)
        grp.POST("/register/finish", h.FinishRegister)
        grp.POST("/login/begin", h.BeginLogin)
        grp.POST("/login/finish", h.FinishLogin)
        grp.GET("/list", h.List)
        grp.POST("/delete/{id}", h.Delete)
        return nil
    default:
        return nil
    }
}

func (p *Plugin) RegisterHooks(_ interface{}) error { return nil }

func (p *Plugin) Migrate() error {
    if p.db == nil { return nil }
    ctx := context.Background()
    _, err := p.db.NewCreateTable().Model((*schema.Passkey)(nil)).IfNotExists().Exec(ctx)
    return err
}