package mfa

import "github.com/rs/xid"

// ==================== Factor Management Request/Response Types ====================

// EnrollFactorRequest represents the request to enroll a new MFA factor.
type EnrollFactorRequest struct {
	// Body fields
	Type     FactorType     `description:"Type of authentication factor to enroll" json:"type"               validate:"required,oneof=totp sms email webauthn push backup question biometric"`
	Priority FactorPriority `description:"Priority level of the factor"            json:"priority,omitempty" validate:"omitempty,oneof=primary backup optional"`
	Name     string         `description:"User-friendly name for the factor"       json:"name,omitempty"     validate:"omitempty,min=1,max=100"`
	Metadata map[string]any `description:"Additional factor-specific metadata"     json:"metadata,omitempty"`
}

// ListFactorsRequest represents the request to list factors.
type ListFactorsRequest struct {
	// Query parameters
	ActiveOnly bool `description:"Return only active factors" query:"activeOnly"`
}

// ListFactorsResponse represents the response containing factors list.
type ListFactorsResponse struct {
	Factors []Factor `description:"List of enrolled factors" json:"factors"`
	Count   int      `description:"Total number of factors"  json:"count"`
}

// GetFactorRequest represents the request to get a specific factor.
type GetFactorRequest struct {
	// Path parameters
	ID string `description:"Factor ID" path:"id" validate:"required"`
}

// UpdateFactorRequest represents the request to update a factor.
type UpdateFactorRequest struct {
	// Path parameters
	ID string `description:"Factor ID" path:"id" validate:"required"`

	// Body fields
	Name     *string         `description:"New name for the factor"                       json:"name,omitempty"     validate:"omitempty,min=1,max=100"`
	Priority *FactorPriority `description:"New priority level"                            json:"priority,omitempty" validate:"omitempty,oneof=primary backup optional"`
	Status   *FactorStatus   `description:"New status (cannot set to pending or revoked)" json:"status,omitempty"   validate:"omitempty,oneof=active disabled"`
	Metadata map[string]any  `description:"Updated metadata"                              json:"metadata,omitempty"`
}

// DeleteFactorRequest represents the request to delete a factor.
type DeleteFactorRequest struct {
	// Path parameters
	ID string `description:"Factor ID to delete" path:"id" validate:"required"`
}

// VerifyEnrolledFactorRequest represents the request to verify an enrolled factor.
type VerifyEnrolledFactorRequest struct {
	// Path parameters
	ID string `description:"Factor ID to verify" path:"id" validate:"required"`

	// Body fields
	Code string         `description:"Verification code for OTP-based factors"                json:"code,omitempty" validate:"required_without=Data"`
	Data map[string]any `description:"Verification data for complex factors (WebAuthn, etc.)" json:"data,omitempty"`
}

// ==================== Challenge & Verification Request/Response Types ====================

// InitiateChallengeRequest represents the request to start an MFA challenge.
type InitiateChallengeRequest struct {
	// Body fields
	FactorTypes []FactorType   `description:"Specific factor types to use for this challenge"      json:"factorTypes,omitempty"`
	Context     string         `description:"Authentication context (login, transaction, step-up)" json:"context,omitempty"     validate:"omitempty,oneof=login transaction step-up"`
	Metadata    map[string]any `description:"Additional context metadata"                          json:"metadata,omitempty"`
}

// VerifyChallengeRequest represents the request to verify an MFA challenge.
type VerifyChallengeRequest struct {
	// Body fields
	ChallengeID    xid.ID         `description:"ID of the challenge to verify"           json:"challengeId"              validate:"required"`
	FactorID       xid.ID         `description:"ID of the factor being used"             json:"factorId"                 validate:"required"`
	Code           string         `description:"Verification code for OTP-based factors" json:"code,omitempty"           validate:"required_without=Data"`
	Data           map[string]any `description:"Verification data for complex factors"   json:"data,omitempty"`
	RememberDevice bool           `description:"Whether to trust this device"            json:"rememberDevice,omitempty"`
	DeviceInfo     *DeviceInfo    `description:"Device identification information"       json:"deviceInfo,omitempty"`
}

// GetChallengeStatusRequest represents the request to get challenge status.
type GetChallengeStatusRequest struct {
	// Path parameters
	ID string `description:"Challenge ID" path:"id" validate:"required"`
}

