# passkey

WebAuthn and FIDO2 sign-in. Face ID, Touch ID, Windows Hello, a YubiKey, or a passkey synced
through iCloud or a password manager.

This is the one auth method on the list that phishing can't beat. The credential is bound to
your origin by the browser, so a lookalike domain gets nothing, and the private key never
leaves the authenticator. Registration and login are both two-step ceremonies: begin, then
finish.

The plugin owns its own credential store, so it brings a migration with it.

## Use it when

- Phishing resistance is the requirement, not a nice-to-have
- You want sign-in that's genuinely faster than typing a password, on devices that already have a biometric
- You're moving off passwords and need something users will not fight you on

## Skip it when

- Your users are on shared or locked-down machines with no authenticator to enrol against
- You can't guarantee HTTPS and a stable domain. `RPID` is the domain, and changing it invalidates every credential your users registered
- You need it to be the only method on day one. Keep a fallback until enrolment is high

## Wiring

```go
authsome.WithPlugin(passkey.New(passkey.Config{
    RPDisplayName: "My App",
    RPID:          "example.com",
    RPOrigins:     []string{"https://example.com"},
}))
```

`RPID` must match the domain the browser sees. Get it wrong and the ceremony fails with an
error that does not say why.

## Config

| Field | Type | Default | What it does |
|---|---|---|---|
| `RPDisplayName` | `string` | `AuthSome` | Name shown in the browser prompt |
| `RPID` | `string` | `localhost` | Relying party ID, normally your bare domain |
| `RPOrigins` | `[]string` | empty | Origins allowed to run ceremonies |
| `SessionTimeout` | `time.Duration` | `5m` | How long a half-finished ceremony stays open |

## Settings

| Key | Default |
|---|---|
| `passkey.rp_display_name` | `AuthSome` |
| `passkey.rp_id` | `localhost` |
| `passkey.rp_origins` | `[]` |
| `passkey.session_timeout_seconds` | `300` |

## Endpoints

Registered under `/v1/passkeys`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/register/begin` | Start enrolment, returns the creation challenge |
| `POST` | `/register/finish` | Complete enrolment and store the credential |
| `POST` | `/login/begin` | Start authentication, returns the assertion challenge |
| `POST` | `/login/finish` | Verify the assertion and mint a session |
| `GET` | `/` | List the calling user's passkeys |
| `DELETE` | `/:credentialId` | Remove a passkey |

## Lifecycle hooks

None on the sign-in path. It registers as an `AuthMethodContributor` so the dashboard and the
sign-in UI know the user has a passkey enrolled, and a `MigrationProvider` for its credential
table.

## Related

`mfa` covers TOTP and SMS if you want a second factor rather than a replacement for the first.
`magiclink` is the other passwordless route.
