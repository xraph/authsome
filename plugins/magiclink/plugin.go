package magiclink

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

type Plugin struct {
	db           *bun.DB
	service      *Service
	notifAdapter *notificationPlugin.Adapter
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "magiclink" }

func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("magiclink plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for magiclink plugin")
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
	router.POST("/magic-link/send", h.Send,
		forge.WithName("magiclink.send"),
		forge.WithSummary("Send magic link"),
		forge.WithDescription("Sends a passwordless authentication link to the specified email address. Rate limited to 5 requests per minute per email"),
		forge.WithResponseSchema(200, "Magic link sent", MagicLinkSendResponse{}),
		forge.WithResponseSchema(400, "Invalid request", MagicLinkErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", MagicLinkErrorResponse{}),
		forge.WithTags("MagicLink", "Authentication"),
		forge.WithValidation(true),
	)
	router.GET("/magic-link/verify", h.Verify,
		forge.WithName("magiclink.verify"),
		forge.WithSummary("Verify magic link"),
		forge.WithDescription("Verifies the magic link token from email and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithResponseSchema(200, "Magic link verified", MagicLinkVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", MagicLinkErrorResponse{}),
		forge.WithTags("MagicLink", "Authentication"),
	)
	return nil
}

// Response types for magic link routes
type MagicLinkErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type MagicLinkSendResponse struct {
	Status string `json:"status" example:"sent"`
	DevURL string `json:"dev_url,omitempty" example:"http://localhost:3000/magic-link/verify?token=abc123"`
}

type MagicLinkVerifyResponse struct {
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
	_, err := p.db.NewCreateTable().Model((*schema.MagicLink)(nil)).IfNotExists().Exec(ctx)
	return err
}
