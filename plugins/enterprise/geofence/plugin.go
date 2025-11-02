package geofence

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

// Plugin implements the AuthSome plugin interface for geofencing
type Plugin struct {
	service           *Service
	config            *Config
	handler           *Handler
	middleware        *Middleware
	repo              Repository
	geoProvider       GeoProvider
	detectionProvider DetectionProvider
}

// NewPlugin creates a new geofencing plugin
func NewPlugin() *Plugin {
	return &Plugin{
		config: DefaultConfig(),
	}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "geofence"
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "Geographic Fencing & Location Security"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Enterprise geofencing with location-based access control, VPN/proxy detection, travel notifications, and GPS authentication"
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
	if err := configManager.Bind("auth.geofence", &config); err != nil {
		// Use defaults if binding fails
		config = *DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid geofencing configuration: %w", err)
	}
	p.config = &config

	if !p.config.Enabled {
		fmt.Println("[Geofence Plugin] Disabled in configuration")
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
			fmt.Printf("[Geofence Plugin] Warning: detection provider %s not available\n", config.Detection.Provider)
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
	)

	// Initialize handler
	p.handler = NewHandler(p.service, p.config)

	// Initialize middleware
	p.middleware = NewMiddleware(p.service, p.config)

	fmt.Printf("[Geofence Plugin] Initialized with provider: %s\n", p.geoProvider.Name())

	return nil
}

// createGeoProvider creates a geolocation provider based on configuration
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
		fmt.Printf("[Geofence Plugin] Unknown provider %s, using static provider\n", name)
		return nil
	}
}

// createDetectionProvider creates a detection provider based on configuration
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

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if !p.config.Enabled || p.handler == nil {
		return nil
	}

	basePath := p.config.API.BasePath

	// Rule Management
	if p.config.API.EnableManagement {
		router.POST(basePath+"/rules", p.handler.CreateRule)
		router.GET(basePath+"/rules", p.handler.ListRules)
		router.GET(basePath+"/rules/:id", p.handler.GetRule)
		router.PUT(basePath+"/rules/:id", p.handler.UpdateRule)
		router.DELETE(basePath+"/rules/:id", p.handler.DeleteRule)
	}

	// Validation
	if p.config.API.EnableValidation {
		router.POST(basePath+"/check", p.handler.CheckLocation)
		router.GET(basePath+"/lookup/:ip", p.handler.LookupIP)
	}

	// Location Events & History
	router.GET(basePath+"/events", p.handler.ListLocationEvents)
	router.GET(basePath+"/events/:id", p.handler.GetLocationEvent)

	// Travel Alerts
	router.GET(basePath+"/travel-alerts", p.handler.ListTravelAlerts)
	router.GET(basePath+"/travel-alerts/:id", p.handler.GetTravelAlert)
	router.POST(basePath+"/travel-alerts/:id/approve", p.handler.ApproveTravelAlert)
	router.POST(basePath+"/travel-alerts/:id/deny", p.handler.DenyTravelAlert)

	// Trusted Locations
	router.POST(basePath+"/trusted-locations", p.handler.CreateTrustedLocation)
	router.GET(basePath+"/trusted-locations", p.handler.ListTrustedLocations)
	router.GET(basePath+"/trusted-locations/:id", p.handler.GetTrustedLocation)
	router.PUT(basePath+"/trusted-locations/:id", p.handler.UpdateTrustedLocation)
	router.DELETE(basePath+"/trusted-locations/:id", p.handler.DeleteTrustedLocation)

	// Violations
	router.GET(basePath+"/violations", p.handler.ListViolations)
	router.GET(basePath+"/violations/:id", p.handler.GetViolation)
	router.POST(basePath+"/violations/:id/resolve", p.handler.ResolveViolation)

	// Metrics & Analytics
	if p.config.API.EnableMetrics {
		router.GET(basePath+"/metrics", p.handler.GetMetrics)
		router.GET(basePath+"/analytics/locations", p.handler.GetLocationAnalytics)
		router.GET(basePath+"/analytics/violations", p.handler.GetViolationAnalytics)
	}

	fmt.Printf("[Geofence Plugin] Registered routes under %s\n", basePath)

	return nil
}

// Middleware returns the geofence middleware for automatic checks
func (p *Plugin) Middleware() func(next func(forge.Context) error) func(forge.Context) error {
	if p.middleware == nil {
		return func(next func(forge.Context) error) func(forge.Context) error {
			return next
		}
	}
	return p.middleware.CheckGeofence
}

// RegisterHooks registers plugin hooks with the hook registry
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if !p.config.Enabled {
		return nil
	}

	// Register authentication hooks for automatic geofence checking
	// hookRegistry.RegisterBeforeSignIn(p.onBeforeSignIn)
	// hookRegistry.RegisterAfterSignIn(p.onAfterSignIn)

	return nil
}

// RegisterServiceDecorators allows plugins to replace core services with decorated versions
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Geofence plugin doesn't decorate core services
	// It provides its own service that's accessed via the plugin
	return nil
}

// Migrate performs database migrations
func (p *Plugin) Migrate() error {
	// Database migrations will be handled by migration system
	// The schema is defined in schema.go and will be registered in migrations/
	return nil
}

// Service returns the geofencing service for direct access
func (p *Plugin) Service() *Service {
	return p.service
}

// Config returns the plugin configuration
func (p *Plugin) Config() *Config {
	return p.config
}

// Shutdown cleanly shuts down the plugin
func (p *Plugin) Shutdown(ctx context.Context) error {
	if p.service != nil {
		// Cleanup expired cache entries
		_, _ = p.repo.DeleteExpiredCache(ctx)
	}
	return nil
}

// Health checks plugin health
func (p *Plugin) Health(ctx context.Context) error {
	if !p.config.Enabled {
		return nil
	}

	if p.service == nil {
		return fmt.Errorf("service not initialized")
	}

	if p.geoProvider == nil {
		return fmt.Errorf("geolocation provider not initialized")
	}

	// Test geolocation provider with a known IP
	_, err := p.geoProvider.Lookup(ctx, "8.8.8.8")
	if err != nil {
		return fmt.Errorf("geolocation provider health check failed: %w", err)
	}

	return nil
}

