package idverification

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// REQUEST DTOs - V2 Architecture (App → Environment → Organization)
// =============================================================================

// CreateSessionRequest represents a request to create a verification session
type CreateSessionRequest struct {
	// V2 Context
	AppID          xid.ID
	EnvironmentID  *xid.ID
	OrganizationID xid.ID
	UserID         xid.ID

	// Session configuration
	Provider       string   // onfido, jumio, stripe_identity
	RequiredChecks []string // document, liveness, age, aml
	SuccessURL     string
	CancelURL      string
	Config         map[string]interface{}
	Metadata       map[string]interface{}

	// Tracking
	IPAddress string
	UserAgent string
}

// CreateVerificationRequest represents a request to create a verification
type CreateVerificationRequest struct {
	// V2 Context
	AppID          xid.ID
	EnvironmentID  *xid.ID
	OrganizationID xid.ID
	UserID         xid.ID

	// Verification details
	Provider         string
	ProviderCheckID  string
	VerificationType string
	DocumentType     string
	Metadata         map[string]interface{}

	// Tracking
	IPAddress string
	UserAgent string
}

// ReverifyRequest represents a request for re-verification
type ReverifyRequest struct {
	Reason string `json:"reason,omitempty"`
}

// BlockUserRequest represents admin request to block a user
type BlockUserRequest struct {
	Reason string `json:"reason"`
}

// UnblockUserRequest represents admin request to unblock a user
type UnblockUserRequest struct {
	// No fields needed, user ID comes from URL
}

// CreateSessionHTTPRequest represents the HTTP request body for session creation
type CreateSessionHTTPRequest struct {
	Provider       string                 `json:"provider"`
	RequiredChecks []string               `json:"requiredChecks,omitempty"`
	SuccessURL     string                 `json:"successUrl,omitempty"`
	CancelURL      string                 `json:"cancelUrl,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// VerificationResult represents the result from a provider
type VerificationResult struct {
	Status           string
	IsVerified       bool
	RiskScore        int
	RiskLevel        string
	ConfidenceScore  int
	RejectionReasons []string
	FailureReason    string
	ProviderData     map[string]interface{}

	// Personal information
	FirstName       string
	LastName        string
	DateOfBirth     *time.Time
	DocumentNumber  string
	DocumentCountry string
	Nationality     string
	Gender          string

	// AML/Sanctions
	IsOnSanctionsList bool
	IsPEP             bool
	SanctionsDetails  string

	// Liveness
	LivenessScore int
	IsLive        bool
}

// =============================================================================
// RESPONSE DTOs - Structured responses replacing map[string]interface{}
// =============================================================================

// VerificationSessionResponse represents a single verification session response
type VerificationSessionResponse struct {
	Session *schema.IdentityVerificationSession `json:"session"`
}

// VerificationResponse represents a single verification response
type VerificationResponse struct {
	Verification *schema.IdentityVerification `json:"verification"`
}

// VerificationListResponse represents a list of verifications with pagination
type VerificationListResponse struct {
	Verifications []*schema.IdentityVerification `json:"verifications"`
	Limit         int                            `json:"limit"`
	Offset        int                            `json:"offset"`
	Total         int                            `json:"total,omitempty"`
}

// UserVerificationStatusResponse represents a user's verification status
type UserVerificationStatusResponse struct {
	Status *schema.UserVerificationStatus `json:"status"`
}

// WebhookResponse represents a webhook processing response
type WebhookResponse struct {
	Received  bool   `json:"received"`
	ProcessedStatus string `json:"status,omitempty"`
}

// Provider interface for KYC providers
type Provider interface {
	// CreateSession creates a verification session with the provider
	CreateSession(ctx context.Context, req *ProviderSessionRequest) (*ProviderSession, error)

	// GetSession retrieves session status from the provider
	GetSession(ctx context.Context, sessionID string) (*ProviderSession, error)

	// GetCheck retrieves a verification check result
	GetCheck(ctx context.Context, checkID string) (*ProviderCheckResult, error)

	// VerifyWebhook verifies a webhook signature
	VerifyWebhook(signature, payload string) (bool, error)

	// ParseWebhook parses a webhook payload
	ParseWebhook(payload []byte) (*WebhookPayload, error)

	// GetProviderName returns the provider name
	GetProviderName() string
}

// ProviderSessionRequest represents a provider session creation request
type ProviderSessionRequest struct {
	// V2 Context
	AppID          xid.ID
	EnvironmentID  *xid.ID
	OrganizationID xid.ID
	UserID         xid.ID

	// Session configuration
	RequiredChecks []string
	SuccessURL     string
	CancelURL      string
	Metadata       map[string]interface{}
}

// ProviderSession represents a provider verification session
type ProviderSession struct {
	ID        string
	URL       string // URL for the user to complete verification
	Token     string // Session token
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// ProviderCheckResult represents the result of a provider check
type ProviderCheckResult struct {
	ID              string
	Type            string // document, liveness, aml
	Status          string
	Result          string // clear, consider, rejected
	SubResults      []CheckSubResult
	Properties      map[string]interface{}
	RiskScore       int
	ConfidenceScore int

	// Document-specific
	DocumentType    string
	DocumentCountry string
	DocumentNumber  string
	DocumentExpiry  *time.Time
	IsDocumentValid bool

	// Personal data extraction
	FirstName   string
	LastName    string
	DateOfBirth *time.Time
	Gender      string
	Nationality string

	// Liveness-specific
	IsLive        bool
	LivenessScore int

	// AML-specific
	IsOnSanctionsList bool
	IsPEP             bool
	Matches           []AMLMatch

	CreatedAt   time.Time
	CompletedAt *time.Time
}

// CheckSubResult represents a sub-result within a check
type CheckSubResult struct {
	Name   string
	Result string
	Reason string
}

// AMLMatch represents a sanctions/PEP match
type AMLMatch struct {
	MatchType   string // sanction, pep, adverse_media
	Name        string
	Score       float64
	Source      string
	Description string
}

// WebhookPayload represents a parsed webhook from a provider
type WebhookPayload struct {
	EventType  string
	CheckID    string
	SessionID  string
	Status     string
	Result     *ProviderCheckResult
	Timestamp  time.Time
	RawPayload map[string]interface{}
}
