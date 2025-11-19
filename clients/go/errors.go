package authsome

import "fmt"

// Auto-generated error types

// Error represents an API error
type Error struct {
	Message    string
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s (status: %d)", e.Code, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s (status: %d)", e.Message, e.StatusCode)
}

// Specific error types
var (
	ErrUnauthorized = &Error{Message: "Unauthorized", StatusCode: 401, Code: "UNAUTHORIZED"}
	ErrForbidden    = &Error{Message: "Forbidden", StatusCode: 403, Code: "FORBIDDEN"}
	ErrNotFound     = &Error{Message: "Not found", StatusCode: 404, Code: "NOT_FOUND"}
	ErrConflict     = &Error{Message: "Conflict", StatusCode: 409, Code: "CONFLICT"}
	ErrRateLimit    = &Error{Message: "Rate limit exceeded", StatusCode: 429, Code: "RATE_LIMIT"}
	ErrServer       = &Error{Message: "Internal server error", StatusCode: 500, Code: "SERVER_ERROR"}
)

// NewError creates an error from a status code and message
func NewError(statusCode int, message string) *Error {
	code := ""
	switch statusCode {
	case 400:
		code = "VALIDATION_ERROR"
	case 401:
		code = "UNAUTHORIZED"
	case 403:
		code = "FORBIDDEN"
	case 404:
		code = "NOT_FOUND"
	case 409:
		code = "CONFLICT"
	case 429:
		code = "RATE_LIMIT"
	case 500:
		code = "SERVER_ERROR"
	}
	
	return &Error{
		Message:    message,
		StatusCode: statusCode,
		Code:       code,
	}
}
