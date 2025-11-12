package errs

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	forgeerrors "github.com/xraph/forge/errors"
)

// =============================================================================
// ERROR CODES
// =============================================================================

// Authsome-specific error codes for structured error handling.
const (
	// Authentication errors
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeEmailNotVerified   = "EMAIL_NOT_VERIFIED"
	CodeAccountLocked      = "ACCOUNT_LOCKED"
	CodeAccountDisabled    = "ACCOUNT_DISABLED"
	CodePasswordExpired    = "PASSWORD_EXPIRED"
	CodeTwoFactorRequired  = "TWO_FACTOR_REQUIRED"
	CodeStepUpRequired     = "STEP_UP_REQUIRED"
	CodeInvalidToken       = "INVALID_TOKEN"
	CodeTokenExpired       = "TOKEN_EXPIRED"
	CodeInvalidOTP         = "INVALID_OTP"
	CodeOTPExpired         = "OTP_EXPIRED"
	CodeMagicLinkExpired   = "MAGIC_LINK_EXPIRED"
	CodeMagicLinkInvalid   = "MAGIC_LINK_INVALID"

	// User errors
	CodeUserNotFound          = "USER_NOT_FOUND"
	CodeUserAlreadyExists     = "USER_ALREADY_EXISTS"
	CodeEmailAlreadyExists    = "EMAIL_ALREADY_EXISTS"
	CodeUsernameAlreadyExists = "USERNAME_ALREADY_EXISTS"
	CodePhoneAlreadyExists    = "PHONE_ALREADY_EXISTS"
	CodeInvalidEmail          = "INVALID_EMAIL"
	CodeInvalidPhone          = "INVALID_PHONE"
	CodeInvalidUsername       = "INVALID_USERNAME"
	CodeWeakPassword          = "WEAK_PASSWORD"

	// Session errors
	CodeSessionNotFound        = "SESSION_NOT_FOUND"
	CodeSessionExpired         = "SESSION_EXPIRED"
	CodeSessionInvalid         = "SESSION_INVALID"
	CodeSessionRevoked         = "SESSION_REVOKED"
	CodeConcurrentSessionLimit = "CONCURRENT_SESSION_LIMIT"
	CodeSessionConflict        = "SESSION_CONFLICT"

	// Organization errors
	CodeOrganizationNotFound     = "ORGANIZATION_NOT_FOUND"
	CodeOrganizationExists       = "ORGANIZATION_EXISTS"
	CodeNotMember                = "NOT_ORGANIZATION_MEMBER"
	CodeInsufficientRole         = "INSUFFICIENT_ROLE"
	CodeInvalidSlug              = "INVALID_SLUG"
	CodeSlugAlreadyExists        = "SLUG_ALREADY_EXISTS"
	CodeOrganizationLimitReached = "ORGANIZATION_LIMIT_REACHED"

	// Team errors
	CodeTeamNotFound      = "TEAM_NOT_FOUND"
	CodeTeamAlreadyExists = "TEAM_ALREADY_EXISTS"
	CodeNotTeamMember     = "NOT_TEAM_MEMBER"

	// Invitation errors
	CodeInvitationNotFound  = "INVITATION_NOT_FOUND"
	CodeInvitationExpired   = "INVITATION_EXPIRED"
	CodeInvitationAccepted  = "INVITATION_ACCEPTED"
	CodeInvitationCancelled = "INVITATION_CANCELLED"

	// RBAC errors
	CodePermissionDenied  = "PERMISSION_DENIED"
	CodeRoleNotFound      = "ROLE_NOT_FOUND"
	CodeRoleAlreadyExists = "ROLE_ALREADY_EXISTS"
	CodePolicyViolation   = "POLICY_VIOLATION"
	CodeInvalidPolicy     = "INVALID_POLICY"

	// Rate limiting
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	CodeTooManyAttempts   = "TOO_MANY_ATTEMPTS"

	// Validation errors
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeInvalidInput     = "INVALID_INPUT"
	CodeRequiredField    = "REQUIRED_FIELD"
	CodeInvalidFormat    = "INVALID_FORMAT"

	// Plugin errors
	CodePluginNotFound   = "PLUGIN_NOT_FOUND"
	CodePluginInitFailed = "PLUGIN_INIT_FAILED"
	CodePluginDisabled   = "PLUGIN_DISABLED"

	// OAuth/SSO errors
	CodeOAuthFailed        = "OAUTH_FAILED"
	CodeInvalidOAuthState  = "INVALID_OAUTH_STATE"
	CodeOAuthProviderError = "OAUTH_PROVIDER_ERROR"
	CodeSAMLError          = "SAML_ERROR"
	CodeOIDCError          = "OIDC_ERROR"

	// API Key errors
	CodeAPIKeyNotFound = "API_KEY_NOT_FOUND"
	CodeAPIKeyExpired  = "API_KEY_EXPIRED"
	CodeAPIKeyRevoked  = "API_KEY_REVOKED"
	CodeAPIKeyInvalid  = "API_KEY_INVALID"

	// Webhook errors
	CodeWebhookNotFound       = "WEBHOOK_NOT_FOUND"
	CodeWebhookDeliveryFailed = "WEBHOOK_DELIVERY_FAILED"

	// Notification errors
	CodeNotificationFailed   = "NOTIFICATION_FAILED"
	CodeTemplateNotFound     = "TEMPLATE_NOT_FOUND"
	CodeTemplateRenderFailed = "TEMPLATE_RENDER_FAILED"

	// Device errors
	CodeDeviceNotFound   = "DEVICE_NOT_FOUND"
	CodeDeviceNotTrusted = "DEVICE_NOT_TRUSTED"
	CodeDeviceBlocked    = "DEVICE_BLOCKED"

	// Passkey errors
	CodePasskeyNotFound           = "PASSKEY_NOT_FOUND"
	CodePasskeyVerificationFailed = "PASSKEY_VERIFICATION_FAILED"
	CodePasskeyRegistrationFailed = "PASSKEY_REGISTRATION_FAILED"

	// Compliance errors
	CodeComplianceViolation    = "COMPLIANCE_VIOLATION"
	CodeDataRetentionViolation = "DATA_RETENTION_VIOLATION"

	// SCIM errors
	CodeSCIMResourceNotFound = "SCIM_RESOURCE_NOT_FOUND"
	CodeSCIMInvalidFilter    = "SCIM_INVALID_FILTER"
	CodeSCIMInvalidPath      = "SCIM_INVALID_PATH"

	// General errors
	CodeInternalError  = "INTERNAL_ERROR"
	CodeNotImplemented = "NOT_IMPLEMENTED"
	CodeDatabaseError  = "DATABASE_ERROR"
	CodeCacheError     = "CACHE_ERROR"
	CodeConfigError    = "CONFIG_ERROR"
	CodeBadRequest     = "BAD_REQUEST"
	CodeNotFound       = "NOT_FOUND"
)

