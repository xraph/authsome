// Package repository provides data access interfaces and implementations for the subscription plugin.
package repository

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/subscription/schema"
)

// PlanRepository defines the interface for plan persistence operations
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

// PlanFilter defines filters for listing plans
type PlanFilter struct {
	AppID    *xid.ID
	IsActive *bool
	IsPublic *bool
	Page     int
	PageSize int
}

// SubscriptionRepository defines the interface for subscription persistence operations
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

// SubscriptionFilter defines filters for listing subscriptions
type SubscriptionFilter struct {
	AppID          *xid.ID
	OrganizationID *xid.ID
	PlanID         *xid.ID
	Status         string
	Page           int
	PageSize       int
}

// AddOnRepository defines the interface for add-on persistence operations
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

// AddOnFilter defines filters for listing add-ons
type AddOnFilter struct {
	AppID    *xid.ID
	IsActive *bool
	IsPublic *bool
	Page     int
	PageSize int
}

// InvoiceRepository defines the interface for invoice persistence operations
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

// InvoiceFilter defines filters for listing invoices
type InvoiceFilter struct {
	OrganizationID *xid.ID
	SubscriptionID *xid.ID
	Status         string
	Page           int
	PageSize       int
}

// UsageRepository defines the interface for usage record persistence operations
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
	GetSummary(ctx context.Context, subscriptionID xid.ID, metricKey string, periodStart, periodEnd interface{}) (*UsageSummary, error)
	
	// GetUnreported retrieves usage records not yet reported to provider
	GetUnreported(ctx context.Context, limit int) ([]*schema.SubscriptionUsageRecord, error)
	
	// MarkReported marks a usage record as reported
	MarkReported(ctx context.Context, id xid.ID, providerRecordID string) error
}

// UsageFilter defines filters for listing usage records
type UsageFilter struct {
	SubscriptionID *xid.ID
	OrganizationID *xid.ID
	MetricKey      string
	Reported       *bool
	Page           int
	PageSize       int
}

// UsageSummary represents aggregated usage data
type UsageSummary struct {
	MetricKey      string
	TotalQuantity  int64
	RecordCount    int64
}

// PaymentMethodRepository defines the interface for payment method persistence operations
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

// CustomerRepository defines the interface for customer persistence operations
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

// EventRepository defines the interface for subscription event persistence operations
type EventRepository interface {
	// Create creates a new event
	Create(ctx context.Context, event *schema.SubscriptionEvent) error
	
	// FindByID retrieves an event by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SubscriptionEvent, error)
	
	// List retrieves events with optional filters
	List(ctx context.Context, filter *EventFilter) ([]*schema.SubscriptionEvent, int, error)
}

// EventFilter defines filters for listing events
type EventFilter struct {
	SubscriptionID *xid.ID
	OrganizationID *xid.ID
	EventType      string
	Page           int
	PageSize       int
}

