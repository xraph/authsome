package notification

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
)

// Adapter provides a simplified interface for plugins to send notifications.
type Adapter struct {
	templateSvc *TemplateService
	appService  any    // app.Service interface to avoid import cycle
	appName     string // Configured app name override
}

// NewAdapter creates a new notification adapter.
func NewAdapter(templateSvc *TemplateService) *Adapter {
	return &Adapter{
		templateSvc: templateSvc,
	}
}

// WithAppService sets the app service for dynamic app name lookup.
func (a *Adapter) WithAppService(appSvc any) *Adapter {
	a.appService = appSvc

	return a
}

// WithAppName sets a static app name override.
func (a *Adapter) WithAppName(name string) *Adapter {
	a.appName = name

	return a
}

// getAppName retrieves the app name to use in notifications
// Priority: 1. Static override, 2. App from database, 3. Fallback to "AuthSome".
func (a *Adapter) getAppName(ctx context.Context, appID xid.ID) string {
	// If static override is set, use it
	if a.appName != "" {
		return a.appName
	}

	// Try to get app name from database
	if a.appService != nil {
		if appSvc, ok := a.appService.(interface {
			FindByID(context.Context, xid.ID) (any, error)
		}); ok {
			if appData, err := appSvc.FindByID(ctx, appID); err == nil && appData != nil {
				// Use type assertion to get name field
				if app, ok := appData.(interface{ GetName() string }); ok {
					if name := app.GetName(); name != "" {
						return name
					}
				}
			}
		}
	}

	// Fallback to default
	return "AuthSome"
}

// SendMFACode sends an MFA verification code via email or SMS.
func (a *Adapter) SendMFACode(ctx context.Context, appID xid.ID, recipient, code string, expiryMinutes int, notifType notification.NotificationType) error {
	userName := "User"                  // Default, can be passed as parameter
	appName := a.getAppName(ctx, appID) // Can be from config

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyMFACode,
		Type:        notifType,
		Recipient:   recipient,
		Variables: map[string]any{
			"userName":      userName,
			"code":          code,
			"expiryMinutes": expiryMinutes,
			"appName":       appName,
		},
	})

	return err
}

// SendEmailOTP sends an email OTP code.
func (a *Adapter) SendEmailOTP(ctx context.Context, appID xid.ID, email, code string, expiryMinutes int) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyEmailOTP,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]any{
			"otp":      code,
			"userName": "User",
			"appName":  appName,
		},
	})

	return err
}

// SendPhoneOTP sends a phone OTP code via SMS.
func (a *Adapter) SendPhoneOTP(ctx context.Context, appID xid.ID, phone, code string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyPhoneOTP,
		Type:        notification.NotificationTypeSMS,
		Recipient:   phone,
		Variables: map[string]any{
			"otp":     code,
			"appName": appName,
		},
	})

	return err
}

// SendMagicLink sends a magic link email.
func (a *Adapter) SendMagicLink(ctx context.Context, appID xid.ID, email, userName, magicLink string, expiryMinutes int) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyMagicLink,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]any{
			"userName": userName,
			"magicURL": magicLink,
			"appName":  appName,
		},
	})

	return err
}

// SendVerificationEmail sends an email verification link.
func (a *Adapter) SendVerificationEmail(ctx context.Context, appID xid.ID, email, userName, verificationURL, verificationCode string, expiryMinutes int) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyVerifyEmail,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]any{
			"userName":        userName,
			"verificationURL": verificationURL,
			"code":            verificationCode,
			"appName":         appName,
		},
	})

	return err
}

// SendPasswordReset sends a password reset email.
func (a *Adapter) SendPasswordReset(ctx context.Context, appID xid.ID, email, userName, resetURL, resetCode string, expiryMinutes int) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyPasswordReset,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]any{
			"userName": userName,
			"resetURL": resetURL,
			"code":     resetCode,
			"appName":  appName,
		},
	})

	return err
}

