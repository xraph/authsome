package apikey

import (
	"fmt"
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// API KEY-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeAPIKeyNotFound           = "API_KEY_NOT_FOUND"
	CodeAPIKeyInactive           = "API_KEY_INACTIVE"
	CodeAPIKeyExpired            = "API_KEY_EXPIRED"
	CodeInsufficientScope        = "INSUFFICIENT_SCOPE"
	CodeInsufficientPermission   = "INSUFFICIENT_PERMISSION"
	CodeIPNotAllowed             = "IP_NOT_ALLOWED"
	CodeInvalidKeyFormat         = "INVALID_KEY_FORMAT"
	CodeMaxKeysReached           = "MAX_KEYS_REACHED"
	CodeAPIKeyAlreadyExists      = "API_KEY_ALREADY_EXISTS"
	CodeAPIKeyCreationFailed     = "API_KEY_CREATION_FAILED"
	CodeAPIKeyUpdateFailed       = "API_KEY_UPDATE_FAILED"
	CodeAPIKeyDeletionFailed     = "API_KEY_DELETION_FAILED"
	CodeAPIKeyRotationFailed     = "API_KEY_ROTATION_FAILED"
	CodeAPIKeyVerificationFailed = "API_KEY_VERIFICATION_FAILED"
	CodeInvalidAPIKeyHash        = "INVALID_API_KEY_HASH"
	CodeMissingAppContext        = "MISSING_APP_CONTEXT"
	CodeMissingEnvContext        = "MISSING_ENV_CONTEXT"
	CodeAccessDenied             = "ACCESS_DENIED"
	CodeInvalidRateLimit         = "INVALID_RATE_LIMIT"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// API Key lookup errors
func APIKeyNotFound() *errs.AuthsomeError {
	return errs.New(CodeAPIKeyNotFound, "API key not found", http.StatusNotFound)
}

func APIKeyInactive() *errs.AuthsomeError {
	return errs.New(CodeAPIKeyInactive, "API key is inactive", http.StatusForbidden)
}

func APIKeyExpired() *errs.AuthsomeError {
	return errs.New(CodeAPIKeyExpired, "API key has expired", http.StatusForbidden)
}

func APIKeyAlreadyExists(prefix string) *errs.AuthsomeError {
	return errs.New(CodeAPIKeyAlreadyExists, "API key already exists", http.StatusConflict).
		WithContext("prefix", prefix)
}

// Permission and scope errors
func InsufficientScope(required string) *errs.AuthsomeError {
	return errs.New(CodeInsufficientScope, "API key lacks required scope", http.StatusForbidden).
		WithContext("required_scope", required)
}

func InsufficientPermission(required string) *errs.AuthsomeError {
	return errs.New(CodeInsufficientPermission, "API key lacks required permission", http.StatusForbidden).
		WithContext("required_permission", required)
}

// Security errors
func IPNotAllowed(ip string) *errs.AuthsomeError {
	return errs.New(CodeIPNotAllowed, "IP address not allowed", http.StatusForbidden).
		WithContext("ip", ip)
}

func InvalidKeyFormat() *errs.AuthsomeError {
	return errs.New(CodeInvalidKeyFormat, "Invalid API key format", http.StatusBadRequest)
}

func InvalidAPIKeyHash() *errs.AuthsomeError {
	return errs.New(CodeInvalidAPIKeyHash, "Invalid API key", http.StatusUnauthorized)
}

// Limit errors
func MaxKeysReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxKeysReached, fmt.Sprintf("Maximum number of API keys reached (%d)", limit), http.StatusForbidden).
		WithContext("max_keys", limit)
}

func InvalidRateLimit(rateLimit, maxRateLimit int) *errs.AuthsomeError {
	return errs.New(CodeInvalidRateLimit, fmt.Sprintf("Rate limit exceeds maximum allowed (%d)", maxRateLimit), http.StatusBadRequest).
		WithContext("rate_limit", rateLimit).
		WithContext("max_rate_limit", maxRateLimit)
}

// CRUD operation errors
func APIKeyCreationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeAPIKeyCreationFailed, "Failed to create API key", http.StatusInternalServerError)
}

func APIKeyUpdateFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeAPIKeyUpdateFailed, "Failed to update API key", http.StatusInternalServerError)
}

func APIKeyDeletionFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeAPIKeyDeletionFailed, "Failed to delete API key", http.StatusInternalServerError)
}

func APIKeyRotationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeAPIKeyRotationFailed, "Failed to rotate API key", http.StatusInternalServerError)
}

func APIKeyVerificationFailed(reason string) *errs.AuthsomeError {
	return errs.New(CodeAPIKeyVerificationFailed, "API key verification failed", http.StatusUnauthorized).
		WithContext("reason", reason)
}

// Context errors
func MissingAppContext() *errs.AuthsomeError {
	return errs.New(CodeMissingAppContext, "App context is required", http.StatusBadRequest)
}

func MissingEnvContext() *errs.AuthsomeError {
	return errs.New(CodeMissingEnvContext, "Environment context is required", http.StatusBadRequest)
}

// Access control errors
func AccessDenied(reason string) *errs.AuthsomeError {
	return errs.New(CodeAccessDenied, "Access denied", http.StatusForbidden).
		WithContext("reason", reason)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrAPIKeyNotFound           = &errs.AuthsomeError{Code: CodeAPIKeyNotFound}
	ErrAPIKeyInactive           = &errs.AuthsomeError{Code: CodeAPIKeyInactive}
	ErrAPIKeyExpired            = &errs.AuthsomeError{Code: CodeAPIKeyExpired}
	ErrInsufficientScope        = &errs.AuthsomeError{Code: CodeInsufficientScope}
	ErrInsufficientPermission   = &errs.AuthsomeError{Code: CodeInsufficientPermission}
	ErrIPNotAllowed             = &errs.AuthsomeError{Code: CodeIPNotAllowed}
	ErrInvalidKeyFormat         = &errs.AuthsomeError{Code: CodeInvalidKeyFormat}
	ErrMaxKeysReached           = &errs.AuthsomeError{Code: CodeMaxKeysReached}
	ErrAPIKeyAlreadyExists      = &errs.AuthsomeError{Code: CodeAPIKeyAlreadyExists}
	ErrAPIKeyCreationFailed     = &errs.AuthsomeError{Code: CodeAPIKeyCreationFailed}
	ErrAPIKeyUpdateFailed       = &errs.AuthsomeError{Code: CodeAPIKeyUpdateFailed}
	ErrAPIKeyDeletionFailed     = &errs.AuthsomeError{Code: CodeAPIKeyDeletionFailed}
	ErrAPIKeyRotationFailed     = &errs.AuthsomeError{Code: CodeAPIKeyRotationFailed}
	ErrAPIKeyVerificationFailed = &errs.AuthsomeError{Code: CodeAPIKeyVerificationFailed}
	ErrInvalidAPIKeyHash        = &errs.AuthsomeError{Code: CodeInvalidAPIKeyHash}
	ErrMissingAppContext        = &errs.AuthsomeError{Code: CodeMissingAppContext}
	ErrMissingEnvContext        = &errs.AuthsomeError{Code: CodeMissingEnvContext}
	ErrAccessDenied             = &errs.AuthsomeError{Code: CodeAccessDenied}
	ErrInvalidRateLimit         = &errs.AuthsomeError{Code: CodeInvalidRateLimit}
)
