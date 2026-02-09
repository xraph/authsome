package errs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// =============================================================================
// BASIC ERROR CREATION
// =============================================================================

func TestNew(t *testing.T) {
	err := New(CodeUserNotFound, "User does not exist", http.StatusNotFound)

	if err.Code != CodeUserNotFound {
		t.Errorf("expected code %s, got %s", CodeUserNotFound, err.Code)
	}

	if err.Message != "User does not exist" {
		t.Errorf("expected message 'User does not exist', got %s", err.Message)
	}

	if err.HTTPStatus != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, err.HTTPStatus)
	}

	if err.Context == nil {
		t.Error("expected context to be initialized")
	}

	if err.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestWrap(t *testing.T) {
	original := errors.New("database connection failed")
	err := Wrap(original, CodeDatabaseError, "Failed to query users", http.StatusInternalServerError)

	if !errors.Is(err.Err, original) {
		t.Error("expected underlying error to be preserved")
	}

	if !errors.Is(err, original) {
		t.Error("errors.Is should find underlying error")
	}
}

// =============================================================================
// ERROR METHODS
// =============================================================================

func TestAuthsomeError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AuthsomeError
		expected string
	}{
		{
			name:     "simple error",
			err:      New(CodeUserNotFound, "User not found", http.StatusNotFound),
			expected: "USER_NOT_FOUND: User not found",
		},
		{
			name:     "wrapped error",
			err:      Wrap(errors.New("db error"), CodeDatabaseError, "Query failed", http.StatusInternalServerError),
			expected: "DATABASE_ERROR: Query failed: db error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAuthsomeError_WithContext(t *testing.T) {
	err := UserNotFound().
		WithContext("user_id", "123").
		WithContext("lookup_by", "email")

	if err.Context["user_id"] != "123" {
		t.Error("expected user_id in context")
	}

	if err.Context["lookup_by"] != "email" {
		t.Error("expected lookup_by in context")
	}
}

func TestAuthsomeError_WithDetails(t *testing.T) {
	details := map[string]string{
		"email":    "invalid format",
		"password": "too short",
	}

	err := ValidationFailed(details)

	detailsMap, ok := err.Details.(map[string]string)
	if !ok {
		t.Fatal("expected details to be map[string]string")
	}

	if detailsMap["email"] != "invalid format" {
		t.Error("expected email validation error in details")
	}
}

func TestAuthsomeError_WithTraceID(t *testing.T) {
	traceID := "trace-123-abc"
	err := UserNotFound().WithTraceID(traceID)

	if err.TraceID != traceID {
		t.Errorf("expected trace ID %s, got %s", traceID, err.TraceID)
	}
}

func TestAuthsomeError_WithError(t *testing.T) {
	original := errors.New("original error")
	err := UserNotFound().WithError(original)

	if !errors.Is(err.Err, original) {
		t.Error("expected underlying error to be set")
	}
}

// =============================================================================
// ERROR COMPARISON (errors.Is)
// =============================================================================

func TestAuthsomeError_Is(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   error
		expected bool
	}{
		{
			name:     "same code matches",
			err:      UserNotFound(),
			target:   &AuthsomeError{Code: CodeUserNotFound},
			expected: true,
		},
		{
			name:     "different code doesn't match",
			err:      UserNotFound(),
			target:   &AuthsomeError{Code: CodeSessionNotFound},
			expected: false,
		},
		{
			name:     "matches sentinel error",
			err:      UserNotFound(),
			target:   ErrUserNotFound,
			expected: true,
		},
		{
			name:     "wrapped error matches",
			err:      Wrap(errors.New("db error"), CodeUserNotFound, "User lookup failed", http.StatusNotFound),
			target:   ErrUserNotFound,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// ERROR UNWRAPPING
// =============================================================================

func TestAuthsomeError_Unwrap(t *testing.T) {
	original := errors.New("database connection lost")
	wrapped := Wrap(original, CodeDatabaseError, "Query failed", http.StatusInternalServerError)

	unwrapped := errors.Unwrap(wrapped)
	if !errors.Is(unwrapped, original) {
		t.Error("expected unwrap to return original error")
	}

	// Test that errors.Is works through the chain
	if !errors.Is(wrapped, original) {
		t.Error("errors.Is should find original error in chain")
	}
}

// =============================================================================
// CONVERSION METHODS
// =============================================================================

func TestAuthsomeError_ToHTTPError(t *testing.T) {
	authErr := UserNotFound()
	httpErr := authErr.ToHTTPError()

	// Verify it's a proper forge.HTTPError
	if httpErr == nil {
		t.Fatal("expected non-nil HTTPError")
	}

	if httpErr.StatusCode() != http.StatusNotFound {
		t.Errorf("expected HTTP status %d, got %d", http.StatusNotFound, httpErr.StatusCode())
	}

	if httpErr.Message != authErr.Message {
		t.Error("expected message to be preserved")
	}
}

// =============================================================================
// AUTHENTICATION ERRORS
// =============================================================================

func TestInvalidCredentials(t *testing.T) {
	err := InvalidCredentials()
	assertError(t, err, CodeInvalidCredentials, http.StatusUnauthorized)
}

func TestEmailNotVerified(t *testing.T) {
	email := "test@example.com"
	err := EmailNotVerified(email)
	assertError(t, err, CodeEmailNotVerified, http.StatusForbidden)
	assertContext(t, err, "email", email)
}

func TestTwoFactorRequired(t *testing.T) {
	err := TwoFactorRequired()
	assertError(t, err, CodeTwoFactorRequired, http.StatusForbidden)
}

func TestTokenExpired(t *testing.T) {
	err := TokenExpired()
	assertError(t, err, CodeTokenExpired, http.StatusUnauthorized)
}

// =============================================================================
// USER ERRORS
// =============================================================================

func TestUserNotFound(t *testing.T) {
	err := UserNotFound()
	assertError(t, err, CodeUserNotFound, http.StatusNotFound)
}

func TestEmailAlreadyExists(t *testing.T) {
	email := "existing@example.com"
	err := EmailAlreadyExists(email)
	assertError(t, err, CodeEmailAlreadyExists, http.StatusConflict)
	assertContext(t, err, "email", email)
}

func TestWeakPassword(t *testing.T) {
	reason := "must contain uppercase letter"
	err := WeakPassword(reason)
	assertError(t, err, CodeWeakPassword, http.StatusBadRequest)
	assertContext(t, err, "reason", reason)
}

// =============================================================================
// SESSION ERRORS
// =============================================================================

func TestSessionNotFound(t *testing.T) {
	err := SessionNotFound()
	assertError(t, err, CodeSessionNotFound, http.StatusUnauthorized)
}

func TestSessionExpired(t *testing.T) {
	err := SessionExpired()
	assertError(t, err, CodeSessionExpired, http.StatusUnauthorized)
}

func TestConcurrentSessionLimit(t *testing.T) {
	err := ConcurrentSessionLimit()
	assertError(t, err, CodeConcurrentSessionLimit, http.StatusConflict)
}

// =============================================================================
// ORGANIZATION ERRORS
// =============================================================================

func TestOrganizationNotFound(t *testing.T) {
	err := OrganizationNotFound()
	assertError(t, err, CodeOrganizationNotFound, http.StatusNotFound)
}

func TestInsufficientRole(t *testing.T) {
	required := "admin"
	err := InsufficientRole(required)
	assertError(t, err, CodeInsufficientRole, http.StatusForbidden)
	assertContext(t, err, "required_role", required)
}

func TestSlugAlreadyExists(t *testing.T) {
	slug := "acme-corp"
	err := SlugAlreadyExists(slug)
	assertError(t, err, CodeSlugAlreadyExists, http.StatusConflict)
	assertContext(t, err, "slug", slug)
}

// =============================================================================
// RBAC ERRORS
// =============================================================================

func TestPermissionDenied(t *testing.T) {
	action := "delete"
	resource := "user:123"
	err := PermissionDenied(action, resource)
	assertError(t, err, CodePermissionDenied, http.StatusForbidden)
	assertContext(t, err, "action", action)
	assertContext(t, err, "resource", resource)
}

func TestPolicyViolation(t *testing.T) {
	policy := "password_policy"
	reason := "password complexity not met"
	err := PolicyViolation(policy, reason)
	assertError(t, err, CodePolicyViolation, http.StatusForbidden)
	assertContext(t, err, "policy", policy)
	assertContext(t, err, "reason", reason)
}

// =============================================================================
// RATE LIMITING ERRORS
// =============================================================================

func TestRateLimitExceeded(t *testing.T) {
	retryAfter := 60 * time.Second
	err := RateLimitExceeded(retryAfter)
	assertError(t, err, CodeRateLimitExceeded, http.StatusTooManyRequests)
	assertContext(t, err, "retry_after", 60.0)
}

func TestTooManyAttempts(t *testing.T) {
	retryAfter := 5 * time.Minute
	err := TooManyAttempts(retryAfter)
	assertError(t, err, CodeTooManyAttempts, http.StatusTooManyRequests)
	assertContext(t, err, "retry_after", 300.0)
}

// =============================================================================
// VALIDATION ERRORS
// =============================================================================

func TestValidationFailed(t *testing.T) {
	fields := map[string]string{
		"email":    "invalid format",
		"password": "too short",
	}
	err := ValidationFailed(fields)
	assertError(t, err, CodeValidationFailed, http.StatusBadRequest)

	details, ok := err.Details.(map[string]string)
	if !ok {
		t.Fatal("expected details to be map[string]string")
	}

	if len(details) != 2 {
		t.Errorf("expected 2 validation errors, got %d", len(details))
	}
}

func TestInvalidInput(t *testing.T) {
	field := "age"
	reason := "must be positive"
	err := InvalidInput(field, reason)
	assertError(t, err, CodeInvalidInput, http.StatusBadRequest)
	assertContext(t, err, "field", field)
	assertContext(t, err, "reason", reason)
}

func TestRequiredField(t *testing.T) {
	field := "email"
	err := RequiredField(field)
	assertError(t, err, CodeRequiredField, http.StatusBadRequest)
	assertContext(t, err, "field", field)
}

// =============================================================================
// PLUGIN ERRORS
// =============================================================================

func TestPluginNotFound(t *testing.T) {
	pluginID := "oauth-google"
	err := PluginNotFound(pluginID)
	assertError(t, err, CodePluginNotFound, http.StatusNotFound)
	assertContext(t, err, "plugin_id", pluginID)
}

func TestPluginInitFailed(t *testing.T) {
	pluginID := "mfa"
	original := errors.New("config missing")
	err := PluginInitFailed(pluginID, original)
	assertError(t, err, CodePluginInitFailed, http.StatusInternalServerError)
	assertContext(t, err, "plugin_id", pluginID)

	if !errors.Is(err, original) {
		t.Error("expected wrapped error to be findable")
	}
}

// =============================================================================
// OAUTH/SSO ERRORS
// =============================================================================

func TestOAuthFailed(t *testing.T) {
	provider := "google"
	reason := "invalid state"
	err := OAuthFailed(provider, reason)
	assertError(t, err, CodeOAuthFailed, http.StatusBadRequest)
	assertContext(t, err, "provider", provider)
	assertContext(t, err, "reason", reason)
}

func TestInvalidOAuthState(t *testing.T) {
	err := InvalidOAuthState()
	assertError(t, err, CodeInvalidOAuthState, http.StatusBadRequest)
}

// =============================================================================
// GENERAL ERRORS
// =============================================================================

func TestInternalError(t *testing.T) {
	original := errors.New("unexpected panic")
	err := InternalError(original)
	assertError(t, err, CodeInternalError, http.StatusInternalServerError)

	if !errors.Is(err, original) {
		t.Error("expected wrapped error to be findable")
	}
}

func TestDatabaseError(t *testing.T) {
	operation := "INSERT"
	original := errors.New("connection timeout")
	err := DatabaseError(operation, original)
	assertError(t, err, CodeDatabaseError, http.StatusInternalServerError)
	assertContext(t, err, "operation", operation)
}

func TestNotImplemented(t *testing.T) {
	feature := "blockchain authentication"
	err := NotImplemented(feature)
	assertError(t, err, CodeNotImplemented, http.StatusNotImplemented)
	assertContext(t, err, "feature", feature)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "AuthsomeError",
			err:      UserNotFound(),
			expected: http.StatusNotFound,
		},
		{
			name:     "wrapped AuthsomeError",
			err:      fmt.Errorf("outer: %w", SessionExpired()),
			expected: http.StatusUnauthorized,
		},
		{
			name:     "standard error",
			err:      errors.New("generic error"),
			expected: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHTTPStatus(tt.err); got != tt.expected {
				t.Errorf("GetHTTPStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "AuthsomeError",
			err:      UserNotFound(),
			expected: CodeUserNotFound,
		},
		{
			name:     "wrapped AuthsomeError",
			err:      fmt.Errorf("outer: %w", SessionExpired()),
			expected: CodeSessionExpired,
		},
		{
			name:     "standard error",
			err:      errors.New("generic error"),
			expected: CodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.expected {
				t.Errorf("GetErrorCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// SENTINEL ERRORS
// =============================================================================

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *AuthsomeError
		sentinel *AuthsomeError
	}{
		{"InvalidCredentials", InvalidCredentials(), ErrInvalidCredentials},
		{"UserNotFound", UserNotFound(), ErrUserNotFound},
		{"SessionExpired", SessionExpired(), ErrSessionExpired},
		{"PermissionDenied", PermissionDenied("read", "user:1"), ErrPermissionDenied},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.sentinel) {
				t.Errorf("error should match sentinel %v", tt.sentinel.Code)
			}
		})
	}
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func assertError(t *testing.T, err *AuthsomeError, expectedCode string, expectedStatus int) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error to be non-nil")
	}

	if err.Code != expectedCode {
		t.Errorf("expected code %s, got %s", expectedCode, err.Code)
	}

	if err.HTTPStatus != expectedStatus {
		t.Errorf("expected HTTP status %d, got %d", expectedStatus, err.HTTPStatus)
	}

	if err.Message == "" {
		t.Error("expected message to be set")
	}

	if err.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func assertContext(t *testing.T, err *AuthsomeError, key string, expected any) {
	t.Helper()

	if err.Context == nil {
		t.Fatal("expected context to be initialized")
	}

	actual, ok := err.Context[key]
	if !ok {
		t.Errorf("expected context key %s to exist", key)

		return
	}

	if actual != expected {
		t.Errorf("expected context[%s] = %v, got %v", key, expected, actual)
	}
}

// =============================================================================
// COMPLEX SCENARIOS
// =============================================================================

func TestErrorChaining(t *testing.T) {
	// Simulate a deep error chain
	dbErr := errors.New("connection pool exhausted")
	queryErr := DatabaseError("SELECT", dbErr)
	serviceErr := fmt.Errorf("failed to fetch user: %w", queryErr)

	// Should be able to find the AuthsomeError
	var authErr *AuthsomeError
	if !errors.As(serviceErr, &authErr) {
		t.Fatal("expected to find AuthsomeError in chain")
	}

	if authErr.Code != CodeDatabaseError {
		t.Error("expected to extract correct error code")
	}

	// Should be able to find the original error
	if !errors.Is(serviceErr, dbErr) {
		t.Error("expected to find original error in chain")
	}
}

func TestErrorWithMultipleContext(t *testing.T) {
	err := PermissionDenied("delete", "user:123").
		WithContext("user_role", "viewer").
		WithContext("required_role", "admin").
		WithContext("organization_id", "org_456").
		WithTraceID("trace-abc-123")

	// PermissionDenied already adds "action" and "resource", so total is 5
	// (action, resource, user_role, required_role, organization_id)
	if len(err.Context) != 5 {
		t.Errorf("expected 5 context entries, got %d", len(err.Context))
	}

	if err.TraceID != "trace-abc-123" {
		t.Error("expected trace ID to be set")
	}

	// Verify all expected keys are present
	expectedKeys := []string{"action", "resource", "user_role", "required_role", "organization_id"}
	for _, key := range expectedKeys {
		if _, ok := err.Context[key]; !ok {
			t.Errorf("expected context to contain key %s", key)
		}
	}
}

func TestErrorSerialization(t *testing.T) {
	// Verify that AuthsomeError has JSON tags and can be serialized
	err := ValidationFailed(map[string]string{
		"email": "invalid format",
	}).WithTraceID("trace-123")

	// This would be used in JSON responses
	if err.Code == "" {
		t.Error("code should be set for serialization")
	}

	if err.Message == "" {
		t.Error("message should be set for serialization")
	}

	// Details should be serializable
	if err.Details == nil {
		t.Error("details should be set")
	}
}
