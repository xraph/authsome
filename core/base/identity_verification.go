package base

import (
	"time"

	"github.com/xraph/authsome/schema"
)

// =============================================================================
// IDENTITY VERIFICATION DTOs (Data Transfer Objects)
// =============================================================================

// IdentityVerificationSession represents a verification session DTO.
type IdentityVerificationSession struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// V2 Multi-tenant context
	AppID          string  `json:"appId"`
	EnvironmentID  *string `json:"environmentId,omitempty"`
	OrganizationID string  `json:"organizationId"`
	UserID         string  `json:"userId"`

	// Session details
	Provider     string `json:"provider"`
	SessionURL   string `json:"sessionUrl"`
	SessionToken string `json:"sessionToken,omitempty"` // Excluded in most responses

	// Configuration
	RequiredChecks []string       `json:"requiredChecks"`
	Config         map[string]any `json:"config,omitempty"`

	// Status tracking
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	ExpiresAt   time.Time  `json:"expiresAt"`

	// Callback URLs
	SuccessURL string `json:"successUrl,omitempty"`
	CancelURL  string `json:"cancelUrl,omitempty"`

	// Tracking
	IPAddress string `json:"ipAddress,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

// IdentityVerification represents a verification attempt DTO.
type IdentityVerification struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// V2 Multi-tenant context
	AppID          string  `json:"appId"`
	EnvironmentID  *string `json:"environmentId,omitempty"`
	OrganizationID string  `json:"organizationId"`
	UserID         string  `json:"userId"`

	// Provider information
	Provider        string `json:"provider"`
	ProviderCheckID string `json:"providerCheckId,omitempty"`

	// Verification type and status
	VerificationType string `json:"verificationType"`
	Status           string `json:"status"`

	// Document information
	DocumentType    string `json:"documentType,omitempty"`
	DocumentNumber  string `json:"documentNumber,omitempty"`
	DocumentCountry string `json:"documentCountry,omitempty"`

	// Verification results
	IsVerified      bool   `json:"isVerified"`
	RiskScore       int    `json:"riskScore,omitempty"`
	RiskLevel       string `json:"riskLevel,omitempty"`
	ConfidenceScore int    `json:"confidenceScore,omitempty"`

	// Personal information extracted
	FirstName   string     `json:"firstName,omitempty"`
	LastName    string     `json:"lastName,omitempty"`
	DateOfBirth *time.Time `json:"dateOfBirth,omitempty"`
	Age         int        `json:"age,omitempty"`
	Gender      string     `json:"gender,omitempty"`
	Nationality string     `json:"nationality,omitempty"`

	// AML/Sanctions screening results
	IsOnSanctionsList bool   `json:"isOnSanctionsList"`
	IsPEP             bool   `json:"isPep"`
	SanctionsDetails  string `json:"sanctionsDetails,omitempty"`

	// Liveness detection
	LivenessScore int  `json:"livenessScore,omitempty"`
	IsLive        bool `json:"isLive"`

	// Rejection/failure information
	RejectionReasons []string `json:"rejectionReasons,omitempty"`
	FailureReason    string   `json:"failureReason,omitempty"`

	// Metadata
	Metadata     map[string]any `json:"metadata,omitempty"`
	ProviderData map[string]any `json:"providerData,omitempty"`
	IPAddress    string         `json:"ipAddress,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`

	// Expiry and validity
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`

	// Webhook tracking
	WebhookDeliveryStatus string     `json:"webhookDeliveryStatus,omitempty"`
	WebhookDeliveredAt    *time.Time `json:"webhookDeliveredAt,omitempty"`
}

// UserVerificationStatus tracks the overall verification status DTO.
type UserVerificationStatus struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// V2 Multi-tenant context
	AppID          string  `json:"appId"`
	EnvironmentID  *string `json:"environmentId,omitempty"`
	OrganizationID string  `json:"organizationId"`
	UserID         string  `json:"userId"`

	// Overall verification status
	IsVerified             bool       `json:"isVerified"`
	VerificationLevel      string     `json:"verificationLevel"`
	LastVerifiedAt         *time.Time `json:"lastVerifiedAt,omitempty"`
	VerificationExpiry     *time.Time `json:"verificationExpiry,omitempty"`
	RequiresReverification bool       `json:"requiresReverification"`

	// Individual check statuses
	DocumentVerified bool `json:"documentVerified"`
	LivenessVerified bool `json:"livenessVerified"`
	AgeVerified      bool `json:"ageVerified"`
	AMLScreened      bool `json:"amlScreened"`
	AMLClear         bool `json:"amlClear"`

	// Most recent verification IDs
	LastDocumentVerificationID string `json:"lastDocumentVerificationId,omitempty"`
	LastLivenessVerificationID string `json:"lastLivenessVerificationId,omitempty"`
	LastAMLVerificationID      string `json:"lastAMLVerificationId,omitempty"`

	// Risk assessment
	OverallRiskLevel string   `json:"overallRiskLevel"`
	RiskFactors      []string `json:"riskFactors,omitempty"`

	// Compliance flags
	IsBlocked   bool       `json:"isBlocked"`
	BlockReason string     `json:"blockReason,omitempty"`
	BlockedAt   *time.Time `json:"blockedAt,omitempty"`

	// Metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// =============================================================================
// CONVERSION FUNCTIONS
// =============================================================================

// FromSchemaIdentityVerificationSession converts schema to DTO.
func FromSchemaIdentityVerificationSession(s *schema.IdentityVerificationSession) *IdentityVerificationSession {
	if s == nil {
		return nil
	}

	return &IdentityVerificationSession{
		ID:             s.ID,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		AppID:          s.AppID,
		EnvironmentID:  s.EnvironmentID,
		OrganizationID: s.OrganizationID,
		UserID:         s.UserID,
		Provider:       s.Provider,
		SessionURL:     s.SessionURL,
		SessionToken:   s.SessionToken,
		RequiredChecks: s.RequiredChecks,
		Config:         s.Config,
		Status:         s.Status,
		CompletedAt:    s.CompletedAt,
		ExpiresAt:      s.ExpiresAt,
		SuccessURL:     s.SuccessURL,
		CancelURL:      s.CancelURL,
		IPAddress:      s.IPAddress,
		UserAgent:      s.UserAgent,
	}
}

// FromSchemaIdentityVerification converts schema to DTO.
func FromSchemaIdentityVerification(v *schema.IdentityVerification) *IdentityVerification {
	if v == nil {
		return nil
	}

	return &IdentityVerification{
		ID:                    v.ID,
		CreatedAt:             v.CreatedAt,
		UpdatedAt:             v.UpdatedAt,
		AppID:                 v.AppID,
		EnvironmentID:         v.EnvironmentID,
		OrganizationID:        v.OrganizationID,
		UserID:                v.UserID,
		Provider:              v.Provider,
		ProviderCheckID:       v.ProviderCheckID,
		VerificationType:      v.VerificationType,
		Status:                v.Status,
		DocumentType:          v.DocumentType,
		DocumentNumber:        v.DocumentNumber,
		DocumentCountry:       v.DocumentCountry,
		IsVerified:            v.IsVerified,
		RiskScore:             v.RiskScore,
		RiskLevel:             v.RiskLevel,
		ConfidenceScore:       v.ConfidenceScore,
		FirstName:             v.FirstName,
		LastName:              v.LastName,
		DateOfBirth:           v.DateOfBirth,
		Age:                   v.Age,
		Gender:                v.Gender,
		Nationality:           v.Nationality,
		IsOnSanctionsList:     v.IsOnSanctionsList,
		IsPEP:                 v.IsPEP,
		SanctionsDetails:      v.SanctionsDetails,
		LivenessScore:         v.LivenessScore,
		IsLive:                v.IsLive,
		RejectionReasons:      v.RejectionReasons,
		FailureReason:         v.FailureReason,
		Metadata:              v.Metadata,
		ProviderData:          v.ProviderData,
		IPAddress:             v.IPAddress,
		UserAgent:             v.UserAgent,
		ExpiresAt:             v.ExpiresAt,
		VerifiedAt:            v.VerifiedAt,
		WebhookDeliveryStatus: v.WebhookDeliveryStatus,
		WebhookDeliveredAt:    v.WebhookDeliveredAt,
	}
}

// FromSchemaIdentityVerifications converts slice of schema to DTOs.
func FromSchemaIdentityVerifications(verifications []*schema.IdentityVerification) []*IdentityVerification {
	if verifications == nil {
		return nil
	}

	result := make([]*IdentityVerification, len(verifications))
	for i, v := range verifications {
		result[i] = FromSchemaIdentityVerification(v)
	}

	return result
}

// FromSchemaUserVerificationStatus converts schema to DTO.
func FromSchemaUserVerificationStatus(s *schema.UserVerificationStatus) *UserVerificationStatus {
	if s == nil {
		return nil
	}

	return &UserVerificationStatus{
		ID:                         s.ID,
		CreatedAt:                  s.CreatedAt,
		UpdatedAt:                  s.UpdatedAt,
		AppID:                      s.AppID,
		EnvironmentID:              s.EnvironmentID,
		OrganizationID:             s.OrganizationID,
		UserID:                     s.UserID,
		IsVerified:                 s.IsVerified,
		VerificationLevel:          s.VerificationLevel,
		LastVerifiedAt:             s.LastVerifiedAt,
		VerificationExpiry:         s.VerificationExpiry,
		RequiresReverification:     s.RequiresReverification,
		DocumentVerified:           s.DocumentVerified,
		LivenessVerified:           s.LivenessVerified,
		AgeVerified:                s.AgeVerified,
		AMLScreened:                s.AMLScreened,
		AMLClear:                   s.AMLClear,
		LastDocumentVerificationID: s.LastDocumentVerificationID,
		LastLivenessVerificationID: s.LastLivenessVerificationID,
		LastAMLVerificationID:      s.LastAMLVerificationID,
		OverallRiskLevel:           s.OverallRiskLevel,
		RiskFactors:                s.RiskFactors,
		IsBlocked:                  s.IsBlocked,
		BlockReason:                s.BlockReason,
		BlockedAt:                  s.BlockedAt,
		Metadata:                   s.Metadata,
	}
}
