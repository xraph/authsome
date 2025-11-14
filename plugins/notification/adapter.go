package notification

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
)

// Adapter provides a simplified interface for plugins to send notifications
type Adapter struct {
	templateSvc *TemplateService
}

// NewAdapter creates a new notification adapter
func NewAdapter(templateSvc *TemplateService) *Adapter {
	return &Adapter{templateSvc: templateSvc}
}

// SendMFACode sends an MFA verification code via email or SMS
func (a *Adapter) SendMFACode(ctx context.Context, appID xid.ID, recipient, code string, expiryMinutes int, notifType notification.NotificationType) error {
	userName := "User"    // Default, can be passed as parameter
	appName := "AuthSome" // Can be from config

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyMFACode,
		Type:        notifType,
		Recipient:   recipient,
		Variables: map[string]interface{}{
			"userName":      userName,
			"code":          code,
			"expiryMinutes": expiryMinutes,
			"appName":       appName,
		},
	})
	return err
}

// SendEmailOTP sends an email OTP code
func (a *Adapter) SendEmailOTP(ctx context.Context, appID xid.ID, email, code string, expiryMinutes int) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyEmailOTP,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]interface{}{
			"otp":      code,
			"userName": "User",
			"appName":  appName,
		},
	})
	return err
}

// SendPhoneOTP sends a phone OTP code via SMS
func (a *Adapter) SendPhoneOTP(ctx context.Context, appID xid.ID, phone, code string) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyPhoneOTP,
		Type:        notification.NotificationTypeSMS,
		Recipient:   phone,
		Variables: map[string]interface{}{
			"otp":     code,
			"appName": appName,
		},
	})
	return err
}

// SendMagicLink sends a magic link email
func (a *Adapter) SendMagicLink(ctx context.Context, appID xid.ID, email, userName, magicLink string, expiryMinutes int) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyMagicLink,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]interface{}{
			"userName": userName,
			"magicURL": magicLink,
			"appName":  appName,
		},
	})
	return err
}

// SendVerificationEmail sends an email verification link
func (a *Adapter) SendVerificationEmail(ctx context.Context, appID xid.ID, email, userName, verificationURL, verificationCode string, expiryMinutes int) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyVerifyEmail,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]interface{}{
			"userName":        userName,
			"verificationURL": verificationURL,
			"code":            verificationCode,
			"appName":         appName,
		},
	})
	return err
}

// SendPasswordReset sends a password reset email
func (a *Adapter) SendPasswordReset(ctx context.Context, appID xid.ID, email, userName, resetURL, resetCode string, expiryMinutes int) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyPasswordReset,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]interface{}{
			"userName": userName,
			"resetURL": resetURL,
			"code":     resetCode,
			"appName":  appName,
		},
	})
	return err
}

// SendWelcomeEmail sends a welcome email to new users
func (a *Adapter) SendWelcomeEmail(ctx context.Context, appID xid.ID, email, userName, loginURL string) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyWelcome,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]interface{}{
			"userName": userName,
			"appName":  appName,
			"loginURL": loginURL,
		},
	})
	return err
}

// SendSecurityAlert sends a security alert notification
func (a *Adapter) SendSecurityAlert(ctx context.Context, appID xid.ID, email, userName, eventType, eventTime, location, device string) error {
	appName := "AuthSome"

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeySecurityAlert,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]interface{}{
			"userName":     userName,
			"alertMessage": eventType,
			"timestamp":    eventTime,
			"location":     location,
			"ipAddress":    device, // Using device as IP for now
			"appName":      appName,
		},
	})
	return err
}

// SendCustom sends a notification using a custom template
func (a *Adapter) SendCustom(ctx context.Context, appID xid.ID, templateKey, recipient string, notifType notification.NotificationType, variables map[string]interface{}) error {
	if variables == nil {
		variables = make(map[string]interface{})
	}

	// Add default app_name if not provided
	if _, ok := variables["app_name"]; !ok {
		variables["app_name"] = "AuthSome"
	}

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: templateKey,
		Type:        notifType,
		Recipient:   recipient,
		Variables:   variables,
	})
	return err
}

// SendDirectEmail sends an email without using a template
func (a *Adapter) SendDirectEmail(ctx context.Context, appID xid.ID, recipient, subject, body string) error {
	_, err := a.templateSvc.SendDirect(ctx, appID, notification.NotificationTypeEmail, recipient, subject, body, nil)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// SendDirectSMS sends an SMS without using a template
func (a *Adapter) SendDirectSMS(ctx context.Context, appID xid.ID, recipient, body string) error {
	_, err := a.templateSvc.SendDirect(ctx, appID, notification.NotificationTypeSMS, recipient, "", body, nil)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	return nil
}
