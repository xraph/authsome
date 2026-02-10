package impersonation

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// IMPERSONATION-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodePermissionDenied      = "IMPERSONATION_PERMISSION_DENIED"
	CodeUserNotFound          = "IMPERSONATION_USER_NOT_FOUND"
	CodeSessionNotFound       = "IMPERSONATION_SESSION_NOT_FOUND"
	CodeImpersonationNotFound = "IMPERSONATION_NOT_FOUND"
	CodeAlreadyImpersonating  = "ALREADY_IMPERSONATING"
	CodeCannotImpersonateSelf = "CANNOT_IMPERSONATE_SELF"
	CodeSessionExpired        = "IMPERSONATION_SESSION_EXPIRED"
	CodeInvalidReason         = "INVALID_IMPERSONATION_REASON"
	CodeInvalidDuration       = "INVALID_IMPERSONATION_DURATION"
	CodeRequireTicket         = "REQUIRE_TICKET_NUMBER"
	CodeInvalidPermission     = "INVALID_IMPERSONATION_PERMISSION"
	CodeSessionAlreadyEnded   = "IMPERSONATION_SESSION_ALREADY_ENDED"
	CodeFailedToCreateSession = "FAILED_TO_CREATE_SESSION"
	CodeFailedToRevokeSession = "FAILED_TO_REVOKE_SESSION"
	CodeTargetUserNotFound    = "TARGET_USER_NOT_FOUND"
	CodeImpersonatorNotFound  = "IMPERSONATOR_NOT_FOUND"
	CodeAuditEventNotFound    = "AUDIT_EVENT_NOT_FOUND"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// PermissionDenied returns an error when impersonation permission is denied.
func PermissionDenied(reason string) *errs.AuthsomeError {
	return errs.New(CodePermissionDenied, "Permission denied for impersonation", http.StatusForbidden).
		WithContext("reason", reason)
}

func InvalidPermission(permission string) *errs.AuthsomeError {
	return errs.New(CodeInvalidPermission, "Invalid impersonation permission", http.StatusForbidden).
		WithContext("permission", permission)
}

// UserNotFound returns an error when a user is not found.
func UserNotFound(userID string) *errs.AuthsomeError {
	return errs.New(CodeUserNotFound, "User not found", http.StatusNotFound).
		WithContext("user_id", userID)
}

func TargetUserNotFound(userID string) *errs.AuthsomeError {
	return errs.New(CodeTargetUserNotFound, "Target user not found", http.StatusNotFound).
		WithContext("target_user_id", userID)
}

func ImpersonatorNotFound(userID string) *errs.AuthsomeError {
	return errs.New(CodeImpersonatorNotFound, "Impersonator not found", http.StatusNotFound).
		WithContext("impersonator_id", userID)
}

// SessionNotFound returns an error when a session is not found.
func SessionNotFound(sessionID string) *errs.AuthsomeError {
	return errs.New(CodeSessionNotFound, "Session not found", http.StatusNotFound).
		WithContext("session_id", sessionID)
}

func ImpersonationNotFound(impersonationID string) *errs.AuthsomeError {
	return errs.New(CodeImpersonationNotFound, "Impersonation session not found", http.StatusNotFound).
		WithContext("impersonation_id", impersonationID)
}

func AlreadyImpersonating(impersonatorID string) *errs.AuthsomeError {
	return errs.New(CodeAlreadyImpersonating, "Already impersonating another user", http.StatusConflict).
		WithContext("impersonator_id", impersonatorID)
}

func CannotImpersonateSelf() *errs.AuthsomeError {
	return errs.New(CodeCannotImpersonateSelf, "Cannot impersonate yourself", http.StatusBadRequest)
}

func SessionExpired(sessionID string) *errs.AuthsomeError {
	return errs.New(CodeSessionExpired, "Impersonation session has expired", http.StatusForbidden).
		WithContext("session_id", sessionID)
}

func SessionAlreadyEnded(impersonationID string) *errs.AuthsomeError {
	return errs.New(CodeSessionAlreadyEnded, "Impersonation session already ended", http.StatusConflict).
		WithContext("impersonation_id", impersonationID)
}

// InvalidReason returns an error when the impersonation reason is invalid.
func InvalidReason(minLength int) *errs.AuthsomeError {
	return errs.New(CodeInvalidReason, "Impersonation reason is invalid", http.StatusBadRequest).
		WithContext("min_length", minLength)
}

func InvalidDuration(min, max int) *errs.AuthsomeError {
	return errs.New(CodeInvalidDuration, "Impersonation duration is invalid", http.StatusBadRequest).
		WithContext("min_minutes", min).
		WithContext("max_minutes", max)
}

func RequireTicket() *errs.AuthsomeError {
	return errs.New(CodeRequireTicket, "Ticket number is required for impersonation", http.StatusBadRequest)
}

// FailedToCreateSession returns an error when impersonation session creation fails.
func FailedToCreateSession(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeFailedToCreateSession, "Failed to create impersonation session", http.StatusInternalServerError)
}

func FailedToRevokeSession(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeFailedToRevokeSession, "Failed to revoke impersonation session", http.StatusInternalServerError)
}

// AuditEventNotFound returns an error when an audit event is not found.
func AuditEventNotFound(eventID string) *errs.AuthsomeError {
	return errs.New(CodeAuditEventNotFound, "Audit event not found", http.StatusNotFound).
		WithContext("event_id", eventID)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrPermissionDenied      = &errs.AuthsomeError{Code: CodePermissionDenied}
	ErrUserNotFound          = &errs.AuthsomeError{Code: CodeUserNotFound}
	ErrSessionNotFound       = &errs.AuthsomeError{Code: CodeSessionNotFound}
	ErrImpersonationNotFound = &errs.AuthsomeError{Code: CodeImpersonationNotFound}
	ErrAlreadyImpersonating  = &errs.AuthsomeError{Code: CodeAlreadyImpersonating}
	ErrCannotImpersonateSelf = &errs.AuthsomeError{Code: CodeCannotImpersonateSelf}
	ErrSessionExpired        = &errs.AuthsomeError{Code: CodeSessionExpired}
	ErrInvalidReason         = &errs.AuthsomeError{Code: CodeInvalidReason}
	ErrInvalidDuration       = &errs.AuthsomeError{Code: CodeInvalidDuration}
	ErrRequireTicket         = &errs.AuthsomeError{Code: CodeRequireTicket}
	ErrInvalidPermission     = &errs.AuthsomeError{Code: CodeInvalidPermission}
	ErrSessionAlreadyEnded   = &errs.AuthsomeError{Code: CodeSessionAlreadyEnded}
	ErrFailedToCreateSession = &errs.AuthsomeError{Code: CodeFailedToCreateSession}
	ErrFailedToRevokeSession = &errs.AuthsomeError{Code: CodeFailedToRevokeSession}
	ErrTargetUserNotFound    = &errs.AuthsomeError{Code: CodeTargetUserNotFound}
	ErrImpersonatorNotFound  = &errs.AuthsomeError{Code: CodeImpersonatorNotFound}
	ErrAuditEventNotFound    = &errs.AuthsomeError{Code: CodeAuditEventNotFound}
)
