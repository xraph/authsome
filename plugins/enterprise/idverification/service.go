package idverification

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// Service handles identity verification operations.
type Service struct {
	repo           Repository
	config         Config
	auditService   *audit.Service
	webhookService *webhook.Service
	providers      map[string]Provider // Provider interface for different KYC providers
}

// NewService creates a new identity verification service.
func NewService(
	repo Repository,
	config Config,
	auditService *audit.Service,
	webhookService *webhook.Service,
) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	s := &Service{
		repo:           repo,
		config:         config,
		auditService:   auditService,
		webhookService: webhookService,
		providers:      make(map[string]Provider),
	}

	// Initialize providers
	if config.Onfido.Enabled {
		provider, err := NewOnfidoProvider(config.Onfido)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Onfido provider: %w", err)
		}

		s.providers["onfido"] = provider
	}

	if config.Jumio.Enabled {
		provider, err := NewJumioProvider(config.Jumio)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Jumio provider: %w", err)
		}

		s.providers["jumio"] = provider
	}

	if config.StripeIdentity.Enabled {
		provider, err := NewStripeIdentityProvider(config.StripeIdentity)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Stripe Identity provider: %w", err)
		}

		s.providers["stripe_identity"] = provider
	}

	return s, nil
}

// CreateVerificationSession creates a new verification session for a user with V2 context.
func (s *Service) CreateVerificationSession(ctx context.Context, req *CreateSessionRequest) (*schema.IdentityVerificationSession, error) {
	// Check if user exists and is not blocked
	status, err := s.repo.GetUserVerificationStatus(ctx, req.AppID, req.OrganizationID, req.UserID)
	if err == nil && status != nil && status.IsBlocked {
		s.audit(ctx, "verification_session_blocked", req.UserID.String(), req.OrganizationID.String(), map[string]any{
			"reason": status.BlockReason,
			"app_id": req.AppID.String(),
		})

		return nil, ErrVerificationBlocked
	}

	// Check rate limits
	if s.config.RateLimitEnabled {
		count, err := s.repo.CountVerificationsByUser(ctx, req.AppID, req.UserID, time.Now().Add(-24*time.Hour))
		if err != nil {
			return nil, fmt.Errorf("failed to check rate limit: %w", err)
		}

		if count >= s.config.MaxVerificationsPerDay {
			return nil, ErrRateLimitExceeded
		}
	}

	// Get provider
	provider, err := s.getProvider(req.Provider)
	if err != nil {
		return nil, err
	}

	// Create provider session with V2 context
	providerSession, err := provider.CreateSession(ctx, &ProviderSessionRequest{
		AppID:          req.AppID,
		EnvironmentID:  req.EnvironmentID,
		OrganizationID: req.OrganizationID,
		UserID:         req.UserID,
		RequiredChecks: req.RequiredChecks,
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
		Metadata:       req.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("provider session creation failed: %w", err)
	}

	// Convert environment ID to string pointer for schema
	var envIDStr *string

	if req.EnvironmentID != nil && !req.EnvironmentID.IsNil() {
		str := req.EnvironmentID.String()
		envIDStr = &str
	}

	// Create session record with V2 context
	session := &schema.IdentityVerificationSession{
		ID:             xid.New().String(),
		AppID:          req.AppID.String(),
		EnvironmentID:  envIDStr,
		OrganizationID: req.OrganizationID.String(),
		UserID:         req.UserID.String(),
		Provider:       req.Provider,
		SessionURL:     providerSession.URL,
		SessionToken:   providerSession.Token,
		RequiredChecks: req.RequiredChecks,
		Config:         req.Config,
		Status:         "created",
		ExpiresAt:      time.Now().Add(s.config.SessionExpiryDuration),
		SuccessURL:     req.SuccessURL,
		CancelURL:      req.CancelURL,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.audit(ctx, "verification_session_created", req.UserID.String(), req.OrganizationID.String(), map[string]any{
		"session_id": session.ID,
		"provider":   req.Provider,
		"checks":     req.RequiredChecks,
		"app_id":     req.AppID.String(),
	})

	return session, nil
}

// GetVerificationSession retrieves a verification session with V2 context.
func (s *Service) GetVerificationSession(ctx context.Context, appID xid.ID, sessionID string) (*schema.IdentityVerificationSession, error) {
	session, err := s.repo.GetSessionByID(ctx, appID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, ErrSessionNotFound
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		if session.Status != "expired" {
			session.Status = "expired"
			session.UpdatedAt = time.Now()
			_ = s.repo.UpdateSession(ctx, session)
		}

		return session, ErrSessionExpired
	}

	return session, nil
}

// CreateVerification creates a new verification record with V2 context.
func (s *Service) CreateVerification(ctx context.Context, req *CreateVerificationRequest) (*schema.IdentityVerification, error) {
	// Check if user is blocked
	status, err := s.repo.GetUserVerificationStatus(ctx, req.AppID, req.OrganizationID, req.UserID)
	if err == nil && status != nil && status.IsBlocked {
		return nil, ErrVerificationBlocked
	}

	// Check max attempts
	if s.config.MaxVerificationAttempts > 0 {
		count, err := s.repo.CountVerificationsByUser(ctx, req.AppID, req.UserID, time.Now().Add(-24*time.Hour))
		if err != nil {
			return nil, fmt.Errorf("failed to check attempts: %w", err)
		}

		if count >= s.config.MaxVerificationAttempts {
			return nil, ErrMaxAttemptsReached
		}
	}

	// Convert environment ID to string pointer for schema
	var envIDStr *string

	if req.EnvironmentID != nil && !req.EnvironmentID.IsNil() {
		str := req.EnvironmentID.String()
		envIDStr = &str
	}

	verification := &schema.IdentityVerification{
		ID:               xid.New().String(),
		AppID:            req.AppID.String(),
		EnvironmentID:    envIDStr,
		OrganizationID:   req.OrganizationID.String(),
		UserID:           req.UserID.String(),
		Provider:         req.Provider,
		ProviderCheckID:  req.ProviderCheckID,
		VerificationType: req.VerificationType,
		Status:           "pending",
		DocumentType:     req.DocumentType,
		Metadata:         req.Metadata,
		IPAddress:        req.IPAddress,
		UserAgent:        req.UserAgent,
	}

	if s.config.VerificationExpiry > 0 {
		expiresAt := time.Now().Add(s.config.VerificationExpiry)
		verification.ExpiresAt = &expiresAt
	}

	if err := s.repo.CreateVerification(ctx, verification); err != nil {
		return nil, fmt.Errorf("failed to create verification: %w", err)
	}

	s.audit(ctx, "verification_created", req.UserID.String(), req.OrganizationID.String(), map[string]any{
		"verification_id": verification.ID,
		"type":            req.VerificationType,
		"provider":        req.Provider,
		"app_id":          req.AppID.String(),
	})

	return verification, nil
}

// ProcessVerificationResult processes the result from a provider with V2 context.
func (s *Service) ProcessVerificationResult(ctx context.Context, appID xid.ID, verificationID string, result *VerificationResult) error {
	verification, err := s.repo.GetVerificationByID(ctx, appID, verificationID)
	if err != nil {
		return fmt.Errorf("failed to get verification: %w", err)
	}

	if verification == nil {
		return ErrVerificationNotFound
	}

	// Update verification with result
	verification.Status = result.Status
	verification.IsVerified = result.IsVerified
	verification.RiskScore = result.RiskScore
	verification.RiskLevel = result.RiskLevel
	verification.ConfidenceScore = result.ConfidenceScore
	verification.RejectionReasons = result.RejectionReasons
	verification.FailureReason = result.FailureReason
	verification.ProviderData = result.ProviderData
	verification.UpdatedAt = time.Now()

	// Update personal information if provided
	if result.FirstName != "" {
		verification.FirstName = result.FirstName
	}

	if result.LastName != "" {
		verification.LastName = result.LastName
	}

	if result.DateOfBirth != nil {
		verification.DateOfBirth = result.DateOfBirth
		verification.Age = calculateAge(*result.DateOfBirth)
	}

	if result.DocumentNumber != "" {
		verification.DocumentNumber = result.DocumentNumber
	}

	if result.DocumentCountry != "" {
		verification.DocumentCountry = result.DocumentCountry
	}

	if result.Nationality != "" {
		verification.Nationality = result.Nationality
	}

	// Update AML/sanctions data
	verification.IsOnSanctionsList = result.IsOnSanctionsList
	verification.IsPEP = result.IsPEP
	verification.SanctionsDetails = result.SanctionsDetails

	// Update liveness data
	verification.LivenessScore = result.LivenessScore
	verification.IsLive = result.IsLive

	if result.IsVerified {
		now := time.Now()
		verification.VerifiedAt = &now
	}

	// Apply business rules
	if err := s.applyBusinessRules(verification); err != nil {
		verification.Status = "failed"
		verification.FailureReason = err.Error()
	}

	if err := s.repo.UpdateVerification(ctx, verification); err != nil {
		return fmt.Errorf("failed to update verification: %w", err)
	}

	// Update user verification status
	if err := s.updateUserVerificationStatus(ctx, verification); err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	// Send webhook
	if s.config.WebhooksEnabled {
		go s.sendWebhook(context.Background(), verification)
	}

	// Audit log
	s.audit(ctx, "verification_processed", verification.UserID, verification.OrganizationID, map[string]any{
		"verification_id": verification.ID,
		"status":          verification.Status,
		"is_verified":     verification.IsVerified,
		"risk_level":      verification.RiskLevel,
	})

	return nil
}

// GetVerification retrieves a verification by ID with V2 context.
func (s *Service) GetVerification(ctx context.Context, appID xid.ID, id string) (*schema.IdentityVerification, error) {
	verification, err := s.repo.GetVerificationByID(ctx, appID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}

	if verification == nil {
		return nil, ErrVerificationNotFound
	}

	return verification, nil
}

// GetUserVerifications retrieves all verifications for a user with V2 context.
func (s *Service) GetUserVerifications(ctx context.Context, appID xid.ID, userID xid.ID, limit, offset int) ([]*schema.IdentityVerification, error) {
	return s.repo.GetVerificationsByUserID(ctx, appID, userID, limit, offset)
}

// GetUserVerificationStatus retrieves the verification status for a user with V2 context.
func (s *Service) GetUserVerificationStatus(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID) (*schema.UserVerificationStatus, error) {
	status, err := s.repo.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user status: %w", err)
	}

	if status == nil {
		// Create default status with V2 context
		status = &schema.UserVerificationStatus{
			ID:                xid.New().String(),
			AppID:             appID.String(),
			EnvironmentID:     nil, // Will be set when first verification is created
			OrganizationID:    orgID.String(),
			UserID:            userID.String(),
			IsVerified:        false,
			VerificationLevel: "none",
			OverallRiskLevel:  "unknown",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
	}

	return status, nil
}

// RequestReverification initiates a re-verification for a user with V2 context.
func (s *Service) RequestReverification(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID, reason string) error {
	if !s.config.EnableReverification {
		return errs.BadRequest("reverification is not enabled")
	}

	status, err := s.repo.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to get user status: %w", err)
	}

	if status == nil {
		return errs.NotFound("user has no verification status")
	}

	status.RequiresReverification = true
	status.UpdatedAt = time.Now()

	if err := s.repo.UpdateUserVerificationStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	s.audit(ctx, "reverification_requested", userID.String(), orgID.String(), map[string]any{
		"reason": reason,
		"app_id": appID.String(),
	})

	return nil
}

// BlockUser blocks a user from verification with V2 context.
func (s *Service) BlockUser(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID, reason string) error {
	status, err := s.repo.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to get user status: %w", err)
	}

	if status == nil {
		status = &schema.UserVerificationStatus{
			ID:             xid.New().String(),
			AppID:          appID.String(),
			OrganizationID: orgID.String(),
			UserID:         userID.String(),
			CreatedAt:      time.Now(),
		}
	}

	status.IsBlocked = true
	status.BlockReason = reason
	now := time.Now()
	status.BlockedAt = &now
	status.UpdatedAt = time.Now()

	if status.CreatedAt.IsZero() {
		if err := s.repo.CreateUserVerificationStatus(ctx, status); err != nil {
			return fmt.Errorf("failed to create status: %w", err)
		}
	} else {
		if err := s.repo.UpdateUserVerificationStatus(ctx, status); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
	}

	s.audit(ctx, "user_blocked", userID.String(), orgID.String(), map[string]any{
		"reason": reason,
		"app_id": appID.String(),
	})

	return nil
}

