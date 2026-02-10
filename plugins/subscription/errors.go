package subscription

import (
	"github.com/xraph/authsome/plugins/subscription/errors"
)

// SubscriptionError is the subscription error type.
type SubscriptionError = errors.SubscriptionError

// NewSubscriptionError creates a new subscription error with context.
var NewSubscriptionError = errors.New

// Re-export domain errors.
var (
	// ErrPlanNotFound is returned when a plan is not found.
	ErrPlanNotFound           = errors.ErrPlanNotFound
	ErrPlanAlreadyExists      = errors.ErrPlanAlreadyExists
	ErrPlanNotActive          = errors.ErrPlanNotActive
	ErrPlanHasSubscriptions   = errors.ErrPlanHasSubscriptions
	ErrInvalidPlanSlug        = errors.ErrInvalidPlanSlug
	ErrInvalidBillingPattern  = errors.ErrInvalidBillingPattern
	ErrInvalidBillingInterval = errors.ErrInvalidBillingInterval

	// ErrSubscriptionNotFound is returned when a subscription is not found.
	ErrSubscriptionNotFound      = errors.ErrSubscriptionNotFound
	ErrSubscriptionAlreadyExists = errors.ErrSubscriptionAlreadyExists
	ErrSubscriptionNotActive     = errors.ErrSubscriptionNotActive
	ErrSubscriptionCanceled      = errors.ErrSubscriptionCanceled
	ErrSubscriptionPaused        = errors.ErrSubscriptionPaused
	ErrCannotDowngrade           = errors.ErrCannotDowngrade
	ErrInvalidQuantity           = errors.ErrInvalidQuantity

	// ErrAddOnNotFound is returned when an add-on is not found.
	ErrAddOnNotFound        = errors.ErrAddOnNotFound
	ErrAddOnAlreadyExists   = errors.ErrAddOnAlreadyExists
	ErrAddOnNotActive       = errors.ErrAddOnNotActive
	ErrAddOnNotAvailable    = errors.ErrAddOnNotAvailable
	ErrAddOnAlreadyAttached = errors.ErrAddOnAlreadyAttached
	ErrAddOnNotAttached     = errors.ErrAddOnNotAttached
	ErrAddOnMaxQuantity     = errors.ErrAddOnMaxQuantity

	// ErrInvoiceNotFound is returned when an invoice is not found.
	ErrInvoiceNotFound    = errors.ErrInvoiceNotFound
	ErrInvoiceAlreadyPaid = errors.ErrInvoiceAlreadyPaid
	ErrInvoiceVoided      = errors.ErrInvoiceVoided
	ErrInvoiceNotOpen     = errors.ErrInvoiceNotOpen

	// ErrUsageRecordNotFound is returned when a usage record is not found.
	ErrUsageRecordNotFound  = errors.ErrUsageRecordNotFound
	ErrDuplicateUsageRecord = errors.ErrDuplicateUsageRecord
	ErrInvalidUsageMetric   = errors.ErrInvalidUsageMetric
	ErrInvalidUsageAction   = errors.ErrInvalidUsageAction

	// ErrPaymentMethodNotFound is returned when a payment method is not found.
	ErrPaymentMethodNotFound      = errors.ErrPaymentMethodNotFound
	ErrPaymentMethodRequired      = errors.ErrPaymentMethodRequired
	ErrPaymentMethodExpired       = errors.ErrPaymentMethodExpired
	ErrDefaultPaymentMethodDelete = errors.ErrDefaultPaymentMethodDelete

	// ErrCustomerNotFound is returned when a customer is not found.
	ErrCustomerNotFound      = errors.ErrCustomerNotFound
	ErrCustomerAlreadyExists = errors.ErrCustomerAlreadyExists

	// ErrProviderNotConfigured errors.
	ErrProviderNotConfigured   = errors.ErrProviderNotConfigured
	ErrProviderAPIError        = errors.ErrProviderAPIError
	ErrWebhookSignatureInvalid = errors.ErrWebhookSignatureInvalid
	ErrWebhookEventUnhandled   = errors.ErrWebhookEventUnhandled

	// ErrFeatureLimitExceeded is returned when a feature limit is exceeded.
	ErrFeatureLimitExceeded = errors.ErrFeatureLimitExceeded
	ErrSeatLimitExceeded    = errors.ErrSeatLimitExceeded
	ErrSubscriptionRequired = errors.ErrSubscriptionRequired
	ErrTrialExpired         = errors.ErrTrialExpired

	// ErrInvalidCurrency is returned when currency is invalid.
	ErrInvalidCurrency = errors.ErrInvalidCurrency
	ErrInvalidAppID    = errors.ErrInvalidAppID
	ErrInvalidOrgID    = errors.ErrInvalidOrgID
	ErrUnauthorized    = errors.ErrUnauthorized
)

// Re-export error helper functions.
var (
	IsNotFoundError   = errors.IsNotFoundError
	IsValidationError = errors.IsValidationError
	IsConflictError   = errors.IsConflictError
	IsLimitError      = errors.IsLimitError
	IsPaymentError    = errors.IsPaymentError
)
