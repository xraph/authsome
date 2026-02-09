package subscription

import (
	"fmt"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
	"github.com/xraph/vessel"
)

// Service name constants for DI container registration.
const (
	ServiceNamePlugin              = "subscription.plugin"
	ServiceNamePlanService         = "subscription.plan"
	ServiceNameSubService          = "subscription.subscription"
	ServiceNameAddOnService        = "subscription.addon"
	ServiceNameInvoiceService      = "subscription.invoice"
	ServiceNameUsageService        = "subscription.usage"
	ServiceNamePaymentService      = "subscription.payment"
	ServiceNameCustomerService     = "subscription.customer"
	ServiceNameEnforcementService  = "subscription.enforcement"
	ServiceNameFeatureService      = "subscription.feature"
	ServiceNameFeatureUsageService = "subscription.feature_usage"
	ServiceNameAlertService        = "subscription.alert"
	ServiceNameAnalyticsService    = "subscription.analytics"
	ServiceNameCouponService       = "subscription.coupon"
	ServiceNameCurrencyService     = "subscription.currency"
	ServiceNameTaxService          = "subscription.tax"
	ServiceNameHookRegistry        = "subscription.hook_registry"
)

// ResolveSubscriptionPlugin resolves the subscription plugin from the container.
func ResolveSubscriptionPlugin(container forge.Container) (*Plugin, error) {
	plugin, err := vessel.InjectType[*Plugin](container)
	if plugin != nil {
		return plugin, nil
	}

	resolved, err := container.Resolve(ServiceNamePlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subscription plugin: %w", err)
	}

	plugin, ok := resolved.(*Plugin)
	if !ok {
		return nil, errs.BadRequest("invalid subscription plugin type")
	}

	return plugin, nil
}

// ResolvePlanService resolves the plan service from the container.
func ResolvePlanService(container forge.Container) (*service.PlanService, error) {
	svc, err := vessel.InjectType[*service.PlanService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNamePlanService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve plan service: %w", err)
	}

	svc, ok := resolved.(*service.PlanService)
	if !ok {
		return nil, errs.BadRequest("invalid plan service type")
	}

	return svc, nil
}

// ResolveSubscriptionService resolves the subscription service from the container.
func ResolveSubscriptionService(container forge.Container) (*service.SubscriptionService, error) {
	svc, err := vessel.InjectType[*service.SubscriptionService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameSubService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subscription service: %w", err)
	}

	svc, ok := resolved.(*service.SubscriptionService)
	if !ok {
		return nil, errs.BadRequest("invalid subscription service type")
	}

	return svc, nil
}

// ResolveAddOnService resolves the add-on service from the container.
func ResolveAddOnService(container forge.Container) (*service.AddOnService, error) {
	svc, err := vessel.InjectType[*service.AddOnService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameAddOnService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve add-on service: %w", err)
	}

	svc, ok := resolved.(*service.AddOnService)
	if !ok {
		return nil, errs.BadRequest("invalid add-on service type")
	}

	return svc, nil
}

// ResolveInvoiceService resolves the invoice service from the container.
func ResolveInvoiceService(container forge.Container) (*service.InvoiceService, error) {
	svc, err := vessel.InjectType[*service.InvoiceService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameInvoiceService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve invoice service: %w", err)
	}

	svc, ok := resolved.(*service.InvoiceService)
	if !ok {
		return nil, errs.BadRequest("invalid invoice service type")
	}

	return svc, nil
}

// ResolveUsageService resolves the usage service from the container.
func ResolveUsageService(container forge.Container) (*service.UsageService, error) {
	svc, err := vessel.InjectType[*service.UsageService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameUsageService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve usage service: %w", err)
	}

	svc, ok := resolved.(*service.UsageService)
	if !ok {
		return nil, errs.BadRequest("invalid usage service type")
	}

	return svc, nil
}

// ResolvePaymentService resolves the payment service from the container.
func ResolvePaymentService(container forge.Container) (*service.PaymentService, error) {
	svc, err := vessel.InjectType[*service.PaymentService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNamePaymentService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve payment service: %w", err)
	}

	svc, ok := resolved.(*service.PaymentService)
	if !ok {
		return nil, errs.BadRequest("invalid payment service type")
	}

	return svc, nil
}

// ResolveCustomerService resolves the customer service from the container.
func ResolveCustomerService(container forge.Container) (*service.CustomerService, error) {
	svc, err := vessel.InjectType[*service.CustomerService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameCustomerService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve customer service: %w", err)
	}

	svc, ok := resolved.(*service.CustomerService)
	if !ok {
		return nil, errs.BadRequest("invalid customer service type")
	}

	return svc, nil
}

