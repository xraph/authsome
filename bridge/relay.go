package bridge

import "context"

// EventRelay is a local webhook/event relay interface. Implementations
// deliver auth events for webhook delivery (e.g., via relay).
type EventRelay interface {
	// Send emits an auth event for webhook delivery.
	Send(ctx context.Context, event *WebhookEvent) error

	// RegisterEventTypes registers authsome's webhook event catalog.
	RegisterEventTypes(ctx context.Context, defs []WebhookDefinition) error
}

// WebhookEvent represents an auth event for webhook delivery.
type WebhookEvent struct {
	Type           string            `json:"type"`
	TenantID       string            `json:"tenant_id,omitempty"`
	EnvID          string            `json:"env_id,omitempty"`
	Data           map[string]string `json:"data,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// WebhookDefinition describes a webhook event type for catalog registration.
type WebhookDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Group       string `json:"group"`
}

// WebhookEventCatalog returns the default authsome webhook event definitions.
func WebhookEventCatalog() []WebhookDefinition {
	return []WebhookDefinition{
		{Name: "user.created", Description: "New user registered", Group: "user"},
		{Name: "user.updated", Description: "User profile updated", Group: "user"},
		{Name: "user.deleted", Description: "User account deleted", Group: "user"},
		{Name: "user.email_verified", Description: "User email verified", Group: "user"},
		{Name: "user.account_deleted", Description: "User deleted their account", Group: "user"},
		{Name: "session.created", Description: "New login session", Group: "session"},
		{Name: "session.revoked", Description: "Session terminated", Group: "session"},
		{Name: "auth.signin", Description: "Successful sign-in", Group: "auth"},
		{Name: "auth.signin.failed", Description: "Failed sign-in attempt", Group: "auth"},
		{Name: "auth.signout", Description: "User signed out", Group: "auth"},
		{Name: "auth.forgot_password", Description: "Password reset requested", Group: "auth"},
		{Name: "auth.password_reset", Description: "Password reset completed", Group: "auth"},
		{Name: "auth.account_locked", Description: "Account locked after failed attempts", Group: "auth"},
		{Name: "auth.mfa.enabled", Description: "MFA enrolled", Group: "auth"},
		{Name: "org.created", Description: "Organization created", Group: "org"},
		{Name: "org.updated", Description: "Organization updated", Group: "org"},
		{Name: "org.member.invited", Description: "Member invitation sent", Group: "org"},
		{Name: "org.member.joined", Description: "Member accepted invite", Group: "org"},
		{Name: "org.member.removed", Description: "Member removed", Group: "org"},
		{Name: "org.member.role_changed", Description: "Member role updated", Group: "org"},
		{Name: "environment.created", Description: "Environment created", Group: "environment"},
		{Name: "environment.updated", Description: "Environment updated", Group: "environment"},
		{Name: "environment.deleted", Description: "Environment deleted", Group: "environment"},
		{Name: "environment.cloned", Description: "Environment cloned", Group: "environment"},

		// Device events
		{Name: "device.registered", Description: "New device registered", Group: "device"},
		{Name: "device.trusted", Description: "Device marked as trusted", Group: "device"},

		// Webhook events
		{Name: "webhook.created", Description: "Webhook endpoint created", Group: "webhook"},

		// RBAC events
		{Name: "rbac.role.created", Description: "RBAC role created", Group: "rbac"},

		// Admin events
		{Name: "admin.user.banned", Description: "Admin banned a user", Group: "admin"},
		{Name: "admin.user.unbanned", Description: "Admin unbanned a user", Group: "admin"},
		{Name: "admin.user.deleted", Description: "Admin deleted a user", Group: "admin"},
		{Name: "admin.sessions.bulk_revoked", Description: "Admin bulk-revoked sessions", Group: "admin"},
		{Name: "admin.impersonate", Description: "Admin started impersonation", Group: "admin"},

		// MFA events
		{Name: "auth.mfa.enrolled", Description: "MFA method enrolled", Group: "auth"},
		{Name: "auth.mfa.verified", Description: "MFA verification completed", Group: "auth"},
		{Name: "auth.mfa.challenged", Description: "MFA challenge completed", Group: "auth"},
		{Name: "auth.mfa.disabled", Description: "MFA method disabled", Group: "auth"},

		// Passkey events
		{Name: "auth.passkey.registered", Description: "Passkey registered", Group: "auth"},
		{Name: "auth.passkey.authenticated", Description: "Passkey login", Group: "auth"},
		{Name: "auth.passkey.deleted", Description: "Passkey removed", Group: "auth"},

		// Social OAuth events
		{Name: "auth.social.signin", Description: "Social OAuth sign-in", Group: "auth"},
		{Name: "auth.social.signup", Description: "Social OAuth sign-up", Group: "auth"},

		// SSO events
		{Name: "auth.sso.signin", Description: "SSO sign-in", Group: "auth"},
		{Name: "auth.sso.signup", Description: "SSO sign-up", Group: "auth"},

		// API key events
		{Name: "apikey.created", Description: "API key created", Group: "apikey"},
		{Name: "apikey.revoked", Description: "API key revoked", Group: "apikey"},

		// Consent events
		{Name: "consent.granted", Description: "User granted consent", Group: "consent"},
		{Name: "consent.revoked", Description: "User revoked consent", Group: "consent"},
	}
}

// EventRelayFunc is an adapter to use a plain function as an EventRelay send-only implementation.
type EventRelayFunc func(ctx context.Context, event *WebhookEvent) error

// Send implements EventRelay.
func (f EventRelayFunc) Send(ctx context.Context, event *WebhookEvent) error {
	return f(ctx, event)
}

// RegisterEventTypes is a no-op for function adapters.
func (f EventRelayFunc) RegisterEventTypes(context.Context, []WebhookDefinition) error {
	return nil
}
