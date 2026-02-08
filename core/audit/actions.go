package audit

// AuditAction represents a standardized audit event action
type AuditAction string

// =============================================================================
// AUTH ACTIONS
// =============================================================================

const (
	// ActionAuthSignup represents a user signup event
	ActionAuthSignup AuditAction = "auth.signup"
	// ActionAuthSignin represents a user signin event
	ActionAuthSignin AuditAction = "auth.signin"
	// ActionAuthSigninFailed represents a failed signin attempt
	ActionAuthSigninFailed AuditAction = "auth.signin.failed"
	// ActionAuthSigninTwoFARequired represents signin requiring 2FA
	ActionAuthSigninTwoFARequired AuditAction = "auth.signin.twofa_required"
	// ActionAuthSignout represents a user signout event
	ActionAuthSignout AuditAction = "auth.signout"
)

// =============================================================================
// SESSION ACTIONS
// =============================================================================

const (
	// ActionSessionCreated represents a session creation event
	ActionSessionCreated AuditAction = "session.created"
	// ActionSessionRefreshed represents a session refresh event
	ActionSessionRefreshed AuditAction = "session.refreshed"
	// ActionSessionChecked represents a session validation check
	ActionSessionChecked AuditAction = "session.checked"
	// ActionSessionRevoked represents a session revocation event
	ActionSessionRevoked AuditAction = "session.revoked"
)

// =============================================================================
// USER ACTIONS
// =============================================================================

const (
	// ActionUserCreated represents a user creation event
	ActionUserCreated AuditAction = "user.created"
	// ActionUserUpdated represents a user update event
	ActionUserUpdated AuditAction = "user.updated"
	// ActionUserDeleted represents a user deletion event
	ActionUserDeleted AuditAction = "user.deleted"
	// ActionUserBanned represents a user ban event
	ActionUserBanned AuditAction = "user.banned"
	// ActionUserUnbanned represents a user unban event
	ActionUserUnbanned AuditAction = "user.unbanned"
	// ActionUserImpersonate represents a user impersonation event
	ActionUserImpersonate AuditAction = "user.impersonate"
)

// =============================================================================
// DEVICE ACTIONS
// =============================================================================

const (
	// ActionDevicesListed represents a device listing event
	ActionDevicesListed AuditAction = "device.listed"
	// ActionDeviceRevoked represents a device revocation event
	ActionDeviceRevoked AuditAction = "device.revoked"
)

// =============================================================================
// PASSWORD ACTIONS
// =============================================================================

const (
	// ActionPasswordResetRequested represents a password reset request
	ActionPasswordResetRequested AuditAction = "password.reset.requested"
	// ActionPasswordResetCompleted represents a completed password reset
	ActionPasswordResetCompleted AuditAction = "password.reset.completed"
	// ActionPasswordChanged represents a password change event
	ActionPasswordChanged AuditAction = "password.changed"
)

// =============================================================================
// EMAIL ACTIONS
// =============================================================================

const (
	// ActionEmailVerified represents an email verification event
	ActionEmailVerified AuditAction = "email.verified"
	// ActionEmailChangeRequested represents an email change request
	ActionEmailChangeRequested AuditAction = "email.change.requested"
	// ActionEmailChangeConfirmed represents a confirmed email change
	ActionEmailChangeConfirmed AuditAction = "email.change.confirmed"
)

// =============================================================================
// API KEY ACTIONS
// =============================================================================

const (
	// ActionAPIKeyCreated represents an API key creation event
	ActionAPIKeyCreated AuditAction = "apikey.created"
	// ActionAPIKeyUpdated represents an API key update event
	ActionAPIKeyUpdated AuditAction = "apikey.updated"
	// ActionAPIKeyDeleted represents an API key deletion event
	ActionAPIKeyDeleted AuditAction = "apikey.deleted"
	// ActionAPIKeyRotated represents an API key rotation event
	ActionAPIKeyRotated AuditAction = "apikey.rotated"
	// ActionAPIKeyRoleAssigned represents a role assignment to API key
	ActionAPIKeyRoleAssigned AuditAction = "apikey.role.assigned"
	// ActionAPIKeyRoleUnassigned represents a role removal from API key
	ActionAPIKeyRoleUnassigned AuditAction = "apikey.role.unassigned"
	// ActionAPIKeyRolesBulkAssigned represents bulk role assignments to API key
	ActionAPIKeyRolesBulkAssigned AuditAction = "apikey.roles.bulk_assigned"
)

// =============================================================================
// JWT ACTIONS
// =============================================================================

const (
	// ActionJWTKeyCreated represents a JWT key creation event
	ActionJWTKeyCreated AuditAction = "jwt.key.created"
)

// =============================================================================
// WEBHOOK ACTIONS
// =============================================================================

const (
	// ActionWebhookCreated represents a webhook creation event
	ActionWebhookCreated AuditAction = "webhook.created"
	// ActionWebhookUpdated represents a webhook update event
	ActionWebhookUpdated AuditAction = "webhook.updated"
	// ActionWebhookDeleted represents a webhook deletion event
	ActionWebhookDeleted AuditAction = "webhook.deleted"
)

