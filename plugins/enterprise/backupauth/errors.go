package backupauth

import "errors"

// Common errors.
var (
	// ErrRecoverySessionNotFound is returned when a recovery session is not found.
	ErrRecoverySessionNotFound      = errors.New("recovery session not found")
	ErrRecoverySessionExpired       = errors.New("recovery session expired")
	ErrRecoverySessionCancelled     = errors.New("recovery session cancelled")
	ErrRecoverySessionInProgress    = errors.New("recovery session already in progress")
	ErrRecoverySessionCompleted     = errors.New("recovery session already completed")
	ErrRecoverySessionLocked        = errors.New("recovery session locked due to too many attempts")
	ErrRecoveryMethodNotEnabled     = errors.New("recovery method not enabled")
	ErrRecoveryStepRequired         = errors.New("recovery step required")
	ErrRecoveryStepAlreadyCompleted = errors.New("recovery step already completed")

	// ErrInvalidRecoveryCode codes errors.
	ErrInvalidRecoveryCode      = errors.New("invalid recovery code")
	ErrRecoveryCodeAlreadyUsed  = errors.New("recovery code already used")
	ErrRecoveryCodeExpired      = errors.New("recovery code expired")
	ErrNoRecoveryCodesAvailable = errors.New("no recovery codes available")

	// ErrSecurityQuestionNotFound is returned when a security question is not found.
	ErrSecurityQuestionNotFound      = errors.New("security question not found")
	ErrInvalidSecurityAnswer         = errors.New("invalid security answer")
	ErrSecurityQuestionAlreadyExists = errors.New("security question already exists")
	ErrInsufficientSecurityQuestions = errors.New("insufficient security questions configured")
	ErrSecurityQuestionLocked        = errors.New("security question locked due to failed attempts")
	ErrCommonAnswer                  = errors.New("answer is too common, please choose a more unique answer")
	ErrAnswerTooShort                = errors.New("answer is too short")
	ErrAnswerTooLong                 = errors.New("answer is too long")

	// ErrTrustedContactNotFound is returned when a trusted contact is not found.
	ErrTrustedContactNotFound           = errors.New("trusted contact not found")
	ErrTrustedContactNotVerified        = errors.New("trusted contact not verified")
	ErrTrustedContactAlreadyExists      = errors.New("trusted contact already exists")
	ErrInsufficientTrustedContacts      = errors.New("insufficient trusted contacts configured")
	ErrTrustedContactLimitExceeded      = errors.New("trusted contact limit exceeded")
	ErrTrustedContactCooldown           = errors.New("trusted contact notification cooldown active")
	ErrTrustedContactNotificationFailed = errors.New("failed to notify trusted contact")

	// ErrInvalidVerificationCode Email/SMS verification errors.
	ErrInvalidVerificationCode         = errors.New("invalid verification code")
	ErrVerificationCodeExpired         = errors.New("verification code expired")
	ErrVerificationCodeAlreadyUsed     = errors.New("verification code already used")
	ErrMaxVerificationAttemptsExceeded = errors.New("maximum verification attempts exceeded")
	ErrEmailNotVerified                = errors.New("email not verified")
	ErrPhoneNotVerified                = errors.New("phone not verified")

	// ErrVideoSessionNotFound is returned when a video session is not found.
	ErrVideoSessionNotFound     = errors.New("video session not found")
	ErrVideoSessionNotScheduled = errors.New("video session not scheduled")
	ErrVideoSessionExpired      = errors.New("video session expired")
	ErrLivenessCheckFailed      = errors.New("liveness check failed")
	ErrVideoVerificationFailed  = errors.New("video verification failed")
	ErrVideoVerificationPending = errors.New("video verification pending review")

	// ErrDocumentVerificationNotFound is returned when a document verification is not found.
	ErrDocumentVerificationNotFound = errors.New("document verification not found")
	ErrInvalidDocumentType          = errors.New("invalid document type")
	ErrDocumentVerificationFailed   = errors.New("document verification failed")
	ErrDocumentVerificationPending  = errors.New("document verification pending review")
	ErrDocumentExpired              = errors.New("document expired")
	ErrDocumentImageRequired        = errors.New("document image required")
	ErrSelfieRequired               = errors.New("selfie required")
	ErrConfidenceScoreTooLow        = errors.New("confidence score too low")

	// ErrRateLimitExceeded is returned when rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrTooManyAttempts   = errors.New("too many recovery attempts")
	ErrAccountLocked     = errors.New("account locked due to too many recovery attempts")
	ErrCooldownActive    = errors.New("cooldown period active, please wait before retrying")

	// ErrHighRiskDetected is returned when high risk is detected.
	ErrHighRiskDetected    = errors.New("high risk detected, additional verification required")
	ErrRiskScoreTooHigh    = errors.New("risk score too high, recovery blocked")
	ErrAdminReviewRequired = errors.New("admin review required for recovery")

	// ErrRecoveryNotConfigured is returned when recovery is not configured.
	ErrRecoveryNotConfigured = errors.New("backup recovery not configured")
	ErrInvalidConfiguration  = errors.New("invalid configuration")
	ErrProviderNotConfigured = errors.New("provider not configured")

	// ErrUnauthorized is returned when the user is unauthorized.
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidSession   = errors.New("invalid session")
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidInput errors.
	ErrInvalidInput         = errors.New("invalid input")
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidEmail         = errors.New("invalid email")
	ErrInvalidPhone         = errors.New("invalid phone")

	// ErrProviderError is returned when a provider error occurs.
	ErrProviderError       = errors.New("provider error")
	ErrProviderTimeout     = errors.New("provider timeout")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrProviderAuthFailed  = errors.New("provider authentication failed")

	// ErrStorageError is returned when a storage error occurs.
	ErrStorageError     = errors.New("storage error")
	ErrFileUploadFailed = errors.New("file upload failed")
	ErrFileNotFound     = errors.New("file not found")
	ErrEncryptionFailed = errors.New("encryption failed")
	ErrDecryptionFailed = errors.New("decryption failed")
)
