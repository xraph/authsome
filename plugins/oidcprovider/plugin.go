package oidcprovider

import (
	"context"
	"fmt"
	"time"
	"github.com/rs/xid"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/oidcprovider/deviceflow"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Config represents the OIDC Provider configuration
type Config struct {
	// Issuer URL for the OIDC Provider
	Issuer string `json:"issuer"`
	
	// Audience for JWT validation (optional)
	Audience string `json:"audience"`
	
	// Enable JWT validation strategy for Bearer tokens
	// When enabled, OAuth/OIDC JWTs can be used to authenticate API requests
	EnableJWTValidation bool `json:"enableJwtValidation"`

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

	// Device Flow configuration (RFC 8628)
	DeviceFlow struct {
		Enabled         bool   `json:"enabled"`
		CodeExpiry      string `json:"codeExpiry"`      // e.g., "10m"
		UserCodeLength  int    `json:"userCodeLength"`  // e.g., 8
		UserCodeFormat  string `json:"userCodeFormat"`  // e.g., "XXXX-XXXX"
		PollingInterval int    `json:"pollingInterval"` // e.g., 5 seconds
		VerificationURI string `json:"verificationUri"` // e.g., "/device"
		MaxPollAttempts int    `json:"maxPollAttempts"` // e.g., 120
		CleanupInterval string `json:"cleanupInterval"` // e.g., "5m"
		LoginURL        string `json:"loginUrl"`        // Custom login URL (e.g., "https://yourapp.com/login" or "/auth/signin" for authsome UI)
		APIMode         bool   `json:"apiMode"`         // If true, returns JSON with loginUrl instead of HTTP redirect (better for SPAs/mobile apps)
	} `json:"deviceFlow"`
}

// Plugin wires the OIDC Provider service and registers routes
type Plugin struct {
	db                  *bun.DB
	service             *Service
	adminHandler        *AdminHandler
	logger              forge.Logger
	config              Config
	defaultConfig       Config
	deviceCleanupTicker *time.Ticker
	deviceCleanupDone   chan bool
	dashboardExt        *DashboardExtension
	basePath            string // Base path from auth instance (e.g., "/api/identity")
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

// WithAudience sets the expected JWT audience for validation
func WithAudience(audience string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Audience = audience
	}
}

// WithJWTValidationEnabled enables JWT validation strategy
// When enabled, OAuth/OIDC JWTs can be used to authenticate API requests globally
func WithJWTValidationEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.EnableJWTValidation = enabled
	}
}

// WithPrivateKeyPath sets the path to the RSA private key file
func WithPrivateKeyPath(path string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Keys.PrivateKeyPath = path
	}
}

// WithPublicKeyPath sets the path to the RSA public key file
func WithPublicKeyPath(path string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Keys.PublicKeyPath = path
	}
}

// WithKeyRotationInterval sets the key rotation interval
func WithKeyRotationInterval(interval string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Keys.RotationInterval = interval
	}
}

// WithKeyLifetime sets the key lifetime
func WithKeyLifetime(lifetime string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Keys.KeyLifetime = lifetime
	}
}

// WithAccessTokenExpiry sets the access token expiry duration
func WithAccessTokenExpiry(expiry string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Tokens.AccessTokenExpiry = expiry
	}
}

// WithIDTokenExpiry sets the ID token expiry duration
func WithIDTokenExpiry(expiry string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Tokens.IDTokenExpiry = expiry
	}
}

// WithRefreshTokenExpiry sets the refresh token expiry duration
func WithRefreshTokenExpiry(expiry string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Tokens.RefreshTokenExpiry = expiry
	}
}

// WithDeviceFlowEnabled enables or disables the device flow
func WithDeviceFlowEnabled(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.Enabled = enabled
	}
}

// WithDeviceFlowCodeExpiry sets the device code expiry duration
func WithDeviceFlowCodeExpiry(expiry string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.CodeExpiry = expiry
	}
}

// WithDeviceFlowUserCodeLength sets the length of the user code
func WithDeviceFlowUserCodeLength(length int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.UserCodeLength = length
	}
}

// WithDeviceFlowUserCodeFormat sets the format of the user code (e.g., "XXXX-XXXX")
func WithDeviceFlowUserCodeFormat(format string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.UserCodeFormat = format
	}
}

// WithDeviceFlowPollingInterval sets the polling interval in seconds
func WithDeviceFlowPollingInterval(interval int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.PollingInterval = interval
	}
}

// WithDeviceFlowVerificationURI sets the verification URI for device flow
func WithDeviceFlowVerificationURI(uri string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.VerificationURI = uri
	}
}

