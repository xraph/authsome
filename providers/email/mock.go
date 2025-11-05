package email

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/core/notification"
)

// MockEmailProvider is a mock email provider for testing
type MockEmailProvider struct {
	SentEmails []MockEmail
}

// MockEmail represents a sent email for testing
type MockEmail struct {
	Recipient string
	Subject   string
	Body      string
}

// NewMockEmailProvider creates a new mock email provider
func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		SentEmails: make([]MockEmail, 0),
	}
}

// ID returns the provider ID
func (p *MockEmailProvider) ID() string {
	return "mock-email"
}

// Type returns the notification type this provider handles
func (p *MockEmailProvider) Type() notification.NotificationType {
	return notification.NotificationTypeEmail
}

// Send sends a mock email notification
func (p *MockEmailProvider) Send(ctx context.Context, notif *notification.Notification) error {
	p.SentEmails = append(p.SentEmails, MockEmail{
		Recipient: notif.Recipient,
		Subject:   notif.Subject,
		Body:      notif.Body,
	})

	// Set a mock provider ID
	notif.ProviderID = fmt.Sprintf("mock-%d", len(p.SentEmails))

	return nil
}

// GetStatus returns the status (always delivered for mock)
func (p *MockEmailProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	return notification.NotificationStatusDelivered, nil
}

// ValidateConfig validates the mock provider (always valid)
func (p *MockEmailProvider) ValidateConfig() error {
	return nil
}

// GetSentEmails returns all sent emails
func (p *MockEmailProvider) GetSentEmails() []MockEmail {
	return p.SentEmails
}

// ClearSentEmails clears all sent emails
func (p *MockEmailProvider) ClearSentEmails() {
	p.SentEmails = make([]MockEmail, 0)
}

// GetLastEmail returns the last sent email
func (p *MockEmailProvider) GetLastEmail() *MockEmail {
	if len(p.SentEmails) == 0 {
		return nil
	}
	return &p.SentEmails[len(p.SentEmails)-1]
}
