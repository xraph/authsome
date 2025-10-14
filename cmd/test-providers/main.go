package main

import (
	"fmt"

	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/providers/email"
	"github.com/xraph/authsome/providers/sms"
)

func main() {
	fmt.Println("Testing AuthSome Providers and Templates")
	fmt.Println("========================================")

	// Test email templates
	fmt.Println("\n1. Testing Email Templates:")
	testEmailTemplates()

	// Test SMS templates
	fmt.Println("\n2. Testing SMS Templates:")
	testSMSTemplates()

	// Test email provider
	fmt.Println("\n3. Testing Email Provider:")
	testEmailProvider()

	// Test SMS provider
	fmt.Println("\n4. Testing SMS Provider:")
	testSMSProvider()

	fmt.Println("\nAll tests completed!")
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
		fmt.Printf("  Testing template: %s\n", templateName)
		
		rendered, err := email.RenderTemplate(templateName, data)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}
		
		fmt.Printf("    ✅ Subject: %s\n", rendered.Subject)
		fmt.Printf("    ✅ HTML body: %d characters\n", len(rendered.HTMLBody))
		fmt.Printf("    ✅ Text body: %d characters\n", len(rendered.TextBody))
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
		fmt.Printf("  Testing template: %s\n", templateName)
		
		rendered, err := sms.RenderTemplate(templateName, data)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}
		
		fmt.Printf("    ✅ Body: %s\n", rendered.Body)
		fmt.Printf("    ✅ Length: %d characters\n", len(rendered.Body))
		
		// Validate template
		if err := sms.ValidateTemplate(templateName); err != nil {
			fmt.Printf("    ⚠️  Validation warning: %v\n", err)
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
	fmt.Printf("  ✅ SMTP Provider created: %s:%d\n", config.Host, config.Port)
	fmt.Printf("  ✅ Provider ID: %s\n", provider.ID())
	fmt.Printf("  ✅ Provider Type: %s\n", provider.Type())

	// Test notification request creation (without actually sending)
	notificationReq := &notification.SendRequest{
		OrganizationID: "test-org",
		Type:           notification.NotificationTypeEmail,
		Recipient:      "user@example.com",
		Subject:        "Test Email",
		Body:           "Hello World\n\nThis is a test email.",
	}

	fmt.Printf("  ✅ Notification request created for: %s\n", notificationReq.Recipient)
	fmt.Printf("  ✅ Subject: %s\n", notificationReq.Subject)
	
	// Note: We're not actually sending the email in this test
	// In a real scenario, you would call: provider.Send(context.Background(), notification)
	fmt.Printf("  ℹ️  Email sending skipped (test mode)\n")
}

func testSMSProvider() {
	// Create Twilio provider
	config := sms.TwilioConfig{
		AccountSID: "test_account_sid",
		AuthToken:  "test_auth_token",
		FromNumber: "+1234567890",
	}

	provider := sms.NewTwilioProvider(config)
	fmt.Printf("  ✅ Twilio Provider created with from number: %s\n", config.FromNumber)
	fmt.Printf("  ✅ Provider ID: %s\n", provider.ID())
	fmt.Printf("  ✅ Provider Type: %s\n", provider.Type())

	// Test SMS notification request creation (without actually sending)
	notificationReq := &notification.SendRequest{
		OrganizationID: "test-org",
		Type:           notification.NotificationTypeSMS,
		Recipient:      "+1987654321",
		Body:           "Hello! This is a test SMS from AuthSome.",
	}

	fmt.Printf("  ✅ SMS notification request created for: %s\n", notificationReq.Recipient)
	fmt.Printf("  ✅ Body: %s\n", notificationReq.Body)
	
	// Note: We're not actually sending the SMS in this test
	// In a real scenario, you would call: provider.Send(context.Background(), notification)
	fmt.Printf("  ℹ️  SMS sending skipped (test mode)\n")
}