package schema

// DefaultAppSettingsSchemaID is the ID for the default app settings schema
const DefaultAppSettingsSchemaID = "app_settings"

// DefaultAppSettingsSchemaName is the name for the default app settings schema
const DefaultAppSettingsSchemaName = "Application Settings"

// Section IDs
const (
	SectionIDGeneral        = "general"
	SectionIDSecurity       = "security"
	SectionIDSession        = "session"
	SectionIDNotification   = "notification"
	SectionIDAuthentication = "authentication"
	SectionIDBranding       = "branding"
)

// DefaultAppSettingsSchema returns the default schema for app settings
func DefaultAppSettingsSchema() *Schema {
	return &Schema{
		ID:          DefaultAppSettingsSchemaID,
		Name:        DefaultAppSettingsSchemaName,
		Description: "Configure your application settings",
		Version:     1,
		Sections: []*Section{
			GeneralSettingsSection(),
			SecuritySettingsSection(),
			SessionSettingsSection(),
			NotificationSettingsSection(),
			AuthenticationSettingsSection(),
			BrandingSettingsSection(),
		},
	}
}

// GeneralSettingsSection returns the general settings section
func GeneralSettingsSection() *Section {
	return NewSectionBuilder(SectionIDGeneral, "General Settings").
		Description("Configure basic application settings").
		Icon("settings").
		Order(10).
		AddFields(
			NewField("appName", FieldTypeText).
				Label("Application Name").
				Description("The display name of your application").
				Placeholder("My Application").
				Required().
				MaxLength(100).
				Order(1).
				Build(),

			NewField("description", FieldTypeTextArea).
				Label("Description").
				Description("A brief description of your application").
				Placeholder("Enter a description...").
				MaxLength(500).
				Order(2).
				WithMetadata("rows", 3).
				Build(),

			NewField("timezone", FieldTypeSelect).
				Label("Default Timezone").
				Description("The default timezone for your application").
				DefaultValue("UTC").
				Options(timezoneOptions()...).
				Order(3).
				Build(),

			NewField("language", FieldTypeSelect).
				Label("Default Language").
				Description("The default language for your application").
				DefaultValue("en").
				Options(languageOptions()...).
				Order(4).
				Build(),

			NewField("dateFormat", FieldTypeSelect).
				Label("Date Format").
				Description("How dates are displayed throughout the application").
				DefaultValue("YYYY-MM-DD").
				LabeledOptions(
					"YYYY-MM-DD", "2024-01-15 (ISO)",
					"MM/DD/YYYY", "01/15/2024 (US)",
					"DD/MM/YYYY", "15/01/2024 (EU)",
					"DD.MM.YYYY", "15.01.2024 (DE)",
				).
				Order(5).
				Build(),

			NewField("supportEmail", FieldTypeEmail).
				Label("Support Email").
				Description("Email address for user support inquiries").
				Placeholder("support@example.com").
				Order(6).
				Build(),

			NewField("websiteUrl", FieldTypeURL).
				Label("Website URL").
				Description("Your application's public website URL").
				Placeholder("https://example.com").
				Order(7).
				Build(),
		).
		Build()
}

