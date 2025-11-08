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
	router.POST("/email-otp/send", h.Send,
		forge.WithName("emailotp.send"),
		forge.WithSummary("Send email OTP"),
		forge.WithDescription("Sends a one-time password (OTP) to the specified email address. Rate limited to 5 requests per minute per email"),
		forge.WithResponseSchema(200, "OTP sent", EmailOTPSendResponse{}),
		forge.WithResponseSchema(400, "Invalid request", EmailOTPErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", EmailOTPErrorResponse{}),
		forge.WithTags("EmailOTP", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/email-otp/verify", h.Verify,
		forge.WithName("emailotp.verify"),
		forge.WithSummary("Verify email OTP"),
		forge.WithDescription("Verifies the OTP code and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithResponseSchema(200, "OTP verified", EmailOTPVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", EmailOTPErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid OTP", EmailOTPErrorResponse{}),
		forge.WithTags("EmailOTP", "Authentication"),
		forge.WithValidation(true),
	)
	return nil
}

// Response types for email OTP routes
type EmailOTPErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type EmailOTPSendResponse struct {
	Status string `json:"status" example:"sent"`
	DevOTP string `json:"dev_otp,omitempty" example:"123456"`
}

type EmailOTPVerifyResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
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