// =============================================================================
// AUTHSOME ERROR
// =============================================================================

// AuthsomeError represents an authsome-specific error with rich context.
// It embeds forge.HTTPError and adds authsome-specific fields.
type AuthsomeError struct {
	// Code is the authsome error code (e.g., "USER_NOT_FOUND")
	Code string `json:"code"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// HTTPStatus is the HTTP status code to return
	HTTPStatus int `json:"-"`

	// Err is the underlying error (if any)
	Err error `json:"-"`

	// Context provides additional debug information
	Context map[string]any `json:"context,omitempty"`

	// Timestamp is when the error occurred
	Timestamp time.Time `json:"timestamp"`

	// TraceID for distributed tracing (optional)
	TraceID string `json:"trace_id,omitempty"`

	// Details for additional structured information
	Details any `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *AuthsomeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *AuthsomeError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is interface for AuthsomeError.
// Compares by error code.
func (e *AuthsomeError) Is(target error) bool {
	t, ok := target.(*AuthsomeError)
	if !ok {
		return false
	}
	return e.Code != "" && e.Code == t.Code
}

// WithContext adds context to the error.
func (e *AuthsomeError) WithContext(key string, value any) *AuthsomeError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// WithDetails adds structured details to the error.
func (e *AuthsomeError) WithDetails(details any) *AuthsomeError {
	e.Details = details
	return e
}

// WithTraceID adds a trace ID for distributed tracing.
func (e *AuthsomeError) WithTraceID(traceID string) *AuthsomeError {
	e.TraceID = traceID
	return e
}

// WithError wraps an underlying error.
func (e *AuthsomeError) WithError(err error) *AuthsomeError {
	e.Err = err
	return e
}

// ToHTTPError converts AuthsomeError to forge HTTPError for handler responses.
func (e *AuthsomeError) ToHTTPError() *forgeerrors.HTTPError {
	return &forgeerrors.HTTPError{
		Code:    e.HTTPStatus,
		Message: e.Message,
		Err:     e.Err,
	}
}

// ToForgeError converts AuthsomeError to forge ForgeError for internal use.
func (e *AuthsomeError) ToForgeError() *forgeerrors.ForgeError {
	return &forgeerrors.ForgeError{
		Code:      e.Code,
		Message:   e.Message,
		Cause:     e.Err,
		Timestamp: e.Timestamp,
		Context:   e.Context,
	}
}

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// New creates a new AuthsomeError with the given parameters.
func New(code, message string, httpStatus int) *AuthsomeError {
	return &AuthsomeError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Timestamp:  time.Now(),
		Context:    make(map[string]any),
	}
}

