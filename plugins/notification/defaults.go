package notification

// TemplateDefault represents a default template
type TemplateDefault struct {
	TemplateKey string
	Type        string
	Subject     string
	BodyText    string
	BodyHTML    string
	Variables   []string
	Description string
}

// DefaultTemplates returns the default notification templates
func DefaultTemplates() []TemplateDefault {
	return []TemplateDefault{
		// Welcome Email
		{
			TemplateKey: "auth.welcome",
			Type:        "email",
			Subject:     "Welcome to {{.app_name}}!",
			BodyText: `Hi {{.user_name}},

Welcome to {{.app_name}}! We're excited to have you on board.

Your account has been successfully created with the email: {{.user_email}}

Get started by logging in to your account.

Best regards,
The {{.app_name}} Team`,
			BodyHTML: `<html>
<body>
<h1>Welcome to {{.app_name}}!</h1>
<p>Hi {{.user_name}},</p>
<p>Welcome to {{.app_name}}! We're excited to have you on board.</p>
<p>Your account has been successfully created with the email: <strong>{{.user_email}}</strong></p>
<p><a href="{{.login_url}}">Get started by logging in to your account</a></p>
<p>Best regards,<br>The {{.app_name}} Team</p>
</body>
</html>`,
			Variables:   []string{"user_name", "user_email", "app_name", "login_url"},
			Description: "Welcome email sent to new users",
		},

		// Email Verification
		{
			TemplateKey: "auth.verify_email",
			Type:        "email",
			Subject:     "Verify your email address",
			BodyText: `Hi {{.user_name}},

Please verify your email address by clicking the link below:

{{.verification_url}}

Or enter this verification code: {{.verification_code}}

This code expires in {{.expiry_minutes}} minutes.

If you didn't create this account, you can safely ignore this email.

Best regards,
The {{.app_name}} Team`,
			BodyHTML: `<html>
<body>
<h2>Verify your email address</h2>
<p>Hi {{.user_name}},</p>
<p>Please verify your email address by clicking the button below:</p>
<p><a href="{{.verification_url}}" style="background-color: #4CAF50; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px;">Verify Email</a></p>
<p>Or enter this verification code: <strong>{{.verification_code}}</strong></p>
<p>This code expires in {{.expiry_minutes}} minutes.</p>
<p>If you didn't create this account, you can safely ignore this email.</p>
<p>Best regards,<br>The {{.app_name}} Team</p>
</body>
</html>`,
			Variables:   []string{"user_name", "verification_url", "verification_code", "expiry_minutes", "app_name"},
			Description: "Email verification message",
		},

		// Password Reset
		{
			TemplateKey: "auth.password_reset",
			Type:        "email",
			Subject:     "Reset your password",
			BodyText: `Hi {{.user_name}},

You requested to reset your password. Click the link below to create a new password:

{{.reset_url}}

Or enter this reset code: {{.reset_code}}

This code expires in {{.expiry_minutes}} minutes.

If you didn't request this, please ignore this email and your password will remain unchanged.

Best regards,
The {{.app_name}} Team`,
			BodyHTML: `<html>
<body>
<h2>Reset your password</h2>
<p>Hi {{.user_name}},</p>
<p>You requested to reset your password. Click the button below to create a new password:</p>
<p><a href="{{.reset_url}}" style="background-color: #f44336; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px;">Reset Password</a></p>
<p>Or enter this reset code: <strong>{{.reset_code}}</strong></p>
<p>This code expires in {{.expiry_minutes}} minutes.</p>
<p>If you didn't request this, please ignore this email and your password will remain unchanged.</p>
<p>Best regards,<br>The {{.app_name}} Team</p>
</body>
</html>`,
			Variables:   []string{"user_name", "reset_url", "reset_code", "expiry_minutes", "app_name"},
			Description: "Password reset email",
		},

		// MFA Code
		{
			TemplateKey: "auth.mfa_code",
			Type:        "email",
			Subject:     "Your verification code",
			BodyText: `Hi {{.user_name}},

Your verification code is: {{.code}}

This code expires in {{.expiry_minutes}} minutes.

If you didn't request this code, please contact support immediately.

Best regards,
The {{.app_name}} Team`,
			BodyHTML: `<html>
<body>
<h2>Your verification code</h2>
<p>Hi {{.user_name}},</p>
<p>Your verification code is:</p>
<h1 style="font-size: 48px; letter-spacing: 8px; color: #4CAF50;">{{.code}}</h1>
<p>This code expires in {{.expiry_minutes}} minutes.</p>
<p>If you didn't request this code, please contact support immediately.</p>
<p>Best regards,<br>The {{.app_name}} Team</p>
</body>
</html>`,
			Variables:   []string{"user_name", "code", "expiry_minutes", "app_name"},
			Description: "MFA verification code email",
		},

		// MFA SMS Code
		{
			TemplateKey: "auth.mfa_code",
			Type:        "sms",
			Subject:     "",
			BodyText:    `Your {{.app_name}} verification code is: {{.code}}. Expires in {{.expiry_minutes}} minutes.`,
			Variables:   []string{"code", "expiry_minutes", "app_name"},
			Description: "MFA verification code SMS",
		},

		// Magic Link
		{
			TemplateKey: "auth.magic_link",
			Type:        "email",
			Subject:     "Your magic sign-in link",
			BodyText: `Hi {{.user_name}},

Click the link below to sign in to your account:

{{.magic_link}}

This link expires in {{.expiry_minutes}} minutes.

If you didn't request this link, please ignore this email.

Best regards,
The {{.app_name}} Team`,
			BodyHTML: `<html>
<body>
<h2>Your magic sign-in link</h2>
<p>Hi {{.user_name}},</p>
<p>Click the button below to sign in to your account:</p>
<p><a href="{{.magic_link}}" style="background-color: #2196F3; color: white; padding: 14px 20px; text-decoration: none; border-radius: 4px;">Sign In</a></p>
<p>This link expires in {{.expiry_minutes}} minutes.</p>
<p>If you didn't request this link, please ignore this email.</p>
<p>Best regards,<br>The {{.app_name}} Team</p>
</body>
</html>`,
			Variables:   []string{"user_name", "magic_link", "expiry_minutes", "app_name"},
			Description: "Magic link sign-in email",
		},

		// Email OTP
		{
			TemplateKey: "auth.email_otp",
			Type:        "email",
			Subject:     "Your one-time code",
			BodyText: `Your one-time code for {{.app_name}} is: {{.code}}

This code expires in {{.expiry_minutes}} minutes.

If you didn't request this code, please ignore this email.`,
			BodyHTML: `<html>
<body>
<h2>Your one-time code</h2>
<p>Your one-time code for {{.app_name}} is:</p>
<h1 style="font-size: 48px; letter-spacing: 8px; color: #FF9800;">{{.code}}</h1>
<p>This code expires in {{.expiry_minutes}} minutes.</p>
<p>If you didn't request this code, please ignore this email.</p>
</body>
</html>`,
			Variables:   []string{"code", "expiry_minutes", "app_name"},
			Description: "Email OTP code",
		},

		// Phone OTP
		{
			TemplateKey: "auth.phone_otp",
			Type:        "sms",
			Subject:     "",
			BodyText:    `Your {{.app_name}} code is: {{.code}}`,
			Variables:   []string{"code", "app_name"},
			Description: "Phone OTP SMS code",
		},

		// Account Security Alert
		{
			TemplateKey: "auth.security_alert",
			Type:        "email",
			Subject:     "Security alert for your account",
			BodyText: `Hi {{.user_name}},

We detected a security event on your account:

Event: {{.event_type}}
Time: {{.event_time}}
Location: {{.location}}
Device: {{.device}}

If this was you, you can safely ignore this email. If you don't recognize this activity, please secure your account immediately by changing your password.

Best regards,
The {{.app_name}} Team`,
			BodyHTML: `<html>
<body>
<h2>Security alert for your account</h2>
<p>Hi {{.user_name}},</p>
<p>We detected a security event on your account:</p>
<ul>
<li><strong>Event:</strong> {{.event_type}}</li>
<li><strong>Time:</strong> {{.event_time}}</li>
<li><strong>Location:</strong> {{.location}}</li>
<li><strong>Device:</strong> {{.device}}</li>
</ul>
<p>If this was you, you can safely ignore this email. If you don't recognize this activity, please secure your account immediately by changing your password.</p>
<p>Best regards,<br>The {{.app_name}} Team</p>
</body>
</html>`,
			Variables:   []string{"user_name", "event_type", "event_time", "location", "device", "app_name"},
			Description: "Security alert notification",
		},
	}
}
