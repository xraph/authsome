// Package repository provides data access interfaces and implementations for the subscription plugin.
package repository

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/core"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// PlanRepository defines the interface for plan persistence operations.
type PlanRepository interface {
	// Create creates a new plan
	Create(ctx context.Context, plan *schema.SubscriptionPlan) error

	// Update updates an existing plan
	Update(ctx context.Context, plan *schema.SubscriptionPlan) error

	// Delete soft-deletes a plan
	Delete(ctx context.Context, id xid.ID) error

	// FindByID retrieves a plan by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionPlan, error)

	// FindBySlug retrieves a plan by slug within an app
	FindBySlug(ctx context.Context, appID xid.ID, slug string) (*schema.SubscriptionPlan, error)

	// FindByProviderID retrieves a plan by provider plan ID (e.g., Stripe product ID)
	FindByProviderID(ctx context.Context, providerPlanID string) (*schema.SubscriptionPlan, error)

	// List retrieves plans with optional filters
	List(ctx context.Context, filter *PlanFilter) ([]*schema.SubscriptionPlan, int, error)

	// CreateFeature creates a plan feature
	CreateFeature(ctx context.Context, feature *schema.SubscriptionPlanFeature) error

	// DeleteFeatures deletes all features for a plan
	DeleteFeatures(ctx context.Context, planID xid.ID) error

	// CreateTier creates a pricing tier
	CreateTier(ctx context.Context, tier *schema.SubscriptionPlanTier) error

	// DeleteTiers deletes all tiers for a plan
	DeleteTiers(ctx context.Context, planID xid.ID) error
}

// PlanFilter defines filters for listing plans.
type PlanFilter struct {
	AppID    *xid.ID
	IsActive *bool
	IsPublic *bool
	Page     int
	PageSize int
}

// SubscriptionRepository defines the interface for subscription persistence operations.
type SubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, sub *schema.Subscription) error

	// Update updates an existing subscription
	Update(ctx context.Context, sub *schema.Subscription) error

	// Delete soft-deletes a subscription
	Delete(ctx context.Context, id xid.ID) error

	// FindByID retrieves a subscription by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.Subscription, error)

	// FindByOrganizationID retrieves the active subscription for an organization
	FindByOrganizationID(ctx context.Context, orgID xid.ID) (*schema.Subscription, error)

	// FindByProviderID retrieves a subscription by provider subscription ID
	FindByProviderID(ctx context.Context, providerSubID string) (*schema.Subscription, error)

	// List retrieves subscriptions with optional filters
	List(ctx context.Context, filter *SubscriptionFilter) ([]*schema.Subscription, int, error)

	// CreateAddOnItem attaches an add-on to a subscription
	CreateAddOnItem(ctx context.Context, item *schema.SubscriptionAddOnItem) error

	// DeleteAddOnItem removes an add-on from a subscription
	DeleteAddOnItem(ctx context.Context, subscriptionID, addOnID xid.ID) error

	// GetAddOnItems retrieves all add-ons for a subscription
	GetAddOnItems(ctx context.Context, subscriptionID xid.ID) ([]*schema.SubscriptionAddOnItem, error)
}

// SubscriptionFilter defines filters for listing subscriptions.
type SubscriptionFilter struct {
	AppID          *xid.ID
	OrganizationID *xid.ID
	PlanID         *xid.ID
	Status         string
	Page           int
	PageSize       int
}

// AddOnRepository defines the interface for add-on persistence operations.
type AddOnRepository interface {
	// Create creates a new add-on
	Create(ctx context.Context, addon *schema.SubscriptionAddOn) error

	// Update updates an existing add-on
	Update(ctx context.Context, addon *schema.SubscriptionAddOn) error

	// Delete soft-deletes an add-on
	Delete(ctx context.Context, id xid.ID) error

	// FindByID retrieves an add-on by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionAddOn, error)

	// FindBySlug retrieves an add-on by slug within an app
	FindBySlug(ctx context.Context, appID xid.ID, slug string) (*schema.SubscriptionAddOn, error)

	// List retrieves add-ons with optional filters
	List(ctx context.Context, filter *AddOnFilter) ([]*schema.SubscriptionAddOn, int, error)

	// CreateFeature creates an add-on feature
	CreateFeature(ctx context.Context, feature *schema.SubscriptionAddOnFeature) error

	// DeleteFeatures deletes all features for an add-on
	DeleteFeatures(ctx context.Context, addOnID xid.ID) error

	// CreateTier creates a pricing tier
	CreateTier(ctx context.Context, tier *schema.SubscriptionAddOnTier) error

	// DeleteTiers deletes all tiers for an add-on
	DeleteTiers(ctx context.Context, addOnID xid.ID) error
}

