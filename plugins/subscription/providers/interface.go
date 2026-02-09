// Package providers defines the payment provider abstraction for the subscription plugin.
package providers

import (
	"github.com/xraph/authsome/plugins/subscription/providers/types"
)

// Re-export types for convenience.
type (
	PaymentProvider      = types.PaymentProvider
	ProviderSubscription = types.ProviderSubscription
	ProviderInvoice      = types.ProviderInvoice
	ProviderProduct      = types.ProviderProduct
	ProviderPrice        = types.ProviderPrice
	PriceRecurring       = types.PriceRecurring
	CheckoutRequest      = types.CheckoutRequest
	CheckoutMode         = types.CheckoutMode
	CheckoutSession      = types.CheckoutSession
	WebhookEvent         = types.WebhookEvent
)

// Re-export checkout modes.
const (
	CheckoutModeSubscription = types.CheckoutModeSubscription
	CheckoutModePayment      = types.CheckoutModePayment
	CheckoutModeSetup        = types.CheckoutModeSetup
)

// Re-export event types.
const (
	EventCustomerCreated          = types.EventCustomerCreated
	EventCustomerUpdated          = types.EventCustomerUpdated
	EventCustomerDeleted          = types.EventCustomerDeleted
	EventSubscriptionCreated      = types.EventSubscriptionCreated
	EventSubscriptionUpdated      = types.EventSubscriptionUpdated
	EventSubscriptionDeleted      = types.EventSubscriptionDeleted
	EventSubscriptionTrialWillEnd = types.EventSubscriptionTrialWillEnd
	EventInvoiceCreated           = types.EventInvoiceCreated
	EventInvoicePaid              = types.EventInvoicePaid
	EventInvoicePaymentFailed     = types.EventInvoicePaymentFailed
	EventCheckoutSessionCompleted = types.EventCheckoutSessionCompleted
	EventPaymentIntentSucceeded   = types.EventPaymentIntentSucceeded
	EventPaymentIntentFailed      = types.EventPaymentIntentFailed
	EventPaymentMethodAttached    = types.EventPaymentMethodAttached
	EventPaymentMethodDetached    = types.EventPaymentMethodDetached
)
