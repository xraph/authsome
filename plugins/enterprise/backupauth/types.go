package backupauth

import (
	"time"

	"github.com/rs/xid"
)

// ===== Recovery Session Requests/Responses =====

// StartRecoveryRequest initiates a recovery session.
type StartRecoveryRequest struct {
	UserID          string         `json:"userId"`
	Email           string         `json:"email,omitempty"`
	PreferredMethod RecoveryMethod `json:"preferredMethod,omitempty"`
	DeviceID        string         `json:"deviceId,omitempty"`
}

// StartRecoveryResponse returns recovery session details.
type StartRecoveryResponse struct {
	SessionID        xid.ID           `json:"sessionId"`
	Status           RecoveryStatus   `json:"status"`
	AvailableMethods []RecoveryMethod `json:"availableMethods"`
	RequiredSteps    int              `json:"requiredSteps"`
	CompletedSteps   int              `json:"completedSteps"`
	ExpiresAt        time.Time        `json:"expiresAt"`
	RiskScore        float64          `json:"riskScore,omitempty"`
	RequiresReview   bool             `json:"requiresReview"`
}

// ContinueRecoveryRequest continues a recovery session with method selection.
type ContinueRecoveryRequest struct {
	SessionID xid.ID         `json:"sessionId"`
	Method    RecoveryMethod `json:"method"`
}

// ContinueRecoveryResponse provides next steps.
type ContinueRecoveryResponse struct {
	SessionID    xid.ID         `json:"sessionId"`
	Method       RecoveryMethod `json:"method"`
	CurrentStep  int            `json:"currentStep"`
	TotalSteps   int            `json:"totalSteps"`
	Instructions string         `json:"instructions"`
	Data         map[string]any `json:"data,omitempty"`
	ExpiresAt    time.Time      `json:"expiresAt"`
}

// CompleteRecoveryRequest finalizes recovery.
type CompleteRecoveryRequest struct {
	SessionID xid.ID `json:"sessionId"`
}

// CompleteRecoveryResponse returns recovery completion details.
type CompleteRecoveryResponse struct {
	SessionID   xid.ID         `json:"sessionId"`
	Status      RecoveryStatus `json:"status"`
	CompletedAt time.Time      `json:"completedAt"`
	Token       string         `json:"token,omitempty"` // Temporary token to reset password
	Message     string         `json:"message"`
}

// CancelRecoveryRequest cancels a recovery session.
type CancelRecoveryRequest struct {
	SessionID xid.ID `json:"sessionId"`
	Reason    string `json:"reason,omitempty"`
}

// ===== Recovery Codes =====

// GenerateRecoveryCodesRequest generates new recovery codes.
type GenerateRecoveryCodesRequest struct {
	Count  int    `json:"count,omitempty"`
	Format string `json:"format,omitempty"` // alphanumeric, numeric, hex
}

// GenerateRecoveryCodesResponse returns generated codes.
type GenerateRecoveryCodesResponse struct {
	Codes       []string  `json:"codes"`
	Count       int       `json:"count"`
	GeneratedAt time.Time `json:"generatedAt"`
	Warning     string    `json:"warning"`
}

// VerifyRecoveryCodeRequest verifies a recovery code.
type VerifyRecoveryCodeRequest struct {
	SessionID xid.ID `json:"sessionId"`
	Code      string `json:"code"`
}

// VerifyRecoveryCodeResponse returns verification result.
type VerifyRecoveryCodeResponse struct {
	Valid          bool   `json:"valid"`
	RemainingCodes int    `json:"remainingCodes,omitempty"`
	Message        string `json:"message"`
}

// ===== Security Questions =====

// SetupSecurityQuestionRequest sets up a security question.
type SetupSecurityQuestionRequest struct {
	QuestionID int    `json:"questionId,omitempty"` // ID of predefined question
	CustomText string `json:"customText,omitempty"` // For custom questions
	Answer     string `json:"answer"`
}

// SetupSecurityQuestionsRequest sets up multiple questions.
type SetupSecurityQuestionsRequest struct {
	Questions []SetupSecurityQuestionRequest `json:"questions"`
}

// SetupSecurityQuestionsResponse returns setup result.
type SetupSecurityQuestionsResponse struct {
	Count   int       `json:"count"`
	Message string    `json:"message"`
	SetupAt time.Time `json:"setupAt"`
}

// GetSecurityQuestionsRequest gets user's security questions.
type GetSecurityQuestionsRequest struct {
	SessionID xid.ID `json:"sessionId"`
}

