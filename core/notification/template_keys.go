package notification

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
)

// =============================================================================
// TEMPLATE METADATA
// =============================================================================

// TemplateMetadata contains metadata about a template including default content
type TemplateMetadata struct {
	Key             string           `json:"key"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Type            NotificationType `json:"type"`
	Variables       []string         `json:"variables"`
	DefaultSubject  string           `json:"defaultSubject"`
	DefaultBody     string           `json:"defaultBody"`
	DefaultBodyHTML string           `json:"defaultBodyHTML,omitempty"`
}

// GetDefaultTemplateMetadata returns metadata for all default templates
func GetDefaultTemplateMetadata() []TemplateMetadata {
	return []TemplateMetadata{
		{
			Key:            TemplateKeyWelcome,
			Name:           "Welcome Email",
			Description:    "Welcome email sent to new users after successful registration",
			Type:           NotificationTypeEmail,
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
			Variables:      []string{"otp", "appName"},
			DefaultSubject: "", // SMS doesn't have subjects
			DefaultBody:    `Your {{.appName}} verification code is: {{.otp}}. Valid for 5 minutes. Do not share this code.`,
		},
		{
			Key:            TemplateKeySecurityAlert,
			Name:           "Security Alert",
			Description:    "Security alert notification for suspicious account activity",
			Type:           NotificationTypeEmail,
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
