package phone

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

func (p *Plugin) ID() string { return "phone" }

func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("phone plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for phone plugin")
	}

	p.db = db

	// TODO: Get notification adapter from service registry when available
	// For now, plugins will work without notification adapter (graceful degradation)
	// The notification plugin should be registered first and will set up its services

	pr := repo.NewPhoneRepository(db)
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
	sessSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessSvc, auth.Config{})
	auditSvc := audit.NewService(repo.NewAuditRepository(db))
	p.service = NewService(pr, userSvc, authSvc, auditSvc, p.notifAdapter, Config{
		CodeLength:          6,
		ExpiryMinutes:       10,
		DevExposeCode:       true,
		AllowImplicitSignup: true,
	})
	return nil
}

func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Router is already scoped to the correct basePath
	rls := rl.NewService(storage.NewMemoryStorage(), rl.Config{Enabled: true, Rules: map[string]rl.Rule{"/phone/send-code": {Window: time.Minute, Max: 5}}})
	h := NewHandler(p.service, rls)
	router.POST("/phone/send-code", h.SendCode,
		forge.WithName("phone.sendcode"),
		forge.WithSummary("Send phone verification code"),
		forge.WithDescription("Sends a verification code via SMS to the specified phone number. Rate limited to 5 requests per minute per phone"),
		forge.WithResponseSchema(200, "Code sent", PhoneSendCodeResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PhoneErrorResponse{}),
		forge.WithResponseSchema(429, "Too many requests", PhoneErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/phone/verify", h.Verify,
		forge.WithName("phone.verify"),
		forge.WithSummary("Verify phone code"),
		forge.WithDescription("Verifies the phone verification code and creates a user session on success. Supports implicit signup if enabled"),
		forge.WithResponseSchema(200, "Code verified", PhoneVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PhoneErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", PhoneErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/phone/signin", h.SignIn,
		forge.WithName("phone.signin"),
		forge.WithSummary("Sign in with phone"),
		forge.WithDescription("Alias for phone verification. Verifies the phone code and creates a user session"),
		forge.WithResponseSchema(200, "Sign in successful", PhoneVerifyResponse{}),
		forge.WithResponseSchema(400, "Invalid request", PhoneErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid code", PhoneErrorResponse{}),
		forge.WithTags("Phone", "Authentication"),
		forge.WithValidation(true),
	)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()
	_, err := p.db.NewCreateTable().Model((*schema.PhoneVerification)(nil)).IfNotExists().Exec(ctx)
	return err
}

// Response types for phone routes
type PhoneErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type PhoneSendCodeResponse struct {
	Status  string `json:"status" example:"sent"`
	DevCode string `json:"dev_code,omitempty" example:"123456"`
}

type PhoneVerifyResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
}
