package backupauth

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// ProviderRegistry manages external verification service providers
type ProviderRegistry interface {
	// Email/SMS providers
	EmailProvider() EmailProvider
	SMSProvider() SMSProvider

	// Video verification providers
	VideoProvider() VideoProvider

	// Document verification providers
	DocumentProvider() DocumentProvider

	// Notification providers
	NotificationProvider() NotificationProvider
}

// EmailProvider handles email sending
type EmailProvider interface {
	SendVerificationEmail(ctx context.Context, to, code string, expiresIn time.Duration) error
	SendRecoveryNotification(ctx context.Context, to, subject, body string) error
}

// SMSProvider handles SMS sending
type SMSProvider interface {
	SendVerificationSMS(ctx context.Context, to, code string, expiresIn time.Duration) error
	SendRecoveryNotification(ctx context.Context, to, message string) error
}

// VideoProvider handles video verification sessions
type VideoProvider interface {
	CreateSession(ctx context.Context, userID xid.ID, scheduledAt time.Time) (*VideoSessionInfo, error)
	GetSession(ctx context.Context, sessionID string) (*VideoSessionInfo, error)
	StartSession(ctx context.Context, sessionID string) (*VideoSessionInfo, error)
	CompleteSession(ctx context.Context, sessionID string, result VideoSessionResult) error
	CancelSession(ctx context.Context, sessionID string) error
}

// VideoSessionInfo contains video session details
type VideoSessionInfo struct {
	SessionID      string
	JoinURL        string
	RecordingURL   string
	Status         string
	ScheduledAt    time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	LivenessScore  float64
	LivenessPassed bool
}

// VideoSessionResult contains verification result
type VideoSessionResult struct {
	Approved       bool
	LivenessPassed bool
	LivenessScore  float64
	Notes          string
	VerifierID     string
}

// DocumentProvider handles document verification
type DocumentProvider interface {
	VerifyDocument(ctx context.Context, req *DocumentVerificationRequest) (*DocumentVerificationResult, error)
	GetVerificationStatus(ctx context.Context, verificationID string) (*DocumentVerificationResult, error)
}

// DocumentVerificationRequest contains document verification request
type DocumentVerificationRequest struct {
	UserID       xid.ID
	DocumentType string
	FrontImage   []byte
	BackImage    []byte
	Selfie       []byte
}

// DocumentVerificationResult contains verification result
type DocumentVerificationResult struct {
	VerificationID   string
	Status           string // pending, verified, rejected
	ConfidenceScore  float64
	ExtractedData    map[string]interface{}
	ProviderResponse map[string]interface{}
	RejectionReason  string
}

// NotificationProvider handles notifications
type NotificationProvider interface {
	NotifyRecoveryStarted(ctx context.Context, userID xid.ID, sessionID xid.ID, method RecoveryMethod) error
	NotifyRecoveryCompleted(ctx context.Context, userID xid.ID, sessionID xid.ID) error
	NotifyRecoveryFailed(ctx context.Context, userID xid.ID, sessionID xid.ID, reason string) error
	NotifyAdminReviewRequired(ctx context.Context, sessionID xid.ID, userID xid.ID, riskScore float64) error
	NotifyHighRiskAttempt(ctx context.Context, userID xid.ID, riskScore float64) error
}

// DefaultProviderRegistry provides default implementations
type DefaultProviderRegistry struct {
	emailProvider        EmailProvider
	smsProvider          SMSProvider
	videoProvider        VideoProvider
	documentProvider     DocumentProvider
	notificationProvider NotificationProvider
}

// NewDefaultProviderRegistry creates a new provider registry
func NewDefaultProviderRegistry() *DefaultProviderRegistry {
	return &DefaultProviderRegistry{
		emailProvider:        &NoOpEmailProvider{},
		smsProvider:          &NoOpSMSProvider{},
		videoProvider:        &NoOpVideoProvider{},
		documentProvider:     &NoOpDocumentProvider{},
		notificationProvider: &NoOpNotificationProvider{},
	}
}

func (r *DefaultProviderRegistry) EmailProvider() EmailProvider {
	return r.emailProvider
}

func (r *DefaultProviderRegistry) SMSProvider() SMSProvider {
	return r.smsProvider
}

func (r *DefaultProviderRegistry) VideoProvider() VideoProvider {
	return r.videoProvider
}

func (r *DefaultProviderRegistry) DocumentProvider() DocumentProvider {
	return r.documentProvider
}

