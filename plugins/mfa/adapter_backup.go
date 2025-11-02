package mfa

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/twofa"
)

// BackupCodeFactorAdapter integrates twofa plugin's backup codes as an MFA factor
type BackupCodeFactorAdapter struct {
	BaseFactorAdapter
	twofaService *twofa.Service
}

// NewBackupCodeFactorAdapter creates a new backup code factor adapter
func NewBackupCodeFactorAdapter(twofaService *twofa.Service, enabled bool) *BackupCodeFactorAdapter {
	return &BackupCodeFactorAdapter{
		BaseFactorAdapter: BaseFactorAdapter{
			factorType: FactorTypeBackup,
			available:  enabled && twofaService != nil,
		},
		twofaService: twofaService,
	}
}

// Enroll generates backup codes for a user
func (a *BackupCodeFactorAdapter) Enroll(ctx context.Context, userID xid.ID, metadata map[string]any) (*FactorEnrollmentResponse, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("backup code factor not available")
	}

	// Generate backup codes using twofa service
	count := 10
	if c, ok := metadata["count"].(int); ok && c > 0 {
		count = c
	}

	codes, err := a.twofaService.GenerateBackupCodes(ctx, userID.String(), count)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	factorID := xid.New()
	return &FactorEnrollmentResponse{
		FactorID: factorID,
		Type:     FactorTypeBackup,
		Status:   FactorStatusActive, // Backup codes are immediately active
		ProvisioningData: map[string]any{
			"codes":   codes,
			"count":   len(codes),
			"warning": "Store these codes securely. Each can only be used once.",
		},
	}, nil
}

// VerifyEnrollment is not needed for backup codes (immediately active)
func (a *BackupCodeFactorAdapter) VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error {
	return nil // No verification needed for backup codes
}

// Challenge creates a backup code verification challenge
func (a *BackupCodeFactorAdapter) Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error) {
	if !a.IsAvailable() {
		return nil, fmt.Errorf("backup code factor not available")
	}

	challenge := &Challenge{
		ID:          xid.New(),
		UserID:      factor.UserID,
		FactorID:    factor.ID,
		Type:        FactorTypeBackup,
		Status:      ChallengeStatusPending,
		Metadata:    metadata,
		Attempts:    0,
		MaxAttempts: 3,
	}

	return challenge, nil
}

// Verify verifies a backup code
func (a *BackupCodeFactorAdapter) Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error) {
	if !a.IsAvailable() {
		return false, fmt.Errorf("backup code factor not available")
	}

	// Use twofa service to verify backup code
	valid, err := a.twofaService.VerifyBackupCode(ctx, challenge.UserID.String(), response)
	if err != nil {
		return false, fmt.Errorf("failed to verify backup code: %w", err)
	}

	return valid, nil
}
