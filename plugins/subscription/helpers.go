package subscription

import (
	"fmt"

	"github.com/xraph/authsome/plugins/subscription/service"
	"github.com/xraph/forge"
)

// Service name constants for DI container registration
const (
	ServiceNamePlugin         = "subscription.plugin"
	ServiceNamePlanService    = "subscription.plan_service"
	ServiceNameSubService     = "subscription.subscription_service"
	ServiceNameAddOnService   = "subscription.addon_service"
	ServiceNameInvoiceService = "subscription.invoice_service"
	ServiceNameUsageService   = "subscription.usage_service"
	ServiceNamePaymentService = "subscription.payment_service"
	ServiceNameCustomerService = "subscription.customer_service"
	ServiceNameEnforcementService = "subscription.enforcement_service"
	ServiceNameHookRegistry   = "subscription.hook_registry"
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

// RegisterServices registers all subscription services in the DI container
func (p *Plugin) RegisterServices(container forge.Container) error {
	// Register plugin itself
	if err := container.Register(ServiceNamePlugin, func(_ forge.Container) (any, error) {
		return p, nil
	}); err != nil {
		return fmt.Errorf("failed to register subscription plugin: %w", err)
	}

	// Register services
	if err := container.Register(ServiceNamePlanService, func(_ forge.Container) (any, error) {
		return p.planSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register plan service: %w", err)
	}

	if err := container.Register(ServiceNameSubService, func(_ forge.Container) (any, error) {
		return p.subscriptionSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register subscription service: %w", err)
	}

	if err := container.Register(ServiceNameAddOnService, func(_ forge.Container) (any, error) {
		return p.addOnSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register add-on service: %w", err)
	}

	if err := container.Register(ServiceNameInvoiceService, func(_ forge.Container) (any, error) {
		return p.invoiceSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register invoice service: %w", err)
	}

	if err := container.Register(ServiceNameUsageService, func(_ forge.Container) (any, error) {
		return p.usageSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register usage service: %w", err)
	}

	if err := container.Register(ServiceNamePaymentService, func(_ forge.Container) (any, error) {
		return p.paymentSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register payment service: %w", err)
	}

	if err := container.Register(ServiceNameCustomerService, func(_ forge.Container) (any, error) {
		return p.customerSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register customer service: %w", err)
	}

	if err := container.Register(ServiceNameEnforcementService, func(_ forge.Container) (any, error) {
		return p.enforcementSvc, nil
	}); err != nil {
		return fmt.Errorf("failed to register enforcement service: %w", err)
	}

	if err := container.Register(ServiceNameHookRegistry, func(_ forge.Container) (any, error) {
		return p.subHookRegistry, nil
	}); err != nil {
		return fmt.Errorf("failed to register subscription hook registry: %w", err)
	}

	return nil
}

// GetServices returns a map of all available services for inspection
func (p *Plugin) GetServices() map[string]interface{} {
	return map[string]interface{}{
		"planService":        p.planSvc,
		"subscriptionService": p.subscriptionSvc,
		"addOnService":       p.addOnSvc,
		"invoiceService":     p.invoiceSvc,
		"usageService":       p.usageSvc,
		"paymentService":     p.paymentSvc,
		"customerService":    p.customerSvc,
		"enforcementService": p.enforcementSvc,
		"hookRegistry":       p.subHookRegistry,
	}
}

