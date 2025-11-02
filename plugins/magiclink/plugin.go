package magiclink

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
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/authsome/storage"
	"github.com/xraph/forge"
)

type Plugin struct {
	db           *bun.DB
	service      *Service
	notifAdapter *notificationPlugin.Adapter
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "magiclink" }

func (p *Plugin) Init(dep interface{}) error {
	db, ok := dep.(*bun.DB)
	if !ok {
		return nil
	}
	p.db = db
	
	// TODO: Get notification adapter from service registry when available
	// For now, plugins will work without notification adapter (graceful degradation)
	// The notification plugin should be registered first and will set up its services
	
	mr := repo.NewMagicLinkRepository(db)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
	// Build full auth service with session
	sessRepo := repo.NewSessionRepository(db)
	sessionSvc := session.NewService(sessRepo, session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	p.service = NewService(mr, userSvc, authSvc, auditSvc, p.notifAdapter, Config{
		BaseURL:             "",
		ExpiryMinutes:       15,
		DevExposeURL:        true,
		AllowImplicitSignup: true,
	})
	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the correct basePath
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/magic-link/send": {Window: time.Minute, Max: 5}}})
	h := NewHandler(p.service, rls)
	router.POST("/magic-link/send", h.Send)
	router.GET("/magic-link/verify", h.Verify)
	return nil
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
