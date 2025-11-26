// Package subscription provides a comprehensive SaaS subscription and billing plugin for AuthSome.
// It supports multiple billing patterns, payment provider integration (Stripe), and organization-scoped subscriptions.
package subscription

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/providers/mock"
	"github.com/xraph/authsome/plugins/subscription/providers/stripe"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
)

// Plugin implements the subscription plugin for AuthSome
type Plugin struct {
	db     *bun.DB
	config Config
	logger forge.Logger

	// Services
	planSvc         *service.PlanService
	subscriptionSvc *service.SubscriptionService
	addOnSvc        *service.AddOnService
	invoiceSvc      *service.InvoiceService
	usageSvc        *service.UsageService
	paymentSvc      *service.PaymentService
	customerSvc     *service.CustomerService
	enforcementSvc  *service.EnforcementService

	// Repositories
	planRepo    repository.PlanRepository
	subRepo     repository.SubscriptionRepository
	addOnRepo   repository.AddOnRepository
	invoiceRepo repository.InvoiceRepository
	usageRepo   repository.UsageRepository
	paymentRepo repository.PaymentMethodRepository
	customerRepo repository.CustomerRepository
	eventRepo   repository.EventRepository

	// Payment provider
	provider providers.PaymentProvider

	// Hook registries
	hookRegistry     *hooks.HookRegistry
	subHookRegistry  *SubscriptionHookRegistry

	// Dashboard extension
	dashboardExt *DashboardExtension

	// Organization service for enforcement
	orgService *organization.Service

	// Default config (set via options)
	defaultConfig Config
}

// PluginOption is a functional option for configuring the plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithRequireSubscription sets whether subscription is required for org creation
func WithRequireSubscription(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireSubscription = required
	}
}

// WithDefaultTrialDays sets the default trial days
func WithDefaultTrialDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultTrialDays = days
	}
}

// WithGracePeriodDays sets the grace period for failed payments
func WithGracePeriodDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.GracePeriodDays = days
	}
}

// WithStripeConfig sets the Stripe configuration
func WithStripeConfig(secretKey, webhookSecret, publishableKey string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Provider = "stripe"
		p.defaultConfig.StripeConfig = StripeConfig{
			SecretKey:      secretKey,
			WebhookSecret:  webhookSecret,
			PublishableKey: publishableKey,
		}
	}
}

// WithAutoSyncSeats enables automatic seat synchronization
func WithAutoSyncSeats(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoSyncSeats = enabled
	}
}