func (r *DefaultProviderRegistry) NotificationProvider() NotificationProvider {
	return r.notificationProvider
}

func (r *DefaultProviderRegistry) SetEmailProvider(provider EmailProvider) {
	r.emailProvider = provider
}

func (r *DefaultProviderRegistry) SetSMSProvider(provider SMSProvider) {
	r.smsProvider = provider
}

func (r *DefaultProviderRegistry) SetVideoProvider(provider VideoProvider) {
	r.videoProvider = provider
}

func (r *DefaultProviderRegistry) SetDocumentProvider(provider DocumentProvider) {
	r.documentProvider = provider
}

func (r *DefaultProviderRegistry) SetNotificationProvider(provider NotificationProvider) {
	r.notificationProvider = provider
}

// ===== No-Op Implementations =====

// NoOpEmailProvider is a no-op implementation
type NoOpEmailProvider struct{}

func (p *NoOpEmailProvider) SendVerificationEmail(ctx context.Context, to, code string, expiresIn time.Duration) error {
	// Log but don't actually send
	return nil
}

func (p *NoOpEmailProvider) SendRecoveryNotification(ctx context.Context, to, subject, body string) error {
	return nil
}

// NoOpSMSProvider is a no-op implementation
type NoOpSMSProvider struct{}

func (p *NoOpSMSProvider) SendVerificationSMS(ctx context.Context, to, code string, expiresIn time.Duration) error {
	return nil
}

func (p *NoOpSMSProvider) SendRecoveryNotification(ctx context.Context, to, message string) error {
	return nil
}

// NoOpVideoProvider is a no-op implementation
type NoOpVideoProvider struct{}

func (p *NoOpVideoProvider) CreateSession(ctx context.Context, userID xid.ID, scheduledAt time.Time) (*VideoSessionInfo, error) {
	return &VideoSessionInfo{
		SessionID:   xid.New().String(),
		JoinURL:     "https://example.com/video/session",
		Status:      "scheduled",
		ScheduledAt: scheduledAt,
	}, nil
}

func (p *NoOpVideoProvider) GetSession(ctx context.Context, sessionID string) (*VideoSessionInfo, error) {
	return &VideoSessionInfo{
		SessionID: sessionID,
		Status:    "pending",
	}, nil
}

func (p *NoOpVideoProvider) StartSession(ctx context.Context, sessionID string) (*VideoSessionInfo, error) {
	now := time.Now()
	return &VideoSessionInfo{
		SessionID: sessionID,
		Status:    "in_progress",
		StartedAt: &now,
	}, nil
}

func (p *NoOpVideoProvider) CompleteSession(ctx context.Context, sessionID string, result VideoSessionResult) error {
	return nil
}

func (p *NoOpVideoProvider) CancelSession(ctx context.Context, sessionID string) error {
	return nil
}

// NoOpDocumentProvider is a no-op implementation
type NoOpDocumentProvider struct{}

func (p *NoOpDocumentProvider) VerifyDocument(ctx context.Context, req *DocumentVerificationRequest) (*DocumentVerificationResult, error) {
	return &DocumentVerificationResult{
		VerificationID:  xid.New().String(),
		Status:          "pending",
		ConfidenceScore: 0.0,
	}, nil
}

func (p *NoOpDocumentProvider) GetVerificationStatus(ctx context.Context, verificationID string) (*DocumentVerificationResult, error) {
	return &DocumentVerificationResult{
		VerificationID: verificationID,
		Status:         "pending",
	}, nil
}

// NoOpNotificationProvider is a no-op implementation
type NoOpNotificationProvider struct{}

func (p *NoOpNotificationProvider) NotifyRecoveryStarted(ctx context.Context, userID xid.ID, sessionID xid.ID, method RecoveryMethod) error {
	return nil
}

func (p *NoOpNotificationProvider) NotifyRecoveryCompleted(ctx context.Context, userID xid.ID, sessionID xid.ID) error {
	return nil
}

func (p *NoOpNotificationProvider) NotifyRecoveryFailed(ctx context.Context, userID xid.ID, sessionID xid.ID, reason string) error {
	return nil
}

func (p *NoOpNotificationProvider) NotifyAdminReviewRequired(ctx context.Context, sessionID xid.ID, userID xid.ID, riskScore float64) error {
	return nil
}

func (p *NoOpNotificationProvider) NotifyHighRiskAttempt(ctx context.Context, userID xid.ID, riskScore float64) error {
	return nil
}
