// Package subscription provides a comprehensive SaaS subscription and billing plugin for AuthSome.
// It supports multiple billing patterns, payment provider integration (Stripe), and organization-scoped subscriptions.
package subscription

import (
	"context"
	"fmt"
	"sync"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/organization"
	"github.com/xraph/authsome/core/rbac"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/plugins/subscription/handlers"
	"github.com/xraph/authsome/plugins/subscription/providers"
	"github.com/xraph/authsome/plugins/subscription/providers/mock"
	"github.com/xraph/authsome/plugins/subscription/providers/stripe"
	"github.com/xraph/authsome/plugins/subscription/repository"
	"github.com/xraph/authsome/plugins/subscription/schema"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Plugin implements the subscription plugin for AuthSome.
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
	featureSvc      *service.FeatureService
	featureUsageSvc *service.FeatureUsageService
	alertSvc        *service.AlertService
	analyticsSvc    *service.AnalyticsService
	couponSvc       *service.CouponService
	currencySvc     *service.CurrencyService
	taxSvc          *service.TaxService
	exportImportSvc *service.ExportImportService

	// Repositories
	planRepo         repository.PlanRepository
	subRepo          repository.SubscriptionRepository
	addOnRepo        repository.AddOnRepository
	invoiceRepo      repository.InvoiceRepository
	usageRepo        repository.UsageRepository
	paymentRepo      repository.PaymentMethodRepository
	customerRepo     repository.CustomerRepository
	eventRepo        repository.EventRepository
	featureRepo      repository.FeatureRepository
	featureUsageRepo repository.FeatureUsageRepository
	alertRepo        repository.AlertRepository
	analyticsRepo    repository.AnalyticsRepository
	couponRepo       repository.CouponRepository
	currencyRepo     repository.CurrencyRepository
	taxRepo          repository.TaxRepository

	// Payment provider
	provider providers.PaymentProvider

	// Hook registries
	hookRegistry    *hooks.HookRegistry
	subHookRegistry *SubscriptionHookRegistry

	// Dashboard extension (lazy initialized)
	dashboardExt     *DashboardExtension
	dashboardExtOnce sync.Once

	// Organization service for enforcement
	orgService *organization.Service

	// Default config (set via options)
	defaultConfig Config

	// Custom provider (set via options)
	customProvider providers.PaymentProvider

	// Feature handlers
	featureHandlers *handlers.FeatureHandlers
	publicHandlers  *handlers.PublicHandlers
	paymentHandlers *handlers.PaymentHandlers
}

// PluginOption is a functional option for configuring the plugin.
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin.
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithRequireSubscription sets whether subscription is required for org creation.
func WithRequireSubscription(required bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RequireSubscription = required
	}
}

// WithDefaultTrialDays sets the default trial days.
func WithDefaultTrialDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultTrialDays = days
	}
}

// WithGracePeriodDays sets the grace period for failed payments.
func WithGracePeriodDays(days int) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.GracePeriodDays = days
	}
}

// WithStripeConfig sets the Stripe configuration.
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

// WithAutoSyncSeats enables automatic seat synchronization.
func WithAutoSyncSeats(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoSyncSeats = enabled
	}
}

// WithAutoSyncPlans enables automatic plan synchronization to payment provider on create/update.
func WithAutoSyncPlans(enabled bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoSyncPlans = enabled
	}
}

// WithProvider sets a custom payment provider implementation
// This allows users to inject their own provider instead of using Stripe or mock provider.
func WithProvider(provider providers.PaymentProvider) PluginOption {
	return func(p *Plugin) {
		p.customProvider = provider
	}
}

// NewPlugin creates a new subscription plugin instance.
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

// ID returns the unique identifier for this plugin.
func (p *Plugin) ID() string {
	return "subscription"
}

// Dependencies declares the plugin dependencies.
func (p *Plugin) Dependencies() []string {
	return []string{"organization"}
}

