package notification

import (
	"regexp"
	"strings"

	"github.com/xraph/authsome/plugins/notification/builder"
)

// =============================================================================
// TEMPLATE KEY CONSTANTS
// =============================================================================

// Template key constants for predefined notification templates
const (
	// Authentication templates
	TemplateKeyWelcome       = "auth.welcome"
	TemplateKeyVerifyEmail   = "auth.verify_email"
	TemplateKeyPasswordReset = "auth.password_reset"
	TemplateKeyMFACode       = "auth.mfa_code"
	TemplateKeyMagicLink     = "auth.magic_link"
	TemplateKeyEmailOTP      = "auth.email_otp"
	TemplateKeyPhoneOTP      = "auth.phone_otp"
	TemplateKeySecurityAlert = "auth.security_alert"

	// Organization templates
	TemplateKeyOrgInvite        = "org.invite"
	TemplateKeyOrgMemberAdded   = "org.member_added"
	TemplateKeyOrgMemberRemoved = "org.member_removed"
	TemplateKeyOrgRoleChanged   = "org.role_changed"
	TemplateKeyOrgTransfer      = "org.transfer"
	TemplateKeyOrgDeleted       = "org.deleted"
	TemplateKeyOrgMemberLeft    = "org.member_left"

	// Account management templates
	TemplateKeyEmailChangeRequest = "account.email_change_request"
	TemplateKeyEmailChanged       = "account.email_changed"
	TemplateKeyPasswordChanged    = "account.password_changed"
	TemplateKeyUsernameChanged    = "account.username_changed"
	TemplateKeyAccountDeleted     = "account.deleted"
	TemplateKeyAccountSuspended   = "account.suspended"
	TemplateKeyAccountReactivated = "account.reactivated"
	TemplateKeyDataExportReady    = "account.data_export_ready"

	// Session/device templates
	TemplateKeyNewDeviceLogin     = "session.new_device"
	TemplateKeyNewLocationLogin   = "session.new_location"
	TemplateKeySuspiciousLogin    = "session.suspicious_login"
	TemplateKeyDeviceRemoved      = "session.device_removed"
	TemplateKeyAllSessionsRevoked = "session.all_revoked"

	// Reminder templates
	TemplateKeyVerificationReminder = "reminder.verification"
	TemplateKeyInactiveAccount      = "reminder.inactive"
	TemplateKeyTrialExpiring        = "reminder.trial_expiring"
	TemplateKeySubscriptionExpiring = "reminder.subscription_expiring"
	TemplateKeyPasswordExpiring     = "reminder.password_expiring"

	// Admin/moderation templates
	TemplateKeyAccountLocked        = "admin.account_locked"
	TemplateKeyAccountUnlocked      = "admin.account_unlocked"
	TemplateKeyTermsUpdate          = "admin.terms_update"
	TemplateKeyPrivacyUpdate        = "admin.privacy_update"
	TemplateKeyMaintenanceScheduled = "admin.maintenance"
	TemplateKeySecurityBreach       = "admin.security_breach"
)

// =============================================================================
// TEMPLATE METADATA
// =============================================================================

