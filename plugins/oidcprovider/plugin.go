package oidcprovider

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Config represents the OIDC Provider configuration
type Config struct {
	// Issuer URL for the OIDC Provider
	Issuer string `json:"issuer"`

	// Key configuration
	Keys struct {
		// Path to RSA private key file (PEM format)
		PrivateKeyPath string `json:"privateKeyPath"`
		// Path to RSA public key file (PEM format)
		PublicKeyPath string `json:"publicKeyPath"`
		// Key rotation settings
		RotationInterval string `json:"rotationInterval"` // e.g., "24h"
		KeyLifetime      string `json:"keyLifetime"`      // e.g., "168h" (7 days)
	} `json:"keys"`

	// Token settings
	Tokens struct {
		AccessTokenExpiry  string `json:"accessTokenExpiry"`  // e.g., "1h"
		IDTokenExpiry      string `json:"idTokenExpiry"`      // e.g., "1h"
		RefreshTokenExpiry string `json:"refreshTokenExpiry"` // e.g., "720h" (30 days)
	} `json:"tokens"`
}

// Plugin wires the OIDC Provider service and registers routes
type Plugin struct {
	db            *bun.DB
	service       *Service
	adminHandler  *AdminHandler
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// PluginOption is a functional option for configuring the OIDC provider plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithIssuer sets the OIDC issuer URL
func WithIssuer(issuer string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Issuer = issuer
	}
}

// NewPlugin creates a new OIDC provider plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		defaultConfig: DefaultConfig(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Plugin) ID() string { return "oidcprovider" }

// Init accepts auth instance with GetDB method
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("oidcprovider plugin requires auth instance")
	}

	// Get dependencies
	p.db = authInst.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available for oidcprovider plugin")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for oidcprovider plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "oidcprovider"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.oidcprovider", &p.config, p.defaultConfig); err != nil {
		p.logger.Warn("failed to bind OIDC provider config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Register Bun models
	p.db.RegisterModel((*schema.OAuthClient)(nil))
	p.db.RegisterModel((*schema.AuthorizationCode)(nil))
	p.db.RegisterModel((*schema.OAuthToken)(nil))
	p.db.RegisterModel((*schema.OAuthConsent)(nil))

	// Create repositories
	clientRepo := repo.NewOAuthClientRepository(p.db)
	codeRepo := repo.NewAuthorizationCodeRepository(p.db)
	tokenRepo := repo.NewOAuthTokenRepository(p.db)
	consentRepo := repo.NewOAuthConsentRepository(p.db)
	userRepo := repo.NewUserRepository(p.db)

	// Create core services
	sessionSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil, nil)
	userSvc := user.NewService(userRepo, user.Config{}, nil, nil)

	// Create OIDC Provider service with config
	p.service = NewServiceWithRepos(clientRepo, p.config)

	// Set all repositories and initialize enterprise services
	p.service.SetRepositories(clientRepo, codeRepo, tokenRepo, consentRepo)
	p.service.SetSessionService(sessionSvc)
	p.service.SetUserService(userSvc)

	// Start automatic key rotation
	p.service.StartKeyRotation()

	// Initialize admin handler
	p.adminHandler = NewAdminHandler(clientRepo, p.service.registration)

	p.logger.Info("OIDC provider plugin initialized",
		forge.F("issuer", p.config.Issuer))

	return nil
}

// RegisterHooks registers plugin hooks
func (p *Plugin) RegisterHooks(hooksRegistry *hooks.HookRegistry) error {
	// No hooks to register currently
	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// No service decorators to register currently
	return nil
}

