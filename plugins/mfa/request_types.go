package mfa

import "github.com/rs/xid"

// ==================== Factor Management Request/Response Types ====================

// EnrollFactorRequest represents the request to enroll a new MFA factor
type EnrollFactorRequest struct {
	// Body fields
	Type     FactorType     `json:"type" validate:"required,oneof=totp sms email webauthn push backup question biometric" description:"Type of authentication factor to enroll"`
	Priority FactorPriority `json:"priority,omitempty" validate:"omitempty,oneof=primary backup optional" description:"Priority level of the factor"`
	Name     string         `json:"name,omitempty" validate:"omitempty,min=1,max=100" description:"User-friendly name for the factor"`
	Metadata map[string]any `json:"metadata,omitempty" description:"Additional factor-specific metadata"`
}

// ListFactorsRequest represents the request to list factors
type ListFactorsRequest struct {
	// Query parameters
	ActiveOnly bool `query:"activeOnly" description:"Return only active factors"`
}

// ListFactorsResponse represents the response containing factors list
type ListFactorsResponse struct {
	Factors []Factor `json:"factors" description:"List of enrolled factors"`
	Count   int      `json:"count" description:"Total number of factors"`
}

// GetFactorRequest represents the request to get a specific factor
type GetFactorRequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"Factor ID"`
}

// UpdateFactorRequest represents the request to update a factor
type UpdateFactorRequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"Factor ID"`
	
	// Body fields
	Name     *string         `json:"name,omitempty" validate:"omitempty,min=1,max=100" description:"New name for the factor"`
	Priority *FactorPriority `json:"priority,omitempty" validate:"omitempty,oneof=primary backup optional" description:"New priority level"`
	Status   *FactorStatus   `json:"status,omitempty" validate:"omitempty,oneof=active disabled" description:"New status (cannot set to pending or revoked)"`
	Metadata map[string]any  `json:"metadata,omitempty" description:"Updated metadata"`
}

// DeleteFactorRequest represents the request to delete a factor
type DeleteFactorRequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"Factor ID to delete"`
}

// VerifyEnrolledFactorRequest represents the request to verify an enrolled factor
type VerifyEnrolledFactorRequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"Factor ID to verify"`
	
	// Body fields
	Code string         `json:"code,omitempty" validate:"required_without=Data" description:"Verification code for OTP-based factors"`
	Data map[string]any `json:"data,omitempty" description:"Verification data for complex factors (WebAuthn, etc.)"`
}

// ==================== Challenge & Verification Request/Response Types ====================

// InitiateChallengeRequest represents the request to start an MFA challenge
type InitiateChallengeRequest struct {
	// Body fields
	FactorTypes []FactorType   `json:"factorTypes,omitempty" description:"Specific factor types to use for this challenge"`
	Context     string         `json:"context,omitempty" validate:"omitempty,oneof=login transaction step-up" description:"Authentication context (login, transaction, step-up)"`
	Metadata    map[string]any `json:"metadata,omitempty" description:"Additional context metadata"`
}

// VerifyChallengeRequest represents the request to verify an MFA challenge
type VerifyChallengeRequest struct {
	// Body fields
	ChallengeID    xid.ID         `json:"challengeId" validate:"required" description:"ID of the challenge to verify"`
	FactorID       xid.ID         `json:"factorId" validate:"required" description:"ID of the factor being used"`
	Code           string         `json:"code,omitempty" validate:"required_without=Data" description:"Verification code for OTP-based factors"`
	Data           map[string]any `json:"data,omitempty" description:"Verification data for complex factors"`
	RememberDevice bool           `json:"rememberDevice,omitempty" description:"Whether to trust this device"`
	DeviceInfo     *DeviceInfo    `json:"deviceInfo,omitempty" description:"Device identification information"`
}

