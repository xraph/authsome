package geofence

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Plugin implements the AuthSome plugin interface for geofencing.
type Plugin struct {
	service           *Service
	config            *Config
	handler           *Handler
	middleware        *Middleware
	repo              Repository
	geoProvider       GeoProvider
	detectionProvider DetectionProvider
}

// NewPlugin creates a new geofencing plugin.
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier.
func (p *Plugin) ID() string {
	return "geofence"
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "Geographic Fencing & Location Security"
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "Enterprise geofencing with location-based access control, VPN/proxy detection, travel notifications, and GPS authentication"
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
	if err := configManager.Bind("auth.geofence", &config); err != nil {
		// Use defaults if binding fails
		config = *DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid geofencing configuration: %w", err)
	}

	p.config = &config

	if !p.config.Enabled {
		return nil
	}

	// Initialize repository
	p.repo = NewBunRepository(db)

	// Initialize geolocation provider
	p.geoProvider = p.createGeoProvider(config.Geolocation.Provider)
	if p.geoProvider == nil {
		return fmt.Errorf("failed to create geolocation provider: %s", config.Geolocation.Provider)
	}

	// Initialize detection provider
	if config.Detection.DetectVPN || config.Detection.DetectProxy || config.Detection.DetectTor {
		p.detectionProvider = p.createDetectionProvider(config.Detection.Provider)
		if p.detectionProvider == nil {
		}
	}

	// Get services from registry
	auditSvc := serviceRegistry.AuditService()
	notificationSvc := serviceRegistry.NotificationService()

	// Initialize service
	p.service = NewService(
		p.config,
		p.repo,
		p.geoProvider,
		p.detectionProvider,
		auditSvc,
		notificationSvc,
		auth, // Pass auth instance for service registry access
	)

	// Initialize handler
	p.handler = NewHandler(p.service, p.config)

	// Initialize middleware
	p.middleware = NewMiddleware(p.service, p.config)

	return nil
}

// createGeoProvider creates a geolocation provider based on configuration.
func (p *Plugin) createGeoProvider(name string) GeoProvider {
	switch strings.ToLower(name) {
	case "maxmind":
		return NewMaxMindProvider(
			p.config.Geolocation.MaxMindLicenseKey,
			p.config.Geolocation.MaxMindDatabasePath,
		)
	case "ipapi":
		return NewIPAPIProvider(p.config.Geolocation.IPAPIKey)
	case "ipinfo":
		return NewIPInfoProvider(p.config.Geolocation.IPInfoToken)
	case "ipgeolocation":
		return NewIPGeolocationProvider(p.config.Geolocation.IPGeolocationKey)
	default:
		// Fallback to a basic provider
		return nil
	}
}

// createDetectionProvider creates a detection provider based on configuration.
func (p *Plugin) createDetectionProvider(name string) DetectionProvider {
	switch strings.ToLower(name) {
	case "ipqs":
		if p.config.Detection.IPQSKey == "" {
			return nil
		}

		return NewIPQSProvider(
			p.config.Detection.IPQSKey,
			p.config.Detection.IPQSStrictness,
			p.config.Detection.IPQSMinScore,
		)
	case "proxycheck":
		if p.config.Detection.ProxyCheckKey == "" {
			return nil
		}

		return NewProxyCheckProvider(p.config.Detection.ProxyCheckKey)
	case "vpnapi":
		if p.config.Detection.VPNAPIKey == "" {
			return nil
		}

		return NewVPNAPIProvider(p.config.Detection.VPNAPIKey)
	case "static":
		return NewStaticDetectionProvider()
	default:
		return nil
	}
}