// SendWelcomeEmail sends a welcome email to new users.
func (a *Adapter) SendWelcomeEmail(ctx context.Context, appID xid.ID, email, userName, loginURL string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyWelcome,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]any{
			"userName": userName,
			"appName":  appName,
			"loginURL": loginURL,
		},
	})

	return err
}

// SendSecurityAlert sends a security alert notification.
func (a *Adapter) SendSecurityAlert(ctx context.Context, appID xid.ID, email, userName, eventType, eventTime, location, device string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeySecurityAlert,
		Type:        notification.NotificationTypeEmail,
		Recipient:   email,
		Variables: map[string]any{
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

// SendCustom sends a notification using a custom template.
func (a *Adapter) SendCustom(ctx context.Context, appID xid.ID, templateKey, recipient string, notifType notification.NotificationType, variables map[string]any) error {
	if variables == nil {
		variables = make(map[string]any)
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

// SendDirectEmail sends an email without using a template.
func (a *Adapter) SendDirectEmail(ctx context.Context, appID xid.ID, recipient, subject, body string) error {
	_, err := a.templateSvc.SendDirect(ctx, appID, notification.NotificationTypeEmail, recipient, subject, body, nil)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendDirectSMS sends an SMS without using a template.
func (a *Adapter) SendDirectSMS(ctx context.Context, appID xid.ID, recipient, body string) error {
	_, err := a.templateSvc.SendDirect(ctx, appID, notification.NotificationTypeSMS, recipient, "", body, nil)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	return nil
}

// SendOrgInvite sends an organization invitation email.
func (a *Adapter) SendOrgInvite(ctx context.Context, appID xid.ID, recipientEmail, userName, inviterName, orgName, role, inviteURL string, expiresIn string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgInvite,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":    userName,
			"inviterName": inviterName,
			"orgName":     orgName,
			"role":        role,
			"inviteURL":   inviteURL,
			"appName":     appName,
			"expiresIn":   expiresIn,
		},
	})

	return err
}

// SendOrgMemberAdded sends notification when a member is added to organization.
func (a *Adapter) SendOrgMemberAdded(ctx context.Context, appID xid.ID, recipientEmail, userName, memberName, orgName, role string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgMemberAdded,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":   userName,
			"memberName": memberName,
			"orgName":    orgName,
			"role":       role,
			"appName":    appName,
		},
	})

	return err
}

// SendOrgMemberRemoved sends notification when a member is removed from organization.
func (a *Adapter) SendOrgMemberRemoved(ctx context.Context, appID xid.ID, recipientEmail, userName, memberName, orgName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgMemberRemoved,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":   userName,
			"memberName": memberName,
			"orgName":    orgName,
			"timestamp":  timestamp,
			"appName":    appName,
		},
	})

	return err
}

// SendOrgRoleChanged sends notification when a member's role is changed.
func (a *Adapter) SendOrgRoleChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, orgName, oldRole, newRole string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgRoleChanged,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName": userName,
			"orgName":  orgName,
			"oldRole":  oldRole,
			"newRole":  newRole,
			"appName":  appName,
		},
	})

	return err
}

// SendOrgTransfer sends notification when organization ownership is transferred.
func (a *Adapter) SendOrgTransfer(ctx context.Context, appID xid.ID, recipientEmail, userName, orgName, transferredTo, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgTransfer,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":      userName,
			"orgName":       orgName,
			"transferredTo": transferredTo,
			"timestamp":     timestamp,
			"appName":       appName,
		},
	})

	return err
}

// SendOrgDeleted sends notification when an organization is deleted.
func (a *Adapter) SendOrgDeleted(ctx context.Context, appID xid.ID, recipientEmail, userName, orgName string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgDeleted,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName": userName,
			"orgName":  orgName,
			"appName":  appName,
		},
	})

	return err
}

// SendOrgMemberLeft sends notification when a member leaves an organization.
func (a *Adapter) SendOrgMemberLeft(ctx context.Context, appID xid.ID, recipientEmail, userName, memberName, orgName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyOrgMemberLeft,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":   userName,
			"memberName": memberName,
			"orgName":    orgName,
			"timestamp":  timestamp,
			"appName":    appName,
		},
	})

	return err
}