// Init initializes the plugin with dependencies.
func (p *Plugin) Init(authInstance core.Authsome) error {
	if authInstance == nil {
		return errs.BadRequest("subscription plugin requires auth instance")
	}

	// Get Forge app
	forgeApp := authInstance.GetForgeApp()
	if forgeApp == nil {
		return errs.InternalServerErrorWithMessage("forge app not available")
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
		return errs.InternalServerErrorWithMessage("database not available")
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
		// Feature system models
		(*schema.Feature)(nil),
		(*schema.FeatureTier)(nil),
		(*schema.PlanFeatureLink)(nil),
		(*schema.OrganizationFeatureUsage)(nil),
		(*schema.FeatureUsageLog)(nil),
		(*schema.FeatureGrant)(nil),
	)

	// Get hook registry
	p.hookRegistry = authInstance.GetHookRegistry()
	if p.hookRegistry == nil {
		return errs.InternalServerErrorWithMessage("hook registry not available")
	}

	// Get service registry
	serviceRegistry := authInstance.GetServiceRegistry()
	if serviceRegistry == nil {
		return errs.InternalServerErrorWithMessage("service registry not available")
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
	p.featureRepo = repository.NewFeatureRepository(p.db)
	p.featureUsageRepo = repository.NewFeatureUsageRepository(p.db)
	p.alertRepo = repository.NewAlertRepository(p.db)
	p.analyticsRepo = repository.NewAnalyticsRepository(p.db)
	p.couponRepo = repository.NewCouponRepository(p.db)
	p.currencyRepo = repository.NewCurrencyRepository(p.db)
	p.taxRepo = repository.NewTaxRepository(p.db)

	// Initialize payment provider
	// Use custom provider if provided via options, otherwise fall back to configured provider
	if p.customProvider != nil {
		p.provider = p.customProvider
		p.logger.Info("using custom payment provider")
	} else if p.config.IsStripeConfigured() {
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
			p.logger.Debug("Stripe provider initialized")
		}
	} else {
		// Use mock provider for development
		p.provider = mock.NewMockProvider()
		p.logger.Warn("using mock payment provider - configure Stripe for production")
	}

	// Initialize services
	p.planSvc = service.NewPlanService(p.planRepo, p.provider, p.eventRepo)
	p.planSvc.SetAutoSyncPlans(p.config.AutoSyncPlans)
	p.customerSvc = service.NewCustomerService(p.customerRepo, p.provider, p.eventRepo)
	p.paymentSvc = service.NewPaymentService(p.paymentRepo, p.customerRepo, p.provider, p.eventRepo)
	p.subscriptionSvc = service.NewSubscriptionService(
		p.subRepo,
		p.planRepo,
		p.customerRepo,
		p.customerSvc,
		p.addOnRepo,
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
	p.featureSvc = service.NewFeatureService(p.featureRepo, p.planRepo, p.eventRepo, p.provider)
	p.featureUsageSvc = service.NewFeatureUsageService(
		p.featureUsageRepo,
		p.featureRepo,
		p.subRepo,
		p.planRepo,
		p.eventRepo,
	)
	p.alertSvc = service.NewAlertService(p.alertRepo, p.usageRepo, p.subRepo)
	p.analyticsSvc = service.NewAnalyticsService(p.analyticsRepo, p.subRepo, p.planRepo)
	p.couponSvc = service.NewCouponService(p.couponRepo, p.subRepo)
	p.currencySvc = service.NewCurrencyService(p.currencyRepo)
	p.taxSvc = service.NewTaxService(p.taxRepo)
	p.exportImportSvc = service.NewExportImportService(p.featureRepo, p.planRepo, p.eventRepo)

	// Set feature repositories on enforcement service for enhanced feature checking
	p.enforcementSvc.SetFeatureRepositories(p.featureRepo, p.featureUsageRepo)

	// Initialize handlers
	p.featureHandlers = handlers.NewFeatureHandlers(p.featureSvc, p.featureUsageSvc)
	p.publicHandlers = handlers.NewPublicHandlers(p.featureSvc, p.planSvc)
	p.paymentHandlers = handlers.NewPaymentHandlers(p.paymentSvc, p.customerSvc)

	// Dashboard extension is lazy-initialized when first accessed via DashboardExtension()

	// Register services in Forge DI container if available
	if container := forgeApp.Container(); container != nil {
		if err := p.RegisterServices(container); err != nil {
			p.logger.Warn("failed to register subscription services in DI container", forge.F("error", err.Error()))
		}
	}

	p.logger.Info("subscription plugin initialized",
		forge.F("require_subscription", p.config.RequireSubscription),
		forge.F("default_trial_days", p.config.DefaultTrialDays),
		forge.F("provider", p.config.Provider))

	return nil
}

// RegisterRoutes registers the plugin's HTTP routes.
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

		subGroup.POST("/:id/sync", p.handleSyncSubscription,
			forge.WithName("subscription.subscriptions.sync"),
			forge.WithSummary("Sync subscription to provider"),
			forge.WithDescription("Sync a subscription to the payment provider (Stripe/Paddle)"),
			forge.WithTags("Subscription", "Subscriptions"),
		)

		subGroup.POST("/:id/sync-from-provider", p.handleSyncSubscriptionFromProvider,
			forge.WithName("subscription.subscriptions.sync_from_provider"),
			forge.WithSummary("Sync subscription from provider"),
			forge.WithDescription("Pull latest subscription data from payment provider and update local record"),
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

		invoiceGroup.POST("/sync", p.handleSyncInvoices,
			forge.WithName("subscription.invoices.sync"),
			forge.WithSummary("Sync invoices from Stripe"),
			forge.WithDescription("Backfill and sync invoices from Stripe to local database"),
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

	// Payment method routes
	paymentGroup := router.Group("/subscription/payment-methods")
	{
		paymentGroup.POST("/setup-intent", p.handleCreateSetupIntent,
			forge.WithName("subscription.payment_methods.setup_intent"),
			forge.WithSummary("Create setup intent"),
			forge.WithDescription("Create a Stripe setup intent for adding payment method"),
			forge.WithTags("Subscription", "Payment Methods"),
			forge.WithValidation(true),
		)

		paymentGroup.POST("", p.handleAddPaymentMethod,
			forge.WithName("subscription.payment_methods.add"),
			forge.WithSummary("Add payment method"),
			forge.WithDescription("Attach a tokenized payment method to organization"),
			forge.WithTags("Subscription", "Payment Methods"),
			forge.WithValidation(true),
		)

		paymentGroup.GET("", p.handleListPaymentMethods,
			forge.WithName("subscription.payment_methods.list"),
			forge.WithSummary("List payment methods"),
			forge.WithDescription("List all payment methods for organization"),
			forge.WithTags("Subscription", "Payment Methods"),
		)

		paymentGroup.GET("/:id", p.handleGetPaymentMethod,
			forge.WithName("subscription.payment_methods.get"),
			forge.WithSummary("Get payment method"),
			forge.WithDescription("Get a specific payment method by ID"),
			forge.WithTags("Subscription", "Payment Methods"),
		)

		paymentGroup.POST("/:id/set-default", p.handleSetDefaultPaymentMethod,
			forge.WithName("subscription.payment_methods.set_default"),
			forge.WithSummary("Set default payment method"),
			forge.WithDescription("Set a payment method as the default"),
			forge.WithTags("Subscription", "Payment Methods"),
		)

		paymentGroup.DELETE("/:id", p.handleRemovePaymentMethod,
			forge.WithName("subscription.payment_methods.remove"),
			forge.WithSummary("Remove payment method"),
			forge.WithDescription("Remove a payment method from organization"),
			forge.WithTags("Subscription", "Payment Methods"),
		)
	}

	// Webhook routes
	router.POST("/subscription/webhooks/stripe", p.handleStripeWebhook,
		forge.WithName("subscription.webhooks.stripe"),
		forge.WithSummary("Stripe webhook"),
		forge.WithDescription("Handle Stripe webhook events"),
		forge.WithTags("Subscription", "Webhooks"),
	)

	// Feature/limit check routes (legacy)
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

	// Register feature management routes
	p.registerFeatureRoutes(router)

	// Register public API routes
	p.registerPublicRoutes(router)

	return nil
}

// RegisterHooks registers hooks for the subscription plugin.
func (p *Plugin) RegisterHooks(hooks *hooks.HookRegistry) error {
	// Register enforcement hooks
	if p.config.RequireSubscription {
		hooks.RegisterBeforeOrganizationCreate(p.enforcementSvc.EnforceSubscriptionRequired)
	}

	// Enforce seat limits when adding members
	hooks.RegisterBeforeMemberAdd(p.enforcementSvc.EnforceSeatLimit)

	return nil
}

// RegisterServiceDecorators registers service decorators.
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	// Register subscription services in the service registry for DI access
	// This allows other plugins and components to retrieve subscription services
	// Note: Uses constants defined in helpers.go for consistent keys
	if err := services.Register(ServiceNamePlanService, p.planSvc); err != nil {
		p.logger.Warn("failed to register plan service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameSubService, p.subscriptionSvc); err != nil {
		p.logger.Warn("failed to register subscription service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameAddOnService, p.addOnSvc); err != nil {
		p.logger.Warn("failed to register addon service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameInvoiceService, p.invoiceSvc); err != nil {
		p.logger.Warn("failed to register invoice service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameUsageService, p.usageSvc); err != nil {
		p.logger.Warn("failed to register usage service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNamePaymentService, p.paymentSvc); err != nil {
		p.logger.Warn("failed to register payment service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameCustomerService, p.customerSvc); err != nil {
		p.logger.Warn("failed to register customer service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameEnforcementService, p.enforcementSvc); err != nil {
		p.logger.Warn("failed to register enforcement service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameFeatureService, p.featureSvc); err != nil {
		p.logger.Warn("failed to register feature service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameFeatureUsageService, p.featureUsageSvc); err != nil {
		p.logger.Warn("failed to register feature usage service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameAlertService, p.alertSvc); err != nil {
		p.logger.Warn("failed to register alert service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameAnalyticsService, p.analyticsSvc); err != nil {
		p.logger.Warn("failed to register analytics service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameCouponService, p.couponSvc); err != nil {
		p.logger.Warn("failed to register coupon service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameCurrencyService, p.currencySvc); err != nil {
		p.logger.Warn("failed to register currency service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameTaxService, p.taxSvc); err != nil {
		p.logger.Warn("failed to register tax service", forge.F("error", err.Error()))
	}

	if err := services.Register(ServiceNameHookRegistry, p.subHookRegistry); err != nil {
		p.logger.Warn("failed to register subscription hook registry", forge.F("error", err.Error()))
	}

	return nil
}

// RegisterRoles implements the PluginWithRoles interface.
func (p *Plugin) RegisterRoles(reg any) error {
	roleRegistry, ok := reg.(*rbac.RoleRegistry)
	if !ok {
		return errs.BadRequest("invalid role registry type")
	}

	// Register subscription admin role
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "subscription_admin",
		DisplayName: "Subscription Administrator",
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

	// Register billing manager role
	if err := roleRegistry.RegisterRole(&rbac.RoleDefinition{
		Name:        "billing_manager",
		DisplayName: "Billing Manager",
		Description: "Can manage billing, subscriptions, and payment methods",
		Permissions: []string{
			"view on subscriptions",
			"manage on payment_methods",
			"view on subscription_invoices",
			"manage on subscription_usage",
		},
		IsPlatform: false,
	}); err != nil {
		p.logger.Debug("billing_manager role registration", forge.F("note", err.Error()))
	}

	return nil
}

// DashboardExtension returns the dashboard extension.
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	p.dashboardExtOnce.Do(func() {
		p.dashboardExt = NewDashboardExtension(p)
	})

	return p.dashboardExt
}

// Migrate runs database migrations for the subscription plugin.
func (p *Plugin) Migrate() error {
	ctx := context.Background()

	// Create tables
	tables := []any{
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
		// Feature system tables
		(*schema.Feature)(nil),
		(*schema.FeatureTier)(nil),
		(*schema.PlanFeatureLink)(nil),
		(*schema.OrganizationFeatureUsage)(nil),
		(*schema.FeatureUsageLog)(nil),
		(*schema.FeatureGrant)(nil),
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

// GetPlanService returns the plan service.
func (p *Plugin) GetPlanService() *service.PlanService {
	return p.planSvc
}

// GetSubscriptionService returns the subscription service.
func (p *Plugin) GetSubscriptionService() *service.SubscriptionService {
	return p.subscriptionSvc
}

// GetAddOnService returns the add-on service.
func (p *Plugin) GetAddOnService() *service.AddOnService {
	return p.addOnSvc
}

// GetInvoiceService returns the invoice service.
func (p *Plugin) GetInvoiceService() *service.InvoiceService {
	return p.invoiceSvc
}

// GetUsageService returns the usage service.
func (p *Plugin) GetUsageService() *service.UsageService {
	return p.usageSvc
}

// GetPaymentService returns the payment service.
func (p *Plugin) GetPaymentService() *service.PaymentService {
	return p.paymentSvc
}

// GetCustomerService returns the customer service.
func (p *Plugin) GetCustomerService() *service.CustomerService {
	return p.customerSvc
}

// GetEnforcementService returns the enforcement service.
func (p *Plugin) GetEnforcementService() *service.EnforcementService {
	return p.enforcementSvc
}

// GetHookRegistry returns the subscription hook registry.
func (p *Plugin) GetHookRegistry() *SubscriptionHookRegistry {
	return p.subHookRegistry
}

// GetConfig returns the plugin configuration.
func (p *Plugin) GetConfig() Config {
	return p.config
}

// GetFeatureService returns the feature service.
func (p *Plugin) GetFeatureService() *service.FeatureService {
	return p.featureSvc
}

// GetFeatureUsageService returns the feature usage service.
func (p *Plugin) GetFeatureUsageService() *service.FeatureUsageService {
	return p.featureUsageSvc
}

// GetAlertService returns the alert service.
func (p *Plugin) GetAlertService() *service.AlertService {
	return p.alertSvc
}

// GetAnalyticsService returns the analytics service.
func (p *Plugin) GetAnalyticsService() *service.AnalyticsService {
	return p.analyticsSvc
}

// GetCouponService returns the coupon service.
func (p *Plugin) GetCouponService() *service.CouponService {
	return p.couponSvc
}

// GetCurrencyService returns the currency service.
func (p *Plugin) GetCurrencyService() *service.CurrencyService {
	return p.currencySvc
}

// GetTaxService returns the tax service.
func (p *Plugin) GetTaxService() *service.TaxService {
	return p.taxSvc
}

// GetExportImportService returns the export/import service.
func (p *Plugin) GetExportImportService() *service.ExportImportService {
	return p.exportImportSvc
}

// GetProvider returns the payment provider.
func (p *Plugin) GetProvider() providers.PaymentProvider {
	return p.provider
}

// registerFeatureRoutes registers feature management routes.
func (p *Plugin) registerFeatureRoutes(router forge.Router) {
	// Feature CRUD routes
	featureGroup := router.Group("/subscription/features")
	{
		featureGroup.POST("", p.handleCreateFeature,
			forge.WithName("subscription.features.create"),
			forge.WithSummary("Create feature"),
			forge.WithDescription("Create a new feature definition"),
			forge.WithTags("Subscription", "Features"),
			forge.WithValidation(true),
		)

		featureGroup.GET("", p.handleListFeatures,
			forge.WithName("subscription.features.list"),
			forge.WithSummary("List features"),
			forge.WithDescription("List all feature definitions"),
			forge.WithTags("Subscription", "Features"),
		)

		featureGroup.GET("/:id", p.handleGetFeature,
			forge.WithName("subscription.features.get"),
			forge.WithSummary("Get feature"),
			forge.WithDescription("Get a specific feature by ID"),
			forge.WithTags("Subscription", "Features"),
		)

		featureGroup.PATCH("/:id", p.handleUpdateFeature,
			forge.WithName("subscription.features.update"),
			forge.WithSummary("Update feature"),
			forge.WithDescription("Update an existing feature"),
			forge.WithTags("Subscription", "Features"),
			forge.WithValidation(true),
		)

		featureGroup.DELETE("/:id", p.handleDeleteFeature,
			forge.WithName("subscription.features.delete"),
			forge.WithSummary("Delete feature"),
			forge.WithDescription("Delete a feature"),
			forge.WithTags("Subscription", "Features"),
		)

		featureGroup.POST("/:id/sync", p.handleSyncFeature,
			forge.WithName("subscription.features.sync"),
			forge.WithSummary("Sync feature to provider"),
			forge.WithDescription("Manually sync a feature to the payment provider"),
			forge.WithTags("Subscription", "Features"),
		)

		featureGroup.POST("/sync-from-provider/:providerId", p.handleSyncFeatureFromProvider,
			forge.WithName("subscription.features.sync_from_provider"),
			forge.WithSummary("Sync feature from provider"),
			forge.WithDescription("Sync a feature from the payment provider"),
			forge.WithTags("Subscription", "Features"),
		)

		featureGroup.POST("/sync-all-from-provider", p.handleSyncAllFeaturesFromProvider,
			forge.WithName("subscription.features.sync_all_from_provider"),
			forge.WithSummary("Sync all features from provider"),
			forge.WithDescription("Sync all features from the payment provider for a product"),
			forge.WithTags("Subscription", "Features"),
		)
	}

	// Plan-Feature linking routes
	planFeatureGroup := router.Group("/subscription/plans/:planId/features")
	{
		planFeatureGroup.POST("", p.handleLinkFeatureToPlan,
			forge.WithName("subscription.plans.features.link"),
			forge.WithSummary("Link feature to plan"),
			forge.WithDescription("Link a feature to a plan with configuration"),
			forge.WithTags("Subscription", "Plans", "Features"),
			forge.WithValidation(true),
		)

		planFeatureGroup.GET("", p.handleGetPlanFeatures,
			forge.WithName("subscription.plans.features.list"),
			forge.WithSummary("Get plan features"),
			forge.WithDescription("Get all features linked to a plan"),
			forge.WithTags("Subscription", "Plans", "Features"),
		)

		planFeatureGroup.PATCH("/:featureId", p.handleUpdatePlanFeatureLink,
			forge.WithName("subscription.plans.features.update"),
			forge.WithSummary("Update plan feature link"),
			forge.WithDescription("Update the feature-plan link configuration"),
			forge.WithTags("Subscription", "Plans", "Features"),
			forge.WithValidation(true),
		)

		planFeatureGroup.DELETE("/:featureId", p.handleUnlinkFeatureFromPlan,
			forge.WithName("subscription.plans.features.unlink"),
			forge.WithSummary("Unlink feature from plan"),
			forge.WithDescription("Remove a feature from a plan"),
			forge.WithTags("Subscription", "Plans", "Features"),
		)
	}

	// Organization feature usage routes
	orgFeatureGroup := router.Group("/subscription/organizations/:orgId/features")
	{
		orgFeatureGroup.GET("", p.handleGetOrgFeatures,
			forge.WithName("subscription.organizations.features.list"),
			forge.WithSummary("Get organization features"),
			forge.WithDescription("Get all feature access for an organization"),
			forge.WithTags("Subscription", "Organizations", "Features"),
		)

		orgFeatureGroup.GET("/:key/usage", p.handleGetFeatureUsage,
			forge.WithName("subscription.organizations.features.usage"),
			forge.WithSummary("Get feature usage"),
			forge.WithDescription("Get usage for a specific feature"),
			forge.WithTags("Subscription", "Organizations", "Features"),
		)

		orgFeatureGroup.GET("/:key/access", p.handleCheckFeatureAccess,
			forge.WithName("subscription.organizations.features.access"),
			forge.WithSummary("Check feature access"),
			forge.WithDescription("Check if organization has access to a feature"),
			forge.WithTags("Subscription", "Organizations", "Features"),
		)

		orgFeatureGroup.POST("/:key/consume", p.handleConsumeFeature,
			forge.WithName("subscription.organizations.features.consume"),
			forge.WithSummary("Consume feature quota"),
			forge.WithDescription("Consume feature quota for an organization"),
			forge.WithTags("Subscription", "Organizations", "Features"),
			forge.WithValidation(true),
		)

		orgFeatureGroup.POST("/:key/grant", p.handleGrantFeature,
			forge.WithName("subscription.organizations.features.grant"),
			forge.WithSummary("Grant feature quota"),
			forge.WithDescription("Grant additional feature quota to an organization"),
			forge.WithTags("Subscription", "Organizations", "Features"),
			forge.WithValidation(true),
		)
	}

	// Grants management
	router.GET("/subscription/organizations/:orgId/grants", p.handleListGrants,
		forge.WithName("subscription.organizations.grants.list"),
		forge.WithSummary("List grants"),
		forge.WithDescription("List all active grants for an organization"),
		forge.WithTags("Subscription", "Organizations", "Grants"),
	)

	router.DELETE("/subscription/grants/:grantId", p.handleRevokeGrant,
		forge.WithName("subscription.grants.revoke"),
		forge.WithSummary("Revoke grant"),
		forge.WithDescription("Revoke a feature grant"),
		forge.WithTags("Subscription", "Grants"),
	)
}

// registerPublicRoutes registers public pricing page routes.
func (p *Plugin) registerPublicRoutes(router forge.Router) {
	publicGroup := router.Group("/subscription/public")
	{
		publicGroup.GET("/plans", p.handleListPublicPlans,
			forge.WithName("subscription.public.plans.list"),
			forge.WithSummary("List public plans"),
			forge.WithDescription("List all public plans with features for pricing pages"),
			forge.WithTags("Subscription", "Public"),
		)

		publicGroup.GET("/plans/:slug", p.handleGetPublicPlan,
			forge.WithName("subscription.public.plans.get"),
			forge.WithSummary("Get public plan"),
			forge.WithDescription("Get a public plan by slug"),
			forge.WithTags("Subscription", "Public"),
		)

		publicGroup.GET("/plans/:slug/features", p.handleGetPublicPlanFeatures,
			forge.WithName("subscription.public.plans.features"),
			forge.WithSummary("Get public plan features"),
			forge.WithDescription("Get features for a public plan"),
			forge.WithTags("Subscription", "Public"),
		)

		publicGroup.GET("/features", p.handleListPublicFeatures,
			forge.WithName("subscription.public.features.list"),
			forge.WithSummary("List public features"),
			forge.WithDescription("List all public features"),
			forge.WithTags("Subscription", "Public"),
		)

		publicGroup.GET("/compare", p.handleComparePlans,
			forge.WithName("subscription.public.compare"),
			forge.WithSummary("Compare plans"),
			forge.WithDescription("Compare features across plans"),
			forge.WithTags("Subscription", "Public"),
		)
	}
}

// Feature handler wrappers

func (p *Plugin) handleCreateFeature(c forge.Context) error {
	return p.featureHandlers.HandleCreateFeature(c)
}

func (p *Plugin) handleListFeatures(c forge.Context) error {
	return p.featureHandlers.HandleListFeatures(c)
}

func (p *Plugin) handleGetFeature(c forge.Context) error {
	return p.featureHandlers.HandleGetFeature(c)
}

func (p *Plugin) handleUpdateFeature(c forge.Context) error {
	return p.featureHandlers.HandleUpdateFeature(c)
}

func (p *Plugin) handleDeleteFeature(c forge.Context) error {
	return p.featureHandlers.HandleDeleteFeature(c)
}

func (p *Plugin) handleLinkFeatureToPlan(c forge.Context) error {
	return p.featureHandlers.HandleLinkFeatureToPlan(c)
}

func (p *Plugin) handleGetPlanFeatures(c forge.Context) error {
	return p.featureHandlers.HandleGetPlanFeatures(c)
}

func (p *Plugin) handleUpdatePlanFeatureLink(c forge.Context) error {
	return p.featureHandlers.HandleUpdatePlanFeatureLink(c)
}

func (p *Plugin) handleUnlinkFeatureFromPlan(c forge.Context) error {
	return p.featureHandlers.HandleUnlinkFeatureFromPlan(c)
}

func (p *Plugin) handleGetOrgFeatures(c forge.Context) error {
	return p.featureHandlers.HandleGetOrgFeatures(c)
}

func (p *Plugin) handleGetFeatureUsage(c forge.Context) error {
	return p.featureHandlers.HandleGetFeatureUsage(c)
}

func (p *Plugin) handleCheckFeatureAccess(c forge.Context) error {
	return p.featureHandlers.HandleCheckFeatureAccess(c)
}

func (p *Plugin) handleConsumeFeature(c forge.Context) error {
	return p.featureHandlers.HandleConsumeFeature(c)
}

func (p *Plugin) handleGrantFeature(c forge.Context) error {
	return p.featureHandlers.HandleGrantFeature(c)
}

func (p *Plugin) handleListGrants(c forge.Context) error {
	return p.featureHandlers.HandleListGrants(c)
}

func (p *Plugin) handleRevokeGrant(c forge.Context) error {
	return p.featureHandlers.HandleRevokeGrant(c)
}

func (p *Plugin) handleSyncFeature(c forge.Context) error {
	return p.featureHandlers.HandleSyncFeature(c)
}

func (p *Plugin) handleSyncFeatureFromProvider(c forge.Context) error {
	return p.featureHandlers.HandleSyncFeatureFromProvider(c)
}

func (p *Plugin) handleSyncAllFeaturesFromProvider(c forge.Context) error {
	return p.featureHandlers.HandleSyncAllFeaturesFromProvider(c)
}

// Public handler wrappers

func (p *Plugin) handleListPublicPlans(c forge.Context) error {
	return p.publicHandlers.HandleListPublicPlans(c)
}

func (p *Plugin) handleGetPublicPlan(c forge.Context) error {
	return p.publicHandlers.HandleGetPublicPlan(c)
}

func (p *Plugin) handleGetPublicPlanFeatures(c forge.Context) error {
	return p.publicHandlers.HandleGetPublicPlanFeatures(c)
}

func (p *Plugin) handleListPublicFeatures(c forge.Context) error {
	return p.publicHandlers.HandleListPublicFeatures(c)
}

func (p *Plugin) handleComparePlans(c forge.Context) error {
	return p.publicHandlers.HandleComparePlans(c)
}

// Payment method handler wrappers

func (p *Plugin) handleCreateSetupIntent(c forge.Context) error {
	return p.paymentHandlers.HandleCreateSetupIntent(c)
}

func (p *Plugin) handleAddPaymentMethod(c forge.Context) error {
	return p.paymentHandlers.HandleAddPaymentMethod(c)
}

func (p *Plugin) handleListPaymentMethods(c forge.Context) error {
	return p.paymentHandlers.HandleListPaymentMethods(c)
}

func (p *Plugin) handleGetPaymentMethod(c forge.Context) error {
	return p.paymentHandlers.HandleGetPaymentMethod(c)
}

func (p *Plugin) handleSetDefaultPaymentMethod(c forge.Context) error {
	return p.paymentHandlers.HandleSetDefaultPaymentMethod(c)
}

func (p *Plugin) handleRemovePaymentMethod(c forge.Context) error {
	return p.paymentHandlers.HandleRemovePaymentMethod(c)
}