// Wrap wraps an existing error with authsome context.
func Wrap(err error, code, message string, httpStatus int) *AuthsomeError {
	return &AuthsomeError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
		Timestamp:  time.Now(),
		Context:    make(map[string]any),
	}
}

// =============================================================================
// AUTHENTICATION ERRORS
// =============================================================================

func InvalidCredentials() *AuthsomeError {
	return New(CodeInvalidCredentials, "Invalid email or password", http.StatusUnauthorized)
}

func EmailNotVerified(email string) *AuthsomeError {
	return New(CodeEmailNotVerified, "Email address not verified", http.StatusForbidden).
		WithContext("email", email)
}

func AccountLocked(reason string) *AuthsomeError {
	return New(CodeAccountLocked, "Account is locked", http.StatusForbidden).
		WithContext("reason", reason)
}

func AccountDisabled() *AuthsomeError {
	return New(CodeAccountDisabled, "Account has been disabled", http.StatusForbidden)
}

func PasswordExpired() *AuthsomeError {
	return New(CodePasswordExpired, "Password has expired and must be changed", http.StatusForbidden)
}

func TwoFactorRequired() *AuthsomeError {
	return New(CodeTwoFactorRequired, "Two-factor authentication required", http.StatusForbidden)
}

func StepUpRequired() *AuthsomeError {
	return New(CodeStepUpRequired, "Step-up authentication required", http.StatusForbidden)
}

func InvalidToken() *AuthsomeError {
	return New(CodeInvalidToken, "Invalid or malformed token", http.StatusUnauthorized)
}

func TokenExpired() *AuthsomeError {
	return New(CodeTokenExpired, "Token has expired", http.StatusUnauthorized)
}

func InvalidOTP() *AuthsomeError {
	return New(CodeInvalidOTP, "Invalid one-time password", http.StatusUnauthorized)
}

func OTPExpired() *AuthsomeError {
	return New(CodeOTPExpired, "One-time password has expired", http.StatusUnauthorized)
}

func MagicLinkExpired() *AuthsomeError {
	return New(CodeMagicLinkExpired, "Magic link has expired", http.StatusGone)
}

func MagicLinkInvalid() *AuthsomeError {
	return New(CodeMagicLinkInvalid, "Invalid magic link", http.StatusBadRequest)
}

// =============================================================================
// USER ERRORS
// =============================================================================

func UserNotFound() *AuthsomeError {
	return New(CodeUserNotFound, "User not found", http.StatusNotFound)
}

func UserAlreadyExists(identifier string) *AuthsomeError {
	return New(CodeUserAlreadyExists, "User already exists", http.StatusConflict).
		WithContext("identifier", identifier)
}

func EmailAlreadyExists(email string) *AuthsomeError {
	return New(CodeEmailAlreadyExists, "Email address already registered", http.StatusConflict).
		WithContext("email", email)
}

