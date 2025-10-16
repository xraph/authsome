package phone

import (
    "context"
    "net/http"
    "github.com/xraph/authsome/schema"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/core/auth"
    "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/core/audit"
    "github.com/xraph/authsome/core/hooks"
    "github.com/xraph/authsome/core/registry"
    rl "github.com/xraph/authsome/core/ratelimit"
    "github.com/xraph/authsome/storage"
    "time"
    "github.com/xraph/forge"
    "github.com/uptrace/bun"
)

type Plugin struct {
    db      *bun.DB
    service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "phone" }

func (p *Plugin) Init(dep interface{}) error {
    db, ok := dep.(*bun.DB)
    if !ok { return nil }
    p.db = db
    pr := repo.NewPhoneRepository(db)
    userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
        sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
    authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
    auditSvc := audit.NewService(repo.NewAuditRepository(db))
    p.service = NewService(pr, userSvc, authSvc, auditSvc, nil, Config{DevExposeCode: true, AllowImplicitSignup: true})
    return nil
}

func (p *Plugin) RegisterRoutes(router interface{}) error {
    if p.service == nil { return nil }
    switch v := router.(type) {
    case *forge.App:
        // For direct forge.App usage (not from Mount method)
        grp := v.Group("/api/auth")
        rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/phone/send-code": {Window: time.Minute, Max: 5}}})
        h := NewHandler(p.service, rls)
        grp.POST("/phone/send-code", h.SendCode)
        grp.POST("/phone/verify", h.Verify)
        grp.POST("/phone/signin", h.SignIn)
        return nil
    case *forge.Group:
        // Use relative paths - the router is already a group with the correct basePath
        rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/phone/send-code": {Window: time.Minute, Max: 5}}})
        h := NewHandler(p.service, rls)
        v.POST("/phone/send-code", h.SendCode)
        v.POST("/phone/verify", h.Verify)
        v.POST("/phone/signin", h.SignIn)
        return nil
    case *http.ServeMux:
        app := forge.NewApp(v)
        grp := app.Group("/api/auth")
        rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/phone/send-code": {Window: time.Minute, Max: 5}}})
        h := NewHandler(p.service, rls)
        grp.POST("/phone/send-code", h.SendCode)
        grp.POST("/phone/verify", h.Verify)
        grp.POST("/phone/signin", h.SignIn)
        return nil
    default:
        return nil
    }
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
    if p.db == nil { return nil }
    ctx := context.Background()
    _, err := p.db.NewCreateTable().Model((*schema.PhoneVerification)(nil)).IfNotExists().Exec(ctx)
    return err
}