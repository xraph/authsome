package main

import (
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/providers/email"
	"github.com/xraph/authsome/providers/sms"
)

func main() {

	// Test email templates

	testEmailTemplates()

	// Test SMS templates

	testSMSTemplates()

	// Test email provider

	testEmailProvider()

	// Test SMS provider

	testSMSProvider()

}

func testEmailTemplates() {
	// Sample template data
	data := &email.TemplateData{
		UserName:         "John Doe",
		UserEmail:        "john@example.com",
		OrganizationName: "Example Organization",
		VerificationURL:  "https://app.example.com/verify?token=abc123",
		ResetURL:         "https://app.example.com/reset?token=def456",
		LoginURL:         "https://app.example.com/login?token=ghi789",
		IPAddress:        "192.168.1.100",
		UserAgent:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
		DeviceName:       "MacBook Pro",
		Location:         "San Francisco, CA",
		Timestamp:        "January 15, 2024 at 10:30 AM PST",
		ExpiryTime:       "24 hours",
		SupportEmail:     "support@example.com",
		CompanyName:      "Example Inc.",
		AppName:          "AuthSome Demo",
	}

	templates := email.ListTemplates()
	for _, templateName := range templates {

		rendered, err := email.RenderTemplate(templateName, data)
		if err != nil {

			continue
		}

	}
}

func testSMSTemplates() {
	// Sample template data
	data := &sms.TemplateData{
		UserName:         "John Doe",
		UserEmail:        "john@example.com",
		OrganizationName: "Example Org",
		VerificationCode: "123456",
		ResetCode:        "789012",
		LoginCode:        "345678",
		IPAddress:        "192.168.1.100",
		DeviceName:       "iPhone 15",
		Location:         "San Francisco, CA",
		Timestamp:        "Jan 15, 10:30 AM",
		ExpiryTime:       "10 minutes",
		AppName:          "AuthSome",
		SupportPhone:     "+1-555-0123",
	}

	templates := sms.ListTemplates()
	for _, templateName := range templates {

		rendered, err := sms.RenderTemplate(templateName, data)
		if err != nil {

			continue
		}

		// Validate template
		if err := sms.ValidateTemplate(templateName); err != nil {

		}
	}
}

func testEmailProvider() {
	// Create SMTP provider
	config := email.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "noreply@example.com",
		FromName: "AuthSome Demo",
	}

	provider := email.NewSMTPProvider(config)

	// Test notification request creation (without actually sending)
	testAppID := xid.New()
	notificationReq := &notification.SendRequest{
		AppID:     testAppID,
		Type:      notification.NotificationTypeEmail,
		Recipient: "user@example.com",
		Subject:   "Test Email",
		Body:      "Hello World\n\nThis is a test email.",
	}

	// Note: We're not actually sending the email in this test
	// In a real scenario, you would call: provider.Send(context.Background(), notification)
}

func testSMSProvider() {
	// Create Twilio provider
	config := sms.TwilioConfig{
		AccountSID: "test_account_sid",
		AuthToken:  "test_auth_token",
		FromNumber: "+1234567890",
	}

	provider := sms.NewTwilioProvider(config)

	// Test SMS notification request creation (without actually sending)
	testSMSAppID := xid.New()
	notificationReq := &notification.SendRequest{
		AppID:     testSMSAppID,
		Type:      notification.NotificationTypeSMS,
		Recipient: "+1987654321",
		Body:      "Hello! This is a test SMS from AuthSome.",
	}

	// Note: We're not actually sending the SMS in this test
	// In a real scenario, you would call: provider.Send(context.Background(), notification)
}