func UsernameAlreadyExists(username string) *AuthsomeError {
	return New(CodeUsernameAlreadyExists, "Username already taken", http.StatusConflict).
		WithContext("username", username)
}

func PhoneAlreadyExists(phone string) *AuthsomeError {
	return New(CodePhoneAlreadyExists, "Phone number already registered", http.StatusConflict).
		WithContext("phone", phone)
}

func InvalidEmail(email string) *AuthsomeError {
	return New(CodeInvalidEmail, "Invalid email address format", http.StatusBadRequest).
		WithContext("email", email)
}

func InvalidPhone(phone string) *AuthsomeError {
	return New(CodeInvalidPhone, "Invalid phone number format", http.StatusBadRequest).
		WithContext("phone", phone)
}

func InvalidUsername(username string) *AuthsomeError {
	return New(CodeInvalidUsername, "Invalid username format", http.StatusBadRequest).
		WithContext("username", username)
}

func WeakPassword(reason string) *AuthsomeError {
	return New(CodeWeakPassword, "Password does not meet security requirements", http.StatusBadRequest).
		WithContext("reason", reason)
}

// =============================================================================
// SESSION ERRORS
// =============================================================================

func SessionNotFound() *AuthsomeError {
	return New(CodeSessionNotFound, "Session not found", http.StatusUnauthorized)
}

func SessionExpired() *AuthsomeError {
	return New(CodeSessionExpired, "Session has expired", http.StatusUnauthorized)
}

func SessionInvalid() *AuthsomeError {
	return New(CodeSessionInvalid, "Invalid session", http.StatusUnauthorized)
}

func SessionRevoked() *AuthsomeError {
	return New(CodeSessionRevoked, "Session has been revoked", http.StatusUnauthorized)
}

func ConcurrentSessionLimit() *AuthsomeError {
	return New(CodeConcurrentSessionLimit, "Concurrent session limit reached", http.StatusConflict)
}

func SessionConflict() *AuthsomeError {
	return New(CodeSessionConflict, "Session conflict detected", http.StatusConflict)
}

// =============================================================================
// ORGANIZATION ERRORS
// =============================================================================

func OrganizationNotFound() *AuthsomeError {
	return New(CodeOrganizationNotFound, "Organization not found", http.StatusNotFound)
}

func OrganizationExists(identifier string) *AuthsomeError {
	return New(CodeOrganizationExists, "Organization already exists", http.StatusConflict).
		WithContext("identifier", identifier)
}

func NotMember() *AuthsomeError {
	return New(CodeNotMember, "Not a member of this organization", http.StatusForbidden)
}

func InsufficientRole(required string) *AuthsomeError {
	return New(CodeInsufficientRole, "Insufficient role for this action", http.StatusForbidden).
		WithContext("required_role", required)
}

func InvalidSlug(slug string) *AuthsomeError {
	return New(CodeInvalidSlug, "Invalid organization slug format", http.StatusBadRequest).
		WithContext("slug", slug)
}

func SlugAlreadyExists(slug string) *AuthsomeError {
	return New(CodeSlugAlreadyExists, "Organization slug already in use", http.StatusConflict).
		WithContext("slug", slug)
}

func OrganizationLimitReached(limit int) *AuthsomeError {
	return New(CodeOrganizationLimitReached, "Organization limit reached", http.StatusForbidden).
		WithContext("limit", limit)
}

// =============================================================================
// TEAM ERRORS
// =============================================================================

func TeamNotFound() *AuthsomeError {
	return New(CodeTeamNotFound, "Team not found", http.StatusNotFound)
}

func TeamAlreadyExists(name string) *AuthsomeError {
	return New(CodeTeamAlreadyExists, "Team already exists", http.StatusConflict).
		WithContext("name", name)
}

func NotTeamMember() *AuthsomeError {
	return New(CodeNotTeamMember, "Not a member of this team", http.StatusForbidden)
}

// =============================================================================
// INVITATION ERRORS
// =============================================================================

func InvitationNotFound() *AuthsomeError {
	return New(CodeInvitationNotFound, "Invitation not found", http.StatusNotFound)
}