// SecuritySettingsSection returns the security settings section
func SecuritySettingsSection() *Section {
	return NewSectionBuilder(SectionIDSecurity, "Security Settings").
		Description("Configure password policies and security features").
		Icon("shield").
		Order(20).
		AddFields(
			NewField("passwordMinLength", FieldTypeNumber).
				Label("Minimum Password Length").
				Description("Minimum number of characters required for passwords").
				DefaultValue(8).
				Min(6).
				Max(128).
				Order(1).
				Build(),

			NewField("requireUppercase", FieldTypeBoolean).
				Label("Require Uppercase").
				Description("Require at least one uppercase letter in passwords").
				DefaultValue(true).
				Order(2).
				Build(),

			NewField("requireLowercase", FieldTypeBoolean).
				Label("Require Lowercase").
				Description("Require at least one lowercase letter in passwords").
				DefaultValue(true).
				Order(3).
				Build(),

			NewField("requireNumbers", FieldTypeBoolean).
				Label("Require Numbers").
				Description("Require at least one number in passwords").
				DefaultValue(true).
				Order(4).
				Build(),

			NewField("requireSpecialChars", FieldTypeBoolean).
				Label("Require Special Characters").
				Description("Require at least one special character in passwords").
				DefaultValue(false).
				Order(5).
				Build(),

			NewField("maxLoginAttempts", FieldTypeNumber).
				Label("Max Login Attempts").
				Description("Maximum failed login attempts before account lockout (0 = disabled)").
				DefaultValue(5).
				Min(0).
				Max(100).
				Order(6).
				Build(),

			NewField("lockoutDuration", FieldTypeNumber).
				Label("Lockout Duration (minutes)").
				Description("Duration of account lockout after max failed attempts").
				DefaultValue(15).
				Min(1).
				Max(1440).
				DependsOn("maxLoginAttempts", 0).
				Order(7).
				Build(),

			NewField("requireMFA", FieldTypeBoolean).
				Label("Require MFA").
				Description("Require multi-factor authentication for all users").
				DefaultValue(false).
				Order(8).
				Build(),

			NewField("allowedMFAMethods", FieldTypeMultiSelect).
				Label("Allowed MFA Methods").
				Description("Which MFA methods users can enable").
				DependsOn("requireMFA", true).
				Options(
					SelectOption{Value: "totp", Label: "Authenticator App (TOTP)"},
					SelectOption{Value: "sms", Label: "SMS"},
					SelectOption{Value: "email", Label: "Email"},
					SelectOption{Value: "webauthn", Label: "Security Key (WebAuthn)"},
				).
				Order(9).
				Build(),

			NewField("allowedIPAddresses", FieldTypeTags).
				Label("Allowed IP Addresses").
				Description("Restrict access to specific IP addresses or CIDR ranges (leave empty for no restriction)").
				HelpText("Enter IP addresses or CIDR ranges like 192.168.1.0/24").
				Order(10).
				Build(),
		).
		Build()
}

// SessionSettingsSection returns the session settings section
func SessionSettingsSection() *Section {
	return NewSectionBuilder(SectionIDSession, "Session Settings").
		Description("Configure user session behavior").
		Icon("lock").
		Order(30).
		AddFields(
			NewField("sessionDuration", FieldTypeNumber).
				Label("Session Duration (hours)").
				Description("How long a session remains valid after login").
				DefaultValue(24).
				Min(1).
				Max(720).
				Order(1).
				Build(),

			NewField("refreshTokenDuration", FieldTypeNumber).
				Label("Refresh Token Duration (days)").
				Description("How long refresh tokens remain valid").
				DefaultValue(30).
				Min(1).
				Max(365).
				Order(2).
				Build(),

			NewField("idleTimeout", FieldTypeNumber).
				Label("Idle Timeout (minutes)").
				Description("Session expires after this period of inactivity (0 = disabled)").
				DefaultValue(0).
				Min(0).
				Max(1440).
				Order(3).
				Build(),

			NewField("allowMultipleSessions", FieldTypeBoolean).
				Label("Allow Multiple Sessions").
				Description("Allow users to be logged in from multiple devices").
				DefaultValue(true).
				Order(4).
				Build(),

			NewField("maxConcurrentSessions", FieldTypeNumber).
				Label("Max Concurrent Sessions").
				Description("Maximum number of concurrent sessions per user (0 = unlimited)").
				DefaultValue(0).
				Min(0).
				Max(100).
				DependsOn("allowMultipleSessions", true).
				Order(5).
				Build(),

			NewField("rememberMeEnabled", FieldTypeBoolean).
				Label("Enable 'Remember Me'").
				Description("Allow users to stay logged in across browser sessions").
				DefaultValue(true).
				Order(6).
				Build(),

			NewField("rememberMeDuration", FieldTypeNumber).
				Label("Remember Me Duration (days)").
				Description("How long 'Remember Me' keeps users logged in").
				DefaultValue(30).
				Min(1).
				Max(365).
				DependsOn("rememberMeEnabled", true).
				Order(7).
				Build(),

			NewField("cookieSameSite", FieldTypeSelect).
				Label("Cookie SameSite Policy").
				Description("SameSite attribute for session cookies").
				DefaultValue("lax").
				LabeledOptions(
					"strict", "Strict",
					"lax", "Lax",
					"none", "None",
				).
				Order(8).
				Build(),

			NewField("cookieSecure", FieldTypeBoolean).
				Label("Secure Cookies Only").
				Description("Only send cookies over HTTPS connections").
				DefaultValue(true).
				Order(9).
				Build(),
		).
		Build()
}

