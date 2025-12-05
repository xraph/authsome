package subscription

import (
	"fmt"

	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
)

// Service name constants for DI container registration
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

// ResolveSubscriptionPlugin resolves the subscription plugin from the container
func ResolveSubscriptionPlugin(container forge.Container) (*Plugin, error) {
	resolved, err := container.Resolve(ServiceNamePlugin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subscription plugin: %w", err)
	}
	plugin, ok := resolved.(*Plugin)
	if !ok {
		return nil, fmt.Errorf("invalid subscription plugin type")
	}
	return plugin, nil
}

// ResolvePlanService resolves the plan service from the container
func ResolvePlanService(container forge.Container) (*service.PlanService, error) {
	resolved, err := container.Resolve(ServiceNamePlanService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve plan service: %w", err)
	}
	svc, ok := resolved.(*service.PlanService)
	if !ok {
		return nil, fmt.Errorf("invalid plan service type")
	}
	return svc, nil
}

// ResolveSubscriptionService resolves the subscription service from the container
func ResolveSubscriptionService(container forge.Container) (*service.SubscriptionService, error) {
	resolved, err := container.Resolve(ServiceNameSubService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subscription service: %w", err)
	}
	svc, ok := resolved.(*service.SubscriptionService)
	if !ok {
		return nil, fmt.Errorf("invalid subscription service type")
	}
	return svc, nil
}

// ResolveAddOnService resolves the add-on service from the container
func ResolveAddOnService(container forge.Container) (*service.AddOnService, error) {
	resolved, err := container.Resolve(ServiceNameAddOnService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve add-on service: %w", err)
	}
	svc, ok := resolved.(*service.AddOnService)
	if !ok {
		return nil, fmt.Errorf("invalid add-on service type")
	}
	return svc, nil
}

// ResolveInvoiceService resolves the invoice service from the container
func ResolveInvoiceService(container forge.Container) (*service.InvoiceService, error) {
	resolved, err := container.Resolve(ServiceNameInvoiceService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve invoice service: %w", err)
	}
	svc, ok := resolved.(*service.InvoiceService)
	if !ok {
		return nil, fmt.Errorf("invalid invoice service type")
	}
	return svc, nil
}

// ResolveUsageService resolves the usage service from the container
func ResolveUsageService(container forge.Container) (*service.UsageService, error) {
	resolved, err := container.Resolve(ServiceNameUsageService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve usage service: %w", err)
	}
	svc, ok := resolved.(*service.UsageService)
	if !ok {
		return nil, fmt.Errorf("invalid usage service type")
	}
	return svc, nil
}

// ResolvePaymentService resolves the payment service from the container
func ResolvePaymentService(container forge.Container) (*service.PaymentService, error) {
	resolved, err := container.Resolve(ServiceNamePaymentService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve payment service: %w", err)
	}
	svc, ok := resolved.(*service.PaymentService)
	if !ok {
		return nil, fmt.Errorf("invalid payment service type")
	}
	return svc, nil
}

// ResolveCustomerService resolves the customer service from the container
func ResolveCustomerService(container forge.Container) (*service.CustomerService, error) {
	resolved, err := container.Resolve(ServiceNameCustomerService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve customer service: %w", err)
	}
	svc, ok := resolved.(*service.CustomerService)
	if !ok {
		return nil, fmt.Errorf("invalid customer service type")
	}
	return svc, nil
}

// ResolveEnforcementService resolves the enforcement service from the container
func ResolveEnforcementService(container forge.Container) (*service.EnforcementService, error) {
	resolved, err := container.Resolve(ServiceNameEnforcementService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve enforcement service: %w", err)
	}
	svc, ok := resolved.(*service.EnforcementService)
	if !ok {
		return nil, fmt.Errorf("invalid enforcement service type")
	}
	return svc, nil
}

// ResolveSubscriptionHookRegistry resolves the subscription hook registry from the container
func ResolveSubscriptionHookRegistry(container forge.Container) (*SubscriptionHookRegistry, error) {
	resolved, err := container.Resolve(ServiceNameHookRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve subscription hook registry: %w", err)
	}
	registry, ok := resolved.(*SubscriptionHookRegistry)
	if !ok {
		return nil, fmt.Errorf("invalid subscription hook registry type")
	}
	return registry, nil
}

// ResolveFeatureService resolves the feature service from the container
func ResolveFeatureService(container forge.Container) (*service.FeatureService, error) {
	resolved, err := container.Resolve(ServiceNameFeatureService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve feature service: %w", err)
	}
	svc, ok := resolved.(*service.FeatureService)
	if !ok {
		return nil, fmt.Errorf("invalid feature service type")
	}
	return svc, nil
}

