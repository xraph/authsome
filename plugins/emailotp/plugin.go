package emailotp

import (
	"context"
	"fmt"
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

// Plugin implements the plugins.Plugin interface for Email OTP
type Plugin struct {
	service      *Service
	notifAdapter *notificationPlugin.Adapter
	db           *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "emailotp" }

func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("emailotp plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for emailotp plugin")
	}

	p.db = db

	// TODO: Get notification adapter from service registry when available
	// For now, plugins will work without notification adapter (graceful degradation)
	// The notification plugin should be registered first and will set up its services

	// wire repo and services
	eotpr := repo.NewEmailOTPRepository(db)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
	sessionSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	p.service = NewService(eotpr, userSvc, authSvc, auditSvc, p.notifAdapter, Config{
		OTPLength:           6,
		ExpiryMinutes:       10,
		DevExposeOTP:        true,
		AllowImplicitSignup: true,
	})
	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the correct basePath
	// Set up a simple in-memory rate limit: 5 sends per minute per email
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/email-otp/send": {Window: time.Minute, Max: 5}}})
	h := NewHandler(p.service, rls)
	router.POST("/email-otp/send", h.Send)
	router.POST("/email-otp/verify", h.Verify)
	return nil
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
