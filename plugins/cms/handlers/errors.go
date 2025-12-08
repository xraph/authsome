package handlers

import (
	"context"
	"io"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/core/contexts"
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

// getContextWithHeaders extracts app and environment IDs from request headers
// and injects them into the context. This allows API clients to specify
// the app context via X-App-ID and X-Environment-ID headers.
func getContextWithHeaders(c forge.Context) context.Context {
	ctx := c.Request().Context()

	// Check if context already has app ID
	if _, ok := contexts.GetAppID(ctx); !ok {
		// Try to get from X-App-ID header
		if appIDStr := c.Request().Header.Get("X-App-ID"); appIDStr != "" {
			if appID, err := xid.FromString(appIDStr); err == nil {
				ctx = contexts.SetAppID(ctx, appID)
			}
		}
	}

	// Check if context already has environment ID
	if _, ok := contexts.GetEnvironmentID(ctx); !ok {
		// Try to get from X-Environment-ID header
		if envIDStr := c.Request().Header.Get("X-Environment-ID"); envIDStr != "" {
			if envID, err := xid.FromString(envIDStr); err == nil {
				ctx = contexts.SetEnvironmentID(ctx, envID)
			}
		}
	}

	return ctx
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