// AddOnFilter defines filters for listing add-ons.
type AddOnFilter struct {
	AppID    *xid.ID
	IsActive *bool
	IsPublic *bool
	Page     int
	PageSize int
}

// InvoiceRepository defines the interface for invoice persistence operations.
type InvoiceRepository interface {
	// Create creates a new invoice
	Create(ctx context.Context, invoice *schema.SubscriptionInvoice) error

	// Update updates an existing invoice
	Update(ctx context.Context, invoice *schema.SubscriptionInvoice) error

	// FindByID retrieves an invoice by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionInvoice, error)

	// FindByNumber retrieves an invoice by number
	FindByNumber(ctx context.Context, number string) (*schema.SubscriptionInvoice, error)

	// FindByProviderID retrieves an invoice by provider invoice ID
	FindByProviderID(ctx context.Context, providerInvoiceID string) (*schema.SubscriptionInvoice, error)

	// List retrieves invoices with optional filters
	List(ctx context.Context, filter *InvoiceFilter) ([]*schema.SubscriptionInvoice, int, error)

	// CreateItem creates an invoice line item
	CreateItem(ctx context.Context, item *schema.SubscriptionInvoiceItem) error

	// GetItems retrieves all items for an invoice
	GetItems(ctx context.Context, invoiceID xid.ID) ([]*schema.SubscriptionInvoiceItem, error)

	// GetNextInvoiceNumber generates the next invoice number
	GetNextInvoiceNumber(ctx context.Context, appID xid.ID) (string, error)
}

// InvoiceFilter defines filters for listing invoices.
type InvoiceFilter struct {
	OrganizationID *xid.ID
	SubscriptionID *xid.ID
	Status         string
	Page           int
	PageSize       int
}

// UsageRepository defines the interface for usage record persistence operations.
type UsageRepository interface {
	// Create creates a new usage record
	Create(ctx context.Context, record *schema.SubscriptionUsageRecord) error

	// FindByID retrieves a usage record by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionUsageRecord, error)

	// FindByIdempotencyKey retrieves a usage record by idempotency key
	FindByIdempotencyKey(ctx context.Context, key string) (*schema.SubscriptionUsageRecord, error)

	// List retrieves usage records with optional filters
	List(ctx context.Context, filter *UsageFilter) ([]*schema.SubscriptionUsageRecord, int, error)

	// GetSummary calculates usage summary for a subscription and metric
	GetSummary(ctx context.Context, subscriptionID xid.ID, metricKey string, periodStart, periodEnd any) (*UsageSummary, error)

	// GetUnreported retrieves usage records not yet reported to provider
	GetUnreported(ctx context.Context, limit int) ([]*schema.SubscriptionUsageRecord, error)

	// MarkReported marks a usage record as reported
	MarkReported(ctx context.Context, id xid.ID, providerRecordID string) error
}

// UsageFilter defines filters for listing usage records.
type UsageFilter struct {
	SubscriptionID *xid.ID
	OrganizationID *xid.ID
	MetricKey      string
	Reported       *bool
	Page           int
	PageSize       int
}

// UsageSummary represents aggregated usage data.
type UsageSummary struct {
	MetricKey     string
	TotalQuantity int64
	RecordCount   int64
}

// PaymentMethodRepository defines the interface for payment method persistence operations.
type PaymentMethodRepository interface {
	// Create creates a new payment method
	Create(ctx context.Context, pm *schema.SubscriptionPaymentMethod) error

	// Update updates an existing payment method
	Update(ctx context.Context, pm *schema.SubscriptionPaymentMethod) error

	// Delete deletes a payment method
	Delete(ctx context.Context, id xid.ID) error

	// FindByID retrieves a payment method by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionPaymentMethod, error)

	// FindByProviderID retrieves a payment method by provider method ID
	FindByProviderID(ctx context.Context, providerMethodID string) (*schema.SubscriptionPaymentMethod, error)

	// ListByOrganization retrieves all payment methods for an organization
	ListByOrganization(ctx context.Context, orgID xid.ID) ([]*schema.SubscriptionPaymentMethod, error)

	// GetDefault retrieves the default payment method for an organization
	GetDefault(ctx context.Context, orgID xid.ID) (*schema.SubscriptionPaymentMethod, error)

	// SetDefault sets a payment method as the default
	SetDefault(ctx context.Context, orgID, paymentMethodID xid.ID) error

	// ClearDefault clears the default flag on all payment methods for an organization
	ClearDefault(ctx context.Context, orgID xid.ID) error
}