// TemplateMetadata contains metadata about a template including default content
type TemplateMetadata struct {
	Key             string               `json:"key"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Type            NotificationType     `json:"type"`
	Priority        NotificationPriority `json:"priority"` // Default priority for this template type
	Variables       []string             `json:"variables"`
	DefaultSubject  string               `json:"defaultSubject"`
	DefaultBody     string               `json:"defaultBody"`
	DefaultBodyHTML string               `json:"defaultBodyHTML,omitempty"`
}

// createTemplateMetadataFromBuilder generates TemplateMetadata from a builder template
func createTemplateMetadataFromBuilder(
	key, name, description, builderKey string,
	variables []string,
	subject string,
) TemplateMetadata {
	return createTemplateMetadataFromBuilderWithPriority(key, name, description, builderKey, variables, subject, PriorityNormal)
}

// createTemplateMetadataFromBuilderWithPriority generates TemplateMetadata with explicit priority
func createTemplateMetadataFromBuilderWithPriority(
	key, name, description, builderKey string,
	variables []string,
	subject string,
	priority NotificationPriority,
) TemplateMetadata {
	// Get builder template
	doc, err := builder.GetSampleTemplate(builderKey)
	if err != nil {
		// Fallback to empty template if builder fails
		return TemplateMetadata{
			Key:            key,
			Name:           name,
			Description:    description,
			Type:           NotificationTypeEmail,
			Priority:       priority,
			Variables:      variables,
			DefaultSubject: subject,
			DefaultBody:    "Template rendering failed",
		}
	}

	// Render to HTML
	renderer := builder.NewRenderer(doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		html = "Template rendering failed"
	}

	// Generate plain text fallback by stripping HTML tags
	textBody := stripHTMLTags(html)

	return TemplateMetadata{
		Key:             key,
		Name:            name,
		Description:     description,
		Type:            NotificationTypeEmail,
		Priority:        priority,
		Variables:       variables,
		DefaultSubject:  subject,
		DefaultBody:     textBody,
		DefaultBodyHTML: html,
	}
}

// stripHTMLTags removes HTML tags for plain text fallback
func stripHTMLTags(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, "")

	// Clean up whitespace
	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	return strings.Join(cleaned, "\n")
}

// GetDefaultTemplateMetadata returns metadata for all default templates
func GetDefaultTemplateMetadata() []TemplateMetadata {
	return []TemplateMetadata{
		{
			Key:            TemplateKeyWelcome,
			Name:           "Welcome Email",
			Description:    "Welcome email sent to new users after successful registration",
			Type:           NotificationTypeEmail,
			Priority:       PriorityNormal, // Welcome emails are nice-to-have
			Variables:      []string{"userName", "appName", "loginURL"},
			DefaultSubject: "Welcome to {{.appName}}!",
			DefaultBody: `Hello {{.userName}},

Welcome to {{.appName}}! We're excited to have you on board.

Your account has been successfully created. You can now log in and start using our services.

Login here: {{.loginURL}}

If you have any questions, feel free to reach out to our support team.

Best regards,
The {{.appName}} Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Welcome to {{.appName}}!</h1>
        <p>Hello {{.userName}},</p>
        <p>Welcome to {{.appName}}! We're excited to have you on board.</p>
        <p>Your account has been successfully created. You can now log in and start using our services.</p>
        <p><a href="{{.loginURL}}" class="button">Login Now</a></p>
        <p>If you have any questions, feel free to reach out to our support team.</p>
        <p>Best regards,<br>The {{.appName}} Team</p>
    </div>
</body>
</html>`,
		},
		{
			Key:            TemplateKeyVerifyEmail,
			Name:           "Email Verification",
			Description:    "Email verification link sent to users to verify their email address",
			Type:           NotificationTypeEmail,
			Priority:       PriorityHigh, // Important but can retry async
			Variables:      []string{"userName", "verificationURL", "code", "appName"},
			DefaultSubject: "Verify your email for {{.appName}}",
			DefaultBody: `Hello {{.userName}},

Please verify your email address to complete your registration with {{.appName}}.

Click the link below to verify your email:
{{.verificationURL}}

Or enter this verification code: {{.code}}

This link will expire in 24 hours.

If you didn't create an account with {{.appName}}, please ignore this email.

Best regards,
The {{.appName}} Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { background-color: #28a745; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block; }
        .code { font-size: 24px; font-weight: bold; letter-spacing: 4px; color: #007bff; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Verify Your Email</h1>
        <p>Hello {{.userName}},</p>
        <p>Please verify your email address to complete your registration with {{.appName}}.</p>
        <p><a href="{{.verificationURL}}" class="button">Verify Email</a></p>
        <p>Or enter this verification code:</p>
        <p class="code">{{.code}}</p>
        <p>This link will expire in 24 hours.</p>
        <p>If you didn't create an account with {{.appName}}, please ignore this email.</p>
        <p>Best regards,<br>The {{.appName}} Team</p>
    </div>
</body>
</html>`,
		},
		{
			Key:            TemplateKeyPasswordReset,
			Name:           "Password Reset",
			Description:    "Password reset link sent to users who request a password reset",
			Type:           NotificationTypeEmail,
			Priority:       PriorityCritical, // Critical - user is blocked without this
			Variables:      []string{"userName", "resetURL", "code", "appName"},
			DefaultSubject: "Reset your password for {{.appName}}",
			DefaultBody: `Hello {{.userName}},

We received a request to reset your password for your {{.appName}} account.

Click the link below to reset your password:
{{.resetURL}}

Or enter this reset code: {{.code}}

This link will expire in 1 hour.

If you didn't request a password reset, please ignore this email or contact support if you have concerns.

Best regards,
The {{.appName}} Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { background-color: #dc3545; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block; }
        .code { font-size: 24px; font-weight: bold; letter-spacing: 4px; color: #dc3545; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Reset Your Password</h1>
        <p>Hello {{.userName}},</p>
        <p>We received a request to reset your password for your {{.appName}} account.</p>
        <p><a href="{{.resetURL}}" class="button">Reset Password</a></p>
        <p>Or enter this reset code:</p>
        <p class="code">{{.code}}</p>
        <p>This link will expire in 1 hour.</p>
        <p>If you didn't request a password reset, please ignore this email or contact support if you have concerns.</p>
        <p>Best regards,<br>The {{.appName}} Team</p>
    </div>
</body>
</html>`,
		},
		{
			Key:            TemplateKeyMFACode,
			Name:           "MFA Code",
			Description:    "Multi-factor authentication code sent to users during login",
			Type:           NotificationTypeEmail,
			Priority:       PriorityCritical, // Critical - blocks auth flow
			Variables:      []string{"userName", "code", "appName"},
			DefaultSubject: "Your {{.appName}} verification code",
			DefaultBody: `Hello {{.userName}},

Your verification code for {{.appName}} is:

{{.code}}

This code will expire in 10 minutes.

If you didn't attempt to log in, please secure your account immediately.

Best regards,
The {{.appName}} Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; text-align: center; }
        .code { font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #007bff; background: #f0f0f0; padding: 20px; border-radius: 8px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Verification Code</h1>
        <p>Hello {{.userName}},</p>
        <p>Your verification code for {{.appName}} is:</p>
        <div class="code">{{.code}}</div>
        <p>This code will expire in 10 minutes.</p>
        <p>If you didn't attempt to log in, please secure your account immediately.</p>
        <p>Best regards,<br>The {{.appName}} Team</p>
    </div>
</body>
</html>`,
		},
		{
			Key:            TemplateKeyMagicLink,
			Name:           "Magic Link",
			Description:    "Passwordless login link sent to users",
			Type:           NotificationTypeEmail,
			Priority:       PriorityCritical, // Critical - required for auth
			Variables:      []string{"userName", "magicURL", "appName"},
			DefaultSubject: "Your {{.appName}} login link",
			DefaultBody: `Hello {{.userName}},

Click the link below to sign in to {{.appName}}:

{{.magicURL}}

This link will expire in 15 minutes and can only be used once.

If you didn't request this link, please ignore this email.

Best regards,
The {{.appName}} Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .button { background-color: #6f42c1; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Sign in to {{.appName}}</h1>
        <p>Hello {{.userName}},</p>
        <p>Click the button below to sign in to {{.appName}}:</p>
        <p><a href="{{.magicURL}}" class="button">Sign In</a></p>
        <p>This link will expire in 15 minutes and can only be used once.</p>
        <p>If you didn't request this link, please ignore this email.</p>
        <p>Best regards,<br>The {{.appName}} Team</p>
    </div>
</body>
</html>`,
		},
		{
			Key:            TemplateKeyEmailOTP,
			Name:           "Email OTP",
			Description:    "One-time password sent via email for authentication",
			Type:           NotificationTypeEmail,
			Priority:       PriorityCritical, // Critical - required for auth
			Variables:      []string{"userName", "otp", "appName"},
			DefaultSubject: "Your {{.appName}} one-time password",
			DefaultBody: `Hello {{.userName}},

Your one-time password (OTP) for {{.appName}} is:

{{.otp}}

This OTP will expire in 5 minutes.

Do not share this OTP with anyone.

Best regards,
The {{.appName}} Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; text-align: center; }
        .otp { font-size: 36px; font-weight: bold; letter-spacing: 10px; color: #28a745; background: #f0f0f0; padding: 20px; border-radius: 8px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>One-Time Password</h1>
        <p>Hello {{.userName}},</p>
        <p>Your one-time password (OTP) for {{.appName}} is:</p>
        <div class="otp">{{.otp}}</div>
        <p>This OTP will expire in 5 minutes.</p>
        <p><strong>Do not share this OTP with anyone.</strong></p>
        <p>Best regards,<br>The {{.appName}} Team</p>
    </div>
</body>
</html>`,
		},
		{
			Key:            TemplateKeyPhoneOTP,
			Name:           "Phone OTP",
			Description:    "One-time password sent via SMS for authentication",
			Type:           NotificationTypeSMS,
			Priority:       PriorityCritical, // Critical - required for auth
			Variables:      []string{"otp", "appName"},
			DefaultSubject: "", // SMS doesn't have subjects
			DefaultBody:    `Your {{.appName}} verification code is: {{.otp}}. Valid for 5 minutes. Do not share this code.`,
		},
		{
			Key:            TemplateKeySecurityAlert,
			Name:           "Security Alert",
			Description:    "Security alert notification for suspicious account activity",
			Type:           NotificationTypeEmail,
			Priority:       PriorityHigh, // High - important but async
			Variables:      []string{"userName", "alertMessage", "timestamp", "ipAddress", "location", "appName"},
			DefaultSubject: "Security alert for your {{.appName}} account",
			DefaultBody: `Hello {{.userName}},

We detected unusual activity on your {{.appName}} account.

Details:
- Activity: {{.alertMessage}}
- Time: {{.timestamp}}
- IP Address: {{.ipAddress}}
- Location: {{.location}}

If this was you, you can safely ignore this email.

If you don't recognize this activity, please secure your account immediately by:
1. Changing your password
2. Reviewing your active sessions
3. Enabling two-factor authentication

Contact support if you need assistance.

Best regards,
The {{.appName}} Security Team`,
			DefaultBodyHTML: `<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .alert { background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; }
        .details { background: #f8f9fa; padding: 15px; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Security Alert</h1>
        <p>Hello {{.userName}},</p>
        <div class="alert">
            <strong>We detected unusual activity on your {{.appName}} account.</strong>
        </div>
        <div class="details">
            <p><strong>Activity:</strong> {{.alertMessage}}</p>
            <p><strong>Time:</strong> {{.timestamp}}</p>
            <p><strong>IP Address:</strong> {{.ipAddress}}</p>
            <p><strong>Location:</strong> {{.location}}</p>
        </div>
        <p>If this was you, you can safely ignore this email.</p>
        <p>If you don't recognize this activity, please secure your account immediately by:</p>
        <ol>
            <li>Changing your password</li>
            <li>Reviewing your active sessions</li>
            <li>Enabling two-factor authentication</li>
        </ol>
        <p>Contact support if you need assistance.</p>
        <p>Best regards,<br>The {{.appName}} Security Team</p>
    </div>
</body>
</html>`,
		},
		// Organization templates
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgInvite,
			"Organization Invitation",
			"Invitation to join an organization or team",
			"org_invite",
			[]string{"userName", "inviterName", "orgName", "role", "inviteURL", "appName", "expiresIn"},
			"{{.inviterName}} invited you to join {{.orgName}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgMemberAdded,
			"Member Added to Organization",
			"Notification when a new member is added to an organization",
			"org_member_added",
			[]string{"userName", "memberName", "orgName", "role", "appName"},
			"{{.memberName}} joined {{.orgName}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgMemberRemoved,
			"Member Removed from Organization",
			"Notification when a member is removed from an organization",
			"org_member_removed",
			[]string{"userName", "memberName", "orgName", "timestamp", "appName"},
			"{{.memberName}} removed from {{.orgName}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgRoleChanged,
			"Role Changed in Organization",
			"Notification when a member's role is changed",
			"org_role_changed",
			[]string{"userName", "orgName", "oldRole", "newRole", "appName"},
			"Your role in {{.orgName}} has been updated",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgTransfer,
			"Organization Ownership Transferred",
			"Notification when organization ownership is transferred",
			"org_transfer",
			[]string{"userName", "orgName", "transferredTo", "timestamp", "appName"},
			"{{.orgName}} ownership transferred",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgDeleted,
			"Organization Deleted",
			"Notification when an organization is deleted",
			"org_deleted",
			[]string{"userName", "orgName", "appName"},
			"{{.orgName}} has been deleted",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyOrgMemberLeft,
			"Member Left Organization",
			"Notification when a member leaves an organization",
			"org_member_left",
			[]string{"userName", "memberName", "orgName", "timestamp", "appName"},
			"{{.memberName}} left {{.orgName}}",
		),
		// Account management templates
		createTemplateMetadataFromBuilder(
			TemplateKeyEmailChangeRequest,
			"Email Change Request",
			"Confirmation request for email address change",
			"email_change_request",
			[]string{"userName", "oldEmail", "newEmail", "confirmURL", "appName"},
			"Confirm your email change for {{.appName}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyEmailChanged,
			"Email Address Changed",
			"Confirmation that email address has been changed",
			"email_changed",
			[]string{"userName", "oldEmail", "newEmail", "changeTime", "appName"},
			"Your email address has been changed",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyPasswordChanged,
			"Password Changed",
			"Confirmation that password has been changed",
			"password_changed",
			[]string{"userName", "changeTime", "appName"},
			"Your password has been changed",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyUsernameChanged,
			"Username Changed",
			"Confirmation that username has been changed",
			"username_changed",
			[]string{"userName", "newUsername", "appName"},
			"Your username has been updated",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyAccountDeleted,
			"Account Deleted",
			"Confirmation that account has been deleted",
			"account_deleted",
			[]string{"userName", "appName"},
			"Your account has been deleted",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyAccountSuspended,
			"Account Suspended",
			"Notification that account has been suspended",
			"account_suspended",
			[]string{"userName", "reason", "suspendedUntil", "appName"},
			"Your account has been suspended",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyAccountReactivated,
			"Account Reactivated",
			"Notification that account has been reactivated",
			"account_reactivated",
			[]string{"userName", "loginURL", "appName"},
			"Welcome back! Your account is active",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyDataExportReady,
			"Data Export Ready",
			"Notification that requested data export is ready for download",
			"data_export_ready",
			[]string{"userName", "downloadURL", "appName"},
			"Your data export is ready",
		),
		// Session/device templates - Low priority (informational, fire-and-forget)
		createTemplateMetadataFromBuilderWithPriority(
			TemplateKeyNewDeviceLogin,
			"New Device Sign-In",
			"Notification of sign-in from a new device",
			"new_device_login",
			[]string{"userName", "deviceName", "browserName", "osName", "location", "timestamp", "confirmURL", "secureAccountURL", "appName"},
			"New device sign-in detected",
			PriorityLow, // Informational - fire and forget
		),
		createTemplateMetadataFromBuilderWithPriority(
			TemplateKeyNewLocationLogin,
			"New Location Sign-In",
			"Notification of sign-in from a new location",
			"new_location_login",
			[]string{"userName", "location", "ipAddress", "timestamp", "secureAccountURL", "appName"},
			"Sign-in from new location",
			PriorityLow, // Informational - fire and forget
		),
		createTemplateMetadataFromBuilderWithPriority(
			TemplateKeySuspiciousLogin,
			"Suspicious Login Detected",
			"Alert for suspicious login attempt",
			"suspicious_login",
			[]string{"userName", "location", "ipAddress", "deviceName", "timestamp", "secureAccountURL", "appName"},
			"Suspicious login attempt on your account",
			PriorityHigh, // Security concern - should retry
		),
		createTemplateMetadataFromBuilderWithPriority(
			TemplateKeyDeviceRemoved,
			"Device Removed",
			"Notification when a device is removed from account",
			"device_removed",
			[]string{"userName", "deviceName", "deviceType", "timestamp", "secureAccountURL", "appName"},
			"Device removed from your account",
			PriorityLow, // Informational - fire and forget
		),
		createTemplateMetadataFromBuilderWithPriority(
			TemplateKeyAllSessionsRevoked,
			"All Sessions Signed Out",
			"Notification when all sessions are revoked for security",
			"all_sessions_revoked",
			[]string{"userName", "loginURL", "appName"},
			"All sessions have been signed out",
			PriorityLow, // Informational - fire and forget
		),
		// Reminder templates
		createTemplateMetadataFromBuilder(
			TemplateKeyVerificationReminder,
			"Email Verification Reminder",
			"Reminder to verify email address",
			"verification_reminder",
			[]string{"userName", "verifyURL", "appName"},
			"Please verify your email for {{.appName}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyInactiveAccount,
			"Inactive Account Reminder",
			"Reminder for inactive user to return",
			"inactive_account",
			[]string{"userName", "loginURL", "appName"},
			"We miss you! Come back to {{.appName}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyTrialExpiring,
			"Trial Expiring",
			"Reminder that trial period is ending soon",
			"trial_expiring",
			[]string{"userName", "planName", "daysRemaining", "expiryDate", "renewURL", "appName"},
			"Your {{.planName}} trial expires in {{.daysRemaining}} days",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeySubscriptionExpiring,
			"Subscription Expiring",
			"Reminder that subscription is expiring soon",
			"subscription_expiring",
			[]string{"userName", "planName", "daysRemaining", "expiryDate", "renewURL", "appName"},
			"Your {{.planName}} subscription expires soon",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyPasswordExpiring,
			"Password Expiring",
			"Reminder to change password before expiry",
			"password_expiring",
			[]string{"userName", "daysRemaining", "changePasswordURL", "appName"},
			"Your password expires in {{.daysRemaining}} days",
		),
		// Admin/moderation templates
		createTemplateMetadataFromBuilder(
			TemplateKeyAccountLocked,
			"Account Locked",
			"Notification that account has been locked by administrator",
			"account_locked",
			[]string{"userName", "lockReason", "unlockTime", "appName"},
			"Your account has been locked",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyAccountUnlocked,
			"Account Unlocked",
			"Notification that account has been unlocked",
			"account_unlocked",
			[]string{"userName", "loginURL", "appName"},
			"Your account has been unlocked",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyTermsUpdate,
			"Terms of Service Updated",
			"Notification of terms of service update",
			"terms_update",
			[]string{"userName", "termsURL", "effectiveDate", "appName"},
			"Our Terms of Service have been updated",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyPrivacyUpdate,
			"Privacy Policy Updated",
			"Notification of privacy policy update",
			"privacy_update",
			[]string{"userName", "privacyURL", "appName"},
			"Our Privacy Policy has been updated",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeyMaintenanceScheduled,
			"Scheduled Maintenance",
			"Notification of upcoming scheduled maintenance",
			"maintenance_scheduled",
			[]string{"userName", "maintenanceStart", "maintenanceEnd", "actionRequired", "appName"},
			"Scheduled maintenance: {{.maintenanceStart}}",
		),
		createTemplateMetadataFromBuilder(
			TemplateKeySecurityBreach,
			"Security Breach Notification",
			"Critical notification about security incident",
			"security_breach",
			[]string{"userName", "breachDetails", "actionRequired", "secureAccountURL", "appName"},
			"URGENT: Security notice for your account",
		),
	}
}

// GetDefaultTemplate retrieves default template metadata by key
func GetDefaultTemplate(key string) (*TemplateMetadata, error) {
	templates := GetDefaultTemplateMetadata()
	for _, template := range templates {
		if template.Key == key {
			return &template, nil
		}
	}
	return nil, TemplateNotFound()
}

// GetTemplateKeysByType returns all template keys for a specific notification type
func GetTemplateKeysByType(notifType NotificationType) []string {
	templates := GetDefaultTemplateMetadata()
	var keys []string
	for _, template := range templates {
		if template.Type == notifType {
			keys = append(keys, template.Key)
		}
	}
	return keys
}

// ValidateTemplateKey checks if a template key is a valid default template key
func ValidateTemplateKey(key string) bool {
	_, err := GetDefaultTemplate(key)
	return err == nil
}

// GetTemplatePriority returns the default priority for a template key
// Returns PriorityNormal if template not found
func GetTemplatePriority(key string) NotificationPriority {
	template, err := GetDefaultTemplate(key)
	if err != nil || template.Priority == "" {
		return PriorityNormal
	}
	return template.Priority
}