// WithDeviceFlowMaxPollAttempts sets the maximum number of polling attempts
func WithDeviceFlowMaxPollAttempts(max int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.MaxPollAttempts = max
	}
}

// WithDeviceFlowCleanupInterval sets the cleanup interval for expired device codes
func WithDeviceFlowCleanupInterval(interval string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.CleanupInterval = interval
	}
}

// WithDeviceFlowLoginURL sets the custom login URL for device flow authentication
// If not set, defaults to "/auth/signin". Use this to redirect to your frontend's login page.
// Example: "https://yourapp.com/login" or "https://app.example.com/auth/login"
func WithDeviceFlowLoginURL(loginURL string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.LoginURL = loginURL
	}
}

// WithDeviceFlowAPIMode enables API mode for device flow authentication
// When enabled, returns JSON with loginUrl instead of HTTP redirect (better for SPAs/mobile apps)
// Default is false (HTML redirect mode for backward compatibility)
func WithDeviceFlowAPIMode(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.APIMode = enabled
	}
}

// WithDeviceFlowConfig sets all device flow configuration at once
func WithDeviceFlowConfig(enabled bool, codeExpiry string, userCodeLength int, userCodeFormat string, pollingInterval int, verificationURI string, maxPollAttempts int, cleanupInterval string, loginURL string, apiMode bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DeviceFlow.Enabled = enabled
		p.defaultConfig.DeviceFlow.CodeExpiry = codeExpiry
		p.defaultConfig.DeviceFlow.UserCodeLength = userCodeLength
		p.defaultConfig.DeviceFlow.UserCodeFormat = userCodeFormat
		p.defaultConfig.DeviceFlow.PollingInterval = pollingInterval
		p.defaultConfig.DeviceFlow.VerificationURI = verificationURI
		p.defaultConfig.DeviceFlow.MaxPollAttempts = maxPollAttempts
		p.defaultConfig.DeviceFlow.LoginURL = loginURL
		p.defaultConfig.DeviceFlow.APIMode = apiMode
		p.defaultConfig.DeviceFlow.CleanupInterval = cleanupInterval
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

// loggerAdapter adapts forge.Logger to Printf interface
type loggerAdapter struct {
	logger forge.Logger
}

func (la *loggerAdapter) Printf(format string, args ...interface{}) {
	la.logger.Info(fmt.Sprintf(format, args...))
}

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
	p.db.RegisterModel((*schema.DeviceCode)(nil))

	// Create repositories
	clientRepo := repo.NewOAuthClientRepository(p.db)
	codeRepo := repo.NewAuthorizationCodeRepository(p.db)
	tokenRepo := repo.NewOAuthTokenRepository(p.db)
	consentRepo := repo.NewOAuthConsentRepository(p.db)
	userRepo := repo.NewUserRepository(p.db)
	deviceCodeRepo := repo.NewDeviceCodeRepository(p.db)

	// Create core services
	sessionSvc := session.NewService(repo.NewSessionRepository(p.db), session.Config{}, nil, nil)
	userSvc := user.NewService(userRepo, user.Config{}, nil, nil)

	// Get app context for key storage
	appSvc := authInst.GetServiceRegistry().AppService()
	appID := xid.NilID() // Use platform keys by default
	if appSvc != nil {
		// Try to get platform app ID
		if platformApp, err := appSvc.GetPlatformApp(context.Background()); err == nil && platformApp != nil {
			appID = platformApp.ID
		}
	}

	// Create logger adapter for service
	serviceLogger := &loggerAdapter{logger: p.logger}

	// Create OIDC Provider service with database-backed keys
	p.service = NewServiceWithRepos(clientRepo, p.config, p.db, appID, serviceLogger)

	// Set all repositories and initialize enterprise services
	p.service.SetRepositories(clientRepo, codeRepo, tokenRepo, consentRepo)
	p.service.SetSessionService(sessionSvc)
	p.service.SetUserService(userSvc)

	// Initialize device flow if enabled
	if p.config.DeviceFlow.Enabled {
		deviceFlowConfig := deviceflow.DefaultConfig()

		// Parse code expiry duration
		if p.config.DeviceFlow.CodeExpiry != "" {
			if expiry, err := time.ParseDuration(p.config.DeviceFlow.CodeExpiry); err == nil {
				deviceFlowConfig.DeviceCodeExpiry = expiry
			}
		}

		// Apply other device flow config values
		if p.config.DeviceFlow.UserCodeLength > 0 {
			deviceFlowConfig.UserCodeLength = p.config.DeviceFlow.UserCodeLength
		}
		if p.config.DeviceFlow.UserCodeFormat != "" {
			deviceFlowConfig.UserCodeFormat = p.config.DeviceFlow.UserCodeFormat
		}
		if p.config.DeviceFlow.PollingInterval > 0 {
			deviceFlowConfig.PollingInterval = p.config.DeviceFlow.PollingInterval
		}
		if p.config.DeviceFlow.VerificationURI != "" {
			deviceFlowConfig.VerificationURI = p.config.DeviceFlow.VerificationURI
		}
		if p.config.DeviceFlow.MaxPollAttempts > 0 {
			deviceFlowConfig.MaxPollAttempts = p.config.DeviceFlow.MaxPollAttempts
		}
		if p.config.DeviceFlow.CleanupInterval != "" {
			if cleanup, err := time.ParseDuration(p.config.DeviceFlow.CleanupInterval); err == nil {
				deviceFlowConfig.CleanupInterval = cleanup
			}
		}

		// Create device flow service
		deviceFlowSvc := deviceflow.NewService(deviceCodeRepo, deviceFlowConfig)
		p.service.SetDeviceFlowService(deviceFlowSvc)

		// Start device code cleanup background job
		p.startDeviceCodeCleanup(deviceFlowSvc, deviceFlowConfig.CleanupInterval)

		p.logger.Info("device flow enabled",
			forge.F("verification_uri", deviceFlowConfig.VerificationURI),
			forge.F("polling_interval", deviceFlowConfig.PollingInterval))
	}

	// Start automatic key rotation
	p.service.StartKeyRotation()

	// Register JWT validation strategy if enabled
	if p.config.EnableJWTValidation {
		// Use OIDC provider's own JWT service (has the signing keys!)
		if p.service == nil || p.service.jwtService == nil {
			p.logger.Error("OIDC JWT service not available, JWT validation disabled")
		} else {
			jwtStrategy := NewJWTValidationStrategy(
				p.service.jwtService,
				userSvc,
				p.config.Issuer,
				p.config.Audience,
				p.logger,
			)

			if err := authInst.RegisterAuthStrategy(jwtStrategy); err != nil {
				p.logger.Error("failed to register JWT validation strategy", forge.F("error", err.Error()))
			} else {
				p.logger.Info("JWT validation strategy registered", forge.F("issuer", p.config.Issuer))
			}
		}
	}

	// Initialize admin handler
	p.adminHandler = NewAdminHandler(clientRepo, p.service.registration)

	// Get base path from auth instance and construct UI path
	p.basePath = authInst.GetBasePath() // e.g., "/api/identity"
	baseUIPath := p.basePath + "/ui"    // e.g., "/api/identity/ui"

	p.logger.Debug("initializing dashboard extension",
		forge.F("basePath", p.basePath),
		forge.F("baseUIPath", baseUIPath))

	// Initialize dashboard extension
	p.dashboardExt = NewDashboardExtension(
		clientRepo,
		tokenRepo,
		consentRepo,
		deviceCodeRepo,
		p.service,
		p.logger,
	)

	// Set the correct base UI path for page navigation
	p.dashboardExt.SetBaseUIPath(baseUIPath)

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
	// Pass the full base path (e.g., "/api/identity/oauth2") to handler
	fullBasePath := p.basePath + "/oauth2"
	h := NewHandler(p.service, fullBasePath, p.config.DeviceFlow.LoginURL, p.config.DeviceFlow.APIMode)

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
	// DEVICE FLOW ENDPOINTS (RFC 8628)
	// =============================================================================

	if p.service.deviceFlowService != nil {
		// Device authorization endpoint - device requests authorization (RFC 8628)
		// This is called by the device itself, not the user
		grp.POST("/device_authorization", h.DeviceAuthorize,
			forge.WithName("oidc.device.authorize"),
			forge.WithSummary("Device authorization endpoint (RFC 8628)"),
			forge.WithDescription("Initiates device authorization flow for input-constrained devices"),
			forge.WithRequestSchema(DeviceAuthorizationRequest{}),
			forge.WithResponseSchema(200, "Device authorization response", DeviceAuthorizationResponse{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithTags("OIDC", "OAuth2", "DeviceFlow"),
			forge.WithValidation(true),
		)

		// User endpoints - browser-based verification (mounted under /oauth2)
		deviceGroup := grp.Group("/device")

		deviceGroup.GET("", h.DeviceCodeEntry,
			forge.WithName("oidc.device.entry"),
			forge.WithSummary("Device code entry form"),
			forge.WithDescription("Shows form for user to enter device code. In API mode, returns JSON with form data."),
			forge.WithResponseSchema(200, "Device code entry page data", DeviceCodeEntryResponse{}),
			forge.WithTags("OIDC", "OAuth2", "DeviceFlow"),
		)

		deviceGroup.POST("/verify", h.DeviceVerify,
			forge.WithName("oidc.device.verify"),
			forge.WithSummary("Verify device code"),
			forge.WithDescription("Verifies user code and shows consent screen. In API mode, returns verification info."),
			forge.WithRequestSchema(DeviceVerificationRequest{}),
			forge.WithResponseSchema(200, "Verification info", DeviceVerifyResponse{}),
			forge.WithResponseSchema(400, "Invalid code", ErrorResponse{}),
			forge.WithTags("OIDC", "OAuth2", "DeviceFlow"),
		)

		deviceGroup.POST("/authorize", h.DeviceAuthorizeDecision,
			forge.WithName("oidc.device.decision"),
			forge.WithSummary("Authorize or deny device"),
			forge.WithDescription("Handles user's authorization decision. In API mode, returns status."),
			forge.WithRequestSchema(DeviceAuthorizationDecisionRequest{}),
			forge.WithResponseSchema(200, "Authorization result", DeviceDecisionResponse{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithTags("OIDC", "OAuth2", "DeviceFlow"),
		)
	}

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

	p.logger.Debug("OIDC provider routes registered")
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
		(*schema.DeviceCode)(nil),
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

	// Device Code indexes
	_, err = p.db.NewCreateIndex().
		Model((*schema.DeviceCode)(nil)).
		Index("idx_device_codes_device_code").
		Column("device_code").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device_codes device_code index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.DeviceCode)(nil)).
		Index("idx_device_codes_user_code").
		Column("user_code").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device_codes user_code index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.DeviceCode)(nil)).
		Index("idx_device_codes_app_env_org").
		Column("app_id", "environment_id", "organization_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device_codes app_env_org index: %w", err)
	}

	p.logger.Info("OIDC provider migrations completed")
	return nil
}