// =============================================================================
// NOTIFICATION ACTIONS
// =============================================================================

const (
	// ActionNotificationTemplateCreated represents a notification template creation
	ActionNotificationTemplateCreated AuditAction = "notification.template.created"
	// ActionNotificationTemplateUpdated represents a notification template update
	ActionNotificationTemplateUpdated AuditAction = "notification.template.updated"
	// ActionNotificationTemplateDeleted represents a notification template deletion
	ActionNotificationTemplateDeleted AuditAction = "notification.template.deleted"
)

// =============================================================================
// DASHBOARD ACTIONS
// =============================================================================

const (
	// ActionDashboardAccess represents a dashboard access event
	ActionDashboardAccess AuditAction = "dashboard.access"
)

// =============================================================================
// MAGIC LINK ACTIONS
// =============================================================================

const (
	// ActionMagicLinkSent represents a magic link being sent
	ActionMagicLinkSent AuditAction = "magiclink.sent"
	// ActionMagicLinkVerifySuccessNewUser represents successful magic link verification for new user
	ActionMagicLinkVerifySuccessNewUser AuditAction = "magiclink.verify.success.new_user"
	// ActionMagicLinkVerifySuccessExistingUser represents successful magic link verification for existing user
	ActionMagicLinkVerifySuccessExistingUser AuditAction = "magiclink.verify.success.existing_user"
	// ActionMagicLinkVerifyFailed represents failed magic link verification
	ActionMagicLinkVerifyFailed AuditAction = "magiclink.verify.failed"
)

// =============================================================================
// PHONE ACTIONS
// =============================================================================

const (
	// ActionPhoneCodeCreated represents a phone verification code creation
	ActionPhoneCodeCreated AuditAction = "phone.code.created"
	// ActionPhoneSMSSent represents an SMS being sent
	ActionPhoneSMSSent AuditAction = "phone.sms.sent"
	// ActionPhoneSMSSendFailed represents a failed SMS send
	ActionPhoneSMSSendFailed AuditAction = "phone.sms.send.failed"
	// ActionPhoneCodeSendFailed represents a failed phone code send
	ActionPhoneCodeSendFailed AuditAction = "phone.code.send.failed"
	// ActionPhoneVerifySuccess represents successful phone verification
	ActionPhoneVerifySuccess AuditAction = "phone.verify.success"
	// ActionPhoneVerifyFailed represents failed phone verification
	ActionPhoneVerifyFailed AuditAction = "phone.verify.failed"
	// ActionPhoneVerifyInvalidCode represents invalid phone verification code
	ActionPhoneVerifyInvalidCode AuditAction = "phone.verify.invalid_code"
	// ActionPhoneVerifyTooManyAttempts represents too many phone verification attempts
	ActionPhoneVerifyTooManyAttempts AuditAction = "phone.verify.too_many_attempts"
	// ActionPhoneVerifyCodeNotFound represents phone verification code not found
	ActionPhoneVerifyCodeNotFound AuditAction = "phone.verify.code_not_found"
	// ActionPhoneVerifyDBError represents database error during phone verification
	ActionPhoneVerifyDBError AuditAction = "phone.verify.db_error"
	// ActionPhoneVerifyUserNotFound represents user not found during phone verification
	ActionPhoneVerifyUserNotFound AuditAction = "phone.verify.user_not_found"
	// ActionPhoneVerifyPasswordGenFailed represents password generation failure
	ActionPhoneVerifyPasswordGenFailed AuditAction = "phone.verify.password_gen_failed"
	// ActionPhoneVerifyUserCreationFailed represents user creation failure during phone verification
	ActionPhoneVerifyUserCreationFailed AuditAction = "phone.verify.user_creation_failed"
	// ActionPhoneVerifyImplicitSignup represents implicit signup during phone verification
	ActionPhoneVerifyImplicitSignup AuditAction = "phone.verify.implicit_signup"
	// ActionPhoneVerifySessionFailed represents session creation failure after phone verification
	ActionPhoneVerifySessionFailed AuditAction = "phone.verify.session_failed"
	// ActionPhoneLoginSuccess represents successful phone login
	ActionPhoneLoginSuccess AuditAction = "phone.login.success"
)

// =============================================================================
// SOCIAL ACTIONS
// =============================================================================

