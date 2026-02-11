package handlers

// Re-export shared response types from core/responses for backward compatibility
// This allows handlers to use responses without importing core/responses directly.
import "github.com/xraph/authsome/core/responses"

//nolint:errname // HTTP response DTO, not a Go error type
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
