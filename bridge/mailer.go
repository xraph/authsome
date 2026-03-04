package bridge

import (
	"context"
	"errors"
)

// Mailer is a local email sending interface. Implementations deliver
// transactional emails (welcome, verification, password reset, invitation).
type Mailer interface {
	// SendEmail delivers a transactional email.
	SendEmail(ctx context.Context, msg *EmailMessage) error
}

// EmailMessage represents a transactional email.
type EmailMessage struct {
	To      []string `json:"to"`
	From    string   `json:"from,omitempty"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// ErrMailerNotAvailable is returned when no mailer bridge is configured.
var ErrMailerNotAvailable = errors.New("authsome: mailer not available")

// MailerFunc is an adapter to use a plain function as a Mailer.
type MailerFunc func(ctx context.Context, msg *EmailMessage) error

// SendEmail implements Mailer.
func (f MailerFunc) SendEmail(ctx context.Context, msg *EmailMessage) error {
	return f(ctx, msg)
}