// RegisterRoutes mounts OIDC Provider endpoints
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	// Create oauth2 group at root level (not under /api/auth)
	grp := router.Group("/oauth2")
	h := NewHandler(p.service)

	// =============================================================================
	// PUBLIC ENDPOINTS (No authentication required)
	// =============================================================================

	// OIDC Discovery endpoint
	grp.GET("/.well-known/openid-configuration", h.Discovery,
		forge.WithName("oidc.discovery"),
		forge.WithSummary("OIDC Discovery Document"),
		forge.WithDescription("Returns the OpenID Connect discovery document with supported endpoints and capabilities"),
		forge.WithResponseSchema(200, "Discovery document", DiscoveryResponse{}),
		forge.WithTags("OIDC", "Discovery"),
	)

	// JWKS endpoint
	grp.GET("/jwks", h.JWKS,
		forge.WithName("oidc.jwks"),
		forge.WithSummary("JSON Web Key Set"),
		forge.WithDescription("Returns public keys for token verification"),
		forge.WithResponseSchema(200, "JWKS", JWKSResponse{}),
		forge.WithTags("OIDC", "JWKS"),
	)

	// Authorization endpoint
	grp.GET("/authorize", h.Authorize,
		forge.WithName("oidc.authorize"),
		forge.WithSummary("OAuth2/OIDC authorization endpoint"),
		forge.WithDescription("Initiates the authorization flow and redirects to consent screen if needed"),
		forge.WithResponseSchema(302, "Redirect to consent or callback", nil),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Authorization"),
	)

	// Consent handling
	grp.POST("/consent", h.HandleConsent,
		forge.WithName("oidc.consent"),
		forge.WithSummary("Handle user consent"),
		forge.WithDescription("Processes user consent for OAuth2/OIDC authorization request"),
		forge.WithRequestSchema(ConsentRequest{}),
		forge.WithResponseSchema(302, "Redirect with authorization code", nil),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Consent"),
		forge.WithValidation(true),
	)

	// Token endpoint
	grp.POST("/token", h.Token,
		forge.WithName("oidc.token"),
		forge.WithSummary("OAuth2 token endpoint"),
		forge.WithDescription("Exchanges authorization code for access token, ID token, and refresh token"),
		forge.WithRequestSchema(TokenRequest{}),
		forge.WithResponseSchema(200, "Token response", TokenResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Token"),
		forge.WithValidation(true),
	)

	// UserInfo endpoint (requires valid access token)
	grp.GET("/userinfo", h.UserInfo,
		forge.WithName("oidc.userinfo"),
		forge.WithSummary("OIDC userinfo endpoint"),
		forge.WithDescription("Returns user information for authenticated access token"),
		forge.WithResponseSchema(200, "User info", UserInfoResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("OIDC", "UserInfo"),
	)

	// =============================================================================
	// ENTERPRISE ENDPOINTS (Require client authentication)
	// =============================================================================

	// Token introspection (RFC 7662)
	grp.POST("/introspect", h.IntrospectToken,
		forge.WithName("oidc.introspect"),
		forge.WithSummary("Token introspection endpoint (RFC 7662)"),
		forge.WithDescription("Returns token metadata for authenticated client. Requires client authentication."),
		forge.WithRequestSchema(TokenIntrospectionRequest{}),
		forge.WithResponseSchema(200, "Introspection result", TokenIntrospectionResponse{}),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Introspection", "Enterprise"),
		forge.WithValidation(true),
	)

	// Token revocation (RFC 7009)
	grp.POST("/revoke", h.RevokeToken,
		forge.WithName("oidc.revoke"),
		forge.WithSummary("Token revocation endpoint (RFC 7009)"),
		forge.WithDescription("Revokes access or refresh token. Requires client authentication."),
		forge.WithRequestSchema(TokenRevocationRequest{}),
		forge.WithResponseSchema(200, "Token revoked", nil),
		forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Revocation", "Enterprise"),
		forge.WithValidation(true),
	)

	// =============================================================================
	// ADMIN ENDPOINTS (Require admin authentication)
	// =============================================================================

	// Dynamic client registration (RFC 7591) - admin only
	grp.POST("/register", p.adminHandler.RegisterClient,
		forge.WithName("oidc.client.register"),
		forge.WithSummary("Dynamic client registration (RFC 7591)"),
		forge.WithDescription("Admin-only endpoint for OAuth2/OIDC client registration"),
		forge.WithRequestSchema(ClientRegistrationRequest{}),
		forge.WithResponseSchema(201, "Client registered", ClientRegistrationResponse{}),
		forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
		forge.WithResponseSchema(403, "Admin required", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Client", "Admin"),
		forge.WithValidation(true),
	)

	// Client management endpoints (admin only)
	clientGroup := grp.Group("/clients")

	clientGroup.GET("", p.adminHandler.ListClients,
		forge.WithName("oidc.clients.list"),
		forge.WithSummary("List OAuth clients"),
		forge.WithDescription("Lists all OAuth clients for the current app/environment/organization"),
		forge.WithResponseSchema(200, "Clients list", ClientsListResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Client", "Admin"),
	)

	clientGroup.GET("/:clientId", p.adminHandler.GetClient,
		forge.WithName("oidc.clients.get"),
		forge.WithSummary("Get OAuth client details"),
		forge.WithDescription("Retrieves detailed information about an OAuth client"),
		forge.WithResponseSchema(200, "Client details", ClientDetailsResponse{}),
		forge.WithResponseSchema(404, "Client not found", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Client", "Admin"),
	)

	clientGroup.PUT("/:clientId", p.adminHandler.UpdateClient,
		forge.WithName("oidc.clients.update"),
		forge.WithSummary("Update OAuth client"),
		forge.WithDescription("Updates an existing OAuth client configuration"),
		forge.WithRequestSchema(ClientUpdateRequest{}),
		forge.WithResponseSchema(200, "Client updated", ClientDetailsResponse{}),
		forge.WithResponseSchema(404, "Client not found", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Client", "Admin"),
		forge.WithValidation(true),
	)

	clientGroup.DELETE("/:clientId", p.adminHandler.DeleteClient,
		forge.WithName("oidc.clients.delete"),
		forge.WithSummary("Delete OAuth client"),
		forge.WithDescription("Deletes an OAuth client and revokes all associated tokens"),
		forge.WithResponseSchema(204, "Client deleted", nil),
		forge.WithResponseSchema(404, "Client not found", ErrorResponse{}),
		forge.WithTags("OIDC", "OAuth2", "Client", "Admin"),
	)

	p.logger.Info("OIDC provider routes registered")
	return nil
}

// Migrate runs database migrations
func (p *Plugin) Migrate() error {
	ctx := context.Background()
	if p.db == nil {
		return fmt.Errorf("database not available")
	}

	// Create tables
	models := []interface{}{
		(*schema.OAuthClient)(nil),
		(*schema.AuthorizationCode)(nil),
		(*schema.OAuthToken)(nil),
		(*schema.OAuthConsent)(nil),
	}

	for _, model := range models {
		_, err := p.db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create table for %T: %w", model, err)
		}
	}

	// Create indexes
	// OAuth Client indexes
	_, err := p.db.NewCreateIndex().
		Model((*schema.OAuthClient)(nil)).
		Index("idx_oauth_clients_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create oauth_clients index: %w", err)
	}

	// Authorization Code indexes
	_, err = p.db.NewCreateIndex().
		Model((*schema.AuthorizationCode)(nil)).
		Index("idx_auth_codes_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create authorization_codes index: %w", err)
	}

	// OAuth Token indexes
	_, err = p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create oauth_tokens index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_jti").
		Column("jti").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create oauth_tokens jti index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.OAuthToken)(nil)).
		Index("idx_oauth_tokens_session").
		Column("session_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create oauth_tokens session index: %w", err)
	}

	// OAuth Consent indexes
	_, err = p.db.NewCreateIndex().
		Model((*schema.OAuthConsent)(nil)).
		Index("idx_oauth_consents_user_client").
		Column("user_id", "client_id", "app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create oauth_consents index: %w", err)
	}

	p.logger.Info("OIDC provider migrations completed")
	return nil
}

// RegisterExtensions registers the plugin with the extension registry
func (p *Plugin) RegisterExtensions(reg interface{}) error {
	// Register as OAuth provider if registry supports it
	// This is optional and can be implemented when registry is available
	return nil
}

// Shutdown performs cleanup when the plugin is shutting down
func (p *Plugin) Shutdown() error {
	if p.service != nil {
		p.service.StopKeyRotation()
	}
	p.logger.Info("OIDC provider plugin shutdown complete")
	return nil
}
