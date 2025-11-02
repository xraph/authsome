package backupauth

import (
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// RecoveryMethod represents different recovery authentication methods
type RecoveryMethod string

const (
	RecoveryMethodCodes          RecoveryMethod = "recovery_codes"
	RecoveryMethodSecurityQ      RecoveryMethod = "security_questions"
	RecoveryMethodTrustedContact RecoveryMethod = "trusted_contact"
	RecoveryMethodEmail          RecoveryMethod = "email_verification"
	RecoveryMethodSMS            RecoveryMethod = "sms_verification"
	RecoveryMethodVideo          RecoveryMethod = "video_verification"
	RecoveryMethodDocument       RecoveryMethod = "document_upload"
)

// RecoveryStatus represents the status of a recovery attempt
type RecoveryStatus string

const (
	RecoveryStatusPending    RecoveryStatus = "pending"
	RecoveryStatusInProgress RecoveryStatus = "in_progress"
	RecoveryStatusCompleted  RecoveryStatus = "completed"
	RecoveryStatusFailed     RecoveryStatus = "failed"
	RecoveryStatusExpired    RecoveryStatus = "expired"
	RecoveryStatusCancelled  RecoveryStatus = "cancelled"
)

// SecurityQuestion stores user's security questions and hashed answers
type SecurityQuestion struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_security_questions,alias:bsq"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	UserID         xid.ID `bun:"user_id,notnull,type:varchar(20),unique:user_question"`
	OrganizationID string `bun:"organization_id,notnull"`
	
	QuestionID   int    `bun:"question_id,notnull,unique:user_question"` // Reference to predefined question
	CustomText   string `bun:"custom_text"` // For custom questions
	AnswerHash   string `bun:"answer_hash,notnull"` // Hashed answer
	Salt         string `bun:"salt,notnull"`
	
	IsActive     bool       `bun:"is_active,notnull,default:true"`
	LastUsedAt   *time.Time `bun:"last_used_at"`
	FailedAttempts int      `bun:"failed_attempts,notnull,default:0"`
}

// TrustedContact stores emergency contact information for account recovery
type TrustedContact struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_trusted_contacts,alias:btc"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	UserID         xid.ID `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string `bun:"organization_id,notnull"`
	
	ContactName  string `bun:"contact_name,notnull"`
	ContactEmail string `bun:"contact_email"`
	ContactPhone string `bun:"contact_phone"`
	Relationship string `bun:"relationship"` // friend, family, colleague, etc.
	
	VerificationToken string     `bun:"verification_token"`
	VerifiedAt        *time.Time `bun:"verified_at"`
	IsActive          bool       `bun:"is_active,notnull,default:true"`
	LastNotifiedAt    *time.Time `bun:"last_notified_at"`
	
	// Metadata for verification
	IPAddress string            `bun:"ip_address"`
	UserAgent string            `bun:"user_agent"`
	Metadata  map[string]string `bun:"metadata,type:jsonb"`
}

// RecoverySession represents an account recovery attempt
type RecoverySession struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_recovery_sessions,alias:brs"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"`
	UserID         xid.ID         `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string         `bun:"organization_id,notnull"`
	
	Status         RecoveryStatus `bun:"status,notnull"`
	Method         RecoveryMethod `bun:"method,notnull"`
	
	// Multi-step flow tracking
	RequiredSteps   []string `bun:"required_steps,type:jsonb"` // Methods required to complete
	CompletedSteps  []string `bun:"completed_steps,type:jsonb"`
	CurrentStep     int      `bun:"current_step,notnull,default:0"`
	
	// Verification data
	VerificationCode string     `bun:"verification_code"` // For email/SMS verification
	CodeExpiresAt    *time.Time `bun:"code_expires_at"`
	Attempts         int        `bun:"attempts,notnull,default:0"`
	MaxAttempts      int        `bun:"max_attempts,notnull,default:5"`
	
	// Security
	IPAddress    string     `bun:"ip_address"`
	UserAgent    string     `bun:"user_agent"`
	DeviceID     string     `bun:"device_id"`
	RiskScore    float64    `bun:"risk_score"`
	
	// Completion
	CompletedAt  *time.Time `bun:"completed_at"`
	ExpiresAt    time.Time  `bun:"expires_at,notnull"`
	CancelledAt  *time.Time `bun:"cancelled_at"`
	
	// Admin review (for high-risk recoveries)
	RequiresReview bool       `bun:"requires_review,notnull,default:false"`
	ReviewedBy     *xid.ID    `bun:"reviewed_by,type:varchar(20)"`
	ReviewedAt     *time.Time `bun:"reviewed_at"`
	ReviewNotes    string     `bun:"review_notes"`
	
	Metadata map[string]interface{} `bun:"metadata,type:jsonb"`
}

