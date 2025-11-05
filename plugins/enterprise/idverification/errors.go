package idverification

import "errors"

// Configuration errors
var (
	ErrNoProviderEnabled      = errors.New("no identity verification provider enabled")
	ErrInvalidDefaultProvider = errors.New("invalid default provider")
	ErrProviderNotEnabled     = errors.New("provider not enabled")
	ErrMissingAPIToken        = errors.New("missing API token")
	ErrMissingAPICredentials  = errors.New("missing API credentials")
	ErrMissingAPIKey          = errors.New("missing API key")
	ErrUnsupportedProvider    = errors.New("unsupported provider")
	ErrInvalidRiskScore       = errors.New("invalid risk score range (must be 0-100)")
	ErrInvalidConfidenceScore = errors.New("invalid confidence score range (must be 0-100)")
	ErrInvalidMinimumAge      = errors.New("invalid minimum age")
	ErrInvalidRateLimit       = errors.New("invalid rate limit")
	ErrInvalidMaxAttempts     = errors.New("invalid max verification attempts")
)

// Verification errors
var (
	ErrVerificationNotFound    = errors.New("verification not found")
	ErrVerificationExpired     = errors.New("verification has expired")
	ErrVerificationFailed      = errors.New("verification failed")
	ErrVerificationPending     = errors.New("verification is still pending")
	ErrMaxAttemptsReached      = errors.New("maximum verification attempts reached")
	ErrSessionNotFound         = errors.New("verification session not found")
	ErrSessionExpired          = errors.New("verification session has expired")
	ErrInvalidVerificationType = errors.New("invalid verification type")
	ErrUserAlreadyVerified     = errors.New("user is already verified")
	ErrVerificationBlocked     = errors.New("user is blocked from verification")
)

// Document errors
var (
	ErrDocumentNotSupported = errors.New("document type not supported")
	ErrCountryNotSupported  = errors.New("country not supported")
	ErrDocumentExpired      = errors.New("document has expired")
	ErrDocumentInvalid      = errors.New("document is invalid")
	ErrDocumentNotFound     = errors.New("document not found")
	ErrInvalidDocumentImage = errors.New("invalid document image")
	ErrDocumentUploadFailed = errors.New("document upload failed")
)

// Risk and compliance errors
var (
	ErrHighRiskDetected    = errors.New("high risk detected")
	ErrSanctionsListMatch  = errors.New("user found on sanctions list")
	ErrPEPDetected         = errors.New("politically exposed person detected")
	ErrAMLCheckFailed      = errors.New("AML check failed")
	ErrAgeBelowMinimum     = errors.New("age below minimum requirement")
	ErrLivenessCheckFailed = errors.New("liveness check failed")
)

// Provider errors
var (
	ErrProviderAPIError        = errors.New("provider API error")
	ErrProviderTimeout         = errors.New("provider request timeout")
	ErrProviderRateLimited     = errors.New("provider rate limit exceeded")
	ErrInvalidProviderResponse = errors.New("invalid provider response")
	ErrProviderWebhookInvalid  = errors.New("invalid provider webhook")
)

// Rate limit errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrTooManyAttempts   = errors.New("too many verification attempts")
)