func InvitationExpired() *AuthsomeError {
	return New(CodeInvitationExpired, "Invitation has expired", http.StatusGone)
}

func InvitationAccepted() *AuthsomeError {
	return New(CodeInvitationAccepted, "Invitation already accepted", http.StatusConflict)
}

func InvitationCancelled() *AuthsomeError {
	return New(CodeInvitationCancelled, "Invitation has been cancelled", http.StatusGone)
}

// =============================================================================
// RBAC ERRORS
// =============================================================================

func PermissionDenied(action, resource string) *AuthsomeError {
	return New(CodePermissionDenied, "Permission denied", http.StatusForbidden).
		WithContext("action", action).
		WithContext("resource", resource)
}

func RoleNotFound(role string) *AuthsomeError {
	return New(CodeRoleNotFound, "Role not found", http.StatusNotFound).
		WithContext("role", role)
}

func RoleAlreadyExists(role string) *AuthsomeError {
	return New(CodeRoleAlreadyExists, "Role already exists", http.StatusConflict).
		WithContext("role", role)
}

func PolicyViolation(policy, reason string) *AuthsomeError {
	return New(CodePolicyViolation, "Policy violation", http.StatusForbidden).
		WithContext("policy", policy).
		WithContext("reason", reason)
}

func InvalidPolicy(reason string) *AuthsomeError {
	return New(CodeInvalidPolicy, "Invalid policy", http.StatusBadRequest).
		WithContext("reason", reason)
}

// =============================================================================
// RATE LIMITING ERRORS
// =============================================================================

func RateLimitExceeded(retryAfter time.Duration) *AuthsomeError {
	return New(CodeRateLimitExceeded, "Rate limit exceeded", http.StatusTooManyRequests).
		WithContext("retry_after", retryAfter.Seconds())
}

func TooManyAttempts(retryAfter time.Duration) *AuthsomeError {
	return New(CodeTooManyAttempts, "Too many failed attempts", http.StatusTooManyRequests).
		WithContext("retry_after", retryAfter.Seconds())
}

// =============================================================================
// VALIDATION ERRORS
// =============================================================================

func ValidationFailed(fields map[string]string) *AuthsomeError {
	return New(CodeValidationFailed, "Validation failed", http.StatusBadRequest).
		WithDetails(fields)
}

func InvalidInput(field, reason string) *AuthsomeError {
	return New(CodeInvalidInput, "Invalid input", http.StatusBadRequest).
		WithContext("field", field).
		WithContext("reason", reason)
}

func RequiredField(field string) *AuthsomeError {
	return New(CodeRequiredField, "Required field missing", http.StatusBadRequest).
		WithContext("field", field)
}

func InvalidFormat(field, expectedFormat string) *AuthsomeError {
	return New(CodeInvalidFormat, "Invalid format", http.StatusBadRequest).
		WithContext("field", field).
		WithContext("expected_format", expectedFormat)
}

// =============================================================================
// PLUGIN ERRORS
// =============================================================================

func PluginNotFound(pluginID string) *AuthsomeError {
	return New(CodePluginNotFound, "Plugin not found", http.StatusNotFound).
		WithContext("plugin_id", pluginID)
}

func PluginInitFailed(pluginID string, err error) *AuthsomeError {
	return Wrap(err, CodePluginInitFailed, "Failed to initialize plugin", http.StatusInternalServerError).
		WithContext("plugin_id", pluginID)
}

func PluginDisabled(pluginID string) *AuthsomeError {
	return New(CodePluginDisabled, "Plugin is disabled", http.StatusServiceUnavailable).
		WithContext("plugin_id", pluginID)
}

// =============================================================================
// OAUTH/SSO ERRORS
// =============================================================================

func OAuthFailed(provider, reason string) *AuthsomeError {
	return New(CodeOAuthFailed, "OAuth authentication failed", http.StatusBadRequest).
		WithContext("provider", provider).
		WithContext("reason", reason)
}

func InvalidOAuthState() *AuthsomeError {
	return New(CodeInvalidOAuthState, "Invalid OAuth state parameter", http.StatusBadRequest)
}

