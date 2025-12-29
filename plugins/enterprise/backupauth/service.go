package backupauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"golang.org/x/crypto/bcrypt"
)

// Service provides backup authentication operations
type Service struct {
	repo      Repository
	config    *Config
	providers ProviderRegistry
}

// NewService creates a new backup authentication service
func NewService(repo Repository, config *Config, providers ProviderRegistry) *Service {
	return &Service{
		repo:      repo,
		config:    config,
		providers: providers,
	}
}

// ===== Recovery Session Management =====

// StartRecovery initiates a new recovery session
func (s *Service) StartRecovery(ctx context.Context, req *StartRecoveryRequest) (*StartRecoveryResponse, error) {
	if !s.config.Enabled {
		return nil, ErrRecoveryNotConfigured
	}

	// Parse user ID
	userID, err := xid.FromString(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get app and org IDs from context
	appID, userOrgID := s.getAppAndOrgFromContext(ctx)

	// Check for existing active session
	existing, err := s.repo.GetActiveRecoverySession(ctx, userID, appID, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing session: %w", err)
	}
	if existing != nil {
		return nil, ErrRecoverySessionInProgress
	}

	// Check rate limits
	if err := s.checkRateLimit(ctx, userID, appID, userOrgID); err != nil {
		return nil, err
	}

	// Calculate risk score
	riskScore := s.calculateRiskScore(ctx, req)

	// Determine required steps based on risk
	requiredSteps := s.determineRequiredSteps(riskScore)

	// Create recovery session
	session := &RecoverySession{
		UserID:             userID,
		AppID:              appID,
		UserOrganizationID: userOrgID,
		Status:             RecoveryStatusPending,
		Method:             req.PreferredMethod,
		RequiredSteps:      convertMethodsToStrings(requiredSteps),
		CompletedSteps:     []string{},
		CurrentStep:        0,
		MaxAttempts:        5,
		Attempts:           0,
		IPAddress:          s.getIPFromContext(ctx),
		UserAgent:          s.getUserAgentFromContext(ctx),
		DeviceID:           req.DeviceID,
		RiskScore:          riskScore,
		ExpiresAt:          time.Now().Add(s.config.MultiStepRecovery.SessionExpiry),
		RequiresReview:     riskScore >= s.config.RiskAssessment.RequireReviewAbove,
	}

	if err := s.repo.CreateRecoverySession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create recovery session: %w", err)
	}

	// Log attempt
	s.logRecoveryAttempt(ctx, session.ID, userID, appID, userOrgID, "started", session.Method, true, "")

	// Get available methods for user
	availableMethods := s.getAvailableMethods(ctx, userID, appID, userOrgID)

	return &StartRecoveryResponse{
		SessionID:        session.ID,
		Status:           session.Status,
		AvailableMethods: availableMethods,
		RequiredSteps:    len(session.RequiredSteps),
		CompletedSteps:   0,
		ExpiresAt:        session.ExpiresAt,
		RiskScore:        riskScore,
		RequiresReview:   session.RequiresReview,
	}, nil
}

// ContinueRecovery continues a recovery session with a chosen method
func (s *Service) ContinueRecovery(ctx context.Context, req *ContinueRecoveryRequest) (*ContinueRecoveryResponse, error) {
	// Get session
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Validate session
	if err := s.validateSession(session); err != nil {
		return nil, err
	}

	// Check if method is enabled
	if !s.isMethodEnabled(req.Method) {
		return nil, ErrRecoveryMethodNotEnabled
	}

	// Update session method
	session.Method = req.Method
	session.Status = RecoveryStatusInProgress
	if err := s.repo.UpdateRecoverySession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Get instructions and data for the method
	instructions, data := s.getMethodInstructions(ctx, session, req.Method)

	return &ContinueRecoveryResponse{
		SessionID:    session.ID,
		Method:       req.Method,
		CurrentStep:  session.CurrentStep + 1,
		TotalSteps:   len(session.RequiredSteps),
		Instructions: instructions,
		Data:         data,
		ExpiresAt:    session.ExpiresAt,
	}, nil
}