// ResolveEnforcementService resolves the enforcement service from the container.
func ResolveEnforcementService(container forge.Container) (*service.EnforcementService, error) {
	svc, err := vessel.InjectType[*service.EnforcementService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameEnforcementService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve enforcement service: %w", err)
	}

	svc, ok := resolved.(*service.EnforcementService)
	if !ok {
		return nil, errs.BadRequest("invalid enforcement service type")
	}

	return svc, nil
}

// ResolveSubscriptionHookRegistry resolves the subscription hook registry from the container.
func ResolveSubscriptionHookRegistry(container forge.Container) (*SubscriptionHookRegistry, error) {
	registry, err := vessel.InjectType[*SubscriptionHookRegistry](container)
	if registry != nil {
		return registry, nil
	}

	resolved, err := container.Resolve(ServiceNameHookRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subscription hook registry: %w", err)
	}

	registry, ok := resolved.(*SubscriptionHookRegistry)
	if !ok {
		return nil, errs.BadRequest("invalid subscription hook registry type")
	}

	return registry, nil
}

// ResolveFeatureService resolves the feature service from the container.
func ResolveFeatureService(container forge.Container) (*service.FeatureService, error) {
	svc, err := vessel.InjectType[*service.FeatureService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameFeatureService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve feature service: %w", err)
	}

	svc, ok := resolved.(*service.FeatureService)
	if !ok {
		return nil, errs.BadRequest("invalid feature service type")
	}

	return svc, nil
}

// ResolveFeatureUsageService resolves the feature usage service from the container.
func ResolveFeatureUsageService(container forge.Container) (*service.FeatureUsageService, error) {
	svc, err := vessel.InjectType[*service.FeatureUsageService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameFeatureUsageService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve feature usage service: %w", err)
	}

	svc, ok := resolved.(*service.FeatureUsageService)
	if !ok {
		return nil, errs.BadRequest("invalid feature usage service type")
	}

	return svc, nil
}

// ResolveAlertService resolves the alert service from the container.
func ResolveAlertService(container forge.Container) (*service.AlertService, error) {
	svc, err := vessel.InjectType[*service.AlertService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameAlertService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve alert service: %w", err)
	}

	svc, ok := resolved.(*service.AlertService)
	if !ok {
		return nil, errs.BadRequest("invalid alert service type")
	}

	return svc, nil
}

// ResolveAnalyticsService resolves the analytics service from the container.
func ResolveAnalyticsService(container forge.Container) (*service.AnalyticsService, error) {
	svc, err := vessel.InjectType[*service.AnalyticsService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameAnalyticsService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve analytics service: %w", err)
	}

	svc, ok := resolved.(*service.AnalyticsService)
	if !ok {
		return nil, errs.BadRequest("invalid analytics service type")
	}

	return svc, nil
}

// ResolveCouponService resolves the coupon service from the container.
func ResolveCouponService(container forge.Container) (*service.CouponService, error) {
	svc, err := vessel.InjectType[*service.CouponService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameCouponService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve coupon service: %w", err)
	}

	svc, ok := resolved.(*service.CouponService)
	if !ok {
		return nil, errs.BadRequest("invalid coupon service type")
	}

	return svc, nil
}

// ResolveCurrencyService resolves the currency service from the container.
func ResolveCurrencyService(container forge.Container) (*service.CurrencyService, error) {
	svc, err := vessel.InjectType[*service.CurrencyService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameCurrencyService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve currency service: %w", err)
	}

	svc, ok := resolved.(*service.CurrencyService)
	if !ok {
		return nil, errs.BadRequest("invalid currency service type")
	}

	return svc, nil
}

// ResolveTaxService resolves the tax service from the container.
func ResolveTaxService(container forge.Container) (*service.TaxService, error) {
	svc, err := vessel.InjectType[*service.TaxService](container)
	if svc != nil {
		return svc, nil
	}

	resolved, err := container.Resolve(ServiceNameTaxService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax service: %w", err)
	}

	svc, ok := resolved.(*service.TaxService)
	if !ok {
		return nil, errs.BadRequest("invalid tax service type")
	}

	return svc, nil
}

