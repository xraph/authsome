package mfa

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/xraph/authsome/bridge"
)

const (
	// smsCodeLength is the length of SMS verification codes.
	smsCodeLength = 6

	// smsCodeTTL is how long an SMS code remains valid.
	smsCodeTTL = 5 * time.Minute
)

// SMSChallenge represents a pending SMS verification challenge.
type SMSChallenge struct {
	Code      string
	ExpiresAt time.Time
}

// GenerateSMSCode generates a random numeric code of the given length.
func GenerateSMSCode(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("mfa: generate SMS code: %w", err)
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}

// SendSMSChallenge generates a code and sends it via the SMS bridge.
func SendSMSChallenge(ctx context.Context, sender bridge.SMSSender, phone string) (*SMSChallenge, error) {
	code, err := GenerateSMSCode(smsCodeLength)
	if err != nil {
		return nil, err
	}

	msg := &bridge.SMSMessage{
		To:   phone,
		Body: fmt.Sprintf("Your verification code is: %s. It expires in %d minutes.", code, int(smsCodeTTL.Minutes())),
	}

	if err := sender.SendSMS(ctx, msg); err != nil {
		return nil, fmt.Errorf("mfa: send SMS challenge: %w", err)
	}

	return &SMSChallenge{
		Code:      code,
		ExpiresAt: time.Now().Add(smsCodeTTL),
	}, nil
}

// ValidateSMSCode checks whether the provided code matches the challenge and is not expired.
func ValidateSMSCode(code string, challenge *SMSChallenge) bool {
	if challenge == nil {
		return false
	}
	if time.Now().After(challenge.ExpiresAt) {
		return false
	}
	return code == challenge.Code
}
