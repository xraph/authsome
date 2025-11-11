package oidcprovider

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin wires the OIDC Provider service and registers routes
type Plugin struct {
	db      *bun.DB
	service *Service
}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "oidcprovider" }

// Init accepts auth instance with GetDB method
func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("oidcprovider plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for oidcprovider plugin")
	}

	p.db = db

	// Create repositories
	clientRepo := repo.NewOAuthClientRepository(db)
	codeRepo := repo.NewAuthorizationCodeRepository(db)
	tokenRepo := repo.NewOAuthTokenRepository(db)
	userRepo := repo.NewUserRepository(db)

	// Create core services
	sessionSvc := session.NewService(repo.NewSessionRepository(db), session.Config{}, nil)
	userSvc := user.NewService(userRepo, user.Config{}, nil)

	// Create default config (TODO: integrate with ConfigManager)
	config := Config{
		Issuer: "http://localhost:3001", // Default issuer
	}
	config.Keys.RotationInterval = "24h"
	config.Keys.KeyLifetime = "168h" // 7 days
	config.Tokens.AccessTokenExpiry = "1h"
	config.Tokens.IDTokenExpiry = "1h"
	config.Tokens.RefreshTokenExpiry = "720h" // 30 days

	// Create OIDC Provider service with all dependencies
	p.service = NewServiceWithRepo(clientRepo, config)
	p.service.SetRepositories(clientRepo, codeRepo, tokenRepo)
	p.service.SetSessionService(sessionSvc)
	p.service.SetUserService(userSvc)

	// Start automatic key rotation
	p.service.StartKeyRotation()
	return nil
}

// RegisterRoutes mounts OIDC Provider endpoints under /oauth2
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return nil
	}
	// Create oauth2 group at root level (not under /api/auth)
	grp := router.Group("/oauth2")
	h := NewHandler(p.service)
	grp.GET("/authorize", h.Authorize,
		forge.WithName("oidc.authorize"),
		forge.WithSummary("OAuth2/OIDC authorization endpoint"),
		forge.WithDescription("OAuth2/OIDC authorization endpoint. Initiates the authorization flow and redirects to consent screen if needed"),
		forge.WithResponseSchema(302, "Redirect to consent or callback", nil),
		forge.WithResponseSchema(400, "Invalid request", OIDCErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Authorization"),
	)
	grp.POST("/consent", h.HandleConsent,
		forge.WithName("oidc.consent"),
		forge.WithSummary("Handle user consent"),
		forge.WithDescription("Processes user consent for OAuth2/OIDC authorization request"),
		forge.WithResponseSchema(302, "Redirect with authorization code", nil),
		forge.WithResponseSchema(400, "Invalid request", OIDCErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Consent"),
		forge.WithValidation(true),
	)
	grp.POST("/token", h.Token,
		forge.WithName("oidc.token"),
		forge.WithSummary("OAuth2 token endpoint"),
		forge.WithDescription("OAuth2/OIDC token endpoint. Exchanges authorization code for access token, ID token, and refresh token"),
		forge.WithResponseSchema(200, "Token response", OIDCTokenResponse{}),
		forge.WithResponseSchema(400, "Invalid request", OIDCErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Token"),
		forge.WithValidation(true),
	)
	grp.GET("/userinfo", h.UserInfo,
		forge.WithName("oidc.userinfo"),
		forge.WithSummary("OIDC userinfo endpoint"),
		forge.WithDescription("OIDC userinfo endpoint. Returns user information for authenticated access token"),
		forge.WithResponseSchema(200, "User info", OIDCUserInfoResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", OIDCErrorResponse{}),
		forge.WithTags("OIDC", "UserInfo"),
	)
	grp.GET("/jwks", h.JWKS,
		forge.WithName("oidc.jwks"),
		forge.WithSummary("JWKS endpoint"),
		forge.WithDescription("JSON Web Key Set (JWKS) endpoint. Returns public keys for token verification"),
		forge.WithResponseSchema(200, "JWKS", OIDCJWKSResponse{}),
		forge.WithTags("OIDC", "JWKS"),
	)
	grp.POST("/register", h.RegisterClient,
		forge.WithName("oidc.client.register"),
		forge.WithSummary("Register OAuth2 client"),
		forge.WithDescription("Dynamic client registration endpoint. Registers a new OAuth2/OIDC client application"),
		forge.WithResponseSchema(201, "Client registered", OIDCClientResponse{}),
		forge.WithResponseSchema(400, "Invalid request", OIDCErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Client"),
		forge.WithValidation(true),
	)
	return nil
}

func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error { return nil }

func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error { return nil }

// Migrate creates required tables for OIDC Provider
// Note: kept simple; production should handle migrations centrally
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}
	ctx := context.Background()

	// Create OAuth clients table
	if _, err := p.db.NewCreateTable().Model((*schema.OAuthClient)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	// Create authorization codes table
	if _, err := p.db.NewCreateTable().Model((*schema.AuthorizationCode)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	// Create OAuth tokens table
	if _, err := p.db.NewCreateTable().Model((*schema.OAuthToken)(nil)).IfNotExists().Exec(ctx); err != nil {
		return err
	}

	// Create indexes for authorization codes
	if _, err := p.db.NewCreateIndex().
		Model((*schema.AuthorizationCode)(nil)).
		Index("idx_authorization_codes_code").
		Column("code").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.AuthorizationCode)(nil)).
		Index("idx_authorization_codes_user_id").
		Column("user_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.AuthorizationCode)(nil)).
		Index("idx_authorization_codes_client_id").
		Column("client_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	// Create indexes for OAuth tokens
	if _, err := p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_access_token").
		Column("access_token").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_refresh_token").
		Column("refresh_token").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_user_id").
		Column("user_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_client_id").
		Column("client_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return nil
}

// Cleanup stops background processes when the plugin is shut down
func (p *Plugin) Cleanup() {
	if p.service != nil {
		p.service.StopKeyRotation()
	}
}

// Response types for OIDC Provider routes
type OIDCErrorResponse struct {
	Error            string `json:"error" example:"invalid_request"`
	ErrorDescription string `json:"error_description,omitempty" example:"The request is missing a required parameter"`
}

type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"3600"`
	RefreshToken string `json:"refresh_token,omitempty" example:"def50200..."`
	IDToken      string `json:"id_token,omitempty" example:"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Scope        string `json:"scope,omitempty" example:"openid profile email"`
}

type OIDCUserInfoResponse struct {
	Sub           string `json:"sub" example:"01HZ..."`
	Email         string `json:"email,omitempty" example:"user@example.com"`
	EmailVerified bool   `json:"email_verified,omitempty" example:"true"`
	Name          string `json:"name,omitempty" example:"John Doe"`
	GivenName     string `json:"given_name,omitempty" example:"John"`
	FamilyName    string `json:"family_name,omitempty" example:"Doe"`
	Picture       string `json:"picture,omitempty" example:"https://example.com/avatar.jpg"`
}

type OIDCJWKSResponse struct {
	Keys []interface{} `json:"keys"`
}

type OIDCClientResponse struct {
	ClientID     string   `json:"client_id" example:"client_123"`
	ClientSecret string   `json:"client_secret,omitempty" example:"secret_456"`
	RedirectURIs []string `json:"redirect_uris" example:"https://example.com/callback"`
}
