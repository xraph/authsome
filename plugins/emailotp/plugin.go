package emailotp

import (
	"context"
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

// Plugin implements the plugins.Plugin interface for Email OTP
type Plugin struct {
	service *Service
	db      *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "emailotp" }

func (p *Plugin) Init(dep interface{}) error {
	if db, ok := dep.(*bun.DB); ok && db != nil {
		p.db = db
		// wire repo and services
		eotpr := repo.NewEmailOTPRepository(db)
		userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
		sessionSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
		authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
		auditSvc := audit.NewService(repo.NewAuditRepository(db))
		p.service = NewService(eotpr, userSvc, authSvc, auditSvc, nil, Config{DevExposeOTP: true, AllowImplicitSignup: true})
	}
	return nil
}

func (p *Plugin) RegisterRoutes(router interface{}) error {
	if p.service == nil {
		return nil
	}
	switch v := router.(type) {
	case *forge.App:
		// For direct forge.App usage (not from Mount method)
		grp := v.Group("/api/auth")
		// Set up a simple in-memory rate limit: 5 sends per minute per email
		rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/api/auth/email-otp/send": {Window: time.Minute, Max: 5}}})
		h := NewHandler(p.service, rls)
		grp.POST("/email-otp/send", h.Send)
		grp.POST("/email-otp/verify", h.Verify)
		return nil
	case *forge.Group:
		// Use relative paths - the router is already a group with the correct basePath
		rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/email-otp/send": {Window: time.Minute, Max: 5}}})
		h := NewHandler(p.service, rls)
		v.POST("/email-otp/send", h.Send)
		v.POST("/email-otp/verify", h.Verify)
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
	_, err := p.db.NewCreateTable().Model((*schema.EmailOTP)(nil)).IfNotExists().Exec(ctx)
	return err
}
