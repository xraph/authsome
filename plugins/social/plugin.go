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
	type authInstance interface {
		GetDB() *bun.DB
	}
	
	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("social plugin requires auth instance with GetDB method")
	}
	
	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for social plugin")
	}
	
	p.db = db

	// Create repositories
	socialRepo := repository.NewSocialAccountRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Create user service (simplified - in production, get from registry)
	userSvc := user.NewService(userRepo, user.Config{}, nil)

	// Create social service
	p.service = NewService(p.config, socialRepo, userSvc)

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
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return fmt.Errorf("social plugin not initialized")
	}

	handler := NewHandler(p.service)

	// Router is already scoped to the correct basePath
	router.POST("/signin/social", handler.SignIn,
		forge.WithName("social.signin"),
		forge.WithSummary("Sign in with social provider"),
		forge.WithDescription("Initiate OAuth sign-in flow with a social provider (Google, GitHub, Facebook, etc.)"),
		forge.WithResponseSchema(200, "OAuth redirect URL", SocialSignInResponse{}),
		forge.WithResponseSchema(400, "Invalid provider", SocialErrorResponse{}),
		forge.WithTags("Social", "Authentication"),
		forge.WithValidation(true),
	)
	router.GET("/callback/:provider", handler.Callback,
		forge.WithName("social.callback"),
		forge.WithSummary("OAuth callback"),
		forge.WithDescription("Handle OAuth provider callback and complete authentication"),
		forge.WithResponseSchema(200, "Authentication successful", SocialCallbackResponse{}),
		forge.WithResponseSchema(400, "OAuth error", SocialErrorResponse{}),
		forge.WithTags("Social", "Authentication"),
	)
	router.POST("/account/link", handler.LinkAccount,
		forge.WithName("social.link"),
		forge.WithSummary("Link social account"),
		forge.WithDescription("Link a social provider account to existing user account"),
		forge.WithResponseSchema(200, "Account linked", SocialLinkResponse{}),
		forge.WithResponseSchema(400, "Invalid request", SocialErrorResponse{}),
		forge.WithTags("Social", "Account Management"),
		forge.WithValidation(true),
	)
	router.DELETE("/account/unlink/:provider", handler.UnlinkAccount,
		forge.WithName("social.unlink"),
		forge.WithSummary("Unlink social account"),
		forge.WithDescription("Unlink a social provider account from user account"),
		forge.WithResponseSchema(200, "Account unlinked", SocialStatusResponse{}),
		forge.WithResponseSchema(404, "Provider not linked", SocialErrorResponse{}),
		forge.WithTags("Social", "Account Management"),
	)
	router.GET("/providers", handler.ListProviders,
		forge.WithName("social.providers.list"),
		forge.WithSummary("List available providers"),
		forge.WithDescription("List all configured social authentication providers"),
		forge.WithResponseSchema(200, "Providers list", SocialProvidersResponse{}),
		forge.WithTags("Social", "Configuration"),
	)
	return nil
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

// DTOs for social routes
type SocialErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type SocialStatusResponse struct {
	Status string `json:"status" example:"success"`
}

type SocialSignInResponse struct {
	RedirectURL string `json:"redirect_url" example:"https://accounts.google.com/o/oauth2/v2/auth?..."`
}

type SocialCallbackResponse struct {
	Token string `json:"token" example:"eyJhbGci..."`
	User  interface{} `json:"user"`
}

type SocialLinkResponse struct {
	Linked bool `json:"linked" example:"true"`
}

type SocialProvidersResponse struct {
	Providers []string `json:"providers" example:"[\"google\",\"github\",\"facebook\"]"`
}
