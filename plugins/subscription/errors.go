package subscription

import (
	"github.com/xraph/authsome/plugins/subscription/errors"
)

// Re-export error types for convenience
type SubscriptionError = errors.SubscriptionError

// NewSubscriptionError creates a new subscription error with context
var NewSubscriptionError = errors.New

// Re-export domain errors
var (
	// Plan errors
	ErrPlanNotFound           = errors.ErrPlanNotFound
	ErrPlanAlreadyExists      = errors.ErrPlanAlreadyExists
	ErrPlanNotActive          = errors.ErrPlanNotActive
	ErrPlanHasSubscriptions   = errors.ErrPlanHasSubscriptions
	ErrInvalidPlanSlug        = errors.ErrInvalidPlanSlug
	ErrInvalidBillingPattern  = errors.ErrInvalidBillingPattern
	ErrInvalidBillingInterval = errors.ErrInvalidBillingInterval

	// Subscription errors
	ErrSubscriptionNotFound      = errors.ErrSubscriptionNotFound
	ErrSubscriptionAlreadyExists = errors.ErrSubscriptionAlreadyExists
	ErrSubscriptionNotActive     = errors.ErrSubscriptionNotActive
	ErrSubscriptionCanceled      = errors.ErrSubscriptionCanceled
	ErrSubscriptionPaused        = errors.ErrSubscriptionPaused
	ErrCannotDowngrade           = errors.ErrCannotDowngrade
	ErrInvalidQuantity           = errors.ErrInvalidQuantity

	// Add-on errors
	ErrAddOnNotFound        = errors.ErrAddOnNotFound
	ErrAddOnAlreadyExists   = errors.ErrAddOnAlreadyExists
	ErrAddOnNotActive       = errors.ErrAddOnNotActive
	ErrAddOnNotAvailable    = errors.ErrAddOnNotAvailable
	ErrAddOnAlreadyAttached = errors.ErrAddOnAlreadyAttached
	ErrAddOnNotAttached     = errors.ErrAddOnNotAttached
	ErrAddOnMaxQuantity     = errors.ErrAddOnMaxQuantity

	// Invoice errors
	ErrInvoiceNotFound    = errors.ErrInvoiceNotFound
	ErrInvoiceAlreadyPaid = errors.ErrInvoiceAlreadyPaid
	ErrInvoiceVoided      = errors.ErrInvoiceVoided
	ErrInvoiceNotOpen     = errors.ErrInvoiceNotOpen

	// Usage errors
	ErrUsageRecordNotFound  = errors.ErrUsageRecordNotFound
	ErrDuplicateUsageRecord = errors.ErrDuplicateUsageRecord
	ErrInvalidUsageMetric   = errors.ErrInvalidUsageMetric
	ErrInvalidUsageAction   = errors.ErrInvalidUsageAction

	// Payment method errors
	ErrPaymentMethodNotFound      = errors.ErrPaymentMethodNotFound
	ErrPaymentMethodRequired      = errors.ErrPaymentMethodRequired
	ErrPaymentMethodExpired       = errors.ErrPaymentMethodExpired
	ErrDefaultPaymentMethodDelete = errors.ErrDefaultPaymentMethodDelete

	// Customer errors
	ErrCustomerNotFound      = errors.ErrCustomerNotFound
	ErrCustomerAlreadyExists = errors.ErrCustomerAlreadyExists

	// Provider errors
	ErrProviderNotConfigured   = errors.ErrProviderNotConfigured
	ErrProviderAPIError        = errors.ErrProviderAPIError
	ErrWebhookSignatureInvalid = errors.ErrWebhookSignatureInvalid
	ErrWebhookEventUnhandled   = errors.ErrWebhookEventUnhandled

	// Feature/limit errors
	ErrFeatureLimitExceeded = errors.ErrFeatureLimitExceeded
	ErrSeatLimitExceeded    = errors.ErrSeatLimitExceeded
	ErrSubscriptionRequired = errors.ErrSubscriptionRequired
	ErrTrialExpired         = errors.ErrTrialExpired

	// General errors
	ErrInvalidCurrency = errors.ErrInvalidCurrency
	ErrInvalidAppID    = errors.ErrInvalidAppID
	ErrInvalidOrgID    = errors.ErrInvalidOrgID
	ErrUnauthorized    = errors.ErrUnauthorized
)

// Re-export error helper functions
var (
	IsNotFoundError   = errors.IsNotFoundError
	IsValidationError = errors.IsValidationError
	IsConflictError   = errors.IsConflictError
	IsLimitError      = errors.IsLimitError
	IsPaymentError    = errors.IsPaymentError
)
