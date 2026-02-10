package jwt

import (
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// =============================================================================
// JWT-SPECIFIC ERROR CODES
// =============================================================================

const (
	CodeJWTKeyNotFound            = "JWT_KEY_NOT_FOUND"
	CodeJWTKeyAlreadyExists       = "JWT_KEY_ALREADY_EXISTS"
	CodeJWTKeyExpired             = "JWT_KEY_EXPIRED"
	CodeJWTKeyInactive            = "JWT_KEY_INACTIVE"
	CodeNoActiveSigningKey        = "NO_ACTIVE_SIGNING_KEY"
	CodeJWTGenerationFailed       = "JWT_GENERATION_FAILED"
	CodeJWTVerificationFailed     = "JWT_VERIFICATION_FAILED"
	CodeInvalidJWTAlgorithm       = "INVALID_JWT_ALGORITHM"
	CodeInvalidJWTKeyType         = "INVALID_JWT_KEY_TYPE"
	CodeJWTKeyDecryptionFailed    = "JWT_KEY_DECRYPTION_FAILED"
	CodeJWTKeyEncryptionFailed    = "JWT_KEY_ENCRYPTION_FAILED"
	CodeJWTParsingFailed          = "JWT_PARSING_FAILED"
	CodeJWTSigningFailed          = "JWT_SIGNING_FAILED"
	CodeMissingKIDHeader          = "MISSING_KID_HEADER"
	CodeInvalidJWTAudience        = "INVALID_JWT_AUDIENCE"
	CodeInvalidJWTTokenType       = "INVALID_JWT_TOKEN_TYPE"
	CodeJWKSGenerationFailed      = "JWKS_GENERATION_FAILED"
	CodeInvalidJWTClaims          = "INVALID_JWT_CLAIMS"
	CodeJWTKeyGenerationFailed    = "JWT_KEY_GENERATION_FAILED"
	CodeCannotSignWithoutPrivate  = "CANNOT_SIGN_WITHOUT_PRIVATE_KEY"
	CodeCannotVerifyWithoutPublic = "CANNOT_VERIFY_WITHOUT_PUBLIC_KEY"
)

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// JWTKeyNotFound returns an error when a JWT key is not found.
func JWTKeyNotFound() *errs.AuthsomeError {
	return errs.New(CodeJWTKeyNotFound, "JWT signing key not found", http.StatusNotFound)
}

func JWTKeyAlreadyExists(keyID string) *errs.AuthsomeError {
	return errs.New(CodeJWTKeyAlreadyExists, "JWT key already exists", http.StatusConflict).
		WithContext("key_id", keyID)
}

func JWTKeyExpired(keyID string) *errs.AuthsomeError {
	return errs.New(CodeJWTKeyExpired, "JWT signing key has expired", http.StatusForbidden).
		WithContext("key_id", keyID)
}

func JWTKeyInactive(keyID string) *errs.AuthsomeError {
	return errs.New(CodeJWTKeyInactive, "JWT signing key is not active", http.StatusForbidden).
		WithContext("key_id", keyID)
}

func NoActiveSigningKey(appID string) *errs.AuthsomeError {
	return errs.New(CodeNoActiveSigningKey, "No active signing key found for app", http.StatusNotFound).
		WithContext("app_id", appID)
}

// JWTGenerationFailed returns an error when JWT generation fails.
func JWTGenerationFailed(reason string) *errs.AuthsomeError {
	return errs.New(CodeJWTGenerationFailed, "Failed to generate JWT token", http.StatusInternalServerError).
		WithContext("reason", reason)
}

func JWTSigningFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeJWTSigningFailed, "Failed to sign JWT token", http.StatusInternalServerError)
}

// JWTVerificationFailed returns an error when JWT verification fails.
func JWTVerificationFailed(reason string) *errs.AuthsomeError {
	return errs.New(CodeJWTVerificationFailed, "JWT token verification failed", http.StatusUnauthorized).
		WithContext("reason", reason)
}

func JWTParsingFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeJWTParsingFailed, "Failed to parse JWT token", http.StatusBadRequest)
}

func MissingKIDHeader() *errs.AuthsomeError {
	return errs.New(CodeMissingKIDHeader, "JWT token missing kid header", http.StatusBadRequest)
}

func InvalidJWTAudience(expected, actual []string) *errs.AuthsomeError {
	return errs.New(CodeInvalidJWTAudience, "JWT token has invalid audience", http.StatusUnauthorized).
		WithContext("expected_audience", expected).
		WithContext("actual_audience", actual)
}

func InvalidJWTTokenType(expected, actual string) *errs.AuthsomeError {
	return errs.New(CodeInvalidJWTTokenType, "JWT token has invalid token type", http.StatusUnauthorized).
		WithContext("expected_type", expected).
		WithContext("actual_type", actual)
}

