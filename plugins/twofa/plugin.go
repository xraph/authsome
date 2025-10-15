package twofa

import (
    "context"
    "net/http"
    "github.com/uptrace/bun"
    repo "github.com/xraph/authsome/repository"
    "github.com/xraph/forge"
    "github.com/xraph/authsome/schema"
    "github.com/xraph/authsome/core/hooks"
    "github.com/xraph/authsome/core/registry"
)

// Plugin implements the plugins.Plugin interface for Two-Factor Authentication
type Plugin struct {
    service *Service
    db      *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "twofa" }

func (p *Plugin) Init(dep interface{}) error {
    // Expect *bun.DB from the auth initializer
    if db, ok := dep.(*bun.DB); ok && db != nil {
        p.db = db
        // Wire repository-backed service
        p.service = NewService(repo.NewTwoFARepository(db))
        return nil
    }
    return nil
}

// RegisterRoutes registers 2FA endpoints under the auth base
func (p *Plugin) RegisterRoutes(router interface{}) error {
    switch v := router.(type) {
    case *forge.App:
        grp := v.Group("/api/auth")
        h := NewHandler(p.service)
        grp.POST("/2fa/enable", h.Enable)
        grp.POST("/2fa/verify", h.Verify)
        grp.POST("/2fa/disable", h.Disable)
        grp.POST("/2fa/generate-backup-codes", h.GenerateBackupCodes)
        grp.POST("/2fa/send-otp", h.SendOTP)
        grp.POST("/2fa/status", h.Status)
        return nil
    case *http.ServeMux:
        app := forge.NewApp(v)
        grp := app.Group("/api/auth")
        h := NewHandler(p.service)
        grp.POST("/2fa/enable", h.Enable)
        grp.POST("/2fa/verify", h.Verify)
        grp.POST("/2fa/disable", h.Disable)
        grp.POST("/2fa/generate-backup-codes", h.GenerateBackupCodes)
        grp.POST("/2fa/send-otp", h.SendOTP)
        grp.POST("/2fa/status", h.Status)
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