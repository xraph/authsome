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
//
// Skipped hooks (low-value or admin-only): auth.signout, auth.refresh,
// session.create, org.update, org.invitation.decline, org.team.*,
// webhook.*, rbac.role.create/update/delete, environment.*.
func DefaultMappings() map[string]*Mapping {
	return map[string]*Mapping{
		// ─── Auth & Security ─────────────────────────────
		"auth.signup": {
			Template: "auth.welcome",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"auth.signin": {
			Template: "security.signin-alert",
			Channels: []string{"inapp"},
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
		"auth.email_verify": {
			Template: "auth.email-verified",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"auth.mfa.enroll": {
			Template: "auth.mfa-code",
			Channels: []string{"sms"},
			Enabled:  true,
		},
		"auth.mfa.challenge": {
			Template: "auth.mfa-challenge",
			Channels: []string{"sms"},
			Enabled:  true,
		},
		"auth.mfa.recovery_used": {
			Template: "auth.mfa-recovery-used",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"auth.mfa.recovery_regenerated": {
			Template: "auth.mfa-recovery-regenerated",
			Channels: []string{"email"},
			Enabled:  true,
		},
		"auth.account_locked": {
			Template: "auth.account-locked",
			Channels: []string{"email", "sms", "inapp"},
			Enabled:  true,
		},
		"auth.mfa.disable": {
			Template: "auth.mfa-disabled",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── User ────────────────────────────────────────
		"user.create": {
			Template: "auth.email-verification",
			Channels: []string{"email"},
			Enabled:  true,
		},
		"user.update": {
			Template: "user.profile-updated",
			Channels: []string{"inapp"},
			Enabled:  true,
		},
		"user.delete": {
			Template: "user.account-deleted",
			Channels: []string{"email"},
			Enabled:  true,
		},
		"user.account_deletion": {
			Template: "user.account-deleted",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"user.data_export": {
			Template: "user.data-export-ready",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── Session ─────────────────────────────────────
		"session.revoke": {
			Template: "security.session-revoked",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── Organization ────────────────────────────────
		"org.create": {
			Template: "org.created",
			Channels: []string{"inapp"},
			Enabled:  true,
		},
		"org.delete": {
			Template: "org.deleted",
			Channels: []string{"inapp"},
			Enabled:  true,
		},
		"org.member.add": {
			Template: "org.member-added",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"org.member.remove": {
			Template: "org.member-removed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"org.member.role_change": {
			Template: "org.role-changed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"org.invitation.accept": {
			Template: "auth.invitation",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── RBAC ────────────────────────────────────────
		"rbac.role.assign": {
			Template: "org.role-changed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"rbac.role.unassign": {
			Template: "org.role-changed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── Admin ───────────────────────────────────────
		"admin.user.ban": {
			Template: "admin.user-banned",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"admin.user.unban": {
			Template: "admin.user-unbanned",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"admin.user.delete": {
			Template: "user.account-deleted",
			Channels: []string{"email"},
			Enabled:  true,
		},
		"admin.impersonate": {
			Template: "security.signin-alert",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── Credentials ─────────────────────────────────
		"passkey.register": {
			Template: "credential.registered",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"passkey.login": {
			Template: "security.signin-alert",
			Channels: []string{"inapp"},
			Enabled:  true,
		},
		"passkey.delete": {
			Template: "credential.removed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"apikey.create": {
			Template: "credential.registered",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"apikey.revoke": {
			Template: "credential.removed",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── Social / SSO ────────────────────────────────
		"social.signin": {
			Template: "security.signin-alert",
			Channels: []string{"inapp"},
			Enabled:  true,
		},
		"social.signup": {
			Template: "auth.welcome",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"sso.signin": {
			Template: "security.signin-alert",
			Channels: []string{"inapp"},
			Enabled:  true,
		},
		"sso.signup": {
			Template: "auth.welcome",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},

		// ─── Legacy ──────────────────────────────────────
		"auth.new_device_login": {
			Template: "auth.new-device-login",
			Channels: []string{"email", "inapp"},
			Enabled:  true,
		},
		"auth.phone_verification": {
			Template: "auth.phone-verification",
			Channels: []string{"sms"},
			Enabled:  true,
		},
	}
}
