// Package errors defines domain errors for the subscription plugin.
package errors

import (
	"errors"
	"fmt"
)

// Domain errors for the subscription plugin
var (
	// Plan errors
	ErrPlanNotFound           = errors.New("plan not found")
	ErrPlanAlreadyExists      = errors.New("plan with this slug already exists")
	ErrPlanNotActive          = errors.New("plan is not active")
	ErrPlanHasSubscriptions   = errors.New("plan has active subscriptions and cannot be deleted")
	ErrInvalidPlanSlug        = errors.New("invalid plan slug")
	ErrInvalidBillingPattern  = errors.New("invalid billing pattern")
	ErrInvalidBillingInterval = errors.New("invalid billing interval")

	// Subscription errors
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("organization already has an active subscription")
	ErrSubscriptionNotActive     = errors.New("subscription is not active")
	ErrSubscriptionCanceled      = errors.New("subscription is already canceled")
	ErrSubscriptionPaused        = errors.New("subscription is paused")
	ErrCannotDowngrade           = errors.New("cannot downgrade subscription with current usage")
	ErrInvalidQuantity           = errors.New("invalid subscription quantity")

	// Add-on errors
	ErrAddOnNotFound        = errors.New("add-on not found")
	ErrAddOnAlreadyExists   = errors.New("add-on with this slug already exists")
	ErrAddOnNotActive       = errors.New("add-on is not active")
	ErrAddOnNotAvailable    = errors.New("add-on is not available for this plan")
	ErrAddOnAlreadyAttached = errors.New("add-on is already attached to subscription")
	ErrAddOnNotAttached     = errors.New("add-on is not attached to subscription")
	ErrAddOnMaxQuantity     = errors.New("add-on maximum quantity exceeded")

	// Invoice errors
	ErrInvoiceNotFound    = errors.New("invoice not found")
	ErrInvoiceAlreadyPaid = errors.New("invoice is already paid")
	ErrInvoiceVoided      = errors.New("invoice has been voided")
	ErrInvoiceNotOpen     = errors.New("invoice is not open for payment")

	// Usage errors
	ErrUsageRecordNotFound  = errors.New("usage record not found")
	ErrDuplicateUsageRecord = errors.New("duplicate usage record (idempotency key)")
	ErrInvalidUsageMetric   = errors.New("invalid usage metric key")
	ErrInvalidUsageAction   = errors.New("invalid usage action")

	// Payment method errors
	ErrPaymentMethodNotFound      = errors.New("payment method not found")
	ErrPaymentMethodRequired      = errors.New("payment method is required")
	ErrPaymentMethodExpired       = errors.New("payment method is expired")
	ErrDefaultPaymentMethodDelete = errors.New("cannot delete default payment method")

	// Customer errors
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerAlreadyExists = errors.New("customer already exists for organization")

	// Provider errors
	ErrProviderNotConfigured   = errors.New("payment provider is not configured")
	ErrProviderAPIError        = errors.New("payment provider API error")
	ErrWebhookSignatureInvalid = errors.New("invalid webhook signature")
	ErrWebhookEventUnhandled   = errors.New("webhook event not handled")

	// Feature/limit errors
	ErrFeatureLimitExceeded  = errors.New("feature limit exceeded")
	ErrSeatLimitExceeded     = errors.New("seat limit exceeded")
	ErrSubscriptionRequired  = errors.New("subscription is required")
	ErrTrialExpired          = errors.New("trial period has expired")
	ErrFeatureNotFound       = errors.New("feature not found")
	ErrFeatureAlreadyExists  = errors.New("feature with this key already exists")
	ErrFeatureInUse          = errors.New("feature is linked to plans and cannot be deleted")
	ErrFeatureAlreadyLinked  = errors.New("feature is already linked to this plan")
	ErrFeatureLinkNotFound   = errors.New("feature link not found")
	ErrFeatureNotAvailable   = errors.New("feature is not available for this subscription")
	ErrInvalidFeatureType    = errors.New("invalid feature type")
	ErrInvalidResetPeriod    = errors.New("invalid reset period")
	ErrInsufficientQuota     = errors.New("insufficient feature quota")
	ErrFeatureGrantNotFound  = errors.New("feature grant not found")
	ErrFeatureUsageNotFound  = errors.New("feature usage record not found")

	// General errors
	ErrInvalidCurrency = errors.New("invalid currency code")
	ErrInvalidAppID    = errors.New("invalid app ID")
	ErrInvalidOrgID    = errors.New("invalid organization ID")
	ErrUnauthorized    = errors.New("unauthorized access")
)

