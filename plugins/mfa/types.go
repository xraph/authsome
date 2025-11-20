package mfa

import (
	"time"

	"github.com/rs/xid"
)

// FactorType represents different authentication factor types
type FactorType string

const (
	FactorTypeTOTP      FactorType = "totp"      // Time-based One-Time Password (Google Authenticator)
	FactorTypeSMS       FactorType = "sms"       // SMS verification code
	FactorTypeEmail     FactorType = "email"     // Email verification code
	FactorTypeWebAuthn  FactorType = "webauthn"  // FIDO2/WebAuthn (security keys, biometrics)
	FactorTypePush      FactorType = "push"      // Push notification approval
	FactorTypeBackup    FactorType = "backup"    // Backup recovery codes
	FactorTypeQuestion  FactorType = "question"  // Security questions
	FactorTypeBiometric FactorType = "biometric" // Biometric authentication
)

// FactorStatus represents the state of an authentication factor
type FactorStatus string

const (
	FactorStatusPending  FactorStatus = "pending"  // Enrolled but not verified
	FactorStatusActive   FactorStatus = "active"   // Verified and active
	FactorStatusDisabled FactorStatus = "disabled" // Temporarily disabled
	FactorStatusRevoked  FactorStatus = "revoked"  // Permanently revoked
)

// FactorPriority defines the priority of a factor
type FactorPriority string

const (
	FactorPriorityPrimary  FactorPriority = "primary"  // Primary authentication factor
	FactorPriorityBackup   FactorPriority = "backup"   // Backup/fallback factor
	FactorPriorityOptional FactorPriority = "optional" // Optional additional security
)

// RiskLevel represents authentication risk assessment
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// ChallengeStatus represents the state of an MFA challenge
type ChallengeStatus string

const (
	ChallengeStatusPending   ChallengeStatus = "pending"
	ChallengeStatusVerified  ChallengeStatus = "verified"
	ChallengeStatusFailed    ChallengeStatus = "failed"
	ChallengeStatusExpired   ChallengeStatus = "expired"
	ChallengeStatusCancelled ChallengeStatus = "cancelled"
)

