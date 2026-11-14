# Changelog

Release notes for the Go module. The GitHub release also lists every commit
since the last tag; this file says what changes for you when you upgrade.

## v1.12.0 (unreleased)

Breaking changes start with "Breaking:" and say what to do about them.

This release adds one migration: the mfa plugin gains an
`authsome_mfa_trusted_devices` table (postgres and SQLite; mongo needs nothing).
It starts empty. The magic link sign-in code needs none: it lives in the
ceremony store next to the link's existing verification row.

### Magic link

- The magic link email can carry a 6-digit code as well. Turn it on with
  `EmailCode` on the plugin's config or the `magiclink.email_code` setting.
  The code and the link share one lifetime, and using either one uses up
  both. Each code allows five tries, and a resend replaces it. The code is
  stored only as a hash.
- New route `POST /v1/magic-link/verify-code` takes `{email, code}` and
  answers like `/verify`, including the 403 `mfa_required` body with an
  `mfa_ticket` for a user with a second factor. It is rate limited like
  `/verify`. An unknown address, a wrong code and an expired one all get the
  same 401.
- A `Mailer` that also implements the new `magiclink.CodeMailer` gets
  `SendMagicLinkWithCode` with a `MagicLinkMessage`: the address, token, code,
  purpose (`signin` or `signup`), expiry and name. The code is only sent
  through it. A plain `Mailer` works as before and gets no code.
- `POST /v1/magic-link/send` answers with `expires_in` (seconds) and, when a
  code is sent, `code_length`, so a client can draw the boxes and a
  countdown. The answer is the same whether or not the address has an
  account.
- Passwordless sign-up. With `AllowSignUp` on the plugin's config or the
  `magiclink.allow_signup` setting, and sign-up enabled for the app, a send
  with `create_user: true` (and an optional `name`) creates an account with
  no password when none has the address, then emails a `signup` link. An
  existing account gets an ordinary sign-in link and the same answer. The
  account is created through the same path as `/v1/signup`, so sign-up
  hooks such as the waitlist, the app's sign-up switch, the sign-up captcha
  and email normalization all apply. A sign-up that is off or refused is a
  403.
- Using a link or a code now marks the email verified the way the
  verification code does: it fires the email-verified hook and webhook, and
  can claim an open platform owner slot.

### Accounts

- New `Engine.SignUpPasswordless` creates an account with no password
  through the sign-up path, and `Engine.MarkEmailVerified` records an inbox
  proven some other way. A disabled sign-up now wraps
  `account.ErrSignUpDisabled`, and `SignUpPasswordless` wraps a hook's
  refusal in `account.ErrSignUpRefused`.
- An account with no password can't be signed in through `/v1/signin` with
  any password, empty or not, and the refusal takes as long as any other.
  Forgot and reset password give it one.

### Trusted devices

- A user can now trust the device they just passed the second factor on, and
  skip it there for 30 days. Send `"trust_device": true` to
  `POST /v1/mfa/challenge`. The response then carries `trusted_device_token`
  and `trusted_device_expires_at`, and sets an httpOnly
  `authsome_trusted_device` cookie (Path `/`, SameSite Lax, Secure whenever
  the session cookie is). Only a hash of the token is stored.
- Every sign-in path honours the token: password, email-verification
  auto-login, magic link, passkey, social, SSO and phone. Browsers send the
  cookie; other clients send the `X-Authsome-Trusted-Device` header, which wins
  when both are present. A trusted device skips the per-user gate and the
  app-wide `MFARequired` rule alike, and its session gets `MFAVerifiedAt` set
  to when the device was trusted, so a host that wants a recent second factor
  still asks for one. An unknown, expired, removed or other user's token
  changes nothing, and a failed lookup asks for the second factor.
- If you call `IssueSession` yourself, set the new
  `IssueSessionRequest.TrustedDeviceToken` from
  `authsome.TrustedDeviceToken(r)`. `account.SignInRequest` has the same field
  for `Engine.SignIn`. Leave it empty and nothing changes.
- New routes. `GET /v1/mfa/trusted-devices` lists the signed-in user's live
  devices with a label, user agent, IP, created, last used and expiry times,
  and marks the one making the request `current`.
  `DELETE /v1/mfa/trusted-devices/:id` removes one and
  `DELETE /v1/mfa/trusted-devices` removes them all. `GET /v1/mfa/enrollment`
  gains a `trusted_devices` count.
