package emailverification

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	"github.com/xraph/forge"
)

// Service implements email verification logic.
type Service struct {
	repo         *VerificationRepository
	users        user.ServiceInterface
	sessions     session.ServiceInterface
	notifAdapter *notificationPlugin.Adapter
	config       Config
	logger       forge.Logger
}

// NewService creates a new email verification service.
func NewService(
	repo *VerificationRepository,
	userSvc user.ServiceInterface,
	sessionSvc session.ServiceInterface,
	notifAdapter *notificationPlugin.Adapter,
	cfg Config,
	logger forge.Logger,
) *Service {
	// Apply defaults
	if cfg.TokenLength == 0 {
		cfg.TokenLength = 32
	}

	if cfg.ExpiryHours == 0 {
		cfg.ExpiryHours = 24
	}

	if cfg.MaxResendPerHour == 0 {
		cfg.MaxResendPerHour = 3
	}

	return &Service{
		repo:         repo,
		users:        userSvc,
		sessions:     sessionSvc,
		notifAdapter: notifAdapter,
		config:       cfg,
		logger:       logger,
	}
}

// SendVerification generates a verification token and sends verification email.
func (s *Service) SendVerification(ctx context.Context, appID, userID xid.ID, email string) (string, error) {
	// Check rate limits
	since := time.Now().Add(-1 * time.Hour)

	count, err := s.repo.CountRecentByUser(ctx, userID, since)
	if err != nil {
		return "", fmt.Errorf("failed to check rate limit: %w", err)
	}

	if count >= s.config.MaxResendPerHour {
		return "", ErrRateLimitExceeded
	}

	// Generate secure random token
	token, err := crypto.GenerateToken(s.config.TokenLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Calculate expiry
	expiresAt := time.Now().Add(time.Duration(s.config.ExpiryHours) * time.Hour)

	// Create verification record
	if err := s.repo.Create(ctx, appID, userID, token, expiresAt); err != nil {
		return "", fmt.Errorf("failed to create verification record: %w", err)
	}

	// Get user for email template
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	// Build verification URL
	verificationURL := s.buildVerificationURL(token)

	// Send verification email via notification adapter
	if s.notifAdapter != nil {
		err := s.notifAdapter.SendVerificationEmail(ctx, appID, email, u.Name, verificationURL, token, s.config.ExpiryHours*60)
		if err != nil {
			s.logger.Error("failed to send verification email",
				forge.F("error", err.Error()),
				forge.F("user_id", userID.String()))
			// Don't fail the operation - token is still created
		}
	}

	// Return token only in dev mode
	if s.config.DevExposeToken {
		return token, nil
	}

	return "", nil
}

// VerifyToken validates and consumes a verification token.
func (s *Service) VerifyToken(ctx context.Context, appID xid.ID, token string, autoLogin bool, ip, ua string) (*VerifyResponse, error) {
	// Find verification by token
	verification, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	// Check if token is expired
	if time.Now().After(verification.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Check if token has already been used
	if verification.Used {
		return nil, ErrTokenAlreadyUsed
	}

	// Get user
	u, err := s.users.FindByID(ctx, verification.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Check if user is already verified
	if u.EmailVerified {
		return nil, ErrAlreadyVerified
	}

	// Mark user as verified
	now := time.Now()
	u.EmailVerified = true
	u.EmailVerifiedAt = &now

	updateReq := &user.UpdateUserRequest{
		EmailVerified: &[]bool{true}[0],
	}

	u, err = s.users.Update(ctx, u, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Mark token as used
	if err := s.repo.MarkAsUsed(ctx, verification.ID); err != nil {
		s.logger.Error("failed to mark token as used",
			forge.F("error", err.Error()),
			forge.F("verification_id", verification.ID.String()))
		// Don't fail - user is already verified
	}

	response := &VerifyResponse{
		Success: true,
		User:    u,
	}

	// Optionally create session (auto-login)
	if autoLogin && s.config.AutoLoginAfterVerify {
		// Extract OrganizationID from context (optional)
		var organizationID *xid.ID
		if orgID, ok := contexts.GetOrganizationID(ctx); ok && !orgID.IsNil() {
			organizationID = &orgID
		}

		// Extract EnvironmentID from context (optional)
		var environmentID *xid.ID
		if envID, ok := contexts.GetEnvironmentID(ctx); ok && !envID.IsNil() {
			environmentID = &envID
		}

		sess, err := s.sessions.Create(ctx, &session.CreateSessionRequest{
			AppID:          appID,
			EnvironmentID:  environmentID,
			OrganizationID: organizationID,
			UserID:         u.ID,
			Remember:       true,
			IPAddress:      ip,
			UserAgent:      ua,
		})
		if err != nil {
			s.logger.Error("failed to create session after verification",
				forge.F("error", err.Error()),
				forge.F("user_id", u.ID.String()))
			// Don't fail - user is verified
		} else {
			response.Session = sess
			response.Token = sess.Token
		}
	}

	return response, nil
}

// ResendVerification sends a new verification email.
func (s *Service) ResendVerification(ctx context.Context, appID xid.ID, email string) error {
	// Find user by email
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil || u == nil {
		return ErrUserNotFound
	}

	// Check if already verified
	if u.EmailVerified {
		return ErrAlreadyVerified
	}

	// Check rate limits
	since := time.Now().Add(-1 * time.Hour)

	count, err := s.repo.CountRecentByUser(ctx, u.ID, since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	if count >= s.config.MaxResendPerHour {
		return ErrRateLimitExceeded
	}

	// Invalidate old unused tokens
	if err := s.repo.InvalidateOldTokens(ctx, u.ID); err != nil {
		s.logger.Warn("failed to invalidate old tokens",
			forge.F("error", err.Error()),
			forge.F("user_id", u.ID.String()))
	}

	// Send new verification
	_, err = s.SendVerification(ctx, appID, u.ID, email)

	return err
}

// GetStatus returns the email verification status for a user.
func (s *Service) GetStatus(ctx context.Context, userID xid.ID) (*StatusResponse, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &StatusResponse{
		EmailVerified:   u.EmailVerified,
		EmailVerifiedAt: u.EmailVerifiedAt,
	}, nil
}

// CleanupExpiredTokens removes expired verification tokens (for scheduled cleanup).
func (s *Service) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	before := time.Now()

	return s.repo.DeleteExpired(ctx, before)
}

// buildVerificationURL constructs the verification URL.
func (s *Service) buildVerificationURL(token string) string {
	if s.config.VerificationURL != "" {
		// Use custom URL template
		return fmt.Sprintf("%s?token=%s", s.config.VerificationURL, token)
	}
	// Default URL format
	return "/verify?token=" + token
}
