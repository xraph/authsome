package audit

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// Error codes for audit operations.
const (
	CodeAuditEventNotFound     = "AUDIT_EVENT_NOT_FOUND"
	CodeAuditEventCreateFailed = "AUDIT_EVENT_CREATE_FAILED"
	CodeInvalidFilter          = "AUDIT_INVALID_FILTER"
	CodeInvalidTimeRange       = "AUDIT_INVALID_TIME_RANGE"
	CodeInvalidPagination      = "AUDIT_INVALID_PAGINATION"
	CodeQueryFailed            = "AUDIT_QUERY_FAILED"
)

// Error constructors

// AuditEventNotFound returns an error when an audit event is not found.
func AuditEventNotFound(id string) *errs.AuthsomeError {
	return errs.New(CodeAuditEventNotFound, "Audit event not found", http.StatusNotFound).
		WithContext("audit_event_id", id)
}

// AuditEventCreateFailed returns an error when creating an audit event fails.
func AuditEventCreateFailed(err error) *errs.AuthsomeError {
	return errs.New(CodeAuditEventCreateFailed, "Failed to create audit event", http.StatusInternalServerError).
		WithError(err)
}

// InvalidFilter returns an error when filter parameters are invalid.
func InvalidFilter(field, reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidFilter, "Invalid filter parameter", http.StatusBadRequest).
		WithContext("field", field).
		WithContext("reason", reason)
}

// InvalidTimeRange returns an error when the time range is invalid.
func InvalidTimeRange(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidTimeRange, "Invalid time range", http.StatusBadRequest).
		WithContext("reason", reason)
}

// InvalidPagination returns an error when pagination parameters are invalid.
func InvalidPagination(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidPagination, "Invalid pagination parameters", http.StatusBadRequest).
		WithContext("reason", reason)
}

// QueryFailed returns an error when a query operation fails.
func QueryFailed(operation string, err error) *errs.AuthsomeError {
	return errs.New(CodeQueryFailed, "Query operation failed", http.StatusInternalServerError).
		WithContext("operation", operation).
		WithError(err)
}
