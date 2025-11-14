package webhook

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// WEBHOOK-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeWebhookNotFound           = "WEBHOOK_NOT_FOUND"
	CodeWebhookAlreadyExists      = "WEBHOOK_ALREADY_EXISTS"
	CodeWebhookCreationFailed     = "WEBHOOK_CREATION_FAILED"
	CodeWebhookUpdateFailed       = "WEBHOOK_UPDATE_FAILED"
	CodeWebhookDeletionFailed     = "WEBHOOK_DELETION_FAILED"
	CodeInvalidEventType          = "INVALID_EVENT_TYPE"
	CodeInvalidURL                = "INVALID_URL"
	CodeInvalidRetryBackoff       = "INVALID_RETRY_BACKOFF"
	CodeDeliveryFailed            = "DELIVERY_FAILED"
	CodeEventCreationFailed       = "EVENT_CREATION_FAILED"
	CodeEventNotFound             = "EVENT_NOT_FOUND"
	CodeDeliveryNotFound          = "DELIVERY_NOT_FOUND"
	CodeMaxWebhooksReached        = "MAX_WEBHOOKS_REACHED"
	CodeMissingAppContext         = "MISSING_APP_CONTEXT"
	CodeMissingEnvironmentContext = "MISSING_ENVIRONMENT_CONTEXT"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

func WebhookNotFound() *errs.AuthsomeError {
	return errs.New(CodeWebhookNotFound, "Webhook not found", http.StatusNotFound)
}

func WebhookAlreadyExists() *errs.AuthsomeError {
	return errs.New(CodeWebhookAlreadyExists, "Webhook already exists for this URL and app", http.StatusConflict)
}

func WebhookCreationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeWebhookCreationFailed, "Failed to create webhook", http.StatusInternalServerError)
}

func WebhookUpdateFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeWebhookUpdateFailed, "Failed to update webhook", http.StatusInternalServerError)
}

func WebhookDeletionFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeWebhookDeletionFailed, "Failed to delete webhook", http.StatusInternalServerError)
}

func InvalidEventType(eventType string) *errs.AuthsomeError {
	return errs.New(CodeInvalidEventType, "Invalid event type", http.StatusBadRequest).
		WithContext("event_type", eventType)
}

func InvalidURL(url string) *errs.AuthsomeError {
	return errs.New(CodeInvalidURL, "Invalid webhook URL", http.StatusBadRequest).
		WithContext("url", url)
}

func InvalidRetryBackoff(backoff string) *errs.AuthsomeError {
	return errs.New(CodeInvalidRetryBackoff, "Invalid retry backoff type", http.StatusBadRequest).
		WithContext("backoff", backoff)
}

func DeliveryFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeDeliveryFailed, "Failed to deliver webhook", http.StatusInternalServerError)
}

func EventCreationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeEventCreationFailed, "Failed to create event", http.StatusInternalServerError)
}

func EventNotFound() *errs.AuthsomeError {
	return errs.New(CodeEventNotFound, "Event not found", http.StatusNotFound)
}

func DeliveryNotFound() *errs.AuthsomeError {
	return errs.New(CodeDeliveryNotFound, "Delivery not found", http.StatusNotFound)
}

func MaxWebhooksReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxWebhooksReached, "Maximum number of webhooks reached", http.StatusForbidden).
		WithContext("limit", limit)
}

func MissingAppContext() *errs.AuthsomeError {
	return errs.New(CodeMissingAppContext, "App context is required", http.StatusBadRequest)
}

func MissingEnvironmentContext() *errs.AuthsomeError {
	return errs.New(CodeMissingEnvironmentContext, "Environment context is required", http.StatusBadRequest)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrWebhookNotFound           = &errs.AuthsomeError{Code: CodeWebhookNotFound}
	ErrWebhookAlreadyExists      = &errs.AuthsomeError{Code: CodeWebhookAlreadyExists}
	ErrWebhookCreationFailed     = &errs.AuthsomeError{Code: CodeWebhookCreationFailed}
	ErrWebhookUpdateFailed       = &errs.AuthsomeError{Code: CodeWebhookUpdateFailed}
	ErrWebhookDeletionFailed     = &errs.AuthsomeError{Code: CodeWebhookDeletionFailed}
	ErrInvalidEventType          = &errs.AuthsomeError{Code: CodeInvalidEventType}
	ErrInvalidURL                = &errs.AuthsomeError{Code: CodeInvalidURL}
	ErrInvalidRetryBackoff       = &errs.AuthsomeError{Code: CodeInvalidRetryBackoff}
	ErrDeliveryFailed            = &errs.AuthsomeError{Code: CodeDeliveryFailed}
	ErrEventCreationFailed       = &errs.AuthsomeError{Code: CodeEventCreationFailed}
	ErrEventNotFound             = &errs.AuthsomeError{Code: CodeEventNotFound}
	ErrDeliveryNotFound          = &errs.AuthsomeError{Code: CodeDeliveryNotFound}
	ErrMaxWebhooksReached        = &errs.AuthsomeError{Code: CodeMaxWebhooksReached}
	ErrMissingAppContext         = &errs.AuthsomeError{Code: CodeMissingAppContext}
	ErrMissingEnvironmentContext = &errs.AuthsomeError{Code: CodeMissingEnvironmentContext}
)
