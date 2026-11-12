# Changelog

Release notes for the Go module. The GitHub release also lists every commit
since the last tag; this file says what changes for you when you upgrade.

## v1.11.0 (unreleased)

Breaking changes start with "Breaking:" and say what to do about them.

This release adds two migrations: invitations gain a nullable `sent_at` column, and
sessions gain a nullable `mfa_verified_at` column. Neither backfills.

### Invitations

- New routes. `POST /v1/orgs/:orgId/invitations/:invitationId/resend` gives a
  pending or expired invitation a new token and a fresh expiry, so the old link
  stops working. `DELETE /v1/orgs/:orgId/invitations/:invitationId` revokes it
  and keeps the row. Both need an admin who may grant the invitation's role,
  both answer 409 for any other status, and resend answers 429 inside a minute
  of the last send. They fire `org.invitation.resend` and
  `org.invitation.revoke` hooks. The SDKs and embedded clients have both.
- Authsome now emails an invitation itself, on create and on resend, when you
  set `InvitationURLTemplate` on the organization plugin's config (a URL with a
  `{token}` placeholder). It sends through the plugin registered as
  `notification`. With no template, or no notification plugin, nothing is sent,
  as before. A failed send is logged and never fails the create or the resend.
- Breaking: declining an invitation needs a signed-in invitee. An anonymous
  decline with just the token is refused with 401. Send the person through
  sign-in first.
- Breaking: when someone who isn't the invitee accepts or declines, they
  get the same 404 as an unknown token (it was 403 for a live one). The check
  runs before status and expiry, and their request never changes the
  invitation. If your client showed a "wrong account" message on 403, show it
  on 404 now.
- A token that was already used answers 409, and an expired one answers 410.
  Both used to be a 500.
- Breaking: only a verified address counts as the invitee: the primary
  email with `EmailVerified`, or a verified secondary one. An invitee whose
  matching address isn't verified gets the 404 above until they verify it.
- `RoleRanks` on the organization plugin's config gives custom role names a
  rank on the built-in scale. Unlisted names keep rank 0, below member, and
  owner, admin and member can't be changed. Use it when your own roles need to
  read the member list.

### Sessions and MFA

- A session now records `MFAVerifiedAt`: when it passed a second factor at the
  sign-in challenge, on verify, recovery verify or SMS verify from the session
  itself, or through a passkey sign-in that verified the user.
  `MarkSessionMFAVerified` updates only that column (and `updated_at`), so it
  can't race a refresh rotation, and the zero time clears it. Authsome's
  `MFARequired` is per app, so a host that enforces two-step per workspace can
  read this instead.
- Breaking: regenerating recovery codes needs a session that passed a
  second factor, or a current TOTP code in the body (the request has a new
  optional `code` field). Sessions that existed before this release have no
  stamp, so your client must pass a code for them. The route is also rate
  limited now.
- `POST /v1/mfa/enroll` and `DELETE /v1/mfa/enrollment` are rate limited like
  the other code-submission routes, keyed off `MFAChallengeLimit`. Both take a
  TOTP or recovery code, and could be brute-forced before.
- Breaking: once your SMS number is verified, `POST /v1/mfa/sms/send` always
  sends to it and ignores a `phone` in the body. Before this, a session that had
  only the password could text a code to any number and pass the second
  factor. To change a verified number, enroll again.
- Breaking: on an SMS enrollment that isn't verified yet, a `phone` in
  `POST /v1/mfa/sms/send` moves the enrollment to that number. It needs the same
  step-up as enrolling: a current TOTP code if the user has one (so the call
  is refused with 401 for them; enroll again with a code instead), or a recent
  sign-in. Confirming that enrollment with `POST /v1/mfa/sms/verify` marks it
  verified but no longer stamps the session. Send and verify a fresh code to
  stamp it.
- Turning MFA off with `DELETE /v1/mfa/enrollment` clears the current
  session's stamp. Other sessions were already revoked.

### Other

- The SCIM plugin's new `SCIMStore()` returns its config and token store, so
  your host app can read an org's provisioning state behind its own
  authorization.
