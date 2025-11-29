package handlers

import (
	"io"
	"strconv"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/internal/errs"
)

// handleError handles errors and returns appropriate HTTP responses
func handleError(c forge.Context, err error) error {
	if err == nil {
		return nil
	}

	// Check if it's an AuthsomeError
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, map[string]interface{}{
			"error":   authErr.Message,
			"code":    authErr.Code,
			"details": authErr.Details,
		})
	}

	// Default to internal server error
	return c.JSON(500, map[string]string{
		"error": err.Error(),
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Code    string      `json:"code,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// parseIntDefault parses an int from a string with a default value
func parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// readBody reads the request body and returns it as bytes
func readBody(c forge.Context) ([]byte, error) {
	return io.ReadAll(c.Request().Body)
}