// NewPlugin creates a new subscription plugin instance
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		defaultConfig:   DefaultConfig(),
		subHookRegistry: NewSubscriptionHookRegistry(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ID returns the unique identifier for this plugin
func (p *Plugin) ID() string {
	return "subscription"
}

// Dependencies declares the plugin dependencies
func (p *Plugin) Dependencies() []string {
	return []string{"organization"}
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return fmt.Errorf("subscription plugin requires auth instance")
	}

	// Get Forge app
	forgeApp := authInstance.GetForgeApp()
	if forgeApp == nil {
		return fmt.Errorf("forge app not available")
	}

	p.logger = forgeApp.Logger().With(forge.F("plugin", "subscription"))
	configManager := forgeApp.Config()

	// Bind plugin configuration
	if err := configManager.BindWithDefault("auth.subscription", &p.config, p.defaultConfig); err != nil {
		p.logger.Warn("failed to bind subscription config", forge.F("error", err.Error()))
		p.config = p.defaultConfig
	}

	// Validate configuration
	if err := p.config.Validate(); err != nil {
		return fmt.Errorf("invalid subscription config: %w", err)
	}

	// Get database
	p.db = authInstance.GetDB()
	if p.db == nil {
		return fmt.Errorf("database not available")
	}

	// Register schema models with Bun
	p.db.RegisterModel(
		(*schema.SubscriptionPlan)(nil),
		(*schema.SubscriptionPlanFeature)(nil),
		(*schema.SubscriptionPlanTier)(nil),
		(*schema.Subscription)(nil),
		(*schema.SubscriptionAddOn)(nil),
		(*schema.SubscriptionAddOnFeature)(nil),
		(*schema.SubscriptionAddOnTier)(nil),
		(*schema.SubscriptionAddOnItem)(nil),
		(*schema.SubscriptionInvoice)(nil),
		(*schema.SubscriptionInvoiceItem)(nil),
		(*schema.SubscriptionUsageRecord)(nil),
		(*schema.SubscriptionPaymentMethod)(nil),
		(*schema.SubscriptionCustomer)(nil),
		(*schema.SubscriptionEvent)(nil),
	)

	// Get hook registry
	p.hookRegistry = authInstance.GetHookRegistry()
	if p.hookRegistry == nil {
		return fmt.Errorf("hook registry not available")
	}

	// Get service registry
	serviceRegistry := authInstance.GetServiceRegistry()
	if serviceRegistry == nil {
		return fmt.Errorf("service registry not available")
	}

	// Get organization service for enforcement
	p.orgService = serviceRegistry.OrganizationService()

	// Initialize repositories
	p.planRepo = repository.NewPlanRepository(p.db)
	p.subRepo = repository.NewSubscriptionRepository(p.db)
	p.addOnRepo = repository.NewAddOnRepository(p.db)
	p.invoiceRepo = repository.NewInvoiceRepository(p.db)
	p.usageRepo = repository.NewUsageRepository(p.db)
	p.paymentRepo = repository.NewPaymentMethodRepository(p.db)
	p.customerRepo = repository.NewCustomerRepository(p.db)
	p.eventRepo = repository.NewEventRepository(p.db)

	// Initialize payment provider
	if p.config.IsStripeConfigured() {
		var err error
		p.provider, err = stripe.NewStripeProvider(
			p.config.StripeConfig.SecretKey,
			p.config.StripeConfig.WebhookSecret,
		)
		if err != nil {
			p.logger.Warn("failed to initialize Stripe provider", forge.F("error", err.Error()))
			// Fall back to mock provider
			p.provider = mock.NewMockProvider()
		} else {
			p.logger.Info("Stripe provider initialized")
		}
	} else {
		// Use mock provider for development
		p.provider = mock.NewMockProvider()
		p.logger.Warn("using mock payment provider - configure Stripe for production")
	}

	// Initialize services
	p.planSvc = service.NewPlanService(p.planRepo, p.provider, p.eventRepo)
	p.customerSvc = service.NewCustomerService(p.customerRepo, p.provider, p.eventRepo)
	p.paymentSvc = service.NewPaymentService(p.paymentRepo, p.customerRepo, p.provider, p.eventRepo)
	p.subscriptionSvc = service.NewSubscriptionService(
		p.subRepo,
		p.planRepo,
		p.customerRepo,
		p.provider,
		p.eventRepo,
		p.subHookRegistry,
		p.config,
	)
	p.addOnSvc = service.NewAddOnService(p.addOnRepo, p.subRepo, p.provider, p.eventRepo)
	p.invoiceSvc = service.NewInvoiceService(p.invoiceRepo, p.subRepo, p.provider, p.eventRepo)
	p.usageSvc = service.NewUsageService(p.usageRepo, p.subRepo, p.provider, p.eventRepo)
	p.enforcementSvc = service.NewEnforcementService(
		p.subRepo,
		p.planRepo,
		p.usageRepo,
		p.orgService,
		p.config,
	)

	// Initialize dashboard extension
	p.dashboardExt = NewDashboardExtension(p)

	p.logger.Info("subscription plugin initialized",
		forge.F("require_subscription", p.config.RequireSubscription),
		forge.F("default_trial_days", p.config.DefaultTrialDays),
		forge.F("provider", p.config.Provider))

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	// Plan routes
	planGroup := router.Group("/subscription/plans")
	{
		planGroup.POST("", p.handleCreatePlan,
			forge.WithName("subscription.plans.create"),
			forge.WithSummary("Create plan"),
			forge.WithDescription("Create a new subscription plan"),
			forge.WithTags("Subscription", "Plans"),
			forge.WithValidation(true),
		)

		planGroup.GET("", p.handleListPlans,
			forge.WithName("subscription.plans.list"),
			forge.WithSummary("List plans"),
			forge.WithDescription("List all subscription plans"),
			forge.WithTags("Subscription", "Plans"),
		)

		planGroup.GET("/:id", p.handleGetPlan,
			forge.WithName("subscription.plans.get"),
			forge.WithSummary("Get plan"),
			forge.WithDescription("Get a specific plan by ID"),
			forge.WithTags("Subscription", "Plans"),
		)

		planGroup.PATCH("/:id", p.handleUpdatePlan,
			forge.WithName("subscription.plans.update"),
			forge.WithSummary("Update plan"),
			forge.WithDescription("Update an existing plan"),
			forge.WithTags("Subscription", "Plans"),
			forge.WithValidation(true),
		)

		planGroup.DELETE("/:id", p.handleDeletePlan,
			forge.WithName("subscription.plans.delete"),
			forge.WithSummary("Delete plan"),
			forge.WithDescription("Delete a plan"),
			forge.WithTags("Subscription", "Plans"),
		)

		planGroup.POST("/:id/sync", p.handleSyncPlan,
			forge.WithName("subscription.plans.sync"),
			forge.WithSummary("Sync plan to provider"),
			forge.WithDescription("Sync a plan to the payment provider"),
			forge.WithTags("Subscription", "Plans"),
		)
	}

	// Subscription routes
	subGroup := router.Group("/subscription/subscriptions")
	{
		subGroup.POST("", p.handleCreateSubscription,
			forge.WithName("subscription.subscriptions.create"),
			forge.WithSummary("Create subscription"),
			forge.WithDescription("Create a new subscription for an organization"),
			forge.WithTags("Subscription", "Subscriptions"),
			forge.WithValidation(true),
		)

		subGroup.GET("", p.handleListSubscriptions,
			forge.WithName("subscription.subscriptions.list"),
			forge.WithSummary("List subscriptions"),
			forge.WithDescription("List all subscriptions"),
			forge.WithTags("Subscription", "Subscriptions"),
		)

		subGroup.GET("/:id", p.handleGetSubscription,
			forge.WithName("subscription.subscriptions.get"),
			forge.WithSummary("Get subscription"),
			forge.WithDescription("Get a specific subscription by ID"),
			forge.WithTags("Subscription", "Subscriptions"),
		)

		subGroup.GET("/organization/:orgId", p.handleGetOrganizationSubscription,
			forge.WithName("subscription.subscriptions.getByOrg"),
			forge.WithSummary("Get organization subscription"),
			forge.WithDescription("Get the active subscription for an organization"),
			forge.WithTags("Subscription", "Subscriptions"),
		)

		subGroup.PATCH("/:id", p.handleUpdateSubscription,
			forge.WithName("subscription.subscriptions.update"),
			forge.WithSummary("Update subscription"),
			forge.WithDescription("Update an existing subscription"),
			forge.WithTags("Subscription", "Subscriptions"),
			forge.WithValidation(true),
		)

		subGroup.POST("/:id/cancel", p.handleCancelSubscription,
			forge.WithName("subscription.subscriptions.cancel"),
			forge.WithSummary("Cancel subscription"),
			forge.WithDescription("Cancel a subscription"),
			forge.WithTags("Subscription", "Subscriptions"),
		)

		subGroup.POST("/:id/pause", p.handlePauseSubscription,
			forge.WithName("subscription.subscriptions.pause"),
			forge.WithSummary("Pause subscription"),
			forge.WithDescription("Pause a subscription"),
			forge.WithTags("Subscription", "Subscriptions"),
		)

		subGroup.POST("/:id/resume", p.handleResumeSubscription,
			forge.WithName("subscription.subscriptions.resume"),
			forge.WithSummary("Resume subscription"),
			forge.WithDescription("Resume a paused subscription"),
			forge.WithTags("Subscription", "Subscriptions"),
		)
	}

	// Add-on routes
	addOnGroup := router.Group("/subscription/addons")
	{
		addOnGroup.POST("", p.handleCreateAddOn,
			forge.WithName("subscription.addons.create"),
			forge.WithSummary("Create add-on"),
			forge.WithDescription("Create a new add-on"),
			forge.WithTags("Subscription", "AddOns"),
			forge.WithValidation(true),
		)

		addOnGroup.GET("", p.handleListAddOns,
			forge.WithName("subscription.addons.list"),
			forge.WithSummary("List add-ons"),
			forge.WithDescription("List all add-ons"),
			forge.WithTags("Subscription", "AddOns"),
		)

		addOnGroup.GET("/:id", p.handleGetAddOn,
			forge.WithName("subscription.addons.get"),
			forge.WithSummary("Get add-on"),
			forge.WithDescription("Get a specific add-on by ID"),
			forge.WithTags("Subscription", "AddOns"),
		)

		addOnGroup.PATCH("/:id", p.handleUpdateAddOn,
			forge.WithName("subscription.addons.update"),
			forge.WithSummary("Update add-on"),
			forge.WithDescription("Update an existing add-on"),
			forge.WithTags("Subscription", "AddOns"),
			forge.WithValidation(true),
		)

		addOnGroup.DELETE("/:id", p.handleDeleteAddOn,
			forge.WithName("subscription.addons.delete"),
			forge.WithSummary("Delete add-on"),
			forge.WithDescription("Delete an add-on"),
			forge.WithTags("Subscription", "AddOns"),
		)
	}

	// Invoice routes
	invoiceGroup := router.Group("/subscription/invoices")
	{
		invoiceGroup.GET("", p.handleListInvoices,
			forge.WithName("subscription.invoices.list"),
			forge.WithSummary("List invoices"),
			forge.WithDescription("List all invoices"),
			forge.WithTags("Subscription", "Invoices"),
		)

		invoiceGroup.GET("/:id", p.handleGetInvoice,
			forge.WithName("subscription.invoices.get"),
			forge.WithSummary("Get invoice"),
			forge.WithDescription("Get a specific invoice by ID"),
			forge.WithTags("Subscription", "Invoices"),
		)
	}

	// Usage routes
	usageGroup := router.Group("/subscription/usage")
	{
		usageGroup.POST("", p.handleRecordUsage,
			forge.WithName("subscription.usage.record"),
			forge.WithSummary("Record usage"),
			forge.WithDescription("Record usage for metered billing"),
			forge.WithTags("Subscription", "Usage"),
			forge.WithValidation(true),
		)

		usageGroup.GET("/summary", p.handleGetUsageSummary,
			forge.WithName("subscription.usage.summary"),
			forge.WithSummary("Get usage summary"),
			forge.WithDescription("Get usage summary for a subscription"),
			forge.WithTags("Subscription", "Usage"),
		)
	}

	// Checkout routes
	checkoutGroup := router.Group("/subscription/checkout")
	{
		checkoutGroup.POST("", p.handleCreateCheckout,
			forge.WithName("subscription.checkout.create"),
			forge.WithSummary("Create checkout session"),
			forge.WithDescription("Create a checkout session for subscription purchase"),
			forge.WithTags("Subscription", "Checkout"),
			forge.WithValidation(true),
		)

		checkoutGroup.POST("/portal", p.handleCreatePortal,
			forge.WithName("subscription.checkout.portal"),
			forge.WithSummary("Create customer portal"),
			forge.WithDescription("Create a customer portal session for billing management"),
			forge.WithTags("Subscription", "Checkout"),
		)
	}

	// Webhook routes
	router.POST("/subscription/webhooks/stripe", p.handleStripeWebhook,
		forge.WithName("subscription.webhooks.stripe"),
		forge.WithSummary("Stripe webhook"),
		forge.WithDescription("Handle Stripe webhook events"),
		forge.WithTags("Subscription", "Webhooks"),
	)

	// Feature/limit check routes
	router.GET("/subscription/features/:orgId/:feature", p.handleCheckFeature,
		forge.WithName("subscription.features.check"),
		forge.WithSummary("Check feature access"),
		forge.WithDescription("Check if an organization has access to a feature"),
		forge.WithTags("Subscription", "Features"),
	)

	router.GET("/subscription/limits/:orgId", p.handleGetLimits,
		forge.WithName("subscription.limits.get"),
		forge.WithSummary("Get organization limits"),
		forge.WithDescription("Get all limits and current usage for an organization"),
		forge.WithTags("Subscription", "Limits"),
	)

	return nil
}

