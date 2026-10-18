package telemetry

import "strings"

// Values of the method label on authsome_signin_attempts_total and of
// authsome.auth.method on the issue session span. AuthMethod folds every
// raw method string into one of these.
const (
	MethodPassword          = "password"
	MethodPasskey           = "passkey"
	MethodSocial            = "social"
	MethodSSO               = "sso"
	MethodMagicLink         = "magic_link"
	MethodPhone             = "phone"
	MethodEmailVerification = "email_verification"
	MethodOther             = "other"
)

// AuthMethod folds the raw auth method an IssueSessionRequest carries into
// the bounded set above. "social:github" and "sso:acme-okta" lose the part
// after the colon, which names a provider or an SSO connection, and a
// "+mfa" suffix is dropped because the second factor has its own counter.
// Anything unrecognised, including a custom plugin's own method name, is
// "other".
func AuthMethod(raw string) string {
	raw = strings.TrimSuffix(raw, "+mfa")
	if i := strings.IndexByte(raw, ':'); i >= 0 {
		raw = raw[:i]
	}
	switch raw {
	case "password":
		return MethodPassword
	case "passkey":
		return MethodPasskey
	case "social":
		return MethodSocial
	case "sso":
		return MethodSSO
	case "magiclink", "magic_link":
		return MethodMagicLink
	case "phone":
		return MethodPhone
	case "email_verification":
		return MethodEmailVerification
	default:
		return MethodOther
	}
}

// Values of the result label on authsome_signin_attempts_total.
//
// SignInInvalid covers a wrong password and an identifier nobody holds
// alike. Nothing authsome records tells those two apart.
const (
	SignInOK              = "ok"
	SignInInvalid         = "invalid"
	SignInLocked          = "locked"
	SignInBanned          = "banned"
	SignInUnverified      = "unverified"
	SignInPasswordExpired = "password_expired"
	SignInMFARequired     = "mfa_required"
	SignInRateLimited     = "rate_limited"
	SignInDenied          = "denied"
	SignInError           = "error"
)

// Values of the flow, method and result labels on
// authsome_mfa_verifications_total.
const (
	MFAFlowSignIn = "signin"
	MFAFlowVerify = "verify"

	MFAMethodTOTP     = "totp"
	MFAMethodSMS      = "sms"
	MFAMethodRecovery = "recovery"

	MFAOK        = "ok"
	MFAInvalid   = "invalid"
	MFAReplayed  = "replayed"
	MFAExpired   = "expired"
	MFAExhausted = "exhausted"
	MFAError     = "error"
)

// Values of the result label on authsome_sessions_refreshed_total and of
// authsome.result on the refresh session span.
const (
	RefreshOK             = "ok"
	RefreshInvalid        = "invalid"
	RefreshExpired        = "expired"
	RefreshBanned         = "banned"
	RefreshReplayDetected = "replay_detected"
	RefreshDPoPRejected   = "dpop_rejected"
	RefreshError          = "error"
)

// Values of authsome.refresh.via on the refresh session span: which
// credential the caller presented.
const (
	RefreshViaRefreshToken = "refresh_token"
	RefreshViaClient       = "oauth2_client"
	RefreshViaSessionToken = "session_token"
	RefreshViaSessionID    = "session_id"
)

// Values of the operation and algorithm labels on
// authsome_password_hash_duration_seconds.
const (
	PasswordHash   = "hash"
	PasswordVerify = "verify"

	AlgorithmArgon2id = "argon2id"
	AlgorithmBcrypt   = "bcrypt"
)

// Values of the result label on authsome_apikey_authentications_total and
// of authsome.result on the authenticate api key span.
//
// APIKeyInvalid covers a malformed key, an unknown prefix and a wrong
// secret alike, so the counter cannot be used to learn which prefixes
// exist.
const (
	APIKeyOK        = "ok"
	APIKeyInvalid   = "invalid"
	APIKeyExpired   = "expired"
	APIKeyThrottled = "throttled"
	APIKeyDenied    = "denied"
	APIKeyError     = "error"
)

// Values of the grant_type and result labels on
// authsome_oauth2_tokens_requested_total.
const (
	GrantAuthorizationCode = "authorization_code"
	GrantRefreshToken      = "refresh_token"
	GrantClientCredentials = "client_credentials"
	GrantDeviceCode        = "device_code"
	GrantTokenExchange     = "token_exchange"
	GrantOther             = "other"

	// OAuth2Rejected is any other refusal: a bad, used or expired code, a
	// refresh token that failed, a malformed request, a bad DPoP proof.
	// OAuth2Pending is a device-code poll answered authorization_pending
	// or slow_down, which is the flow working as designed.
	OAuth2OK            = "ok"
	OAuth2InvalidClient = "invalid_client"
	OAuth2Rejected      = "rejected"
	OAuth2Pending       = "pending"
	OAuth2Error         = "error"
)

// Values of the kind label on authsome_provider_request_duration_seconds
// and of authsome.provider.kind on the client spans.
const (
	ProviderSocial       = "social"
	ProviderOIDC         = "oidc"
	ProviderSAMLMetadata = "saml_metadata"
	ProviderJWKS         = "jwks"
	ProviderSMS          = "sms"
	ProviderEmail        = "email"
	ProviderCaptcha      = "captcha"
	ProviderBreach       = "breach"
	ProviderCRM          = "crm"
	ProviderWebhook      = "webhook"
)

// Values of the result label on authsome_provider_request_duration_seconds.
const (
	ProviderOK       = "ok"
	Provider4xx      = "4xx"
	Provider5xx      = "5xx"
	ProviderTimeout  = "timeout"
	ProviderCanceled = "canceled"
	ProviderError    = "error"
)

// Values of the kind label on authsome_retention_rows_deleted_total and of
// authsome.retention.kind on the sweep failed event.
const (
	RetentionSessions             = "sessions"
	RetentionVerifications        = "verifications"
	RetentionPasswordResets       = "password_resets"
	RetentionRevokedRefreshTokens = "revoked_refresh_tokens"
	RetentionCeremonyState        = "ceremony_state"
	RetentionPlugins              = "plugins"
)

// Values of the result label on authsome_retention_sweeps_attempted_total.
const (
	SweepOK      = "ok"
	SweepError   = "error"
	SweepSkipped = "skipped"
)

// Values of the provider and outcome labels on
// authsome_retention_jobs_processed_total.
const (
	CRMHubSpot = "hubspot"
	CRMGeneric = "generic"
	CRMOther   = "other"

	JobDone       = "done"
	JobRetry      = "retry"
	JobDead       = "dead"
	JobDeferred   = "deferred"
	JobSuppressed = "suppressed"
)

// Values of the control label on authsome_controls_degraded_total: a
// security control that could not run and let the request through.
const (
	ControlRateLimit       = "ratelimit"
	ControlIdentifierLimit = "identifier_limit"
	ControlSettings        = "settings"
	ControlPasswordHistory = "password_history"
	ControlBreachCheck     = "breach_check"
)

// Values of the limiter label on authsome_requests_throttled_total and
// authsome_ratelimiter_errors_total.
const (
	LimiterRoute      = "route"
	LimiterIdentifier = "identifier"
)

// Bounded folds a hook action, resource or outcome into a label value. The
// built-in values are dotted lower-case constants; anything that is not
// (too long, or carrying characters a constant never would) is "other", so
// a plugin that builds one from input cannot grow the series without bound.
func Bounded(s string) string {
	if s == "" {
		return ""
	}
	if len(s) > 64 {
		return "other"
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '.', c == '_', c == '-', c == ':':
		default:
			return "other"
		}
	}
	return s
}