// SubscriptionError represents a domain-specific error with additional context
type SubscriptionError struct {
	Err     error
	Message string
	Code    string
	Details map[string]interface{}
}

// Error implements the error interface
func (e *SubscriptionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *SubscriptionError) Unwrap() error {
	return e.Err
}

// New creates a new subscription error with context
func New(err error, message string) *SubscriptionError {
	return &SubscriptionError{
		Err:     err,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithCode adds an error code
func (e *SubscriptionError) WithCode(code string) *SubscriptionError {
	e.Code = code
	return e
}

// WithDetails adds details to the error
func (e *SubscriptionError) WithDetails(key string, value interface{}) *SubscriptionError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// IsNotFoundError checks if the error is a "not found" type error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrPlanNotFound) ||
		errors.Is(err, ErrSubscriptionNotFound) ||
		errors.Is(err, ErrAddOnNotFound) ||
		errors.Is(err, ErrInvoiceNotFound) ||
		errors.Is(err, ErrUsageRecordNotFound) ||
		errors.Is(err, ErrPaymentMethodNotFound) ||
		errors.Is(err, ErrCustomerNotFound) ||
		errors.Is(err, ErrFeatureNotFound) ||
		errors.Is(err, ErrFeatureLinkNotFound) ||
		errors.Is(err, ErrFeatureGrantNotFound) ||
		errors.Is(err, ErrFeatureUsageNotFound)
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidPlanSlug) ||
		errors.Is(err, ErrInvalidBillingPattern) ||
		errors.Is(err, ErrInvalidBillingInterval) ||
		errors.Is(err, ErrInvalidQuantity) ||
		errors.Is(err, ErrInvalidUsageMetric) ||
		errors.Is(err, ErrInvalidUsageAction) ||
		errors.Is(err, ErrInvalidCurrency) ||
		errors.Is(err, ErrInvalidAppID) ||
		errors.Is(err, ErrInvalidOrgID) ||
		errors.Is(err, ErrInvalidFeatureType) ||
		errors.Is(err, ErrInvalidResetPeriod)
}

// IsConflictError checks if the error is a conflict/duplicate error
func IsConflictError(err error) bool {
	return errors.Is(err, ErrPlanAlreadyExists) ||
		errors.Is(err, ErrSubscriptionAlreadyExists) ||
		errors.Is(err, ErrAddOnAlreadyExists) ||
		errors.Is(err, ErrAddOnAlreadyAttached) ||
		errors.Is(err, ErrCustomerAlreadyExists) ||
		errors.Is(err, ErrDuplicateUsageRecord) ||
		errors.Is(err, ErrFeatureAlreadyExists) ||
		errors.Is(err, ErrFeatureAlreadyLinked)
}

// IsLimitError checks if the error is a limit-related error
func IsLimitError(err error) bool {
	return errors.Is(err, ErrFeatureLimitExceeded) ||
		errors.Is(err, ErrSeatLimitExceeded) ||
		errors.Is(err, ErrAddOnMaxQuantity) ||
		errors.Is(err, ErrInsufficientQuota)
}

// IsPaymentError checks if the error is payment-related
func IsPaymentError(err error) bool {
	return errors.Is(err, ErrPaymentMethodNotFound) ||
		errors.Is(err, ErrPaymentMethodRequired) ||
		errors.Is(err, ErrPaymentMethodExpired) ||
		errors.Is(err, ErrProviderNotConfigured) ||
		errors.Is(err, ErrProviderAPIError)
}

