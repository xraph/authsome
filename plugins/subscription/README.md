# Subscription Plugin for AuthSome

A comprehensive SaaS subscription and billing plugin for AuthSome, providing complete subscription management with organization-scoped billing.

## 🆕 New Features (December 2025)

### Export & Import
- **Export** all features and plans as JSON for backup or migration
- **Import** features and plans from JSON to quickly set up new environments
- Perfect for local development and multi-environment deployments
- See [Export/Import Guide](./SUBSCRIPTION_SYNC_EXPORT_IMPORT_GUIDE.md)

### Enhanced Stripe Sync
- **Sync FROM Stripe** - Import existing Stripe products into AuthSome
- **Sync TO Stripe** - Push AuthSome plans to Stripe  
- Automatic metadata management for seamless bi-directional sync
- Bulk sync all plans with one click

## Features

- **Plan Management**: Create and manage subscription plans with flexible pricing
  - Flat rate pricing
  - Per-seat pricing
  - Tiered pricing
  - Usage-based (metered) billing
  - Hybrid billing patterns
  
- **Subscription Lifecycle**: Full subscription management
  - Trial periods
  - Plan upgrades/downgrades
  - Subscription pausing/resuming
  - Cancellation with grace periods
  
- **Add-ons**: Additional features and services
  - One-time purchases
  - Recurring add-ons
  - Usage-based add-ons
  
- **Usage Tracking**: Metered billing support
  - API call tracking
  - Storage usage
  - Custom metrics
  
- **Payment Providers**: Pluggable payment gateway integration
  - Stripe (fully implemented)
  - Mock provider for development/testing
  - Custom provider support via `WithProvider` option
  
- **Enforcement**: Plan limit enforcement via hooks
  - Seat/member limits
  - Feature access control
  - Usage limits

## Installation

Add the subscription plugin when creating your AuthSome instance:

```go
import (
    "github.com/xraph/authsome"
    "github.com/xraph/authsome/plugins/subscription"
)

// Create plugin with configuration
subPlugin := subscription.NewPlugin(
    subscription.WithStripeConfig(
        os.Getenv("STRIPE_SECRET_KEY"),
        os.Getenv("STRIPE_WEBHOOK_SECRET"),
        os.Getenv("STRIPE_PUBLISHABLE_KEY"),
    ),
    subscription.WithDefaultTrialDays(14),
    subscription.WithRequireSubscription(false),
    subscription.WithGracePeriodDays(7),
    subscription.WithAutoSyncSeats(true),
)

// Or use a custom payment provider
customProvider := mypackage.NewCustomProvider()
subPlugin := subscription.NewPlugin(
    subscription.WithProvider(customProvider),
    subscription.WithDefaultTrialDays(14),
)

// Register with AuthSome
auth, err := authsome.New(
    authsome.WithPlugins(subPlugin),
    // ... other options
)
```

## Configuration

The plugin can be configured via YAML or environment variables:

```yaml
auth:
  subscription:
    enabled: true
    requireSubscription: false
    defaultTrialDays: 14
    gracePeriodDays: 7
    autoSyncSeats: false
    provider: stripe
    stripe:
      secretKey: ${STRIPE_SECRET_KEY}
      webhookSecret: ${STRIPE_WEBHOOK_SECRET}
      publishableKey: ${STRIPE_PUBLISHABLE_KEY}
```

## Custom Payment Provider

You can implement your own payment provider by implementing the `providers.PaymentProvider` interface:

```go
import (
    "github.com/xraph/authsome/plugins/subscription/providers"
    "github.com/xraph/authsome/plugins/subscription/providers/types"
)

type CustomProvider struct {
    // Your custom fields
}

// Implement all PaymentProvider interface methods
func (p *CustomProvider) CreateCustomer(ctx context.Context, req *types.CreateCustomerRequest) (*types.ProviderCustomer, error) {
    // Your implementation
}

func (p *CustomProvider) CreateProduct(ctx context.Context, req *types.CreateProductRequest) (*types.ProviderProduct, error) {
    // Your implementation
}

// ... implement remaining methods

// Use your custom provider
customProvider := &CustomProvider{}
subPlugin := subscription.NewPlugin(
    subscription.WithProvider(customProvider),
)
```

This is useful for:
- Integrating with payment providers other than Stripe (PayPal, Paddle, etc.)
- Testing with mock providers
- Custom billing logic specific to your application
- Migrating from existing billing systems

## API Endpoints

### Plans

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/subscription/plans` | Create a plan |
| GET | `/subscription/plans` | List all plans |
| GET | `/subscription/plans/:id` | Get plan details |
| PATCH | `/subscription/plans/:id` | Update a plan |
| DELETE | `/subscription/plans/:id` | Delete a plan |
| POST | `/subscription/plans/:id/sync` | Sync plan to provider |

### Subscriptions

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/subscription/subscriptions` | Create subscription |
| GET | `/subscription/subscriptions` | List subscriptions |
| GET | `/subscription/subscriptions/:id` | Get subscription |
| GET | `/subscription/subscriptions/organization/:orgId` | Get org subscription |
| PATCH | `/subscription/subscriptions/:id` | Update subscription |
| POST | `/subscription/subscriptions/:id/cancel` | Cancel subscription |
| POST | `/subscription/subscriptions/:id/pause` | Pause subscription |
| POST | `/subscription/subscriptions/:id/resume` | Resume subscription |

### Add-ons

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/subscription/addons` | Create add-on |
| GET | `/subscription/addons` | List add-ons |
| GET | `/subscription/addons/:id` | Get add-on |
| PATCH | `/subscription/addons/:id` | Update add-on |
| DELETE | `/subscription/addons/:id` | Delete add-on |

### Invoices

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/subscription/invoices` | List invoices |
| GET | `/subscription/invoices/:id` | Get invoice |