func OAuthProviderError(provider string, err error) *AuthsomeError {
	return Wrap(err, CodeOAuthProviderError, "OAuth provider error", http.StatusBadGateway).
		WithContext("provider", provider)
}

func SAMLError(reason string) *AuthsomeError {
	return New(CodeSAMLError, "SAML authentication error", http.StatusBadRequest).
		WithContext("reason", reason)
}

func OIDCError(reason string) *AuthsomeError {
	return New(CodeOIDCError, "OIDC authentication error", http.StatusBadRequest).
		WithContext("reason", reason)
}

// =============================================================================
// API KEY ERRORS
// =============================================================================

func APIKeyNotFound() *AuthsomeError {
	return New(CodeAPIKeyNotFound, "API key not found", http.StatusUnauthorized)
}

func APIKeyExpired() *AuthsomeError {
	return New(CodeAPIKeyExpired, "API key has expired", http.StatusUnauthorized)
}

func APIKeyRevoked() *AuthsomeError {
	return New(CodeAPIKeyRevoked, "API key has been revoked", http.StatusUnauthorized)
}

func APIKeyInvalid() *AuthsomeError {
	return New(CodeAPIKeyInvalid, "Invalid API key format", http.StatusUnauthorized)
}

// =============================================================================
// WEBHOOK ERRORS
// =============================================================================

func WebhookNotFound() *AuthsomeError {
	return New(CodeWebhookNotFound, "Webhook not found", http.StatusNotFound)
}

func WebhookDeliveryFailed(err error) *AuthsomeError {
	return Wrap(err, CodeWebhookDeliveryFailed, "Webhook delivery failed", http.StatusBadGateway)
}

// =============================================================================
// NOTIFICATION ERRORS
// =============================================================================

func NotificationFailed(err error) *AuthsomeError {
	return Wrap(err, CodeNotificationFailed, "Notification delivery failed", http.StatusInternalServerError)
}

func TemplateNotFound(templateID string) *AuthsomeError {
	return New(CodeTemplateNotFound, "Template not found", http.StatusNotFound).
		WithContext("template_id", templateID)
}

func TemplateRenderFailed(templateID string, err error) *AuthsomeError {
	return Wrap(err, CodeTemplateRenderFailed, "Template rendering failed", http.StatusInternalServerError).
		WithContext("template_id", templateID)
}

// =============================================================================
// DEVICE ERRORS
// =============================================================================

func DeviceNotFound() *AuthsomeError {
	return New(CodeDeviceNotFound, "Device not found", http.StatusNotFound)
}

func DeviceNotTrusted() *AuthsomeError {
	return New(CodeDeviceNotTrusted, "Device is not trusted", http.StatusForbidden)
}

func DeviceBlocked() *AuthsomeError {
	return New(CodeDeviceBlocked, "Device has been blocked", http.StatusForbidden)
}

// =============================================================================
// PASSKEY ERRORS
// =============================================================================

func PasskeyNotFound() *AuthsomeError {
	return New(CodePasskeyNotFound, "Passkey not found", http.StatusNotFound)
}

func PasskeyVerificationFailed(reason string) *AuthsomeError {
	return New(CodePasskeyVerificationFailed, "Passkey verification failed", http.StatusUnauthorized).
		WithContext("reason", reason)
}

func PasskeyRegistrationFailed(reason string) *AuthsomeError {
	return New(CodePasskeyRegistrationFailed, "Passkey registration failed", http.StatusBadRequest).
		WithContext("reason", reason)
}

// =============================================================================
// COMPLIANCE ERRORS
// =============================================================================

func ComplianceViolation(policy, reason string) *AuthsomeError {
	return New(CodeComplianceViolation, "Compliance policy violation", http.StatusForbidden).
		WithContext("policy", policy).
		WithContext("reason", reason)
}

func DataRetentionViolation(reason string) *AuthsomeError {
	return New(CodeDataRetentionViolation, "Data retention policy violation", http.StatusForbidden).
		WithContext("reason", reason)
}

// =============================================================================
// SCIM ERRORS
// =============================================================================

