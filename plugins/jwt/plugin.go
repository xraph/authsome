package jwt

import (
	"fmt"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/forge"
)

// Plugin implements the JWT authentication plugin
type Plugin struct {
	service       *jwt.Service
	handler       *Handler
	logger        forge.Logger
	config        Config
	defaultConfig Config
}

// Config holds the JWT plugin configuration
type Config struct {
	// Issuer is the JWT issuer claim
	Issuer string `json:"issuer"`
	// AccessExpirySeconds is the access token expiry in seconds
	AccessExpirySeconds int `json:"accessExpirySeconds"`
	// RefreshExpirySeconds is the refresh token expiry in seconds
	RefreshExpirySeconds int `json:"refreshExpirySeconds"`
	// SigningAlgorithm is the JWT signing algorithm (HS256, RS256, etc.)
	SigningAlgorithm string `json:"signingAlgorithm"`
	// IncludeAppIDClaim includes app_id in JWT claims
	IncludeAppIDClaim bool `json:"includeAppIDClaim"`
}

// DefaultConfig returns the default JWT plugin configuration
func DefaultConfig() Config {
	return Config{
		Issuer:               "authsome",
		AccessExpirySeconds:  3600,    // 1 hour
		RefreshExpirySeconds: 2592000, // 30 days
		SigningAlgorithm:     "HS256",
		IncludeAppIDClaim:    true,
	}
}

// PluginOption is a functional option for configuring the JWT plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithIssuer sets the JWT issuer
func WithIssuer(issuer string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Issuer = issuer
	}
}

// WithAccessExpiry sets the access token expiry
func WithAccessExpiry(seconds int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AccessExpirySeconds = seconds
	}
}

// WithRefreshExpiry sets the refresh token expiry
func WithRefreshExpiry(seconds int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RefreshExpirySeconds = seconds
	}
}

// WithSigningAlgorithm sets the signing algorithm
func WithSigningAlgorithm(algorithm string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.SigningAlgorithm = algorithm
	}
}

// WithIncludeAppIDClaim sets whether to include app_id claim
func WithIncludeAppIDClaim(include bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.IncludeAppIDClaim = include
	}
}

// NewPlugin creates a new JWT plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Init initializes the JWT plugin
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("jwt plugin requires auth instance")
	}

	forgeApp := authInst.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available for jwt plugin")
	}

	// Initialize logger
	p.logger = forgeApp.Logger().With(forge.F("plugin", "jwt"))

	// Get config manager and bind configuration
	configManager := forgeApp.Config()
	if err := configManager.BindWithDefault("auth.jwt", &p.config, p.defaultConfig); err != nil {
		// Log warning but continue with defaults
		p.logger.Warn("failed to bind JWT config, using defaults",
			forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Get JWT service from registry - it should already be initialized
	serviceRegistry := authInst.GetServiceRegistry()
	p.service = serviceRegistry.JWTService()

	if p.service == nil {
		return fmt.Errorf("JWT service not available in service registry")
	}

	p.handler = NewHandler(p.service)

	p.logger.Info("JWT plugin initialized",
		forge.F("issuer", p.config.Issuer),
		forge.F("access_expiry_seconds", p.config.AccessExpirySeconds),
		forge.F("algorithm", p.config.SigningAlgorithm))

	return nil
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "jwt"
}

// GetHandler returns the JWT handler
func (p *Plugin) GetHandler() *Handler {
	return p.handler
}

// RegisterRoutes registers the JWT plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.handler == nil {
		return fmt.Errorf("jwt plugin not initialized")
	}

	// JWT key management routes
	jwtKeys := router.Group("/jwt/keys")
	{
		jwtKeys.POST("", p.handler.CreateJWTKey,
			forge.WithName("jwt.keys.create"),
			forge.WithSummary("Create JWT key"),
			forge.WithDescription("Create a new JWT signing key for token generation and verification"),
			forge.WithRequestSchema(jwt.CreateJWTKeyRequest{}),
			forge.WithResponseSchema(200, "JWT key created", jwt.JWTKey{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
			forge.WithTags("JWT", "Keys"),
			forge.WithValidation(true),
		)

		jwtKeys.GET("", p.handler.ListJWTKeys,
			forge.WithName("jwt.keys.list"),
			forge.WithSummary("List JWT keys"),
			forge.WithDescription("List all JWT signing keys for the app with pagination"),
			forge.WithResponseSchema(200, "JWT keys retrieved", jwt.ListJWTKeysResponse{}),
			forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
			forge.WithTags("JWT", "Keys"),
		)
	}

	// JWT token routes
	jwtTokens := router.Group("/jwt")
	{
		jwtTokens.POST("/generate", p.handler.GenerateToken,
			forge.WithName("jwt.generate"),
			forge.WithSummary("Generate JWT token"),
			forge.WithDescription("Generate a new JWT token for authenticated access. Requires valid session or API key."),
			forge.WithRequestSchema(jwt.GenerateTokenRequest{}),
			forge.WithResponseSchema(200, "Token generated", jwt.GenerateTokenResponse{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", ErrorResponse{}),
			forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
			forge.WithTags("JWT", "Tokens"),
			forge.WithValidation(true),
		)

		jwtTokens.POST("/verify", p.handler.VerifyToken,
			forge.WithName("jwt.verify"),
			forge.WithSummary("Verify JWT token"),
			forge.WithDescription("Verify the validity and signature of a JWT token"),
			forge.WithRequestSchema(jwt.VerifyTokenRequest{}),
			forge.WithResponseSchema(200, "Token verified", jwt.VerifyTokenResponse{}),
			forge.WithResponseSchema(400, "Invalid request", ErrorResponse{}),
			forge.WithResponseSchema(401, "Invalid or expired token", ErrorResponse{}),
			forge.WithTags("JWT", "Tokens"),
			forge.WithValidation(true),
		)

		jwtTokens.GET("/jwks", p.handler.GetJWKS,
			forge.WithName("jwt.jwks"),
			forge.WithSummary("Get JSON Web Key Set (JWKS)"),
			forge.WithDescription("Retrieve the public keys used for JWT signature verification in JWKS format (RFC 7517)"),
			forge.WithResponseSchema(200, "JWKS retrieved", jwt.JWKSResponse{}),
			forge.WithResponseSchema(500, "Internal server error", ErrorResponse{}),
			forge.WithTags("JWT", "Keys"),
		)
	}

	p.logger.Info("JWT routes registered")
	return nil
}

// RegisterHooks registers plugin hooks (no-op for JWT)
func (p *Plugin) RegisterHooks(_ *hooks.HookRegistry) error {
	return nil
}

// RegisterServiceDecorators registers service decorators (no-op for JWT)
func (p *Plugin) RegisterServiceDecorators(_ *registry.ServiceRegistry) error {
	return nil
}

// Migrate runs plugin migrations (no-op for JWT - migrations handled at app level)
func (p *Plugin) Migrate() error {
	return nil
}

// ErrorResponse represents an error response for JWT operations - use shared response from core
type ErrorResponse = responses.ErrorResponse
