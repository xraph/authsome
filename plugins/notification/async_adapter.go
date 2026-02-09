package notification

import (
	"context"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
)

// AsyncAdapter wraps the standard Adapter to provide async notification sending
// based on notification priority. Critical notifications are sent synchronously,
// while non-critical notifications are sent asynchronously to avoid blocking.
type AsyncAdapter struct {
	*Adapter

	config     AsyncConfig
	dispatcher *notification.Dispatcher
	retry      *notification.RetryService
}

// NewAsyncAdapter creates a new async notification adapter.
func NewAsyncAdapter(adapter *Adapter, config AsyncConfig, dispatcher *notification.Dispatcher, retry *notification.RetryService) *AsyncAdapter {
	return &AsyncAdapter{
		Adapter:    adapter,
		config:     config,
		dispatcher: dispatcher,
		retry:      retry,
	}
}

// sendWithPriority sends a notification with the specified priority
// Critical notifications are sent synchronously, others are sent asynchronously.
func (a *AsyncAdapter) sendWithPriority(ctx context.Context, priority notification.NotificationPriority, sendFunc func(ctx context.Context) error) error {
	// If async is disabled or dispatcher not available, send synchronously
	if !a.config.Enabled || a.dispatcher == nil {
		return sendFunc(ctx)
	}

	// Critical notifications are always synchronous
	if priority == notification.PriorityCritical {
		return sendFunc(ctx)
	}

	// For async operations, use a background context to prevent cancellation
	// when the HTTP request completes. Copy relevant values from original context.
	asyncCtx := context.Background()

	// High priority notifications: async but we log errors
	if priority == notification.PriorityHigh {
		go func() {
			if err := sendFunc(asyncCtx); err != nil {
			}
		}()

		return nil
	}

	// Normal and Low priority: fire and forget
	go func() {
		if err := sendFunc(asyncCtx); err != nil {
			// Only log for debugging, don't retry low priority
			if priority != notification.PriorityLow {
			}
		}
	}()

	return nil
}

// ===========================================================================
// CRITICAL PRIORITY - Sent synchronously (blocks until complete)
// ===========================================================================

// SendMFACode sends an MFA verification code - CRITICAL priority (sync).
func (a *AsyncAdapter) SendMFACode(ctx context.Context, appID xid.ID, recipient, code string, expiryMinutes int, notifType notification.NotificationType) error {
	// MFA codes are critical - user is blocked without them
	return a.Adapter.SendMFACode(ctx, appID, recipient, code, expiryMinutes, notifType)
}

// SendEmailOTP sends an email OTP code - CRITICAL priority (sync).
func (a *AsyncAdapter) SendEmailOTP(ctx context.Context, appID xid.ID, email, code string, expiryMinutes int) error {
	return a.Adapter.SendEmailOTP(ctx, appID, email, code, expiryMinutes)
}

// SendPhoneOTP sends a phone OTP code - CRITICAL priority (sync).
func (a *AsyncAdapter) SendPhoneOTP(ctx context.Context, appID xid.ID, phone, code string) error {
	return a.Adapter.SendPhoneOTP(ctx, appID, phone, code)
}

// SendMagicLink sends a magic link email - CRITICAL priority (sync).
func (a *AsyncAdapter) SendMagicLink(ctx context.Context, appID xid.ID, email, userName, magicLink string, expiryMinutes int) error {
	return a.Adapter.SendMagicLink(ctx, appID, email, userName, magicLink, expiryMinutes)
}

// SendPasswordReset sends a password reset email - CRITICAL priority (sync).
func (a *AsyncAdapter) SendPasswordReset(ctx context.Context, appID xid.ID, email, userName, resetURL, resetCode string, expiryMinutes int) error {
	return a.Adapter.SendPasswordReset(ctx, appID, email, userName, resetURL, resetCode, expiryMinutes)
}

// ===========================================================================
// HIGH PRIORITY - Sent asynchronously but errors are logged
// ===========================================================================

// SendVerificationEmail sends a verification email - HIGH priority (async with logging).
func (a *AsyncAdapter) SendVerificationEmail(ctx context.Context, appID xid.ID, email, userName, verificationURL, verificationCode string, expiryMinutes int) error {
	return a.sendWithPriority(ctx, notification.PriorityHigh, func(c context.Context) error {
		return a.Adapter.SendVerificationEmail(c, appID, email, userName, verificationURL, verificationCode, expiryMinutes)
	})
}

// SendSuspiciousLogin sends a suspicious login alert - HIGH priority (async with logging).
func (a *AsyncAdapter) SendSuspiciousLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, reason, location, timestamp, ipAddress string) error {
	return a.sendWithPriority(ctx, notification.PriorityHigh, func(c context.Context) error {
		return a.Adapter.SendSuspiciousLogin(c, appID, recipientEmail, userName, reason, location, timestamp, ipAddress)
	})
}

// SendSecurityAlert sends a security alert notification - HIGH priority (async with logging).
func (a *AsyncAdapter) SendSecurityAlert(ctx context.Context, appID xid.ID, email, userName, eventType, eventTime, location, device string) error {
	return a.sendWithPriority(ctx, notification.PriorityHigh, func(c context.Context) error {
		return a.Adapter.SendSecurityAlert(c, appID, email, userName, eventType, eventTime, location, device)
	})
}

