package twofa

import (
	"context"
	"fmt"

	"github.com/pquerna/otp/totp"
	"github.com/rs/xid"
)

// TOTPSecret represents a generated TOTP secret bundle
type TOTPSecret struct {
	Secret string
	URI    string
}

// GenerateTOTPSecret creates a new TOTP secret and provisioning URI
func (s *Service) GenerateTOTPSecret(ctx context.Context, userID string) (*TOTPSecret, error) {
	// Generate GA-compatible secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "authsome",
		AccountName: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate totp secret: %w", err)
	}
	return &TOTPSecret{Secret: key.Secret(), URI: key.URL()}, nil
}

// VerifyTOTP checks a TOTP code against stored secret
func (s *Service) VerifyTOTP(userID, code string) (bool, error) {
	uid, err := xid.FromString(userID)
	if err != nil {
		return false, err
	}
	// Load secret
	sec, err := s.repo.GetSecret(context.Background(), uid)
	if err != nil || sec == nil {
		return false, err
	}
	if sec.Method != "totp" || !sec.Enabled {
		return false, nil
	}
	return totp.Validate(code, sec.Secret), nil
}
