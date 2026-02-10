package consent

import "errors"

var (
	// ErrConsentNotFound is returned when a consent record is not found.
	ErrConsentNotFound      = errors.New("consent record not found")
	ErrConsentAlreadyExists = errors.New("consent record already exists")
	ErrConsentExpired       = errors.New("consent has expired")
	ErrConsentRevoked       = errors.New("consent has been revoked")
	ErrInvalidConsentType   = errors.New("invalid consent type")
	ErrConsentRequired      = errors.New("consent is required")

	// ErrPolicyNotFound Errors.
	ErrPolicyNotFound       = errors.New("consent policy not found")
	ErrPolicyAlreadyExists  = errors.New("consent policy already exists")
	ErrPolicyInactive       = errors.New("consent policy is not active")
	ErrInvalidPolicyVersion = errors.New("invalid policy version")
	ErrPolicyRequired       = errors.New("policy acceptance is required")

	// ErrDPANotFound Errors.
	ErrDPANotFound      = errors.New("data processing agreement not found")
	ErrDPAExpired       = errors.New("data processing agreement has expired")
	ErrDPANotActive     = errors.New("data processing agreement is not active")
	ErrInvalidSignature = errors.New("invalid digital signature")

	// ErrCookieConsentNotFound Consent Errors.
	ErrCookieConsentNotFound    = errors.New("cookie consent not found")
	ErrInvalidCookiePreferences = errors.New("invalid cookie preferences")

	// ErrExportNotFound Export Errors.
	ErrExportNotFound       = errors.New("data export request not found")
	ErrExportAlreadyPending = errors.New("data export request already pending")
	ErrExportFailed         = errors.New("data export failed")
	ErrExportExpired        = errors.New("data export has expired")
	ErrInvalidExportFormat  = errors.New("invalid export format")

	// ErrDeletionNotFound Deletion Errors.
	ErrDeletionNotFound       = errors.New("data deletion request not found")
	ErrDeletionAlreadyPending = errors.New("data deletion request already pending")
	ErrDeletionFailed         = errors.New("data deletion failed")
	ErrDeletionNotApproved    = errors.New("data deletion request not approved")
	ErrRetentionExempt        = errors.New("data is exempt from deletion due to legal retention")

	// ErrPrivacySettingsNotFound Settings Errors.
	ErrPrivacySettingsNotFound = errors.New("privacy settings not found")
	ErrInvalidRetentionPeriod  = errors.New("invalid data retention period")

	// ErrUnauthorized Errors.
	ErrUnauthorized         = errors.New("unauthorized access")
	ErrInvalidRequest       = errors.New("invalid request")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrUserNotFound         = errors.New("user not found")
)
