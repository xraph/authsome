package multisession

import (
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/auth"
	dev "github.com/xraph/authsome/core/device"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/core/webhook"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Plugin wires the multi-session service and registers routes
type Plugin struct {
	db      *bun.DB
	service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "multisession" }

// Init expects a *bun.DB and constructs local services
func (p *Plugin) Init(dep interface{}) error {
	if db, ok := dep.(*bun.DB); ok && db != nil {
		p.db = db
		// Core services used for auth context
		auditSvc := audit.NewService(repo.NewAuditRepository(db))
		webhookSvc := webhook.NewService(webhook.Config{}, repo.NewWebhookRepository(db), auditSvc)
		userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, webhookSvc)
		sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{AllowMultiple: true}, webhookSvc)
		authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
		devSvc := dev.NewService(repo.NewDeviceRepository(db))
		p.service = NewService(repo.NewSessionRepository(db), repo.NewDeviceRepository(db), authSvc, devSvc)
	}
	return nil
}

// RegisterRoutes mounts endpoints under /api/auth/multi-session
func (p *Plugin) RegisterRoutes(router interface{}) error {
	if p.service == nil {
		return nil
	}
	switch v := router.(type) {
	case *forge.App:
		// For direct forge.App usage (not from Mount method)
		grp := v.Group("/api/auth/multi-session")
		h := NewHandler(p.service)
		grp.GET("/list", h.List)
		grp.POST("/set-active", h.SetActive)
		grp.POST("/delete/{id}", h.Delete)
		return nil
	case *forge.Group:
		// Use relative paths - the router is already a group with the correct basePath
		grp := v.Group("/multi-session")
		h := NewHandler(p.service)
		grp.GET("/list", h.List)
		grp.POST("/set-active", h.SetActive)
		grp.POST("/delete/{id}", h.Delete)
		return nil
	default:
		return nil
	}
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }
func (p *Plugin) Migrate() error                                              { return nil }

// GetAuthService returns the auth service for testing
func (p *Plugin) GetAuthService() *auth.Service {
	if p.service == nil {
		return nil
	}
	return p.service.auth
}
