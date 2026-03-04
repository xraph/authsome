package bridge

import (
	"context"
	"errors"
)

// SMSSender sends SMS messages for MFA verification.
type SMSSender interface {
	// SendSMS sends an SMS message to the given phone number.
	SendSMS(ctx context.Context, msg *SMSMessage) error
}

// SMSMessage represents an SMS to be sent.
type SMSMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

// ErrSMSNotAvailable is returned when no SMS bridge is configured.
var ErrSMSNotAvailable = errors.New("authsome: SMS sender not available")