### Usage

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/subscription/usage` | Record usage |
| GET | `/subscription/usage/summary` | Get usage summary |

### Checkout

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/subscription/checkout` | Create checkout session |
| POST | `/subscription/checkout/portal` | Create customer portal |

### Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/subscription/webhooks/stripe` | Stripe webhook handler |

## Using Services in Your Application

The subscription plugin registers its services with the Forge DI container:

```go
// Get services from the plugin
plugin := auth.GetPlugin("subscription").(*subscription.Plugin)

// Plan Service
planSvc := plugin.GetPlanService()
plan, err := planSvc.Create(ctx, &core.CreatePlanRequest{
    Name:            "Pro Plan",
    Slug:            "pro",
    BillingPattern:  "per_seat",
    BillingInterval: "monthly",
    BasePrice:       2999, // $29.99 in cents
    Currency:        "usd",
})

// Subscription Service
subSvc := plugin.GetSubscriptionService()
sub, err := subSvc.Create(ctx, &core.CreateSubscriptionRequest{
    OrganizationID: orgID,
    PlanID:         plan.ID,
    Quantity:       5,
})

// Usage Service
usageSvc := plugin.GetUsageService()
err := usageSvc.RecordUsage(ctx, &core.CreateUsageRecordRequest{
    SubscriptionID: sub.ID,
    MetricKey:      "api_calls",
    Quantity:       1000,
})

// Enforcement Service
enforceSvc := plugin.GetEnforcementService()
hasAccess, err := enforceSvc.CheckFeatureAccess(ctx, orgID, "advanced_analytics")
```

## Resolving Services from Container

```go
// Using helper functions
planSvc, err := subscription.ResolvePlanService(container)
subSvc, err := subscription.ResolveSubscriptionService(container)
enforceSvc, err := subscription.ResolveEnforcementService(container)

// Or resolve directly
svc, err := container.Resolve("subscription.plan")
planSvc := svc.(*service.PlanService)
```

## Hooks

The plugin provides hooks for subscription lifecycle events:

```go
// Get hook registry
hookRegistry := plugin.GetHookRegistry()

// Before subscription creation
hookRegistry.RegisterBeforeSubscriptionCreate(func(ctx context.Context, orgID, planID xid.ID) error {
    // Custom validation logic
    return nil
})

// After subscription creation
hookRegistry.RegisterAfterSubscriptionCreate(func(ctx context.Context, sub *core.Subscription) error {
    // Send welcome email, provision resources, etc.
    return nil
})

// On payment success
hookRegistry.RegisterOnPaymentSuccess(func(ctx context.Context, subID, invoiceID xid.ID, amount int64, currency string) error {
    // Record in analytics, send receipt, etc.
    return nil
})

// On payment failure
hookRegistry.RegisterOnPaymentFailed(func(ctx context.Context, subID, invoiceID xid.ID, amount int64, currency string, reason string) error {
    // Send notification, flag account, etc.
    return nil
})

// On trial ending (typically 3 days before)
hookRegistry.RegisterOnTrialEnding(func(ctx context.Context, subID xid.ID, daysRemaining int) error {
    // Send reminder email
    return nil
})
```

## Organization Enforcement

The plugin automatically integrates with AuthSome's organization service:

```go
// These hooks are registered automatically
// Enforces seat limits when adding members
hooks.RegisterBeforeMemberAdd(enforcementSvc.EnforceSeatLimit)

// Enforces subscription requirement for org creation (if enabled)
hooks.RegisterBeforeOrganizationCreate(enforcementSvc.EnforceSubscriptionRequired)
```

## Dashboard UI

The plugin provides a dashboard extension with:
- Billing overview page
- Plans management
- Subscriptions list
- Add-ons management
- Invoices viewer
- Usage dashboard
- Billing settings

## Plan Features

Plans can include features with different types:

```go
plan := &core.Plan{
    Features: []core.PlanFeature{
        {
            Key:   "max_members",
            Type:  "limit",
            Value: "10",  // JSON value
        },
        {
            Key:   "api_access",
            Type:  "boolean",
            Value: "true",
        },
        {
            Key:   "storage",
            Type:  "limit",
            Value: "5368709120", // 5GB in bytes
        },
        {
            Key:   "unlimited_projects",
            Type:  "unlimited",
            Value: "",
        },
    },
}
```

## Billing Patterns

### Flat Rate
```go
plan := &core.CreatePlanRequest{
    BillingPattern: "flat",
    BasePrice:      9999, // $99.99/month
}
```

### Per-Seat
```go
plan := &core.CreatePlanRequest{
    BillingPattern: "per_seat",
    BasePrice:      1999, // $19.99/seat/month
}
```

### Tiered
```go
plan := &core.CreatePlanRequest{
    BillingPattern: "tiered",
    TierMode:       "graduated",
    PriceTiers: []core.PriceTier{
        {UpTo: 10, UnitPrice: 1000},    // First 10: $10/each
        {UpTo: 50, UnitPrice: 800},     // 11-50: $8/each
        {UpTo: 0, UnitPrice: 500},      // 51+: $5/each (0 = unlimited)
    },
}
```

### Usage-Based
```go
plan := &core.CreatePlanRequest{
    BillingPattern: "usage",
    MeteredFeatures: []core.MeteredFeature{
        {Key: "api_calls", UnitPrice: 1, UnitName: "request"},
    },
}
```

## License

See the main AuthSome LICENSE file.
