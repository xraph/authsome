package mfa

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/passkey"
)

// WebAuthnFactorAdapter integrates passkey plugin as an MFA factor
// NOTE: This adapter is currently experimental as the passkey plugin is in beta
type WebAuthnFactorAdapter struct {
	BaseFactorAdapter
	passkeyService *passkey.Service
}

// NewWebAuthnFactorAdapter creates a new WebAuthn factor adapter
func NewWebAuthnFactorAdapter(passkeyService *passkey.Service, enabled bool) *WebAuthnFactorAdapter {
	return &WebAuthnFactorAdapter{
		BaseFactorAdapter: BaseFactorAdapter{
			factorType: FactorTypeWebAuthn,
			available:  enabled && passkeyService != nil,
		},
		passkeyService: passkeyService,
	}
}

// Enroll initiates WebAuthn credential registration for MFA
func (a *WebAuthnFactorAdapter) Enroll(ctx context.Context, userID xid.ID, metadata map[string]any) (*FactorEnrollmentResponse, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("WebAuthn factor not available")
	}

	// Use passkey service to begin registration
	options, err := a.passkeyService.BeginRegistration(ctx, userID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn registration: %w", err)
	}

	factorID := xid.New()
	return &FactorEnrollmentResponse{
		FactorID: factorID,
		Type:     FactorTypeWebAuthn,
		Status:   FactorStatusPending,
		ProvisioningData: map[string]any{
			"options": options,
			"message": "Complete registration using your security key or biometric authentication",
		},
	}, nil
}

// VerifyEnrollment completes WebAuthn credential registration
func (a *WebAuthnFactorAdapter) VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error {
	if !a.IsAvailable() {
		return fmt.Errorf("WebAuthn factor not available")
	}

	// This would:
	// 1. Look up the pending enrollment and user
	// 2. Use passkey service to finish registration
	// 3. Store the credential
	// Implementation depends on enrollment storage
	return nil
}

// Challenge initiates a WebAuthn authentication challenge
func (a *WebAuthnFactorAdapter) Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("WebAuthn factor not available")
	}

	// Use passkey service to begin login (authentication)
	options, err := a.passkeyService.BeginLogin(ctx, factor.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn authentication: %w", err)
	}

	challenge := &Challenge{
		ID:       xid.New(),
		UserID:   factor.UserID,
		FactorID: factor.ID,
		Type:     FactorTypeWebAuthn,
		Status:   ChallengeStatusPending,
		Metadata: map[string]any{
			"options": options,
		},
		Attempts:    0,
		MaxAttempts: 3,
	}

	return challenge, nil
}

// Verify verifies a WebAuthn authentication response
func (a *WebAuthnFactorAdapter) Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error) {
	if !a.IsAvailable() {
		return false, fmt.Errorf("WebAuthn factor not available")
	}

	// Extract credential response from data
	credentialResponse, ok := data["credential"]
	if !ok {
		return false, fmt.Errorf("no credential in response data")
	}

	// Use passkey service to finish login (verify assertion)
	// Note: The passkey service's FinishLogin creates a session, which we don't want for MFA
	// We only want to verify the WebAuthn assertion
	// This would need to be refactored in the passkey service to separate verification from session creation

	// For now, this is a placeholder
	// In production, this would call a verification-only method
	_ = credentialResponse

	return true, nil // Placeholder
}