// CustomerRepository defines the interface for customer persistence operations.
type CustomerRepository interface {
	// Create creates a new customer
	Create(ctx context.Context, customer *schema.SubscriptionCustomer) error

	// Update updates an existing customer
	Update(ctx context.Context, customer *schema.SubscriptionCustomer) error

	// Delete deletes a customer
	Delete(ctx context.Context, id xid.ID) error

	// FindByID retrieves a customer by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionCustomer, error)

	// FindByOrganizationID retrieves a customer by organization ID
	FindByOrganizationID(ctx context.Context, orgID xid.ID) (*schema.SubscriptionCustomer, error)

	// FindByProviderID retrieves a customer by provider customer ID
	FindByProviderID(ctx context.Context, providerCustomerID string) (*schema.SubscriptionCustomer, error)
}

// EventRepository defines the interface for subscription event persistence operations.
type EventRepository interface {
	// Create creates a new event
	Create(ctx context.Context, event *schema.SubscriptionEvent) error

	// FindByID retrieves an event by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionEvent, error)

	// List retrieves events with optional filters
	List(ctx context.Context, filter *EventFilter) ([]*schema.SubscriptionEvent, int, error)
}

// EventFilter defines filters for listing events.
type EventFilter struct {
	SubscriptionID *xid.ID
	OrganizationID *xid.ID
	EventType      string
	Page           int
	PageSize       int
}

// FeatureRepository defines the interface for feature persistence operations.
type FeatureRepository interface {
	// Create creates a new feature
	Create(ctx context.Context, feature *schema.Feature) error

	// Update updates an existing feature
	Update(ctx context.Context, feature *schema.Feature) error

	// Delete deletes a feature
	Delete(ctx context.Context, id xid.ID) error

	// FindByID retrieves a feature by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.Feature, error)

	// FindByKey retrieves a feature by key within an app
	FindByKey(ctx context.Context, appID xid.ID, key string) (*schema.Feature, error)

	// List retrieves features with optional filters
	List(ctx context.Context, filter *FeatureFilter) ([]*schema.Feature, int, error)

	// CreateTier creates a feature tier
	CreateTier(ctx context.Context, tier *schema.FeatureTier) error

	// DeleteTiers deletes all tiers for a feature
	DeleteTiers(ctx context.Context, featureID xid.ID) error

	// GetTiers retrieves all tiers for a feature
	GetTiers(ctx context.Context, featureID xid.ID) ([]*schema.FeatureTier, error)

	// LinkToPlan links a feature to a plan
	LinkToPlan(ctx context.Context, link *schema.PlanFeatureLink) error

	// UpdatePlanLink updates a feature-plan link
	UpdatePlanLink(ctx context.Context, link *schema.PlanFeatureLink) error

	// UnlinkFromPlan removes a feature from a plan
	UnlinkFromPlan(ctx context.Context, planID, featureID xid.ID) error

	// GetPlanLinks retrieves all feature links for a plan
	GetPlanLinks(ctx context.Context, planID xid.ID) ([]*schema.PlanFeatureLink, error)

	// GetPlanLink retrieves a specific feature link
	GetPlanLink(ctx context.Context, planID, featureID xid.ID) (*schema.PlanFeatureLink, error)

	// GetFeaturePlans retrieves all plans that have a feature
	GetFeaturePlans(ctx context.Context, featureID xid.ID) ([]*schema.PlanFeatureLink, error)
}

// FeatureFilter defines filters for listing features.
type FeatureFilter struct {
	AppID    *xid.ID
	Type     string
	IsPublic *bool
	Page     int
	PageSize int
}