// SendNewDeviceLogin sends notification when a user logs in from a new device.
func (a *Adapter) SendNewDeviceLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, deviceName, location, timestamp, ipAddress string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyNewDeviceLogin,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":   userName,
			"deviceName": deviceName,
			"location":   location,
			"timestamp":  timestamp,
			"ipAddress":  ipAddress,
			"appName":    appName,
		},
	})

	return err
}

// SendNewLocationLogin sends notification when a user logs in from a new location.
func (a *Adapter) SendNewLocationLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, location, timestamp, ipAddress string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyNewLocationLogin,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"location":  location,
			"timestamp": timestamp,
			"ipAddress": ipAddress,
			"appName":   appName,
		},
	})

	return err
}

// SendSuspiciousLogin sends notification when suspicious login activity is detected.
func (a *Adapter) SendSuspiciousLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, reason, location, timestamp, ipAddress string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeySuspiciousLogin,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"reason":    reason,
			"location":  location,
			"timestamp": timestamp,
			"ipAddress": ipAddress,
			"appName":   appName,
		},
	})

	return err
}

// SendDeviceRemoved sends notification when a device is removed from the account.
func (a *Adapter) SendDeviceRemoved(ctx context.Context, appID xid.ID, recipientEmail, userName, deviceName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyDeviceRemoved,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":   userName,
			"deviceName": deviceName,
			"timestamp":  timestamp,
			"appName":    appName,
		},
	})

	return err
}

// SendAllSessionsRevoked sends notification when all sessions are signed out.
func (a *Adapter) SendAllSessionsRevoked(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyAllSessionsRevoked,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"timestamp": timestamp,
			"appName":   appName,
		},
	})

	return err
}

// SendEmailChangeRequest sends notification when user requests to change their email.
func (a *Adapter) SendEmailChangeRequest(ctx context.Context, appID xid.ID, recipientEmail, userName, newEmail, confirmationUrl, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyEmailChangeRequest,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":        userName,
			"newEmail":        newEmail,
			"confirmationUrl": confirmationUrl,
			"timestamp":       timestamp,
			"appName":         appName,
		},
	})

	return err
}

// SendEmailChanged sends notification when email address is successfully changed.
func (a *Adapter) SendEmailChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, oldEmail, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyEmailChanged,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"oldEmail":  oldEmail,
			"timestamp": timestamp,
			"appName":   appName,
		},
	})

	return err
}

// SendPasswordChanged sends notification when password is changed.
func (a *Adapter) SendPasswordChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyPasswordChanged,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"timestamp": timestamp,
			"appName":   appName,
		},
	})

	return err
}

// SendUsernameChanged sends notification when username is changed.
func (a *Adapter) SendUsernameChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, oldUsername, newUsername, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyUsernameChanged,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":    userName,
			"oldUsername": oldUsername,
			"newUsername": newUsername,
			"timestamp":   timestamp,
			"appName":     appName,
		},
	})

	return err
}

// SendAccountDeleted sends notification when account is deleted.
func (a *Adapter) SendAccountDeleted(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyAccountDeleted,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"timestamp": timestamp,
			"appName":   appName,
		},
	})

	return err
}

// SendAccountSuspended sends notification when account is suspended.
func (a *Adapter) SendAccountSuspended(ctx context.Context, appID xid.ID, recipientEmail, userName, reason, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyAccountSuspended,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"reason":    reason,
			"timestamp": timestamp,
			"appName":   appName,
		},
	})

	return err
}

// SendAccountReactivated sends notification when account is reactivated.
func (a *Adapter) SendAccountReactivated(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	appName := a.getAppName(ctx, appID)

	_, err := a.templateSvc.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: notification.TemplateKeyAccountReactivated,
		Type:        notification.NotificationTypeEmail,
		Recipient:   recipientEmail,
		Variables: map[string]any{
			"userName":  userName,
			"timestamp": timestamp,
			"appName":   appName,
		},
	})

	return err
}