// VideoVerificationSession stores video verification details
type VideoVerificationSession struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_video_sessions,alias:bvs"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	RecoveryID     xid.ID `bun:"recovery_id,notnull,type:varchar(20)"`
	UserID         xid.ID `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string `bun:"organization_id,notnull"`
	
	// Video session details
	SessionURL       string     `bun:"session_url"`
	RecordingURL     string     `bun:"recording_url"`
	ProviderSessionID string    `bun:"provider_session_id"` // Zoom, Teams, etc.
	
	ScheduledAt      time.Time  `bun:"scheduled_at,notnull"`
	StartedAt        *time.Time `bun:"started_at"`
	CompletedAt      *time.Time `bun:"completed_at"`
	
	// Verification
	VerifierID     *xid.ID `bun:"verifier_id,type:varchar(20)"` // Admin who verified
	VerifiedAt     *time.Time `bun:"verified_at"`
	VerificationResult string `bun:"verification_result"` // approved, rejected, pending
	VerificationNotes  string `bun:"verification_notes"`
	
	// Liveness checks
	LivenessCheckPassed bool   `bun:"liveness_check_passed,default:false"`
	LivenessScore       float64 `bun:"liveness_score"`
	
	Status   string            `bun:"status,notnull"` // scheduled, in_progress, completed, failed
	Metadata map[string]string `bun:"metadata,type:jsonb"`
}

// DocumentVerification stores ID document uploads for verification
type DocumentVerification struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_document_verifications,alias:bdv"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	RecoveryID     xid.ID `bun:"recovery_id,notnull,type:varchar(20)"`
	UserID         xid.ID `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string `bun:"organization_id,notnull"`
	
	// Document details
	DocumentType   string `bun:"document_type,notnull"` // passport, drivers_license, national_id, etc.
	DocumentNumber string `bun:"document_number"` // Encrypted
	FrontImageURL  string `bun:"front_image_url"`
	BackImageURL   string `bun:"back_image_url"`
	SelfieURL      string `bun:"selfie_url"`
	
	// OCR/Extraction results
	ExtractedData map[string]interface{} `bun:"extracted_data,type:jsonb"`
	
	// Verification
	VerificationStatus string  `bun:"verification_status,notnull"` // pending, verified, rejected
	ConfidenceScore    float64 `bun:"confidence_score"`
	VerifiedAt         *time.Time `bun:"verified_at"`
	VerifiedBy         *xid.ID    `bun:"verified_by,type:varchar(20)"`
	
	// Provider integration (Stripe Identity, Onfido, etc.)
	ProviderName    string `bun:"provider_name"`
	ProviderID      string `bun:"provider_id"`
	ProviderResponse map[string]interface{} `bun:"provider_response,type:jsonb"`
	
	RejectionReason string `bun:"rejection_reason"`
	ExpiresAt       time.Time `bun:"expires_at,notnull"`
	
	Metadata map[string]string `bun:"metadata,type:jsonb"`
}

// RecoveryAttemptLog provides immutable audit trail of recovery attempts
type RecoveryAttemptLog struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_recovery_logs,alias:brl"`

	ID             xid.ID         `bun:"id,pk,type:varchar(20)"`
	RecoveryID     xid.ID         `bun:"recovery_id,notnull,type:varchar(20)"`
	UserID         xid.ID         `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string         `bun:"organization_id,notnull"`
	
	Action       string         `bun:"action,notnull"` // started, step_completed, verified, failed, etc.
	Method       RecoveryMethod `bun:"method,notnull"`
	Step         int            `bun:"step"`
	
	Success      bool   `bun:"success"`
	FailureReason string `bun:"failure_reason"`
	
	IPAddress string `bun:"ip_address"`
	UserAgent string `bun:"user_agent"`
	DeviceID  string `bun:"device_id"`
	
	Metadata map[string]interface{} `bun:"metadata,type:jsonb"`
}

// RecoveryConfiguration stores organization-level recovery settings
type RecoveryConfiguration struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_recovery_configs,alias:brc"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	OrganizationID string `bun:"organization_id,notnull,unique"`
	
	// Enabled methods
	EnabledMethods []RecoveryMethod `bun:"enabled_methods,type:jsonb"`
	
	// Multi-step requirements
	RequireMultipleSteps bool `bun:"require_multiple_steps,notnull,default:true"`
	MinimumStepsRequired int  `bun:"minimum_steps_required,notnull,default:2"`
	
	// Security settings
	RequireAdminReview   bool `bun:"require_admin_review,default:false"`
	RiskScoreThreshold   float64 `bun:"risk_score_threshold,default:70.0"`
	
	// Time limits
	SessionExpiryMinutes int `bun:"session_expiry_minutes,default:30"`
	CodeExpiryMinutes    int `bun:"code_expiry_minutes,default:15"`
	
	// Rate limiting
	MaxAttemptsPerDay    int           `bun:"max_attempts_per_day,default:3"`
	LockoutDuration      time.Duration `bun:"lockout_duration,default:24h"`
	
	Settings map[string]interface{} `bun:"settings,type:jsonb"`
}

// RecoveryCodeUsage tracks when recovery codes are used (separate from 2FA backup codes)
type RecoveryCodeUsage struct {
	schema.AuditableModel
	bun.BaseModel `bun:"table:backup_code_usage,alias:bcu"`

	ID             xid.ID `bun:"id,pk,type:varchar(20)"`
	UserID         xid.ID `bun:"user_id,notnull,type:varchar(20)"`
	OrganizationID string `bun:"organization_id,notnull"`
	RecoveryID     xid.ID `bun:"recovery_id,notnull,type:varchar(20)"`
	
	CodeHash   string    `bun:"code_hash,notnull"`
	UsedAt     time.Time `bun:"used_at,notnull"`
	IPAddress  string    `bun:"ip_address"`
	UserAgent  string    `bun:"user_agent"`
	DeviceID   string    `bun:"device_id"`
}