// DashboardExtension implements the PluginWithDashboardExtension interface
// This allows the dashboard plugin to discover and register our UI components
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	return p.dashboardExt
}

// RegisterExtensions registers the plugin with the extension registry (deprecated - use DashboardExtension)
func (p *Plugin) RegisterExtensions(reg interface{}) error {
	// Try to register dashboard extension
	if dashReg, ok := reg.(interface {
		Register(ext interface{}) error
	}); ok {
		if err := dashReg.Register(p.dashboardExt); err != nil {
			p.logger.Error("failed to register OIDC provider dashboard extension",
				forge.F("error", err.Error()))
			return err
		}
		p.logger.Info("OIDC provider dashboard extension registered")
	}

	return nil
}

// Shutdown performs cleanup when the plugin is shutting down
func (p *Plugin) Shutdown() error {
	if p.service != nil {
		p.service.StopKeyRotation()
	}

	// Stop device code cleanup if running
	if p.deviceCleanupTicker != nil {
		p.deviceCleanupTicker.Stop()
		if p.deviceCleanupDone != nil {
			close(p.deviceCleanupDone)
		}
	}

	p.logger.Info("OIDC provider plugin shutdown complete")
	return nil
}

// startDeviceCodeCleanup starts a background job to clean up expired device codes
func (p *Plugin) startDeviceCodeCleanup(deviceFlowSvc *deviceflow.Service, interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Minute // Default to 5 minutes
	}

	p.deviceCleanupTicker = time.NewTicker(interval)
	p.deviceCleanupDone = make(chan bool)

	go func() {
		p.logger.Info("device code cleanup job started",
			forge.F("interval", interval.String()))

		for {
			select {
			case <-p.deviceCleanupTicker.C:
				ctx := context.Background()

				// Clean up expired device codes
				expiredCount, err := deviceFlowSvc.CleanupExpiredCodes(ctx)
				if err != nil {
					p.logger.Error("failed to cleanup expired device codes",
						forge.F("error", err.Error()))
				} else if expiredCount > 0 {
					p.logger.Debug("cleaned up expired device codes",
						forge.F("count", expiredCount))
				}

				// Clean up old consumed device codes (older than 7 days)
				consumedCount, err := deviceFlowSvc.CleanupOldConsumedCodes(ctx, 7*24*time.Hour)
				if err != nil {
					p.logger.Error("failed to cleanup old consumed device codes",
						forge.F("error", err.Error()))
				} else if consumedCount > 0 {
					p.logger.Debug("cleaned up old consumed device codes",
						forge.F("count", consumedCount))
				}

			case <-p.deviceCleanupDone:
				p.logger.Info("device code cleanup job stopped")
				return
			}
		}
	}()
}