- Trusted devices are forgotten when the user turns MFA off, changes or resets
  their password, or is deleted.
- `mfa.Config` gains `TrustedDeviceTTL` (30 days when left at zero; a negative
  value turns the feature off) and `DisableTrustedDevices`. The
  `mfa.trusted_devices_enabled` setting turns it off for one app; off, existing
  tokens stop working too.
- Breaking, if you implement `mfa.Store` yourself: it gains six
  trusted-device methods. The built-in stores and `EncryptedStore` have them.
- New hooks: `auth.mfa.trusted_device_add` and
  `auth.mfa.trusted_device_remove`.

### Social and SSO sign-in from a browser

- Breaking: when a browser social callback (`GET /v1/social/:provider/callback`) needs a
  second factor, it now redirects with `302` to the `redirect_url` the flow started with
  (else its `frontend_url`), adding `mfa_ticket=<ticket>&mfa_methods=<a,b>` to the query.
  It used to answer the JSON 403 `mfa_required` body, which the browser rendered as a
  page. Read the two parameters on your landing page and finish at the MFA challenge with
  the ticket. A flow started without either URL still gets the JSON 403.
- Breaking: the SSO OIDC landing and SAML ACS now redirect with `mfa_ticket` and
  `mfa_methods` too, where they used to redirect with `sso_error=auth_failed`. The JSON
  `POST /v1/sso/:provider/callback` still answers the 403.
- The ticket keeps the auth method (`social:google`, `social:kineta`, `sso:<provider>`),
  so the session minted after the challenge records it as before (with `+mfa`).
- A successful browser social callback still sets the session cookie and serves the
  page that closes a popup, and now also adds a one-time `code` to the URL it redirects
  to. An app that keeps tokens on its own origin redeems it once, within 60 seconds and
  with its publishable key, at the new `POST /v1/social/exchange {code}`, which answers
  like the API callback (`session_token`, `refresh_token`, `expires_at`, `user`,
  `provider`, `is_new_user`). The exchange rotates the session, so the cookie's token
  stops working; use one or the other. `POST /v1/sso/exchange` shares the code and now
  also refuses a code two replicas try to redeem at once.

### Sign-up gates

- Breaking: a first social or SSO sign-in now has to pass the same gates as a password
  sign-up before the account is created: the app's sign-up switch (`signup_enabled`) and
  every `BeforeSignUp` plugin hook (waitlist, IP reputation, geofencing, the password
  plugin's `AllowedDomains`, your own). The `AfterSignUp` hooks fire after. Accounts that
  already exist are never gated. A refused browser sign-in redirects with
  `error=<code>` (social) or `sso_error=<code>` (SSO), where the code is
  `signup_disabled`, `waitlist_required` or `signup_refused`, and writes nothing. API
  callers get 403 `{"type":"signup_refused","reason":"<code>"}`. If you relied on social
  or SSO to let people past a waitlist or a closed sign-up, approve them first.
- Captcha is not asked for on these sign-ups. It is a check on the sign-up form, and the
  identity provider round trip already shows a person is there.
- `Engine.GateFederatedSignUp` and `Engine.FinishFederatedSignUp` run those gates for
  any other path that creates accounts. A refusal is `*authsome.SignUpRefusedError`.
- `account.SignUpRequest` gains `AuthMethod` (`social:google`, `sso:acme`; empty for a
  password sign-up), so a hook can tell the paths apart.
- A `BeforeSignUp` hook can name its own refusal code by returning an error that
  implements `authsome.SignUpRefusal`. The waitlist's refusal is
  `waitlist.ErrApprovalRequired`, with the same message as before.

### SSO discovery

- New route `GET /v1/sso/discover?email=` answers `{connection_id, provider, name,
  protocol, domain, enforced}` for the active connection on the email's domain, or 404
  `{"error":"no sso connection for this domain"}`. Unlike `POST /v1/sso/login` it starts
  no login. It shares the SSO routes' per-address rate limit. `Plugin.DiscoverByEmail`
  answers the same in-process and returns `sso.ErrNoConnectionForDomain` on a miss.

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
- Breaking: `organization.ErrInvitationEmailMismatch` is gone, since that case
  is now the same not-found error as an unknown token. If your code checks for
  it, check for `store.ErrNotFound` instead.
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
- The MFA routes now count against their own per-address budget
  (`MFAChallengeLimit`), kept apart from the one sign-up, email verification
  and sign-in share. Before, someone who enrolled straight after signing up
  could have their first code refused with 429.
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