// GetChallengeStatusRequest represents the request to get challenge status
type GetChallengeStatusRequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"Challenge ID"`
}

// GetChallengeStatusResponse represents the challenge status response
type GetChallengeStatusResponse struct {
	ChallengeID      xid.ID          `json:"challengeId" description:"Unique challenge identifier"`
	Status           ChallengeStatus `json:"status" description:"Current status of the challenge"`
	FactorsRequired  int             `json:"factorsRequired" description:"Number of factors required"`
	FactorsVerified  int             `json:"factorsVerified" description:"Number of factors verified"`
	Attempts         int             `json:"attempts" description:"Number of verification attempts"`
	MaxAttempts      int             `json:"maxAttempts" description:"Maximum allowed attempts"`
	AvailableFactors []FactorInfo    `json:"availableFactors" description:"Available factors for this challenge"`
}

// ==================== Trusted Devices Request/Response Types ====================

// TrustDeviceRequest represents the request to trust a device
type TrustDeviceRequest struct {
	// Body fields
	DeviceID string         `json:"deviceId" validate:"required" description:"Unique device identifier"`
	Name     string         `json:"name,omitempty" validate:"omitempty,min=1,max=100" description:"User-friendly device name"`
	Metadata map[string]any `json:"metadata,omitempty" description:"Device metadata (OS, browser, etc.)"`
}

// ListTrustedDevicesResponse represents the response containing trusted devices
type ListTrustedDevicesResponse struct {
	Devices []TrustedDevice `json:"devices" description:"List of trusted devices"`
	Count   int             `json:"count" description:"Total number of trusted devices"`
}

// RevokeTrustedDeviceRequest represents the request to revoke a trusted device
type RevokeTrustedDeviceRequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"Trusted device ID to revoke"`
}

// ==================== Status & Info Request/Response Types ====================

// GetStatusRequest represents the request to get MFA status
type GetStatusRequest struct {
	// Query parameters
	DeviceID string `query:"deviceId" description:"Device ID to check trust status"`
}

// UpdatePolicyRequest represents the request to update MFA policy (admin only)
type UpdatePolicyRequest struct {
	// Body fields
	RequiredFactorCount    *int          `json:"requiredFactorCount,omitempty" validate:"omitempty,min=0,max=5" description:"Number of factors required"`
	AllowedFactorTypes     []FactorType  `json:"allowedFactorTypes,omitempty" description:"Permitted factor types"`
	RequiredFactorTypes    []FactorType  `json:"requiredFactorTypes,omitempty" description:"Mandatory factor types"`
	GracePeriodDays        *int          `json:"gracePeriodDays,omitempty" validate:"omitempty,min=0,max=365" description:"Days before MFA is enforced"`
	TrustedDeviceDays      *int          `json:"trustedDeviceDays,omitempty" validate:"omitempty,min=1,max=365" description:"Days device remains trusted"`
	StepUpRequired         *bool         `json:"stepUpRequired,omitempty" description:"Require step-up authentication for sensitive operations"`
	AdaptiveMFAEnabled     *bool         `json:"adaptiveMfaEnabled,omitempty" description:"Enable risk-based MFA"`
	MaxFailedAttempts      *int          `json:"maxFailedAttempts,omitempty" validate:"omitempty,min=1,max=10" description:"Maximum failed verification attempts"`
	LockoutDurationMinutes *int          `json:"lockoutDurationMinutes,omitempty" validate:"omitempty,min=1,max=1440" description:"Account lockout duration in minutes"`
}

// ResetUserMFARequest represents the request to reset user's MFA (admin only)
type ResetUserMFARequest struct {
	// Path parameters
	ID string `path:"id" validate:"required" description:"User ID whose MFA should be reset"`
	
	// Body fields
	Reason string `json:"reason,omitempty" validate:"omitempty,min=1,max=500" description:"Reason for MFA reset (for audit trail)"`
}

// ResetUserMFAResponse represents the response after resetting user's MFA
type ResetUserMFAResponse struct {
	Success        bool   `json:"success" description:"Whether the reset was successful"`
	Message        string `json:"message" description:"Human-readable message"`
	FactorsReset   int    `json:"factorsReset" description:"Number of factors that were reset"`
	DevicesRevoked int    `json:"devicesRevoked" description:"Number of trusted devices revoked"`
}

// ==================== Common Response Types ====================

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string         `json:"error" description:"Error message"`
	Code    string         `json:"code,omitempty" description:"Error code for programmatic handling"`
	Details map[string]any `json:"details,omitempty" description:"Additional error details"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string         `json:"message" description:"Success message"`
	Data    map[string]any `json:"data,omitempty" description:"Additional response data"`
}