// GetChallengeStatusResponse represents the challenge status response.
type GetChallengeStatusResponse struct {
	ChallengeID      xid.ID          `description:"Unique challenge identifier"          json:"challengeId"`
	Status           ChallengeStatus `description:"Current status of the challenge"      json:"status"`
	FactorsRequired  int             `description:"Number of factors required"           json:"factorsRequired"`
	FactorsVerified  int             `description:"Number of factors verified"           json:"factorsVerified"`
	Attempts         int             `description:"Number of verification attempts"      json:"attempts"`
	MaxAttempts      int             `description:"Maximum allowed attempts"             json:"maxAttempts"`
	AvailableFactors []FactorInfo    `description:"Available factors for this challenge" json:"availableFactors"`
}

// ==================== Trusted Devices Request/Response Types ====================

// TrustDeviceRequest represents the request to trust a device.
type TrustDeviceRequest struct {
	// Body fields
	DeviceID string         `description:"Unique device identifier"            json:"deviceId"           validate:"required"`
	Name     string         `description:"User-friendly device name"           json:"name,omitempty"     validate:"omitempty,min=1,max=100"`
	Metadata map[string]any `description:"Device metadata (OS, browser, etc.)" json:"metadata,omitempty"`
}

// ListTrustedDevicesResponse represents the response containing trusted devices.
type ListTrustedDevicesResponse struct {
	Devices []TrustedDevice `description:"List of trusted devices"         json:"devices"`
	Count   int             `description:"Total number of trusted devices" json:"count"`
}

// RevokeTrustedDeviceRequest represents the request to revoke a trusted device.
type RevokeTrustedDeviceRequest struct {
	// Path parameters
	ID string `description:"Trusted device ID to revoke" path:"id" validate:"required"`
}

// ==================== Status & Info Request/Response Types ====================

// GetStatusRequest represents the request to get MFA status.
type GetStatusRequest struct {
	// Query parameters
	DeviceID string `description:"Device ID to check trust status" query:"deviceId"`
}

// UpdatePolicyRequest represents the request to update MFA policy (admin only).
type UpdatePolicyRequest struct {
	// Body fields
	RequiredFactorCount    *int         `description:"Number of factors required"                              json:"requiredFactorCount,omitempty"    validate:"omitempty,min=0,max=5"`
	AllowedFactorTypes     []FactorType `description:"Permitted factor types"                                  json:"allowedFactorTypes,omitempty"`
	RequiredFactorTypes    []FactorType `description:"Mandatory factor types"                                  json:"requiredFactorTypes,omitempty"`
	GracePeriodDays        *int         `description:"Days before MFA is enforced"                             json:"gracePeriodDays,omitempty"        validate:"omitempty,min=0,max=365"`
	TrustedDeviceDays      *int         `description:"Days device remains trusted"                             json:"trustedDeviceDays,omitempty"      validate:"omitempty,min=1,max=365"`
	StepUpRequired         *bool        `description:"Require step-up authentication for sensitive operations" json:"stepUpRequired,omitempty"`
	AdaptiveMFAEnabled     *bool        `description:"Enable risk-based MFA"                                   json:"adaptiveMfaEnabled,omitempty"`
	MaxFailedAttempts      *int         `description:"Maximum failed verification attempts"                    json:"maxFailedAttempts,omitempty"      validate:"omitempty,min=1,max=10"`
	LockoutDurationMinutes *int         `description:"Account lockout duration in minutes"                     json:"lockoutDurationMinutes,omitempty" validate:"omitempty,min=1,max=1440"`
}

// ResetUserMFARequest represents the request to reset user's MFA (admin only).
type ResetUserMFARequest struct {
	// Path parameters
	ID string `description:"User ID whose MFA should be reset" path:"id" validate:"required"`

	// Body fields
	Reason string `description:"Reason for MFA reset (for audit trail)" json:"reason,omitempty" validate:"omitempty,min=1,max=500"`
}

// ResetUserMFAResponse represents the response after resetting user's MFA.
type ResetUserMFAResponse struct {
	Success        bool   `description:"Whether the reset was successful"  json:"success"`
	Message        string `description:"Human-readable message"            json:"message"`
	FactorsReset   int    `description:"Number of factors that were reset" json:"factorsReset"`
	DevicesRevoked int    `description:"Number of trusted devices revoked" json:"devicesRevoked"`
}

// ==================== Common Response Types ====================

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error   string         `description:"Error message"                        json:"error"`
	Code    string         `description:"Error code for programmatic handling" json:"code,omitempty"`
	Details map[string]any `description:"Additional error details"             json:"details,omitempty"`
}

// SuccessResponse represents a standard success response.
type SuccessResponse struct {
	Message string         `description:"Success message"          json:"message"`
	Data    map[string]any `description:"Additional response data" json:"data,omitempty"`
}