// SecurityQuestionInfo provides question info without answer.
type SecurityQuestionInfo struct {
	ID           xid.ID `json:"id"`
	QuestionID   int    `json:"questionId,omitempty"`
	QuestionText string `json:"questionText"`
	IsCustom     bool   `json:"isCustom"`
}

// GetSecurityQuestionsResponse returns questions.
type GetSecurityQuestionsResponse struct {
	Questions []SecurityQuestionInfo `json:"questions"`
}

// VerifySecurityAnswersRequest verifies security answers.
type VerifySecurityAnswersRequest struct {
	SessionID xid.ID            `json:"sessionId"`
	Answers   map[string]string `json:"answers"` // questionID -> answer
}

// VerifySecurityAnswersResponse returns verification result.
type VerifySecurityAnswersResponse struct {
	Valid           bool   `json:"valid"`
	CorrectAnswers  int    `json:"correctAnswers"`
	RequiredAnswers int    `json:"requiredAnswers"`
	AttemptsLeft    int    `json:"attemptsLeft"`
	Message         string `json:"message"`
}

// ===== Trusted Contacts =====

// AddTrustedContactRequest adds a trusted contact.
type AddTrustedContactRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Relationship string `json:"relationship,omitempty"`
}

// AddTrustedContactResponse returns added contact.
type AddTrustedContactResponse struct {
	ContactID xid.ID    `json:"contactId"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Verified  bool      `json:"verified"`
	AddedAt   time.Time `json:"addedAt"`
	Message   string    `json:"message"`
}

// VerifyTrustedContactRequest verifies a trusted contact.
type VerifyTrustedContactRequest struct {
	Token string `json:"token"`
}

// VerifyTrustedContactResponse returns verification result.
type VerifyTrustedContactResponse struct {
	ContactID  xid.ID    `json:"contactId"`
	Verified   bool      `json:"verified"`
	VerifiedAt time.Time `json:"verifiedAt"`
	Message    string    `json:"message"`
}

// RequestTrustedContactVerificationRequest requests contact verification.
type RequestTrustedContactVerificationRequest struct {
	SessionID xid.ID `json:"sessionId"`
	ContactID xid.ID `json:"contactId"`
}

// RequestTrustedContactVerificationResponse returns request result.
type RequestTrustedContactVerificationResponse struct {
	ContactID   xid.ID    `json:"contactId"`
	ContactName string    `json:"contactName"`
	NotifiedAt  time.Time `json:"notifiedAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Message     string    `json:"message"`
}

// ListTrustedContactsResponse returns user's trusted contacts.
type ListTrustedContactsResponse struct {
	Contacts []TrustedContactInfo `json:"contacts"`
	Count    int                  `json:"count"`
}

// TrustedContactInfo provides contact information.
type TrustedContactInfo struct {
	ID           xid.ID     `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email,omitempty"`
	Phone        string     `json:"phone,omitempty"`
	Relationship string     `json:"relationship,omitempty"`
	Verified     bool       `json:"verified"`
	VerifiedAt   *time.Time `json:"verifiedAt,omitempty"`
	Active       bool       `json:"active"`
}

// RemoveTrustedContactRequest removes a trusted contact.
type RemoveTrustedContactRequest struct {
	ContactID xid.ID `json:"contactId"`
}

// ===== Email/SMS Verification =====

// SendVerificationCodeRequest sends a verification code.
type SendVerificationCodeRequest struct {
	SessionID xid.ID         `json:"sessionId"`
	Method    RecoveryMethod `json:"method"`           // email_verification or sms_verification
	Target    string         `json:"target,omitempty"` // Email or phone if different from user's
}

// SendVerificationCodeResponse returns send result.
type SendVerificationCodeResponse struct {
	Sent         bool      `json:"sent"`
	MaskedTarget string    `json:"maskedTarget"` // e.g., "j***@example.com" or "+1***5678"
	ExpiresAt    time.Time `json:"expiresAt"`
	Message      string    `json:"message"`
}

// VerifyCodeRequest verifies a sent code.
type VerifyCodeRequest struct {
	SessionID xid.ID `json:"sessionId"`
	Code      string `json:"code"`
}

// VerifyCodeResponse returns verification result.
type VerifyCodeResponse struct {
	Valid        bool   `json:"valid"`
	AttemptsLeft int    `json:"attemptsLeft"`
	Message      string `json:"message"`
}