// FeatureUsageRepository defines the interface for feature usage persistence operations.
type FeatureUsageRepository interface {
	// CreateUsage creates or updates feature usage for an organization
	CreateUsage(ctx context.Context, usage *schema.OrganizationFeatureUsage) error

	// UpdateUsage updates feature usage
	UpdateUsage(ctx context.Context, usage *schema.OrganizationFeatureUsage) error

	// FindUsage retrieves feature usage for an organization and feature
	FindUsage(ctx context.Context, orgID, featureID xid.ID) (*schema.OrganizationFeatureUsage, error)

	// FindUsageByKey retrieves feature usage by feature key
	FindUsageByKey(ctx context.Context, orgID, appID xid.ID, featureKey string) (*schema.OrganizationFeatureUsage, error)

	// ListUsage retrieves all feature usage for an organization
	ListUsage(ctx context.Context, orgID xid.ID) ([]*schema.OrganizationFeatureUsage, error)

	// IncrementUsage atomically increments usage by a quantity
	IncrementUsage(ctx context.Context, orgID, featureID xid.ID, quantity int64) (*schema.OrganizationFeatureUsage, error)

	// DecrementUsage atomically decrements usage by a quantity
	DecrementUsage(ctx context.Context, orgID, featureID xid.ID, quantity int64) (*schema.OrganizationFeatureUsage, error)

	// ResetUsage resets usage to zero
	ResetUsage(ctx context.Context, orgID, featureID xid.ID) error

	// CreateLog creates a usage log entry
	CreateLog(ctx context.Context, log *schema.FeatureUsageLog) error

	// ListLogs retrieves usage logs with filters
	ListLogs(ctx context.Context, filter *FeatureUsageLogFilter) ([]*schema.FeatureUsageLog, int, error)

	// FindLogByIdempotencyKey finds a log by idempotency key
	FindLogByIdempotencyKey(ctx context.Context, key string) (*schema.FeatureUsageLog, error)

	// CreateGrant creates a feature grant
	CreateGrant(ctx context.Context, grant *schema.FeatureGrant) error

	// UpdateGrant updates a feature grant
	UpdateGrant(ctx context.Context, grant *schema.FeatureGrant) error

	// DeleteGrant deletes a feature grant
	DeleteGrant(ctx context.Context, id xid.ID) error

	// FindGrantByID retrieves a grant by ID
	FindGrantByID(ctx context.Context, id xid.ID) (*schema.FeatureGrant, error)

	// ListGrants retrieves all active grants for an organization and feature
	ListGrants(ctx context.Context, orgID, featureID xid.ID) ([]*schema.FeatureGrant, error)

	// ListAllOrgGrants retrieves all active grants for an organization
	ListAllOrgGrants(ctx context.Context, orgID xid.ID) ([]*schema.FeatureGrant, error)

	// Dashboard queries
	// GetCurrentUsageSnapshot retrieves all current usage across all organizations for an app
	GetCurrentUsageSnapshot(ctx context.Context, appID xid.ID) ([]*core.CurrentUsage, error)

	// GetUsageByOrg retrieves usage statistics by organization
	GetUsageByOrg(ctx context.Context, appID xid.ID, startDate, endDate time.Time) ([]*core.OrgUsageStats, error)

	// GetUsageTrends retrieves usage trends over time for a feature
	GetUsageTrends(ctx context.Context, appID xid.ID, featureID *xid.ID, startDate, endDate time.Time) ([]*core.UsageTrend, error)

	// GetTopConsumers retrieves top consuming organizations
	GetTopConsumers(ctx context.Context, appID xid.ID, featureID *xid.ID, startDate, endDate time.Time, limit int) ([]*core.OrgUsageStats, error)

	// GetUsageByFeatureType retrieves usage aggregated by feature type
	GetUsageByFeatureType(ctx context.Context, appID xid.ID, startDate, endDate time.Time) (map[core.FeatureType]*core.UsageStats, error)

	// GetTotalGrantedValue calculates total granted quota for an organization and feature
	GetTotalGrantedValue(ctx context.Context, orgID, featureID xid.ID) (int64, error)

	// ExpireGrants marks expired grants as inactive
	ExpireGrants(ctx context.Context) error

	// GetUsageNeedingReset retrieves usage records that need to be reset
	GetUsageNeedingReset(ctx context.Context, resetPeriod string) ([]*schema.OrganizationFeatureUsage, error)
}

// FeatureUsageLogFilter defines filters for listing usage logs.
type FeatureUsageLogFilter struct {
	OrganizationID *xid.ID
	FeatureID      *xid.ID
	Action         string
	Page           int
	PageSize       int
}