func InvalidJWTClaims(reason string) *errs.AuthsomeError {
	return errs.New(CodeInvalidJWTClaims, "JWT token has invalid claims", http.StatusBadRequest).
		WithContext("reason", reason)
}

// InvalidJWTAlgorithm returns an error when an invalid JWT algorithm is provided.
func InvalidJWTAlgorithm(algorithm string) *errs.AuthsomeError {
	return errs.New(CodeInvalidJWTAlgorithm, "Invalid JWT signing algorithm", http.StatusBadRequest).
		WithContext("algorithm", algorithm)
}

func InvalidJWTKeyType(keyType string) *errs.AuthsomeError {
	return errs.New(CodeInvalidJWTKeyType, "Invalid JWT key type", http.StatusBadRequest).
		WithContext("key_type", keyType)
}

// JWTKeyDecryptionFailed returns an error when JWT key decryption fails.
func JWTKeyDecryptionFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeJWTKeyDecryptionFailed, "Failed to decrypt JWT private key", http.StatusInternalServerError)
}

func JWTKeyEncryptionFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeJWTKeyEncryptionFailed, "Failed to encrypt JWT private key", http.StatusInternalServerError)
}

// JWTKeyGenerationFailed returns an error when JWT key generation fails.
func JWTKeyGenerationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeJWTKeyGenerationFailed, "Failed to generate JWT key pair", http.StatusInternalServerError)
}

func CannotSignWithoutPrivateKey() *errs.AuthsomeError {
	return errs.New(CodeCannotSignWithoutPrivate, "Cannot sign JWT without private key", http.StatusInternalServerError)
}

func CannotVerifyWithoutPublicKey() *errs.AuthsomeError {
	return errs.New(CodeCannotVerifyWithoutPublic, "Cannot verify JWT without public key", http.StatusInternalServerError)
}

// JWKSGenerationFailed returns an error when JWKS generation fails.
func JWKSGenerationFailed(err error) *errs.AuthsomeError {
	return errs.Wrap(err, CodeJWKSGenerationFailed, "Failed to generate JWKS", http.StatusInternalServerError)
}

// =============================================================================
// SENTINEL ERRORS (for use with errors.Is)
// =============================================================================

var (
	ErrJWTKeyNotFound            = &errs.AuthsomeError{Code: CodeJWTKeyNotFound}
	ErrJWTKeyAlreadyExists       = &errs.AuthsomeError{Code: CodeJWTKeyAlreadyExists}
	ErrJWTKeyExpired             = &errs.AuthsomeError{Code: CodeJWTKeyExpired}
	ErrJWTKeyInactive            = &errs.AuthsomeError{Code: CodeJWTKeyInactive}
	ErrNoActiveSigningKey        = &errs.AuthsomeError{Code: CodeNoActiveSigningKey}
	ErrJWTGenerationFailed       = &errs.AuthsomeError{Code: CodeJWTGenerationFailed}
	ErrJWTVerificationFailed     = &errs.AuthsomeError{Code: CodeJWTVerificationFailed}
	ErrInvalidJWTAlgorithm       = &errs.AuthsomeError{Code: CodeInvalidJWTAlgorithm}
	ErrInvalidJWTKeyType         = &errs.AuthsomeError{Code: CodeInvalidJWTKeyType}
	ErrJWTKeyDecryptionFailed    = &errs.AuthsomeError{Code: CodeJWTKeyDecryptionFailed}
	ErrJWTKeyEncryptionFailed    = &errs.AuthsomeError{Code: CodeJWTKeyEncryptionFailed}
	ErrJWTParsingFailed          = &errs.AuthsomeError{Code: CodeJWTParsingFailed}
	ErrJWTSigningFailed          = &errs.AuthsomeError{Code: CodeJWTSigningFailed}
	ErrMissingKIDHeader          = &errs.AuthsomeError{Code: CodeMissingKIDHeader}
	ErrInvalidJWTAudience        = &errs.AuthsomeError{Code: CodeInvalidJWTAudience}
	ErrInvalidJWTTokenType       = &errs.AuthsomeError{Code: CodeInvalidJWTTokenType}
	ErrJWKSGenerationFailed      = &errs.AuthsomeError{Code: CodeJWKSGenerationFailed}
	ErrInvalidJWTClaims          = &errs.AuthsomeError{Code: CodeInvalidJWTClaims}
	ErrJWTKeyGenerationFailed    = &errs.AuthsomeError{Code: CodeJWTKeyGenerationFailed}
	ErrCannotSignWithoutPrivate  = &errs.AuthsomeError{Code: CodeCannotSignWithoutPrivate}
	ErrCannotVerifyWithoutPublic = &errs.AuthsomeError{Code: CodeCannotVerifyWithoutPublic}
)