// Factor represents an enrolled authentication factor
type Factor struct {
	ID         xid.ID         `json:"id"`
	UserID     xid.ID         `json:"userId"`
	Type       FactorType     `json:"type"`
	Status     FactorStatus   `json:"status"`
	Priority   FactorPriority `json:"priority"`
	Name       string         `json:"name"`     // User-friendly name
	Secret     string         `json:"-"`        // Encrypted secret data
	Metadata   map[string]any `json:"metadata"` // Factor-specific metadata
	LastUsedAt *time.Time     `json:"lastUsedAt"`
	VerifiedAt *time.Time     `json:"verifiedAt"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	ExpiresAt  *time.Time     `json:"expiresAt,omitempty"`
}

// Challenge represents an active MFA challenge
type Challenge struct {
	ID          xid.ID          `json:"id"`
	UserID      xid.ID          `json:"userId"`
	FactorID    xid.ID          `json:"factorId"`
	Type        FactorType      `json:"type"`
	Status      ChallengeStatus `json:"status"`
	Code        string          `json:"-"` // Hashed verification code
	Metadata    map[string]any  `json:"metadata"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"maxAttempts"`
	IPAddress   string          `json:"ipAddress"`
	UserAgent   string          `json:"userAgent"`
	CreatedAt   time.Time       `json:"createdAt"`
	ExpiresAt   time.Time       `json:"expiresAt"`
	VerifiedAt  *time.Time      `json:"verifiedAt,omitempty"`
}

// TrustedDevice represents a device that can skip MFA
type TrustedDevice struct {
	ID         xid.ID         `json:"id"`
	UserID     xid.ID         `json:"userId"`
	DeviceID   string         `json:"deviceId"` // Fingerprint/identifier
	Name       string         `json:"name"`     // User-friendly name
	Metadata   map[string]any `json:"metadata"` // Device info
	IPAddress  string         `json:"ipAddress"`
	UserAgent  string         `json:"userAgent"`
	LastUsedAt *time.Time     `json:"lastUsedAt"`
	CreatedAt  time.Time      `json:"createdAt"`
	ExpiresAt  time.Time      `json:"expiresAt"`
}

// MFASession represents an MFA verification session
type MFASession struct {
	ID              xid.ID         `json:"id"`
	UserID          xid.ID         `json:"userId"`
	SessionToken    string         `json:"sessionToken"`
	FactorsRequired int            `json:"factorsRequired"`
	FactorsVerified int            `json:"factorsVerified"`
	VerifiedFactors []xid.ID       `json:"verifiedFactors"`
	RiskLevel       RiskLevel      `json:"riskLevel"`
	IPAddress       string         `json:"ipAddress"`
	UserAgent       string         `json:"userAgent"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"createdAt"`
	ExpiresAt       time.Time      `json:"expiresAt"`
	CompletedAt     *time.Time     `json:"completedAt,omitempty"`
}

// MFAPolicy defines organization-level MFA requirements
type MFAPolicy struct {
	ID                     xid.ID       `json:"id"`
	OrganizationID         xid.ID       `json:"organizationId"`
	RequiredFactorCount    int          `json:"requiredFactorCount"` // Number of factors required
	AllowedFactorTypes     []FactorType `json:"allowedFactorTypes"`  // Permitted factor types
	RequiredFactorTypes    []FactorType `json:"requiredFactorTypes"` // Mandatory factor types
	GracePeriodDays        int          `json:"gracePeriodDays"`     // Days before MFA is enforced
	TrustedDeviceDays      int          `json:"trustedDeviceDays"`   // Days device is trusted
	StepUpRequired         bool         `json:"stepUpRequired"`      // Require step-up for sensitive ops
	AdaptiveMFAEnabled     bool         `json:"adaptiveMfaEnabled"`  // Enable risk-based MFA
	MaxFailedAttempts      int          `json:"maxFailedAttempts"`
	LockoutDurationMinutes int          `json:"lockoutDurationMinutes"`
	CreatedAt              time.Time    `json:"createdAt"`
	UpdatedAt              time.Time    `json:"updatedAt"`
}

// FactorEnrollmentRequest represents a request to enroll a new factor
type FactorEnrollmentRequest struct {
	Type     FactorType     `json:"type"`
	Priority FactorPriority `json:"priority,omitempty"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// FactorEnrollmentResponse contains data needed to complete enrollment
type FactorEnrollmentResponse struct {
	FactorID         xid.ID         `json:"factorId"`
	Type             FactorType     `json:"type"`
	Status           FactorStatus   `json:"status"`
	ProvisioningData map[string]any `json:"provisioningData"` // Type-specific setup data
	// For TOTP: { "secret": "...", "qr_code": "...", "uri": "..." }
	// For WebAuthn: { "challenge": "...", "rp": {...}, "user": {...} }
	// For SMS/Email: { "masked_destination": "+1***-***-1234" }
}

// FactorVerificationRequest verifies an enrolled factor
type FactorVerificationRequest struct {
	FactorID xid.ID         `json:"factorId"`
	Code     string         `json:"code,omitempty"` // For OTP-based factors
	Data     map[string]any `json:"data,omitempty"` // For complex factors (WebAuthn, etc.)
}

// ChallengeRequest initiates an MFA challenge
type ChallengeRequest struct {
	UserID      xid.ID         `json:"userId"`
	FactorTypes []FactorType   `json:"factorTypes,omitempty"` // Specific factor types to use
	Context     string         `json:"context,omitempty"`     // "login", "transaction", "step-up"
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ChallengeResponse contains challenge details
type ChallengeResponse struct {
	ChallengeID      xid.ID       `json:"challengeId"`
	SessionID        xid.ID       `json:"sessionId"`
	FactorsRequired  int          `json:"factorsRequired"`
	AvailableFactors []FactorInfo `json:"availableFactors"`
	ExpiresAt        time.Time    `json:"expiresAt"`
}

// FactorInfo provides minimal factor information for challenge selection
type FactorInfo struct {
	FactorID xid.ID         `json:"factorId"`
	Type     FactorType     `json:"type"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata,omitempty"` // Masked phone, email, etc.
}

// VerificationRequest verifies a challenge
type VerificationRequest struct {
	ChallengeID    xid.ID         `json:"challengeId"`
	FactorID       xid.ID         `json:"factorId"`
	Code           string         `json:"code,omitempty"`
	Data           map[string]any `json:"data,omitempty"`
	RememberDevice bool           `json:"rememberDevice,omitempty"`
	DeviceInfo     *DeviceInfo    `json:"deviceInfo,omitempty"`
}

// DeviceInfo contains device identification data
type DeviceInfo struct {
	DeviceID string         `json:"deviceId"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// VerificationResponse indicates verification result
type VerificationResponse struct {
	Success          bool       `json:"success"`
	SessionComplete  bool       `json:"sessionComplete"`
	FactorsRemaining int        `json:"factorsRemaining,omitempty"`
	Token            string     `json:"token,omitempty"` // MFA completion token
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
}

// ChallengeStatusResponse contains the current status of an MFA challenge
type ChallengeStatusResponse struct {
	SessionID        xid.ID     `json:"sessionId"`
	Status           string     `json:"status"` // pending, completed, expired
	FactorsRequired  int        `json:"factorsRequired"`
	FactorsVerified  int        `json:"factorsVerified"`
	FactorsRemaining int        `json:"factorsRemaining"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// MFAPolicyResponse contains MFA policy details
type MFAPolicyResponse struct {
	ID                  xid.ID   `json:"id"`
	AppID               xid.ID   `json:"appId"`
	OrganizationID      *xid.ID  `json:"organizationId,omitempty"`
	Enabled             bool     `json:"enabled"`
	RequiredFactorCount int      `json:"requiredFactorCount"`
	AllowedFactorTypes  []string `json:"allowedFactorTypes"`
	GracePeriodDays     int      `json:"gracePeriodDays"`
}

// MFABypassResponse contains MFA bypass details
type MFABypassResponse struct {
	ID        xid.ID    `json:"id"`
	UserID    xid.ID    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
	Reason    string    `json:"reason"`
}

// MFAStatus represents overall MFA status for a user
type MFAStatus struct {
	Enabled         bool         `json:"enabled"`
	EnrolledFactors []FactorInfo `json:"enrolledFactors"`
	RequiredCount   int          `json:"requiredCount"`
	PolicyActive    bool         `json:"policyActive"`
	GracePeriod     *time.Time   `json:"gracePeriod,omitempty"`
	TrustedDevice   bool         `json:"trustedDevice"`
}

// RiskAssessment represents authentication risk evaluation
type RiskAssessment struct {
	Level       RiskLevel      `json:"level"`
	Score       float64        `json:"score"`       // 0-100
	Factors     []string       `json:"factors"`     // Risk factors identified
	Recommended []FactorType   `json:"recommended"` // Recommended factor types
	Metadata    map[string]any `json:"metadata"`
}
