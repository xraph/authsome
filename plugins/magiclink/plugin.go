package magiclink

import (
	"context"
	"net/http"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

type Plugin struct {
	db      *bun.DB
	service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "magiclink" }

func (p *Plugin) Init(dep interface{}) error {
	db, ok := dep.(*bun.DB)
	if !ok {
		return nil
	}
	p.db = db
	mr := repo.NewMagicLinkRepository(db)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
	// Build full auth service with session
	sessRepo := repo.NewSessionRepository(db)
	sessionSvc := session.NewService(sessRepo, session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	p.service = NewService(mr, userSvc, authSvc, auditSvc, nil, Config{BaseURL: "", DevExposeURL: true, AllowImplicitSignup: true})
	return nil
}

func (p *Plugin) RegisterRoutes(router interface{}) error {
    if p.service == nil { return nil }
    switch v := router.(type) {
    case *forge.App:
        // For direct forge.App usage (not from Mount method)
        grp := v.Group("/api/auth")
        rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/magic-link/send": {Window: time.Minute, Max: 5}}})
        h := NewHandler(p.service, rls)
        grp.POST("/magic-link/send", h.Send)
        grp.GET("/magic-link/verify", h.Verify)
        return nil
    case *forge.Group:
        // Use relative paths - the router is already a group with the correct basePath
        rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/magic-link/send": {Window: time.Minute, Max: 5}}})
        h := NewHandler(p.service, rls)
        v.POST("/magic-link/send", h.Send)
        v.GET("/magic-link/verify", h.Verify)
        return nil
    case *http.ServeMux:
        app := forge.NewApp(v)
        grp := app.Group("/api/auth")
        rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/magic-link/send": {Window: time.Minute, Max: 5}}})
        h := NewHandler(p.service, rls)
        grp.POST("/magic-link/send", h.Send)
        grp.GET("/magic-link/verify", h.Verify)
        return nil
    default:
        return nil
    }
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.MagicLink)(nil)).IfNotExists().Exec(ctx)
	return err
}
