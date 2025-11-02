package mtls

import "errors"

var (
	// Certificate Errors
	ErrCertificateNotFound      = errors.New("certificate not found")
	ErrCertificateExpired       = errors.New("certificate has expired")
	ErrCertificateRevoked       = errors.New("certificate has been revoked")
	ErrCertificateInvalid       = errors.New("certificate is invalid")
	ErrCertificateNotYetValid   = errors.New("certificate is not yet valid")
	ErrCertificateSuspended     = errors.New("certificate is suspended")
	
	// Validation Errors
	ErrInvalidSignature         = errors.New("invalid certificate signature")
	ErrUntrustedCA              = errors.New("certificate issued by untrusted CA")
	ErrInvalidKeyUsage          = errors.New("invalid key usage for authentication")
	ErrKeyTooWeak               = errors.New("certificate key size too weak")
	ErrUnsupportedAlgorithm     = errors.New("unsupported key or signature algorithm")
	ErrCertificateChainInvalid  = errors.New("certificate chain validation failed")
	
	// Pinning Errors
	ErrCertificateNotPinned     = errors.New("certificate not pinned (required by policy)")
	ErrPinExpired               = errors.New("certificate pin has expired")
	ErrPinMismatch              = errors.New("certificate does not match pinned fingerprint")
	
	// Revocation Errors
	ErrCRLCheckFailed           = errors.New("CRL check failed")
	ErrOCSPCheckFailed          = errors.New("OCSP check failed")
	ErrRevocationUnavailable    = errors.New("revocation status unavailable")
	
	// PIV/CAC Errors
	ErrNotPIVCertificate        = errors.New("certificate is not a PIV certificate")
	ErrNotCACCertificate        = errors.New("certificate is not a CAC certificate")
	ErrSmartCardNotPresent      = errors.New("smart card not present")
	ErrSmartCardLocked          = errors.New("smart card is locked")
	ErrPINRequired              = errors.New("smart card PIN required")
	ErrInvalidPIN               = errors.New("invalid smart card PIN")
	
	// HSM Errors
	ErrHSMNotConfigured         = errors.New("HSM not configured")
	ErrHSMConnectionFailed      = errors.New("HSM connection failed")
	ErrHSMKeyNotFound           = errors.New("HSM key not found")
	ErrHSMOperationFailed       = errors.New("HSM operation failed")
	ErrHSMProviderUnsupported   = errors.New("HSM provider not supported")
	
	// Policy Errors
	ErrPolicyNotFound           = errors.New("certificate policy not found")
	ErrPolicyViolation          = errors.New("certificate policy violation")
	ErrPolicyRequired           = errors.New("certificate policy required but not found")
	
	// Trust Anchor Errors
	ErrTrustAnchorNotFound      = errors.New("trust anchor not found")
	ErrTrustAnchorExpired       = errors.New("trust anchor has expired")
	ErrNoTrustAnchors           = errors.New("no trust anchors configured")
	
	// General Errors
	ErrCertificateParseFailed   = errors.New("failed to parse certificate")
	ErrCRLParseFailed           = errors.New("failed to parse CRL")
	ErrOCSPParseFailed          = errors.New("failed to parse OCSP response")
	ErrInvalidPEM               = errors.New("invalid PEM format")
	ErrMissingClientCert        = errors.New("client certificate not provided")
)

// ValidationError provides detailed validation failure information
type ValidationError struct {
	Code       string
	Message    string
	Field      string
	Details    map[string]interface{}
	Underlying error
}

func (e *ValidationError) Error() string {
	if e.Underlying != nil {
		return e.Message + ": " + e.Underlying.Error()
	}
	return e.Message
}

func (e *ValidationError) Unwrap() error {
	return e.Underlying
}

// NewValidationError creates a new validation error
func NewValidationError(code, message, field string, underlying error) *ValidationError {
	return &ValidationError{
		Code:       code,
		Message:    message,
		Field:      field,
		Details:    make(map[string]interface{}),
		Underlying: underlying,
	}
}

// WithDetail adds a detail to the validation error
func (e *ValidationError) WithDetail(key string, value interface{}) *ValidationError {
	e.Details[key] = value
	return e
}

