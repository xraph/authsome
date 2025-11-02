package mtls

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

// Plugin implements the AuthSome plugin interface for mTLS
type Plugin struct {
	service    *Service
	config     *Config
	handler    *Handler
	validator  *CertificateValidator
	revChecker *RevocationChecker
	smartCard  *SmartCardProvider
	hsmManager *HSMManager
	repo       Repository
}

// NewPlugin creates a new mTLS plugin
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "mtls"
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "Certificate-Based Authentication (mTLS)"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Enterprise mTLS authentication with X.509 certificates, PIV/CAC smart cards, and HSM integration"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Init initializes the plugin with AuthSome dependencies
func (p *Plugin) Init(auth interface{}) error {
	// Type assert to get the auth instance with required methods
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return fmt.Errorf("invalid auth instance type")
	}

	db := authInstance.GetDB()
	forgeApp := authInstance.GetForgeApp()
	configManager := forgeApp.Config()
	serviceRegistry := authInstance.GetServiceRegistry()

	// Load configuration from Forge config manager
	var config Config
	if err := configManager.Bind("auth.mtls", &config); err != nil {
		// Use defaults if binding fails
		config = *DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid mTLS configuration: %w", err)
	}
	p.config = &config

	if !p.config.Enabled {
		return nil
	}

	// Initialize repository
	p.repo = NewBunRepository(db)

	// Initialize revocation checker
	p.revChecker = NewRevocationChecker(p.config, p.repo)

	// Initialize certificate validator
	p.validator = NewCertificateValidator(p.config, p.repo, p.revChecker)

	// Initialize smart card provider
	p.smartCard = NewSmartCardProvider(p.config, p.repo)

	// Initialize HSM manager
	p.hsmManager = NewHSMManager(p.config, p.repo)
	if err := p.hsmManager.Init(context.Background()); err != nil {
		// HSM initialization failure is not fatal - log and continue
		fmt.Printf("Warning: HSM initialization failed: %v\n", err)
	}

	// Initialize service
	p.service = NewService(
		p.config,
		p.repo,
		p.validator,
		p.revChecker,
		p.smartCard,
		p.hsmManager,
	)

	// Initialize handler
	p.handler = NewHandler(p.service)

	// Register hooks
	hookRegistry := serviceRegistry.HookRegistry()
	if err := p.registerHooks(hookRegistry); err != nil {
		return fmt.Errorf("failed to register hooks: %w", err)
	}

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}
	
	basePath := p.config.API.BasePath
	
	// Certificate Management
	if p.config.API.EnableManagement {
		router.POST(basePath+"/certificates", p.handler.RegisterCertificate)
		router.GET(basePath+"/certificates", p.handler.ListCertificates)
		router.GET(basePath+"/certificates/:id", p.handler.GetCertificate)
		router.POST(basePath+"/certificates/:id/revoke", p.handler.RevokeCertificate)
		router.GET(basePath+"/certificates/expiring", p.handler.GetExpiringCertificates)
	}
	
	// Authentication
	router.POST(basePath+"/authenticate", p.handler.AuthenticateWithCertificate)
	
	// Trust Anchors
	if p.config.API.EnableManagement {
		router.POST(basePath+"/trust-anchors", p.handler.AddTrustAnchor)
		router.GET(basePath+"/trust-anchors", p.handler.GetTrustAnchors)
	}
	
	// Policies
	if p.config.API.EnableManagement {
		router.POST(basePath+"/policies", p.handler.CreatePolicy)
		router.GET(basePath+"/policies/:id", p.handler.GetPolicy)
	}
	
	// Validation
	if p.config.API.EnableValidation {
		router.POST(basePath+"/validate", p.handler.ValidateCertificate)
	}
	
	// Statistics
	if p.config.API.EnableMetrics {
		router.GET(basePath+"/stats/auth", p.handler.GetAuthStats)
	}
	
	return nil
}

// registerHooks registers lifecycle hooks
func (p *Plugin) registerHooks(hookRegistry *hooks.HookRegistry) error {
	// Register authentication hooks if needed
	// hookRegistry.RegisterBeforeSignIn(p.onBeforeSignIn)
	// hookRegistry.RegisterAfterSignIn(p.onAfterSignIn)
	
	return nil
}

// RegisterHooks registers plugin hooks with the hook registry (implements Plugin interface)
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	return p.registerHooks(hookRegistry)
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// mTLS plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations
func (p *Plugin) Migrate() error {
	// Database migrations will be handled by migration system
	// The schema is defined in schema.go and will be registered in migrations/
	return nil
}

// Service returns the mTLS service for direct access (optional public method)
func (p *Plugin) Service() *Service {
	return p.service
}

// Validator returns the certificate validator for direct access (optional public method)
func (p *Plugin) Validator() *CertificateValidator {
	return p.validator
}

// SmartCardProvider returns the smart card provider for direct access (optional public method)
func (p *Plugin) SmartCardProvider() *SmartCardProvider {
	return p.smartCard
}

// HSMManager returns the HSM manager for direct access (optional public method)
func (p *Plugin) HSMManager() *HSMManager {
	return p.hsmManager
}

// Shutdown cleanly shuts down the plugin
func (p *Plugin) Shutdown() error {
	if p.hsmManager != nil {
		return p.hsmManager.Shutdown()
	}
	return nil
}

