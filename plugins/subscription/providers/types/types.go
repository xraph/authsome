// Package types defines shared types for payment providers.
package types

import (
	"context"

	"github.com/xraph/authsome/plugins/subscription/core"
)

// PaymentProvider defines the interface for payment provider implementations
type PaymentProvider interface {
	// Provider info
	Name() string

	// Customer management
	CreateCustomer(ctx context.Context, email, name string, metadata map[string]interface{}) (customerID string, err error)
	UpdateCustomer(ctx context.Context, customerID, email, name string, metadata map[string]interface{}) error
	DeleteCustomer(ctx context.Context, customerID string) error

	// Product/Price sync (push to provider)
	SyncPlan(ctx context.Context, plan *core.Plan) error
	SyncAddOn(ctx context.Context, addon *core.AddOn) error

	// Product/Price fetch (pull from provider)
	ListProducts(ctx context.Context) ([]*ProviderProduct, error)
	GetProduct(ctx context.Context, productID string) (*ProviderProduct, error)
	ListPrices(ctx context.Context, productID string) ([]*ProviderPrice, error)

	// Subscription management
	CreateSubscription(ctx context.Context, customerID string, priceID string, quantity int, trialDays int, metadata map[string]interface{}) (subscriptionID string, err error)
	UpdateSubscription(ctx context.Context, subscriptionID string, priceID string, quantity int) error
	CancelSubscription(ctx context.Context, subscriptionID string, immediate bool) error
	PauseSubscription(ctx context.Context, subscriptionID string) error
	ResumeSubscription(ctx context.Context, subscriptionID string) error
	GetSubscription(ctx context.Context, subscriptionID string) (*ProviderSubscription, error)

	// Checkout
	CreateCheckoutSession(ctx context.Context, req *CheckoutRequest) (*CheckoutSession, error)
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (url string, err error)

	// Payment methods
	CreateSetupIntent(ctx context.Context, customerID string) (*core.SetupIntentResult, error)
	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*core.PaymentMethod, error)
	AttachPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error
	DetachPaymentMethod(ctx context.Context, paymentMethodID string) error
	SetDefaultPaymentMethod(ctx context.Context, customerID, paymentMethodID string) error

	// Usage reporting
	ReportUsage(ctx context.Context, subscriptionItemID string, records []*core.UsageRecord) (recordID string, err error)

	// Invoices
	GetInvoice(ctx context.Context, invoiceID string) (*ProviderInvoice, error)
	GetInvoicePDF(ctx context.Context, invoiceID string) (url string, err error)
	VoidInvoice(ctx context.Context, invoiceID string) error

	// Webhooks
	HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
}

// ProviderSubscription represents subscription data from the provider
type ProviderSubscription struct {
	ID                 string
	CustomerID         string
	Status             string
	CurrentPeriodStart int64
	CurrentPeriodEnd   int64
	TrialStart         *int64
	TrialEnd           *int64
	CancelAt           *int64
	CanceledAt         *int64
	EndedAt            *int64
	PriceID            string
	Quantity           int
	Metadata           map[string]interface{}
}

// ProviderProduct represents a product from the payment provider
type ProviderProduct struct {
	ID          string
	Name        string
	Description string
	Active      bool
	Metadata    map[string]string
}

// ProviderPrice represents a price from the payment provider
type ProviderPrice struct {
	ID         string
	ProductID  string
	Active     bool
	Currency   string
	UnitAmount int64
	Recurring  *PriceRecurring
	Metadata   map[string]string
}

// PriceRecurring represents recurring billing details for a price
type PriceRecurring struct {
	Interval      string // month, year, week, day
	IntervalCount int
}

// ProviderInvoice represents invoice data from the provider
type ProviderInvoice struct {
	ID             string
	CustomerID     string
	SubscriptionID string
	Status         string
	Currency       string
	AmountDue      int64
	AmountPaid     int64
	Total          int64
	PeriodStart    int64
	PeriodEnd      int64
	PDFURL         string
	HostedURL      string
}

// CheckoutRequest represents a checkout session request
type CheckoutRequest struct {
	CustomerID      string
	PriceID         string
	Quantity        int
	SuccessURL      string
	CancelURL       string
	Mode            CheckoutMode
	AllowPromoCodes bool
	TrialDays       int
	Metadata        map[string]interface{}
}

// CheckoutMode defines the checkout mode
type CheckoutMode string

const (
	CheckoutModeSubscription CheckoutMode = "subscription"
	CheckoutModePayment      CheckoutMode = "payment"
	CheckoutModeSetup        CheckoutMode = "setup"
)

// CheckoutSession represents a checkout session response
type CheckoutSession struct {
	ID            string
	URL           string
	CustomerID    string
	PaymentStatus string
}

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	ID        string
	Type      string
	Data      map[string]interface{}
	Object    interface{}
	Timestamp int64
}

// Common webhook event types
const (
	EventCustomerCreated          = "customer.created"
	EventCustomerUpdated          = "customer.updated"
	EventCustomerDeleted          = "customer.deleted"
	EventSubscriptionCreated      = "customer.subscription.created"
	EventSubscriptionUpdated      = "customer.subscription.updated"
	EventSubscriptionDeleted      = "customer.subscription.deleted"
	EventSubscriptionTrialWillEnd = "customer.subscription.trial_will_end"
	EventInvoiceCreated           = "invoice.created"
	EventInvoicePaid              = "invoice.paid"
	EventInvoicePaymentFailed     = "invoice.payment_failed"
	EventCheckoutSessionCompleted = "checkout.session.completed"
	EventPaymentIntentSucceeded   = "payment_intent.succeeded"
	EventPaymentIntentFailed      = "payment_intent.payment_failed"
	EventPaymentMethodAttached    = "payment_method.attached"
	EventPaymentMethodDetached    = "payment_method.detached"
)