// NotificationSettingsSection returns the notification settings section
func NotificationSettingsSection() *Section {
	return NewSectionBuilder(SectionIDNotification, "Notification Settings").
		Description("Configure email and SMS notification preferences").
		Icon("bell").
		Order(40).
		AddFields(
			NewField("emailNotificationsEnabled", FieldTypeBoolean).
				Label("Enable Email Notifications").
				Description("Send email notifications to users").
				DefaultValue(true).
				Order(1).
				Build(),

			NewField("smsNotificationsEnabled", FieldTypeBoolean).
				Label("Enable SMS Notifications").
				Description("Send SMS notifications to users").
				DefaultValue(false).
				Order(2).
				Build(),

			NewField("emailFromAddress", FieldTypeEmail).
				Label("From Email Address").
				Description("Email address used as the sender for notifications").
				Placeholder("noreply@example.com").
				DependsOn("emailNotificationsEnabled", true).
				Order(3).
				Build(),

			NewField("emailFromName", FieldTypeText).
				Label("From Name").
				Description("Display name for the email sender").
				Placeholder("My Application").
				MaxLength(100).
				DependsOn("emailNotificationsEnabled", true).
				Order(4).
				Build(),

			NewField("emailReplyTo", FieldTypeEmail).
				Label("Reply-To Address").
				Description("Email address for replies (optional)").
				Placeholder("support@example.com").
				DependsOn("emailNotificationsEnabled", true).
				Order(5).
				Build(),

			NewField("welcomeEmailEnabled", FieldTypeBoolean).
				Label("Send Welcome Email").
				Description("Send a welcome email when users sign up").
				DefaultValue(true).
				DependsOn("emailNotificationsEnabled", true).
				Order(6).
				Build(),

			NewField("passwordResetNotification", FieldTypeBoolean).
				Label("Password Reset Notification").
				Description("Notify users when their password is changed").
				DefaultValue(true).
				DependsOn("emailNotificationsEnabled", true).
				Order(7).
				Build(),

			NewField("loginAlertEnabled", FieldTypeBoolean).
				Label("New Login Alerts").
				Description("Alert users when a new device logs into their account").
				DefaultValue(false).
				DependsOn("emailNotificationsEnabled", true).
				Order(8).
				Build(),
		).
		Build()
}

// AuthenticationSettingsSection returns the authentication settings section
func AuthenticationSettingsSection() *Section {
	return NewSectionBuilder(SectionIDAuthentication, "Authentication Methods").
		Description("Configure which authentication methods are available").
		Icon("key").
		Order(50).
		AddFields(
			NewField("emailPasswordEnabled", FieldTypeBoolean).
				Label("Email/Password Login").
				Description("Allow users to sign in with email and password").
				DefaultValue(true).
				Order(1).
				Build(),

			NewField("usernamePasswordEnabled", FieldTypeBoolean).
				Label("Username/Password Login").
				Description("Allow users to sign in with username and password").
				DefaultValue(false).
				Order(2).
				Build(),

			NewField("magicLinkEnabled", FieldTypeBoolean).
				Label("Magic Link Login").
				Description("Allow passwordless login via email links").
				DefaultValue(false).
				Order(3).
				Build(),

			NewField("magicLinkExpiry", FieldTypeNumber).
				Label("Magic Link Expiry (minutes)").
				Description("How long magic links remain valid").
				DefaultValue(15).
				Min(5).
				Max(60).
				DependsOn("magicLinkEnabled", true).
				Order(4).
				Build(),

			NewField("phoneAuthEnabled", FieldTypeBoolean).
				Label("Phone Number Authentication").
				Description("Allow users to sign in with their phone number").
				DefaultValue(false).
				Order(5).
				Build(),

			NewField("passkeyEnabled", FieldTypeBoolean).
				Label("Passkey Login").
				Description("Allow passwordless login with passkeys/WebAuthn").
				DefaultValue(false).
				Order(6).
				Build(),

			NewField("anonymousEnabled", FieldTypeBoolean).
				Label("Anonymous Sessions").
				Description("Allow users to use the app without signing up").
				DefaultValue(false).
				Order(7).
				Build(),

			NewField("signupEnabled", FieldTypeBoolean).
				Label("Allow Sign Up").
				Description("Allow new users to create accounts").
				DefaultValue(true).
				Order(8).
				Build(),

			NewField("emailVerificationRequired", FieldTypeBoolean).
				Label("Require Email Verification").
				Description("Require users to verify their email before accessing the app").
				DefaultValue(true).
				Order(9).
				Build(),
		).
		Build()
}