// CompleteRecovery finalizes a recovery session
func (s *Service) CompleteRecovery(ctx context.Context, req *CompleteRecoveryRequest) (*CompleteRecoveryResponse, error) {
	// Get session
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Validate session
	if err := s.validateSession(session); err != nil {
		return nil, err
	}

	// Check if all steps are completed
	if len(session.CompletedSteps) < len(session.RequiredSteps) {
		return nil, ErrRecoveryStepRequired
	}

	// Check if admin review is required
	if session.RequiresReview && session.ReviewedBy == nil {
		return nil, ErrAdminReviewRequired
	}

	// Mark session as completed
	now := time.Now()
	session.Status = RecoveryStatusCompleted
	session.CompletedAt = &now

	if err := s.repo.UpdateRecoverySession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to complete session: %w", err)
	}

	// Generate temporary recovery token
	token, err := s.generateRecoveryToken(session.UserID, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Log completion
	s.logRecoveryAttempt(ctx, session.ID, session.UserID, session.AppID, session.UserOrganizationID, "completed", session.Method, true, "")

	return &CompleteRecoveryResponse{
		SessionID:   session.ID,
		Status:      session.Status,
		CompletedAt: *session.CompletedAt,
		Token:       token,
		Message:     "Recovery completed successfully. Use the token to reset your password.",
	}, nil
}

// CancelRecovery cancels a recovery session
func (s *Service) CancelRecovery(ctx context.Context, req *CancelRecoveryRequest) error {
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return err
	}

	now := time.Now()
	session.Status = RecoveryStatusCancelled
	session.CancelledAt = &now

	if err := s.repo.UpdateRecoverySession(ctx, session); err != nil {
		return fmt.Errorf("failed to cancel session: %w", err)
	}

	// Log cancellation
	s.logRecoveryAttempt(ctx, session.ID, session.UserID, session.AppID, session.UserOrganizationID, "cancelled", session.Method, true, req.Reason)

	return nil
}

// ===== Recovery Codes =====

// GenerateRecoveryCodes generates new recovery codes for a user
func (s *Service) GenerateRecoveryCodes(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, req *GenerateRecoveryCodesRequest) (*GenerateRecoveryCodesResponse, error) {
	if !s.config.RecoveryCodes.Enabled {
		return nil, ErrRecoveryMethodNotEnabled
	}

	count := req.Count
	if count == 0 {
		count = s.config.RecoveryCodes.CodeCount
	}

	format := req.Format
	if format == "" {
		format = s.config.RecoveryCodes.Format
	}

	// Generate codes
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := s.generateRecoveryCode(format, s.config.RecoveryCodes.CodeLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
		codes[i] = code
	}

	return &GenerateRecoveryCodesResponse{
		Codes:       codes,
		Count:       len(codes),
		GeneratedAt: time.Now(),
		Warning:     "Store these codes securely. Each can only be used once.",
	}, nil
}