const (
	// ActionSocialSigninInitiated represents initiated social signin
	ActionSocialSigninInitiated AuditAction = "social.signin.initiated"
	// ActionSocialLinkInitiated represents initiated social account linking
	ActionSocialLinkInitiated AuditAction = "social.link.initiated"
	// ActionSocialCallbackReceived represents received OAuth callback
	ActionSocialCallbackReceived AuditAction = "social.callback.received"
	// ActionSocialTokenExchangeSuccess represents successful token exchange
	ActionSocialTokenExchangeSuccess AuditAction = "social.token.exchange.success"
	// ActionSocialTokenExchangeFailed represents failed token exchange
	ActionSocialTokenExchangeFailed AuditAction = "social.token.exchange.failed"
	// ActionSocialUserInfoFetched represents successful user info fetch
	ActionSocialUserInfoFetched AuditAction = "social.userinfo.fetched"
	// ActionSocialProviderNotFound represents social provider not found
	ActionSocialProviderNotFound AuditAction = "social.provider.not_found"
	// ActionSocialProviderLoadFailed represents failed provider loading
	ActionSocialProviderLoadFailed AuditAction = "social.provider.load_failed"
	// ActionSocialStateInvalid represents invalid OAuth state
	ActionSocialStateInvalid AuditAction = "social.state.invalid"
	// ActionSocialStateMismatch represents OAuth state mismatch
	ActionSocialStateMismatch AuditAction = "social.state.mismatch"
	// ActionSocialEmailNotVerified represents unverified email from social provider
	ActionSocialEmailNotVerified AuditAction = "social.email.not_verified"
)

// =============================================================================
// USERNAME ACTIONS
// =============================================================================

const (
	// ActionUsernameSignupSuccess represents successful username signup
	ActionUsernameSignupSuccess AuditAction = "username.signup.success"
	// ActionUsernameSignupFailed represents failed username signup
	ActionUsernameSignupFailed AuditAction = "username.signup.failed"
	// ActionUsernameSignupAttempt represents username signup attempt
	ActionUsernameSignupAttempt AuditAction = "username.signup.attempt"
	// ActionUsernameAlreadyExists represents username already exists
	ActionUsernameAlreadyExists AuditAction = "username.already_exists"
	// ActionUsernameWeakPassword represents weak password during signup
	ActionUsernameWeakPassword AuditAction = "username.weak_password"
	// ActionUsernameSigninSuccess represents successful username signin
	ActionUsernameSigninSuccess AuditAction = "username.signin.success"
	// ActionUsernameSigninFailed represents failed username signin
	ActionUsernameSigninFailed AuditAction = "username.signin.failed"
	// ActionUsernameSigninAttempt represents username signin attempt
	ActionUsernameSigninAttempt AuditAction = "username.signin.attempt"
	// ActionUsernameInvalidCredentials represents invalid credentials
	ActionUsernameInvalidCredentials AuditAction = "username.invalid_credentials"
	// ActionUsernameAccountLocked represents locked account
	ActionUsernameAccountLocked AuditAction = "username.account.locked"
	// ActionUsernameAccountLockedAuto represents automatically locked account
	ActionUsernameAccountLockedAuto AuditAction = "username.account.locked.auto"
	// ActionUsernamePasswordExpired represents expired password
	ActionUsernamePasswordExpired AuditAction = "username.password.expired"
	// ActionUsernameFailedAttemptRecorded represents recorded failed attempt
	ActionUsernameFailedAttemptRecorded AuditAction = "username.failed_attempt.recorded"
	// ActionUsernameFailedAttemptsCleared represents cleared failed attempts
	ActionUsernameFailedAttemptsCleared AuditAction = "username.failed_attempts.cleared"
)

// =============================================================================
// EMAIL OTP ACTIONS
// =============================================================================

const (
	// ActionEmailOTPSent represents an email OTP being sent
	ActionEmailOTPSent AuditAction = "emailotp.sent"
	// ActionEmailOTPVerifySuccess represents successful email OTP verification
	ActionEmailOTPVerifySuccess AuditAction = "emailotp.verify.success"
	// ActionEmailOTPVerifyFailed represents failed email OTP verification
	ActionEmailOTPVerifyFailed AuditAction = "emailotp.verify.failed"
	// ActionEmailOTPLogin represents email OTP login
	ActionEmailOTPLogin AuditAction = "emailotp.login"
)

// =============================================================================
// PASSKEY ACTIONS
// =============================================================================

const (
	// ActionPasskeyRegistered represents passkey registration
	ActionPasskeyRegistered AuditAction = "passkey.registered"
	// ActionPasskeyLogin represents passkey login
	ActionPasskeyLogin AuditAction = "passkey.login"
	// ActionPasskeyDeleted represents passkey deletion
	ActionPasskeyDeleted AuditAction = "passkey.deleted"
)

// =============================================================================
// IMPERSONATION ACTIONS
// =============================================================================

const (
	// ActionImpersonationStarted represents started impersonation session
	ActionImpersonationStarted AuditAction = "impersonation.started"
	// ActionImpersonationEnded represents ended impersonation session
	ActionImpersonationEnded AuditAction = "impersonation.ended"
)

// =============================================================================
// ROLE ACTIONS
// =============================================================================

const (
	// ActionRoleAssigned represents a role assignment
	ActionRoleAssigned AuditAction = "role.assigned"
	// ActionRoleRevoked represents a role revocation
	ActionRoleRevoked AuditAction = "role.revoked"
)

// String returns the string representation of the action
func (a AuditAction) String() string {
	return string(a)
}
