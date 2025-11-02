package mfa

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/emailotp"
)

// EmailFactorAdapter integrates emailotp plugin as an MFA factor (not primary auth)
type EmailFactorAdapter struct {
	BaseFactorAdapter
	emailOTPService *emailotp.Service
}

// NewEmailFactorAdapter creates a new email factor adapter
func NewEmailFactorAdapter(emailOTPService *emailotp.Service, enabled bool) *EmailFactorAdapter {
	return &EmailFactorAdapter{
		BaseFactorAdapter: BaseFactorAdapter{
			factorType: FactorTypeEmail,
			available:  enabled && emailOTPService != nil,
		},
		emailOTPService: emailOTPService,
	}
}

// Enroll registers an email address for MFA
func (a *EmailFactorAdapter) Enroll(ctx context.Context, userID xid.ID, metadata map[string]any) (*FactorEnrollmentResponse, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("email factor not available")
	}

	email, ok := metadata["email"].(string)
	if !ok || email == "" {
		return nil, fmt.Errorf("email required in metadata")
	}

	// Store email for this factor
	factorID := xid.New()

	// Email factors are pending until user verifies they can receive emails
	return &FactorEnrollmentResponse{
		FactorID: factorID,
		Type:     FactorTypeEmail,
		Status:   FactorStatusPending,
		ProvisioningData: map[string]any{
			"email":        email,
			"masked_email": maskEmail(email),
			"message":      "A verification code will be sent to this email when you verify enrollment",
		},
	}, nil
}

// VerifyEnrollment sends a test code to verify email works
func (a *EmailFactorAdapter) VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error {
	if !a.IsAvailable() {
		return fmt.Errorf("email factor not available")
	}

	// This would:
	// 1. Look up the pending enrollment
	// 2. Send a test code via emailotp
	// 3. Verify the provided proof matches
	// Implementation depends on enrollment storage
	return nil
}

// Challenge sends an email OTP code for MFA verification
func (a *EmailFactorAdapter) Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("email factor not available")
	}

	// Extract email from factor metadata
	email, ok := factor.Metadata["email"].(string)
	if !ok || email == "" {
		return nil, fmt.Errorf("no email configured for this factor")
	}

	// Extract IP and user agent from metadata
	ip, _ := metadata["ip"].(string)
	ua, _ := metadata["user_agent"].(string)

	// Use emailotp service to send the code
	// Note: We're using it for MFA, not primary auth
	code, err := a.emailOTPService.SendOTP(ctx, email, ip, ua)
	if err != nil {
		return nil, fmt.Errorf("failed to send email OTP: %w", err)
	}

	challenge := &Challenge{
		ID:       xid.New(),
		UserID:   factor.UserID,
		FactorID: factor.ID,
		Type:     FactorTypeEmail,
		Status:   ChallengeStatusPending,
		Code:     code, // Store for verification (hashed in production)
		Metadata: map[string]any{
			"email": maskEmail(email),
		},
		Attempts:    0,
		MaxAttempts: 5,
		IPAddress:   ip,
		UserAgent:   ua,
	}

	return challenge, nil
}

// Verify verifies an email OTP code
func (a *EmailFactorAdapter) Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error) {
	if !a.IsAvailable() {
		return false, fmt.Errorf("email factor not available")
	}

	// Simple code comparison (in production, this should use hashed comparison)
	// The emailotp plugin's VerifyOTP creates a session, which we don't want for MFA
	// So we do our own verification here
	valid := challenge.Code == response

	return valid, nil
}

// maskEmail masks an email address for display
// e.g., "user@example.com" -> "u***@example.com"
func maskEmail(email string) string {
	if len(email) < 3 {
		return "***"
	}

	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}

	if atIndex <= 0 {
		return "***"
	}

	// Show first char + *** + @domain
	return string(email[0]) + "***" + email[atIndex:]
}