// ResolveFeatureUsageService resolves the feature usage service from the container
func ResolveFeatureUsageService(container forge.Container) (*service.FeatureUsageService, error) {
	resolved, err := container.Resolve(ServiceNameFeatureUsageService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve feature usage service: %w", err)
	}
	svc, ok := resolved.(*service.FeatureUsageService)
	if !ok {
		return nil, fmt.Errorf("invalid feature usage service type")
	}
	return svc, nil
}

// ResolveAlertService resolves the alert service from the container
func ResolveAlertService(container forge.Container) (*service.AlertService, error) {
	resolved, err := container.Resolve(ServiceNameAlertService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve alert service: %w", err)
	}
	svc, ok := resolved.(*service.AlertService)
	if !ok {
		return nil, fmt.Errorf("invalid alert service type")
	}
	return svc, nil
}

// ResolveAnalyticsService resolves the analytics service from the container
func ResolveAnalyticsService(container forge.Container) (*service.AnalyticsService, error) {
	resolved, err := container.Resolve(ServiceNameAnalyticsService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve analytics service: %w", err)
	}
	svc, ok := resolved.(*service.AnalyticsService)
	if !ok {
		return nil, fmt.Errorf("invalid analytics service type")
	}
	return svc, nil
}

// ResolveCouponService resolves the coupon service from the container
func ResolveCouponService(container forge.Container) (*service.CouponService, error) {
	resolved, err := container.Resolve(ServiceNameCouponService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve coupon service: %w", err)
	}
	svc, ok := resolved.(*service.CouponService)
	if !ok {
		return nil, fmt.Errorf("invalid coupon service type")
	}
	return svc, nil
}

// ResolveCurrencyService resolves the currency service from the container
func ResolveCurrencyService(container forge.Container) (*service.CurrencyService, error) {
	resolved, err := container.Resolve(ServiceNameCurrencyService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve currency service: %w", err)
	}
	svc, ok := resolved.(*service.CurrencyService)
	if !ok {
		return nil, fmt.Errorf("invalid currency service type")
	}
	return svc, nil
}

// ResolveTaxService resolves the tax service from the container
func ResolveTaxService(container forge.Container) (*service.TaxService, error) {
	resolved, err := container.Resolve(ServiceNameTaxService)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax service: %w", err)
	}
	svc, ok := resolved.(*service.TaxService)
	if !ok {
		return nil, fmt.Errorf("invalid tax service type")
	}
	return svc, nil
}

// RegisterServices registers all subscription services in the DI container
func (p *Plugin) RegisterServices(container forge.Container) error {
	// Register plugin itself
	forge.RegisterSingleton(container, ServiceNamePlugin, func(_ forge.Container) (any, error) {
		return p, nil
	})

	// Register core services
	forge.RegisterSingleton(container, ServiceNamePlanService, func(_ forge.Container) (any, error) {
		return p.planSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameSubService, func(_ forge.Container) (any, error) {
		return p.subscriptionSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameAddOnService, func(_ forge.Container) (any, error) {
		return p.addOnSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameInvoiceService, func(_ forge.Container) (any, error) {
		return p.invoiceSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameUsageService, func(_ forge.Container) (any, error) {
		return p.usageSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNamePaymentService, func(_ forge.Container) (any, error) {
		return p.paymentSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameCustomerService, func(_ forge.Container) (any, error) {
		return p.customerSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameEnforcementService, func(_ forge.Container) (any, error) {
		return p.enforcementSvc, nil
	})

	// Register feature services
	forge.RegisterSingleton(container, ServiceNameFeatureService, func(_ forge.Container) (any, error) {
		return p.featureSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameFeatureUsageService, func(_ forge.Container) (any, error) {
		return p.featureUsageSvc, nil
	})

	// Register additional services
	forge.RegisterSingleton(container, ServiceNameAlertService, func(_ forge.Container) (any, error) {
		return p.alertSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameAnalyticsService, func(_ forge.Container) (any, error) {
		return p.analyticsSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameCouponService, func(_ forge.Container) (any, error) {
		return p.couponSvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameCurrencyService, func(_ forge.Container) (any, error) {
		return p.currencySvc, nil
	})

	forge.RegisterSingleton(container, ServiceNameTaxService, func(_ forge.Container) (any, error) {
		return p.taxSvc, nil
	})

	// Register hook registry
	forge.RegisterSingleton(container, ServiceNameHookRegistry, func(_ forge.Container) (any, error) {
		return p.subHookRegistry, nil
	})

	return nil
}

// GetServices returns a map of all available services for inspection
func (p *Plugin) GetServices() map[string]interface{} {
	return map[string]interface{}{
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