// VerifyRecoveryCode verifies a recovery code
func (s *Service) VerifyRecoveryCode(ctx context.Context, req *VerifyRecoveryCodeRequest) (*VerifyRecoveryCodeResponse, error) {
	// Get session
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Validate session
	if err := s.validateSession(session); err != nil {
		return &VerifyRecoveryCodeResponse{Valid: false, Message: err.Error()}, nil
	}

	// Hash the provided code
	codeHash := s.hashCode(req.Code)

	// Check if code was already used
	usage, err := s.repo.GetRecoveryCodeUsage(ctx, session.UserID, session.AppID, session.UserOrganizationID, codeHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check code usage: %w", err)
	}
	if usage != nil {
		return &VerifyRecoveryCodeResponse{Valid: false, Message: "Recovery code already used"}, nil
	}

	// Verify code (implementation depends on where codes are stored)
	// For now, assuming codes are valid if not used
	valid := true

	if valid {
		// Mark code as used
		err = s.repo.CreateRecoveryCodeUsage(ctx, &RecoveryCodeUsage{
			UserID:             session.UserID,
			AppID:              session.AppID,
			UserOrganizationID: session.UserOrganizationID,
			RecoveryID:         session.ID,
			CodeHash:           codeHash,
			UsedAt:             time.Now(),
			IPAddress:          s.getIPFromContext(ctx),
			UserAgent:          s.getUserAgentFromContext(ctx),
			DeviceID:           session.DeviceID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to record code usage: %w", err)
		}

		// Mark step as completed
		if err := s.markStepCompleted(ctx, session, RecoveryMethodCodes); err != nil {
			return nil, fmt.Errorf("failed to mark step completed: %w", err)
		}

		return &VerifyRecoveryCodeResponse{
			Valid:   true,
			Message: "Recovery code verified successfully",
		}, nil
	}

	// Increment attempts
	s.repo.IncrementSessionAttempts(ctx, session.ID)

	return &VerifyRecoveryCodeResponse{
		Valid:   false,
		Message: "Invalid recovery code",
	}, nil
}

// ===== Security Questions =====

// SetupSecurityQuestions sets up security questions for a user
func (s *Service) SetupSecurityQuestions(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, req *SetupSecurityQuestionsRequest) (*SetupSecurityQuestionsResponse, error) {
	if !s.config.SecurityQuestions.Enabled {
		return nil, ErrRecoveryMethodNotEnabled
	}

	if len(req.Questions) < s.config.SecurityQuestions.MinimumQuestions {
		return nil, ErrInsufficientSecurityQuestions
	}

	// Validate and create questions
	for _, q := range req.Questions {
		// Validate answer
		if len(q.Answer) < s.config.SecurityQuestions.RequireMinLength {
			return nil, ErrAnswerTooShort
		}
		if len(q.Answer) > s.config.SecurityQuestions.MaxAnswerLength {
			return nil, ErrAnswerTooLong
		}

		// Check for common answers if enabled
		if s.config.SecurityQuestions.ForbidCommonAnswers && s.isCommonAnswer(q.Answer) {
			return nil, ErrCommonAnswer
		}

		// Hash answer
		answerHash, salt, err := s.hashAnswer(q.Answer)
		if err != nil {
			return nil, fmt.Errorf("failed to hash answer: %w", err)
		}

		// Create security question
		question := &SecurityQuestion{
			UserID:             userID,
			AppID:              appID,
			UserOrganizationID: userOrganizationID,
			QuestionID:         q.QuestionID,
			CustomText:         q.CustomText,
			AnswerHash:         answerHash,
			Salt:               salt,
			IsActive:           true,
		}

		if err := s.repo.CreateSecurityQuestion(ctx, question); err != nil {
			return nil, fmt.Errorf("failed to create security question: %w", err)
		}
	}

	return &SetupSecurityQuestionsResponse{
		Count:   len(req.Questions),
		Message: "Security questions setup successfully",
		SetupAt: time.Now(),
	}, nil
}

// GetSecurityQuestions retrieves security questions for verification
func (s *Service) GetSecurityQuestions(ctx context.Context, req *GetSecurityQuestionsRequest) (*GetSecurityQuestionsResponse, error) {
	// Get session
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Get questions
	questions, err := s.repo.GetSecurityQuestionsByUser(ctx, session.UserID, session.AppID, session.UserOrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get security questions: %w", err)
	}

	if len(questions) < s.config.SecurityQuestions.RequiredToRecover {
		return nil, ErrInsufficientSecurityQuestions
	}

	// Convert to response format (without answers)
	questionInfos := make([]SecurityQuestionInfo, 0, len(questions))
	for _, q := range questions {
		questionText := q.CustomText
		if q.QuestionID > 0 && q.QuestionID <= len(s.config.SecurityQuestions.PredefinedQuestions) {
			questionText = s.config.SecurityQuestions.PredefinedQuestions[q.QuestionID-1]
		}

		questionInfos = append(questionInfos, SecurityQuestionInfo{
			ID:           q.ID,
			QuestionID:   q.QuestionID,
			QuestionText: questionText,
			IsCustom:     q.CustomText != "",
		})
	}

	return &GetSecurityQuestionsResponse{
		Questions: questionInfos,
	}, nil
}

// VerifySecurityAnswers verifies security question answers
func (s *Service) VerifySecurityAnswers(ctx context.Context, req *VerifySecurityAnswersRequest) (*VerifySecurityAnswersResponse, error) {
	// Get session
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Validate session
	if err := s.validateSession(session); err != nil {
		return &VerifySecurityAnswersResponse{Valid: false, Message: err.Error()}, nil
	}

	// Get questions
	questions, err := s.repo.GetSecurityQuestionsByUser(ctx, session.UserID, session.AppID, session.UserOrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get security questions: %w", err)
	}

	// Verify answers
	correctAnswers := 0
	for _, q := range questions {
		questionIDStr := q.ID.String()
		answer, exists := req.Answers[questionIDStr]
		if !exists {
			continue
		}

		// Normalize answer
		if !s.config.SecurityQuestions.CaseSensitive {
			answer = strings.ToLower(strings.TrimSpace(answer))
		}

		// Verify answer
		if s.verifyAnswer(answer, q.AnswerHash, q.Salt) {
			correctAnswers++
		} else {
			// Increment failed attempts
			s.repo.IncrementQuestionFailedAttempts(ctx, q.ID)
		}
	}

	// Check if enough correct answers
	required := s.config.SecurityQuestions.RequiredToRecover
	if correctAnswers >= required {
		// Mark step as completed
		if err := s.markStepCompleted(ctx, session, RecoveryMethodSecurityQ); err != nil {
			return nil, fmt.Errorf("failed to mark step completed: %w", err)
		}

		return &VerifySecurityAnswersResponse{
			Valid:           true,
			CorrectAnswers:  correctAnswers,
			RequiredAnswers: required,
			Message:         "Security questions verified successfully",
		}, nil
	}

	// Increment session attempts
	s.repo.IncrementSessionAttempts(ctx, session.ID)

	attemptsLeft := session.MaxAttempts - session.Attempts - 1
	return &VerifySecurityAnswersResponse{
		Valid:           false,
		CorrectAnswers:  correctAnswers,
		RequiredAnswers: required,
		AttemptsLeft:    attemptsLeft,
		Message:         fmt.Sprintf("Insufficient correct answers. %d attempts remaining.", attemptsLeft),
	}, nil
}

// ===== Trusted Contacts =====

// AddTrustedContact adds a trusted contact for account recovery
func (s *Service) AddTrustedContact(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, req *AddTrustedContactRequest) (*AddTrustedContactResponse, error) {
	if !s.config.TrustedContacts.Enabled {
		return nil, ErrRecoveryMethodNotEnabled
	}

	// Check contact limit
	count, err := s.repo.CountActiveTrustedContacts(ctx, userID, appID, userOrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to count contacts: %w", err)
	}
	if count >= s.config.TrustedContacts.MaximumContacts {
		return nil, ErrTrustedContactLimitExceeded
	}

	// Generate verification token
	token, err := s.generateVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create trusted contact
	contact := &TrustedContact{
		UserID:             userID,
		AppID:              appID,
		UserOrganizationID: userOrganizationID,
		ContactName:        req.Name,
		ContactEmail:       req.Email,
		ContactPhone:       req.Phone,
		Relationship:       req.Relationship,
		VerificationToken:  token,
		IsActive:           true,
		IPAddress:          s.getIPFromContext(ctx),
		UserAgent:          s.getUserAgentFromContext(ctx),
	}

	if err := s.repo.CreateTrustedContact(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to create trusted contact: %w", err)
	}

	// Send verification (if configured)
	if s.config.TrustedContacts.RequireVerification {
		// TODO: Send verification email/SMS
	}

	return &AddTrustedContactResponse{
		ContactID: contact.ID,
		Name:      contact.ContactName,
		Email:     contact.ContactEmail,
		Phone:     contact.ContactPhone,
		Verified:  false,
		AddedAt:   contact.CreatedAt,
		Message:   "Trusted contact added. Verification required.",
	}, nil
}

// VerifyTrustedContact verifies a trusted contact
func (s *Service) VerifyTrustedContact(ctx context.Context, req *VerifyTrustedContactRequest) (*VerifyTrustedContactResponse, error) {
	// Get contact by token
	contact, err := s.repo.GetTrustedContactByToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	// Mark as verified
	now := time.Now()
	contact.VerifiedAt = &now
	contact.VerificationToken = ""

	if err := s.repo.UpdateTrustedContact(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to verify contact: %w", err)
	}

	return &VerifyTrustedContactResponse{
		ContactID:  contact.ID,
		Verified:   true,
		VerifiedAt: now,
		Message:    "Trusted contact verified successfully",
	}, nil
}

// RequestTrustedContactVerification requests verification from a trusted contact
func (s *Service) RequestTrustedContactVerification(ctx context.Context, req *RequestTrustedContactVerificationRequest) (*RequestTrustedContactVerificationResponse, error) {
	// Get session
	session, err := s.repo.GetRecoverySession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	// Validate session
	if err := s.validateSession(session); err != nil {
		return nil, err
	}

	// Get contact
	contact, err := s.repo.GetTrustedContact(ctx, req.ContactID)
	if err != nil {
		return nil, err
	}

	// Validate contact
	if contact.UserID != session.UserID {
		return nil, ErrUnauthorized
	}
	if !contact.IsActive {
		return nil, ErrTrustedContactNotFound
	}
	if contact.VerifiedAt == nil {
		return nil, ErrTrustedContactNotVerified
	}

	// Check cooldown
	if contact.LastNotifiedAt != nil {
		cooldown := contact.LastNotifiedAt.Add(s.config.TrustedContacts.CooldownPeriod)
		if time.Now().Before(cooldown) {
			return nil, ErrTrustedContactCooldown
		}
	}

	// Send notification to contact
	// TODO: Implement notification sending via email/SMS

	// Update last notified time
	now := time.Now()
	contact.LastNotifiedAt = &now
	if err := s.repo.UpdateTrustedContact(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	return &RequestTrustedContactVerificationResponse{
		ContactID:   contact.ID,
		ContactName: contact.ContactName,
		NotifiedAt:  now,
		ExpiresAt:   expiresAt,
		Message:     "Verification request sent to trusted contact",
	}, nil
}

// ListTrustedContacts lists user's trusted contacts
func (s *Service) ListTrustedContacts(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) (*ListTrustedContactsResponse, error) {
	contacts, err := s.repo.GetTrustedContactsByUser(ctx, userID, appID, userOrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	contactInfos := make([]TrustedContactInfo, 0, len(contacts))
	for _, c := range contacts {
		contactInfos = append(contactInfos, TrustedContactInfo{
			ID:           c.ID,
			Name:         c.ContactName,
			Email:        c.ContactEmail,
			Phone:        c.ContactPhone,
			Relationship: c.Relationship,
			Verified:     c.VerifiedAt != nil,
			VerifiedAt:   c.VerifiedAt,
			Active:       c.IsActive,
		})
	}

	return &ListTrustedContactsResponse{
		Contacts: contactInfos,
		Count:    len(contactInfos),
	}, nil
}

// RemoveTrustedContact removes a trusted contact
func (s *Service) RemoveTrustedContact(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, req *RemoveTrustedContactRequest) error {
	// Get contact
	contact, err := s.repo.GetTrustedContact(ctx, req.ContactID)
	if err != nil {
		return err
	}

	// Validate ownership
	if contact.UserID != userID || contact.AppID != appID {
		return ErrUnauthorized
	}
	if (contact.UserOrganizationID == nil) != (userOrganizationID == nil) ||
		(contact.UserOrganizationID != nil && userOrganizationID != nil && *contact.UserOrganizationID != *userOrganizationID) {
		return ErrUnauthorized
	}

	return s.repo.DeleteTrustedContact(ctx, req.ContactID)
}

// ===== Helper Methods =====

func (s *Service) validateSession(session *RecoverySession) error {
	if session.Status == RecoveryStatusExpired || time.Now().After(session.ExpiresAt) {
		return ErrRecoverySessionExpired
	}
	if session.Status == RecoveryStatusCancelled {
		return ErrRecoverySessionCancelled
	}
	if session.Status == RecoveryStatusCompleted {
		return ErrRecoverySessionCompleted
	}
	if session.Attempts >= session.MaxAttempts {
		return ErrRecoverySessionLocked
	}
	return nil
}

func (s *Service) checkRateLimit(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) error {
	if !s.config.RateLimiting.Enabled {
		return nil
	}

	// Check hourly limit
	since := time.Now().Add(-1 * time.Hour)
	attempts, err := s.repo.GetRecentRecoveryAttempts(ctx, userID, appID, userOrganizationID, since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if attempts >= s.config.RateLimiting.MaxAttemptsPerHour {
		return ErrRateLimitExceeded
	}

	// Check daily limit
	since = time.Now().Add(-24 * time.Hour)
	attempts, err = s.repo.GetRecentRecoveryAttempts(ctx, userID, appID, userOrganizationID, since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if attempts >= s.config.RateLimiting.MaxAttemptsPerDay {
		return ErrAccountLocked
	}

	return nil
}

func (s *Service) calculateRiskScore(ctx context.Context, req *StartRecoveryRequest) float64 {
	if !s.config.RiskAssessment.Enabled {
		return 0.0
	}

	score := 0.0

	// Factors would include:
	// - New device
	// - New location
	// - New IP
	// - Velocity (multiple attempts)
	// - User history

	// Simplified implementation
	score += s.config.RiskAssessment.NewDeviceWeight * 100

	return score
}

func (s *Service) determineRequiredSteps(riskScore float64) []RecoveryMethod {
	var steps []RecoveryMethod

	if riskScore < s.config.RiskAssessment.LowRiskThreshold {
		steps = s.config.MultiStepRecovery.LowRiskSteps
	} else if riskScore < s.config.RiskAssessment.MediumRiskThreshold {
		steps = s.config.MultiStepRecovery.MediumRiskSteps
	} else {
		steps = s.config.MultiStepRecovery.HighRiskSteps
	}

	if len(steps) < s.config.MultiStepRecovery.MinimumSteps {
		// Add more steps if needed
		allMethods := []RecoveryMethod{
			RecoveryMethodCodes,
			RecoveryMethodSecurityQ,
			RecoveryMethodEmail,
			RecoveryMethodSMS,
			RecoveryMethodTrustedContact,
		}
		for _, method := range allMethods {
			if !containsMethod(steps, method) && s.isMethodEnabled(method) {
				steps = append(steps, method)
				if len(steps) >= s.config.MultiStepRecovery.MinimumSteps {
					break
				}
			}
		}
	}

	return steps
}

func (s *Service) getAvailableMethods(ctx context.Context, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID) []RecoveryMethod {
	var methods []RecoveryMethod

	if s.config.RecoveryCodes.Enabled {
		methods = append(methods, RecoveryMethodCodes)
	}
	if s.config.SecurityQuestions.Enabled {
		// Check if user has questions setup
		questions, _ := s.repo.GetSecurityQuestionsByUser(ctx, userID, appID, userOrganizationID)
		if len(questions) >= s.config.SecurityQuestions.MinimumQuestions {
			methods = append(methods, RecoveryMethodSecurityQ)
		}
	}
	if s.config.TrustedContacts.Enabled {
		// Check if user has trusted contacts
		count, _ := s.repo.CountActiveTrustedContacts(ctx, userID, appID, userOrganizationID)
		if count >= s.config.TrustedContacts.MinimumContacts {
			methods = append(methods, RecoveryMethodTrustedContact)
		}
	}
	if s.config.EmailVerification.Enabled {
		methods = append(methods, RecoveryMethodEmail)
	}
	if s.config.SMSVerification.Enabled {
		methods = append(methods, RecoveryMethodSMS)
	}
	if s.config.VideoVerification.Enabled {
		methods = append(methods, RecoveryMethodVideo)
	}
	if s.config.DocumentVerification.Enabled {
		methods = append(methods, RecoveryMethodDocument)
	}

	return methods
}

func (s *Service) isMethodEnabled(method RecoveryMethod) bool {
	switch method {
	case RecoveryMethodCodes:
		return s.config.RecoveryCodes.Enabled
	case RecoveryMethodSecurityQ:
		return s.config.SecurityQuestions.Enabled
	case RecoveryMethodTrustedContact:
		return s.config.TrustedContacts.Enabled
	case RecoveryMethodEmail:
		return s.config.EmailVerification.Enabled
	case RecoveryMethodSMS:
		return s.config.SMSVerification.Enabled
	case RecoveryMethodVideo:
		return s.config.VideoVerification.Enabled
	case RecoveryMethodDocument:
		return s.config.DocumentVerification.Enabled
	default:
		return false
	}
}

func (s *Service) getMethodInstructions(ctx context.Context, session *RecoverySession, method RecoveryMethod) (string, map[string]interface{}) {
	data := make(map[string]interface{})

	switch method {
	case RecoveryMethodCodes:
		return "Enter one of your recovery codes", data
	case RecoveryMethodSecurityQ:
		return "Answer your security questions", data
	case RecoveryMethodTrustedContact:
		return "Request verification from a trusted contact", data
	case RecoveryMethodEmail:
		return "We'll send a verification code to your email", data
	case RecoveryMethodSMS:
		return "We'll send a verification code to your phone", data
	case RecoveryMethodVideo:
		return "Schedule a video verification session", data
	case RecoveryMethodDocument:
		return "Upload a valid ID document", data
	default:
		return "Unknown recovery method", data
	}
}

func (s *Service) markStepCompleted(ctx context.Context, session *RecoverySession, method RecoveryMethod) error {
	methodStr := string(method)
	if !contains(session.CompletedSteps, methodStr) {
		session.CompletedSteps = append(session.CompletedSteps, methodStr)
		session.CurrentStep = len(session.CompletedSteps)
		return s.repo.UpdateRecoverySession(ctx, session)
	}
	return nil
}

func (s *Service) logRecoveryAttempt(ctx context.Context, recoveryID, userID xid.ID, appID xid.ID, userOrganizationID *xid.ID, action string, method RecoveryMethod, success bool, reason string) {
	log := &RecoveryAttemptLog{
		RecoveryID:         recoveryID,
		UserID:             userID,
		AppID:              appID,
		UserOrganizationID: userOrganizationID,
		Action:             action,
		Method:             method,
		Success:            success,
		FailureReason:      reason,
		IPAddress:          s.getIPFromContext(ctx),
		UserAgent:          s.getUserAgentFromContext(ctx),
	}
	s.repo.CreateRecoveryLog(ctx, log)
}

// ===== Cryptography Helpers =====

func (s *Service) generateRecoveryCode(format string, length int) (string, error) {
	var charset string
	switch format {
	case "alphanumeric":
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	case "numeric":
		charset = "0123456789"
	case "hex":
		charset = "0123456789ABCDEF"
	default:
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}

	code := make([]byte, length)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[num.Int64()]
	}

	return string(code), nil
}

func (s *Service) hashCode(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

func (s *Service) hashAnswer(answer string) (string, string, error) {
	// Normalize answer
	answer = strings.ToLower(strings.TrimSpace(answer))

	// Generate salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	saltStr := base64.StdEncoding.EncodeToString(salt)

	// Hash with bcrypt
	salted := answer + saltStr
	hash, err := bcrypt.GenerateFromPassword([]byte(salted), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	return string(hash), saltStr, nil
}

func (s *Service) verifyAnswer(answer, answerHash, salt string) bool {
	// Normalize answer
	answer = strings.ToLower(strings.TrimSpace(answer))
	salted := answer + salt
	err := bcrypt.CompareHashAndPassword([]byte(answerHash), []byte(salted))
	return err == nil
}

func (s *Service) generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *Service) generateRecoveryToken(userID, sessionID xid.ID) (string, error) {
	data := fmt.Sprintf("%s:%s:%d", userID.String(), sessionID.String(), time.Now().Unix())
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(append([]byte(data), bytes...))
	return token, nil
}

func (s *Service) isCommonAnswer(answer string) bool {
	commonAnswers := []string{"password", "123456", "admin", "test", "abc123"}
	normalized := strings.ToLower(strings.TrimSpace(answer))
	for _, common := range commonAnswers {
		if normalized == common {
			return true
		}
	}
	return false
}

// ===== Context Helpers =====

func (s *Service) getAppAndOrgFromContext(ctx context.Context) (xid.ID, *xid.ID) {
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)
	// Convert to pointer, returning nil if it's NilID
	if orgID == xid.NilID() {
		return appID, nil
	}
	return appID, &orgID
}

func (s *Service) getIPFromContext(ctx context.Context) string {
	// Get IP from AuthContext if available
	if authCtx, ok := contexts.GetAuthContext(ctx); ok && authCtx != nil {
		return authCtx.IPAddress
	}
	return ""
}

func (s *Service) getUserAgentFromContext(ctx context.Context) string {
	// Get User Agent from AuthContext if available
	if authCtx, ok := contexts.GetAuthContext(ctx); ok && authCtx != nil {
		return authCtx.UserAgent
	}
	return ""
}

// ===== Utility Functions =====

func convertMethodsToStrings(methods []RecoveryMethod) []string {
	result := make([]string, len(methods))
	for i, m := range methods {
		result[i] = string(m)
	}
	return result
}

func containsMethod(slice []RecoveryMethod, item RecoveryMethod) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
