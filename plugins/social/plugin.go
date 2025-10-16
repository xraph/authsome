package social

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the social OAuth plugin
type Plugin struct {
	db      *bun.DB
	service *Service
	config  Config
}

// NewPlugin creates a new social OAuth plugin
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "social"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(dep interface{}) error {
	switch v := dep.(type) {
	case *bun.DB:
		if v != nil {
			p.db = v

			// Create repositories
			socialRepo := repository.NewSocialAccountRepository(v)
			userRepo := repository.NewUserRepository(v)

			// Create user service (simplified - in production, get from registry)
			userSvc := user.NewService(userRepo, user.Config{}, nil)

			// Create social service
			p.service = NewService(p.config, socialRepo, userSvc)
		}
	case map[string]interface{}:
		// Accept configuration map
		if db, ok := v["db"].(*bun.DB); ok {
			p.db = db
		}
		if config, ok := v["config"].(Config); ok {
			p.config = config
		}
	}

	return nil
}

// SetConfig allows setting configuration after plugin creation
func (p *Plugin) SetConfig(config Config) {
	p.config = config
	if p.service != nil {
		// Reinitialize service with new config
		socialRepo := repository.NewSocialAccountRepository(p.db)
		userRepo := repository.NewUserRepository(p.db)
		userSvc := user.NewService(userRepo, user.Config{}, nil)
		p.service = NewService(config, socialRepo, userSvc)
	}
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router interface{}) error {
	if p.service == nil {
		return fmt.Errorf("social plugin not initialized")
	}

	handler := NewHandler(p.service)

	switch v := router.(type) {
	case *forge.App:
		// Direct forge.App usage
		authGroup := v.Group("/api/auth")
		authGroup.POST("/signin/social", handler.SignIn)
		authGroup.GET("/callback/:provider", handler.Callback)
		authGroup.POST("/account/link", handler.LinkAccount)
		authGroup.DELETE("/account/unlink/:provider", handler.UnlinkAccount)
		authGroup.GET("/providers", handler.ListProviders)
		return nil

	case *forge.Group:
		// Already within a group (e.g., /api/auth)
		v.POST("/signin/social", handler.SignIn)
		v.GET("/callback/:provider", handler.Callback)
		v.POST("/account/link", handler.LinkAccount)
		v.DELETE("/account/unlink/:provider", handler.UnlinkAccount)
		v.GET("/providers", handler.ListProviders)
		return nil
	default:
		return fmt.Errorf("unsupported router type: %T", router)
	}
}

// RegisterHooks registers plugin hooks
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error {
	// TODO: Add hooks for OAuth flow events
	// - OnSocialSignIn
	// - OnSocialSignUp
	// - OnAccountLinked
	// - OnAccountUnlinked
	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	// No service decorators needed for social plugin
	return nil
}

// Migrate creates database tables
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create social_accounts table
	_, err := p.db.NewCreateTable().
		Model((*schema.SocialAccount)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create social_accounts table: %w", err)
	}

	// Create indexes for performance
	indexes := []struct {
		name    string
		columns []string
	}{
		{"idx_social_accounts_user_id", []string{"user_id"}},
		{"idx_social_accounts_provider", []string{"provider"}},
		{"idx_social_accounts_provider_id", []string{"provider_id"}},
		{"idx_social_accounts_org_id", []string{"organization_id"}},
		{"idx_social_accounts_email", []string{"email"}},
		{"idx_social_accounts_provider_provider_id_org", []string{"provider", "provider_id", "organization_id"}},
	}

	for _, idx := range indexes {
		_, err := p.db.NewCreateIndex().
			Model((*schema.SocialAccount)(nil)).
			Index(idx.name).
			Column(idx.columns...).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	return nil
}

// GetService returns the social service (for testing/internal use)
func (p *Plugin) GetService() *Service {
	return p.service
}
