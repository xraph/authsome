package session

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// SESSION-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeSessionNotFound         = "SESSION_NOT_FOUND"
	CodeSessionExpired          = "SESSION_EXPIRED"
	CodeSessionCreationFailed   = "SESSION_CREATION_FAILED"
	CodeSessionRevocationFailed = "SESSION_REVOCATION_FAILED"
	CodeInvalidToken            = "INVALID_TOKEN"
	CodeMaxSessionsReached      = "MAX_SESSIONS_REACHED"
	CodeMissingAppContext       = "MISSING_APP_CONTEXT"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

func SessionNotFound() *errs.AuthsomeError {
	return errs.New(CodeSessionNotFound, "Session not found", http.StatusNotFound)
}

func SessionExpired() *errs.AuthsomeError {
	return errs.New(CodeSessionExpired, "Session has expired", http.StatusUnauthorized)
}

func SessionCreationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeSessionCreationFailed, "Failed to create session", http.StatusInternalServerError)
}

func SessionRevocationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeSessionRevocationFailed, "Failed to revoke session", http.StatusInternalServerError)
}

func InvalidToken() *errs.AuthsomeError {
	return errs.New(CodeInvalidToken, "Invalid session token", http.StatusUnauthorized)
}

func MaxSessionsReached(limit int) *errs.AuthsomeError {
	return errs.New(CodeMaxSessionsReached, "Maximum number of sessions reached for user", http.StatusForbidden).
		WithContext("limit", limit)
}

func MissingAppContext() *errs.AuthsomeError {
	return errs.New(CodeMissingAppContext, "App context is required", http.StatusBadRequest)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrSessionNotFound         = &errs.AuthsomeError{Code: CodeSessionNotFound}
	ErrSessionExpired          = &errs.AuthsomeError{Code: CodeSessionExpired}
	ErrSessionCreationFailed   = &errs.AuthsomeError{Code: CodeSessionCreationFailed}
	ErrSessionRevocationFailed = &errs.AuthsomeError{Code: CodeSessionRevocationFailed}
	ErrInvalidToken            = &errs.AuthsomeError{Code: CodeInvalidToken}
	ErrMaxSessionsReached      = &errs.AuthsomeError{Code: CodeMaxSessionsReached}
	ErrMissingAppContext       = &errs.AuthsomeError{Code: CodeMissingAppContext}
)
