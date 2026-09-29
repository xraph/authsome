// Package hook provides a global event bus that fires on every engine action.
// This is the single interception point for logging, auditing, metrics,
// and custom logic.
package hook

import (
	"context"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"
)

// Severity levels an event can carry. They mirror the audit sink's levels so
// a call site states how alarming an action is once, at the point it happens.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Outcome values. Emit derives Outcome from Err when the call site leaves it
// empty.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
)

// Event represents a global hook event emitted on every engine action.
//
// Metadata is written to the audit trail verbatim, so it must never hold a
// credential. Anything a downstream handler needs but the trail must not keep
// (a reset token, an OTP code) goes in Private, which is dropped before
// recording and excluded from JSON.
type Event struct {
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	ResourceID string            `json:"resource_id,omitempty"`
	ActorID    string            `json:"actor_id,omitempty"`
	Tenant     string            `json:"tenant,omitempty"`
	OrgID      string            `json:"org_id,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Private    map[string]string `json:"-"`
	Err        error             `json:"-"`

	// Audit shape. Severity defaults to info and Outcome is derived from Err
	// when left empty. Category groups events for reporting ("auth", "admin").
	Severity string `json:"severity,omitempty"`
	Category string `json:"category,omitempty"`
	Outcome  string `json:"outcome,omitempty"`
	Reason   string `json:"reason,omitempty"`

	// Request correlation, filled by Emit from the context when empty.
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
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

// enrich fills the defaults every event carries: timestamp, outcome from Err,
// info severity, and the request correlation fields from the context. It
// also warns about an event with no tenant, because such an event cannot be
// found again by the tenant it concerns.
func (b *Bus) enrich(ctx context.Context, event *Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.Outcome == "" {
		if event.Err != nil {
			event.Outcome = OutcomeFailure
		} else {
			event.Outcome = OutcomeSuccess
		}
	}
	if event.Severity == "" {
		event.Severity = SeverityInfo
	}
	if event.Reason == "" && event.Err != nil {
		event.Reason = event.Err.Error()
	}
	if info, ok := RequestInfoFrom(ctx); ok {
		if event.IP == "" {
			event.IP = info.IP
		}
		if event.UserAgent == "" {
			event.UserAgent = info.UserAgent
		}
		if event.RequestID == "" {
			event.RequestID = info.RequestID
		}
		if event.SessionID == "" {
			event.SessionID = info.SessionID
		}
	}
	if event.Tenant == "" && b.logger != nil {
		b.logger.Warn("global hook event without a tenant",
			log.String("action", event.Action),
			log.String("resource", event.Resource),
		)
	}
}

// Emit dispatches an event to all registered handlers. Handler errors are
// logged and never propagated; use EmitCritical when a failed record must
// stop the caller.
func (b *Bus) Emit(ctx context.Context, event *Event) {
	_ = b.dispatch(ctx, event)
}

// EmitCritical dispatches like Emit but returns the first handler error.
// Every handler still runs, so metrics and notifications observe the event
// even when the audit sink refused it.
func (b *Bus) EmitCritical(ctx context.Context, event *Event) error {
	return b.dispatch(ctx, event)
}

func (b *Bus) dispatch(ctx context.Context, event *Event) error {
	b.enrich(ctx, event)
	var first error
	for _, h := range b.handlers {
		if err := h.handler(ctx, event); err != nil {
			if first == nil {
				first = fmt.Errorf("hook %q: %w", h.name, err)
			}
			if b.logger != nil {
				b.logger.Warn("global hook error",
					log.String("hook", h.name),
					log.String("action", event.Action),
					log.String("error", err.Error()),
				)
			}
		}
	}
	return first
}

// ──────────────────────────────────────────────────
// Action constants
// ──────────────────────────────────────────────────

const (
	ActionSignUp  = "auth.signup"
	ActionSignIn  = "auth.signin"
	ActionSignOut = "auth.signout"
	ActionRefresh = "auth.refresh"
	// ActionPrincipalAuth fires after a non-human caller has authenticated.
	// The typed BeforePrincipalAuth hook is what denies; this is the audit
	// and relay signal, so Chronicle picks up machine auth without any
	// subscriber implementing a plugin interface.
	ActionPrincipalAuth = "auth.principal"

	// ActionDelegationGrant fires when a delegation grant is recorded,
	// letting an actor act on a subject's behalf.
	ActionDelegationGrant = "principal.delegation.grant"
	// ActionDelegationRevoke fires when a delegation grant is revoked.
	ActionDelegationRevoke = "principal.delegation.revoke"

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

	ActionPasswordReset  = "auth.password_reset"
	ActionPasswordChange = "auth.password_change"
	ActionEmailVerify    = "auth.email_verify"
	// ActionEmailVerificationRequested fires when the engine mints a
	// fresh email-verification token. Subscribers (notification plugin,
	// custom mailers) read Metadata["verification_token"] to render the
	// link the user clicks. Distinct from ActionEmailVerify which
	// fires on successful verification.
	ActionEmailVerificationRequested = "auth.email_verification_requested"
	ActionMFAEnroll                  = "auth.mfa.enroll"
	ActionMFAChallenge               = "auth.mfa.challenge"
	ActionMFARecoveryUsed            = "auth.mfa.recovery_used"
	ActionMFARecoveryRegenerated     = "auth.mfa.recovery_regenerated"
	ActionAccountLocked              = "auth.account_locked"
	// ActionRefreshTokenReplayed fires when a refresh-token presented to
	// Engine.Refresh has already been rotated, indicating a leaked token
	// is being replayed. The engine cascade-revokes the entire session
	// family (RFC 6819 §5.2.2.3) and emits this hook with metadata
	// {family_id, ip, user_agent}. The actor is intentionally left blank —
	// the caller is by definition not trusted.
	// #nosec G101 -- not a credential: an env var name, collection name or public OAuth2 endpoint URL.
	ActionRefreshTokenReplayed = "auth.refresh_token_replayed"

	ActionRoleCreate   = "rbac.role.create"
	ActionRoleUpdate   = "rbac.role.update"
	ActionRoleDelete   = "rbac.role.delete"
	ActionRoleAssign   = "rbac.role.assign"
	ActionRoleUnassign = "rbac.role.unassign"

	ActionAdminBanUser    = "admin.user.ban"
	ActionAdminUnbanUser  = "admin.user.unban"
	ActionAdminDeleteUser = "admin.user.delete"
	ActionImpersonate     = "admin.impersonate"
	// ActionImpersonateStop fires when an impersonation session is ended.
	ActionImpersonateStop = "admin.impersonate.stop"
	// ActionAdminCopyUser fires when an admin clones a user into another app.
	ActionAdminCopyUser = "admin.user.copy"
	// ActionAdminBulkImport fires once per bulk user import.
	ActionAdminBulkImport = "admin.user.bulk_import"
	// ActionAdminBulkRevokeSessions fires when an admin revokes every session
	// of one user.
	ActionAdminBulkRevokeSessions = "admin.session.bulk_revoke"
	// ActionPasswordResetRequested fires when a reset link is issued. Distinct
	// from ActionPasswordReset, which delivers the link, so the trail can tell
	// a request from a completed reset.
	ActionPasswordResetRequested = "auth.password_reset_requested"
	// Settings changes are recorded with the key, scope and both values so
	// an auditor can see who weakened what.
	ActionSettingsUpdate    = "settings.update"
	ActionSettingsEnforce   = "settings.enforce"
	ActionSettingsUnenforce = "settings.unenforce"
	ActionSettingsDelete    = "settings.delete"
	// ActionAuditRead fires when the audit trail itself is queried through
	// authsome.
	ActionAuditRead = "audit.read"
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

	// ActionDPoPKeyMismatch fires when a structurally valid DPoP proof is
	// presented for the wrong key, on the refresh-token path. A proof that
	// verifies but does not match the bound thumbprint has no innocent
	// explanation the way a changed IP does.
	ActionDPoPKeyMismatch = "dpop_key_mismatch"

	// ActionDPoPProofInvalid fires when a bound token arrives without a usable
	// proof. Expect volume: it is nearly always a client that has not been
	// updated, so it is recorded at info severity and is not on its own a
	// signal of attack.
	ActionDPoPProofInvalid = "dpop_proof_invalid"

	// ActionDPoPProofReplayed fires when a proof's jti is seen twice inside
	// its window. Rare, and interesting when it happens: it means somebody
	// captured a proof and the token it was minted for.
	ActionDPoPProofReplayed = "dpop_proof_replayed"
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