// RegisterHooks registers hooks for the subscription plugin
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Register enforcement hooks
	if p.config.RequireSubscription {
		hooks.RegisterBeforeOrganizationCreate(p.enforcementSvc.EnforceSubscriptionRequired)
	}

	// Enforce seat limits when adding members
	hooks.RegisterBeforeMemberAdd(p.enforcementSvc.EnforceSeatLimit)

	return nil
}

// RegisterServiceDecorators registers service decorators
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// No decorators needed - we expose our services through the plugin
	return nil
}

// RegisterRoles implements the PluginWithRoles interface
func (p *Plugin) RegisterRoles(reg interface{}) error {
	roleRegistry, ok := reg.(*rbac.RoleRegistry)
	if !ok {
		return fmt.Errorf("invalid role registry type")
	}

	// Register subscription admin role
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "subscription_admin",
		Description: "Can manage all subscription-related resources",
		Permissions: []string{
			"manage on subscription_plans",
			"manage on subscriptions",
			"manage on subscription_addons",
			"view on subscription_invoices",
			"manage on subscription_usage",
		},
		IsPlatform: false,
	}); err != nil {
		// Ignore duplicate role errors (role might already exist)
		p.logger.Debug("subscription_admin role registration", forge.F("note", err.Error()))
	}

	return nil
}