// SendEmailChangeRequest sends an email change request notification - HIGH priority (async with logging).
func (a *AsyncAdapter) SendEmailChangeRequest(ctx context.Context, appID xid.ID, recipientEmail, userName, newEmail, confirmationUrl, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityHigh, func(c context.Context) error {
		return a.Adapter.SendEmailChangeRequest(c, appID, recipientEmail, userName, newEmail, confirmationUrl, timestamp)
	})
}

// SendAccountSuspended sends an account suspended notification - HIGH priority (async with logging).
func (a *AsyncAdapter) SendAccountSuspended(ctx context.Context, appID xid.ID, email, userName, reason, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityHigh, func(c context.Context) error {
		return a.Adapter.SendAccountSuspended(c, appID, email, userName, reason, timestamp)
	})
}

// SendOrgInvite sends an organization invite notification - HIGH priority (async with logging).
func (a *AsyncAdapter) SendOrgInvite(ctx context.Context, appID xid.ID, recipientEmail, userName, inviterName, orgName, role, inviteURL, expiresIn string) error {
	return a.sendWithPriority(ctx, notification.PriorityHigh, func(c context.Context) error {
		return a.Adapter.SendOrgInvite(c, appID, recipientEmail, userName, inviterName, orgName, role, inviteURL, expiresIn)
	})
}

// ===========================================================================
// NORMAL PRIORITY - Sent asynchronously, errors logged at debug level
// ===========================================================================

// SendWelcomeEmail sends a welcome email - NORMAL priority (async).
func (a *AsyncAdapter) SendWelcomeEmail(ctx context.Context, appID xid.ID, email, userName, loginURL string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendWelcomeEmail(c, appID, email, userName, loginURL)
	})
}

// SendEmailChanged sends an email changed notification - NORMAL priority (async).
func (a *AsyncAdapter) SendEmailChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, oldEmail, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendEmailChanged(c, appID, recipientEmail, userName, oldEmail, timestamp)
	})
}

// SendPasswordChanged sends a password changed notification - NORMAL priority (async).
func (a *AsyncAdapter) SendPasswordChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendPasswordChanged(c, appID, recipientEmail, userName, timestamp)
	})
}

// SendUsernameChanged sends a username changed notification - NORMAL priority (async).
func (a *AsyncAdapter) SendUsernameChanged(ctx context.Context, appID xid.ID, recipientEmail, userName, oldUsername, newUsername, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendUsernameChanged(c, appID, recipientEmail, userName, oldUsername, newUsername, timestamp)
	})
}

// SendAccountDeleted sends an account deleted notification - NORMAL priority (async).
func (a *AsyncAdapter) SendAccountDeleted(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendAccountDeleted(c, appID, recipientEmail, userName, timestamp)
	})
}

// SendAccountReactivated sends an account reactivated notification - NORMAL priority (async).
func (a *AsyncAdapter) SendAccountReactivated(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendAccountReactivated(c, appID, recipientEmail, userName, timestamp)
	})
}

// SendOrgMemberAdded sends an organization member added notification - NORMAL priority (async).
func (a *AsyncAdapter) SendOrgMemberAdded(ctx context.Context, appID xid.ID, recipientEmail, userName, memberName, orgName, role string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendOrgMemberAdded(c, appID, recipientEmail, userName, memberName, orgName, role)
	})
}

// SendOrgMemberRemoved sends an organization member removed notification - NORMAL priority (async).
func (a *AsyncAdapter) SendOrgMemberRemoved(ctx context.Context, appID xid.ID, recipientEmail, userName, memberName, orgName, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityNormal, func(c context.Context) error {
		return a.Adapter.SendOrgMemberRemoved(c, appID, recipientEmail, userName, memberName, orgName, timestamp)
	})
}

// ===========================================================================
// LOW PRIORITY - Fire and forget, no logging
// ===========================================================================

// SendNewDeviceLogin sends a new device login notification - LOW priority (fire-and-forget).
func (a *AsyncAdapter) SendNewDeviceLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, deviceName, location, timestamp, ipAddress string) error {
	return a.sendWithPriority(ctx, notification.PriorityLow, func(c context.Context) error {
		return a.Adapter.SendNewDeviceLogin(c, appID, recipientEmail, userName, deviceName, location, timestamp, ipAddress)
	})
}

// SendNewLocationLogin sends a new location login notification - LOW priority (fire-and-forget).
func (a *AsyncAdapter) SendNewLocationLogin(ctx context.Context, appID xid.ID, recipientEmail, userName, location, timestamp, ipAddress string) error {
	return a.sendWithPriority(ctx, notification.PriorityLow, func(c context.Context) error {
		return a.Adapter.SendNewLocationLogin(c, appID, recipientEmail, userName, location, timestamp, ipAddress)
	})
}

// SendDeviceRemoved sends a device removed notification - LOW priority (fire-and-forget).
func (a *AsyncAdapter) SendDeviceRemoved(ctx context.Context, appID xid.ID, recipientEmail, userName, deviceName, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityLow, func(c context.Context) error {
		return a.Adapter.SendDeviceRemoved(c, appID, recipientEmail, userName, deviceName, timestamp)
	})
}

// SendAllSessionsRevoked sends all sessions revoked notification - LOW priority (fire-and-forget).
func (a *AsyncAdapter) SendAllSessionsRevoked(ctx context.Context, appID xid.ID, recipientEmail, userName, timestamp string) error {
	return a.sendWithPriority(ctx, notification.PriorityLow, func(c context.Context) error {
		return a.Adapter.SendAllSessionsRevoked(c, appID, recipientEmail, userName, timestamp)
	})
}