func SCIMResourceNotFound(resourceType string) *AuthsomeError {
	return New(CodeSCIMResourceNotFound, "SCIM resource not found", http.StatusNotFound).
		WithContext("resource_type", resourceType)
}

func SCIMInvalidFilter(filter string) *AuthsomeError {
	return New(CodeSCIMInvalidFilter, "Invalid SCIM filter", http.StatusBadRequest).
		WithContext("filter", filter)
}

func SCIMInvalidPath(path string) *AuthsomeError {
	return New(CodeSCIMInvalidPath, "Invalid SCIM path", http.StatusBadRequest).
		WithContext("path", path)
}

// =============================================================================
// GENERAL ERRORS
// =============================================================================

func InternalError(err error) *AuthsomeError {
	return Wrap(err, CodeInternalError, "Internal server error", http.StatusInternalServerError)
}

func NotImplemented(feature string) *AuthsomeError {
	return New(CodeNotImplemented, "Feature not implemented", http.StatusNotImplemented).
		WithContext("feature", feature)
}

func DatabaseError(operation string, err error) *AuthsomeError {
	return Wrap(err, CodeDatabaseError, "Database operation failed", http.StatusInternalServerError).
		WithContext("operation", operation)
}

func CacheError(operation string, err error) *AuthsomeError {
	return Wrap(err, CodeCacheError, "Cache operation failed", http.StatusInternalServerError).
		WithContext("operation", operation)
}

func ConfigError(key string, err error) *AuthsomeError {
	return Wrap(err, CodeConfigError, "Configuration error", http.StatusInternalServerError).
		WithContext("config_key", key)
}

func BadRequest(msg string) *AuthsomeError {
	return New(CodeBadRequest, msg, http.StatusBadRequest)
}

func InternalServerError(msg string) *AuthsomeError {
	return New(CodeInternalError, msg, http.StatusInternalServerError)
}

func NotFound(msg string) *AuthsomeError {
	return New(CodeNotFound, msg, http.StatusNotFound)
}

// =============================================================================
// ERROR HELPERS
// =============================================================================

// Is checks if an error matches the target AuthsomeError by code.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target type.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// GetHTTPStatus extracts HTTP status code from error, returns 500 if not found.
func GetHTTPStatus(err error) int {
	var authErr *AuthsomeError
	if errors.As(err, &authErr) {
		return authErr.HTTPStatus
	}

	var httpErr *forgeerrors.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Code
	}

	return http.StatusInternalServerError
}

// GetErrorCode extracts error code from AuthsomeError, returns CodeInternalError if not found.
func GetErrorCode(err error) string {
	var authErr *AuthsomeError
	if errors.As(err, &authErr) {
		return authErr.Code
	}
	return CodeInternalError
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

// Sentinel errors for common cases (no message, just for comparison).
var (
	ErrInvalidCredentials   = &AuthsomeError{Code: CodeInvalidCredentials}
	ErrEmailNotVerified     = &AuthsomeError{Code: CodeEmailNotVerified}
	ErrAccountLocked        = &AuthsomeError{Code: CodeAccountLocked}
	ErrUserNotFound         = &AuthsomeError{Code: CodeUserNotFound}
	ErrUserAlreadyExists    = &AuthsomeError{Code: CodeUserAlreadyExists}
	ErrEmailAlreadyExists   = &AuthsomeError{Code: CodeEmailAlreadyExists}
	ErrSessionNotFound      = &AuthsomeError{Code: CodeSessionNotFound}
	ErrSessionExpired       = &AuthsomeError{Code: CodeSessionExpired}
	ErrOrganizationNotFound = &AuthsomeError{Code: CodeOrganizationNotFound}
	ErrPermissionDenied     = &AuthsomeError{Code: CodePermissionDenied}
	ErrRateLimitExceeded    = &AuthsomeError{Code: CodeRateLimitExceeded}
	ErrValidationFailed     = &AuthsomeError{Code: CodeValidationFailed}
	ErrPluginNotFound       = &AuthsomeError{Code: CodePluginNotFound}
	ErrInternalError        = &AuthsomeError{Code: CodeInternalError}
)