// UnblockUser unblocks a user with V2 context.
func (s *Service) UnblockUser(ctx context.Context, appID xid.ID, orgID xid.ID, userID xid.ID) error {
	status, err := s.repo.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to get user status: %w", err)
	}

	if status == nil {
		return errs.NotFound("user has no verification status")
	}

	status.IsBlocked = false
	status.BlockReason = ""
	status.BlockedAt = nil
	status.UpdatedAt = time.Now()

	if err := s.repo.UpdateUserVerificationStatus(ctx, status); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	s.audit(ctx, "user_unblocked", userID.String(), orgID.String(), map[string]any{
		"app_id": appID.String(),
	})

	return nil
}

// Helper methods

func (s *Service) getProvider(name string) (Provider, error) {
	if name == "" {
		name = s.config.DefaultProvider
	}

	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found or not enabled", name)
	}

	return provider, nil
}

func (s *Service) applyBusinessRules(verification *schema.IdentityVerification) error {
	// Check risk score
	if s.config.AutoRejectHighRisk && verification.RiskScore > s.config.MaxAllowedRiskScore {
		return ErrHighRiskDetected
	}

	// Check confidence score
	if verification.ConfidenceScore < s.config.MinConfidenceScore {
		return fmt.Errorf("confidence score too low: %d < %d", verification.ConfidenceScore, s.config.MinConfidenceScore)
	}

	// Check sanctions list
	if verification.IsOnSanctionsList {
		return ErrSanctionsListMatch
	}

	// Check PEP
	if verification.IsPEP && s.config.AutoRejectHighRisk {
		return ErrPEPDetected
	}

	// Check age
	if s.config.RequireAgeVerification && verification.Age > 0 && verification.Age < s.config.MinimumAge {
		return ErrAgeBelowMinimum
	}

	// Check document type
	if verification.DocumentType != "" {
		allowed := slices.Contains(s.config.AcceptedDocuments, verification.DocumentType)

		if !allowed {
			return ErrDocumentNotSupported
		}
	}

	// Check country
	if len(s.config.AcceptedCountries) > 0 && verification.DocumentCountry != "" {
		allowed := slices.Contains(s.config.AcceptedCountries, verification.DocumentCountry)

		if !allowed {
			return ErrCountryNotSupported
		}
	}

	return nil
}