// DashboardExtension returns the dashboard extension
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	return p.dashboardExt
}

// Migrate runs database migrations for the subscription plugin
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	// Create tables
	tables := []interface{}{
		(*schema.SubscriptionPlan)(nil),
		(*schema.SubscriptionPlanFeature)(nil),
		(*schema.SubscriptionPlanTier)(nil),
		(*schema.Subscription)(nil),
		(*schema.SubscriptionAddOn)(nil),
		(*schema.SubscriptionAddOnFeature)(nil),
		(*schema.SubscriptionAddOnTier)(nil),
		(*schema.SubscriptionAddOnItem)(nil),
		(*schema.SubscriptionInvoice)(nil),
		(*schema.SubscriptionInvoiceItem)(nil),
		(*schema.SubscriptionUsageRecord)(nil),
		(*schema.SubscriptionPaymentMethod)(nil),
		(*schema.SubscriptionCustomer)(nil),
		(*schema.SubscriptionEvent)(nil),
	}

	for _, model := range tables {
		if _, err := p.db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	p.logger.Info("subscription plugin migrations completed")
	return nil
}

// GetPlanService returns the plan service
func (p *Plugin) GetPlanService() *service.PlanService {
	return p.planSvc
}

// GetSubscriptionService returns the subscription service
func (p *Plugin) GetSubscriptionService() *service.SubscriptionService {
	return p.subscriptionSvc
}

// GetAddOnService returns the add-on service
func (p *Plugin) GetAddOnService() *service.AddOnService {
	return p.addOnSvc
}

// GetInvoiceService returns the invoice service
func (p *Plugin) GetInvoiceService() *service.InvoiceService {
	return p.invoiceSvc
}

// GetUsageService returns the usage service
func (p *Plugin) GetUsageService() *service.UsageService {
	return p.usageSvc
}

// GetPaymentService returns the payment service
func (p *Plugin) GetPaymentService() *service.PaymentService {
	return p.paymentSvc
}

// GetCustomerService returns the customer service
func (p *Plugin) GetCustomerService() *service.CustomerService {
	return p.customerSvc
}

// GetEnforcementService returns the enforcement service
func (p *Plugin) GetEnforcementService() *service.EnforcementService {
	return p.enforcementSvc
}

// GetHookRegistry returns the subscription hook registry
func (p *Plugin) GetHookRegistry() *SubscriptionHookRegistry {
	return p.subHookRegistry
}

// GetConfig returns the plugin configuration
func (p *Plugin) GetConfig() Config {
	return p.config
}

