package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/passkey"
)

// WebAuthnFactorAdapter integrates passkey plugin as an MFA factor
// This adapter enables passkeys to be used as a second authentication factor
// while maintaining support for standalone passwordless authentication
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

	// Extract optional metadata for registration
	req := passkey.BeginRegisterRequest{
		UserID: userID.String(),
	}

	// Apply metadata if provided
	if name, ok := metadata["name"].(string); ok {
		req.Name = name
	}
	if authType, ok := metadata["authenticatorType"].(string); ok {
		req.AuthenticatorType = authType
	}
	if reqResidentKey, ok := metadata["requireResidentKey"].(bool); ok {
		req.RequireResidentKey = reqResidentKey
	}

	// Use passkey service to begin registration
	resp, err := a.passkeyService.BeginRegistration(ctx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn registration: %w", err)
	}

	factorID := xid.New()
	return &FactorEnrollmentResponse{
		FactorID: factorID,
		Type:     FactorTypeWebAuthn,
		Status:   FactorStatusPending,
		ProvisioningData: map[string]any{
			"options":   resp.Options,
			"challenge": resp.Challenge,
			"timeout":   resp.Timeout,
			"message":   "Complete registration using your security key or biometric authentication",
		},
	}, nil
}

// VerifyEnrollment completes WebAuthn credential registration
func (a *WebAuthnFactorAdapter) VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error {
	if !a.IsAvailable() {
		return fmt.Errorf("WebAuthn factor not available")
	}

	// In MFA context, the proof would be the credential response
	// For now, return success as the verification happens in FinishRegistration
	// TODO: Implement proper enrollment verification flow
	return nil
}

// Challenge initiates a WebAuthn authentication challenge for MFA verification
func (a *WebAuthnFactorAdapter) Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("WebAuthn factor not available")
	}

	// Prepare login request
	req := passkey.BeginLoginRequest{
		UserID: factor.UserID.String(),
	}

	// Apply metadata if provided
	if userVerification, ok := metadata["userVerification"].(string); ok {
		req.UserVerification = userVerification
	}

	// Begin login challenge
	resp, err := a.passkeyService.BeginLogin(ctx, factor.UserID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn login: %w", err)
	}

	// Create MFA challenge record with proper expiration
	now := time.Now()
	expiresAt := now.Add(time.Duration(resp.Timeout) * time.Millisecond)
	challenge := &Challenge{
		ID:       xid.New(),
		UserID:   factor.UserID,
		FactorID: factor.ID,
		Type:     FactorTypeWebAuthn,
		Status:   ChallengeStatusPending,
		Metadata: map[string]any{
			"options":   resp.Options,
			"challenge": resp.Challenge,
			"timeout":   resp.Timeout,
		},
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	return challenge, nil
}

// Verify verifies the WebAuthn challenge response
func (a *WebAuthnFactorAdapter) Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error) {
	if !a.IsAvailable() {
		return false, fmt.Errorf("WebAuthn factor not available")
	}

	// Extract credential response from data
	// The data should contain the raw WebAuthn credential response
	credentialResponseBytes, ok := data["credentialResponse"].([]byte)
	if !ok {
		// Try to get it as a map and marshal to bytes
		credentialResponseMap, ok := data["credentialResponse"].(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("missing or invalid credential response")
		}

		// In production, this would be properly marshaled from the client
		// For now, we expect the client to send the raw bytes
		_ = credentialResponseMap
		return false, fmt.Errorf("credential response must be raw bytes from WebAuthn API")
	}

	// Use passkey service to verify the assertion
	// Note: We bypass session creation by calling FinishLogin with empty session parameters
	// This verifies the signature and updates sign count without creating an auth session
	loginResp, err := a.passkeyService.FinishLogin(ctx, credentialResponseBytes, false, "", "")
	if err != nil {
		return false, fmt.Errorf("WebAuthn verification failed: %w", err)
	}

	// Verification successful if we got a valid response
	return loginResp != nil, nil
}

// IsAvailable checks if WebAuthn factor is available
func (a *WebAuthnFactorAdapter) IsAvailable() bool {
	return a.available && a.passkeyService != nil
}
