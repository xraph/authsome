package twofa

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Two-Factor Authentication
type Plugin struct {
	service *Service
	db      *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "twofa" }

func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}
	
	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("twofa plugin requires auth instance with GetDB method")
	}
	
	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for twofa plugin")
	}
	
	p.db = db
	// Wire repository-backed service
	p.service = NewService(repo.NewTwoFARepository(db))
	return nil
}

// RegisterRoutes registers 2FA endpoints under the auth base
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Router is already scoped to the correct basePath by the auth mount
	h := NewHandler(p.service)
	router.POST("/2fa/enable", h.Enable)
	router.POST("/2fa/verify", h.Verify)
	router.POST("/2fa/disable", h.Disable)
	router.POST("/2fa/generate-backup-codes", h.GenerateBackupCodes)
	router.POST("/2fa/send-otp", h.SendOTP)
	router.POST("/2fa/status", h.Status)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	// Create required tables if not exist
	if _, err := p.db.NewCreateTable().Model((*schema.TwoFASecret)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := p.db.NewCreateTable().Model((*schema.BackupCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := p.db.NewCreateTable().Model((*schema.TrustedDevice)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	if _, err := p.db.NewCreateTable().Model((*schema.OTPCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}
	return nil
}
