package core

import (
	"fmt"
	"net/http"

	"github.com/xraph/authsome/internal/errs"
)

// Error codes for the secrets plugin.
const (
	ErrCodeSecretNotFound    = "SECRET_NOT_FOUND"
	ErrCodeSecretExists      = "SECRET_EXISTS"
	ErrCodeInvalidPath       = "INVALID_PATH"
	ErrCodeInvalidValueType  = "INVALID_VALUE_TYPE"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeSchemaInvalid     = "SCHEMA_INVALID"
	ErrCodeDecryptionFailed  = "DECRYPTION_FAILED"
	ErrCodeEncryptionFailed  = "ENCRYPTION_FAILED"
	ErrCodeMasterKeyRequired = "MASTER_KEY_REQUIRED"
	ErrCodeMasterKeyInvalid  = "MASTER_KEY_INVALID"
	ErrCodeSecretExpired     = "SECRET_EXPIRED"
	ErrCodeVersionNotFound   = "VERSION_NOT_FOUND"
	ErrCodeRollbackFailed    = "ROLLBACK_FAILED"
	ErrCodeAccessDenied      = "ACCESS_DENIED"
	ErrCodeInvalidRequest    = "INVALID_REQUEST"
)

// =============================================================================
// Error Constructors
// =============================================================================

// ErrSecretNotFound returns a not found error for a secret.
func ErrSecretNotFound(identifier string) error {
	return errs.NotFound("secret not found: " + identifier)
}

// ErrSecretNotFoundByPath returns a not found error for a secret by path.
func ErrSecretNotFoundByPath(path string) error {
	return errs.NotFound("secret not found at path: " + path)
}

// ErrSecretExists returns a conflict error when a secret already exists.
func ErrSecretExists(path string) error {
	return errs.Conflict("secret already exists at path: " + path)
}

// ErrInvalidPath returns a bad request error for invalid path format.
func ErrInvalidPath(path string, reason string) error {
	msg := "invalid secret path: " + path
	if reason != "" {
		msg += " (" + reason + ")"
	}

	return errs.BadRequest(msg)
}

// ErrInvalidValueType returns a bad request error for invalid value type.
func ErrInvalidValueType(valueType string) error {
	return errs.BadRequest("invalid value type: " + valueType + "; must be one of: plain, json, yaml, binary")
}

// ErrValidationFailed returns a bad request error when value validation fails.
func ErrValidationFailed(reason string, cause error) error {
	msg := "value validation failed: " + reason
	if cause != nil {
		msg += ": " + cause.Error()
	}

	return errs.BadRequest(msg)
}

// ErrSchemaInvalid returns a bad request error when the JSON schema is invalid.
func ErrSchemaInvalid(reason string, cause error) error {
	msg := "invalid JSON schema: " + reason
	if cause != nil {
		msg += ": " + cause.Error()
	}

	return errs.BadRequest(msg)
}

// ErrDecryptionFailed returns an internal error when decryption fails.
func ErrDecryptionFailed(cause error) error {
	return errs.InternalServerError("failed to decrypt secret value", cause)
}

// ErrEncryptionFailed returns an internal error when encryption fails.
func ErrEncryptionFailed(cause error) error {
	return errs.InternalServerError("failed to encrypt secret value", cause)
}

// ErrMasterKeyRequired returns an internal error when the master key is not configured.
func ErrMasterKeyRequired() error {
	return errs.InternalServerErrorWithMessage("secrets master key is not configured; set AUTHSOME_SECRETS_MASTER_KEY environment variable")
}

// ErrMasterKeyInvalid returns an internal error when the master key format is invalid.
func ErrMasterKeyInvalid(reason string) error {
	return errs.InternalServerErrorWithMessage("invalid master key: " + reason)
}

// ErrSecretExpired returns a gone error when the secret has expired.
func ErrSecretExpired(path string) error {
	return errs.New(ErrCodeSecretExpired, "secret has expired: "+path, http.StatusGone)
}

// ErrVersionNotFound returns a not found error for a specific version.
func ErrVersionNotFound(secretID string, version int) error {
	return errs.NotFound(fmt.Sprintf("version %d not found for secret %s", version, secretID))
}

// ErrRollbackFailed returns an internal error when rollback fails.
func ErrRollbackFailed(reason string, cause error) error {
	return errs.InternalServerError("rollback failed: "+reason, cause)
}

// ErrAccessDenied returns a forbidden error when access is denied.
func ErrAccessDenied(reason string) error {
	return errs.New(ErrCodeAccessDenied, "access denied: "+reason, http.StatusForbidden)
}

// ErrInvalidRequest returns a bad request error for generic invalid requests.
func ErrInvalidRequest(reason string, cause error) error {
	msg := "invalid request: " + reason
	if cause != nil {
		msg += ": " + cause.Error()
	}

	return errs.BadRequest(msg)
}

// ErrAppContextRequired returns a bad request error when app context is missing.
func ErrAppContextRequired() error {
	return errs.BadRequest("app context is required for this operation")
}

// ErrEnvironmentContextRequired returns a bad request error when environment context is missing.
func ErrEnvironmentContextRequired() error {
	return errs.BadRequest("environment context is required for this operation")
}

// ErrValueRequired returns a bad request error when value is missing.
func ErrValueRequired() error {
	return errs.BadRequest("secret value is required")
}

// ErrPathRequired returns a bad request error when path is missing.
func ErrPathRequired() error {
	return errs.BadRequest("secret path is required")
}

// ErrSerializationFailed returns an internal error when serialization fails.
func ErrSerializationFailed(valueType string, cause error) error {
	return errs.InternalServerError("failed to serialize "+valueType+" value", cause)
}

// ErrDeserializationFailed returns an internal error when deserialization fails.
func ErrDeserializationFailed(valueType string, cause error) error {
	return errs.InternalServerError("failed to deserialize "+valueType+" value", cause)
}
