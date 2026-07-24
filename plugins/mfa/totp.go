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

// totpPeriod is the TOTP time-step in seconds. totpSkew is the number of steps
// on each side of "now" that are accepted, matching the pquerna default that
// totp.Validate uses (±1 step ≈ a 90s window).
const (
	totpPeriod = 30
	totpSkew   = 1
)

// ValidateTOTPStep validates a code and, on success, returns the time-step
// (unix seconds / period) the code corresponds to. The step uniquely
// identifies the code within its validity window, letting callers reject a
// replay of the same code. Acceptance matches ValidateTOTP (±1 step).
func ValidateTOTPStep(code, secret string) (valid bool, step int64) {
	now := time.Now()
	for delta := int64(-totpSkew); delta <= totpSkew; delta++ {
		ts := now.Add(time.Duration(delta*totpPeriod) * time.Second)
		ok, err := totp.ValidateCustom(code, secret, ts, totp.ValidateOpts{
			Period:    totpPeriod,
			Skew:      0,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err == nil && ok {
			return true, ts.Unix() / totpPeriod
		}
	}
	return false, 0
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
