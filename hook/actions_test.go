package hook_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/hook"
)

// allActions lists every Action constant. A new constant belongs here; the
// reviewer catches an omission, the test catches a malformed name.
func allActions() []string {
	return []string{
		hook.ActionSignUp, hook.ActionSignIn, hook.ActionSignOut, hook.ActionRefresh, hook.ActionPrincipalAuth,
		hook.ActionDelegationGrant, hook.ActionDelegationRevoke,
		hook.ActionUserCreate, hook.ActionUserUpdate, hook.ActionUserDelete,
		hook.ActionSessionCreate, hook.ActionSessionRevoke,
		hook.ActionOrgCreate, hook.ActionOrgUpdate, hook.ActionOrgDelete,
		hook.ActionMemberAdd, hook.ActionMemberRemove, hook.ActionMemberRoleChange,
		hook.ActionInvitationAccept, hook.ActionInvitationDecline,
		hook.ActionTeamCreate, hook.ActionTeamUpdate, hook.ActionTeamDelete,
		hook.ActionWebhookCreate, hook.ActionWebhookUpdate, hook.ActionWebhookDelete,
		hook.ActionPasswordReset, hook.ActionPasswordChange, hook.ActionEmailVerify, hook.ActionEmailVerificationRequested,
		hook.ActionMFAEnroll, hook.ActionMFAChallenge, hook.ActionMFARecoveryUsed, hook.ActionMFARecoveryRegenerated,
		hook.ActionAccountLocked, hook.ActionRefreshTokenReplayed,
		hook.ActionRoleCreate, hook.ActionRoleUpdate, hook.ActionRoleDelete, hook.ActionRoleAssign, hook.ActionRoleUnassign,
		hook.ActionAdminBanUser, hook.ActionAdminUnbanUser, hook.ActionAdminDeleteUser,
		hook.ActionImpersonate, hook.ActionImpersonateStop,
		hook.ActionAdminCopyUser, hook.ActionAdminBulkImport, hook.ActionAdminBulkRevokeSessions,
		hook.ActionPasswordResetRequested,
		hook.ActionSettingsUpdate, hook.ActionSettingsEnforce, hook.ActionSettingsUnenforce, hook.ActionSettingsDelete,
		hook.ActionAuditRead,
		hook.ActionAccountDeletion, hook.ActionDataExport,
		hook.ActionAppCreate, hook.ActionAppUpdate, hook.ActionAppDelete,
		hook.ActionEnvironmentCreate, hook.ActionEnvironmentUpdate, hook.ActionEnvironmentDelete, hook.ActionEnvironmentClone,
		hook.ActionPasskeyRegister, hook.ActionPasskeyLogin, hook.ActionPasskeyDelete,
		hook.ActionAPIKeyCreate, hook.ActionAPIKeyRevoke,
		hook.ActionSocialSignIn, hook.ActionSocialSignUp, hook.ActionSSOSignIn, hook.ActionSSOSignUp,
		hook.ActionMFADisable,
		hook.ActionWaitlistJoin, hook.ActionWaitlistApprove, hook.ActionWaitlistReject,
	}
}

// legacyActions predate the dotted naming scheme. They are queried by name in
// existing deployments, so they keep their spelling; nothing new joins them.
func legacyActions() []string {
	return []string{hook.ActionDPoPKeyMismatch, hook.ActionDPoPProofInvalid, hook.ActionDPoPProofReplayed}
}

func TestActionConstantsFollowTheNamingScheme(t *testing.T) {
	re := regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)+$`)
	seen := map[string]bool{}
	for _, a := range allActions() {
		assert.Regexp(t, re, a, "action %q must be dotted lower snake case", a)
		assert.False(t, seen[a], "action %q is defined twice", a)
		seen[a] = true
	}
	for _, a := range legacyActions() {
		assert.False(t, seen[a], "legacy action %q must not also be in the modern list", a)
	}
}