// BrandingSettingsSection returns the branding settings section
func BrandingSettingsSection() *Section {
	return NewSectionBuilder(SectionIDBranding, "Branding").
		Description("Customize the look and feel of authentication pages").
		Icon("globe").
		Order(60).
		Collapsible().
		AddFields(
			NewField("logoUrl", FieldTypeURL).
				Label("Logo URL").
				Description("URL to your application logo (recommended: 200x50px)").
				Placeholder("https://example.com/logo.png").
				Order(1).
				Build(),

			NewField("faviconUrl", FieldTypeURL).
				Label("Favicon URL").
				Description("URL to your application favicon (recommended: 32x32px)").
				Placeholder("https://example.com/favicon.ico").
				Order(2).
				Build(),

			NewField("primaryColor", FieldTypeColor).
				Label("Primary Color").
				Description("Main brand color for buttons and accents").
				DefaultValue("#3B82F6").
				Order(3).
				Build(),

			NewField("backgroundColor", FieldTypeColor).
				Label("Background Color").
				Description("Background color for authentication pages").
				DefaultValue("#FFFFFF").
				Order(4).
				Build(),

			NewField("fontFamily", FieldTypeSelect).
				Label("Font Family").
				Description("Font used for authentication pages").
				DefaultValue("system-ui").
				LabeledOptions(
					"system-ui", "System Default",
					"Inter", "Inter",
					"Roboto", "Roboto",
					"Open Sans", "Open Sans",
					"Lato", "Lato",
					"Poppins", "Poppins",
				).
				Order(5).
				Build(),

			NewField("customCSS", FieldTypeTextArea).
				Label("Custom CSS").
				Description("Additional CSS to apply to authentication pages").
				Placeholder("/* Custom styles */").
				Order(6).
				WithMetadata("rows", 6).
				Build(),

			NewField("termsUrl", FieldTypeURL).
				Label("Terms of Service URL").
				Description("Link to your terms of service").
				Placeholder("https://example.com/terms").
				Order(7).
				Build(),

			NewField("privacyUrl", FieldTypeURL).
				Label("Privacy Policy URL").
				Description("Link to your privacy policy").
				Placeholder("https://example.com/privacy").
				Order(8).
				Build(),
		).
		Build()
}

// timezoneOptions returns common timezone options
func timezoneOptions() []SelectOption {
	return []SelectOption{
		{Value: "UTC", Label: "UTC"},
		{Value: "America/New_York", Label: "Eastern Time (US)", Group: "Americas"},
		{Value: "America/Chicago", Label: "Central Time (US)", Group: "Americas"},
		{Value: "America/Denver", Label: "Mountain Time (US)", Group: "Americas"},
		{Value: "America/Los_Angeles", Label: "Pacific Time (US)", Group: "Americas"},
		{Value: "America/Sao_Paulo", Label: "Sao Paulo", Group: "Americas"},
		{Value: "Europe/London", Label: "London", Group: "Europe"},
		{Value: "Europe/Paris", Label: "Paris", Group: "Europe"},
		{Value: "Europe/Berlin", Label: "Berlin", Group: "Europe"},
		{Value: "Europe/Moscow", Label: "Moscow", Group: "Europe"},
		{Value: "Asia/Dubai", Label: "Dubai", Group: "Asia"},
		{Value: "Asia/Singapore", Label: "Singapore", Group: "Asia"},
		{Value: "Asia/Tokyo", Label: "Tokyo", Group: "Asia"},
		{Value: "Asia/Shanghai", Label: "Shanghai", Group: "Asia"},
		{Value: "Asia/Kolkata", Label: "India (IST)", Group: "Asia"},
		{Value: "Australia/Sydney", Label: "Sydney", Group: "Pacific"},
		{Value: "Pacific/Auckland", Label: "Auckland", Group: "Pacific"},
	}
}

// languageOptions returns supported language options
func languageOptions() []SelectOption {
	return []SelectOption{
		{Value: "en", Label: "English"},
		{Value: "es", Label: "Español"},
		{Value: "fr", Label: "Français"},
		{Value: "de", Label: "Deutsch"},
		{Value: "it", Label: "Italiano"},
		{Value: "pt", Label: "Português"},
		{Value: "zh", Label: "中文"},
		{Value: "ja", Label: "日本語"},
		{Value: "ko", Label: "한국어"},
		{Value: "ar", Label: "العربية"},
		{Value: "hi", Label: "हिन्दी"},
		{Value: "ru", Label: "Русский"},
	}
}

// RegisterDefaultSections registers all default sections in the global registry
func RegisterDefaultSections() error {
	schema := DefaultAppSettingsSchema()
	for _, section := range schema.Sections {
		if err := RegisterSection(section); err != nil {
			return err
		}
	}
	return nil
}
