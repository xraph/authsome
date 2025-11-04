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
	router.POST("/username/signup", h.SignUp)
	router.POST("/username/signin", h.SignIn)
	return nil
}

// RegisterHooks placeholder
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate placeholder for DB migrations
func (p *Plugin) Migrate() error { return nil }
