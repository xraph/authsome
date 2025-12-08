package notification

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// NOTIFICATION-SPECIFIC ERROR CODES
// =============================================================================

const (
	// Template errors
	CodeTemplateNotFound      = "TEMPLATE_NOT_FOUND"
	CodeTemplateAlreadyExists = "TEMPLATE_ALREADY_EXISTS"
	CodeInvalidTemplate       = "INVALID_TEMPLATE"
	CodeTemplateInactive      = "TEMPLATE_INACTIVE"
	CodeTemplateRenderFailed  = "TEMPLATE_RENDER_FAILED"

	// Notification errors
	CodeNotificationNotFound   = "NOTIFICATION_NOT_FOUND"
	CodeNotificationFailed     = "NOTIFICATION_FAILED"
	CodeNotificationSendFailed = "NOTIFICATION_SEND_FAILED"

	// Provider errors
	CodeProviderNotConfigured    = "PROVIDER_NOT_CONFIGURED"
	CodeProviderNotFound         = "PROVIDER_NOT_FOUND"
	CodeProviderValidationFailed = "PROVIDER_VALIDATION_FAILED"

	// Version errors
	CodeVersionNotFound      = "VERSION_NOT_FOUND"
	CodeVersionRestoreFailed = "VERSION_RESTORE_FAILED"

	// Test errors
	CodeTestNotFound = "TEST_NOT_FOUND"
	CodeTestFailed   = "TEST_FAILED"

	// Validation errors
	CodeInvalidNotificationType = "INVALID_NOTIFICATION_TYPE"
	CodeInvalidRecipient        = "INVALID_RECIPIENT"
	CodeMissingTemplateVariable = "MISSING_TEMPLATE_VARIABLE"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// Template errors
func TemplateNotFound() *errs.AuthsomeError {
	return errs.New(CodeTemplateNotFound, "Notification template not found", http.StatusNotFound)
}

func TemplateAlreadyExists(key string) *errs.AuthsomeError {
	return errs.New(CodeTemplateAlreadyExists, "Notification template already exists", http.StatusConflict).
		WithContext("template_key", key)
}

func InvalidTemplate(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidTemplate, "Invalid notification template", http.StatusBadRequest).
		WithContext("reason", reason)
}

func TemplateInactive(key string) *errs.AuthsomeError {
	return errs.New(CodeTemplateInactive, "Notification template is inactive", http.StatusForbidden).
		WithContext("template_key", key)
}

func TemplateRenderFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeTemplateRenderFailed, "Failed to render notification template", http.StatusInternalServerError)
}

// Notification errors
func NotificationNotFound() *errs.AuthsomeError {
	return errs.New(CodeNotificationNotFound, "Notification not found", http.StatusNotFound)
}

func NotificationFailed(reason string) *errs.AuthsomeError {
	return errs.New(CodeNotificationFailed, "Notification operation failed", http.StatusInternalServerError).
		WithContext("reason", reason)
}

func NotificationSendFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeNotificationSendFailed, "Failed to send notification", http.StatusInternalServerError)
}

// Provider errors
func ProviderNotConfigured(notificationType NotificationType) *errs.AuthsomeError {
	return errs.New(CodeProviderNotConfigured, "No provider configured for notification type", http.StatusServiceUnavailable).
		WithContext("notification_type", notificationType)
}

func ProviderNotFound(providerID string) *errs.AuthsomeError {
	return errs.New(CodeProviderNotFound, "Notification provider not found", http.StatusNotFound).
		WithContext("provider_id", providerID)
}

func ProviderValidationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeProviderValidationFailed, "Provider configuration validation failed", http.StatusBadRequest)
}

// Version errors
func VersionNotFound() *errs.AuthsomeError {
	return errs.New(CodeVersionNotFound, "Template version not found", http.StatusNotFound)
}

func VersionRestoreFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeVersionRestoreFailed, "Failed to restore template version", http.StatusInternalServerError)
}

// Test errors
func TestNotFound() *errs.AuthsomeError {
	return errs.New(CodeTestNotFound, "Notification test not found", http.StatusNotFound)
}

func TestFailed(reason string) *errs.AuthsomeError {
	return errs.New(CodeTestFailed, "Notification test failed", http.StatusInternalServerError).
		WithContext("reason", reason)
}

// Validation errors
func InvalidNotificationType(notifType string) *errs.AuthsomeError {
	return errs.New(CodeInvalidNotificationType, "Invalid notification type", http.StatusBadRequest).
		WithContext("type", notifType)
}

func InvalidRecipient(recipient string) *errs.AuthsomeError {
	return errs.New(CodeInvalidRecipient, "Invalid recipient address", http.StatusBadRequest).
		WithContext("recipient", recipient)
}

func MissingTemplateVariable(variable string) *errs.AuthsomeError {
	return errs.New(CodeMissingTemplateVariable, "Missing required template variable", http.StatusBadRequest).
		WithContext("variable", variable)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrTemplateNotFound         = &errs.AuthsomeError{Code: CodeTemplateNotFound}
	ErrTemplateAlreadyExists    = &errs.AuthsomeError{Code: CodeTemplateAlreadyExists}
	ErrInvalidTemplate          = &errs.AuthsomeError{Code: CodeInvalidTemplate}
	ErrTemplateInactive         = &errs.AuthsomeError{Code: CodeTemplateInactive}
	ErrTemplateRenderFailed     = &errs.AuthsomeError{Code: CodeTemplateRenderFailed}
	ErrNotificationNotFound     = &errs.AuthsomeError{Code: CodeNotificationNotFound}
	ErrNotificationFailed       = &errs.AuthsomeError{Code: CodeNotificationFailed}
	ErrNotificationSendFailed   = &errs.AuthsomeError{Code: CodeNotificationSendFailed}
	ErrProviderNotConfigured    = &errs.AuthsomeError{Code: CodeProviderNotConfigured}
	ErrProviderNotFound         = &errs.AuthsomeError{Code: CodeProviderNotFound}
	ErrProviderValidationFailed = &errs.AuthsomeError{Code: CodeProviderValidationFailed}
	ErrVersionNotFound          = &errs.AuthsomeError{Code: CodeVersionNotFound}
	ErrVersionRestoreFailed     = &errs.AuthsomeError{Code: CodeVersionRestoreFailed}
	ErrTestNotFound             = &errs.AuthsomeError{Code: CodeTestNotFound}
	ErrTestFailed               = &errs.AuthsomeError{Code: CodeTestFailed}
	ErrInvalidNotificationType  = &errs.AuthsomeError{Code: CodeInvalidNotificationType}
	ErrInvalidRecipient         = &errs.AuthsomeError{Code: CodeInvalidRecipient}
	ErrMissingTemplateVariable  = &errs.AuthsomeError{Code: CodeMissingTemplateVariable}
)
