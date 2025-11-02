// Package passkey provides WebAuthn/FIDO2 passkey authentication.
//
// ⚠️ EXPERIMENTAL / BETA STATUS ⚠️
//
// This plugin is currently in experimental/beta status. The WebAuthn implementation
// is a basic stub and NOT production-ready. Critical cryptographic operations including
// challenge generation, attestation verification, and signature validation are not
// properly implemented.
//
// DO NOT USE IN PRODUCTION without completing the WebAuthn implementation.
// See plugins/passkey/README.md for details and roadmap.
package passkey

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

type Plugin struct {
	db      *bun.DB
	service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "passkey" }

func (p *Plugin) Init(dep interface{}) error {
	db, ok := dep.(*bun.DB)
	if !ok {
		return nil
	}
	p.db = db
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
	sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	p.service = NewService(db, userSvc, authSvc, auditSvc, Config{RPID: "localhost", RPName: "Authsome"})
	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the auth basePath, create passkey sub-group
	grp := router.Group("/passkey")
	h := NewHandler(p.service)
	grp.POST("/register/begin", h.BeginRegister)
	grp.POST("/register/finish", h.FinishRegister)
	grp.POST("/login/begin", h.BeginLogin)
	grp.POST("/login/finish", h.FinishLogin)
	grp.GET("/list", h.List)
	grp.DELETE("/:id", h.Delete)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.Passkey)(nil)).IfNotExists().Exec(ctx)
	return err
}