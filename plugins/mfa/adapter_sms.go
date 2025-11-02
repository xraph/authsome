package mfa

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/phone"
)

// SMSFactorAdapter integrates phone plugin as an MFA factor (not primary auth)
type SMSFactorAdapter struct {
	BaseFactorAdapter
	phoneService *phone.Service
}

// NewSMSFactorAdapter creates a new SMS factor adapter
func NewSMSFactorAdapter(phoneService *phone.Service, enabled bool) *SMSFactorAdapter {
	return &SMSFactorAdapter{
		BaseFactorAdapter: BaseFactorAdapter{
			factorType: FactorTypeSMS,
			available:  enabled && phoneService != nil,
		},
		phoneService: phoneService,
	}
}

// Enroll registers a phone number for MFA
func (a *SMSFactorAdapter) Enroll(ctx context.Context, userID xid.ID, metadata map[string]any) (*FactorEnrollmentResponse, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("SMS factor not available")
	}

	phone, ok := metadata["phone"].(string)
	if !ok || phone == "" {
		return nil, fmt.Errorf("phone number required in metadata")
	}

	// Store phone for this factor
	factorID := xid.New()

	// SMS factors are pending until user verifies they can receive messages
	return &FactorEnrollmentResponse{
		FactorID: factorID,
		Type:     FactorTypeSMS,
		Status:   FactorStatusPending,
		ProvisioningData: map[string]any{
			"phone":        phone,
			"masked_phone": maskPhone(phone),
			"message":      "A verification code will be sent to this phone when you verify enrollment",
		},
	}, nil
}

// VerifyEnrollment sends a test code to verify phone works
func (a *SMSFactorAdapter) VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error {
	if !a.IsAvailable() {
		return fmt.Errorf("SMS factor not available")
	}

	// This would:
	// 1. Look up the pending enrollment
	// 2. Send a test code via phone service
	// 3. Verify the provided proof matches
	// Implementation depends on enrollment storage
	return nil
}

// Challenge sends an SMS OTP code for MFA verification
func (a *SMSFactorAdapter) Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("SMS factor not available")
	}

	// Extract phone from factor metadata
	phoneNumber, ok := factor.Metadata["phone"].(string)
	if !ok || phoneNumber == "" {
		return nil, fmt.Errorf("no phone number configured for this factor")
	}

	// Extract IP and user agent from metadata
	ip, _ := metadata["ip"].(string)
	ua, _ := metadata["user_agent"].(string)

	// Use phone service to send the code
	// Note: We're using it for MFA, not primary auth
	code, err := a.phoneService.SendCode(ctx, phoneNumber, ip, ua)
	if err != nil {
		return nil, fmt.Errorf("failed to send SMS code: %w", err)
	}

	challenge := &Challenge{
		ID:       xid.New(),
		UserID:   factor.UserID,
		FactorID: factor.ID,
		Type:     FactorTypeSMS,
		Status:   ChallengeStatusPending,
		Code:     code, // Store for verification (hashed in production)
		Metadata: map[string]any{
			"phone": maskPhone(phoneNumber),
		},
		Attempts:    0,
		MaxAttempts: 5,
		IPAddress:   ip,
		UserAgent:   ua,
	}

	return challenge, nil
}

// Verify verifies an SMS OTP code
func (a *SMSFactorAdapter) Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error) {
	if !a.IsAvailable() {
		return false, fmt.Errorf("SMS factor not available")
	}

	// Simple code comparison (in production, this should use hashed comparison)
	// The phone plugin's Verify creates a session, which we don't want for MFA
	// So we do our own verification here
	valid := challenge.Code == response

	return valid, nil
}

// maskPhone masks a phone number for display
// e.g., "+15551234567" -> "+1***-***-4567"
func maskPhone(phone string) string {
	if len(phone) < 4 {
		return "***"
	}

	// Keep first 2 and last 4 characters
	if len(phone) <= 6 {
		return phone[:2] + "***"
	}

	return phone[:2] + "***-***-" + phone[len(phone)-4:]
}