// ===== Video Verification =====

// ScheduleVideoSessionRequest schedules a video verification.
type ScheduleVideoSessionRequest struct {
	SessionID   xid.ID    `json:"sessionId"`
	ScheduledAt time.Time `json:"scheduledAt"`
	TimeZone    string    `json:"timeZone,omitempty"`
}

// ScheduleVideoSessionResponse returns scheduled session.
type ScheduleVideoSessionResponse struct {
	VideoSessionID xid.ID    `json:"videoSessionId"`
	ScheduledAt    time.Time `json:"scheduledAt"`
	JoinURL        string    `json:"joinUrl,omitempty"`
	Instructions   string    `json:"instructions"`
	Message        string    `json:"message"`
}

// StartVideoSessionRequest starts a video session.
type StartVideoSessionRequest struct {
	VideoSessionID xid.ID `json:"videoSessionId"`
}

// StartVideoSessionResponse returns session details.
type StartVideoSessionResponse struct {
	VideoSessionID xid.ID    `json:"videoSessionId"`
	SessionURL     string    `json:"sessionUrl"`
	StartedAt      time.Time `json:"startedAt"`
	ExpiresAt      time.Time `json:"expiresAt"`
	Message        string    `json:"message"`
}

// CompleteVideoSessionRequest completes video verification (admin).
type CompleteVideoSessionRequest struct {
	VideoSessionID     xid.ID  `json:"videoSessionId"`
	VerificationResult string  `json:"verificationResult"` // approved, rejected
	Notes              string  `json:"notes,omitempty"`
	LivenessPassed     bool    `json:"livenessPassed"`
	LivenessScore      float64 `json:"livenessScore,omitempty"`
}

// CompleteVideoSessionResponse returns completion result.
type CompleteVideoSessionResponse struct {
	VideoSessionID xid.ID    `json:"videoSessionId"`
	Result         string    `json:"result"`
	CompletedAt    time.Time `json:"completedAt"`
	Message        string    `json:"message"`
}

// ===== Document Verification =====

// UploadDocumentRequest uploads verification documents.
type UploadDocumentRequest struct {
	SessionID    xid.ID `json:"sessionId"`
	DocumentType string `json:"documentType"`        // passport, drivers_license, etc.
	FrontImage   string `json:"frontImage"`          // Base64 encoded
	BackImage    string `json:"backImage,omitempty"` // Base64 encoded
	Selfie       string `json:"selfie,omitempty"`    // Base64 encoded
}

// UploadDocumentResponse returns upload result.
type UploadDocumentResponse struct {
	DocumentID     xid.ID    `json:"documentId"`
	Status         string    `json:"status"`
	UploadedAt     time.Time `json:"uploadedAt"`
	ProcessingTime string    `json:"processingTime,omitempty"`
	Message        string    `json:"message"`
}

// GetDocumentVerificationRequest gets verification status.
type GetDocumentVerificationRequest struct {
	DocumentID xid.ID `json:"documentId"`
}

// GetDocumentVerificationResponse returns verification status.
type GetDocumentVerificationResponse struct {
	DocumentID      xid.ID     `json:"documentId"`
	Status          string     `json:"status"` // pending, verified, rejected
	ConfidenceScore float64    `json:"confidenceScore,omitempty"`
	VerifiedAt      *time.Time `json:"verifiedAt,omitempty"`
	RejectionReason string     `json:"rejectionReason,omitempty"`
	Message         string     `json:"message"`
}

// ReviewDocumentRequest reviews document (admin).
type ReviewDocumentRequest struct {
	DocumentID      xid.ID `json:"documentId"`
	Approved        bool   `json:"approved"`
	RejectionReason string `json:"rejectionReason,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

// ===== Admin Endpoints =====

// ListRecoverySessionsRequest lists recovery sessions (admin).
type ListRecoverySessionsRequest struct {
	OrganizationID string         `json:"organizationId,omitempty"`
	Status         RecoveryStatus `json:"status,omitempty"`
	RequiresReview bool           `json:"requiresReview,omitempty"`
	Page           int            `json:"page,omitempty"`
	PageSize       int            `json:"pageSize,omitempty"`
}

// ListRecoverySessionsResponse returns sessions.
type ListRecoverySessionsResponse struct {
	Sessions   []RecoverySessionInfo `json:"sessions"`
	TotalCount int                   `json:"totalCount"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
}