// RegisterRoutes registers HTTP routes for the plugin.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	basePath := p.config.API.BasePath

	// Rule Management
	if p.config.API.EnableManagement {
		router.POST(basePath+"/rules", p.handler.CreateRule,
			forge.WithName("geofence.rules.create"),
			forge.WithSummary("Create geofence rule"),
			forge.WithDescription("Create a new geographic restriction rule for country/region-based access control"),
			forge.WithResponseSchema(200, "Rule created", GeofenceRuleResponse{}),
			forge.WithResponseSchema(400, "Invalid request", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Rules"),
			forge.WithValidation(true),
		)
		router.GET(basePath+"/rules", p.handler.ListRules,
			forge.WithName("geofence.rules.list"),
			forge.WithSummary("List geofence rules"),
			forge.WithDescription("List all geographic restriction rules with optional filtering"),
			forge.WithResponseSchema(200, "Rules retrieved", GeofenceRulesResponse{}),
			forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Rules"),
		)
		router.GET(basePath+"/rules/:id", p.handler.GetRule,
			forge.WithName("geofence.rules.get"),
			forge.WithSummary("Get geofence rule"),
			forge.WithDescription("Retrieve a specific geofence rule by ID"),
			forge.WithResponseSchema(200, "Rule retrieved", GeofenceRuleResponse{}),
			forge.WithResponseSchema(404, "Rule not found", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Rules"),
		)
		router.PUT(basePath+"/rules/:id", p.handler.UpdateRule,
			forge.WithName("geofence.rules.update"),
			forge.WithSummary("Update geofence rule"),
			forge.WithDescription("Update an existing geofence rule's countries, regions, or behavior"),
			forge.WithResponseSchema(200, "Rule updated", GeofenceRuleResponse{}),
			forge.WithResponseSchema(400, "Invalid request", GeofenceErrorResponse{}),
			forge.WithResponseSchema(404, "Rule not found", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Rules"),
			forge.WithValidation(true),
		)
		router.DELETE(basePath+"/rules/:id", p.handler.DeleteRule,
			forge.WithName("geofence.rules.delete"),
			forge.WithSummary("Delete geofence rule"),
			forge.WithDescription("Remove a geofence rule"),
			forge.WithResponseSchema(200, "Rule deleted", GeofenceStatusResponse{}),
			forge.WithResponseSchema(404, "Rule not found", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Rules"),
		)
	}

	// Validation
	if p.config.API.EnableValidation {
		router.POST(basePath+"/check", p.handler.CheckLocation,
			forge.WithName("geofence.check"),
			forge.WithSummary("Check location"),
			forge.WithDescription("Validate if a location (IP or coordinates) is allowed by geofence rules"),
			forge.WithResponseSchema(200, "Location checked", GeofenceCheckResponse{}),
			forge.WithResponseSchema(400, "Invalid request", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Validation"),
			forge.WithValidation(true),
		)
		router.GET(basePath+"/lookup/:ip", p.handler.LookupIP,
			forge.WithName("geofence.lookup"),
			forge.WithSummary("Lookup IP location"),
			forge.WithDescription("Get geographic information for an IP address"),
			forge.WithResponseSchema(200, "IP lookup successful", GeofenceLookupResponse{}),
			forge.WithResponseSchema(400, "Invalid IP address", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Validation"),
		)
	}

	// Location Events & History
	router.GET(basePath+"/events", p.handler.ListLocationEvents,
		forge.WithName("geofence.events.list"),
		forge.WithSummary("List location events"),
		forge.WithDescription("Retrieve history of location-based authentication events"),
		forge.WithResponseSchema(200, "Events retrieved", GeofenceEventsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Events"),
	)
	router.GET(basePath+"/events/:id", p.handler.GetLocationEvent,
		forge.WithName("geofence.events.get"),
		forge.WithSummary("Get location event"),
		forge.WithDescription("Retrieve details of a specific location event"),
		forge.WithResponseSchema(200, "Event retrieved", GeofenceEventResponse{}),
		forge.WithResponseSchema(404, "Event not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Events"),
	)

	// Travel Alerts
	router.GET(basePath+"/travel-alerts", p.handler.ListTravelAlerts,
		forge.WithName("geofence.travel.list"),
		forge.WithSummary("List travel alerts"),
		forge.WithDescription("List pending travel notifications requiring approval"),
		forge.WithResponseSchema(200, "Travel alerts retrieved", GeofenceTravelAlertsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Travel"),
	)
	router.GET(basePath+"/travel-alerts/:id", p.handler.GetTravelAlert,
		forge.WithName("geofence.travel.get"),
		forge.WithSummary("Get travel alert"),
		forge.WithDescription("Retrieve details of a specific travel alert"),
		forge.WithResponseSchema(200, "Travel alert retrieved", GeofenceTravelAlertResponse{}),
		forge.WithResponseSchema(404, "Travel alert not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Travel"),
	)
	router.POST(basePath+"/travel-alerts/:id/approve", p.handler.ApproveTravelAlert,
		forge.WithName("geofence.travel.approve"),
		forge.WithSummary("Approve travel alert"),
		forge.WithDescription("Approve a travel notification to allow access from the new location"),
		forge.WithResponseSchema(200, "Travel approved", GeofenceStatusResponse{}),
		forge.WithResponseSchema(404, "Travel alert not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Travel"),
	)
	router.POST(basePath+"/travel-alerts/:id/deny", p.handler.DenyTravelAlert,
		forge.WithName("geofence.travel.deny"),
		forge.WithSummary("Deny travel alert"),
		forge.WithDescription("Deny a travel notification to block access from the new location"),
		forge.WithResponseSchema(200, "Travel denied", GeofenceStatusResponse{}),
		forge.WithResponseSchema(404, "Travel alert not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Travel"),
	)

	// Trusted Locations
	router.POST(basePath+"/trusted-locations", p.handler.CreateTrustedLocation,
		forge.WithName("geofence.trusted.create"),
		forge.WithSummary("Add trusted location"),
		forge.WithDescription("Mark a location as trusted for a user to bypass geofence restrictions"),
		forge.WithResponseSchema(200, "Trusted location created", GeofenceTrustedLocationResponse{}),
		forge.WithResponseSchema(400, "Invalid request", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Trusted Locations"),
		forge.WithValidation(true),
	)
	router.GET(basePath+"/trusted-locations", p.handler.ListTrustedLocations,
		forge.WithName("geofence.trusted.list"),
		forge.WithSummary("List trusted locations"),
		forge.WithDescription("List all trusted locations for users"),
		forge.WithResponseSchema(200, "Trusted locations retrieved", GeofenceTrustedLocationsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Trusted Locations"),
	)
	router.GET(basePath+"/trusted-locations/:id", p.handler.GetTrustedLocation,
		forge.WithName("geofence.trusted.get"),
		forge.WithSummary("Get trusted location"),
		forge.WithDescription("Retrieve details of a specific trusted location"),
		forge.WithResponseSchema(200, "Trusted location retrieved", GeofenceTrustedLocationResponse{}),
		forge.WithResponseSchema(404, "Trusted location not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Trusted Locations"),
	)
	router.PUT(basePath+"/trusted-locations/:id", p.handler.UpdateTrustedLocation,
		forge.WithName("geofence.trusted.update"),
		forge.WithSummary("Update trusted location"),
		forge.WithDescription("Update a trusted location's details or expiration"),
		forge.WithResponseSchema(200, "Trusted location updated", GeofenceTrustedLocationResponse{}),
		forge.WithResponseSchema(400, "Invalid request", GeofenceErrorResponse{}),
		forge.WithResponseSchema(404, "Trusted location not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Trusted Locations"),
		forge.WithValidation(true),
	)
	router.DELETE(basePath+"/trusted-locations/:id", p.handler.DeleteTrustedLocation,
		forge.WithName("geofence.trusted.delete"),
		forge.WithSummary("Remove trusted location"),
		forge.WithDescription("Remove a trusted location"),
		forge.WithResponseSchema(200, "Trusted location deleted", GeofenceStatusResponse{}),
		forge.WithResponseSchema(404, "Trusted location not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Trusted Locations"),
	)

	// Violations
	router.GET(basePath+"/violations", p.handler.ListViolations,
		forge.WithName("geofence.violations.list"),
		forge.WithSummary("List geofence violations"),
		forge.WithDescription("List all geofence policy violations with details"),
		forge.WithResponseSchema(200, "Violations retrieved", GeofenceViolationsResponse{}),
		forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Violations"),
	)
	router.GET(basePath+"/violations/:id", p.handler.GetViolation,
		forge.WithName("geofence.violations.get"),
		forge.WithSummary("Get geofence violation"),
		forge.WithDescription("Retrieve details of a specific geofence violation"),
		forge.WithResponseSchema(200, "Violation retrieved", GeofenceViolationResponse{}),
		forge.WithResponseSchema(404, "Violation not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Violations"),
	)
	router.POST(basePath+"/violations/:id/resolve", p.handler.ResolveViolation,
		forge.WithName("geofence.violations.resolve"),
		forge.WithSummary("Resolve violation"),
		forge.WithDescription("Mark a geofence violation as resolved with optional notes"),
		forge.WithResponseSchema(200, "Violation resolved", GeofenceStatusResponse{}),
		forge.WithResponseSchema(404, "Violation not found", GeofenceErrorResponse{}),
		forge.WithTags("Geofence", "Violations"),
	)

	// Metrics & Analytics
	if p.config.API.EnableMetrics {
		router.GET(basePath+"/metrics", p.handler.GetMetrics,
			forge.WithName("geofence.metrics"),
			forge.WithSummary("Get geofence metrics"),
			forge.WithDescription("Retrieve geofence usage metrics and statistics"),
			forge.WithResponseSchema(200, "Metrics retrieved", GeofenceMetricsResponse{}),
			forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Analytics"),
		)
		router.GET(basePath+"/analytics/locations", p.handler.GetLocationAnalytics,
			forge.WithName("geofence.analytics.locations"),
			forge.WithSummary("Get location analytics"),
			forge.WithDescription("Analyze authentication patterns by geographic location"),
			forge.WithResponseSchema(200, "Location analytics retrieved", GeofenceLocationAnalyticsResponse{}),
			forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Analytics"),
		)
		router.GET(basePath+"/analytics/violations", p.handler.GetViolationAnalytics,
			forge.WithName("geofence.analytics.violations"),
			forge.WithSummary("Get violation analytics"),
			forge.WithDescription("Analyze geofence violation patterns and trends"),
			forge.WithResponseSchema(200, "Violation analytics retrieved", GeofenceViolationAnalyticsResponse{}),
			forge.WithResponseSchema(500, "Internal server error", GeofenceErrorResponse{}),
			forge.WithTags("Geofence", "Analytics"),
		)
	}

	return nil
}

// Middleware returns the geofence middleware for automatic checks.
func (p *Plugin) Middleware() func(next func(forge.Context) error) func(forge.Context) error {
	if p.middleware == nil {
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}

	return p.middleware.CheckGeofence
}

// RegisterHooks registers plugin hooks with the hook registry.
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if !p.config.Enabled {
		return nil
	}

	// Register session security notification hooks
	if p.config.Notifications.Enabled {
		hookRegistry.RegisterAfterSignIn(func(ctx context.Context, response *responses.AuthResponse) error {
			if response == nil || response.User == nil {
				return nil
			}

			// Get AuthContext with complete authentication state
			authCtx, ok := contexts.GetAuthContext(ctx)
			if !ok || authCtx == nil {
				return nil
			}

			// Use typed fields directly from AuthContext
			userID := response.User.ID
			appID := authCtx.AppID
			ipAddress := authCtx.IPAddress

			// Validate required fields
			if appID.IsNil() || ipAddress == "" {
				return nil
			}

			// Perform security checks asynchronously (don't block authentication)
			go func() {
				bgCtx := context.Background()
				// Copy AuthContext to background context
				bgCtx = contexts.SetAuthContext(bgCtx, authCtx)

				if err := p.service.CheckSessionSecurity(bgCtx, userID, appID, ipAddress); err != nil {
				}
			}()

			return nil
		})
	}

	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions.
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Geofence plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations.
func (p *Plugin) Migrate() error {
	// Database migrations will be handled by migration system
	// The schema is defined in schema.go and will be registered in migrations/
	return nil
}

// Service returns the geofencing service for direct access.
func (p *Plugin) Service() *Service {
	return p.service
}

// Config returns the plugin configuration.
func (p *Plugin) Config() *Config {
	return p.config
}

// Shutdown cleanly shuts down the plugin.
func (p *Plugin) Shutdown(ctx context.Context) error {
	if p.service != nil {
		// Cleanup expired cache entries
		_, _ = p.repo.DeleteExpiredCache(ctx)
	}

	return nil
}

// Health checks plugin health.
func (p *Plugin) Health(ctx context.Context) error {
	if !p.config.Enabled {
		return nil
	}

	if p.service == nil {
		return errs.InternalServerErrorWithMessage("service not initialized")
	}

	if p.geoProvider == nil {
		return errs.InternalServerErrorWithMessage("geolocation provider not initialized")
	}

	// Test geolocation provider with a known IP
	_, err := p.geoProvider.Lookup(ctx, "8.8.8.8")
	if err != nil {
		return fmt.Errorf("geolocation provider health check failed: %w", err)
	}

	return nil
}

// DTOs for geofence routes.
type GeofenceErrorResponse struct {
	Error string `example:"Error message" json:"error"`
}

type GeofenceStatusResponse struct {
	Status string `example:"success" json:"status"`
}

type GeofenceRuleResponse struct {
	ID string `example:"rule_123" json:"id"`
}

type GeofenceRulesResponse struct {
	Rules []any `json:"rules"`
}

type GeofenceCheckResponse struct {
	Allowed bool   `example:"true" json:"allowed"`
	Country string `example:"US"   json:"country,omitempty"`
}

type GeofenceLookupResponse struct {
	Country   string  `example:"US"            json:"country"`
	City      string  `example:"San Francisco" json:"city,omitempty"`
	Latitude  float64 `example:"37.7749"       json:"latitude,omitempty"`
	Longitude float64 `example:"-122.4194"     json:"longitude,omitempty"`
}

type GeofenceEventsResponse struct {
	Events []any `json:"events"`
}

type GeofenceEventResponse struct {
	ID string `example:"event_123" json:"id"`
}

type GeofenceTravelAlertsResponse struct {
	TravelAlerts []any `json:"travel_alerts"`
}

type GeofenceTravelAlertResponse struct {
	ID string `example:"alert_123" json:"id"`
}

type GeofenceTrustedLocationResponse struct {
	ID string `example:"trusted_123" json:"id"`
}

type GeofenceTrustedLocationsResponse struct {
	TrustedLocations []any `json:"trusted_locations"`
}

type GeofenceViolationsResponse struct {
	Violations []any `json:"violations"`
}

type GeofenceViolationResponse struct {
	ID string `example:"violation_123" json:"id"`
}

type GeofenceMetricsResponse struct {
	Metrics any `json:"metrics"`
}

type GeofenceLocationAnalyticsResponse struct {
	Analytics any `json:"analytics"`
}

type GeofenceViolationAnalyticsResponse struct {
	Analytics any `json:"analytics"`
}
