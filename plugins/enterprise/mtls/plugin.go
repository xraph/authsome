package mtls

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Plugin implements the AuthSome plugin interface for mTLS.
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

// NewPlugin creates a new mTLS plugin.
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return "mtls"
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "Certificate-Based Authentication (mTLS)"
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "Enterprise mTLS authentication with X.509 certificates, PIV/CAC smart cards, and HSM integration"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Init initializes the plugin with AuthSome dependencies.
func (p *Plugin) Init(auth any) error {
	// Type assert to get the auth instance with required methods
	authInstance, ok := auth.(interface {
		GetDB() *bun.DB
		GetForgeApp() forge.App
		GetServiceRegistry() *registry.ServiceRegistry
	})
	if !ok {
		return errs.InternalServerErrorWithMessage("invalid auth instance type")
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

// RegisterRoutes registers HTTP routes for the plugin.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	basePath := p.config.API.BasePath

	// Certificate Management
	if p.config.API.EnableManagement {
		if err := router.POST(basePath+"/certificates", p.handler.RegisterCertificate,
			forge.WithName("mtls.certificates.register"),
			forge.WithSummary("Register certificate"),
			forge.WithDescription("Registers a new X.509 certificate for mTLS authentication"),
			forge.WithResponseSchema(201, "Certificate registered", MTLSCertificateResponse{}),
			forge.WithResponseSchema(400, "Invalid request", mTLSErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Certificates"),
			forge.WithValidation(true),
		); err != nil {
			return err
		}
		if err := router.GET(basePath+"/certificates", p.handler.ListCertificates,
			forge.WithName("mtls.certificates.list"),
			forge.WithSummary("List certificates"),
			forge.WithDescription("Lists all registered certificates for the authenticated user or organization"),
			forge.WithResponseSchema(200, "Certificates retrieved", MTLSCertificateListResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Certificates"),
		); err != nil {
			return err
		}
		if err := router.GET(basePath+"/certificates/:id", p.handler.GetCertificate,
			forge.WithName("mtls.certificates.get"),
			forge.WithSummary("Get certificate"),
			forge.WithDescription("Retrieves details of a specific certificate by ID"),
			forge.WithResponseSchema(200, "Certificate retrieved", MTLSCertificateResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithResponseSchema(404, "Certificate not found", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Certificates"),
		); err != nil {
			return err
		}
		if err := router.POST(basePath+"/certificates/:id/revoke", p.handler.RevokeCertificate,
			forge.WithName("mtls.certificates.revoke"),
			forge.WithSummary("Revoke certificate"),
			forge.WithDescription("Revokes a certificate by ID. Certificate can no longer be used for authentication"),
			forge.WithResponseSchema(200, "Certificate revoked", mTLSStatusResponse{}),
			forge.WithResponseSchema(400, "Invalid request", mTLSErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithResponseSchema(404, "Certificate not found", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Certificates"),
		); err != nil {
			return err
		}
		if err := router.GET(basePath+"/certificates/expiring", p.handler.GetExpiringCertificates,
			forge.WithName("mtls.certificates.expiring"),
			forge.WithSummary("Get expiring certificates"),
			forge.WithDescription("Lists certificates that are expiring within the configured warning period"),
			forge.WithResponseSchema(200, "Expiring certificates retrieved", MTLSCertificateListResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Certificates"),
		); err != nil {
			return err
		}
	}

	// Authentication
	if err := router.POST(basePath+"/authenticate", p.handler.AuthenticateWithCertificate,
		forge.WithName("mtls.authenticate"),
		forge.WithSummary("Authenticate with certificate"),
		forge.WithDescription("Authenticates using client certificate from TLS connection. Requires valid X.509 certificate"),
		forge.WithResponseSchema(200, "Authentication successful", mTLSAuthResponse{}),
		forge.WithResponseSchema(400, "Invalid request", mTLSErrorResponse{}),
		forge.WithResponseSchema(401, "Authentication failed", mTLSErrorResponse{}),
		forge.WithTags("mTLS", "Authentication"),
	); err != nil {
		return err
	}

	// Trust Anchors
	if p.config.API.EnableManagement {
		if err := router.POST(basePath+"/trust-anchors", p.handler.AddTrustAnchor,
			forge.WithName("mtls.trustanchors.add"),
			forge.WithSummary("Add trust anchor"),
			forge.WithDescription("Adds a new trust anchor (CA certificate) for certificate validation"),
			forge.WithResponseSchema(201, "Trust anchor added", mTLSTrustAnchorResponse{}),
			forge.WithResponseSchema(400, "Invalid request", mTLSErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "TrustAnchors"),
			forge.WithValidation(true),
		); err != nil {
			return err
		}
		if err := router.GET(basePath+"/trust-anchors", p.handler.GetTrustAnchors,
			forge.WithName("mtls.trustanchors.list"),
			forge.WithSummary("List trust anchors"),
			forge.WithDescription("Lists all configured trust anchors (CA certificates)"),
			forge.WithResponseSchema(200, "Trust anchors retrieved", mTLSTrustAnchorListResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "TrustAnchors"),
		); err != nil {
			return err
		}
	}

	// Policies
	if p.config.API.EnableManagement {
		if err := router.POST(basePath+"/policies", p.handler.CreatePolicy,
			forge.WithName("mtls.policies.create"),
			forge.WithSummary("Create certificate policy"),
			forge.WithDescription("Creates a new certificate validation policy with rules and constraints"),
			forge.WithResponseSchema(201, "Policy created", mTLSPolicyResponse{}),
			forge.WithResponseSchema(400, "Invalid request", mTLSErrorResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Policies"),
			forge.WithValidation(true),
		); err != nil {
			return err
		}
		if err := router.GET(basePath+"/policies/:id", p.handler.GetPolicy,
			forge.WithName("mtls.policies.get"),
			forge.WithSummary("Get certificate policy"),
			forge.WithDescription("Retrieves details of a specific certificate validation policy"),
			forge.WithResponseSchema(200, "Policy retrieved", mTLSPolicyResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithResponseSchema(404, "Policy not found", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Policies"),
		); err != nil {
			return err
		}
	}

	// Validation
	if p.config.API.EnableValidation {
		if err := router.POST(basePath+"/validate", p.handler.ValidateCertificate,
			forge.WithName("mtls.validate"),
			forge.WithSummary("Validate certificate"),
			forge.WithDescription("Validates a certificate without authenticating. Checks signature, expiration, revocation, and policy compliance"),
			forge.WithResponseSchema(200, "Certificate validated", mTLSValidationResponse{}),
			forge.WithResponseSchema(400, "Invalid request", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Validation"),
			forge.WithValidation(true),
		); err != nil {
			return err
		}
	}

	// Statistics
	if p.config.API.EnableMetrics {
		if err := router.GET(basePath+"/stats/auth", p.handler.GetAuthStats,
			forge.WithName("mtls.stats.auth"),
			forge.WithSummary("Get authentication statistics"),
			forge.WithDescription("Returns mTLS authentication statistics including success rates, certificate usage, and error counts"),
			forge.WithResponseSchema(200, "Statistics retrieved", mTLSStatsResponse{}),
			forge.WithResponseSchema(401, "Unauthorized", mTLSErrorResponse{}),
			forge.WithTags("mTLS", "Statistics"),
		); err != nil {
			return err
		}
	}

	return nil
}

// registerHooks registers lifecycle hooks.
func (p *Plugin) registerHooks(hookRegistry *hooks.HookRegistry) error {
	// Register authentication hooks if needed
	// hookRegistry.RegisterBeforeSignIn(p.onBeforeSignIn)
	// hookRegistry.RegisterAfterSignIn(p.onAfterSignIn)
	return nil
}

// RegisterHooks registers plugin hooks with the hook registry (implements Plugin interface).
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	return p.registerHooks(hookRegistry)
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions.
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// mTLS plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations.
func (p *Plugin) Migrate() error {
	// Database migrations will be handled by migration system
	// The schema is defined in schema.go and will be registered in migrations/
	return nil
}

// Service returns the mTLS service for direct access (optional public method).
func (p *Plugin) Service() *Service {
	return p.service
}

// Validator returns the certificate validator for direct access (optional public method).
func (p *Plugin) Validator() *CertificateValidator {
	return p.validator
}

// SmartCardProvider returns the smart card provider for direct access (optional public method).
func (p *Plugin) SmartCardProvider() *SmartCardProvider {
	return p.smartCard
}

// HSMManager returns the HSM manager for direct access (optional public method).
func (p *Plugin) HSMManager() *HSMManager {
	return p.hsmManager
}

// Shutdown cleanly shuts down the plugin.
func (p *Plugin) Shutdown() error {
	if p.hsmManager != nil {
		return p.hsmManager.Shutdown()
	}

	return nil
}

// Response types for mTLS routes.
type mTLSErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

type mTLSStatusResponse struct {
	Status string `example:"success" json:"status"`
}

type MTLSCertificateResponse struct {
	Certificate any `json:"certificate"`
}

type MTLSCertificateListResponse struct {
	Certificates []any `json:"certificates"`
}

type mTLSAuthResponse struct {
	Success       bool     `example:"true"          json:"success"`
	UserID        string   `example:"01HZ..."       json:"userId,omitempty"`
	CertificateID string   `example:"cert_123"      json:"certificateId,omitempty"`
	Errors        []string `json:"errors,omitempty"`
}

type mTLSTrustAnchorResponse struct {
	TrustAnchor any `json:"trust_anchor"`
}

type mTLSTrustAnchorListResponse struct {
	TrustAnchors []any `json:"trust_anchors"`
}

type mTLSPolicyResponse struct {
	Policy any `json:"policy"`
}

type mTLSValidationResponse struct {
	Valid    bool     `example:"true"            json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type mTLSStatsResponse struct {
	Stats any `json:"stats"`
}
