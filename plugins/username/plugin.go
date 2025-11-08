package username

import (
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/auth"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/forge"
)

// Plugin implements the plugins.Plugin interface for Username auth
type Plugin struct {
	service *Service
	db      *bun.DB
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "username" }

// Init accepts auth instance with GetDB method
func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("username plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for username plugin")
	}

	p.db = db
	// Construct local core services
	userSvc := user.NewService(repo.NewUserRepository(db), user.Config{}, nil)
	sessionSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
	authSvc := auth.NewService(userSvc, sessionSvc, auth.Config{})
	p.service = NewService(userSvc, authSvc)
	return nil
}

// RegisterRoutes registers Username plugin routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Router is already scoped to the correct basePath
	h := NewHandler(p.service, repo.NewTwoFARepository(p.db))
	router.POST("/username/signup", h.SignUp,
		forge.WithName("username.signup"),
		forge.WithSummary("Sign up with username"),
		forge.WithDescription("Creates a new user account with username and password"),
		forge.WithResponseSchema(201, "User created", UsernameStatusResponse{}),
		forge.WithResponseSchema(400, "Invalid request", UsernameErrorResponse{}),
		forge.WithTags("Username", "Authentication"),
		forge.WithValidation(true),
	)
	router.POST("/username/signin", h.SignIn,
		forge.WithName("username.signin"),
		forge.WithSummary("Sign in with username"),
		forge.WithDescription("Authenticates user with username and password. Returns 2FA requirement if enabled and device is not trusted"),
		forge.WithResponseSchema(200, "Sign in successful", UsernameSignInResponse{}),
		forge.WithResponseSchema(200, "2FA required", Username2FARequiredResponse{}),
		forge.WithResponseSchema(400, "Invalid request", UsernameErrorResponse{}),
		forge.WithResponseSchema(401, "Invalid credentials", UsernameErrorResponse{}),
		forge.WithTags("Username", "Authentication"),
		forge.WithValidation(true),
	)
	return nil
}

// Response types for username routes
type UsernameErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type UsernameStatusResponse struct {
	Status string `json:"status" example:"created"`
}

type UsernameSignInResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token" example:"session_token_abc123"`
}

type Username2FARequiredResponse struct {
	User         interface{} `json:"user"`
	RequireTwoFA bool        `json:"require_twofa" example:"true"`
	DeviceID     string      `json:"device_id" example:"device_fingerprint"`
}

// RegisterHooks placeholder
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate placeholder for DB migrations
func (p *Plugin) Migrate() error { return nil }
