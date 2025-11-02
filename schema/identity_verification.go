package schema

import (
	"time"

	"github.com/uptrace/bun"
)

// IdentityVerification represents a KYC verification attempt
type IdentityVerification struct {
	bun.BaseModel `bun:"table:identity_verifications,alias:iv"`

	ID             string    `bun:"id,pk,type:varchar(255)" json:"id"`
	UserID         string    `bun:"user_id,notnull,type:varchar(255)" json:"userId"`
	OrganizationID string    `bun:"organization_id,notnull,type:varchar(255)" json:"organizationId"`
	
	// Provider information
	Provider        string `bun:"provider,notnull,type:varchar(50)" json:"provider"` // onfido, jumio, stripe_identity
	ProviderCheckID string `bun:"provider_check_id,type:varchar(255)" json:"providerCheckId"`
	
	// Verification type and status
	VerificationType string `bun:"verification_type,notnull,type:varchar(50)" json:"verificationType"` // document, liveness, age, aml
	Status           string `bun:"status,notnull,type:varchar(50)" json:"status"`                      // pending, in_progress, completed, failed, expired
	
	// Document information (if applicable)
	DocumentType   string `bun:"document_type,type:varchar(50)" json:"documentType,omitempty"`     // passport, drivers_license, national_id
	DocumentNumber string `bun:"document_number,type:varchar(255)" json:"documentNumber,omitempty"` // Encrypted
	DocumentCountry string `bun:"document_country,type:varchar(2)" json:"documentCountry,omitempty"` // ISO 3166-1 alpha-2
	
	// Verification results
	IsVerified      bool   `bun:"is_verified,default:false" json:"isVerified"`
	RiskScore       int    `bun:"risk_score,type:int" json:"riskScore,omitempty"` // 0-100, higher is riskier
	RiskLevel       string `bun:"risk_level,type:varchar(20)" json:"riskLevel,omitempty"` // low, medium, high
	ConfidenceScore int    `bun:"confidence_score,type:int" json:"confidenceScore,omitempty"` // 0-100
	
	// Personal information extracted
	FirstName   string     `bun:"first_name,type:varchar(255)" json:"firstName,omitempty"`
	LastName    string     `bun:"last_name,type:varchar(255)" json:"lastName,omitempty"`
	DateOfBirth *time.Time `bun:"date_of_birth,type:date" json:"dateOfBirth,omitempty"`
	Age         int        `bun:"age,type:int" json:"age,omitempty"`
	Gender      string     `bun:"gender,type:varchar(20)" json:"gender,omitempty"`
	Nationality string     `bun:"nationality,type:varchar(2)" json:"nationality,omitempty"` // ISO 3166-1 alpha-2
	
	// AML/Sanctions screening results
	IsOnSanctionsList bool   `bun:"is_on_sanctions_list,default:false" json:"isOnSanctionsList"`
	IsPEP             bool   `bun:"is_pep,default:false" json:"isPep"` // Politically Exposed Person
	SanctionsDetails  string `bun:"sanctions_details,type:text" json:"sanctionsDetails,omitempty"`
	
	// Liveness detection
	LivenessScore int  `bun:"liveness_score,type:int" json:"livenessScore,omitempty"` // 0-100
	IsLive        bool `bun:"is_live,default:false" json:"isLive"`
	
	// Rejection/failure information
	RejectionReasons []string `bun:"rejection_reasons,type:jsonb" json:"rejectionReasons,omitempty"`
	FailureReason    string   `bun:"failure_reason,type:text" json:"failureReason,omitempty"`
	
	// Metadata
	Metadata       map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	ProviderData   map[string]interface{} `bun:"provider_data,type:jsonb" json:"providerData,omitempty"` // Raw provider response
	IPAddress      string                 `bun:"ip_address,type:varchar(45)" json:"ipAddress,omitempty"`
	UserAgent      string                 `bun:"user_agent,type:text" json:"userAgent,omitempty"`
	
	// Expiry and validity
	ExpiresAt  *time.Time `bun:"expires_at,type:timestamptz" json:"expiresAt,omitempty"`
	VerifiedAt *time.Time `bun:"verified_at,type:timestamptz" json:"verifiedAt,omitempty"`
	
	// Webhook tracking
	WebhookDeliveryStatus string     `bun:"webhook_delivery_status,type:varchar(50)" json:"webhookDeliveryStatus,omitempty"`
	WebhookDeliveredAt    *time.Time `bun:"webhook_delivered_at,type:timestamptz" json:"webhookDeliveredAt,omitempty"`
	
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	
	// Relations
	User         *User         `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}

// IdentityVerificationDocument represents uploaded documents for verification
type IdentityVerificationDocument struct {
	bun.BaseModel `bun:"table:identity_verification_documents,alias:ivd"`

	ID             string `bun:"id,pk,type:varchar(255)" json:"id"`
	VerificationID string `bun:"verification_id,notnull,type:varchar(255)" json:"verificationId"`
	
	// Document details
	DocumentSide string `bun:"document_side,type:varchar(20)" json:"documentSide"` // front, back, selfie
	FileURL      string `bun:"file_url,type:text" json:"fileUrl"` // Encrypted storage URL
	FileHash     string `bun:"file_hash,type:varchar(64)" json:"fileHash"` // SHA-256 for integrity
	MimeType     string `bun:"mime_type,type:varchar(100)" json:"mimeType"`
	FileSize     int64  `bun:"file_size,type:bigint" json:"fileSize"`
	
	// Processing status
	ProcessingStatus string `bun:"processing_status,type:varchar(50)" json:"processingStatus"` // pending, processing, processed, failed
	
	// Extracted data
	ExtractedData map[string]interface{} `bun:"extracted_data,type:jsonb" json:"extractedData,omitempty"`
	
	// Retention policy
	RetainUntil *time.Time `bun:"retain_until,type:timestamptz" json:"retainUntil,omitempty"`
	DeletedAt   *time.Time `bun:"deleted_at,type:timestamptz" json:"deletedAt,omitempty"`
	
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	
	// Relations
	Verification *IdentityVerification `bun:"rel:belongs-to,join:verification_id=id" json:"verification,omitempty"`
}

// IdentityVerificationSession represents a verification session/flow
type IdentityVerificationSession struct {
	bun.BaseModel `bun:"table:identity_verification_sessions,alias:ivs"`

	ID             string `bun:"id,pk,type:varchar(255)" json:"id"`
	UserID         string `bun:"user_id,notnull,type:varchar(255)" json:"userId"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(255)" json:"organizationId"`
	
	// Session details
	Provider   string `bun:"provider,notnull,type:varchar(50)" json:"provider"`
	SessionURL string `bun:"session_url,type:text" json:"sessionUrl"` // URL for user to complete verification
	SessionToken string `bun:"session_token,type:varchar(255)" json:"sessionToken"` // Encrypted
	
	// Configuration
	RequiredChecks []string               `bun:"required_checks,type:jsonb" json:"requiredChecks"` // [document, liveness, aml]
	Config         map[string]interface{} `bun:"config,type:jsonb" json:"config,omitempty"`
	
	// Status tracking
	Status        string     `bun:"status,notnull,type:varchar(50)" json:"status"` // created, started, completed, abandoned, expired
	CompletedAt   *time.Time `bun:"completed_at,type:timestamptz" json:"completedAt,omitempty"`
	ExpiresAt     time.Time  `bun:"expires_at,notnull,type:timestamptz" json:"expiresAt"`
	
	// Callback URLs
	SuccessURL string `bun:"success_url,type:text" json:"successUrl,omitempty"`
	CancelURL  string `bun:"cancel_url,type:text" json:"cancelUrl,omitempty"`
	
	// Tracking
	IPAddress string `bun:"ip_address,type:varchar(45)" json:"ipAddress,omitempty"`
	UserAgent string `bun:"user_agent,type:text" json:"userAgent,omitempty"`
	
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	
	// Relations
	User         *User         `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}

// UserVerificationStatus tracks the overall verification status of a user
type UserVerificationStatus struct {
	bun.BaseModel `bun:"table:user_verification_status,alias:uvs"`

	ID             string `bun:"id,pk,type:varchar(255)" json:"id"`
	UserID         string `bun:"user_id,notnull,unique,type:varchar(255)" json:"userId"`
	OrganizationID string `bun:"organization_id,notnull,type:varchar(255)" json:"organizationId"`
	
	// Overall verification status
	IsVerified           bool       `bun:"is_verified,default:false" json:"isVerified"`
	VerificationLevel    string     `bun:"verification_level,type:varchar(50)" json:"verificationLevel"` // none, basic, enhanced, full
	LastVerifiedAt       *time.Time `bun:"last_verified_at,type:timestamptz" json:"lastVerifiedAt,omitempty"`
	VerificationExpiry   *time.Time `bun:"verification_expiry,type:timestamptz" json:"verificationExpiry,omitempty"`
	RequiresReverification bool     `bun:"requires_reverification,default:false" json:"requiresReverification"`
	
	// Individual check statuses
	DocumentVerified bool       `bun:"document_verified,default:false" json:"documentVerified"`
	LivenessVerified bool       `bun:"liveness_verified,default:false" json:"livenessVerified"`
	AgeVerified      bool       `bun:"age_verified,default:false" json:"ageVerified"`
	AMLScreened      bool       `bun:"aml_screened,default:false" json:"amlScreened"`
	AMLClear         bool       `bun:"aml_clear,default:false" json:"amlClear"`
	
	// Most recent verification IDs
	LastDocumentVerificationID string `bun:"last_document_verification_id,type:varchar(255)" json:"lastDocumentVerificationId,omitempty"`
	LastLivenessVerificationID string `bun:"last_liveness_verification_id,type:varchar(255)" json:"lastLivenessVerificationId,omitempty"`
	LastAMLVerificationID      string `bun:"last_aml_verification_id,type:varchar(255)" json:"lastAMLVerificationId,omitempty"`
	
	// Risk assessment
	OverallRiskLevel string `bun:"overall_risk_level,type:varchar(20)" json:"overallRiskLevel"` // low, medium, high
	RiskFactors      []string `bun:"risk_factors,type:jsonb" json:"riskFactors,omitempty"`
	
	// Compliance flags
	IsBlocked    bool   `bun:"is_blocked,default:false" json:"isBlocked"`
	BlockReason  string `bun:"block_reason,type:text" json:"blockReason,omitempty"`
	BlockedAt    *time.Time `bun:"blocked_at,type:timestamptz" json:"blockedAt,omitempty"`
	
	// Metadata
	Metadata map[string]interface{} `bun:"metadata,type:jsonb" json:"metadata,omitempty"`
	
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	
	// Relations
	User         *User         `bun:"rel:belongs-to,join:user_id=id" json:"user,omitempty"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"organization,omitempty"`
}

