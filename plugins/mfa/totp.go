package mfa

import (
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig configures TOTP generation.
type TOTPConfig struct {
	// Issuer is the name of the application (shown in authenticator apps).
	Issuer string

	// AccountName is the user's identifier (usually email).
	AccountName string
}

// GenerateTOTPKey creates a new TOTP secret key.
func GenerateTOTPKey(cfg TOTPConfig) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      cfg.Issuer,
		AccountName: cfg.AccountName,
	})
	if err != nil {
		return nil, fmt.Errorf("mfa: generate totp key: %w", err)
	}
	return key, nil
}

// ValidateTOTP validates a TOTP code against a secret.
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}

// GenerateTOTPCode generates a current TOTP code for a given secret.
// Primarily useful for testing.
func GenerateTOTPCode(secret string) (string, error) {
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", fmt.Errorf("mfa: generate totp code: %w", err)
	}
	return code, nil
}
