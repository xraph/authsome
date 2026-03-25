// Package hook provides a global event bus that fires on every engine action.
// This is the single interception point for logging, auditing, metrics,
// and custom logic.
package hook

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"
)

// Event represents a global hook event emitted on every engine action.
type Event struct {
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	ResourceID string            `json:"resource_id,omitempty"`
	ActorID    string            `json:"actor_id,omitempty"`
	Tenant     string            `json:"tenant,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Err        error             `json:"-"`
}

// Handler is a function that handles a global hook event.
type Handler func(ctx context.Context, event *Event) error

type namedHandler struct {
	name    string
	handler Handler
}

// Bus dispatches global hook events to all registered handlers.
type Bus struct {
	handlers []namedHandler
	logger   log.Logger
}

// NewBus creates a new event bus with the given logger.
func NewBus(logger log.Logger) *Bus {
	return &Bus{logger: logger}
}

// On registers a named handler for all events.
func (b *Bus) On(name string, handler Handler) {
	b.handlers = append(b.handlers, namedHandler{name, handler})
}

// Emit dispatches an event to all registered handlers.
// Errors from handlers are logged but never propagated.
func (b *Bus) Emit(ctx context.Context, event *Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	for _, h := range b.handlers {
		if err := h.handler(ctx, event); err != nil {
			b.logger.Warn("global hook error",
				log.String("hook", h.name),
				log.String("action", event.Action),
				log.String("error", err.Error()),
			)
		}
	}
}

// ──────────────────────────────────────────────────
// Action constants
// ──────────────────────────────────────────────────

const (
	ActionSignUp  = "auth.signup"
	ActionSignIn  = "auth.signin"
	ActionSignOut = "auth.signout"
	ActionRefresh = "auth.refresh"

	ActionUserCreate = "user.create"
	ActionUserUpdate = "user.update"
	ActionUserDelete = "user.delete"

	ActionSessionCreate = "session.create"
	ActionSessionRevoke = "session.revoke"

	ActionOrgCreate        = "org.create"
	ActionOrgUpdate        = "org.update"
	ActionOrgDelete        = "org.delete"
	ActionMemberAdd        = "org.member.add"
	ActionMemberRemove     = "org.member.remove"
	ActionMemberRoleChange = "org.member.role_change"

	ActionInvitationAccept  = "org.invitation.accept"
	ActionInvitationDecline = "org.invitation.decline"

	ActionTeamCreate = "org.team.create"
	ActionTeamUpdate = "org.team.update"
	ActionTeamDelete = "org.team.delete"

	ActionWebhookCreate = "webhook.create"
	ActionWebhookUpdate = "webhook.update"
	ActionWebhookDelete = "webhook.delete"

	ActionPasswordReset          = "auth.password_reset"
	ActionPasswordChange         = "auth.password_change"
	ActionEmailVerify            = "auth.email_verify"
	ActionMFAEnroll              = "auth.mfa.enroll"
	ActionMFAChallenge           = "auth.mfa.challenge"
	ActionMFARecoveryUsed        = "auth.mfa.recovery_used"
	ActionMFARecoveryRegenerated = "auth.mfa.recovery_regenerated"
	ActionAccountLocked          = "auth.account_locked"

	ActionRoleCreate   = "rbac.role.create"
	ActionRoleUpdate   = "rbac.role.update"
	ActionRoleDelete   = "rbac.role.delete"
	ActionRoleAssign   = "rbac.role.assign"
	ActionRoleUnassign = "rbac.role.unassign"

	ActionAdminBanUser    = "admin.user.ban"
	ActionAdminUnbanUser  = "admin.user.unban"
	ActionAdminDeleteUser = "admin.user.delete"
	ActionImpersonate     = "admin.impersonate"
	ActionAccountDeletion = "user.account_deletion"
	ActionDataExport      = "user.data_export"

	ActionAppCreate = "app.create"
	ActionAppUpdate = "app.update"
	ActionAppDelete = "app.delete"

	ActionEnvironmentCreate = "environment.create"
	ActionEnvironmentUpdate = "environment.update"
	ActionEnvironmentDelete = "environment.delete"
	ActionEnvironmentClone  = "environment.clone"

	ActionPasskeyRegister = "passkey.register"
	ActionPasskeyLogin    = "passkey.login"
	ActionPasskeyDelete   = "passkey.delete"

	ActionAPIKeyCreate = "apikey.create"
	ActionAPIKeyRevoke = "apikey.revoke"

	ActionSocialSignIn = "social.signin"
	ActionSocialSignUp = "social.signup"

	ActionSSOSignIn = "sso.signin"
	ActionSSOSignUp = "sso.signup"

	ActionMFADisable = "auth.mfa.disable"

	ActionWaitlistJoin    = "waitlist.join"
	ActionWaitlistApprove = "waitlist.approve"
	ActionWaitlistReject  = "waitlist.reject"
)

// ──────────────────────────────────────────────────
// Resource constants
// ──────────────────────────────────────────────────

const (
	ResourceUser         = "user"
	ResourceSession      = "session"
	ResourceOrganization = "organization"
	ResourceMember       = "member"
	ResourceApp          = "app"
	ResourceDevice       = "device"
	ResourceTeam         = "team"
	ResourceInvitation   = "invitation"
	ResourceWebhook      = "webhook"
	ResourceRole         = "role"
	ResourcePermission   = "permission"
	ResourceEnvironment  = "environment"
	ResourcePasskey      = "passkey"
	ResourceAPIKey       = "apikey"
	ResourceWaitlist     = "waitlist"
)
