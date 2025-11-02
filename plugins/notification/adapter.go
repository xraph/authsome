package notification

import (
	"context"
	"fmt"

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
func (a *Adapter) SendMFACode(ctx context.Context, orgID, recipient, code string, expiryMinutes int, notifType notification.NotificationType) error {
	userName := "User" // Default, can be passed as parameter
	appName := "AuthSome" // Can be from config
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.mfa_code",
		Type:           notifType,
		Recipient:      recipient,
		Variables: map[string]interface{}{
			"user_name":      userName,
			"code":           code,
			"expiry_minutes": expiryMinutes,
			"app_name":       appName,
		},
	})
	return err
}

// SendEmailOTP sends an email OTP code
func (a *Adapter) SendEmailOTP(ctx context.Context, orgID, email, code string, expiryMinutes int) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.email_otp",
		Type:           notification.NotificationTypeEmail,
		Recipient:      email,
		Variables: map[string]interface{}{
			"code":           code,
			"expiry_minutes": expiryMinutes,
			"app_name":       appName,
		},
	})
	return err
}

// SendPhoneOTP sends a phone OTP code via SMS
func (a *Adapter) SendPhoneOTP(ctx context.Context, orgID, phone, code string) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.phone_otp",
		Type:           notification.NotificationTypeSMS,
		Recipient:      phone,
		Variables: map[string]interface{}{
			"code":     code,
			"app_name": appName,
		},
	})
	return err
}

// SendMagicLink sends a magic link email
func (a *Adapter) SendMagicLink(ctx context.Context, orgID, email, userName, magicLink string, expiryMinutes int) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.magic_link",
		Type:           notification.NotificationTypeEmail,
		Recipient:      email,
		Variables: map[string]interface{}{
			"user_name":      userName,
			"magic_link":     magicLink,
			"expiry_minutes": expiryMinutes,
			"app_name":       appName,
		},
	})
	return err
}

// SendVerificationEmail sends an email verification link
func (a *Adapter) SendVerificationEmail(ctx context.Context, orgID, email, userName, verificationURL, verificationCode string, expiryMinutes int) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.verify_email",
		Type:           notification.NotificationTypeEmail,
		Recipient:      email,
		Variables: map[string]interface{}{
			"user_name":         userName,
			"verification_url":  verificationURL,
			"verification_code": verificationCode,
			"expiry_minutes":    expiryMinutes,
			"app_name":          appName,
		},
	})
	return err
}

// SendPasswordReset sends a password reset email
func (a *Adapter) SendPasswordReset(ctx context.Context, orgID, email, userName, resetURL, resetCode string, expiryMinutes int) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.password_reset",
		Type:           notification.NotificationTypeEmail,
		Recipient:      email,
		Variables: map[string]interface{}{
			"user_name":      userName,
			"reset_url":      resetURL,
			"reset_code":     resetCode,
			"expiry_minutes": expiryMinutes,
			"app_name":       appName,
		},
	})
	return err
}

// SendWelcomeEmail sends a welcome email to new users
func (a *Adapter) SendWelcomeEmail(ctx context.Context, orgID, email, userName, loginURL string) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.welcome",
		Type:           notification.NotificationTypeEmail,
		Recipient:      email,
		Variables: map[string]interface{}{
			"user_name":  userName,
			"user_email": email,
			"app_name":   appName,
			"login_url":  loginURL,
		},
	})
	return err
}

// SendSecurityAlert sends a security alert notification
func (a *Adapter) SendSecurityAlert(ctx context.Context, orgID, email, userName, eventType, eventTime, location, device string) error {
	appName := "AuthSome"
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    "auth.security_alert",
		Type:           notification.NotificationTypeEmail,
		Recipient:      email,
		Variables: map[string]interface{}{
			"user_name":  userName,
			"event_type": eventType,
			"event_time": eventTime,
			"location":   location,
			"device":     device,
			"app_name":   appName,
		},
	})
	return err
}

// SendCustom sends a notification using a custom template
func (a *Adapter) SendCustom(ctx context.Context, orgID, templateKey, recipient string, notifType notification.NotificationType, variables map[string]interface{}) error {
	if variables == nil {
		variables = make(map[string]interface{})
	}
	
	// Add default app_name if not provided
	if _, ok := variables["app_name"]; !ok {
		variables["app_name"] = "AuthSome"
	}
	
	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    templateKey,
		Type:           notifType,
		Recipient:      recipient,
		Variables:      variables,
	})
	return err
}

// SendDirectEmail sends an email without using a template
func (a *Adapter) SendDirectEmail(ctx context.Context, orgID, recipient, subject, body string) error {
	_, err := a.templateSvc.SendDirect(ctx, orgID, notification.NotificationTypeEmail, recipient, subject, body, nil)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

// SendDirectSMS sends an SMS without using a template
func (a *Adapter) SendDirectSMS(ctx context.Context, orgID, recipient, body string) error {
	_, err := a.templateSvc.SendDirect(ctx, orgID, notification.NotificationTypeSMS, recipient, "", body, nil)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	return nil
}