// RecoverySessionInfo provides session information.
type RecoverySessionInfo struct {
	ID             xid.ID         `json:"id"`
	UserID         xid.ID         `json:"userId"`
	UserEmail      string         `json:"userEmail,omitempty"`
	Status         RecoveryStatus `json:"status"`
	Method         RecoveryMethod `json:"method"`
	CurrentStep    int            `json:"currentStep"`
	TotalSteps     int            `json:"totalSteps"`
	RiskScore      float64        `json:"riskScore"`
	RequiresReview bool           `json:"requiresReview"`
	CreatedAt      time.Time      `json:"createdAt"`
	ExpiresAt      time.Time      `json:"expiresAt"`
	CompletedAt    *time.Time     `json:"completedAt,omitempty"`
}

// ApproveRecoveryRequest approves a recovery session (admin).
type ApproveRecoveryRequest struct {
	SessionID xid.ID `json:"sessionId"`
	Notes     string `json:"notes,omitempty"`
}

// ApproveRecoveryResponse returns approval result.
type ApproveRecoveryResponse struct {
	SessionID  xid.ID    `json:"sessionId"`
	Approved   bool      `json:"approved"`
	ApprovedAt time.Time `json:"approvedAt"`
	Message    string    `json:"message"`
}

// RejectRecoveryRequest rejects a recovery session (admin).
type RejectRecoveryRequest struct {
	SessionID xid.ID `json:"sessionId"`
	Reason    string `json:"reason"`
	Notes     string `json:"notes,omitempty"`
}

// RejectRecoveryResponse returns rejection result.
type RejectRecoveryResponse struct {
	SessionID  xid.ID    `json:"sessionId"`
	Rejected   bool      `json:"rejected"`
	RejectedAt time.Time `json:"rejectedAt"`
	Reason     string    `json:"reason"`
	Message    string    `json:"message"`
}

// ===== Analytics & Reporting =====

// GetRecoveryStatsRequest gets recovery statistics.
type GetRecoveryStatsRequest struct {
	OrganizationID string    `json:"organizationId,omitempty"`
	StartDate      time.Time `json:"startDate"`
	EndDate        time.Time `json:"endDate"`
}

// GetRecoveryStatsResponse returns statistics.
type GetRecoveryStatsResponse struct {
	TotalAttempts        int                    `json:"totalAttempts"`
	SuccessfulRecoveries int                    `json:"successfulRecoveries"`
	FailedRecoveries     int                    `json:"failedRecoveries"`
	PendingRecoveries    int                    `json:"pendingRecoveries"`
	SuccessRate          float64                `json:"successRate"`
	MethodStats          map[RecoveryMethod]int `json:"methodStats"`
	AverageRiskScore     float64                `json:"averageRiskScore"`
	HighRiskAttempts     int                    `json:"highRiskAttempts"`
	AdminReviewsRequired int                    `json:"adminReviewsRequired"`
}

// ===== Configuration =====

// UpdateRecoveryConfigRequest updates recovery configuration (admin).
type UpdateRecoveryConfigRequest struct {
	EnabledMethods       []RecoveryMethod `json:"enabledMethods,omitempty"`
	RequireMultipleSteps bool             `json:"requireMultipleSteps,omitempty"`
	MinimumStepsRequired int              `json:"minimumStepsRequired,omitempty"`
	RequireAdminReview   bool             `json:"requireAdminReview,omitempty"`
	RiskScoreThreshold   float64          `json:"riskScoreThreshold,omitempty"`
}

// GetRecoveryConfigResponse returns configuration.
type GetRecoveryConfigResponse struct {
	EnabledMethods       []RecoveryMethod `json:"enabledMethods"`
	RequireMultipleSteps bool             `json:"requireMultipleSteps"`
	MinimumStepsRequired int              `json:"minimumStepsRequired"`
	RequireAdminReview   bool             `json:"requireAdminReview"`
	RiskScoreThreshold   float64          `json:"riskScoreThreshold"`
}

// ===== Health Check =====

// HealthCheckResponse returns plugin health status.
type HealthCheckResponse struct {
	Healthy         bool              `json:"healthy"`
	Version         string            `json:"version"`
	EnabledMethods  []RecoveryMethod  `json:"enabledMethods"`
	ProvidersStatus map[string]string `json:"providersStatus,omitempty"`
	Message         string            `json:"message,omitempty"`
}

// ===== Common Types =====

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string         `json:"error"`
	Message string         `json:"message"`
	Code    string         `json:"code,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