func (s *Service) updateUserVerificationStatus(ctx context.Context, verification *schema.IdentityVerification) error {
	// Convert string IDs to xid.ID for repository call
	appID, _ := xid.FromString(verification.AppID)
	orgID, _ := xid.FromString(verification.OrganizationID)
	userID, _ := xid.FromString(verification.UserID)

	status, err := s.repo.GetUserVerificationStatus(ctx, appID, orgID, userID)
	if err != nil {
		return err
	}

	if status == nil {
		status = &schema.UserVerificationStatus{
			ID:                xid.New().String(),
			AppID:             verification.AppID,
			EnvironmentID:     verification.EnvironmentID,
			OrganizationID:    verification.OrganizationID,
			UserID:            verification.UserID,
			VerificationLevel: "none",
			CreatedAt:         time.Now(),
		}
	}

	// Update based on verification type
	switch verification.VerificationType {
	case "document":
		status.DocumentVerified = verification.IsVerified
		if verification.IsVerified {
			status.LastDocumentVerificationID = verification.ID
		}
	case "liveness":
		status.LivenessVerified = verification.IsVerified
		if verification.IsVerified {
			status.LastLivenessVerificationID = verification.ID
		}
	case "age":
		status.AgeVerified = verification.IsVerified
	case "aml":
		status.AMLScreened = true

		status.AMLClear = !verification.IsOnSanctionsList && !verification.IsPEP
		if status.AMLScreened {
			status.LastAMLVerificationID = verification.ID
		}
	}

	// Update overall status
	status.IsVerified = status.DocumentVerified &&
		(!s.config.RequireLivenessDetection || status.LivenessVerified) &&
		(!s.config.RequireAgeVerification || status.AgeVerified) &&
		(!s.config.RequireAMLScreening || (status.AMLScreened && status.AMLClear))

	if status.IsVerified {
		now := time.Now()
		status.LastVerifiedAt = &now
		status.VerificationLevel = "full"

		if s.config.VerificationExpiry > 0 {
			expiry := time.Now().Add(s.config.VerificationExpiry)
			status.VerificationExpiry = &expiry
		}
	}

	// Update risk level
	if verification.RiskLevel != "" {
		status.OverallRiskLevel = verification.RiskLevel
	}

	status.UpdatedAt = time.Now()

	if status.CreatedAt.IsZero() {
		return s.repo.CreateUserVerificationStatus(ctx, status)
	}

	return s.repo.UpdateUserVerificationStatus(ctx, status)
}

func (s *Service) sendWebhook(ctx context.Context, verification *schema.IdentityVerification) {
	eventType := "verification." + verification.Status

	// Check if this event type should be sent
	if len(s.config.WebhookEvents) > 0 {
		found := slices.Contains(s.config.WebhookEvents, eventType)

		if !found {
			return
		}
	}

	// TODO: Implement webhook delivery using webhook service
	// This would require registering webhooks and using the Deliver method
	// For now, this is a placeholder for webhook integration
}

func (s *Service) audit(ctx context.Context, action, userID, orgID string, metadata map[string]any) {
	// TODO: Implement proper audit logging integration
	// This would require converting the metadata to string and using proper user ID format
	// For now, this is a placeholder for audit integration
}

func calculateAge(dob time.Time) int {
	now := time.Now()

	years := now.Year() - dob.Year()
	if now.YearDay() < dob.YearDay() {
		years--
	}

	return years
}
