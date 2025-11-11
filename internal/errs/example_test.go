package errs_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/internal/errs"
)

// Example_basic demonstrates basic error creation and usage.
func Example_basic() {
	err := errs.UserNotFound()
	fmt.Println(err.Error())
	fmt.Println(err.Code)
	fmt.Println(err.HTTPStatus)

	// Output:
	// USER_NOT_FOUND: User not found
	// USER_NOT_FOUND
	// 404
}

// Example_withContext demonstrates adding context to errors.
func Example_withContext() {
	err := errs.EmailAlreadyExists("user@example.com").
		WithContext("attempted_at", time.Now().Format(time.RFC3339)).
		WithTraceID("trace-123")

	fmt.Println(err.Code)
	fmt.Println(err.Context["email"])
	fmt.Println(err.TraceID)

	// Output:
	// EMAIL_ALREADY_EXISTS
	// user@example.com
	// trace-123
}

// Example_wrapping demonstrates error wrapping.
func Example_wrapping() {
	dbErr := errors.New("connection timeout")
	err := errs.DatabaseError("SELECT", dbErr)

	fmt.Println(err.Code)
	fmt.Println(errors.Is(err, dbErr)) // Can still find original error

	// Output:
	// DATABASE_ERROR
	// true
}

// Example_sentinelComparison demonstrates using sentinel errors.
func Example_sentinelComparison() {
	err := errs.UserNotFound()

	// Use errors.Is for comparison
	if errors.Is(err, errs.ErrUserNotFound) {
		fmt.Println("User not found!")
	}

	// Output:
	// User not found!
}

// Example_extraction demonstrates extracting AuthsomeError details.
func Example_extraction() {
	err := errs.PermissionDenied("delete", "user:123")

	// Extract the AuthsomeError
	var authErr *errs.AuthsomeError
	if errors.As(err, &authErr) {
		fmt.Printf("Code: %s\n", authErr.Code)
		fmt.Printf("Status: %d\n", authErr.HTTPStatus)
		fmt.Printf("Action: %s\n", authErr.Context["action"])
	}

	// Output:
	// Code: PERMISSION_DENIED
	// Status: 403
	// Action: delete
}

// Example_validationErrors demonstrates validation error handling.
func Example_validationErrors() {
	validationErrors := map[string]string{
		"email":    "invalid format",
		"password": "too short",
	}

	err := errs.ValidationFailed(validationErrors)

	fmt.Println(err.Code)
	fmt.Println(err.HTTPStatus)

	if details, ok := err.Details.(map[string]string); ok {
		fmt.Printf("Validation errors: %d\n", len(details))
	}

	// Output:
	// VALIDATION_FAILED
	// 400
	// Validation errors: 2
}

// Example_rateLimiting demonstrates rate limiting errors.
func Example_rateLimiting() {
	retryAfter := 60 * time.Second
	err := errs.RateLimitExceeded(retryAfter)

	fmt.Println(err.Code)
	fmt.Println(err.HTTPStatus)
	fmt.Printf("Retry after: %.0f seconds\n", err.Context["retry_after"])

	// Output:
	// RATE_LIMIT_EXCEEDED
	// 429
	// Retry after: 60 seconds
}

// Example_helpers demonstrates helper functions.
func Example_helpers() {
	err := errs.UserNotFound()

	// Extract HTTP status
	status := errs.GetHTTPStatus(err)
	fmt.Printf("HTTP Status: %d\n", status)

	// Extract error code
	code := errs.GetErrorCode(err)
	fmt.Printf("Error Code: %s\n", code)

	// Output:
	// HTTP Status: 404
	// Error Code: USER_NOT_FOUND
}

// Example_service demonstrates typical service layer usage.
func Example_service() {
	// Simulate a service method
	createUser := func(ctx context.Context, email string) error {
		// Check if user exists (simulated)
		userExists := true
		if userExists {
			return errs.EmailAlreadyExists(email).
				WithContext("source", "registration").
				WithTraceID("req-abc-123")
		}

		return nil
	}

	err := createUser(context.Background(), "existing@example.com")
	if err != nil {
		var authErr *errs.AuthsomeError
		if errors.As(err, &authErr) {
			fmt.Printf("Error: %s\n", authErr.Message)
			fmt.Printf("Email: %s\n", authErr.Context["email"])
			fmt.Printf("Source: %s\n", authErr.Context["source"])
		}
	}

	// Output:
	// Error: Email address already registered
	// Email: existing@example.com
	// Source: registration
}

// Example_errorChaining demonstrates error chain navigation.
func Example_errorChaining() {
	// Create a chain of errors
	originalErr := errors.New("network timeout")
	dbErr := errs.DatabaseError("query", originalErr)
	serviceErr := fmt.Errorf("failed to fetch user: %w", dbErr)

	// Can still find original error in the chain
	fmt.Println(errors.Is(serviceErr, originalErr))

	// Extract the AuthsomeError
	var authErr *errs.AuthsomeError
	if errors.As(serviceErr, &authErr) {
		fmt.Printf("Code: %s\n", authErr.Code)
		fmt.Printf("Has underlying error: %v\n", authErr.Err != nil)
	}

	// Output:
	// true
	// Code: DATABASE_ERROR
	// Has underlying error: true
}

// Example_multipleContexts demonstrates building rich error context.
func Example_multipleContexts() {
	err := errs.PermissionDenied("update", "organization:123").
		WithContext("user_id", "usr_456").
		WithContext("user_role", "viewer").
		WithContext("required_role", "admin").
		WithContext("ip_address", "192.168.1.100").
		WithTraceID("trace-789")

	fmt.Printf("Action denied: %s on %s\n",
		err.Context["action"],
		err.Context["resource"])
	fmt.Printf("User %s (role: %s) needs role: %s\n",
		err.Context["user_id"],
		err.Context["user_role"],
		err.Context["required_role"])

	// Output:
	// Action denied: update on organization:123
	// User usr_456 (role: viewer) needs role: admin
}

// Example_oauth demonstrates OAuth error handling.
func Example_oauth() {
	err := errs.OAuthFailed("google", "invalid state parameter").
		WithContext("client_id", "client_123").
		WithContext("redirect_uri", "https://app.example.com/callback")

	fmt.Println(err.Code)
	fmt.Println(err.Message)
	fmt.Printf("Provider: %s\n", err.Context["provider"])
	fmt.Printf("Reason: %s\n", err.Context["reason"])

	// Output:
	// OAUTH_FAILED
	// OAuth authentication failed
	// Provider: google
	// Reason: invalid state parameter
}

// Example_customError demonstrates creating custom errors.
func Example_customError() {
	err := errs.New(
		"CUSTOM_BUSINESS_RULE",
		"Cannot perform action during maintenance window",
		503,
	).WithContext("maintenance_until", "2024-01-15T18:00:00Z").
		WithContext("feature", "user_registration")

	fmt.Println(err.Code)
	fmt.Println(err.Message)
	fmt.Println(err.HTTPStatus)

	// Output:
	// CUSTOM_BUSINESS_RULE
	// Cannot perform action during maintenance window
	// 503
}