// RegisterServices registers all subscription services in the DI container
// Uses vessel.ProvideConstructor for type-safe, constructor-based dependency injection.
func (p *Plugin) RegisterServices(container forge.Container) error {
	// Register plugin itself
	if err := forge.ProvideConstructor(container, func() (*Plugin, error) {
		return p, nil
	}, vessel.WithAliases(ServiceNamePlugin)); err != nil {
		return fmt.Errorf("failed to register subscription plugin: %w", err)
	}

	// Register core services
	if err := forge.ProvideConstructor(container, func() (*service.PlanService, error) {
		return p.planSvc, nil
	}, vessel.WithAliases(ServiceNamePlanService)); err != nil {
		return fmt.Errorf("failed to register plan service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.SubscriptionService, error) {
		return p.subscriptionSvc, nil
	}, vessel.WithAliases(ServiceNameSubService)); err != nil {
		return fmt.Errorf("failed to register subscription service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.AddOnService, error) {
		return p.addOnSvc, nil
	}, vessel.WithAliases(ServiceNameAddOnService)); err != nil {
		return fmt.Errorf("failed to register add-on service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.InvoiceService, error) {
		return p.invoiceSvc, nil
	}, vessel.WithAliases(ServiceNameInvoiceService)); err != nil {
		return fmt.Errorf("failed to register invoice service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.UsageService, error) {
		return p.usageSvc, nil
	}, vessel.WithAliases(ServiceNameUsageService)); err != nil {
		return fmt.Errorf("failed to register usage service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.PaymentService, error) {
		return p.paymentSvc, nil
	}, vessel.WithAliases(ServiceNamePaymentService)); err != nil {
		return fmt.Errorf("failed to register payment service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.CustomerService, error) {
		return p.customerSvc, nil
	}, vessel.WithAliases(ServiceNameCustomerService)); err != nil {
		return fmt.Errorf("failed to register customer service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.EnforcementService, error) {
		return p.enforcementSvc, nil
	}, vessel.WithAliases(ServiceNameEnforcementService)); err != nil {
		return fmt.Errorf("failed to register enforcement service: %w", err)
	}

	// Register feature services
	if err := forge.ProvideConstructor(container, func() (*service.FeatureService, error) {
		return p.featureSvc, nil
	}, vessel.WithAliases(ServiceNameFeatureService)); err != nil {
		return fmt.Errorf("failed to register feature service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.FeatureUsageService, error) {
		return p.featureUsageSvc, nil
	}, vessel.WithAliases(ServiceNameFeatureUsageService)); err != nil {
		return fmt.Errorf("failed to register feature usage service: %w", err)
	}

	// Register additional services
	if err := forge.ProvideConstructor(container, func() (*service.AlertService, error) {
		return p.alertSvc, nil
	}, vessel.WithAliases(ServiceNameAlertService)); err != nil {
		return fmt.Errorf("failed to register alert service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.AnalyticsService, error) {
		return p.analyticsSvc, nil
	}, vessel.WithAliases(ServiceNameAnalyticsService)); err != nil {
		return fmt.Errorf("failed to register analytics service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.CouponService, error) {
		return p.couponSvc, nil
	}, vessel.WithAliases(ServiceNameCouponService)); err != nil {
		return fmt.Errorf("failed to register coupon service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.CurrencyService, error) {
		return p.currencySvc, nil
	}, vessel.WithAliases(ServiceNameCurrencyService)); err != nil {
		return fmt.Errorf("failed to register currency service: %w", err)
	}

	if err := forge.ProvideConstructor(container, func() (*service.TaxService, error) {
		return p.taxSvc, nil
	}, vessel.WithAliases(ServiceNameTaxService)); err != nil {
		return fmt.Errorf("failed to register tax service: %w", err)
	}

	// Register hook registry
	if err := forge.ProvideConstructor(container, func() (*SubscriptionHookRegistry, error) {
		return p.subHookRegistry, nil
	}, vessel.WithAliases(ServiceNameHookRegistry)); err != nil {
		return fmt.Errorf("failed to register hook registry: %w", err)
	}

	return nil
}

// GetServices returns a map of all available services for inspection.
func (p *Plugin) GetServices() map[string]any {
	return map[string]any{
		"planService":         p.planSvc,
		"subscriptionService": p.subscriptionSvc,
		"addOnService":        p.addOnSvc,
		"invoiceService":      p.invoiceSvc,
		"usageService":        p.usageSvc,
		"paymentService":      p.paymentSvc,
		"customerService":     p.customerSvc,
		"enforcementService":  p.enforcementSvc,
		"featureService":      p.featureSvc,
		"featureUsageService": p.featureUsageSvc,
		"alertService":        p.alertSvc,
		"analyticsService":    p.analyticsSvc,
		"couponService":       p.couponSvc,
		"currencyService":     p.currencySvc,
		"taxService":          p.taxSvc,
		"hookRegistry":        p.subHookRegistry,
	}
}
