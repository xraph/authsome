package mfa

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/twofa"
)

// TOTPFactorAdapter integrates twofa plugin's TOTP functionality as an MFA factor
type TOTPFactorAdapter struct {
	BaseFactorAdapter
	twofaService *twofa.Service
}

// NewTOTPFactorAdapter creates a new TOTP factor adapter
func NewTOTPFactorAdapter(twofaService *twofa.Service, enabled bool) *TOTPFactorAdapter {
	return &TOTPFactorAdapter{
		BaseFactorAdapter: BaseFactorAdapter{
			factorType: FactorTypeTOTP,
			available:  enabled && twofaService != nil,
		},
		twofaService: twofaService,
	}
}

// Enroll initiates TOTP enrollment
func (a *TOTPFactorAdapter) Enroll(ctx context.Context, userID xid.ID, metadata map[string]any) (*FactorEnrollmentResponse, error) {
	if !a.IsAvailable() {
		return nil, errs.BadRequest("TOTP MFA factor not available")
	}

	// Use twofa service to generate TOTP secret
	secret, err := a.twofaService.GenerateTOTPSecret(ctx, userID.String())
	if err != nil {
		return nil, errs.Wrap(err, "GENERATE_TOTP_SECRET_FAILED", "Failed to generate TOTP secret", 500)
	}

	// Create enrollment response with provisioning data
	factorID := xid.New()
	return &FactorEnrollmentResponse{
		FactorID: factorID,
		Type:     FactorTypeTOTP,
		Status:   FactorStatusPending,
		ProvisioningData: map[string]any{
			"secret":  secret.Secret,
			"qr_uri":  secret.URI,
			"issuer":  "AuthSome",
			"account": userID.String(),
		},
	}, nil
}

// VerifyEnrollment verifies TOTP enrollment by checking first code
func (a *TOTPFactorAdapter) VerifyEnrollment(ctx context.Context, enrollmentID xid.ID, proof string) error {
	if !a.IsAvailable() {
		return errs.BadRequest("TOTP MFA factor not available")
	}

	// This would verify the first TOTP code to ensure the user set it up correctly
	// Implementation depends on how we store pending enrollments
	// For now, this is a placeholder
	return nil
}

// Challenge initiates a TOTP verification challenge
// For TOTP, there's no async challenge - user provides code directly
func (a *TOTPFactorAdapter) Challenge(ctx context.Context, factor *Factor, metadata map[string]any) (*Challenge, error) {
	if !a.IsAvailable() {
		return nil, errs.BadRequest("TOTP MFA factor not available")
	}

	// TOTP doesn't require sending anything - user provides code from their authenticator
	challenge := &Challenge{
		ID:          xid.New(),
		UserID:      factor.UserID,
		FactorID:    factor.ID,
		Type:        FactorTypeTOTP,
		Status:      ChallengeStatusPending,
		Metadata:    metadata,
		Attempts:    0,
		MaxAttempts: 3,
	}

	return challenge, nil
}

// Verify verifies a TOTP code
func (a *TOTPFactorAdapter) Verify(ctx context.Context, challenge *Challenge, response string, data map[string]any) (bool, error) {
	if !a.IsAvailable() {
		return false, errs.BadRequest("TOTP MFA factor not available")
	}

	// Use twofa service to verify the TOTP code
	valid, err := a.twofaService.VerifyTOTP(challenge.UserID.String(), response)
	if err != nil {
		return false, errs.Wrap(err, "VERIFY_TOTP_FAILED", "Failed to verify TOTP", 400)
	}

	return valid, nil
}
