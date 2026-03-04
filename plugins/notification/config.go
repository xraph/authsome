// Package notification provides a Herald-backed notification plugin for AuthSome.
//
// The plugin subscribes to authsome lifecycle hooks and maps them to Herald
// notification templates. It replaces the legacy email plugin with a unified
// multi-channel notification system.
package notification

// Config configures the notification plugin.
type Config struct {
	// AppName is used in notification template data (e.g. "My App").
	AppName string

	// BaseURL is the application root URL for building links in notifications
	// (e.g. "https://example.com").
	BaseURL string

	// DefaultLocale is the default locale for notifications (e.g. "en").
	// If empty, defaults to "en".
	DefaultLocale string

	// Async controls whether notifications are sent asynchronously via
	// Dispatch (if available). Defaults to false (synchronous).
	Async bool

	// Mappings overrides the default hook-to-notification mappings.
	// Nil entries disable the corresponding hook notification.
	Mappings map[string]*Mapping
}

// Mapping describes how a hook action maps to a Herald notification.
type Mapping struct {
	// Template is the Herald template slug (e.g. "auth.welcome").
	Template string

	// Channels lists the channels to send on (e.g. ["email", "inapp"]).
	Channels []string

	// Enabled controls whether this mapping is active. Defaults to true
	// when the mapping is present.
	Enabled bool
}

// DefaultMappings returns the default hook-to-notification mappings.
// These map authsome lifecycle events to Herald template slugs.
func DefaultMappings() map[string]*Mapping {
	return map[string]*Mapping{
		"auth.signup": {
			Template: "auth.welcome",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"user.create": {
			Template: "auth.email-verification",
			Channels: []string{"email"},
			Enabled:  true,
		},
		"auth.password_reset": {
			Template: "auth.password-reset",
			Channels: []string{"email"},
			Enabled:  true,
		},
		"auth.password_change": {
			Template: "auth.password-changed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"org.invitation.accept": {
			Template: "auth.invitation",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"auth.mfa.enroll": {
			Template: "auth.mfa-code",
			Channels: []string{"sms"},
			Enabled:  true,
		},
		"auth.account_locked": {
			Template: "auth.account-locked",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
	}
}
